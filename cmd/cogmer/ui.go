package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strings"
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
	EventID  string `json:"eventId"`
	PeerID   string `json:"peerId"`
	PeerName string `json:"peerName"`
	Display  string `json:"userDisplayName"`
	// Label is what THIS person calls the peer, chosen when they paired. Empty
	// for your own turns and for a peer recorded by a script (D-094).
	Label string `json:"label"`
	// Verified is a fact, not a constant. The view marked every remote turn
	// unverified because it had no way to ask -- a comment written before D-054,
	// which now refuses an unverified peer's events outright. So the marker sat on
	// every turn that could exist, and a warning that is always on is not a
	// warning. Seeing one now means a filter failed, which is worth shouting about.
	Verified  bool   `json:"verified"`
	EventType string `json:"eventType"`
	Content   string `json:"content"`
	// HTML is Content rendered as Markdown by the daemon that shows the room, never by the
	// page, so that what a colleague's text can do in the reader's browser is decided
	// where a test reads it (D-211). It is made for the view's stream only: it is not
	// stored, not sent to a peer, and not in what is injected into a session.
	HTML      string   `json:"html"`
	Clock     string   `json:"clock"`
	ToolCalls int      `json:"toolCalls"`
	Tools     []string `json:"tools"`
	Mine      bool     `json:"mine"`
	// At is the local date and time the turn was written, which Clock leaves out.
	At string `json:"at"`
	// NotInSession is true for a colleague's turn that no session on this machine
	// has been confirmed to hold, so the page can say that a turn it shows is one
	// the model has not been given. It is never set in a show-only room, which
	// gives nothing to a session (D-200).
	NotInSession bool `json:"notInSession,omitempty"`
}

