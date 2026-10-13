package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const codexTranscript = `
{"type":"event_msg","payload":{"type":"task_started","turn_id":"old"}}
{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"STALE"}]}}
{"type":"event_msg","payload":{"type":"task_started","turn_id":"turn"}}
{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"environment context"}]}}
{"type":"response_item","payload":{"type":"message","id":"alpha","role":"assistant","content":[{"type":"output_text","text":"ALPHA"}],"internal_chat_message_metadata_passthrough":{"turn_id":"turn"}}}
{"type":"event_msg","payload":{"type":"agent_message","message":"ALPHA"}}
{"type":"response_item","payload":{"type":"reasoning","summary":[{"type":"summary_text","text":"PRIVATE"}]}}
{"type":"response_item","payload":{"type":"function_call","call_id":"tool1","name":"exec_command","arguments":"SECRET"}}
{"type":"response_item","payload":{"type":"custom_tool_call","call_id":"tool2","name":"apply_patch","input":"SECRET"}}
{"type":"response_item","payload":{"type":"function_call_output","call_id":"tool1","output":"SECRET"}}
{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"OTHER TURN"}],"internal_chat_message_metadata_passthrough":{"turn_id":"different"}}}
`

func TestCodexCaptureIncludesCommentaryWithoutEventCopiesOrPrivateData(t *testing.T) {
	turn, err := ReassembleCodexTurn(write(t, codexTranscript), "OMEGA", "turn")
	if err != nil {
		t.Fatal(err)
	}
	if turn.Text != "ALPHA\n\nOMEGA" || len(turn.ToolCalls) != 2 {
		t.Fatalf("capture = %+v", turn)
	}
	flushed := codexTranscript + `
{"type":"response_item","payload":{"type":"message","id":"omega","role":"assistant","content":[{"type":"output_text","text":"OMEGA"}]}}
{"type":"response_item","payload":{"type":"message","id":"omega","role":"assistant","content":[{"type":"output_text","text":"OMEGA"}]}}
{"type":"event_msg","payload":{"type":"task_started","turn_id":"next"}}
{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"NEXT TURN"}]}}
`
	turn, err = ReassembleCodexTurn(write(t, flushed), "OMEGA", "turn")
	if err != nil || turn.Text != "ALPHA\n\nOMEGA" {
		t.Fatalf("flushed capture = %+v, %v", turn, err)
	}
}

func TestCodexCaptureRefusesUnidentifiedTurnsAndSubagentTranscripts(t *testing.T) {
	for _, body := range []string{`{"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"WRONG TURN"}]}}`,
		`{"type":"session_meta","payload":{"source":{"subagent":{"thread_spawn":{}}}}}` + "\n" + codexTranscript} {
		if _, err := ReassembleCodexTurn(write(t, body), "OMEGA", "turn"); err == nil {
			t.Error("accepted a transcript without a matching root turn")
		}
	}
	if _, err := ReassembleCodexTurn(write(t, codexTranscript), "OMEGA", ""); err == nil {
		t.Error("accepted an empty turn id")
	}
}

func codexEvidence(role, kind, text string) string {
	b, _ := json.Marshal(map[string]any{"type": "response_item", "payload": map[string]any{"type": "message", "role": role,
		"content": []any{map[string]string{"type": "input_text", "text": text}},
		"internal_chat_message_metadata_passthrough": map[string]any{"content_item_kinds": []string{kind}}}})
	return string(b) + "\n"
}

func TestCodexDeliveryRequiresHookMetadataAndTheCompleteBlock(t *testing.T) {
	s := testStore(t)
	seedTeammate(t, s, 1)
	events, _ := s.UndeliveredFor("test", "codex:mine")
	block := FormatTeamContext(events, nil)
	_ = s.RecordPending("turn", "codex:mine", events, block)
	for _, record := range []string{codexEvidence("user", "hooks.additional_context", block),
		codexEvidence("assistant", "hooks.additional_context", block), codexEvidence("developer", "unknown", block),
		codexEvidence("developer", "hooks.additional_context", block[:30])} {
		blocks, err := CodexInjectedBlocks(write(t, record))
		if err != nil {
			t.Fatal(err)
		}
		for _, b := range blocks {
			if n, _ := s.ConfirmDelivered("codex:mine", []string{HashBlock(b)}); n != 0 {
				t.Fatal("unproven delivery was confirmed")
			}
		}
	}
	blocks, err := CodexInjectedBlocks(write(t, codexEvidence("developer", "hooks.additional_context", block)))
	if err != nil || len(blocks) != 1 {
		t.Fatalf("blocks=%v, %v", blocks, err)
	}
	if n, err := s.ConfirmDelivered("codex:mine", []string{HashBlock(blocks[0])}); err != nil || n != 1 {
		t.Fatalf("confirmed=%d, %v", n, err)
	}
}

