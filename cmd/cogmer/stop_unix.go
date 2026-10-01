//go:build !windows

package main

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// What stopping a daemon needs from the operating system: who listens on a port,
// what that process is called, and a way to end it (D-123). Apart from main.go
// because syscall.Kill does not exist on Windows, and the release builds Windows.

// listeningPIDs is every process listening on a TCP port on this machine.
func listeningPIDs(port string) ([]int, error) {
	out, err := exec.Command("lsof", "-nP", "-iTCP:"+port, "-sTCP:LISTEN", "-t").Output()
	if err != nil {
		// lsof exits 1 when nothing matches, which is an answer, not a failure.
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(strings.TrimSpace(string(out))) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("lsof: %w", err)
	}
	var pids []int
	for _, f := range strings.Fields(string(out)) {
		if pid, err := strconv.Atoi(f); err == nil {
			pids = append(pids, pid)
		}
	}
	return pids, nil
}

// processName is the base name of a process's executable.
func processName(pid int) string {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=").Output()
	if err != nil {
		return ""
	}
	return filepath.Base(strings.TrimSpace(string(out)))
}

// terminate asks a process to stop, waits for it, and kills it if it has not.
// The daemon closes its listeners and stores on SIGTERM; being killed outright is
// survivable too, because a write is durable before it is published (§23).
func terminate(pid int, grace time.Duration) error {
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return err
	}
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if syscall.Kill(pid, 0) != nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && syscall.Kill(pid, 0) == nil {
		return err
	}
	return nil
}
