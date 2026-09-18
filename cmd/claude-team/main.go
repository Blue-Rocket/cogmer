package main

import (
	"bytes"
	"encoding/json"
	"errors"
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
	case "pair":
		runPair(os.Args[2:])
	case "allow":
		runAllow(os.Args[2:])
	case "forget":
		runForget(os.Args[2:])
	case "verify":
		runVerify(os.Args[2:])
	case "rooms":
		runRooms()
	case "create":
		runCreateRoom()
	case "join":
		runJoin(os.Args[2:])
	case "leave":
		runLeave()
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

Peers — durable, above any room. Done once with each colleague:
  claude-team pair <string>   Record a peer AND verify it, on a call with them
  claude-team peers           List peers this machine knows
  claude-team verify <peer>   Re-run just the two-word check
  claude-team allow <id> [nm] Record a peer WITHOUT verifying (scripts, tests)
  claude-team forget <id>     Discard a peer entirely

Rooms — per room, repeated as often as you like:

  claude-team rooms           List rooms
  claude-team create          Create a room
  claude-team join <room>     Make a room current, so new sessions join it
  claude-team leave           Leave the current room
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
	id, err := LoadIdentity()
	if err != nil {
		log.Fatalf("identity: %v", err)
	}
	members, err := OpenMembership()
	if err != nil {
		log.Fatalf("membership: %v", err)
	}

	// CLAUDE_TEAM_ROOM is a convenience for choosing the current room, not a
	// statement about what this daemon serves: it serves every room this peer
	// belongs to, and a session says which one it is in (§5).
	if name := os.Getenv("CLAUDE_TEAM_ROOM"); name != "" {
		r, err := members.FindRoom(name)
		switch {
		case err == nil:
			_ = members.SetCurrentRoom(r.RoomID)
		case errors.Is(err, errNoSuchRoom):
			log.Printf("no room named %q; `claude-team rooms` lists them, `create` makes one", name)
		default:
			log.Printf("CLAUDE_TEAM_ROOM: %v", err)
		}
	}

	d := &Daemon{id: id, members: members, claudeVersion: ClaudeVersion()}
	defer d.closeStores()

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

	rooms, _ := members.Rooms()
	log.Printf("claude-team daemon  peer=%s (%s)  serving %d room(s)",
		id.UserDisplayName, id.PeerName, len(rooms))
	if cur, ok := members.CurrentRoom(); ok {
		g, _ := members.Guests(cur.RoomID)
		log.Printf("  current room  %s (%d guest(s))", cur.RoomName, len(g))
	} else {
		log.Printf("  current room  none — `claude-team create` or `claude-team join <room>`")
	}
	log.Printf("  hooks and UI  http://%s  (loopback)", addr())
	log.Printf("  peer sync     http://%s", peerAddr())
	if !isLoopback(peerAddr()) {
		log.Printf("  WARNING: the peer API is reachable from other machines. Requests are")
		log.Printf("           authenticated and events are signed, and only each room's")
		log.Printf("           guests are admitted — but anything reachable is worth knowing about.")
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
			state := "unverified"
			if p.VerifiedAt != "" {
				state = "verified " + p.VerifiedAt[:10]
			}
			fmt.Printf("  %-22s %-14s %s\n", p.Name, state, p.PeerID)
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
		fmt.Printf("recorded %s as %s — UNVERIFIED.\n", PeerName(args[0]), firstNonEmpty(name, PeerName(args[0])))
		fmt.Println("nothing yet says this key is theirs rather than someone who intercepted it.")
		fmt.Printf("finish with:  claude-team verify %s\n", firstNonEmpty(name, PeerName(args[0])))
		fmt.Println("(`claude-team pair` does both at once, and is the ordinary way.)")
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
		cur, hasCur := m.CurrentRoom()
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
			mark := " "
			if hasCur && r.RoomID == cur.RoomID {
				mark = "*"
			}
			fmt.Printf("%s %-24s %-8s %d guest(s)  %d session(s)\n",
				mark, r.RoomName, r.State, len(g), m.SessionsInRoom(r.RoomID))
		}
	})
}

