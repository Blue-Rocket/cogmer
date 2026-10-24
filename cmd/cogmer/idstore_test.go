package main

import (
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// The membership.db schema as it was before identity_records, machines and the
// guest-list cache were added, copied here so that the migration is tested against
// the store users hold now rather than against whatever the schema becomes.
const membershipSchemaBeforeUserIdentity = `
CREATE TABLE known_peers (
  peer_id     TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  added_at    TEXT NOT NULL,
  verified_at TEXT,
  endpoint    TEXT,
  endpoint_at TEXT
);
CREATE TABLE pending_offers (
  room_id      TEXT PRIMARY KEY,
  room_name    TEXT NOT NULL,
  host_peer_id TEXT NOT NULL,
  endpoint     TEXT NOT NULL,
  offered_at   TEXT NOT NULL
);
CREATE TABLE rooms (
  room_id         TEXT PRIMARY KEY,
  room_name       TEXT NOT NULL,
  state           TEXT NOT NULL,
  issued_sequence INTEGER NOT NULL DEFAULT 0,
  created_at      TEXT NOT NULL,
  show_only       INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE room_guests (
  room_id TEXT NOT NULL,
  peer_id TEXT NOT NULL,
  PRIMARY KEY (room_id, peer_id)
);
CREATE TABLE session_rooms (
  session_id TEXT PRIMARY KEY,
  room_id    TEXT NOT NULL,
  joined_at  TEXT NOT NULL,
  left_at    TEXT
);`

// A membership.db made before the user identity opens, gains every new table, and
// carries each known peer's address into machines, keeping every row it held. A
// store that gained the tables and lost the addresses would leave the readers that
// move to machines with nowhere to dial (D-103, an address belongs to a machine and
// is stored in one place).
func TestAMembershipStoreFromBeforeTheUserIdentityMigrates(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("COGMER_HOME", "")
	if err := os.MkdirAll(homeDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	alice, bob := testIdentity(t).PeerID, testIdentity(t).PeerID
	old, err := sql.Open("sqlite", filepath.Join(homeDir(), "membership.db"))
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		membershipSchemaBeforeUserIdentity,
		`INSERT INTO known_peers (peer_id, name, added_at, verified_at, endpoint, endpoint_at)
		 VALUES ('` + alice + `','alice','2026-09-01T00:00:00Z','2026-09-02T00:00:00Z','alice.example:7777','2026-09-03T00:00:00Z')`,
		`INSERT INTO known_peers (peer_id, name, added_at) VALUES ('` + bob + `','bob','2026-09-01T00:00:00Z')`,
		`INSERT INTO rooms (room_id, room_name, state, issued_sequence, created_at) VALUES ('room-a','misty-canyon','joined',7,'2026-09-01T00:00:00Z')`,
		`INSERT INTO room_guests (room_id, peer_id) VALUES ('room-a','` + alice + `')`,
		`INSERT INTO session_rooms (session_id, room_id, joined_at) VALUES ('s1','room-a','2026-09-01T00:00:00Z')`,
		`INSERT INTO pending_offers VALUES ('room-b','quiet-lake','` + bob + `','bob.example:7777','2026-09-01T00:00:00Z')`,
	} {
		if _, err := old.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	old.Close()

	m, err := OpenMembership()
	if err != nil {
		t.Fatalf("opening a store from before the user identity: %v", err)
	}
	t.Cleanup(func() { m.Close() })

	for _, table := range []string{"identity_records", "machines", "room_guests_cache", "room_guest_cutoffs"} {
		var n int
		if err := m.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n); err != nil || n != 1 {
			t.Errorf("the migrated store has no %s table", table)
		}
	}

	got, err := m.UserMachines(alice)
	if err != nil {
		t.Fatal(err)
	}
	want := []KnownMachine{{MachineKey: alice, User: alice, Endpoint: "alice.example:7777", EndpointAt: "2026-09-03T00:00:00Z"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("alice's address was not carried into machines:\n got %+v\nwant %+v", got, want)
	}
	if e, _ := m.MachineEndpoint(bob); e != "" {
		t.Errorf("a peer with no address gained one: %q", e)
	}
	if got, _ := m.UserMachines(bob); len(got) != 0 {
		t.Errorf("a peer with no address was given a machine row: %+v", got)
	}

	// Every old row is still there, and the readers that still use known_peers
	// still find the address there.
	if m.PeerEndpoint(alice) != "alice.example:7777" {
		t.Errorf("known_peers lost alice's address: %q", m.PeerEndpoint(alice))
	}
	peers, err := m.KnownPeers()
	if err != nil || len(peers) != 2 || !m.IsVerified(alice) {
		t.Errorf("known peers did not survive: %+v %v", peers, err)
	}
	if r, ok := m.RoomByID("room-a"); !ok || r.RoomName != "misty-canyon" || m.IssuedSequence("room-a") != 7 {
		t.Errorf("the room did not survive: %+v", r)
	}
	if !m.IsGuest("room-a", alice) {
		t.Error("the room's guest did not survive")
	}
	if r, ok := m.RoomForSession("s1"); !ok || r.RoomID != "room-a" {
		t.Error("the session's room did not survive")
	}
	if o, ok := m.FindOffer("quiet-lake"); !ok || o.Host != bob {
		t.Error("the pending offer did not survive")
	}

	// Opening again changes nothing, and an address written to machines itself is
	// not overwritten by the copy from known_peers.
	if err := m.SetMachineEndpoint(alice, alice, "alice.example:8888"); err != nil {
		t.Fatal(err)
	}
	m.Close()
	m2, err := OpenMembership()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m2.Close() })
	if e, _ := m2.MachineEndpoint(alice); e != "alice.example:8888" {
		t.Errorf("reopening overwrote the machine's own address with known_peers': %q", e)
	}
}

