package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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
	// Continue from whatever is already there: a teammate seeded twice is one
	// teammate speaking twice, not one restarting its sequence.
	from, err := s.HighestSequence(other.PeerID)
	if err != nil {
		t.Fatal(err)
	}
	var out []Event
	for i := 0; i < n; i++ {
		e, err := s.Append(from+int64(i+1), other, "test", "alice-session", EventUserPrompt, "teammate turn", nil)
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
	if _, err := s.Append(1, me, "test", "mine", EventUserPrompt, "my own turn", nil); err != nil {
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

// Stored and read back: an event whose scheme is not persisted cannot be verified
// on the way out, which is what a peer refetching a room's history receives.
func TestStoredEventsKeepTheirScheme(t *testing.T) {
	s := testStore(t)
	id := testIdentity(t)
	ev := Event{
		EventID: "e1", PeerID: id.PeerID, PeerSequence: 1, RoomID: "r1",
		Timestamp: "2026-09-18T00:00:00Z", EventType: EventUserPrompt, Content: "hi",
	}
	ev.Sign(id.private)
	if _, err := s.Insert(&ev); err != nil {
		t.Fatal(err)
	}

	got, err := s.EventsSince("r1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("stored %d events, want 1", len(got))
	}
	if got[0].SigVersion != currentSigVersion {
		t.Errorf("scheme came back as %d, want %d", got[0].SigVersion, currentSigVersion)
	}
	if err := got[0].Verify(); err != nil {
		t.Errorf("an event read back from storage does not verify: %v", err)
	}
}

// §21's limits are a safety valve, and each bounds something different. The block
// budget is the one that bounds what actually reaches a context window: events that
// pass both other limits can still render a block nobody would want injected.
func TestTheInjectedBlockIsBounded(t *testing.T) {
	t.Setenv("COGMER_MAX_EVENTS", "100")
	t.Setenv("COGMER_MAX_EVENT_CHARS", "11000")
	t.Setenv("COGMER_MAX_BLOCK_CHARS", "30000")

	var evs []Event
	for i := 0; i < 40; i++ {
		evs = append(evs, Event{
			PeerID: "ed25519:x", UserDisplayName: "Alice", EventType: EventUserPrompt,
			Timestamp: "2026-09-18T10:00:00Z",
			Content:   strings.Repeat("x", 10000) + fmt.Sprintf(" turn-%02d", i),
		})
	}

	out := FormatTeamContext(evs, fixedFacts{verified: true})
	if len(out) > 40000 {
		t.Errorf("block is %d characters; the budget was 30000 and every event passed the other two limits", len(out))
	}
	if !strings.Contains(out, "CONTEXT_CATCHUP_REQUIRED") {
		t.Error("turns were dropped without saying so")
	}
	// The NEWEST turns are what survive: a referent is most likely to point at
	// recent conversation.
	if !strings.Contains(out, "turn-39") {
		t.Error("the most recent turn was dropped")
	}
	if strings.Contains(out, "turn-00") {
		t.Error("the oldest turn survived a budget that should have dropped it first")
	}
}

// Configurable, per §21. A limit nobody can change is one that is wrong for
// somebody, and the daemon must not need restarting to change it.
func TestLimitsAreConfigurable(t *testing.T) {
	if got := limits(); got.events != defaultMaxInjectedEvents || got.block != defaultMaxInjectedBlock {
		t.Fatalf("defaults not applied: %+v", got)
	}
	t.Setenv("COGMER_MAX_EVENTS", "3")
	if got := limits().events; got != 3 {
		t.Errorf("event limit is %d, want 3", got)
	}
	// Nonsense is ignored rather than obeyed: a zero limit would inject nothing
	// and look exactly like a room where nobody is talking.
	t.Setenv("COGMER_MAX_EVENTS", "0")
	if got := limits().events; got != defaultMaxInjectedEvents {
		t.Errorf("a zero limit was accepted, giving %d", got)
	}
	t.Setenv("COGMER_MAX_EVENTS", "not-a-number")
	if got := limits().events; got != defaultMaxInjectedEvents {
		t.Errorf("an unparseable limit was accepted, giving %d", got)
	}
}

// The host stores a hook's output of more than about 10,000 characters in a file and gives
// the model a preview of its first part, and the newest turns are at the end of the block,
// so the block is kept under that whatever the turns hold (B34). It is measured on the
// block as written, not estimated, since JSON escaping and the framing add to the turns.
func TestTheBlockIsKeptUnderWhatTheHostInjectsWholeWhateverTheTurnsHold(t *testing.T) {
	mk := func(n int, size int, filler string) []Event {
		var evs []Event
		for i := 0; i < n; i++ {
			evs = append(evs, Event{PeerID: "ed25519:x", UserDisplayName: "Alice", EventType: EventAssistantMessage,
				Timestamp: "2026-10-14T10:00:00Z", Content: fmt.Sprintf("turn-%03d ", i) + strings.Repeat(filler, size/len(filler))})
		}
		return evs
	}
	for name, c := range map[string]struct {
		evs      []Event
		wantNote bool
	}{
		"185 turns of about two thousand characters": {mk(185, 2000, "x"), true},
		"a few ordinary turns":                       {mk(3, 800, "x"), false},
		"turns that JSON escapes heavily":            {mk(60, 3000, "\"\n\\"), true},
		"one enormous turn":                          {mk(1, 40000, "x"), false},
		"two turns at the limit of one":              {mk(2, defaultMaxInjectedChars, "x"), true},
	} {
		out := FormatTeamContext(c.evs, fixedFacts{verified: true})
		if len(out) >= 10000 || len(out) > defaultMaxInjectedBlock {
			t.Errorf("%s: the block is %d characters; the host stores anything over 10,000 as a preview, and the budget is %d", name, len(out), defaultMaxInjectedBlock)
		}
		if got := strings.Contains(out, "CONTEXT_CATCHUP_REQUIRED"); got != c.wantNote {
			t.Errorf("%s: note about omitted turns = %v, want %v", name, got, c.wantNote)
		}
		last := fmt.Sprintf("turn-%03d", len(c.evs)-1)
		if len(c.evs) > 1 && !strings.Contains(out, last) {
			t.Errorf("%s: the newest turn, %s, is not in the block", name, last)
		}
	}
	// the count the note gives is the number left out
	out := FormatTeamContext(mk(185, 2000, "x"), fixedFacts{verified: true})
	re := regexp.MustCompile(`(\d+) earlier room events were omitted`)
	m := re.FindStringSubmatch(out)
	if m == nil {
		t.Fatal("no count of omitted turns")
	}
	var kept int
	var payload struct {
		Turns []json.RawMessage `json:"turns"`
	}
	start := strings.Index(out, `{"turns"`)
	end := strings.Index(out[start:], "}\n</team-conversation")
	if start < 0 || end < 0 || json.Unmarshal([]byte(out[start:start+end+1]), &payload) != nil {
		t.Fatalf("the block's turns could not be read back")
	}
	kept = len(payload.Turns)
	if omitted, _ := strconv.Atoi(m[1]); omitted+kept != 185 {
		t.Errorf("%d omitted and %d kept is not 185 turns", omitted, kept)
	}
}

// A block the host stored as a preview is said so in the daemon's log, since nothing else
// shows that a session was given a preview and a file.
func TestABlockTheHostStoredAsAPreviewIsNotCountedDeliveredAndIsSaid(t *testing.T) {
	preview := "<persisted-output>\nOutput too large. Full output saved to: /x\n\nPreview:\n<team-conversation\n...\n</persisted-output>"
	if !hasPersistedBlock([]string{"an ordinary block", preview}) {
		t.Error("a preview was not recognized")
	}
	if hasPersistedBlock([]string{"an ordinary block"}) {
		t.Error("an ordinary block was taken for a preview")
	}
	var logged bytes.Buffer
	log.SetOutput(&logged)
	defer log.SetOutput(os.Stderr)
	s := testStore(t)
	seedTeammate(t, s, 1)
	events, _ := s.UndeliveredFor("test", "mine")
	if err := s.RecordPending("turn", "mine", events, "the block that was offered"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "t.jsonl")
	rec, _ := json.Marshal(map[string]any{"type": "attachment", "attachment": map[string]any{
		"type": "hook_success", "hookName": "UserPromptSubmit", "hookEvent": "UserPromptSubmit", "content": preview}})
	if err := os.WriteFile(path, append(rec, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	d := &Daemon{}
	d.confirmDelivery(s, stopReq{SessionID: "mine", PromptID: "turn", TranscriptPath: path}, claudeHost{})
	if left, _ := s.UndeliveredFor("test", "mine"); len(left) != 1 {
		t.Error("a block stored as a preview was counted as delivered")
	}
	if !strings.Contains(logged.String(), "stored a block of session") {
		t.Errorf("nothing said that the host stored the block as a preview: %q", logged.String())
	}
}
