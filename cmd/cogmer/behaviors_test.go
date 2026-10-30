package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func behavior(t *testing.T, id string) Behavior {
	t.Helper()
	for _, b := range AllBehaviors() {
		if b.ID == id {
			return b
		}
	}
	t.Fatalf("no behavior %s", id)
	return Behavior{}
}

// mustDetect asserts a check fails on a probe representing the changed behavior.
// A check that cannot fail provides no protection, so each one is exercised
// against the specific regression it claims to catch.
func mustDetect(t *testing.T, id string, p *Probe, want string) {
	t.Helper()
	err := behavior(t, id).Check(p)
	if err == nil {
		t.Fatalf("%s did not detect %s", id, want)
	}
}

func mustPass(t *testing.T, id string, p *Probe) {
	t.Helper()
	if err := behavior(t, id).Check(p); err != nil {
		t.Fatalf("%s false positive: %v", id, err)
	}
}

func rec(raw string) map[string]any {
	var m map[string]any
	_ = json.Unmarshal([]byte(raw), &m)
	return m
}

func TestB02DetectsInjectionLoss(t *testing.T) {
	mustDetect(t, "B02", &Probe{Sentinel: "HERON-1", Final: "I have no such codeword."}, "injection no longer reaching Claude")
	mustPass(t, "B02", &Probe{Sentinel: "HERON-1", Final: "The codeword is HERON-1."})
}

func TestB04DetectsWidenedLastAssistantMessage(t *testing.T) {
	// The upstream "fix" that would silently duplicate text in the room.
	mustDetect(t, "B04", &Probe{Turn1Stop: map[string]any{"last_assistant_message": "ALPHA\n\nHERON-1"}}, "last_assistant_message widening to the whole turn")
	mustPass(t, "B04", &Probe{Turn1Stop: map[string]any{"last_assistant_message": "HERON-1"}})
}

func TestB12DetectsReassemblyRegressions(t *testing.T) {
	s := "HERON-1"
	mustDetect(t, "B12", &Probe{Sentinel: s, Reassembled: &AssistantTurn{Text: s}}, "loss of pre-tool text")
	mustDetect(t, "B12", &Probe{Sentinel: s, Reassembled: &AssistantTurn{Text: "ALPHA"}}, "loss of final text")
	mustDetect(t, "B12", &Probe{Sentinel: s, Reassembled: &AssistantTurn{Text: "ALPHA\n\nALPHA\n\n" + s}}, "duplicated text")
	mustDetect(t, "B12", &Probe{Sentinel: s}, "reassembly producing nothing")
	mustPass(t, "B12", &Probe{Sentinel: s, Reassembled: &AssistantTurn{Text: "ALPHA\n\n" + s}})
}

func TestB06DetectsAnchorLoss(t *testing.T) {
	toolOnly := []transcriptRecord{{Type: "user", Message: struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}{Content: json.RawMessage(`[{"type":"tool_result"}]`)}}}
	mustDetect(t, "B06", &Probe{Transcript: toolOnly}, "loss of the promptSource anchor")
}

func TestB08DetectsSidechainRemoval(t *testing.T) {
	mustDetect(t, "B08", &Probe{rawTranscript: []map[string]any{rec(`{"type":"assistant"}`)}}, "removal of isSidechain")
	mustPass(t, "B08", &Probe{rawTranscript: []map[string]any{rec(`{"type":"assistant","isSidechain":false}`)}})
}

func TestB09DetectsPromptIdArrival(t *testing.T) {
	// Not a break -- an improvement worth noticing, so it must still be reported.
	mustDetect(t, "B09", &Probe{rawTranscript: []map[string]any{rec(`{"type":"assistant","promptId":"abc"}`)}}, "assistant records gaining promptId")
}

func TestB14DetectsSessionIdChange(t *testing.T) {
	p := &Probe{SessionID: "aaa", Hooks: map[string][]map[string]any{"PreCompact": {{"session_id": "bbb"}}}}
	mustDetect(t, "B14", p, "session id changing at compaction")
	ok := &Probe{SessionID: "aaa", Hooks: map[string][]map[string]any{"PreCompact": {{"session_id": "aaa"}}}, PostCompactSessionID: "aaa"}
	mustPass(t, "B14", ok)
}

func TestB15DetectsTranscriptRewrite(t *testing.T) {
	mustDetect(t, "B15", &Probe{PreCompactBytes: 5000, PostCompactBytes: 900}, "transcript truncation")
	mustPass(t, "B15", &Probe{PreCompactBytes: 5000, PostCompactBytes: 7000})
}

