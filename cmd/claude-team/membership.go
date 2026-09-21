package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// membership.db holds what must survive a room (§22): which peers this machine
// knows, which rooms it belongs to, and who may enter each.
//
// It sits beside identity.json rather than inside any room, and that placement is
// the point. A room's database can be lost while the peer survives; a value stored
// only inside the thing whose loss it guards against is no guard at all.
//
// Two scopes, because knowing someone and admitting them to a particular
// conversation are different decisions (§12). Collapsing them would make it
// impossible to know six colleagues and admit two to a room about customer data.

const membershipSchema = `
CREATE TABLE IF NOT EXISTS known_peers (
  peer_id     TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  added_at    TEXT NOT NULL,
  -- When two people compared a SAS and said it matched (D-048/D-052). NULL until
  -- they have. Without this the "unverified" marker §25 requires inside injected
  -- text is true of every peer forever, and a marker that can never change is one
  -- a reader learns to stop seeing.
  verified_at TEXT,
  -- Where this peer was last known to listen. A bootstrap hint in §12's sense,
  -- held at MACHINE scope because pairing precedes any room: without it there is
  -- nowhere to reach a peer until a room already exists, which would force
  -- verification to follow admission rather than precede it.
  endpoint TEXT,
  -- When this address was last advertised by the peer whose address it is. An
  -- address is a snapshot of where a machine was, so age is the only thing that
  -- distinguishes one worth trying from one worth trying last (§4).
  endpoint_at TEXT
);
-- An invitation that has arrived and has not been accepted. Held apart from the
-- rooms table deliberately: a room recorded there would be synchronized and shown
-- before anybody agreed to join it, and an offer is a thing to accept rather than
-- a room you are in (D-105).
CREATE TABLE IF NOT EXISTS pending_offers (
  room_id      TEXT PRIMARY KEY,
  room_name    TEXT NOT NULL,
  host_peer_id TEXT NOT NULL,
  endpoint     TEXT NOT NULL,
  offered_at   TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS rooms (
  room_id         TEXT PRIMARY KEY,
  -- Not UNIQUE. Names collide by design (D-017) and this table holds rooms other
  -- peers named, so a uniqueness rule here is not a rule about the world -- it is
  -- this machine refusing to record something that has already happened.
  room_name       TEXT NOT NULL,
  state           TEXT NOT NULL,
  issued_sequence INTEGER NOT NULL DEFAULT 0,
  created_at      TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS room_guests (
  room_id TEXT NOT NULL,
  peer_id TEXT NOT NULL,
  PRIMARY KEY (room_id, peer_id)
);

-- Which room each agent session belongs to. A session holds membership in at most
-- one room at a time (§12a), which is expressible only because this is keyed on the
-- session rather than on the daemon.
-- A session's room is fixed at first sight and never changes. There is no
-- "injected yet?" column: a second mechanism for one rule is how one of them rots,
-- and this one was written and never read (D-056).
CREATE TABLE IF NOT EXISTS session_rooms (
  session_id TEXT PRIMARY KEY,
  room_id    TEXT NOT NULL,
  joined_at  TEXT NOT NULL,
  -- When this session left. The row is kept rather than deleted, because it
  -- answers two questions that must not share an answer: whether the session is
  -- in a room now, and which room it has ever been in. Delete it and leaving
  -- becomes a move to another room with two extra keystrokes, which is the one
  -- thing §12a forbids (D-071).
  left_at    TEXT
);`

type Membership struct{ db *sql.DB }

type KnownPeer struct {
	PeerID     string `json:"peerId"`
	Name       string `json:"name"`
	AddedAt    string `json:"addedAt"`
	VerifiedAt string `json:"verifiedAt,omitempty"`
	Endpoint   string `json:"endpoint,omitempty"`
}

type Room struct {
	RoomID    string `json:"roomId"`
	RoomName  string `json:"roomName"`
	State     string `json:"state"`
	CreatedAt string `json:"createdAt"`
}

