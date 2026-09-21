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
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// version is stamped at build time with -ldflags "-X main.version=...". It is how
// the plugin decides whether the binary on disk is the one it expects, so a plugin
// update can replace a stale binary rather than silently keep using it.
var version = "dev"

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
	case "version":
		fmt.Println(version)
	case "where":
		runWhere()
	case "whoami":
		runWhoami()
	case "conflicts":
		runConflicts(os.Args[2:])
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
  claude-team version         Print the build version
  claude-team where           Print the address to watch the room at
  claude-team whoami          Show this peer's identity and room
  claude-team conflicts [room] Show quarantined events (sequence conflicts)

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
  claude-team invite <peer>   Admit a known peer   (inside the room's session)
  claude-team revoke <peer>   Withdraw admission   (inside the room's session)
  claude-team doctor [--deep] Verify relied-on Claude Code behaviors
  claude-team behaviors       List those behaviors (--markdown to render docs)

Environment:
  CLAUDE_TEAM_ADDR        hooks and UI address (loopback only, default 127.0.0.1:4782)
  CLAUDE_TEAM_PEER_ADDR   peer sync address (default 127.0.0.1:4783)
  CLAUDE_TEAM_PREFLIGHT   set to "off" to skip behavior checks on new rooms
  CLAUDE_TEAM_PEERS       comma-separated peer addresses to synchronize with
  CLAUDE_TEAM_SYNC_MS     poll interval in milliseconds (default 1000)
  CLAUDE_TEAM_MAX_EVENTS       turns per injected block (default 40)
  CLAUDE_TEAM_MAX_EVENT_CHARS  characters per turn (default 12000)
  CLAUDE_TEAM_MAX_BLOCK_CHARS  characters per block (default 60000)
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

// Room-scoped commands resolve one way, and only one way: the room the invoking
// session is in.
//
// There is nowhere else to look, by design. Every room-scoped command exists to
// fulfil a slash command, and a slash command always runs inside a session — so a
// terminal that wants one emulates a session, which is what every test here already
// does. The machine-level pointer that used to answer this is gone (D-080): it had
// no user left once the browser view took per-room URLs, and while it existed it
// answered three times with the wrong room.

// roomToChange is for a command that alters who can see what. §22 puts membership
// in a session, so a terminal has no room to admit anybody to (D-079).
func roomToChange(m *Membership) Room {
	sid := sessionID()
	if sid == "" {
		log.Fatal("this changes who can read a room, and only a session that is in one can do it.\n" +
			"Run the matching slash command inside the Claude Code session that is in the room.")
	}
	r, ok := m.RoomForSession(sid)
	if !ok {
		log.Fatal("this session is not in a room. /room-create makes one, /room-join enters one")
	}
	return r
}

// sessionRoom is for a command that only shows something. Same rule, without the
// standing: showing you a room is not an exercise of membership.
func sessionRoom(m *Membership) Room {
	sid := sessionID()
	if sid == "" {
		log.Fatal("no session, so no room. Run the matching slash command inside a Claude Code\n" +
			"session, or set CLAUDE_CODE_SESSION_ID to emulate one.")
	}
	r, ok := m.RoomForSession(sid)
	if !ok {
		log.Fatal("this session is not in a room. /room-create makes one, /room-join enters one")
	}
	return r
}

// openLocal opens the room a person is working in, which since D-046 is a room in
// `membership.db` rather than a name in `config.json`. It read the old config for
// longer than that was true, so `log` reported an empty room called "default"
// while a live room held seven events -- and an empty room reads as a quiet one,
// which is why the staleness went unnoticed (Phase 5 findings).
// sessionID is the Claude Code session a command was invoked from, empty at a
// terminal. Claude Code exports it into every tool call's environment and it is
// the same id the hooks report (B21), so a slash command that shells out needs to
// pass nothing -- which is what lets `create` and `join` put THIS session in a room
// rather than leaving a machine-level setting for some later session to adopt.
func sessionID() string { return os.Getenv("CLAUDE_CODE_SESSION_ID") }

// bindInvokingSession puts the calling session in a room, and says what happened.
// Run at a terminal there is no session to bind, which is not a failure: the room
// exists and command-line commands will act on it.
// watchLine tells a person where to watch the room.
//
// The daemon logs this address once, at startup, into a file — which was fine
// while a person started the daemon themselves and read its output. Since the
// session-start hook began starting it detached, nobody sees that line, so the
// view existed and was unfindable. It is said here instead: at the moments a room
// begins, which is when there is something to watch.
// watchLine names the room when there is one. "This room" is true where a room was
// just created or joined and false where the command stands alone: `where` said
// "watch this room" on a machine with no rooms at all, which is an invented fact in
// the one line whose job is to send somebody somewhere.
func watchLine(room string) string {
	if room == "" {
		return fmt.Sprintf("the room view is at http://%s — it will list rooms once you create or join one", addr())
	}
	// The room's own URL, so what is handed over is a stable link to THAT room
	// rather than a window whose contents can change identity (D-077).
	return fmt.Sprintf("watch %s at http://%s/room/%s", room, addr(), room)
}

// runWhere prints where to watch, naming the room this session is in if it is in
// one. Like whoami, it must answer in no room at all, so it does not use
// currentRoom, which exits.
func runWhere() {
	m, err := OpenMembership()
	if err != nil {
		fmt.Println(watchLine(""))
		return
	}
	defer m.Close()

	// Inside a session, the only room that means anything is the one that session
	// is in. Falling back to the current room here would tell a session with no
	// room to go and watch somebody else's — which is the conflation D-064 exists
	// to prevent: that pointer says what the COMMAND LINE acts on.
	if sid := sessionID(); sid != "" {
		if r, ok := m.RoomForSession(sid); ok {
			fmt.Println(watchLine(r.RoomName))
			return
		}
		fmt.Printf("this session is not in a room. /room-create or /room-join puts it in one.\n")
		fmt.Printf("%s\n", watchLine(""))
		return
	}

	// At a terminal there is no session and so no room. The index lists them.
	fmt.Println(watchLine(""))
}

func bindInvokingSession(m *Membership, r Room) {
	sid := sessionID()
	if sid == "" {
		fmt.Printf("no Claude Code session to put in %s — run /room-create or /room-join\n", r.RoomName)
		fmt.Println("inside a session to put that session in a room. `claude-team` commands")
		fmt.Printf("typed here will act on %s.\n", r.RoomName)
		return
	}
	if err := m.BindSession(sid, r.RoomID); err != nil {
		// Explained rather than merely refused. A person meeting this has done
		// nothing wrong, cannot undo what happened, and needs the way forward more
		// than they need the rule (D-072).
		var bound *BoundElsewhereError
		if errors.As(err, &bound) {
			fmt.Printf("This session cannot join %s.\n\n", bound.Wanted.RoomName)
			fmt.Printf("It has been in %s, and it still holds that room's conversation in its\n", bound.Was.RoomName)
			fmt.Println("context. Joining a second room would carry the first room's conversation into")
			fmt.Println("it — not by copying anything, but by way of what this session says next,")
			fmt.Println("which may be shaped by everything it has read. That would be invisible to")
			fmt.Println("both rooms' members and impossible to take back, because a context window")
			fmt.Println("cannot be un-read. Refusing now is the only moment that prevents it.")
			fmt.Printf("\nTo work in %s, start a second Claude Code session and join from there.\n", bound.Wanted.RoomName)
			fmt.Printf("This session can still rejoin %s.\n", bound.Was.RoomName)
			os.Exit(1)
		}
		log.Fatalf("%v", err)
	}
	fmt.Printf("this session is now in %s.\n", r.RoomName)
}

func openLocal() (*Store, *Identity, Room) {
	id, err := LoadIdentity()
	if err != nil {
		log.Fatalf("identity: %v", err)
	}
	m, err := OpenMembership()
	if err != nil {
		log.Fatalf("membership: %v", err)
	}
	defer m.Close()

	room := sessionRoom(m)
	store, err := OpenStore(room.RoomID)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	return store, id, room
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

	d := &Daemon{id: id, members: members, claudeVersion: ClaudeVersion()}
	defer d.closeStores()

	if !isLoopback(addr()) {
		log.Fatalf("refusing to serve hooks on %s: the hook API publishes into the room "+
			"and reads the conversation back, so it must stay on loopback (§25)", addr())
	}
	local, err := net.Listen("tcp", addr())
	if err != nil {
		// A busy port means one of two very different things, and dying with the
		// same message for both is what made this failure invisible. Several
		// sessions routinely start at once and race to start a daemon: exactly one
		// wins and the rest find the port taken. That is the ORDINARY outcome, not
		// an error (§29). Anything else holding the port is a real fault -- and one
		// nobody would otherwise see, because a hook starts this detached into a log
		// that is not read.
		if daemonAlreadyServing(addr()) {
			clearDaemonState()
			log.Printf("a claude-team daemon is already serving %s; leaving it to it", addr())
			return
		}
		reportDaemonBlocked("hooks and the local view", addr(), err)
		return
	}
	peer, err := net.Listen("tcp", peerAddr())
	if err != nil {
		local.Close()
		reportDaemonBlocked("peer sync", peerAddr(), err)
		return
	}
	// Serving, so anything recorded about a previous failure to serve is stale.
	clearDaemonState()

	// A second way in, for peers on other networks. Optional and best effort: a
	// daemon that cannot start it still serves TCP, because a peer on the same
	// network is unaffected and saying nothing would be worse than saying less.
	advertised := peerAddr()
	var tcListener net.Listener
	if tailcatEnabled() {
		l, endpoint, err := StartTailcat()
		if err != nil {
			log.Printf("  tailcat unavailable (%v) — reachable only at %s", err, peerAddr())
		} else {
			tcListener, advertised = l, endpoint
			defer l.Close()
		}
	}
	// Recorded so that `whoami` and `invite` -- separate processes -- publish the
	// endpoint this daemon is actually listening on rather than the one it binds.
	if err := recordEndpoint(advertised); err != nil {
		log.Printf("  could not record the endpoint (%v); invitations may carry the wrong one", err)
	}

	rooms, _ := members.Rooms()
	log.Printf("claude-team daemon  peer=%s (%s)  serving %d room(s)",
		id.UserDisplayName, id.PeerName, len(rooms))
	if len(rooms) > 0 {
		for _, r := range rooms {
			g, _ := members.Guests(r.RoomID)
			log.Printf("  room          %s (%d guest(s), %d session(s))",
				r.RoomName, len(g), members.SessionsInRoom(r.RoomID))
		}
	} else {
		log.Printf("  current room  none — `claude-team create` or `claude-team join <room>`")
	}
	log.Printf("  hooks and UI  http://%s  (loopback)", addr())
	log.Printf("  peer sync     http://%s", peerAddr())
	if tcListener != nil {
		log.Printf("  reachable at  %s", shortEndpoint(advertised))
	}
	if !isLoopback(peerAddr()) {
		log.Printf("  WARNING: the peer API is reachable from other machines. Requests are")
		log.Printf("           authenticated and events are signed, and only each room's")
		log.Printf("           guests are admitted — but anything reachable is worth knowing about.")
	}

	go d.RunSync(peerList(), syncInterval())
	// One set of routes over both listeners. Every check a request passes is the
	// same whichever carried it, which is the point of the listener interface
	// rather than a happy accident.
	routes := d.PeerRoutes()
	if tcListener != nil {
		go func() {
			if err := http.Serve(tcListener, routes); err != nil && !errors.Is(err, net.ErrClosed) {
				log.Printf("tailcat server stopped: %v", err)
			}
		}()
	}
	go func() {
		if err := http.Serve(peer, routes); err != nil {
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
	resp, err := localPost(client, "http://"+addr()+endpoint, body)
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
	// A simulated peer, so its sequence is simulated too: there is no membership
	// record to reserve from, and none should be invented for a fixture.
	for i, t := range turns {
		if _, err := store.Append(int64(i+1), alice, room.RoomID, aliceSession, t.kind, t.text, nil); err != nil {
			log.Fatalf("seed: %v", err)
		}
	}
	fmt.Printf("seeded %d simulated teammate events into %s\n", len(turns), room.RoomName)
}

func runLog() {
	store, _, room := openLocal()
	defer store.Close()

	evs, err := store.ListRoom(room.RoomID)
	if err != nil {
		log.Fatalf("log: %v", err)
	}
	fmt.Printf("ROOM %s -- %d events\n\n", strings.ToUpper(room.RoomName), len(evs))
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
// resolvePeer turns what a person typed into a key.
//
// It refuses an ambiguous name rather than choosing. Recording two keys under one
// name is now prevented (D-074), but a database written before that could hold
// one, and picking between them would admit a peer nobody named.
// explainNameTaken says what a name collision means, because it is the one place a
// key change can surface and it looks like a naming mistake.
func explainNameTaken(err error) {
	var taken *NameTakenError
	if !errors.As(err, &taken) {
		return
	}
	fmt.Printf("\nYou already know a different key as %q.\n\n", taken.Name)
	fmt.Printf("  already known:  %s\n", taken.Existing)
	fmt.Printf("  offered now:    %s\n\n", taken.Offered)
	fmt.Println("If you expected these to be the same person, their key has changed — and a")
	fmt.Println("changed key is indistinguishable from somebody else's key sent in their name.")
	fmt.Println("That is what an interception looks like after the fact, so check with them on")
	fmt.Println("a call before recording it.")
	fmt.Printf("\nIf they really do have a new key, `%s forget` the old one first.\n", invocation())
	fmt.Printf("That discards its admissions too, so you will invite them again deliberately.\n\n")
}

func resolvePeer(m *Membership, arg string) (string, error) {
	if _, err := PublicFromPeerID(arg); err == nil {
		return arg, nil
	}
	known, err := m.KnownPeers()
	if err != nil {
		return "", err
	}
	var matches []string
	for _, p := range known {
		if p.Name == arg || PeerName(p.PeerID) == arg {
			matches = append(matches, p.PeerID)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", fmt.Errorf("no peer known as %q; pass its identifier, or run /peer-list", arg)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%q names %d different keys, so it names nobody. Say which:", arg, len(matches))
	for _, k := range matches {
		fmt.Fprintf(&b, "\n  %s", k)
	}
	b.WriteString("\n\nTwo keys under one name is what a substituted key looks like once it has been")
	b.WriteString("\nrecorded. If you did not knowingly record both, ask the person which is theirs")
	b.WriteString("\nover a channel you can recognise them on, and `" + invocation() + " forget` the other.")
	return "", errors.New(b.String())
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
			// The first thing a new user sees, so it has to point at the ordinary
			// path. It used to name `allow`, which D-053 reserves for scripts and
			// tests: it records a peer WITHOUT verifying, which lands somebody in
			// exactly the state D-054 refuses to sync, with no hint why. It also
			// printed a bare key rather than the pairing string, so the colleague
			// received no address and could not reach them.
			fmt.Println("No peers yet — pairing is how somebody becomes one.")
			printPairingInvitation(id)
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
			explainNameTaken(err)
			log.Fatalf("allow: %v", err)
		}
		fmt.Printf("recorded %s as %s — UNVERIFIED.\n", PeerName(args[0]), firstNonEmpty(name, PeerName(args[0])))
		fmt.Println("nothing yet says this key is theirs rather than someone who intercepted it.")
		fmt.Printf("finish with:  %s verify %s\n", invocation(), firstNonEmpty(name, PeerName(args[0])))
		fmt.Println("(/peer-pair does both at once, in a browser, and is the ordinary way.)")
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
		cur, hasCur := m.RoomForSession(sessionID())
		rooms, err := m.Rooms()
		if err != nil {
			log.Fatalf("rooms: %v", err)
		}
		if len(rooms) == 0 {
			fmt.Println("no rooms. /room-create in a Claude Code session makes one.")
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
		fmt.Printf("created %s\n  %s\n", r.RoomName, r.RoomID)
		bindInvokingSession(m, r)
		fmt.Printf("%s\n", watchLine(r.RoomName))
		fmt.Println("\ninvite someone you have paired with:")
		fmt.Println("  claude-team invite <name>")
	})
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
		// The session is bound BEFORE anything claims success. A session already
		// in another room cannot be moved (§12a), and printing "joined" and then
		// refusing reads as a half-completed join rather than a refused one.
		bindInvokingSession(m, r)
		fmt.Printf("joined %s. %s\n", r.RoomName, watchLine(r.RoomName))
		if host != "" {
			fmt.Printf("admitted %s, who invited you.\n", PeerName(host))
			if !m.IsVerified(host) {
				fmt.Printf("their key is UNVERIFIED, so nothing will sync yet. On a call with\n")
				fmt.Printf("them, both run /peer-pair — it opens the two-word check in a browser.\n")
			}
		}
		if peers := m.RoomPeers(r.RoomID); len(peers) > 0 {
			fmt.Printf("reaching its members at: %s\n", strings.Join(peers, ", "))
		}
		fmt.Println("Other sessions are unaffected: a session is in a room because someone put")
		fmt.Println("it there, and cannot be moved once it has been told something.")
	})
}

// runLeave takes THIS session out of its room. Membership is held by a session
// (§22), so leaving is per-session: a peer with three sessions in a room leaves
// three times, and that follows from where membership lives rather than being a
// quirk.
//
// It deliberately leaves the current-room pointer alone. Clearing it used to be
// the whole of leaving, and it blanked the browser view — the accumulated
// transcript vanished at the moment a person stopped adding to it, which is the
// opposite of what leaving should mean (D-071).
func runLeave() {
	sid := sessionID()
	if sid == "" {
		fmt.Println("no Claude Code session to take out of a room — run /room-leave inside")
		fmt.Println("the session you want to remove. Membership is held by a session, so")
		fmt.Println("there is nothing at a terminal to leave.")
		return
	}
	withMembership(func(m *Membership, _ *Identity) {
		room, err := m.LeaveSession(sid)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		fmt.Printf("this session has left %s. It stops capturing and stops receiving.\n", room.RoomName)
		fmt.Println("Nothing is hidden or undone: the room's history is unchanged, still readable")
		fmt.Printf("with /room-log, and still shown at http://%s.\n", addr())
		fmt.Println("What you already published stays in the room; leaving does not un-say it.")
		fmt.Printf("\nThis session may rejoin %s. It may not join a different one — what it has\n", room.RoomName)
		fmt.Println("been told cannot be withdrawn from its context.")
	})
}

func runGuests() {
	withMembership(func(m *Membership, id *Identity) {
		r := sessionRoom(m)
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
		log.Fatal("usage: claude-team invite <peer>   (from inside the session that is in the room)")
	}
	withMembership(func(m *Membership, self *Identity) {
		r := roomToChange(m)
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
			fmt.Printf("      direction until it is. On a call with them, both run\n")
			fmt.Printf("      /peer-pair, which opens the two-word check in a browser.\n")
		}
		fmt.Println()
		// An invitation carries a room's identity and where to reach it. It carries
		// no secret: admission is the guest list entry just made, proved later by
		// possession of their key (D-026). Interception reveals that a room exists.
		fmt.Println("give them, to run in a Claude Code session:")
		fmt.Printf("  /room-join %s\n", invitation(r, AdvertisedEndpoint(), self.PeerID))
	})
}

func runRevoke(args []string) {
	if len(args) == 0 {
		log.Fatal("usage: claude-team revoke <peer>   (from inside the session that is in the room)")
	}
	withMembership(func(m *Membership, _ *Identity) {
		r := roomToChange(m)
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

// runConflicts is the one room-scoped command that may be run without a session.
//
// It is a diagnostic, and a diagnostic wanted precisely when a room is misbehaving
// — possibly with no healthy session to ask. Naming a room to READ it is not an
// exercise of standing, which is why this is safe here and not in `invite`
// (D-078, D-080).
func reportConflicts(store *Store, room Room) {
	cs, err := store.ListConflicts()
	if err != nil {
		log.Fatalf("conflicts: %v", err)
	}
	if len(cs) == 0 {
		fmt.Printf("no sequence conflicts in %s\n", room.RoomName)
		return
	}
	fmt.Printf("%d sequence conflict(s) in %s\n\n", len(cs), room.RoomName)
	for _, c := range cs {
		fmt.Printf("  %s  peer %s (%s) sequence %d\n", c.DetectedAt, PeerName(c.PeerID), c.PeerID, c.PeerSequence)
		fmt.Printf("    held:     %s\n    rejected: %s\n\n", c.HeldEventID, c.IncomingEventID)
	}
	fmt.Println("A conflict means that peer's sequence counter went backwards.")
	fmt.Println("Either it lost its local state, or an event was forged. Its events")
	fmt.Println("since then have not been stored, and anti-entropy cannot recover them.")
}

// runConflicts is the one room-scoped command that may be run without a session.
//
// It is a diagnostic, and one wanted precisely when a room is misbehaving —
// possibly with no healthy session to ask. Naming a room in order to READ it is not
// an exercise of standing, which is why this is safe here and not in `invite`
// (D-078, D-080).
func runConflicts(args []string) {
	if len(args) > 0 {
		m, err := OpenMembership()
		if err != nil {
			log.Fatalf("membership: %v", err)
		}
		defer m.Close()
		r, ferr := m.FindRoom(args[0])
		if ferr != nil {
			log.Fatal(ferr)
		}
		store, serr := OpenStore(r.RoomID)
		if serr != nil {
			log.Fatalf("store: %v", serr)
		}
		defer store.Close()
		reportConflicts(store, r)
		return
	}
	store, _, room := openLocal()
	defer store.Close()
	reportConflicts(store, room)
}

func runWhoami() {
	// Deliberately not openLocal: who you are is answerable in no room at all,
	// and a command that reports your identity must not fail for want of one.
	id, err := LoadIdentity()
	if err != nil {
		log.Fatalf("identity: %v", err)
	}
	m, err := OpenMembership()
	if err != nil {
		log.Fatalf("membership: %v", err)
	}
	defer m.Close()

	room := "none"
	if cur, ok := m.RoomForSession(sessionID()); ok {
		room = cur.RoomName
	}
	buf, _ := json.MarshalIndent(map[string]any{
		"identity": id, "peerName": id.PeerName, "room": room, "addr": addr(),
	}, "", "  ")
	fmt.Println(string(buf))
	printPairingInvitation(id)
}

// runVerify is the two-word check (D-048). Both people run it, at the same time,
// on a call where each can recognise the other's voice.
//
// The recognition is the point and cannot be automated: the exchange proves both
// sides hold the keys they named, and only a person can say that the voice saying
// the words is the colleague rather than somebody in their place (§25).
func runVerify(args []string) {
	args, terminal := takeFlag(args, "--terminal")
	if len(args) == 0 {
		log.Fatalf("usage: %s verify <peer> [--terminal]   (both of you, on a call, at the same time)", invocation())
	}
	var peerID, name, mine string
	withMembership(func(m *Membership, id *Identity) {
		pid, err := resolvePeer(m, args[0])
		if err != nil {
			log.Fatal(err)
		}
		peerID, name, mine = pid, PeerName(pid), id.PeerName
	})

	beginCeremony(peerID, name, mine, terminal)
}

func postLocal(path string, body, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	// The exchange waits up to verifyTimeout for the other person to type the
	// command, so the client must outlast it.
	client := &http.Client{Timeout: verifyTimeout + 30*time.Second}
	resp, err := localPost(client, "http://"+addr()+path, raw)
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
	args, terminal := takeFlag(args, "--terminal")
	if len(args) == 0 {
		// Not an error. Pairing has two halves -- sending yours and receiving
		// theirs -- and somebody who runs this with nothing is at the first one.
		// Failing with a usage line would answer a question they did not ask.
		withMembership(func(_ *Membership, id *Identity) { printPairingInvitation(id) })
		return
	}
	peerID, endpoint := parsePairing(args[0])
	name := ""
	if len(args) > 1 {
		name = args[1]
	}

	var mine string
	withMembership(func(m *Membership, id *Identity) {
		if err := m.Allow(peerID, name); err != nil {
			explainNameTaken(err)
			log.Fatalf("pair: %v", err)
		}
		if err := m.SetPeerEndpoint(peerID, endpoint); err != nil {
			log.Fatalf("pair: %v", err)
		}
		mine = id.PeerName
	})

	fmt.Printf("recorded %s.\n", PeerName(peerID))
	if endpoint == "" {
		// Not a dead end. Only ONE side needs a usable address: RunVerification
		// checks for an inbound exchange before it dials, so if they reach us the
		// nonce is already here and the same two words come out. Refusing to start
		// made a working arrangement look broken -- and it is the arrangement a
		// colleague behind a NAT that nothing can traverse actually needs.
		fmt.Println("They sent a key but no address, so you cannot dial them. That still works:")
		fmt.Println("only one of you needs a reachable address, and they can reach you.")
		fmt.Printf("If nothing happens within %s, ask them for their whole pairing string.\n\n", verifyTimeout)
	}
	beginCeremony(peerID, PeerName(peerID), mine, terminal)
}

// takeFlag removes a flag from args and reports whether it was there.
func takeFlag(args []string, flag string) ([]string, bool) {
	out := args[:0:0]
	found := false
	for _, a := range args {
		if a == flag {
			found = true
			continue
		}
		out = append(out, a)
	}
	return out, found
}

// beginCeremony puts the two-word comparison somewhere a person can do it.
//
// The view first (D-086): a terminal is an operator surface, and asking somebody to
// open one to meet a colleague was never a user experience. The terminal path is
// kept and is not a legacy -- it is what happens on a machine with no browser, and
// on one reached over SSH, where the view cannot be shown at all.
//
// Opening the view does NOT wait. The exchange blocks for up to verifyTimeout while
// the other person starts their side, and that wait belongs in the page, not in a
// command a slash command is holding open.
func beginCeremony(peerID, name, mine string, preferTerminal bool) {
	// BOTH paths need the daemon: the terminal one posts to /verify/start just as
	// the page does. Falling back to it when the daemon is down produced a second
	// failure with a different message, which reads as two problems rather than
	// one. Say the one true thing instead.
	if !daemonAlreadyServing(addr()) {
		log.Fatalf("the claude-team daemon is not running, and pairing needs it.\n"+
			"  It starts with a Claude Code session. If one is open, %s doctor says what is wrong.",
			invocation())
	}
	if !preferTerminal {
		var res pairNewResponse
		if err := postLocal("/pair/new", pairNewRequest{Peer: peerID}, &res); err != nil {
			fmt.Printf("could not open a pairing page (%v), so this is happening here instead.\n\n", err)
		} else if res.Error != "" {
			fmt.Printf("%s\n\n", res.Error)
		} else if err := openInBrowser(res.URL); err != nil {
			// A machine with no browser. Say so once, then do the thing that works.
			fmt.Printf("%v, so this is happening here instead.\n\n", err)
		} else {
			fmt.Printf("Opened the pairing page:\n  %s\n\n", res.URL)
			fmt.Printf("Compare the two words there, on a call with %s.\n", name)
			fmt.Printf("They must do the same at the same time — on their machine you are %s.\n", mine)
			return
		}
	}
	fmt.Printf("ask %s to run their side now (on their machine you are %s) — this waits %s.\n\n",
		name, mine, verifyTimeout)
	verifyWith(peerID, name)
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

// --- a daemon that cannot serve says so where somebody will find it ---
//
// The daemon is started detached by a hook, with its output appended to a log
// nobody reads. So a failure to bind used to be perfectly silent: no daemon, no
// capture, no injection, and no statement anywhere about why. The room simply
// looked like a room where nobody was talking.
//
// This writes the reason to a file the session-start hook reads and hands to the
// model, which is the only route from here to a person (D-033, D-036). It is the
// same mechanism D-075 built for a half-finished install, for the same reason:
// absence meant several things and every reader guessed the same one.

func daemonStateFile() string { return filepath.Join(homeDir(), "daemon-state") }

// daemonAlreadyServing reports whether the thing holding an address is one of our
// own daemons. /healthz names the peer it belongs to, so this distinguishes "a
// colleague of mine is already running" from "something unrelated has the port".
func daemonAlreadyServing(address string) bool {
	c := &http.Client{Timeout: 700 * time.Millisecond}
	resp, err := c.Get("http://" + address + "/healthz")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	var body struct {
		OK     bool   `json:"ok"`
		PeerID string `json:"peerId"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&body); err != nil {
		return false
	}
	return body.OK && body.PeerID != ""
}

func reportDaemonBlocked(what, address string, cause error) {
	detail := fmt.Sprintf("%s cannot be served: %s is held by something that is not a claude-team daemon (%v)", what, address, cause)
	recordDaemonState("blocked", detail)
	log.Printf("%s", detail)
	log.Printf("  Nothing is captured or shared until that address is free.")
	log.Printf("  Find what holds it:  lsof -nP -iTCP:%s -sTCP:LISTEN", portOf(address))
	log.Printf("  Or move this daemon: CLAUDE_TEAM_ADDR=127.0.0.1:<port> (hooks and view), CLAUDE_TEAM_PEER_ADDR (peer sync)")
}

// recordDaemonState mirrors the install-state format -- state, when, detail -- so
// the shell side has one shape to read rather than two.
func recordDaemonState(state, detail string) {
	_ = os.MkdirAll(homeDir(), 0o700)
	line := fmt.Sprintf("%s\t%d\t%s\n", state, time.Now().Unix(), strings.ReplaceAll(detail, "\t", " "))
	_ = os.WriteFile(daemonStateFile(), []byte(line), 0o600)
}

func clearDaemonState() { _ = os.Remove(daemonStateFile()) }

func portOf(address string) string {
	if _, port, err := net.SplitHostPort(address); err == nil {
		return port
	}
	return address
}

// invocation is what a PERSON should type to run this binary again.
//
// Not the bare word "claude-team". The plugin installs into ~/.claude-team/bin,
// which is on nobody's PATH, and §29 forbids editing a shell profile to put it
// there. So every line of ours that said "run claude-team pair" was a line that
// fails with "command not found" for every plugin-installed user -- which is most
// of them, and precisely the ones least equipped to work out why. The binary knows
// where it is; a path it prints about itself cannot go stale.
//
// $HOME is spelled ~ because that is what a person recognises, and it pastes into
// any shell unchanged.
func invocation() string {
	exe, err := os.Executable()
	if err != nil {
		return "claude-team"
	}
	if resolved, rerr := filepath.EvalSymlinks(exe); rerr == nil {
		exe = resolved
	}
	if home, herr := os.UserHomeDir(); herr == nil {
		if strings.HasPrefix(exe, home+string(os.PathSeparator)) {
			return "~" + exe[len(home):]
		}
	}
	return exe
}

// printPairingInvitation says what to send a colleague, and what happens next.
//
// The string is safe to send by any means: an identifier is a public key and an
// address is where a daemon listens, and neither admits anybody (D-026, D-042).
// What admits somebody is the two-word comparison, which happens in a browser
// (D-088) rather than at a terminal, because meeting a colleague was never an
// operator task.
func printPairingInvitation(id *Identity) {
	endpoint := AdvertisedEndpoint()
	fmt.Printf("\nSend your colleague this — any channel will do, it is not a secret:\n\n  %s\n",
		pairingString(id.PeerID, endpoint))
	// Attached to the string rather than to a command: three commands print it,
	// and only one of them used to warn.
	if ok, why := pairingReachable(endpoint); !ok {
		fmt.Printf("\nNOTE: %s\n", why)
	}
	fmt.Printf("\nWhen they send you theirs, run /peer-pair with it. A page opens in your\n")
	fmt.Printf("browser showing two words. Get on a call, both of you do this at the same\n")
	fmt.Printf("time, and read the words to each other. They must match.\n")
}

// pairingReachable reports whether a pairing string is usable by the person who
// receives it, and says why when it is not.
//
// The check this replaces asked isLoopback(peerAddr()) -- the address this daemon
// BINDS. That is loopback by default and stays loopback even when tailcat has
// negotiated a perfectly routable endpoint, so the warning fired on every pairing
// string ever printed, including all the ones that worked. A warning that is always
// on is not a warning; it is noise that trains somebody to ignore the real case.
func pairingReachable(endpoint string) (bool, string) {
	e, err := ParseEndpoint(endpoint)
	if err != nil {
		return false, "that address cannot be read, so nobody can reach you at it."
	}
	if e.Scheme == schemeTailcat {
		// Negotiated through DERP and routable by construction. This is the
		// ordinary case and must not warn.
		return true, ""
	}
	// A bare TCP endpoint has to be a host and a port. ParseEndpoint accepts
	// anything without a scheme as TCP, and isLoopback answers "false" for what it
	// cannot parse -- so without this, a malformed address reads as "not loopback"
	// and therefore as fine.
	if _, _, err := net.SplitHostPort(e.Value); err != nil {
		return false, "that address is not a host and port, so nobody can reach you at it."
	}
	if isLoopback(e.Value) {
		if !endpointRecorded() {
			return false, "no daemon has published an address yet, so that one is a guess.\n" +
				"      The daemon records a real one when it starts, which happens when a\n" +
				"      Claude Code session starts. Start one and run this again."
		}
		return false, "that address is loopback, so nobody else can reach it. No route out\n" +
			"      was negotiated; set CLAUDE_TEAM_PEER_ADDR to an address they can reach\n" +
			"      and restart the daemon."
	}
	return true, ""
}

// endpointRecorded distinguishes "a daemon published this" from "we guessed".
func endpointRecorded() bool {
	_, err := os.Stat(filepath.Join(homeDir(), endpointFile))
	return err == nil
}
