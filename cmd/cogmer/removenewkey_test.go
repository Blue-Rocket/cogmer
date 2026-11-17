package main

import (
	"strings"
	"testing"
)

// Removing a machine that someone other than the user could still use certifies a new
// shared key in the same act (D-251, removing a machine that someone other than the user
// could still use always certifies a new shared key).

func TestRemovingAMachineSomeoneElseCouldUseCertifiesANewKeyInTheSameAct(t *testing.T) {
	a, b := twoMachines(t)
	room := sharedWith(t, a, b, "FROM THE LAPTOP")
	say1(t, b, room, "FROM THE DESK BEFORE")
	pullOn(t, a, b.addr)
	if err := a.d.members.BindSession("a-session", room.RoomID); err != nil {
		t.Fatal(err)
	}
	phrase := strings.Join(fixedPhrase().Words(), " ")

	var nk newKeyResult
	a.local(t, "/machines/newkey", newKeyRequest{Phrase: phrase,
		Machines: []string{b.d.id.PeerID},
		Remove:   []newKeyRemoval{{Machine: b.d.id.PeerID}}}, &nk)
	if nk.Status != "certified" || nk.Removed != 1 || nk.Version != 2 {
		t.Fatalf("certifying and removing: %+v", nk)
	}

	st := a.d.members.UserState(a.d.id.User, nil)
	ms := st.Machines[b.d.id.PeerID]
	if st.Current != 2 || !ms.Removed || !ms.Current {
		t.Errorf("after the act the key is %d and the machine is removed %v, certified again %v", st.Current, ms.Removed, ms.Current)
	}
	if st.Counts(b.d.id.PeerID, b.d.id.PeerID, room.RoomID, 1) {
		t.Error("a turn of a machine someone else could use counts, though the page proposed that none do")
	}
	if got := promptOn(t, a, "a-session", "p1", "hello"); strings.Contains(got, "FROM THE DESK BEFORE") {
		t.Errorf("the removed machine's turn was given to a session: %s", got)
	}
	// the removal is the master's, so a thief who signs with the old shared key cannot undo it
	for _, r := range func() []Removal { rs, _, _ := a.d.members.IdentityRecords(a.d.id.User); return rs.Removals }() {
		if r.Machine == b.d.id.PeerID && r.SignerLevel != LevelMaster {
			t.Errorf("the removal was signed at level %d, not by the master", r.SignerLevel)
		}
	}
}

func TestANewKeyRemovalNamesTheTurnsTheUserConfirmed(t *testing.T) {
	a, b := twoMachines(t)
	room := sharedWith(t, a, b, "FROM THE LAPTOP")
	say1(t, b, room, "THE DESK, KNOWN TO BE THE USER'S")
	say1(t, b, room, "THE DESK, AFTER IT WAS TAKEN")
	pullOn(t, a, b.addr)

	var prop removeProposeResponse
	a.local(t, "/machines/remove/propose", removeProposeRequest{Machine: b.d.id.PeerID, Stolen: true}, &prop)
	if len(prop.Rooms) != 1 || prop.Rooms[0].Proposed != 0 || prop.Rooms[0].Last < 2 {
		t.Fatalf("the page proposes %+v for a machine someone else could use", prop.Rooms)
	}
	var nk newKeyResult
	a.local(t, "/machines/newkey", newKeyRequest{Phrase: strings.Join(fixedPhrase().Words(), " "),
		Remove: []newKeyRemoval{{Machine: b.d.id.PeerID, Cutoffs: map[string]int64{prop.Rooms[0].Room: 1}}}}, &nk)
	if nk.Status != "certified" || nk.Removed != 1 {
		t.Fatalf("certifying: %+v", nk)
	}
	st := a.d.members.UserState(a.d.id.User, nil)
	if !st.Counts(b.d.id.PeerID, b.d.id.PeerID, room.RoomID, 1) || st.Counts(b.d.id.PeerID, b.d.id.PeerID, room.RoomID, 2) {
		t.Error("the cutoff the user named does not split the machine's turns")
	}
}

func TestACombinedRemovalIsRefusedWhenItCannotBeCarriedOut(t *testing.T) {
	a, b := twoMachines(t)
	room := sharedWith(t, a, b, "FROM THE LAPTOP")
	say1(t, b, room, "FROM THE DESK")
	pullOn(t, a, b.addr)
	phrase := strings.Join(fixedPhrase().Words(), " ")
	// a cloud environment is removed on its own
	var added cloudAddResponse
	a.local(t, "/machines/cloud/add", cloudAddRequest{Name: "env"}, &added)
	var st0 newKeyPreview
	a.local(t, "/machines/newkey/preview", struct{}{}, &st0)
	var cloudKey string
	for _, m := range st0.Machines {
		if m.Name == "env" {
			cloudKey = m.Machine
		}
	}
	if cloudKey == "" {
		t.Fatal("the cloud environment is not among the machines")
	}
	for name, remove := range map[string][]newKeyRemoval{
		"this machine":                {{Machine: a.d.id.PeerID}},
		"a stranger":                  {{Machine: testIdentity(t).PeerID}},
		"a cloud environment":         {{Machine: cloudKey}},
		"a machine named twice":       {{Machine: b.d.id.PeerID}, {Machine: b.d.id.PeerID}},
		"a cutoff past the last turn": {{Machine: b.d.id.PeerID, Cutoffs: map[string]int64{room.RoomID: 99}}},
	} {
		var nk newKeyResult
		a.local(t, "/machines/newkey", newKeyRequest{Phrase: phrase, Remove: remove}, &nk)
		if nk.Error == "" || nk.Status != "" {
			t.Errorf("%s: answered %+v", name, nk)
		}
		if v := a.d.members.UserState(a.d.id.User, nil).Current; v != 1 {
			t.Fatalf("%s: a refused request still certified key %d", name, v)
		}
	}
}
