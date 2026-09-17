package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultAddr = "127.0.0.1:4782" // localhost only (§25)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "daemon":
		runDaemon()
	case "hook":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		runHook(os.Args[2])
	case "seed":
		runSeed()
	case "log":
		runLog()
	case "whoami":
		runWhoami()
	case "conflicts":
		runConflicts()
	case "doctor":
		deep := len(os.Args) > 2 && os.Args[2] == "--deep"
		runDoctor(deep)
	case "behaviors":
		if len(os.Args) > 2 && os.Args[2] == "--markdown" {
			fmt.Print(MarkdownReport())
			return
		}
		for _, b := range Behaviors {
			fmt.Printf("  %s  [%s]  %s\n", b.ID, b.Tier, b.Title)
		}
	case "probe-hook":
		if len(os.Args) < 4 {
			os.Exit(0)
		}
		runProbeHook(os.Args[2], os.Args[3])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `claude-team -- Phase 0 integration spike

  claude-team daemon          Run the local daemon on `+defaultAddr+`
  claude-team hook prompt     UserPromptSubmit hook (capture + inject)
  claude-team hook stop       Stop hook (capture assistant turn)
  claude-team seed            Insert a simulated teammate conversation
  claude-team log             Print the room transcript
  claude-team whoami          Show this peer's identity and room
  claude-team conflicts       Show quarantined events (sequence conflicts)
  claude-team doctor [--deep] Verify relied-on Claude Code behaviors
  claude-team behaviors       List those behaviors (--markdown to render docs)

Environment:
  CLAUDE_TEAM_ROOM        override the active room
  CLAUDE_TEAM_ADDR        override the daemon address
  CLAUDE_TEAM_PREFLIGHT   set to "off" to skip behavior checks on new rooms
  CLAUDE_TEAM_PEERS       comma-separated peer addresses to synchronize with
  CLAUDE_TEAM_SYNC_MS     poll interval in milliseconds (default 1000)
`)
}

func addr() string {
	if a := os.Getenv("CLAUDE_TEAM_ADDR"); a != "" {
		return a
	}
	return defaultAddr
}

func syncInterval() time.Duration {
	if v := os.Getenv("CLAUDE_TEAM_SYNC_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Millisecond
		}
	}
	return time.Second
}

func openLocal() (*Store, *Identity, string) {
	id, err := LoadIdentity()
	if err != nil {
		log.Fatalf("identity: %v", err)
	}
	cfg := LoadConfig()
	store, err := OpenStore(cfg.Room)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	return store, id, cfg.Room
}

func runDaemon() {
	// Detect room formation before opening, so the first daemon for a room can
	// verify the behaviors that room's correctness depends on.
	newRoom := !roomExists(LoadConfig().Room)

	store, id, room := openLocal()
	defer store.Close()

	if newRoom {
		EnsureVerified(room)
	}

	d := &Daemon{store: store, id: id, room: room, claudeVersion: ClaudeVersion()}
	ln, err := net.Listen("tcp", addr())
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("claude-team daemon on http://%s  room=%s  peer=%s (%s)",
		addr(), room, id.UserDisplayName, id.PeerName)
	go d.RunSync(peerList(), syncInterval())
	if err := http.Serve(ln, d.Routes()); err != nil {
		log.Fatal(err)
	}
}

