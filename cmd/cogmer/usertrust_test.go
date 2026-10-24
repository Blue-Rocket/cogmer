package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
)

// The trust slice of the user identity: a machine speaks for its user, and a
// colleague who verified the user once trusts every machine of theirs (D-219, a user
// is a master key that certifies machines through a shared key).

// testUser is a user made as the identity page will make one: a recovery phrase, the
// master key derived from it, a shared key under version 1, and a certificate for
// each machine the user adds.
type testUser struct {
	master  ed25519.PrivateKey
	id      string
	shared  ed25519.PrivateKey
	version uint64
	serial  uint64
}

func newTestUser(t *testing.T) *testUser {
	t.Helper()
	p, err := NewPhrase()
	if err != nil {
		t.Fatal(err)
	}
	_, shared, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &testUser{master: p.MasterKey(), id: p.UserID(), shared: shared, version: 1}
}

func (u *testUser) sharedCert() SharedKeyCert {
	return CertifySharedKey(u.master, u.version, u.shared.Public().(ed25519.PublicKey))
}

// chainFor certifies a machine key under the user's current shared key.
func (u *testUser) chainFor(pub ed25519.PublicKey, name string) Chain {
	u.serial++
	return Chain{Shared: u.sharedCert(), Machine: CertifyMachine(u.shared, u.id, u.version, pub, KindMachine, name, u.serial)}
}

// machine makes one machine of the user: a key of its own and an identity holding the
// user and the chain, as identity.json will.
func (u *testUser) machine(t *testing.T, name string) *Identity {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	chain := u.chainFor(pub, name)
	id := &Identity{PeerID: PeerIDFromPublic(pub), UserDisplayName: name, private: priv, User: u.id, Chain: &chain}
	id.PeerName = PeerName(id.PeerID)
	return id
}

// rotate has the master certify a new shared key, as after a lost machine (D-234, a machine not certified under the current shared key is suspended).
func (u *testUser) rotate(t *testing.T) {
	t.Helper()
	_, shared, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	u.shared, u.version = shared, u.version+1
}

