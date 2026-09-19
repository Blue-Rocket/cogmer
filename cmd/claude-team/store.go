package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Event is the immutable append-only unit defined in spec §7. Once written it
// is never mutated; peers relay it verbatim (§13).
type Event struct {
	EventID         string `json:"eventId"`
	PeerID          string `json:"peerId"`
	PeerSequence    int64  `json:"peerSequence"`
	RoomID          string `json:"roomId"`
	Timestamp       string `json:"timestamp"`
	UserID          string `json:"userId"`
	UserDisplayName string `json:"userDisplayName"`
	MachineID       string `json:"machineId"`
	// OriginSessionID names the agent session a turn came from. It is opaque:
	// that it is presently a Claude Code session id is a fact about the adapter
	// that captured it, not about this record.
	OriginSessionID string          `json:"originSessionId"`
	EventType       string          `json:"eventType"`
	Content         string          `json:"content"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	// Signature is made by the originating peer over signingBytes, and is what
	// distinguishes a relayed event from one the relayer composed (§13).
	Signature string `json:"signature,omitempty"`
	// SigVersion names the scheme Signature was made under, so that an event
	// signed before a field was added still verifies. Zero means "before this was
	// recorded", which is v2 -- the only scheme that existed then. It is
	// deliberately NOT covered by the signature: an attacker who alters it only
	// causes verification to fail, and every scheme is Ed25519 over length-prefixed
	// fields, so there is no weaker one to be downgraded to.
	SigVersion int `json:"sigVersion,omitempty"`
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
  origin_session_id TEXT,
  event_type        TEXT NOT NULL,
  content           TEXT,
  metadata          TEXT,
  signature         TEXT,
  -- Which signing scheme the signature column was made under. NULL or 0 means v2,
  -- the only scheme that existed before this column. An event is immutable and can
  -- never be re-signed, so this is what lets a field be added later without
  -- invalidating every event already written (D-058).
  sig_version       INTEGER,
  UNIQUE(peer_id, peer_sequence)
);
CREATE INDEX IF NOT EXISTS idx_events_room ON events(room_id, rowid_alias);

-- Events that could not be stored because their (peer_id, peer_sequence) was
-- already held by a DIFFERENT event. Quarantined rather than dropped: discarding
-- one destroys the evidence needed to tell lost peer state from forgery.
CREATE TABLE IF NOT EXISTS event_conflicts (
  id                INTEGER PRIMARY KEY AUTOINCREMENT,
  detected_at       TEXT NOT NULL,
  room_id           TEXT NOT NULL,
  peer_id           TEXT NOT NULL,
  peer_sequence     INTEGER NOT NULL,
  held_event_id     TEXT NOT NULL,
  incoming_event_id TEXT NOT NULL,
  incoming          TEXT NOT NULL
);

-- §19 delivery state, derived from evidence rather than recorded as intent.
--
-- A set rather than a watermark: delivery is confirmed per injection by
-- observing it in the transcript, so a lost injection leaves a hole that a
-- contiguous watermark could not represent. Can be compacted to a watermark
-- plus exceptions if the row count ever matters.
CREATE TABLE IF NOT EXISTS session_delivered (
  origin_session_id TEXT NOT NULL,
  event_id          TEXT NOT NULL,
  PRIMARY KEY (origin_session_id, event_id)
);

-- Injections offered to a session but not yet observed in its transcript.
-- content_hash is the sha256 of the exact block written to the hook's stdout;
-- Claude Code records that same text as a hook_success attachment, which is what
-- makes confirmation possible.
CREATE TABLE IF NOT EXISTS pending_injection (
  prompt_id         TEXT PRIMARY KEY,
  origin_session_id TEXT NOT NULL,
  event_ids         TEXT NOT NULL,
  content_hash      TEXT NOT NULL,
  created_at        TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_pending_session ON pending_injection(origin_session_id);
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
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// migrate brings an existing room's schema up to date.
//
// `CREATE TABLE IF NOT EXISTS` creates a table and then ignores it forever, so a
// room opened by a newer binary keeps whatever shape it was born with. Every
// column added since is therefore missing from every room that predates it --
// which surfaces not at open but at the first query that names it.
//
// Rooms are migrated rather than orphaned: a developer who has been working in one
// should not lose it to a change that mattered only to us.
func migrate(db *sql.DB) error {
	have := map[string]bool{}
	rows, err := db.Query(`SELECT name FROM pragma_table_info('events')`)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return err
		}
		have[n] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(have) == 0 {
		return nil // freshly created by the schema above
	}

	if have["claude_session_id"] && !have["origin_session_id"] {
		if _, err := db.Exec(`ALTER TABLE events RENAME COLUMN claude_session_id TO origin_session_id`); err != nil {
			return fmt.Errorf("migrate origin_session_id: %w", err)
		}
		have["origin_session_id"] = true
	}
	// Added columns, in the order they were introduced. Adding one here is the
	// whole of what a future migration needs.
	for _, add := range []struct{ name, ddl string }{
		{"signature", `ALTER TABLE events ADD COLUMN signature TEXT`},
		{"sig_version", `ALTER TABLE events ADD COLUMN sig_version INTEGER`},
	} {
		if !have[add.name] {
			if _, err := db.Exec(add.ddl); err != nil {
				return fmt.Errorf("migrate %s: %w", add.name, err)
			}
		}
	}
	return nil
}

// nextSequence returns this peer's next monotonic sequence number (§8).
// HighestSequence is the highest sequence this room holds from a peer. It is NOT
// where the next one comes from: a room that has lost its database reports 0 while
// the peer has issued hundreds, which is precisely the condition worth detecting
// rather than papering over (D-029).
func (s *Store) HighestSequence(peerID string) (int64, error) {
	var seq sql.NullInt64
	err := s.db.QueryRow(`SELECT MAX(peer_sequence) FROM events WHERE peer_id = ?`, peerID).Scan(&seq)
	if err != nil {
		return 0, err
	}
	return seq.Int64, nil
}

// Append durably commits a locally generated event before it is considered
// published (§23).
// The sequence is passed in rather than derived here, because deriving it from
// this database is exactly what breaks when this database is lost. It is reserved
// outside the room first -- see Membership.ReserveSequence and D-029.
func (s *Store) Append(seq int64, id *Identity, room, sessionID, eventType, content string, meta map[string]any) (*Event, error) {
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
		OriginSessionID: sessionID,
		EventType:       eventType,
		Content:         content,
		Metadata:        rawMeta,
	}
	if id.private != nil {
		ev.Sign(id.private)
	}
	res, err := s.Insert(ev)
	if err != nil {
		return nil, err
	}
	if res == InsertConflict {
		// Locally generated sequences come from MAX()+1, so this means the local
		// store is inconsistent rather than that a peer misbehaved.
		return nil, fmt.Errorf("local sequence %d for peer %s is already held by another event", ev.PeerSequence, ev.PeerID)
	}
	return ev, nil
}

// InsertResult distinguishes the three things that can happen on receipt. The
// middle and last cases look identical to a bare INSERT OR IGNORE, which is why
// this is not one.
type InsertResult int

const (
	// InsertStored: a new event was written.
	InsertStored InsertResult = iota
	// InsertDuplicate: the same event was already held. Ordinary redelivery --
	// this is what makes anti-entropy (§10) and transitive relay (§13) safe.
	InsertDuplicate
	// InsertConflict: this peer and sequence are already held by a DIFFERENT
	// event. Not a duplicate. Either the sending peer lost its state and
	// restarted its counter, or an event was forged.
	InsertConflict
)

func (r InsertResult) String() string {
	switch r {
	case InsertStored:
		return "stored"
	case InsertDuplicate:
		return "duplicate"
	default:
		return "conflict"
	}
}

// Insert stores an event, distinguishing redelivery from conflict.
//
// §8 makes (peerId, peerSequence) an event's identity, and §9 builds anti-entropy
// on the highest contiguous sequence per peer. Both assume a counter only moves
// forward. A peer that loses its room database restarts at 1 while others still
// hold higher numbers under its identifier, so everything it publishes afterwards
// collides.
//
// Absorbing that as a duplicate is silently wrong and unrecoverable: the sender
// believes it shared, the receiver never sees it, and anti-entropy cannot repair
// it because the sender's highest sequence is now BELOW what the receiver reports
// holding. So a conflict is quarantined and reported instead -- keeping the
// rejected event, because discarding it destroys the evidence that distinguishes
// lost state from forgery.
func (s *Store) Insert(ev *Event) (InsertResult, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return InsertStored, err
	}
	defer tx.Rollback()

	var heldID string
	err = tx.QueryRow(`SELECT event_id FROM events WHERE peer_id = ? AND peer_sequence = ?`,
		ev.PeerID, ev.PeerSequence).Scan(&heldID)
	switch {
	case err == sql.ErrNoRows:
		// nothing holds this slot; fall through to insert
	case err != nil:
		return InsertStored, err
	case heldID == ev.EventID:
		return InsertDuplicate, nil
	default:
		raw, _ := json.Marshal(ev)
		if _, err := tx.Exec(`
			INSERT INTO event_conflicts
			  (detected_at, room_id, peer_id, peer_sequence, held_event_id, incoming_event_id, incoming)
			VALUES (?,?,?,?,?,?,?)`,
			time.Now().UTC().Format(time.RFC3339Nano), ev.RoomID, ev.PeerID, ev.PeerSequence,
			heldID, ev.EventID, string(raw)); err != nil {
			return InsertConflict, err
		}
		return InsertConflict, tx.Commit()
	}

	// A different event id may still collide on event_id alone if the same event
	// arrives having been assigned a different sequence somewhere. Treat that as
	// redelivery too rather than writing it twice.
	var exists int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM events WHERE event_id = ?`, ev.EventID).Scan(&exists); err != nil {
		return InsertStored, err
	}
	if exists > 0 {
		return InsertDuplicate, nil
	}

	if _, err := tx.Exec(`
		INSERT INTO events
		  (event_id, peer_id, peer_sequence, room_id, timestamp, user_id,
		   user_display_name, machine_id, origin_session_id, event_type, content, metadata,
		   signature, sig_version)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		ev.EventID, ev.PeerID, ev.PeerSequence, ev.RoomID, ev.Timestamp, ev.UserID,
		ev.UserDisplayName, ev.MachineID, ev.OriginSessionID, ev.EventType, ev.Content,
		string(ev.Metadata), ev.Signature, ev.SigVersion); err != nil {
		return InsertStored, err
	}
	return InsertStored, tx.Commit()
}

// Conflict is a quarantined event, kept for diagnosis.
type Conflict struct {
	DetectedAt      string `json:"detectedAt"`
	RoomID          string `json:"roomId"`
	PeerID          string `json:"peerId"`
	PeerSequence    int64  `json:"peerSequence"`
	HeldEventID     string `json:"heldEventId"`
	IncomingEventID string `json:"incomingEventId"`
}

func (s *Store) ListConflicts() ([]Conflict, error) {
	rows, err := s.db.Query(`SELECT detected_at, room_id, peer_id, peer_sequence,
		held_event_id, incoming_event_id FROM event_conflicts ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Conflict
	for rows.Next() {
		var c Conflict
		if err := rows.Scan(&c.DetectedAt, &c.RoomID, &c.PeerID, &c.PeerSequence,
			&c.HeldEventID, &c.IncomingEventID); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func scanEvents(rows *sql.Rows) ([]Event, error) {
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var meta, sig sql.NullString
		var rowid int64
		if err := rows.Scan(&rowid, &e.EventID, &e.PeerID, &e.PeerSequence, &e.RoomID, &e.Timestamp,
			&e.UserID, &e.UserDisplayName, &e.MachineID, &e.OriginSessionID, &e.EventType, &e.Content,
			&meta, &sig, &e.SigVersion); err != nil {
			return nil, err
		}
		e.Signature = sig.String
		if meta.Valid && meta.String != "" {
			e.Metadata = json.RawMessage(meta.String)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

const selectCols = `rowid_alias, event_id, peer_id, peer_sequence, room_id, timestamp,
	user_id, user_display_name, machine_id, origin_session_id, event_type, content, metadata,
	signature, COALESCE(sig_version, 0)`

// orderBy is the deterministic total order every peer must agree on (review B1).
//
// Local insert order is NOT it: two peers receive events in different orders and
// would render the same room differently. (timestamp, peer_id, peer_sequence) is
// total without further tie-breaking, because §8 makes peer_id plus peer_sequence
// unique. Timestamps are RFC3339 UTC with fixed formatting, so lexicographic
// comparison is chronological.
//
// Injection uses the same order (review B2): anti-entropy delivers old events
// late, so arrival order is not chronological, and injecting out of sequence is
// how a referent resolves to the wrong turn.
const orderBy = `ORDER BY timestamp, peer_id, peer_sequence`

func (s *Store) ListRoom(room string) ([]Event, error) {
	rows, err := s.db.Query(`SELECT `+selectCols+` FROM events WHERE room_id = ? `+orderBy, room)
	if err != nil {
		return nil, err
	}
	return scanEvents(rows)
}

// UndeliveredFor returns room events this Claude session has not been confirmed
// to have received, excluding events the session itself produced -- those are
// already in its own context window (§19, step 2).
func (s *Store) UndeliveredFor(room, sessionID string) ([]Event, error) {
	rows, err := s.db.Query(`SELECT `+selectCols+` FROM events
		WHERE room_id = ? AND origin_session_id != ?
		  AND event_id NOT IN (SELECT event_id FROM session_delivered WHERE origin_session_id = ?)
		`+orderBy, room, sessionID, sessionID)
	if err != nil {
		return nil, err
	}
	return scanEvents(rows)
}

// HashBlock is the identity of an injected block: the sha256 of the exact text
// handed to the hook's stdout, whitespace-trimmed because Claude Code trims it
// before recording the attachment.
func HashBlock(text string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(text)))
	return hex.EncodeToString(sum[:])
}

// RecordPending notes that a block was offered to a session. Nothing is
// considered delivered until it is observed in that session's transcript.
func (s *Store) RecordPending(promptID, sessionID string, events []Event, text string) error {
	if len(events) == 0 || promptID == "" {
		return nil
	}
	ids := make([]string, len(events))
	for i, e := range events {
		ids[i] = e.EventID
	}
	blob, _ := json.Marshal(ids)
	_, err := s.db.Exec(`
		INSERT INTO pending_injection (prompt_id, origin_session_id, event_ids, content_hash, created_at)
		VALUES (?,?,?,?,?)
		ON CONFLICT(prompt_id) DO UPDATE SET
		  event_ids = excluded.event_ids, content_hash = excluded.content_hash`,
		promptID, sessionID, string(blob), HashBlock(text), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

// ConfirmDelivered marks as delivered every pending injection for this session
// whose block is present among the observed hashes. Transcript attachments
// accumulate across turns, so this is idempotent and self-healing: an injection
// missed at its own Stop is confirmed at a later one.
func (s *Store) ConfirmDelivered(sessionID string, observed []string) (int, error) {
	if len(observed) == 0 {
		return 0, nil
	}
	seen := make(map[string]bool, len(observed))
	for _, h := range observed {
		seen[h] = true
	}
	rows, err := s.db.Query(`SELECT prompt_id, event_ids, content_hash FROM pending_injection WHERE origin_session_id = ?`, sessionID)
	if err != nil {
		return 0, err
	}
	type pend struct{ promptID, ids string }
	var matched []pend
	for rows.Next() {
		var p pend
		var hash string
		if err := rows.Scan(&p.promptID, &p.ids, &hash); err != nil {
			rows.Close()
			return 0, err
		}
		if seen[hash] {
			matched = append(matched, p)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	n := 0
	for _, p := range matched {
		c, err := s.commitPendingRow(sessionID, p.promptID, p.ids)
		if err != nil {
			return n, err
		}
		n += c
	}
	return n, nil
}

// CommitPending is the degraded path for when a transcript carries no
// hook_success attachments at all -- the delivery evidence this design depends
// on is unavailable, so fall back to trusting that the turn carried the block.
func (s *Store) CommitPending(sessionID, promptID string) (int, error) {
	var ids string
	err := s.db.QueryRow(`SELECT event_ids FROM pending_injection WHERE prompt_id = ? AND origin_session_id = ?`,
		promptID, sessionID).Scan(&ids)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return s.commitPendingRow(sessionID, promptID, ids)
}

func (s *Store) commitPendingRow(sessionID, promptID, idsJSON string) (int, error) {
	var ids []string
	if err := json.Unmarshal([]byte(idsJSON), &ids); err != nil {
		return 0, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	for _, id := range ids {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO session_delivered (origin_session_id, event_id) VALUES (?,?)`,
			sessionID, id); err != nil {
			return 0, err
		}
	}
	if _, err := tx.Exec(`DELETE FROM pending_injection WHERE prompt_id = ?`, promptID); err != nil {
		return 0, err
	}
	return len(ids), tx.Commit()
}

// SyncState reports the highest CONTIGUOUS sequence held from each peer (§9).
//
// Contiguous, not maximum: a gap means the events after it have not truly been
// received, and reporting the maximum would tell a peer we hold events we lack,
// which it would then never send.
func (s *Store) SyncState(room string) (map[string]int64, error) {
	rows, err := s.db.Query(`SELECT peer_id, peer_sequence FROM events
		WHERE room_id = ? ORDER BY peer_id, peer_sequence`, room)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	state := map[string]int64{}
	for rows.Next() {
		var peer string
		var seq int64
		if err := rows.Scan(&peer, &seq); err != nil {
			return nil, err
		}
		if seq == state[peer]+1 {
			state[peer] = seq
		}
	}
	return state, rows.Err()
}

// EventsSince returns what a peer reporting `have` is missing from this room,
// in the deterministic order so a receiver sees them as this peer does.
func (s *Store) EventsSince(room string, have map[string]int64) ([]Event, error) {
	rows, err := s.db.Query(`SELECT `+selectCols+` FROM events WHERE room_id = ? `+orderBy, room)
	if err != nil {
		return nil, err
	}
	all, err := scanEvents(rows)
	if err != nil {
		return nil, err
	}
	var out []Event
	for _, e := range all {
		if e.PeerSequence > have[e.PeerID] {
			out = append(out, e)
		}
	}
	return out, nil
}

// PendingCount reports how many injections are awaiting confirmation.
func (s *Store) PendingCount(sessionID string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM pending_injection WHERE origin_session_id = ?`, sessionID).Scan(&n)
	return n, err
}

// PrunePending bounds unmatched pending rows, which accumulate only when
// injections genuinely never arrive.
func (s *Store) PrunePending(sessionID string, keep int) error {
	_, err := s.db.Exec(`
		DELETE FROM pending_injection
		WHERE origin_session_id = ? AND prompt_id NOT IN (
		  SELECT prompt_id FROM pending_injection WHERE origin_session_id = ?
		  ORDER BY created_at DESC LIMIT ?)`, sessionID, sessionID, keep)
	return err
}
