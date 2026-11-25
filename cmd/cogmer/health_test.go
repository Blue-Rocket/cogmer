package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// `health` answers at once and spends no turn, and says what the behavior check recorded for the
// host's version (D-298).
func TestHealthSaysWhatTheBehaviorCheckRecorded(t *testing.T) {
	claudePreflightHome(t) // a Claude Code 9.9.9 on the path
	t.Setenv("CLAUDE_CODE_REMOTE", "")
	t.Setenv(cloudKeyVar, "")
	report := func() string { return checksReport(claudeHost{}) }

	if got := report(); !strings.Contains(got, "has not been checked") {
		t.Errorf("a version never checked: %q", got)
	}
	saveVerification(verification{Version: "9.9.9", When: "2026-11-25T10:00:00Z",
		Failures: map[string]string{probeCouldNotRun: "probe session failed: not logged in"}})
	if got := report(); !strings.Contains(got, "could not run") || !strings.Contains(got, "not logged in") || strings.Contains(got, "failed 1") {
		t.Errorf("a probe that could not run: %q", got)
	}
	saveVerification(verification{Version: "9.9.9", When: "2026-11-25T10:00:00Z", Failures: map[string]string{"B20": "no attachment"}})
	got := report()
	if !strings.Contains(got, "failed 1 behavior") || !strings.Contains(got, "B20 ") {
		t.Errorf("a failed behavior is not named: %q", got)
	}
	saveVerification(verification{Version: "9.9.9", When: "2026-11-25T10:00:00Z"})
	if got := report(); !strings.Contains(got, "passed every check") {
		t.Errorf("a pass: %q", got)
	}
}

func TestHealthInACloudContainerSaysTheChecksAreNotRunAndRunsNone(t *testing.T) {
	claudePreflightHome(t)
	t.Setenv("CLAUDE_CODE_REMOTE", "true")
	if got := checksReport(claudeHost{}); !strings.Contains(got, "not run in a cloud container") {
		t.Errorf("report in a cloud container: %q", got)
	}
	if got := startExtendedRun(claudeHost{}, time.Now()); !strings.Contains(got, "not run in a cloud container") {
		t.Errorf("the whole check was started in a cloud container: %q", got)
	}
	if _, err := os.Stat(extendedLogPath(claudeHost{})); err == nil {
		t.Error("a log was written for a run that was not started")
	}
}

// standInForTheWholeCheck is a program that prints what the whole check prints, with a failure.
func standInForTheWholeCheck(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in is a shell script")
	}
	path := filepath.Join(t.TempDir(), "cogmer")
	script := "#!/bin/sh\necho 'cogmer doctor'\necho '  FAIL  B20  injected output is recorded'\necho\necho '2 checked, 1 failed'\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	old := extendedProgram
	extendedProgram = func() (string, error) { return path, nil }
	t.Cleanup(func() { extendedProgram = old })
}

func TestTheWholeCheckStartsInTheBackgroundAndLaterReportsWhatItFound(t *testing.T) {
	claudePreflightHome(t)
	t.Setenv("CLAUDE_CODE_REMOTE", "")
	t.Setenv(cloudKeyVar, "")
	standInForTheWholeCheck(t)

	if got := startExtendedRun(claudeHost{}, time.Now()); !strings.Contains(got, "Started the whole check") {
		t.Fatalf("start: %q", got)
	}
	end := time.Now().Add(30 * time.Second)
	for time.Now().Before(end) {
		if run, ok := readExtendedRun(claudeHost{}); ok && run.Finished {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	got := extendedReport(claudeHost{}, time.Now())
	if !strings.Contains(got, "FAIL  B20") || !strings.Contains(got, "2 checked, 1 failed") {
		t.Errorf("what the whole check found was not reported: %q", got)
	}
	if fi, err := os.Stat(extendedLogPath(claudeHost{})); err != nil || (runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600) {
		t.Errorf("the log is not private to the account: %v %v", fi, err)
	}
}

func TestARunThatHasNotFinishedIsLeftAloneUntilItIsTooOldToBeRunning(t *testing.T) {
	claudePreflightHome(t)
	t.Setenv("CLAUDE_CODE_REMOTE", "")
	t.Setenv(cloudKeyVar, "")
	standInForTheWholeCheck(t)
	now := time.Now()
	started := func(at time.Time) {
		if err := os.WriteFile(extendedLogPath(claudeHost{}), []byte(extendedStart+at.UTC().Format(time.RFC3339)+"\ncogmer doctor\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	started(now.Add(-30 * time.Second))
	if got := extendedReport(claudeHost{}, now); !strings.Contains(got, "is running") {
		t.Errorf("a run 30 seconds old: %q", got)
	}
	if got := startExtendedRun(claudeHost{}, now); !strings.Contains(got, "already running") {
		t.Errorf("a second run was started over the first: %q", got)
	}
	started(now.Add(-10 * time.Minute))
	if got := extendedReport(claudeHost{}, now); !strings.Contains(got, "did not finish") {
		t.Errorf("a run 10 minutes old: %q", got)
	}
	if got := startExtendedRun(claudeHost{}, now); !strings.Contains(got, "Started the whole check") {
		t.Errorf("a run that died was not replaced: %q", got)
	}
}

func TestHealthTakesNoArgumentOrExtended(t *testing.T) {
	t.Setenv("COGMER_HOME", t.TempDir())
	// Neither prints a turn's worth of anything, and anything else is refused without starting a run.
	runHealth([]string{"deep"})
	if _, err := os.Stat(extendedLogPath(claudeHost{})); err == nil {
		t.Error("an unknown argument started a run")
	}
}

// One host's run of the whole check says nothing about another's, so each reads only its own log.
func TestEachHostReadsOnlyItsOwnRunOfTheWholeCheck(t *testing.T) {
	t.Setenv("COGMER_HOME", t.TempDir())
	if err := ensureStateDir(); err != nil {
		t.Fatal(err)
	}
	log := extendedStart + time.Now().UTC().Format(time.RFC3339) + "\n2 checked, 1 failed\n"
	if err := os.WriteFile(extendedLogPath(claudeHost{}), []byte(log), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := extendedReport(claudeHost{}, time.Now()); !strings.Contains(got, "2 checked, 1 failed") {
		t.Errorf("a host did not read its own run: %q", got)
	}
	if got := extendedReport(codexHost{}, time.Now()); got != "" {
		t.Errorf("Codex was shown the Claude Code run: %q", got)
	}
}
