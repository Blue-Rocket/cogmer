package main

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// appendFrom writes a turn by a peer into a room, and returns its event id.
func appendFrom(t *testing.T, d *Daemon, room Room, peer string, seq int64, content string) string {
	t.Helper()
	store, err := d.storeFor(room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	ev, err := store.Append(seq, &Identity{PeerID: peer, UserDisplayName: "X"}, room.RoomID, "s", EventUserPrompt, content, nil)
	if err != nil {
		t.Fatal(err)
	}
	return ev.EventID
}

func markDelivered(t *testing.T, d *Daemon, room Room, session, eventID string) {
	t.Helper()
	store, _ := d.storeFor(room.RoomID)
	if _, err := store.db.Exec(`INSERT INTO session_delivered (origin_session_id, event_id) VALUES (?, ?)`, session, eventID); err != nil {
		t.Fatal(err)
	}
}

// A turn the page shows that no session holds is the difference between reading the
// room and the model having it, and only a colleague's turn can be in that state.
func TestTheViewSaysWhichColleagueTurnsNoSessionHolds(t *testing.T) {
	d, room := testDaemon(t)
	colleague := testIdentity(t).PeerID
	held := appendFrom(t, d, room, colleague, 1, "held")
	appendFrom(t, d, room, colleague, 2, "not held")
	appendFrom(t, d, room, d.id.PeerID, 1, "mine")
	markDelivered(t, d, room, "some-session", held)

	st, err := d.snapshot(room)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, e := range st.Events {
		got[e.Content] = e.NotInSession
	}
	if got["held"] {
		t.Error("a turn a session was confirmed to hold is marked as not in a session")
	}
	if !got["not held"] {
		t.Error("a colleague's turn no session holds is not marked, so the page says nothing about it")
	}
	if got["mine"] {
		t.Error("your own turn is marked as not in a session; it is in the one that wrote it")
	}
}

// A show-only room gives nothing to a session, so a marker on every turn would say
// nothing (D-200).
func TestAShowOnlyRoomMarksNoTurnAsNotInASession(t *testing.T) {
	d, _ := testDaemon(t)
	room, err := d.members.CreateRoom(d.id.PeerID, true)
	if err != nil {
		t.Fatal(err)
	}
	appendFrom(t, d, room, testIdentity(t).PeerID, 1, "hello")
	st, err := d.snapshot(room)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Events) == 0 || st.Events[0].NotInSession {
		t.Errorf("a show-only room marks a turn as not in a session: %+v", st.Events)
	}
}

// A colleague who is reachable and whose turns have stopped arriving is the case the
// dot cannot show, and the page needs the time of their last turn to show it.
func TestThePeerListCarriesWhenEachColleagueLastWroteATurn(t *testing.T) {
	d, room := testDaemon(t)
	writer, silent := testIdentity(t).PeerID, testIdentity(t).PeerID
	for i, p := range []string{writer, silent} {
		if err := d.members.Allow(p, []string{"alice", "bob"}[i]); err != nil {
			t.Fatal(err)
		}
		if err := d.members.SetPeerEndpoint(p, "127.0.0.1:5999"+string(rune('0'+i))); err != nil {
			t.Fatal(err)
		}
	}
	appendFrom(t, d, room, writer, 1, "hello")
	st, err := d.snapshot(room)
	if err != nil {
		t.Fatal(err)
	}
	last := map[string]string{}
	for _, p := range st.Peers {
		last[p.PeerID] = p.LastTurn
	}
	if last[writer] == "" {
		t.Errorf("a colleague who wrote a turn has no last turn: %+v", st.Peers)
	}
	if v, ok := last[silent]; !ok || v != "" {
		t.Errorf("a colleague who wrote nothing has a last turn, or is missing: %+v", st.Peers)
	}
}

func TestTurnTimeShowsTheDateOnlyWhenTheDayIsNotToday(t *testing.T) {
	now := time.Date(2026, 10, 13, 15, 0, 0, 0, time.Local)
	if got := turnTime(time.Date(2026, 10, 13, 9, 5, 0, 0, time.Local), now); got != "09:05" {
		t.Errorf("a turn from today reads %q", got)
	}
	if got := turnTime(time.Date(2026, 10, 12, 9, 5, 0, 0, time.Local), now); got != "Oct 12 09:05" {
		t.Errorf("a turn from another day reads %q", got)
	}
}

