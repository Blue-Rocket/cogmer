package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tailscale/tailcat"
)

// newRelayClient is a client of the relay library with its output filtered.
func newRelayClient(addr string) *tailcat.Client {
	c := tailcat.NewClient(tailcat.Addr(addr))
	c.Logf = relayLibraryLogf(log.Printf)
	return c
}

// The daemon's log is bounded (D-261). The file is opened for appending by a shell redirect
// in the plugin's scripts and by startDaemon, and the daemon's own output is that file, so the
// daemon cannot move it: its descriptor, and the shell's, would follow the file to its new
// name. What it can do is copy the file aside and empty it in place, which every appending
// writer survives, whichever way it opened the file.

const (
	// daemonLogMax is the size at which the log is copied aside and emptied.
	daemonLogMax = 10 << 20
	// daemonLogCheckEvery is how often the daemon looks at the log's size.
	daemonLogCheckEvery = time.Minute
)

func daemonLogPath() string { return filepath.Join(homeDir(), "daemon.log") }

// boundLog replaces path+".1" with a copy of path and then empties path, when path is larger
// than max. It reports whether it did. The copy is written to a file of its own and renamed
// over the older one, so that a failure leaves the older copy whole, and it is made with mode
// 0600 because the log names peers and addresses. Lines written between the copy and the
// emptying are lost, which is a few lines in a minute.
func boundLog(path string, max int64) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.Size() <= max {
		return false, nil
	}
	src, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer src.Close()
	tmp := path + ".1.tmp"
	dst, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return false, err
	}
	if _, err := io.CopyN(dst, src, info.Size()); err != nil && err != io.EOF {
		dst.Close()
		os.Remove(tmp)
		return false, err
	}
	if err := dst.Close(); err != nil {
		os.Remove(tmp)
		return false, err
	}
	if err := os.Rename(tmp, path+".1"); err != nil {
		os.Remove(tmp)
		return false, err
	}
	if err := os.Truncate(path, 0); err != nil {
		return false, err
	}
	return true, nil
}

// keepDaemonLogBounded checks the log at once and then at every interval, for as long as the
// daemon runs.
func keepDaemonLogBounded(path string, max int64, every time.Duration) {
	check := func() {
		if _, err := boundLog(path, max); err != nil {
			fmt.Fprintf(os.Stderr, "%s could not bound the log: %v\n", time.Now().UTC().Format(time.RFC3339), err)
		}
	}
	check()
	go func() {
		for range time.Tick(every) {
			check()
		}
	}()
}

// relayLibraryNoise names the prefixes of the relay library's own output that say nothing a
// person reading the log about a failed connection needs. They were about two thirds of the
// file.
var relayLibraryNoise = []string{"netstack:", "wg:", "[v1]", "magicsock:", "netcheck:"}

// relayLibraryLogf is the logger given to the relay library's clients. It drops the library's
// routine lines and passes the rest to the daemon's log.
func relayLibraryLogf(out func(string, ...any)) func(string, ...any) {
	return func(format string, args ...any) {
		line := strings.TrimSpace(fmt.Sprintf(format, args...))
		for _, p := range relayLibraryNoise {
			if strings.HasPrefix(line, p) {
				return
			}
		}
		out("%s", line)
	}
}