// identityFixture is one user with a shared key, a machine, a cloud key with a
// session, a removal and a list change, all validly signed.
type identityFixture struct {
	master, shared, machine, cloud, session ed25519.PrivateKey
	user                                    string
	sharedCert                              SharedKeyCert
	machineCert, cloudCert                  MachineCert
	sessionCert                             SessionCert
	removal                                 Removal
	change                                  ListChange
}

func newIdentityFixture(t *testing.T) *identityFixture {
	t.Helper()
	key := func() ed25519.PrivateKey {
		_, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatal(err)
		}
		return priv
	}
	pub := func(k ed25519.PrivateKey) ed25519.PublicKey { return k.Public().(ed25519.PublicKey) }
	f := &identityFixture{master: key(), shared: key(), machine: key(), cloud: key(), session: key()}
	f.user = PeerIDFromPublic(pub(f.master))
	f.sharedCert = CertifySharedKey(f.master, 1, pub(f.shared))
	f.machineCert = CertifyMachine(f.shared, f.user, 1, pub(f.machine), KindMachine, "laptop", 1)
	f.cloudCert = CertifyMachine(f.shared, f.user, 1, pub(f.cloud), KindCloud, "cloud", 2)
	f.sessionCert = CertifySession(f.cloud, f.user, pub(f.session), time.Now().Add(time.Hour))
	var err error
	f.removal, err = SignRemoval(f.shared, LevelShared, 1, f.user, f.cloudCert.Machine, []RoomCutoff{{Room: "room-a", Sequence: 4}})
	if err != nil {
		t.Fatal(err)
	}
	f.change = SignListChange(f.machine, f.user, "colleague:"+testIdentity(t).PeerID, "carol", 1)
	return f
}

func putRecord(t *testing.T, m *Membership, kind, user string, rec any, hash [32]byte) bool {
	t.Helper()
	body, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	got, isNew, err := m.PutIdentityRecord(kind, user, body)
	if err != nil {
		t.Fatal(err)
	}
	if got != hash {
		t.Errorf("a %s record was stored under %x, not its own hash %x", kind, got[:8], hash[:8])
	}
	return isNew
}

