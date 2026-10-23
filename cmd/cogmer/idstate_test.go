package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// Keys here come from a name, so each test sees the same keys and the same
// hashes on every run, and which of two records has the lower hash is computed
// rather than assumed.

type stateKey struct {
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
	id   string
}

func stateKeyFrom(name string) stateKey {
	seed := sha256.Sum256([]byte("idstate-test/" + name))
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := priv.Public().(ed25519.PublicKey)
	return stateKey{priv, pub, PeerIDFromPublic(pub)}
}

type stateUser struct {
	name   string
	master stateKey
	shared map[uint64]stateKey
}

func newStateUser(name string) *stateUser {
	return &stateUser{name: name, master: stateKeyFrom(name + "/master"), shared: map[uint64]stateKey{}}
}

func (u *stateUser) id() string { return u.master.id }

// sharedKey is the user's shared key of version v, made on first use.
func (u *stateUser) sharedKey(v uint64) stateKey {
	k, ok := u.shared[v]
	if !ok {
		k = stateKeyFrom(fmt.Sprintf("%s/shared/%d", u.name, v))
		u.shared[v] = k
	}
	return k
}

func (u *stateUser) share(v uint64) SharedKeyCert {
	return CertifySharedKey(u.master.priv, v, u.sharedKey(v).pub)
}

func (u *stateUser) certify(v uint64, m stateKey, kind, name string, serial uint64) MachineCert {
	return CertifyMachine(u.sharedKey(v).priv, u.id(), v, m.pub, kind, name, serial)
}

