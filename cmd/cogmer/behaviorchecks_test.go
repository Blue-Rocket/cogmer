package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// D-130 requires that each behavior check has a test showing it fails on the change
// it is meant to catch. A check with no such test reads as protection and gives
// none, and a check can report the wrong result for a reason of its own.

func assistantAt(content string) transcriptRecord {
	var r transcriptRecord
	r.Type = "assistant"
	r.Message.Content = json.RawMessage(content)
	return r
}

func userAt(content string) transcriptRecord {
	var r transcriptRecord
	r.Type = "user"
	r.Message.Content = json.RawMessage(content)
	return r
}

// B03: the Stop hook must fire and carry the last message, or responses are not
// captured, or lose their final block.
func TestB03DetectsALostStopHookOrLostFields(t *testing.T) {
	full := map[string][]map[string]any{"Stop": {{"last_assistant_message": "HERON-1", "session_id": "s"}}}
	mustPass(t, "B03", &Probe{Hooks: full})

	mustDetect(t, "B03", &Probe{Hooks: map[string][]map[string]any{}}, "Stop no longer firing")
	mustDetect(t, "B03", &Probe{Hooks: map[string][]map[string]any{"Stop": {{"session_id": "s"}}}}, "last_assistant_message dropped from Stop")
	mustDetect(t, "B03", &Probe{Hooks: map[string][]map[string]any{"Stop": {{"last_assistant_message": "x"}}}}, "session_id dropped from Stop")
	mustDetect(t, "B03", &Probe{Hooks: map[string][]map[string]any{"Stop": {{"last_assistant_message": nil, "session_id": "s"}}}}, "last_assistant_message present and null")
}

// B05: the final text block is not in the transcript when Stop fires. The check
// reports the day that stops being true, so the entry can be corrected.
func TestB05DetectsTheTranscriptGainingTheFinalBlockAtStop(t *testing.T) {
	const s = "HERON-1"
	preToolOnly := []transcriptRecord{assistantAt(`[{"type":"text","text":"ALPHA"},{"type":"tool_use","name":"Bash"}]`)}
	mustPass(t, "B05", &Probe{Sentinel: s, AtStop: preToolOnly})
	mustPass(t, "B05", &Probe{Sentinel: s})
	// A user record that happens to hold the codeword is not the final block.
	mustPass(t, "B05", &Probe{Sentinel: s, AtStop: []transcriptRecord{userAt(`[{"type":"text","text":"HERON-1"}]`)}})

	flushed := append(append([]transcriptRecord{}, preToolOnly...), assistantAt(`[{"type":"text","text":"The codeword is HERON-1."}]`))
	mustDetect(t, "B05", &Probe{Sentinel: s, AtStop: flushed}, "the final block already in the transcript at Stop")
}

// B07: reassembly publishes blocks of type text and counts those of type tool_use.
// A rename of either loses text or tool counts without anything failing.
func TestB07DetectsARenamedBlockType(t *testing.T) {
	both := []transcriptRecord{assistantAt(`[{"type":"text","text":"ALPHA"},{"type":"tool_use","name":"Bash"}]`)}
	mustPass(t, "B07", &Probe{Transcript: both})

	mustDetect(t, "B07", &Probe{Transcript: []transcriptRecord{assistantAt(`[{"type":"output_text","text":"ALPHA"},{"type":"tool_use","name":"Bash"}]`)}}, "text renamed")
	mustDetect(t, "B07", &Probe{Transcript: []transcriptRecord{assistantAt(`[{"type":"text","text":"ALPHA"},{"type":"tool_call","name":"Bash"}]`)}}, "tool_use renamed")
	mustDetect(t, "B07", &Probe{Transcript: []transcriptRecord{assistantAt(`[{"type":"text","text":"ALPHA"}]`)}}, "no tool_use block")
	mustDetect(t, "B07", &Probe{Transcript: []transcriptRecord{userAt(`[{"type":"text","text":"x"}]`)}}, "no assistant record")
	mustDetect(t, "B07", &Probe{}, "an empty transcript")
}

