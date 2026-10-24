package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"slices"
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

// guestState is what one set of counted invitations implies about the room.
type guestState struct {
	counted  []bool                      // by index into the room's invitations
	ever     map[string]bool             // the founder and every subject of a counted invitation
	revokes  []GuestEvent                // the counted revocations
	revsOf   map[string]map[string]bool  // revoked user to counted revocations' identifiers
	cutoffs  map[string]map[string]int64 // revoked user to machine key to last counting sequence
	admitted map[string]bool             // each user ever invited to whether on the list
}

func newGuestState(founding, invites, revokes []GuestEvent, counted []bool) *guestState {
	s := &guestState{
		counted:  counted,
		ever:     map[string]bool{},
		revsOf:   map[string]map[string]bool{},
		cutoffs:  map[string]map[string]int64{},
		admitted: map[string]bool{},
	}
	var in []GuestEvent // the counted invitations, the founding ones among them
	in = append(in, founding...)
	for i, e := range invites {
		if counted[i] {
			in = append(in, e)
		}
	}
	for _, e := range in {
		s.ever[e.Subject] = true
	}

	// c. Every revocation by a user ever invited counts, whatever has become of
	// its author since, so two guests who revoke each other are both out, and
	// neither revocation depends on the other (D-235, a room's one guest list).
	for _, e := range revokes {
		if !s.ever[e.Author] {
			continue
		}
		s.revokes = append(s.revokes, e)
		if s.revsOf[e.Subject] == nil {
			s.revsOf[e.Subject] = map[string]bool{}
		}
		s.revsOf[e.Subject][e.EventID] = true
	}

	// d. Each revoked user's machines take the lowest cutoff any counted
	// revocation gives them, a machine a revocation leaves out getting 0 from
	// it, so a cutoff only moves earlier (D-236, a revocation names the last
	// turn of each of the revoked user's machines that counts), as a removal's
	// does (D-220). A machine whose lowest is 0 is left out, which reads the
	// same.
	for user := range s.revsOf {
		var mine []GuestEvent
		machines := map[string]bool{}
		for _, e := range s.revokes {
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
		s.cutoffs[user] = cut
	}

	// f. A user is on the list when one counted invitation lifts every counted
	// revocation of them, so a revocation expels the user from every invitation
	// that did not see it (D-235). A founding self-invitation lifts nothing, so
	// it admits its creator only until a revocation of them counts.
	for user := range s.ever {
		s.admitted[user] = false
	}
	for _, e := range in {
		lifted := map[string]bool{}
		for _, id := range e.Lifts {
			lifted[id] = true
		}
		all := true
		for id := range s.revsOf[e.Subject] {
			if !lifted[id] {
				all = false
				break
			}
		}
		if all {
			s.admitted[e.Subject] = true
		}
	}
	return s
}

// next gives the invitations this state counts: an invitation counts when its
// author was ever invited and is on the list, or made it within their own
// cutoff, so that an expelled user cannot let in an accomplice after the cutoff
// while a readmitted user invites as any guest does (D-236, a later invitation
// that lifts the revocation lets turns after the cutoff count again).
func (s *guestState) next(invites []GuestEvent) []bool {
	out := make([]bool, len(invites))
	for i, e := range invites {
		switch {
		case !s.ever[e.Author]:
		case s.admitted[e.Author]:
			out[i] = true
		default:
			cut, revoked := s.cutoffs[e.Author]
			last, ok := cut[e.Machine]
			out[i] = !revoked || (ok && e.Sequence <= last)
		}
	}
	return out
}

// onList is the sorted list of users this state admits.
func (s *guestState) onList() []string {
	var out []string
	for u, on := range s.admitted {
		if on {
			out = append(out, u)
		}
	}
	sort.Strings(out)
	return out
}

func countedKey(counted []bool) string {
	b := make([]byte, len(counted))
	for i, c := range counted {
		b[i] = '0'
		if c {
			b[i] = '1'
		}
	}
	return string(b)
}

// ResolveGuests computes room's guest list from its guest events.
//
// It works in rounds because the rules depend on one another in a circle:
// whether an invitation counts depends on its author's cutoff, a cutoff on
// which revocations count, and those on who was ever invited, which depends on
// which invitations count (D-235, a room's one guest list; D-236, a revocation
// names the last turn that counts). A single pass has to break the circle
// somewhere, and breaking it by ignoring cutoffs when deciding who was ever
// invited lets an expelled guest invite an accomplice after the cutoff, whose
// own invitations and revocations would then count. So each round starts from a
// set of counted invitations, beginning with the founding one alone, derives
// who was ever invited, the revocations, the cutoffs and the list from it, and
// takes the invitations that state counts as the next set, until a set repeats.
//
// A set that repeats at once is the answer. A record can also loop, such as a
// guest revoking, with a cutoff before it, the invitation that let them in. Then
// a user is on the list only when every state of the loop admits them, and the
// cutoffs and counted changes are those of the state admitting fewest, then
// the one whose sorted list comes first, so that a record in conflict keeps
// people out rather than letting them in, and every machine holding the same
// events settles on the same answer. The sets are finite, so a repeat comes.
func ResolveGuests(room string, events []GuestEvent) *GuestList {
	g := &GuestList{
		Room:     room,
		Admitted: map[string]bool{},
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

	// The rounds, remembering each set to find the one that repeats.
	var states []*guestState
	seen := map[string]int{}
	counted := make([]bool, len(invites))
	var loop []*guestState
	for {
		k := countedKey(counted)
		if j, ok := seen[k]; ok {
			loop = states[j:]
			break
		}
		seen[k] = len(states)
		s := newGuestState(founding, invites, revokes, counted)
		states = append(states, s)
		counted = s.next(invites)
	}

	kept := loop[0]
	for _, s := range loop[1:] {
		a, b := s.onList(), kept.onList()
		if len(a) < len(b) || (len(a) == len(b) && slices.Compare(a, b) < 0) {
			kept = s
		}
	}
	for _, s := range loop {
		for u := range s.ever {
			g.Admitted[u] = true
		}
	}
	for u := range g.Admitted {
		for _, s := range loop {
			if !s.admitted[u] {
				g.Admitted[u] = false
				break
			}
		}
	}
	g.Cutoffs = kept.cutoffs

	settled := len(loop) == 1
	changes := append([]GuestEvent{}, founding...)
	for i, e := range invites {
		switch {
		case kept.counted[i]:
			changes = append(changes, e)
		case !kept.ever[e.Author]:
			discard(e, "an invitation by a user who was never invited to the room")
		case !settled:
			discard(e, "an invitation outside the state kept of a record that does not settle, which keeps out whom it cannot place")
		default:
			discard(e, "an invitation its author made after the cutoff of a revocation of them (D-236)")
		}
	}
	for _, e := range revokes {
		if kept.ever[e.Author] { // c., as newGuestState counted it

			changes = append(changes, e)
		} else {
			discard(e, "a revocation by a user who was never invited to the room")
		}
	}

	sort.Slice(changes, func(i, j int) bool {
		if changes[i].EventID != changes[j].EventID {
			return changes[i].EventID < changes[j].EventID
		}
		return guestKey(changes[i]) < guestKey(changes[j])
	})
	g.Changes = changes
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