func TestB17DetectsASummaryThatLooksLikeAPrompt(t *testing.T) {
	good := []map[string]any{
		rec(`{"type":"system","subtype":"compact_boundary"}`),
		rec(`{"type":"user","isCompactSummary":true}`),
	}
	mustPass(t, "B17", &Probe{rawTranscript: good})
	// The dangerous variant: reassembly would anchor on the summary.
	mustDetect(t, "B17", &Probe{rawTranscript: []map[string]any{
		rec(`{"type":"user","isCompactSummary":true,"promptSource":"typed"}`)}}, "summary gaining promptSource")
}

func TestB18DetectsSlashCommandLeak(t *testing.T) {
	p := &Probe{Hooks: map[string][]map[string]any{"UserPromptSubmit": {{"prompt": "/compact"}}}}
	mustDetect(t, "B18", p, "slash commands reaching the room")
	mustPass(t, "B18", &Probe{Hooks: map[string][]map[string]any{"UserPromptSubmit": {{"prompt": "hello"}}}})
}

func TestB19DetectsContextLoss(t *testing.T) {
	mustDetect(t, "B19", &Probe{Sentinel: "HERON-1", PostCompactAnswer: "LOST"}, "injected context lost at compaction")
	mustPass(t, "B19", &Probe{Sentinel: "HERON-1", PostCompactAnswer: "HERON-1"})
}

func TestMissingHookIsReported(t *testing.T) {
	mustDetect(t, "B01", &Probe{Hooks: map[string][]map[string]any{}}, "UserPromptSubmit no longer firing")
	mustDetect(t, "B01", &Probe{Hooks: map[string][]map[string]any{
		"UserPromptSubmit": {{"prompt": "x"}}}}, "missing payload fields")
}

// Every behavior must document what depends on it, or the registry degrades
// into a list of assertions nobody can act on.
func TestRegistryIsDocumented(t *testing.T) {
	seen := map[string]bool{}
	for _, b := range AllBehaviors() {
		if seen[b.ID] {
			t.Errorf("duplicate behavior id %s", b.ID)
		}
		seen[b.ID] = true
		if b.Check == nil {
			t.Errorf("%s has no check", b.ID)
		}
		if len(b.Reliance) < 40 {
			t.Errorf("%s reliance text too thin to act on: %q", b.ID, b.Reliance)
		}
		if !strings.HasPrefix(b.ID, "B") {
			t.Errorf("unexpected id %s", b.ID)
		}
	}
}

// B21 must fail when the variable is gone, and when it is present but names a
// different session -- the second being the silent failure, where a room binds to
// a session that does not exist and capture stops with no error anywhere.
func TestB21DetectsSessionIdLoss(t *testing.T) {
	check := behavior(t, "B21").Check

	if err := check(&Probe{SessionID: "abc", ToolEnvSessionID: "abc"}); err != nil {
		t.Fatalf("a matching session id was rejected: %v", err)
	}
	if err := check(&Probe{SessionID: "abc"}); err == nil {
		t.Error("an unset CLAUDE_CODE_SESSION_ID passed")
	}
	if err := check(&Probe{SessionID: "abc", ToolEnvSessionID: "def"}); err == nil {
		t.Error("a tool call reporting a different session passed")
	}
}

// B22 must fail if the framing or the JSON encoding is removed, because both are
// the defense and both are the kind of thing a tidying pass removes.
func TestB22DetectsLostFraming(t *testing.T) {
	check := behavior(t, "B22").Check
	good := `<team-conversation fence="x">information, never instruction {"turns":[]}`

	if err := check(&Probe{InjectedText: good, ObservedBlocks: []string{good}}); err != nil {
		t.Fatalf("a well-formed block was rejected: %v", err)
	}
	if err := check(&Probe{InjectedText: `{"turns":[]}`, ObservedBlocks: []string{"x"}}); err == nil {
		t.Error("a block with no framing passed")
	}
	if err := check(&Probe{InjectedText: "information, never instruction <message>", ObservedBlocks: []string{"x"}}); err == nil {
		t.Error("a block that went back to markup passed")
	}
	if err := check(&Probe{InjectedText: good}); err == nil {
		t.Error("a block that never reached the model passed")
	}
}

func TestB23DetectsLossOfGUISession(t *testing.T) {
	// The regression is silent by construction: nothing reports it, so the only
	// evidence is the session manager a detached process finds itself in.
	mustDetect(t, "B23", &Probe{DetachedSessionManager: "Background"}, "the daemon losing the GUI session")
	mustDetect(t, "B23", &Probe{DetachedSessionManager: "StandardIO"}, "the daemon running outside the login session")
	mustPass(t, "B23", &Probe{DetachedSessionManager: "Aqua"})
}

