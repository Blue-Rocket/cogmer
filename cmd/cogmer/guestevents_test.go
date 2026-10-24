package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

// A room's guest list as events in the room (D-235, a room's one guest list, changed
// by any guest through events in the room, decides whose events count).

// newFounder is newUserPeer for a machine that founds the room: it has its own HOME
// and creates the room itself, so the room's identifier is known only once it has.
// Every room is founded before the next peer moves HOME, since a store opens under
// the HOME of the moment.
func newFounder(t *testing.T, id *Identity, u *testUser, rooms int) (*offlinePeer, []Room) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	m, err := OpenMembership()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	if err := saveSharedKey(u.shared, u.version); err != nil {
		t.Fatal(err)
	}
	d := &Daemon{id: id, members: m, stores: map[string]*Store{}}
	d.askRooms = func(_ string, rooms []Room) []Room { return rooms }
	t.Cleanup(d.closeStores)
	var out []Room
	for i := 0; i < rooms; i++ {
		r, err := d.FoundRoom(false)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	up := &atomic.Bool{}
	up.Store(true)
	addr := servePeerTLSHandler(t, d, d.PeerRoutes())
	return &offlinePeer{d: d, store: d.stores[out[0].RoomID], addr: addr, up: up}, out
}

// has reports whether a peer's store holds a turn with this text.
func has(t *testing.T, p *offlinePeer, roomID, text string) bool {
	t.Helper()
	for _, e := range heldEvents(t, p, roomID) {
		if e.Content == text {
			return true
		}
	}
	return false
}

func pull(t *testing.T, from, to *offlinePeer) int {
	t.Helper()
	n, err := from.d.pullFrom(to.addr)
	if err != nil {
		t.Fatalf("pull: %v", err)
	}
	return n
}

// The room's identifier is the hash of its creator and the value its first event
// carries, and that event puts the creator on the list by the rule every guest is
// (D-240, a room's identifier is the hash of its creator and a random value).
func TestFoundingMakesTheCreatorAGuestAndTheRoomsIdentifierHashesTheCreatorAndNonce(t *testing.T) {
	ana := newTestUser(t)
	a, rooms := newFounder(t, ana.machine(t, "laptop"), ana, 1)
	room := rooms[0]

	evs := heldEvents(t, a, room.RoomID)
	if len(evs) != 1 || evs[0].PeerSequence != 1 || evs[0].EventType != GuestInvite || evs[0].SigVersion != 4 {
		t.Fatalf("the room's events are %+v", evs)
	}
	g, ok := guestEventFrom(evs[0])
	if !ok || g.Author != ana.id || g.Subject != ana.id || len(g.Nonce) != 16 {
		t.Fatalf("the founding event reads as %+v, %v", g, ok)
	}
	if RoomIDFor(ana.id, g.Nonce) != room.RoomID {
		t.Error("the room's identifier is not the hash of its creator and the founding value")
	}
	if !a.d.members.EventListed(room.RoomID) || !a.d.members.IsGuest(room.RoomID, ana.id) {
		t.Error("the creator is not on the room's list")
	}
	if a.d.members.IsGuest(room.RoomID, newTestUser(t).id) {
		t.Error("a stranger is on the room's list")
	}
	if got, _ := a.d.members.Guests(room.RoomID); !reflect.DeepEqual(got, []string{ana.id}) {
		t.Errorf("the room's guests are %v", got)
	}
	// The view shows the founding as a change to the list.
	st, err := a.d.snapshot(room)
	if err != nil || len(st.Events) != 1 || st.Events[0].Content != "founded the room" || st.Events[0].NotInSession {
		t.Errorf("the view shows %+v, %v", st.Events, err)
	}
}

