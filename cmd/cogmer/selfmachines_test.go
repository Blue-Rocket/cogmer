package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// A user's own machines (D-229, D-230, D-231, D-228): the identity page that makes the
// user, adding a machine with its two words, recovery, renaming, and the user's lists
// copied between the user's machines.

// machine is one daemon of a test, with a state directory of its own and its peer
// routes served over TLS.
type machine struct {
	d    *Daemon
	home string
	addr string
}

// newMachine makes a daemon with no user identity, as a fresh install has.
func newMachine(t *testing.T) *machine {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	m, err := OpenMembership()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	id := testIdentity(t)
	d := &Daemon{id: id, members: m, stores: map[string]*Store{}, home: filepath.Join(home, stateDirName)}
	d.askRooms = func(_ string, rooms []Room) []Room { return rooms }
	t.Cleanup(d.closeStores)
	t.Cleanup(d.background.Wait)
	addr := servePeerTLSHandler(t, d, d.PeerRoutes())
	d.advertise = addr
	return &machine{d: d, home: home, addr: addr}
}

// use points the process's state directory at this machine, for the calls that read
// it, such as opening a room's database.
func (m *machine) use(t *testing.T) { t.Setenv("HOME", m.home) }

// local sends a guarded request to the machine's local routes, as its page does.
func (m *machine) local(t *testing.T, path string, body, out any) string {
	t.Helper()
	raw, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	r.Header.Set(localGuardHeader, localGuardValue)
	w := httptest.NewRecorder()
	m.d.LocalRoutes().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
	}
	if out != nil {
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			t.Fatalf("%s: %v in %s", path, err, w.Body.String())
		}
	}
	return w.Body.String()
}

// fixedPhrase is a phrase whose words are long enough that none of them turns up by
// chance inside base64 text.
func fixedPhrase() Phrase {
	var p Phrase
	for i := range p {
		p[i] = byte(0xA0 + i)
	}
	return p
}

// createUser makes the user on m through the page, with the phrase fixed.
func createUser(t *testing.T, m *machine, p Phrase, display, name string) []string {
	t.Helper()
	m.d.newPhrase = func() (Phrase, error) { return p, nil }
	var ph phraseResponse
	bodies := []string{m.local(t, "/identity/phrase", struct{}{}, &ph)}
	words := map[int]string{}
	for _, n := range ph.Confirm {
		words[n] = ph.Words[n-1]
	}
	var res identityResult
	bodies = append(bodies, m.local(t, "/identity/create", createRequest{CreateID: ph.CreateID, Words: words, DisplayName: display, MachineName: name}, &res))
	if res.Status != "created" {
		t.Fatalf("the user was not created: %+v", res)
	}
	return bodies
}

// addMachine adds b to a's user through both pages, and returns the words each page
// showed.
func addMachine(t *testing.T, a, b *machine, name string) (string, string) {
	t.Helper()
	addAwaitEvery = 20 * time.Millisecond
	var js joinStartResponse
	b.local(t, "/identity/join/start", struct{}{}, &js)
	key, _, _ := parsePairing(js.String)
	if key != b.d.id.PeerID {
		t.Fatalf("the new machine's string names %s, not its own key", key)
	}
	var as addStartResponse
	a.local(t, "/machines/add/start", addStartRequest{String: js.String}, &as)
	if as.Error != "" {
		t.Fatalf("adding: %s", as.Error)
	}
	var st joinStatus
	b.local(t, "/identity/join/status", struct{}{}, &st)
	if st.State != "compare" {
		t.Fatalf("the new machine's page is in state %q, want compare", st.State)
	}
	b.local(t, "/identity/join/confirm", joinConfirmRequest{Matched: true, MachineName: name}, nil)
	var ac addConfirmResponse
	a.local(t, "/machines/add/confirm", addConfirmRequest{AddID: as.AddID, Matched: true}, &ac)
	if ac.Status != "added" {
		t.Fatalf("the machine was not added: %+v", ac)
	}
	return as.Words, st.Words
}