type uiPeer struct {
	// Name is the person when the address belongs to a known peer, and the address
	// otherwise, because a colleague is a name and an address is not.
	Name   string `json:"name"`
	PeerID string `json:"peerId,omitempty"`
	Online bool   `json:"online"`
	Since  string `json:"since"`
	// Trying is how long this daemon has failed to reach the address since it last
	// could, or since it started trying, and Error is the last reason. Empty while
	// the peer answers.
	Trying string `json:"trying,omitempty"`
	Error  string `json:"error,omitempty"`
	// HeardFrom is how long ago this peer last made an authenticated request of this
	// daemon, empty if it has not since the daemon started. A peer this machine cannot
	// reach, and that reached it a minute ago, has a daemon that is running.
	HeardFrom string `json:"heardFrom,omitempty"`
	// LastTurn is when this peer last wrote a turn in this room, empty if it has
	// not. An online peer whose turns stop arriving shows here, where the dot alone
	// says it is reachable.
	LastTurn string `json:"lastTurn,omitempty"`
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

// snapshot renders one named room.
//
// The room is a parameter rather than something looked up, because the view used
// to follow the machine-level pointer and so showed whichever room was created
// last — right by luck with one room, silently wrong with two. A window whose
// contents can change identity underneath a reader is worse than one that shows
// nothing (D-077).
func (d *Daemon) snapshot(cur Room) (uiState, error) {
	if cur.RoomID == "" {
		// No room. Showing an empty feed would imply a quiet room rather than no
		// room, which are different things to a person looking at it.
		return uiState{SelfPeerID: d.id.PeerID}, nil
	}
	store, err := d.storeFor(cur.RoomID)
	if err != nil {
		return uiState{}, err
	}
	d.mu.Lock()
	evs, err := store.ListRoom(cur.RoomID)
	var delivered map[string]bool
	if err == nil && !cur.ShowOnly {
		delivered, err = store.DeliveredEventIDs()
	}
	d.mu.Unlock()
	if err != nil {
		return uiState{}, err
	}
	now := time.Now()
	lastTurn := map[string]string{}
	st := uiState{Room: cur.RoomID, RoomName: cur.RoomName, SelfPeerID: d.id.PeerID, Peers: d.peerStatus()}
	labels := d.members.Labels()
	for _, e := range evs {
		ts, at := e.Timestamp, e.Timestamp
		if t, err := time.Parse(time.RFC3339Nano, e.Timestamp); err == nil {
			ts = t.Local().Format("15:04:05")
			at = t.Local().Format("2006-01-02 15:04:05")
			lastTurn[e.PeerID] = turnTime(t, now)
		}
		ue := uiEvent{
			EventID: e.EventID, PeerID: e.PeerID, PeerName: PeerName(e.PeerID),
			Display: e.UserDisplayName, Label: labels[e.PeerID],
			Verified:  e.PeerID == d.id.PeerID || d.members.IsVerified(e.PeerID),
			EventType: e.EventType, Content: e.Content, Clock: ts,
			// §6 and D-139 require the identifier be shown for any peer whose identity
			// is unverified. That is every remote peer today. It is not required
			// for your own turns, where it identifies nothing you did not know.
			Mine: e.PeerID == d.id.PeerID, At: at,
		}
		ue.NotInSession = !ue.Mine && !cur.ShowOnly && !delivered[e.EventID]
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
	for i := range st.Peers {
		st.Peers[i].LastTurn = lastTurn[st.Peers[i].PeerID]
	}
	return st, nil
}

// viewSnapshot is the room as the view's stream sends it: the snapshot with each turn's
// Markdown rendered. The export and every other reader of a snapshot take the text as
// written (D-211).
func (d *Daemon) viewSnapshot(cur Room) (uiState, error) {
	st, err := d.snapshot(cur)
	for i := range st.Events {
		st.Events[i].HTML = d.rendered.get(st.Events[i].EventID, st.Events[i].Content)
	}
	return st, err
}

// turnTime is a turn's time for a reader: the clock for today, and the date with it
// for any other day. It is absolute, since a relative age would be wrong by the time
// the page next redrew.
func turnTime(t, now time.Time) string {
	t, now = t.Local(), now.Local()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04")
	}
	return t.Format("Jan 2 15:04")
}

// roomFromPath resolves /room/<name> to a room, and "/" to nothing.
//
// A room's own URL is what makes the address handed over at `create` a stable
// link: two rooms are two tabs, each showing what it says it shows.
func (d *Daemon) roomFromPath(path string) (Room, bool) {
	name := strings.TrimPrefix(path, "/room/")
	if name == path || name == "" {
		return Room{}, false
	}
	r, err := d.members.FindRoom(name)
	if err != nil {
		return Room{}, false
	}
	return r, true
}

// roomPagePolicy lets the room page run its own script and style, and read this daemon's
// stream, and nothing else. Its images are none, so an address a colleague put in a turn
// cannot be fetched even if the renderer let one through (D-212).
const roomPagePolicy = "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; " +
	"connect-src 'self'; img-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

func (d *Daemon) handleUI(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/":
		// The index lists rooms rather than picking one. Picking is what the
		// pointer did.
		rooms, _ := d.members.Rooms()
		if len(rooms) == 1 {
			http.Redirect(w, r, "/room/"+rooms[0].RoomName, http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(roomIndex(rooms))
		return
	case strings.HasPrefix(r.URL.Path, "/room/"):
		if _, ok := d.roomFromPath(r.URL.Path); !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", roomPagePolicy)
		w.Header().Set("Referrer-Policy", "no-referrer")
		_, _ = w.Write(uiPage)
		return
	}
	http.NotFound(w, r)
}

// roomIndex is the page shown when the daemon serves more than one room. It is
// deliberately plain: it exists so that no room is chosen on a reader's behalf.
func roomIndex(rooms []Room) []byte {
	var b strings.Builder
	b.WriteString("<!doctype html><meta charset=utf-8><title>rooms</title>")
	b.WriteString("<style>body{font:16px/1.6 ui-monospace,SFMono-Regular,Menlo,monospace;max-width:40rem;margin:4rem auto;padding:0 1.5rem}")
	b.WriteString("a{display:block;padding:.6rem 0;border-bottom:1px solid #8883}h1{font-size:1rem;letter-spacing:.1em;text-transform:uppercase;opacity:.6}</style>")
	b.WriteString("<h1>rooms on this machine</h1>")
	if len(rooms) == 0 {
		b.WriteString("<p>None yet. Create a room in your session: <code>/cogmer:room-create</code> in Claude Code or <code>$cogmer room-create</code> in Codex.</p>")
	}
	for _, r := range rooms {
		fmt.Fprintf(&b, "<a href=\"/room/%s\">%s</a>", template.HTMLEscapeString(r.RoomName), template.HTMLEscapeString(r.RoomName))
	}
	return []byte(b.String())
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

	// The room comes from the request, so two tabs stream two rooms and neither
	// changes identity when somebody creates a third.
	room, ok := d.roomFromPath("/room/" + r.URL.Query().Get("room"))
	if !ok {
		http.Error(w, "no such room", http.StatusNotFound)
		return
	}

	ch := d.subscribe()
	defer d.unsubscribe(ch)

	send := func() bool {
		st, err := d.viewSnapshot(room)
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

// handleExport serves the room as a Markdown file the reader keeps. It is a read of
// what /stream already sends, so it is not guarded (D-087), and it is a download so
// that nothing in it is rendered by the browser.
func (d *Daemon) handleExport(w http.ResponseWriter, r *http.Request) {
	room, ok := d.roomFromPath("/room/" + r.URL.Query().Get("room"))
	if !ok {
		http.Error(w, "no such room", http.StatusNotFound)
		return
	}
	st, err := d.snapshot(room)
	if err != nil {
		http.Error(w, "could not read the room", http.StatusInternalServerError)
		return
	}
	now := time.Now()
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-%s.md"`,
		exportFileStem(room.RoomName), now.Format("2006-01-02")))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write([]byte(exportMarkdown(st, now)))
}

// exportFileStem keeps a room name to characters a file name and a header can hold.
func exportFileStem(name string) string {
	var b strings.Builder
	for _, c := range name {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_', c == '.':
			b.WriteRune(c)
		default:
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return "room"
	}
	return b.String()
}

// exportMarkdown writes a room as a document. Each turn is a heading naming who wrote
// it and when, so the file reads without the page. A turn from a peer that is not
// verified says so, as the page does, and a tool operation is counted and named,
// since a turn that only ran tools would otherwise be an empty heading.
func exportMarkdown(st uiState, at time.Time) string {
	var b strings.Builder
	title := st.RoomName
	if title == "" {
		title = st.Room
	}
	fmt.Fprintf(&b, "# %s\n\nExported %s from cogmer. %d turns.\n", title, at.Local().Format("2006-01-02 15:04"), len(st.Events))
	for _, e := range st.Events {
		who := firstNonEmpty(e.Label, e.Display)
		if e.EventType == "ASSISTANT_MESSAGE" {
			who = "Assistant, " + who
		}
		fmt.Fprintf(&b, "\n## %s, %s", who, e.At)
		if !e.Mine {
			fmt.Fprintf(&b, " (%s", e.PeerName)
			if !e.Verified {
				b.WriteString(", unverified")
			}
			b.WriteString(")")
		}
		fmt.Fprintf(&b, "\n\n%s\n", strings.TrimRight(e.Content, "\n"))
		if e.ToolCalls > 0 {
			counted := map[string]int{}
			var order []string
			for _, n := range e.Tools {
				if counted[n] == 0 {
					order = append(order, n)
				}
				counted[n]++
			}
			var parts []string
			for _, n := range order {
				if counted[n] > 1 {
					n = fmt.Sprintf("%s ×%d", n, counted[n])
				}
				parts = append(parts, n)
			}
			ops := "operation"
			if e.ToolCalls > 1 {
				ops = "operations"
			}
			fmt.Fprintf(&b, "\n*%d tool %s", e.ToolCalls, ops)
			if len(parts) > 0 {
				fmt.Fprintf(&b, ": %s", strings.Join(parts, ", "))
			}
			b.WriteString("*\n")
		}
	}
	return b.String()
}

// Peer health.
//
// A peer that cannot be reached is reported when it STOPS being reachable and when
// it starts again, not once per poll. Polling is every second, so the old behavior
// wrote two lines a second saying the same thing, which fills a log during a
// partition. A message repeated that often is not a signal; it buries
// the one line that matters, which is the transition.

// peerState is what is known about one address.
type peerState struct {
	lastSeen  time.Time
	downSince time.Time
	lastErr   string
	reported  time.Time // when the current outage was last mentioned
	rebuilt   time.Time // when the connection held to this peer was last discarded
}

const (
	// rebuildAfter is how long a peer must have failed before the connection held to it is
	// discarded, and rebuildEvery the least time between two such discards. A connection is
	// built once and kept, so one that got into a state it cannot leave would fail for as long
	// as the daemon ran, which one user's did for 22 hours.
	rebuildAfter = 3 * time.Minute
	rebuildEvery = 5 * time.Minute
	// diagnoseWithin bounds one diagnosis.
	diagnoseWithin = 15 * time.Second
)

// downRepeat is how often a continuing outage is mentioned again. Long enough that
// it is not noise, short enough that somebody reading a log an hour later learns
// the peer is still gone rather than only that it once went.
const downRepeat = 10 * time.Minute

// reachUnknown is what a peer's "since" says when this daemon has no record of having
// reached it. The record lives in memory and is empty after a restart, so the absence
// says only that the answer is not known yet, which is not the same as the colleague
// never having been reached: the room may hold a long exchange with them. A reader is
// told that it is being checked, and nothing about how the daemon came to know less.
const reachUnknown = "checking"

// peerStatus reports reachability for the UI. A peer that has never answered is
// shown as offline rather than omitted: a teammate who cannot be reached is a
// fact worth seeing, not an absence.
//
// It reports every address this daemon would try, not only those named in the
// environment, because peers learned by pairing or by joining a room are most of
// them.
func (d *Daemon) peerStatus() []uiPeer {
	targets := d.syncTargets()
	d.peerMu.Lock()
	defer d.peerMu.Unlock()
	owners := d.members.EndpointOwners()
	var out []uiPeer
	for _, addr := range targets {
		p := uiPeer{Name: shortEndpoint(addr), Since: reachUnknown}
		if id, ok := owners[addr]; ok {
			p.PeerID = id
			p.Name = firstNonEmpty(d.members.Label(id), PeerName(id))
			if at, ok := d.heard[id]; ok {
				p.HeardFrom = roughDuration(time.Since(at))
			}
		}
		if st, ok := d.peerSeen[addr]; ok {
			if !st.lastSeen.IsZero() {
				if age := time.Since(st.lastSeen); age < 15*time.Second {
					p.Online = true
				} else if age < time.Minute {
					p.Since = "just now"
				} else {
					p.Since = fmt.Sprintf("%dm ago", int(age.Minutes()))
				}
			}
			// A peer that answered in the last 15 seconds is still online, so one
			// failed poll does not read as an outage.
			if !p.Online && !st.downSince.IsZero() {
				p.Trying = roughDuration(time.Since(st.downSince))
				p.Error = st.lastErr
			}
		}
		out = append(out, p)
	}
	return out
}

// roughDuration says how long in the units a person reads: seconds under a minute,
// then minutes, then hours.
func roughDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

// markPeerHeardFrom records that a peer has just made a request that proved it holds
// its key. Only an authenticated request counts, since an unauthenticated one names
// nobody.
func (d *Daemon) markPeerHeardFrom(peerID string) {
	d.peerMu.Lock()
	defer d.peerMu.Unlock()
	if d.heard == nil {
		d.heard = map[string]time.Time{}
	}
	d.heard[peerID] = time.Now()
}

// handleSelf answers whether this daemon can be reached through its relay, for the
// commands that tell a person whether the address they hand out works. A check opens
// a connection through the relay, so a pass is kept for 20 seconds. A failure is not
// kept, because the next ask is usually after something was fixed.
func (d *Daemon) handleSelf(w http.ResponseWriter, r *http.Request) {
	if d.selfCheck == nil {
		writeJSON(w, SelfCheck{})
		return
	}
	d.selfMu.Lock()
	defer d.selfMu.Unlock()
	if !d.selfLast.Reached || time.Since(d.selfAt) > 20*time.Second {
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		d.selfLast, d.selfAt = d.selfCheck(ctx), time.Now()
		d.noteSelfResult(d.selfLast)
	}
	out := d.selfLast
	out.FailingFor = d.failingFor()
	writeJSON(w, out)
}

// handlePeers answers with the state of every peer this daemon tries, for the
// commands that tell a person whom they can reach. A read like the view's, so it
// needs no header, and the daemon sends no CORS headers (D-087).
func (d *Daemon) handlePeers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"peers": d.peerStatus(), "self": d.selfStatus(), "hostCalls": d.hostCalls.ages()})
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
		log.Printf("peer %s is reachable again after %s", shortEndpoint(addr), outage)
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
	rebuild := !first && time.Since(since) >= rebuildAfter && time.Since(st.rebuilt) >= rebuildEvery
	if rebuild {
		st.rebuilt = time.Now()
	}
	diagnose := due && !d.diagnosing[addr]
	if diagnose {
		if d.diagnosing == nil {
			d.diagnosing = map[string]bool{}
		}
		d.diagnosing[addr] = true
	}
	st.lastErr = err.Error()
	d.peerMu.Unlock()

	if rebuild {
		d.discardConnection(addr, time.Since(since))
	}
	if diagnose {
		d.diagnose(addr)
	}
	if !due {
		return
	}
	if first {
		log.Printf("peer %s is unreachable: %v", shortEndpoint(addr), err)
		return
	}
	log.Printf("peer %s still unreachable after %s: %v", shortEndpoint(addr), time.Since(since).Round(time.Second), err)
}

