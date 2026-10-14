package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTheCoworkLineSaysWhetherCoworkHasReachedThisMachine(t *testing.T) {
	cw := []string{"claude-session", "cowork:abc", "cowork:def"}
	for _, c := range []struct {
		name     string
		sessions []string
		calls    map[string]string
		up       bool
		want     []string
		none     bool
	}{
		{"calls have arrived", cw, map[string]string{"cowork": "3m"}, true, []string{"2 session(s)", "last reached this machine 3m ago"}, false},
		{"none has arrived", cw, map[string]string{"claude": "1s"}, true, []string{"no call from Cowork", "yet", "typing a cogmer command"}, false},
		{"the room has no Cowork session", []string{"claude-session", "codex:x"}, nil, true, nil, true},
		{"no daemon answered", cw, nil, false, nil, true},
	} {
		got := coworkLine(c.sessions, c.calls, c.up)
		if c.none {
			if got != "" {
				t.Errorf("%s: said %q", c.name, got)
			}
			continue
		}
		for _, w := range c.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: %q lacks %q", c.name, got, w)
			}
		}
		// a user has no use for how long the daemon has run
		for _, bad := range []string{"daemon", "started", "restart", "never"} {
			if strings.Contains(strings.ToLower(got), bad) {
				t.Errorf("%s: %q says %q", c.name, got, bad)
			}
		}
	}
}

func TestTheDaemonSaysWhenEachHostLastCalled(t *testing.T) {
	m := testMembership(t)
	id := testIdentity(t)
	room, _ := m.CreateRoom(id.PeerID, false)
	for _, s := range []string{"claude", "cowork:cw"} {
		if err := m.BindSession(s, room.RoomID); err != nil {
			t.Fatal(err)
		}
	}
	d := &Daemon{id: id, members: m}
	t.Cleanup(d.closeStores)
	peers := func() map[string]string {
		t.Helper()
		w := httptest.NewRecorder()
		d.LocalRoutes().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/peers", nil))
		var body struct {
			HostCalls map[string]string `json:"hostCalls"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.HostCalls
	}
	if got := peers(); len(got) != 0 {
		t.Errorf("a daemon nothing has called reports %v", got)
	}
	post := func(endpoint string, payload any) {
		b, _ := json.Marshal(payload)
		r := httptest.NewRequest(http.MethodPost, endpoint, bytes.NewReader(b))
		r.Header.Set(localGuardHeader, localGuardValue)
		w := httptest.NewRecorder()
		d.LocalRoutes().ServeHTTP(w, r)
	}
	post("/hook/prompt", promptReq{Host: "cowork", SessionID: "cw", PromptID: "p", Prompt: "hello"})
	got := peers()
	if _, ok := got["cowork"]; !ok || len(got) != 1 {
		t.Errorf("after a Cowork prompt the hosts that called are %v", got)
	}
	post("/hook/stop", stopReq{SessionID: "claude", PromptID: "p", LastAssistantMessage: "x"})
	if got := peers(); len(got) != 2 {
		t.Errorf("after a Claude Code stop the hosts that called are %v", got)
	}
	post("/hook/prompt", promptReq{Host: "emacs", SessionID: "x", PromptID: "p", Prompt: "hello"})
	if got := peers(); len(got) != 2 {
		t.Errorf("a call from a host cogmer does not support was counted: %v", got)
	}
	// a Cowork call that is refused for lack of an identifier is not a call that arrived
	d2 := &Daemon{id: id, members: m}
	post2 := func() {
		b, _ := json.Marshal(promptReq{Host: "cowork", SessionID: "", PromptID: "p"})
		r := httptest.NewRequest(http.MethodPost, "/hook/prompt", bytes.NewReader(b))
		r.Header.Set(localGuardHeader, localGuardValue)
		d2.LocalRoutes().ServeHTTP(httptest.NewRecorder(), r)
	}
	post2()
	if len(d2.hostCalls.ages()) != 0 {
		t.Error("a refused call was counted as one that arrived")
	}
}

func TestTheSessionsOfARoomAreListedByTheirIdentifiers(t *testing.T) {
	m := testMembership(t)
	id := testIdentity(t)
	room, _ := m.CreateRoom(id.PeerID, false)
	other, _ := m.CreateRoom(id.PeerID, false)
	for _, s := range []string{"cowork:b", "a", "gone"} {
		if err := m.BindSession(s, room.RoomID); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.BindSession("elsewhere", other.RoomID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.LeaveSession("gone"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(m.SessionIDsInRoom(room.RoomID), ","); got != "a,cowork:b" {
		t.Errorf("the sessions in the room are %q, want a,cowork:b", got)
	}
}
