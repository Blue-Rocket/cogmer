package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Event is the immutable append-only unit defined in spec §7. Once written it
// is never mutated; peers relay it verbatim (§13).
type Event struct {
	EventID         string          `json:"eventId"`
	PeerID          string          `json:"peerId"`
	PeerSequence    int64           `json:"peerSequence"`
	RoomID          string          `json:"roomId"`
	Timestamp       string          `json:"timestamp"`
	UserID          string          `json:"userId"`
	UserDisplayName string          `json:"userDisplayName"`
	MachineID       string          `json:"machineId"`
	ClaudeSessionID string          `json:"claudeSessionId"`
	EventType       string          `json:"eventType"`
	Content         string          `json:"content"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
}

const (
	EventUserPrompt       = "USER_PROMPT"
	EventAssistantMessage = "ASSISTANT_MESSAGE"
)

type Store struct{ db *sql.DB }

const schema = `
CREATE TABLE IF NOT EXISTS events (
  rowid_alias       INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id          TEXT NOT NULL UNIQUE,
  peer_id           TEXT NOT NULL,
  peer_sequence     INTEGER NOT NULL,
  room_id           TEXT NOT NULL,
  timestamp         TEXT NOT NULL,
  user_id           TEXT,
  user_display_name TEXT,
  machine_id        TEXT,
  claude_session_id TEXT,
  event_type        TEXT NOT NULL,
  content           TEXT,
  metadata          TEXT,
  UNIQUE(peer_id, peer_sequence)
);
CREATE INDEX IF NOT EXISTS idx_events_room ON events(room_id, rowid_alias);

-- §19: which room events each local Claude session has already been shown.
CREATE TABLE IF NOT EXISTS session_context (
  claude_session_id TEXT PRIMARY KEY,
  last_delivered    INTEGER NOT NULL
);
`

// roomExists reports whether a room already has a local replica.
func roomExists(room string) bool {
	_, err := os.Stat(filepath.Join(homeDir(), "rooms", room+".db"))
	return err == nil
}

func OpenStore(room string) (*Store, error) {
	dir := filepath.Join(homeDir(), "rooms")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, room+".db")
	// WAL keeps the UI's reads from blocking hook writes.
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// nextSequence returns this peer's next monotonic sequence number (§8).
func (s *Store) nextSequence(peerID string) (int64, error) {
	var seq sql.NullInt64
	err := s.db.QueryRow(`SELECT MAX(peer_sequence) FROM events WHERE peer_id = ?`, peerID).Scan(&seq)
	if err != nil {
		return 0, err
	}
	return seq.Int64 + 1, nil
}

// Append durably commits a locally generated event before it is considered
// published (§23).
func (s *Store) Append(id *Identity, room, sessionID, eventType, content string, meta map[string]any) (*Event, error) {
	seq, err := s.nextSequence(id.PeerID)
	if err != nil {
		return nil, err
	}
	var rawMeta json.RawMessage
	if meta != nil {
		if b, err := json.Marshal(meta); err == nil {
			rawMeta = b
		}
	}
	ev := &Event{
		EventID:         fmt.Sprintf("%013d-%s", time.Now().UTC().UnixMilli(), randomID()),
		PeerID:          id.PeerID,
		PeerSequence:    seq,
		RoomID:          room,
		Timestamp:       time.Now().UTC().Format(time.RFC3339Nano),
		UserID:          id.UserID,
		UserDisplayName: id.UserDisplayName,
		MachineID:       id.MachineID,
		ClaudeSessionID: sessionID,
		EventType:       eventType,
		Content:         content,
		Metadata:        rawMeta,
	}
	return ev, s.Insert(ev)
}

// Insert is idempotent: re-delivery of an event already held is a no-op, which
// is what makes anti-entropy sync (§10) and transitive relay (§13) safe.
func (s *Store) Insert(ev *Event) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO events
		  (event_id, peer_id, peer_sequence, room_id, timestamp, user_id,
		   user_display_name, machine_id, claude_session_id, event_type, content, metadata)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		ev.EventID, ev.PeerID, ev.PeerSequence, ev.RoomID, ev.Timestamp, ev.UserID,
		ev.UserDisplayName, ev.MachineID, ev.ClaudeSessionID, ev.EventType, ev.Content, string(ev.Metadata))
	return err
}

func scanEvents(rows *sql.Rows) ([]Event, error) {
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var meta sql.NullString
		var rowid int64
		if err := rows.Scan(&rowid, &e.EventID, &e.PeerID, &e.PeerSequence, &e.RoomID, &e.Timestamp,
			&e.UserID, &e.UserDisplayName, &e.MachineID, &e.ClaudeSessionID, &e.EventType, &e.Content, &meta); err != nil {
			return nil, err
		}
		if meta.Valid && meta.String != "" {
			e.Metadata = json.RawMessage(meta.String)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

const selectCols = `rowid_alias, event_id, peer_id, peer_sequence, room_id, timestamp,
	user_id, user_display_name, machine_id, claude_session_id, event_type, content, metadata`

func (s *Store) ListRoom(room string) ([]Event, error) {
	rows, err := s.db.Query(`SELECT `+selectCols+` FROM events WHERE room_id = ? ORDER BY rowid_alias`, room)
	if err != nil {
		return nil, err
	}
	return scanEvents(rows)
}

// UndeliveredFor returns room events this Claude session has not yet been
// shown, excluding events the session itself produced -- those are already in
// its own context window (§19, step 2).
func (s *Store) UndeliveredFor(room, sessionID string) ([]Event, int64, error) {
	var watermark int64
	err := s.db.QueryRow(`SELECT last_delivered FROM session_context WHERE claude_session_id = ?`, sessionID).Scan(&watermark)
	if err != nil && err != sql.ErrNoRows {
		return nil, 0, err
	}
	rows, err := s.db.Query(`SELECT `+selectCols+` FROM events
		WHERE room_id = ? AND rowid_alias > ? AND claude_session_id != ?
		ORDER BY rowid_alias`, room, watermark, sessionID)
	if err != nil {
		return nil, 0, err
	}
	evs, err := scanEvents(rows)
	if err != nil {
		return nil, 0, err
	}
	var high int64
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(rowid_alias),0) FROM events WHERE room_id = ?`, room).Scan(&high); err != nil {
		return nil, 0, err
	}
	return evs, high, nil
}

// MarkDelivered advances the session's incorporated-event watermark so the same
// teammate turns are never injected twice (§19, "Do not repeatedly inject the
// entire room").
func (s *Store) MarkDelivered(sessionID string, watermark int64) error {
	_, err := s.db.Exec(`
		INSERT INTO session_context (claude_session_id, last_delivered) VALUES (?, ?)
		ON CONFLICT(claude_session_id) DO UPDATE SET last_delivered = excluded.last_delivered`,
		sessionID, watermark)
	return err
}