// discardConnection throws away the connection held to a peer that has not answered for a
// while, so that the next poll builds a new one.
func (d *Daemon) discardConnection(addr string, failing time.Duration) {
	e, err := ParseEndpoint(addr)
	if err != nil {
		return
	}
	if r, ok := dialers[e.Scheme].(Resetter); ok && r.Reset(e) {
		log.Printf("peer %s: discarded the connection held to it after %s of failures, and the next poll builds a new one", shortEndpoint(addr), failing.Round(time.Second))
	}
}

// diagnose says in the log where a connection to a peer fails, when the transport can. It runs
// in the background, since a diagnosis waits on the network, and at most one runs for an
// address.
func (d *Daemon) diagnose(addr string) {
	e, err := ParseEndpoint(addr)
	dg, ok := Diagnoser(nil), false
	if err == nil {
		dg, ok = dialers[e.Scheme].(Diagnoser)
	}
	if !ok {
		d.peerMu.Lock()
		delete(d.diagnosing, addr)
		d.peerMu.Unlock()
		return
	}
	d.inBackground(func() {
		ctx, cancel := context.WithTimeout(context.Background(), diagnoseWithin)
		defer cancel()
		text := dg.Diagnose(ctx, e)
		d.peerMu.Lock()
		delete(d.diagnosing, addr)
		d.peerMu.Unlock()
		log.Printf("peer %s: why it cannot be reached: %s", shortEndpoint(addr), text)
	})
}