// runHook implements both hook entry points.
//
// Invariant from §3.1: Claude Code must keep working when the daemon is down.
// Every failure path here exits 0 with empty stdout, so a dead daemon degrades
// to "no collaboration" rather than a broken session.
func runHook(kind string) {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil || len(bytes.TrimSpace(raw)) == 0 {
		os.Exit(0)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		os.Exit(0)
	}

	var endpoint string
	switch kind {
	case "prompt":
		endpoint = "/hook/prompt"
	case "stop":
		endpoint = "/hook/stop"
	default:
		os.Exit(0)
	}

	body, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post("http://"+addr()+endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "claude-team: daemon unreachable (%v); continuing without collaboration\n", err)
		os.Exit(0)
	}
	defer resp.Body.Close()

	var out struct {
		Context string `json:"context"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		os.Exit(0)
	}
	// stdout of a UserPromptSubmit hook is appended to the pending turn's
	// context -- verified against Claude Code 2.1.273.
	if out.Context != "" {
		fmt.Println(out.Context)
	}
	os.Exit(0)
}

// runSeed inserts a simulated teammate exchange (§36 step 6) so cross-session
// injection can be exercised on one machine, before any networking exists.
func runSeed() {
	store, _, room := openLocal()
	defer store.Close()

	alice := &Identity{
		PeerID:          "peer-simulated-alice",
		UserID:          "alice",
		UserDisplayName: "Alice",
		MachineID:       "alices-thinkpad",
	}
	const aliceSession = "simulated-alice-session"
	turns := []struct{ kind, text string }{
		{EventUserPrompt, "Could idle pool expiration explain the SessionLambda timeout?"},
		{EventAssistantMessage, "Yes. The OkHttp connection pool holds idle sockets for 300s, but the upstream load balancer silently drops them at 60s. SessionLambda then reuses a dead socket and blocks until the read timeout fires."},
	}
	for _, t := range turns {
		if _, err := store.Append(alice, room, aliceSession, t.kind, t.text, nil); err != nil {
			log.Fatalf("seed: %v", err)
		}
	}
	fmt.Printf("seeded %d simulated teammate events into room %q\n", len(turns), room)
}

func runLog() {
	store, _, room := openLocal()
	defer store.Close()

	evs, err := store.ListRoom(room)
	if err != nil {
		log.Fatalf("log: %v", err)
	}
	fmt.Printf("ROOM %s -- %d events\n\n", strings.ToUpper(room), len(evs))
	for _, e := range evs {
		speaker := e.UserDisplayName
		if e.EventType == EventAssistantMessage {
			speaker = "Claude — " + e.UserDisplayName
		}
		ts := e.Timestamp
		if t, err := time.Parse(time.RFC3339Nano, e.Timestamp); err == nil {
			ts = t.Local().Format("15:04:05")
		}
		content := e.Content
		if len(content) > 400 {
			content = content[:400] + " …"
		}
		fmt.Printf("%-22s %s  [%s/%d]\n%s\n\n", speaker, ts, PeerName(e.PeerID), e.PeerSequence, content)
	}
}

// runConflicts surfaces quarantined events. A conflict means a peer's sequence
// counter went backwards -- lost local state, or a forged event. It is never
// routine, and it is invisible unless asked for.
func runConflicts() {
	store, _, room := openLocal()
	defer store.Close()

	cs, err := store.ListConflicts()
	if err != nil {
		log.Fatalf("conflicts: %v", err)
	}
	if len(cs) == 0 {
		fmt.Printf("no sequence conflicts in room %q\n", room)
		return
	}
	fmt.Printf("%d sequence conflict(s) in room %q\n\n", len(cs), room)
	for _, c := range cs {
		fmt.Printf("  %s  peer %s (%s) sequence %d\n", c.DetectedAt, PeerName(c.PeerID), c.PeerID, c.PeerSequence)
		fmt.Printf("    held:     %s\n    rejected: %s\n\n", c.HeldEventID, c.IncomingEventID)
	}
	fmt.Println("A conflict means that peer's sequence counter went backwards.")
	fmt.Println("Either it lost its local state, or an event was forged. Its events")
	fmt.Println("since then have not been stored, and anti-entropy cannot recover them.")
}

func runWhoami() {
	store, id, room := openLocal()
	defer store.Close()
	buf, _ := json.MarshalIndent(map[string]any{
		"identity": id, "peerName": id.PeerName, "room": room, "addr": addr(),
		"nameSpace": PeerNameSpace(),
	}, "", "  ")
	fmt.Println(string(buf))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
