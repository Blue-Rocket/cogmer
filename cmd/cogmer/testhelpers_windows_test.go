//go:build windows

package main

import "os"

// cannotNameProcesses says whether tasklist is refused in this session, as it is over ssh, where it
// answers "Access denied" for every process. Stopping a daemon names the process that holds the
// port first, so a test of stopping one cannot run there.
func cannotNameProcesses() bool { return processName(os.Getpid()) == "" }