// A machine of an invited user receives the room, and a machine of a colleague who
// is verified and was never invited is refused (D-045, admission is enforced).
func TestAnInvitedUsersMachineSyncsAndAnUninvitedOneIsRefused(t *testing.T) {
	ana, ben, cal := newTestUser(t), newTestUser(t), newTestUser(t)
	a, rooms := newFounder(t, ana.machine(t, "laptop"), ana, 1)
	room := rooms[0]
	b := newUserPeer(t, room.RoomID, room.RoomName, ben.machine(t, "desktop"), ben)
	c := newUserPeer(t, room.RoomID, room.RoomName, cal.machine(t, "tablet"), cal)
	introduceUsers(t, a, b, room.RoomID)
	introduceUsers(t, a, c, room.RoomID)
	if err := a.d.InviteUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	a.say(t, room.RoomID, "s", EventUserPrompt, "from the founder")

	if n := pull(t, b, a); n != 3 {
		t.Errorf("the invited machine took %d events, want the founding, the invitation and the turn", n)
	}
	if !has(t, b, room.RoomID, "from the founder") || !b.d.members.EventListed(room.RoomID) {
		t.Error("the invited machine did not receive the room and its list")
	}
	if n := pull(t, c, a); n != 0 || len(heldEvents(t, c, room.RoomID)) != 0 {
		t.Errorf("the uninvited machine took %d events", n)
	}
	if a.d.members.IsGuest(room.RoomID, cal.id) {
		t.Error("the uninvited user is a guest")
	}
}

// A revocation names, for each machine of the revoked user, the last turn that counts.
// A turn within it is kept and relayed, a turn past it is not stored, and an
// invitation that lifts the revocation lets the later turn arrive by the next sync,
// since it was left on offer (D-236, a revocation names the last turn that counts).
func TestARevokedGuestsTurnPastTheCutoffIsNotStoredAndArrivesOnceAnInvitationLiftsIt(t *testing.T) {
	ana, ben, cal := newTestUser(t), newTestUser(t), newTestUser(t)
	a, rooms := newFounder(t, ana.machine(t, "laptop"), ana, 1)
	room := rooms[0]
	b := newUserPeer(t, room.RoomID, room.RoomName, ben.machine(t, "desktop"), ben)
	c := newUserPeer(t, room.RoomID, room.RoomName, cal.machine(t, "tablet"), cal)
	introduceUsers(t, a, b, room.RoomID)
	introduceUsers(t, a, c, room.RoomID)
	introduceUsers(t, b, c, room.RoomID)
	for _, u := range []string{ben.id, cal.id} {
		if err := a.d.InviteUser(room.RoomID, u); err != nil {
			t.Fatal(err)
		}
	}
	pull(t, b, a)
	pull(t, c, a)

	b.say(t, room.RoomID, "s", EventUserPrompt, "within the cutoff")
	pull(t, a, b)
	if err := a.d.RevokeUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	if g := a.d.guestList(room.RoomID); g == nil || g.Admitted[ben.id] || g.Cutoffs[ben.id][b.d.id.PeerID] != 1 {
		t.Fatalf("the revocation left the list %+v", g)
	}
	b.say(t, room.RoomID, "s", EventUserPrompt, "past the cutoff")

	pull(t, c, a) // the revocation, and the turn within it relayed
	pull(t, c, b) // the turn past it, from its author
	pull(t, a, b)
	for _, p := range []*offlinePeer{a, c} {
		if !has(t, p, room.RoomID, "within the cutoff") {
			t.Error("a turn within the cutoff was not kept")
		}
		if has(t, p, room.RoomID, "past the cutoff") {
			t.Error("a turn past the cutoff was stored")
		}
	}

	if err := a.d.InviteUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	if !a.d.members.IsGuest(room.RoomID, ben.id) {
		t.Fatal("the invitation did not lift the revocation")
	}
	pull(t, c, a)
	pull(t, c, b)
	if !has(t, c, room.RoomID, "past the cutoff") {
		t.Error("the turn past a lifted cutoff did not arrive by the next sync")
	}
}

// Every revocation by a user ever invited counts, so two guests who revoke each other
// are both out, and the machines of both are refused (D-235).
func TestTwoGuestsRevokingEachOtherAreBothRefused(t *testing.T) {
	ana, ben, cal := newTestUser(t), newTestUser(t), newTestUser(t)
	a, rooms := newFounder(t, ana.machine(t, "laptop"), ana, 1)
	room := rooms[0]
	b := newUserPeer(t, room.RoomID, room.RoomName, ben.machine(t, "desktop"), ben)
	c := newUserPeer(t, room.RoomID, room.RoomName, cal.machine(t, "tablet"), cal)
	introduceUsers(t, a, b, room.RoomID)
	introduceUsers(t, a, c, room.RoomID)
	for _, u := range []string{ben.id, cal.id} {
		if err := a.d.InviteUser(room.RoomID, u); err != nil {
			t.Fatal(err)
		}
	}
	pull(t, b, a)
	pull(t, c, a)
	if err := b.d.RevokeUser(room.RoomID, cal.id); err != nil {
		t.Fatal(err)
	}
	if err := c.d.RevokeUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	pull(t, a, b)
	pull(t, a, c)
	for _, u := range []string{ben.id, cal.id} {
		if a.d.members.IsGuest(room.RoomID, u) {
			t.Errorf("%s is still a guest after the two revoked each other", PeerName(u))
		}
	}
	a.say(t, room.RoomID, "s", EventUserPrompt, "after both were revoked")
	for _, p := range []*offlinePeer{b, c} {
		if n := pull(t, p, a); n != 0 || has(t, p, room.RoomID, "after both were revoked") {
			t.Errorf("a revoked guest's machine took %d events", n)
		}
	}
}

