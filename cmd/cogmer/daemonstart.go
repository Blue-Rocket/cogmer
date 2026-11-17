package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// The bridge keeps the daemon running for a Cowork user (D-208). A session of another
// host starts the daemon from the plugin's shell function when it begins, and a Cowork
// user's machine has no plugin scripts and no session-start hook that reaches it, so the
// binary starts one itself, detached and logged as that function does (D-150).

// stoppedMarker is the file that says the user stopped the daemon. `stop` writes it and a
// daemon removes it when it starts, so it is there exactly while the user's stop stands.
func stoppedMarker() string { return filepath.Join(homeDir(), "stopped") }

// startDaemon starts a daemon of this binary that outlives the process that started it,
// with its output appended to the daemon's log.
func startDaemon() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := ensureStateDir(); err != nil {
		return err
	}
	logFile, err := os.OpenFile(filepath.Join(homeDir(), "daemon.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer logFile.Close()
	cmd := exec.Command(exe, "daemon")
	cmd.Stdin = nil
	cmd.Stdout, cmd.Stderr = logFile, logFile
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	fmt.Fprintf(logFile, "%s bridge: starting %s on %s (pid %d)\n", time.Now().UTC().Format(time.RFC3339), exe, addr(), cmd.Process.Pid)
	return cmd.Process.Release()
}

// daemonControl is what the bridge does to the daemon, apart from calling it.
type daemonControl interface {
	// Answering says whether a cogmer daemon answers on this installation's address, and
	// the version it runs.
	Answering() (string, bool)
	Start() error
	// Stop runs `cogmer stop`, with the flag a restart passes, and returns what it said.
	Stop(forRestart bool) (string, error)
	Stopped() bool
	SetStopped(bool)
}

type localDaemon struct{}

func (localDaemon) Answering() (string, bool) { return daemonVersionAt(addr()) }
func (localDaemon) Start() error              { return startDaemon() }
func (localDaemon) Stopped() bool             { _, err := os.Stat(stoppedMarker()); return err == nil }

func (localDaemon) SetStopped(stopped bool) {
	if stopped {
		_ = ensureStateDir()
		_ = os.WriteFile(stoppedMarker(), []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o600)
		return
	}
	_ = os.Remove(stoppedMarker())
}

func (localDaemon) Stop(forRestart bool) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	args := []string{"stop"}
	if forRestart {
		args = append(args, "--for-restart")
	}
	var out limitedBuffer
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Stdout, cmd.Stderr = &out, &out
	cmd.WaitDelay = time.Second
	err = cmd.Run()
	return string(out.b), err
}

// limitedBuffer keeps what a command printed, up to a bound.
type limitedBuffer struct{ b []byte }

func (l *limitedBuffer) Write(p []byte) (int, error) {
	if room := typedCommandMaxOutput - len(l.b); room > 0 {
		if len(p) > room {
			l.b = append(l.b, p[:room]...)
		} else {
			l.b = append(l.b, p...)
		}
	}
	return len(p), nil
}

var _ io.Writer = (*limitedBuffer)(nil)
