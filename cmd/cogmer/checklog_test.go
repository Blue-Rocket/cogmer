package main

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// A recorded behavior-check failure is also written to daemon.log by the process that ran
// the check (D-283, a recorded behavior-check failure is also written to daemon.log by the
// process that ran the check).

func daemonLogLines(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(daemonLogPath())
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
}

func TestARecordedBehaviorFailureIsWrittenToTheDaemonLogOnceAndAReminderWritesNone(t *testing.T) {
	claudePreflightHome(t)
	standInForClaudeChecks(t, []Result{
		{B: Behavior{ID: "B20"}, Err: errors.New("no attachment\nin the transcript")},
		{B: Behavior{ID: "B01"}},
	}, nil)
	EnsureVerified("room")

	lines := daemonLogLines(t)
	if len(lines) != 1 {
		t.Fatalf("the log holds %d lines after one failure: %q", len(lines), lines)
	}
	for _, want := range []string{"check: claude 9.9.9 failed B20 (" + registeredBehaviorTitle("B20") + "): no attachment in the transcript"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("the line %q does not hold %q", lines[0], want)
		}
	}
	if registeredBehaviorTitle("B20") == "" {
		t.Error("B20 has no title in the registry, so the line names none")
	}
	if info, err := os.Stat(daemonLogPath()); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("the log is %v, %v, want mode 600", info, err)
	}

	// A later room on the same version reminds the user and records nothing.
	EnsureVerified("another room")
	EnsureVerified("a third")
	if again := daemonLogLines(t); len(again) != 1 {
		t.Errorf("a reminder wrote to the log: %q", again)
	}
}

func TestAProbeThatCouldNotRunIsWrittenToTheDaemonLogOncePerAttemptAndAPassWritesNothing(t *testing.T) {
	claudePreflightHome(t)
	calls := standInForClaudeChecks(t, nil, errors.New("probe session failed: not logged in\nlong output follows"))
	for i := 0; i < 3; i++ {
		EnsureVerified("room")
	}
	if *calls != 1 {
		t.Fatalf("the checks ran %d times", *calls)
	}
	lines := daemonLogLines(t)
	if len(lines) != 1 || !strings.Contains(lines[0], "check: claude 9.9.9 could not run its probe: probe session failed: not logged in") ||
		strings.Contains(lines[0], "long output follows") {
		t.Errorf("a probe that could not run is logged as %q", lines)
	}

	claudePreflightHome(t)
	standInForClaudeChecks(t, []Result{{B: Behavior{ID: "B01"}}, {B: Behavior{ID: "B02"}}}, nil)
	EnsureVerified("room")
	if got := daemonLogLines(t); len(got) != 0 {
		t.Errorf("a pass wrote %q", got)
	}
}

func TestACodexAndACoworkFailureNameTheirHostsInTheDaemonLog(t *testing.T) {
	t.Setenv("COGMER_HOME", t.TempDir())
	recordCodexVerification([]Result{{B: CodexBehaviors[0], Err: errors.New("hooks disagree")}})
	recordCoworkVerification([]Result{{B: CoworkBehaviors[0], Err: errors.New("no reply shape")}})
	lines := strings.Join(daemonLogLines(t), "\n")
	for _, want := range []string{"check: codex ", " failed " + CodexBehaviors[0].ID + " (" + CodexBehaviors[0].Title + "): hooks disagree",
		"check: cowork ", " failed " + CoworkBehaviors[0].ID + " (" + CoworkBehaviors[0].Title + "): no reply shape"} {
		if !strings.Contains(lines, want) {
			t.Errorf("the log does not hold %q: %s", want, lines)
		}
	}
}

func TestALongFailureIsOneBoundedLine(t *testing.T) {
	t.Setenv("COGMER_HOME", t.TempDir())
	saveVerification(verification{Version: "9.9.9", Failures: map[string]string{"B20": strings.Repeat("word ", 400) + "\nnext"}})
	lines := daemonLogLines(t)
	if len(lines) != 1 || len(lines[0]) > 700 {
		t.Errorf("a long failure is %d lines, the first %d bytes", len(lines), len(lines[0]))
	}
}