func mustRemove(t *testing.T, signer stateKey, level Level, version uint64, user, machine string, cutoffs ...RoomCutoff) Removal {
	t.Helper()
	r, err := SignRemoval(signer.priv, level, version, user, machine, cutoffs)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func hashLess(a, b [32]byte) bool { return bytes.Compare(a[:], b[:]) < 0 }

func discardedFor(s *UserState, h [32]byte) (Discarded, bool) {
	for _, d := range s.Discarded {
		if d.Hash == h {
			return d, true
		}
	}
	return Discarded{}, false
}

func wantDiscarded(t *testing.T, s *UserState, h [32]byte, kind, reason string) {
	t.Helper()
	d, ok := discardedFor(s, h)
	if !ok {
		t.Fatalf("%s record %x is not discarded; discarded: %+v", kind, h[:4], s.Discarded)
	}
	if d.Kind != kind || !strings.Contains(d.Reason, reason) {
		t.Fatalf("discarded as %q %q, want %q containing %q", d.Kind, d.Reason, kind, reason)
	}
}

func TestPrevailsOrdersByLevelThenRemovalThenLowestHash(t *testing.T) {
	lo, hi := [32]byte{1}, [32]byte{2}
	var claims []Claim
	for _, l := range []Level{LevelMachine, LevelShared, LevelMaster} {
		for _, rm := range []bool{false, true} {
			for _, h := range [][32]byte{lo, hi} {
				claims = append(claims, Claim{l, rm, h})
			}
		}
	}
	// rank is D-224 written out: a lower rank wins.
	rank := func(c Claim) int {
		r := int(LevelMaster-c.Level) * 4
		if !c.Removal {
			r += 2
		}
		if c.Hash == hi {
			r++
		}
		return r
	}
	for _, a := range claims {
		for _, b := range claims {
			want := rank(a) < rank(b)
			if got := Prevails(a, b); got != want {
				t.Errorf("Prevails(%+v, %+v) = %v, want %v", a, b, got, want)
			}
		}
	}

	named := []struct {
		what string
		a, b Claim
	}{
		{"a master addition over a shared removal", Claim{LevelMaster, false, hi}, Claim{LevelShared, true, lo}},
		{"a shared addition over a machine removal", Claim{LevelShared, false, hi}, Claim{LevelMachine, true, lo}},
		{"a removal over an addition at one level, whatever the hash", Claim{LevelShared, true, hi}, Claim{LevelShared, false, lo}},
		{"the lower hash at one level and kind", Claim{LevelMaster, false, lo}, Claim{LevelMaster, false, hi}},
	}
	for _, n := range named {
		if !Prevails(n.a, n.b) || Prevails(n.b, n.a) {
			t.Errorf("want %s", n.what)
		}
	}
	same := Claim{LevelShared, true, lo}
	if Prevails(same, same) {
		t.Error("a claim prevails over an identical one")
	}
}

func TestTwoSharedKeysForOneVersionGoToTheLowestHash(t *testing.T) {
	u := newStateUser("rival")
	a := u.share(1)
	keyA := u.sharedKey(1)
	keyB := stateKeyFrom("rival/shared/other")
	b := CertifySharedKey(u.master.priv, 1, keyB.pub)

	win, lose, winKey, loseKey := a, b, keyA, keyB
	if hashLess(b.Hash(), a.Hash()) {
		win, lose, winKey, loseKey = b, a, keyB, keyA
	}
	m1, m2 := stateKeyFrom("rival/m1"), stateKeyFrom("rival/m2")
	byWinner := CertifyMachine(winKey.priv, u.id(), 1, m1.pub, KindMachine, "by winner", 1)
	byLoser := CertifyMachine(loseKey.priv, u.id(), 1, m2.pub, KindMachine, "by loser", 2)

	s := Resolve(u.id(), RecordSet{Shared: []SharedKeyCert{lose, win}, Machines: []MachineCert{byLoser, byWinner}})
	if s.SharedKeys[1] != win.SharedKey {
		t.Fatalf("version 1 went to %.24s, want the lower hash %.24s", s.SharedKeys[1], win.SharedKey)
	}
	wantDiscarded(t, s, lose.Hash(), "shared", "D-224")
	if _, ok := s.Machines[m1.id]; !ok {
		t.Error("the machine certified by the winning key is missing")
	}
	if _, ok := s.Machines[m2.id]; ok {
		t.Error("a machine certified by the losing key was kept")
	}
	wantDiscarded(t, s, byLoser.Hash(), "machine", "not signed by the winning shared key")
}

func TestTheCurrentVersionIsTheHighestWithAWinner(t *testing.T) {
	u := newStateUser("versions")
	if s := Resolve(u.id(), RecordSet{}); s.Current != 0 {
		t.Fatalf("Current with no records = %d, want 0", s.Current)
	}
	thief := stateKeyFrom("versions/thief")
	forged := CertifySharedKey(thief.priv, 9, stateKeyFrom("versions/forged").pub)
	forged.User = u.id()
	s := Resolve(u.id(), RecordSet{Shared: []SharedKeyCert{u.share(1), u.share(5), u.share(3), forged}})
	if s.Current != 5 {
		t.Fatalf("Current = %d, want 5", s.Current)
	}
	if len(s.SharedKeys) != 3 {
		t.Fatalf("SharedKeys = %v, want versions 1, 3 and 5", s.SharedKeys)
	}
	wantDiscarded(t, s, forged.Hash(), "shared", "signature does not match")
}

func TestAMachineRecertifiedUnderAHigherVersionTakesThatCert(t *testing.T) {
	u := newStateUser("recert")
	m1, m2 := stateKeyFrom("recert/m1"), stateKeyFrom("recert/m2")
	rs := RecordSet{Shared: []SharedKeyCert{u.share(1), u.share(2)}}
	old := u.certify(1, m1, KindMachine, "laptop", 1)
	rs.Machines = []MachineCert{u.certify(2, m1, KindMachine, "laptop", 3), old, u.certify(1, m2, KindMachine, "desktop", 2)}

	s := Resolve(u.id(), rs)
	if got := s.Machines[m1.id]; got.Cert.SharedVersion != 2 || !got.Current {
		t.Fatalf("recertified machine = %+v, want its version 2 cert and current", got)
	}
	if got := s.Machines[m2.id]; got.Current {
		t.Fatal("a machine left on version 1 is current under version 2")
	}
	wantDiscarded(t, s, old.Hash(), "machine", "superseded")
}

func TestTwoCertsForOneMachineUnderOneVersionGoToTheLowestHash(t *testing.T) {
	u := newStateUser("twin")
	m := stateKeyFrom("twin/m")
	a := u.certify(1, m, KindMachine, "one name", 4)
	b := u.certify(1, m, KindMachine, "another name", 5)
	win, lose := a, b
	if hashLess(b.Hash(), a.Hash()) {
		win, lose = b, a
	}
	s := Resolve(u.id(), RecordSet{Shared: []SharedKeyCert{u.share(1)}, Machines: []MachineCert{lose, win}})
	if got := s.Machines[m.id].Cert; got != win {
		t.Fatalf("machine took %q, want the lower hash %q", got.Name, win.Name)
	}
	wantDiscarded(t, s, lose.Hash(), "machine", "superseded")
}

// The design's reading of D-224 (a conflict between identity records goes to
// the higher level): a removal and the certificate it removes are not in
// conflict, so a machine-level self-removal ends a shared-level certificate.
func TestARemovalDoesNotConflictWithTheCertificateItRemovesSoASelfRemovalTakesEffect(t *testing.T) {
	u := newStateUser("self")
	m := stateKeyFrom("self/m")
	r := mustRemove(t, m, LevelMachine, 0, u.id(), m.id, RoomCutoff{"room", 3})
	s := Resolve(u.id(), RecordSet{
		Shared:   []SharedKeyCert{u.share(1)},
		Machines: []MachineCert{u.certify(1, m, KindMachine, "laptop", 1)},
		Removals: []Removal{r},
	})
	if !s.Machines[m.id].Removed {
		t.Fatal("a machine that removed itself is not removed")
	}
	if _, ok := s.Removals[r.Hash()]; !ok {
		t.Fatal("the self-removal is not among the valid removals")
	}
}

func TestAMachineMayNotRemoveAnotherMachine(t *testing.T) {
	u := newStateUser("other")
	m1, m2 := stateKeyFrom("other/m1"), stateKeyFrom("other/m2")
	r := mustRemove(t, m1, LevelMachine, 0, u.id(), m2.id)
	s := Resolve(u.id(), RecordSet{
		Shared:   []SharedKeyCert{u.share(1)},
		Machines: []MachineCert{u.certify(1, m1, KindMachine, "a", 1), u.certify(1, m2, KindMachine, "b", 2)},
		Removals: []Removal{r},
	})
	if s.Machines[m2.id].Removed {
		t.Fatal("one machine removed another")
	}
	wantDiscarded(t, s, r.Hash(), "removal", "only itself")
}

func TestASessionKeyMayRemoveNothing(t *testing.T) {
	u := newStateUser("session")
	cloud, session := stateKeyFrom("session/cloud"), stateKeyFrom("session/session")
	itself := mustRemove(t, session, LevelMachine, 0, u.id(), session.id)
	itsCloud := mustRemove(t, session, LevelMachine, 0, u.id(), cloud.id)
	s := Resolve(u.id(), RecordSet{
		Shared:   []SharedKeyCert{u.share(1)},
		Machines: []MachineCert{u.certify(1, cloud, KindCloud, "cloud", 1)},
		Removals: []Removal{itself, itsCloud},
	})
	if len(s.Removals) != 0 || s.Machines[cloud.id].Removed {
		t.Fatalf("a session key's removal took effect: %v", s.Removals)
	}
	wantDiscarded(t, s, itself.Hash(), "removal", "no valid machine certificate")
	wantDiscarded(t, s, itsCloud.Hash(), "removal", "only itself")
}

func TestACloudKeyRemovesItself(t *testing.T) {
	u := newStateUser("cloud")
	cloud := stateKeyFrom("cloud/cloud")
	s := Resolve(u.id(), RecordSet{
		Shared:   []SharedKeyCert{u.share(1)},
		Machines: []MachineCert{u.certify(1, cloud, KindCloud, "cloud", 1)},
		Removals: []Removal{mustRemove(t, cloud, LevelMachine, 0, u.id(), cloud.id)},
	})
	if !s.Machines[cloud.id].Removed {
		t.Fatal("a cloud key that removed itself is not removed")
	}
}

func TestTheMasterRemovesAnyMachine(t *testing.T) {
	u := newStateUser("master")
	m := stateKeyFrom("master/m")
	s := Resolve(u.id(), RecordSet{
		Shared:   []SharedKeyCert{u.share(1)},
		Machines: []MachineCert{u.certify(1, m, KindMachine, "laptop", 1)},
		Removals: []Removal{mustRemove(t, u.master, LevelMaster, 0, u.id(), m.id)},
	})
	if !s.Machines[m.id].Removed {
		t.Fatal("the master's removal did not take effect")
	}

	// A key that is not the user's master cannot claim the master level.
	thief := stateKeyFrom("master/thief")
	forged := mustRemove(t, thief, LevelMaster, 0, u.id(), m.id)
	s = Resolve(u.id(), RecordSet{
		Shared:   []SharedKeyCert{u.share(1)},
		Machines: []MachineCert{u.certify(1, m, KindMachine, "laptop", 1)},
		Removals: []Removal{forged},
	})
	if s.Machines[m.id].Removed {
		t.Fatal("a removal claiming the master level from another key took effect")
	}
	wantDiscarded(t, s, forged.Hash(), "removal", "master key")
}

func TestASharedKeyOfAnOlderWinningVersionRemoves(t *testing.T) {
	u := newStateUser("older")
	m := stateKeyFrom("older/m")
	rs := RecordSet{Shared: []SharedKeyCert{u.share(1), u.share(2)}}
	rs.Machines = []MachineCert{u.certify(2, m, KindMachine, "laptop", 1)}
	rs.Removals = []Removal{mustRemove(t, u.sharedKey(1), LevelShared, 1, u.id(), m.id)}
	s := Resolve(u.id(), rs)
	if s.Current != 2 || !s.Machines[m.id].Removed {
		t.Fatal("a removal by the winning shared key of an older version did not take effect")
	}
}

func TestARemovalSignedByALosingSharedKeyIsRefused(t *testing.T) {
	u := newStateUser("loser")
	a := u.share(1)
	keyB := stateKeyFrom("loser/shared/other")
	b := CertifySharedKey(u.master.priv, 1, keyB.pub)
	winKey, loseKey := u.sharedKey(1), keyB
	if hashLess(b.Hash(), a.Hash()) {
		winKey, loseKey = keyB, u.sharedKey(1)
	}
	m := stateKeyFrom("loser/m")
	r := mustRemove(t, loseKey, LevelShared, 1, u.id(), m.id)
	s := Resolve(u.id(), RecordSet{
		Shared:   []SharedKeyCert{a, b},
		Machines: []MachineCert{CertifyMachine(winKey.priv, u.id(), 1, m.pub, KindMachine, "laptop", 1)},
		Removals: []Removal{r},
	})
	if s.Machines[m.id].Removed {
		t.Fatal("a removal signed by a losing shared key took effect")
	}
	wantDiscarded(t, s, r.Hash(), "removal", "winning shared key of version 1")
}

func TestARemovalForAnotherUserIsRefused(t *testing.T) {
	u, v := newStateUser("me"), newStateUser("them")
	m := stateKeyFrom("me/m")
	r := mustRemove(t, v.master, LevelMaster, 0, v.id(), m.id)
	s := Resolve(u.id(), RecordSet{
		Shared:   []SharedKeyCert{u.share(1)},
		Machines: []MachineCert{u.certify(1, m, KindMachine, "laptop", 1)},
		Removals: []Removal{r},
	})
	if s.Machines[m.id].Removed {
		t.Fatal("another user's removal took effect")
	}
	wantDiscarded(t, s, r.Hash(), "removal", "another user")
}

func TestRecordsForAnotherUserAreDiscardedAndDuplicatesCountOnce(t *testing.T) {
	u, v := newStateUser("mine"), newStateUser("theirs")
	m := stateKeyFrom("mine/m")
	theirShared := v.share(1)
	theirMachine := v.certify(1, m, KindMachine, "theirs", 1)
	mine := u.certify(1, m, KindMachine, "laptop", 1)
	s := Resolve(u.id(), RecordSet{
		Shared:   []SharedKeyCert{u.share(1), u.share(1), theirShared},
		Machines: []MachineCert{mine, mine, theirMachine},
	})
	if len(s.Discarded) != 2 {
		t.Fatalf("discarded %+v, want only the other user's two records", s.Discarded)
	}
	wantDiscarded(t, s, theirShared.Hash(), "shared", "another user")
	wantDiscarded(t, s, theirMachine.Hash(), "machine", "another user")
	if s.Machines[m.id].Cert != mine {
		t.Fatal("the user's own certificate is not the machine's")
	}
}

func TestAMachineCertWithNoSharedKeyOfItsVersionIsDiscarded(t *testing.T) {
	u := newStateUser("early")
	u.share(2)
	m := stateKeyFrom("early/m")
	c := u.certify(2, m, KindMachine, "laptop", 1)
	s := Resolve(u.id(), RecordSet{Machines: []MachineCert{c}})
	wantDiscarded(t, s, c.Hash(), "machine", "no valid shared-key certificate for version 2")
}

func TestARemovalUnderAnUnknownSchemeNeverEntersRemovals(t *testing.T) {
	u := newStateUser("scheme")
	m := stateKeyFrom("scheme/m")
	r := mustRemove(t, u.master, LevelMaster, 0, u.id(), m.id)
	r.Scheme = 99
	s := Resolve(u.id(), RecordSet{
		Shared:   []SharedKeyCert{u.share(1)},
		Machines: []MachineCert{u.certify(1, m, KindMachine, "laptop", 1)},
		Removals: []Removal{r},
	})
	if _, ok := s.Removals[[32]byte{}]; ok || len(s.Removals) != 0 {
		t.Fatal("a removal under an unknown scheme entered Removals")
	}
	if s.Machines[m.id].Removed {
		t.Fatal("a removal under an unknown scheme took effect")
	}
	wantDiscarded(t, s, [32]byte{}, "removal", "upgrade")
	if !s.Lacks(1, [][32]byte{{}}) {
		t.Fatal("the zero hash reads as a removal this state holds")
	}
}

func removedState(t *testing.T, removals ...func(u *stateUser, m stateKey) Removal) (*UserState, stateKey) {
	t.Helper()
	u := newStateUser("cutoff")
	m := stateKeyFrom("cutoff/m")
	rs := RecordSet{
		Shared:   []SharedKeyCert{u.share(1)},
		Machines: []MachineCert{u.certify(1, m, KindMachine, "laptop", 1)},
	}
	for _, f := range removals {
		rs.Removals = append(rs.Removals, f(u, m))
	}
	return Resolve(u.id(), rs), m
}

func TestTheLowestCutoffAcrossRemovalsAppliesInEachRoom(t *testing.T) {
	s, m := removedState(t,
		func(u *stateUser, m stateKey) Removal {
			return mustRemove(t, u.master, LevelMaster, 0, u.id(), m.id, RoomCutoff{"a", 5}, RoomCutoff{"b", 9})
		},
		func(u *stateUser, m stateKey) Removal {
			return mustRemove(t, u.sharedKey(1), LevelShared, 1, u.id(), m.id, RoomCutoff{"a", 7}, RoomCutoff{"b", 8}, RoomCutoff{"c", 4})
		},
	)
	// c is absent from the master's removal, which therefore gives it 0.
	want := map[string]int64{"a": 5, "b": 8}
	if got := s.Machines[m.id].Cutoffs; !reflect.DeepEqual(got, want) {
		t.Fatalf("cutoffs = %v, want %v", got, want)
	}
	for _, c := range []struct {
		room string
		seq  int64
		want bool
	}{
		{"a", 4, true}, {"a", 5, true}, {"a", 6, false},
		{"b", 8, true}, {"b", 9, false},
		{"c", 1, false}, {"elsewhere", 1, false},
	} {
		if got := s.Counts(m.id, c.room, c.seq); got != c.want {
			t.Errorf("Counts(%s, %d) = %v, want %v", c.room, c.seq, got, c.want)
		}
	}
	if s.Counts(stateKeyFrom("cutoff/unknown").id, "a", 1) {
		t.Error("an unknown machine's event counts")
	}
}

func TestAnUnremovedMachineCountsEverywhere(t *testing.T) {
	s, m := removedState(t)
	if !s.Counts(m.id, "any", 1_000_000) {
		t.Fatal("an unremoved machine's event does not count")
	}
}

func TestACutoffCannotMoveLater(t *testing.T) {
	early := func(u *stateUser, m stateKey) Removal {
		return mustRemove(t, u.master, LevelMaster, 0, u.id(), m.id, RoomCutoff{"a", 5})
	}
	late := func(u *stateUser, m stateKey) Removal {
		return mustRemove(t, u.master, LevelMaster, 0, u.id(), m.id, RoomCutoff{"a", 9})
	}
	for _, order := range [][]func(*stateUser, stateKey) Removal{{early, late}, {late, early}} {
		s, m := removedState(t, order...)
		if got := s.Machines[m.id].Cutoffs["a"]; got != 5 {
			t.Fatalf("cutoff = %d, want 5 whichever removal came first", got)
		}
	}
}

func TestARemovalArrivingBeforeItsMachinesCertificateWaitsForIt(t *testing.T) {
	u := newStateUser("before")
	m := stateKeyFrom("before/m")
	r := mustRemove(t, u.master, LevelMaster, 0, u.id(), m.id, RoomCutoff{"a", 2})
	cert := u.certify(1, m, KindMachine, "laptop", 1)
	rs := RecordSet{Shared: []SharedKeyCert{u.share(1)}, Removals: []Removal{r}}

	s := Resolve(u.id(), rs)
	if _, ok := s.Removals[r.Hash()]; !ok {
		t.Fatal("a valid removal of a machine not yet certified was dropped")
	}
	if _, err := s.Check(Chain{Shared: u.share(1), Machine: cert}, m.id); err == nil {
		t.Fatal("the chain of a machine with a removal held was accepted")
	}

	rs.Machines = []MachineCert{cert}
	s = Resolve(u.id(), rs)
	if !s.Machines[m.id].Removed || s.Machines[m.id].Cutoffs["a"] != 2 {
		t.Fatalf("machine = %+v once its certificate arrived, want removed with cutoff 2", s.Machines[m.id])
	}
}

func TestListGivesTheMachinesNotRemovedBySerialThenKey(t *testing.T) {
	u := newStateUser("list")
	a, b, c, gone := stateKeyFrom("list/a"), stateKeyFrom("list/b"), stateKeyFrom("list/c"), stateKeyFrom("list/gone")
	ca := u.certify(1, a, KindMachine, "a", 2)
	cb := u.certify(1, b, KindMachine, "b", 1)
	cc := u.certify(1, c, KindCloud, "c", 2)
	s := Resolve(u.id(), RecordSet{
		Shared:   []SharedKeyCert{u.share(1)},
		Machines: []MachineCert{ca, cc, u.certify(1, gone, KindMachine, "gone", 0), cb},
		Removals: []Removal{mustRemove(t, u.master, LevelMaster, 0, u.id(), gone.id)},
	})
	want := []MachineCert{cb, ca, cc}
	if a.id > c.id {
		want = []MachineCert{cb, cc, ca}
	}
	if got := s.List(); !reflect.DeepEqual(got, want) {
		var names []string
		for _, g := range got {
			names = append(names, g.Name)
		}
		t.Fatalf("List = %v", names)
	}
}

func TestCheckRefusesWhatTheStateKnowsBetter(t *testing.T) {
	u := newStateUser("check")
	m, unheard, gone := stateKeyFrom("check/m"), stateKeyFrom("check/unheard"), stateKeyFrom("check/gone")
	cloud, session := stateKeyFrom("check/cloud"), stateKeyFrom("check/session")
	shared := u.share(1)
	rival := stateKeyFrom("check/shared/rival")
	rivalCert := CertifySharedKey(u.master.priv, 1, rival.pub)
	winner, loser, winKey, loseKey := shared, rivalCert, u.sharedKey(1), rival
	if hashLess(rivalCert.Hash(), shared.Hash()) {
		winner, loser, winKey, loseKey = rivalCert, shared, rival, u.sharedKey(1)
	}
	cm := CertifyMachine(winKey.priv, u.id(), 1, m.pub, KindMachine, "laptop", 1)
	cg := CertifyMachine(winKey.priv, u.id(), 1, gone.pub, KindMachine, "gone", 2)
	cc := CertifyMachine(winKey.priv, u.id(), 1, cloud.pub, KindCloud, "cloud", 3)
	s := Resolve(u.id(), RecordSet{
		Shared:   []SharedKeyCert{shared, rivalCert},
		Machines: []MachineCert{cm, cg, cc},
		Removals: []Removal{mustRemove(t, u.master, LevelMaster, 0, u.id(), gone.id)},
	})
	sess := CertifySession(cloud.priv, u.id(), session.pub, time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))

	if link, err := s.Check(Chain{Shared: winner, Machine: cm}, m.id); err != nil || link.Machine != m.id {
		t.Fatalf("a good chain: %v", err)
	}
	if _, err := s.Check(Chain{Shared: winner, Machine: cc, Session: &sess}, session.id); err != nil {
		t.Fatalf("a good session chain: %v", err)
	}
	cu := CertifyMachine(winKey.priv, u.id(), 1, unheard.pub, KindMachine, "new", 4)
	if _, err := s.Check(Chain{Shared: winner, Machine: cu}, unheard.id); err != nil {
		t.Fatalf("a chain for a machine not heard of yet: %v", err)
	}
	later := u.share(7)
	cl := u.certify(7, unheard, KindMachine, "new", 4)
	if _, err := s.Check(Chain{Shared: later, Machine: cl}, unheard.id); err != nil {
		t.Fatalf("a chain under a version not held yet: %v", err)
	}
	if _, err := s.Check(Chain{Shared: winner, Machine: cg}, gone.id); err == nil || !strings.Contains(err.Error(), "removed") {
		t.Fatalf("a removed machine's chain: %v", err)
	}
	cx := CertifyMachine(loseKey.priv, u.id(), 1, m.pub, KindMachine, "laptop", 1)
	if _, err := s.Check(Chain{Shared: loser, Machine: cx}, m.id); err == nil || !strings.Contains(err.Error(), "lost") {
		t.Fatalf("a chain whose shared-key certificate lost: %v", err)
	}
	if _, err := s.Check(Chain{Shared: winner, Machine: cm}, unheard.id); err == nil {
		t.Fatal("a chain that does not end at the signer was accepted")
	}
}

