package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// A session-start hook starts the daemon and returns at once, and the prompt hook of a woken
// Cowork cloud task runs about a second later, before the daemon listens. That hook found no
// daemon and failed open, so the prompt reached no room. A prompt hook that finds nothing
// listening, within a few seconds of a start, writes the prompt here, and the daemon records it
// when it starts (D-294, a prompt sent while the daemon starts is kept for it).
const (
	// spoolMarkerFresh is how long after a daemon was started a prompt is kept for it. The marker
	// is written by whatever starts the daemon, and a daemon that never starts leaves nothing to
	// keep prompts for after this long.
	spoolMarkerFresh = 15 * time.Second
	// spoolMaxAge is how old a kept prompt may be when the daemon finds it. An older one is
	// dropped, so that a prompt is never published long after its turn.
	spoolMaxAge = 60 * time.Second
	// spoolMaxFiles bounds the directory against a daemon that never comes up.
	spoolMaxFiles = 50
)

func spoolDir() string       { return filepath.Join(homeDir(), "spool") }
func startingMarker() string { return filepath.Join(homeDir(), "daemon-starting") }

// markDaemonStarting says that a daemon was just started, as the plugin's start function does
// for the hooks of Claude Code and Codex.
func markDaemonStarting() {
	if ensureStateDir() != nil {
		return
	}
	f, err := os.OpenFile(startingMarker(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return
	}
	f.Close()
	now := time.Now()
	_ = os.Chtimes(startingMarker(), now, now)
}

// A kept prompt holds the hook's request exactly as it would have been posted, and when.
type spooledPrompt struct {
	At   time.Time       `json:"at"`
	Body json.RawMessage `json:"body"`
}

// isDialError says that a request never reached a server: nothing was listening. Only then is it
// kept, since a request that timed out may have been handled and a kept copy would repeat it.
func isDialError(err error) bool {
	var op *net.OpError
	return errors.As(err, &op) && op.Op == "dial"
}

// spoolPrompt keeps a prompt for the daemon that was just started, and says whether it did. It
// keeps one only when the marker is fresh and the directory is not full.
func spoolPrompt(body []byte, now time.Time) bool {
	st, err := os.Stat(startingMarker())
	if err != nil || now.Sub(st.ModTime()) > spoolMarkerFresh {
		return false
	}
	return writeSpool(body, now) == nil
}

func writeSpool(body []byte, at time.Time) error {
	dir := spoolDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) >= spoolMaxFiles {
		return fmt.Errorf("the spool holds %d prompts", len(entries))
	}
	b, err := json.Marshal(spooledPrompt{At: at.UTC(), Body: body})
	if err != nil {
		return err
	}
	name := fmt.Sprintf("%020d-%d", at.UnixNano(), os.Getpid())
	tmp := filepath.Join(dir, name+".tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, name+".json"))
}

// replaySpool records the prompts a hook kept while this daemon started, once each and in the
// order they were kept, and drops any that are too old. It then removes the marker, so that a
// prompt is kept only while a start is under way.
func (d *Daemon) replaySpool() {
	dir := spoolDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		_ = os.Remove(startingMarker())
		return
	}
	captured, dropped := 0, 0
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		if !strings.HasSuffix(e.Name(), ".json") {
			// What a hook or an earlier daemon left half done.
			if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > spoolMaxAge {
				_ = os.Remove(path)
			}
			continue
		}
		// Taken by renaming, so that two daemons that start together record it once.
		taken := path + ".taken"
		if os.Rename(path, taken) != nil {
			continue
		}
		b, err := os.ReadFile(taken)
		_ = os.Remove(taken)
		var sp spooledPrompt
		var req promptReq
		if err != nil || json.Unmarshal(b, &sp) != nil || json.Unmarshal(sp.Body, &req) != nil ||
			time.Since(sp.At) > spoolMaxAge {
			dropped++
			continue
		}
		if d.capturePrompt(req) {
			captured++
		} else {
			dropped++
		}
	}
	_ = os.Remove(startingMarker())
	if captured+dropped > 0 {
		log.Printf("spool: %d prompt(s) kept for this daemon recorded, %d not recorded", captured, dropped)
	}
}

// capturePrompt records a prompt in the room of its session, as handlePrompt does, and answers
// nothing: the hook that sent it has gone, so nothing is offered or recorded as pending, and the
// next prompt is given what was waiting. A command is not recorded (D-209), and a session in no
// room records nothing.
func (d *Daemon) capturePrompt(req promptReq) bool {
	host, ok := hostByName(req.Host)
	if !ok {
		return false
	}
	if err := host.NormalizeHook(&req.SessionID, &req.PromptID, req.TurnID); err != nil {
		return false
	}
	req.Prompt = withoutHostReminders(req.Prompt)
	if _, isCommand := parseTypedCommand(req.Prompt, host.CommandPrefix()); isCommand {
		d.commandPrompts.note(req.SessionID, req.PromptID)
		return false
	}
	pasted := d.pastedImages(host, req)
	d.mu.Lock()
	defer d.mu.Unlock()
	room, ok := d.members.RoomForSession(req.SessionID)
	if !ok {
		return false
	}
	store, err := d.storeFor(room.RoomID)
	if err != nil {
		return false
	}
	meta := map[string]any{"cwd": req.CWD, "promptId": req.PromptID}
	if refs := storeImages(store, pasted); len(refs) > 0 {
		meta["images"] = refs
	}
	ev, err := d.appendLocal(store, room, req.SessionID, EventUserPrompt, req.Prompt, meta)
	if err != nil {
		return false
	}
	d.watchForLateImages(host, room, store, req, ev)
	d.notify()
	log.Printf("USER_PROMPT room=%s session=%.8s kept while the daemon started, nothing offered", room.RoomName, req.SessionID)
	return true
}