// Changes to the list are relayed like any event, so a guest that never reaches the
// machine that made them still receives them, and every guest computes one list
// (D-235; §13, transitive synchronization).
func TestGuestEventsReachASecondGuestByRelayAndEveryGuestComputesTheSameList(t *testing.T) {
	ana, ben, cal, dee := newTestUser(t), newTestUser(t), newTestUser(t), newTestUser(t)
	a, rooms := newFounder(t, ana.machine(t, "laptop"), ana, 1)
	room := rooms[0]
	b := newUserPeer(t, room.RoomID, room.RoomName, ben.machine(t, "desktop"), ben)
	c := newUserPeer(t, room.RoomID, room.RoomName, cal.machine(t, "tablet"), cal)
	introduceUsers(t, a, b, room.RoomID)
	introduceUsers(t, b, c, room.RoomID)
	for _, u := range []string{ben.id, cal.id, dee.id} {
		if err := a.d.InviteUser(room.RoomID, u); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.d.RevokeUser(room.RoomID, dee.id); err != nil {
		t.Fatal(err)
	}
	pull(t, b, a)
	pull(t, c, b) // c never reaches a

	if n := len(heldEvents(t, c, room.RoomID)); n != 5 {
		t.Fatalf("the second guest holds %d of the 5 changes", n)
	}
	want := a.d.guestList(room.RoomID)
	if want == nil || !want.Admitted[cal.id] || want.Admitted[dee.id] {
		t.Fatalf("the founder's list is %+v", want)
	}
	for _, p := range []*offlinePeer{b, c} {
		got := p.d.guestList(room.RoomID)
		if got == nil || !reflect.DeepEqual(got.Admitted, want.Admitted) || !reflect.DeepEqual(got.Cutoffs, want.Cutoffs) {
			t.Errorf("a guest computes %+v, the founder %+v", got, want)
		}
	}
}

// A change to the list holds no conversation and is never given to an assistant,
// though it is an event of the room like any turn (D-235).
func TestGuestEventsAreNeverInjected(t *testing.T) {
	ana, ben := newTestUser(t), newTestUser(t)
	a, rooms := newFounder(t, ana.machine(t, "laptop"), ana, 1)
	room := rooms[0]
	if err := a.d.InviteUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	if err := a.d.RevokeUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	a.say(t, room.RoomID, "another-session", EventUserPrompt, "A TURN TO INJECT")
	if err := a.d.members.BindSession("s", room.RoomID); err != nil {
		t.Fatal(err)
	}

	pending, err := a.store.UndeliveredFor(room.RoomID, "s")
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].Content != "A TURN TO INJECT" {
		t.Errorf("the session is offered %d events: %+v", len(pending), pending)
	}

	body, _ := json.Marshal(promptReq{SessionID: "s", PromptID: "s-1", Prompt: "hello"})
	r := httptest.NewRequest(http.MethodPost, "/hook/prompt", bytes.NewReader(body))
	r.Header.Set(localGuardHeader, localGuardValue)
	w := httptest.NewRecorder()
	a.d.LocalRoutes().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	out := w.Body.String()
	if !strings.Contains(out, "A TURN TO INJECT") {
		t.Fatalf("the turn was not injected, so the test shows nothing: %s", out)
	}
	for _, leak := range []string{GuestInvite, GuestRevoke, `\"format\"`, ben.id} {
		if strings.Contains(out, leak) {
			t.Errorf("the injected context holds %q: %s", leak, out)
		}
	}
	if got := onlyCounting(heldEvents(t, a, room.RoomID), a.d.guestList(room.RoomID)); len(got) != 2 { // the turn above and the prompt just sent
		t.Errorf("the filter before injection keeps %d events", len(got))
	}
}

