package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"
)

// --- carrying a machine key and its rooms into the user identity ---
//
// A machine that used cogmer before its user had an identity keeps its key and is
// certified under the user with that key. When it presents its chain in a request it
// signs, a colleague's machine that held the key as a colleague's identifier treats the
// key as the user's: the verification, the label, every guest-list entry naming the key
// and every turn the key signed pass to the user (D-242, a machine key from before the
// user identity carries its verification, label and turns to the first user it proves
// it belongs to). A room from before user identities is then carried into them with one
// guest list (D-243, a room from before user identities takes one guest list, founded
// by the users whose machines wrote its turns).

// machineLinksSchema is the first user each machine key known here as a colleague has
// proved it belongs to. The first user is final: a second user the key proves it
// belongs to is recorded as a conflict and carries nothing, since only a copied or
// stolen key is certified under two users.
const machineLinksSchema = `
CREATE TABLE IF NOT EXISTS machine_links (
  machine_key   TEXT PRIMARY KEY,
  user          TEXT NOT NULL,
  linked_at     TEXT NOT NULL,
  conflict_user TEXT,
  conflict_at   TEXT
);
CREATE INDEX IF NOT EXISTS idx_machine_links_user ON machine_links(user);`

// LinkConflictError is a machine key that proved it belongs to one user here and now
// presents a chain to another.
type LinkConflictError struct {
	Machine, First, Second string
}

func (e *LinkConflictError) Error() string {
	return fmt.Sprintf("%s, a machine key this machine knows as %s's, now presents certificates for a different user, %s. "+
		"Only a copied or stolen key is certified under two users, so nothing passes to %s: "+
		"pair with them to confirm who they are",
		PeerName(e.Machine), PeerName(e.First), PeerName(e.Second), PeerName(e.Second))
}

// LinkedUser is the user a machine key known here as a colleague's has proved it
// belongs to, or false when it has proved none.
func (m *Membership) LinkedUser(key string) (string, bool) {
	var user string
	if err := m.db.QueryRow(`SELECT user FROM machine_links WHERE machine_key = ?`, key).Scan(&user); err != nil {
		return "", false
	}
	return user, true
}

// Links maps every machine key linked to a user here to that user.
func (m *Membership) Links() map[string]string {
	out := map[string]string{}
	rows, err := m.db.Query(`SELECT machine_key, user FROM machine_links`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var k, u string
		if rows.Scan(&k, &u) == nil {
			out[k] = u
		}
	}
	return out
}

// linkedKeys is every machine key linked to user here.
func (m *Membership) linkedKeys(user string) []string {
	rows, err := m.db.Query(`SELECT machine_key FROM machine_links WHERE user = ? ORDER BY machine_key`, user)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var k string
		if rows.Scan(&k) == nil {
			out = append(out, k)
		}
	}
	return out
}

// colleagueOf is the identifier a colleague's record is kept under: the user a
// linked machine key belongs to, and any other identifier as it is.
func (m *Membership) colleagueOf(peer string) string {
	if u, ok := m.LinkedUser(peer); ok {
		return u
	}
	return peer
}

