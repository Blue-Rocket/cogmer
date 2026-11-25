package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func fresh(t *testing.T) {
	t.Helper()
	markDaemonStarting()
	if _, err := os.Stat(startingMarker()); err != nil {
		t.Fatalf("the marker was not written: %v", err)
	}
}

func spoolFiles(t *testing.T) []string {
	t.Helper()
	entries, _ := os.ReadDir(spoolDir())
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func promptBody(t *testing.T, session, prompt, id string) []byte {
	t.Helper()
	b, err := json.Marshal(promptReq{Host: "claude", SessionID: session, Prompt: prompt, PromptID: id})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A prompt is kept for a daemon that was just started, and only then (D-294).
func TestAPromptIsKeptOnlyWhileADaemonIsStarting(t *testing.T) {
	setHome(t, t.TempDir())
	body := promptBody(t, "s", "kept", "p1")

	if spoolPrompt(body, time.Now()) {
		t.Fatal("a prompt was kept with no daemon being started")
	}
	fresh(t)
	if !spoolPrompt(body, time.Now()) {
		t.Fatal("a prompt was not kept for a daemon that was just started")
	}
	if got := spoolFiles(t); len(got) != 1 || !strings.HasSuffix(got[0], ".json") {
		t.Fatalf("the spool holds %v, want one .json file", got)
	}
	if runtime.GOOS != "windows" {
		for path, want := range map[string]os.FileMode{spoolDir(): 0o700, filepath.Join(spoolDir(), spoolFiles(t)[0]): 0o600} {
			if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != want {
				t.Errorf("%s has mode %v (%v), want %v: a kept prompt is the user's words", path, fi.Mode().Perm(), err, want)
			}
		}
	}
	// A start that is old keeps nothing.
	old := time.Now().Add(-2 * spoolMarkerFresh)
	if err := os.Chtimes(startingMarker(), old, old); err != nil {
		t.Fatal(err)
	}
	if spoolPrompt(body, time.Now()) {
		t.Error("a prompt was kept for a start that is older than the limit")
	}
}

func TestTheSpoolIsBounded(t *testing.T) {
	setHome(t, t.TempDir())
	fresh(t)
	for i := 0; i < spoolMaxFiles; i++ {
		if !spoolPrompt(promptBody(t, "s", "p", "id"), time.Now().Add(time.Duration(i))) {
			t.Fatalf("prompt %d was not kept", i)
		}
	}
	if spoolPrompt(promptBody(t, "s", "one more", "id"), time.Now()) {
		t.Errorf("the spool took more than %d prompts", spoolMaxFiles)
	}
}

func TestOnlyAConnectionThatWasNeverMadeCountsAsADialError(t *testing.T) {
	if isDialError(nil) {
		t.Error("nil is a dial error")
	}
	// A refused connection to a port nothing listens on.
	err := func() error {
		_, err := localPost(&http.Client{Timeout: 2 * time.Second}, "http://"+freeAddr(t)+"/hook/prompt", []byte("{}"))
		return err
	}()
	if !isDialError(err) {
		t.Errorf("a refused connection is not a dial error: %v", err)
	}
}

// The daemon records a kept prompt once, in its session's room, and drops one that is too old, one
// from a session in no room, and a command.
func TestTheDaemonRecordsAKeptPromptOnceAndDropsTheRest(t *testing.T) {
	d, room, _ := promptRoom(t, false)
	fresh(t)
	if !spoolPrompt(promptBody(t, "s", "the kept words", "p1"), time.Now()) {
		t.Fatal("not kept")
	}
	if !spoolPrompt(promptBody(t, "no-such-session", "nobody's", "p2"), time.Now().Add(time.Millisecond)) {
		t.Fatal("not kept")
	}
	if !spoolPrompt(promptBody(t, "s", "/cogmer:room-log", "p3"), time.Now().Add(2*time.Millisecond)) {
		t.Fatal("not kept")
	}
	if err := writeSpool(promptBody(t, "s", "too old", "p4"), time.Now().Add(-2*spoolMaxAge)); err != nil {
		t.Fatal(err)
	}

	d.replaySpool()
	d.replaySpool() // a second run finds nothing, so nothing is recorded twice

	store, err := d.storeFor(room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	evs, err := store.ListRoom(room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	var prompts []string
	for _, e := range evs {
		if e.EventType == EventUserPrompt {
			prompts = append(prompts, e.Content)
		}
	}
	if len(prompts) != 1 || prompts[0] != "the kept words" {
		t.Errorf("the room holds the prompts %q, want only the kept one, once", prompts)
	}
	if got := spoolFiles(t); len(got) != 0 {
		t.Errorf("the spool still holds %v", got)
	}
	if _, err := os.Stat(startingMarker()); err == nil {
		t.Error("the marker stays after the daemon recorded the spool, so later prompts would be kept")
	}
}

// The hook itself: with nothing listening, a prompt is kept when a start is fresh and is not
// otherwise, and the hook still exits 0 and writes nothing to the model.
func TestAPromptHookKeepsItsPromptForAStartingDaemon(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	bin := buildVersion(t, t.TempDir(), "0.0.1-spool")
	home := filepath.Join(t.TempDir(), ".cogmer")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	run := func(prompt string) (string, string) {
		cmd := exec.Command(bin, "hook", "prompt")
		cmd.Env = append(os.Environ(), "COGMER_HOME="+home, "COGMER_ADDR="+freeAddr(t), "COGMER_PEER_ADDR="+freeAddr(t),
			"COGMER_TAILCAT=off", "COGMER_HOST=claude", "COGMER_PREFLIGHT=off", "CLAUDE_CODE_SESSION_ID=s-1")
		b, _ := json.Marshal(map[string]string{"session_id": "s-1", "prompt": prompt, "prompt_id": "p-1"})
		cmd.Stdin = bytes.NewReader(b)
		var out, errOut bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errOut
		if err := cmd.Run(); err != nil {
			t.Fatalf("the hook failed: %v", err)
		}
		return out.String(), errOut.String()
	}
	if out, _ := run("words to keep"); out != "" {
		t.Errorf("the hook wrote %q to the model", out)
	}
	if entries, _ := os.ReadDir(filepath.Join(home, "spool")); len(entries) != 0 {
		t.Fatalf("a prompt was kept with no daemon being started: %v", entries)
	}
	if err := os.WriteFile(filepath.Join(home, "daemon-starting"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	// A command is never kept, since its text can hold a pairing string or an invitation (D-209).
	run("/cogmer:peer-pair ed25519:secretish somebody")
	if entries, _ := os.ReadDir(filepath.Join(home, "spool")); len(entries) != 0 {
		t.Fatalf("a typed command was written to the spool: %v", entries)
	}
	if out, errOut := run("words to keep"); out != "" || !strings.Contains(errOut, "daemon unreachable") {
		t.Errorf("with a fresh start the hook wrote %q to the model and %q to stderr", out, errOut)
	}
	entries, _ := os.ReadDir(filepath.Join(home, "spool"))
	if len(entries) != 1 {
		t.Fatalf("the spool holds %v, want the prompt", entries)
	}
	b, err := os.ReadFile(filepath.Join(home, "spool", entries[0].Name()))
	if err != nil || !strings.Contains(string(b), "words to keep") {
		t.Errorf("the kept prompt does not hold the words: %v %q", err, b)
	}
}

// The plugin's start function says that a daemon is being started, which is what lets a prompt
// hook keep its prompt in the seconds before the daemon listens (D-294).
func TestTheSessionStartHookMarksThatADaemonIsStarting(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the hook is a shell script")
	}
	standIn := filepath.Join(t.TempDir(), "cogmer")
	if err := os.WriteFile(standIn, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, home := runStartIn(t, "COGMER_BIN="+standIn)
	// The function returns before the stand-in daemon has run, so the marker is there at once.
	if fi, err := os.Stat(filepath.Join(home, "daemon-starting")); err != nil || time.Since(fi.ModTime()) > spoolMarkerFresh {
		t.Errorf("the session-start hook left no fresh marker that a daemon is starting: %v", err)
	}
}

// A Cowork cloud task sends transcript_path as an empty string, and its replies were refused for
// not having a .jsonl name. With no path there is no transcript, so the turn is the Stop hook's
// last message, and a path that is given but is not a transcript is still refused (D-262).
func TestAReplyWithNoTranscriptPathIsRecordedFromTheLastMessage(t *testing.T) {
	d, room, _ := promptRoom(t, false)
	stop := func(path, message string) {
		t.Helper()
		b, _ := json.Marshal(stopReq{Host: "claude", SessionID: "s", PromptID: "p1", TranscriptPath: path, LastAssistantMessage: message})
		r := httptest.NewRequest(http.MethodPost, "/hook/stop", bytes.NewReader(b))
		r.Header.Set(localGuardHeader, localGuardValue)
		w := httptest.NewRecorder()
		d.LocalRoutes().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("/hook/stop: %d %s", w.Code, w.Body.String())
		}
	}
	replies := func() []string {
		store, err := d.storeFor(room.RoomID)
		if err != nil {
			t.Fatal(err)
		}
		evs, err := store.ListRoom(room.RoomID)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, e := range evs {
			if e.EventType == EventAssistantMessage {
				out = append(out, e.Content)
			}
		}
		return out
	}

	stop("", "  the final words  ")
	if got := replies(); len(got) != 1 || got[0] != "the final words" {
		t.Fatalf("with no transcript path the room holds the replies %q, want the last message", got)
	}
	stop("", "")
	if got := replies(); len(got) != 1 {
		t.Errorf("an empty last message was recorded: %q", got)
	}
	stop("/some/where/transcript.json", "not recorded")
	if got := replies(); len(got) != 1 {
		t.Errorf("a path that is not a .jsonl transcript was read or recorded: %q", got)
	}
}