// A room's sync targets are the machines from which a session has joined it, so a
// machine of a guest that never joined is not asked, until it sends a sync request
// for the room and is admitted (D-103, a room's sync targets are the machines from
// which a session has joined it).
func TestAGuestsMachineThatNeverJoinedIsNotASyncTargetUntilItAsksForTheRoom(t *testing.T) {
	ana, ben := newTestUser(t), newTestUser(t)
	a, rooms := newFounder(t, ana.machine(t, "laptop"), ana, 1)
	room := rooms[0]
	b1 := newUserPeer(t, room.RoomID, room.RoomName, ben.machine(t, "desktop"), ben)
	b2 := newUserPeer(t, room.RoomID, room.RoomName, ben.machine(t, "tablet"), ben)
	introduceUsers(t, a, b1, room.RoomID)
	copyColleague(t, b2, a, room.RoomID)
	if err := a.d.InviteUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	// This machine knows where the second machine listens, as it would from another
	// room or an offer, which is not a reason to ask it about this room.
	if err := a.d.members.SetMachineEndpoint(b2.d.id.PeerID, ben.id, "tc://second"); err != nil {
		t.Fatal(err)
	}

	t.Setenv("COGMER_PEER_ENDPOINT", "tc://first")
	pull(t, b1, a)
	if got := a.d.members.RoomPeers(room.RoomID, a.d.id.PeerID); !reflect.DeepEqual(got, []string{"tc://first"}) {
		t.Errorf("the room's targets are %v before the second machine asks", got)
	}

	t.Setenv("COGMER_PEER_ENDPOINT", "tc://second")
	pull(t, b2, a)
	got := a.d.members.RoomPeers(room.RoomID, a.d.id.PeerID)
	sortStrings(got)
	if !reflect.DeepEqual(got, []string{"tc://first", "tc://second"}) {
		t.Errorf("the room's targets are %v once the second machine asked", got)
	}
	if ts := a.d.syncTargets(); len(ts) != 2 {
		t.Errorf("the machine polls %v", ts)
	}
}

