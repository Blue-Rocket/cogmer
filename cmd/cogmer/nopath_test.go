package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A Cowork cloud task sends no transcript path, so nothing it was offered can be confirmed from
// one. Its stop call counts the turns offered at that prompt as delivered, and the next prompt is
// given only what is new (D-296). A path that is given and is refused confirms nothing.
func TestASessionWithNoTranscriptPathIsNotOfferedTheRoomAgain(t *testing.T) {
	m := testMembership(t)
	id := testIdentity(t)
	room, _ := m.CreateRoom(id.PeerID, false)
	for _, sid := range []string{"a", "cloud"} {
		if err := m.BindSession(sid, room.RoomID); err != nil {
			t.Fatal(err)
		}
	}
	d := &Daemon{id: id, members: m}
	t.Cleanup(d.closeStores)
	t.Cleanup(d.background.Wait)
	post := func(endpoint string, payload any) map[string]any {
		t.Helper()
		b, _ := json.Marshal(payload)
		r := httptest.NewRequest(http.MethodPost, endpoint, bytes.NewReader(b))
		r.Header.Set(localGuardHeader, localGuardValue)
		w := httptest.NewRecorder()
		d.LocalRoutes().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", endpoint, w.Code, w.Body.String())
		}
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return out
	}
	given := func(prompt, promptID string) string {
		t.Helper()
		text, _ := post("/hook/prompt", promptReq{Host: "claude", SessionID: "cloud", PromptID: promptID, Prompt: prompt})["context"].(string)
		return text
	}

	post("/hook/prompt", promptReq{Host: "claude", SessionID: "a", PromptID: "a1", Prompt: "FROM A"})
	if text := given("one", "c1"); !strings.Contains(text, "FROM A") {
		t.Fatalf("the first prompt was not given the colleague's turn: %s", text)
	}
	post("/hook/stop", stopReq{Host: "claude", SessionID: "cloud", PromptID: "c1", TranscriptPath: "", LastAssistantMessage: "R1"})
	if text := given("two", "c2"); strings.Contains(text, "FROM A") {
		t.Errorf("a turn offered at a prompt whose stop call arrived with no transcript path was offered again: %s", text)
	}
	post("/hook/stop", stopReq{Host: "claude", SessionID: "cloud", PromptID: "c2", TranscriptPath: "", LastAssistantMessage: "R2"})

	post("/hook/prompt", promptReq{Host: "claude", SessionID: "a", PromptID: "a2", Prompt: "NEWER"})
	text := given("three", "c3")
	if !strings.Contains(text, "NEWER") || strings.Contains(text, "FROM A") {
		t.Errorf("the prompt after a new turn was given %s, want only the new turn", text)
	}

	// A path that is given and is refused is not the absence of a transcript, and confirms nothing.
	post("/hook/stop", stopReq{Host: "claude", SessionID: "cloud", PromptID: "c3", TranscriptPath: "/some/where/t.json", LastAssistantMessage: "R3"})
	if text := given("four", "c4"); !strings.Contains(text, "NEWER") {
		t.Errorf("a turn was counted delivered on a refused transcript path: %s", text)
	}
}
