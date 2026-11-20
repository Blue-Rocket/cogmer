package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// The rooms pane lists every room, the most recently active first, with what a reader
// needs to tell the rooms apart without opening each: who wrote last, how many of the
// colleagues' turns there are to compare with the last look, how many no session here
// has been given, and whether a session here is in the room (D-289).
func TestTheRoomListPutsTheLatestRoomFirstAndCountsItsTurns(t *testing.T) {
	d, quiet := testDaemon(t)
	busy, err := d.members.CreateRoom(d.id.PeerID, false)
	if err != nil {
		t.Fatal(err)
	}
	store, err := d.storeFor(busy.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	colleague := testIdentity(t)
	for i, ts := range []string{"2026-09-20T10:00:00Z", "2026-09-20T11:00:00Z"} {
		ev := Event{EventID: "c" + string(rune('1'+i)), PeerID: colleague.PeerID, PeerSequence: int64(i + 1), RoomID: busy.RoomID,
			Timestamp: ts, EventType: EventUserPrompt, Content: "from a colleague"}
		ev.Sign(colleague.private)
		if _, err := store.Insert(&ev); err != nil {
			t.Fatal(err)
		}
	}

	rec := httptest.NewRecorder()
	d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest("GET", "/rooms", nil))
	var list []roomSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("/rooms answered %d with %q: %v", rec.Code, rec.Body.String(), err)
	}
	if len(list) != 2 {
		t.Fatalf("the list holds %d rooms, want 2: %+v", len(list), list)
	}
	if list[0].Name != busy.RoomName || list[1].Name != quiet.RoomName {
		t.Fatalf("the room with turns is not first: %+v", list)
	}
	b := list[0]
	if b.Theirs != 2 || b.Waiting != 2 || b.LastMine || b.LastAt == "" || b.LastBy == "" {
		t.Errorf("the busy room reads %+v, want two colleague turns, both waiting, the last a colleague's", b)
	}
	if q := list[1]; q.Theirs != 0 || q.LastAt != "" {
		t.Errorf("the quiet room reads %+v, want no turns", q)
	}

	// A turn of this user's own, written later, leads the room and is not counted as a
	// colleague's.
	if _, err := store.Append(1, d.id, busy.RoomID, "s", EventUserPrompt, "mine", nil); err != nil {
		t.Fatal(err)
	}
	list, err = d.roomSummaries()
	if err != nil {
		t.Fatal(err)
	}
	if b := list[0]; b.Theirs != 2 || !b.LastMine || b.LastBy != "you" {
		t.Errorf("after this user's turn the room reads %+v", b)
	}
}

// The list carries a colleague's chosen name, so the rooms pane writes every value as text
// and never as markup.
func TestTheRoomsPaneWritesValuesOnlyAsText(t *testing.T) {
	src := string(uiPage)
	i := strings.Index(src, "// The rooms pane lists every room")
	if i < 0 {
		t.Fatal("the room page has no rooms pane")
	}
	pane := src[i:]
	if !strings.Contains(pane, "textContent") || !strings.Contains(pane, "/rooms/stream") {
		t.Error("the rooms pane no longer reads /rooms/stream and writes with textContent")
	}
	for _, frag := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "document.write"} {
		if strings.Contains(pane, frag) {
			t.Errorf("the rooms pane uses %s, which would read a value as markup", frag)
		}
	}
}