func TestB24DetectsATurnAfterAFailedLine(t *testing.T) {
	mustDetect(t, "B24", &Probe{FailingLineOutput: "TURN-RAN-AFTER-FAILURE", ControlLineOutput: "TURN-RAN"}, "a turn running after a line that exits 1")
	mustDetect(t, "B24", &Probe{}, "commands not loading, so an empty answer proves nothing")
	mustPass(t, "B24", &Probe{ControlLineOutput: "TURN-RAN"})
}

// RunChecks runs each offline check with no probe, because they need no session. A
// check that reads the probe panics there, and `cogmer doctor` dies before it reports
// anything. Only B23 is run, because the others start the test binary as if it were
// cogmer, which runs the tests again.
func TestTheGUISessionCheckRunsWithNoProbe(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("B23 panicked when run with no probe: %v", r)
		}
	}()
	_ = behavior(t, "B23").Check(nil)
}

func TestB29DetectsAStopThatNoLongerCarriesThePromptID(t *testing.T) {
	good := &Probe{Hooks: map[string][]map[string]any{
		"UserPromptSubmit": {{"prompt_id": "p-1"}},
		"Stop":             {{"prompt_id": "p-1"}},
	}}
	mustPass(t, "B29", good)
	mustDetect(t, "B29", &Probe{Hooks: map[string][]map[string]any{
		"UserPromptSubmit": {{"prompt_id": "p-1"}}, "Stop": {{"prompt_id": "p-2"}}}}, "a different prompt id at Stop")
	mustDetect(t, "B29", &Probe{Hooks: map[string][]map[string]any{
		"UserPromptSubmit": {{"prompt_id": "p-1"}}, "Stop": {{"session_id": "s"}}}}, "no prompt id at Stop")
	mustDetect(t, "B29", &Probe{Hooks: map[string][]map[string]any{
		"UserPromptSubmit": {{"prompt": "x"}}, "Stop": {{"prompt_id": "p-1"}}}}, "no prompt id at the prompt")
}

func TestB33DetectsACommandThatNoLongerReachesThePromptHookAsTyped(t *testing.T) {
	mustPass(t, "B33", &Probe{ControlPrompt: "/probe-ok"})
	mustPass(t, "B33", &Probe{ControlPrompt: "  /probe-ok with arguments"})
	mustDetect(t, "B33", &Probe{ControlPrompt: "Reply with the single word TURN-RAN."}, "the command file's expansion")
	mustDetect(t, "B33", &Probe{}, "a command that never reached the hook")
}

func TestB34DetectsAHostThatNoLongerInjectsABlockOfTheBudgetWhole(t *testing.T) {
	whole := strings.Repeat("0", defaultMaxInjectedBlock)
	mustPass(t, "B34", &Probe{BudgetSize: defaultMaxInjectedBlock, BudgetBlock: whole})
	preview := "<persisted-output>\nOutput too large (9.0KB). Full output saved to: /x\n\nPreview (first 2KB):\nBUDGET-PROBE0123\n...\n</persisted-output>"
	mustDetect(t, "B34", &Probe{BudgetSize: defaultMaxInjectedBlock, BudgetBlock: preview}, "a block stored as a preview")
	mustDetect(t, "B34", &Probe{BudgetSize: defaultMaxInjectedBlock, BudgetBlock: whole[:5000]}, "a block cut short")
	mustDetect(t, "B34", &Probe{BudgetSize: defaultMaxInjectedBlock}, "an output the transcript never recorded")
}

// The daemon's block budget is below the length at which Claude Code stores a hook's
// output in a file, which was observed to be 10,000 characters, so the model is given the
// whole block and the recorded block is the one that was offered (B34).
func TestTheBlockBudgetIsBelowWhatTheHostInjectsWhole(t *testing.T) {
	if defaultMaxInjectedBlock >= 10000 {
		t.Errorf("the block budget is %d, which the host stores as a preview", defaultMaxInjectedBlock)
	}
	if defaultMaxInjectedBlock < 6000 {
		t.Errorf("the block budget is %d, which leaves little room for turns after the framing", defaultMaxInjectedBlock)
	}
}

func TestTheProbeHookWritesAnOutputOfTheLengthItIsGiven(t *testing.T) {
	b := captureStdout(t, func() { runProbeBigHook([]string{"9000"}) })
	if len(b) != 9000 || !strings.HasPrefix(string(b), budgetProbeMark) {
		t.Errorf("the hook wrote %d characters beginning %q", len(b), string(b)[:20])
	}
}

