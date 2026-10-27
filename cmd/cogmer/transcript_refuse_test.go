//go:build !windows

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// refusedPaths makes one of each kind of path a hook could name that is not a transcript.
func refusedPaths(t *testing.T) map[string]string {
	t.Helper()
	dir := t.TempDir()
	out := map[string]string{}
	out["a FIFO"] = filepath.Join(dir, "fifo.jsonl")
	if err := syscall.Mkfifo(out["a FIFO"], 0o600); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	out["a directory"] = filepath.Join(dir, "dir.jsonl")
	if err := os.Mkdir(out["a directory"], 0o700); err != nil {
		t.Fatal(err)
	}
	out["a device"] = "/dev/null"
	out["a file with another name"] = filepath.Join(dir, "secrets.txt")
	writeFile(t, out["a file with another name"], []byte(strings.TrimSpace(raceTranscript)+"\n"))
	return out
}

// within fails the test if f does not return, which is what opening a FIFO does.
func within(t *testing.T, what string, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() { defer close(done); f() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("%s did not return", what)
	}
}

func TestEveryTranscriptReaderRefusesWhatIsNotARegularJSONLFile(t *testing.T) {
	for name, p := range refusedPaths(t) {
		t.Run(name, func(t *testing.T) {
			within(t, "a transcript reader", func() {
				if _, err := ReassembleLastTurn(p, "x"); !errors.Is(err, errTranscriptRefused) {
					t.Errorf("ReassembleLastTurn: %v", err)
				}
				if _, err := InjectedBlocks(p); !errors.Is(err, errTranscriptRefused) {
					t.Errorf("InjectedBlocks: %v", err)
				}
				if _, err := ReassembleCodexTurn(p, "x", "turn"); !errors.Is(err, errTranscriptRefused) {
					t.Errorf("ReassembleCodexTurn: %v", err)
				}
				if _, err := CodexInjectedBlocks(p); !errors.Is(err, errTranscriptRefused) {
					t.Errorf("CodexInjectedBlocks: %v", err)
				}
				if imgs, found, _, _ := codexTurnImages(p, "turn", "x", 0); found || imgs != nil {
					t.Errorf("codexTurnImages read %s", name)
				}
			})
		})
	}
}

func TestARefusalNeverQuotesThePath(t *testing.T) {
	p := filepath.Join(t.TempDir(), "SECRET-NAME.txt")
	writeFile(t, p, []byte("x"))
	_, err := ReassembleLastTurn(p, "x")
	if err == nil || strings.Contains(err.Error(), "SECRET-NAME") {
		t.Fatalf("error = %v", err)
	}
}

func TestATranscriptOverTheSizeLimitIsRefused(t *testing.T) {
	p := filepath.Join(t.TempDir(), "big.jsonl")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxTranscriptBytes + 1); err != nil {
		t.Skipf("no sparse files here: %v", err)
	}
	f.Close()
	if _, err := ReassembleLastTurn(p, "x"); !errors.Is(err, errTranscriptRefused) {
		t.Errorf("an oversize transcript: %v", err)
	}
	if err := os.Truncate(p, maxTranscriptBytes); err != nil {
		t.Fatal(err)
	}
	if _, err := ReassembleLastTurn(p, "x"); errors.Is(err, errTranscriptRefused) {
		t.Errorf("a transcript at the limit was refused: %v", err)
	}
}

func TestARealTranscriptOfEachHostStillReads(t *testing.T) {
	if turn, err := ReassembleLastTurn(write(t, raceTranscript), "OMEGA"); err != nil || turn.Text != "ALPHA\n\nOMEGA" {
		t.Errorf("Claude Code transcript: %+v, %v", turn, err)
	}
	if turn, err := ReassembleCodexTurn(write(t, codexTranscript), "OMEGA", "turn"); err != nil || turn.Text != "ALPHA\n\nOMEGA" {
		t.Errorf("Codex transcript: %+v, %v", turn, err)
	}
	rollout := filepath.Join(t.TempDir(), "rollout-2026-10-27T10-00-00-0199aaaa-bbbb-cccc-dddd-eeeeeeeeeeee.jsonl")
	writeFile(t, rollout, []byte(strings.TrimSpace(codexTranscript)+"\n"))
	if _, err := ReassembleCodexTurn(rollout, "OMEGA", "turn"); err != nil {
		t.Errorf("a rollout name: %v", err)
	}
}

// A Stop hook naming a FIFO is answered, and the turn is not stored.
func TestAStopHookNamingAFIFOIsAnsweredAndNothingIsStored(t *testing.T) {
	m := testMembership(t)
	id := testIdentity(t)
	room, _ := m.CreateRoom(id.PeerID, false)
	if err := m.BindSession("s", room.RoomID); err != nil {
		t.Fatal(err)
	}
	d := &Daemon{id: id, members: m}
	t.Cleanup(d.closeStores)
	fifo := filepath.Join(t.TempDir(), "t.jsonl")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	b, _ := json.Marshal(stopReq{SessionID: "s", PromptID: "p", TranscriptPath: fifo, LastAssistantMessage: "OMEGA"})
	r := httptest.NewRequest(http.MethodPost, "/hook/stop", bytes.NewReader(b))
	r.Header.Set(localGuardHeader, localGuardValue)
	w := httptest.NewRecorder()
	within(t, "the stop hook", func() { d.LocalRoutes().ServeHTTP(w, r) })
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if w.Code != http.StatusOK || out["stored"] != false {
		t.Errorf("%d %s", w.Code, w.Body.String())
	}
}
