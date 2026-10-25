package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// A removal of a cloud key names a cutoff for each of its session keys in each room
// (D-250), under a scheme of its own beside the removal of a machine (D-058).

// The frozen removal of a cloud key, for the reason vectors_test.go gives: a removal
// a colleague already holds was signed over these bytes.
func TestFrozenCloudRemovalStillVerifiesAndHashesAsItDid(t *testing.T) {
	k := testIDKeys(t)
	raw, err := os.ReadFile("testdata/identity_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Keys struct {
			Master, Shared, Cloud, Session string
		} `json:"keys"`
		CloudRemoval struct {
			Record Removal `json:"record"`
			Hash   string  `json:"hash"`
		} `json:"cloudRemoval"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	r := v.CloudRemoval.Record
	if r.Scheme != cloudRemovalScheme {
		t.Fatalf("the frozen removal of a cloud key is under scheme %d", r.Scheme)
	}
	if err := r.VerifyBy(k.shared.pub); err != nil {
		t.Errorf("the frozen removal of a cloud key no longer verifies: %v; the bytes scheme v2 signs have changed; add a scheme and leave v2 as it was (D-058, signature schemes are kept, never replaced)", err)
	}
	h := r.Hash()
	if got := hex.EncodeToString(h[:]); got != v.CloudRemoval.Hash {
		t.Errorf("the frozen removal of a cloud key hashes to %s, was %s; a hash is how machines name a record to one another", got, v.CloudRemoval.Hash)
	}
	again, err := SignCloudRemoval(k.shared.priv, LevelShared, 3, k.master.id, k.cloud.id,
		[]SessionCutoff{{"room-b", k.session.id, 2}, {"room-a", k.other.id, 6}, {"room-a", k.session.id, 4}})
	if err != nil || again.Hash() != h || again.Signature != r.Signature {
		t.Errorf("signing the frozen removal's cutoffs again gives another removal: %v", err)
	}

	// The encoding the design states, written out without the code's helpers.
	var b []byte
	str := func(s string) { b = append(binary.BigEndian.AppendUint32(b, uint32(len(s))), s...) }
	num := func(n uint64) { b = binary.BigEndian.AppendUint64(b, n) }
	str("peer-room/cloud-removal/v2")
	str(v.Keys.Master)
	str(v.Keys.Cloud)
	num(2)
	str(v.Keys.Shared)
	num(3)
	num(3)
	str("room-a")
	str(v.Keys.Session)
	num(4)
	str("room-a")
	str(k.other.id)
	num(6)
	str("room-b")
	str(v.Keys.Session)
	num(2)
	if sha256Hex(b) != v.CloudRemoval.Hash {
		t.Error("the frozen removal of a cloud key is not a hash of user, cloud key, signer and each room, session and sequence under its tag")
	}

	// The check above cannot fail if verification accepts anything.
	r.SessionCutoffs = append([]SessionCutoff{}, r.SessionCutoffs...)
	r.SessionCutoffs[0].Sequence++
	if r.VerifyBy(k.shared.pub) == nil {
		t.Error("an altered frozen removal of a cloud key verified")
	}
}

// Each scheme signs one kind of cutoff, so a cutoff of the other kind carried beside it
// is vouched for by nothing and refuses the removal.
func TestARemovalCarriesOnlyTheKindOfCutoffItsSchemeSigns(t *testing.T) {
	k := testIDKeys(t)
	machine := k.removal(t)
	machine.SessionCutoffs = []SessionCutoff{{"room-a", k.session.id, 9}}
	if err := machine.VerifyBy(k.shared.pub); err == nil || !strings.Contains(err.Error(), "does not sign") {
		t.Errorf("a removal of a machine carrying a session cutoff: %v", err)
	}
	cloud, err := SignCloudRemoval(k.shared.priv, LevelShared, 3, k.master.id, k.cloud.id,
		[]SessionCutoff{{"room-a", k.session.id, 4}})
	if err != nil {
		t.Fatal(err)
	}
	cloud.Cutoffs = []RoomCutoff{{"room-a", 9}}
	if err := cloud.VerifyBy(k.shared.pub); err == nil || !strings.Contains(err.Error(), "does not sign") {
		t.Errorf("a removal of a cloud key carrying a room cutoff: %v", err)
	}
	for name, cutoffs := range map[string][]SessionCutoff{
		"twice":         {{"room-a", k.session.id, 4}, {"room-a", k.session.id, 5}},
		"no room":       {{"", k.session.id, 4}},
		"not a key":     {{"room-a", "ed25519:nope", 4}},
		"the cloud key": {{"room-a", k.cloud.id, 4}},
		"negative":      {{"room-a", k.session.id, -1}},
	} {
		if _, err := SignCloudRemoval(k.shared.priv, LevelShared, 3, k.master.id, k.cloud.id, cutoffs); err == nil {
			t.Errorf("%s: a removal of a cloud key was signed", name)
		}
	}
	// Carried over to a new shared key, a removal keeps its scheme and its cutoffs.
	again, err := cloud.Again(k.other.priv, LevelShared, 4)
	if err != nil {
		t.Fatal(err)
	}
	cloud.Cutoffs = nil
	if again.Scheme != cloudRemovalScheme || len(again.SessionCutoffs) != 1 || again.SessionCutoffs[0] != cloud.SessionCutoffs[0] ||
		again.VerifyBy(k.other.pub) != nil {
		t.Errorf("a removal of a cloud key carried over as %+v", again)
	}
}

// cloudState is a user with a cloud key c, resolved with the removals given.
func cloudState(t *testing.T, removals ...func(u *stateUser, c stateKey) Removal) (*UserState, stateKey) {
	t.Helper()
	u := newStateUser("cloud-cutoff")
	c := stateKeyFrom("cloud-cutoff/cloud")
	rs := RecordSet{
		Shared:   []SharedKeyCert{u.share(1)},
		Machines: []MachineCert{u.certify(1, c, KindCloud, "web", 2)},
	}
	for _, f := range removals {
		rs.Removals = append(rs.Removals, f(u, c))
	}
	return Resolve(u.id(), rs), c
}

func mustRemoveCloud(t *testing.T, signer stateKey, level Level, version uint64, user, cloud string, cutoffs ...SessionCutoff) Removal {
	t.Helper()
	r, err := SignCloudRemoval(signer.priv, level, version, user, cloud, cutoffs)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

type countCase struct {
	session string
	room    string
	seq     int64
	want    bool
}

func checkCounts(t *testing.T, s *UserState, cloud string, cases []countCase) {
	t.Helper()
	for _, c := range cases {
		if got := s.Counts(cloud, c.session, c.room, c.seq); got != c.want {
			t.Errorf("Counts(%s, %s, %d) = %v, want %v", PeerName(c.session), c.room, c.seq, got, c.want)
		}
	}
}

func TestARemovedCloudKeysSessionsCountUpToTheirOwnCutoffsAndAnUnnamedSessionCountsNothing(t *testing.T) {
	s1, s2, s3 := stateKeyFrom("cloud-cutoff/s1").id, stateKeyFrom("cloud-cutoff/s2").id, stateKeyFrom("cloud-cutoff/s3").id
	st, c := cloudState(t, func(u *stateUser, c stateKey) Removal {
		return mustRemoveCloud(t, u.sharedKey(1), LevelShared, 1, u.id(), c.id,
			SessionCutoff{"a", s1, 3}, SessionCutoff{"a", s2, 2}, SessionCutoff{"b", s1, 1})
	})
	if !st.Machines[c.id].Removed {
		t.Fatal("the cloud key is not removed")
	}
	checkCounts(t, st, c.id, []countCase{
		{s1, "a", 1, true}, {s1, "a", 3, true}, {s1, "a", 4, false},
		{s2, "a", 2, true}, {s2, "a", 3, false},
		{s1, "b", 1, true}, {s1, "b", 2, false},
		{s2, "b", 1, false},
		{s3, "a", 1, false},
	})
	// A session's sequence is judged by its own cutoff, never by the room's.
	if st.Counts(c.id, c.id, "a", 1) {
		t.Error("an event signed by the cloud key itself counts")
	}
}

func TestTheLowestSessionCutoffAcrossRemovalsOfACloudKeyApplies(t *testing.T) {
	s1, s2 := stateKeyFrom("cloud-cutoff/s1").id, stateKeyFrom("cloud-cutoff/s2").id
	st, c := cloudState(t,
		func(u *stateUser, c stateKey) Removal {
			return mustRemoveCloud(t, u.master, LevelMaster, 0, u.id(), c.id,
				SessionCutoff{"a", s1, 5}, SessionCutoff{"a", s2, 3})
		},
		func(u *stateUser, c stateKey) Removal {
			return mustRemoveCloud(t, u.sharedKey(1), LevelShared, 1, u.id(), c.id,
				SessionCutoff{"a", s1, 4}, SessionCutoff{"a", s2, 7}, SessionCutoff{"b", s1, 6})
		},
	)
	checkCounts(t, st, c.id, []countCase{
		{s1, "a", 4, true}, {s1, "a", 5, false},
		{s2, "a", 3, true}, {s2, "a", 4, false},
		// The master's removal leaves s1 out of b, which counts as 0.
		{s1, "b", 1, false},
	})
}

// A removal of a cloud key under the machine scheme names no session, so every
// session's turns are past their cutoff, which is what such a removal always meant,
// since the cloud key signs no turn of its own.
func TestAMachineSchemeRemovalOfACloudKeyMarksEverySessionsTurns(t *testing.T) {
	s1 := stateKeyFrom("cloud-cutoff/s1").id
	old := func(u *stateUser, c stateKey) Removal {
		return mustRemove(t, u.sharedKey(1), LevelShared, 1, u.id(), c.id, RoomCutoff{"a", 5})
	}
	st, c := cloudState(t, old)
	if !st.Machines[c.id].Removed {
		t.Fatal("the cloud key is not removed")
	}
	checkCounts(t, st, c.id, []countCase{{s1, "a", 1, false}, {s1, "b", 1, false}})

	st, c = cloudState(t, old, func(u *stateUser, c stateKey) Removal {
		return mustRemoveCloud(t, u.master, LevelMaster, 0, u.id(), c.id, SessionCutoff{"a", s1, 5})
	})
	checkCounts(t, st, c.id, []countCase{{s1, "a", 1, false}})
}

// --- the page, and a colleague ---

// cloudTurns is a cloud environment of a's user, and turns of its sessions in a room
// that a and a colleague's machine both hold.
type cloudTurns struct {
	a     *machine
	bp    *offlinePeer
	room  Room
	user  string
	cloud string
	keys  []ed25519.PrivateKey
	chain []Chain
	// events[i] is session i's turns, in order. The last session's turns are held by
	// the colleague alone.
	events [][]Event
}

func (f *cloudTurns) session(i int) string {
	return PeerIDFromPublic(f.keys[i].Public().(ed25519.PublicKey))
}

func newCloudTurns(t *testing.T, turns ...int) *cloudTurns {
	t.Helper()
	a := newMachine(t)
	createUser(t, a, fixedPhrase(), "Dana", "laptop")
	setup, err := parseCloudValue(addCloudEnv(t, a, "web"))
	if err != nil {
		t.Fatal(err)
	}
	room := foundAndSay(t, a, "FROM THE LAPTOP")
	f := &cloudTurns{a: a, room: room, user: a.d.id.User, cloud: setup.cloud.Machine}
	ben := newTestUser(t)
	f.bp = newUserPeer(t, room.RoomID, room.RoomName, ben.machine(t, "desktop"), ben)
	for _, pair := range []struct {
		m    *Membership
		peer string
	}{{a.d.members, ben.id}, {f.bp.d.members, f.user}} {
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

	start := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)
	for i, n := range turns {
		pub, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatal(err)
		}
		cert := CertifySession(setup.key, f.user, pub, time.Now().Add(24*time.Hour))
		chain := Chain{Shared: setup.shared, Machine: setup.cloud, Session: &cert}
		f.keys, f.chain = append(f.keys, priv), append(f.chain, chain)
		var evs []Event
		for seq := 1; seq <= n; seq++ {
			ev := Event{EventID: fmt.Sprintf("cloud-%d-%d", i, seq), PeerID: f.user, PeerSequence: int64(seq),
				RoomID: room.RoomID, Timestamp: start.Add(time.Duration(i)*time.Hour + time.Duration(seq)*time.Minute).Format(time.RFC3339),
				MachineID: "web", OriginSessionID: fmt.Sprintf("agent-session-%d", i), EventType: EventUserPrompt,
				Content: fmt.Sprintf("SESSION %d TURN %d", i, seq)}
			ev.SignV4(priv, chain, nil)
			evs = append(evs, ev)
			holders := []*Daemon{f.bp.d}
			if i < len(turns)-1 {
				holders = append(holders, a.d)
			}
			for _, d := range holders {
				store, err := d.storeFor(room.RoomID)
				if err != nil {
					t.Fatal(err)
				}
				e := ev
				d.mu.Lock()
				err = d.judgeEvent(&e)
				if err == nil {
					_, err = store.Insert(&e)
				}
				d.mu.Unlock()
				if err != nil {
					t.Fatalf("storing session %d's turn %d: %v", i, seq, err)
				}
			}
		}
		f.events = append(f.events, evs)
	}
	return f
}

// remove removes the cloud environment on a's page, as stolen or not, and pushes the
// removal to the colleague, as the identity loop does (D-225).
func (f *cloudTurns) remove(t *testing.T, stolen bool) {
	t.Helper()
	var res identityResult
	f.a.local(t, "/machines/remove", removeRequest{Machine: f.cloud, Stolen: stolen}, &res)
	if res.Status != "removed" {
		t.Fatalf("removing the cloud environment: %+v", res)
	}
	if err := f.a.d.members.SetMachineEndpoint(f.bp.d.id.PeerID, f.bp.d.id.User, f.bp.addr); err != nil {
		t.Fatal(err)
	}
	if err := f.a.d.members.StoreChain(*f.bp.d.id.Chain); err != nil {
		t.Fatal(err)
	}
	if n := f.a.d.pushIdentityRecords(time.Now()); n == 0 {
		t.Fatal("the removal was acknowledged by nobody")
	}
}

// standing is how the colleague's view and the colleague's assistant take one turn:
// whether it counts in the view, and whether a prompt is given it.
func (f *cloudTurns) standing(ev Event) (shown standing, given bool) {
	shown = f.bp.d.standingsFor(f.room.RoomID).of(ev)
	given = len(f.bp.d.onlyIdentityKnown([]Event{ev}, f.room.RoomID)) == 1
	return shown, given
}

func TestTheRemovalPageProposesEachSessionsLastHeldTurnByRoomAndWhenItStarted(t *testing.T) {
	f := newCloudTurns(t, 3, 2, 1)
	var p removeProposeResponse
	f.a.local(t, "/machines/remove/propose", removeProposeRequest{Machine: f.cloud}, &p)
	if !p.Cloud || p.Name != "web" || len(p.Rooms) != 0 {
		t.Fatalf("the proposal for a cloud environment is %+v", p)
	}
	got := map[string]removalSession{}
	for _, s := range p.Sessions {
		got[s.Session] = s
	}
	if len(got) != 2 {
		t.Fatalf("the page proposes %d sessions, want the two whose turns this machine holds: %+v", len(got), p.Sessions)
	}
	for i, want := range []struct {
		last    int64
		started string
	}{{3, "2026-10-25T09:01:00Z"}, {2, "2026-10-25T10:01:00Z"}} {
		s := got[f.session(i)]
		if s.Room != f.room.RoomID || s.RoomName != f.room.RoomName || s.Last != want.last || s.Proposed != want.last ||
			s.Started != want.started || s.Name != PeerName(f.session(i)) {
			t.Errorf("session %d is proposed as %+v", i, s)
		}
	}
	f.a.local(t, "/machines/remove/propose", removeProposeRequest{Machine: f.cloud, Stolen: true}, &p)
	for _, s := range p.Sessions {
		if s.Proposed != 0 || s.Last == 0 {
			t.Errorf("stolen, session %s proposes %d with last %d", s.Name, s.Proposed, s.Last)
		}
	}
	var res identityResult
	f.a.local(t, "/machines/remove", removeRequest{Machine: f.cloud,
		SessionCutoffs: map[string]map[string]int64{f.room.RoomID: {f.session(0): 4}}}, &res)
	if res.Error == "" {
		t.Error("a session cutoff past the last held turn was taken")
	}
}

func TestARetiredCloudEnvironmentsSessionTurnsAllStillCountAtAColleague(t *testing.T) {
	f := newCloudTurns(t, 3, 2, 1)
	f.remove(t, false)
	st := f.a.d.members.UserState(f.user, nil)
	for _, r := range st.Removals {
		if r.Scheme != cloudRemovalScheme || len(r.SessionCutoffs) != 2 {
			t.Fatalf("the page signed %+v", r)
		}
	}
	for i, evs := range f.events {
		for _, ev := range evs {
			shown, given := f.standing(ev)
			// The last session's turns never reached the removing machine, so no cutoff
			// names it and its turns are marked (D-250's limits).
			want := i < len(f.events)-1
			if (shown == standingCounts) != want || given != want {
				t.Errorf("at the colleague, %q shows as %d and is given %v; want counting %v", ev.Content, shown, given, want)
			}
			if !want && shown != standingRemovedMachine {
				t.Errorf("at the colleague, %q is not marked as a removed machine's", ev.Content)
			}
		}
	}
	// Every session of the cloud key is refused a live act once the removal is held.
	if _, err := f.bp.d.members.checkLiveChain(f.user, f.session(0), f.chain[0], time.Now()); err == nil || !strings.Contains(err.Error(), "removed") {
		t.Errorf("the colleague accepts a live act of a removed cloud key's session: %v", err)
	}
	// A next turn of a session named by the removal is past its cutoff.
	next := Event{EventID: "cloud-0-4", PeerID: f.user, PeerSequence: 4, RoomID: f.room.RoomID,
		Timestamp: "2026-10-25T11:00:00Z", OriginSessionID: "agent-session-0", EventType: EventUserPrompt, Content: "AFTER"}
	next.SignV4(f.keys[0], f.chain[0], nil)
	f.bp.d.mu.Lock()
	err := f.bp.d.judgeEvent(&next)
	f.bp.d.mu.Unlock()
	if !errors.Is(err, errNotCounting) {
		t.Errorf("the colleague judges a session's turn past its cutoff %v", err)
	}
}

func TestAStolenCloudEnvironmentsSessionTurnsAreAllMarkedAtAColleague(t *testing.T) {
	f := newCloudTurns(t, 3, 2)
	f.remove(t, true)
	for _, evs := range f.events {
		for _, ev := range evs {
			if shown, given := f.standing(ev); shown != standingRemovedMachine || given {
				t.Errorf("at the colleague, %q shows as %d and is given %v", ev.Content, shown, given)
			}
		}
	}
}

// A removal the earlier scheme signed for a cloud key still reads: it names no
// session, so it marks every session's turns, wherever it came from.
func TestAMachineSchemeRemovalOfACloudEnvironmentStillMarksAllAtAColleague(t *testing.T) {
	f := newCloudTurns(t, 2)
	shared, version, held, err := loadSharedKeyIn(f.a.d.homeDirectory())
	if err != nil || !held {
		t.Fatalf("the laptop holds no shared key: %v", err)
	}
	old, err := SignRemoval(shared, LevelShared, version, f.user, f.cloud, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(old)
	if _, _, err := f.a.d.members.PutIdentityRecord(RecordRemoval, f.user, raw); err != nil {
		t.Fatal(err)
	}
	if err := f.a.d.members.SetMachineEndpoint(f.bp.d.id.PeerID, f.bp.d.id.User, f.bp.addr); err != nil {
		t.Fatal(err)
	}
	if err := f.a.d.members.StoreChain(*f.bp.d.id.Chain); err != nil {
		t.Fatal(err)
	}
	if n := f.a.d.pushIdentityRecords(time.Now()); n == 0 {
		t.Fatal("the removal was acknowledged by nobody")
	}
	for _, ev := range f.events[0] {
		if shown, given := f.standing(ev); shown != standingRemovedMachine || given {
			t.Errorf("at the colleague, %q shows as %d and is given %v", ev.Content, shown, given)
		}
	}
	_ = sha256.Sum256
}