// A record is stored under the hash it gives itself, whatever its sender might
// claim, and storage refuses a body that names another user, one that is not the
// kind it is stored as, and one under a scheme this build does not know, whose hash
// would be zero and win every tie.
func TestAnIdentityRecordIsKeyedByItsOwnHash(t *testing.T) {
	m := testMembership(t)
	f := newIdentityFixture(t)
	for _, c := range []struct {
		kind string
		rec  any
		hash [32]byte
	}{
		{RecordShared, f.sharedCert, f.sharedCert.Hash()},
		{RecordMachine, f.machineCert, f.machineCert.Hash()},
		{RecordSession, f.sessionCert, f.sessionCert.Hash()},
		{RecordRemoval, f.removal, f.removal.Hash()},
		{RecordList, f.change, f.change.Hash()},
	} {
		body, _ := json.Marshal(c.rec)
		hash, isNew, err := m.PutIdentityRecord(c.kind, f.user, body)
		if err != nil || !isNew || hash != c.hash {
			t.Errorf("%s: stored under %x, new %v, err %v; want its own hash %x", c.kind, hash[:8], isNew, err, c.hash[:8])
			continue
		}
		kind, got, ok, err := m.IdentityRecordByHash(c.hash)
		if err != nil || !ok || kind != c.kind || string(got) != string(body) {
			t.Errorf("%s: fetched by its hash as %q %q, ok %v, err %v", c.kind, kind, got, ok, err)
		}
	}
	if _, _, ok, err := m.IdentityRecordByHash([32]byte{9}); ok || err != nil {
		t.Errorf("a hash never stored was found: %v %v", ok, err)
	}

	other := newIdentityFixture(t)
	shared, _ := json.Marshal(f.sharedCert)
	session, _ := json.Marshal(f.sessionCert)
	unknown := f.machineCert
	unknown.Scheme = 99
	unknownBody, _ := json.Marshal(unknown)
	for name, c := range map[string]struct {
		kind, user string
		body       []byte
	}{
		"a record for another user":          {RecordShared, other.user, shared},
		"a session certificate for another":  {RecordSession, other.user, session},
		"a shared certificate as a machine":  {RecordMachine, f.user, shared},
		"a session certificate as a removal": {RecordRemoval, f.user, session},
		"a record under an unknown scheme":   {RecordMachine, f.user, unknownBody},
		"a body that is not JSON":            {RecordMachine, f.user, []byte("{not json")},
		"a record of no known kind":          {"certificate", f.user, shared},
	} {
		if _, _, err := m.PutIdentityRecord(c.kind, c.user, c.body); err == nil {
			t.Errorf("%s was stored", name)
		}
	}
	var n int
	if err := m.db.QueryRow(`SELECT COUNT(*) FROM identity_records`).Scan(&n); err != nil || n != 5 {
		t.Errorf("the store holds %d records, want the 5 accepted", n)
	}
}

// Every kind of record stored comes back decoded into the form Resolve takes, so
// that the state a machine computes from its store is the state it would compute
// from the records as they arrived. A record stored twice is stored once, and a
// row that does not decode is left out without hiding the user's other records.
func TestIdentityRecordsComeBackAsTheRecordsThatWereStored(t *testing.T) {
	m := testMembership(t)
	f := newIdentityFixture(t)

	if !putRecord(t, m, RecordShared, f.user, f.sharedCert, f.sharedCert.Hash()) {
		t.Error("a new record was reported as already held")
	}
	if putRecord(t, m, RecordShared, f.user, f.sharedCert, f.sharedCert.Hash()) {
		t.Error("a record stored twice was reported as new the second time")
	}
	putRecord(t, m, RecordMachine, f.user, f.machineCert, f.machineCert.Hash())
	putRecord(t, m, RecordMachine, f.user, f.cloudCert, f.cloudCert.Hash())
	putRecord(t, m, RecordSession, f.user, f.sessionCert, f.sessionCert.Hash())
	putRecord(t, m, RecordRemoval, f.user, f.removal, f.removal.Hash())
	putRecord(t, m, RecordList, f.user, f.change, f.change.Hash())
	// A row written by something other than PutIdentityRecord, which refuses a body
	// that does not decode, is still left out rather than failing the read.
	if _, err := m.db.Exec(`INSERT INTO identity_records (hash, user, kind, body, received_at)
		VALUES ('01', ?, 'machine', '{not json', '2026-10-24T00:00:00Z')`, f.user); err != nil {
		t.Fatal(err)
	}

	other := newIdentityFixture(t)
	putRecord(t, m, RecordShared, other.user, other.sharedCert, other.sharedCert.Hash())

	rs, changes, err := m.IdentityRecords(f.user)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rs.Shared, []SharedKeyCert{f.sharedCert}) {
		t.Errorf("shared-key certificates: %+v", rs.Shared)
	}
	if len(rs.Machines) != 2 {
		t.Errorf("machine certificates: got %d, want the 2 that decode", len(rs.Machines))
	}
	if !reflect.DeepEqual(rs.Removals, []Removal{f.removal}) {
		t.Errorf("removals: %+v", rs.Removals)
	}
	if !reflect.DeepEqual(changes, []ListChange{f.change}) {
		t.Errorf("list changes: %+v", changes)
	}

	s := Resolve(f.user, rs)
	if len(s.Discarded) != 0 {
		t.Errorf("records read back from the store do not verify: %+v", s.Discarded)
	}
	if ms := s.Machines[f.machineCert.Machine]; ms == nil || !ms.Current || ms.Removed {
		t.Errorf("the machine read back from the store is not current: %+v", ms)
	}
	if ms := s.Machines[f.cloudCert.Machine]; ms == nil || !ms.Removed || ms.Cutoffs["room-a"] != 4 {
		t.Errorf("the removal read back from the store did not take effect: %+v", ms)
	}
	if merged := MergeListChanges(f.user, changes); merged[f.change.Entry].Value != "carol" {
		t.Errorf("the list change read back from the store does not merge: %+v", merged)
	}

	users, err := m.IdentityUsers()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{f.user, other.user}
	if want[0] > want[1] {
		want[0], want[1] = want[1], want[0]
	}
	if !reflect.DeepEqual(users, want) {
		t.Errorf("users: got %v, want %v", users, want)
	}

	var stored string
	h := f.sharedCert.Hash()
	if err := m.db.QueryRow(`SELECT hash FROM identity_records WHERE kind = 'shared' AND user = ?`, f.user).Scan(&stored); err != nil || stored != hex.EncodeToString(h[:]) {
		t.Errorf("a record is not keyed on the hex of its hash: %q", stored)
	}
}

