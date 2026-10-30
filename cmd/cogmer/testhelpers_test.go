package main

import (
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// Helpers that tests of every system share. They were in stop_test.go, which is built only
// where a daemon can be found with lsof, and leaving them there kept every test that used
// one from compiling on Windows.

// exeSuffix is the extension the operating system runs a program by.
var exeSuffix = func() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}()

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

// testBuildVersion is the version of a binary that a test builds without needing a particular one.
// Tests that name a version of their own, because they compare versions, still get their own.
const testBuildVersion = "test-build"

// A build links the whole program and takes about 4.5s, and 17 tests wanted one, so the process
// builds each version once, in a directory of its own, and hands every test a copy at the path it
// asked for. A copy and not a link, since a test may write over its binary.
var (
	sharedBuildsMu  sync.Mutex
	sharedBuildsDir string
	sharedBuilds    = map[string]*sharedBuild{}
)

type sharedBuild struct {
	once sync.Once
	path string
	err  string
}

func buildVersion(t *testing.T, dir, v string) string {
	t.Helper()
	sharedBuildsMu.Lock()
	if sharedBuildsDir == "" {
		d, err := os.MkdirTemp("", "cogmer-test-builds-")
		if err != nil {
			sharedBuildsMu.Unlock()
			t.Fatal(err)
		}
		sharedBuildsDir = d
	}
	b := sharedBuilds[v]
	if b == nil {
		b = &sharedBuild{}
		sharedBuilds[v] = b
	}
	root := sharedBuildsDir
	sharedBuildsMu.Unlock()

	b.once.Do(func() {
		b.path = filepath.Join(root, v, "cogmer"+exeSuffix)
		cmd := exec.Command("go", "build", "-ldflags", "-X main.version="+v+" -X main.noBrowser=1", "-o", b.path, ".")
		cmd.Env = append(os.Environ(), realGoCaches...)
		if out, err := cmd.CombinedOutput(); err != nil {
			b.err = err.Error() + "\n" + string(out)
		}
	})
	if b.err != "" {
		t.Fatalf("build %s: %s", v, b.err)
	}
	out := filepath.Join(dir, v, "cogmer"+exeSuffix)
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatal(err)
	}
	src, err := os.Open(b.path)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dst, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		t.Fatal(err)
	}
	if err := dst.Close(); err != nil {
		t.Fatal(err)
	}
	return out
}

// removeSharedBuilds deletes the binaries the process built, when the tests have finished.
func removeSharedBuilds() {
	sharedBuildsMu.Lock()
	defer sharedBuildsMu.Unlock()
	if sharedBuildsDir != "" {
		_ = os.RemoveAll(sharedBuildsDir)
	}
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

// setHome points the account's home at dir for the test, as the operating system names it: HOME
// where there is one, and USERPROFILE on Windows, where os.UserHomeDir reads that instead and a
// test that set only HOME would reach the account's real state directory.
func setHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}
}

// captureStdout runs fn with os.Stdout going to a pipe that is read while fn runs, and returns what
// fn wrote. A pipe read only afterwards holds 4KB on Windows, where a longer write blocks for good.
func captureStdout(t *testing.T, fn func()) []byte {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan []byte)
	go func() {
		b, _ := io.ReadAll(r)
		done <- b
	}()
	old := os.Stdout
	os.Stdout = w
	func() {
		defer func() {
			os.Stdout = old
			w.Close()
		}()
		fn()
	}()
	return <-done
}

// modesMeanSomething says whether a file's mode bits are permissions the operating system
// enforces. On Windows they are not: a file reads as 0666 and a directory as 0777 whatever its
// access list says, so a test of who may read a file cannot be made with them there.
func modesMeanSomething() bool { return runtime.GOOS != "windows" }

// needsToFindADaemon skips a test where the program cannot find the process that holds a port: it
// uses lsof on Unix systems, and netstat and tasklist on Windows, which every Windows has.
func needsToFindADaemon(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		if cannotNameProcesses() {
			t.Skip("tasklist is refused in this session, as it is over ssh, so no process can be named")
		}
		return
	}
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("no lsof on this machine")
	}
}

// standInProgram writes a program that prints or exits as a test needs, and returns its path. A
// shell script is not a program Windows runs, so there it is a batch file: sh is the body of the
// script and cmd the body of the batch file.
func standInProgram(t *testing.T, dir, name, sh, cmd string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	body := "#!/bin/sh\n" + sh + "\n"
	if runtime.GOOS == "windows" {
		path += ".cmd"
		body = "@echo off\r\n" + cmd + "\r\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// pathWith is a PATH that holds dir first and what a stand-in needs beyond it.
func pathWith(dir string) string {
	if runtime.GOOS == "windows" {
		return dir + ";" + filepath.Join(os.Getenv("SystemRoot"), "System32")
	}
	return dir + ":/usr/bin:/bin"
}
