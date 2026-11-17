package main

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// A room under test. Users and machines are plain names, since this layer takes
// authenticity as given and only compares identifiers; the founder's nonce is
// fixed, so the room's identifier is the same on every run.
type guestRoom struct {
	room    string
	founder string
	nonce   []byte
	events  []GuestEvent
}

func newGuestRoom(founder string) *guestRoom {
	nonce := []byte("nonce for " + founder)
	g := &guestRoom{room: RoomIDFor(founder, nonce), founder: founder, nonce: nonce}
	g.add(GuestEvent{Author: founder, Machine: founder + "/laptop", Sequence: 1, Kind: GuestInvite,
		Subject: founder, Nonce: nonce})
	return g
}

// add gives e the next event identifier, which sorts in the order added.
func (g *guestRoom) add(e GuestEvent) string {
	e.EventID = fmt.Sprintf("ev%03d", len(g.events))
	g.events = append(g.events, e)
	return e.EventID
}

func (g *guestRoom) invite(author string, seq int64, subject string, lifts ...string) string {
	return g.add(GuestEvent{Author: author, Machine: author + "/laptop", Sequence: seq, Kind: GuestInvite,
		Subject: subject, Lifts: lifts})
}

func (g *guestRoom) revoke(author string, seq int64, subject string, cutoffs map[string]int64) string {
	return g.add(GuestEvent{Author: author, Machine: author + "/laptop", Sequence: seq, Kind: GuestRevoke,
		Subject: subject, Cutoffs: cutoffs})
}

func (g *guestRoom) resolve() *GuestList { return ResolveGuests(g.room, g.events) }

func wantGuestDiscarded(t *testing.T, l *GuestList, eventID, reason string) {
	t.Helper()
	h := sha256.Sum256([]byte(eventID))
	for _, d := range l.Discarded {
		if d.Hash == h {
			if d.Kind != "guest" || !strings.Contains(d.Reason, reason) {
				t.Fatalf("event %s discarded as %q %q, want guest containing %q", eventID, d.Kind, d.Reason, reason)
			}
			return
		}
	}
	t.Fatalf("event %s is not discarded; discarded: %+v", eventID, l.Discarded)
}

func wantOnList(t *testing.T, l *GuestList, want map[string]bool) {
	t.Helper()
	for user, on := range want {
		if got := l.Admitted[user]; got != on {
			t.Errorf("%s on the list: %v, want %v", user, got, on)
		}
	}
}

func TestFoundingAdmitsTheCreator(t *testing.T) {
	g := newGuestRoom("alice")
	l := g.resolve()
	if !reflect.DeepEqual(l.Admitted, map[string]bool{"alice": true}) {
		t.Fatalf("Admitted = %v, want only the creator", l.Admitted)
	}
	if len(l.Changes) != 1 || l.Changes[0].EventID != "ev000" || len(l.Discarded) != 0 {
		t.Fatalf("Changes %+v, Discarded %+v, want the founding invitation alone", l.Changes, l.Discarded)
	}
	if !l.Counts("alice", "alice/laptop", 1_000_000) {
		t.Fatal("the creator's event does not count")
	}
}

func TestASelfInvitationThatDoesNotHashToTheRoomIsDiscarded(t *testing.T) {
	// An outsider who knows the room's identifier, and even the creator's nonce,
	// cannot make their own identifier hash to it.
	g := newGuestRoom("alice")
	noNonce := g.invite("mallory", 1, "mallory")
	stolen := g.add(GuestEvent{Author: "mallory", Machine: "mallory/laptop", Sequence: 2, Kind: GuestInvite,
		Subject: "mallory", Nonce: g.nonce})
	byMallory := g.invite("mallory", 3, "trudy")
	l := g.resolve()
	if _, ever := l.Admitted["mallory"]; ever {
		t.Fatal("an outsider's self-invitation put them on the list")
	}
	if _, ever := l.Admitted["trudy"]; ever {
		t.Fatal("an outsider's invitation of another user counted")
	}
	wantGuestDiscarded(t, l, noNonce, "does not hash to the room's identifier")
	wantGuestDiscarded(t, l, stolen, "does not hash to the room's identifier")
	wantGuestDiscarded(t, l, byMallory, "never invited")
	if l.Counts("mallory", "mallory/laptop", 1) {
		t.Fatal("an outsider's event counts")
	}
}