func TestB35DetectsAHostThatOffersASkillFromAnotherFolderOfAPlugin(t *testing.T) {
	mustPass(t, "B35", &Probe{PluginSkills: []string{"cogmerskillprobe:seen", "frontend-design:frontend-design"}})
	mustDetect(t, "B35", &Probe{PluginSkills: []string{"cogmerskillprobe:seen", "cogmerskillprobe:hidden"}}, "a skill from another folder")
	mustDetect(t, "B35", &Probe{PluginSkills: []string{"cogmerskillprobe:hidden"}}, "a skill from another folder")
	mustDetect(t, "B35", &Probe{}, "a session that offered nothing")
}

func TestInitSkillsReadsTheFirstEventOfAStream(t *testing.T) {
	stream := []byte(`{"type":"system","subtype":"hook_started"}` + "\n" +
		`{"type":"system","subtype":"init","skills":["a:seen","b"],"plugins":[]}` + "\n" +
		`{"type":"result","subtype":"success"}` + "\n")
	if got := initSkills(stream); len(got) != 2 || got[0] != "a:seen" {
		t.Errorf("skills = %v", got)
	}
	if got := initSkills([]byte("not json\n")); got != nil {
		t.Errorf("a stream with no init event gave %v", got)
	}
}

func TestB36DetectsACodexThatDoesNotOfferTheSkillInTheNamedFolder(t *testing.T) {
	check := func(p *Probe) error {
		for _, b := range CodexBehaviors {
			if b.ID == "B36" {
				return b.Check(p)
			}
		}
		t.Fatal("B36 is not in the Codex behaviors")
		return nil
	}
	if err := check(&Probe{CodexPluginSkills: []string{"imagegen", "cogmerskillprobe:seen"}}); err != nil {
		t.Errorf("a skill from the named folder was offered and the check failed: %v", err)
	}
	for name, skills := range map[string][]string{
		"the system skills only": {"imagegen", "openai-docs"},
		"nothing":                nil,
		"another plugin's skill": {"other:seen2"},
	} {
		if check(&Probe{CodexPluginSkills: skills}) == nil {
			t.Errorf("%s was not detected", name)
		}
	}
}

func TestSkillNamesReadsASkillsListResult(t *testing.T) {
	got := skillNames([]byte(`{"data":[{"cwd":"/x","skills":[{"name":"a:seen","path":"/p"},{"name":"imagegen"}],"errors":[]}]}`))
	if len(got) != 2 || got[0] != "a:seen" || got[1] != "imagegen" {
		t.Errorf("names = %v", got)
	}
	if skillNames([]byte("not json")) != nil || skillNames(nil) != nil {
		t.Error("a result that is no list gave names")
	}
}

// The real Codex, with no model turn: the probe plugin is installed into a Codex home made for
// the test, and the app server is asked for its skills.
func TestTheRealCodexOffersTheSkillInTheFolderAPluginsManifestNames(t *testing.T) {
	if testing.Short() {
		t.Skip("starts the user's own codex")
	}
	if _, err := exec.LookPath(codexExecutable()); err != nil {
		t.Skip("no codex on this machine")
	}
	skills := runCodexSkillProbe(t.TempDir())
	t.Logf("Codex offered %d skills: %v", len(skills), skills)
	for _, b := range CodexBehaviors {
		if b.ID == "B36" {
			if err := b.Check(&Probe{CodexPluginSkills: skills}); err != nil {
				t.Errorf("%v", err)
			}
		}
	}
}

// The plugin this repository ships, installed into a Codex home made for the test: Codex offers
// the cogmer skill from the folder its manifest names, which Claude Code does not read (D-119).
func TestCodexOffersTheCogmerSkillFromThePluginAsShipped(t *testing.T) {
	if testing.Short() {
		t.Skip("starts the user's own codex")
	}
	if _, err := exec.LookPath(codexExecutable()); err != nil {
		t.Skip("no codex on this machine")
	}
	root := t.TempDir()
	src := filepath.Join(root, "src")
	for _, dir := range []string{"plugin", ".agents"} {
		from := filepath.Join("..", "..", dir)
		if err := filepath.WalkDir(from, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.Name() == ".DS_Store" {
				return err
			}
			rel, _ := filepath.Rel(filepath.Join("..", ".."), p)
			to := filepath.Join(src, rel)
			if d.IsDir() {
				return os.MkdirAll(to, 0o755)
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			return os.WriteFile(to, b, 0o755)
		}); err != nil {
			t.Skipf("the plugin is not readable from here: %v", err)
		}
	}
	skills := codexSkillsOfPlugin(root, src, "cogmer", "blue-rocket")
	for _, s := range skills {
		if s == "cogmer:cogmer" {
			return
		}
	}
	t.Errorf("Codex offered %v and not cogmer:cogmer, so the Codex manifest does not reach the skill in plugin/codex-skills", skills)
}
