package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Removal, recovery and what the user sees (D-220, D-223, D-232, D-233, D-234, D-194,
// D-241): removing a machine from the page, certifying a new shared key with the phrase
// typed into the page, and what the view, the export, injection and the room status say
// of removed, suspended and expelled turns.

// otherPhrase is a second fixed phrase, for a second user made through the page.
func otherPhrase() Phrase {
	var p Phrase
	for i := range p {
		p[i] = byte(0x40 + 3*i)
	}
	return p
}

// promptOn submits a prompt on a machine as the hook does and returns what the session is given.
func promptOn(t *testing.T, m *machine, session, promptID, text string) string {
	t.Helper()
	m.use(t)
	body, _ := json.Marshal(promptReq{SessionID: session, PromptID: promptID, Prompt: text})
	r := httptest.NewRequest(http.MethodPost, "/hook/prompt", bytes.NewReader(body))
	r.Header.Set(localGuardHeader, localGuardValue)
	w := httptest.NewRecorder()
	m.d.LocalRoutes().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("prompt: %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Context string `json:"context"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out.Context
}

// pageAndExport is what the room view and the export show of a room on a machine.
func pageAndExport(t *testing.T, m *machine, room Room) (uiState, string) {
	t.Helper()
	m.use(t)
	st, err := m.d.snapshot(room)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	m.d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest("GET", "/export?room="+room.RoomName, nil))
	return st, rec.Body.String()
}

func shownTurn(st uiState, text string) *uiEvent {
	for i := range st.Events {
		if st.Events[i].Content == text {
			return &st.Events[i]
		}
	}
	return nil
}

// sharedWith is a room founded on a and joined by name on b, with b holding the room.
func sharedWith(t *testing.T, a, b *machine, say string) Room {
	t.Helper()
	room := foundAndSay(t, a, say)
	a.d.exchangeSelfLists()
	joinByName(t, b, room.RoomName, "b-session")
	b.use(t)
	if _, err := b.d.pullFrom(a.addr); err != nil {
		t.Fatal(err)
	}
	keepAddresses(t, a, b)
	return room
}

func pullOn(t *testing.T, m *machine, from string) {
	t.Helper()
	m.use(t)
	if _, err := m.d.pullFrom(from); err != nil {
		t.Fatal(err)
	}
}

// keepAddresses records each machine's own address on the others. A signed request
// carries the address the process advertises, which every daemon in one test process
// shares, so a pull leaves the machine it reached holding that one for the puller.
func keepAddresses(t *testing.T, ms ...*machine) {
	t.Helper()
	for _, m := range ms {
		for _, o := range ms {
			if o == m || o.d.id.User != m.d.id.User {
				continue
			}
			if err := m.d.members.SetMachineEndpoint(o.d.id.PeerID, o.d.id.User, o.addr); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// The new-key flow takes the phrase in the page's request body and nowhere else: no
// command parses a phrase, the routes refuse a request without the guard header, and no
// response, log line or file written carries a word of it (D-223, the recovery phrase
// is typed only into the daemon's browser page).
func TestThePhraseReachesNoCommandArgumentOrLogInTheNewKeyFlow(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	a, b := twoMachines(t)
	sharedWith(t, a, b, "BEFORE THE NEW KEY")

	words := fixedPhrase().Words()
	var bodies []string
	var pv newKeyPreview
	bodies = append(bodies, a.local(t, "/machines/newkey/preview", struct{}{}, &pv))
	var res newKeyResult
	bodies = append(bodies, a.local(t, "/machines/newkey", newKeyRequest{Phrase: strings.Join(words, " "),
		Machines: []string{b.d.id.PeerID}}, &res))
	if res.Status != "certified" || res.Version != 2 {
		t.Fatalf("certifying a new key: %+v", res)
	}
	// A wrong phrase is refused without a word of it.
	wrong := append([]string{}, words...)
	wrong[3] = words[5]
	bodies = append(bodies, a.local(t, "/machines/newkey", newKeyRequest{Phrase: strings.Join(wrong, " ")}, &res))
	if res.Error == "" {
		t.Fatal("a wrong phrase was taken")
	}
	a.d.exchangeSelfLists()

	var files []string
	for _, m := range []*machine{a, b} {
		_ = filepath.Walk(m.d.home, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				if raw, err := os.ReadFile(path); err == nil {
					files = append(files, path+"\n"+string(raw))
				}
			}
			return nil
		})
	}
	for _, w := range words {
		for i, body := range bodies {
			if hasWord(body, w) {
				t.Errorf("response %d holds a word of the phrase: %s", i, body)
			}
		}
		if hasWord(logs.String(), w) {
			t.Errorf("the log holds a word of the phrase:\n%s", logs.String())
		}
		for _, f := range files {
			if hasWord(f, w) {
				t.Errorf("a file holds a word of the phrase: %s", strings.SplitN(f, "\n", 2)[0])
			}
		}
	}

	// Without the guard header a page from another site cannot send the phrase.
	for _, path := range []string{"/machines/newkey", "/machines/newkey/preview", "/machines/remove", "/machines/remove/propose"} {
		w := httptest.NewRecorder()
		a.d.LocalRoutes().ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`)))
		if w.Code == http.StatusOK {
			t.Errorf("%s answered a request without the guard header", path)
		}
	}
	// Only the page's handlers parse a phrase; no command or argument does.
	srcs, _ := filepath.Glob("*.go")
	for _, src := range srcs {
		if strings.HasSuffix(src, "_test.go") || src == "userphrase.go" {
			continue
		}
		raw, _ := os.ReadFile(src)
		if bytes.Contains(raw, []byte("ParsePhrase(")) && src != "identitypage.go" && src != "removal.go" {
			t.Errorf("%s parses a phrase outside the page's handlers", src)
		}
	}
}

// Not stolen, the page proposes in each room the last of the machine's turns this machine
// holds; stolen, it proposes 0 in every room, and a cutoff past the last held turn is
// refused (D-220). Two rooms, each with its own last turn.
func TestARemovalProposesTheLastHeldTurnPerRoomOrZeroWhenStolen(t *testing.T) {
	a, b := twoMachines(t)
	first := sharedWith(t, a, b, "FIRST ROOM")
	second := foundAndSay(t, a, "SECOND ROOM")
	a.d.exchangeSelfLists()
	joinByName(t, b, second.RoomName, "b-second")
	pullOn(t, b, a.addr)
	keepAddresses(t, a, b)
	say1(t, b, first, "ONE")
	say1(t, b, first, "TWO")
	say1(t, b, second, "THREE")
	pullOn(t, a, b.addr)

	var p removeProposeResponse
	a.local(t, "/machines/remove/propose", removeProposeRequest{Machine: b.d.id.PeerID}, &p)
	got := map[string][2]int64{}
	for _, r := range p.Rooms {
		got[r.Room] = [2]int64{r.Last, r.Proposed}
	}
	if got[first.RoomID] != [2]int64{2, 2} || got[second.RoomID] != [2]int64{1, 1} || p.Name != "desk" {
		t.Fatalf("not stolen, the proposal is %+v", p)
	}
	a.local(t, "/machines/remove/propose", removeProposeRequest{Machine: b.d.id.PeerID, Stolen: true}, &p)
	for _, r := range p.Rooms {
		if r.Proposed != 0 || r.Last == 0 {
			t.Errorf("stolen, %s proposes %d with last %d", r.RoomName, r.Proposed, r.Last)
		}
	}

	var res identityResult
	a.local(t, "/machines/remove", removeRequest{Machine: b.d.id.PeerID, Cutoffs: map[string]int64{first.RoomID: 3}}, &res)
	if res.Error == "" {
		t.Error("a cutoff past the last held turn was taken")
	}
	a.local(t, "/machines/remove", removeRequest{Machine: b.d.id.PeerID, Cutoffs: map[string]int64{first.RoomID: 1}}, &res)
	if res.Status != "removed" {
		t.Fatalf("removing: %+v", res)
	}
	st := a.d.members.UserState(a.d.id.User, nil)
	ms := st.Machines[b.d.id.PeerID]
	if ms == nil || !ms.Removed || ms.Cutoffs[first.RoomID] != 1 || ms.Cutoffs[second.RoomID] != 1 {
		t.Fatalf("the removal left %+v", ms)
	}
	for h, r := range st.Removals {
		if r.SignerLevel != LevelShared || r.SignerVersion != 1 {
			t.Errorf("removal %x is signed at level %d version %d, not by the current shared key", h[:4], r.SignerLevel, r.SignerVersion)
		}
	}
}

// A removal is pushed to a colleague, who from then on does not count the removed
// machine's turns past its cutoff (D-220; D-225, a machine sends each removal to every
// colleague until each acknowledges).
func TestARemovalIsPushedAndAColleagueStopsCountingTheRemovedMachinesLaterTurns(t *testing.T) {
	a, b := twoMachines(t)
	room := foundAndSay(t, a, "FROM THE LAPTOP")
	a.d.exchangeSelfLists()
	ben := newTestUser(t)
	bp := newUserPeer(t, room.RoomID, room.RoomName, ben.machine(t, "desktop"), ben)
	user := a.d.id.User
	for _, pair := range []struct {
		m    *Membership
		peer string
	}{{a.d.members, ben.id}, {bp.d.members, user}, {b.d.members, ben.id}} {
		if err := pair.m.Allow(pair.peer, ""); err != nil {
			t.Fatal(err)
		}
		if err := pair.m.MarkVerified(pair.peer); err != nil {
			t.Fatal(err)
		}
	}
	a.use(t)
	if err := a.d.InviteUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	joinByName(t, b, room.RoomName, "b-session")
	pullOn(t, b, a.addr)
	say1(t, b, room, "B WITHIN")
	say1(t, b, room, "B PAST")
	pullOn(t, a, b.addr)
	// The colleague holds the turn within, and a's own store holds both.
	a.use(t)
	store, _ := a.d.storeFor(room.RoomID)
	evs, _ := store.ListRoom(room.RoomID)
	var past *Event
	for i := range evs {
		if evs[i].Content == "B PAST" {
			past = &evs[i]
		}
	}
	if past == nil {
		t.Fatal("the laptop does not hold the desk's later turn")
	}
	if _, err := bp.d.pullFrom(a.addr); err != nil {
		t.Fatal(err)
	}
	keepAddresses(t, a, b)
	if !has(t, bp, room.RoomID, "B WITHIN") {
		t.Fatal("the colleague did not receive the desk's turn")
	}
	judge := func() error {
		ev := *past
		bp.d.mu.Lock()
		defer bp.d.mu.Unlock()
		return bp.d.judgeEvent(&ev)
	}
	if err := judge(); err != nil {
		t.Fatalf("before the removal the colleague does not count the later turn: %v", err)
	}

	var res identityResult
	a.local(t, "/machines/remove", removeRequest{Machine: b.d.id.PeerID, Cutoffs: map[string]int64{room.RoomID: 1}}, &res)
	if res.Status != "removed" {
		t.Fatalf("removing: %+v", res)
	}
	if err := a.d.members.SetMachineEndpoint(bp.d.id.PeerID, ben.id, bp.addr); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.StoreChain(*bp.d.id.Chain); err != nil {
		t.Fatal(err)
	}
	if n := a.d.pushIdentityRecords(time.Now()); n == 0 {
		t.Fatal("the removal was acknowledged by nobody")
	}
	if err := judge(); !errors.Is(err, errNotCounting) {
		t.Errorf("after the removal arrived the colleague judges the later turn %v", err)
	}
	st := bp.d.members.UserState(user, nil)
	if !st.Counts(b.d.id.PeerID, room.RoomID, 1) || st.Counts(b.d.id.PeerID, room.RoomID, 2) {
		t.Error("the colleague does not count the turn within the cutoff and only it")
	}
}

// Certifying a new shared key carries only the removals the user kept, certifies again
// the machines kept and gives them the new key over the pinned connection, and leaves a
// machine not certified again suspended (D-220, D-234, D-219).
func TestANewSharedKeyCarriesOnlyTheConfirmedRemovalsAndSuspendsAMachineNotCertifiedAgain(t *testing.T) {
	a, b := twoMachines(t)
	c, e := newMachine(t), newMachine(t)
	addMachine(t, a, c, "old")
	addMachine(t, a, e, "strange")
	keepAddresses(t, a, b, c, e)
	a.d.exchangeSelfLists()
	user := a.d.id.User

	// The user removes the old machine from the page; a thief holding the shared key
	// removes the desk.
	var res identityResult
	a.local(t, "/machines/remove", removeRequest{Machine: c.d.id.PeerID}, &res)
	if res.Status != "removed" {
		t.Fatalf("removing: %+v", res)
	}
	shared, version, _, _ := loadSharedKeyIn(a.d.home)
	thief, err := SignRemoval(shared, LevelShared, version, user, b.d.id.PeerID, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(thief)
	if _, _, err := a.d.members.PutIdentityRecord(RecordRemoval, user, raw); err != nil {
		t.Fatal(err)
	}

	var pv newKeyPreview
	a.local(t, "/machines/newkey/preview", struct{}{}, &pv)
	th := thief.Hash()
	var keep []string
	signers := map[string]string{}
	for _, r := range pv.Removals {
		signers[r.Hash] = r.SignedBy
		if r.Hash != hex.EncodeToString(th[:]) {
			keep = append(keep, r.Hash)
		}
	}
	if len(pv.Removals) != 2 || len(keep) != 1 || signers[keep[0]] != "shared key version 1" || pv.Version != 2 {
		t.Fatalf("the page lists %+v", pv)
	}
	var nk newKeyResult
	a.local(t, "/machines/newkey", newKeyRequest{Phrase: strings.Join(fixedPhrase().Words(), " "), Keep: keep,
		Machines: []string{b.d.id.PeerID, c.d.id.PeerID}}, &nk)
	if nk.Status != "certified" || nk.Removals != 1 || nk.Suspended != 1 {
		t.Fatalf("certifying: %+v", nk)
	}

	st := a.d.members.UserState(user, nil)
	switch {
	case st.Current != 2:
		t.Errorf("the current version is %d", st.Current)
	case !st.Machines[b.d.id.PeerID].Current || st.Machines[b.d.id.PeerID].Removed:
		t.Error("the thief's removal carried over to the new key")
	case !st.Machines[c.d.id.PeerID].Current || !st.Machines[c.d.id.PeerID].Removed:
		t.Error("the removal the user kept did not carry over")
	case st.Machines[e.d.id.PeerID].Current || st.Counts(e.d.id.PeerID, "any", 1):
		t.Error("the machine not certified again is not suspended")
	}
	if a.d.heldSharedVersion() != 2 || a.d.id.Chain.Shared.Version != 2 {
		t.Errorf("this machine holds key %d and chain %d", a.d.heldSharedVersion(), a.d.id.Chain.Shared.Version)
	}

	// The desk is certified again and is sent the new key; the strange machine is not.
	keepAddresses(t, a, b, c, e)
	a.d.exchangeSelfLists()
	if b.d.heldSharedVersion() != 2 || b.d.id.Chain.Shared.Version != 2 {
		t.Errorf("the desk holds key %d and chain %d after the exchange", b.d.heldSharedVersion(), b.d.id.Chain.Shared.Version)
	}
	if e.d.heldSharedVersion() != 1 {
		t.Errorf("the suspended machine holds key %d", e.d.heldSharedVersion())
	}
}

// A suspended machine's turn is in no view, export or hook output, and returns once
// the machine is certified again under the current key (D-234).
func TestASuspendedMachinesTurnAppearsInNoViewExportOrHookOutputAndReturnsWhenCertifiedAgain(t *testing.T) {
	a, b := twoMachines(t)
	room := sharedWith(t, a, b, "FROM THE LAPTOP")
	say1(t, b, room, "FROM THE DESK")
	pullOn(t, a, b.addr)
	if err := a.d.members.BindSession("a-session", room.RoomID); err != nil {
		t.Fatal(err)
	}

	var nk newKeyResult
	a.local(t, "/machines/newkey", newKeyRequest{Phrase: strings.Join(fixedPhrase().Words(), " ")}, &nk)
	if nk.Status != "certified" || nk.Suspended != 1 {
		t.Fatalf("certifying: %+v", nk)
	}
	st, export := pageAndExport(t, a, room)
	if shownTurn(st, "FROM THE DESK") != nil || strings.Contains(export, "FROM THE DESK") {
		t.Error("the suspended machine's turn is in the view or the export")
	}
	if !strings.Contains(export, "FROM THE LAPTOP") {
		t.Fatalf("the export lost the laptop's turn: %s", export)
	}
	if got := promptOn(t, a, "a-session", "p1", "hello"); strings.Contains(got, "FROM THE DESK") {
		t.Errorf("the suspended machine's turn was given to a session: %s", got)
	}

	a.use(t)
	if err := a.d.certifyMachine(b.d.id.PeerID, "desk"); err != nil {
		t.Fatal(err)
	}
	st, export = pageAndExport(t, a, room)
	if shownTurn(st, "FROM THE DESK") == nil || !strings.Contains(export, "FROM THE DESK") {
		t.Error("the turn did not return once the machine was certified again")
	}
	if got := promptOn(t, a, "a-session", "p2", "again"); !strings.Contains(got, "FROM THE DESK") {
		t.Errorf("the returned turn was not given to the session: %s", got)
	}
}

// A turn stored before the removal that cuts it off is shown marked, left out of the
// export's count, and never given to a session (D-220).
func TestAnEventPastACutoffNeverReachesAHookAndIsMarkedInTheView(t *testing.T) {
	a, b := twoMachines(t)
	room := sharedWith(t, a, b, "FROM THE LAPTOP")
	say1(t, b, room, "DESK WITHIN")
	say1(t, b, room, "DESK PAST")
	pullOn(t, a, b.addr)
	if err := a.d.members.BindSession("a-session", room.RoomID); err != nil {
		t.Fatal(err)
	}
	var res identityResult
	a.local(t, "/machines/remove", removeRequest{Machine: b.d.id.PeerID, Cutoffs: map[string]int64{room.RoomID: 1}}, &res)
	if res.Status != "removed" {
		t.Fatalf("removing: %+v", res)
	}

	got := promptOn(t, a, "a-session", "p1", "hello")
	if strings.Contains(got, "DESK PAST") || !strings.Contains(got, "DESK WITHIN") {
		t.Errorf("the session was given: %s", got)
	}
	st, export := pageAndExport(t, a, room)
	past, within := shownTurn(st, "DESK PAST"), shownTurn(st, "DESK WITHIN")
	if past == nil || past.Marked != "from a removed machine" || past.NotInSession {
		t.Errorf("the turn past the cutoff is shown as %+v", past)
	}
	if within == nil || within.Marked != "" {
		t.Errorf("the turn within the cutoff is shown as %+v", within)
	}
	// Four turns count: the founding, the laptop's, the one within and the prompt.
	if !strings.Contains(export, "from a removed machine; it does not count") || !strings.Contains(export, "4 turns.") {
		t.Errorf("the export does not mark the turn or counts it:\n%s", export)
	}
}

// The injected block carries the machine's name as a field of its own marked as the
// user's claim, and this user's own turn from another machine under the label "you",
// with the field this machine sets (D-233, D-232, D-149).
func TestTheInjectedBlockCarriesTheMachineNameAsItsOwnFieldAndAnOwnTurnUnderYou(t *testing.T) {
	a, b := twoMachines(t)
	room := sharedWith(t, a, b, "FROM THE LAPTOP")
	say1(t, b, room, "MY OWN FROM THE DESK")
	pullOn(t, a, b.addr)
	if err := a.d.members.BindSession("a-session", room.RoomID); err != nil {
		t.Fatal(err)
	}
	got := promptOn(t, a, "a-session", "p1", "hello")
	for _, want := range []string{"your own user's words from another session", "not addressed to this session", `"machine" names the machine`} {
		if !strings.Contains(got, want) {
			t.Errorf("the framing lacks %q", want)
		}
	}
	start, end := strings.Index(got, `{"turns"`), strings.LastIndex(got, "}\n</team-conversation")
	if start < 0 || end < start {
		t.Fatalf("no turns in %s", got)
	}
	var block struct {
		Turns []map[string]any `json:"turns"`
	}
	if err := json.Unmarshal([]byte(got[start:end+1]), &block); err != nil {
		t.Fatal(err)
	}
	var own map[string]any
	for _, tu := range block.Turns {
		if tu["text"] == "MY OWN FROM THE DESK" {
			own = tu
		}
	}
	if own == nil {
		t.Fatalf("the own turn from the desk was not given: %s", got)
	}
	m, _ := own["machine"].(map[string]any)
	if own["label"] != "you" || own["yourUser"] != true || m["name"] != "desk" || m["assertedByUser"] != true {
		t.Errorf("the own turn is %v", own)
	}
	if s, _ := own["speaker"].(string); strings.Contains(s, "desk") {
		t.Errorf("the machine's name is joined to the speaker: %q", s)
	}
	// A colleague's turn is not the user's own.
	block2 := FormatTeamContext([]Event{{PeerID: testIdentity(t).PeerID, EventType: EventUserPrompt, Content: "theirs", MachineID: "studio"}},
		receivingFacts{m: a.d.members, self: a.d.id.Peer()})
	if strings.Contains(block2, `"yourUser":true`) || !strings.Contains(block2, `"machine":{"name":"studio","assertedByUser":true}`) {
		t.Errorf("a colleague's turn is injected as %s", block2)
	}
	// In the view the turn is the user's own, carries its machine, and is marked as not yet
	// given to a session while none here holds it (D-198, D-204).
	st, _ := pageAndExport(t, a, room)
	if e := shownTurn(st, "MY OWN FROM THE DESK"); e == nil || !e.Mine || e.Machine != "desk" {
		t.Errorf("the view shows the own turn as %+v", e)
	}
	if e := shownTurn(st, "FROM THE LAPTOP"); e == nil || !e.Mine || e.NotInSession {
		t.Errorf("the view shows this machine's own turn as %+v", e)
	}
}

// One colleague with two machines in the room is one line, reachable while either can
// be reached, and unreachable only when neither can; the user's own machine that joined
// the room is listed by name, unreachable as it is. A second room without the colleague
// shows no line for them (D-194).
func TestAColleagueWithOneReachableMachineOfTwoIsOneReachableLine(t *testing.T) {
	a, b := twoMachines(t)
	room := sharedWith(t, a, b, "FIRST")
	other := foundAndSay(t, a, "SECOND")
	ben := newTestUser(t)
	k1, k2 := ben.machine(t, "desktop").PeerID, ben.machine(t, "laptop").PeerID
	m := a.d.members
	if err := m.Allow(ben.id, "ben"); err != nil {
		t.Fatal(err)
	}
	for k, addr := range map[string]string{k1: "tc://ben-one", k2: "tc://ben-two"} {
		if err := m.SetMachineEndpoint(k, ben.id, addr); err != nil {
			t.Fatal(err)
		}
		if err := m.RecordRoomMachine(room.RoomID, k, ben.id); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.RecordRoomMachine(room.RoomID, b.d.id.PeerID, a.d.id.User); err != nil {
		t.Fatal(err)
	}
	if err := m.CacheGuestList(&GuestList{Room: room.RoomID, Admitted: map[string]bool{a.d.id.User: true, ben.id: true}}); err != nil {
		t.Fatal(err)
	}
	set := func(addr string, online bool) {
		a.d.peerMu.Lock()
		defer a.d.peerMu.Unlock()
		st := a.d.peerStateLocked(addr)
		if online {
			st.lastSeen, st.downSince = time.Now(), time.Time{}
		} else {
			st.lastSeen, st.downSince, st.lastErr = time.Time{}, time.Now().Add(-time.Hour), "context deadline exceeded"
		}
	}
	set("tc://ben-one", true)
	set("tc://ben-two", false)
	set(b.addr, false)
	keepAddresses(t, a, b)

	lines := func(roomID string) (ben []uiPeer, own []uiPeer) {
		rec := httptest.NewRecorder()
		a.d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest("GET", "/peers?room="+roomID, nil))
		var body struct {
			Peers []uiPeer `json:"peers"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		for _, p := range body.Peers {
			switch {
			case p.Own:
				own = append(own, p)
			case p.PeerID != "":
				ben = append(ben, p)
			}
		}
		return ben, own
	}
	colleague, own := lines(room.RoomID)
	if len(colleague) != 1 || !colleague[0].Online || colleague[0].Machines != 2 || colleague[0].Name != "ben" {
		t.Fatalf("the colleague's lines are %+v", colleague)
	}
	if len(own) != 1 || own[0].Name != "desk" || own[0].Online || own[0].Trying == "" {
		t.Fatalf("the user's own machines are %+v", own)
	}
	rows := []guestRow{{Name: "you", ID: a.d.id.User, Self: true}, {Name: "ben", ID: ben.id}}
	var out bytes.Buffer
	renderGuests(&out, room.RoomName, rows, append(colleague, own...), nil, true)
	text := out.String()
	if strings.Count(text, "reachable") < 1 || strings.Contains(text, "    NOT REACHABLE") || !strings.Contains(text, "your machine desk: NOT REACHABLE") {
		t.Errorf("the room status reads:\n%s", text)
	}

	set("tc://ben-one", false)
	colleague, _ = lines(room.RoomID)
	if len(colleague) != 1 || colleague[0].Online || colleague[0].Trying == "" {
		t.Errorf("with neither machine reachable the colleague is %+v", colleague)
	}
	if colleague, own := lines(other.RoomID); len(colleague) != 0 || len(own) != 0 {
		t.Errorf("the second room shows %+v and %+v", colleague, own)
	}
	// self-status lists every machine of the user, the unreachable one among them.
	views := a.d.machineViews()
	if len(views) != 2 || views[1].Name != "desk" || views[1].Reach != "not reachable" {
		t.Errorf("self-status's machines are %+v", views)
	}
}

// expelledSetup is a user on a page-made machine invited to two rooms by a colleague,
// with a session in each, and revoked from the first.
func expelledSetup(t *testing.T) (a *machine, ben *offlinePeer, first, second Room) {
	t.Helper()
	bu := newTestUser(t)
	ben, rooms := newFounder(t, bu.machine(t, "desktop"), bu, 2)
	first, second = rooms[0], rooms[1]
	a = newMachine(t)
	createUser(t, a, otherPhrase(), "Dana", "laptop")
	user := a.d.id.User
	if err := ben.d.members.Allow(user, ""); err != nil {
		t.Fatal(err)
	}
	if err := ben.d.members.MarkVerified(user); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.Allow(bu.id, "ben"); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.MarkVerified(bu.id); err != nil {
		t.Fatal(err)
	}
	for i, r := range rooms {
		if err := ben.d.InviteUser(r.RoomID, user); err != nil {
			t.Fatal(err)
		}
		if err := a.d.members.RecordRoom(r.RoomID, r.RoomName, false); err != nil {
			t.Fatal(err)
		}
		if err := a.d.members.BindSession([]string{"in-first", "in-second"}[i], r.RoomID); err != nil {
			t.Fatal(err)
		}
	}
	pullOn(t, a, ben.addr)
	if err := ben.d.RevokeUser(first.RoomID, user); err != nil {
		t.Fatal(err)
	}
	pullOn(t, a, ben.addr)
	return a, ben, first, second
}

// A session in a room its user was removed from is told so at its next prompt, once,
// with who removed the user and that a new session can join another room; its turns go
// on being captured, the page and the room status mark the room, and a session in the
// user's other room is told nothing (D-241).
func TestAnExpelledSessionsNextPromptCarriesTheNoticeOnceAndItsTurnsAreStillCaptured(t *testing.T) {
	a, _, first, second := expelledSetup(t)

	got := promptOn(t, a, "in-first", "p1", "STILL HERE ONE")
	for _, want := range []string{"You were removed from this room by ben", "This session cannot join another room; a new session can"} {
		if !strings.Contains(got, want) {
			t.Errorf("the first prompt lacks %q: %s", want, got)
		}
	}
	if again := promptOn(t, a, "in-first", "p2", "STILL HERE TWO"); strings.Contains(again, "removed from this room") {
		t.Errorf("the notice was given twice: %s", again)
	}
	if other := promptOn(t, a, "in-second", "q1", "IN THE OTHER ROOM"); strings.Contains(other, "removed") {
		t.Errorf("a session in the other room was told: %s", other)
	}
	st, _ := pageAndExport(t, a, first)
	for _, say := range []string{"STILL HERE ONE", "STILL HERE TWO"} {
		if shownTurn(st, say) == nil {
			t.Errorf("the expelled session's turn %q was not captured", say)
		}
	}
	if st.Removed == nil || !st.Removed.Expelled || len(st.Removed.By) != 1 || st.Removed.By[0] != "ben" || st.Removed.Resuming {
		t.Errorf("the page marks the room as %+v", st.Removed)
	}
	if st2, _ := pageAndExport(t, a, second); st2.Removed != nil {
		t.Errorf("the page marks the other room as %+v", st2.Removed)
	}
	// The guest list shows who revoked whom.
	var me *uiGuest
	for i := range st.Guests {
		if st.Guests[i].Self {
			me = &st.Guests[i]
		}
	}
	if me == nil || me.State != "removed" || len(me.RevokedBy) != 1 || me.RevokedBy[0] != "ben" {
		t.Errorf("the guest list shows the user as %+v", me)
	}

	// The room status says it too, and while a guest's offer is pending, that the room is
	// coming back.
	s, ok := localRemoval(a.d.members, a.d.id, first)
	var out bytes.Buffer
	renderRemoval(&out, s, ok, false)
	if !strings.Contains(out.String(), "YOU WERE REMOVED FROM THIS ROOM") || !strings.Contains(out.String(), "ben") {
		t.Errorf("the room status says:\n%s", out.String())
	}
	if err := a.d.members.RecordResumption(first.RoomID, "x", nil); err != nil {
		t.Fatal(err)
	}
	if st, _ := pageAndExport(t, a, first); st.Removed == nil || !st.Removed.Resuming {
		t.Errorf("with an offer pending the page says %+v", st.Removed)
	}
	out.Reset()
	renderRemoval(&out, removalSummary{}, false, true)
	if !strings.Contains(out.String(), "refusing this machine, which usually means you were removed") {
		t.Errorf("a refused machine's room status says:\n%s", out.String())
	}
}

// A revoked guest's turn stored before a revocation with an earlier cutoff reached this
// machine is shown marked as from a removed guest, marked in the export and never given
// to a session (D-236).
func TestARevokedGuestsTurnPastTheCutoffIsMarkedAndNeverGiven(t *testing.T) {
	a := newMachine(t)
	createUser(t, a, fixedPhrase(), "Dana", "laptop")
	room := foundAndSay(t, a, "FROM THE LAPTOP")
	ben := newTestUser(t)
	bp := newUserPeer(t, room.RoomID, room.RoomName, ben.machine(t, "desktop"), ben)
	for _, pair := range []struct {
		m    *Membership
		peer string
	}{{a.d.members, ben.id}, {bp.d.members, a.d.id.User}} {
		if err := pair.m.Allow(pair.peer, "ben"); err != nil {
			t.Fatal(err)
		}
		if err := pair.m.MarkVerified(pair.peer); err != nil {
			t.Fatal(err)
		}
	}
	a.use(t)
	if err := a.d.InviteUser(room.RoomID, ben.id); err != nil {
		t.Fatal(err)
	}
	if _, err := bp.d.pullFrom(a.addr); err != nil {
		t.Fatal(err)
	}
	bp.say(t, room.RoomID, "b-session", EventUserPrompt, "BEN WITHIN")
	bp.say(t, room.RoomID, "b-session", EventUserPrompt, "BEN PAST")
	pullOn(t, a, bp.addr)
	if err := a.d.members.BindSession("a-session", room.RoomID); err != nil {
		t.Fatal(err)
	}
	// A revocation as a guest makes it whose machine had received only the first turn.
	store, _ := a.d.storeFor(room.RoomID)
	a.d.mu.Lock()
	_, _, err := a.d.appendGuestEvent(store, room, GuestRevoke, guestContent{User: ben.id,
		Cutoffs: map[string]int64{bp.d.id.PeerID: 1}})
	a.d.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}

	got := promptOn(t, a, "a-session", "p1", "hello")
	if strings.Contains(got, "BEN PAST") || !strings.Contains(got, "BEN WITHIN") {
		t.Errorf("the session was given: %s", got)
	}
	st, export := pageAndExport(t, a, room)
	if e := shownTurn(st, "BEN PAST"); e == nil || e.Marked != "from a removed guest" || e.NotInSession {
		t.Errorf("the turn past the revocation's cutoff is shown as %+v", e)
	}
	if e := shownTurn(st, "BEN WITHIN"); e == nil || e.Marked != "" {
		t.Errorf("the turn within the cutoff is shown as %+v", e)
	}
	if !strings.Contains(export, "from a removed guest; it does not count") {
		t.Errorf("the export does not mark the turn:\n%s", export)
	}
}

// A machine removes itself from its own page: the removal is signed and kept, the
// machine stops holding the shared key, and its later turns stop counting (D-220).
func TestAMachineRemovesItselfFromItsOwnPage(t *testing.T) {
	a, b := twoMachines(t)
	room := sharedWith(t, a, b, "FROM THE LAPTOP")
	say1(t, b, room, "DESK BEFORE")
	var res identityResult
	b.local(t, "/machines/remove", removeRequest{Machine: b.d.id.PeerID}, &res)
	if res.Status != "removed" {
		t.Fatalf("removing itself: %+v", res)
	}
	if b.d.heldSharedVersion() != 0 {
		t.Error("the machine that removed itself still holds the shared key")
	}
	st := b.d.members.UserState(b.d.id.User, nil)
	ms := st.Machines[b.d.id.PeerID]
	if ms == nil || !ms.Removed || ms.Cutoffs[room.RoomID] != 1 || st.Counts(b.d.id.PeerID, room.RoomID, 2) {
		t.Errorf("after removing itself the machine is %+v", ms)
	}
}