func TestAnInvitationOfAnotherUserCarryingAFoundingValueIsDiscarded(t *testing.T) {
	g := newGuestRoom("alice")
	e := g.add(GuestEvent{Author: "alice", Machine: "alice/laptop", Sequence: 2, Kind: GuestInvite,
		Subject: "bob", Nonce: []byte("x")})
	l := g.resolve()
	wantGuestDiscarded(t, l, e, "founding value")
	if _, ever := l.Admitted["bob"]; ever {
		t.Fatal("an invitation carrying a founding value admitted its subject")
	}
}

func TestAnExpelledCreatorsNewSelfInvitationDoesNotReadmitThem(t *testing.T) {
	g := newGuestRoom("alice")
	g.invite("alice", 2, "bob")
	r := g.revoke("bob", 1, "alice", map[string]int64{"alice/laptop": 2})
	again := g.add(GuestEvent{Author: "alice", Machine: "alice/laptop", Sequence: 3, Kind: GuestInvite,
		Subject: "alice", Nonce: g.nonce, Lifts: []string{r}})
	bare := g.invite("alice", 4, "alice", r)
	l := g.resolve()
	wantOnList(t, l, map[string]bool{"alice": false, "bob": true})
	wantGuestDiscarded(t, l, bare, "does not hash")
	for _, c := range l.Changes {
		if c.EventID == again && c.Lifts != nil {
			t.Fatalf("a self-invitation is shown lifting %v, and it lifts nothing", c.Lifts)
		}
	}

	// Another guest's invitation that lifts the revocation lets her back in.
	g.invite("bob", 2, "alice", r)
	wantOnList(t, g.resolve(), map[string]bool{"alice": true, "bob": true})
}

func TestAnInvitationByAGuestAdmits(t *testing.T) {
	g := newGuestRoom("alice")
	g.invite("alice", 2, "bob")
	g.invite("bob", 1, "carol")
	l := g.resolve()
	wantOnList(t, l, map[string]bool{"alice": true, "bob": true, "carol": true})
	if len(l.Changes) != 3 {
		t.Fatalf("Changes = %+v, want the three invitations", l.Changes)
	}
}

func TestARevocationExpelsTheUserWhateverInvitationAdmittedThem(t *testing.T) {
	g := newGuestRoom("alice")
	g.invite("alice", 2, "bob")
	g.invite("alice", 3, "carol")
	// bob's invitation of carol never names the revocation, made or not when he
	// sent it, so it does not keep her in (D-235).
	g.invite("bob", 1, "carol")
	g.revoke("alice", 4, "carol", map[string]int64{"carol/laptop": 0})
	g.invite("bob", 2, "carol")
	l := g.resolve()
	wantOnList(t, l, map[string]bool{"alice": true, "bob": true, "carol": false})
	if _, ok := l.Cutoffs["carol"]; !ok {
		t.Fatal("the revoked user has no cutoffs entry")
	}
}

func TestAnInvitationThatLiftsEveryRevocationReadmitsAndOneThatLiftsSomeDoesNot(t *testing.T) {
	g := newGuestRoom("alice")
	g.invite("alice", 2, "bob")
	g.invite("alice", 3, "carol")
	r1 := g.revoke("alice", 4, "carol", map[string]int64{"carol/laptop": 3})
	r2 := g.revoke("bob", 1, "carol", map[string]int64{"carol/laptop": 5})
	g.invite("bob", 2, "carol", r1)
	wantOnList(t, g.resolve(), map[string]bool{"carol": false})

	g.invite("alice", 5, "carol", r2, r1)
	l := g.resolve()
	wantOnList(t, l, map[string]bool{"carol": true})
	if !l.Counts("carol", "carol/laptop", 100) {
		t.Fatal("a readmitted user's event past the old cutoff does not count")
	}
}

func TestTwoGuestsRevokingEachOtherAreBothOut(t *testing.T) {
	g := newGuestRoom("alice")
	g.invite("alice", 2, "bob")
	g.invite("alice", 3, "carol")
	g.revoke("bob", 1, "carol", map[string]int64{"carol/laptop": 1})
	g.revoke("carol", 1, "bob", map[string]int64{"bob/laptop": 1})
	l := g.resolve()
	wantOnList(t, l, map[string]bool{"alice": true, "bob": false, "carol": false})
	if len(l.Discarded) != 0 {
		t.Fatalf("a revocation was discarded: %+v", l.Discarded)
	}
}

