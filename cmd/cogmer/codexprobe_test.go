package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"
)

// The probe runs one real Codex turn. A machine where it cannot run would run it
// again at every room command, with a model turn and a wait of up to two minutes each
// time, so an attempt that could not run is recorded and left alone for an hour. A
// result about Codex, passing or failing, is never repeated by itself.

func codexProbeHome(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the stand-in for Codex is a shell script")
	}
	t.Setenv("COGMER_HOME", t.TempDir())
	t.Setenv("COGMER_PREFLIGHT", "")
	stand := filepath.Join(t.TempDir(), "codex")
	if err := os.WriteFile(stand, []byte("#!/bin/sh\necho codex-cli 9.9.9\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COGMER_CODEX_BIN", stand)
}

func standInForTheProbe(t *testing.T, results []Result, err error) *int {
	t.Helper()
	calls := new(int)
	old := runCodexChecks
	runCodexChecks = func() ([]Result, error) { *calls++; return results, err }
	t.Cleanup(func() { runCodexChecks = old })
	return calls
}

func TestAProbeThatCouldNotRunIsNotRepeatedWithinTheRetryWindow(t *testing.T) {
	codexProbeHome(t)
	calls := standInForTheProbe(t, nil, errors.New("codex exec: not signed in\nlong output follows"))

	EnsureCodexVerified()
	EnsureCodexVerified()
	EnsureCodexVerified()
	if *calls != 1 {
		t.Errorf("the probe ran %d times across three room commands, want once", *calls)
	}
	v := loadVerifications()["codex:codex-cli 9.9.9"]
	if why := v.Failures[codexProbeCouldNotRun]; why != "codex exec: not signed in" {
		t.Errorf("the attempt that could not run is recorded as %+v, want its first line", v.Failures)
	}
}

func TestAProbeThatCouldNotRunIsTriedAgainAfterTheWindow(t *testing.T) {
	codexProbeHome(t)
	saveVerification(verification{Version: "codex:codex-cli 9.9.9",
		When:     time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
		Failures: map[string]string{codexProbeCouldNotRun: "not signed in"}})
	calls := standInForTheProbe(t, []Result{{B: Behavior{ID: "B25"}}}, nil)

	EnsureCodexVerified()
	if *calls != 1 {
		t.Fatalf("an attempt from two hours ago was not tried again: %d runs", *calls)
	}
	if v := loadVerifications()["codex:codex-cli 9.9.9"]; len(v.Failures) != 0 {
		t.Errorf("a passing probe left failures recorded: %+v", v.Failures)
	}
}

func TestAResultAboutCodexIsNeverRepeatedByItself(t *testing.T) {
	codexProbeHome(t)
	old := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
	for name, failures := range map[string]map[string]string{
		"a pass":                  nil,
		"a failed behavior":       {"B27": "no matching evidence"},
		"a failure and a non-run": {"B27": "x", codexProbeCouldNotRun: "y"},
	} {
		saveVerification(verification{Version: "codex:codex-cli 9.9.9", When: old, Failures: failures})
		calls := standInForTheProbe(t, nil, errors.New("must not run"))
		EnsureCodexVerified()
		if *calls != 0 {
			t.Errorf("%s from two days ago was run again", name)
		}
	}
}

func TestPreflightOffNeverRunsTheProbe(t *testing.T) {
	codexProbeHome(t)
	t.Setenv("COGMER_PREFLIGHT", "off")
	calls := standInForTheProbe(t, nil, errors.New("must not run"))
	EnsureCodexVerified()
	if *calls != 0 {
		t.Error("COGMER_PREFLIGHT=off ran the probe")
	}
}

// The real cogmer plugin is enabled in the user's Codex configuration, so a probe that
// loaded it would fire the real hooks inside the probe session.
func TestTheProbeSessionDoesNotLoadTheUsersCodexConfiguration(t *testing.T) {
	args := codexProbeArgs("/bin/cogmer", "/tmp/probe")
	for _, want := range []string{"--ignore-user-config", "--dangerously-bypass-hook-trust"} {
		if !slices.Contains(args, want) {
			t.Errorf("the probe is started without %s: %v", want, args)
		}
	}
	// The transcript is read after the turn, so the session has to be kept.
	if slices.Contains(args, "--ephemeral") {
		t.Error("the probe session is ephemeral, so its transcript cannot be read afterwards")
	}
	hooks := 0
	for _, a := range args {
		if len(a) > 6 && a[:6] == "hooks." {
			hooks++
		}
	}
	if hooks != 3 {
		t.Errorf("%d hooks are given to the probe, want the three it records", hooks)
	}
}
