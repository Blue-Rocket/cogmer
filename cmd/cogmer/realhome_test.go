package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// A development or test build never opens the account's real state directory (D-265, a
// development or test build never opens the account's real state directory). These tests
// name a stand-in for the account's home, so none of them involves the real directory.

func standInForTheAccountsHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	was := realHomeForGuard
	realHomeForGuard = home
	t.Cleanup(func() { realHomeForGuard = was })
	return home
}

func TestADevelopmentBuildRefusesTheAccountsStateDirectoryHoweverItIsReached(t *testing.T) {
	home := standInForTheAccountsHome(t)
	real := filepath.Join(home, stateDirName)
	if err := os.MkdirAll(filepath.Join(real, "rooms"), 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "looks-harmless")
	if err := os.Symlink(real, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("a link needs a privilege an ordinary Windows account does not hold: %v", err)
		}
		t.Fatal(err)
	}
	for name, dir := range map[string]string{
		"the directory itself": real,
		"a folder inside it":   filepath.Join(real, "rooms"),
		"a path not made yet":  filepath.Join(real, "rooms", "new", "deeper"),
		"a link to it":         link,
		"a link into it":       filepath.Join(link, "rooms"),
		"an unclean spelling":  real + string(os.PathSeparator) + "." + string(os.PathSeparator) + "rooms" + string(os.PathSeparator) + "..",
		"a relative reach in":  relativeTo(t, filepath.Join(real, "rooms")),
	} {
		if err := refusedHome(dir); err == nil {
			t.Errorf("%s was not refused", name)
		}
	}
	for name, dir := range map[string]string{
		"a directory made for the run":     t.TempDir(),
		"a sibling that shares the prefix": real + "-test",
		"the account's home itself":        home,
		"another dot directory":            filepath.Join(home, ".cogmer-probe"),
	} {
		if err := refusedHome(dir); err != nil {
			t.Errorf("%s was refused: %v", name, err)
		}
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		if refusedHome(strings.ToUpper(real)) == nil {
			t.Error("a spelling that differs only in case was not refused, on a file system that ignores case")
		}
	}
}

// relativeTo is a path to dir that is relative to the working directory.
func relativeTo(t *testing.T, dir string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(wd, dir)
	if err != nil {
		t.Skip("no relative path to the stand-in directory on this system")
	}
	return rel
}

func TestAReleaseBuildIsNotRefusedItsOwnStateDirectory(t *testing.T) {
	home := standInForTheAccountsHome(t)
	real := filepath.Join(home, stateDirName)
	for _, c := range []struct {
		name                string
		version, noBrowser  string
		inTest, development bool
	}{
		{"a release", "0.10.5", "", false, false},
		{"a source build", "dev", "", false, true},
		{"a build a test made", "0.10.5", "1", false, true},
		{"a test binary", "0.10.5", "", true, true},
	} {
		if got := developmentBuild(c.version, c.noBrowser, c.inTest); got != c.development {
			t.Errorf("%s: development is %v, want %v", c.name, got, c.development)
		}
	}
	if err := refusedHomeIf(false, real); err != nil {
		t.Errorf("a release build was refused: %v", err)
	}
	if err := refusedHomeIf(true, real); err == nil {
		t.Error("a development build was not refused")
	}
}

// In a test, a refusal is a panic, so a test that forgot to name a directory fails where
// it would otherwise have used the real one.
func TestHomeDirPanicsInsteadOfReturningTheAccountsStateDirectory(t *testing.T) {
	home := standInForTheAccountsHome(t)
	real := filepath.Join(home, stateDirName)
	panics := func() (did bool) {
		defer func() { did = recover() != nil }()
		homeDir()
		return false
	}
	t.Setenv("COGMER_HOME", "")
	setHome(t, home)
	if !panics() {
		t.Error("with no COGMER_HOME and the account's home as HOME, homeDir returned the real directory")
	}
	t.Setenv("COGMER_HOME", real)
	if !panics() {
		t.Error("COGMER_HOME naming the real directory was accepted")
	}
	t.Setenv("COGMER_HOME", t.TempDir())
	if panics() {
		t.Error("a directory made for the test was refused")
	}
	t.Setenv("COGMER_HOME", "")
	setHome(t, t.TempDir())
	if panics() {
		t.Error("a test that sets HOME to a directory of its own was refused")
	}
}

func buildWithFlags(t *testing.T, ldflags string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "cogmer"+exeSuffix)
	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", out, ".")
	cmd.Env = append(os.Environ(), realGoCaches...)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, b)
	}
	return out
}

func TestADevelopmentBuildStopsBeforeTouchingTheRealDirectoryAndAReleaseBuildDoesNot(t *testing.T) {
	if testing.Short() {
		t.Skip("builds binaries")
	}
	stand := t.TempDir()
	real := filepath.Join(stand, stateDirName)
	dev := buildWithFlags(t, "-X main.realHomeForGuard="+stand)
	built := buildWithFlags(t, "-X main.version=0.0.1-guard -X main.noBrowser=1 -X main.realHomeForGuard="+stand)
	run := func(bin string, stdin string, args ...string) (string, int, time.Duration) {
		cmd := exec.Command(bin, args...)
		cmd.Env = append(os.Environ(), "COGMER_HOME="+real, "COGMER_ADDR="+freeAddr(t), "COGMER_PEER_ADDR="+freeAddr(t),
			"COGMER_TAILCAT=off", "COGMER_HOST=claude", "COGMER_PREFLIGHT=off")
		if stdin != "" {
			cmd.Stdin = strings.NewReader(stdin)
		}
		var out bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &out
		start := time.Now()
		err := cmd.Run()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
		return out.String(), code, time.Since(start)
	}
	untouched := func(what string) {
		t.Helper()
		if _, err := os.Stat(real); !os.IsNotExist(err) {
			t.Errorf("%s: a development build made or opened the stand-in for the real directory: %v", what, err)
		}
	}

	out, code, _ := run(dev, "", "whoami")
	if code != 2 || !strings.Contains(out, "development build") || !strings.Contains(out, "There is no way to override this") {
		t.Errorf("a command: exit %d, output %q", code, out)
	}
	untouched("a command")

	out, code, took := run(dev, `{"session_id":"s","prompt":"hi","transcript_path":"/nonexistent"}`, "hook", "prompt")
	if code != 0 || out != "" || took > 3*time.Second {
		t.Errorf("a hook was not silent and quick: exit %d, output %q, %s", code, out, took)
	}
	untouched("a hook")

	// A build a test made refuses out loud, so a test that forgot a directory cannot pass by silence.
	out, code, _ = run(built, `{"session_id":"s","prompt":"hi","transcript_path":"/nonexistent"}`, "hook", "prompt")
	if code != 2 || !strings.Contains(out, "development build") {
		t.Errorf("a hook of a build a test made was not refused out loud: exit %d, output %q", code, out)
	}
	untouched("a hook of a build a test made")

	out, code, _ = run(dev, "", "daemon")
	if code == 0 || !strings.Contains(out, "not starting") {
		t.Errorf("the daemon: exit %d, output %q", code, out)
	}
	untouched("the daemon")

	release := buildWithFlags(t, "-X main.version=0.0.1-guard-release -X main.realHomeForGuard="+stand)
	if out, code, _ := run(release, "", "whoami"); code != 0 || strings.Contains(out, "development build") {
		t.Errorf("a release build was refused its own state directory: exit %d, output %q", code, out)
	}
}
