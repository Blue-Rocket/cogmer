package main

import (
	"os"
	"strings"
	"testing"
)

// §3.7 forbids a remote event causing inference in an INTERACTIVE session. This
// test enforces something stricter: that peer-handling code cannot start a process
// at all.
//
// The stricter rule is deliberate and is not what §3.7 requires. Whether a peer
// event may cause a separate run is explicitly left open -- addressing another
// developer's Claude would need it. But nothing needs it yet, and the questions it
// raises (whose subscription, what tool access, what was agreed to) have no answers
// yet either.
//
// So the guard stands until those are answered. Checking imports rather than call
// graphs is crude, and it fails the moment the capability lands in a file that
// handles peer traffic -- which is when someone should be asked to justify it,
// rather than discovering it later in a review of something else.
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
