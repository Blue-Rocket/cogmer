package main

import (
	"errors"
	"testing"
)

// twoKeysDerivingOneName finds two keys whose derived names are the same, which
// among 8,280 names takes about a hundred keys.
func twoKeysDerivingOneName(t *testing.T) (string, string) {
	t.Helper()
	seen := map[string]string{}
	for i := 0; i < 20000; i++ {
		p := testIdentity(t).PeerID
		if q, ok := seen[PeerName(p)]; ok {
			return q, p
		}
		seen[PeerName(p)] = p
	}
	t.Fatal("no two of 20,000 keys derived one name")
	return "", ""
}

// Two colleagues given no name are both recorded even when their keys derive one
// name, since a derived name is shown and never stored, so it is held by no key
// (D-021, D-074). Before, the second was refused as a changed key.
func TestColleaguesWhoseKeysDeriveOneNameAreBothRecorded(t *testing.T) {
	m := testMembership(t)
	a, b := twoKeysDerivingOneName(t)
	for _, p := range []string{a, b} {
		if err := m.Allow(p, ""); err != nil {
			t.Fatalf("a colleague given no name was refused: %v", err)
		}
		if m.Label(p) != "" {
			t.Errorf("a colleague given no name has the label %q", m.Label(p))
		}
	}
	known, err := m.KnownPeers()
	if err != nil {
		t.Fatal(err)
	}
	if len(known) != 2 {
		t.Fatalf("want two colleagues, got %d", len(known))
	}
	for _, p := range known {
		if p.Name != PeerName(p.PeerID) {
			t.Errorf("a colleague given no name is listed as %q, not by the derived name %q", p.Name, PeerName(p.PeerID))
		}
	}
	if len(m.Labels()) != 0 {
		t.Errorf("colleagues given no name have labels: %v", m.Labels())
	}
}

// A name chosen for one colleague may not be the derived name another colleague is
// shown by, so no two colleagues are ever shown under one name.
func TestAChosenNameMayNotBeAnotherColleaguesDerivedName(t *testing.T) {
	m := testMembership(t)
	a, b := testIdentity(t).PeerID, testIdentity(t).PeerID
	if err := m.Allow(a, ""); err != nil {
		t.Fatal(err)
	}
	var taken *NameTakenError
	if err := m.Allow(b, PeerName(a)); !errors.As(err, &taken) || taken.Existing != a {
		t.Fatalf("a chosen name equal to another colleague's derived name was accepted: %v", err)
	}
}

// Giving a colleague their derived name outright clears the name they had, since
// a derived name is no name anybody gave.
func TestGivingTheDerivedNameClearsTheLabel(t *testing.T) {
	m := testMembership(t)
	a := testIdentity(t).PeerID
	if err := m.Allow(a, "alice"); err != nil {
		t.Fatal(err)
	}
	if err := m.Allow(a, PeerName(a)); err != nil {
		t.Fatal(err)
	}
	if m.Label(a) != "" {
		t.Errorf("the label is %q after the derived name was given", m.Label(a))
	}
}

// A store that recorded a colleague under their derived name has it cleared, so a
// colleague whose key derives the same name can be recorded beside them.
func TestADerivedNameStoredBeforeIsCleared(t *testing.T) {
	m := testMembership(t)
	a, b := twoKeysDerivingOneName(t)
	if _, err := m.db.Exec(`INSERT INTO known_peers (peer_id, name, added_at) VALUES (?,?,?), (?,?,?)`,
		a, PeerName(a), "2026-11-01T00:00:00Z", "ed25519:not-a-real-key-but-chosen", "chosen", "2026-11-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if err := clearDerivedLabels(m.db); err != nil {
		t.Fatal(err)
	}
	var name string
	if err := m.db.QueryRow(`SELECT name FROM known_peers WHERE peer_id = ?`, a).Scan(&name); err != nil || name != "" {
		t.Fatalf("the stored derived name is %q after the migration (%v)", name, err)
	}
	if err := m.db.QueryRow(`SELECT name FROM known_peers WHERE peer_id = ?`, "ed25519:not-a-real-key-but-chosen").Scan(&name); err != nil || name != "chosen" {
		t.Fatalf("a chosen name became %q (%v)", name, err)
	}
	if err := m.Allow(b, ""); err != nil {
		t.Fatalf("a colleague deriving the same name was refused after the migration: %v", err)
	}
}

// A label the user cleared on one machine is cleared on the others when the user's
// lists are copied (D-228, a user's labels are copied between the user's machines).
func TestAClearedLabelIsClearedWhereTheListsAreCopied(t *testing.T) {
	m := testMembership(t)
	a := testIdentity(t).PeerID
	if err := m.Allow(a, "alice"); err != nil {
		t.Fatal(err)
	}
	if err := m.applyColleague(a, colleagueEntry{Label: ""}); err != nil {
		t.Fatal(err)
	}
	if m.Label(a) != "" {
		t.Errorf("the copy kept the label %q that the user cleared", m.Label(a))
	}
}

// A colleague who has a label is still shown by their derived name beside it
// (D-094), so that name is not free for somebody else either.
func TestALabelledColleaguesDerivedNameIsNotFreeForAnother(t *testing.T) {
	m := testMembership(t)
	a, b := testIdentity(t).PeerID, testIdentity(t).PeerID
	if err := m.Allow(a, "alice"); err != nil {
		t.Fatal(err)
	}
	var taken *NameTakenError
	if err := m.Allow(b, PeerName(a)); !errors.As(err, &taken) || taken.Existing != a {
		t.Fatalf("a labelled colleague's derived name was given to another: %v", err)
	}
}
