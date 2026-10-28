package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Propagation of a user's identity records: the withhold rule (D-221, an event whose
// identity version the receiver lacks is withheld from the assistant until fetched),
// the fetch route, the push with acknowledgement (D-225, a machine sends each removal
// and new shared key to every colleague it knows until each acknowledges), the
// refusals that stop a cut-off machine's events being offered on every poll (D-236),
// and the refusal that carries a revocation (D-241).

// switchablePeer is newUserPeer with a server that can be made unreachable, and the
// HOME its state was opened under.
type switchablePeer struct {
	*offlinePeer
	home string
	hits *atomic.Int64
}

func newSwitchablePeer(t *testing.T, roomID, roomName string, id *Identity, u *testUser) *switchablePeer {
	t.Helper()
	p := newUserPeer(t, roomID, roomName, id, u)
	home := os.Getenv("HOME")
	hits := &atomic.Int64{}
	routes := p.d.PeerRoutes()
	p.addr = servePeerTLSHandler(t, p.d, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !p.up.Load() {
			http.Error(w, "unreachable", http.StatusServiceUnavailable)
			return
		}
		hits.Add(1)
		routes.ServeHTTP(w, r)
	}))
	return &switchablePeer{offlinePeer: p, home: home, hits: hits}
}

// removalOf signs a removal of machine by the user's current shared key, naming no
// room, and stores it on each of holders.
func removalOf(t *testing.T, u *testUser, machine string, holders ...*offlinePeer) (Removal, string) {
	t.Helper()
	r, err := SignRemoval(u.shared, LevelShared, u.version, u.id, machine, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(r)
	for _, p := range holders {
		if _, _, err := p.d.members.PutIdentityRecord(RecordRemoval, u.id, raw); err != nil {
			t.Fatal(err)
		}
	}
	h := r.Hash()
	return r, hex.EncodeToString(h[:])
}

func holdsRecord(t *testing.T, p *offlinePeer, hexHash string) bool {
	t.Helper()
	var h [32]byte
	b, _ := hex.DecodeString(hexHash)
	copy(h[:], b)
	_, _, ok, err := p.d.members.IdentityRecordByHash(h)
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

// prompt submits a prompt in session as the hook does, and returns the context it is
// given.
func prompt(t *testing.T, p *offlinePeer, session, promptID string) string {
	t.Helper()
	body, _ := json.Marshal(promptReq{SessionID: session, PromptID: promptID, Prompt: "next"})
	r := httptest.NewRequest(http.MethodPost, "/hook/prompt", bytes.NewReader(body))
	r.Header.Set(localGuardHeader, localGuardValue)
	w := httptest.NewRecorder()
	p.d.LocalRoutes().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	return w.Body.String()
}

// An event naming a removal the receiver lacks is stored and shown, marked as not yet
// given to the assistant, and given at the next prompt once the removal arrives. The
// removal comes from a third machine: the machine that signed the event is offline, and
// the relay the event came through does not hold it, so it is the receiving user's own
// other machine that answers (D-221, asking only the sender lets a thief withhold the
// records).
func TestAnEventNamingARemovalTheReceiverLacksIsShownAndWithheldUntilTheRecordArrivesFromAThirdMachine(t *testing.T) {
	ana, ben, cal := newTestUser(t), newTestUser(t), newTestUser(t)
	a := newUserPeer(t, trustRoom, trustRoomName, ana.machine(t, "laptop"), ana)
	c := newUserPeer(t, trustRoom, trustRoomName, cal.machine(t, "relay"), cal)
	b := newUserPeer(t, trustRoom, trustRoomName, ben.machine(t, "desktop"), ben)
	b2 := newUserPeer(t, trustRoom, trustRoomName, ben.machine(t, "laptop"), ben)
	introduceUsers(t, a, b, trustRoom)
	introduceUsers(t, b, c, trustRoom)
	introduceUsers(t, a, c, trustRoom)
	if err := a.d.members.StoreChain(*a.d.id.Chain); err != nil {
		t.Fatal(err)
	}
	old := ana.machine(t, "old")
	_, h := removalOf(t, ana, old.PeerID, a, b2)

	a.say(t, trustRoom, "a-session", EventUserPrompt, "the turn that names a removal")
	if evs := heldEvents(t, a, trustRoom); len(evs) != 1 || len(evs[0].RemovalsSeen) != 1 || evs[0].RemovalsSeen[0] != h {
		t.Fatalf("the turn does not name the removal: %+v", evs)
	}
	carry(t, a, c, trustRoom)
	// The signer is offline, and the receiver's own other machine holds the removal.
	if err := b.d.members.SetMachineEndpoint(a.d.id.PeerID, ana.id, "127.0.0.1:1"); err != nil {
		t.Fatal(err)
	}
	if err := b.d.members.SetMachineEndpoint(b2.d.id.PeerID, ben.id, b2.addr); err != nil {
		t.Fatal(err)
	}

	if n := pull(t, b, c); n != 1 {
		t.Fatalf("the receiver took %d events from the relay", n)
	}
	if err := b.d.members.BindSession("s", trustRoom); err != nil {
		t.Fatal(err)
	}
	if out := prompt(t, b, "s", "s-1"); strings.Contains(out, "the turn that names a removal") {
		t.Fatalf("a turn naming a removal the receiver lacks was given to the assistant: %s", out)
	}
	st, err := b.d.snapshot(Room{RoomID: trustRoom, RoomName: trustRoomName})
	if err != nil {
		t.Fatal(err)
	}
	shown := false
	for _, e := range st.Events {
		if e.Content == "the turn that names a removal" {
			shown = true
			if !e.NotInSession {
				t.Error("the withheld turn is not marked as not yet given to the assistant")
			}
		}
	}
	if !shown {
		t.Fatal("the withheld turn is not shown in the room view")
	}

	if n := b.d.fetchIdentityWants(time.Now()); n != 1 {
		t.Fatalf("the fetch satisfied %d wants, want the one removal", n)
	}
	if !holdsRecord(t, b, h) {
		t.Fatal("the removal did not arrive")
	}
	if out := prompt(t, b, "s", "s-2"); !strings.Contains(out, "the turn that names a removal") {
		t.Errorf("the turn was not given to the assistant once the removal arrived: %s", out)
	}
}

// fetchAs posts a fetch request signed by id for scope.
func fetchAs(t *testing.T, d *Daemon, id *Identity, scope string, sign bool, hashes ...string) *httptest.ResponseRecorder {
	t.Helper()
	req := identityFetchRequest{Protocol: wireVersion, Hashes: hashes, RoomID: scope}
	if sign {
		c, err := signRequestAs(id, scope, "")
		if err != nil {
			t.Fatal(err)
		}
		req.PeerID, req.Timestamp, req.Nonce, req.Signature, req.Signer, req.Chain = c.PeerID, c.Timestamp, c.Nonce, c.Signature, c.Signer, c.Chain
	} else {
		req.PeerID = id.Peer()
	}
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	d.PeerRoutes().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/identity/fetch", bytes.NewReader(body)))
	return w
}

// The fetch route answers a known colleague with the records it holds of those asked
// for, and refuses a request that is unsigned, from a stranger, or signed for another
// route.
func TestTheFetchRouteReturnsStoredRecordsAndRefusesAnUnsignedOrStrangersRequest(t *testing.T) {
	ana, ben, dan := newTestUser(t), newTestUser(t), newTestUser(t)
	a := newUserPeer(t, trustRoom, trustRoomName, ana.machine(t, "laptop"), ana)
	b := newUserPeer(t, trustRoom, trustRoomName, ben.machine(t, "desktop"), ben)
	introduceUsers(t, a, b, trustRoom)
	_, h := removalOf(t, ana, ana.machine(t, "old").PeerID, a)
	missing := strings.Repeat("ab", 32)

	w := fetchAs(t, a.d, b.d.id, identityFetchScope, true, h, missing)
	if w.Code != http.StatusOK {
		t.Fatalf("a known colleague's fetch was answered %d %s", w.Code, w.Body.String())
	}
	var out identityFetchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Records) != 1 || out.Records[0].Kind != RecordRemoval {
		t.Fatalf("the fetch returned %+v; want the one removal held", out.Records)
	}
	if _, got, err := identityRecordHash(out.Records[0].Kind, out.Records[0].Record); err != nil || hex.EncodeToString(got[:]) != h {
		t.Errorf("the record returned is not the one asked for: %v", err)
	}

	for name, w := range map[string]*httptest.ResponseRecorder{
		"unsigned":                fetchAs(t, a.d, b.d.id, identityFetchScope, false, h),
		"a stranger's":            fetchAs(t, a.d, dan.machine(t, "unknown"), identityFetchScope, true, h),
		"signed for another room": fetchAs(t, a.d, b.d.id, trustRoom, true, h),
		"signed for the push":     fetchAs(t, a.d, b.d.id, identityPushScope, true, h),
	} {
		if w.Code != http.StatusUnauthorized || strings.Contains(w.Body.String(), "signature") {
			t.Errorf("%s request was answered %d %s", name, w.Code, w.Body.String())
		}
	}
}

// pushSetup is a user with one machine and a colleague with two, both offline, which
// the first machine knows the addresses of, and a removal of the first user's that the
// first machine holds.
func pushSetup(t *testing.T) (a, desk, lap *switchablePeer, h string) {
	t.Helper()
	ana, ben := newTestUser(t), newTestUser(t)
	a = newSwitchablePeer(t, trustRoom, trustRoomName, ana.machine(t, "laptop"), ana)
	desk = newSwitchablePeer(t, trustRoom, trustRoomName, ben.machine(t, "desktop"), ben)
	lap = newSwitchablePeer(t, trustRoom, trustRoomName, ben.machine(t, "laptop"), ben)
	introduceUsers(t, a.offlinePeer, desk.offlinePeer, trustRoom)
	copyColleague(t, lap.offlinePeer, a.offlinePeer, trustRoom)
	for _, m := range []*switchablePeer{desk, lap} {
		if err := a.d.members.SetMachineEndpoint(m.d.id.PeerID, ben.id, m.addr); err != nil {
			t.Fatal(err)
		}
	}
	if err := desk.d.members.SetMachineEndpoint(lap.d.id.PeerID, ben.id, lap.addr); err != nil {
		t.Fatal(err)
	}
	// What a's sync with Ben leaves it holding: Ben's chain, so Ben is a user.
	if err := a.d.members.StoreChain(*desk.d.id.Chain); err != nil {
		t.Fatal(err)
	}
	_, h = removalOf(t, ana, ana.machine(t, "stolen").PeerID, a.offlinePeer)
	desk.up.Store(false)
	lap.up.Store(false)
	return a, desk, lap, h
}

func pushRow(t *testing.T, m *Membership, hash, target string) (attempts int, acked bool, ok bool) {
	t.Helper()
	var at *string
	err := m.db.QueryRow(`SELECT attempts, acked_at FROM identity_push WHERE hash = ? AND target = ?`, hash, target).Scan(&attempts, &at)
	if err != nil {
		return 0, false, false
	}
	return attempts, at != nil, true
}

// A removal sent while the colleague's machines are offline is sent again later, not
// at once, reaches the colleague once one machine of theirs is back, and is not sent
// again once that machine has acknowledged it, though the colleague's other machine
// never answered (D-225).
func TestARemovalReachesAColleagueWhoWasOfflineByResendingAndOneMachinesAcknowledgementStopsTheResends(t *testing.T) {
	a, desk, lap, h := pushSetup(t)
	ben := desk.d.id.User
	now := time.Now()

	a.d.pushIdentityRecords(now)
	if attempts, acked, ok := pushRow(t, a.d.members, h, ben); !ok || acked || attempts != 1 {
		t.Fatalf("the push to an offline colleague left attempts %d, acked %v, recorded %v", attempts, acked, ok)
	}
	if n := a.d.pushIdentityRecords(now); n != 0 {
		t.Errorf("the push was sent again at once, and %d acknowledged", n)
	}

	desk.up.Store(true)
	a.d.pushIdentityRecords(now.Add(time.Minute))
	if _, acked, _ := pushRow(t, a.d.members, h, ben); !acked {
		t.Fatal("the colleague's machine that came back did not acknowledge the removal")
	}
	if !holdsRecord(t, desk.offlinePeer, h) {
		t.Fatal("the colleague's machine does not hold the removal")
	}
	if holdsRecord(t, lap.offlinePeer, h) {
		t.Fatal("the colleague's offline machine holds the removal, so the test shows nothing")
	}

	before := desk.hits.Load() + lap.hits.Load()
	lap.up.Store(true)
	for _, later := range []time.Duration{time.Hour, 24 * time.Hour} {
		a.d.pushIdentityRecords(now.Add(later))
	}
	if after := desk.hits.Load() + lap.hits.Load(); after != before {
		t.Errorf("the removal was sent %d more times after one machine acknowledged it", after-before)
	}
}

// A colleague's machine that receives a removal copies it to the colleague's other
// machine, resending until that one acknowledges (D-225; D-222, a machine tells every
// machine it knows, until each has acknowledged).
func TestAReceivedRemovalReachesTheColleaguesOtherMachine(t *testing.T) {
	a, desk, lap, h := pushSetup(t)
	now := time.Now()
	desk.up.Store(true)
	a.d.pushIdentityRecords(now)
	if !holdsRecord(t, desk.offlinePeer, h) {
		t.Fatal("the removal did not reach the colleague's first machine")
	}

	desk.d.pushIdentityRecords(now) // the other machine is still offline
	if attempts, acked, ok := pushRow(t, desk.d.members, h, lap.d.id.PeerID); !ok || acked || attempts != 1 {
		t.Fatalf("the copy to the colleague's other machine left attempts %d, acked %v, recorded %v", attempts, acked, ok)
	}
	lap.up.Store(true)
	desk.d.pushIdentityRecords(now.Add(time.Minute))
	if !holdsRecord(t, lap.offlinePeer, h) {
		t.Fatal("the removal did not reach the colleague's other machine")
	}
	if _, acked, _ := pushRow(t, desk.d.members, h, lap.d.id.PeerID); !acked {
		t.Error("the copy is not recorded as acknowledged")
	}
}

// What a machine owes is in membership.db before it is sent, so a daemon started again
// sends what the last one had not had acknowledged, with its attempts counted.
func TestAPendingPushSurvivesARestart(t *testing.T) {
	a, desk, _, h := pushSetup(t)
	ben := desk.d.id.User
	now := time.Now()
	a.d.pushIdentityRecords(now)
	if _, _, ok := pushRow(t, a.d.members, h, ben); !ok {
		t.Fatal("nothing was recorded as owed")
	}
	if err := a.d.members.Close(); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", a.home)
	m, err := OpenMembership()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	restarted := &Daemon{id: a.d.id, members: m, stores: map[string]*Store{}}
	t.Cleanup(restarted.closeStores)
	if attempts, acked, ok := pushRow(t, m, h, ben); !ok || acked || attempts != 1 {
		t.Fatalf("after the restart the push has attempts %d, acked %v, recorded %v", attempts, acked, ok)
	}
	desk.up.Store(true)
	restarted.pushIdentityRecords(now.Add(time.Minute))
	if !holdsRecord(t, desk.offlinePeer, h) {
		t.Fatal("the restarted daemon did not send the pending removal")
	}
}

// A revoked user's machine's events past the cutoff are refused once, recorded so that
// sync no longer asks for them, so the machine that holds them offers them once and not
// on every poll, and they arrive by the next sync once an invitation lifts the
// revocation (D-236).
func TestACutOffMachinesEventsAreOfferedOnceAndArriveAfterTheRevocationIsLifted(t *testing.T) {
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
	b.say(t, room.RoomID, "s", EventUserPrompt, "past one")
	b.say(t, room.RoomID, "s", EventUserPrompt, "past two")
	pull(t, c, a)
	pull(t, c, b)

	offered := func() int {
		have, err := c.store.SyncState(room.RoomID)
		if err != nil {
			t.Fatal(err)
		}
		evs, err := b.store.EventsSince(room.RoomID, have)
		if err != nil {
			t.Fatal(err)
		}
		return len(evs)
	}
	if n := offered(); n != 0 {
		t.Errorf("after one poll the cut-off machine still offers %d events", n)
	}
	if n := pull(t, c, b); n != 0 || has(t, c, room.RoomID, "past one") {
		t.Errorf("a later poll stored %d events", n)
	}

	if err := a.d.InviteUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	pull(t, c, a)
	if n := offered(); n != 2 {
		t.Errorf("once the revocation is lifted the cut-off machine offers %d events, want 2", n)
	}
	pull(t, c, b)
	if !has(t, c, room.RoomID, "past one") || !has(t, c, room.RoomID, "past two") {
		t.Error("the turns past a lifted cutoff did not arrive by the next sync")
	}
}

// syncAs posts a sync request signed by id for room to d.
func syncAs(t *testing.T, d *Daemon, id *Identity, roomID string) *httptest.ResponseRecorder {
	t.Helper()
	c, err := signRequestAs(id, roomID, "")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(syncRequest{Protocol: wireVersion, RoomID: roomID, PeerID: c.PeerID, Timestamp: c.Timestamp,
		Nonce: c.Nonce, Signature: c.Signature, Signer: c.Signer, Chain: c.Chain})
	w := httptest.NewRecorder()
	d.PeerRoutes().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/sync", bytes.NewReader(body)))
	return w
}

// A revoked user's sync request is refused with the room's guest-list events and none
// of its conversation, their machine checks and stores them, and Expulsion reads the
// revocation. A user who was never invited is refused with nothing, and a refusal for
// one room carries none of another room's events (D-241).
func TestARevokedUsersRefusedRequestCarriesTheGuestListEventsAndNoConversation(t *testing.T) {
	ana, ben, cal := newTestUser(t), newTestUser(t), newTestUser(t)
	a, rooms := newFounder(t, ana.machine(t, "laptop"), ana, 2)
	room, other := rooms[0], rooms[1]
	if err := a.d.InviteUser(other.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	if err := a.d.RevokeUser(other.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	b := newUserPeer(t, room.RoomID, room.RoomName, ben.machine(t, "desktop"), ben)
	c := newUserPeer(t, room.RoomID, room.RoomName, cal.machine(t, "tablet"), cal)
	introduceUsers(t, a, b, room.RoomID)
	introduceUsers(t, a, c, room.RoomID)
	if err := a.d.InviteUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	pull(t, b, a)
	held := len(heldEvents(t, b, room.RoomID))
	a.say(t, room.RoomID, "s", EventUserPrompt, "said before the revocation")
	if err := a.d.RevokeUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	a.say(t, room.RoomID, "s", EventUserPrompt, "said after the revocation")

	w := syncAs(t, a.d, b.d.id, room.RoomID)
	if w.Code != http.StatusForbidden {
		t.Fatalf("a revoked user's request was answered %d %s", w.Code, w.Body.String())
	}
	var refusal revocationRefusal
	if err := json.Unmarshal(w.Body.Bytes(), &refusal); err != nil {
		t.Fatal(err)
	}
	// The founding, the invitation of ben and his revocation, in that order, and no
	// turn of the room.
	kinds := func(evs []wireEvent, roomID string) []string {
		var out []string
		for _, e := range evs {
			if e.RoomID != roomID {
				out = append(out, "another room's "+e.EventType)
				continue
			}
			out = append(out, e.EventType)
		}
		return out
	}
	want := []string{GuestInvite, GuestInvite, GuestRevoke}
	if got := kinds(refusal.GuestEvents, room.RoomID); !reflect.DeepEqual(got, want) {
		t.Fatalf("the refusal carries %v; want this room's guest-list events %v", got, want)
	}
	var elsewhere revocationRefusal
	if w := syncAs(t, a.d, b.d.id, other.RoomID); w.Code != http.StatusForbidden || json.Unmarshal(w.Body.Bytes(), &elsewhere) != nil ||
		!reflect.DeepEqual(kinds(elsewhere.GuestEvents, other.RoomID), want) {
		t.Errorf("the refusal for the other room was %d %s", w.Code, w.Body.String())
	}
	for _, leak := range []string{"said before the revocation", "said after the revocation", `"events"`} {
		if strings.Contains(w.Body.String(), leak) {
			t.Errorf("the refusal carries %q", leak)
		}
	}
	if w := syncAs(t, a.d, c.d.id, room.RoomID); w.Code != http.StatusUnauthorized || strings.Contains(w.Body.String(), GuestRevoke) {
		t.Errorf("a user never invited was answered %d %s", w.Code, w.Body.String())
	}

	pull(t, b, a)
	evs := heldEvents(t, b, room.RoomID)
	if len(evs) != held+1 || evs[len(evs)-1].EventType != GuestRevoke {
		t.Fatalf("the expelled machine holds %d events, %d before; want the revocation added alone", len(evs), held)
	}
	ex, ok := b.d.Expulsion(room.RoomID)
	if !ok || !ex.Expelled || len(ex.Revocations) != 1 || ex.Revocations[0].Author != ana.id {
		t.Errorf("the expelled machine reads %+v, %v", ex, ok)
	}
	if _, ok := a.d.Expulsion(room.RoomID); ok {
		t.Error("the revoking machine reads an expulsion of its own user")
	}
}

// A revocation counts only when its author is on the list, so a refusal carries the
// room's invitations with its revocations: an expelled machine that never held the
// invitation of the guest who revoked it still learns that it was expelled (D-241).
func TestAnExpelledMachineThatNeverHeldTheRevokersInvitationStillReadsItselfExpelled(t *testing.T) {
	ana, ben, cal := newTestUser(t), newTestUser(t), newTestUser(t)
	a, rooms := newFounder(t, ana.machine(t, "laptop"), ana, 1)
	room := rooms[0]
	b := newUserPeer(t, room.RoomID, room.RoomName, ben.machine(t, "desktop"), ben)
	c := newUserPeer(t, room.RoomID, room.RoomName, cal.machine(t, "tablet"), cal)
	introduceUsers(t, a, b, room.RoomID)
	introduceUsers(t, a, c, room.RoomID)
	introduceUsers(t, b, c, room.RoomID)
	if err := a.d.InviteUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	pull(t, b, a) // ben's machine holds the founding and its own invitation
	if err := a.d.InviteUser(room.RoomID, cal.id); err != nil {
		t.Fatal(err)
	}
	pull(t, c, a)
	if err := c.d.RevokeUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	for _, e := range heldEvents(t, b, room.RoomID) {
		if g, ok := guestEventFrom(e); ok && g.Subject == cal.id {
			t.Fatal("the setup is wrong: ben's machine already holds cal's invitation")
		}
	}

	pull(t, b, c) // refused, with the room's guest-list events
	ex, ok := b.d.Expulsion(room.RoomID)
	if !ok || !ex.Expelled || len(ex.Revocations) != 1 || ex.Revocations[0].Author != cal.id {
		t.Errorf("the expelled machine reads %+v, %v; want expelled by cal", ex, ok)
	}
}

// Inviting back an expelled user delivers an offer as any invitation does, and their
// machine, on an offer from a guest still on the list, syncs and serves the room
// again, so the turn their session captured while expelled reaches the room by
// ordinary sync with no other step (D-105, an invitation is delivered as an offer;
// D-241, an invitation that lifts the revocation lets those turns reach the room).
func TestAnExpelledUsersTurnReachesTheRoomOnceTheyAreInvitedBack(t *testing.T) {
	ana, ben := newTestUser(t), newTestUser(t)
	a, rooms := newFounder(t, ana.machine(t, "laptop"), ana, 1)
	room := rooms[0]
	b := newUserPeer(t, room.RoomID, room.RoomName, ben.machine(t, "desktop"), ben)
	introduceUsers(t, a, b, room.RoomID)
	if err := a.d.InviteUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	pull(t, b, a)
	if err := b.d.members.SetMachineEndpoint(a.d.id.PeerID, ana.id, a.addr); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.SetMachineEndpoint(b.d.id.PeerID, ben.id, b.addr); err != nil {
		t.Fatal(err)
	}
	if err := a.d.RevokeUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	pull(t, b, a) // refused, with the revocation
	if !b.d.expelledFrom(room.RoomID) {
		t.Fatal("ben's machine did not learn that he was expelled")
	}
	b.say(t, room.RoomID, "s", EventUserPrompt, "written while expelled")
	pull(t, a, b)
	if has(t, a, room.RoomID, "written while expelled") {
		t.Fatal("a turn written while expelled reached the room before the invitation")
	}

	// The invitation, as /cogmer:room-invite makes it: the event, then the offer.
	if err := a.d.InviteUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	if err := a.d.deliverOffer(ben.id, room); err != nil {
		t.Fatalf("the offer was not delivered: %v", err)
	}
	if b.d.expelledFrom(room.RoomID) {
		t.Fatal("the offer did not resume the room on ben's machine")
	}
	if _, ok := b.d.members.FindOffer(room.RoomID); ok {
		t.Error("a room ben's sessions are still in was recorded as an offer to accept")
	}

	pull(t, b, a)
	pull(t, a, b)
	if !has(t, a, room.RoomID, "written while expelled") {
		t.Error("the turn written while expelled did not reach the room by ordinary sync")
	}
	if _, expelled := b.d.listExpels(room.RoomID); expelled {
		t.Error("ben's machine did not receive the invitation that lifted the revocation")
	}
	if _, _, ok := b.d.members.Resumption(room.RoomID); ok {
		t.Error("the resumption outlived the invitation it waited for")
	}
}

// An offer resumes a room only from a guest the list admits, and a revocation made
// after the offer ends the resumption (D-241).
func TestAResumptionNeedsAnAdmittedGuestAndEndsAtANewRevocation(t *testing.T) {
	ana, ben, cal := newTestUser(t), newTestUser(t), newTestUser(t)
	a, rooms := newFounder(t, ana.machine(t, "laptop"), ana, 1)
	room := rooms[0]
	b := newUserPeer(t, room.RoomID, room.RoomName, ben.machine(t, "desktop"), ben)
	introduceUsers(t, a, b, room.RoomID)
	if err := a.d.InviteUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	pull(t, b, a)
	if err := a.d.RevokeUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	pull(t, b, a)
	if b.d.resumeOnOffer(offerRequest{RoomID: room.RoomID, PeerID: cal.id}) || !b.d.expelledFrom(room.RoomID) {
		t.Fatal("an offer from a user not on the list resumed the room")
	}
	if !b.d.resumeOnOffer(offerRequest{RoomID: room.RoomID, PeerID: ana.id, Signer: a.d.id.PeerID, Endpoint: a.addr}) ||
		b.d.expelledFrom(room.RoomID) {
		t.Fatal("an offer from a guest on the list did not resume the room")
	}
	// A second revocation, made after the offer, arrives in the next refusal.
	if err := a.d.RevokeUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	pull(t, b, a)
	if !b.d.expelledFrom(room.RoomID) {
		t.Error("a revocation made after the offer left the room resumed")
	}
}

// A machine whose user is revoked from one room answers none of that room's sync
// requests and drops it from its targets, and keeps serving and asking for a room it
// is still a guest of (D-241).
func TestARevokedMachineStopsServingTheRoom(t *testing.T) {
	ana, ben := newTestUser(t), newTestUser(t)
	a, rooms := newFounder(t, ana.machine(t, "laptop"), ana, 2)
	out, still := rooms[0], rooms[1]
	b := newUserPeer(t, out.RoomID, out.RoomName, ben.machine(t, "desktop"), ben)
	if err := b.d.members.RecordRoom(still.RoomID, still.RoomName, false); err != nil {
		t.Fatal(err)
	}
	if _, err := b.d.storeFor(still.RoomID); err != nil { // under b's HOME
		t.Fatal(err)
	}
	for _, r := range rooms {
		introduceUsers(t, a, b, r.RoomID)
		if err := a.d.InviteUser(r.RoomID, ben.id); err != nil {
			t.Fatal(err)
		}
	}
	pull(t, b, a)
	for _, r := range rooms {
		if err := b.d.members.RecordRoomMachine(r.RoomID, a.d.id.PeerID, ana.id); err != nil {
			t.Fatal(err)
		}
	}
	if err := b.d.members.SetMachineEndpoint(a.d.id.PeerID, ana.id, a.addr); err != nil {
		t.Fatal(err)
	}
	if err := a.d.RevokeUser(out.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	pull(t, b, a) // learns of the revocation from the refusal
	if !b.d.expelledFrom(out.RoomID) || b.d.expelledFrom(still.RoomID) {
		t.Fatal("the machine does not hold its user's revocation from the one room alone")
	}

	if w := syncAs(t, b.d, a.d.id, out.RoomID); w.Code != http.StatusUnauthorized {
		t.Errorf("the expelled machine answered a request for the room it was revoked from: %d", w.Code)
	}
	if w := syncAs(t, b.d, a.d.id, still.RoomID); w.Code != http.StatusOK {
		t.Errorf("the machine refused a request for the room it is still a guest of: %d %s", w.Code, w.Body.String())
	}
	all, _ := b.d.members.Rooms()
	asked := b.d.roomsToAsk(a.addr, all)
	if len(asked) != 1 || asked[0].RoomID != still.RoomID {
		t.Errorf("the machine asks %v of the guest's address", asked)
	}
	if got := b.d.roomTargets(out.RoomID); len(got) != 0 {
		t.Errorf("the room it was revoked from has targets %v", got)
	}
	if got := b.d.roomTargets(still.RoomID); len(got) != 1 {
		t.Errorf("the room it is still a guest of has targets %v", got)
	}
}

// A record is owed to no removed machine and for no more than a week of failed attempts
// (D-268, a record is owed to no removed machine and for no more than a week of failed
// attempts).
func TestARemovedMachineIsOwedNothingAndWhatWasOwedItIsDropped(t *testing.T) {
	a, b := twoMachines(t)
	keep := b.d.id.PeerID
	owed := func() int {
		var n int
		_ = a.d.members.db.QueryRow(`SELECT COUNT(*) FROM identity_push WHERE target = ?`, keep).Scan(&n)
		return n
	}
	if err := a.d.members.owePushes(a.d.id); err != nil {
		t.Fatal(err)
	}
	if owed() == 0 {
		t.Fatal("a machine that is not removed is owed nothing, so the test shows nothing")
	}
	var res identityResult
	a.local(t, "/machines/remove", removeRequest{Machine: keep}, &res)
	if res.Status != "removed" {
		t.Fatalf("removing: %+v", res)
	}
	if err := a.d.members.owePushes(a.d.id); err != nil {
		t.Fatal(err)
	}
	if n := owed(); n != 0 {
		t.Errorf("%d records are still owed to a machine that was removed", n)
	}
	// and a record made after the removal is owed to it no more than what was owed before
	if err := a.d.members.oweToOwnMachines("deadbeef", RecordRemoval, a.d.id, ""); err != nil {
		t.Fatal(err)
	}
	if n := owed(); n != 0 {
		t.Errorf("a new record was owed to the removed machine (%d rows)", n)
	}
}

func TestARecordIsNotSentAfterAWeekOfFailedAttempts(t *testing.T) {
	a, _ := twoMachines(t)
	insert := func(hash string, attempts int) {
		t.Helper()
		if _, err := a.d.members.db.Exec(`INSERT INTO identity_push (hash, kind, target, target_kind, attempts)
			VALUES (?, ?, 'ed25519:someone', ?, ?)`, hash, RecordRemoval, pushToMachine, attempts); err != nil {
			t.Fatal(err)
		}
	}
	insert("under", pushAttemptLimit-1)
	insert("at", pushAttemptLimit)
	insert("past", pushAttemptLimit+100)
	due, err := a.d.members.duePushes(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, p := range due {
		got[p.Hash] = true
	}
	if !got["under"] || got["at"] || got["past"] {
		t.Errorf("due after a week of attempts: %v", got)
	}
	// the backoff means the limit is about a week: ten minutes at the longest wait
	if weeks := time.Duration(pushAttemptLimit) * propagationBackoff(1000); weeks < 6*24*time.Hour || weeks > 8*24*time.Hour {
		t.Errorf("the limit of %d attempts is %s of waiting at the longest backoff, not about a week", pushAttemptLimit, weeks)
	}
}
