package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestACoworkSessionIsStoredUnderItsHostAndNeedsBothIdentifiers(t *testing.T) {
	session, prompt := "abc", "p-1"
	if err := (coworkHost{}).NormalizeHook(&session, &prompt, ""); err != nil {
		t.Fatal(err)
	}
	if session != "cowork:abc" || prompt != "p-1" {
		t.Errorf("session %q, prompt %q; the session must carry the host prefix and the prompt identifier must stay as given", session, prompt)
	}
	for _, c := range [][2]string{{"", "p"}, {"s", ""}, {"", ""}} {
		s, p := c[0], c[1]
		if (coworkHost{}).NormalizeHook(&s, &p, "") == nil {
			t.Errorf("a call with session %q and prompt %q was accepted; the stop call could not be matched to its prompt", c[0], c[1])
		}
	}
	if h, ok := hostByName("cowork"); !ok || h.Name() != "cowork" {
		t.Error("the daemon cannot resolve a call that declares cowork")
	}
}

// A Cowork command runs in the daemon from the text of the prompt, so no process carries a
// session in its environment, and no environment may be taken for Cowork's.
func TestCoworkIsNeverFoundInTheEnvironment(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	h, err := hostFromEnv(env(map[string]string{"CLAUDE_CODE_SESSION_ID": "x"}))
	if err != nil || h == nil || h.Name() != "claude" {
		t.Errorf("one Claude Code variable gave %v %v", h, err)
	}
	h, err = hostFromEnv(env(map[string]string{"COGMER_HOST": "cowork"}))
	if err != nil || h == nil || h.Name() != "cowork" {
		t.Errorf("a declared Cowork host gave %v %v", h, err)
	}
	if h, err := hostFromEnv(env(map[string]string{"": "x"})); err != nil || h != nil {
		t.Errorf("an empty variable name matched a host: %v %v", h, err)
	}
	var amb *AmbiguousHostError
	_, err = hostFromEnv(env(map[string]string{"CLAUDE_CODE_SESSION_ID": "x", "CODEX_THREAD_ID": "y"}))
	if !errorsAs(err, &amb) || strings.Contains(strings.Join(amb.Hosts, ","), "cowork") {
		t.Errorf("an ambiguous environment named %v", err)
	}
	t.Setenv("COGMER_HOST", "cowork")
	t.Setenv("CLAUDE_CODE_SESSION_ID", "leaked")
	if got := sessionID(); got != "" {
		t.Errorf("a Cowork command took the session %q from the environment", got)
	}
}

func errorsAs(err error, target **AmbiguousHostError) bool {
	a, ok := err.(*AmbiguousHostError)
	if ok {
		*target = a
	}
	return ok
}

func TestACoworkTurnIsTheLastMessageWithNoToolCount(t *testing.T) {
	turn, err := (coworkHost{}).ReassembleTurn("/in/the/sandbox/t.jsonl", "the final reply", "")
	if err != nil || turn.Text != "the final reply" || len(turn.ToolCalls) != 0 {
		t.Errorf("turn %+v %v", turn, err)
	}
	blocks, err := (coworkHost{}).InjectedBlocks("/in/the/sandbox/t.jsonl")
	if err != nil || len(blocks) != 0 {
		t.Errorf("a transcript the host cannot read produced evidence: %v %v", blocks, err)
	}
}