func TestLacksNamesAHigherVersionOrAnUnknownRemoval(t *testing.T) {
	u := newStateUser("lacks")
	m := stateKeyFrom("lacks/m")
	r := mustRemove(t, u.master, LevelMaster, 0, u.id(), m.id)
	s := Resolve(u.id(), RecordSet{Shared: []SharedKeyCert{u.share(1), u.share(2)}, Removals: []Removal{r}})
	other := mustRemove(t, u.master, LevelMaster, 0, u.id(), stateKeyFrom("lacks/n").id)
	for _, c := range []struct {
		version  uint64
		removals [][32]byte
		want     bool
	}{
		{2, nil, false},
		{1, [][32]byte{r.Hash()}, false},
		{3, nil, true},
		{2, [][32]byte{r.Hash(), other.Hash()}, true},
	} {
		if got := s.Lacks(c.version, c.removals); got != c.want {
			t.Errorf("Lacks(%d, %d removals) = %v, want %v", c.version, len(c.removals), got, c.want)
		}
	}
}

func TestResolveDoesNotDependOnTheOrderOfRecords(t *testing.T) {
	u := newStateUser("order")
	m1, m2, m3, cloud := stateKeyFrom("order/m1"), stateKeyFrom("order/m2"), stateKeyFrom("order/m3"), stateKeyFrom("order/cloud")
	rivalKey := stateKeyFrom("order/rival")
	rs := RecordSet{
		Shared: []SharedKeyCert{u.share(1), u.share(2), u.share(1),
			CertifySharedKey(u.master.priv, 2, rivalKey.pub), newStateUser("order/x").share(1)},
		Machines: []MachineCert{
			u.certify(1, m1, KindMachine, "m1", 1), u.certify(2, m1, KindMachine, "m1", 4),
			u.certify(2, m2, KindMachine, "m2", 2), u.certify(2, m2, KindMachine, "m2 again", 2),
			u.certify(1, m3, KindMachine, "m3", 3),
			CertifyMachine(rivalKey.priv, u.id(), 2, cloud.pub, KindCloud, "cloud", 5),
			u.certify(2, cloud, KindCloud, "cloud", 5),
		},
		Removals: []Removal{
			mustRemove(t, u.master, LevelMaster, 0, u.id(), m2.id, RoomCutoff{"a", 3}, RoomCutoff{"b", 2}),
			mustRemove(t, u.sharedKey(1), LevelShared, 1, u.id(), m2.id, RoomCutoff{"a", 6}),
			mustRemove(t, m3, LevelMachine, 0, u.id(), m3.id, RoomCutoff{"a", 1}),
			mustRemove(t, m1, LevelMachine, 0, u.id(), m3.id),
			mustRemove(t, u.master, LevelMaster, 0, u.id(), stateKeyFrom("order/later").id),
		},
	}
	odd := rs.Removals[0]
	odd.Scheme = 7
	rs.Removals = append(rs.Removals, odd)

	want := Resolve(u.id(), rs)
	if len(want.Discarded) < 5 || len(want.Removals) != 4 {
		t.Fatalf("the set does not exercise what it should: %d discarded, %d removals", len(want.Discarded), len(want.Removals))
	}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		var shuffled RecordSet
		shuffled.Shared = append([]SharedKeyCert{}, rs.Shared...)
		shuffled.Machines = append([]MachineCert{}, rs.Machines...)
		shuffled.Removals = append([]Removal{}, rs.Removals...)
		rng.Shuffle(len(shuffled.Shared), func(i, j int) { shuffled.Shared[i], shuffled.Shared[j] = shuffled.Shared[j], shuffled.Shared[i] })
		rng.Shuffle(len(shuffled.Machines), func(i, j int) {
			shuffled.Machines[i], shuffled.Machines[j] = shuffled.Machines[j], shuffled.Machines[i]
		})
		rng.Shuffle(len(shuffled.Removals), func(i, j int) {
			shuffled.Removals[i], shuffled.Removals[j] = shuffled.Removals[j], shuffled.Removals[i]
		})
		if got := Resolve(u.id(), shuffled); !reflect.DeepEqual(got, want) {
			t.Fatalf("order %d resolved differently:\n got %+v\nwant %+v", i, got, want)
		}
	}
	if !sort.SliceIsSorted(want.Discarded, func(i, j int) bool {
		return want.Discarded[i].Kind < want.Discarded[j].Kind
	}) {
		t.Fatal("Discarded is not sorted by kind")
	}
}
