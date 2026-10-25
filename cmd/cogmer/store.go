package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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
	// its host-specific format is a fact about the adapter
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
	// Signer is the key that signed the event. Under v3 it is always the peer's own
	// key, PeerID, and it is not among v3's signed fields, which never change (D-058,
	// signature schemes are kept, never replaced). It is stored so that a machine
	// signing for its user, whose events name the user as their peer, can be told
	// apart from the user's other machines (D-239, an event a machine signs for its
	// user names the user and the signing key as signed fields).
	Signer string `json:"signer,omitempty"`
	// IdentityVersion and RemovalsSeen are the shared-key version the signing
	// machine holds and the hashes, in hex and sorted, of the removals it has seen
	// for its user. Both are signed fields under v4 and absent under v3 (D-239, an
	// event a machine signs for its user names the shared-key version and the
	// removals seen), and a receiver that lacks them withholds the event from the
	// assistant until it has fetched them (D-221, an event whose identity version the receiver lacks is withheld until fetched).
	IdentityVersion uint64   `json:"identityVersion,omitempty"`
	RemovalsSeen    []string `json:"removalsSeen,omitempty"`
	// Chain ties Signer to the user PeerID names, and is not signed: it is the
	// certificates' own signatures that make it true, so an event can be checked
	// wherever it arrives, relayed or fetched again after a lost database (D-239, an event a machine signs for its user names the signing key).
	Chain *Chain `json:"chain,omitempty"`
}

const (
	EventUserPrompt       = "USER_PROMPT"
	EventAssistantMessage = "ASSISTANT_MESSAGE"
	// EventUserImage carries images that a host showed after the prompt they belong to was
	// published. It names the prompt's event, and is no turn of the room (D-214).
	EventUserImage = "USER_IMAGE"
)

type Store struct {
	db *sql.DB
	// imageDir holds the room's pasted images as files named by their hash (D-213). It is
	// fixed when the store opens, since the state directory is resolved from the environment.
	imageDir string
}

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
  -- invalidating every event already written (D-058, signature schemes are kept, never replaced).
  sig_version       INTEGER,
  -- The key that signed the event. For a v3 event it is peer_id, and an event a
  -- machine signs for its user names the user as peer_id and the machine here
  -- (D-239, an event a machine signs for its user names the signing key). Its index
  -- is made in migrate, since a table made before the column cannot be indexed on
  -- it until the column is added.
  signer            TEXT,
  -- What a v4 event carries beyond v3's fields: the chain that ties its signer to
  -- its user, as JSON, the shared-key version its signer holds and the hex hashes of
  -- the removals it has seen, as a JSON array (D-239, an event a machine signs for its user names the signing key). NULL for a v3 event.
  chain             TEXT,
  identity_version  INTEGER,
  removals_seen     TEXT,
  -- An event is identified by its signer, room and sequence, since each machine of a
  -- user numbers its own events (D-239, an event a machine signs for its user names the signing key; D-027, a sequence conflict is quarantined).
  -- For a v3 event the signer is the peer, so this is the old (peer_id,
  -- peer_sequence) rule.
  UNIQUE(signer, peer_sequence)
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
  incoming          TEXT NOT NULL,
  -- The key whose sequence collided, which for a v3 event is peer_id (D-239, an event a machine signs for its user names the signing key).
  signer            TEXT
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

-- Where each pasted image a room's events name stands on this machine (D-213). The event
-- names the image and its hash; this is whether the bytes are here yet. state is 'have' when
-- the file is stored and 'wanted' while it is still to be fetched from a peer.
CREATE TABLE IF NOT EXISTS images (
  sha256     TEXT PRIMARY KEY,
  media_type TEXT NOT NULL,
  size       INTEGER NOT NULL,
  state      TEXT NOT NULL,
  attempts   INTEGER NOT NULL DEFAULT 0,
  next_try   TEXT NOT NULL DEFAULT ''
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
	if err := migrate(db); err != nil {
		return nil, err
	}
	if _, err := db.Exec(refusalSchema); err != nil {
		return nil, fmt.Errorf("init refusals: %w", err)
	}
	return &Store{db: db, imageDir: filepath.Join(dir, room+".images")}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// migrate brings an existing room's schema up to date.
//
// `CREATE TABLE IF NOT EXISTS` creates a table and then ignores it forever, so a
// room opened by a newer binary keeps whatever shape it was born with. Every
// column added since is therefore missing from every room that predates it --
// which surfaces not at open but at the first query that names it.
//
// Rooms are migrated rather than orphaned: a person who has been working in one
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
	for _, c := range []string{"signer TEXT", "chain TEXT", "identity_version INTEGER", "removals_seen TEXT"} {
		name, typ, _ := strings.Cut(c, " ")
		if err := addColumnIfMissing(db, "events", name, typ); err != nil {
			return fmt.Errorf("migrate %s: %w", name, err)
		}
	}
	if err := addColumnIfMissing(db, "event_conflicts", "signer", "TEXT"); err != nil {
		return fmt.Errorf("migrate event_conflicts signer: %w", err)
	}
	// Every event stored before the column was signed under v3 or earlier, by its
	// peer's own key, so its signer is its peer. Run on every open, and cheap once
	// done since the index finds the rows with no signer, so an event an older build
	// wrote after a downgrade gains its signer too.
	for _, stmt := range []string{
		`CREATE INDEX IF NOT EXISTS idx_events_signer ON events(signer, peer_sequence)`,
		`UPDATE events SET signer = peer_id WHERE signer IS NULL`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate signer: %w", err)
		}
	}
	return rekeyEventsOnSigner(db)
}

