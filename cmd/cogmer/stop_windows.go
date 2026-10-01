//go:build windows

package main

import (
	"errors"
	"time"
)

// Windows has no lsof, so nothing here can find the process holding a port. A
// daemon of another version is then left serving, and a stop reports why.

var errStopUnsupported = errors.New("finding the process on a port is not supported on Windows")

func listeningPIDs(string) ([]int, error) { return nil, errStopUnsupported }
func processName(int) string              { return "" }
func terminate(int, time.Duration) error  { return errStopUnsupported }