func runCreateRoom() {
	withMembership(func(m *Membership, id *Identity) {
		r, err := m.CreateRoom(id.PeerID)
		if err != nil {
			log.Fatalf("create: %v", err)
		}
		if err := m.SetCurrentRoom(r.RoomID); err != nil {
			log.Fatalf("create: %v", err)
		}
		fmt.Printf("created %s and made it current\n  %s\n\n", r.RoomName, r.RoomID)
		fmt.Println("invite someone with:")
		fmt.Println("  claude-team allow <their identifier> <name>")
		fmt.Printf("  claude-team invite <name>\n")
	})
}

// currentRoom is the room a developer is working in. Commands act on it so that
// nobody has to name a room they are already inside.
func currentRoom(m *Membership) Room {
	if name := os.Getenv("CLAUDE_TEAM_ROOM"); name != "" {
		r, err := m.FindRoom(name)
		if err == nil {
			return r
		}
		if errors.Is(err, errNoSuchRoom) {
			log.Fatalf("no room named %q; `claude-team rooms` lists them", name)
		}
		log.Fatal(err)
	}
	r, ok := m.CurrentRoom()
	if !ok {
		log.Fatal("not in a room. `claude-team create` makes one, `claude-team join <room>` enters one")
	}
	return r
}

// invitation renders what a guest needs: which room, and somewhere to start
// looking for it. Both are public.
func invitation(r Room, endpoint, hostPeerID string) string {
	return fmt.Sprintf("%s/%s@%s#%s", r.RoomName, r.RoomID, endpoint, hostPeerID)
}

// parseInvitation accepts a full invitation or the name of a room already known.
func parseInvitation(arg string) (name, id, endpoint, host string, full bool) {
	hash := strings.LastIndex(arg, "#")
	if hash > 0 {
		host = arg[hash+1:]
		arg = arg[:hash]
	}
	at := strings.LastIndex(arg, "@")
	slash := strings.Index(arg, "/")
	if at < 0 || slash < 0 || slash > at {
		return arg, "", "", "", false
	}
	return arg[:slash], arg[slash+1 : at], arg[at+1:], host, true
}

func runJoin(args []string) {
	if len(args) == 0 {
		log.Fatal("usage: claude-team join <invitation>   (or a room you already know)")
	}
	withMembership(func(m *Membership, id *Identity) {
		name, roomID, endpoint, host, full := parseInvitation(args[0])
		if full {
			// A room learned from an invitation: the guest did not create it, so
			// its identity comes from the invitation rather than being invented.
			if err := m.RecordRoom(roomID, name); err != nil {
				log.Fatalf("join: %v", err)
			}
			if err := m.AddRoomPeer(roomID, endpoint); err != nil {
				log.Fatalf("join: %v", err)
			}
			// Admit whoever offered the invitation. Synchronisation is a pull in
			// both directions, so a guest that records the room and not its host
			// can read that room and never be read -- which looks like one-way
			// collaboration and is really a one-sided guest list.
			if host != "" {
				if err := m.Allow(host, ""); err != nil {
					log.Fatalf("join: %v", err)
				}
				if err := m.Invite(roomID, host); err != nil {
					log.Fatalf("join: %v", err)
				}
			}
		}
		r, err := m.FindRoom(name)
		if errors.Is(err, errNoSuchRoom) {
			log.Fatalf("no room named %q. Ask its host for an invitation; yours to give them is:\n  %s",
				name, id.PeerID)
		}
		if err != nil {
			log.Fatal(err)
		}
		if err := m.SetCurrentRoom(r.RoomID); err != nil {
			log.Fatalf("join: %v", err)
		}
		fmt.Printf("now in %s. Sessions started from here join it.\n", r.RoomName)
		if host != "" {
			fmt.Printf("admitted %s, who invited you.\n", PeerName(host))
			if !m.IsVerified(host) {
				fmt.Printf("their key is UNVERIFIED, so nothing will sync yet. On a call with them,\n")
				fmt.Printf("both run:\n  claude-team verify %s\n", PeerName(host))
			}
		}
		if peers := m.RoomPeers(r.RoomID); len(peers) > 0 {
			fmt.Printf("reaching its members at: %s\n", strings.Join(peers, ", "))
		}
		fmt.Println("Sessions already running stay where they are: a session cannot be moved once")
		fmt.Println("it has been told something, because what it was told cannot be withdrawn.")
	})
}

