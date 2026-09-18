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
  verified_at TEXT
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
CREATE TABLE IF NOT EXISTS session_rooms (
  session_id TEXT PRIMARY KEY,
  room_id    TEXT NOT NULL,
  joined_at  TEXT NOT NULL,
  injected   INTEGER NOT NULL DEFAULT 0
);

-- Where a room's other members can be reached. An endpoint is reachability, not
-- identity (D-018): it changes when a machine moves and is only ever a hint.
CREATE TABLE IF NOT EXISTS room_peers (
  room_id  TEXT NOT NULL,
  endpoint TEXT NOT NULL,
  PRIMARY KEY (room_id, endpoint)
);

-- Machine-level state. current_room is the room a session joins when it begins:
-- a session cannot be asked which room it wants, because nothing knows a session
-- exists until its first hook fires.
CREATE TABLE IF NOT EXISTS settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);`

type Membership struct{ db *sql.DB }

type KnownPeer struct {
	PeerID     string `json:"peerId"`
	Name       string `json:"name"`
	AddedAt    string `json:"addedAt"`
	VerifiedAt string `json:"verifiedAt,omitempty"`
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
	if err := addColumnIfMissing(db, "known_peers", "verified_at", "TEXT"); err != nil {
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

// Allow records a peer this machine knows. The identifier is a public key, so
// receiving one requires no confidentiality and creates no exposure (§25).
func (m *Membership) Allow(peerID, name string) error {
	if _, err := PublicFromPeerID(peerID); err != nil {
		return fmt.Errorf("refusing to record an identifier that names no key: %w", err)
	}
	if name == "" {
		name = PeerName(peerID)
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
	_, err := m.db.Exec(`DELETE FROM known_peers WHERE peer_id = ?`, peerID)
	return err
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

func (m *Membership) IsVerified(peerID string) bool {
	var at string
	if err := m.db.QueryRow(`SELECT COALESCE(verified_at,'') FROM known_peers WHERE peer_id = ?`, peerID).Scan(&at); err != nil {
		return false
	}
	return at != ""
}

func (m *Membership) KnownPeers() ([]KnownPeer, error) {
	rows, err := m.db.Query(`SELECT peer_id, name, added_at, COALESCE(verified_at,'') FROM known_peers ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []KnownPeer
	for rows.Next() {
		var p KnownPeer
		if err := rows.Scan(&p.PeerID, &p.Name, &p.AddedAt, &p.VerifiedAt); err != nil {
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

func (m *Membership) AddRoomPeer(roomID, endpoint string) error {
	_, err := m.db.Exec(`INSERT OR IGNORE INTO room_peers (room_id, endpoint) VALUES (?,?)`, roomID, endpoint)
	return err
}

// RoomPeers lists where a room's other members might be. Endpoints go stale, so
// this is a set of things to try rather than a directory.
func (m *Membership) RoomPeers(roomID string) []string {
	rows, err := m.db.Query(`SELECT endpoint FROM room_peers WHERE room_id = ?`, roomID)
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

// SetCurrentRoom names the room new sessions join. It is deliberately machine-level
// rather than per-terminal: a developer works in one room at a time, and asking
// every session which room it wants would mean asking at a moment when nobody is
// there to answer.
func (m *Membership) SetCurrentRoom(roomID string) error {
	if roomID == "" {
		_, err := m.db.Exec(`DELETE FROM settings WHERE key = 'current_room'`)
		return err
	}
	_, err := m.db.Exec(`
		INSERT INTO settings (key, value) VALUES ('current_room', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, roomID)
	return err
}

func (m *Membership) CurrentRoom() (Room, bool) {
	var roomID string
	if err := m.db.QueryRow(`SELECT value FROM settings WHERE key = 'current_room'`).Scan(&roomID); err != nil {
		return Room{}, false
	}
	return m.RoomByID(roomID)
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

// RoomForSession binds a session to a room on first sight and keeps it there.
//
// Binding at first sight rather than asking is what makes §12a's constraint
// enforceable: a session that has already been told something cannot be moved, and
// there is no moment later at which moving it would be safe.
func (m *Membership) RoomForSession(sessionID string) (Room, bool) {
	var roomID string
	err := m.db.QueryRow(`SELECT room_id FROM session_rooms WHERE session_id = ?`, sessionID).Scan(&roomID)
	if err == nil {
		return m.RoomByID(roomID)
	}
	cur, ok := m.CurrentRoom()
	if !ok {
		return Room{}, false // no room joined; this session collaborates with nobody
	}
	_, _ = m.db.Exec(`INSERT OR IGNORE INTO session_rooms (session_id, room_id, joined_at) VALUES (?,?,?)`,
		sessionID, cur.RoomID, time.Now().UTC().Format(time.RFC3339))
	return cur, true
}

// MarkInjected records that a session has received teammate context, after which
// §12a forbids moving it to another room: injected context cannot be withdrawn.
func (m *Membership) MarkInjected(sessionID string) error {
	_, err := m.db.Exec(`UPDATE session_rooms SET injected = 1 WHERE session_id = ?`, sessionID)
	return err
}

func (m *Membership) HasReceivedContext(sessionID string) bool {
	var n int
	err := m.db.QueryRow(`SELECT injected FROM session_rooms WHERE session_id = ?`, sessionID).Scan(&n)
	return err == nil && n == 1
}

// SessionsInRoom reports how many sessions are bound to a room, which is what
// presence is derived from rather than asserted.
func (m *Membership) SessionsInRoom(roomID string) int {
	var n int
	_ = m.db.QueryRow(`SELECT COUNT(*) FROM session_rooms WHERE room_id = ?`, roomID).Scan(&n)
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
