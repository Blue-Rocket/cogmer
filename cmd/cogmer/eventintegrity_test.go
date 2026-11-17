package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// An event signed for another room is that room's, and a machine relaying it in this
// room's sync answer cannot make it take the slot of this room's own event with the
// same signer and sequence.
func TestAnEventSignedForAnotherRoomIsRejected(t *testing.T) {
	const room = "room-integrity"
	a := newOfflinePeer(t, room, "misty-canyon")
	b := newOfflinePeer(t, room, "misty-canyon")
	introduce(t, a, b, room)

	elsewhere := Event{EventID: "elsewhere-1", PeerID: a.d.id.PeerID, PeerSequence: 1, RoomID: "room-other",
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano), EventType: EventUserPrompt, Content: "said in another room"}
	elsewhere.Sign(a.d.id.private)
	relay := servePeerTLSHandler(t, a.d, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, syncResponse{Protocol: wireVersion, RoomID: room, Events: []wireEvent{toWire(elsewhere)}})
	}))
	if n, err := b.d.pullFrom(relay); err != nil || n != 0 {
		t.Fatalf("pulling the copied event stored %d: %v", n, err)
	}

	a.say(t, room, "s", EventUserPrompt, "said in this room")
	if n := pull(t, b, a); n != 1 {
		t.Fatalf("the room's own event stored %d", n)
	}
	if !has(t, b, room, "said in this room") || has(t, b, room, "said in another room") {
		t.Fatal("the room holds the other room's event, or lost its own")
	}
	if cs, _ := b.store.ListConflicts(); len(cs) != 0 {
		t.Fatalf("the room's own event was quarantined: %+v", cs)
	}
}

// An event that carries the identifier of one held is a redelivery only when every
// signed field matches. One that differs, at the same slot or another, is quarantined,
// and the held event stands.
func TestAnEventReusingAHeldIdentifierIsAConflict(t *testing.T) {
	s := testStore(t)
	genuine := &Event{EventID: "evt-1", PeerID: "peer-alice", PeerSequence: 1, RoomID: "test",
		Timestamp: "2026-09-16T00:00:00Z", EventType: EventUserPrompt, Content: "what Alice said"}
	if res, err := s.Insert(genuine); err != nil || res != InsertStored {
		t.Fatalf("setup: %v %v", res, err)
	}

	sameSlot := *genuine
	sameSlot.Content = "what Alice never said"
	if res, err := s.Insert(&sameSlot); err != nil || res != InsertConflict {
		t.Fatalf("the same identifier and slot with other content was %v (%v), want a conflict", res, err)
	}
	otherSlot := *genuine
	otherSlot.PeerID, otherSlot.Content = "peer-mallory", "Mallory's turn under Alice's identifier"
	for i := 0; i < 2; i++ {
		if res, err := s.Insert(&otherSlot); err != nil || res != InsertConflict {
			t.Fatalf("the same identifier at another slot was %v (%v), want a conflict", res, err)
		}
	}

	evs, _ := s.ListRoom("test")
	if len(evs) != 1 || evs[0].Content != "what Alice said" {
		t.Fatalf("the held event was displaced: %+v", evs)
	}
	cs, _ := s.ListConflicts()
	if len(cs) != 2 {
		raw, _ := json.Marshal(cs)
		t.Fatalf("want each differing event quarantined once, got %s", raw)
	}
	// A quarantined slot is not asked for again on every poll.
	if st, _ := s.SyncState("test"); st["peer-mallory"] != 1 {
		t.Errorf("sync state %v asks for the quarantined slot again", st)
	}
	if res, _ := s.Insert(genuine); res != InsertDuplicate {
		t.Errorf("a true redelivery was %v", res)
	}
}

// A signed request captured on its way to one machine cannot be presented to another
// by a machine that did not sign it: the request is bound to the key of the connection
// it arrives on (D-143, admission is proved by a signed request bound to the
// connection).
func TestASyncRequestReplayedByAnotherMachineIsRefused(t *testing.T) {
	const room = "room-replay"
	a := newOfflinePeer(t, room, "misty-canyon")
	b := newOfflinePeer(t, room, "misty-canyon")
	c := newOfflinePeer(t, room, "misty-canyon")
	introduce(t, a, c, room)
	introduce(t, b, c, room)
	c.say(t, room, "s", EventUserPrompt, "for the room's guests")

	send := func(from *offlinePeer, cred requestCreds) int {
		t.Helper()
		conf, err := from.d.clientConfig("")
		if err != nil {
			t.Fatal(err)
		}
		client, err := clientFor(c.addr, 10*time.Second, conf)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := json.Marshal(syncRequest{Protocol: wireVersion, RoomID: room, PeerID: cred.PeerID,
			Timestamp: cred.Timestamp, Nonce: cred.Nonce, Signature: cred.Signature, Endpoint: cred.Endpoint})
		resp, err := client.Post(peerURL("/sync"), "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	captured, err := signRequestAs(a.d.id, room, "")
	if err != nil {
		t.Fatal(err)
	}
	if code := send(b, captured); code != http.StatusUnauthorized {
		t.Fatalf("a request a signed, sent by b, was answered %d", code)
	}
	fresh, _ := signRequestAs(a.d.id, room, "")
	if code := send(a, fresh); code != http.StatusOK {
		t.Fatalf("a's own request was answered %d", code)
	}
}
