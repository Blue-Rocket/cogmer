package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// The page is embedded rather than served from disk: a teammate installs one file
// (D-001), and a UI that can go missing is a UI that fails differently on each
// machine.
//
//go:embed ui.html
var uiPage []byte

// uiEvent is an event as a person reads it. Attribution anchors on the derived
// peer name, never on the self-asserted display name (D-021) -- two peers claimed
// the same display name during the first two-peer run.
type uiEvent struct {
	EventID   string   `json:"eventId"`
	PeerID    string   `json:"peerId"`
	PeerName  string   `json:"peerName"`
	Display   string   `json:"userDisplayName"`
	EventType string   `json:"eventType"`
	Content   string   `json:"content"`
	Clock     string   `json:"clock"`
	ToolCalls int      `json:"toolCalls"`
	Tools     []string `json:"tools"`
	Mine      bool     `json:"mine"`
}

type uiPeer struct {
	Name   string `json:"name"`
	Online bool   `json:"online"`
	Since  string `json:"since"`
}

type uiState struct {
	Room       string    `json:"room"`
	RoomName   string    `json:"roomName"`
	SelfPeerID string    `json:"selfPeerId"`
	Events     []uiEvent `json:"events"`
	Peers      []uiPeer  `json:"peers"`
}

// subscribe registers a channel woken whenever the room changes.
func (d *Daemon) subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	d.subsMu.Lock()
	if d.subs == nil {
		d.subs = map[chan struct{}]bool{}
	}
	d.subs[ch] = true
	d.subsMu.Unlock()
	return ch
}

func (d *Daemon) unsubscribe(ch chan struct{}) {
	d.subsMu.Lock()
	delete(d.subs, ch)
	d.subsMu.Unlock()
}

// notify wakes every reader. Non-blocking: a reader that has not drained its
// previous wake-up does not need a second one, because it re-reads the whole room.
func (d *Daemon) notify() {
	d.subsMu.Lock()
	defer d.subsMu.Unlock()
	for ch := range d.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (d *Daemon) snapshot() (uiState, error) {
	cur, ok := d.members.CurrentRoom()
	if !ok {
		// No room joined. Showing an empty feed would imply a quiet room rather
		// than no room, which are different things to a person looking at it.
		return uiState{SelfPeerID: d.id.PeerID}, nil
	}
	store, err := d.storeFor(cur.RoomID)
	if err != nil {
		return uiState{}, err
	}
	d.mu.Lock()
	evs, err := store.ListRoom(cur.RoomID)
	d.mu.Unlock()
	if err != nil {
		return uiState{}, err
	}
	st := uiState{Room: cur.RoomID, RoomName: cur.RoomName, SelfPeerID: d.id.PeerID, Peers: d.peerStatus()}
	for _, e := range evs {
		ts := e.Timestamp
		if t, err := time.Parse(time.RFC3339Nano, e.Timestamp); err == nil {
			ts = t.Local().Format("15:04:05")
		}
		ue := uiEvent{
			EventID: e.EventID, PeerID: e.PeerID, PeerName: PeerName(e.PeerID),
			Display: e.UserDisplayName, EventType: e.EventType, Content: e.Content, Clock: ts,
			// §6/D-021 require the identifier be shown for any peer whose identity
			// is unverified. That is every remote peer today. It is not required
			// for your own turns, where it identifies nothing you did not know.
			Mine: e.PeerID == d.id.PeerID,
		}
		if len(e.Metadata) > 0 {
			var m struct {
				ToolCalls int      `json:"toolCalls"`
				Tools     []string `json:"tools"`
			}
			if json.Unmarshal(e.Metadata, &m) == nil {
				ue.ToolCalls = m.ToolCalls
				ue.Tools = m.Tools
			}
		}
		st.Events = append(st.Events, ue)
	}
	return st, nil
}

func (d *Daemon) handleUI(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(uiPage)
}

// handleStream pushes the room over server-sent events.
//
// This is the one path where pushing earns its cost (§16): a person notices a
// delay a machine does not. It sends whole snapshots rather than deltas, because
// events are ordered by timestamp and a late arrival can sort before something
// already displayed -- an incremental stream would have to describe insertions,
// and a room is bounded by the pairing that created it.
func (d *Daemon) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := d.subscribe()
	defer d.unsubscribe(ch)

	send := func() bool {
		st, err := d.snapshot()
		if err != nil {
			return false
		}
		buf, err := json.Marshal(st)
		if err != nil {
			return false
		}
		fmt.Fprintf(w, "data: %s\n\n", buf)
		flusher.Flush()
		return true
	}
	if !send() {
		return
	}

	// A heartbeat keeps intermediaries from closing an idle stream, and lets the
	// browser notice a dead daemon rather than showing a frozen room as live.
	beat := time.NewTicker(20 * time.Second)
	defer beat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			if !send() {
				return
			}
		case <-beat.C:
			fmt.Fprint(w, ": beat\n\n")
			flusher.Flush()
		}
	}
}