func TestCodexSessionIdentityDoesNotCollideWithClaude(t *testing.T) {
	t.Setenv("CLAUDE_CODE_SESSION_ID", "same")
	t.Setenv("CODEX_THREAD_ID", "same")
	t.Setenv("COGMER_HOST", "")
	if sessionID() != "same" {
		t.Fatal("Claude session changed identity")
	}
	t.Setenv("COGMER_HOST", "codex")
	if sessionID() != "codex:same" {
		t.Fatal("Codex session has no host namespace")
	}
	t.Setenv("CODEX_THREAD_ID", "")
	if sessionID() != "" {
		t.Fatal("Codex command borrowed the ambient Claude session")
	}
	t.Setenv("CLAUDE_CODE_SESSION_ID", "")
	t.Setenv("CODEX_THREAD_ID", "same")
	t.Setenv("COGMER_HOST", "")
	if sessionID() != "codex:same" {
		t.Fatal("native Codex environment was not recognized")
	}
}

func TestMixedHostRoomCapturesAndDeliversBothDirections(t *testing.T) {
	m := testMembership(t)
	id := testIdentity(t)
	room, _ := m.CreateRoom(id.PeerID, false)
	for _, sid := range []string{"claude", "codex:codex"} {
		if err := m.BindSession(sid, room.RoomID); err != nil {
			t.Fatal(err)
		}
	}
	d := &Daemon{id: id, members: m}
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
	out := post("/hook/prompt", promptReq{Host: "codex", SessionID: "codex", TurnID: "turn", Prompt: "FROM CODEX"})
	block, _ := out["context"].(string)
	if !strings.Contains(block, "FROM CLAUDE") || strings.Contains(block, "FROM CODEX") {
		t.Fatalf("Codex context = %s", block)
	}
	path := write(t, codexTranscript+codexEvidence("developer", "hooks.additional_context", block))
	post("/hook/stop", stopReq{Host: "codex", SessionID: "codex", TurnID: "turn", TranscriptPath: path, LastAssistantMessage: "OMEGA"})
	out = post("/hook/prompt", promptReq{SessionID: "claude", PromptID: "claude-next", Prompt: "next"})
	text, _ := out["context"].(string)
	if !strings.Contains(text, "FROM CODEX") || !strings.Contains(text, "ALPHA") || !strings.Contains(text, "OMEGA") {
		t.Fatalf("Claude context = %s", text)
	}
	store, _ := d.storeFor(room.RoomID)
	left, _ := store.UndeliveredFor(room.RoomID, "codex:codex")
	for _, e := range left {
		if e.Content == "FROM CLAUDE" {
			t.Fatal("confirmed Claude prompt was offered again to Codex")
		}
	}
	_ = store.Close()
}

func TestCodexHookRejectsMissingTurnIdentity(t *testing.T) {
	for _, host := range []string{"codex", "unsupported"} {
		sid, pid := "session", ""
		if normalizeHookSession(host, &sid, &pid, "") == nil {
			t.Errorf("accepted %s without a turn", host)
		}
	}
}

func TestB25DetectsLostOrMismatchedCodexHookIdentity(t *testing.T) {
	good := &Probe{Hooks: map[string][]map[string]any{
		"UserPromptSubmit": {{"session_id": "s", "turn_id": "t", "transcript_path": "p", "prompt": "go"}},
		"Stop":             {{"session_id": "s", "turn_id": "t", "transcript_path": "p", "last_assistant_message": "done"}},
	}}
	mustPass(t, "B25", good)
	good.Hooks["Stop"][0]["turn_id"] = "wrong"
	mustDetect(t, "B25", good, "a mismatched turn")
	delete(good.Hooks["Stop"][0], "session_id")
	mustDetect(t, "B25", good, "a missing session")
}

func TestB26DetectsPartialCodexCapture(t *testing.T) {
	p := &Probe{Reassembled: &AssistantTurn{Text: "ALPHA\n\nOMEGA", ToolCalls: []string{"exec"}}}
	mustPass(t, "B26", p)
	p.Reassembled.Text = "OMEGA"
	mustDetect(t, "B26", p, "lost commentary")
	p.Reassembled.Text = "ALPHA\n\nOMEGA"
	p.Reassembled.ToolCalls = nil
	mustDetect(t, "B26", p, "lost tools")
}

func TestB27DetectsCodexContextLossOrMissingDeliveryEvidence(t *testing.T) {
	p := &Probe{Sentinel: "CODEWORD", Final: "CODEWORD", InjectedText: "complete block", ObservedBlocks: []string{"complete block"}}
	mustPass(t, "B27", p)
	p.ObservedBlocks = []string{"complete"}
	mustDetect(t, "B27", p, "a truncated attachment")
	p.ObservedBlocks = []string{"complete block"}
	p.Final = "LOST"
	mustDetect(t, "B27", p, "context that never reached the model")
}

func TestB28DetectsAnAbsentOrDifferentCodexToolSession(t *testing.T) {
	p := &Probe{SessionID: "s", ToolEnvSessionID: "s"}
	mustPass(t, "B28", p)
	p.ToolEnvSessionID = "other"
	mustDetect(t, "B28", p, "a different tool session")
	p.SessionID = ""
	p.ToolEnvSessionID = ""
	mustDetect(t, "B28", p, "two missing identifiers")
}