func TestDeliveredEventIDsAreThoseAnySessionHolds(t *testing.T) {
	d, room := testDaemon(t)
	markDelivered(t, d, room, "one", "e1")
	markDelivered(t, d, room, "two", "e1")
	markDelivered(t, d, room, "two", "e2")
	store, _ := d.storeFor(room.RoomID)
	got, err := store.DeliveredEventIDs()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !got["e1"] || !got["e2"] {
		t.Errorf("delivered = %v", got)
	}
}

// The export is a document somebody keeps, so it has to say who wrote each turn and
// when, and keep what a colleague wrote exactly as written.
func TestTheExportNamesEachTurnAndKeepsItsContent(t *testing.T) {
	d, room := testDaemon(t)
	stranger := testIdentity(t).PeerID
	hostile := "<script>alert(1)</script>\n```\nrm -rf /\n```"
	appendFrom(t, d, room, stranger, 1, hostile)
	appendFrom(t, d, room, d.id.PeerID, 1, "my turn")
	st, err := d.snapshot(room)
	if err != nil {
		t.Fatal(err)
	}
	st.Events[0].ToolCalls, st.Events[0].Tools = 3, []string{"Bash", "Bash", "Read"}
	out := exportMarkdown(st, time.Date(2026, 10, 13, 15, 0, 0, 0, time.Local))
	for _, want := range []string{"# " + room.RoomName, "Exported 2026-10-13 15:00", "2 turns", hostile, "my turn",
		"3 tool operations: Bash ×2, Read", PeerName(stranger) + ", unverified"} {
		if !strings.Contains(out, want) {
			t.Errorf("the export lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, PeerName(d.id.PeerID)) {
		t.Error("your own turn carries a derived name and a marker, which the page leaves off")
	}
}

func TestTheExportRouteDownloadsTheNamedRoom(t *testing.T) {
	d, room := testDaemon(t)
	appendFrom(t, d, room, d.id.PeerID, 1, "kept")
	rec := httptest.NewRecorder()
	d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest("GET", "/export?room="+room.RoomName, nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "kept") {
		t.Fatalf("export gave %d: %s", rec.Code, rec.Body.String())
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.HasPrefix(cd, "attachment;") || !strings.Contains(cd, ".md") {
		t.Errorf("the export is not a download: %q", cd)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("a colleague's text could be sniffed as a page")
	}
	for _, path := range []string{"/export", "/export?room=no-such-room"} {
		rec = httptest.NewRecorder()
		d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != 404 || strings.Contains(rec.Body.String(), "kept") {
			t.Errorf("%s served a room nobody named (status %d)", path, rec.Code)
		}
	}
}

func TestExportFileStemHoldsNothingAHeaderCouldMisread(t *testing.T) {
	if got := exportFileStem("quiet-room.1"); got != "quiet-room.1" {
		t.Errorf("a plain name changed: %q", got)
	}
	if got := exportFileStem("a\"b\r\nc/d"); strings.ContainsAny(got, "\"\r\n/") {
		t.Errorf("the stem keeps a character that breaks a header or a path: %q", got)
	}
	if exportFileStem("") != "room" {
		t.Error("an empty name gives an empty file name")
	}
}

// No test runs the page's script, so the two rules that keep a reader's place hold
// by what the source says. The feed is not rebuilt on an arrival, which closed every
// tool list the reader had opened, and a new turn the reader has not scrolled to is
// counted (D-204).
func TestThePageKeepsTheReadersPlace(t *testing.T) {
	src := string(uiPage)
	if strings.Contains(src, "feed.innerHTML=state.events") {
		t.Error("the page rebuilds the whole feed on every arrival, closing the tool lists a reader opened")
	}
	for _, want := range []string{"insertBefore", "unread", "document.title", "/export?room="} {
		if !strings.Contains(src, want) {
			t.Errorf("the page no longer has %q", want)
		}
	}
}
