//go:build !windows

package main

import (
	"bufio"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestShouldReplace(t *testing.T) {
	cases := []struct {
		running, mine string
		want          bool
	}{
		{"0.7.2", "0.7.3", true},
		{"0.7.3", "0.7.2", false}, // an older copy of the plugin does not move a machine back (D-215)
		{"", "0.7.3", true},       // a daemon from before /healthz carried a version
		{"0.7.3", "0.7.3", false},
		{"dev", "0.7.3", false}, // a maintainer's hand-started build
		{"0.7.3", "dev", false},
		{"0.9.1", "0.10.0", true}, // numbers, not letters: 0.10.0 is the newer
		{"0.10.0", "0.9.1", false},
		{"0.10.2", "0.7.4", false},
		{"0.7", "0.7.1", true}, // a missing number is zero
		{"0.7.0", "0.7", false},
		{"0.7.4-rc1", "0.7.4", true}, // no order to go by, so a difference replaces
		{"0.7.4", "garbage", true},
	}
	for _, c := range cases {
		if got := shouldReplace(c.running, c.mine); got != c.want {
			t.Errorf("shouldReplace(%q, %q) = %v, want %v", c.running, c.mine, got, c.want)
		}
	}
}

// The helper a test runs as a separate process: something listening on a port
// that is not a cogmer daemon, because its executable is cogmer.test.
func TestHelperListener(t *testing.T) {
	addr := os.Getenv("COGMER_TEST_LISTEN")
	if addr == "" {
		t.Skip("run as a helper process only")
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		os.Exit(3)
	}
	os.Stdout.WriteString("listening\n")
	for {
		c, err := l.Accept()
		if err != nil {
			os.Exit(0)
		}
		c.Close()
	}
}

func TestStopDeclinesAProcessNotNamedCogmer(t *testing.T) {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("no lsof on this machine")
	}
	addr := freeAddr(t)
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperListener$")
	cmd.Env = append(os.Environ(), "COGMER_TEST_LISTEN="+addr)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	if line, _ := bufio.NewReader(out).ReadString('\n'); strings.TrimSpace(line) != "listening" {
		t.Fatalf("helper did not start listening: %q", line)
	}

	outcomes, err := stopDaemonsAt([]string{addr})
	if err != nil {
		t.Fatalf("stopDaemonsAt: %v", err)
	}
	if len(outcomes) != 1 || outcomes[0].PID != cmd.Process.Pid {
		t.Fatalf("expected one outcome for pid %d, got %+v", cmd.Process.Pid, outcomes)
	}
	if o := outcomes[0]; o.Stopped || o.Err != nil || o.Name == "cogmer" {
		t.Fatalf("stop acted on a process that is not a cogmer daemon: %+v", o)
	}
	if conn, err := net.Dial("tcp", addr); err != nil {
		t.Fatalf("the declined process stopped listening: %v", err)
	} else {
		conn.Close()
	}
}

// After an update, the daemon the hook starts replaces one of another version
// that is still serving, and `stop` then ends it. Two real builds, two real
// processes, with the overlay off so nothing leaves the machine.
func TestADaemonOfAnotherVersionIsReplaced(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary twice")
	}
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("no lsof on this machine")
	}
	dir := t.TempDir()
	old := buildVersion(t, dir, "0.0.1")
	cur := buildVersion(t, dir, "0.0.2")

	hooks, peer := freeAddr(t), freeAddr(t)
	env := append(os.Environ(),
		"COGMER_HOME="+filepath.Join(dir, "home"),
		"COGMER_ADDR="+hooks, "COGMER_PEER_ADDR="+peer,
		"COGMER_TAILCAT=off", "COGMER_PREFLIGHT=off", "COGMER_PEERS=")

	first := exec.Command(old, "daemon")
	first.Env = env
	if err := first.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan error, 1)
	go func() { exited <- first.Wait() }()
	defer func() { _ = first.Process.Kill() }()
	waitForVersion(t, hooks, "0.0.1")

	second := exec.Command(cur, "daemon")
	second.Env = env
	if err := second.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = second.Process.Kill(); _ = second.Wait() }()
	waitForVersion(t, hooks, "0.0.2")

	select {
	case <-exited:
	case <-time.After(10 * time.Second):
		t.Fatal("the 0.0.1 daemon was still running after 0.0.2 took its address")
	}

	stop := exec.Command(cur, "stop")
	stop.Env = env
	if out, err := stop.CombinedOutput(); err != nil || !strings.Contains(string(out), "stopped the cogmer daemon") {
		t.Fatalf("cogmer stop: %v\n%s", err, out)
	}
	if _, ok := daemonVersionAt(hooks); ok {
		t.Fatal("a daemon still answers after cogmer stop")
	}
}

