package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// The state directory carries one data version, in a file of its own, which every
// entry point reads before it reads or writes anything else there (D-244, the data in
// the state directory carries one version, and a program that finds a newer one
// touches nothing). A program that finds a version it does not know leaves the
// directory as it found it: an older program would otherwise drop the fields it does
// not know when it rewrites a file, and misread every store it opens.

// dataVersionFile holds the version as a decimal number. A directory without it holds
// version 0, the data of every program before the version existed.
const dataVersionFile = "data-version"

func dataVersionPath() string { return filepath.Join(homeDir(), dataVersionFile) }

// NewerDataError is a state directory this program must not touch: its version is
// higher than this program knows, or its version cannot be read, which a program
// cannot tell from a higher one.
type NewerDataError struct {
	Found int   // the version found, when it could be read
	Cause error // why it could not be read, or nil
}

func (e *NewerDataError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("the data version in %s cannot be read (%v)", homeDir(), e.Cause)
	}
	return fmt.Sprintf("the data in %s is version %d, and this cogmer (%s) knows versions up to %d",
		homeDir(), e.Found, version, dataVersion)
}

// storedDataVersion is the state directory's data version, 0 when it has none. It
// returns a *NewerDataError when the version is one this program must not touch.
func storedDataVersion() (int, error) {
	raw, err := os.ReadFile(dataVersionPath())
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, &NewerDataError{Cause: err}
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || n < 0 {
		return 0, &NewerDataError{Cause: fmt.Errorf("%s holds %q, which is not a version", dataVersionFile, strings.TrimSpace(string(raw)))}
	}
	if n > dataVersion {
		return n, &NewerDataError{Found: n}
	}
	return n, nil
}

// upgradeStateDirectory copies the databases (D-256, the databases are copied before
// the state directory is migrated, and a program that cannot migrate it stops), runs
// every migration this program has on the state directory below its version, and then
// records the version. Room stores are otherwise
// migrated only when something opens them, so every store under rooms/ is opened
// here, and a store nothing would open for months is not left in an older format
// behind a version that says otherwise. A migration that fails leaves the version as
// it was, and the next program to start tries again.
func upgradeStateDirectory(stored int) error {
	if stored >= dataVersion {
		return nil
	}
	if _, err := backupStateDirectory(stored); err != nil {
		return &backupError{err}
	}
	m, err := OpenMembership()
	if err != nil {
		return fmt.Errorf("membership: %w", err)
	}
	m.Close()
	files, err := filepath.Glob(filepath.Join(homeDir(), "rooms", "*.db"))
	if err != nil {
		return err
	}
	for _, f := range files {
		room := strings.TrimSuffix(filepath.Base(f), ".db")
		s, err := OpenStore(room)
		if err != nil {
			return fmt.Errorf("room %s: %w", room, err)
		}
		s.Close()
	}
	return writeDataVersion(dataVersion)
}

// writeDataVersion records v through a file renamed into place, so a reader finds the
// old version or the new one and never a partial file. It never lowers the version a
// program that started at the same moment recorded.
func writeDataVersion(v int) error {
	if err := os.MkdirAll(homeDir(), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(homeDir(), "."+dataVersionFile+"-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := fmt.Fprintf(tmp, "%d\n", v); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if now, err := storedDataVersion(); err != nil || now >= v {
		return err
	}
	return os.Rename(tmp.Name(), dataVersionPath())
}

// noDataEntryPoints are the subcommands that never touch the state directory: they
// print what is built in, or write only to the directory a behavior check gives them.
var noDataEntryPoints = map[string]bool{
	"version": true, "behaviors": true,
	"probe-hook": true, "probe-mcp": true, "codex-probe-hook": true, "probe-bighook": true,
}

// guardStateDirectory is what every entry point does before anything else, and it
// returns only when the entry point may go on. On data newer than this program, a
// hook ends at once with no output, the bridge answers with nothing, the daemon logs
// why and does not start, and a command says what to do. Otherwise the daemon and the
// commands bring the directory up to this program's version first, and a hook
// leaves that to them, so that a hook never waits on a migration.
func guardStateDirectory(sub string) {
	if noDataEntryPoints[sub] {
		return
	}
	stored, err := storedDataVersion()
	var newer *NewerDataError
	if errors.As(err, &newer) {
		refuseNewerData(sub, newer)
		return
	}
	switch sub {
	case "hook", "bridge":
		return
	}
	if err := upgradeStateDirectory(stored); err != nil {
		refuseFailedUpgrade(sub, stored, err)
	}
}

// refuseFailedUpgrade ends the entry point after an upgrade that stopped partway, so
// that nothing is written on top of databases that are partly new (D-256). It says
// where the copy is and when it is safe to put it back. The stop command goes on,
// since a user is told to stop cogmer before restoring the copy.
func refuseFailedUpgrade(sub string, stored int, err error) {
	if sub == "stop" {
		return
	}
	text := failedUpgradeText(stored, err)
	if sub == "daemon" {
		log.Print("not starting: " + strings.TrimSpace(text))
	} else {
		fmt.Fprint(os.Stderr, text)
	}
	os.Exit(1)
}

// refuseNewerData ends the entry point, except the bridge, which keeps answering its
// host with nothing.
func refuseNewerData(sub string, e *NewerDataError) {
	switch sub {
	case "hook":
		// Drained, as the hook does when it runs, so the host never finds its write
		// refused. Nothing is written and the exit is 0 (§3.1, first do no harm).
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	case "bridge":
		runRefusingBridge(newerDataText(e))
		os.Exit(0)
	case "daemon":
		log.Printf("not starting: %v. This daemon reads and writes nothing there; a newer cogmer wrote it, and updating the cogmer plugin brings one.", e)
		os.Exit(0)
	}
	fmt.Println(newerDataText(e))
	os.Exit(1)
}

// newerDataText is what a command says on data it must not touch, for the model to
// relay.
func newerDataText(e *NewerDataError) string {
	if e.Cause != nil {
		return fmt.Sprintf("cogmer did nothing: %v, so it cannot tell whether this cogmer can read the data there, and has left it untouched.\n\n"+
			"A newer cogmer writes that file. Updating the cogmer plugin brings the newest one, and its README says how under Updating.", e)
	}
	return fmt.Sprintf("cogmer did nothing: the data on this machine was written by a newer cogmer than this one. %s.\n\n"+
		"This cogmer has left the data untouched, since reading or changing it could damage the user's rooms and identity. "+
		"Updating the cogmer plugin brings a cogmer that reads it, and the plugin's README says how under Updating. "+
		"Until then nothing is captured or shared on this machine.", capitalize(e.Error()))
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
