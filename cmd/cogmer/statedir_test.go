package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// The state directory holds the databases, the keys and the logs, so a mode that lets
// another account in is a leak of every conversation. These tests hold every creator of
// the directory to 0700 whatever the umask was, and hold a program that finds an older
// directory open to tightening it.

func skipWithoutModeBits(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no group or other mode bits")
	}
}

func modeOf(t *testing.T, p string) os.FileMode {
	t.Helper()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Mode().Perm()
}

// looseState returns a state directory, with a file in it, that an older program or a
// shell under a umask of 022 would have made.
func looseState(t *testing.T) string {
	t.Helper()
	home := filepath.Join(t.TempDir(), "state")
	if err := os.MkdirAll(filepath.Join(home, "rooms"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(home, "rooms"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "daemon.log"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COGMER_HOME", home)
	return home
}

func wantTight(t *testing.T, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		if m := modeOf(t, d); m != 0o700 {
			t.Errorf("%s is %o, want 700", d, m)
		}
	}
}

func TestOpeningMembershipTightensALooseStateDirectory(t *testing.T) {
	skipWithoutModeBits(t)
	home := looseState(t)
	m, err := OpenMembership()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	wantTight(t, home)
}

func TestOpeningARoomTightensTheStateDirectoryAndTheRoomsFolder(t *testing.T) {
	skipWithoutModeBits(t)
	home := looseState(t)
	s, err := OpenStore("r1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	wantTight(t, home, filepath.Join(home, "rooms"))
}

func TestANewRoomsFolderIsCreatedPrivateUnderUmask022(t *testing.T) {
	skipWithoutModeBits(t)
	home := filepath.Join(t.TempDir(), "state")
	t.Setenv("COGMER_HOME", home)
	old := setUmask(0o022)
	defer setUmask(old)
	s, err := OpenStore("r1")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	wantTight(t, home, filepath.Join(home, "rooms"))
}

func TestCreatingTheIdentityTightensALooseStateDirectory(t *testing.T) {
	skipWithoutModeBits(t)
	home := looseState(t)
	if _, err := LoadIdentity(); err != nil {
		t.Fatal(err)
	}
	wantTight(t, home)
}

func TestCreatingTheKeyTightensALooseStateDirectory(t *testing.T) {
	skipWithoutModeBits(t)
	home := looseState(t)
	if _, err := loadOrCreateKey(); err != nil {
		t.Fatal(err)
	}
	wantTight(t, home)
}

func TestRecordingAVerificationTightensALooseStateDirectory(t *testing.T) {
	skipWithoutModeBits(t)
	home := looseState(t)
	saveVerification(verification{Version: "9.9.9"})
	wantTight(t, home)
}

func TestRecordingDaemonStateAndTheStoppedMarkerTightenALooseStateDirectory(t *testing.T) {
	skipWithoutModeBits(t)
	home := looseState(t)
	recordDaemonState("up", "")
	wantTight(t, home)
	if err := os.Chmod(home, 0o755); err != nil {
		t.Fatal(err)
	}
	localDaemon{}.SetStopped(true)
	wantTight(t, home)
}

func TestRecordingTheEndpointTightensALooseStateDirectory(t *testing.T) {
	skipWithoutModeBits(t)
	home := looseState(t)
	if err := recordEndpoint("127.0.0.1:1"); err != nil {
		t.Fatal(err)
	}
	wantTight(t, home)
}

func TestSavingAnIdentityIntoADirectoryTightensIt(t *testing.T) {
	skipWithoutModeBits(t)
	home := looseState(t)
	if err := saveIdentityIn(home, &Identity{PeerID: "p", UserID: "u"}); err != nil {
		t.Fatal(err)
	}
	wantTight(t, home)
}

// A script run under the usual umask must still leave the directory and its logs closed
// to other accounts, for a directory that does not exist and for one that does.
func TestSessionStartUnderUmask022CreatesAPrivateStateDirectory(t *testing.T) {
	skipWithoutModeBits(t)
	for _, existing := range []bool{false, true} {
		root, base := cliHarness(t)
		home := filepath.Join(base, "state")
		if existing {
			if err := os.MkdirAll(home, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(home, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		src, err := os.ReadFile("../../plugin/hooks-handlers/session-start.sh")
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, "hooks-handlers", "session-start.sh")
		if err := os.WriteFile(path, src, 0o755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("bash", "-c", `umask 022; exec bash "$1"`, "x", path)
		cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + base, "COGMER_HOME=" + home, "CLAUDE_PLUGIN_ROOT=" + root, "COGMER_BIN=/missing", "COGMER_ADDR=127.0.0.1:1"}
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		if !waitFor(t, filepath.Join(home, "started")) {
			t.Fatal("startup did not start the installer")
		}
		wantTight(t, home)
		for _, f := range []string{"install.log", "daemon.log"} {
			if m := modeOf(t, filepath.Join(home, f)); m&0o077 != 0 {
				t.Errorf("%s is %o, readable by other accounts", f, m)
			}
		}
	}
}

func TestCLIUnderUmask022CreatesAPrivateStateDirectory(t *testing.T) {
	skipWithoutModeBits(t)
	root, base := cliHarness(t)
	home := filepath.Join(base, "state")
	cmd := exec.Command("bash", "-c", `umask 022; exec bash "$1" whoami`, "x", filepath.Join(root, "cli.sh"))
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + base, "COGMER_HOME=" + home, "CLAUDE_PLUGIN_ROOT=" + root}
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if !waitFor(t, filepath.Join(home, "started")) {
		t.Fatal("the command did not start the installer")
	}
	wantTight(t, home)
	if m := modeOf(t, filepath.Join(home, "install.log")); m&0o077 != 0 {
		t.Errorf("install.log is %o, readable by other accounts", m)
	}
}