func TestARevocationByAUserNeverInvitedIsDiscarded(t *testing.T) {
	g := newGuestRoom("alice")
	r := g.revoke("mallory", 1, "alice", nil)
	l := g.resolve()
	wantOnList(t, l, map[string]bool{"alice": true})
	wantGuestDiscarded(t, l, r, "never invited")
}

func TestAnInvitationByAnExpelledUserCountsOnlyWithinTheirCutoff(t *testing.T) {
	g := newGuestRoom("alice")
	g.invite("alice", 2, "bob")
	g.revoke("alice", 3, "bob", map[string]int64{"bob/laptop": 5})
	g.invite("bob", 5, "carol")
	late := g.invite("bob", 6, "dave")
	other := g.add(GuestEvent{Author: "bob", Machine: "bob/phone", Sequence: 1, Kind: GuestInvite,
		Subject: "erin"})
	l := g.resolve()
	wantOnList(t, l, map[string]bool{"bob": false, "carol": true, "dave": false, "erin": false})
	wantGuestDiscarded(t, l, late, "after the cutoff")
	wantGuestDiscarded(t, l, other, "after the cutoff")
}

func TestTheLowestCutoffAcrossRevocationsAppliesAndAMachineNotNamedGetsZero(t *testing.T) {
	g := newGuestRoom("alice")
	g.invite("alice", 2, "bob")
	g.invite("alice", 3, "carol")
	g.revoke("alice", 4, "carol", map[string]int64{"carol/laptop": 5, "carol/phone": 3})
	g.revoke("bob", 1, "carol", map[string]int64{"carol/laptop": 7, "carol/phone": 2, "carol/tablet": 4})
	l := g.resolve()
	want := map[string]int64{"carol/laptop": 5, "carol/phone": 2}
	if got := l.Cutoffs["carol"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("cutoffs = %v, want %v, the tablet getting 0 from the revocation that does not name it", got, want)
	}
	for _, c := range []struct {
		machine string
		seq     int64
		want    bool
	}{
		{"carol/laptop", 4, true}, {"carol/laptop", 5, true}, {"carol/laptop", 6, false},
		{"carol/phone", 2, true}, {"carol/phone", 3, false},
		{"carol/tablet", 1, false}, {"carol/unnamed", 1, false},
	} {
		if got := l.Counts("carol", c.machine, c.seq); got != c.want {
			t.Errorf("Counts(carol, %s, %d) = %v, want %v", c.machine, c.seq, got, c.want)
		}
	}
}

func TestCountsGivesNothingToAUserNeverInvitedEvenWithACutoff(t *testing.T) {
	g := newGuestRoom("alice")
	g.revoke("alice", 2, "mallory", map[string]int64{"mallory/laptop": 9})
	l := g.resolve()
	if l.Counts("mallory", "mallory/laptop", 1) {
		t.Fatal("a never-invited user's event counts because a revocation named a cutoff for them")
	}
	if l.Counts("nobody", "nobody/laptop", 1) {
		t.Fatal("an unknown user's event counts")
	}
	// The revocation stands, so an invitation that does not lift it admits nobody.
	g.invite("alice", 3, "mallory")
	wantOnList(t, g.resolve(), map[string]bool{"mallory": false})
}

func TestResolveGuestsDoesNotDependOnTheOrderOfEvents(t *testing.T) {
	g := newGuestRoom("alice")
	g.invite("alice", 2, "bob")
	g.invite("alice", 3, "carol")
	g.invite("bob", 1, "dave")
	r1 := g.revoke("bob", 2, "carol", map[string]int64{"carol/laptop": 2, "carol/phone": 9})
	g.revoke("carol", 1, "bob", map[string]int64{"bob/laptop": 1})
	r3 := g.revoke("alice", 4, "carol", map[string]int64{"carol/laptop": 4})
	g.invite("dave", 1, "carol", r3, r1)
	g.invite("bob", 2, "erin")
	g.invite("mallory", 1, "mallory")
	g.revoke("mallory", 2, "alice", nil)
	g.add(GuestEvent{Author: "alice", Machine: "alice/laptop", Sequence: 9, Kind: "GUEST_SHRUG", Subject: "bob"})
	// One event given twice, once with its lifts in another order.
	dup := g.events[7]
	dup.Lifts = []string{r1, r3}
	g.events = append(g.events, dup, g.events[1])
	// An invitation decoded with an empty list is the same event as one with none.
	empty := g.events[2]
	empty.Lifts = []string{}
	g.events = append(g.events, empty)

	want := g.resolve()
	if len(want.Changes) < 6 || len(want.Discarded) < 4 {
		t.Fatalf("the events do not exercise what they should: %d changes, %d discarded", len(want.Changes), len(want.Discarded))
	}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		shuffled := append([]GuestEvent{}, g.events...)
		rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		if got := ResolveGuests(g.room, shuffled); !reflect.DeepEqual(got, want) {
			t.Fatalf("order %d resolved differently:\n got %+v\nwant %+v", i, got, want)
		}
	}
}