// A machine's address is kept per machine, and a machine stays with the user it
// was recorded for.
func TestAMachinesAddressIsKeptPerMachine(t *testing.T) {
	m := testMembership(t)
	user, laptop, desktop := testIdentity(t).PeerID, testIdentity(t).PeerID, testIdentity(t).PeerID
	if err := m.SetMachineEndpoint(laptop, user, "laptop.example:7777"); err != nil {
		t.Fatal(err)
	}
	if err := m.SetMachineEndpoint(desktop, user, "desktop.example:7777"); err != nil {
		t.Fatal(err)
	}
	if err := m.SetMachineEndpoint(laptop, user, "laptop.example:9999"); err != nil {
		t.Fatal(err)
	}
	if e, at := m.MachineEndpoint(laptop); e != "laptop.example:9999" || at == "" {
		t.Errorf("laptop's address: %q at %q", e, at)
	}
	got, err := m.UserMachines(user)
	if err != nil || len(got) != 2 {
		t.Fatalf("the user's machines: %+v %v", got, err)
	}
	if err := m.SetMachineEndpoint(laptop, testIdentity(t).PeerID, "elsewhere:1"); err == nil {
		t.Error("a machine's address was accepted under another user")
	}
	if e, _ := m.MachineEndpoint(laptop); e != "laptop.example:9999" {
		t.Errorf("an address offered under another user replaced the machine's: %q", e)
	}
}

// The cached guest list is the list ResolveGuests computed, as far as Counts can
// tell, and caching a room again replaces its list rather than adding to it.
func TestACachedGuestListCountsAsTheListItWasComputedAs(t *testing.T) {
	m := testMembership(t)
	g := newGuestRoom("alice")
	g.invite("alice", 2, "bob")
	g.invite("alice", 3, "carol")
	g.revoke("alice", 4, "bob", map[string]int64{"bob/laptop": 5, "bob/desktop": 2})
	g.revoke("alice", 5, "carol", nil)
	list := g.resolve()
	if err := m.CacheGuestList(list); err != nil {
		t.Fatal(err)
	}

	got, err := m.CachedGuestList(g.room)
	if err != nil || got == nil {
		t.Fatalf("no cached list: %v", err)
	}
	if got.Room != g.room || !reflect.DeepEqual(got.Admitted, list.Admitted) {
		t.Errorf("admitted:\n got %v\nwant %v", got.Admitted, list.Admitted)
	}
	if !reflect.DeepEqual(got.Cutoffs["bob"], list.Cutoffs["bob"]) {
		t.Errorf("bob's cutoffs:\n got %v\nwant %v", got.Cutoffs["bob"], list.Cutoffs["bob"])
	}
	for _, c := range []struct {
		user, machine string
		seq           int64
	}{
		{"alice", "alice/laptop", 99}, {"bob", "bob/laptop", 5}, {"bob", "bob/laptop", 6},
		{"bob", "bob/desktop", 2}, {"bob", "bob/desktop", 3}, {"carol", "carol/laptop", 1},
		{"dave", "dave/laptop", 1},
	} {
		if got.Counts(c.user, c.machine, c.seq) != list.Counts(c.user, c.machine, c.seq) {
			t.Errorf("the cached list and the computed one disagree on %s's event %d from %s",
				c.user, c.seq, c.machine)
		}
	}

	only := newGuestRoom("alice")
	only.room = g.room
	if err := m.CacheGuestList(only.resolve()); err != nil {
		t.Fatal(err)
	}
	got, err = m.CachedGuestList(g.room)
	if err != nil || got == nil {
		t.Fatalf("no cached list: %v", err)
	}
	if len(got.Admitted) != 1 || len(got.Cutoffs) != 0 {
		t.Errorf("caching the room again kept rows of the old list: %+v", got)
	}

	if none, err := m.CachedGuestList("room-never-cached"); err != nil || none != nil {
		t.Errorf("a room never cached came back as %+v, %v", none, err)
	}
}

