package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// fillRoom stores events from three signers with gaps, a refused sequence and a
// quarantined conflict, which are the cases SyncState's contiguity has to get right.
func fillRoom(t *testing.T, s *Store) {
	t.Helper()
	insert := func(peer string, seq int64, ts string) {
		t.Helper()
		ev := &Event{EventID: fmt.Sprintf("%s-%d", peer, seq), PeerID: peer, PeerSequence: seq, RoomID: "test",
			Timestamp: ts, EventType: EventUserPrompt, Content: strings.Repeat("x", 4000)}
		if res, err := s.Insert(ev); err != nil || res != InsertStored {
			t.Fatalf("%s %d: %v %v", peer, seq, res, err)
		}
	}
	for seq := int64(1); seq <= 40; seq++ {
		insert("peer-alice", seq, fmt.Sprintf("2026-11-25T10:%02d:00Z", seq))
	}
	for _, seq := range []int64{1, 2, 3, 5, 6} { // 4 is missing
		insert("peer-bob", seq, fmt.Sprintf("2026-11-25T10:%02d:30Z", seq))
	}
	for _, seq := range []int64{1, 3} { // 2 is refused, so it counts as held
		insert("peer-carol", seq, "2026-11-25T11:00:00Z")
	}
	if err := s.RecordRefusal(&Event{PeerID: "peer-carol", PeerSequence: 2}); err != nil {
		t.Fatal(err)
	}
	clash := &Event{EventID: "peer-alice-7", PeerID: "peer-dave", PeerSequence: 1, RoomID: "test",
		Timestamp: "2026-11-25T12:00:00Z", EventType: EventUserPrompt, Content: "another event under a held identifier"}
	if res, err := s.Insert(clash); err != nil || res != InsertConflict {
		t.Fatalf("the clash was %v (%v), want a conflict", res, err)
	}
}

// What a poll reads through the index is what a scan of the whole room gives: the
// highest contiguous sequence of each signer, counting refused and quarantined ones as
// held, and every event after what the asker holds, in the room's order.
func TestAPollReadThroughTheIndexMatchesAScanOfTheWholeRoom(t *testing.T) {
	s := testStore(t)
	fillRoom(t, s)

	scan := map[string]int64{}
	rows, err := s.db.Query(`SELECT k, seq FROM (
		  SELECT COALESCE(signer, peer_id) AS k, peer_sequence AS seq FROM events WHERE room_id = ?
		  UNION SELECT signer, peer_sequence FROM refused_events
		  UNION SELECT COALESCE(signer, peer_id), peer_sequence FROM event_conflicts WHERE room_id = ?)
		ORDER BY k, seq`, "test", "test")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var k string
		var seq int64
		if err := rows.Scan(&k, &seq); err != nil {
			t.Fatal(err)
		}
		if seq == scan[k]+1 {
			scan[k] = seq
		}
	}
	rows.Close()
	got, err := s.SyncState("test")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int64{"peer-alice": 40, "peer-bob": 3, "peer-carol": 3, "peer-dave": 1}
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(scan, want) {
		t.Fatalf("sync state %v, the whole room's scan %v, want %v", got, scan, want)
	}

	all, err := s.ListRoom("test")
	if err != nil {
		t.Fatal(err)
	}
	for _, have := range []map[string]int64{
		nil,
		{"peer-alice": 40, "peer-bob": 6, "peer-carol": 3},
		{"peer-alice": 37, "peer-bob": 2},
		{"peer-alice": 39, "peer-elsewhere": 5},
	} {
		var wantEvs []string
		for _, e := range all {
			if e.PeerSequence > have[e.signerKey()] {
				wantEvs = append(wantEvs, e.EventID)
			}
		}
		evs, err := s.EventsSince("test", have)
		if err != nil {
			t.Fatal(err)
		}
		var gotEvs []string
		for _, e := range evs {
			gotEvs = append(gotEvs, e.EventID)
		}
		if !reflect.DeepEqual(gotEvs, wantEvs) {
			t.Errorf("asked with %v: got %v, want %v", have, gotEvs, wantEvs)
		}
	}
}

// An idle poll's cost must not grow with the room's history. On 2026-11-25 a pair of
// daemons with one day of turns in a room spent about ten times the CPU of an empty
// room on idle polls and wrote about 8MB a second of SQLite's temporary files, because
// each poll read and sorted every event the room held (a6ac576). The plans of the poll's
// queries say whether they read the table or sort it.
func TestAnIdlePollReadsNeitherTheRoomsContentNorSortsItsHistory(t *testing.T) {
	s := testStore(t)
	fillRoom(t, s)
	plan := func(query string, args ...any) string {
		t.Helper()
		rows, err := s.db.Query(`EXPLAIN QUERY PLAN `+query, args...)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var lines []string
		for rows.Next() {
			var id, parent, notused int
			var detail string
			if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
				t.Fatal(err)
			}
			lines = append(lines, detail)
		}
		return strings.Join(lines, "\n")
	}
	state := plan(heldSequencesQuery, "test")
	if !strings.Contains(state, "COVERING INDEX idx_events_room_signer") || strings.Contains(state, "TEMP B-TREE") {
		t.Errorf("the sync state's query reads the table or sorts it:\n%s", state)
	}
	signers := plan(nextSignerQuery, "test", "")
	if !strings.Contains(signers, "COVERING INDEX idx_events_room_signer") || strings.Contains(signers, "TEMP B-TREE") {
		t.Errorf("listing the signers reads the table or sorts it:\n%s", signers)
	}
	since := plan(eventsAfterQuery(2),
		"test", "peer-alice", 40, "test", "peer-bob", 6)
	if !strings.Contains(since, "MULTI-INDEX OR") || strings.Contains(since, "SCAN events") {
		t.Errorf("the missing events are found by reading the whole table:\n%s", since)
	}
}