func TestResolveGuestsLeavesTheCallersEventsAsTheyWere(t *testing.T) {
	g := newGuestRoom("alice")
	g.invite("alice", 2, "bob", "zz", "aa")
	g.revoke("alice", 3, "bob", map[string]int64{"bob/laptop": 1})
	l := g.resolve()
	if g.events[1].Lifts[0] != "zz" {
		t.Fatal("ResolveGuests sorted the caller's lifts in place")
	}
	l.Changes[2].Cutoffs["bob/laptop"] = 99
	if g.events[2].Cutoffs["bob/laptop"] != 1 {
		t.Fatal("the list shares a cutoff map with the caller's event")
	}
}

func TestAnExpelledGuestsLateInvitationGivesTheirAccompliceNoStanding(t *testing.T) {
	g := newGuestRoom("alice")
	g.invite("alice", 2, "bob")
	g.revoke("alice", 3, "bob", map[string]int64{"bob/laptop": 5})
	late := g.invite("bob", 6, "mallory")
	byMallory := g.invite("mallory", 1, "eve")
	expel := g.revoke("mallory", 2, "alice", nil)
	l := g.resolve()
	if _, ever := l.Admitted["mallory"]; ever {
		t.Fatal("an invitation past its author's cutoff made its subject ever invited")
	}
	if _, ever := l.Admitted["eve"]; ever {
		t.Fatal("the accomplice's invitation counted")
	}
	wantOnList(t, l, map[string]bool{"alice": true, "bob": false})
	wantGuestDiscarded(t, l, late, "after the cutoff")
	wantGuestDiscarded(t, l, byMallory, "never invited")
	wantGuestDiscarded(t, l, expel, "never invited")
	if _, ok := l.Cutoffs["alice"]; ok {
		t.Fatal("the accomplice's revocation of the creator set a cutoff")
	}
}

func TestAReadmittedUserCanInviteAfterTheirOldCutoff(t *testing.T) {
	g := newGuestRoom("alice")
	g.invite("alice", 2, "bob")
	r := g.revoke("alice", 3, "bob", map[string]int64{"bob/laptop": 5})
	g.invite("alice", 4, "bob", r)
	g.invite("bob", 9, "carol")
	l := g.resolve()
	wantOnList(t, l, map[string]bool{"alice": true, "bob": true, "carol": true})
	if !l.Counts("bob", "bob/laptop", 9) {
		t.Fatal("a readmitted user's event past the old cutoff does not count")
	}
}

// guestCycle builds a record that never settles. Carol, whom Bob invited, revokes
// Bob with a cutoff before that invitation, which takes away the invitation that
// gave her the standing to revoke; Dave, whom Carol invited, stands only while
// she does.
func guestCycle() *guestRoom {
	g := newGuestRoom("alice")
	g.invite("alice", 2, "bob")
	g.invite("bob", 2, "carol")
	g.revoke("carol", 1, "bob", map[string]int64{"bob/laptop": 1})
	g.invite("carol", 2, "dave")
	return g
}

func TestARecordThatNeverSettlesKeepsOutEveryoneNotAdmittedInEveryState(t *testing.T) {
	g := guestCycle()
	l := g.resolve()
	// In some states of the loop Carol's revocation counts and Bob is out, and in
	// others Bob's invitation of her does not and she and Dave are out; only Alice
	// is in every one.
	wantOnList(t, l, map[string]bool{"alice": true, "bob": false, "carol": false, "dave": false})
	for _, user := range []string{"bob", "carol", "dave"} {
		if _, ever := l.Admitted[user]; !ever {
			t.Errorf("%s, invited in some state of the cycle, is absent from the list", user)
		}
	}
	// Bob's cutoff is the lowest any state of the loop gives him: Carol's revocation.
	if got := l.Cutoffs["bob"]; !reflect.DeepEqual(got, map[string]int64{"bob/laptop": 1}) {
		t.Fatalf("Bob's cutoffs = %v, want those of the state that revokes him", got)
	}
	if !l.Counts("bob", "bob/laptop", 1) || l.Counts("bob", "bob/laptop", 2) || l.Counts("dave", "dave/laptop", 1) {
		t.Fatal("Counts does not follow the state the cycle rule chose")
	}

	want := l
	rng := rand.New(rand.NewSource(2))
	for i := 0; i < 50; i++ {
		shuffled := append([]GuestEvent{}, g.events...)
		rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		if got := ResolveGuests(g.room, shuffled); !reflect.DeepEqual(got, want) {
			t.Fatalf("order %d resolved the cycle differently:\n got %+v\nwant %+v", i, got, want)
		}
	}
}

