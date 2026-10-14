package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestB30DetectsAHarnessThatDropsTheJSONReply(t *testing.T) {
	good := &Probe{McpJSONWord: "JSONFORM-1", McpAnswer: "UserPromptSubmit hook additional context: JSONFORM-1"}
	mustPass(t, "B30", good)
	mustDetect(t, "B30", &Probe{McpJSONWord: "JSONFORM-1", McpAnswer: "NONE"}, "a reply that never reached the model")
	mustDetect(t, "B30", &Probe{McpJSONWord: "JSONFORM-1", McpErr: errors.New("exit status 1")}, "a session that failed")
}

func TestB31DetectsAnAbsentServerThatShowsAnErrorOrFailsTheSession(t *testing.T) {
	mustPass(t, "B31", &Probe{McpAbsentServer: coworkAbsentServer, McpAnswer: "NONE"})
	mustDetect(t, "B31", &Probe{McpAbsentServer: coworkAbsentServer, McpAnswer: "hook error: server " + coworkAbsentServer + " not found"}, "an error that names the server")
	mustDetect(t, "B31", &Probe{McpAbsentServer: coworkAbsentServer, McpErr: errors.New("exit status 1")}, "a session that failed")
}

func TestB32DetectsAHookOnSessionStartThatNowReachesItsServer(t *testing.T) {
	mustPass(t, "B32", &Probe{McpCalls: []string{"2026-10-14T00:00:00Z event=UserPromptSubmit format=json"}})
	mustDetect(t, "B32", &Probe{McpCalls: []string{"2026-10-14T00:00:00Z event=SessionStart format=plain",
		"2026-10-14T00:00:01Z event=UserPromptSubmit format=json"}}, "a call at launch")
	mustDetect(t, "B32", &Probe{}, "a probe that saw no call at all")
}

func coworkHome(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in for Claude Code is a shell script")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte("#!/bin/sh\necho '9.9.9 (Claude Code)'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":/usr/bin:/bin")
	t.Setenv("COGMER_HOME", t.TempDir())
	t.Setenv("COGMER_PREFLIGHT", "")
}

func standInForCoworkChecks(t *testing.T, results []Result, err error) *int {
	t.Helper()
	calls := new(int)
	old := runCoworkChecks
	runCoworkChecks = func() ([]Result, error) { *calls++; return results, err }
	t.Cleanup(func() { runCoworkChecks = old })
	return calls
}

func TestTheCoworkChecksAreRecordedApartFromClaudeCodes(t *testing.T) {
	coworkHome(t)
	saveVerification(verification{Version: "9.9.9", Failures: map[string]string{"B20": "no attachment"}})
	calls := standInForCoworkChecks(t, []Result{{B: Behavior{ID: coworkReplyShape}, Err: errors.New("dropped")}}, nil)
	EnsureCoworkVerified("room")
	EnsureCoworkVerified("room")
	if *calls != 1 {
		t.Errorf("the checks ran %d times, want once for a version checked", *calls)
	}
	cowork := loadVerifications()["cowork:9.9.9"]
	if _, failing := cowork.Failures[coworkReplyShape]; !failing {
		t.Errorf("the failing check is not recorded under Cowork's key: %+v", cowork)
	}
	if _, kept := loadVerifications()["9.9.9"].Failures["B20"]; !kept {
		t.Error("Cowork's record replaced Claude Code's for the same version")
	}
	// the failure withdraws the trust the delivery rule gives, for this version only
	if (coworkHost{}).MayTrustWithoutEvidence("cowork:9.9.9") {
		t.Error("a Cowork turn is trusted on a version whose reply-shape check is recorded as failing")
	}
	if !(coworkHost{}).MayTrustWithoutEvidence("cowork:9.9.8") {
		t.Error("a version with no failure recorded is not trusted")
	}
}

func TestACoworkProbeThatCouldNotRunIsRecordedAndNotRepeatedWithinTheWindow(t *testing.T) {
	coworkHome(t)
	calls := standInForCoworkChecks(t, nil, errors.New("probe session failed: not logged in\nlong output"))
	for i := 0; i < 3; i++ {
		EnsureCoworkVerified("room")
	}
	if *calls != 1 {
		t.Errorf("the checks ran %d times across three rooms, want once", *calls)
	}
	if why := loadVerifications()["cowork:9.9.9"].Failures[probeCouldNotRun]; why != "probe session failed: not logged in" {
		t.Errorf("recorded as %q", why)
	}
	saveVerification(verification{Version: "cowork:9.9.9", When: time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
		Failures: map[string]string{probeCouldNotRun: "not logged in"}})
	EnsureCoworkVerified("room")
	if *calls != 2 {
		t.Errorf("an attempt from two hours ago was not tried again: %d runs", *calls)
	}
}

func TestTheCoworkPreflightIsSkippedWhenOffOrWithNoClaudeCode(t *testing.T) {
	coworkHome(t)
	calls := standInForCoworkChecks(t, nil, nil)
	t.Setenv("COGMER_PREFLIGHT", "off")
	EnsureCoworkVerified("room")
	t.Setenv("COGMER_PREFLIGHT", "")
	t.Setenv("PATH", t.TempDir())
	EnsureCoworkVerified("room")
	if *calls != 0 {
		t.Errorf("the checks ran %d times with the preflight off or no Claude Code", *calls)
	}
}

func TestTheCoworkHostRunsItsOwnPreflightAndDoctor(t *testing.T) {
	coworkHome(t)
	calls := standInForCoworkChecks(t, []Result{{B: Behavior{ID: coworkReplyShape}}}, nil)
	(coworkHost{}).Preflight("room")
	if *calls != 1 {
		t.Errorf("forming a room from Cowork ran the checks %d times", *calls)
	}
	ok, err := (coworkHost{}).Doctor(false)
	if err != nil || !ok {
		t.Errorf("doctor with a passing check gave %v %v", ok, err)
	}
	if _, err := (coworkHost{}).Doctor(true); !errors.Is(err, errDeepUnsupported) {
		t.Errorf("--deep for Cowork gave %v", err)
	}
	if got := (coworkHost{}).Version(); got != "cowork:9.9.9" {
		t.Errorf("version %q: the key must differ from Claude Code's", got)
	}
	if len((coworkHost{}).Behaviors()) != len(CoworkBehaviors) || len(CoworkBehaviors) != 3 {
		t.Errorf("the Cowork behaviors are %d", len(CoworkBehaviors))
	}
}

func TestTheProbeServerAnswersAsTheCheckNeeds(t *testing.T) {
	text := string((coworkHost{}).PromptHookOutput("JSONFORM-1"))
	if !strings.Contains(text, `"additionalContext":"JSONFORM-1"`) {
		t.Errorf("the JSON form is %q", text)
	}
}
