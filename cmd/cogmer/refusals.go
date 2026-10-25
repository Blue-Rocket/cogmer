package main

import (
	"log"
	"time"
)

// --- events this machine refused, so that they are offered once ---
//
// Sync is a pull against the highest unbroken run of each signer's sequences (§9).
// An event refused because its machine is past a cutoff or suspended is not stored,
// so it left a gap, and the sender offered it and everything after it on every poll
// for as long as the cutoff stood. The refusal is recorded instead, SyncState counts
// it as held, and the sender stops offering it. Nothing is lost by that: a refusal is
// cleared as soon as the event would count, such as when an invitation lifts the
// revocation or the user certifies a suspended machine again, and the event then
// arrives by the next sync from any machine that holds it (D-236, a revocation names
// the last turn of each of the revoked user's machines that counts; D-234, a machine
// not certified under the current shared key is suspended until it is certified
// again).
//
// An unverified author's events are not recorded here. D-054 (verification gates
// synchronization and injection) leaves them on offer, so that verifying brings the
// backlog.

// refusalSchema is the room database's table of refused sequences. It is apart from
// the events table, since nothing of a refused event is kept but where it sits in
// its signer's sequence and whose it is.
const refusalSchema = `
CREATE TABLE IF NOT EXISTS refused_events (
  signer        TEXT NOT NULL,
  peer_sequence INTEGER NOT NULL,
  -- The user the event named as its peer, and whether it was signed for a user under
  -- v4, which together say what has to change for it to count.
  user          TEXT NOT NULL,
  v4            INTEGER NOT NULL,
  refused_at    TEXT NOT NULL,
  PRIMARY KEY (signer, peer_sequence)
);`

// refusal is one refused sequence.
type refusal struct {
	Signer   string
	Sequence int64
	User     string
	V4       bool
}

// RecordRefusal records that this machine refused an event that does not count.
func (s *Store) RecordRefusal(ev *Event) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO refused_events (signer, peer_sequence, user, v4, refused_at)
		VALUES (?,?,?,?,?)`, ev.signerKey(), ev.PeerSequence, ev.PeerID, boolInt(ev.SigVersion == 4),
		time.Now().UTC().Format(time.RFC3339))
	return err
}

// Refusals is every refused sequence this room holds.
func (s *Store) Refusals() ([]refusal, error) {
	rows, err := s.db.Query(`SELECT signer, peer_sequence, user, v4 FROM refused_events ORDER BY signer, peer_sequence`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []refusal
	for rows.Next() {
		var r refusal
		var v4 int
		if err := rows.Scan(&r.Signer, &r.Sequence, &r.User, &v4); err != nil {
			return nil, err
		}
		r.V4 = v4 != 0
		out = append(out, r)
	}
	return out, rows.Err()
}

// ClearRefusal forgets one refused sequence, so that sync asks for it again.
func (s *Store) ClearRefusal(signer string, sequence int64) error {
	_, err := s.db.Exec(`DELETE FROM refused_events WHERE signer = ? AND peer_sequence = ?`, signer, sequence)
	return err
}

// reviewRefusals clears every refusal of a room whose event would count now, by the
// room's guest list and by what this machine holds for the event's user, so that an
// invitation that lifted a revocation, or a machine certified again, brings the
// events by the next sync (D-236, a later invitation that lifts the revocation lets
// the user's turns after the cutoff count again). It is run before each pull, and
// costs one query when the room has refused nothing. Called with d.mu held, since it
// writes the room's database.
func (d *Daemon) reviewRefusals(store *Store, roomID string) {
	refused, err := store.Refusals()
	if err != nil || len(refused) == 0 {
		return
	}
	list := d.guestList(roomID)
	states := map[string]*UserState{}
	cleared := 0
	for _, r := range refused {
		counts := list == nil || list.Counts(r.User, r.Signer, r.Sequence)
		if counts && r.V4 {
			st, ok := states[r.User]
			if !ok {
				st = d.members.UserState(r.User, nil)
				states[r.User] = st
			}
			counts = st.Counts(d.members.machineOfSigner(r.User, r.Signer), r.Signer, roomID, r.Sequence)
		}
		if !counts {
			continue
		}
		if err := store.ClearRefusal(r.Signer, r.Sequence); err != nil {
			log.Printf("sync: could not clear a refusal in %s: %v", roomID, err)
			continue
		}
		cleared++
	}
	if cleared > 0 {
		log.Printf("sync: %d event(s) refused earlier in %s count now, and are asked for again", cleared, roomID)
	}
}