// A guest who is revoked and then revokes every guest who remains, from past the
// cutoff, with cutoffs that show they saw the revocation, changes nothing (D-235, an
// invitation or a revocation by a revoked user counts only within that user's cutoff).
func TestARevokedGuestsRevengeRevocationsChangeNothing(t *testing.T) {
	g := newGuestRoom("alice")
	g.invite("alice", 2, "bob")
	g.invite("alice", 3, "carol")
	g.revoke("alice", 4, "bob", map[string]int64{"bob/laptop": 3})
	ofAlice := g.revoke("bob", 5, "alice", map[string]int64{"alice/laptop": 4})
	ofCarol := g.revoke("bob", 6, "carol", nil)
	l := g.resolve()
	wantOnList(t, l, map[string]bool{"alice": true, "bob": false, "carol": true})
	wantGuestDiscarded(t, l, ofAlice, "after the cutoff")
	wantGuestDiscarded(t, l, ofCarol, "after the cutoff")
	if _, ok := l.Cutoffs["alice"]; ok {
		t.Fatal("the revenge revocation set a cutoff for the creator")
	}

	// A revocation within the cutoff counts as any guest's does.
	g.revoke("bob", 2, "carol", map[string]int64{"carol/laptop": 1})
	wantOnList(t, g.resolve(), map[string]bool{"alice": true, "carol": false})
}

// Two guests who revoke each other at once, each past the cutoff the other's
// revocation gives them, are both out through the rule for a record that never
// settles, and what each wrote up to the other's cutoff still counts.
func TestTwoGuestsRevokingEachOtherPastTheirCutoffsAreBothOut(t *testing.T) {
	g := newGuestRoom("alice")
	g.invite("alice", 2, "bob")
	g.invite("alice", 3, "carol")
	g.revoke("bob", 5, "carol", map[string]int64{"carol/laptop": 3})
	g.revoke("carol", 4, "bob", map[string]int64{"bob/laptop": 4})
	l := g.resolve()
	wantOnList(t, l, map[string]bool{"alice": true, "bob": false, "carol": false})
	if !l.Counts("bob", "bob/laptop", 4) || l.Counts("bob", "bob/laptop", 5) {
		t.Fatal("bob's turns do not stop at the cutoff carol's revocation gives")
	}
	if !l.Counts("carol", "carol/laptop", 3) || l.Counts("carol", "carol/laptop", 4) {
		t.Fatal("carol's turns do not stop at the cutoff bob's revocation gives")
	}
}

// A change that shares its machine's sequence with a different change counts for
// nobody, so a revoked guest cannot sign a revocation under a sequence inside the
// cutoff (D-027, a sequence conflict is quarantined).
func TestAChangeSharingItsMachinesSequenceCountsForNobody(t *testing.T) {
	g := newGuestRoom("alice")
	g.invite("alice", 2, "bob")
	g.invite("alice", 3, "carol")
	g.revoke("alice", 4, "bob", map[string]int64{"bob/laptop": 2})
	honest := g.invite("bob", 2, "dave")
	forged := g.revoke("bob", 2, "carol", map[string]int64{"carol/laptop": 0})
	l := g.resolve()
	wantOnList(t, l, map[string]bool{"alice": true, "carol": true, "bob": false})
	if _, ever := l.Admitted["dave"]; ever {
		t.Fatal("an invitation sharing its sequence with another change counted")
	}
	wantGuestDiscarded(t, l, honest, "shares its machine's sequence")
	wantGuestDiscarded(t, l, forged, "shares its machine's sequence")
}