// B20: delivery is confirmed by matching what was injected against what the transcript
// recorded, so an attachment that is absent or differs means nothing is ever
// confirmed.
func TestB20DetectsAMissingOrAlteredAttachment(t *testing.T) {
	const injected = "<team-conversation>HERON-1</team-conversation>"
	mustPass(t, "B20", &Probe{InjectedText: injected, ObservedBlocks: []string{injected}})
	// Whitespace is trimmed before hashing, because Claude Code trims it.
	mustPass(t, "B20", &Probe{InjectedText: injected + "\n", ObservedBlocks: []string{injected}})
	mustPass(t, "B20", &Probe{InjectedText: injected, ObservedBlocks: []string{"unrelated", injected}})

	mustDetect(t, "B20", &Probe{InjectedText: injected}, "no hook_success attachment recorded")
	mustDetect(t, "B20", &Probe{InjectedText: injected, ObservedBlocks: []string{"<team-conversation>HERON-2</team-conversation>"}}, "an attachment that differs")
	mustDetect(t, "B20", &Probe{InjectedText: injected, ObservedBlocks: []string{injected[:20]}}, "a truncated attachment")
}

// B11: a hook with no daemon must exit 0 and write nothing, or every prompt in every
// session is disturbed whenever the daemon is down. The stand-ins are scripts, since
// running the test binary as the hook would run this suite again.
func TestB11DetectsAHookThatDisturbsAPromptWithNoDaemon(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the stand-ins are shell scripts")
	}
	stand := func(name, body string) string {
		p := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	if err := checkFailOpenWith(stand("quiet", "exit 0")); err != nil {
		t.Errorf("a hook that exits 0 and says nothing was reported: %v", err)
	}
	if err := checkFailOpenWith(stand("fails", "exit 1")); err == nil {
		t.Error("a hook that exits non-zero with no daemon was not detected")
	}
	if err := checkFailOpenWith(stand("noisy", "echo stray output; exit 0")); err == nil {
		t.Error("a hook that writes to stdout with no daemon was not detected")
	}
	if err := checkFailOpenWith(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Error("a hook that cannot be run was not detected")
	}
}

// The real hook of a real build is silent with no daemon, which is what the check
// exists to keep true.
func TestTheRealHookIsSilentWithNoDaemon(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	t.Setenv("COGMER_HOME", filepath.Join(t.TempDir(), stateDirName)) // the build a test made refuses the real directory (D-265)
	if err := checkFailOpenWith(buildVersion(t, t.TempDir(), "0.0.1")); err != nil {
		t.Errorf("the hook of a real build disturbs a prompt when the daemon is down: %v", err)
	}
}

// Every behavior in the registry is named by a test that makes its check fail, so a
// behavior cannot be added without one. A test names a behavior by calling
// mustDetect or behavior with its id, or by being called Test<ID>.
func TestEveryBehaviorHasATestThatMakesItsCheckFail(t *testing.T) {
	files, err := filepath.Glob("*_test.go")
	if err != nil || len(files) == 0 {
		t.Fatal("found no test files to read")
	}
	var all strings.Builder
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		all.Write(b)
		all.WriteString("\n")
	}
	text := all.String()
	for _, b := range AllBehaviors() {
		named := regexp.MustCompile(`(mustDetect\(t, |behavior\(t, )"` + b.ID + `"|func Test` + b.ID)
		if !named.MatchString(text) {
			t.Errorf("%s (%s) has no test that makes its check fail (D-130); add one that feeds it the change it is meant to catch", b.ID, b.Title)
		}
	}
	if len(Behaviors) < 20 {
		t.Errorf("the registry holds %d behaviors, so the walk is reading the wrong thing", len(Behaviors))
	}
}