// Peer health.
//
// A peer that cannot be reached is reported when it STOPS being reachable and when
// it starts again, not once per poll. Polling is every second, so the old behaviour
// wrote two lines a second saying the same thing -- observed filling a log through
// a partition in Phase 5. A message repeated that often is not a signal; it buries
// the one line that matters, which is the transition.

// peerState is what is known about one address.
type peerState struct {
	lastSeen  time.Time
	downSince time.Time
	lastErr   string
	reported  time.Time // when the current outage was last mentioned
}

// downRepeat is how often a continuing outage is mentioned again. Long enough that
// it is not noise, short enough that somebody reading a log an hour later learns
// the peer is still gone rather than only that it once went.
const downRepeat = 10 * time.Minute

// peerStatus reports reachability for the UI. A peer that has never answered is
// shown as offline rather than omitted: a teammate who cannot be reached is a
// fact worth seeing, not an absence.
//
// It reports every address this daemon would try, not only those named in the
// environment -- peers learned by pairing or by joining a room were invisible here
// until D-061, which is most of them.
func (d *Daemon) peerStatus() []uiPeer {
	targets := d.syncTargets()
	d.peerMu.Lock()
	defer d.peerMu.Unlock()
	var out []uiPeer
	for _, addr := range targets {
		p := uiPeer{Name: addr, Since: "never reached"}
		if st, ok := d.peerSeen[addr]; ok && !st.lastSeen.IsZero() {
			if age := time.Since(st.lastSeen); age < 15*time.Second {
				p.Online = true
			} else if age < time.Minute {
				p.Since = "just now"
			} else {
				p.Since = fmt.Sprintf("%dm ago", int(age.Minutes()))
			}
		}
		out = append(out, p)
	}
	return out
}

func (d *Daemon) peerStateLocked(addr string) *peerState {
	if d.peerSeen == nil {
		d.peerSeen = map[string]*peerState{}
	}
	st, ok := d.peerSeen[addr]
	if !ok {
		st = &peerState{}
		d.peerSeen[addr] = st
	}
	return st
}

// markPeerSeen records a successful exchange, and says so only if the peer had
// been reported missing -- coming back is worth a line; still being here is not.
func (d *Daemon) markPeerSeen(addr string) {
	d.peerMu.Lock()
	st := d.peerStateLocked(addr)
	wasDown := !st.downSince.IsZero()
	outage := time.Since(st.downSince).Round(time.Second)
	st.lastSeen = time.Now()
	st.downSince = time.Time{}
	st.lastErr = ""
	st.reported = time.Time{}
	d.peerMu.Unlock()

	if wasDown {
		log.Printf("peer %s is reachable again after %s", addr, outage)
	}
}

// markPeerUnreachable reports an outage on its transition, then at a slow repeat.
// It returns whether anything was said, so callers need no logging of their own.
func (d *Daemon) markPeerUnreachable(addr string, err error) {
	d.peerMu.Lock()
	st := d.peerStateLocked(addr)
	first := st.downSince.IsZero()
	if first {
		st.downSince = time.Now()
	}
	since := st.downSince
	due := first || time.Since(st.reported) >= downRepeat
	if due {
		st.reported = time.Now()
	}
	st.lastErr = err.Error()
	d.peerMu.Unlock()

	if !due {
		return
	}
	if first {
		log.Printf("peer %s is unreachable: %v", addr, err)
		return
	}
	log.Printf("peer %s still unreachable after %s: %v", addr, time.Since(since).Round(time.Second), err)
}

var _ = sync.Mutex{}
