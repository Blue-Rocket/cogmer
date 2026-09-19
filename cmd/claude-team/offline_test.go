package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Phase 5: offline and reconnection.
//
// Two peers work while partitioned and converge when the partition heals, with
// nothing lost and both agreeing on the order. Anti-entropy is the whole mechanism
// (§10) -- there is no queue of undelivered messages to drain, because a pull
// against a watermark makes "what did you miss" a question asked rather than state
// anybody has to keep.

// offlinePeer is a daemon with its own HOME, its own store, and a server that can
// be made unreachable without losing its address.
type offlinePeer struct {
	d     *Daemon
	store *Store
	addr  string
	up    *atomic.Bool
}

func newOfflinePeer(t *testing.T, roomID, roomName string) *offlinePeer {
	t.Helper()
	// Everything below is opened while HOME points at this peer's directory.
	// Nothing may re-resolve it afterwards, because the next peer moves it.
	t.Setenv("HOME", t.TempDir())

	m, err := OpenMembership()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	if err := m.RecordRoom(roomID, roomName); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(roomID)
	if err != nil {
		t.Fatal(err)
	}
	id := testIdentity(t)
	if err := m.Invite(roomID, id.PeerID); err != nil {
		t.Fatal(err)
	}

	d := &Daemon{id: id, members: m, stores: map[string]*Store{roomID: store}}
	t.Cleanup(d.closeStores)

	up := &atomic.Bool{}
	up.Store(true)
	routes := d.PeerRoutes()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !up.Load() {
			// Stands for unreachable. The address survives so the peer can come
			// back at it, which is what makes reconnection testable at all.
			http.Error(w, "unreachable", http.StatusServiceUnavailable)
			return
		}
		routes.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	return &offlinePeer{d: d, store: store, addr: strings.TrimPrefix(srv.URL, "http://"), up: up}
}

// introduce makes two peers guests of the room and verified to each other, which
// D-054 requires before any transcript crosses between them.
func introduce(t *testing.T, a, b *offlinePeer, roomID string) {
	t.Helper()
	for _, pair := range []struct{ from, to *offlinePeer }{{a, b}, {b, a}} {
		if err := pair.from.d.members.Allow(pair.to.d.id.PeerID, ""); err != nil {
			t.Fatal(err)
		}
		if err := pair.from.d.members.MarkVerified(pair.to.d.id.PeerID); err != nil {
			t.Fatal(err)
		}
		if err := pair.from.d.members.Invite(roomID, pair.to.d.id.PeerID); err != nil {
			t.Fatal(err)
		}
	}
}

// settle runs anti-entropy in both directions until neither side takes anything
// new. Repeating is the point: convergence is a property of the loop, not of one
// exchange, and a single pull each way would hide an event that needs two rounds.
func settle(t *testing.T, a, b *offlinePeer, room Room) {
	t.Helper()
	for round := 0; round < 8; round++ {
		na, _ := a.d.pullFrom(b.addr)
		nb, _ := b.d.pullFrom(a.addr)
		if na == 0 && nb == 0 {
			return
		}
	}
	t.Fatal("peers did not converge in 8 rounds")
}

// say writes a turn the way the daemon does: reserve a sequence outside the room,
// then write the event that uses it (D-029). Tests that bypass this would not
// exercise the path that survives losing a room.
func (p *offlinePeer) say(t *testing.T, roomID, session, kind, text string) {
	t.Helper()
	if _, err := p.d.appendLocal(p.store, Room{RoomID: roomID}, session, kind, text, nil); err != nil {
		t.Fatal(err)
	}
}

func eventIDs(t *testing.T, s *Store, roomID string) []string {
	t.Helper()
	evs, err := s.EventsSince(roomID, nil)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(evs))
	for _, e := range evs {
		out = append(out, e.EventID)
	}
	return out
}

