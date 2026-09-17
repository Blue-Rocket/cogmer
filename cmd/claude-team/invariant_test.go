package main

import (
	"os"
	"strings"
	"testing"
)

// §3.7: a remote peer must never initiate execution on a receiving machine.
//
// Claude Code edits files and runs commands, and peer identity is not verified,
// so a path from received data to execution is arbitrary execution authorised by
// whoever can reach the port.
//
// The guard is structural: the files that handle peer traffic must not be able to
// start a process at all. Checking imports rather than call graphs is crude, and
// it fails loudly the moment someone adds the capability to the wrong file --
// which is the point at which a reviewer should be asked to justify it.
func TestPeerFacingCodeCannotExecute(t *testing.T) {
	for _, file := range []string{"sync.go", "daemon.go", "store.go", "transcript.go"} {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		head := string(src)
		if i := strings.Index(head, "\nfunc "); i > 0 {
			head = head[:i] // imports only
		}
		for _, forbidden := range []string{`"os/exec"`, `"syscall"`} {
			if strings.Contains(head, forbidden) {
				t.Errorf("%s imports %s — peer traffic reaches this file, and §3.7 forbids "+
					"a remote event initiating local execution", file, forbidden)
			}
		}
	}
}

// The inverse: execution that does exist must be reachable only from a local
// command, never from anything that handles received data.
func TestExecutionLivesOnlyInTheProbe(t *testing.T) {
	src, err := os.ReadFile("probe.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), `exec.LookPath("claude")`) {
		t.Skip("probe no longer starts claude; this guard needs rewriting")
	}
	// probe.go is reached from runDoctor (a typed command) and EnsureVerified
	// (daemon startup). Neither is reachable from a peer.
	for _, file := range []string{"sync.go", "daemon.go"} {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, call := range []string{"RunProbe(", "EnsureVerified(", "runDoctor("} {
			if strings.Contains(string(b), call) {
				t.Errorf("%s calls %s — that puts execution on a path that handles peer traffic", file, call)
			}
		}
	}
}
