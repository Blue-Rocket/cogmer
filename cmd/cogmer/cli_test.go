package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// plugin/cli.sh is what every slash command runs. Claude Code abandons a command
// whose `!` line exits non-zero, so no model turn runs and the explanation never
// reaches anybody, and these tests hold the script to exiting 0 with everything on
// stdout, and to answering at once when there is no binary (D-191).

// cliHarness copies cli.sh and its shared functions into a plugin root of its own,
// with an installer that only records that it was started, and a state directory
// holding nothing.
func cliHarness(t *testing.T) (root, home string) {
	t.Helper()
	root, home = t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "hooks-handlers"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"cli.sh", "hooks-handlers/common.sh"} {
		src, err := os.ReadFile(filepath.Join("../../plugin", f))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, f), src, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	stub := "#!/usr/bin/env bash\necho started > \"$COGMER_HOME/started\"\n"
	if err := os.WriteFile(filepath.Join(root, "hooks-handlers/install.sh"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	return root, home
}

// runCLI runs the script with a path that holds no cogmer, and returns what it
// printed and its exit status.
func runCLI(t *testing.T, root, home string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{filepath.Join(root, "cli.sh")}, args...)...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + home, "COGMER_HOME=" + home, "CLAUDE_PLUGIN_ROOT=" + root}
	out, err := cmd.Output()
	code := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal(err)
		}
		code = ee.ExitCode()
	}
	return string(out), code
}

// flat joins an explanation's lines, so a phrase can be asserted wherever the
// script happens to wrap it.
func flat(s string) string { return strings.Join(strings.Fields(s), " ") }

