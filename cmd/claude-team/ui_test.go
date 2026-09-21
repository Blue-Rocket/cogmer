package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func uiDaemon(t *testing.T) *Daemon {
	t.Helper()
	d, _ := testDaemon(t)
	return d
}

// The UI is local-only. It reads the whole conversation, so serving it to peers
// would hand the room to anyone who can reach the sync port (§25, D-030).
func TestUIIsNotServedToPeers(t *testing.T) {
	d := uiDaemon(t)
	for _, path := range []string{"/", "/stream", "/room/anything", "/events"} {
		rec := httptest.NewRecorder()
		d.PeerRoutes().ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != 404 {
			t.Errorf("peer listener serves %s (status %d)", path, rec.Code)
		}
	}
}

// Each room has its own URL, so the address handed over at `create` is a stable
// link to that room rather than a window that can change identity (D-077).
func TestEachRoomHasItsOwnURL(t *testing.T) {
	d, room := testDaemon(t)

	rec := httptest.NewRecorder()
	d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest("GET", "/room/"+room.RoomName, nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "claude-team") {
		t.Errorf("a room's own URL does not serve the page (status %d)", rec.Code)
	}

	// A room nobody has does not resolve to somebody else's room.
	rec = httptest.NewRecorder()
	d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest("GET", "/room/no-such-room", nil))
	if rec.Code != 404 {
		t.Errorf("an unknown room answered %d rather than 404", rec.Code)
	}

	// With one room, the index goes straight there rather than making somebody
	// choose from a list of one.
	rec = httptest.NewRecorder()
	d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 302 || rec.Header().Get("Location") != "/room/"+room.RoomName {
		t.Errorf("the index gave %d → %q", rec.Code, rec.Header().Get("Location"))
	}

	// With two, it lists them and chooses nothing.
	if _, err := d.members.CreateRoom(d.id.PeerID); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 200 {
		t.Fatalf("with two rooms the index gave %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), room.RoomName) {
		t.Error("the index does not list the rooms it is meant to be listing")
	}
}

