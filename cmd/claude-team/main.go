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

const (
	// defaultAddr carries hooks and the local UI. Loopback, always (§25).
	defaultAddr = "127.0.0.1:4782"
	// defaultPeerAddr carries synchronization. Loopback by default so that
	// nothing is reachable until someone decides it should be.
	defaultPeerAddr = "127.0.0.1:4783"
)

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
	case "peers":
		runPeers()
	case "allow":
		runAllow(os.Args[2:])
	case "forget":
		runForget(os.Args[2:])
	case "rooms":
		runRooms()
	case "create":
		runCreateRoom()
	case "guests":
		runGuests()
	case "invite":
		runInvite(os.Args[2:])
	case "revoke":
		runRevoke(os.Args[2:])
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

  claude-team peers           List peers this machine knows
  claude-team allow <id> [nm] Record a peer, so it can be admitted to a room
  claude-team forget <id>     Discard a peer entirely

  claude-team rooms           List rooms
  claude-team create          Create a room
  claude-team guests          List who may enter the current room
  claude-team invite <peer>   Admit a known peer to the current room
  claude-team revoke <peer>   Withdraw admission
  claude-team doctor [--deep] Verify relied-on Claude Code behaviors
  claude-team behaviors       List those behaviors (--markdown to render docs)

Environment:
  CLAUDE_TEAM_ROOM        override the active room
  CLAUDE_TEAM_ADDR        hooks and UI address (loopback only, default 127.0.0.1:4782)
  CLAUDE_TEAM_PEER_ADDR   peer sync address (default 127.0.0.1:4783)
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

func peerAddr() string {
	if a := os.Getenv("CLAUDE_TEAM_PEER_ADDR"); a != "" {
		return a
	}
	return defaultPeerAddr
}

