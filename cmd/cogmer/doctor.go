package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ClaudeVersion reports the installed Claude Code version. The verification
// cache is keyed on it, because that -- not the room -- is what determines
// whether the relied-on behaviors still hold.
func ClaudeVersion() string {
	out, err := exec.Command("claude", "--version").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(strings.Fields(string(out))[0])
}

type verification struct {
	Version  string            `json:"version"`
	When     string            `json:"when"`
	Deep     bool              `json:"deep"`
	Failures map[string]string `json:"failures,omitempty"`
}

func verifyPath() string { return filepath.Join(homeDir(), "verified.json") }

func loadVerifications() map[string]verification {
	m := map[string]verification{}
	if b, err := os.ReadFile(verifyPath()); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	return m
}

func saveVerification(v verification) {
	m := loadVerifications()
	m[v.Version] = v
	if err := os.MkdirAll(homeDir(), 0o755); err != nil {
		return
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(verifyPath(), b, 0o600)
}

// BehaviorKnownBroken reports whether a behavior is recorded as FAILING for this
// Claude Code version.
//
// This gates the delivery fallback. "No evidence" must not be read as "the
// format changed" -- it is equally consistent with the injection never having
// arrived, which is the failure the evidence-derived design exists to catch.
// Only a check that actually failed justifies committing on trust.
func BehaviorKnownBroken(version, id string) bool {
	v, ok := loadVerifications()[version]
	if !ok {
		return false
	}
	_, failed := v.Failures[id]
	return failed
}

// Result is one behavior's outcome.
type Result struct {
	B   Behavior
	Err error
}

// RunChecks runs every behavior at or below the requested tier.
func RunChecks(deep bool) ([]Result, error) {
	maxTier := TierSession
	if deep {
		maxTier = TierCompaction
	}

	var results []Result
	// Offline checks need no session, so run them even if the probe fails.
	for _, b := range Behaviors {
		if b.Tier == TierOffline {
			results = append(results, Result{B: b, Err: b.Check(nil)})
		}
	}

	probe, err := RunProbe(deep)
	if err != nil {
		return results, err
	}
	for _, b := range Behaviors {
		if b.Tier == TierOffline || b.Tier > maxTier {
			continue
		}
		results = append(results, Result{B: b, Err: b.Check(probe)})
	}
	return results, nil
}

func runDoctor(deep bool) {
	version := ClaudeVersion()
	tier := "session"
	if deep {
		tier = "session + compaction"
	}
	fmt.Printf("cogmer doctor — Claude Code %s (%s checks)\n\n", version, tier)
	if deep {
		fmt.Println("Driving a real session and a compaction; this takes a minute.")
	} else {
		fmt.Println("Driving one real Claude turn.")
	}
	fmt.Println()

	results, err := RunChecks(deep)
	if err != nil {
		fmt.Fprintf(os.Stderr, "probe failed: %v\n", err)
	}

	failures := map[string]string{}
	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("  FAIL  %s  %s\n        %v\n        relied on for: %s\n\n", r.B.ID, r.B.Title, r.Err, r.B.Reliance)
			failures[r.B.ID] = r.Err.Error()
		} else {
			fmt.Printf("  ok    %s  %s\n", r.B.ID, r.B.Title)
		}
	}

	fmt.Printf("\n%d checked, %d failed\n", len(results), len(failures))
	if err == nil {
		saveVerification(verification{Version: version, When: time.Now().UTC().Format(time.RFC3339), Deep: deep, Failures: failures})
	}
	if len(failures) > 0 {
		fmt.Println("\nA failure means Claude Code changed under an assumption this project depends on.")
		fmt.Println("Read the failing behaviour in docs/relied-on-behaviors.md before trusting the room.")
		os.Exit(1)
	}
}

// EnsureVerified runs the session-tier checks the first time a room is formed on
// an unverified Claude Code version.
//
// Keyed on version, not room: forming a tenth room on a version already verified
// costs nothing. Set COGMER_PREFLIGHT=off to skip, which is what CI and
// scripted use should do -- the check spends a real turn of the user's quota.
func EnsureVerified(room string) {
	if strings.EqualFold(os.Getenv("COGMER_PREFLIGHT"), "off") {
		return
	}
	version := ClaudeVersion()
	if v, ok := loadVerifications()[version]; ok {
		if len(v.Failures) > 0 {
			fmt.Fprintf(os.Stderr, "cogmer: WARNING — Claude Code %s previously failed %d behavior check(s): %s\n",
				version, len(v.Failures), strings.Join(keysOfStr(v.Failures), ", "))
			fmt.Fprintln(os.Stderr, "cogmer: run `cogmer doctor` for detail.")
		}
		return
	}

	fmt.Fprintf(os.Stderr, "cogmer: new room %q on unverified Claude Code %s — running behavior checks (one Claude turn)...\n", room, version)
	results, err := RunChecks(false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cogmer: preflight could not run (%v); collaboration continues unverified\n", err)
		return
	}
	failures := map[string]string{}
	for _, r := range results {
		if r.Err != nil {
			failures[r.B.ID] = r.Err.Error()
		}
	}
	saveVerification(verification{Version: version, When: time.Now().UTC().Format(time.RFC3339), Failures: failures})

	if len(failures) == 0 {
		fmt.Fprintf(os.Stderr, "cogmer: all %d behavior checks passed on %s\n", len(results), version)
		return
	}
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(os.Stderr, "cogmer: FAILED %s (%s): %v\n", r.B.ID, r.B.Title, r.Err)
		}
	}
	fmt.Fprintln(os.Stderr, "cogmer: the room still works, but the failed assumptions above no longer hold. Run `cogmer doctor`.")
}

func keysOfStr(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