func runLeave() {
	withMembership(func(m *Membership, _ *Identity) {
		cur, ok := m.CurrentRoom()
		if !ok {
			fmt.Println("not in a room")
			return
		}
		if err := m.SetCurrentRoom(""); err != nil {
			log.Fatalf("leave: %v", err)
		}
		fmt.Printf("left %s. New sessions collaborate with nobody until you join a room.\n", cur.RoomName)
		fmt.Println("What you already published stays in the room; leaving does not un-say it.")
	})
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
	withMembership(func(m *Membership, self *Identity) {
		r := currentRoom(m)
		pid, err := resolvePeer(m, args[0])
		if err != nil {
			log.Fatal(err)
		}
		if err := m.Invite(r.RoomID, pid); err != nil {
			log.Fatalf("invite: %v", err)
		}
		fmt.Printf("%s may now enter %s\n", PeerName(pid), r.RoomName)
		if !m.IsVerified(pid) {
			// Admission and verification are separate gates and BOTH are required
			// (D-054). Inviting an unverified peer is allowed and does nothing on
			// its own, so say so plainly: a room that looks empty for a reason
			// nobody stated is worse than a refusal.
			fmt.Printf("\nNOTE: %s is UNVERIFIED, so no transcript will pass in either\n", PeerName(pid))
			fmt.Printf("      direction until it is. On a call with them, both run:\n")
			fmt.Printf("        claude-team verify %s\n", PeerName(pid))
		}
		fmt.Println()
		// An invitation carries a room's identity and where to reach it. It carries
		// no secret: admission is the guest list entry just made, proved later by
		// possession of their key (D-026). Interception reveals that a room exists.
		fmt.Println("give them:")
		fmt.Printf("  claude-team join %s\n", invitation(r, peerAddr(), self.PeerID))
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
	// The pairing string is what a colleague actually needs, and it is safe to
	// send by any means: an identifier is a public key and an address is where a
	// daemon listens. Neither admits anyone (D-026, D-042).
	fmt.Printf("\nyour pairing string — send it to a colleague however is convenient:\n  %s\n",
		pairingString(id.PeerID, peerAddr()))
	fmt.Println("\nthey run:  claude-team pair <that string>")
	fmt.Println("you run:   claude-team pair <theirs>")
	fmt.Println("both at once, on a call, and you each compare two words.")
	if isLoopback(peerAddr()) {
		fmt.Println("\nNOTE: that address is loopback, so nobody else can reach it. Set")
		fmt.Println("CLAUDE_TEAM_PEER_ADDR to an address they can, and restart the daemon.")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// runVerify is the two-word check (D-048). Both people run it, at the same time,
// on a call where each can recognise the other's voice.
//
// The recognition is the point and cannot be automated: the exchange proves both
// sides hold the keys they named, and only a person can say that the voice saying
// the words is the colleague rather than somebody in their place (§25).
func runVerify(args []string) {
	if len(args) == 0 {
		log.Fatal("usage: claude-team verify <peer>   (both of you, on a call, at the same time)")
	}
	var peerID, name, mine string
	withMembership(func(m *Membership, id *Identity) {
		pid, err := resolvePeer(m, args[0])
		if err != nil {
			log.Fatal(err)
		}
		peerID, name, mine = pid, PeerName(pid), id.PeerName
	})

	fmt.Printf("verifying %s. ask them to run `claude-team verify %s` now — this waits %s.\n\n",
		name, mine, verifyTimeout)
	verifyWith(peerID, name)
}

func postLocal(path string, body, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	// The exchange waits up to verifyTimeout for the other person to type the
	// command, so the client must outlast it.
	client := &http.Client{Timeout: verifyTimeout + 30*time.Second}
	resp, err := client.Post("http://"+addr()+path, "application/json", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, bytes.TrimSpace(msg))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// --- pairing: a durable act between two machines, above any room ---

// pairingString is what one person sends another, once, ever. It is an identifier
// and a bootstrap address, and it is not a secret: an identifier is a public key,
// and an address is where a daemon listens. Interception yields both and admits
// nobody (D-042, D-026).
func pairingString(peerID, endpoint string) string {
	if endpoint == "" {
		return peerID
	}
	return peerID + "@" + endpoint
}

func parsePairing(s string) (peerID, endpoint string) {
	if at := strings.LastIndex(s, "@"); at > 0 {
		return s[:at], s[at+1:]
	}
	return s, ""
}

// runPair is peer onboarding, and it is deliberately not a room operation.
//
// Pairing with someone happens ONCE between two machines and outlasts every room
// they ever share; inviting them to a room happens repeatedly and means nothing
// outside that room. §12 already keeps those as two lists -- known peers per
// machine, guests per room -- and this is the command vocabulary catching up with
// the data model.
//
// It stays a terminal command rather than becoming a slash command because it is
// interactive, because it blocks on another person, and because the two words must
// reach your eyes without passing through a model that reads room content from
// unverified peers.
func runPair(args []string) {
	if len(args) == 0 {
		log.Fatal("usage: claude-team pair <identifier>[@address] [name]\n" +
			"  both of you run it, at the same time, on a call")
	}
	peerID, endpoint := parsePairing(args[0])
	name := ""
	if len(args) > 1 {
		name = args[1]
	}

	var mine string
	withMembership(func(m *Membership, id *Identity) {
		if err := m.Allow(peerID, name); err != nil {
			log.Fatalf("pair: %v", err)
		}
		if err := m.SetPeerEndpoint(peerID, endpoint); err != nil {
			log.Fatalf("pair: %v", err)
		}
		mine = id.PeerName
	})

	fmt.Printf("recorded %s.\n", PeerName(peerID))
	if endpoint == "" {
		fmt.Println("no address was given, so there is nowhere to reach them yet. Ask for the")
		fmt.Println("whole pairing string — `claude-team whoami` prints it — and run this again.")
		return
	}
	fmt.Printf("now confirming the key is theirs. ask them to run `claude-team pair %s@…` now.\n\n",
		mine)

	verifyWith(peerID, PeerName(peerID))
}

// verifyWith drives the ceremony and records the answer. Shared by `pair` and
// `verify`: the first is a first meeting, the second is confirming a key recorded
// some other way, and the ceremony is identical.
func verifyWith(peerID, name string) {
	var out verifyStartResponse
	if err := postLocal("/verify/start", verifyStartRequest{Peer: peerID}, &out); err != nil {
		log.Fatalf("verify: %v  (is the daemon running?)", err)
	}
	if out.Error != "" {
		log.Fatalf("verify: %s", out.Error)
	}

	fmt.Printf("        %s\n\n", out.Words)
	fmt.Println("say those two words aloud. ask them to say theirs back.")
	fmt.Print("did they say the same two words? [y/N] ")

	var answer string
	fmt.Scanln(&answer)
	matched := answer == "y" || answer == "Y"

	var res map[string]string
	if err := postLocal("/verify/confirm", verifyConfirmRequest{Peer: peerID, Matched: matched}, &res); err != nil {
		log.Fatalf("verify: %v", err)
	}
	if matched {
		fmt.Printf("\npaired with %s. this holds for every room you ever share,\n", name)
		fmt.Println("and you will not be asked again. if this key changes, that is an alarm")
		fmt.Println("rather than a new first meeting.")
		return
	}

	fmt.Println()
	fmt.Println("STOP. different words mean you are not connected to each other:")
	fmt.Println("something is relaying this exchange and showing each of you a different key.")
	fmt.Printf("\n  the key you were given for %s:\n    %s\n\n", name, Fingerprint(peerID))
	fmt.Println("this is not a transient error and running it again will not clear it.")
	fmt.Println("tell the person on the call what you saw — it is evidence, and it is the")
	fmt.Println("only place this becomes visible.")
	os.Exit(1)
}
