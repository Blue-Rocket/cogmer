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
	for _, path := range []string{"/", "/stream"} {
		rec := httptest.NewRecorder()
		d.PeerRoutes().ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != 404 {
			t.Errorf("peer listener serves %s (status %d)", path, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "claude-team") {
		t.Errorf("local listener does not serve the page (status %d)", rec.Code)
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
	st, err := d.snapshot()
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
	st, _ := d.snapshot()
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