// The events table as it was before the signer column, copied here for the same
// reason as the membership schema above.
const eventsSchemaBeforeSigner = `
CREATE TABLE events (
  rowid_alias       INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id          TEXT NOT NULL UNIQUE,
  peer_id           TEXT NOT NULL,
  peer_sequence     INTEGER NOT NULL,
  room_id           TEXT NOT NULL,
  timestamp         TEXT NOT NULL,
  user_id           TEXT,
  user_display_name TEXT,
  machine_id        TEXT,
  origin_session_id TEXT,
  event_type        TEXT NOT NULL,
  content           TEXT,
  metadata          TEXT,
  signature         TEXT,
  sig_version       INTEGER,
  UNIQUE(peer_id, peer_sequence)
);
CREATE INDEX idx_events_room ON events(room_id, rowid_alias);`

// A room's database made before the signer column opens, gains the column and its
// index, and gives every event it held its peer as signer, while each event still
// verifies under the scheme that signed it (D-058, signature schemes are kept,
// never replaced).
func TestARoomStoreFromBeforeTheSignerColumnMigrates(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("COGMER_HOME", "")
	dir := filepath.Join(homeDir(), "rooms")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	id := testIdentity(t)
	ev := signed(t, id, "written before the signer column")
	ev.RoomID = "room-a"
	ev.Sign(id.private)

	old, err := sql.Open("sqlite", filepath.Join(dir, "room-a.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.Exec(eventsSchemaBeforeSigner); err != nil {
		t.Fatal(err)
	}
	if _, err := old.Exec(`INSERT INTO events (event_id, peer_id, peer_sequence, room_id, timestamp, user_id,
		user_display_name, machine_id, origin_session_id, event_type, content, metadata, signature, sig_version)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		ev.EventID, ev.PeerID, ev.PeerSequence, ev.RoomID, ev.Timestamp, ev.UserID, ev.UserDisplayName,
		ev.MachineID, ev.OriginSessionID, ev.EventType, ev.Content, "", ev.Signature, ev.SigVersion); err != nil {
		t.Fatal(err)
	}
	old.Close()

	s, err := OpenStore("room-a")
	if err != nil {
		t.Fatalf("opening a room from before the signer column: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	// Read the column itself: the read path falls back to the peer, so only the
	// stored value shows whether the migration filled it.
	var signer sql.NullString
	if err := s.db.QueryRow(`SELECT signer FROM events WHERE event_id = ?`, ev.EventID).Scan(&signer); err != nil {
		t.Fatalf("the migrated room has no signer column: %v", err)
	}
	if signer.String != id.PeerID {
		t.Errorf("an old event's stored signer is %q, want its peer", signer.String)
	}
	var idx int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM pragma_index_list('events') WHERE name = 'idx_events_signer'`).Scan(&idx); err != nil || idx != 1 {
		t.Error("the migrated room has no index on (signer, peer_sequence)")
	}

	got, err := s.ListRoom("room-a")
	if err != nil || len(got) != 1 {
		t.Fatalf("the old event did not survive: %+v %v", got, err)
	}
	if got[0].Signer != id.PeerID {
		t.Errorf("an old event reads back with signer %q, want its peer", got[0].Signer)
	}
	if err := got[0].Verify(); err != nil {
		t.Errorf("an old event no longer verifies after the migration: %v", err)
	}

	// An event inserted now stores its peer as signer, whatever it claims.
	next := signed(t, id, "written after")
	next.EventID, next.PeerSequence, next.RoomID = "evt-2", 2, "room-a"
	next.Sign(id.private)
	next.Signer = "ed25519:somebody-else"
	if _, err := s.Insert(next); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT signer FROM events WHERE event_id = 'evt-2'`).Scan(&signer); err != nil || signer.String != id.PeerID {
		t.Errorf("a new event's stored signer is %q, want its peer", signer.String)
	}
}

// Signer is outside v3's signed fields, so an event verifies whatever it says.
// Adding it to the v3 bytes would make every event already signed fail to verify.
func TestTheSignerFieldIsNotSignedUnderV3(t *testing.T) {
	id := testIdentity(t)
	e := signed(t, id, "content")
	e.Signer = "ed25519:anything"
	if err := e.Verify(); err != nil {
		t.Errorf("setting Signer broke a v3 signature: %v", err)
	}
}

// shared.key is readable by this account alone, comes back as it was written, and
// is absent until written.
func TestTheSharedKeyIsKeptInAFileOfItsOwn(t *testing.T) {
	t.Setenv("COGMER_HOME", t.TempDir())
	if _, _, ok, err := loadSharedKey(); ok || err != nil {
		t.Fatalf("a machine with no shared key reported one: %v %v", ok, err)
	}
	_, priv, _ := ed25519.GenerateKey(nil)
	if err := saveSharedKey(priv, 3); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(sharedKeyPath())
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("shared.key is %o, want 600", perm)
	}
	got, version, ok, err := loadSharedKey()
	if err != nil || !ok || version != 3 || !got.Equal(priv) {
		t.Errorf("shared.key read back as version %d, ok %v, err %v, same key %v", version, ok, err, got.Equal(priv))
	}
	if err := saveSharedKey(priv, 0); err == nil {
		t.Error("a shared key of version 0 was saved")
	}
}

// keyEncodings are the forms in which a private key could leak into a file or an
// output: the whole key and its seed, in each encoding a program might choose.
func keyEncodings(priv ed25519.PrivateKey) []string {
	var out []string
	for _, b := range [][]byte{priv, priv.Seed()} {
		out = append(out,
			base64.RawURLEncoding.EncodeToString(b), base64.RawStdEncoding.EncodeToString(b),
			base64.URLEncoding.EncodeToString(b), base64.StdEncoding.EncodeToString(b),
			hex.EncodeToString(b))
	}
	return out
}

// The shared key's private half never reaches identity.json, whoami's output or any
// JSON the identity marshals to, even once the identity carries the user and the
// chain, since a user may hand all three to a colleague (D-153, the private key is
// kept in a file of its own).
func TestTheSharedKeyNeverAppearsInTheIdentity(t *testing.T) {
	home := t.TempDir()
	t.Setenv("COGMER_HOME", home)
	t.Setenv("HOME", home)
	t.Setenv("COGMER_ADDR", "127.0.0.1:1")
	t.Setenv("COGMER_PEER_ENDPOINT", "")
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")

	f := newIdentityFixture(t)
	if err := saveSharedKey(f.shared, 1); err != nil {
		t.Fatal(err)
	}
	id, err := LoadIdentity()
	if err != nil {
		t.Fatal(err)
	}
	machine := CertifyMachine(f.shared, f.user, 1, id.private.Public().(ed25519.PublicKey), KindMachine, "laptop", 1)
	id.User = f.user
	id.Chain = &Chain{Shared: f.sharedCert, Machine: machine}
	buf, err := json.MarshalIndent(id, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "identity.json"), buf, 0o600); err != nil {
		t.Fatal(err)
	}
	reloaded, err := LoadIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.User != f.user || reloaded.Chain == nil || reloaded.Chain.Machine.Machine != id.PeerID {
		t.Fatalf("identity.json did not keep the user and the chain: %+v", reloaded)
	}
	marshalled, err := json.Marshal(reloaded)
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.ReadFile(filepath.Join(home, "identity.json"))
	if err != nil {
		t.Fatal(err)
	}

	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	runWhoami()
	w.Close()
	os.Stdout = stdout
	printed, _ := io.ReadAll(r)
	if !strings.Contains(string(printed), f.user) {
		t.Errorf("whoami does not print the user it holds:\n%s", printed)
	}

	for name, text := range map[string]string{
		"identity.json": string(file), "whoami": string(printed), "the identity's JSON": string(marshalled),
	} {
		for _, enc := range keyEncodings(f.shared) {
			if strings.Contains(text, enc) {
				t.Errorf("%s carries the shared private key", name)
			}
		}
		for _, enc := range keyEncodings(id.private) {
			if strings.Contains(text, enc) {
				t.Errorf("%s carries the machine's private key", name)
			}
		}
	}
}