// newUserPeer is newOfflinePeer for a given identity, with the shared key written to
// shared.key when the identity has a user, as on a machine the user added.
func newUserPeer(t *testing.T, roomID, roomName string, id *Identity, shared *testUser) *offlinePeer {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	m, err := OpenMembership()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	if err := m.RecordRoom(roomID, roomName, false); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(roomID)
	if err != nil {
		t.Fatal(err)
	}
	if shared != nil {
		if err := saveSharedKey(shared.shared, shared.version); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.Invite(roomID, id.Peer()); err != nil {
		t.Fatal(err)
	}
	d := &Daemon{id: id, members: m, stores: map[string]*Store{roomID: store}}
	d.askRooms = func(_ string, rooms []Room) []Room { return rooms }
	t.Cleanup(d.closeStores)
	up := &atomic.Bool{}
	up.Store(true)
	addr := servePeerTLSHandler(t, d, d.PeerRoutes())
	return &offlinePeer{d: d, store: store, addr: addr, up: up}
}

// introduceUsers is introduce by user: each side records the other's user, verified
// and a guest of the room, as a pairing and an invitation leave it.
func introduceUsers(t *testing.T, a, b *offlinePeer, roomID string) {
	t.Helper()
	for _, pair := range []struct{ from, to *offlinePeer }{{a, b}, {b, a}} {
		peer := pair.to.d.id.Peer()
		if err := pair.from.d.members.Allow(peer, ""); err != nil {
			t.Fatal(err)
		}
		if err := pair.from.d.members.MarkVerified(peer); err != nil {
			t.Fatal(err)
		}
		if err := pair.from.d.members.Invite(roomID, peer); err != nil {
			t.Fatal(err)
		}
	}
}

// copyColleague records on one of a user's machines the colleague the user verified
// on another, as copying the user's lists will (D-228, a user's lists are copied between the user's machines).
func copyColleague(t *testing.T, machine, colleague *offlinePeer, roomID string) {
	t.Helper()
	peer := colleague.d.id.Peer()
	if err := machine.d.members.Allow(peer, ""); err != nil {
		t.Fatal(err)
	}
	if err := machine.d.members.MarkVerified(peer); err != nil {
		t.Fatal(err)
	}
	if err := machine.d.members.Invite(roomID, peer); err != nil {
		t.Fatal(err)
	}
}

func heldEvents(t *testing.T, p *offlinePeer, roomID string) []Event {
	t.Helper()
	evs, err := p.store.ListRoom(roomID)
	if err != nil {
		t.Fatal(err)
	}
	return evs
}

func contents(evs []Event) string {
	var out []string
	for _, e := range evs {
		out = append(out, e.Content)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

// carry copies a machine's events into another store, as a machine that received
// them earlier holds them and relays them (§13).
func carry(t *testing.T, from, to *offlinePeer, roomID string) {
	t.Helper()
	for _, e := range heldEvents(t, from, roomID) {
		ev := e
		if _, err := to.store.Insert(&ev); err != nil {
			t.Fatal(err)
		}
	}
}

const trustRoom, trustRoomName = "room-trust", "quiet-orchard"

// --- frozen v4 events ---

type frozenV4 struct {
	Keys     struct{ Master, Shared, Machine, Cloud, Session string } `json:"keys"`
	Removal  Removal                                                  `json:"removal"`
	EventsV4 []wireEvent                                              `json:"eventsV4"`
}

func loadV4Vectors(t *testing.T) frozenV4 {
	t.Helper()
	raw, err := os.ReadFile("testdata/identity_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v frozenV4
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	if len(v.EventsV4) != 2 {
		t.Fatalf("the frozen v4 events are missing: %d", len(v.EventsV4))
	}
	return v
}

// Two events signed under v4 from the fixed seeds, one by a machine key with a
// removal seen and one by a cloud session's key, still verify. Signing and verifying a
// fresh event cannot see an edit to the signing bytes; a frozen one can (D-058,
// signature schemes are kept, never replaced).
func TestEventsSignedForAUserBeforeAnyEditStillVerify(t *testing.T) {
	v := loadV4Vectors(t)
	for _, w := range v.EventsV4 {
		e := fromWire(w)
		if e.SigVersion != 4 || e.PeerID != v.Keys.Master {
			t.Fatalf("frozen event %s is not a v4 event for the frozen user", e.EventID)
		}
		if err := e.Verify(); err != nil {
			t.Errorf("frozen v4 event %s no longer verifies: %v. The bytes v4 signs have changed; add a scheme and leave v4 as it was (D-058, signature schemes are kept, never replaced)", e.EventID, err)
		}
	}
	if s := fromWire(v.EventsV4[0]).Signer; s != v.Keys.Machine {
		t.Errorf("the first frozen event is signed by %s, not the machine", s)
	}
	if s := fromWire(v.EventsV4[1]).Signer; s != v.Keys.Session {
		t.Errorf("the second frozen event is signed by %s, not the session", s)
	}

	// The encoding the design states, written out without the code's helpers: v3's
	// fields under the v4 tag, then the signer, the version and the removals seen.
	e := fromWire(v.EventsV4[0])
	var b []byte
	str := func(s string) { b = append(binary.BigEndian.AppendUint32(b, uint32(len(s))), s...) }
	num := func(n uint64) { b = binary.BigEndian.AppendUint64(b, n) }
	str("peer-room/event/v4")
	for _, f := range []string{e.EventID, e.PeerID} {
		str(f)
	}
	num(uint64(e.PeerSequence))
	for _, f := range []string{e.RoomID, e.Timestamp, e.UserID, e.UserDisplayName, e.MachineID,
		e.OriginSessionID, e.EventType, e.Content, string(e.Metadata)} {
		str(f)
	}
	str(v.Keys.Machine)
	num(3)
	num(1)
	h := v.Removal.Hash()
	str(hex.EncodeToString(h[:]))
	sig, _ := b64url(e.Signature)
	pub, _ := PublicFromPeerID(v.Keys.Machine)
	if !ed25519.Verify(pub, b, sig) {
		t.Error("the frozen v4 event is not signed over the fields in the order the design gives")
	}
}

// Every field v4 adds is signed, and the chain proves itself, so altering any of
// them, or carrying the event under another user, refuses it.
func TestAChangeToAFrozenV4EventFailsVerification(t *testing.T) {
	v := loadV4Vectors(t)
	k := testIDKeys(t)
	for name, alter := range map[string]func(*Event){
		"content":          func(e *Event) { e.Content += "x" },
		"signer":           func(e *Event) { e.Signer = k.other.id },
		"identity version": func(e *Event) { e.IdentityVersion++ },
		"removals seen":    func(e *Event) { e.RemovalsSeen = nil },
		"no chain":         func(e *Event) { e.Chain = nil },
		"another user":     func(e *Event) { e.PeerID = k.other.id },
		"chain of another machine": func(e *Event) {
			c := *e.Chain
			c.Machine = CertifyMachine(k.shared.priv, k.master.id, 3, k.other.pub, KindMachine, "m", 1)
			e.Chain = &c
		},
	} {
		e := fromWire(v.EventsV4[0])
		alter(&e)
		if e.Verify() == nil {
			t.Errorf("a frozen v4 event with its %s altered verified", name)
		}
	}
	e := fromWire(v.EventsV4[0])
	e.SigVersion = 9
	if err := e.Verify(); err == nil || !strings.Contains(err.Error(), "upgrade") {
		t.Errorf("an unknown scheme is not refused as a version problem: %v", err)
	}
}

// --- a machine with no user identity is unchanged ---

// A machine with no user identity signs under v3, its events verify and are stored
// with their peer as signer and nothing of v4, its requests carry no chain and its
// certificate no extension, and two such machines pair and sync as before.
func TestAMachineOnlyPairingAndSyncBehaveAsBefore(t *testing.T) {
	a := newOfflinePeer(t, trustRoom, trustRoomName)
	b := newOfflinePeer(t, trustRoom, trustRoomName)
	introduce(t, a, b, trustRoom)
	if a.d.id.HasUser() || a.d.id.Peer() != a.d.id.PeerID {
		t.Fatal("a test identity has a user")
	}
	a.say(t, trustRoom, "s", EventUserPrompt, "from a")
	b.say(t, trustRoom, "s", EventUserPrompt, "from b")
	settle(t, a, b, Room{RoomID: trustRoom})

	for _, p := range []*offlinePeer{a, b} {
		evs := heldEvents(t, p, trustRoom)
		if contents(evs) != "from a,from b" {
			t.Fatalf("the room holds %q", contents(evs))
		}
		for _, e := range evs {
			if e.SigVersion != 3 || e.Signer != e.PeerID || e.Chain != nil || e.IdentityVersion != 0 || len(e.RemovalsSeen) != 0 {
				t.Errorf("a machine-only event reads back as %+v", e)
			}
			if err := e.Verify(); err != nil {
				t.Errorf("a machine-only event does not verify: %v", err)
			}
		}
		var chains int
		if err := p.store.db.QueryRow(`SELECT COUNT(*) FROM events WHERE chain IS NOT NULL OR identity_version IS NOT NULL OR removals_seen IS NOT NULL`).Scan(&chains); err != nil || chains != 0 {
			t.Errorf("a machine-only room stores v4 fields: %d, %v", chains, err)
		}
	}
	for _, w := range loadVectors(t).Events {
		if e := fromWire(w); e.Verify() != nil {
			t.Error("a frozen v3 event no longer verifies")
		}
	}

	cred, err := signRequestAs(a.d.id, trustRoom, "")
	if err != nil || cred.Chain != nil || cred.Signer != "" || cred.PeerID != a.d.id.PeerID {
		t.Errorf("a machine-only request is %+v, %v", cred, err)
	}
	cert, err := peerCert(a.d.id)
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := x509.ParseCertificate(cert.Certificate[0])
	if c, _ := certChain(parsed); c != nil {
		t.Error("a machine-only certificate carries a chain")
	}
	// The address a request signs is recorded against the peer, which is one
	// machine: its key is both the machine and the user.
	ms, err := a.d.members.UserMachines(b.d.id.PeerID)
	if err != nil || len(ms) != 1 || ms[0].MachineKey != b.d.id.PeerID {
		t.Errorf("the peer's address is recorded as %+v, %v", ms, err)
	}
}

// --- events signed for a user ---

// A colleague verified the user once, on the user's first machine. A machine the user
// adds later syncs with the colleague in both directions, and its turns arrive as the
// user's, with no verification of the new machine.
func TestAMachineTheUserAddedLaterSyncsWithAColleagueWhoVerifiedOnlyTheFirst(t *testing.T) {
	ana, ben := newTestUser(t), newTestUser(t)
	first := newUserPeer(t, trustRoom, trustRoomName, ana.machine(t, "laptop"), ana)
	colleague := newUserPeer(t, trustRoom, trustRoomName, ben.machine(t, "desktop"), ben)
	introduceUsers(t, first, colleague, trustRoom)
	first.say(t, trustRoom, "s1", EventUserPrompt, "from the first machine")
	settle(t, first, colleague, Room{RoomID: trustRoom})

	added := newUserPeer(t, trustRoom, trustRoomName, ana.machine(t, "tablet"), ana)
	// The user's colleagues and verifications are the user's and are copied to every
	// machine the user adds (D-228, a user's colleagues, verifications, labels and
	// rooms belong to the user), which a later slice builds. Recorded here as that
	// copy will leave them. The colleague's side is untouched: it verified the user
	// once, on the first machine.
	copyColleague(t, added, colleague, trustRoom)
	added.say(t, trustRoom, "s2", EventUserPrompt, "from the added machine")
	colleague.say(t, trustRoom, "s3", EventUserPrompt, "from the colleague")

	if n, err := colleague.d.pullFrom(added.addr); err != nil || n != 1 {
		t.Fatalf("the colleague pulled %d from the added machine: %v", n, err)
	}
	if n, err := added.d.pullFrom(colleague.addr); err != nil || n < 1 {
		t.Fatalf("the added machine pulled %d from the colleague: %v", n, err)
	}
	var fromAdded *Event
	for _, e := range heldEvents(t, colleague, trustRoom) {
		if e.Content == "from the added machine" {
			ev := e
			fromAdded = &ev
		}
	}
	if fromAdded == nil {
		t.Fatal("the added machine's turn did not reach the colleague")
	}
	if fromAdded.PeerID != ana.id || fromAdded.Signer != added.d.id.PeerID || fromAdded.SigVersion != 4 || fromAdded.Chain == nil {
		t.Errorf("the turn is stored as %+v", fromAdded)
	}
	// Both machines of the user number their turns from 1, and both are held.
	if contents(heldEvents(t, colleague, trustRoom)) != "from the added machine,from the colleague,from the first machine" {
		t.Errorf("the colleague holds %q", contents(heldEvents(t, colleague, trustRoom)))
	}
	if colleague.d.members.Knows(added.d.id.PeerID) || colleague.d.members.IsVerified(added.d.id.PeerID) {
		t.Error("the added machine was recorded as a peer of its own")
	}
	if got := onlyVerified(heldEvents(t, colleague, trustRoom), colleague.d.id.Peer(), colleague.d.authorVerified); len(got) != 3 {
		t.Errorf("injection keeps %d of 3 turns", len(got))
	}
	// The colleague learned where the added machine listens, against that machine.
	if ms, _ := colleague.d.members.UserMachines(ana.id); len(ms) == 0 {
		t.Error("the colleague recorded no machine of the user")
	}
	if !strings.Contains(contents(heldEvents(t, added, trustRoom)), "from the colleague") {
		t.Error("the colleague's turn did not reach the added machine")
	}
}

// A chain that ends at a master nobody here verified is refused: its events are not
// stored and its requests are not served, though the key is recorded.
func TestAChainToAnUnverifiedMasterIsRefused(t *testing.T) {
	ana, ben := newTestUser(t), newTestUser(t)
	a := newUserPeer(t, trustRoom, trustRoomName, ana.machine(t, "laptop"), ana)
	b := newUserPeer(t, trustRoom, trustRoomName, ben.machine(t, "desktop"), ben)
	for _, pair := range []struct{ from, to *offlinePeer }{{a, b}, {b, a}} {
		if err := pair.from.d.members.Allow(pair.to.d.id.Peer(), ""); err != nil {
			t.Fatal(err)
		}
		if err := pair.from.d.members.Invite(trustRoom, pair.to.d.id.Peer()); err != nil {
			t.Fatal(err)
		}
	}
	a.say(t, trustRoom, "s", EventUserPrompt, "unverified")
	if n, _ := b.d.pullFrom(a.addr); n != 0 {
		t.Errorf("%d event(s) of an unverified user were stored", n)
	}
	if len(heldEvents(t, b, trustRoom)) != 0 {
		t.Error("an unverified user's turn is in the room")
	}
	cred, err := signRequestAs(a.d.id, trustRoom, "")
	if err != nil {
		t.Fatal(err)
	}
	req := syncRequest{RoomID: trustRoom, PeerID: cred.PeerID, Timestamp: cred.Timestamp, Nonce: cred.Nonce,
		Signature: cred.Signature, Signer: cred.Signer, Chain: cred.Chain}
	if err := b.d.verifyRequest(req, trustRoom); err == nil || !strings.Contains(err.Error(), "UNVERIFIED") {
		t.Errorf("a request for an unverified user was not refused as unverified: %v", err)
	}

	// Verified, the same backlog arrives on the next pull (D-054, verification gates
	// synchronization and injection).
	if err := b.d.members.MarkVerified(ana.id); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.MarkVerified(ben.id); err != nil {
		t.Fatal(err)
	}
	if n, err := b.d.pullFrom(a.addr); err != nil || n != 1 {
		t.Errorf("after verification %d arrived: %v", n, err)
	}
}

// A request signed by one machine and presented under another user's identifier, or
// with a chain that ends at another machine, is refused.
func TestARequestWhoseChainDoesNotEndAtItsSignerIsRefused(t *testing.T) {
	ana, ben := newTestUser(t), newTestUser(t)
	a := newUserPeer(t, trustRoom, trustRoomName, ana.machine(t, "laptop"), ana)
	b := newUserPeer(t, trustRoom, trustRoomName, ben.machine(t, "desktop"), ben)
	introduceUsers(t, a, b, trustRoom)
	other := ana.machine(t, "other")
	for name, alter := range map[string]func(*syncRequest){
		"chain of another machine": func(r *syncRequest) { r.Chain = other.Chain },
		"another user named":       func(r *syncRequest) { r.PeerID = ben.id },
		"signer swapped":           func(r *syncRequest) { r.Signer = other.PeerID },
	} {
		cred, err := signRequestAs(a.d.id, trustRoom, "")
		if err != nil {
			t.Fatal(err)
		}
		req := syncRequest{RoomID: trustRoom, PeerID: cred.PeerID, Timestamp: cred.Timestamp, Nonce: cred.Nonce,
			Signature: cred.Signature, Signer: cred.Signer, Chain: cred.Chain}
		alter(&req)
		if b.d.verifyRequest(req, trustRoom) == nil {
			t.Errorf("%s: the request was served", name)
		}
	}
	cred, _ := signRequestAs(a.d.id, trustRoom, "")
	req := syncRequest{RoomID: trustRoom, PeerID: cred.PeerID, Timestamp: cred.Timestamp, Nonce: cred.Nonce,
		Signature: cred.Signature, Signer: cred.Signer, Chain: cred.Chain}
	if err := b.d.verifyRequest(req, trustRoom); err != nil {
		t.Errorf("the unaltered request was refused: %v", err)
	}
}

// A removed machine's turns up to its cutoff in a room count and later ones do not,
// so a colleague holding the removal stores the first and not the rest, even relayed
// by another machine (D-220, a removal names the last event of a machine that counts).
func TestARemovedMachinesEventPastItsCutoffIsNotStoredAndWithinItIs(t *testing.T) {
	ana, ben := newTestUser(t), newTestUser(t)
	relay := newUserPeer(t, trustRoom, trustRoomName, ana.machine(t, "laptop"), ana)
	stolen := newUserPeer(t, trustRoom, trustRoomName, ana.machine(t, "stolen"), ana)
	colleague := newUserPeer(t, trustRoom, trustRoomName, ben.machine(t, "desktop"), ben)
	introduceUsers(t, relay, colleague, trustRoom)
	for _, s := range []string{"one", "two", "three"} {
		stolen.say(t, trustRoom, "s", EventUserPrompt, s)
	}
	carry(t, stolen, relay, trustRoom)

	removal, err := SignRemoval(ana.shared, LevelShared, ana.version, ana.id, stolen.d.id.PeerID,
		[]RoomCutoff{{Room: trustRoom, Sequence: 2}})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(removal)
	if _, _, err := colleague.d.members.PutIdentityRecord(RecordRemoval, ana.id, raw); err != nil {
		t.Fatal(err)
	}
	colleague.d.pullFrom(relay.addr)
	if got := contents(heldEvents(t, colleague, trustRoom)); got != "one,two" {
		t.Errorf("the colleague holds %q; want the two turns within the cutoff", got)
	}

	// The removed machine is refused when it asks for itself.
	cred, _ := signRequestAs(stolen.d.id, trustRoom, "")
	req := syncRequest{RoomID: trustRoom, PeerID: cred.PeerID, Timestamp: cred.Timestamp, Nonce: cred.Nonce,
		Signature: cred.Signature, Signer: cred.Signer, Chain: cred.Chain}
	if err := colleague.d.verifyRequest(req, trustRoom); err == nil || !strings.Contains(err.Error(), "removed") {
		t.Errorf("a removed machine's request was not refused for its removal: %v", err)
	}
}

// After the user certifies a new shared key, a machine not certified under it is
// suspended: its turns are not stored, even relayed, and its own requests are refused
// (D-234, a machine not certified under the current shared key is suspended).
func TestASuspendedMachinesEventIsNotStoredAndItsSyncRequestIsRefused(t *testing.T) {
	ana, ben := newTestUser(t), newTestUser(t)
	oldID := ana.machine(t, "old")
	ana.rotate(t)
	keptID := ana.machine(t, "kept")
	kept := newUserPeer(t, trustRoom, trustRoomName, keptID, ana)
	old := newUserPeer(t, trustRoom, trustRoomName, oldID, nil)
	colleague := newUserPeer(t, trustRoom, trustRoomName, ben.machine(t, "desktop"), ben)
	introduceUsers(t, kept, colleague, trustRoom)

	old.say(t, trustRoom, "s", EventUserPrompt, "from the suspended machine")
	kept.say(t, trustRoom, "s", EventUserPrompt, "from the kept machine")
	carry(t, old, kept, trustRoom)

	// The kept machine asks first, which tells the colleague of shared key 2.
	if _, err := kept.d.pullFrom(colleague.addr); err != nil {
		t.Fatal(err)
	}
	colleague.d.pullFrom(kept.addr)
	if got := contents(heldEvents(t, colleague, trustRoom)); got != "from the kept machine" {
		t.Errorf("the colleague holds %q; want the kept machine's turn alone", got)
	}

	cred, _ := signRequestAs(old.d.id, trustRoom, "")
	req := syncRequest{RoomID: trustRoom, PeerID: cred.PeerID, Timestamp: cred.Timestamp, Nonce: cred.Nonce,
		Signature: cred.Signature, Signer: cred.Signer, Chain: cred.Chain}
	if err := colleague.d.verifyRequest(req, trustRoom); err == nil || !strings.Contains(err.Error(), "suspended") {
		t.Errorf("a suspended machine's request was not refused for its suspension: %v", err)
	}
	if _, err := colleague.d.pullFrom(old.addr); err == nil {
		t.Error("the colleague completed a connection to a suspended machine")
	}
}

// --- TLS ---

func userDaemon(t *testing.T, id *Identity) *Daemon {
	t.Helper()
	d := &Daemon{id: id, members: testMembership(t)}
	t.Cleanup(d.closeStores)
	return d
}

// A handshake from a machine whose chain ends at a recorded user completes, though
// the machine's own key was never recorded, and one from a machine whose chain ends
// at a stranger does not (D-101, peer connections are TLS pinned to a key this
// machine has recorded).
func TestATLSHandshakeFromAMachineOfARecordedUserSucceedsAndOfAStrangerFails(t *testing.T) {
	ana, ben, eve := newTestUser(t), newTestUser(t), newTestUser(t)
	server := userDaemon(t, ana.machine(t, "laptop"))
	colleague := userDaemon(t, ben.machine(t, "desktop"))
	stranger := userDaemon(t, eve.machine(t, "other"))
	for _, d := range []*Daemon{colleague, stranger} {
		if err := d.members.Allow(ana.id, ""); err != nil {
			t.Fatal(err)
		}
	}
	if err := server.members.Allow(ben.id, ""); err != nil {
		t.Fatal(err)
	}
	addr := servePeerTLS(t, server)

	if code, err := get(t, colleague, addr, ana.id); err != nil || code != http.StatusOK {
		t.Errorf("a machine of a recorded user could not connect: %d, %v", code, err)
	}
	if _, err := get(t, stranger, addr, ana.id); err == nil {
		t.Error("a machine whose chain ends at a stranger completed a handshake")
	}
	// Dialled for one named user, a machine of another is refused.
	if _, err := get(t, colleague, addr, ben.id); err == nil {
		t.Error("an address answering for another user was accepted")
	}
	// The user's own second machine connects though nobody recorded it.
	own := userDaemon(t, ana.machine(t, "tablet"))
	if code, err := get(t, own, addr, ""); err != nil || code != http.StatusOK {
		t.Errorf("the user's own machine could not connect: %d, %v", code, err)
	}
}

// A certificate's chain is a non-critical extension that parses back to the chain.
func TestACertificateCarriesItsMachinesChain(t *testing.T) {
	ana := newTestUser(t)
	id := ana.machine(t, "laptop")
	cert, err := peerCert(id)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("a certificate carrying a chain does not parse: %v", err)
	}
	c, err := certChain(parsed)
	if err != nil || c == nil {
		t.Fatalf("no chain read back: %v", err)
	}
	if _, err := c.Verify(ana.id, id.PeerID); err != nil {
		t.Errorf("the chain read back does not verify: %v", err)
	}
	for _, ext := range parsed.Extensions {
		if ext.Id.Equal(chainExtensionOID) && ext.Critical {
			t.Error("the chain extension is critical, so an older build would refuse the certificate")
		}
	}
}

// --- pairing by user ---

// Two users pair: each machine names its user, the two words are computed over the
// two users' identifiers and match, and the verification then covers a machine one of
// them adds later (D-048, verification is a live commit-then-reveal exchange; D-052, verifying a peer is its own act, run once).
func TestTwoUsersPairOverTheirUserIdentifiersAndTheVerificationCoversAThirdMachine(t *testing.T) {
	ana, ben := newTestUser(t), newTestUser(t)
	a := newUserPeer(t, trustRoom, trustRoomName, ana.machine(t, "laptop"), ana)
	b := newUserPeer(t, trustRoom, trustRoomName, ben.machine(t, "desktop"), ben)

	// What each sends the other is the user's identifier and this machine's address.
	peer, addr, _ := parsePairing(pairingString(a.d.id.Peer(), a.addr, ""))
	if peer != ana.id || addr != a.addr {
		t.Fatalf("the pairing string names %s at %s", peer, addr)
	}
	for _, p := range []struct {
		d          *Daemon
		user, addr string
	}{{a.d, ben.id, b.addr}, {b.d, ana.id, a.addr}} {
		if err := p.d.members.Allow(p.user, ""); err != nil {
			t.Fatal(err)
		}
		if err := p.d.members.SetPeerEndpoint(p.user, p.addr); err != nil {
			t.Fatal(err)
		}
	}

	type result struct {
		words string
		err   error
	}
	done := make(chan result, 2)
	go func() {
		w, err := a.d.RunVerification(ben.id, a.d.verifyTargets(ben.id))
		done <- result{w, err}
	}()
	go func() {
		w, err := b.d.RunVerification(ana.id, b.d.verifyTargets(ana.id))
		done <- result{w, err}
	}()
	first, second := <-done, <-done
	if first.err != nil || second.err != nil {
		t.Fatalf("the exchange failed: %v / %v", first.err, second.err)
	}
	if first.words != second.words || len(strings.Fields(first.words)) != sasWords {
		t.Fatalf("the two users saw %q and %q", first.words, second.words)
	}
	// Each side computed the words over its own user's identifier and the one it was
	// asked to verify, and they match, so both were computed over the two users'.
	if a.d.members.Knows(b.d.id.PeerID) || b.d.members.Knows(a.d.id.PeerID) {
		t.Error("a machine key was recorded as a peer")
	}
	for _, p := range []struct {
		d    *Daemon
		user string
	}{{a.d, ben.id}, {b.d, ana.id}} {
		if err := p.d.members.MarkVerified(p.user); err != nil {
			t.Fatal(err)
		}
		if err := p.d.members.Invite(trustRoom, p.user); err != nil {
			t.Fatal(err)
		}
	}

	third := newUserPeer(t, trustRoom, trustRoomName, ana.machine(t, "tablet"), ana)
	copyColleague(t, third, b, trustRoom)
	third.say(t, trustRoom, "s", EventUserPrompt, "from the third machine")
	if n, err := b.d.pullFrom(third.addr); err != nil || n != 1 {
		t.Fatalf("the colleague took %d turns from the third machine: %v", n, err)
	}
	if n, err := third.d.pullFrom(b.addr); err != nil {
		t.Fatalf("the third machine's request was refused: %d, %v", n, err)
	}
}

// A verification step signed by a machine and carrying a chain to another user, or no
// chain at all for a user's identifier, is refused.
func TestAVerificationStepMustBeSignedByAMachineOfTheUserItNames(t *testing.T) {
	ana, ben := newTestUser(t), newTestUser(t)
	a := userDaemon(t, ana.machine(t, "laptop"))
	b := userDaemon(t, ben.machine(t, "desktop"))
	msg := a.signedVerifyMessage("commit", []byte("payload"))
	payload, _ := b64url(msg.Payload)
	sig, _ := b64url(msg.Signature)
	if err := b.checkVerifyMessage(msg, payload, sig); err != nil {
		t.Fatalf("a step from a machine of the user it names was refused: %v", err)
	}
	stripped := msg
	stripped.Chain, stripped.Signer = nil, ""
	if b.checkVerifyMessage(stripped, payload, sig) == nil {
		t.Error("a step naming a user and signed by a machine passed with no chain")
	}
	other := msg
	other.Chain = b.id.Chain
	if b.checkVerifyMessage(other, payload, sig) == nil {
		t.Error("a step carrying another user's chain passed")
	}
}

// --- a user's own machines and addresses ---

// A user's own second machine is a sync target of the rooms the user is a guest of,
// this machine is not, and the second machine's turns arrive as the user's own, held
// beside this machine's under the same sequence numbers (D-103, an address belongs to
// a machine; D-222, any two of a user's machines exchange each room both joined).
func TestAUsersOwnSecondMachineIsASyncTargetAndItsEventsAreTheUsersOwn(t *testing.T) {
	t.Setenv("COGMER_PEERS", "")
	ana := newTestUser(t)
	one := newUserPeer(t, trustRoom, trustRoomName, ana.machine(t, "laptop"), ana)
	two := newUserPeer(t, trustRoom, trustRoomName, ana.machine(t, "tablet"), ana)
	if err := one.d.members.RecordMachineAddress(two.d.id.PeerID, ana.id, two.addr); err != nil {
		t.Fatal(err)
	}
	if err := one.d.members.RecordMachineAddress(one.d.id.PeerID, ana.id, "tc://this-machine"); err != nil {
		t.Fatal(err)
	}
	targets := one.d.syncTargets()
	if len(targets) != 1 || targets[0] != two.addr {
		t.Fatalf("the sync targets are %v; want the second machine alone", targets)
	}

	one.say(t, trustRoom, "s1", EventUserPrompt, "on the laptop")
	two.say(t, trustRoom, "s2", EventUserPrompt, "on the tablet")
	if n, err := one.d.pullFrom(two.addr); err != nil || n != 1 {
		t.Fatalf("the laptop took %d from the tablet: %v", n, err)
	}
	evs := heldEvents(t, one, trustRoom)
	if contents(evs) != "on the laptop,on the tablet" {
		t.Fatalf("the laptop holds %q", contents(evs))
	}
	for _, e := range evs {
		if e.PeerID != ana.id || e.PeerSequence != 1 {
			t.Errorf("a turn of the user is stored as %s sequence %d", PeerName(e.PeerID), e.PeerSequence)
		}
	}
	if got := onlyVerified(evs, one.d.id.Peer(), one.d.authorVerified); len(got) != 2 {
		t.Errorf("injection keeps %d of the user's own 2 turns", len(got))
	}
	if have, _ := one.store.SyncState(trustRoom); have[one.d.id.PeerID] != 1 || have[two.d.id.PeerID] != 1 {
		t.Errorf("the watermark is not kept per machine: %v", have)
	}
}

// Two rooms with a different guest user in each: a room's targets are the machines of
// its own guests and never a machine of the other room's guest.
func TestARoomsTargetsAreOnlyTheMachinesOfItsOwnGuests(t *testing.T) {
	m := testMembership(t)
	self := newTestUser(t).machine(t, "here")
	a, _ := m.CreateRoom(self.Peer(), false)
	b, _ := m.CreateRoom(self.Peer(), false)
	inA, inB := newTestUser(t), newTestUser(t)
	for user, addrs := range map[*testUser][]string{inA: {"tc://a-one", "tc://a-two"}, inB: {"tc://b-one"}} {
		if err := m.Allow(user.id, ""); err != nil {
			t.Fatal(err)
		}
		for _, addr := range addrs {
			if err := m.RecordMachineAddress(user.machine(t, addr).PeerID, user.id, addr); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := m.Invite(a.RoomID, inA.id); err != nil {
		t.Fatal(err)
	}
	if err := m.Invite(b.RoomID, inB.id); err != nil {
		t.Fatal(err)
	}
	has := func(room Room) string {
		got := m.RoomPeers(room.RoomID, self.PeerID)
		sortStrings(got)
		return strings.Join(got, ",")
	}
	if got := has(a); got != "tc://a-one,tc://a-two" {
		t.Errorf("the first room's targets are %q", got)
	}
	if got := has(b); got != "tc://b-one" {
		t.Errorf("the second room's targets are %q", got)
	}
}

// A bootstrap address recorded at pairing is stored against the user, and the
// machine that then signs a request from it takes its place; a machine recorded under
// its own key moves to its user once its chain is known.
func TestAMachinesAddressIsLearnedFromItsSignedRequestAndReplacesTheBootstrap(t *testing.T) {
	m := testMembership(t)
	ana := newTestUser(t)
	laptop := ana.machine(t, "laptop")
	if err := m.Allow(ana.id, ""); err != nil {
		t.Fatal(err)
	}
	if err := m.SetPeerEndpoint(ana.id, "tc://laptop"); err != nil {
		t.Fatal(err)
	}
	if got := m.PeerEndpointsOf(ana.id); len(got) != 1 || got[0] != "tc://laptop" {
		t.Fatalf("the bootstrap address is %v", got)
	}
	if err := m.RecordMachineAddress(laptop.PeerID, ana.id, "tc://laptop"); err != nil {
		t.Fatal(err)
	}
	ms, _ := m.UserMachines(ana.id)
	if len(ms) != 1 || ms[0].MachineKey != laptop.PeerID {
		t.Errorf("the user's machines are %+v; want the laptop alone", ms)
	}

	// An address copied from known_peers has the machine as its own user.
	old := ana.machine(t, "old")
	if _, err := m.db.Exec(`INSERT INTO machines (machine_key, user, endpoint, endpoint_at) VALUES (?,?,?,?)`,
		old.PeerID, old.PeerID, "tc://old", "2026-10-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if err := m.RecordMachineAddress(old.PeerID, ana.id, "tc://old-moved"); err != nil {
		t.Fatal(err)
	}
	if ep, _ := m.MachineEndpoint(old.PeerID); ep != "tc://old-moved" {
		t.Errorf("the migrated machine's address is %q", ep)
	}
	ms, _ = m.UserMachines(ana.id)
	if len(ms) != 2 {
		t.Errorf("the migrated machine did not move to its user: %+v", ms)
	}
	// A machine given to one user does not move to another.
	if err := m.AssignMachineUser(old.PeerID, newTestUser(t).id); err != nil {
		t.Fatal(err)
	}
	if ms, _ := m.UserMachines(ana.id); len(ms) != 2 {
		t.Error("a machine moved from one user to another")
	}
}

// --- the room's events table ---

// A room made when events were unique by peer and sequence is rebuilt to be unique by
// signer and sequence, so two machines of one user can each hold sequence 1, and the
// events it held still verify (D-239, an event is identified by its signer, room and
// sequence).
func TestARoomFromBeforeV4HoldsTwoMachinesOfOneUserAtOneSequence(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("COGMER_HOME", "")
	dir := filepath.Join(homeDir(), "rooms")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	old, err := sql.Open("sqlite", filepath.Join(dir, "room-a.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.Exec(eventsSchemaBeforeSigner); err != nil {
		t.Fatal(err)
	}
	v3 := testIdentity(t)
	ev := signed(t, v3, "written before v4")
	ev.RoomID = "room-a"
	ev.Sign(v3.private)
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
		t.Fatalf("opening a room from before v4: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	ana := newTestUser(t)
	for i, id := range []*Identity{ana.machine(t, "one"), ana.machine(t, "two")} {
		if _, err := s.AppendAs(1, id, nil, "room-a", "s", EventUserPrompt, "machine "+string(rune('1'+i)), nil); err != nil {
			t.Fatalf("machine %d's sequence 1: %v", i+1, err)
		}
	}
	got, err := s.ListRoom("room-a")
	if err != nil || len(got) != 3 {
		t.Fatalf("the room holds %d events: %v", len(got), err)
	}
	for _, e := range got {
		if err := e.Verify(); err != nil {
			t.Errorf("event %s does not verify after the rebuild: %v", e.EventID, err)
		}
	}
	if conflicts, _ := s.ListConflicts(); len(conflicts) != 0 {
		t.Errorf("two machines' sequence 1 were set aside as conflicts: %+v", conflicts)
	}
	// One machine reissuing a sequence is still a conflict.
	dup := got[1]
	dup.EventID = "another"
	if res, err := s.Insert(&dup); err != nil || res != InsertConflict {
		t.Errorf("one signer's sequence twice is %v, %v; want a conflict", res, err)
	}
}

// The shared key a user identity is made with is kept in shared.key and never in
// identity.json, which whoami prints (D-153, the private key is kept in a file of
// its own).
func TestTheTestUserKeepsItsSharedKeyApart(t *testing.T) {
	ana := newTestUser(t)
	p := newUserPeer(t, trustRoom, trustRoomName, ana.machine(t, "laptop"), ana)
	key, version, ok, err := loadSharedKey()
	if err != nil || !ok || version != ana.version || !key.Equal(ana.shared) {
		t.Fatalf("shared.key holds %v %v %v", version, ok, err)
	}
	raw, _ := json.Marshal(p.d.id)
	seed := ana.shared.Seed()
	sum := sha256.Sum256(seed)
	for _, enc := range []string{hex.EncodeToString(seed), hex.EncodeToString(sum[:])} {
		if strings.Contains(string(raw), enc) {
			t.Error("identity.json would hold the shared key")
		}
	}
	if !strings.Contains(string(raw), ana.id) || !strings.Contains(string(raw), `"chain"`) {
		t.Error("identity.json does not hold the user and the chain")
	}
}