func TestPeersConvergeAfterAPartition(t *testing.T) {
	const roomID, roomName = "room-offline", "misty-canyon"
	a := newOfflinePeer(t, roomID, roomName)
	b := newOfflinePeer(t, roomID, roomName)
	introduce(t, a, b, roomID)
	room := Room{RoomID: roomID, RoomName: roomName}

	// Connected: one turn each, and they agree.
	a.say(t, roomID, "sess-a", EventUserPrompt, "before-a")
	b.say(t, roomID, "sess-b", EventUserPrompt, "before-b")
	settle(t, a, b, room)
	if len(eventIDs(t, a.store, roomID)) != 2 {
		t.Fatalf("before the partition the peers had not converged: %d events", len(eventIDs(t, a.store, roomID)))
	}

	// Partitioned. Both keep working -- §26 requires that offline operation be
	// ordinary rather than degraded.
	b.up.Store(false)
	a.up.Store(false)
	for _, text := range []string{"alice-1", "alice-2", "alice-3"} {
		a.say(t, roomID, "sess-a", EventUserPrompt, text)
	}
	for _, text := range []string{"david-1", "david-2"} {
		b.say(t, roomID, "sess-b", EventAssistantMessage, text)
	}

	// A pull while partitioned must fail without leaving anything behind.
	if n, err := a.d.pullFrom(b.addr); err == nil {
		t.Error("a pull from an unreachable peer reported success")
	} else if n != 0 {
		t.Errorf("a failed pull claimed to store %d events", n)
	}
	if got := len(eventIDs(t, a.store, roomID)); got != 5 {
		t.Errorf("a failed pull changed the store: %d events, want 5", got)
	}

	// Healed.
	a.up.Store(true)
	b.up.Store(true)
	settle(t, a, b, room)

	idsA, idsB := eventIDs(t, a.store, roomID), eventIDs(t, b.store, roomID)
	if len(idsA) != 7 {
		t.Errorf("A holds %d events after reconnection, want 7", len(idsA))
	}
	// Identical ORDER, not merely identical contents. Two peers that agree on
	// which events exist and disagree on their order render different rooms, and
	// §24's ordering is what prevents that.
	if strings.Join(idsA, ",") != strings.Join(idsB, ",") {
		t.Errorf("peers disagree after reconnection:\n A: %v\n B: %v", idsA, idsB)
	}

	// Nothing was quarantined: a peer resuming is redelivery, not a conflict
	// (D-027), and treating it as one would strand real events.
	for _, p := range []*offlinePeer{a, b} {
		c, err := p.store.ListConflicts()
		if err != nil {
			t.Fatal(err)
		}
		if len(c) != 0 {
			t.Errorf("reconnection produced %d quarantined event(s): %+v", len(c), c)
		}
	}

	// And the watermark advanced for both peers on both sides, so the next poll
	// asks for what comes after rather than refetching the backlog forever.
	for _, p := range []*offlinePeer{a, b} {
		have, err := p.store.SyncState(roomID)
		if err != nil {
			t.Fatal(err)
		}
		if have[a.d.id.PeerID] != 4 || have[b.d.id.PeerID] != 3 {
			t.Errorf("watermark is %v; want 4 from A and 3 from B", have)
		}
	}
}

// Re-syncing after convergence must take nothing and change nothing. Anti-entropy
// runs every second forever, so an exchange that is not idempotent is not a bug
// that shows up once.
func TestReconnectionIsIdempotent(t *testing.T) {
	const roomID, roomName = "room-idem", "quiet-fell"
	a := newOfflinePeer(t, roomID, roomName)
	b := newOfflinePeer(t, roomID, roomName)
	introduce(t, a, b, roomID)
	room := Room{RoomID: roomID, RoomName: roomName}

	a.say(t, roomID, "sess-a", EventUserPrompt, "one")
	settle(t, a, b, room)
	before := eventIDs(t, b.store, roomID)

	for i := 0; i < 3; i++ {
		n, err := b.d.pullFrom(a.addr)
		if err != nil {
			t.Fatalf("round %d: %v", i, err)
		}
		if n != 0 {
			t.Errorf("round %d took %d events from a converged peer", i, n)
		}
	}
	if got := eventIDs(t, b.store, roomID); strings.Join(got, ",") != strings.Join(before, ",") {
		t.Errorf("repeated sync changed the room:\n was %v\n now %v", before, got)
	}
}

// Review B2, which could not be checked until now.
//
// A watermark over ARRIVAL order is the right mechanism for "has this session seen
// it", but anti-entropy and transitive relay deliver old events late -- so an event
// authored an hour ago can arrive after one authored a minute ago. Injecting in
// arrival order hands Claude a conversation out of sequence, which is exactly the
// kind of thing that makes a referent resolve to the wrong turn.
//
// With one live peer this is untriggerable: the block holds only that peer's turns,
// already in sequence. It takes a peer reconnecting with a backlog alongside a live
// one, which is the state a partition produces.
func TestABacklogIsInjectedInChronologicalOrder(t *testing.T) {
	s := testStore(t)
	const roomID = "test"
	live, returning := testIdentity(t), testIdentity(t)

	insert := func(id *Identity, seq int64, ts, text string) {
		t.Helper()
		ev := &Event{
			EventID: text, PeerID: id.PeerID, PeerSequence: seq, RoomID: roomID,
			Timestamp: ts, UserDisplayName: "peer", OriginSessionID: "theirs",
			EventType: EventUserPrompt, Content: text,
		}
		ev.Sign(id.private)
		if _, err := s.Insert(ev); err != nil {
			t.Fatal(err)
		}
	}

	// Arrival order. The live peer is talking; the returning peer's backlog lands
	// afterwards, carrying timestamps from before and during the partition.
	insert(live, 1, "2026-09-18T10:05:00Z", "live-at-1005")
	insert(live, 2, "2026-09-18T10:10:00Z", "live-at-1010")
	insert(returning, 1, "2026-09-18T10:00:00Z", "backlog-at-1000")
	insert(returning, 2, "2026-09-18T10:07:00Z", "backlog-at-1007")

	pending, err := s.UndeliveredFor(roomID, "mine")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range pending {
		got = append(got, e.EventID)
	}
	want := []string{"backlog-at-1000", "live-at-1005", "backlog-at-1007", "live-at-1010"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("injected in arrival order rather than chronological order:\n got %v\nwant %v", got, want)
	}

	// And the rendered block carries that order, since the block is what the model
	// reads. Ordering the query and then rendering some other way would be worse
	// than not ordering at all -- it would look correct in every test but this one.
	text := FormatTeamContext(pending, func(string) bool { return true })
	last := -1
	for _, id := range want {
		at := strings.Index(text, id)
		if at < 0 {
			t.Fatalf("%s is missing from the injected block", id)
		}
		if at < last {
			t.Errorf("%s appears out of order in the injected block", id)
		}
		last = at
	}
}