// realGoCaches names the module and build caches the way the test process was started. A test
// that has pointed HOME at a temporary directory would otherwise make `go build` fill a new
// module cache under it, which takes about a minute and cannot be deleted when the test
// ends, since the module cache is read-only.
var realGoCaches = func() []string {
	out, err := exec.Command("go", "env", "GOMODCACHE", "GOCACHE").Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 2 {
		return nil
	}
	return []string{"GOMODCACHE=" + lines[0], "GOCACHE=" + lines[1]}
}()

func buildVersion(t *testing.T, dir, v string) string {
	t.Helper()
	out := filepath.Join(dir, v, "cogmer")
	cmd := exec.Command("go", "build", "-ldflags", "-X main.version="+v+" -X main.noBrowser=1", "-o", out, ".")
	cmd.Env = append(os.Environ(), realGoCaches...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build %s: %v\n%s", v, err, b)
	}
	return out
}

func waitForVersion(t *testing.T, addr, want string) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if v, ok := daemonVersionAt(addr); ok && v == want {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	got, ok := daemonVersionAt(addr)
	t.Fatalf("no daemon reporting %s on %s within 20s (got %q, answering %v)", want, addr, got, ok)
}

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

// The 09-22 case: a cogmer daemon left running from another state directory holds
// the peer-sync port, and a second daemon says so and names `stop`. Real builds,
// real processes, the overlay off.
func TestADaemonBlockedByAnotherCogmerSaysSoAndNamesStop(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("no lsof on this machine")
	}
	dir := t.TempDir()
	bin := buildVersion(t, dir, "0.0.1")
	if resolved, err := filepath.EvalSymlinks(bin); err == nil {
		bin = resolved
	}
	peer := freeAddr(t)
	env := func(home, hooks string) []string {
		return append(os.Environ(), "COGMER_HOME="+filepath.Join(dir, home),
			"COGMER_ADDR="+hooks, "COGMER_PEER_ADDR="+peer,
			"COGMER_TAILCAT=off", "COGMER_PREFLIGHT=off", "COGMER_PEERS=")
	}

	hooks1 := freeAddr(t)
	first := exec.Command(bin, "daemon")
	first.Env = env("home1", hooks1)
	if err := first.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = first.Process.Kill(); _ = first.Wait() }()
	waitForVersion(t, hooks1, "0.0.1")

	second := exec.Command(bin, "daemon")
	second.Env = env("home2", freeAddr(t))
	out, err := second.CombinedOutput()
	if err != nil {
		t.Fatalf("the blocked daemon should report and exit cleanly: %v\n%s", err, out)
	}
	for _, want := range []string{"held by another cogmer daemon", "pid " + strconv.Itoa(first.Process.Pid), bin + " stop"} {
		if !strings.Contains(string(out), want) && !strings.Contains(strings.ReplaceAll(string(out), "~", os.Getenv("HOME")), want) {
			t.Errorf("the report lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(string(out), "not a cogmer daemon") {
		t.Errorf("the report says a cogmer daemon is not one:\n%s", out)
	}
}
