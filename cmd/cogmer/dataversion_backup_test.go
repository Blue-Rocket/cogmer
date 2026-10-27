package main

import (
	"database/sql"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The databases are copied before they are migrated, so that an upgrade that stops
// partway can be undone (D-256, the databases are copied before the state directory is
// migrated, and a program that cannot migrate it stops).

// oldSignerRoom writes a room database from before the signer column, holding one event.
func oldSignerRoom(t *testing.T, name, content string) string {
	t.Helper()
	dir := filepath.Join(homeDir(), "rooms")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name+".db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(eventsSchemaBeforeSigner); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO events (event_id, peer_id, peer_sequence, room_id, timestamp, event_type, content)
		VALUES ('e1', 'p', 1, ?, '2026-10-27T00:00:00Z', 'USER_PROMPT', ?)`, name, content); err != nil {
		t.Fatal(err)
	}
	return path
}

func hasSignerColumn(t *testing.T, path string) bool {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('events') WHERE name = 'signer'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n == 1
}

func eventContent(t *testing.T, path string) string {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var c string
	if err := db.QueryRow(`SELECT content FROM events WHERE event_id = 'e1'`).Scan(&c); err != nil {
		t.Fatal(err)
	}
	return c
}

// readOnlyRoom is an old room database a migration cannot write, which is how a
// migration is made to fail after the copy has been made.
func readOnlyRoom(t *testing.T, name string) string {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only file stops a migration only for an ordinary user on a system with file modes")
	}
	path := oldSignerRoom(t, name, "kept")
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0o644) })
	return path
}

func TestTheDatabasesAreCopiedAsTheyWereBeforeTheyAreMigrated(t *testing.T) {
	if dataVersion == 0 {
		t.Skip("a build that knows no data version has nothing to upgrade to")
	}
	home := useHome(t)
	m, err := OpenMembership()
	if err != nil {
		t.Fatal(err)
	}
	m.Close()
	a := oldSignerRoom(t, "room-a", "written before the upgrade")
	oldSignerRoom(t, "room-b", "also written before")

	if err := upgradeStateDirectory(0); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	copyDir := backupPath(0)
	if filepath.Base(copyDir) != "backup-data-version-0" || filepath.Dir(copyDir) != home {
		t.Fatalf("the copy is at %s", copyDir)
	}
	for _, f := range []string{"membership.db", "rooms/room-a.db", "rooms/room-b.db"} {
		if _, err := os.Stat(filepath.Join(copyDir, f)); err != nil {
			t.Errorf("the copy lacks %s: %v", f, err)
		}
	}
	if !hasSignerColumn(t, a) {
		t.Error("the database was not migrated")
	}
	copied := filepath.Join(copyDir, "rooms", "room-a.db")
	if hasSignerColumn(t, copied) {
		t.Error("the copy was taken after the migration, so it holds the new shape")
	}
	if got := eventContent(t, copied); got != "written before the upgrade" {
		t.Errorf("the copy holds %q", got)
	}
	if left, _ := filepath.Glob(filepath.Join(home, ".backup-incomplete-*")); len(left) != 0 {
		t.Errorf("the copy left %v behind", left)
	}
	if runtime.GOOS != "windows" {
		if fi, err := os.Stat(copied); err != nil || fi.Mode().Perm() != 0o600 {
			t.Errorf("a copied database is readable by others: %v %v", fi, err)
		}
		if fi, err := os.Stat(copyDir); err != nil || fi.Mode().Perm() != 0o700 {
			t.Errorf("the copy's folder is open to others: %v %v", fi, err)
		}
	}
}

// A migration that stops partway leaves some databases new, and a second attempt must
// not replace the copy with them.
func TestAFailedMigrationKeepsTheCopyAndARetryDoesNotReplaceIt(t *testing.T) {
	if dataVersion == 0 {
		t.Skip("a build that knows no data version has nothing to upgrade to")
	}
	home := useHome(t)
	a := oldSignerRoom(t, "room-a", "in the first room")
	readOnlyRoom(t, "room-b")

	if err := upgradeStateDirectory(0); err == nil {
		t.Fatal("a database that cannot be written was reported as migrated")
	}
	if _, err := os.Stat(filepath.Join(home, dataVersionFile)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a failed migration wrote the version: %v", err)
	}
	if !hasSignerColumn(t, a) {
		t.Fatal("the first room was not migrated, so the retry has nothing to prove")
	}
	copied := filepath.Join(backupPath(0), "rooms", "room-a.db")
	if hasSignerColumn(t, copied) || eventContent(t, copied) != "in the first room" {
		t.Error("the copy does not hold the first room as it was")
	}

	if err := upgradeStateDirectory(0); err == nil {
		t.Fatal("the retry succeeded on a database that cannot be written")
	}
	if hasSignerColumn(t, copied) {
		t.Error("the retry replaced the copy with a database that was already migrated")
	}
}

// Nothing is migrated without a copy, so a database that cannot be copied leaves
// every database as it was.
func TestNothingIsMigratedWhenTheCopyCannotBeMade(t *testing.T) {
	if dataVersion == 0 {
		t.Skip("a build that knows no data version has nothing to upgrade to")
	}
	home := useHome(t)
	a := oldSignerRoom(t, "room-a", "kept")
	broken := filepath.Join(home, "rooms", "room-z.db")
	if err := os.WriteFile(broken, []byte(strings.Repeat("not a database ", 512)), 0o600); err != nil {
		t.Fatal(err)
	}

	err := upgradeStateDirectory(0)
	var noCopy *backupError
	if !errors.As(err, &noCopy) {
		t.Fatalf("the upgrade went on without a copy: %v", err)
	}
	if hasSignerColumn(t, a) {
		t.Error("a database was migrated though the copy could not be made")
	}
	if _, err := os.Stat(backupPath(0)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("an unfinished copy took the copy's name: %v", err)
	}
	if left, _ := filepath.Glob(filepath.Join(home, ".backup-incomplete-*")); len(left) != 0 {
		t.Errorf("the failed copy left %v behind", left)
	}
	if _, err := os.Stat(filepath.Join(home, dataVersionFile)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the version was written: %v", err)
	}
	if text := failedUpgradeText(0, err); !strings.Contains(text, "No database was changed") {
		t.Errorf("the message does not say the databases are as they were:\n%s", text)
	}
}

func TestADirectoryWithNoDatabaseIsNotCopied(t *testing.T) {
	if dataVersion == 0 {
		t.Skip("a build that knows no data version has nothing to upgrade to")
	}
	useHome(t)
	if err := upgradeStateDirectory(0); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if _, err := os.Stat(backupPath(0)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a directory with nothing in it was copied: %v", err)
	}
}

// The user is told where the copy is, and told when putting it back is unsafe.
func TestTheMessageSaysWhereTheCopyIsAndWhenToUseIt(t *testing.T) {
	home := useHome(t)
	if err := os.MkdirAll(backupPath(0), 0o700); err != nil {
		t.Fatal(err)
	}
	text := failedUpgradeText(0, errors.New("room room-b: attempt to write a readonly database"))
	for _, want := range []string{home, backupPath(0), "membership.db-wal", "has not run since", "reuse numbers"} {
		if !strings.Contains(text, want) {
			t.Errorf("the message does not say %q:\n%s", want, text)
		}
	}
}

// A program that cannot migrate the data stops and says so, a hook stays silent, and
// stop still goes on, since the user is told to stop cogmer before restoring the copy.
func TestAProgramThatCannotMigrateStopsAndSaysWhereTheCopyIs(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	if dataVersion == 0 {
		t.Skip("a build that knows no data version has nothing to upgrade to")
	}
	bin := buildVersion(t, t.TempDir(), "0.0.1-failed-migration")
	home := useHome(t)
	oldSignerRoom(t, "room-a", "kept")
	readOnlyRoom(t, "room-b")
	env := append(os.Environ(), "COGMER_HOME="+home, "COGMER_ADDR="+freeAddr(t), "COGMER_PEER_ADDR="+freeAddr(t),
		"COGMER_TAILCAT=off", "COGMER_HOST=claude", "CLAUDE_CODE_SESSION_ID=s-1")

	daemon := exec.Command(bin, "daemon")
	daemon.Env = env
	done := make(chan []byte, 1)
	go func() { b, _ := daemon.CombinedOutput(); done <- b }()
	select {
	case b := <-done:
		if !strings.Contains(string(b), "not starting") || !strings.Contains(string(b), "backup-data-version-0") {
			t.Errorf("the daemon did not say why it stopped or where the copy is:\n%s", b)
		}
		if daemon.ProcessState == nil || daemon.ProcessState.Success() {
			t.Error("the daemon that could not migrate exited as a success")
		}
	case <-time.After(15 * time.Second):
		_ = daemon.Process.Kill()
		t.Fatal("the daemon started on data it could not migrate")
	}

	cmd := exec.Command(bin, "whoami")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "backup-data-version-0") {
		t.Errorf("a command on data that cannot be migrated went on or said nothing: %v\n%s", err, out)
	}

	hook := exec.Command(bin, "hook", "prompt")
	hook.Env = env
	hook.Stdin = strings.NewReader(`{"session_id":"s-1","prompt":"hello","transcript_path":"/nonexistent"}`)
	// A hook never migrates, so it fails open as it does with no daemon: it exits 0 and
	// says nothing about the data.
	if b, err := hook.CombinedOutput(); err != nil || strings.Contains(string(b), "could not bring the data") {
		t.Errorf("a hook on data that cannot be migrated did not fail open: %v %q", err, b)
	}

	stop := exec.Command(bin, "stop")
	stop.Env = env
	if b, _ := stop.CombinedOutput(); strings.Contains(string(b), "could not bring the data") {
		t.Errorf("stop was refused, though a user is told to stop cogmer before restoring:\n%s", b)
	}
}