var _ = sync.Mutex{}

// Reachability of this daemon through its own relay.
//
// A daemon whose overlay stopped being reachable looked the same as a healthy one
// for as long as nobody tried to reach it, which in one case was sixteen minutes
// and could have been weeks. So the daemon asks itself on a timer, says so when the
// answer changes, and keeps the answer where the commands can show it (D-197). It
// only reports. Restarting the overlay is a person's call, with /cogmer:self-restart.

// selfWatch is what the watcher knows, and how often it asks. The intervals are
// fields so a test need not wait for the real ones.
type selfWatch struct {
	mu         sync.Mutex
	failSince  time.Time
	failErr    string
	reported   time.Time
	lastRelay  string
	detached   time.Time // when the cheap check first found no relay
	every      time.Duration
	cheapEvery time.Duration
	confirm    time.Duration
}

const (
	selfWatchEvery   = 12 * time.Minute
	selfWatchCheap   = time.Minute
	selfWatchConfirm = 30 * time.Second
)

// RunSelfWatch asks, for as long as the daemon runs, whether it can be reached. A
// cheap read of the attached relay runs every minute, and a connection through the
// relay every twelve, with a random offset so machines do not ask in step. A failure
// is asked again half a minute later before it counts, since a laptop that has just
// woken fails once and is fine.
func (d *Daemon) RunSelfWatch(stop <-chan struct{}) {
	if d.selfCheck == nil {
		return
	}
	w := &d.watch
	w.mu.Lock()
	if w.every == 0 {
		w.every, w.cheapEvery, w.confirm = selfWatchEvery, selfWatchCheap, selfWatchConfirm
	}
	every, cheap, confirm := w.every, w.cheapEvery, w.confirm
	w.mu.Unlock()

	// Soon after start, but not at once: the server needs a few seconds to attach.
	next := time.Now().Add(jitter(every / 12))
	tick := time.NewTicker(cheap)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
		}
		d.watchRelay()
		if time.Now().Before(next) {
			continue
		}
		next = time.Now().Add(every + jitter(every/5))
		d.watchDial(stop, confirm)
	}
}

