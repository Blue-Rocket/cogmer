package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Before a program migrates the state directory it copies the databases the migration
// is about to change (D-256, the databases are copied before the state directory is
// migrated, and a program that cannot migrate it stops). The copy is the data as the
// older version left it, so that a migration that stops partway, leaving some
// databases new and some old, can be undone by putting the copy back.

// backupPath is the folder holding the copy of the data as it stood at version
// stored, inside the state directory. Nothing else reads it.
func backupPath(stored int) string {
	return filepath.Join(homeDir(), fmt.Sprintf("backup-data-version-%d", stored))
}

// databasesToMigrate is every database in the state directory that an upgrade opens:
// the membership index, then each room database.
func databasesToMigrate() []string {
	var out []string
	if _, err := os.Stat(filepath.Join(homeDir(), "membership.db")); err == nil {
		out = append(out, filepath.Join(homeDir(), "membership.db"))
	}
	rooms, _ := filepath.Glob(filepath.Join(homeDir(), "rooms", "*.db"))
	return append(out, rooms...)
}

// backupStateDirectory copies every database an upgrade from version stored would
// change into backupPath(stored), and returns that path. It copies nothing for a
// directory that holds no database. A copy already in place is the data from before the
// first attempt, because a migration that failed has left the databases partly new, so
// it is never replaced. A copy that was not finished is never mistaken for one:
// the databases go into a folder of their own, which takes its name only when the last
// one is in.
func backupStateDirectory(stored int) (string, error) {
	files := databasesToMigrate()
	if len(files) == 0 {
		return "", nil
	}
	final := backupPath(stored)
	if _, err := os.Stat(final); err == nil {
		return final, nil
	}
	tmp, err := os.MkdirTemp(homeDir(), ".backup-incomplete-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	for _, f := range files {
		rel, err := filepath.Rel(homeDir(), f)
		if err != nil {
			return "", err
		}
		dest := filepath.Join(tmp, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			return "", err
		}
		if err := copyDatabase(f, dest); err != nil {
			return "", fmt.Errorf("%s: %w", rel, err)
		}
	}
	if err := os.Rename(tmp, final); err != nil {
		// A program that started at the same moment finished its copy first.
		if _, statErr := os.Stat(final); statErr == nil {
			return final, nil
		}
		return "", err
	}
	return final, nil
}

// copyDatabase writes a consistent copy of the database at src to dest, which is a
// database of its own with no write-ahead log beside it. It opens src without
// migrating it.
func copyDatabase(src, dest string) error {
	db, err := sql.Open("sqlite", src)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec(`VACUUM INTO ?`, dest); err != nil {
		return err
	}
	return os.Chmod(dest, 0o600)
}

// backupError is a copy that could not be made, which leaves every database as it was
// because nothing is migrated without one.
type backupError struct{ err error }

func (e *backupError) Error() string {
	return "copying the databases before the upgrade: " + e.err.Error()
}
func (e *backupError) Unwrap() error { return e.err }

// failedUpgradeText says what a program prints when it cannot bring the state
// directory to its version, and what to do about it.
func failedUpgradeText(stored int, err error) string {
	var b strings.Builder
	fmt.Fprintf(&b, "cogmer could not bring the data in %s to version %d: %v\n", homeDir(), dataVersion, err)
	var noCopy *backupError
	switch {
	case errors.As(err, &noCopy):
		b.WriteString("No database was changed.\n")
	case fileExists(backupPath(stored)):
		fmt.Fprintf(&b, "A copy of the databases as they were before the upgrade is in %s. ", backupPath(stored))
		b.WriteString("To go back, stop cogmer, then copy membership.db and the rooms folder from that copy over the ones in " +
			homeDir() + ", and delete membership.db-wal and membership.db-shm there. " +
			"Do that only if cogmer has not run since, because the copy lacks every turn written after it was made, " +
			"and its counters would reuse numbers that colleagues hold.\n")
	}
	return b.String()
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
