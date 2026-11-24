package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Claude Code 2.1.331 prints an error for a `!` line that exits 1 and a warning in the days
// before a sign-in ends, and neither is a model turn. A probe that counted any output as a turn
// failed B24 for every user on that version and never ran B33 and B34.
func TestOutputThatIsNotTheReplyIsNotATurn(t *testing.T) {
	err := "Shell command failed for pattern \"!`sh -c 'echo before; exit 1'`\": [stderr]\nbefore"
	warn := "Warning: this sign-in ends between 2026-11-25 and 2026-11-28 (in 1 day at the earliest)."
	for name, c := range map[string]struct{ out, reply, want string }{
		"an error for a failed line":     {err, failingReply, ""},
		"a sign-in warning":              {warn, controlReply, ""},
		"nothing":                        {"", controlReply, ""},
		"the reply":                      {"TURN-RAN", controlReply, "TURN-RAN"},
		"the reply after a warning":      {warn + "\nTURN-RAN", controlReply, warn + "\nTURN-RAN"},
		"the failing reply, a real turn": {"TURN-RAN-AFTER-FAILURE", failingReply, "TURN-RAN-AFTER-FAILURE"},
	} {
		if got := turnOutput(c.out, c.reply); got != c.want {
			t.Errorf("%s: turnOutput = %q, want %q", name, got, c.want)
		}
	}
}

// fakeClaude writes a program that answers as Claude Code 2.1.331 does: the failing command
// prints the error and no reply, and the control prints a warning and its reply.
func fakeClaude(t *testing.T, failingRunsATurn bool) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake Claude Code is a shell script")
	}
	failing := `echo 'Shell command failed for pattern "!sh": [stderr]'; echo before`
	if failingRunsATurn {
		failing = `echo TURN-RAN-AFTER-FAILURE`
	}
	script := "#!/bin/sh\n" +
		"case \"$*\" in\n" +
		"*probe-fail*) " + failing + " ;;\n" +
		"*probe-ok*) echo 'Warning: this sign-in ends between 2026-11-25 and 2026-11-28'; echo TURN-RAN ;;\n" +
		"esac\n"
	path := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTheCommandProbeRunsTheControlWhenAFailedLineOnlyPrintsAnError(t *testing.T) {
	failing, control, _, _ := runCommandProbe(fakeClaude(t, false), t.TempDir())
	if failing != "" {
		t.Errorf("a failed line that printed an error and no reply was taken for a turn: %q", failing)
	}
	if !strings.Contains(control, "TURN-RAN") {
		t.Errorf("the control did not run after a failed line that ran no turn: %q", control)
	}
}

func TestTheCommandProbeStillSeesATurnAfterAFailedLine(t *testing.T) {
	failing, control, _, _ := runCommandProbe(fakeClaude(t, true), t.TempDir())
	if !strings.Contains(failing, "TURN-RAN-AFTER-FAILURE") {
		t.Errorf("a turn after a failed line was not seen: %q", failing)
	}
	if control != "" {
		t.Errorf("the control ran after a turn that proves B24 changed: %q", control)
	}
}
