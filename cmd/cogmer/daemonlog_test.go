package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTheDaemonLogStaysUnderItsCapAcrossAWriteLoop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.log")
	// Opened for appending, as the shell redirect and startDaemon open it.
	w, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	const max, line = 10_000, 100
	rotations := 0
	for i := 0; i < 1000; i++ {
		if _, err := w.WriteString(strings.Repeat("x", line-1) + "\n"); err != nil {
			t.Fatal(err)
		}
		if i%10 == 9 {
			did, err := boundLog(path, max)
			if err != nil {
				t.Fatal(err)
			}
			if did {
				rotations++
			}
		}
		if info, _ := os.Stat(path); info.Size() > max+10*line {
			t.Fatalf("after %d lines the log is %d bytes, over the cap of %d", i+1, info.Size(), max)
		}
	}
	if rotations < 5 {
		t.Fatalf("the log was bounded %d times in 100000 bytes of output", rotations)
	}
}

func TestTheOlderCopyIsReplacedNotAccumulated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "daemon.log")
	for _, gen := range []string{"first", "second", "third"} {
		f, _ := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		f.WriteString(gen + strings.Repeat("y", 200) + "\n")
		f.Close()
		if did, err := boundLog(path, 100); err != nil || !did {
			t.Fatalf("boundLog = %v, %v", did, err)
		}
	}
	old, err := os.ReadFile(path + ".1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(old), "third") || strings.Contains(string(old), "first") || strings.Contains(string(old), "second") {
		t.Fatalf("the older copy holds %q", old[:20])
	}
	if info, _ := os.Stat(path); info.Size() != 0 {
		t.Fatalf("the log holds %d bytes after being bounded", info.Size())
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Fatalf("the directory holds %d files, want daemon.log and daemon.log.1", len(entries))
	}
}

func TestTheBoundedLogKeepsItsModeAndWritersContinueIntoIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.log")
	w, _ := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	defer w.Close()
	w.WriteString(strings.Repeat("z", 500))
	if did, err := boundLog(path, 100); err != nil || !did {
		t.Fatalf("boundLog = %v, %v", did, err)
	}
	w.WriteString("after\n")
	got, _ := os.ReadFile(path)
	if string(got) != "after\n" {
		t.Fatalf("the open writer's next line landed as %q", got)
	}
	for _, p := range []string{path, path + ".1"} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if modesMeanSomething() && info.Mode().Perm() != 0o600 {
			t.Fatalf("%s has mode %v, want 0600", p, info.Mode().Perm())
		}
	}
}

func TestALogUnderTheCapIsLeftAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.log")
	_ = os.WriteFile(path, []byte("small\n"), 0o600)
	if did, err := boundLog(path, 100); err != nil || did {
		t.Fatalf("boundLog = %v, %v", did, err)
	}
	if _, err := os.Stat(path + ".1"); err == nil {
		t.Fatal("an older copy was made of a log under the cap")
	}
	if did, err := boundLog(path+".absent", 100); err != nil || did {
		t.Fatalf("boundLog of a missing file = %v, %v", did, err)
	}
}

func TestTheRelayLibrarysRoutineLinesAreDroppedAndCogmersAreKept(t *testing.T) {
	var got []string
	logf := relayLibraryLogf(func(_ string, a ...any) { got = append(got, a[0].(string)) })
	for _, l := range []string{"netstack: x", "wg: peer", "[v1] magicsock thing", "magicsock: derp", "netcheck: report"} {
		logf("%s", l)
	}
	logf("%s: %d", "wg", 3)
	if len(got) != 0 {
		t.Fatalf("kept %v", got)
	}
	logf("relay %s refused the connection", "r1")
	logf("tailcat: %v", "dial failed")
	if len(got) != 2 {
		t.Fatalf("kept %v", got)
	}
}

func TestTheRelayClientsAreGivenTheFilter(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	c := newRelayClient("x")
	if c.Logf == nil {
		t.Fatal("the client logs through the library's default")
	}
	c.Logf("netstack: noise")
	c.Logf("a line cogmer wants")
	if strings.Contains(buf.String(), "noise") || !strings.Contains(buf.String(), "a line cogmer wants") {
		t.Fatalf("log holds %q", buf.String())
	}
}