// twoMachines is a user made on one machine through the page and a second machine
// added from it.
func twoMachines(t *testing.T) (*machine, *machine) {
	t.Helper()
	a, b := newMachine(t), newMachine(t)
	createUser(t, a, fixedPhrase(), "Dana", "laptop")
	addMachine(t, a, b, "desk")
	return a, b
}

// hasWord reports whether text holds word as a word of its own, so that a run of
// letters inside base64 is not taken for it.
func hasWord(text, word string) bool {
	return regexp.MustCompile(`(?i)(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(word) + `([^A-Za-z0-9_]|$)`).MatchString(text)
}

func TestCreatingAUserThroughThePageWritesTheIdentityAndThePhraseLeaksNowhere(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	m := newMachine(t)
	p := fixedPhrase()
	bodies := createUser(t, m, p, "Dana", "laptop")
	bodies = append(bodies, m.local(t, "/identity/state", struct{}{}, nil))

	if !m.d.id.HasUser() || m.d.id.User != p.UserID() {
		t.Fatalf("the machine does not speak for the phrase's user: %+v", m.d.id)
	}
	c := m.d.id.Chain
	if c.Shared.Version != 1 || c.Machine.Serial != 1 || c.Machine.Name != "laptop" || c.Machine.Kind != KindMachine {
		t.Errorf("the chain is %+v", c)
	}
	if _, err := c.Verify(p.UserID(), m.d.id.PeerID); err != nil {
		t.Errorf("the chain does not verify: %v", err)
	}
	if m.d.id.UserDisplayName != "Dana" || !m.d.id.NameChosen || m.d.id.MachineID != "laptop" {
		t.Errorf("the identity is %+v", m.d.id)
	}
	idFile, err := os.ReadFile(filepath.Join(m.d.home, "identity.json"))
	if err != nil {
		t.Fatal(err)
	}
	var onDisk Identity
	if err := json.Unmarshal(idFile, &onDisk); err != nil || onDisk.User != p.UserID() || onDisk.Chain == nil || !onDisk.NameChosen {
		t.Errorf("identity.json holds %s", idFile)
	}
	shared, version, ok, err := loadSharedKeyIn(m.d.home)
	if err != nil || !ok || version != 1 || PeerIDFromPublic(shared.Public().(ed25519.PublicKey)) != c.Shared.SharedKey {
		t.Errorf("shared.key holds version %d, ok %v, err %v", version, ok, err)
	}
	if bytes.Contains(idFile, []byte(base64.RawURLEncoding.EncodeToString(shared))) {
		t.Error("identity.json holds the shared key")
	}

	// The phrase is in the one response that shows it, and nowhere else.
	words := p.Words()
	if !hasWord(bodies[0], words[0]) {
		t.Fatalf("the phrase response does not show the words: %s", bodies[0])
	}
	for _, w := range words {
		for i, b := range bodies[1:] {
			if hasWord(b, w) {
				t.Errorf("response %d holds a word of the phrase: %s", i+1, b)
			}
		}
		if hasWord(string(idFile), w) {
			t.Errorf("identity.json holds a word of the phrase")
		}
		if hasWord(logs.String(), w) {
			t.Errorf("the daemon's log holds a word of the phrase:\n%s", logs.String())
		}
	}
	// The machine's own certificates are among the user's records (D-219).
	rs, _, err := m.d.members.IdentityRecords(p.UserID())
	if err != nil || len(rs.Shared) != 1 || len(rs.Machines) != 1 ||
		rs.Shared[0].Hash() != c.Shared.Hash() || rs.Machines[0].Hash() != c.Machine.Hash() {
		t.Errorf("the user's records hold %d shared and %d machine certificates, not this machine's own chain: %v",
			len(rs.Shared), len(rs.Machines), err)
	}
	m.d.flows.mu.Lock()
	left := len(m.d.flows.creating)
	m.d.flows.mu.Unlock()
	if left != 0 {
		t.Errorf("%d phrases are still held after the user was made", left)
	}
}

func TestAWrongConfirmationWordRefusesCreation(t *testing.T) {
	m := newMachine(t)
	p := fixedPhrase()
	m.d.newPhrase = func() (Phrase, error) { return p, nil }
	var ph phraseResponse
	m.local(t, "/identity/phrase", struct{}{}, &ph)
	words := map[int]string{}
	for _, n := range ph.Confirm {
		words[n] = ph.Words[n-1]
	}
	wrong := ph.Confirm[1]
	words[wrong] = ph.Words[(wrong)%phraseWords] // the next word along
	var res identityResult
	m.local(t, "/identity/create", createRequest{CreateID: ph.CreateID, Words: words, DisplayName: "Dana", MachineName: "laptop"}, &res)
	if res.Status == "created" || !strings.Contains(res.Error, "Word ") {
		t.Fatalf("a wrong word was accepted: %+v", res)
	}
	for _, w := range p.Words() {
		if hasWord(res.Error, w) {
			t.Errorf("the refusal quotes a word of the phrase: %q", res.Error)
		}
	}
	if m.d.id.HasUser() {
		t.Error("the machine speaks for a user after a refused confirmation")
	}
	if _, err := os.Stat(filepath.Join(m.d.home, "shared.key")); !os.IsNotExist(err) {
		t.Error("a shared key was written after a refused confirmation")
	}
}

func TestAddingAMachineCertifiesItAndGivesItTheSharedKeyAndTheUsersColleagues(t *testing.T) {
	a, b := newMachine(t), newMachine(t)
	createUser(t, a, fixedPhrase(), "Dana", "laptop")
	carol := testIdentity(t).PeerID
	if err := a.d.members.Allow(carol, "carol"); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.MarkVerified(carol); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.RecordColleague(a.d.id, carol); err != nil {
		t.Fatal(err)
	}

	wa, wb := addMachine(t, a, b, "desk")
	if wa == "" || wa != wb {
		t.Fatalf("the two pages showed %q and %q", wa, wb)
	}
	if !b.d.id.HasUser() || b.d.id.User != a.d.id.User {
		t.Fatalf("the new machine does not speak for the user: %+v", b.d.id)
	}
	if st := b.d.members.UserState(b.d.id.User, nil); st.Machines[b.d.id.PeerID] == nil || !st.Machines[b.d.id.PeerID].Current {
		t.Error("the new machine's own certificate is not among its records of the user")
	}
	if b.d.id.Chain.Machine.Name != "desk" || b.d.id.Chain.Machine.Serial != 2 || b.d.id.UserDisplayName != "Dana" {
		t.Errorf("the new machine's chain is %+v and its name %q", b.d.id.Chain.Machine, b.d.id.UserDisplayName)
	}
	sa, va, _, _ := loadSharedKeyIn(a.d.home)
	sb, vb, ok, _ := loadSharedKeyIn(b.d.home)
	if !ok || va != vb || !bytes.Equal(sa, sb) {
		t.Error("the new machine was not given the user's shared key")
	}
	if !b.d.members.IsVerified(carol) || b.d.members.Label(carol) != "carol" {
		t.Errorf("the new machine does not know the user's colleague: verified %v, label %q",
			b.d.members.IsVerified(carol), b.d.members.Label(carol))
	}
	certs, _ := a.d.members.displayCerts(a.d.id.User)
	if certs[b.d.id.PeerID].Name != "desk" {
		t.Error("the adding machine does not hold the new machine's certificate")
	}
}

func TestADifferentAnswerOnTheNewMachineAddsNothing(t *testing.T) {
	a, b := newMachine(t), newMachine(t)
	createUser(t, a, fixedPhrase(), "Dana", "laptop")
	addAwaitEvery = 20 * time.Millisecond
	var js joinStartResponse
	b.local(t, "/identity/join/start", struct{}{}, &js)
	var as addStartResponse
	a.local(t, "/machines/add/start", addStartRequest{String: pairingString(b.d.id.PeerID, b.addr, "")}, &as)
	b.local(t, "/identity/join/confirm", joinConfirmRequest{Matched: false}, nil)
	var ac addConfirmResponse
	a.local(t, "/machines/add/confirm", addConfirmRequest{AddID: as.AddID, Matched: true}, &ac)
	if ac.Status == "added" || ac.Error == "" {
		t.Fatalf("a machine whose page said the words differ was added: %+v", ac)
	}
	if b.d.id.HasUser() {
		t.Error("the new machine speaks for the user")
	}
	if _, ok := func() (MachineCert, bool) {
		c, _ := a.d.members.displayCerts(a.d.id.User)
		x, ok := c[b.d.id.PeerID]
		return x, ok
	}(); ok {
		t.Error("the new machine was certified")
	}
}

func TestTheMachineWordsAreFrozenAndDifferFromAColleaguePairingsWords(t *testing.T) {
	raw, err := os.ReadFile("testdata/identity_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		MachineSAS struct{ MachineA, NonceA, MachineB, NonceB, Words, CommitmentA string } `json:"machineSAS"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	f := v.MachineSAS
	na, _ := base64.RawURLEncoding.DecodeString(f.NonceA)
	nb, _ := base64.RawURLEncoding.DecodeString(f.NonceB)
	if got := SASMachine(f.MachineA, na, f.MachineB, nb); got != f.Words {
		t.Errorf("the machine words for a frozen exchange are %q, were %q; two versions would show the user different words", got, f.Words)
	}
	if got := SASMachine(f.MachineB, nb, f.MachineA, na); got != f.Words {
		t.Errorf("the machine words depend on which machine computes them: %q", got)
	}
	if got := base64.RawURLEncoding.EncodeToString(sasMachineCommitment(f.MachineA, na)); got != f.CommitmentA {
		t.Errorf("the frozen machine commitment changed to %s", got)
	}
	if colleague := SAS(f.MachineA, na, f.MachineB, nb); colleague == f.Words {
		t.Errorf("a colleague pairing and a machine-add derive the same words, %q, for the same keys and nonces", colleague)
	}
	if bytes.Equal(sasCommitment(f.MachineA, na), sasMachineCommitment(f.MachineA, na)) {
		t.Error("a colleague commitment opens as a machine-add commitment")
	}
}

func TestAColleagueVerifiedOnOneMachineIsVerifiedOnTheOtherAfterListExchange(t *testing.T) {
	a, b := twoMachines(t)
	carol := testIdentity(t).PeerID
	if err := a.d.members.Allow(carol, ""); err != nil {
		t.Fatal(err)
	}
	// The pairing's confirmation, as the page sends it.
	pid := a.d.newPairing(carol, "carol", true)
	a.local(t, "/verify/confirm", verifyConfirmRequest{PairID: pid, Matched: true}, nil)
	if b.d.members.IsVerified(carol) {
		t.Fatal("the second machine knew of the verification before any exchange")
	}
	a.d.exchangeSelfLists()
	if !b.d.members.IsVerified(carol) || b.d.members.Label(carol) != "carol" {
		t.Errorf("after the exchange the second machine has carol verified %v, labeled %q",
			b.d.members.IsVerified(carol), b.d.members.Label(carol))
	}

	// Forgetting on the second machine forgets on the first.
	if err := b.d.members.Forget(carol); err != nil {
		t.Fatal(err)
	}
	if err := b.d.members.RecordColleague(b.d.id, carol); err != nil {
		t.Fatal(err)
	}
	b.d.exchangeSelfLists()
	if a.d.members.Knows(carol) {
		t.Error("a colleague forgotten on one machine is still known on the other")
	}
}

func TestTwoConcurrentLabelChangesOnTwoMachinesSettleToTheSameValueOnBoth(t *testing.T) {
	a, b := twoMachines(t)
	carol := testIdentity(t).PeerID
	for _, m := range []*machine{a, b} {
		if err := m.d.members.Allow(carol, ""); err != nil {
			t.Fatal(err)
		}
	}
	// Both label her before either hears of the other's label.
	for m, label := range map[*machine]string{a: "carol", b: "caz"} {
		if err := m.d.members.Allow(carol, label); err != nil {
			t.Fatal(err)
		}
		if err := m.d.members.RecordColleague(m.d.id, carol); err != nil {
			t.Fatal(err)
		}
	}
	a.d.exchangeSelfLists()
	b.d.exchangeSelfLists()
	la, lb := a.d.members.Label(carol), b.d.members.Label(carol)
	if la != lb || (la != "carol" && la != "caz") {
		t.Errorf("the two machines settled on %q and %q", la, lb)
	}
}

// foundAndSay founds a room on m and writes one turn in it.
func foundAndSay(t *testing.T, m *machine, say string) Room {
	t.Helper()
	m.use(t)
	room, err := m.d.FoundRoom(false)
	if err != nil {
		t.Fatal(err)
	}
	say1(t, m, room, say)
	return room
}

func say1(t *testing.T, m *machine, room Room, say string) {
	t.Helper()
	m.use(t)
	store, err := m.d.storeFor(room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	m.d.mu.Lock()
	_, err = m.d.appendLocal(store, room, "s-"+m.d.id.PeerName, EventUserPrompt, say, nil)
	m.d.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
}

func holds(t *testing.T, m *machine, room Room, say string) bool {
	t.Helper()
	m.use(t)
	store, err := m.d.storeFor(room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	evs, err := store.ListRoom(room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range evs {
		if e.Content == say {
			return true
		}
	}
	return false
}

// joinByName is what `room-join <name>` does on a machine of the user that has not
// recorded the room.
func joinByName(t *testing.T, m *machine, name, session string) Room {
	t.Helper()
	m.use(t)
	found, err := joinFromList(m.d.members, m.d.id, name, false)
	if err != nil || !found {
		t.Fatalf("the room list did not hold %s: %v", name, err)
	}
	r, err := m.d.members.FindRoom(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.d.members.BindSession(session, r.RoomID); err != nil {
		t.Fatal(err)
	}
	if err := m.d.members.RecordRoomMachine(r.RoomID, m.d.id.PeerID, m.d.id.User); err != nil {
		t.Fatal(err)
	}
	if err := m.d.members.RecordRoomOnList(m.d.id, r.RoomID, r.RoomName, withMachine(m.d.id.PeerID)); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestASessionOnTheSecondMachineJoinsARoomByNameAndReceivesOnlyThatRoom(t *testing.T) {
	a, b := twoMachines(t)
	first := foundAndSay(t, a, "SAID IN THE FIRST ROOM")
	second := foundAndSay(t, a, "SAID IN THE SECOND ROOM")
	a.d.exchangeSelfLists()

	// The second machine holds both rooms on the user's list and neither room itself.
	for _, r := range []Room{first, second} {
		if _, v, ok := b.d.members.RoomOnList(b.d.id.User, r.RoomName); !ok || len(v.Machines) != 1 || v.Machines[0] != a.d.id.PeerID {
			t.Fatalf("the second machine's room list holds %s as %+v, %v", r.RoomName, v, ok)
		}
		if _, ok := b.d.members.RoomByID(r.RoomID); ok {
			t.Errorf("the second machine recorded %s before any session joined it", r.RoomName)
		}
	}

	joined := joinByName(t, b, first.RoomName, "b-session")
	if joined.RoomID != first.RoomID {
		t.Fatalf("joining %s by name joined %s", first.RoomName, joined.RoomID)
	}
	b.use(t)
	if _, err := b.d.pullFrom(a.addr); err != nil {
		t.Fatal(err)
	}
	if !holds(t, b, first, "SAID IN THE FIRST ROOM") {
		t.Error("the second machine did not receive the room it joined by name")
	}
	if _, ok := b.d.members.RoomByID(second.RoomID); ok {
		t.Error("the second machine recorded the room no session there joined")
	}
	st, err := b.d.snapshot(joined)
	if err != nil {
		t.Fatal(err)
	}
	var page strings.Builder
	for _, e := range st.Events {
		page.WriteString(e.Content)
	}
	if !strings.Contains(page.String(), "SAID IN THE FIRST ROOM") || strings.Contains(page.String(), "SECOND ROOM") {
		t.Errorf("the second machine's page of the room holds %q", page.String())
	}
}

func TestTheTwoMachinesTurnsReachEachOtherWithNoColleagueOnline(t *testing.T) {
	a, b := twoMachines(t)
	room := foundAndSay(t, a, "FROM THE LAPTOP")
	a.d.exchangeSelfLists()
	joinByName(t, b, room.RoomName, "b-session")
	b.use(t)
	if _, err := b.d.pullFrom(a.addr); err != nil {
		t.Fatal(err)
	}
	say1(t, b, room, "FROM THE DESK")
	a.use(t)
	if _, err := a.d.pullFrom(b.addr); err != nil {
		t.Fatal(err)
	}
	for _, m := range []*machine{a, b} {
		for _, say := range []string{"FROM THE LAPTOP", "FROM THE DESK"} {
			if !holds(t, m, room, say) {
				t.Errorf("%s does not hold %q", m.d.id.MachineID, say)
			}
		}
	}
	// The first machine knows the second joined the room, so the second is one of the
	// room's sync targets there (D-103, a room's sync targets are the machines from
	// which a session has joined it).
	found := false
	for _, k := range a.d.members.RoomMachines(room.RoomID) {
		found = found || k == b.d.id.PeerID
	}
	if !found {
		t.Error("the first machine does not record that the second joined the room")
	}
}

func TestRecoveryCertifiesANewVersionAndTheMachineNotAddedBackIsSuspended(t *testing.T) {
	a, b := twoMachines(t)
	room := foundAndSay(t, a, "BEFORE THE THEFT")
	a.d.exchangeSelfLists()
	joinByName(t, b, room.RoomName, "b-session")
	b.use(t)
	if _, err := b.d.pullFrom(a.addr); err != nil {
		t.Fatal(err)
	}

	var res identityResult
	a.local(t, "/identity/recover", recoverRequest{Phrase: strings.Join(fixedPhrase().Words(), " ")}, &res)
	if res.Status != "recovered" {
		t.Fatalf("recovery: %+v", res)
	}
	if a.d.id.Chain.Shared.Version != 2 || a.d.id.Chain.Machine.Name != "laptop" {
		t.Errorf("after recovery the chain is version %d, named %q", a.d.id.Chain.Shared.Version, a.d.id.Chain.Machine.Name)
	}
	if _, v, _, _ := loadSharedKeyIn(a.d.home); v != 2 {
		t.Errorf("shared.key holds version %d after recovery", v)
	}
	st := a.d.members.UserState(a.d.id.User, nil)
	if !st.Machines[a.d.id.PeerID].Current || st.Machines[a.d.id.PeerID].Cert.Hash() != a.d.id.Chain.Machine.Hash() {
		t.Error("the recovered machine's own new certificate is not among its records")
	}
	if st.Current != 2 || !st.Machines[a.d.id.PeerID].Current {
		t.Errorf("the recovered machine is not current: %+v", st.Machines[a.d.id.PeerID])
	}
	if ms := st.Machines[b.d.id.PeerID]; ms == nil || ms.Current || st.Counts(b.d.id.PeerID, room.RoomID, 1) {
		t.Errorf("the machine not added back is not suspended: %+v", ms)
	}

	say1(t, b, room, "AFTER THE THEFT")
	a.use(t)
	_, _ = a.d.pullFrom(b.addr)
	if holds(t, a, room, "AFTER THE THEFT") {
		t.Error("a turn of the suspended machine was stored")
	}
	if err := b.d.exchangeWith(a.d.id.PeerID, a.addr); err == nil {
		t.Error("the suspended machine still exchanges the user's lists")
	}
}

func TestRenamingAMachineChangesTheNameItsTurnsCarry(t *testing.T) {
	a, b := twoMachines(t)
	room := foundAndSay(t, a, "FIRST")
	a.d.exchangeSelfLists()
	joinByName(t, b, room.RoomName, "b-session")

	var res identityResult
	a.local(t, "/machines/rename", renameRequest{Machine: b.d.id.PeerID, Name: "studio"}, &res)
	if res.Status != "renamed" {
		t.Fatalf("rename: %+v", res)
	}
	a.d.exchangeSelfLists()
	if b.d.id.Chain.Machine.Name != "studio" || b.d.id.MachineID != "studio" {
		t.Fatalf("the renamed machine carries %q and %q", b.d.id.Chain.Machine.Name, b.d.id.MachineID)
	}
	say1(t, b, room, "AFTER THE RENAME")
	b.use(t)
	store, _ := b.d.storeFor(room.RoomID)
	evs, _ := store.ListRoom(room.RoomID)
	var got *Event
	for i := range evs {
		if evs[i].Content == "AFTER THE RENAME" {
			got = &evs[i]
		}
	}
	if got == nil || got.Chain == nil || got.Chain.Machine.Name != "studio" || got.MachineID != "studio" {
		t.Fatalf("the turn carries %+v", got)
	}
	if err := got.Verify(); err != nil {
		t.Errorf("the turn after the rename does not verify: %v", err)
	}
	var names []string
	for _, v := range a.d.machineViews() {
		names = append(names, v.Name)
	}
	if strings.Join(names, ",") != "laptop,studio" {
		t.Errorf("the first machine lists the machines as %v", names)
	}
}

func TestSelfStatusListsTheUsersMachinesAndWhetherEachIsReachable(t *testing.T) {
	a, b := twoMachines(t)
	a.d.exchangeSelfLists()
	views := a.d.machineViews()
	if len(views) != 2 || !views[0].Self || views[1].Name != "desk" || views[1].Reach != "reachable" {
		t.Fatalf("the machines are %+v", views)
	}
	var out bytes.Buffer
	t.Setenv("COGMER_ADDR", "127.0.0.1:1") // no daemon answers, so the list comes from the records
	writeOwnMachines(&out, a.d.members, a.d.id)
	for _, want := range []string{"Your machines:", "laptop", "this machine", "desk", "/cogmer:self-machines"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("self-status lacks %q:\n%s", want, out.String())
		}
	}
	_ = b
}

func TestSelfMachinesExistsAndOpensThePageAtItsMachines(t *testing.T) {
	src, err := os.ReadFile("../../plugin/commands/self-machines.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), `cli.sh" machines`) || !strings.Contains(string(src), "disable-model-invocation: true") {
		t.Errorf("self-machines.md does not run the machines subcommand out of the model's reach:\n%s", src)
	}
	if steps := typedCommands["self-machines"]; len(steps) != 1 || steps[0].sub != "machines" {
		t.Errorf("a Cowork session cannot type self-machines: %v", steps)
	}
	if u := identityPageURL("machines"); !strings.HasSuffix(u, "/identity#machines") {
		t.Errorf("the page is opened at %s", u)
	}
	m := newMachine(t)
	w := httptest.NewRecorder()
	m.d.LocalRoutes().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/identity", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `id="machines"`) {
		t.Errorf("the page has no machines section: %d", w.Code)
	}

	if testing.Short() {
		return
	}
	dir := t.TempDir()
	bin := buildVersion(t, dir, "self-machines-test")
	run := func(args ...string) (string, int) {
		cmd := exec.Command(bin, args...)
		cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + dir, "COGMER_HOME=" + filepath.Join(dir, "home"),
			"COGMER_TAILCAT=off", "COGMER_PREFLIGHT=off", "COGMER_ADDR=127.0.0.1:1", "CLAUDE_CODE_SESSION_ID=s"}
		out, err := cmd.CombinedOutput()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
		return string(out), code
	}
	// With no user identity, a command that needs one points to the page and does
	// nothing else (D-230).
	for _, args := range [][]string{{"machines"}, {"create"}, {"join", "misty-canyon"}} {
		out, code := run(args...)
		if code != 0 || !strings.Contains(flat(out), "needs your cogmer identity") || !strings.Contains(flat(out), "daemon is not running") {
			t.Errorf("%v: exit %d\n%s", args, code, out)
		}
	}
	if out, _ := run("rooms"); !strings.Contains(out, "no rooms") {
		t.Errorf("a command stopped at the identity page recorded a room:\n%s", out)
	}
}

// The local routes the page uses are guarded like every other route that writes or
// reads what only the user's page should (D-087).
func TestTheIdentityPagesRoutesRefuseARequestWithoutTheGuardHeader(t *testing.T) {
	m := newMachine(t)
	for _, path := range []string{"/identity/state", "/identity/phrase", "/identity/create", "/identity/recover",
		"/identity/join/start", "/identity/join/status", "/identity/join/confirm",
		"/machines/add/start", "/machines/add/confirm", "/machines/rename"} {
		w := httptest.NewRecorder()
		m.d.LocalRoutes().ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}")))
		if w.Code != http.StatusForbidden {
			t.Errorf("%s answered %d without the guard header", path, w.Code)
		}
	}
}