func OpenMembership() (*Membership, error) {
	if err := os.MkdirAll(homeDir(), 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(homeDir(), "membership.db")
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(membershipSchema); err != nil {
		return nil, fmt.Errorf("init membership: %w", err)
	}
	if err := migrateMembership(db); err != nil {
		return nil, fmt.Errorf("migrate membership: %w", err)
	}
	return &Membership{db: db}, nil
}

// migrateMembership brings an existing membership.db up to date. CREATE TABLE IF
// NOT EXISTS leaves an older table exactly as it was, so a rule relaxed in the
// schema above is still enforced on every database created before this ran.
//
// The rule in question is UNIQUE on rooms.room_name. It cannot be dropped in
// place -- SQLite has no DROP CONSTRAINT -- so the table is rebuilt.
func migrateMembership(db *sql.DB) error {
	rows, err := db.Query(`SELECT name, origin FROM pragma_index_list('rooms')`)
	if err != nil {
		return err
	}
	unique := ""
	for rows.Next() {
		var name, origin string
		if err := rows.Scan(&name, &origin); err != nil {
			rows.Close()
			return err
		}
		// origin 'u' is an index SQLite created for a UNIQUE column constraint.
		if origin == "u" {
			unique = name
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, c := range []struct{ table, column, typ string }{
		{"known_peers", "verified_at", "TEXT"},
		{"known_peers", "endpoint", "TEXT"},
		{"known_peers", "endpoint_at", "TEXT"},
		{"session_rooms", "left_at", "TEXT"},
	} {
		if err := addColumnIfMissing(db, c.table, c.column, c.typ); err != nil {
			return err
		}
	}
	// Dropped rather than left in place. It would be harmless -- it has a default
	// and nothing writes it -- but a column encoding a rule that was removed is a
	// rule somebody will later find and reinstate (D-056).
	if err := dropColumnIfPresent(db, "session_rooms", "injected"); err != nil {
		return err
	}
	// An address belongs to a peer (D-103). This table held addresses with no peer
	// column, so nothing can be carried out of it -- which is the defect, not a
	// migration difficulty. Peers re-advertise on their next synchronization and
	// the rows rebuild themselves, in the place that names whose they are.
	if _, err := db.Exec(`DROP TABLE IF EXISTS room_peers`); err != nil {
		return err
	}
	if unique == "" {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, stmt := range []string{
		`CREATE TABLE rooms_new (
		  room_id         TEXT PRIMARY KEY,
		  room_name       TEXT NOT NULL,
		  state           TEXT NOT NULL,
		  issued_sequence INTEGER NOT NULL DEFAULT 0,
		  created_at      TEXT NOT NULL
		)`,
		`INSERT INTO rooms_new (room_id, room_name, state, issued_sequence, created_at)
		   SELECT room_id, room_name, state, issued_sequence, created_at FROM rooms`,
		`DROP TABLE rooms`,
		`ALTER TABLE rooms_new RENAME TO rooms`,
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("rebuild rooms: %w", err)
		}
	}
	return tx.Commit()
}

func (m *Membership) Close() error { return m.db.Close() }

// NameTakenError is a name already held by a different key.
//
// It is the only place a key change can surface. A peerId IS a key (D-042), so
// "the same person with a new key" is not expressible — to this machine it is
// simply a peer it has never seen. The one moment the two can be connected is
// when somebody records the new key under a name they already use, and that is
// what this catches (D-074).
type NameTakenError struct {
	Name     string
	Existing string // the key already known by that name
	Offered  string // the key being recorded
}

func (e *NameTakenError) Error() string {
	return fmt.Sprintf("you already know a different key as %q", e.Name)
}

// Allow records a peer this machine knows. The identifier is a public key, so
// receiving one requires no confidentiality and creates no exposure (§25).
// SetPeerEndpoint records where a peer was last known to listen. It is a hint and
// is allowed to be wrong: §12 requires only that it be correct once, since a peer
// that has joined a room learns how to reach the others.
func (m *Membership) SetPeerEndpoint(peerID, endpoint string) error {
	if endpoint == "" {
		return nil
	}
	// Written on every synchronization a peer performs, which is often and almost
	// always the same value. Comparing first keeps that from being a database
	// write per poll, and the timestamp is only interesting when it moved.
	if cur, at := m.peerEndpointAt(peerID); cur == endpoint && at != "" {
		return nil
	}
	_, err := m.db.Exec(`UPDATE known_peers SET endpoint = ?, endpoint_at = ? WHERE peer_id = ?`,
		endpoint, time.Now().UTC().Format(time.RFC3339), peerID)
	return err
}

// PeerEndpoint is the address recorded for one peer.
//
// The query that should always have been here. Its absence is why verification
// dialled every address this machine knew: there was no way to ask which one was a
// particular peer's, so it asked for all of them (D-103).
func (m *Membership) PeerEndpoint(peerID string) string {
	e, _ := m.peerEndpointAt(peerID)
	return e
}

func (m *Membership) peerEndpointAt(peerID string) (endpoint, at string) {
	var e, a sql.NullString
	if err := m.db.QueryRow(`SELECT endpoint, endpoint_at FROM known_peers WHERE peer_id = ?`,
		peerID).Scan(&e, &a); err != nil {
		return "", ""
	}
	return e.String, a.String
}

// PeerEndpoints is every machine-scope address worth trying. Room membership
// supplies others; these are the ones that exist before any room does.
func (m *Membership) PeerEndpoints() []string {
	rows, err := m.db.Query(`SELECT endpoint FROM known_peers WHERE endpoint IS NOT NULL AND endpoint <> ''`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var e string
		if rows.Scan(&e) == nil {
			out = append(out, e)
		}
	}
	return out
}

// NameFree reports whether a label may be attached to this peer.
//
// A name must mean one key. Without that, recording a substituted key under a
// colleague's name produced a second row and nothing said so — and `invite alice`
// would then admit whichever came back first, which is the failure the whole
// verification apparatus exists to prevent (D-074).
//
// Separate from Allow so a pairing can check the label is available BEFORE spending
// ninety seconds on a ceremony, rather than discovering the clash once the words
// have already matched — which would leave a verified peer and an unusable name,
// and so a second thing for somebody to finish (D-093).
// Label is the name this person gave one peer, or empty if they gave none.
//
// A label equal to the derived name is a placeholder rather than a choice, and
// reporting it as one would present a machine-made word pair as somebody's
// decision.
func (m *Membership) Label(peerID string) string {
	var name string
	if err := m.db.QueryRow(`SELECT name FROM known_peers WHERE peer_id = ?`, peerID).Scan(&name); err != nil {
		return ""
	}
	if name == PeerName(peerID) {
		return ""
	}
	return name
}

// Labels maps each known peer to the name this person gave it.
//
// The label is the only name that is both memorable and bound to one key: the
// display name is the peer's own claim and can be anything, and the derived name is
// computed from the key and means nothing to anybody weeks later (D-094).
func (m *Membership) Labels() map[string]string {
	out := map[string]string{}
	known, err := m.KnownPeers()
	if err != nil {
		return out
	}
	for _, p := range known {
		// A label equal to the derived name is a placeholder, not a choice, and
		// offering it as though somebody picked it would be a lie.
		if p.Name != "" && p.Name != PeerName(p.PeerID) {
			out[p.PeerID] = p.Name
		}
	}
	return out
}

func (m *Membership) NameFree(name, peerID string) error {
	if name == "" {
		return nil
	}
	var existing string
	if err := m.db.QueryRow(`SELECT peer_id FROM known_peers WHERE name = ? AND peer_id <> ?`,
		name, peerID).Scan(&existing); err == nil {
		return &NameTakenError{Name: name, Existing: existing, Offered: peerID}
	}
	return nil
}

func (m *Membership) Allow(peerID, name string) error {
	if _, err := PublicFromPeerID(peerID); err != nil {
		return fmt.Errorf("refusing to record an identifier that names no key: %w", err)
	}
	if name == "" {
		name = PeerName(peerID)
	}
	if err := m.NameFree(name, peerID); err != nil {
		return err
	}

	_, err := m.db.Exec(`
		INSERT INTO known_peers (peer_id, name, added_at) VALUES (?,?,?)
		ON CONFLICT(peer_id) DO UPDATE SET name = excluded.name`,
		peerID, name, time.Now().UTC().Format(time.RFC3339))
	return err
}

// Forget discards an identity entirely, so a later meeting is a first meeting
// again. Distinct from revoking, which withdraws admission to one room (§12).
func (m *Membership) Forget(peerID string) error {
	// Admission goes with the identity, or "a later meeting is a first meeting"
	// (§12) is not true. Without this, forgetting a peer and later meeting them
	// again would readmit them to every room they had ever been in, without the
	// host inviting them to any of it — and the host would not be asked, because
	// from the code's point of view they were never un-invited (D-073).
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM room_guests WHERE peer_id = ?`, peerID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM known_peers WHERE peer_id = ?`, peerID); err != nil {
		return err
	}
	return tx.Commit()
}

// Knows reports whether this machine holds a key for a peer. Verification
// confirms a key already on file, so there is nothing to confirm for a stranger.
func (m *Membership) Knows(peerID string) bool {
	var n int
	if err := m.db.QueryRow(`SELECT COUNT(*) FROM known_peers WHERE peer_id = ?`, peerID).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

// MarkVerified records that two people compared a SAS and it matched. It is
// recorded only on a human saying so: the exchange proves both sides hold the keys
// they named, and only a person can say that the voice on the call was the
// colleague rather than somebody in their place (§25).
func (m *Membership) MarkVerified(peerID string) error {
	res, err := m.db.Exec(`UPDATE known_peers SET verified_at = ? WHERE peer_id = ?`,
		time.Now().UTC().Format(time.RFC3339), peerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("%s is not a peer this machine knows", PeerName(peerID))
	}
	return nil
}

// VerifiedAt is when a peer was verified, or empty if they have not been. Reported
// so that "already paired" can say since when, which is the difference between an
// assertion and something a person can check against their memory.
func (m *Membership) VerifiedAt(peerID string) string {
	var at string
	if err := m.db.QueryRow(`SELECT COALESCE(verified_at,'') FROM known_peers WHERE peer_id = ?`,
		peerID).Scan(&at); err != nil {
		return ""
	}
	return at
}

func (m *Membership) IsVerified(peerID string) bool {
	var at string
	if err := m.db.QueryRow(`SELECT COALESCE(verified_at,'') FROM known_peers WHERE peer_id = ?`, peerID).Scan(&at); err != nil {
		return false
	}
	return at != ""
}

func (m *Membership) KnownPeers() ([]KnownPeer, error) {
	rows, err := m.db.Query(`SELECT peer_id, name, added_at, COALESCE(verified_at,''), COALESCE(endpoint,'')
		FROM known_peers ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []KnownPeer
	for rows.Next() {
		var p KnownPeer
		if err := rows.Scan(&p.PeerID, &p.Name, &p.AddedAt, &p.VerifiedAt, &p.Endpoint); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CreateRoom mints a room and makes its creator the first guest.
//
// Names need be unique only among the rooms this peer hosts, since a name is only
// ever resolved against a specific peer (§12) -- so a collision is regenerated
// locally rather than coordinated globally.
func (m *Membership) CreateRoom(selfPeerID string) (Room, error) {
	for attempt := 0; attempt < 20; attempt++ {
		r := Room{
			RoomID: uuidV4(), RoomName: NewRoomName(), State: "joined",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		// Asked rather than caught. A name this peer mints is one it is free to
		// mint differently, so avoiding a local clash costs nothing -- unlike a
		// name that arrives with a room somebody else already named.
		var taken int
		if err := m.db.QueryRow(`SELECT COUNT(*) FROM rooms WHERE room_name = ?`, r.RoomName).Scan(&taken); err != nil {
			return Room{}, err
		}
		if taken > 0 {
			continue
		}
		if _, err := m.db.Exec(`INSERT INTO rooms (room_id, room_name, state, created_at) VALUES (?,?,?,?)`,
			r.RoomID, r.RoomName, r.State, r.CreatedAt); err != nil {
			return Room{}, err
		}
		if err := m.Invite(r.RoomID, selfPeerID); err != nil {
			return Room{}, err
		}
		return r, nil
	}
	return Room{}, fmt.Errorf("could not find an unused room name after 20 attempts")
}

func (m *Membership) Rooms() ([]Room, error) {
	rows, err := m.db.Query(`SELECT room_id, room_name, state, created_at FROM rooms ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Room
	for rows.Next() {
		var r Room
		if err := rows.Scan(&r.RoomID, &r.RoomName, &r.State, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

var errNoSuchRoom = errors.New("no such room")

// FindRoom resolves either identifier. A name is for people and an id is for
// machines, and a person typing one should not have to know which (§3.2).
//
// It reports three outcomes rather than two, because a name can now match more
// than one room. Returning "not found" for an ambiguous name would be a lie that
// sends someone looking for a room they are already in; returning whichever row
// came back first would pick on their behalf without saying so.
func (m *Membership) FindRoom(nameOrID string) (Room, error) {
	rows, err := m.db.Query(`SELECT room_id, room_name, state, created_at FROM rooms
		WHERE room_id = ? OR room_name = ? ORDER BY created_at`, nameOrID, nameOrID)
	if err != nil {
		return Room{}, err
	}
	defer rows.Close()
	var found []Room
	for rows.Next() {
		var r Room
		if err := rows.Scan(&r.RoomID, &r.RoomName, &r.State, &r.CreatedAt); err != nil {
			return Room{}, err
		}
		found = append(found, r)
	}
	if err := rows.Err(); err != nil {
		return Room{}, err
	}
	switch len(found) {
	case 0:
		return Room{}, fmt.Errorf("%w: %q", errNoSuchRoom, nameOrID)
	case 1:
		return found[0], nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d rooms are called %q. Say which by its identity:", len(found), nameOrID)
	for _, r := range found {
		fmt.Fprintf(&b, "\n  %s  (%s, joined %s)", r.RoomID, r.State, r.CreatedAt)
	}
	return Room{}, errors.New(b.String())
}

// Invite admits a peer to a room. Nothing here issues a token: admission is a
// record that this peer may enter, proved later by possession of its key (D-026).
func (m *Membership) Invite(roomID, peerID string) error {
	if _, err := PublicFromPeerID(peerID); err != nil {
		return fmt.Errorf("refusing to admit an identifier that names no key: %w", err)
	}
	_, err := m.db.Exec(`INSERT OR IGNORE INTO room_guests (room_id, peer_id) VALUES (?,?)`, roomID, peerID)
	return err
}

// Revoke withdraws admission to one room. It does not retract what that peer has
// already seen (§25).
func (m *Membership) Revoke(roomID, peerID string) error {
	_, err := m.db.Exec(`DELETE FROM room_guests WHERE room_id = ? AND peer_id = ?`, roomID, peerID)
	return err
}

func (m *Membership) Guests(roomID string) ([]string, error) {
	rows, err := m.db.Query(`SELECT peer_id FROM room_guests WHERE room_id = ? ORDER BY peer_id`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// RecordRoom stores a room learned from an invitation. The guest did not create it
// and cannot invent its identity, so both come from the invitation.
func (m *Membership) RecordRoom(roomID, roomName string) error {
	_, err := m.db.Exec(`
		INSERT INTO rooms (room_id, room_name, state, created_at) VALUES (?,?,'joined',?)
		ON CONFLICT(room_id) DO UPDATE SET state = 'joined'`,
		roomID, roomName, time.Now().UTC().Format(time.RFC3339))
	return err
}

// RoomPeers lists where a room's other members might be. Endpoints go stale, so
// this is a set of things to try rather than a directory.
// ReserveSequence issues the next peer sequence for a room and records it HERE,
// outside the room's own database (D-029).
//
// The order is the whole point: reserve, then publish the event that uses the
// number. The reverse publishes a number with no record of it, so a crash between
// the two reissues it -- and a reissued sequence is a different event under an
// identifier a peer already holds, which D-027 quarantines and nobody can repair.
//
// It also survives losing the room. The record of what was issued lives with the
// identity, so a peer that loses a room database resumes above its high-water mark
// instead of restarting at 1 and colliding with everything it ever sent.
func (m *Membership) ReserveSequence(roomID string) (int64, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var issued int64
	if err := tx.QueryRow(`SELECT issued_sequence FROM rooms WHERE room_id = ?`, roomID).Scan(&issued); err != nil {
		return 0, fmt.Errorf("no record of room %s to reserve a sequence in: %w", roomID, err)
	}
	issued++
	if _, err := tx.Exec(`UPDATE rooms SET issued_sequence = ? WHERE room_id = ?`, issued, roomID); err != nil {
		return 0, err
	}
	return issued, tx.Commit()
}

// IssuedSequence is the highest sequence this peer has ever issued in a room,
// whether or not the room still holds the events.
func (m *Membership) IssuedSequence(roomID string) int64 {
	var issued int64
	_ = m.db.QueryRow(`SELECT issued_sequence FROM rooms WHERE room_id = ?`, roomID).Scan(&issued)
	return issued
}

// RoomPeers is where to reach the other members of a room.
//
// A join rather than a list of its own: a room's members are its guests, and an
// address belongs to a peer (D-103). Asking the question this way returns the
// identity alongside each address, which is what lets a failure be attributed and
// a peer that moves overwrite one row instead of adding another.
//
// self is excluded explicitly. You are a guest of your own rooms and are never in
// your own known-peers list, so the join would drop you anyway -- by the absence of
// a row, which is the right answer for the wrong reason and stops being right the
// moment somebody adds one.
func (m *Membership) RoomPeers(roomID, self string) []string {
	rows, err := m.db.Query(`
		SELECT k.endpoint FROM room_guests g
		JOIN known_peers k ON k.peer_id = g.peer_id
		WHERE g.room_id = ? AND g.peer_id <> ? AND k.endpoint IS NOT NULL AND k.endpoint <> ''`,
		roomID, self)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var e string
		if rows.Scan(&e) == nil {
			out = append(out, e)
		}
	}
	return out
}

// RoomByID resolves a room by its identity alone. FindRoom also accepts a name,
// which is right at a command line and wrong on the wire: room names collide by
// design (D-017), so a request naming one can address a room its sender did not
// mean. Anything arriving from a peer resolves here.
func (m *Membership) RoomByID(roomID string) (Room, bool) {
	if roomID == "" {
		return Room{}, false
	}
	var r Room
	err := m.db.QueryRow(`SELECT room_id, room_name, state, created_at FROM rooms WHERE room_id = ?`, roomID).
		Scan(&r.RoomID, &r.RoomName, &r.State, &r.CreatedAt)
	return r, err == nil
}

// RoomForSession reports the room a session was put in, and never puts it in one.
//
// A session joins a room because somebody ran /team-create or /team-join inside it
// (D-064). It does not join one by being started while a machine-level setting
// happened to point somewhere: a room created and forgotten would otherwise be
// silently joined weeks later by a session in an unrelated repository, capturing
// and publishing without anyone doing anything. Nothing derives a room from a
// directory (D-015), so nothing else would have caught it either.
//
// A session in no room is an ordinary Claude Code session: nothing captured,
// nothing injected, nothing shared (§12a).
func (m *Membership) RoomForSession(sessionID string) (Room, bool) {
	var roomID string
	if err := m.db.QueryRow(`SELECT room_id FROM session_rooms
		WHERE session_id = ? AND left_at IS NULL`, sessionID).Scan(&roomID); err != nil {
		return Room{}, false
	}
	return m.RoomByID(roomID)
}

// LeaveSession stops a session participating, without foreclosing its return.
//
// Three things it deliberately does not do (D-071): it does not foreclose
// rejoining the same room, because §12a forbids MOVING to another room and
// returning to the one you were in is not a move; it does not touch the room's
// events; and it does not change which room command-line commands or the browser
// view act on, so the transcript as it accumulated stays exactly as readable as it
// was a moment earlier.
func (m *Membership) LeaveSession(sessionID string) (Room, error) {
	room, ok := m.RoomForSession(sessionID)
	if !ok {
		return Room{}, errors.New("this session is not in a room")
	}
	_, err := m.db.Exec(`UPDATE session_rooms SET left_at = ? WHERE session_id = ?`,
		time.Now().UTC().Format(time.RFC3339), sessionID)
	return room, err
}

// BoundElsewhereError is a session being refused a second room.
//
// A typed error rather than a formatted string, because the refusal needs
// explaining and the explanation belongs where it is shown to a person, not
// inside an error value that tests compare and logs prefix with a timestamp
// (D-072). Both rooms are carried so the explanation can name them.
type BoundElsewhereError struct {
	Was    Room
	Wanted Room
}

func (e *BoundElsewhereError) Error() string {
	return fmt.Sprintf("this session has been in %s and cannot join %s", e.Was.RoomName, e.Wanted.RoomName)
}

// BindSession puts a session in a room, once and for good. §12a forbids moving it
// afterwards, because injected context cannot be withdrawn from a context window,
// so a second call for a session already bound is refused rather than obeyed.
func (m *Membership) BindSession(sessionID, roomID string) error {
	if sessionID == "" {
		return errors.New("no session to bind")
	}
	// Every room this session has EVER been in, not only the one it is in now. A
	// session that left is still barred from a different room -- otherwise leaving
	// would launder the move §12a forbids.
	var was string
	err := m.db.QueryRow(`SELECT room_id FROM session_rooms WHERE session_id = ?`, sessionID).Scan(&was)
	if err == nil {
		if was != roomID {
			prior, _ := m.RoomByID(was)
			wanted, _ := m.RoomByID(roomID)
			return &BoundElsewhereError{Was: prior, Wanted: wanted}
		}
		// Returning to the room it was in. Not a move, so permitted.
		_, err := m.db.Exec(`UPDATE session_rooms SET left_at = NULL WHERE session_id = ?`, sessionID)
		return err
	}
	_, err = m.db.Exec(`INSERT INTO session_rooms (session_id, room_id, joined_at) VALUES (?,?,?)`,
		sessionID, roomID, time.Now().UTC().Format(time.RFC3339))
	return err
}

// SessionsInRoom reports how many sessions are bound to a room, which is what
// presence is derived from rather than asserted.
func (m *Membership) SessionsInRoom(roomID string) int {
	var n int
	_ = m.db.QueryRow(`SELECT COUNT(*) FROM session_rooms WHERE room_id = ? AND left_at IS NULL`, roomID).Scan(&n)
	return n
}

// IsGuest is the admission decision. Authentication established who is asking;
// this establishes whether they may (D-044).
func (m *Membership) IsGuest(roomID, peerID string) bool {
	var n int
	err := m.db.QueryRow(`SELECT COUNT(*) FROM room_guests WHERE room_id = ? AND peer_id = ?`,
		roomID, peerID).Scan(&n)
	return err == nil && n > 0
}

// addColumnIfMissing is the whole of a column migration. CREATE TABLE IF NOT
// EXISTS ignores an existing table, so a column added to the schema above is
// absent from every database created before it and fails at first query rather
// than at open.
func addColumnIfMissing(db *sql.DB, table, column, typ string) error {
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		return err
	}
	defer rows.Close()
	any := false
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return err
		}
		any = true
		if n == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !any {
		return nil // no such table yet; the schema above will have made it
	}
	_, err = db.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + typ)
	return err
}

// dropColumnIfPresent removes a column that has stopped meaning anything. SQLite
// has supported DROP COLUMN since 3.35, and the column is unindexed and unreferenced
// here, which is the case it supports.
func dropColumnIfPresent(db *sql.DB, table, column string) error {
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		return err
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return err
		}
		if n == column {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !found {
		return nil
	}
	_, err = db.Exec("ALTER TABLE " + table + " DROP COLUMN " + column)
	return err
}

// --- invitations that have arrived and not been accepted (D-105) ---

// Offer is a room somebody has admitted you to and told you about.
type Offer struct {
	RoomID    string
	RoomName  string
	Host      string
	Endpoint  string
	OfferedAt string
}

// RecordOffer stores an invitation delivered over the channel. Repeating one is
// ordinary rather than an error: a host that cannot tell whether the last delivery
// landed should send again, and the newest details win.
func (m *Membership) RecordOffer(o Offer) error {
	_, err := m.db.Exec(`
		INSERT INTO pending_offers (room_id, room_name, host_peer_id, endpoint, offered_at)
		VALUES (?,?,?,?,?)
		ON CONFLICT(room_id) DO UPDATE SET
			room_name = excluded.room_name,
			host_peer_id = excluded.host_peer_id,
			endpoint = excluded.endpoint,
			offered_at = excluded.offered_at`,
		o.RoomID, o.RoomName, o.Host, o.Endpoint, time.Now().UTC().Format(time.RFC3339))
	return err
}

// Offers lists invitations waiting to be accepted.
func (m *Membership) Offers() []Offer {
	rows, err := m.db.Query(`SELECT room_id, room_name, host_peer_id, endpoint, offered_at
		FROM pending_offers ORDER BY offered_at`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Offer
	for rows.Next() {
		var o Offer
		if rows.Scan(&o.RoomID, &o.RoomName, &o.Host, &o.Endpoint, &o.OfferedAt) == nil {
			out = append(out, o)
		}
	}
	return out
}

// FindOffer resolves an offer by room name or id, so a person can accept one by
// typing the name they were shown.
func (m *Membership) FindOffer(nameOrID string) (Offer, bool) {
	var o Offer
	err := m.db.QueryRow(`SELECT room_id, room_name, host_peer_id, endpoint, offered_at
		FROM pending_offers WHERE room_id = ? OR room_name = ?`, nameOrID, nameOrID).
		Scan(&o.RoomID, &o.RoomName, &o.Host, &o.Endpoint, &o.OfferedAt)
	return o, err == nil
}

// DropOffer removes one once it has been accepted.
func (m *Membership) DropOffer(roomID string) error {
	_, err := m.db.Exec(`DELETE FROM pending_offers WHERE room_id = ?`, roomID)
	return err
}

// RoomsAdmitting lists the rooms a peer has been admitted to, which is what a host
// owes them an invitation for. Used to deliver whatever was waiting when a
// verification completes (D-106).
func (m *Membership) RoomsAdmitting(peerID string) []Room {
	rows, err := m.db.Query(`
		SELECT r.room_id, r.room_name, r.state, r.created_at FROM room_guests g
		JOIN rooms r ON r.room_id = g.room_id
		WHERE g.peer_id = ?`, peerID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Room
	for rows.Next() {
		var r Room
		if rows.Scan(&r.RoomID, &r.RoomName, &r.State, &r.CreatedAt) == nil {
			out = append(out, r)
		}
	}
	return out
}

// WithheldFor counts the rooms a peer has been admitted to but cannot be told
// about, because the two have not verified each other (D-106). It is what turns
// "unverified" from a state into a reason.
func (m *Membership) WithheldFor(peerID string) int {
	if m.IsVerified(peerID) {
		return 0
	}
	var n int
	if err := m.db.QueryRow(`SELECT COUNT(*) FROM room_guests WHERE peer_id = ?`, peerID).Scan(&n); err != nil {
		return 0
	}
	return n
}