// rekeyEventsOnSigner rebuilds an events table whose uniqueness is on (peer_id,
// peer_sequence) so that it is on (signer, peer_sequence). Two machines of one user
// name the same peer and number their events from 1 each (D-239, an event a machine signs for its user names the signing key), so under the old
// rule the second machine's events would all be set aside as conflicts. SQLite
// cannot drop a constraint in place, so the table is copied. Every row already has a
// signer, filled above, and for those rows signer is peer_id, so nothing that was
// unique stops being so.
func rekeyEventsOnSigner(db *sql.DB) error {
	var ddl string
	if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'events'`).Scan(&ddl); err != nil {
		return fmt.Errorf("read events schema: %w", err)
	}
	if !strings.Contains(strings.ReplaceAll(ddl, " ", ""), "UNIQUE(peer_id,peer_sequence)") {
		return nil
	}
	cols := `rowid_alias, event_id, peer_id, peer_sequence, room_id, timestamp, user_id,
		user_display_name, machine_id, origin_session_id, event_type, content, metadata,
		signature, sig_version, signer, chain, identity_version, removals_seen`
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	newTable := strings.Replace(schema[strings.Index(schema, "CREATE TABLE IF NOT EXISTS events ("):strings.Index(schema, "CREATE INDEX IF NOT EXISTS idx_events_room")],
		"CREATE TABLE IF NOT EXISTS events (", "CREATE TABLE events_rekeyed (", 1)
	for _, stmt := range []string{
		newTable,
		`INSERT INTO events_rekeyed (` + cols + `) SELECT ` + cols + ` FROM events`,
		`DROP TABLE events`,
		`ALTER TABLE events_rekeyed RENAME TO events`,
		`CREATE INDEX IF NOT EXISTS idx_events_room ON events(room_id, rowid_alias)`,
		`CREATE INDEX IF NOT EXISTS idx_events_signer ON events(signer, peer_sequence)`,
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("rekey events on signer: %w", err)
		}
	}
	return tx.Commit()
}

// nextSequence returns this peer's next monotonic sequence number (§8).
// HighestSequence is the highest sequence this room holds from a peer. It is NOT
// where the next one comes from: a room that has lost its database reports 0 while
// the peer has issued hundreds, which is precisely the condition worth detecting
// rather than papering over (D-029).
//
// It is asked by signer, since a sequence is the signing machine's own (D-239, an event a machine signs for its user names the signing key); for
// a v3 event the signer is the peer.
func (s *Store) HighestSequence(signer string) (int64, error) {
	var seq sql.NullInt64
	err := s.db.QueryRow(`SELECT MAX(peer_sequence) FROM events WHERE signer = ?`, signer).Scan(&seq)
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
	return s.AppendAs(seq, id, nil, room, sessionID, eventType, content, meta)
}

// AppendAs is Append for a machine that may speak for a user: the event names the
// user as its peer and is signed under v4 with the removals this machine has seen
// (D-239, an event a machine signs for its user names the signing key), and a machine with no user identity writes the v3 event Append always
// wrote.
func (s *Store) AppendAs(seq int64, id *Identity, removalsSeen []string, room, sessionID, eventType, content string, meta map[string]any) (*Event, error) {
	var rawMeta json.RawMessage
	if meta != nil {
		if b, err := json.Marshal(meta); err == nil {
			rawMeta = b
		}
	}
	ev := &Event{
		EventID:         fmt.Sprintf("%013d-%s", time.Now().UTC().UnixMilli(), randomID()),
		PeerID:          id.Peer(),
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
		ev.SignAs(id, removalsSeen)
	}
	res, err := s.Insert(ev)
	if err != nil {
		return nil, err
	}
	if res == InsertConflict {
		// Locally generated sequences come from MAX()+1, so this means the local
		// store is inconsistent rather than that a peer misbehaved.
		return nil, fmt.Errorf("local sequence %d for signer %s is already held by another event", ev.PeerSequence, ev.signerKey())
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

	signer := ev.signerKey()
	// An event is a redelivery only when the event held under its slot, or under its
	// identifier, is the same in every signed field. One that differs is a conflict,
	// whichever of the two it shares: an identifier is chosen by its signer, so a
	// signed event can carry another's, and treating it as a redelivery would drop
	// whichever arrived second (D-027, a sequence conflict is quarantined).
	held, err := heldEvent(tx, `signer = ? AND peer_sequence = ?`, signer, ev.PeerSequence)
	if err != nil {
		return InsertStored, err
	}
	if held == nil {
		if held, err = heldEvent(tx, `event_id = ?`, ev.EventID); err != nil {
			return InsertStored, err
		}
	}
	if held != nil {
		if sameSignedEvent(held, ev) {
			return InsertDuplicate, nil
		}
		raw, _ := json.Marshal(ev)
		if _, err := tx.Exec(`
			INSERT INTO event_conflicts
			  (detected_at, room_id, peer_id, peer_sequence, held_event_id, incoming_event_id, incoming, signer)
			SELECT ?,?,?,?,?,?,?,?
			WHERE NOT EXISTS (SELECT 1 FROM event_conflicts
			  WHERE signer = ? AND peer_sequence = ? AND held_event_id = ? AND incoming = ?)`,
			time.Now().UTC().Format(time.RFC3339Nano), ev.RoomID, ev.PeerID, ev.PeerSequence,
			held.EventID, ev.EventID, string(raw), signer,
			signer, ev.PeerSequence, held.EventID, string(raw)); err != nil {
			return InsertConflict, err
		}
		return InsertConflict, tx.Commit()
	}

	// The v4 fields are stored only for a v4 event, where they are signed or, for
	// the chain, checked against signatures of their own. A v3 event's claims in
	// them are not stored, since nothing checked them.
	var chain, removals sql.NullString
	var version sql.NullInt64
	if ev.SigVersion == 4 {
		if ev.Chain != nil {
			if b, err := json.Marshal(ev.Chain); err == nil {
				chain = sql.NullString{String: string(b), Valid: true}
			}
		}
		if b, err := json.Marshal(ev.RemovalsSeen); err == nil && len(ev.RemovalsSeen) > 0 {
			removals = sql.NullString{String: string(b), Valid: true}
		}
		version = sql.NullInt64{Int64: int64(ev.IdentityVersion), Valid: true}
	}
	if _, err := tx.Exec(`
		INSERT INTO events
		  (event_id, peer_id, peer_sequence, room_id, timestamp, user_id,
		   user_display_name, machine_id, origin_session_id, event_type, content, metadata,
		   signature, sig_version, signer, chain, identity_version, removals_seen)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		ev.EventID, ev.PeerID, ev.PeerSequence, ev.RoomID, ev.Timestamp, ev.UserID,
		ev.UserDisplayName, ev.MachineID, ev.OriginSessionID, ev.EventType, ev.Content,
		string(ev.Metadata), ev.Signature, ev.SigVersion, signer, chain, version, removals); err != nil {
		return InsertStored, err
	}
	return InsertStored, tx.Commit()
}

