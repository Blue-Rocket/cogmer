package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeBrowsers puts a program of every name the opener runs at the front of PATH. Each
// writes a marker file, so a test can tell whether a page would have been opened on the
// machine that runs the tests.
func fakeBrowsers(t *testing.T) (pathValue, marker string) {
	t.Helper()
	dir := t.TempDir()
	marker = filepath.Join(dir, "opened")
	script := "#!/bin/sh\necho \"$@\" >> " + marker + "\n"
	for _, name := range []string{"open", "xdg-open", "rundll32"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir + ":/usr/bin:/bin", marker
}

// A test has no person at the screen. A page opened from one is a stray tab in the
// maintainer's browser that points at a daemon which has already exited, and a full run
// of the suite opened several.
func TestNoTestOpensABrowser(t *testing.T) {
	path, marker := fakeBrowsers(t)
	t.Setenv("PATH", path)
	if err := openInBrowser("http://127.0.0.1:1/identity"); err == nil || !strings.Contains(err.Error(), "could not open a browser") {
		t.Errorf("a test process opened a page, or reported no failure: %v", err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("a test process ran a browser")
	}
}

// A command that needs the identity page opens it only when a daemon is there to serve it,
// so the binary runs one.
func TestABinaryBuiltByATestOpensNoBrowser(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	dir := t.TempDir()
	bin := buildVersion(t, dir, "no-browser-test")
	path, marker := fakeBrowsers(t)
	hooks, peer := freeAddr(t), freeAddr(t)
	env := []string{"PATH=" + path, "HOME=" + dir, "COGMER_HOME=" + filepath.Join(dir, "home"),
		"COGMER_ADDR=" + hooks, "COGMER_PEER_ADDR=" + peer,
		"COGMER_TAILCAT=off", "COGMER_PREFLIGHT=off", "COGMER_PEERS=", "CLAUDE_CODE_SESSION_ID=s"}
	daemon := exec.Command(bin, "daemon")
	daemon.Env = env
	if err := daemon.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = daemon.Process.Kill(); _ = daemon.Wait() }()
	waitForVersion(t, hooks, "no-browser-test")

	cmd := exec.Command(bin, "machines")
	cmd.Env = env
	out, _ := cmd.CombinedOutput()
	if _, err := os.Stat(marker); err == nil {
		t.Errorf("a binary built for a test ran a browser:\n%s", out)
	}
	if !strings.Contains(string(out), "needs your cogmer identity") {
		t.Errorf("the command did not say what the page is for:\n%s", out)
	}
}
