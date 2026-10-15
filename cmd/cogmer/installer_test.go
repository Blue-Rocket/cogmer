//go:build !windows

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

// The installer is the plugin's bash script, run here as it is, against a stand-in program and a
// release host that counts what it is asked for (D-215).

// versionScript is a program that reports a version, which is all the installer asks of one.
func versionScript(v string) []byte {
	return []byte("#!/bin/sh\nif [ \"$1\" = version ]; then echo " + v + "; fi\n")
}

type installerRun struct {
	home     string
	requests *atomic.Int32
	log      string
	output   string // what the script printed, which is nothing unless something went wrong
	version  string // what the program at the install path reports afterwards
}

// runInstaller runs install.sh for a plugin that expects want, on a machine whose installed program
// reports installed, or has none when installed is empty.
func runInstaller(t *testing.T, installed, want string) installerRun {
	t.Helper()
	for _, tool := range []string{"bash", "curl"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("no %s", tool)
		}
	}
	osName := map[string]string{"darwin": "darwin", "linux": "linux"}[runtime.GOOS]
	arch := map[string]string{"arm64": "arm64", "amd64": "amd64"}[runtime.GOARCH]
	if osName == "" || arch == "" {
		t.Skip("the installer does not run on this platform")
	}
	asset := fmt.Sprintf("cogmer_%s_%s_%s", want, osName, arch)
	released := versionScript(want)
	sum := sha256.Sum256(released)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "VERSION"), []byte(want+"\n"))
	writeFile(t, filepath.Join(root, "checksums.txt"), []byte(hex.EncodeToString(sum[:])+"  "+asset+"\n"))

	requests := &atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path == "/v"+want+"/"+asset {
			_, _ = w.Write(released)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	home := t.TempDir()
	target := filepath.Join(home, "bin", "cogmer")
	if installed != "" {
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, versionScript(installed), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("bash", "plugin/hooks-handlers/install.sh")
	cmd.Dir = "../.."
	cmd.Env = append(os.Environ(), "COGMER_HOME="+home, "CLAUDE_PLUGIN_ROOT="+root, "COGMER_RELEASE_URL="+srv.URL,
		"COGMER_ADDR=127.0.0.1:1", "HOME="+t.TempDir())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh: %v\n%s", err, out)
	}
	run := installerRun{home: home, requests: requests, output: string(out)}
	if b, err := os.ReadFile(filepath.Join(home, "install.log")); err == nil {
		run.log = string(b)
	}
	if out, err := exec.Command(target, "version").Output(); err == nil {
		run.version = strings.TrimSpace(string(out))
	}
	return run
}

// A plugin that expects an older version than the program on the machine leaves the program alone.
// That is what makes two copies of the plugin harmless: whichever starts a session, the newest
// program stays.
func TestAnOlderPluginLeavesANewerProgramAlone(t *testing.T) {
	for _, c := range []struct{ installed, want string }{
		{"0.10.2", "0.7.4"},
		{"0.10.0", "0.9.1"}, // numbers, not letters: the program is newer though "0.1" sorts before "0.9"
		{"0.7.4.1", "0.7.4"},
		{"1.0.0", "0.99.99"},
	} {
		run := runInstaller(t, c.installed, c.want)
		if run.version != c.installed {
			t.Errorf("a plugin expecting %s replaced %s with %q", c.want, c.installed, run.version)
		}
		if n := run.requests.Load(); n != 0 {
			t.Errorf("a plugin expecting %s fetched %d times with %s installed", c.want, n, c.installed)
		}
		if !strings.Contains(run.log, "keeping "+c.installed) {
			t.Errorf("nothing says why %s was kept:\n%s", c.installed, run.log)
		}
	}
}

func TestANewerPluginReplacesAnOlderProgram(t *testing.T) {
	for _, c := range []struct{ installed, want string }{
		{"0.7.4", "0.10.2"},
		{"0.9.1", "0.10.0"}, // the numbers, which a comparison of letters would get backwards
		{"0.7.4", "0.7.4.1"},
		{"dev", "0.10.2"}, // no order to go by, so a difference replaces, as it always has
		{"", "0.10.2"},    // a machine with no program
	} {
		run := runInstaller(t, c.installed, c.want)
		if run.version != c.want {
			t.Errorf("a plugin expecting %s left %q where %q was", c.want, run.version, c.installed)
		}
		if n := run.requests.Load(); n != 1 {
			t.Errorf("replacing %q with %s took %d fetches", c.installed, c.want, n)
		}
		if strings.Contains(run.log, "keeping") {
			t.Errorf("%q was replaced and the log says it was kept:\n%s", c.installed, run.log)
		}
	}
}

func TestThePluginsOwnVersionIsLeftAloneWithoutAFetch(t *testing.T) {
	run := runInstaller(t, "0.10.2", "0.10.2")
	if run.version != "0.10.2" || run.requests.Load() != 0 || strings.Contains(run.log, "keeping") {
		t.Errorf("version %q, %d fetches, log:\n%s", run.version, run.requests.Load(), run.log)
	}
}

// A version that is not dotted numbers has no order, so it is replaced, and the script says
// nothing while it does: a shell error here would reach the session that started it.
func TestAProgramWithNoOrderToGoByIsReplacedWithoutAShellError(t *testing.T) {
	for _, installed := range []string{"dev", "1x", "0.7.4-rc1", "v0.7.4", "0..1", "9999999999999999999999.1", "0.07.4"} {
		run := runInstaller(t, installed, "0.10.2")
		if run.version != "0.10.2" {
			t.Errorf("%q was not replaced: %q", installed, run.version)
		}
		if strings.TrimSpace(run.output) != "" {
			t.Errorf("%q made the installer print %q", installed, run.output)
		}
	}
}