// A room from before user identities is founded by the users given as its founders,
// each admitted as a founding self-invitation admits its creator, and a founder
// lifts no revocation (D-243, a room from before user identities takes one guest list).
func TestFoundersOfARoomFromBeforeUserIdentities(t *testing.T) {
	var evs []GuestEvent
	add := func(e GuestEvent) string {
		e.EventID = fmt.Sprintf("ev%03d", len(evs))
		evs = append(evs, e)
		return e.EventID
	}
	add(GuestEvent{Author: "alice", Machine: "alice/laptop", Sequence: 7, Kind: GuestInvite, Subject: "carol"})
	l := ResolveGuests("an-old-room", evs, "alice", "bob")
	wantOnList(t, l, map[string]bool{"alice": true, "bob": true, "carol": true})
	if l := ResolveGuests("an-old-room", evs); len(l.Admitted) != 0 {
		t.Fatalf("a room with no founders and no founding invitation admits %v", l.Admitted)
	}

	r := add(GuestEvent{Author: "alice", Machine: "alice/laptop", Sequence: 8, Kind: GuestRevoke, Subject: "bob",
		Cutoffs: map[string]int64{"bob/laptop": 3}})
	wantOnList(t, ResolveGuests("an-old-room", evs, "alice", "bob"), map[string]bool{"bob": false})
	add(GuestEvent{Author: "alice", Machine: "alice/laptop", Sequence: 9, Kind: GuestInvite, Subject: "bob", Lifts: []string{r}})
	wantOnList(t, ResolveGuests("an-old-room", evs, "alice", "bob"), map[string]bool{"bob": true})
}

// settleRounds stops at its cap, and judges the rounds by their later half, when
// no state repeats before it.
func TestSettleRoundsStopsAtItsCap(t *testing.T) {
	step := func(n int) (int, string) { return n + 1, fmt.Sprint(n + 1) }
	loop, rounds := settleRounds(0, "0", step, 10)
	if rounds != 10 || !reflect.DeepEqual(loop, []int{5, 6, 7, 8, 9}) {
		t.Fatalf("ran %d rounds and kept %v; want 10 rounds and the later half", rounds, loop)
	}
	loop, rounds = settleRounds(0, "0", func(n int) (int, string) { return (n + 1) % 3, fmt.Sprint((n + 1) % 3) }, 10)
	if rounds != 3 || !reflect.DeepEqual(loop, []int{0, 1, 2}) {
		t.Fatalf("a loop of three ran %d rounds and kept %v", rounds, loop)
	}
}

// A record contrived from loops of different lengths, each a chain of invitations
// whose last guest revokes its first with a cutoff before the invitation that leads
// to them, returns to a state only after the least common multiple of the loops'
// lengths, which grows faster than any power of the number of changes. The rounds
// stop at the cap, a number proportional to the changes, and every machine computes
// the same list from them.
func TestAContrivedRecordStopsAtTheRoundCap(t *testing.T) {
	g := newGuestRoom("alice")
	n := 0
	for _, length := range []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13} {
		users := make([]string, length)
		for i := range users {
			users[i] = fmt.Sprintf("u%d-%d", length, i)
		}
		g.invite("alice", int64(2+n), users[0])
		for i := 0; i+1 < length; i++ {
			g.invite(users[i], 1, users[i+1])
		}
		g.revoke(users[length-1], 2, users[0], nil)
		n++
	}
	l, rounds := resolveGuests(g.room, g.events, nil)
	if cap := guestRoundCap(len(g.events) - 1); rounds != cap {
		t.Fatalf("ran %d rounds; want the cap of %d, which a record this contrived reaches", rounds, cap)
	}
	wantOnList(t, l, map[string]bool{"alice": true, "u5-0": false})
	rng := rand.New(rand.NewSource(3))
	shuffled := append([]GuestEvent{}, g.events...)
	rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	if got := ResolveGuests(g.room, shuffled); !reflect.DeepEqual(got, l) {
		t.Fatal("the contrived record resolved differently in another order")
	}
}

// An honest chain of invitations far longer than any room holds settles before the
// cap, with every guest on the list.
func TestALongChainOfInvitationsSettlesBeforeTheCap(t *testing.T) {
	g := newGuestRoom("alice")
	prev := "alice"
	for i := 0; i < 200; i++ {
		next := fmt.Sprintf("guest%03d", i)
		g.invite(prev, int64(2+i), next)
		prev = next
	}
	l, rounds := resolveGuests(g.room, g.events, nil)
	if rounds >= guestRoundCap(len(g.events)-1) {
		t.Fatalf("an honest chain reached the cap after %d rounds", rounds)
	}
	if !l.Admitted["guest199"] {
		t.Fatal("the last guest of the chain is not on the list")
	}
}