func TestCodexCannotUseAClaudeDeliveryFallback(t *testing.T) {
	s := testStore(t)
	seedTeammate(t, s, 1)
	events, _ := s.UndeliveredFor("test", "codex:mine")
	_ = s.RecordPending("turn", "codex:mine", events, "complete block")
	saveVerification(verification{Version: "test-claude", Failures: map[string]string{"B20": "missing attachment"}})
	d := &Daemon{claudeVersion: "test-claude"}
	d.confirmDelivery(s, stopReq{Host: "codex", SessionID: "codex:mine", PromptID: "turn", TranscriptPath: write(t, codexTranscript)})
	left, _ := s.UndeliveredFor("test", "codex:mine")
	if len(left) != 1 {
		t.Fatal("Claude's broken attachment check confirmed a Codex injection without evidence")
	}
}

func TestCodexHookOutputsValidContextAndFailsOpen(t *testing.T) {
	bin := buildVersion(t, t.TempDir(), "codex-hook-test")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(localGuardHeader) != localGuardValue {
			t.Error("hook omitted the local guard")
		}
		var req promptReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Host != "codex" || req.TurnID != "turn" {
			t.Errorf("Codex payload = %+v, %v", req, err)
		}
		writeJSON(w, map[string]string{"context": "quote: \" <team-conversation>\ncomplete block"})
	}))
	defer server.Close()
	run := func(addr, payload, kind string) []byte {
		t.Helper()
		cmd := exec.Command(bin, "hook", kind)
		cmd.Env = append(os.Environ(), "COGMER_ADDR="+addr)
		cmd.Stdin = strings.NewReader(payload)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("hook did not fail open: %v", err)
		}
		return out
	}
	out := run(strings.TrimPrefix(server.URL, "http://"), `{"session_id":"mine","turn_id":"turn","prompt":"go"}`, "codex-prompt")
	var decoded struct {
		Output struct {
			Event   string `json:"hookEventName"`
			Context string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if json.Unmarshal(out, &decoded) != nil || decoded.Output.Event != "UserPromptSubmit" || !strings.Contains(decoded.Output.Context, "complete block") {
		t.Fatalf("invalid Codex output: %s", out)
	}
	for _, kind := range []string{"codex-prompt", "codex-stop"} {
		for _, payload := range []string{`{"session_id":"mine","turn_id":"turn"}`, "{partial", ""} {
			if out := run("127.0.0.1:1", payload, kind); len(out) != 0 {
				t.Fatalf("failed hook emitted %s", out)
			}
		}
	}
}

func TestCodexCommandWrapperRequiresItsOwnSessionAndRendersItsCommands(t *testing.T) {
	bin := buildVersion(t, t.TempDir(), "codex-command-test")
	root, err := filepath.Abs("../../plugin")
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	run := func(sid string, args ...string) string {
		t.Helper()
		cmd := exec.Command("bash", append([]string{filepath.Join(root, "codex-cli.sh")}, args...)...)
		cmd.Env = append(os.Environ(), "COGMER_BIN="+bin, "COGMER_HOME="+home, "COGMER_PREFLIGHT=off", "CODEX_THREAD_ID="+sid, "CLAUDE_CODE_SESSION_ID=ambient-claude")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err)
		}
		return string(out)
	}
	if out := run("", "create"); !strings.Contains(out, "No Codex session") {
		t.Fatalf("borrowed a Claude session: %s", out)
	}
	if out := run("codex-one", "create", "--show-only"); !strings.Contains(out, "this session is now in") || !strings.Contains(out, "your agent") {
		t.Fatalf("create: %s", out)
	}
	if out := run("codex-two", "log"); !strings.Contains(out, "$cogmer room-join") || strings.Contains(out, "/cogmer:") {
		t.Fatalf("host commands: %s", out)
	}
	t.Setenv("COGMER_HOME", home)
	m, err := OpenMembership()
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if _, ok := m.RoomForSession("codex:codex-one"); !ok {
		t.Fatal("created room did not bind Codex session")
	}
	if _, ok := m.RoomForSession("ambient-claude"); ok {
		t.Fatal("Codex command bound the inherited Claude session")
	}
}

func TestSharedStartupEmitsOneJSONValueWhileInstallAndDaemonFail(t *testing.T) {
	root, home := cliHarness(t)
	src, err := os.ReadFile("../../plugin/hooks-handlers/session-start.sh")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "hooks-handlers", "session-start.sh")
	if err := os.WriteFile(path, src, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", path)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + home, "COGMER_HOME=" + home, "CLAUDE_PLUGIN_ROOT=" + root, "COGMER_BIN=/missing", "COGMER_ADDR=127.0.0.1:1"}
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(out, &value); err != nil {
		t.Fatalf("startup emitted more than one JSON value: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "standing policy") || !strings.Contains(string(out), "not been fetched") {
		t.Fatalf("startup lost policy or installation status: %s", out)
	}
	if !waitFor(t, filepath.Join(home, "started")) {
		t.Fatal("startup did not start the installer")
	}
}
