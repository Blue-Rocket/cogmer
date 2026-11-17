package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"maps"
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
	// Aliases maps a machine key that signed events before its user existed to the
	// user, so that Counts reads such an event as the user's (D-242, a machine key from
	// before the user identity carries its turns to its user). ResolveGuests leaves it
	// empty, and the daemon fills it from what it knows.
	Aliases map[string]string
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

// guestState is what one set of counted changes implies about the room: the
// invitations and revocations a round counts, and who was ever invited, who is
// revoked with which cutoffs, and who is on the list as a result.
type guestState struct {
	counted  []bool                      // by index into the room's invitations
	revoked  []bool                      // by index into the room's revocations
	ever     map[string]bool             // the founders and every subject of a counted invitation
	revsOf   map[string]map[string]bool  // revoked user to counted revocations' identifiers
	cutoffs  map[string]map[string]int64 // revoked user to machine key to last counting sequence
	admitted map[string]bool             // each user ever invited to whether on the list
}

func newGuestState(founding, invites, revokes []GuestEvent, counted, revoked []bool) *guestState {
	s := &guestState{
		counted:  counted,
		revoked:  revoked,
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

	// c. The revocations this round counts, which the round before chose by the
	// rule it chose invitations by (stands).
	mine := map[string][]GuestEvent{}
	for j, e := range revokes {
		if !revoked[j] {
			continue
		}
		if s.revsOf[e.Subject] == nil {
			s.revsOf[e.Subject] = map[string]bool{}
		}
		s.revsOf[e.Subject][e.EventID] = true
		mine[e.Subject] = append(mine[e.Subject], e)
	}

	// d. Each revoked user's machines take the lowest cutoff any counted
	// revocation gives them, a machine a revocation leaves out getting 0 from
	// it, so a cutoff only moves earlier (D-236, a revocation names the last
	// turn of each of the revoked user's machines that counts), as a removal's
	// does (D-220). A machine whose lowest is 0 is left out, which reads the
	// same.
	for user, revs := range mine {
		cut := map[string]int64{}
		for m := range revs[0].Cutoffs {
			low := revs[0].Cutoffs[m]
			for _, e := range revs[1:] {
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
	// that did not see it (D-235). A founder lifts nothing, so a founder is on the
	// list only until a revocation of them counts.
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

// stands reports whether a change its author made from machine with sequence
// counts in this state: its author was ever invited, and is on the list, or is not
// revoked, or made it within their own cutoff. So an expelled user can neither let
// in an accomplice after the cutoff nor expel the guests who remain, while a
// readmitted user changes the list as any guest does (D-235, a room's one guest
// list; D-236, a later invitation that lifts the revocation lets turns after the
// cutoff count again).
func (s *guestState) stands(author, machine string, sequence int64) bool {
	if !s.ever[author] {
		return false
	}
	if s.admitted[author] {
		return true
	}
	cut, revoked := s.cutoffs[author]
	last, ok := cut[machine]
	return !revoked || (ok && sequence <= last)
}

// next gives the invitations and the revocations this state counts.
func (s *guestState) next(invites, revokes []GuestEvent) ([]bool, []bool) {
	counted := make([]bool, len(invites))
	for i, e := range invites {
		counted[i] = s.stands(e.Author, e.Machine, e.Sequence)
	}
	revoked := make([]bool, len(revokes))
	for j, e := range revokes {
		revoked[j] = s.stands(e.Author, e.Machine, e.Sequence)
	}
	return counted, revoked
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

func countedKey(sets ...[]bool) string {
	var b []byte
	for _, set := range sets {
		for _, c := range set {
			if c {
				b = append(b, '1')
			} else {
				b = append(b, '0')
			}
		}
		b = append(b, '|')
	}
	return string(b)
}

// guestRoundCap is the most rounds ResolveGuests runs for a room with n changes to
// its list. An honest record settles within about a round for each invitation in
// its longest chain of them, and a record contrived to wander through its sets of
// counted changes could otherwise run a number of rounds exponential in n, all of
// them while the daemon holds its lock.
func guestRoundCap(n int) int { return 2*n + 4 }

// settleRounds runs step from start until a key repeats or cap rounds have run,
// and returns the states of the loop it ends in and the number of rounds run. A key
// that repeats closes the loop from its first appearance. Rounds that reach cap
// without a repeat are treated as a loop of their later half, which every machine
// computes alike, so that the rule for a record that never settles applies to them.
func settleRounds[S any](start S, key string, step func(S) (S, string), cap int) (loop []S, rounds int) {
	var states []S
	seen := map[string]int{}
	s, k := start, key
	for rounds < cap {
		if j, ok := seen[k]; ok {
			return states[j:], rounds
		}
		seen[k] = len(states)
		states = append(states, s)
		s, k = step(s)
		rounds++
	}
	return states[len(states)/2:], rounds
}

// ResolveGuests computes room's guest list from its guest events. founders are the
// users who found a room made before user identities, which has no founding
// self-invitation: the users whose machines signed its turns under the scheme from
// before user identities (D-243, a room from before user identities takes one guest
// list, founded by the users whose machines wrote its turns). A founder is admitted
// as a founding self-invitation admits its creator, and lifts no revocation.
//
// It works in rounds because the rules depend on one another in a circle: whether
// an invitation or a revocation counts depends on its author's cutoff, a cutoff on
// which revocations count, and those on who was ever invited, which depends on
// which invitations count (D-235, a room's one guest list; D-236, a revocation names
// the last turn that counts). A single pass has to break the circle somewhere, and
// breaking it by ignoring cutoffs lets an expelled guest invite an accomplice after
// the cutoff, or expel every guest who remains. So each round starts from a set of
// counted changes, beginning with the founders alone, derives who was ever invited,
// the cutoffs and the list from it, and takes the changes that state counts as the
// next set, until a set repeats.
//
// A set that repeats at once is the answer. A record can also loop, such as two
// guests revoking each other at once, or a guest revoking, with a cutoff before it,
// the invitation that let them in. Then a user is on the list only when every
// state of the loop admits them, the counted changes are those of the state
// admitting fewest, then the one whose sorted list comes first, and each machine's
// cutoff is the lowest any state of the loop gives it, so that a record in conflict
// keeps people out rather than letting them in, and every machine holding the same
// events settles on the same answer. Rounds that reach
// guestRoundCap without a repeat are judged by the same rule over their later half.
//
// A change that shares its machine's sequence with a different change counts for
// nobody (D-027, a sequence conflict is quarantined), so that a revoked user cannot
// sign a change under a sequence within the cutoff. The caller leaves out a change
// that shares its sequence with an event of another kind.
func ResolveGuests(room string, events []GuestEvent, founders ...string) *GuestList {
	g, _ := resolveGuests(room, events, founders)
	return g
}

// resolveGuests is ResolveGuests, which also reports how many rounds it ran.
func resolveGuests(room string, events []GuestEvent, founders []string) (*GuestList, int) {
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
	type place struct {
		machine  string
		sequence int64
	}
	var unique []GuestEvent
	slot := map[place]map[string]bool{} // each machine's sequence to the distinct changes there
	for i, k := range all {
		if i > 0 && all[i-1].key == k.key {
			continue
		}
		unique = append(unique, k.e)
		at := place{k.e.Machine, k.e.Sequence}
		if slot[at] == nil {
			slot[at] = map[string]bool{}
		}
		slot[at][k.key] = true
	}

	var founding, invites, revokes []GuestEvent
	for _, u := range founders {
		founding = append(founding, GuestEvent{EventID: founderEventID(u), Author: u, Kind: GuestInvite, Subject: u})
	}
	for _, e := range unique {
		if len(slot[place{e.Machine, e.Sequence}]) > 1 {
			discard(e, "a change that shares its machine's sequence with a different change, so it counts for nobody (D-027, a sequence conflict is quarantined)")
			continue
		}
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

	// The rounds, from the founders alone, until a set of counted changes repeats.
	counted, revoked := make([]bool, len(invites)), make([]bool, len(revokes))
	loop, rounds := settleRounds(newGuestState(founding, invites, revokes, counted, revoked), countedKey(counted, revoked),
		func(s *guestState) (*guestState, string) {
			c, r := s.next(invites, revokes)
			return newGuestState(founding, invites, revokes, c, r), countedKey(c, r)
		}, guestRoundCap(len(invites)+len(revokes)))

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
	// A user revoked in some state of the loop takes, for each machine, the lowest
	// cutoff any of those states gives it, so that a record in conflict leaves the
	// least of their turns counting and still keeps those every state counts.
	g.Cutoffs = map[string]map[string]int64{}
	for _, s := range loop {
		for user, cut := range s.cutoffs {
			have, ok := g.Cutoffs[user]
			if !ok {
				g.Cutoffs[user] = maps.Clone(cut)
				continue
			}
			for m, seq := range have {
				if other, ok := cut[m]; ok {
					have[m] = min(seq, other)
				} else {
					delete(have, m)
				}
			}
		}
	}

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
	for j, e := range revokes {
		switch {
		case kept.revoked[j]:
			changes = append(changes, e)
		case !kept.ever[e.Author]:
			discard(e, "a revocation by a user who was never invited to the room")
		case !settled:
			discard(e, "a revocation outside the state kept of a record that does not settle, which keeps out whom it cannot place")
		default:
			discard(e, "a revocation its author made after the cutoff of a revocation of them (D-236)")
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
	return g, rounds
}

// founderEventID names the founding of a room from before user identities by one
// of its founders, which is no event, so that the list's changes show it as they
// show a founding self-invitation.
func founderEventID(user string) string { return "founder:" + user }

// Counts reports whether user's event with this sequence from machine counts in
// the room. A user on the list counts in full, including after an earlier
// cutoff, since the invitation that readmitted them lifted the revocation that
// set it (D-236). An expelled user's events count up to their machine's cutoff,
// and a user never invited counts for nothing.
func (g *GuestList) Counts(user, machine string, sequence int64) bool {
	if u, ok := g.Aliases[user]; ok {
		user = u
	}
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
