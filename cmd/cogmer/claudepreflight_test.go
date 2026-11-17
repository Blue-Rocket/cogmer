package main

import (
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// The preflight runs two Claude turns the first time a room is formed under a Claude
// Code version this machine has not checked (D-129). A machine where it cannot run
// must not run it again at every room command, and the probe session must not load the
// user's own plugin.

func claudePreflightHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	standInProgram(t, dir, "claude", "echo '9.9.9 (Claude Code)'", "echo 9.9.9 (Claude Code)")
	t.Setenv("PATH", pathWith(dir))
	t.Setenv("COGMER_HOME", t.TempDir())
	t.Setenv("COGMER_PREFLIGHT", "")
}

func standInForClaudeChecks(t *testing.T, results []Result, err error) *int {
	t.Helper()
	calls := new(int)
	old := runChecks
	runChecks = func(bool) ([]Result, error) { *calls++; return results, err }
	t.Cleanup(func() { runChecks = old })
	return calls
}

func TestAClaudePreflightThatCouldNotRunIsNotRepeatedWithinTheRetryWindow(t *testing.T) {
	claudePreflightHome(t)
	calls := standInForClaudeChecks(t, nil, errors.New("probe session failed: not logged in\nlong output follows"))
	for i := 0; i < 3; i++ {
		EnsureVerified("room")
	}
	if *calls != 1 {
		t.Errorf("the checks ran %d times across three rooms, want once", *calls)
	}
	v := loadVerifications()["9.9.9"]
	if why := v.Failures[probeCouldNotRun]; why != "probe session failed: not logged in" {
		t.Errorf("the attempt that could not run is recorded as %+v", v.Failures)
	}
}

func TestAClaudePreflightThatCouldNotRunIsTriedAgainAfterTheWindow(t *testing.T) {
	claudePreflightHome(t)
	saveVerification(verification{Version: "9.9.9", When: time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
		Failures: map[string]string{probeCouldNotRun: "not logged in"}})
	calls := standInForClaudeChecks(t, []Result{{B: Behavior{ID: "B01"}}}, nil)
	EnsureVerified("room")
	if *calls != 1 {
		t.Fatalf("an attempt from two hours ago was not tried again: %d runs", *calls)
	}
	if v := loadVerifications()["9.9.9"]; len(v.Failures) != 0 {
		t.Errorf("a passing run left failures recorded: %+v", v.Failures)
	}
}

func TestAClaudeResultIsNeverRepeatedByItself(t *testing.T) {
	claudePreflightHome(t)
	old := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
	for name, failures := range map[string]map[string]string{
		"a pass":                  nil,
		"a failed behavior":       {"B20": "no attachment"},
		"a failure and a non-run": {"B20": "x", probeCouldNotRun: "y"},
	} {
		saveVerification(verification{Version: "9.9.9", When: old, Failures: failures})
		calls := standInForClaudeChecks(t, nil, errors.New("must not run"))
		EnsureVerified("room")
		if *calls != 0 {
			t.Errorf("%s from two days ago was run again", name)
		}
	}
}

func TestTheClaudePreflightIsSkippedWhenAskedOrWhenThereIsNoClaudeToCheck(t *testing.T) {
	claudePreflightHome(t)
	t.Setenv("COGMER_PREFLIGHT", "off")
	calls := standInForClaudeChecks(t, nil, errors.New("must not run"))
	EnsureVerified("room")
	if *calls != 0 {
		t.Error("COGMER_PREFLIGHT=off ran the checks")
	}
	t.Setenv("COGMER_PREFLIGHT", "")
	t.Setenv("PATH", t.TempDir()) // no claude on it
	EnsureVerified("room")
	if *calls != 0 {
		t.Error("the checks ran on a machine with no Claude Code")
	}
}

// Forming a room from a Claude Code session runs the Claude preflight and not Codex's.
func TestAClaudeHostRunsItsPreflightWhenARoomIsFormed(t *testing.T) {
	claudePreflightHome(t)
	calls := standInForClaudeChecks(t, []Result{{B: Behavior{ID: "B01"}}}, nil)
	codex := standInForTheProbe(t, nil, errors.New("must not run"))
	(claudeHost{}).Preflight("room")
	if *calls != 1 || *codex != 0 {
		t.Errorf("Claude's preflight ran %d times and Codex's %d", *calls, *codex)
	}
}

// Every run of Claude Code the probe makes loads only the project's settings, so the
// user's real cogmer plugin cannot fire inside it.
func TestEveryClaudeProbeRunLoadsOnlyTheProjectSettings(t *testing.T) {
	args := claudeArgs("-p", "x")
	if len(args) < 4 || args[0] != "--setting-sources" || args[1] != "project" || args[2] != "-p" {
		t.Fatalf("claudeArgs = %v", args)
	}
	src, err := os.ReadFile("probe.go")
	if err != nil {
		t.Fatal(err)
	}
	runs := regexp.MustCompile(`exec\.Command\(claude,[^\n]*`).FindAllString(string(src), -1)
	if len(runs) == 0 {
		t.Fatal("found no run of Claude Code in probe.go, so the check reads nothing")
	}
	for _, r := range runs {
		if !strings.Contains(r, "claudeArgs(") {
			t.Errorf("a probe run starts Claude Code without claudeArgs, so the user's plugin could fire in it: %s", r)
		}
	}
}

// These are the commands a user is meant to type. What a user reads does not tell them
// to type the installed binary to forget a peer or to diagnose a daemon, which have a
// command, or to start one.
func TestNoUserFacingMessageSendsAUserToTheBinaryForForgetOrDoctor(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(string(src), "\n") {
		if !strings.Contains(line, "invocation()") {
			continue
		}
		if strings.Contains(line, "forget") || strings.Contains(line, "doctor") {
			t.Errorf("main.go:%d tells a user to type the binary: %s", i+1, strings.TrimSpace(line))
		}
	}
}
