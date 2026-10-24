package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"sort"
)

// A room's guest list, computed from the invitation and revocation events in
// the room (D-235, a room's one guest list, changed by any guest through events
// in the room, decides whose events count). Every machine holding the same
// events computes the same list, whatever order they arrived in, because
// nothing here reads anything but the events and the room's identifier.
//
// The events reaching this layer are authentic: the caller has checked each
// signature and chain, so Author is the user whose machine signed it. This layer
// decides membership only.

const (
	GuestInvite = "GUEST_INVITE"
	GuestRevoke = "GUEST_REVOKE"
)

type GuestEvent struct {
	EventID  string
	Author   string // the user who signed it (master identifier)
	Machine  string // the key of the machine that signed it
	Sequence int64  // that machine's sequence in the room
	Kind     string // GuestInvite or GuestRevoke
	Subject  string // the invited or revoked user
	// Lifts, on a GuestInvite, names the revocations of Subject it lifts, by
	// event identifier.
	Lifts []string
	// Nonce, on the founding self-invitation only, is the random value the
	// room's identifier hashes with its creator (D-240).
	Nonce []byte
	// Cutoffs, on a GuestRevoke, maps each of Subject's machine keys to the
	// machine's last sequence that counts (D-236).
	Cutoffs map[string]int64
}

type GuestList struct {
	Room string
	// Admitted holds every user ever invited, to whether the user is on the list
	// now. A user absent was never invited, which Counts tells apart from a user
	// who was and has been expelled.
	Admitted map[string]bool
	// Cutoffs maps each revoked user to their machines' last counting
	// sequences. A machine absent, like one at 0, counts nothing.
	Cutoffs   map[string]map[string]int64
	Changes   []GuestEvent // the counted changes, sorted, for the view
	Discarded []Discarded
}

// guestKey is a canonical encoding of an event, which orders events and finds
// one event given twice. Lifts is a set, so it is sorted first; JSON writes a
// map's keys in order.
func guestKey(e GuestEvent) string {
	raw, _ := json.Marshal(e)
	return string(raw)
}

// normalGuestEvent copies e's slices and map, so nothing here changes the
// caller's events, and gives an empty one a single form, so that an event
// decoded with an empty list and the same event with none are one event.
func normalGuestEvent(e GuestEvent) GuestEvent {
	lifts, cutoffs, nonce := e.Lifts, e.Cutoffs, e.Nonce
	e.Lifts, e.Cutoffs, e.Nonce = nil, nil, nil
	if len(nonce) > 0 {
		e.Nonce = append([]byte{}, nonce...)
	}
	if len(lifts) > 0 {
		e.Lifts = append([]string{}, lifts...)
		sort.Strings(e.Lifts)
	}
	if len(cutoffs) > 0 {
		c := make(map[string]int64, len(cutoffs))
		for k, v := range cutoffs {
			c[k] = v
		}
		e.Cutoffs = c
	}
	return e
}