// Two rooms founded on one machine, with a different guest in each: neither room's
// guest is a guest of the other, neither is a target of the other, and a request for
// the other room is refused (roomscope_test.go, for rooms that keep their lists as
// events).
func TestTwoFoundedRoomsHoldOnlyTheirOwnGuestsAndTargets(t *testing.T) {
	ana, ben, cal := newTestUser(t), newTestUser(t), newTestUser(t)
	a, rooms := newFounder(t, ana.machine(t, "laptop"), ana, 2)
	first, second := rooms[0], rooms[1]
	b := newUserPeer(t, first.RoomID, first.RoomName, ben.machine(t, "desktop"), ben)
	c := newUserPeer(t, second.RoomID, second.RoomName, cal.machine(t, "tablet"), cal)
	introduceUsers(t, a, b, first.RoomID)
	introduceUsers(t, a, c, second.RoomID)
	if err := a.d.InviteUser(first.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	if err := a.d.InviteUser(second.RoomID, cal.id); err != nil {
		t.Fatal(err)
	}

	t.Setenv("COGMER_PEER_ENDPOINT", "tc://ben")
	pull(t, b, a)
	t.Setenv("COGMER_PEER_ENDPOINT", "tc://cal")
	pull(t, c, a)

	if a.d.members.IsGuest(first.RoomID, cal.id) || a.d.members.IsGuest(second.RoomID, ben.id) {
		t.Error("a guest of one room is a guest of the other")
	}
	if got := a.d.members.RoomPeers(first.RoomID, a.d.id.PeerID); !reflect.DeepEqual(got, []string{"tc://ben"}) {
		t.Errorf("the first room's targets are %v", got)
	}
	if got := a.d.members.RoomPeers(second.RoomID, a.d.id.PeerID); !reflect.DeepEqual(got, []string{"tc://cal"}) {
		t.Errorf("the second room's targets are %v", got)
	}
	for _, ask := range []struct {
		who  *offlinePeer
		room Room
	}{{b, second}, {c, first}} {
		cred, err := signRequestAs(ask.who.d.id, ask.room.RoomID, "")
		if err != nil {
			t.Fatal(err)
		}
		req := syncRequest{Protocol: wireVersion, RoomID: ask.room.RoomID, PeerID: cred.PeerID,
			Timestamp: cred.Timestamp, Nonce: cred.Nonce, Signature: cred.Signature, Signer: cred.Signer, Chain: cred.Chain}
		if err := a.d.verifyRequest(req, ask.room.RoomID); err == nil || !strings.Contains(err.Error(), "not a guest") {
			t.Errorf("a guest of one room asking for %s was answered: %v", ask.room.RoomName, err)
		}
	}
}

// Forgetting a colleague leaves a room's list, kept as events, as it is: the list
// belongs to the room (D-073, forgetting a peer changes no room's guest list).
func TestForgettingLeavesTheRoomsListAlone(t *testing.T) {
	ana, ben := newTestUser(t), newTestUser(t)
	a, rooms := newFounder(t, ana.machine(t, "laptop"), ana, 1)
	room := rooms[0]
	if err := a.d.members.Allow(ben.id, "ben"); err != nil {
		t.Fatal(err)
	}
	if err := a.d.InviteUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.Forget(ben.id); err != nil {
		t.Fatal(err)
	}
	if !a.d.members.IsGuest(room.RoomID, ben.id) || a.d.members.Knows(ben.id) {
		t.Error("forgetting changed the room's list, or left the colleague known")
	}
	if got, _ := a.d.members.Guests(room.RoomID); len(got) != 2 {
		t.Errorf("the room's guests are %v", got)
	}
	if n := len(heldEvents(t, a, room.RoomID)); n != 2 {
		t.Errorf("forgetting changed the room's events: %d", n)
	}
}

// A room with no valid founding self-invitation keeps room_guests as its list, even
// on a machine with a user identity and even when its events hold an invitation that
// does not hash to the room's identifier: admission, sync targets and storage behave
// as before (D-240, rooms made before the user identity are not carried into it).
func TestARoomWithoutAFoundingEventBehavesAsBefore(t *testing.T) {
	ana, ben := newTestUser(t), newTestUser(t)
	a := newUserPeer(t, trustRoom, trustRoomName, ana.machine(t, "laptop"), ana)
	b := newUserPeer(t, trustRoom, trustRoomName, ben.machine(t, "desktop"), ben)
	introduceUsers(t, a, b, trustRoom)

	// A self-invitation whose value does not hash to the room founds nothing.
	a.d.mu.Lock()
	_, g, err := a.d.appendGuestEvent(a.store, Room{RoomID: trustRoom}, GuestInvite, guestContent{User: ana.id, Nonce: "AAAAAAAA"})
	a.d.mu.Unlock()
	if err != nil || len(g.Admitted) != 0 {
		t.Fatalf("a self-invitation that does not hash to the room gave %+v, %v", g, err)
	}
	if a.d.members.EventListed(trustRoom) || a.d.guestList(trustRoom) != nil {
		t.Fatal("a room without a valid founding event keeps its list as events")
	}
	if !a.d.members.IsGuest(trustRoom, ben.id) || a.d.members.IsGuest(trustRoom, newTestUser(t).id) {
		t.Error("room_guests no longer decides admission")
	}
	// Every machine of a guest is a target, as before.
	for key, addr := range map[string]string{"one": "tc://ben-one", "two": "tc://ben-two"} {
		m := ben.machine(t, key)
		if err := a.d.members.SetMachineEndpoint(m.PeerID, ben.id, addr); err != nil {
			t.Fatal(err)
		}
	}
	if got := a.d.members.RoomPeers(trustRoom, a.d.id.PeerID); len(got) != 2 {
		t.Errorf("the room's targets are %v", got)
	}
	b.say(t, trustRoom, "s", EventUserPrompt, "from a guest")
	settle(t, a, b, Room{RoomID: trustRoom})
	if !has(t, a, trustRoom, "from a guest") {
		t.Error("a guest's turn was not stored")
	}
	// The other machine stores the stray change as a change: verified and signed for
	// a user, and still founding nothing there either.
	if b.d.members.EventListed(trustRoom) || !b.d.members.IsGuest(trustRoom, ana.id) {
		t.Error("the stray change made the other machine's room keep its list as events")
	}
}