// A peer that is in several rooms must not go silent in all of them because one
// fails. The failures worth surviving are the lasting ones -- a corrupt store, a
// peer speaking another protocol version, a room whose host is down while others
// are up -- and each of those would otherwise starve every room after it, forever.
//
// Note that a 401 is deliberately NOT such a failure: a peer hosting rooms we are
// not in is ordinary, and pullRoom treats it as nothing to report. The failure used
// here is a room whose store cannot be opened, which is both real and lasting.
func TestOneFailingRoomDoesNotStarveTheOthers(t *testing.T) {
	const goodID = "room-good"
	// A room id that cannot become a file path, so opening its store fails every
	// time rather than once.
	const badID = "room-bad/unopenable"

	a := newOfflinePeer(t, goodID, "sunny-basin")
	b := newOfflinePeer(t, goodID, "sunny-basin")
	introduce(t, a, b, goodID)

	if err := a.d.members.RecordRoom(badID, "aaa-broken"); err != nil {
		t.Fatal(err)
	}
	// Forced to sort first, because that is the case that starves: rooms are
	// attempted in created_at order, and returning on the first error means every
	// room after it is never reached.
	if _, err := a.d.members.db.Exec(
		`UPDATE rooms SET created_at = '2000-01-01T00:00:00Z' WHERE room_id = ?`, badID); err != nil {
		t.Fatal(err)
	}

	b.say(t, goodID, "sess-b", EventUserPrompt, "hello")

	n, err := a.d.pullFrom(b.addr)
	if n != 1 {
		t.Errorf("received %d events, want 1; the good room was starved by the failing one", n)
	}
	// Surviving a failure must not mean hiding it.
	if err == nil {
		t.Fatal("a room that could not be pulled was not reported")
	}
	if !strings.Contains(err.Error(), "aaa-broken") {
		t.Errorf("the report does not name the room that failed: %v", err)
	}
}

// Review C-1, and D-029's requirement that losing a room be recoverable rather
// than catastrophic.
//
// Deleting a room database used to restart the sequence counter at 1, reissuing
// numbers that peers already held under different event ids -- the exact condition
// D-027 quarantines on the receiving side, caused by a peer that is never told.
func TestLosingARoomDoesNotReissueSequences(t *testing.T) {
	const roomID = "room-lost"
	p := newOfflinePeer(t, roomID, "lost-hollow")

	for _, text := range []string{"one", "two", "three"} {
		p.say(t, roomID, "sess", EventUserPrompt, text)
	}
	issued := p.d.members.IssuedSequence(roomID)
	if issued != 3 {
		t.Fatalf("issued %d sequences, want 3", issued)
	}

	// The room is gone. Membership and sequence position are not: they live
	// beside the identity, precisely so that this is survivable (D-029).
	fresh, err := OpenStore(roomID + "-replacement")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { fresh.Close() })
	p.store = fresh
	p.d.stores[roomID] = fresh

	if held, _ := fresh.HighestSequence(p.d.id.PeerID); held != 0 {
		t.Fatalf("the replacement room is not empty: holds up to %d", held)
	}
	if got := p.d.members.IssuedSequence(roomID); got != 3 {
		t.Errorf("losing the room lost the sequence position too: %d", got)
	}

	// The next event must resume ABOVE what was issued, not restart at 1.
	p.say(t, roomID, "sess", EventUserPrompt, "after the loss")
	evs, err := fresh.EventsSince(roomID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 1 {
		t.Fatalf("expected 1 event in the replacement room, got %d", len(evs))
	}
	if evs[0].PeerSequence != 4 {
		t.Errorf("resumed at sequence %d; want 4, above the 3 already issued", evs[0].PeerSequence)
	}
}

// Reserving before publishing means a crash between the two loses a number rather
// than reissuing one. Losing a number is harmless: the watermark is the highest
// CONTIGUOUS sequence, so a gap simply means peers wait rather than skip.
func TestAReservationIsSpentEvenIfNothingIsWritten(t *testing.T) {
	const roomID = "room-reserve"
	p := newOfflinePeer(t, roomID, "spent-reach")

	first, err := p.d.members.ReserveSequence(roomID)
	if err != nil {
		t.Fatal(err)
	}
	// Nothing is written with `first` -- the crash.
	second, err := p.d.members.ReserveSequence(roomID)
	if err != nil {
		t.Fatal(err)
	}
	if second == first {
		t.Fatalf("sequence %d was issued twice; a reissued sequence is a different event under an identifier peers already hold", first)
	}
	if second != first+1 {
		t.Errorf("reservations jumped from %d to %d", first, second)
	}
}