func TestTheCoworkReplyIsTheJSONFormWithoutANewline(t *testing.T) {
	out := (coworkHost{}).PromptHookOutput("<team-conversation>x</team-conversation>")
	if bytes.HasSuffix(out, []byte("\n")) {
		t.Error("the reply ends in a newline, which is not part of a tool result")
	}
	var v struct {
		H struct {
			Event   string `json:"hookEventName"`
			Context string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out, &v); err != nil || v.H.Event != "UserPromptSubmit" || v.H.Context != "<team-conversation>x</team-conversation>" {
		t.Errorf("reply %q: %v", out, err)
	}
	if !bytes.Equal(bytes.TrimSpace((codexHost{}).PromptHookOutput("c")), (coworkHost{}).PromptHookOutput("c")) {
		t.Error("Codex and Cowork write different JSON; the form is one")
	}
}

// D-206: with no transcript, a turn counts as delivered when its stop call arrives,
// unless the check that the harness still adds the JSON form is recorded as failing.
func TestTheCoworkDeliveryRule(t *testing.T) {
	pending := func(t *testing.T) (*Store, stopReq) {
		s := testStore(t)
		saveVerification(verification{Version: "1.0", Failures: map[string]string{coworkReplyShape: "the JSON form was dropped"}})
		saveVerification(verification{Version: "3.0"})
		seedTeammate(t, s, 1)
		events, _ := s.UndeliveredFor("test", "cowork:s")
		if err := s.RecordPending("turn", "cowork:s", events, "complete block"); err != nil {
			t.Fatal(err)
		}
		return s, stopReq{Host: "cowork", SessionID: "cowork:s", PromptID: "turn", TranscriptPath: "/in/the/sandbox/t.jsonl"}
	}
	undelivered := func(s *Store) int {
		left, _ := s.UndeliveredFor("test", "cowork:s")
		return len(left)
	}
	daemonSeeing := func(t *testing.T, version string) *Daemon {
		d := &Daemon{}
		d.versions.lookup = func(Host) string { return version }
		t.Cleanup(d.background.Wait)
		return d
	}
	for _, c := range []struct {
		name, version string
		delivered     bool
	}{
		{"the check has not failed on this machine's Claude Code", "3.0", true},
		{"the check is recorded as failing", "1.0", false},
		{"no Claude Code on the machine, so no check", "unknown", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			d := daemonSeeing(t, c.version)
			s, req := pending(t)
			d.confirmDelivery(s, req, coworkHost{})
			if undelivered(s) != 1 {
				t.Error("a version not looked up yet was trusted, which loses a turn if the harness changed")
			}
			d.background.Wait()
			d.confirmDelivery(s, req, coworkHost{})
			if got := undelivered(s) == 0; got != c.delivered {
				t.Errorf("delivered = %v, want %v", got, c.delivered)
			}
		})
	}
}

// A room shared by a Claude Code session and a Cowork session carries turns both ways,
// through the daemon's own handlers and with no transcript for Cowork.
func TestMixedRoomWithACoworkSessionCapturesAndDeliversBothDirections(t *testing.T) {
	m := testMembership(t)
	id := testIdentity(t)
	room, _ := m.CreateRoom(id.PeerID, false)
	for _, sid := range []string{"claude", "cowork:cw"} {
		if err := m.BindSession(sid, room.RoomID); err != nil {
			t.Fatal(err)
		}
	}
	d := &Daemon{id: id, members: m}
	d.versions.lookup = func(Host) string { return "3.0" }
	d.hostVersion(coworkHost{}) // looks the version up in the background
	d.background.Wait()
	t.Cleanup(d.closeStores)
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
	post("/hook/prompt", promptReq{SessionID: "claude", PromptID: "claude-turn", Prompt: "FROM CLAUDE"})
	out := post("/hook/prompt", promptReq{Host: "cowork", SessionID: "cw", PromptID: "cw-1", Prompt: "FROM COWORK"})
	block, _ := out["context"].(string)
	if !strings.Contains(block, "FROM CLAUDE") || strings.Contains(block, "FROM COWORK") {
		t.Fatalf("Cowork context = %s", block)
	}
	if got := post("/hook/stop", stopReq{Host: "cowork", SessionID: "cw", PromptID: "cw-1",
		TranscriptPath: "/in/the/sandbox/t.jsonl", LastAssistantMessage: "OMEGA"}); got["stored"] != true {
		t.Fatalf("the Cowork reply was not captured: %v", got)
	}
	again := post("/hook/prompt", promptReq{Host: "cowork", SessionID: "cw", PromptID: "cw-2", Prompt: "second"})
	if text, _ := again["context"].(string); strings.Contains(text, "FROM CLAUDE") {
		t.Errorf("a turn whose stop call arrived was offered to Cowork again: %s", text)
	}
	next := post("/hook/prompt", promptReq{SessionID: "claude", PromptID: "claude-next", Prompt: "next"})
	text, _ := next["context"].(string)
	if !strings.Contains(text, "FROM COWORK") || !strings.Contains(text, "OMEGA") {
		t.Fatalf("Claude context = %s", text)
	}
	if got := post("/hook/stop", stopReq{Host: "cowork", SessionID: "cw", PromptID: "cw-2", LastAssistantMessage: "  "}); got["stored"] != false {
		t.Errorf("an empty reply was stored: %v", got)
	}
	if code := func() int {
		b, _ := json.Marshal(promptReq{Host: "cowork", SessionID: "", PromptID: "x", Prompt: "p"})
		r := httptest.NewRequest(http.MethodPost, "/hook/prompt", bytes.NewReader(b))
		r.Header.Set(localGuardHeader, localGuardValue)
		w := httptest.NewRecorder()
		d.LocalRoutes().ServeHTTP(w, r)
		return w.Code
	}(); code != http.StatusBadRequest {
		t.Errorf("a Cowork call with no session was accepted: %d", code)
	}
}
