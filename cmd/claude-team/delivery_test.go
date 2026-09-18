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
	text := FormatTeamContext(offered, nil)
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
	text := FormatTeamContext(offered, nil)
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
	firstText := FormatTeamContext(first, nil)
	_ = s.RecordPending("prompt-1", "mine", first, firstText)

	seedTeammate(t, s, 1)
	second, _ := s.UndeliveredFor("test", "mine") // still includes the unconfirmed one
	if len(second) != 2 {
		t.Fatalf("expected both events still outstanding, got %d", len(second))
	}
	secondText := FormatTeamContext(second, nil)
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
	_ = s.RecordPending("prompt-1", "mine", offered, FormatTeamContext(offered, nil))

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
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM pending_injection WHERE origin_session_id='mine'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n > 5 {
		t.Fatalf("pruning left %d rows, want <= 5", n)
	}
}

// B4: a peer that loses its room database restarts its counter, so everything it
// publishes afterwards lands on a sequence another peer already holds. Absorbing
// that as a duplicate is silently unrecoverable.
func TestSequenceConflictIsQuarantinedNotAbsorbed(t *testing.T) {
	s := testStore(t)
	held := &Event{
		EventID: "evt-original", PeerID: "peer-alice", PeerSequence: 1,
		RoomID: "test", Timestamp: "2026-09-16T00:00:00Z", EventType: EventUserPrompt,
		Content: "the turn Alice actually sent",
	}
	if res, err := s.Insert(held); err != nil || res != InsertStored {
		t.Fatalf("setup: %v %v", res, err)
	}

	// Same peer and sequence, different event: Alice restarted from a lost database.
	conflicting := &Event{
		EventID: "evt-after-restore", PeerID: "peer-alice", PeerSequence: 1,
		RoomID: "test", Timestamp: "2026-09-16T01:00:00Z", EventType: EventUserPrompt,
		Content: "a different turn entirely",
	}
	res, err := s.Insert(conflicting)
	if err != nil {
		t.Fatal(err)
	}
	if res != InsertConflict {
		t.Fatalf("got %v, want conflict — a restarted counter was absorbed silently", res)
	}

	// The room keeps what it held; the conflicting event does not overwrite it.
	evs, err := s.ListRoom("test")
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 1 || evs[0].EventID != "evt-original" {
		t.Fatalf("room was modified by a conflicting event: %+v", evs)
	}

	// The rejected event is kept, because discarding it destroys the evidence
	// that distinguishes lost state from forgery.
	cs, err := s.ListConflicts()
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 1 {
		t.Fatalf("conflict not recorded: %d", len(cs))
	}
	if cs[0].HeldEventID != "evt-original" || cs[0].IncomingEventID != "evt-after-restore" {
		t.Errorf("conflict record does not identify both events: %+v", cs[0])
	}
}

// Ordinary redelivery must stay silent, or anti-entropy and transitive relay
// would report a conflict on every repeated exchange.
func TestRedeliveryIsNotAConflict(t *testing.T) {
	s := testStore(t)
	ev := &Event{
		EventID: "evt-1", PeerID: "peer-alice", PeerSequence: 1,
		RoomID: "test", Timestamp: "2026-09-16T00:00:00Z", EventType: EventUserPrompt, Content: "hello",
	}
	if res, _ := s.Insert(ev); res != InsertStored {
		t.Fatalf("first insert: %v", res)
	}
	for i := 0; i < 3; i++ {
		res, err := s.Insert(ev)
		if err != nil || res != InsertDuplicate {
			t.Fatalf("redelivery %d: %v %v", i, res, err)
		}
	}
	if cs, _ := s.ListConflicts(); len(cs) != 0 {
		t.Errorf("redelivery recorded %d conflict(s)", len(cs))
	}
	if evs, _ := s.ListRoom("test"); len(evs) != 1 {
		t.Errorf("redelivery duplicated the event: %d copies", len(evs))
	}
}

// Distinct events from the same peer must still store normally.
func TestSequentialEventsStoreNormally(t *testing.T) {
	s := testStore(t)
	for i := int64(1); i <= 3; i++ {
		ev := &Event{
			EventID: "evt-" + string(rune('a'+i)), PeerID: "peer-alice", PeerSequence: i,
			RoomID: "test", Timestamp: "2026-09-16T00:00:00Z", EventType: EventUserPrompt, Content: "x",
		}
		if res, err := s.Insert(ev); err != nil || res != InsertStored {
			t.Fatalf("sequence %d: %v %v", i, res, err)
		}
	}
	if evs, _ := s.ListRoom("test"); len(evs) != 3 {
		t.Fatalf("want 3 events, got %d", len(evs))
	}
}