func writeInstallState(t *testing.T, home, state string, age time.Duration, detail string) {
	t.Helper()
	line := fmt.Sprintf("%s\t%d\t%s\n", state, time.Now().Add(-age).Unix(), detail)
	if err := os.WriteFile(filepath.Join(home, "install-state"), []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
}

// waitFor polls for a file, so a test does not end while a detached start is still
// writing into the directory the test is about to remove.
func waitFor(t *testing.T, path string) bool {
	t.Helper()
	for i := 0; i < 100; i++ {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func TestACommandWithABinaryPrintsEverythingAndExitsZero(t *testing.T) {
	root, home := cliHarness(t)
	if err := os.MkdirAll(filepath.Join(home, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	bin := "#!/usr/bin/env bash\necho \"said on stdout\"\necho \"said on stderr\" >&2\nexit 3\n"
	if err := os.WriteFile(filepath.Join(home, "bin", "cogmer"), []byte(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	out, code := runCLI(t, root, home, "guests")
	if code != 0 {
		t.Errorf("exit status %d; Claude Code abandons a command whose line exits non-zero, so the explanation is never read", code)
	}
	for _, want := range []string{"said on stdout", "said on stderr"} {
		if !strings.Contains(flat(out), want) {
			t.Errorf("output lacks %q; a failure the binary reports on stderr must reach the model: %q", want, out)
		}
	}
}

func TestACommandWithNoBinaryStartsTheDownloadAndAnswersAtOnce(t *testing.T) {
	root, home := cliHarness(t)
	start := time.Now()
	out, code := runCLI(t, root, home, "whoami")
	if code != 0 {
		t.Errorf("exit status %d", code)
	}
	if time.Since(start) > 5*time.Second {
		t.Errorf("the command took %s; it must answer at once and never wait for the download", time.Since(start))
	}
	for _, want := range []string{"has started downloading", "15 to 20 seconds", "run the same command again"} {
		if !strings.Contains(flat(out), want) {
			t.Errorf("output lacks %q: %q", want, out)
		}
	}
	if !waitFor(t, filepath.Join(home, "started")) {
		t.Error("the installer was never started, so a person who installs mid-session never gets a binary")
	}
}

func TestACommandWhileAnInstallRunsSaysToWaitAndStartsNothing(t *testing.T) {
	root, home := cliHarness(t)
	writeInstallState(t, home, "installing", 4*time.Second, "fetching v0.7.4")
	out, code := runCLI(t, root, home, "whoami")
	if code != 0 {
		t.Errorf("exit status %d", code)
	}
	for _, want := range []string{"still downloading", "Nothing is wrong", "15 to 20 seconds", "run the same command again"} {
		if !strings.Contains(flat(out), want) {
			t.Errorf("output lacks %q: %q", want, out)
		}
	}
	time.Sleep(300 * time.Millisecond)
	if _, err := os.Stat(filepath.Join(home, "started")); err == nil {
		t.Error("a second download was started while one was running")
	}
}

func TestACommandAfterAFailedInstallSaysWhyAndHowToTryAgain(t *testing.T) {
	root, home := cliHarness(t)
	writeInstallState(t, home, "failed", 2*time.Minute, "no verified download and no usable Go toolchain")
	out, code := runCLI(t, root, home, "whoami")
	if code != 0 {
		t.Errorf("exit status %d", code)
	}
	for _, want := range []string{"could not be installed", "no verified download", "remove ~/.cogmer/install-state", "run the same command again"} {
		if !strings.Contains(flat(out), want) {
			t.Errorf("output lacks %q: %q", want, out)
		}
	}
	time.Sleep(300 * time.Millisecond)
	if _, err := os.Stat(filepath.Join(home, "started")); err == nil {
		t.Error("an install was started inside the cooldown after a failure")
	}
}

func TestACommandAfterAStalledInstallClearsTheLockAndStartsAgain(t *testing.T) {
	root, home := cliHarness(t)
	writeInstallState(t, home, "installing", 10*time.Minute, "fetching v0.7.4")
	lock := filepath.Join(home, ".install.lock")
	if err := os.Mkdir(lock, 0o755); err != nil {
		t.Fatal(err)
	}
	out, code := runCLI(t, root, home, "whoami")
	if code != 0 {
		t.Errorf("exit status %d", code)
	}
	for _, want := range []string{"never finished", "cleared what it left behind", "run the same command again"} {
		if !strings.Contains(flat(out), want) {
			t.Errorf("output lacks %q: %q", want, out)
		}
	}
	if _, err := os.Stat(lock); err == nil {
		t.Error("the lock a dead install left behind is still there, so every later install finds it and leaves")
	}
	if !waitFor(t, filepath.Join(home, "started")) {
		t.Error("the download was not started again")
	}
}

// A restart stops the daemon and starts one, and says whether it is answering. The
// first test uses a stand-in that records how it was called, so the order and the
// flags are checked without a daemon. The second runs real builds.

func TestARestartStopsThenStartsAndSaysWhenNothingAnswers(t *testing.T) {
	root, home := cliHarness(t)
	if err := os.MkdirAll(filepath.Join(home, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(home, "calls")
	bin := "#!/usr/bin/env bash\necho \"$@\" >> \"" + log + "\"\n"
	if err := os.WriteFile(filepath.Join(home, "bin", "cogmer"), []byte(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	// What the daemon would have recorded when it could not bind.
	state := "blocked\t1\tpeer sync cannot be served: 127.0.0.1:4783 is held by nginx, pid 7, which is not a cogmer daemon\n"
	if err := os.WriteFile(filepath.Join(home, "daemon-state"), []byte(state), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", filepath.Join(root, "cli.sh"), "restart")
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + home, "COGMER_HOME=" + home, "CLAUDE_PLUGIN_ROOT=" + root,
		"COGMER_ADDR=127.0.0.1:1", "COGMER_RESTART_WAIT=1"}
	raw, err := cmd.Output()
	if err != nil {
		t.Fatalf("restart exited non-zero, so Claude Code abandons the command and nothing is explained: %v", err)
	}
	out := flat(string(raw))

	var calls []string
	for i := 0; i < 100; i++ {
		b, _ := os.ReadFile(log)
		calls = strings.Fields(strings.ReplaceAll(string(b), "\n", " ; "))
		if strings.Contains(string(b), "daemon") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	joined := strings.Join(calls, " ")
	if !strings.HasPrefix(joined, "stop --for-restart") {
		t.Errorf("the daemon was not stopped first, with the flag that keeps stop from saying nothing runs: %q", joined)
	}
	if !strings.Contains(joined, "daemon") {
		t.Errorf("no daemon was started after the stop: %q", joined)
	}
	for _, want := range []string{"was started and is not answering", "It recorded why", "held by nginx", "daemon.log"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q: %q", want, out)
		}
	}
}

func TestARestartReplacesARunningDaemonWithOneThatAnswers(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	needsToFindADaemon(t)
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("no curl on this machine")
	}
	dir := t.TempDir()
	bin := buildVersion(t, dir, "0.0.1")
	hooks, peer := freeAddr(t), freeAddr(t)
	env := append(os.Environ(), "COGMER_HOME="+filepath.Join(dir, "home"), "COGMER_ADDR="+hooks,
		"COGMER_PEER_ADDR="+peer, "COGMER_TAILCAT=off", "COGMER_PREFLIGHT=off", "COGMER_PEERS=",
		"COGMER_BIN="+bin, "CLAUDE_PLUGIN_ROOT="+filepath.Join("..", "..", "plugin"))

	first := exec.Command(bin, "daemon")
	first.Env = env
	if err := first.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan error, 1)
	go func() { exited <- first.Wait() }()
	defer func() { _ = first.Process.Kill() }()
	waitForVersion(t, hooks, "0.0.1")
	// Whatever the restart leaves running is stopped afterwards.
	t.Cleanup(func() {
		stop := exec.Command(bin, "stop")
		stop.Env = env
		_ = stop.Run()
	})

	cmd := exec.Command("bash", filepath.Join("..", "..", "plugin", "cli.sh"), "restart")
	cmd.Env = env
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("restart: %v\n%s", err, raw)
	}
	out := string(raw)
	select {
	case <-exited:
	case <-time.After(10 * time.Second):
		t.Fatal("the old daemon was still running after the restart")
	}
	if !strings.Contains(flat(out), "was restarted and is answering") {
		t.Errorf("the restart does not say the daemon answers:\n%s", out)
	}
	if strings.Contains(flat(out), "A new Claude Code session starts one") {
		t.Errorf("the restart says no daemon runs, which stop says and a restart contradicts:\n%s", out)
	}
	if _, ok := daemonVersionAt(hooks); !ok {
		t.Error("no daemon answers on the hooks address after the restart")
	}
}
