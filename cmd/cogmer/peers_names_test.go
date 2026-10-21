package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func insertNamedEvent(t *testing.T, s *Store, peer string, seq int, ts, name string) {
	t.Helper()
	_, err := s.db.Exec(`INSERT INTO events (event_id, peer_id, peer_sequence, room_id, timestamp,
		user_display_name, event_type) VALUES (?,?,?,?,?,?,?)`,
		peer+"/"+ts, peer, seq, "room", ts, name, EventUserPrompt)
	if err != nil {
		t.Fatal(err)
	}
}

// The display name a peer asserts changes over time and appears in every room, so the
// list says what the latest event in any room carried.
func TestReportedNamesTakeTheLatestFromAnyRoom(t *testing.T) {
	t.Setenv("COGMER_HOME", t.TempDir())
	peer, quiet, absent := testIdentity(t).PeerID, testIdentity(t).PeerID, testIdentity(t).PeerID

	a, err := OpenStore("room-a")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := OpenStore("room-b")
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	insertNamedEvent(t, a, peer, 1, "2026-10-01T10:00:00Z", "Arielle")
	insertNamedEvent(t, b, peer, 1, "2026-10-02T10:00:00Z", "Ariellemelinda")
	insertNamedEvent(t, a, peer, 2, "2026-10-03T10:00:00Z", "Newest")
	insertNamedEvent(t, a, quiet, 1, "2026-10-01T10:00:00Z", "")

	// A file that is not a room database is skipped.
	if err := os.WriteFile(filepath.Join(homeDir(), "rooms", "junk.db"), []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := reportedNames([]string{peer, quiet, absent})
	if got[peer] != "Newest" {
		t.Errorf("the name reported by the latest event is %q, want Newest", got[peer])
	}
	if _, ok := got[quiet]; ok {
		t.Errorf("a peer whose events carry no name has one: %q", got[quiet])
	}
	if _, ok := got[absent]; ok {
		t.Errorf("a peer with no events has a name: %q", got[absent])
	}
}

// A name from another machine is reduced before it is printed, since it can carry
// anything.
func TestAReportedNameIsReducedBeforeItIsShown(t *testing.T) {
	t.Setenv("COGMER_HOME", t.TempDir())
	peer := testIdentity(t).PeerID
	s, err := OpenStore("room-a")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	insertNamedEvent(t, s, peer, 1, "2026-10-01T10:00:00Z", "Mal\nory\x1b[31m "+strings.Repeat("x", 80))
	got := reportedNames([]string{peer})[peer]
	if strings.ContainsAny(got, "\n\x1b") || len(got) > 32 {
		t.Errorf("the name was not reduced: %q", got)
	}
}

// Each of the three names is said to be what it is, so a reader never has to know which
// column is which.
func TestAPeersEntryNamesEachOfItsThreeNames(t *testing.T) {
	key := testIdentity(t).PeerID
	derived := PeerName(key)

	labelled := formatPeer(KnownPeer{PeerID: key, Name: "alice", VerifiedAt: "2026-10-12T09:00:00Z"}, "Arielle")
	for _, want := range []string{
		"alice  verified 2026-10-12",
		"what you call them:         alice",
		"derived from their key:     " + derived,
		"what they call themselves:  \"Arielle\" (their own claim)",
		"key:                        " + key,
	} {
		if !strings.Contains(labelled, want) {
			t.Errorf("a labelled peer's entry lacks %q:\n%s", want, labelled)
		}
	}

	// With no label the stored name is the derived one, which leads and is not offered as
	// a name the user gave.
	unlabelled := formatPeer(KnownPeer{PeerID: key, Name: derived, VerifiedAt: "2026-10-12T09:00:00Z"}, "")
	for _, want := range []string{
		derived + "  verified 2026-10-12",
		"what you call them:         none given",
		"what they call themselves:  not seen yet",
	} {
		if !strings.Contains(unlabelled, want) {
			t.Errorf("an unlabelled peer's entry lacks %q:\n%s", want, unlabelled)
		}
	}
	if strings.Contains(unlabelled, "what you call them:         "+derived) {
		t.Errorf("the derived name was shown as the name the user gave:\n%s", unlabelled)
	}

	unfinished := formatPeer(KnownPeer{PeerID: key, Name: derived}, "")
	if !strings.Contains(unfinished, derived+"  not finished") {
		t.Errorf("an unfinished peer's entry does not say so:\n%s", unfinished)
	}
}
