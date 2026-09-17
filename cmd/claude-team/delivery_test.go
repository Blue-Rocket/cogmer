package main

import (
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	s, err := OpenStore("test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func seedTeammate(t *testing.T, s *Store, n int) []Event {
	t.Helper()
	other := &Identity{PeerID: "peer-alice", UserID: "alice", UserDisplayName: "Alice"}
	var out []Event
	for i := 0; i < n; i++ {
		e, err := s.Append(other, "test", "alice-session", EventUserPrompt, "teammate turn", nil)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, *e)
	}
	return out
}

// The bug this design exists to fix: the daemon's reply never reaches the hook,
// so Claude never saw the block. Nothing may be marked delivered.
func TestLostInjectionIsReofferedNotLost(t *testing.T) {
	s := testStore(t)
	seedTeammate(t, s, 2)

	offered, err := s.UndeliveredFor("test", "mine")
	if err != nil || len(offered) != 2 {
		t.Fatalf("expected 2 offered, got %d (%v)", len(offered), err)
	}
	text := FormatTeamContext(offered)
	if err := s.RecordPending("prompt-1", "mine", offered, text); err != nil {
		t.Fatal(err)
	}

	// Response lost: no evidence ever appears in the transcript.
	if n, err := s.ConfirmDelivered("mine", []string{HashBlock("something else entirely")}); err != nil || n != 0 {
		t.Fatalf("confirmed %d on non-matching evidence (%v)", n, err)
	}
	again, err := s.UndeliveredFor("test", "mine")
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 2 {
		t.Fatalf("lost injection was not re-offered: got %d events", len(again))
	}
}

func TestConfirmedInjectionIsNotReoffered(t *testing.T) {
	s := testStore(t)
	seedTeammate(t, s, 2)
	offered, _ := s.UndeliveredFor("test", "mine")
	text := FormatTeamContext(offered)
	if err := s.RecordPending("prompt-1", "mine", offered, text); err != nil {
		t.Fatal(err)
	}
	// Claude Code trims the block before recording it; confirmation must survive that.
	n, err := s.ConfirmDelivered("mine", []string{HashBlock("\n  " + text + "  \n")})
	if err != nil || n != 2 {
		t.Fatalf("confirmed %d, want 2 (%v)", n, err)
	}
	again, _ := s.UndeliveredFor("test", "mine")
	if len(again) != 0 {
		t.Fatalf("delivered events re-offered: %d", len(again))
	}
}

// Attachments accumulate across turns, so an injection missed at its own Stop
// must still be confirmable at a later one.
func TestConfirmationIsSelfHealing(t *testing.T) {
	s := testStore(t)
	seedTeammate(t, s, 1)
	first, _ := s.UndeliveredFor("test", "mine")
	firstText := FormatTeamContext(first)
	_ = s.RecordPending("prompt-1", "mine", first, firstText)

	seedTeammate(t, s, 1)
	second, _ := s.UndeliveredFor("test", "mine") // still includes the unconfirmed one
	if len(second) != 2 {
		t.Fatalf("expected both events still outstanding, got %d", len(second))
	}
	secondText := FormatTeamContext(second)
	_ = s.RecordPending("prompt-2", "mine", second, secondText)

	// A later Stop sees both attachments at once.
	n, err := s.ConfirmDelivered("mine", []string{HashBlock(firstText), HashBlock(secondText)})
	if err != nil || n == 0 {
		t.Fatalf("self-healing confirmation failed: %d (%v)", n, err)
	}
	if again, _ := s.UndeliveredFor("test", "mine"); len(again) != 0 {
		t.Fatalf("still outstanding after confirmation: %d", len(again))
	}
}

// A session is never shown its own events.
func TestOwnEventsAreNeverOffered(t *testing.T) {
	s := testStore(t)
	me := &Identity{PeerID: "peer-me", UserID: "david", UserDisplayName: "David"}
	if _, err := s.Append(me, "test", "mine", EventUserPrompt, "my own turn", nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.UndeliveredFor("test", "mine"); len(got) != 0 {
		t.Fatalf("session offered its own events: %d", len(got))
	}
	// ...but another local session in the same room must see them.
	if got, _ := s.UndeliveredFor("test", "other-session"); len(got) != 1 {
		t.Fatalf("second local session did not see peer events: %d", len(got))
	}
}

// When the transcript carries no evidence at all, delivery degrades to trust
// rather than re-injecting the same context forever.
func TestFallbackCommitWhenEvidenceUnavailable(t *testing.T) {
	s := testStore(t)
	seedTeammate(t, s, 2)
	offered, _ := s.UndeliveredFor("test", "mine")
	_ = s.RecordPending("prompt-1", "mine", offered, FormatTeamContext(offered))

	n, err := s.CommitPending("mine", "prompt-1")
	if err != nil || n != 2 {
		t.Fatalf("fallback committed %d, want 2 (%v)", n, err)
	}
	if again, _ := s.UndeliveredFor("test", "mine"); len(again) != 0 {
		t.Fatalf("fallback did not stop re-offering: %d", len(again))
	}
	if n, _ := s.CommitPending("mine", "prompt-1"); n != 0 {
		t.Error("fallback commit is not idempotent")
	}
}

func TestPrunePendingBoundsGrowth(t *testing.T) {
	s := testStore(t)
	ev := seedTeammate(t, s, 1)
	for i := 0; i < 30; i++ {
		if err := s.RecordPending(filepath.Join("p", string(rune('a'+i%26)), string(rune('0'+i/26))), "mine", ev, "block"); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.PrunePending("mine", 5); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM pending_injection WHERE claude_session_id='mine'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n > 5 {
		t.Fatalf("pruning left %d rows, want <= 5", n)
	}
}
