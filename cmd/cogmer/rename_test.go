package main

import (
	"strings"
	"testing"
)

// Renaming a colleague changes the name the user gives them and nothing else (D-257, a
// user renames a colleague with a command that changes the label and nothing else).

func TestRenamingAColleagueChangesTheLabelOnEveryMachineAndNothingElse(t *testing.T) {
	a, b := twoMachines(t)
	carol := testIdentity(t).PeerID
	if err := a.d.members.Allow(carol, ""); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.MarkVerified(carol); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.RecordColleague(a.d.id, carol); err != nil {
		t.Fatal(err)
	}
	a.d.exchangeSelfLists()
	verifiedAt := a.d.members.VerifiedAt(carol)

	// found by the derived name the user was given before naming anybody
	out, err := renamePeer(a.d.members, a.d.id, PeerName(carol), "carol")
	if err != nil || !strings.Contains(out, "renamed "+PeerName(carol)+" to carol") {
		t.Fatalf("renaming: %q, %v", out, err)
	}
	if a.d.members.Label(carol) != "carol" || !a.d.members.IsVerified(carol) || a.d.members.VerifiedAt(carol) != verifiedAt {
		t.Errorf("after renaming, label %q, verified %v at %q (was %q)", a.d.members.Label(carol),
			a.d.members.IsVerified(carol), a.d.members.VerifiedAt(carol), verifiedAt)
	}
	a.d.exchangeSelfLists()
	b.d.exchangeSelfLists()
	if b.d.members.Label(carol) != "carol" || !b.d.members.IsVerified(carol) {
		t.Errorf("the user's other machine has label %q, verified %v", b.d.members.Label(carol), b.d.members.IsVerified(carol))
	}

	// found by the name just given, and given another
	out, err = renamePeer(a.d.members, a.d.id, "carol", "Carol Ames")
	if err != nil || !strings.Contains(out, "renamed carol to Carol Ames") {
		t.Fatalf("renaming by the new name: %q, %v", out, err)
	}
	a.d.exchangeSelfLists()
	b.d.exchangeSelfLists()
	if b.d.members.Label(carol) != "Carol Ames" {
		t.Errorf("the second rename did not reach the other machine: %q", b.d.members.Label(carol))
	}
}

// Two keys under one name is how a substituted key surfaces, so a name another
// colleague has is refused and the colleague keeps the name they had.
func TestRenamingToANameAnotherColleagueHasIsRefused(t *testing.T) {
	a, _ := twoMachines(t)
	carol, dave := testIdentity(t).PeerID, testIdentity(t).PeerID
	for peer, name := range map[string]string{carol: "carol", dave: "dave"} {
		if err := a.d.members.Allow(peer, name); err != nil {
			t.Fatal(err)
		}
	}
	_, err := renamePeer(a.d.members, a.d.id, "carol", "dave")
	if err == nil || !strings.Contains(err.Error(), `already call a different peer "dave"`) {
		t.Fatalf("renaming onto another colleague's name: %v", err)
	}
	if a.d.members.Label(carol) != "carol" || a.d.members.Label(dave) != "dave" {
		t.Errorf("a refused rename changed a label: %q and %q", a.d.members.Label(carol), a.d.members.Label(dave))
	}
}

func TestRenamingNobodyKnownOrToNothingSaysSo(t *testing.T) {
	a, _ := twoMachines(t)
	carol := testIdentity(t).PeerID
	if err := a.d.members.Allow(carol, "carol"); err != nil {
		t.Fatal(err)
	}
	stranger := testIdentity(t).PeerID
	for _, who := range []string{"nobody", stranger} {
		if _, err := renamePeer(a.d.members, a.d.id, who, "x"); err == nil || !strings.Contains(err.Error(), "no peer known") {
			t.Errorf("renaming %q gave %v", who, err)
		}
	}
	if a.d.members.Knows(stranger) {
		t.Error("renaming an unknown key recorded it")
	}
	for _, name := range []string{"", "  ", "#@"} {
		if _, err := renamePeer(a.d.members, a.d.id, "carol", name); err == nil || !strings.Contains(err.Error(), "usage") {
			t.Errorf("renaming to %q gave %v", name, err)
		}
	}
	if a.d.members.Label(carol) != "carol" {
		t.Errorf("a refused rename changed the label to %q", a.d.members.Label(carol))
	}
	if out, err := renamePeer(a.d.members, a.d.id, "carol", "carol"); err != nil || !strings.Contains(out, "already called carol") {
		t.Errorf("renaming to the same name: %q, %v", out, err)
	}
}

// A name arrives from the keyboard, so it is cut to what a label holds, and the user
// is told.
func TestARenameThatCannotBeKeptWholeSaysWhatWasKept(t *testing.T) {
	a, _ := twoMachines(t)
	carol := testIdentity(t).PeerID
	if err := a.d.members.Allow(carol, "carol"); err != nil {
		t.Fatal(err)
	}
	out, err := renamePeer(a.d.members, a.d.id, "carol", "Carol#Ames @work")
	if err != nil || a.d.members.Label(carol) != "CarolAmes work" || !strings.Contains(out, "shortened") {
		t.Errorf("label %q, output %q, %v", a.d.members.Label(carol), out, err)
	}
}
