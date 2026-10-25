package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

// --- a user expelled from a room, as the room's guests tell them and as they act on it ---
//
// A guest's machine that refuses a sync request because the asking user was revoked
// sends with the refusal the room's guest-list events, every invitation and
// revocation it holds, and none of the room's conversation, so that each of the
// expelled user's machines learns why the next time it asks, one that was offline
// when the revocation was made included. The invitations are sent with the
// revocations because a revocation counts only when its author is on the list, and
// an expelled machine may never have received the invitation that admitted the
// guest who revoked it. The expelled user's machine checks and stores them, and stops
// serving the room to anyone and asking for it, since the room's other guests hold
// its history (D-241, an expelled user is told, through the refusal that carries the
// revocation). What the page, the room status and the prompt say of it is read
// through Expulsion.

// revocationRefusal is the body of a refused sync request to a revoked user.
type revocationRefusal struct {
	Protocol int    `json:"protocol"`
	RoomID   string `json:"roomId"`
	// GuestEvents are every GUEST_INVITE and GUEST_REVOKE event the refusing machine
	// holds for the room, as signed.
	GuestEvents []wireEvent `json:"guestEvents"`
}

// revokedIn reports whether a room whose list is kept as events has revoked user,
// who was invited and is not on the list now.
func (d *Daemon) revokedIn(store *Store, roomID, user string) bool {
	g, _, err := d.resolveRoomGuests(store, roomID)
	if err != nil {
		return false
	}
	admitted, ever := g.Admitted[d.userOf(user)]
	return ever && !admitted
}

// guestListEvents is every invitation and revocation a store holds for a room, as
// stored, which is the whole of what a refusal to a revoked user carries.
func guestListEvents(store *Store, roomID string) ([]Event, error) {
	rows, err := store.db.Query(`SELECT `+selectCols+` FROM events WHERE room_id = ? AND event_type IN (?, ?) `+orderBy,
		roomID, GuestInvite, GuestRevoke)
	if err != nil {
		return nil, err
	}
	return scanEvents(rows)
}

