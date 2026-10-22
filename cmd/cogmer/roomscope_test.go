package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A machine knows every peer and every room, and a room's surface may show only that
// room's. Each test here sets up two rooms with a different guest in each and checks one
// surface for what it must not hold.

// The page, its stream and the export all read one room's turns.
func TestAPageAndItsExportHoldOnlyThatRoomsTurns(t *testing.T) {
	d, first := testDaemon(t)
	second, err := d.members.CreateRoom(d.id.PeerID, false)
	if err != nil {
		t.Fatal(err)
	}
	appendFrom(t, d, first, d.id.PeerID, 1, "said in the first room")
	appendFrom(t, d, second, d.id.PeerID, 1, "said in the second room")

	for _, c := range []struct {
		room        Room
		have, avoid string
	}{
		{first, "said in the first room", "said in the second room"},
		{second, "said in the second room", "said in the first room"},
	} {
		st, err := d.snapshot(c.room)
		if err != nil {
			t.Fatal(err)
		}
		var page strings.Builder
		for _, e := range st.Events {
			page.WriteString(e.Content + e.HTML)
		}
		if !strings.Contains(page.String(), c.have) || strings.Contains(page.String(), c.avoid) {
			t.Errorf("the page of %s holds %q", c.room.RoomName, page.String())
		}

		rec := httptest.NewRecorder()
		d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest("GET", "/export?room="+c.room.RoomName, nil))
		if body := rec.Body.String(); !strings.Contains(body, c.have) || strings.Contains(body, c.avoid) {
			t.Errorf("the export of %s holds %q", c.room.RoomName, body)
		}
	}
}

// A room's guests are the addresses it is reached at, and a colleague of another room is
// not among them.
func TestARoomsPeersAreOnlyItsOwnGuests(t *testing.T) {
	m := testMembership(t)
	self := testIdentity(t)
	a, _ := m.CreateRoom(self.PeerID, false)
	b, _ := m.CreateRoom(self.PeerID, false)
	inA, inB, inBoth := testIdentity(t).PeerID, testIdentity(t).PeerID, testIdentity(t).PeerID
	for peer, addr := range map[string]string{inA: "tc://in-a", inB: "tc://in-b", inBoth: "tc://in-both"} {
		if err := m.Allow(peer, ""); err != nil {
			t.Fatal(err)
		}
		if err := m.SetPeerEndpoint(peer, addr); err != nil {
			t.Fatal(err)
		}
	}
	for _, g := range []struct{ room, peer string }{{a.RoomID, inA}, {a.RoomID, inBoth}, {b.RoomID, inB}, {b.RoomID, inBoth}} {
		if err := m.Invite(g.room, g.peer); err != nil {
			t.Fatal(err)
		}
	}
	has := func(room Room) string {
		got := m.RoomPeers(room.RoomID, self.PeerID)
		sortStrings(got)
		return strings.Join(got, ",")
	}
	if got := has(a); got != "tc://in-a,tc://in-both" {
		t.Errorf("the first room's peers are %q", got)
	}
	if got := has(b); got != "tc://in-b,tc://in-both" {
		t.Errorf("the second room's peers are %q", got)
	}
	if m.IsGuest(a.RoomID, inB) || m.IsGuest(b.RoomID, inA) {
		t.Error("a guest of one room is a guest of the other")
	}
}

func sortStrings(s []string) {
	for i := range s {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// An image is served from the room that holds it, to a guest of that room.
func TestAnImageIsServedOnlyInItsOwnRoomAndOnlyToThatRoomsGuests(t *testing.T) {
	a := newOfflinePeer(t, imageRoom, imageRoomName)
	const otherRoom = "room-other"
	if err := a.d.members.RecordRoom(otherRoom, "other-harbor", false); err != nil {
		t.Fatal(err)
	}
	other, err := OpenStore(otherRoom) // while HOME is still this peer's
	if err != nil {
		t.Fatal(err)
	}
	a.d.stores[otherRoom] = other
	b := newOfflinePeer(t, imageRoom, imageRoomName)
	introduce(t, a, b, imageRoom)

	img := pngBytes(t, 31)
	ref := a.shareImage(t, img, nil)

	// A guest of both rooms asks for it by the room that does not hold it.
	if err := a.d.members.Invite(otherRoom, b.d.id.PeerID); err != nil {
		t.Fatal(err)
	}
	if w := blobAs(t, a.d, b.d.id, imageRoom, ref.SHA256, 0); w.Code != http.StatusOK || !bytes.Equal(w.Body.Bytes(), img) {
		t.Fatalf("a guest of the room is served %d", w.Code)
	}
	if w := blobAs(t, a.d, b.d.id, otherRoom, ref.SHA256, 0); w.Code != http.StatusNotFound {
		t.Errorf("an image of one room is served through another room: %d", w.Code)
	}

	// A guest of the other room only asks for it by the room that holds it.
	c := testIdentity(t)
	if err := a.d.members.Allow(c.PeerID, ""); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.MarkVerified(c.PeerID); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.Invite(otherRoom, c.PeerID); err != nil {
		t.Fatal(err)
	}
	if w := blobAs(t, a.d, c, imageRoom, ref.SHA256, 0); w.Code != http.StatusUnauthorized || w.Body.Len() > 40 {
		t.Errorf("a guest of another room is served %d with %d bytes", w.Code, w.Body.Len())
	}
}

// A session is given its own room's turns and none of another room's, though the same
// colleague writes in both.
func TestASessionIsGivenOnlyItsOwnRoomsTurns(t *testing.T) {
	m := testMembership(t)
	self := testIdentity(t)
	colleague := testIdentity(t)
	if err := m.Allow(colleague.PeerID, "colleague"); err != nil {
		t.Fatal(err)
	}
	if err := m.MarkVerified(colleague.PeerID); err != nil {
		t.Fatal(err)
	}
	first, _ := m.CreateRoom(self.PeerID, false)
	second, _ := m.CreateRoom(self.PeerID, false)
	for _, r := range []Room{first, second} {
		if err := m.Invite(r.RoomID, colleague.PeerID); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.BindSession("in-first", first.RoomID); err != nil {
		t.Fatal(err)
	}
	if err := m.BindSession("in-second", second.RoomID); err != nil {
		t.Fatal(err)
	}
	d := &Daemon{id: self, members: m}
	t.Cleanup(d.closeStores)
	t.Cleanup(d.background.Wait)
	appendFrom(t, d, first, colleague.PeerID, 1, "FIRST-ROOM-SECRET")
	appendFrom(t, d, second, colleague.PeerID, 1, "SECOND-ROOM-SECRET")

	given := func(session string) string {
		t.Helper()
		body, _ := json.Marshal(promptReq{SessionID: session, PromptID: session + "-1", Prompt: "hello"})
		r := httptest.NewRequest(http.MethodPost, "/hook/prompt", bytes.NewReader(body))
		r.Header.Set(localGuardHeader, localGuardValue)
		w := httptest.NewRecorder()
		d.LocalRoutes().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", session, w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	if got := given("in-first"); !strings.Contains(got, "FIRST-ROOM-SECRET") || strings.Contains(got, "SECOND-ROOM-SECRET") {
		t.Errorf("the session in the first room was given: %s", got)
	}
	if got := given("in-second"); !strings.Contains(got, "SECOND-ROOM-SECRET") || strings.Contains(got, "FIRST-ROOM-SECRET") {
		t.Errorf("the session in the second room was given: %s", got)
	}
}