// The events endpoint serves the room it was asked for, and does not fall back to
// one when the question is unanswerable.
func TestEventsServeTheRoomAsked(t *testing.T) {
	d, room := testDaemon(t)
	store, err := d.storeFor(room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	id := testIdentity(t)
	ev := Event{EventID: "e1", PeerID: id.PeerID, PeerSequence: 1, RoomID: room.RoomID,
		Timestamp: "2026-09-20T00:00:00Z", EventType: EventUserPrompt, Content: "hello"}
	ev.Sign(id.private)
	if _, err := store.Insert(&ev); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest("GET", "/events?room="+room.RoomName, nil))
	if !strings.Contains(rec.Body.String(), "hello") {
		t.Errorf("the named room's events are missing: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest("GET", "/events", nil))
	if strings.Contains(rec.Body.String(), "hello") {
		t.Error("with no room named, events were served from a room nobody asked for")
	}
}

// Attribution anchors on the derived peer name. Display names are self-asserted
// and collided during the first two-peer run, so the UI must not rely on them
// alone to tell two people apart (D-021).
func TestSnapshotDistinguishesPeersClaimingOneName(t *testing.T) {
	d, room := testDaemon(t)
	store, err := d.storeFor(room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	for i, p := range []string{"peer-aaa", "peer-bbb"} {
		if _, err := store.Append(int64(i+1), &Identity{PeerID: p, UserDisplayName: "David"},
			room.RoomID, "s", EventUserPrompt, "hello from "+p, nil); err != nil {
			t.Fatal(err)
		}
	}
	st, err := d.snapshot(room)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Events) != 2 {
		t.Fatalf("want 2 events, got %d", len(st.Events))
	}
	if st.Events[0].PeerName == st.Events[1].PeerName {
		t.Error("two peers claiming one display name are indistinguishable in the UI")
	}
	for _, e := range st.Events {
		if e.PeerName == "" {
			t.Error("event carries no derived peer name")
		}
	}
}

// Room content arrives from other peers. It must reach the browser as data, never
// as markup the server has pre-rendered.
func TestSnapshotCarriesContentAsData(t *testing.T) {
	d, room := testDaemon(t)
	store, err := d.storeFor(room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	hostile := `<script>alert(1)</script>`
	if _, err := store.Append(1, d.id, room.RoomID, "s", EventUserPrompt, hostile, nil); err != nil {
		t.Fatal(err)
	}
	st, _ := d.snapshot(room)
	if len(st.Events) == 0 || st.Events[0].Content != hostile {
		t.Errorf("content was altered in transit: %q", st.Events[0].Content)
	}
	buf, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	// JSON-encoded, so the payload cannot break out of the SSE data frame.
	if strings.Contains(string(buf), "\n") {
		t.Error("snapshot JSON contains a newline; it would truncate the SSE frame")
	}
}

// The page must escape before applying any markup, or a teammate's message
// becomes script in everyone else's browser.
func TestPageEscapesBeforeRendering(t *testing.T) {
	src := string(uiPage)
	iEsc := strings.Index(src, "const esc=")
	iUse := strings.Index(src, "esc(md)")
	if iEsc < 0 || iUse < 0 {
		t.Fatal("the page no longer escapes content before rendering it")
	}
	if iUse < iEsc {
		t.Error("content is rendered before the escape function exists")
	}
	for _, frag := range []string{"innerHTML=md", "innerHTML=e.content", "innerHTML = md"} {
		if strings.Contains(src, frag) {
			t.Errorf("page writes unescaped content to innerHTML: %q", frag)
		}
	}
}

// A reader that has not drained a previous wake-up must not block the daemon.
func TestNotifyDoesNotBlockOnASlowReader(t *testing.T) {
	d := uiDaemon(t)
	ch := d.subscribe()
	defer d.unsubscribe(ch)
	for i := 0; i < 100; i++ {
		d.notify() // never drained
	}
	if len(ch) != 1 {
		t.Errorf("wake-ups accumulated (%d); a slow reader would stall publishing", len(ch))
	}
}

// The label is the only name that is both memorable and bound to one key: a display
// name is the peer's own claim, and a derived name means nothing to anybody weeks
// later. It lived in the database and reached neither the view nor the model
// (D-094).
func TestTheViewCarriesTheNameYouChose(t *testing.T) {
	d, room := testDaemon(t)
	store, err := d.storeFor(room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	named := testIdentity(t).PeerID
	unnamed := testIdentity(t).PeerID
	if err := d.members.Allow(named, "alice"); err != nil {
		t.Fatal(err)
	}
	if err := d.members.MarkVerified(named); err != nil {
		t.Fatal(err)
	}
	// Recorded with no label of its own, as a script would (D-053).
	if err := d.members.Allow(unnamed, ""); err != nil {
		t.Fatal(err)
	}
	for i, p := range []string{named, unnamed} {
		if _, err := store.Append(int64(i+1), &Identity{PeerID: p, UserDisplayName: "Ec2-user"},
			room.RoomID, "s", EventUserPrompt, "hello", nil); err != nil {
			t.Fatal(err)
		}
	}

	st, err := d.snapshot(room)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Events) != 2 {
		t.Fatalf("want 2 events, got %d", len(st.Events))
	}
	if st.Events[0].Label != "alice" {
		t.Errorf("label is %q, want alice; the name chosen at pairing did not reach the view", st.Events[0].Label)
	}
	// A placeholder is not a choice, and offering it as one would be a lie.
	if st.Events[1].Label != "" {
		t.Errorf("an unnamed peer carries label %q; the derived placeholder was offered as a chosen name", st.Events[1].Label)
	}
	// Both still carry the derived name, which is what a key change surfaces on.
	for _, e := range st.Events {
		if e.PeerName == "" {
			t.Error("the derived anchor was dropped when a label appeared")
		}
	}
}

// The marker used to be rendered on every remote turn, because the view had no way
// to ask. D-054 refuses an unverified peer's events outright, so anything displayed
// is verified -- and a warning that is always on is not a warning.
func TestTheUnverifiedMarkerInTheViewIsAFact(t *testing.T) {
	d, room := testDaemon(t)
	store, err := d.storeFor(room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	verified := testIdentity(t).PeerID
	stranger := testIdentity(t).PeerID
	if err := d.members.Allow(verified, "alice"); err != nil {
		t.Fatal(err)
	}
	if err := d.members.MarkVerified(verified); err != nil {
		t.Fatal(err)
	}
	if err := d.members.Allow(stranger, "bob"); err != nil {
		t.Fatal(err)
	}
	for i, p := range []string{verified, stranger} {
		if _, err := store.Append(int64(i+1), &Identity{PeerID: p, UserDisplayName: "X"},
			room.RoomID, "s", EventUserPrompt, "hello", nil); err != nil {
			t.Fatal(err)
		}
	}
	st, err := d.snapshot(room)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Events[0].Verified {
		t.Error("a verified peer's turn is still marked unverified, so the marker says nothing")
	}
	if st.Events[1].Verified {
		t.Error("an unverified peer's turn is reported verified; the marker is the backstop for a failed filter")
	}
	// Your own turns need no marker and no anchor (D-021).
	if _, err := store.Append(3, d.id, room.RoomID, "s", EventUserPrompt, "mine", nil); err != nil {
		t.Fatal(err)
	}
	st, _ = d.snapshot(room)
	own := st.Events[len(st.Events)-1]
	if !own.Mine || !own.Verified {
		t.Error("your own turn is reported as somebody else's, or as unverified")
	}
}
