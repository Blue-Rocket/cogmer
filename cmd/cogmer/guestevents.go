package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
)

// --- a room's guest list as events in the room ---
//
// A room founded by a machine with a user identity keeps its guest list as
// invitation and revocation events among its other events, signed under v4 by a
// guest's machine, synchronized and relayed like any turn, and never given to an
// assistant (D-235, a room's one guest list, changed by any guest through events in
// the room, decides whose events count). ResolveGuests computes the list from them,
// and membership.db caches it, so that every check on a hot path reads a table.
//
// A room with no valid founding self-invitation among its events, which is every
// room a machine with no user identity made, keeps room_guests as its list on a
// machine that speaks for no user. A machine that speaks for a user carries such a
// room into one list kept as events, founded by the users whose machines wrote its
// turns (carryRoom; D-243, a room from before user identities takes one guest list).

// guestFormat is the version of the content a guest event carries. Every guest
// event names it, and a format this build does not know is not a guest event to it,
// so that a later format is added beside this one and never edits it.
const guestFormat = 1

// guestContent is a guest event's content, as JSON (design section 3.3).
type guestContent struct {
	Format int `json:"format"`
	// User is the invited or revoked user.
	User string `json:"user"`
	// Lifts, on an invitation, names by event identifier every revocation of User it
	// lifts, so that inviting someone back works (D-235).
	Lifts []string `json:"lifts,omitempty"`
	// Nonce, on the founding self-invitation only, is the random value the room's
	// identifier hashes with its creator, in base64url (D-240).
	Nonce string `json:"nonce,omitempty"`
	// Cutoffs, on a revocation, maps each of User's machine keys to the last of that
	// machine's sequences that counts (D-236, a revocation names the last turn of each
	// of the revoked user's machines that counts).
	Cutoffs map[string]int64 `json:"cutoffs,omitempty"`
}

// isGuestEventType reports whether an event type is a change to a room's guest
// list, which is never a turn of the room.
func isGuestEventType(t string) bool {
	return t == GuestInvite || t == GuestRevoke
}