// LinkConflicts is every machine key that proved it belongs to a second user, with
// the first user and the second.
func (m *Membership) LinkConflicts() []LinkConflictError {
	rows, err := m.db.Query(`SELECT machine_key, user, conflict_user FROM machine_links
		WHERE conflict_user IS NOT NULL ORDER BY machine_key`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []LinkConflictError
	for rows.Next() {
		var c LinkConflictError
		if rows.Scan(&c.Machine, &c.First, &c.Second) == nil {
			out = append(out, c)
		}
	}
	return out
}

// LinkMachineKey records that key, a machine key, has proved with a chain it
// presented in something it signed that it is a machine of user. It reports whether
// this made a new link, which happens only for a key this machine knows as a
// colleague: the colleague's record, with its verification and label, moves to the
// user, and the key's addresses and the rooms it joined move with it. A key linked to
// another user already links to nothing more and returns a LinkConflictError.
func (m *Membership) LinkMachineKey(key, user string) (bool, error) {
	if key == "" || user == "" || key == user {
		return false, nil
	}
	if first, ok := m.LinkedUser(key); ok {
		if first == user {
			return false, nil
		}
		if _, err := m.db.Exec(`UPDATE machine_links SET conflict_user = ?, conflict_at = ? WHERE machine_key = ?`,
			user, time.Now().UTC().Format(time.RFC3339), key); err != nil {
			return false, err
		}
		return false, &LinkConflictError{Machine: key, First: first, Second: user}
	}
	tx, err := m.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	var name, added string
	var verified sql.NullString
	err = tx.QueryRow(`SELECT name, added_at, verified_at FROM known_peers WHERE peer_id = ?`, key).Scan(&name, &added, &verified)
	if err == sql.ErrNoRows {
		return false, nil // a key nobody here knows carries nothing
	}
	if err != nil {
		return false, err
	}
	var userName string
	var userVerified sql.NullString
	err = tx.QueryRow(`SELECT name, verified_at FROM known_peers WHERE peer_id = ?`, user).Scan(&userName, &userVerified)
	switch {
	case err == sql.ErrNoRows:
		// The ordinary case: the colleague's record moves to their user whole, with
		// the label the user gave and the verification they made (D-242).
		if name == PeerName(key) {
			name = PeerName(user)
		}
		if _, err := tx.Exec(`UPDATE known_peers SET peer_id = ?, name = ? WHERE peer_id = ?`, user, name, key); err != nil {
			return false, err
		}
	case err != nil:
		return false, err
	default:
		// The user is known here as well, as one who paired with this machine from a
		// machine of theirs: the key's verification and label fill what the user's
		// record lacks.
		if _, err := tx.Exec(`DELETE FROM known_peers WHERE peer_id = ?`, key); err != nil {
			return false, err
		}
		if !userVerified.Valid && verified.Valid {
			if _, err := tx.Exec(`UPDATE known_peers SET verified_at = ? WHERE peer_id = ?`, verified.String, user); err != nil {
				return false, err
			}
		}
		if userName == PeerName(user) && name != PeerName(key) {
			if _, err := tx.Exec(`UPDATE known_peers SET name = ? WHERE peer_id = ?`, name, user); err != nil {
				return false, err
			}
		}
	}
	if _, err := tx.Exec(`INSERT INTO machine_links (machine_key, user, linked_at) VALUES (?,?,?)`,
		key, user, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return false, err
	}
	if err := assignMachineUserIn(tx, key, user); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// --- the daemon's side ---

// userOf is the user a key speaks for in a room from before user identities: this
// machine's own user for its own key, the user a colleague's key is linked to, and
// the key itself otherwise.
func (d *Daemon) userOf(key string) string {
	if d.id.HasUser() && key == d.id.PeerID {
		return d.id.User
	}
	return d.members.colleagueOf(key)
}

// aliases maps each key that stands for a user in this machine's rooms to the user:
// its own key, and each colleague's linked key.
func (d *Daemon) aliases() map[string]string {
	out := d.members.Links()
	if d.id.HasUser() {
		out[d.id.PeerID] = d.id.User
	}
	return out
}

// carryMachineKey links a machine key that has just shown, by signing, that it
// speaks for user. A new link changes whose events count in every room, so each
// room's list is computed again, now when wait is true, and otherwise once the
// caller, a TLS handshake that must not wait on the daemon's lock, has returned.
func (d *Daemon) carryMachineKey(key, user string, wait bool) error {
	linked, err := d.members.LinkMachineKey(key, user)
	if err != nil {
		var conflict *LinkConflictError
		if errors.As(err, &conflict) {
			log.Printf("identity: %v", err)
		}
		return err
	}
	if !linked {
		return nil
	}
	log.Printf("identity: %s, which you knew as a colleague, proved it is a machine of %s; its verification, label and turns are theirs now",
		PeerName(key), PeerName(user))
	refresh := func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.refreshEventListed()
	}
	if wait {
		refresh()
	} else {
		d.inBackground(refresh)
	}
	return nil
}

// refreshEventListed computes again the list of every room whose list is kept as
// events. Called with d.mu held.
func (d *Daemon) refreshEventListed() {
	rooms, err := d.members.Rooms()
	if err != nil {
		return
	}
	for _, r := range rooms {
		if !d.members.EventListed(r.RoomID) {
			continue
		}
		store, err := d.storeFor(r.RoomID)
		if err != nil {
			continue
		}
		if _, err := d.refreshGuestList(store, r.RoomID); err != nil {
			log.Printf("guests: could not compute the list of %s again: %v", r.RoomName, err)
		}
	}
}

// guestInputs is what a room's list is computed from: its guest events, each subject
// read as the user its key is linked to, and, for a room from before user identities
// on a machine that speaks for a user, its founders (D-243): the users whose machines
// signed its turns under the scheme from before user identities, each key read as its
// user once it has proved it belongs to one (D-242).
func (d *Daemon) guestInputs(store *Store, roomID string) ([]GuestEvent, []string, error) {
	evs, err := store.GuestEvents(roomID)
	if err != nil {
		return nil, nil, err
	}
	founded := false
	for i := range evs {
		e := &evs[i]
		if e.Kind == GuestInvite && e.Author == e.Subject && len(e.Nonce) > 0 && RoomIDFor(e.Author, e.Nonce) == roomID {
			founded = true
		}
		e.Subject = d.userOf(e.Subject)
	}
	if founded || !d.id.HasUser() {
		return evs, nil, nil
	}
	keys, err := store.legacyAuthors(roomID)
	if err != nil {
		return nil, nil, err
	}
	seen := map[string]bool{}
	var founders []string
	for _, k := range keys {
		if u := d.userOf(k); !seen[u] {
			seen[u] = true
			founders = append(founders, u)
		}
	}
	sort.Strings(founders)
	return evs, founders, nil
}

// resolveRoomGuests computes a room's list from what its store holds, with the keys
// that stand for users, so that Counts reads an event a key signed before its user
// existed as the user's.
func (d *Daemon) resolveRoomGuests(store *Store, roomID string) (*GuestList, []GuestEvent, error) {
	evs, founders, err := d.guestInputs(store, roomID)
	if err != nil {
		return nil, nil, err
	}
	g := ResolveGuests(roomID, evs, founders...)
	g.Aliases = d.aliases()
	return g, evs, nil
}

// legacyAuthors is every key that signed an event of the room under a scheme from
// before user identities, which is the key of the machine that wrote it.
func (s *Store) legacyAuthors(roomID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT peer_id FROM events
		WHERE room_id = ? AND COALESCE(sig_version, 0) <> 4 AND event_type NOT IN (?, ?) ORDER BY peer_id`,
		roomID, GuestInvite, GuestRevoke)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// carryRooms carries every room from before user identities into them, on a machine
// that speaks for a user. It is run when the machine becomes one of its user's and each
// time the daemon starts, and does nothing for a room already carried but invite a
// guest its earlier list admitted whom it has not invited yet. Called with d.mu held.
func (d *Daemon) carryRooms() {
	if !d.id.HasUser() {
		return
	}
	rooms, err := d.members.Rooms()
	if err != nil {
		log.Printf("guests: could not list the rooms to carry into your user identity: %v", err)
		return
	}
	for _, r := range rooms {
		if err := d.carryRoom(r); err != nil {
			log.Printf("guests: %s was not carried into your user identity: %v", r.RoomName, err)
		}
	}
}

// carryRoom carries one room from before user identities into them (D-243). Its list
// is founded by the users whose machines wrote its turns, so a room with no turns is
// not carried and a room founded under user identities has nothing to carry. This
// machine invites every user its own earlier list admitted, so that a guest who
// joined and never spoke is on the list too, and records the machines that wrote the
// room's turns and the guests on its earlier list as machines that joined it, so
// that they stay its sync targets (D-103, a room's sync targets are the machines from
// which a session has joined it). From then on the room's list is its events', and
// its earlier list admits nobody. Called with d.mu held.
func (d *Daemon) carryRoom(room Room) error {
	store, err := d.storeFor(room.RoomID)
	if err != nil {
		return err
	}
	evs, founders, err := d.guestInputs(store, room.RoomID)
	if err != nil || len(founders) == 0 {
		return err
	}
	keys, err := store.legacyAuthors(room.RoomID)
	if err != nil {
		return err
	}
	guests, err := d.members.legacyGuests(room.RoomID)
	if err != nil {
		return err
	}
	for _, k := range append(keys, guests...) {
		if k == d.id.PeerID || d.userOf(k) == d.id.User {
			continue
		}
		if _, err := PublicFromPeerID(k); err != nil {
			continue
		}
		if err := d.members.RecordRoomMachine(room.RoomID, k, d.userOf(k)); err != nil {
			return err
		}
	}
	invited := map[string]bool{}
	for _, e := range evs {
		if e.Kind == GuestInvite && e.Author == d.id.User {
			invited[e.Subject] = true
		}
	}
	for _, k := range guests {
		u := d.userOf(k)
		if u == d.id.User || invited[u] {
			continue
		}
		if _, err := PublicFromPeerID(u); err != nil {
			continue
		}
		if _, _, err := d.appendGuestEvent(store, room, GuestInvite, guestContent{User: u}); err != nil {
			return fmt.Errorf("inviting %s, whom the room's earlier list admitted: %w", PeerName(u), err)
		}
		invited[u] = true
	}
	_, err = d.refreshGuestList(store, room.RoomID)
	return err
}

// legacyGuests is the rows of a room's earlier list, room_guests.
func (m *Membership) legacyGuests(roomID string) ([]string, error) {
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

// fromBeforeUsers reports whether a room has no founding self-invitation among its
// events, which is so of every room made before user identities.
func (d *Daemon) fromBeforeUsers(roomID string) bool {
	store, err := d.storeFor(roomID)
	if err != nil {
		return false
	}
	evs, err := store.GuestEvents(roomID)
	if err != nil {
		return false
	}
	for _, e := range evs {
		if e.Kind == GuestInvite && e.Author == e.Subject && len(e.Nonce) > 0 && RoomIDFor(e.Author, e.Nonce) == roomID {
			return false
		}
	}
	return true
}
