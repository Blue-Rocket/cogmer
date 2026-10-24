package main

import (
	"errors"
	"log"
	"net/http"
	"sort"
	"strings"
)

// --- a user expelled from a room, as the room's guests tell them and as they act on it ---
//
// A guest's machine that refuses a sync request because the asking user was revoked
// sends the revocations that expelled them with the refusal, and nothing else of the
// room, so that each of the expelled user's machines learns why the next time it asks,
// one that was offline when the revocation was made included. The expelled user's
// machine checks and stores them, and stops serving the room to anyone and asking for
// it, since the room's other guests hold its history (D-241, an expelled user is told,
// through the refusal that carries the revocation). What the page, the room status
// and the prompt say of it is read through Expulsion.

// revocationRefusal is the body of a refused sync request to a revoked user.
type revocationRefusal struct {
	Protocol int    `json:"protocol"`
	RoomID   string `json:"roomId"`
	// Revocations are the counted GUEST_REVOKE events of the asking user, as signed.
	Revocations []wireEvent `json:"revocations"`
}

// revocationsOf is the counted revocations of user in a room whose list is kept as
// events, as the store holds them, or none when the list admits user or never did.
func (d *Daemon) revocationsOf(store *Store, roomID, user string) ([]Event, error) {
	gevs, err := store.GuestEvents(roomID)
	if err != nil || len(gevs) == 0 {
		return nil, err
	}
	g := ResolveGuests(roomID, gevs)
	if admitted, ever := g.Admitted[user]; !ever || admitted {
		return nil, nil
	}
	ids := map[string]bool{}
	for _, c := range g.Changes {
		if c.Kind == GuestRevoke && c.Subject == user {
			ids[c.EventID] = true
		}
	}
	rows, err := store.db.Query(`SELECT `+selectCols+` FROM events WHERE room_id = ? AND event_type = ? `+orderBy,
		roomID, GuestRevoke)
	if err != nil {
		return nil, err
	}
	all, err := scanEvents(rows)
	if err != nil {
		return nil, err
	}
	var out []Event
	for _, e := range all {
		if ids[e.EventID] {
			out = append(out, e)
		}
	}
	return out, nil
}

// refuseNonGuest answers a sync request whose asker is established and is not on the
// room's list. A revoked user is sent the revocations that expelled them, and nothing
// else of the room; anybody else is told only "unauthenticated", as before, since the
// room is none of their business.
func (d *Daemon) refuseNonGuest(w http.ResponseWriter, room Room, user string) {
	if d.members.EventListed(room.RoomID) {
		if store, err := d.storeFor(room.RoomID); err == nil {
			if revs, err := d.revocationsOf(store, room.RoomID, user); err == nil && len(revs) > 0 {
				out := revocationRefusal{Protocol: wireVersion, RoomID: room.RoomID}
				for _, e := range revs {
					out.Revocations = append(out.Revocations, toWire(e))
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

// errNotARevocationOfUs is a refusal's event that is not a revocation of this
// machine's own user, which a refusal has no business carrying.
var errNotARevocationOfUs = errors.New("a refusal carried something other than a revocation of this machine's user")

// keepRevocations checks and stores the revocations a refusal carried, by the rules
// any guest event is stored by in pullRoom, and only those that revoke this machine's
// own user, then computes the room's list again. It returns how many were new. Called
// with d.mu held, since it writes the room's database.
func (d *Daemon) keepRevocations(store *Store, room Room, addr string, revs []wireEvent) int {
	if !d.id.HasUser() {
		return 0
	}
	stored := 0
	for _, w := range revs {
		ev := fromWire(w)
		if ev.RoomID != room.RoomID || ev.EventType != GuestRevoke {
			log.Printf("sync: %s refused %s with %v", shortEndpoint(addr), room.RoomName, errNotARevocationOfUs)
			continue
		}
		if err := ev.Verify(); err != nil {
			log.Printf("sync: REJECTED a revocation %.12s from %s — %v", ev.EventID, addr, err)
			continue
		}
		g, ok := guestEventFrom(ev)
		if !ok || g.Subject != d.id.User {
			log.Printf("sync: %s refused %s with %v", shortEndpoint(addr), room.RoomName, errNotARevocationOfUs)
			continue
		}
		if err := d.judgeEvent(&ev); err != nil {
			log.Printf("sync: a revocation %.12s from %s was not kept — %v", ev.EventID, addr, err)
			continue
		}
		if res, err := store.Insert(&ev); err == nil && res == InsertStored {
			stored++
		}
	}
	if stored > 0 {
		if _, err := d.refreshGuestList(store, room.RoomID); err != nil {
			log.Printf("sync: could not compute the guest list of %s: %v", room.RoomName, err)
		}
		log.Printf("sync: %s refused %s and sent the revocation that expelled you; this machine stops serving the room",
			shortEndpoint(addr), room.RoomName)
		d.notify()
	}
	return stored
}

// expelledFrom reports whether this machine's own user was invited to a room whose
// list is kept as events and is not on it now. Such a machine serves the room to
// nobody and asks nobody for it (D-241).
func (d *Daemon) expelledFrom(roomID string) bool {
	if !d.id.HasUser() {
		return false
	}
	g := d.guestList(roomID)
	if g == nil {
		return false
	}
	admitted, ever := g.Admitted[d.id.User]
	return ever && !admitted
}

// Expulsion is what this machine holds of its own user's removal from a room.
type Expulsion struct {
	// Expelled is whether the room's list, as this machine computes it, has expelled
	// this machine's user. A revocation held whose author this machine cannot place on
	// the list, such as one whose invitation it never received, leaves it false.
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
	out.Expelled = d.expelledFrom(roomID)
	return out, true
}