// jitter is a random duration up to max.
func jitter(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(max)))
}

// watchRelay reads the attached relay and says when there has been none for
// several checks running, which is a state a person cannot see from anywhere.
func (d *Daemon) watchRelay() {
	if d.relayNow == nil {
		return
	}
	relay := d.relayNow()
	w := &d.watch
	w.mu.Lock()
	defer w.mu.Unlock()
	if relay != "" {
		if !w.detached.IsZero() {
			log.Printf("relay: attached to the relay again (%s) after %s", relay, roughDuration(time.Since(w.detached)))
		}
		w.detached, w.lastRelay = time.Time{}, relay
		return
	}
	if w.detached.IsZero() {
		w.detached = time.Now()
		return
	}
	// Said once the absence has lasted three checks, and then at the slow repeat.
	if time.Since(w.detached) >= 3*w.cheapEvery && time.Since(w.reported) >= downRepeat {
		w.reported = time.Now()
		log.Printf("relay: this daemon has not been attached to a relay for %s, so colleagues cannot reach it. /cogmer:self-restart restarts it.", roughDuration(time.Since(w.detached)))
	}
}

// watchDial runs the full check, and again after a pause when it fails, and records
// the outcome.
func (d *Daemon) watchDial(stop <-chan struct{}, confirm time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	sc := d.selfCheck(ctx)
	cancel()
	if !sc.Reached {
		select {
		case <-stop:
			return
		case <-time.After(confirm):
		}
		ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
		sc = d.selfCheck(ctx)
		cancel()
	}
	d.noteSelfResult(sc)
}