// ResolveGuests computes room's guest list from its guest events. The steps run
// in an order in which none needs a later one's result, so the list has one
// answer and no circle: who was ever invited, then which revocations count,
// then the cutoffs, then which invitations count, then who is on the list.
func ResolveGuests(room string, events []GuestEvent) *GuestList {
	g := &GuestList{
		Room:     room,
		Admitted: map[string]bool{},
		Cutoffs:  map[string]map[string]int64{},
	}
	discard := func(e GuestEvent, reason string) {
		g.Discarded = append(g.Discarded, Discarded{
			Hash: sha256.Sum256([]byte(e.EventID)), Kind: "guest", Reason: "event " + e.EventID + ": " + reason,
		})
	}

	// Sorting first and dropping an event given twice makes every loop below
	// see the same events in the same order, however they arrived.
	type keyed struct {
		key string
		e   GuestEvent
	}
	var all []keyed
	for _, e := range events {
		e = normalGuestEvent(e)
		all = append(all, keyed{guestKey(e), e})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].e.EventID != all[j].e.EventID {
			return all[i].e.EventID < all[j].e.EventID
		}
		return all[i].key < all[j].key
	})
	var founding, invites, revokes []GuestEvent
	for i, k := range all {
		if i > 0 && all[i-1].key == k.key {
			continue
		}
		e := k.e
		switch e.Kind {
		case GuestInvite:
			// a. Founding. A self-invitation counts only when it hashes to the
			// room's identifier, which only the creator can make it do, and it
			// lifts nothing, so it works once, when the room is founded (D-240, a
			// room's identifier is the hash of its creator and a random value).
			switch {
			case e.Author == e.Subject:
				if RoomIDFor(e.Author, e.Nonce) != room {
					discard(e, "a self-invitation that does not hash to the room's identifier, so it founds nothing (D-240, a room's identifier is the hash of its creator and a random value)")
					continue
				}
				e.Lifts = nil
				founding = append(founding, e)
			case len(e.Nonce) > 0:
				discard(e, "an invitation of another user carries a founding value, which only a self-invitation may")
			default:
				invites = append(invites, e)
			}
		case GuestRevoke:
			revokes = append(revokes, e)
		default:
			discard(e, "not a guest-list change")
		}
	}

	// b. Ever invited, ignoring cutoffs, since they come from revocations,
	// which count by this set. It only grows, so the loop ends.
	ever := map[string]bool{}
	for _, e := range founding {
		ever[e.Subject] = true
	}
	for grew := true; grew; {
		grew = false
		for _, e := range invites {
			if ever[e.Author] && !ever[e.Subject] {
				ever[e.Subject] = true
				grew = true
			}
		}
	}

	// c. Every revocation by a user ever invited counts, whatever has become of
	// its author since, so two guests who revoke each other are both out, and
	// neither revocation depends on the other (D-235).
	revsOf := map[string]map[string]bool{} // revoked user to counted revocations' identifiers
	var counted []GuestEvent
	for _, e := range revokes {
		if !ever[e.Author] {
			discard(e, "a revocation by a user who was never invited to the room")
			continue
		}
		counted = append(counted, e)
		if revsOf[e.Subject] == nil {
			revsOf[e.Subject] = map[string]bool{}
		}
		revsOf[e.Subject][e.EventID] = true
	}

	// d. Each revoked user's machines take the lowest cutoff any counted
	// revocation gives them, a machine a revocation leaves out getting 0 from
	// it, so a cutoff only moves earlier (D-236, a revocation names the last
	// turn of each of the revoked user's machines that counts), as a removal's
	// does (D-220). A machine whose lowest is 0 is left out, which reads the
	// same.
	for user := range revsOf {
		var mine []GuestEvent
		machines := map[string]bool{}
		for _, e := range counted {
			if e.Subject == user {
				mine = append(mine, e)
				for m := range e.Cutoffs {
					machines[m] = true
				}
			}
		}
		cut := map[string]int64{}
		for m := range machines {
			low := mine[0].Cutoffs[m]
			for _, e := range mine[1:] {
				low = min(low, e.Cutoffs[m])
			}
			if low > 0 {
				cut[m] = low
			}
		}
		g.Cutoffs[user] = cut
	}

	// e. An invitation counts when its author was ever invited and made it
	// within the author's own cutoff, if the author has been revoked, so that
	// an expelled user cannot let in an accomplice after the cutoff.
	for _, e := range invites {
		if !ever[e.Author] {
			discard(e, "an invitation by a user who was never invited to the room")
			continue
		}
		if cut, revoked := g.Cutoffs[e.Author]; revoked {
			if last, ok := cut[e.Machine]; !ok || e.Sequence > last {
				discard(e, "an invitation its author made after the cutoff of a revocation of them (D-236)")
				continue
			}
		}
		counted = append(counted, e)
	}
	counted = append(counted, founding...)

	// f. A user is on the list when one counted invitation lifts every counted
	// revocation of them, so a revocation expels the user from every invitation
	// that did not see it (D-235). A founding self-invitation lifts nothing, so
	// it admits its creator only until a revocation of them counts.
	for user := range ever {
		g.Admitted[user] = false
	}
	for _, e := range counted {
		if e.Kind != GuestInvite {
			continue
		}
		lifted := map[string]bool{}
		for _, id := range e.Lifts {
			lifted[id] = true
		}
		all := true
		for id := range revsOf[e.Subject] {
			if !lifted[id] {
				all = false
				break
			}
		}
		if all {
			g.Admitted[e.Subject] = true
		}
	}

	sort.Slice(counted, func(i, j int) bool {
		if counted[i].EventID != counted[j].EventID {
			return counted[i].EventID < counted[j].EventID
		}
		return guestKey(counted[i]) < guestKey(counted[j])
	})
	g.Changes = counted
	sort.Slice(g.Discarded, func(i, j int) bool {
		a, b := g.Discarded[i], g.Discarded[j]
		if c := bytes.Compare(a.Hash[:], b.Hash[:]); c != 0 {
			return c < 0
		}
		return a.Reason < b.Reason
	})
	return g
}

// Counts reports whether user's event with this sequence from machine counts in
// the room. A user on the list counts in full, including after an earlier
// cutoff, since the invitation that readmitted them lifted the revocation that
// set it (D-236). An expelled user's events count up to their machine's cutoff,
// and a user never invited counts for nothing.
func (g *GuestList) Counts(user, machine string, sequence int64) bool {
	admitted, ever := g.Admitted[user]
	if !ever {
		return false
	}
	if admitted {
		return true
	}
	last, ok := g.Cutoffs[user][machine]
	return ok && sequence <= last
}