// heldEvent is the event the condition finds, or nil when none does.
func heldEvent(tx *sql.Tx, where string, args ...any) (*Event, error) {
	rows, err := tx.Query(`SELECT `+selectCols+` FROM events WHERE `+where+` LIMIT 1`, args...)
	if err != nil {
		return nil, err
	}
	evs, err := scanEvents(rows)
	if err != nil || len(evs) == 0 {
		return nil, err
	}
	return &evs[0], nil
}

// sameSignedEvent reports whether two events agree in their scheme and in every
// field a scheme signs. The signature itself is not compared, since two encodings of
// one signature are the same event.
func sameSignedEvent(a, b *Event) bool {
	if a.SigVersion != b.SigVersion || a.signerKey() != b.signerKey() {
		return false
	}
	if !bytes.Equal(a.eventBytes(""), b.eventBytes("")) {
		return false
	}
	if a.SigVersion != 4 {
		return true
	}
	return a.IdentityVersion == b.IdentityVersion && slices.Equal(a.RemovalsSeen, b.RemovalsSeen)
}

// signerKey is the key that signed an event, which is what its sequence numbers
// (D-239, an event a machine signs for its user names the signing key). Under v4 it is the signed Signer field. Under v3 and earlier it is the
// peer, whatever the unsigned Signer field claims, since only the peer's own key
// signs those schemes.
func (e *Event) signerKey() string {
	if e.SigVersion == 4 && e.Signer != "" {
		return e.Signer
	}
	return e.PeerID
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
		var meta, sig, chain, removals sql.NullString
		var version sql.NullInt64
		var rowid int64
		if err := rows.Scan(&rowid, &e.EventID, &e.PeerID, &e.PeerSequence, &e.RoomID, &e.Timestamp,
			&e.UserID, &e.UserDisplayName, &e.MachineID, &e.OriginSessionID, &e.EventType, &e.Content,
			&meta, &sig, &e.SigVersion, &e.Signer, &chain, &version, &removals); err != nil {
			return nil, err
		}
		e.Signature = sig.String
		e.IdentityVersion = uint64(version.Int64)
		if chain.Valid && chain.String != "" {
			var c Chain
			if json.Unmarshal([]byte(chain.String), &c) == nil {
				e.Chain = &c
			}
		}
		if removals.Valid && removals.String != "" {
			_ = json.Unmarshal([]byte(removals.String), &e.RemovalsSeen)
		}
		if meta.Valid && meta.String != "" {
			e.Metadata = json.RawMessage(meta.String)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

const selectCols = `rowid_alias, event_id, peer_id, peer_sequence, room_id, timestamp,
	user_id, user_display_name, machine_id, origin_session_id, event_type, content, metadata,
	signature, COALESCE(sig_version, 0), COALESCE(signer, peer_id), chain, identity_version, removals_seen`

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
//
// The signer sits between the peer and the sequence because two machines of one
// user name one peer and number their events each from 1 (D-239, an event a machine signs for its user names the signing key). For a v3 event it
// is the peer, so the order of every room written before is unchanged.
const orderBy = `ORDER BY timestamp, peer_id, signer, peer_sequence`

func (s *Store) ListRoom(room string) ([]Event, error) {
	rows, err := s.db.Query(`SELECT `+selectCols+` FROM events WHERE room_id = ? `+orderBy, room)
	if err != nil {
		return nil, err
	}
	return scanEvents(rows)
}

// UndeliveredFor returns room events this agent session has not been confirmed
// to have received, excluding events the session itself produced -- those are
// already in its own context window (§19). Changes to the room's guest list are
// never given to an assistant, since they are no turn of the room (D-235, a room's
// one guest list, changed by any guest through events in the room).
func (s *Store) UndeliveredFor(room, sessionID string) ([]Event, error) {
	rows, err := s.db.Query(`SELECT `+selectCols+` FROM events
		WHERE room_id = ? AND origin_session_id != ? AND event_type NOT IN (?, ?, ?)
		  AND event_id NOT IN (SELECT event_id FROM session_delivered WHERE origin_session_id = ?)
		`+orderBy, room, sessionID, EventUserImage, GuestInvite, GuestRevoke, sessionID)
	if err != nil {
		return nil, err
	}
	return scanEvents(rows)
}

// DeliveredEventIDs is every event any session on this machine has been confirmed to
// hold, for a view that says which turns the model has been given.
func (s *Store) DeliveredEventIDs() (map[string]bool, error) {
	rows, err := s.db.Query(`SELECT DISTINCT event_id FROM session_delivered`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
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
//
// It is keyed by signer, the key whose sequence it is (D-239, an event a machine signs for its user names the signing key). For a v3 event the
// signer is the peer, so a peer on an older build reads the same map it always did.
//
// A sequence this machine refused because its machine was past a cutoff or suspended
// counts as held, so that it is not asked for again on every poll and a cut-off
// machine's later events do not leave a gap that keeps everything after it on offer
// (D-236, a revocation names the last turn that counts). A refusal is cleared when the
// event would count, and the sequence is then asked for again. A quarantined
// conflict counts as held too, so that an event whose identifier another event holds
// is quarantined once rather than asked for on every poll (D-027, a sequence
// conflict is quarantined).
func (s *Store) SyncState(room string) (map[string]int64, error) {
	rows, err := s.db.Query(`SELECT k, seq FROM (
		  SELECT COALESCE(signer, peer_id) AS k, peer_sequence AS seq FROM events WHERE room_id = ?
		  UNION SELECT signer, peer_sequence FROM refused_events
		  UNION SELECT COALESCE(signer, peer_id), peer_sequence FROM event_conflicts WHERE room_id = ?)
		ORDER BY k, seq`, room, room)
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
		if e.PeerSequence > have[e.signerKey()] {
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
