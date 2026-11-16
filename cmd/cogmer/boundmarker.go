package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// A file in bound-sessions/ stands for each session bound to a room, so that the hook
// script can tell a session in no room by looking for a file, and end the hook there
// without starting this program or handing it the session's prompt (D-284).
//
// session_rooms in membership.db is the record. The files are a hint that is allowed
// to be wrong in one direction only: a file without a binding costs one call to the
// daemon, which answers that the session is in no room, while a binding without a file
// would stop capture with nothing anywhere saying so. Every write below is ordered for
// that: a file is written before its binding is committed and removed after its leave
// is, and nothing here removes a file except a leave.

const boundDirName = "bound-sessions"

func boundDir() string { return filepath.Join(homeDir(), boundDirName) }

// markerName is the file that stands for a session, or "" for an identifier the hook
// script would not look up, which gets no file. The script calls the daemon for any
// identifier holding a character outside the ones accepted here, so such a session is
// served as if it had a file. A Codex session is stored as "codex:<thread>" and its
// file is "codex_<thread>", which is what the script builds from the thread it is given.
func markerName(sessionID string) string {
	name := strings.Replace(sessionID, ":", "_", 1)
	if name == "" || len(name) > 200 {
		return ""
	}
	for _, c := range name {
		ok := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_'
		if !ok {
			return ""
		}
	}
	return name
}

func writeMarker(dir, sessionID string) error {
	name := markerName(sessionID)
	if name == "" {
		return nil
	}
	return os.WriteFile(filepath.Join(dir, name), nil, 0o600)
}

// boundSessions lists every session in a room now.
func (m *Membership) boundSessions() ([]string, error) {
	rows, err := m.db.Query(`SELECT session_id FROM session_rooms WHERE left_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ensureMarkers makes the directory exist with a file for every session bound now, or
// makes it not exist at all.
//
// The script calls the daemon for every session while the directory is missing, so
// the directory appears only once it is full: it is filled under another name and
// renamed into place. A store written before these files existed gets them the first
// time it is opened. When a file cannot be written the directory is removed, because
// without it every session is served as it would be without the files, and with it a
// bound session missing its file would be served as one in no room.
func (m *Membership) ensureMarkers() {
	if err := m.fillMarkers(); err != nil {
		abandonMarkers(err)
	}
}

// markBound writes the file for a session about to be bound, after making sure the
// directory holds every other bound session's file.
func (m *Membership) markBound(sessionID string) {
	m.ensureMarkers()
	if err := writeMarker(boundDir(), sessionID); err != nil && !errors.Is(err, os.ErrNotExist) {
		abandonMarkers(err)
	}
}

// markLeft removes the file of a session that has left, and writes it again if the
// session was bound once more meanwhile by another process.
func (m *Membership) markLeft(sessionID string) {
	name := markerName(sessionID)
	if name == "" {
		return
	}
	_ = os.Remove(filepath.Join(boundDir(), name))
	if _, ok := m.RoomForSession(sessionID); ok {
		m.markBound(sessionID)
	}
}

func abandonMarkers(err error) {
	log.Printf("bound-session files: %v; removing %s, so every hook calls the daemon", err, boundDir())
	_ = os.RemoveAll(boundDir())
}

func (m *Membership) fillMarkers() error {
	dir := boundDir()
	bound, err := m.boundSessions()
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		tmp, err := os.MkdirTemp(homeDir(), "."+boundDirName+"-")
		if err != nil {
			return err
		}
		for _, s := range bound {
			if err := writeMarker(tmp, s); err != nil {
				_ = os.RemoveAll(tmp)
				return err
			}
		}
		if err := os.Rename(tmp, dir); err == nil {
			return nil
		}
		// Another process put the directory in place first. Its files may predate a
		// binding committed since, so the ones this process found are added to it.
		_ = os.RemoveAll(tmp)
	}
	for _, s := range bound {
		if err := writeMarker(dir, s); err != nil {
			return err
		}
	}
	return nil
}