// guestEventFrom reads a stored or received event as a change to its room's guest
// list. It reports false for any other event, for one not signed under v4, since only
// a machine that speaks for a user writes one, and for content that does not decode
// as its kind. It checks no signature: the caller stores only events that verified.
func guestEventFrom(ev Event) (GuestEvent, bool) {
	if !isGuestEventType(ev.EventType) || ev.SigVersion != 4 || ev.Signer == "" {
		return GuestEvent{}, false
	}
	var c guestContent
	dec := json.NewDecoder(bytes.NewReader([]byte(ev.Content)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil || dec.More() || c.Format != guestFormat {
		return GuestEvent{}, false
	}
	if _, err := PublicFromPeerID(c.User); err != nil {
		return GuestEvent{}, false
	}
	g := GuestEvent{
		EventID: ev.EventID, Author: ev.PeerID, Machine: ev.signerKey(), Sequence: ev.PeerSequence,
		Kind: ev.EventType, Subject: c.User,
	}
	switch ev.EventType {
	case GuestInvite:
		if len(c.Cutoffs) > 0 {
			return GuestEvent{}, false
		}
		if c.Nonce != "" {
			nonce, err := base64.RawURLEncoding.DecodeString(c.Nonce)
			if err != nil || len(nonce) == 0 {
				return GuestEvent{}, false
			}
			g.Nonce = nonce
		}
		g.Lifts = c.Lifts
	case GuestRevoke:
		if len(c.Lifts) > 0 || c.Nonce != "" {
			return GuestEvent{}, false
		}
		for _, seq := range c.Cutoffs {
			if seq < 0 {
				return GuestEvent{}, false
			}
		}
		g.Cutoffs = c.Cutoffs
	}
	return g, true
}

// GuestEvents is every change to a room's guest list this store holds, read as
// guest events. An event of a guest type that does not read as one is left out, as
// ResolveGuests would discard it, and so is one whose machine and sequence a
// different event also arrived with, which counts for nobody (D-235, a change to the
// list that shares its machine's sequence with a different event counts for nobody).
func (s *Store) GuestEvents(room string) ([]GuestEvent, error) {
	rows, err := s.db.Query(`SELECT `+selectCols+` FROM events WHERE room_id = ? AND event_type IN (?, ?)
		AND event_id NOT IN (SELECT held_event_id FROM event_conflicts) `+orderBy,
		room, GuestInvite, GuestRevoke)
	if err != nil {
		return nil, err
	}
	evs, err := scanEvents(rows)
	if err != nil {
		return nil, err
	}
	var out []GuestEvent
	for _, e := range evs {
		if g, ok := guestEventFrom(e); ok {
			out = append(out, g)
		}
	}
	return out, nil
}

// lastSequences is, for each machine that signed an event of user in the room, the
// highest of its sequences this store holds, which a revocation names as its cutoff
// (D-236, a revocation names the last turn that counts, as the revoking machine saw
// it). keys are the machine keys that signed for user before user existed, whose
// earlier events name the key as their peer (D-242).
func (s *Store) lastSequences(room, user string, keys ...string) (map[string]int64, error) {
	ids := append([]string{user}, keys...)
	args := []any{room}
	marks := make([]string, len(ids))
	for i, id := range ids {
		args = append(args, id)
		marks[i] = "?"
	}
	rows, err := s.db.Query(`SELECT COALESCE(signer, peer_id), MAX(peer_sequence) FROM events
		WHERE room_id = ? AND peer_id IN (`+strings.Join(marks, ",")+`) GROUP BY COALESCE(signer, peer_id)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var machine string
		var seq int64
		if err := rows.Scan(&machine, &seq); err != nil {
			return nil, err
		}
		out[machine] = seq
	}
	return out, rows.Err()
}

// refreshGuestList computes a room's list from the guest events its store holds and
// caches it, and is run whenever a guest event is stored. A room whose events found
// nothing computes an empty list, and caching it leaves the room without one, so it
// keeps room_guests. Called with d.mu held when a guest event has just been written,
// since it reads what the write left.
func (d *Daemon) refreshGuestList(store *Store, roomID string) (*GuestList, error) {
	g, evs, err := d.resolveRoomGuests(store, roomID)
	if err != nil {
		return nil, err
	}
	if err := d.members.CacheGuestList(g); err != nil {
		return nil, err
	}
	d.settleResumption(g, evs)
	return g, nil
}

// guestList is the cached list of a room whose events found it, or nil for a room
// whose list is room_guests. It is what every hot path reads.
func (d *Daemon) guestList(roomID string) *GuestList {
	g, err := d.members.CachedGuestList(roomID)
	if err != nil {
		log.Printf("guests: could not read the cached list of %s: %v", roomID, err)
		return nil
	}
	if g != nil {
		g.Aliases = d.aliases()
	}
	return g
}

// appendGuestEvent writes a change to a room's guest list as the room's next event
// from this machine and caches the list it leaves. Called with d.mu held, as every
// write to a room's database is.
func (d *Daemon) appendGuestEvent(store *Store, room Room, kind string, c guestContent) (*Event, *GuestList, error) {
	if !d.id.HasUser() {
		return nil, nil, errors.New("only a machine that speaks for a user changes a guest list kept as events")
	}
	c.Format = guestFormat
	raw, err := json.Marshal(c)
	if err != nil {
		return nil, nil, err
	}
	ev, err := d.appendLocal(store, room, "", kind, string(raw), nil)
	if err != nil {
		return nil, nil, err
	}
	g, err := d.refreshGuestList(store, room.RoomID)
	if err != nil {
		return ev, nil, err
	}
	return ev, g, nil
}

// FoundRoom creates a room on a machine that speaks for a user. The room's
// identifier is the hash of the user and a fresh random value, and its first event
// is the user's invitation of themselves carrying that value, so the creator is on
// the list by the rule every guest is, and nobody who learns the identifier can
// admit themselves (D-240, a room's identifier is the hash of its creator and a
// random value).
func (d *Daemon) FoundRoom(showOnly bool) (Room, error) {
	if !d.id.HasUser() {
		return Room{}, errors.New("this machine speaks for no user, so its rooms keep their guest list as before")
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return Room{}, err
	}
	roomID := RoomIDFor(d.id.User, nonce)
	d.mu.Lock()
	defer d.mu.Unlock()
	room, err := d.members.recordNewRoom(roomID, showOnly)
	if err != nil {
		return Room{}, err
	}
	fail := func(err error) (Room, error) {
		_ = d.members.dropRoomRecord(roomID)
		return Room{}, err
	}
	store, err := d.storeFor(roomID)
	if err != nil {
		return fail(err)
	}
	_, g, err := d.appendGuestEvent(store, room, GuestInvite, guestContent{
		User: d.id.User, Nonce: base64.RawURLEncoding.EncodeToString(nonce),
	})
	if err != nil {
		return fail(fmt.Errorf("founding the room: %w", err))
	}
	if !g.Admitted[d.id.User] {
		return fail(errors.New("the founding invitation did not put its creator on the room's list"))
	}
	if err := d.members.RecordRoomMachine(roomID, d.id.PeerID, d.id.User); err != nil {
		return Room{}, err
	}
	// The room is the user's, so it goes on the room list the user's machines share,
	// with this machine as one that joined it (D-228).
	if err := d.members.RecordRoomOnList(d.id, roomID, room.RoomName, withMachine(d.id.PeerID)); err != nil {
		log.Printf("guests: %s was not written to your room list: %v", room.RoomName, err)
	}
	return room, nil
}

// eventListedRoom is a room whose list is kept as events, with its store, for a
// change to the list. Called with d.mu held.
func (d *Daemon) eventListedRoom(roomID string) (Room, *Store, error) {
	if !d.id.HasUser() {
		return Room{}, nil, errors.New("this machine speaks for no user, so it cannot change a guest list kept as events")
	}
	room, ok := d.members.RoomByID(roomID)
	if !ok {
		return Room{}, nil, errNoSuchRoom
	}
	if !d.members.EventListed(roomID) {
		return Room{}, nil, fmt.Errorf("%s keeps its guest list in room_guests, not as events", room.RoomName)
	}
	store, err := d.storeFor(roomID)
	return room, store, err
}

// InviteUser adds user to a room's list with an invitation that lifts every counted
// revocation of user this machine holds, so that inviting back somebody who was
// revoked works (D-235). A user who is not on the list cannot invite, since their
// invitation would count for nothing past their cutoff, and a user cannot invite
// themselves, since a self-invitation counts only at founding (D-240).
func (d *Daemon) InviteUser(roomID, user string) error {
	if _, err := PublicFromPeerID(user); err != nil {
		return fmt.Errorf("refusing to invite an identifier that names no key: %w", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	room, store, err := d.eventListedRoom(roomID)
	if err != nil {
		return err
	}
	user = d.userOf(user)
	if user == d.id.User {
		return errors.New("you cannot invite yourself; another guest can invite you back")
	}
	g, _, err := d.resolveRoomGuests(store, roomID)
	if err != nil {
		return err
	}
	if !g.Admitted[d.id.User] {
		return fmt.Errorf("you are not on the guest list of %s, so an invitation of yours would not count", room.RoomName)
	}
	var lifts []string
	for _, c := range g.Changes {
		if c.Kind == GuestRevoke && c.Subject == user {
			lifts = append(lifts, c.EventID)
		}
	}
	sort.Strings(lifts)
	_, _, err = d.appendGuestEvent(store, room, GuestInvite, guestContent{User: user, Lifts: lifts})
	return err
}

// RevokeUser expels user from a room with a revocation whose cutoff for each machine
// of theirs is the highest of that machine's sequences this machine holds in the
// room, so that every guest keeps the same history of what they wrote before it
// (D-236, a revocation names the last turn of each of the revoked user's machines
// that counts).
func (d *Daemon) RevokeUser(roomID, user string) error {
	if _, err := PublicFromPeerID(user); err != nil {
		return fmt.Errorf("refusing to revoke an identifier that names no key: %w", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	room, store, err := d.eventListedRoom(roomID)
	if err != nil {
		return err
	}
	user = d.userOf(user)
	cutoffs, err := store.lastSequences(roomID, user, d.members.linkedKeys(user)...)
	if err != nil {
		return err
	}
	_, _, err = d.appendGuestEvent(store, room, GuestRevoke, guestContent{User: user, Cutoffs: cutoffs})
	return err
}

// guestChangeText is what the room view shows for a change to the guest list, which
// is a membership change and not a turn.
func guestChangeText(e Event, labels map[string]string) string {
	g, ok := guestEventFrom(e)
	if !ok {
		return "a change to the guest list this build cannot read"
	}
	who := firstNonEmpty(labels[g.Subject], PeerName(g.Subject))
	switch {
	case g.Kind == GuestInvite && g.Subject == g.Author && len(g.Nonce) > 0:
		return "founded the room"
	case g.Kind == GuestInvite:
		return "invited " + who
	default:
		return "revoked " + who
	}
}

// --- the local route a command uses to change a guest list ---

type guestChangeRequest struct {
	// Op is "found", "invite" or "revoke".
	Op       string `json:"op"`
	RoomID   string `json:"roomId,omitempty"`
	User     string `json:"user,omitempty"`
	ShowOnly bool   `json:"showOnly,omitempty"`
}

type guestChangeResponse struct {
	Room  *Room  `json:"room,omitempty"`
	Error string `json:"error,omitempty"`
}

// handleGuestChange lets a command change a guest list kept as events. The command
// runs in a process of its own, and the daemon writes the event, since every write
// to a room's database is made under the daemon's lock.
func (d *Daemon) handleGuestChange(w http.ResponseWriter, r *http.Request) {
	var req guestChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var err error
	var out guestChangeResponse
	switch req.Op {
	case "found":
		var room Room
		if room, err = d.FoundRoom(req.ShowOnly); err == nil {
			out.Room = &room
		}
	case "invite":
		err = d.InviteUser(req.RoomID, req.User)
	case "revoke":
		err = d.RevokeUser(req.RoomID, req.User)
	default:
		err = fmt.Errorf("%q is not a change to a guest list", req.Op)
	}
	if err != nil {
		out.Error = err.Error()
	} else {
		d.notify()
	}
	writeJSON(w, out)
}