// isLoopback reports whether an address is unreachable from another machine.
func isLoopback(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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


	// Rooms are records, not arbitrary strings: a name must resolve to something
	// with an identity and a guest list, or admission has nothing to consult.
	members, err := OpenMembership()
	if err != nil {
		log.Fatalf("membership: %v", err)
	}
	rec, ok := members.FindRoom(room)
	if !ok {
		// A daemon pointed at a room nobody created makes one and admits its
		// creator, so an existing setup keeps working. §12 has invitation as the
		// deliberate act; this is the bridge until a session joins a room rather
		// than a daemon serving one.
		rec, err = members.CreateRoom(id.PeerID)
		if err != nil {
			log.Fatalf("create room: %v", err)
		}
		log.Printf("no room named %q existed; created %s", room, rec.RoomName)
	}

	d := &Daemon{store: store, id: id, room: room, roomID: rec.RoomID,
		members: members, claudeVersion: ClaudeVersion()}

	if !isLoopback(addr()) {
		log.Fatalf("refusing to serve hooks on %s: the hook API publishes into the room "+
			"and reads the conversation back, so it must stay on loopback (§25)", addr())
	}
	local, err := net.Listen("tcp", addr())
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	peer, err := net.Listen("tcp", peerAddr())
	if err != nil {
		log.Fatalf("listen (peer): %v", err)
	}

	guests, _ := members.Guests(rec.RoomID)
	log.Printf("claude-team daemon  room=%s (%d guest(s))  peer=%s (%s)",
		rec.RoomName, len(guests), id.UserDisplayName, id.PeerName)
	log.Printf("  hooks and UI  http://%s  (loopback)", addr())
	log.Printf("  peer sync     http://%s", peerAddr())
	if !isLoopback(peerAddr()) {
		log.Printf("  WARNING: the peer API is reachable from other machines.")
		log.Printf("           Requests are authenticated, so you will know which peer asked,")
		log.Printf("           and events are signed, so nothing can be forged. But nothing yet")
		log.Printf("           Only this room's %d guest(s) are admitted; others are refused.", len(guests))
	}

	// Behaviour checks spend a Claude turn and take a few seconds. Run them
	// alongside the daemon rather than ahead of it: D-009 says a failed check
	// never blocks the room, and a check that delays the room from answering at
	// all is the same fault in a smaller form.
	if newRoom {
		go EnsureVerified(room)
	}

	go d.RunSync(peerList(), syncInterval())
	go func() {
		if err := http.Serve(peer, d.PeerRoutes()); err != nil {
			log.Fatalf("peer server: %v", err)
		}
	}()
	if err := http.Serve(local, d.LocalRoutes()); err != nil {
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
// resolvePeer accepts an identifier or a name this machine already knows, so a
// person need not paste a key to refer to a colleague they have met.
func resolvePeer(m *Membership, arg string) (string, error) {
	if _, err := PublicFromPeerID(arg); err == nil {
		return arg, nil
	}
	known, err := m.KnownPeers()
	if err != nil {
		return "", err
	}
	for _, p := range known {
		if p.Name == arg || PeerName(p.PeerID) == arg {
			return p.PeerID, nil
		}
	}
	return "", fmt.Errorf("no peer known as %q; pass its identifier, or run `claude-team peers`", arg)
}

func withMembership(fn func(*Membership, *Identity)) {
	id, err := LoadIdentity()
	if err != nil {
		log.Fatalf("identity: %v", err)
	}
	m, err := OpenMembership()
	if err != nil {
		log.Fatalf("membership: %v", err)
	}
	defer m.Close()
	fn(m, id)
}

func runPeers() {
	withMembership(func(m *Membership, id *Identity) {
		known, err := m.KnownPeers()
		if err != nil {
			log.Fatalf("peers: %v", err)
		}
		if len(known) == 0 {
			fmt.Println("no known peers. `claude-team allow <identifier>` records one.")
			fmt.Printf("\nyours, to give to a colleague — safe to send anywhere:\n  %s\n", id.PeerID)
			return
		}
		for _, p := range known {
			fmt.Printf("  %-22s %s\n", p.Name, p.PeerID)
		}
	})
}

func runAllow(args []string) {
	if len(args) == 0 {
		log.Fatal("usage: claude-team allow <identifier> [name]")
	}
	name := ""
	if len(args) > 1 {
		name = args[1]
	}
	withMembership(func(m *Membership, _ *Identity) {
		if err := m.Allow(args[0], name); err != nil {
			log.Fatalf("allow: %v", err)
		}
		fmt.Printf("recorded %s as %s\n", PeerName(args[0]), firstNonEmpty(name, PeerName(args[0])))
		fmt.Println("verify the whole identifier with them over a channel this did not travel on:")
		fmt.Printf("  %s\n", Fingerprint(args[0]))
	})
}

func runForget(args []string) {
	if len(args) == 0 {
		log.Fatal("usage: claude-team forget <peer>")
	}
	withMembership(func(m *Membership, _ *Identity) {
		pid, err := resolvePeer(m, args[0])
		if err != nil {
			log.Fatal(err)
		}
		if err := m.Forget(pid); err != nil {
			log.Fatalf("forget: %v", err)
		}
		fmt.Printf("forgot %s. A later meeting will be a first meeting.\n", PeerName(pid))
	})
}

func runRooms() {
	withMembership(func(m *Membership, _ *Identity) {
		rooms, err := m.Rooms()
		if err != nil {
			log.Fatalf("rooms: %v", err)
		}
		if len(rooms) == 0 {
			fmt.Println("no rooms. `claude-team create` makes one.")
			return
		}
		for _, r := range rooms {
			g, _ := m.Guests(r.RoomID)
			fmt.Printf("  %-24s %-8s %d guest(s)  %s\n", r.RoomName, r.State, len(g), r.RoomID)
		}
	})
}

func runCreateRoom() {
	withMembership(func(m *Membership, id *Identity) {
		r, err := m.CreateRoom(id.PeerID)
		if err != nil {
			log.Fatalf("create: %v", err)
		}
		fmt.Printf("created %s\n  %s\n\n", r.RoomName, r.RoomID)
		fmt.Printf("run the daemon on it with:\n  CLAUDE_TEAM_ROOM=%s claude-team daemon\n", r.RoomName)
	})
}

func currentRoom(m *Membership) Room {
	r, ok := m.FindRoom(LoadConfig().Room)
	if !ok {
		log.Fatalf("no room named %q; `claude-team rooms` lists them", LoadConfig().Room)
	}
	return r
}

func runGuests() {
	withMembership(func(m *Membership, id *Identity) {
		r := currentRoom(m)
		g, err := m.Guests(r.RoomID)
		if err != nil {
			log.Fatalf("guests: %v", err)
		}
		fmt.Printf("%s\n", r.RoomName)
		for _, p := range g {
			mine := ""
			if p == id.PeerID {
				mine = "  (you)"
			}
			fmt.Printf("  %-22s %s%s\n", PeerName(p), p[:32]+"…", mine)
		}
	})
}

func runInvite(args []string) {
	if len(args) == 0 {
		log.Fatal("usage: claude-team invite <peer>")
	}
	withMembership(func(m *Membership, _ *Identity) {
		r := currentRoom(m)
		pid, err := resolvePeer(m, args[0])
		if err != nil {
			log.Fatal(err)
		}
		if err := m.Invite(r.RoomID, pid); err != nil {
			log.Fatalf("invite: %v", err)
		}
		fmt.Printf("%s may now enter %s\n", PeerName(pid), r.RoomName)
		fmt.Printf("tell them:\n  CLAUDE_TEAM_ROOM=%s claude-team daemon\n", r.RoomName)
	})
}

func runRevoke(args []string) {
	if len(args) == 0 {
		log.Fatal("usage: claude-team revoke <peer>")
	}
	withMembership(func(m *Membership, _ *Identity) {
		r := currentRoom(m)
		pid, err := resolvePeer(m, args[0])
		if err != nil {
			log.Fatal(err)
		}
		if err := m.Revoke(r.RoomID, pid); err != nil {
			log.Fatalf("revoke: %v", err)
		}
		fmt.Printf("%s may no longer enter %s. What they already hold remains theirs.\n",
			PeerName(pid), r.RoomName)
	})
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

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
	}, "", "  ")
	fmt.Println(string(buf))
	// The whole identifier, grouped for reading aloud. §25 requires a comparison
	// be made against all of a key: a short mnemonic catches an accident, not an
	// adversary.
	fmt.Printf("\nfingerprint (compare in full, over a channel the invitation did not travel on):\n  %s\n",
		Fingerprint(id.PeerID))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
