//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

// What stopping a daemon needs from Windows (D-123): the same three things as on a
// Unix, found with netstat and tasklist, which every Windows has. There is no
// SIGTERM, so a daemon is ended outright, which is survivable because a write is
// durable before it is published (§23).

func listeningPIDs(port string) ([]int, error) {
	out, err := exec.Command("netstat", "-ano", "-p", "tcp").Output()
	if err != nil {
		return nil, fmt.Errorf("netstat: %w", err)
	}
	return parseNetstatListeners(string(out), port), nil
}

func processName(pid int) string {
	out, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/FO", "CSV", "/NH").Output()
	if err != nil {
		return ""
	}
	return parseTasklistName(string(out))
}

func terminate(pid int, grace time.Duration) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := p.Kill(); err != nil {
		return err
	}
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if processName(pid) == "" {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("process %d was still running %s after it was ended", pid, grace)
}