// noteSelfResult records an answer, from the watcher or from a command, and says
// so on the transitions: when the daemon stops being reachable, then at a slow
// repeat, and when it is reachable again.
func (d *Daemon) noteSelfResult(sc SelfCheck) {
	w := &d.watch
	w.mu.Lock()
	defer w.mu.Unlock()
	if sc.Reached {
		if !w.failSince.IsZero() {
			log.Printf("relay: reachable again after %s", roughDuration(time.Since(w.failSince)))
		}
		w.failSince, w.failErr, w.reported = time.Time{}, "", time.Time{}
		return
	}
	first := w.failSince.IsZero()
	if first {
		w.failSince = time.Now()
	}
	w.failErr = sc.Error
	if first || time.Since(w.reported) >= downRepeat {
		w.reported = time.Now()
		log.Printf("relay: this daemon cannot be reached through its relay (%s), so colleagues cannot reach it. It has failed this check for %s. /cogmer:self-restart restarts it.",
			shortReason(sc.Error), roughDuration(time.Since(w.failSince)))
	}
}

// failingFor is how long the daemon has been failing the reachability check, empty
// while it passes.
func (d *Daemon) failingFor() string {
	d.watch.mu.Lock()
	defer d.watch.mu.Unlock()
	if d.watch.failSince.IsZero() {
		return ""
	}
	return roughDuration(time.Since(d.watch.failSince))
}

// uiSelf is what the commands show about this daemon's own reachability. It is
// absent when there is no overlay, since there is then nothing to be unreachable.
type uiSelf struct {
	Failing    bool   `json:"failing"`
	FailingFor string `json:"failingFor,omitempty"`
	Error      string `json:"error,omitempty"`
	Detached   string `json:"detached,omitempty"`
}

// selfStatus is the watcher's last word, read without opening a connection.
func (d *Daemon) selfStatus() *uiSelf {
	if d.selfCheck == nil {
		return nil
	}
	w := &d.watch
	w.mu.Lock()
	defer w.mu.Unlock()
	s := &uiSelf{}
	if !w.failSince.IsZero() {
		s.Failing, s.FailingFor, s.Error = true, roughDuration(time.Since(w.failSince)), w.failErr
	}
	if !w.detached.IsZero() && time.Since(w.detached) >= 3*w.cheapEvery {
		s.Failing, s.Detached = true, roughDuration(time.Since(w.detached))
	}
	return s
}