// refuseNonGuest answers a sync request whose asker is established and is not on the
// room's list. A revoked user is sent the room's guest-list events, and nothing of
// its conversation; anybody else is told only "unauthenticated", as before, since the
// room is none of their business.
func (d *Daemon) refuseNonGuest(w http.ResponseWriter, room Room, user string) {
	if d.members.EventListed(room.RoomID) {
		if store, err := d.storeFor(room.RoomID); err == nil && d.revokedIn(store, room.RoomID, user) {
			if evs, err := guestListEvents(store, room.RoomID); err == nil && len(evs) > 0 {
				out := revocationRefusal{Protocol: wireVersion, RoomID: room.RoomID}
				for _, e := range evs {
					out.GuestEvents = append(out.GuestEvents, toWire(e))
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				writeJSON(w, out)
				return
			}
		}
	}
	http.Error(w, "unauthenticated", http.StatusUnauthorized)
}

// errNotAGuestEvent is a refusal's event that is not a change to the room's guest
// list, which a refusal has no business carrying.
var errNotAGuestEvent = errors.New("a refusal carried something other than a change to the room's guest list")

// keepGuestEvents checks and stores the guest-list events a refusal carried, by the
// rules any guest event is stored by in pullRoom, then computes the room's list
// again. It returns how many were new. Called with d.mu held, since it writes the
// room's database.
func (d *Daemon) keepGuestEvents(store *Store, room Room, addr string, evs []wireEvent) int {
	if !d.id.HasUser() {
		return 0
	}
	stored, revokedUs := 0, false
	for _, w := range evs {
		ev := fromWire(w)
		if ev.RoomID != room.RoomID || !isGuestEventType(ev.EventType) {
			log.Printf("sync: %s refused %s with %v", shortEndpoint(addr), room.RoomName, errNotAGuestEvent)
			continue
		}
		if err := ev.Verify(); err != nil {
			log.Printf("sync: REJECTED a guest-list change %.12s from %s — %v", ev.EventID, addr, err)
			continue
		}
		g, ok := guestEventFrom(ev)
		if !ok {
			log.Printf("sync: %s refused %s with %v", shortEndpoint(addr), room.RoomName, errNotAGuestEvent)
			continue
		}
		if err := d.judgeEvent(&ev); err != nil {
			log.Printf("sync: a guest-list change %.12s from %s was not kept — %v", ev.EventID, addr, err)
			continue
		}
		if res, err := store.Insert(&ev); err == nil && res == InsertStored {
			stored++
			revokedUs = revokedUs || (g.Kind == GuestRevoke && g.Subject == d.id.User)
		}
	}
	if stored > 0 {
		if _, err := d.refreshGuestList(store, room.RoomID); err != nil {
			log.Printf("sync: could not compute the guest list of %s: %v", room.RoomName, err)
		}
		if revokedUs {
			log.Printf("sync: %s refused %s and sent the revocation that expelled you; this machine stops serving the room",
				shortEndpoint(addr), room.RoomName)
		}
		d.notify()
	}
	return stored
}

// expelledFrom reports whether this machine serves a room to nobody and asks nobody
// for it, because its own user was invited to the room, whose list is kept as events,
// and is not on it now (D-241). A room an admitted guest has offered again since is
// served and asked for, so that the invitation lifting the revocation arrives by the
// next sync and the turns captured meanwhile reach the room.
func (d *Daemon) expelledFrom(roomID string) bool {
	g, expelled := d.listExpels(roomID)
	if !expelled {
		return false
	}
	host, _, ok := d.members.Resumption(roomID)
	return !ok || !g.Admitted[host]
}

// listExpels reports whether a room's list, as this machine computes it, has
// expelled this machine's own user, with the list.
func (d *Daemon) listExpels(roomID string) (*GuestList, bool) {
	if !d.id.HasUser() {
		return nil, false
	}
	g := d.guestList(roomID)
	if g == nil {
		return nil, false
	}
	admitted, ever := g.Admitted[d.id.User]
	return g, ever && !admitted
}

// revocationsOfUs is the identifiers of every revocation of this machine's user among
// a room's guest events, sorted.
func (d *Daemon) revocationsOfUs(gevs []GuestEvent) []string {
	var out []string
	for _, g := range gevs {
		if g.Kind == GuestRevoke && g.Subject == d.id.User {
			out = append(out, g.EventID)
		}
	}
	sort.Strings(out)
	return out
}

// resumeOnOffer is what an offer for a room this machine's user was expelled from
// does, when the guest who offered it is on the room's list: the machine syncs and
// serves the room again, and learns where the offering machine listens, so that the
// invitation that lifted the revocation arrives by the next sync (D-105, an
// invitation is delivered as an offer; D-241, an invitation that lifts the revocation
// lets the expelled user's turns reach the room). It reports whether it resumed. An
// offer from a guest the list does not admit resumes nothing, since only an admitted
// guest's invitation could let the user back in.
func (d *Daemon) resumeOnOffer(req offerRequest) bool {
	g, expelled := d.listExpels(req.RoomID)
	if !expelled || !g.Admitted[req.PeerID] {
		return false
	}
	store, err := d.storeFor(req.RoomID)
	if err != nil {
		return false
	}
	gevs, err := store.GuestEvents(req.RoomID)
	if err != nil {
		return false
	}
	if err := d.members.RecordResumption(req.RoomID, req.PeerID, d.revocationsOfUs(gevs)); err != nil {
		log.Printf("offer: could not resume %s: %v", req.RoomID, err)
		return false
	}
	machine := req.PeerID
	if req.Signer != "" {
		machine = req.Signer
	}
	if err := d.members.RecordMachineAddress(machine, req.PeerID, req.Endpoint); err != nil {
		log.Printf("offer: could not record where %s listens: %v", PeerName(machine), err)
	}
	if err := d.members.RecordRoomMachine(req.RoomID, machine, req.PeerID); err != nil {
		log.Printf("offer: could not record %s as a machine of %s: %v", PeerName(machine), req.RoomID, err)
	}
	return true
}

// settleResumption ends a room's resumption once it has done its work, when the list
// admits this machine's user again, or once it no longer applies, when a revocation
// of the user arrives that was not held when the offer came. Run whenever the room's
// list is computed again.
func (d *Daemon) settleResumption(g *GuestList, gevs []GuestEvent) {
	if !d.id.HasUser() {
		return
	}
	_, revs, ok := d.members.Resumption(g.Room)
	if !ok {
		return
	}
	held := map[string]bool{}
	for _, r := range revs {
		held[r] = true
	}
	end := g.Admitted[d.id.User]
	for _, r := range d.revocationsOfUs(gevs) {
		end = end || !held[r]
	}
	if end {
		if err := d.members.DropResumption(g.Room); err != nil {
			log.Printf("guests: could not end the resumption of %s: %v", g.Room, err)
		}
	}
}

// RecordResumption marks a room this machine's user was expelled from as offered
// again by host, with the revocations of the user held at the time.
func (m *Membership) RecordResumption(roomID, host string, revocations []string) error {
	raw, err := json.Marshal(revocations)
	if err != nil {
		return err
	}
	_, err = m.db.Exec(`INSERT INTO resumed_rooms (room_id, host, revocations, resumed_at) VALUES (?,?,?,?)
		ON CONFLICT(room_id) DO UPDATE SET host = excluded.host, revocations = excluded.revocations,
			resumed_at = excluded.resumed_at`,
		roomID, host, string(raw), time.Now().UTC().Format(time.RFC3339))
	return err
}

// Resumption reads a room's resumption: the guest who offered it again and the
// revocations held then.
func (m *Membership) Resumption(roomID string) (string, []string, bool) {
	var host, raw string
	if m.db.QueryRow(`SELECT host, revocations FROM resumed_rooms WHERE room_id = ?`, roomID).Scan(&host, &raw) != nil {
		return "", nil, false
	}
	var revs []string
	if json.Unmarshal([]byte(raw), &revs) != nil {
		return "", nil, false
	}
	return host, revs, true
}

// DropResumption ends a room's resumption.
func (m *Membership) DropResumption(roomID string) error {
	_, err := m.db.Exec(`DELETE FROM resumed_rooms WHERE room_id = ?`, roomID)
	return err
}

// Expulsion is what this machine holds of its own user's removal from a room.
type Expulsion struct {
	// Expelled is whether the room's list, as this machine computes it, has expelled
	// this machine's user. A refusal carries the room's invitations with its
	// revocations, so the author of a revocation it brings can be placed on the list.
	Expelled bool
	// Revocations are every revocation of this machine's user the room holds, sorted
	// by event: who revoked them in Author and Machine, and from which of their turns
	// on, by machine, in Cutoffs.
	Revocations []GuestEvent
}

// Expulsion reads what this machine holds of its own user's removal from a room, for
// the page, the room status and the prompt (D-241). It reports false when the room
// holds no revocation of this machine's user.
func (d *Daemon) Expulsion(roomID string) (Expulsion, bool) {
	if !d.id.HasUser() {
		return Expulsion{}, false
	}
	store, err := d.storeFor(roomID)
	if err != nil {
		return Expulsion{}, false
	}
	gevs, err := store.GuestEvents(roomID)
	if err != nil {
		return Expulsion{}, false
	}
	var out Expulsion
	for _, g := range gevs {
		if g.Kind == GuestRevoke && g.Subject == d.id.User {
			out.Revocations = append(out.Revocations, g)
		}
	}
	if len(out.Revocations) == 0 {
		return Expulsion{}, false
	}
	sort.Slice(out.Revocations, func(i, j int) bool {
		return strings.Compare(out.Revocations[i].EventID, out.Revocations[j].EventID) < 0
	})
	_, out.Expelled = d.listExpels(roomID)
	return out, true
}
