package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unicode"
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
	guardRealHome(os.Args[1])
	guardStateDirectory(os.Args[1])
	switch os.Args[1] {
	case "daemon":
		runDaemon()
	case "hook":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		runHookCommand(os.Args[2:])
	case "bridge":
		runBridge()
	case "seed":
		runSeed()
	case "log":
		runLog()
	case "version":
		fmt.Println(version)
	case "stop":
		runStop(os.Args[2:])
	case "where":
		runWhere()
	case "whoami":
		runWhoami()
	case "machines":
		runMachines()
	case "name":
		runName(os.Args[2:])
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
	case "rename":
		runRename(os.Args[2:])
	case "verify":
		runVerify(os.Args[2:])
	case "rooms":
		runRooms()
	case "create":
		runCreateRoom(os.Args[2:])
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
		runDoctorCommand(os.Args[2:])
	case "diagnose-endpoint":
		runDiagnoseEndpoint(os.Args[2:])
	case "probe-bighook":
		runProbeBigHook(os.Args[2:])
	case "cowork-setup":
		runCoworkSetup(os.Args[2:])
	case "probe-mcp":
		if len(os.Args) >= 3 {
			runProbeMCP(os.Args[2])
		}
	case "codex-probe-hook":
		if len(os.Args) == 4 {
			runCodexProbeHook(os.Args[2], os.Args[3])
		}
	case "behaviors":
		if len(os.Args) > 2 && os.Args[2] == "--markdown" {
			fmt.Print(MarkdownReport())
			return
		}
		for _, b := range AllBehaviors() {
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
	fmt.Fprint(os.Stderr, `cogmer -- a conversation shared between Claude Code and Codex sessions

  cogmer daemon          Run the local daemon on `+defaultAddr+`
  cogmer stop            Stop the cogmer daemon on this installation's addresses
  cogmer hook prompt     UserPromptSubmit hook (capture + inject)
  cogmer hook stop       Stop hook (capture assistant turn)
  cogmer bridge          the MCP server Claude Cowork's hooks call (D-205)
  cogmer cowork-setup    add the bridge to Claude Desktop and write the plugin for Cowork
  cogmer seed            Insert a simulated teammate conversation
  cogmer log             Print the room transcript
  cogmer version         Print the build version
  cogmer where           Print the address to watch the room at
  cogmer whoami          Show this peer's identity and room
  cogmer machines        Open the page where your machines are added, renamed and recovered
  cogmer name [text]     Show or set the name other people see for you
  cogmer conflicts [room] Show quarantined events (sequence conflicts)

Peers — durable, above any room. Done once with each colleague:
  cogmer pair <string>   Record a peer AND verify it, on a call with them
  cogmer peers           List peers this machine knows
  cogmer verify <peer>   Re-run just the two-word check
  cogmer allow <id> [nm] Record a peer WITHOUT verifying (scripts, tests)
  cogmer rename <peer> <name> Change what you call a peer
  cogmer forget <id>     Discard a peer entirely

Rooms — per room, repeated as often as you like:

  cogmer rooms           List rooms
  cogmer create          Create a room
  cogmer join <room>     Put this session in a room, from its invitation or its name
  cogmer leave           Take this session out of its room
  cogmer guests          List who may enter this session's room
  cogmer invite <peer>   Admit a known peer   (inside the room's session)
  cogmer revoke <peer>   Withdraw admission   (inside the room's session)
  cogmer doctor [--deep] [--host NAME]
                         Verify the behaviors relied on in the host this session runs in,
                         or in each installed host (claude, codex); --deep adds a compaction
                         for Claude Code
  cogmer behaviors       List those behaviors (--markdown to render docs)

Environment:
  COGMER_ADDR        hooks and UI address (loopback only, default 127.0.0.1:4782)
  COGMER_PEER_ADDR   peer sync address (default 127.0.0.1:4783)
  COGMER_PREFLIGHT   set to "off" to skip behavior checks on new rooms
  COGMER_IDENTITY_PAGE  set to "off" to run a command on a machine with no user identity (tests)
  COGMER_PEERS       comma-separated peer addresses to synchronize with
  COGMER_SYNC_MS     poll interval in milliseconds (default 1000)
  COGMER_MAX_EVENTS       turns per injected block (default 40)
  COGMER_MAX_EVENT_CHARS  characters per turn (default 5000)
  COGMER_MAX_BLOCK_CHARS  characters per block (default 9000)
`)
}

func addr() string {
	if a := os.Getenv("COGMER_ADDR"); a != "" {
		return a
	}
	return defaultAddr
}

func peerAddr() string {
	if a := os.Getenv("COGMER_PEER_ADDR"); a != "" {
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
	if v := os.Getenv("COGMER_SYNC_MS"); v != "" {
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
// fulfill a slash command, and a slash command always runs inside a session — so a
// terminal that wants one emulates a session, which is what every test here already
// does. The machine-level pointer that used to answer this is gone (D-080): it had
// no user left once the browser view took per-room URLs, and while it existed it
// answered three times with the wrong room.

// roomToChange is for a command that alters who can see what. §3.6 puts membership
// in a session, so a terminal has no room to admit anybody to (D-079).
func roomToChange(m *Membership) Room {
	sid := sessionID()
	if sid == "" {
		log.Fatal("this changes who can read a room, and only a session that is in one can do it.\n" +
			"Run the matching slash command inside the agent session that is in the room." + noSessionNote())
	}
	r, ok := m.RoomForSession(sid)
	if !ok {
		log.Fatal(commandText("this session is not in a room. /cogmer:room-create makes one, /cogmer:room-join enters one"))
	}
	return r
}

// sessionRoom is for a command that only shows something. Same rule, without the
// standing: showing you a room is not an exercise of membership.
func sessionRoom(m *Membership) Room {
	sid := sessionID()
	if sid == "" {
		log.Fatal("no session, so no room. Run the matching command inside an agent\n" +
			"session." + noSessionNote())
	}
	r, ok := m.RoomForSession(sid)
	if !ok {
		log.Fatal(commandText("this session is not in a room. /cogmer:room-create makes one, /cogmer:room-join enters one"))
	}
	return r
}

// sessionID is the agent session a command was invoked from, empty at a
// terminal. Claude Code exports it into every tool call's environment and it is
// the same id the hooks report (B21), so a slash command that shells out needs to
// pass nothing -- which is what lets `create` and `join` put THIS session in a room
// rather than leaving a machine-level setting for some later session to adopt.

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
// sessionRoom, which exits.
func runWhere() {
	m, err := OpenMembership()
	if err != nil {
		fmt.Println(watchLine(""))
		return
	}
	defer m.Close()

	// Inside a session, the only room that means anything is the one that session
	// is in. Falling back to some other room here would tell a session with no
	// room to go and watch somebody else's, which is the conflation D-064 exists
	// to prevent (D-080).
	if sid := sessionID(); sid != "" {
		if r, ok := m.RoomForSession(sid); ok {
			fmt.Println(watchLine(r.RoomName))
			return
		}
		fmt.Printf("%s", commandText("this session is not in a room. /cogmer:room-create or /cogmer:room-join puts it in one.\n"))
		fmt.Printf("%s\n", watchLine(""))
		return
	}

	// At a terminal there is no session and so no room. The index lists them.
	fmt.Println(watchLine(""))
}

func bindInvokingSession(m *Membership, r Room) {
	sid := sessionID()
	if sid == "" {
		fmt.Printf(commandText("no agent session to put in %s — run /cogmer:room-create or /cogmer:room-join\n"), r.RoomName)
		fmt.Println("inside a session to put that session in a room. `cogmer` commands")
		fmt.Printf("typed here will act on %s.\n", r.RoomName)
		if note := noSessionNote(); note != "" {
			fmt.Println(strings.TrimPrefix(note, "\n"))
		}
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
			fmt.Printf("\nTo work in %s, start a second agent session and join from there.\n", bound.Wanted.RoomName)
			fmt.Printf("This session can still rejoin %s.\n", bound.Was.RoomName)
			os.Exit(1)
		}
		log.Fatalf("%v", err)
	}
	fmt.Printf("this session is now in %s.\n", r.RoomName)
	if h, err := currentHost(); err == nil && h != nil {
		h.Preflight(r.RoomName)
	}
}

// openLocal opens the room a person is working in, which since D-046 is a room in
// `membership.db` rather than a name in `config.json`. It read the old config for
// longer than that was true, so `log` reported an empty room called "default"
// while a live room held seven events -- and an empty room reads as a quiet one,
// which is why the staleness went unnoticed until the partition run of 2026-09-18.
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

// replaceDaemon stops a daemon of another version holding this installation's
// addresses, and returns the hooks listener once this binary holds it (D-158). It
// returns nil when the address could not be taken, and says why in the log.
//
// Several sessions can start replacements at once. Each stops whatever is still
// in the way, one binds, and the rest find a daemon of their own version serving,
// which they leave alone.
func replaceDaemon(running string) net.Listener {
	log.Printf("the daemon serving %s runs %s and this is %s; replacing it",
		addr(), firstNonEmpty(running, "a version from before daemons reported one"), version)
	outcomes, err := stopDaemonsAt([]string{addr(), peerAddr()})
	for _, o := range outcomes {
		switch {
		case o.Stopped:
			log.Printf("  stopped pid %d", o.PID)
		case o.Err != nil:
			log.Printf("  could not stop pid %d: %v", o.PID, o.Err)
		default:
			log.Printf("  left %s, pid %d, alone: it is not a cogmer daemon", firstNonEmpty(o.Name, "an unnamed process"), o.PID)
		}
	}
	if err != nil {
		log.Printf("  could not replace it (%v); it keeps serving", err)
		return nil
	}
	stopped := false
	for _, o := range outcomes {
		stopped = stopped || o.Stopped
	}
	if !stopped {
		log.Printf("  nothing was stopped, so it keeps serving")
		return nil
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		l, lerr := net.Listen("tcp", addr())
		if lerr == nil {
			return l
		}
		if v, ok := daemonVersionAt(addr()); ok && !shouldReplace(v, version) {
			log.Printf("  a daemon of this version is serving %s now; leaving it to it", addr())
			return nil
		}
		if time.Now().After(deadline) {
			reportDaemonBlocked("hooks and the local view", addr(), lerr)
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// listenWithin listens on an address, retrying until the wait is over.
func listenWithin(address string, wait time.Duration) (net.Listener, error) {
	deadline := time.Now().Add(wait)
	for {
		l, err := net.Listen("tcp", address)
		if err == nil || time.Now().After(deadline) {
			return l, err
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func runDaemon() {
	// A daemon that is running ends the user's stop, so the marker that says one stands
	// is removed here, whoever started this daemon (D-208).
	_ = os.Remove(stoppedMarker())
	keepDaemonLogBounded(daemonLogPath(), daemonLogMax, daemonLogCheckEvery)
	id, err := LoadIdentity()
	if err != nil {
		log.Fatalf("identity: %v", err)
	}
	members, err := OpenMembership()
	if err != nil {
		log.Fatalf("membership: %v", err)
	}

	d := &Daemon{id: id, members: members}
	defer d.closeStores()
	// A cloud session's daemon speaks for its user with a session key the cloud key in
	// its environment certifies (D-216, D-247). The variable is dropped from this
	// process's environment once read, so that nothing the daemon starts inherits it.
	if v, ok := os.LookupEnv(cloudKeyVar); ok {
		_ = os.Unsetenv(cloudKeyVar)
		d.startCloud(v)
	}
	// The machine's own chain is kept with the user's records, so that its own state
	// and the removals its events name include its own certificates, whatever wrote
	// identity.json (D-219, a user is a master key that certifies machines through a
	// shared key).
	d.storeOwnChain()
	// A room left uncarried by a machine that stopped while becoming its user's is
	// carried now (D-243, a room from before user identities takes one guest list).
	d.mu.Lock()
	d.carryRooms()
	d.mu.Unlock()

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
		running, ok := daemonVersionAt(addr())
		if !ok {
			reportDaemonBlocked("hooks and the local view", addr(), err)
			return
		}
		if !shouldReplace(running, version) {
			clearDaemonState()
			log.Printf("a cogmer daemon is already serving %s; leaving it to it", addr())
			return
		}
		if local = replaceDaemon(running); local == nil {
			return
		}
	}
	// The peer address can lag the hooks address by a moment while a replaced
	// daemon exits, so it is tried for a few seconds before it counts as blocked.
	peer, err := listenWithin(peerAddr(), 5*time.Second)
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
		l, endpoint, srv, err := StartTailcat()
		if err != nil {
			log.Printf("  tailcat unavailable (%v) — reachable only at %s", err, peerAddr())
		} else {
			tcListener, advertised = l, endpoint
			log.Printf("  relay connection carried %s", relayMode())
			d.selfCheck = newSelfCheck(srv, endpoint)
			d.relayNow = newRelayProbe(srv)
			defer l.Close()
			defer srv.Close()
		}
	}
	// Recorded so that `whoami` and `invite` -- separate processes -- publish the
	// endpoint this daemon is actually listening on rather than the one it binds.
	if err := recordEndpoint(advertised); err != nil {
		log.Printf("  could not record the endpoint (%v); invitations may carry the wrong one", err)
	}

	// Both peer listeners, including the overlay's, which is already encrypted:
	// uniform TLS means confidentiality never depends on which path a dial took.
	serverTLS, terr := d.serverConfig()
	if terr != nil {
		log.Fatalf("peer tls: %v", terr)
	}
	peer = tls.NewListener(peer, serverTLS)
	if tcListener != nil {
		tcListener = tls.NewListener(tcListener, serverTLS)
	}

	rooms, _ := members.Rooms()
	log.Printf("cogmer daemon  peer=%s (%s)  serving %d room(s)",
		d.id.UserDisplayName, d.id.PeerName, len(rooms))
	if len(rooms) > 0 {
		for _, r := range rooms {
			g, _ := members.Guests(r.RoomID)
			log.Printf("  room          %s (%d guest(s), %d session(s))",
				r.RoomName, len(g), members.SessionsInRoom(r.RoomID))
		}
	} else {
		log.Printf("%s", commandText("  rooms         none yet — /cogmer:room-create in a session makes one"))
	}
	log.Printf("  hooks and UI  http://%s  (loopback)", addr())
	log.Printf("  peer sync     https://%s  (TLS, pinned to known peers)", peerAddr())
	if tcListener != nil {
		log.Printf("  reachable at  %s", shortEndpoint(advertised))
	}
	if !isLoopback(peerAddr()) {
		log.Printf("  NOTE: the peer API is bound where other machines can reach it. Every")
		log.Printf("        connection is TLS pinned to a key this machine already knows, so")
		log.Printf("        a stranger completes no handshake — but anything reachable is")
		log.Printf("        worth knowing about.")
	}

	go d.RunSync(peerList(), syncInterval())
	go d.RunImages(2 * time.Second)
	go d.RunIdentity(5 * time.Second)
	go d.RunSelfLists(10 * time.Second)
	if d.cloud != nil {
		go d.RunCloud(5 * time.Second)
	}
	// One set of routes over both listeners. Every check a request passes is the
	// same whichever carried it, which is the point of the listener interface
	// rather than a happy accident.
	routes := d.PeerRoutes()
	if tcListener != nil {
		go func() {
			if err := peerServer(routes).Serve(tcListener); err != nil && !errors.Is(err, net.ErrClosed) {
				log.Printf("tailcat server stopped: %v", err)
			}
		}()
	}
	// SIGTERM is how `stop` and a replacing daemon end this one (D-123). Closing the
	// listeners ends the servers, and returning runs the deferred closes of the
	// overlay and the stores.
	var stopping atomic.Bool
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, os.Interrupt)
	stopWatch := make(chan struct{})
	go d.RunSelfWatch(stopWatch)
	go func() {
		s := <-signals
		close(stopWatch)
		stopping.Store(true)
		log.Printf("stopping on %v", s)
		local.Close()
		peer.Close()
		if tcListener != nil {
			tcListener.Close()
		}
	}()
	go func() {
		if err := peerServer(routes).Serve(peer); err != nil && !stopping.Load() {
			log.Fatalf("peer server: %v", err)
		}
	}()
	if err := http.Serve(local, d.LocalHandler()); err != nil && !stopping.Load() {
		log.Fatal(err)
	}
}

// runHookCommand reads `hook [--host NAME] prompt|stop`. A hook line written for a
// host says which host it is for, and one that does not is Claude Code's.
func runHookCommand(args []string) {
	args, hostName := takeValue(args, "--host")
	if len(args) != 1 {
		os.Exit(0) // a hook never disturbs the host, whatever it is given
	}
	host, ok := hostByName(hostName)
	if !ok {
		fmt.Fprintf(os.Stderr, "cogmer: a hook was declared for host %q, which cogmer does not support; continuing without collaboration\n", hostName)
		os.Exit(0)
	}
	runHook(args[0], host)
}

// takeValue removes `flag value` from args and returns the value.
func takeValue(args []string, flag string) ([]string, string) {
	out := args[:0:0]
	value := ""
	for i := 0; i < len(args); i++ {
		if args[i] == flag && i+1 < len(args) {
			value = args[i+1]
			i++
			continue
		}
		out = append(out, args[i])
	}
	return out, value
}

// addEntrypoint puts into a hook's payload what the host calls the program running the session.
// Claude Code names it in the environment of its hooks, and the images of a prompt are found
// differently for the terminal and the desktop app.
func addEntrypoint(payload map[string]any, getenv func(string) string) {
	if v := getenv("CLAUDE_CODE_ENTRYPOINT"); v != "" {
		payload["entrypoint"] = v
	}
}

// runHook implements both hook entry points for one host.
//
// Invariant from §3.1: the host must keep working when the daemon is down.
// Every failure path here exits 0 with empty stdout, so a dead daemon degrades
// to "no collaboration" rather than a broken session.
func runHook(kind string, host Host) {
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
	payload["host"] = host.Name()
	addEntrypoint(payload, os.Getenv)

	body, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := localPost(client, "http://"+addr()+endpoint, body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cogmer: daemon unreachable (%v); continuing without collaboration\n", err)
		os.Exit(0)
	}
	defer resp.Body.Close()

	var out struct {
		Context string `json:"context"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		os.Exit(0)
	}
	// What a prompt hook writes to stdout is appended to the pending turn's
	// context, in the form the host reads (verified against Claude Code 2.1.273 and
	// Codex 0.159.3).
	if out.Context != "" {
		_, _ = os.Stdout.Write(host.PromptHookOutput(out.Context))
	}
	os.Exit(0)
}

// runDiagnoseEndpoint is `cogmer diagnose-endpoint <endpoint>`: it says where a connection to an
// endpoint fails, as the daemon does in its log when a peer has stopped answering. It is for
// finding out why a peer cannot be reached from this machine.
func runDiagnoseEndpoint(args []string) {
	if len(args) != 1 {
		log.Fatal("diagnose-endpoint needs one endpoint")
	}
	e, err := ParseEndpoint(args[0])
	if err != nil {
		log.Fatalf("diagnose-endpoint: %v", err)
	}
	dg, ok := dialers[e.Scheme].(Diagnoser)
	if !ok {
		log.Fatalf("diagnose-endpoint: a %s endpoint has no diagnosis", e.Scheme)
	}
	// The network library writes a great deal to the log, and a person asking for a diagnosis
	// wants the diagnosis.
	log.SetOutput(io.Discard)
	ctx, cancel := context.WithTimeout(context.Background(), diagnoseWithin)
	defer cancel()
	fmt.Println(dg.Diagnose(ctx, e))
}

// runDoctorCommand reads `doctor [--deep] [--host NAME]` and checks each host it
// names, or the host this session runs in, or at a terminal every host installed.
func runDoctorCommand(args []string) {
	deep, explicit := false, ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--deep":
			deep = true
		case "--host":
			if i+1 >= len(args) {
				log.Fatal("--host needs the name of a host")
			}
			i++
			explicit = args[i]
		default:
			log.Fatalf("unknown doctor option %q", args[i])
		}
	}
	current, err := currentHost()
	if err != nil {
		log.Fatalf("doctor: %v", err)
	}
	targets, err := doctorTargets(explicit, current, func(h Host) bool { return h.Installed() })
	if err != nil {
		log.Fatalf("doctor: %v", err)
	}
	allPassed := true
	for i, h := range targets {
		if i > 0 {
			fmt.Println()
		}
		ok, err := h.Doctor(deep)
		if errors.Is(err, errDeepUnsupported) {
			if len(targets) == 1 {
				log.Fatalf("doctor: %v; run it without --deep", err)
			}
			fmt.Printf("%s: %v, so its session checks run without --deep.\n\n", h.Name(), err)
			ok, err = h.Doctor(false)
		}
		if err != nil {
			log.Fatalf("doctor: %v", err)
		}
		allPassed = allPassed && ok
	}
	if !allPassed {
		os.Exit(1)
	}
}

// runSeed inserts a simulated teammate exchange so cross-session
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
	m, err := OpenMembership()
	if err != nil {
		log.Fatalf("membership: %v", err)
	}
	defer m.Close()

	evs, err := store.ListRoom(room.RoomID)
	if err != nil {
		log.Fatalf("log: %v", err)
	}
	// Judged as the view judges them, from this machine's records (D-234, a machine not
	// certified under the current shared key is suspended).
	list, _ := m.CachedGuestList(room.RoomID)
	judge := &standings{members: m, room: room.RoomID, list: list, states: map[string]*UserState{}}
	fmt.Print(formatRoomLog(room, evs, judge.of, randomID()))
}

// runConflicts surfaces quarantined events. A conflict means a peer's sequence
// counter went backwards -- lost local state, or a forged event. It is never
// routine, and it is invisible unless asked for.
// resolvePeer accepts an identifier or a name this machine already knows, so a
// person need not paste a key to refer to a colleague they have met.
// resolvePeer turns what a person typed into a key.
//
// It refuses an ambiguous name rather than choosing (D-174). Recording two keys
// under one name is prevented (D-074), but a database written before that could
// hold one, and picking between them would admit a peer nobody named.
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
	fmt.Print(commandText("\nIf they really do have a new key, /cogmer:peer-forget the old one first.\n"))
	fmt.Println("Forgetting leaves every room's guest list as it is, so invite the new key to")
	fmt.Println("their rooms on purpose; the old key stays on the lists it was on.")
	fmt.Println("A colleague who made a user identity in cogmer does not come here: their machine")
	fmt.Println("proves the new identity with the key you verified, and keeps the name you gave them.")
	fmt.Println()
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
		return "", fmt.Errorf(commandText("no peer known as %q; pass its identifier, or run /cogmer:peer-list"), arg)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%q names %d different keys, so it names nobody. Say which:", arg, len(matches))
	for _, k := range matches {
		fmt.Fprintf(&b, "\n  %s", k)
	}
	b.WriteString("\n\nTwo keys under one name is what a substituted key looks like once it has been")
	b.WriteString("\nrecorded. If you did not knowingly record both, ask the person which is theirs")
	b.WriteString(commandText("\nover a channel you can recognize them on, and /cogmer:peer-forget the other."))
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
			// path. It used to name `allow`, which D-165 reserves for scripts and
			// tests: it records a peer WITHOUT verifying, which lands somebody in
			// exactly the state D-054 refuses to sync, with no hint why. It also
			// printed a bare key rather than the pairing string, so the colleague
			// received no address and could not reach them.
			// A prefix names its target (D-096), so this does not print your own
			// string: /cogmer:peer-list is about other people, and there are none.
			fmt.Println("No peers yet. Pairing is how somebody becomes one, and it takes the")
			fmt.Println("string your colleague sends you plus a name for them:")
			fmt.Println()
			fmt.Println(commandText("  /cogmer:peer-pair <their string> <what you call them>"))
			fmt.Println()
			fmt.Println(commandText("Your own string, to send them, is in /cogmer:self-status."))
			return
		}
		var unfinished []string
		ids := make([]string, len(known))
		for i, p := range known {
			ids[i] = p.PeerID
		}
		reported := reportedNames(ids)
		for _, p := range known {
			if p.VerifiedAt == "" {
				unfinished = append(unfinished, p.Name)
			}
			fmt.Print(formatPeer(p, reported[p.PeerID]))
		}
		// A key that proved it belongs to a second user carried nothing to them, and
		// only a pairing says who that user is (D-242).
		for _, c := range m.LinkConflicts() {
			fmt.Printf("\n%s\n", c.Error())
		}
		// Named, not counted. The usual room has two people in it, so a tally is
		// always one and says nothing; what a person can act on is which
		// colleague it is and what to type (D-188).
		for _, who := range unfinished {
			fmt.Printf("\nThere is still work to do with %s — nothing passes between you until\n", who)
			fmt.Printf("you finish pairing. Two words, on a call, both at once:\n\n")
			fmt.Printf(commandText("  /cogmer:peer-pair %s\n"), who)
		}
	})
}

func runAllow(args []string) {
	if len(args) == 0 {
		log.Fatal("usage: cogmer allow <identifier> [name]")
	}
	name := ""
	if len(args) > 1 {
		name = args[1]
	}
	withMembership(func(m *Membership, id *Identity) {
		if err := m.Allow(args[0], name); err != nil {
			explainNameTaken(err)
			log.Fatalf("allow: %v", err)
		}
		if err := m.RecordColleague(id, args[0]); err != nil {
			fmt.Printf("recorded here, but your other machines were not told: %v\n", err)
		}
		fmt.Printf("recorded %s as %s — UNVERIFIED.\n", PeerName(args[0]), firstNonEmpty(name, PeerName(args[0])))
		fmt.Println("nothing yet says this key is theirs rather than someone who intercepted it.")
		fmt.Printf("finish with:  %s verify %s\n", invocation(), firstNonEmpty(name, PeerName(args[0])))
		fmt.Println(commandText("(/cogmer:peer-pair does both at once, in a browser, and is the ordinary way.)"))
	})
}

func runRename(args []string) {
	if len(args) < 2 {
		log.Fatal(commandText("usage: /cogmer:peer-rename <peer> <new name>"))
	}
	withMembership(func(m *Membership, id *Identity) {
		text, err := renamePeer(m, id, args[0], strings.Join(args[1:], " "))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(text)
	})
}

// renamePeer changes the name this user gives a colleague and nothing else: whether
// they are verified, and every room, stay as they were (D-257, a user renames a
// colleague with a command that changes the label and nothing else). The change goes
// to the user's other machines as a label does (D-228). It returns what to tell the
// user.
func renamePeer(m *Membership, id *Identity, who, name string) (string, error) {
	pid, err := resolvePeer(m, who)
	if err != nil {
		return "", err
	}
	pid = m.colleagueOf(pid)
	if !m.Knows(pid) {
		return "", fmt.Errorf(commandText("no peer known as %q; run /cogmer:peer-list"), who)
	}
	newName := sanitizeName(name)
	if newName == "" {
		return "", errors.New(commandText("usage: /cogmer:peer-rename <peer> <new name>"))
	}
	old := firstNonEmpty(m.Label(pid), PeerName(pid))
	if newName == old {
		return fmt.Sprintf("%s is already called %s.", PeerName(pid), newName), nil
	}
	if err := m.Allow(pid, newName); err != nil {
		var taken *NameTakenError
		if errors.As(err, &taken) {
			return "", fmt.Errorf(commandText("you already call a different peer %q (%s), so %s is unchanged. Pick another name, or /cogmer:peer-forget the other one first."),
				taken.Name, PeerName(taken.Existing), old)
		}
		return "", fmt.Errorf("rename: %w", err)
	}
	out := fmt.Sprintf("renamed %s to %s.", old, newName)
	if newName != strings.TrimSpace(name) {
		out += fmt.Sprintf(" (The name was shortened to what a label can hold: %q.)", newName)
	}
	if err := m.RecordColleague(id, pid); err != nil {
		out += fmt.Sprintf("\nRenamed here, but your other machines were not told: %v", err)
	}
	return out, nil
}

func runForget(args []string) {
	if len(args) == 0 {
		log.Fatal(commandText("usage: /cogmer:peer-forget <peer>"))
	}
	withMembership(func(m *Membership, id *Identity) {
		pid, err := resolvePeer(m, args[0])
		if err != nil {
			log.Fatal(err)
		}
		// Read before forgetting: the label is about to be discarded with the
		// peer, and naming them by a word pair they were never called reads as
		// having forgotten somebody else.
		who := firstNonEmpty(m.Label(pid), PeerName(pid))
		if err := m.Forget(pid); err != nil {
			log.Fatalf("forget: %v", err)
		}
		// Forgetting on one machine forgets on every machine of the user (D-228;
		// D-073, forgetting a peer changes no room's guest list).
		if err := m.RecordColleague(id, pid); err != nil {
			fmt.Printf("forgot here, but your other machines were not told: %v\n", err)
		}
		fmt.Printf("forgot %s. A later meeting will be a first meeting.\n", who)
	})
}

func runRooms() {
	withMembership(func(m *Membership, id *Identity) {
		cur, hasCur := m.RoomForSession(sessionID())
		rooms, err := m.Rooms()
		if err != nil {
			log.Fatalf("rooms: %v", err)
		}
		var toJoin []listedRoom
		if id.HasUser() {
			toJoin = m.RoomsToJoin(id.User)
		}
		if len(rooms) == 0 && len(toJoin) == 0 {
			if len(m.Offers()) == 0 {
				fmt.Println(commandText("no rooms. /cogmer:room-create in an agent session makes one."))
				return
			}
		}
		for _, r := range rooms {
			g, _ := m.Guests(r.RoomID)
			mark := " "
			if hasCur && r.RoomID == cur.RoomID {
				mark = "*"
			}
			note := ""
			if r.ShowOnly {
				note = "  show-only"
			}
			fmt.Printf("%s %-24s %-8s %d guest(s)  %d session(s)%s\n",
				mark, r.RoomName, r.State, len(g), m.SessionsInRoom(r.RoomID), note)
		}
		// Rooms somebody has admitted you to and told you about. Listed here
		// rather than anywhere else because they are rooms, and kept apart from
		// the ones above because an offer is a thing to accept rather than a room
		// you are in (D-105).
		if offers := m.Offers(); len(offers) > 0 {
			fmt.Println("\nwaiting for you to accept:")
			for _, o := range offers {
				fmt.Printf(commandText("  %-24s from %-20s  /cogmer:room-join %s\n"),
					o.RoomName, firstNonEmpty(m.Label(o.Host), PeerName(o.Host)), o.RoomName)
			}
		}
		// Rooms of yours that a session on another of your machines joined, which a
		// session here joins by name.
		if len(toJoin) > 0 {
			fmt.Println("\non your room list, not joined on this machine:")
			for _, r := range toJoin {
				fmt.Printf(commandText("  %-24s /cogmer:room-join %s\n"), r.Name, r.Name)
			}
		}
	})
}

func runCreateRoom(args []string) {
	rest, showOnly := takeFlag(args, "--show-only")
	rest, another := takeFlag(rest, "--another")
	if len(rest) > 0 {
		log.Fatalf("create takes no argument but --show-only and --another, and was given %q", strings.Join(rest, " "))
	}
	refuseWhenTheHostIsUnclear()
	withMembership(func(m *Membership, id *Identity) {
		if needsUserIdentity(id) {
			return
		}
		if !another {
			if note := createRefusal(m, sessionID()); note != "" {
				fmt.Print(note)
				os.Exit(1)
			}
		}
		r, err := createRoom(m, id, showOnly)
		if err != nil {
			log.Fatalf("create: %v", err)
		}
		fmt.Printf("created %s\n  %s\n", r.RoomName, r.RoomID)
		if r.ShowOnly {
			fmt.Print(showOnlyNote(r))
		}
		bindInvokingSession(m, r)
		fmt.Printf("%s\n", watchLine(r.RoomName))
		fmt.Println("\ninvite someone you have paired with:")
		fmt.Println(commandText("  /cogmer:room-invite <name>"))
	})
}

// createRefusal says why a session that already has a room should not make another, and
// is empty for a session with none. A second room is a new room each time, and the
// session cannot be put in it, so a repeated command is refused unless the user asks for
// another with --another.
func createRefusal(m *Membership, sid string) string {
	if sid == "" {
		return ""
	}
	room, left, bound := m.SessionBinding(sid)
	if !bound {
		return ""
	}
	if left {
		return fmt.Sprintf("This session left %s and cannot be put in a different room, so a room made now would have no one of yours in it.\n"+
			"Nothing was created. To make another room from here anyway, run the command again with --another.\n", room.RoomName)
	}
	return fmt.Sprintf("This session is already in %s. Nothing was created.\n"+
		"A session cannot be moved to a second room, so a room made now could be entered only from another session. To make another room anyway, run the command again with --another.\n", room.RoomName)
}

// createRoom makes a room here. A machine that speaks for a user has the daemon found
// it, since its first event is the user's invitation of themselves and every write to
// a room's database is the daemon's (D-240, a room's identifier is the hash of its
// creator and a random value). A machine with no user identity records the room and
// its creator in room_guests, as it always has.
func createRoom(m *Membership, id *Identity, showOnly bool) (Room, error) {
	if !id.HasUser() {
		r, err := m.CreateRoom(id.Peer(), showOnly)
		if err != nil {
			return Room{}, err
		}
		return r, m.RecordRoomMachine(r.RoomID, id.PeerID, id.Peer())
	}
	var res guestChangeResponse
	if err := postLocal("/room/guests", guestChangeRequest{Op: "found", ShowOnly: showOnly}, &res); err != nil {
		return Room{}, fmt.Errorf("the daemon did not answer, so no room was created: %w", err)
	}
	if res.Error != "" || res.Room == nil {
		return Room{}, errors.New(firstNonEmpty(res.Error, "the daemon created no room"))
	}
	return *res.Room, nil
}

// changeGuestList asks the daemon to write an invitation or a revocation into a room
// whose guest list is kept as events (D-235, a room's one guest list, changed by any
// guest through events in the room).
func changeGuestList(op, roomID, user string) error {
	var res guestChangeResponse
	if err := postLocal("/room/guests", guestChangeRequest{Op: op, RoomID: roomID, User: user}, &res); err != nil {
		return fmt.Errorf("the daemon did not answer, so nothing was changed: %w", err)
	}
	if res.Error != "" {
		return errors.New(res.Error)
	}
	return nil
}

// refuseWhenTheHostIsUnclear stops a command that is about to write a room, a peer
// or an admission when the host it runs in cannot be told. Such a command binds the
// session that ran it, and with no way to name that session the room would be left
// with nobody in it. A terminal has no host and no error, and goes on as it always
// has (D-203).
func refuseWhenTheHostIsUnclear() {
	if _, err := currentHost(); err != nil {
		fmt.Printf("%v\nNothing was created or recorded.\n", err)
		os.Exit(1)
	}
}

// showOnlyNote says what a show-only room means on this machine, and what it does
// not. The setting is this person's own: it stops this machine's sessions receiving
// the room, and it cannot stop a colleague's machine doing so.
func showOnlyNote(r Room) string {
	return "\nThis room is show-only on this machine. Its conversation appears in the room view, and none of it\n" +
		"is given to your agent. Your own turns are still shared with the room's other members, and their\n" +
		"sessions receive them unless they chose show-only too. This is fixed for this room.\n"
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
	args, showOnlyFlag := takeFlag(args, "--show-only")
	if len(args) == 0 {
		log.Fatal(commandText("usage: /cogmer:room-join <invitation> [--show-only]   (or a room you already know)"))
	}
	refuseWhenTheHostIsUnclear()
	withMembership(func(m *Membership, id *Identity) {
		if needsUserIdentity(id) {
			return
		}
		name, roomID, endpoint, host, full := parseInvitation(args[0])
		// A name on its own may be an offer already delivered over the channel
		// (D-105), in which case the four facts an invitation line carries are
		// already here and the person types only what they were shown.
		if !full {
			if o, ok := m.FindOffer(name); ok {
				name, roomID, endpoint, host, full = o.RoomName, o.RoomID, o.Endpoint, o.Host, true
				defer func() { _ = m.DropOffer(o.RoomID) }()
			}
		}
		// The setting is the person's own and is fixed when a room is first recorded
		// here (D-200). A room already recorded keeps its setting, and a request for
		// the other one is refused before anything is written, so a refused join
		// admits nobody and records no address.
		showOnly := showOnlyFlag
		if full {
			if existing, ok := m.RoomByID(roomID); ok && !showOnlyFlag {
				showOnly = existing.ShowOnly
			}
			if err := m.CheckShowOnly(roomID, showOnly); err != nil {
				fmt.Printf("%v\n", err)
				os.Exit(1)
			}
		}
		if full {
			// A room learned from an invitation: the guest did not create it, so
			// its identity comes from the invitation rather than being invented.
			if err := m.RecordRoom(roomID, name, showOnly); err != nil {
				log.Fatalf("join: %v", err)
			}
			// Admit whoever offered the invitation. Synchronization is a pull in
			// both directions, so a guest that records the room and not its host
			// can read that room and never be read -- which looks like one-way
			// collaboration and is really a one-sided guest list.
			//
			// The address is recorded against the host and after Allow, because
			// SetPeerEndpoint is an UPDATE: run before the row exists it matches
			// nothing and the only address anybody had is lost (D-103).
			if host != "" {
				if err := m.Allow(host, ""); err != nil {
					log.Fatalf("join: %v", err)
				}
				if err := m.SetPeerEndpoint(host, endpoint); err != nil {
					log.Fatalf("join: %v", err)
				}
				if err := m.Invite(roomID, host); err != nil {
					log.Fatalf("join: %v", err)
				}
				// The inviting machine has joined the room, and is where this machine
				// first asks for it. Its key is not in the invitation, so it is recorded
				// under the identifier its address was recorded under, until its own
				// sync request names the machine (D-103, a room's sync targets are the
				// machines from which a session has joined it).
				if err := m.RecordRoomMachine(roomID, host, host); err != nil {
					log.Fatalf("join: %v", err)
				}
			}
		}
		// A room on the user's room list, created or joined from another of the user's
		// machines or offered to the user there, is joined here by its name (D-228).
		if !full {
			if _, err := joinFromList(m, id, name, showOnlyFlag); err != nil {
				log.Fatalf("join: %v", err)
			}
		}
		r, err := m.FindRoom(name)
		if errors.Is(err, errNoSuchRoom) {
			if offers := m.Offers(); len(offers) > 0 {
				fmt.Printf("No room named %q. Waiting for you:\n", name)
				for _, o := range offers {
					fmt.Printf("  %-20s from %s\n", o.RoomName,
						firstNonEmpty(m.Label(o.Host), PeerName(o.Host)))
				}
				return
			}
			log.Fatalf("no room named %q. Ask its host for an invitation; yours to give them is:\n  %s",
				name, id.Peer())
		}
		if err != nil {
			log.Fatal(err)
		}
		if !full {
			// A room named on its own is one already recorded here, and the setting it
			// was recorded with stands (D-200).
			if err := m.CheckShowOnly(r.RoomID, showOnlyFlag || r.ShowOnly); err != nil {
				fmt.Printf("%v\n", err)
				os.Exit(1)
			}
		}
		// The session is bound BEFORE anything claims success. A session already
		// in another room cannot be moved (§12a), and printing "joined" and then
		// refusing reads as a half-completed join rather than a refused one.
		bindInvokingSession(m, r)
		// A session here has joined the room, so this machine is one of its machines.
		if err := m.RecordRoomMachine(r.RoomID, id.PeerID, id.Peer()); err != nil {
			log.Fatalf("join: %v", err)
		}
		// The room and this machine's joining it go on the user's room list, so the
		// user's other machines can join it by name and sync it with this one (D-228).
		if err := m.RecordRoomOnList(id, r.RoomID, r.RoomName, withMachine(id.PeerID)); err != nil {
			fmt.Printf("joined here, but your other machines were not told: %v\n", err)
		}
		fmt.Printf("joined %s. %s\n", r.RoomName, watchLine(r.RoomName))
		if r.ShowOnly {
			fmt.Print(showOnlyNote(r))
		}
		if host != "" {
			// The derived name, correctly: the host was recorded a moment ago with
			// no label, because nobody has chosen one for them yet.
			fmt.Printf("admitted %s, who invited you.\n", PeerName(host))
			if !m.IsVerified(host) {
				fmt.Printf("their key is UNVERIFIED, so nothing will sync yet. On a call with\n")
				fmt.Printf("%s", commandText("them, both run /cogmer:peer-pair — it opens the two-word check in a browser.\n"))
			}
		}
		if peers := m.RoomPeers(r.RoomID, id.PeerID); len(peers) > 0 {
			fmt.Printf("reaching its members at: %s\n", strings.Join(peers, ", "))
		}
		fmt.Println("Other sessions are unaffected: a session is in a room because someone put")
		fmt.Println("it there, and cannot be moved once it has been told something.")
	})
}

// runLeave takes THIS session out of its room. Membership is held by a session
// (§3.6), so leaving is per-session: a peer with three sessions in a room leaves
// three times, and that follows from where membership lives rather than being a
// quirk.
//
// It leaves the room's history alone, so the browser view keeps the accumulated
// transcript at the moment a person stops adding to it (D-071).
func runLeave() {
	sid := sessionID()
	if sid == "" {
		fmt.Println(commandText("no agent session to take out of a room — run /cogmer:room-leave inside"))
		fmt.Println("the session you want to remove. Membership is held by a session, so")
		fmt.Println("there is nothing at a terminal to leave.")
		return
	}
	withMembership(func(m *Membership, id *Identity) {
		room, err := m.LeaveSession(sid)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		// With no session of this machine left in the room, the user's room list no
		// longer names this machine as one that joined it (D-228).
		if m.SessionsInRoom(room.RoomID) == 0 {
			if err := m.RecordRoomOnList(id, room.RoomID, room.RoomName, withoutMachine(id.PeerID)); err != nil {
				fmt.Printf("left here, but your other machines were not told: %v\n", err)
			}
		}
		fmt.Printf("this session has left %s. It stops capturing and stops receiving.\n", room.RoomName)
		fmt.Println("Nothing is hidden or undone: the room's history is unchanged, still readable")
		fmt.Printf(commandText("with /cogmer:room-log, and still shown at http://%s.\n"), addr())
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
		reach, self, up, refusing, fromBefore := fetchPeers(r.RoomID)
		var rows []guestRow
		for _, p := range g {
			rows = append(rows, guestRow{Name: firstNonEmpty(m.Label(p), PeerName(p)), ID: p, Self: p == id.Peer()})
		}
		// A user removed from the room is no longer on its list, and is told so here,
		// by whom and from which turns on (D-241).
		removal, removed := localRemoval(m, id, r)
		if removed && !slices.Contains(g, id.Peer()) {
			rows = append([]guestRow{{Name: "you", ID: id.Peer(), Self: true}}, rows...)
		}
		renderGuests(os.Stdout, r.RoomName, rows, reach, self, up)
		removal.FromBeforeUsers = fromBefore
		renderRemoval(os.Stdout, removal, removed, refusing)
		if line := coworkLine(m.SessionIDsInRoom(r.RoomID), fetchHostCalls(), up); line != "" {
			fmt.Println()
			fmt.Println(line)
		}
	})
}

// coworkLine says whether a Cowork session of this room has reached this machine, when
// the room has one. A Cowork session whose hooks stop reaching the bridge captures and
// injects nothing and shows nothing, so this is where it shows (D-205). It says nothing
// when the room has no Cowork session or no daemon answered.
func coworkLine(sessions []string, calls map[string]string, daemonUp bool) string {
	if !daemonUp {
		return ""
	}
	n := 0
	for _, s := range sessions {
		if strings.HasPrefix(s, (coworkHost{}).SessionID("")) {
			n++
		}
	}
	if n == 0 {
		return ""
	}
	if age, ok := calls["cowork"]; ok {
		return fmt.Sprintf("Cowork: %d session(s) in this room. A Cowork session last reached this machine %s ago.", n, age)
	}
	return fmt.Sprintf("Cowork: %d session(s) in this room, and no call from Cowork has reached this machine yet. "+
		"If one is in use, typing a cogmer command in it shows whether cogmer answers there.", n)
}

// fetchHostCalls asks the running daemon when each host last called it.
func fetchHostCalls() map[string]string {
	c := &http.Client{Timeout: 2 * time.Second}
	resp, err := c.Get("http://" + addr() + "/peers")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var body struct {
		HostCalls map[string]string `json:"hostCalls"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body) != nil {
		return nil
	}
	return body.HostCalls
}

// localRemoval reads this machine's record of its user's removal from a room, which is
// in the room's own events, so the room status says it whether or not a daemon runs.
func localRemoval(m *Membership, id *Identity, r Room) (removalSummary, bool) {
	d := &Daemon{id: id, members: m}
	defer d.closeStores()
	return d.removalFrom(r.RoomID)
}

// renderRemoval says that the user was removed from the room, by whom and from which
// of their turns on, and that their sessions keep their turns; or, when this machine
// holds no revocation and the room's guests refuse it, that they are refusing it and
// what that usually means, and no more (D-241).
func renderRemoval(w io.Writer, s removalSummary, removed, refusing bool) {
	switch {
	case removed:
		fmt.Fprintln(w)
		fmt.Fprintln(w, "YOU WERE REMOVED FROM THIS ROOM. "+s.sentence())
		fmt.Fprintln(w, "The room up to then stays readable here. Your sessions in it keep recording your turns on this")
		fmt.Fprintln(w, "machine, and they reach the room if a guest invites you back.")
		if s.Resuming {
			fmt.Fprintln(w, "A guest has offered the room to you again, so it is coming back once their invitation arrives.")
		}
	case refusing && s.FromBeforeUsers:
		// A room made before user identities is refused by a guest's machine that does
		// not count this user yet, which is the ordinary state while its guests upgrade,
		// and not a removal (D-242; D-243).
		fmt.Fprintln(w)
		fmt.Fprintln(w, "The room's guests are refusing this machine. The room was made before user identities, and a")
		fmt.Fprintln(w, "guest's machine refuses yours until it counts you as a guest: once it has heard from this machine")
		fmt.Fprintln(w, "since you made your identity, and a guest whose earlier list admitted you has upgraded and made theirs.")
	case refusing:
		fmt.Fprintln(w)
		fmt.Fprintln(w, "The room's guests are refusing this machine, which usually means you were removed from the room.")
	}
}

// fetchPeers asks the running daemon whom it can reach for one room: its colleagues,
// one line each, the user's own machines that joined it, one line each, and whether
// its guests are refusing this machine (D-194; D-241), and whether the room was made
// before user identities, which such a refusal usually means (D-243). The answer lives in the daemon,
// which is the process that tries, so a command that is a separate process has to ask
// it. The third result is false when no daemon answered.
func fetchPeers(roomID string) (peers []uiPeer, self *uiSelf, up, refusing, fromBeforeUsers bool) {
	c := &http.Client{Timeout: 2 * time.Second}
	resp, err := c.Get("http://" + addr() + "/peers?room=" + url.QueryEscape(roomID))
	if err != nil {
		return nil, nil, false, false, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, false, false, false
	}
	var body struct {
		Peers      []uiPeer `json:"peers"`
		Self       *uiSelf  `json:"self"`
		Refusing   bool     `json:"refusing"`
		FromBefore bool     `json:"fromBeforeUsers"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return nil, nil, false, false, false
	}
	return body.Peers, body.Self, true, body.Refusing, body.FromBefore
}

type guestRow struct {
	Name string
	ID   string
	Self bool
}

// renderGuests lists who may enter a room and, for each colleague, whether this
// machine can reach them. A colleague this machine cannot reach looks the same as a
// quiet room, so it is said in words beside their name and not left to a log.
func renderGuests(w io.Writer, room string, rows []guestRow, reach []uiPeer, self *uiSelf, daemonUp bool) {
	fmt.Fprintf(w, "%s\n", room)
	if self != nil && self.Failing {
		// Said before the colleagues, because it is the cause of the others: a machine
		// nobody can reach does not hand its turns to anybody, since peers pull.
		fmt.Fprintf(w, "  THIS MACHINE CANNOT BE REACHED through its relay%s, so colleagues do not receive\n", selfFailingFor(self))
		fmt.Fprintf(w, "%s", commandText("  your turns. /cogmer:self-restart restarts the daemon.\n"))
	}
	byID := map[string]*uiPeer{}
	for i := range reach {
		if reach[i].PeerID != "" {
			byID[reach[i].PeerID] = &reach[i]
		}
	}
	unreachable := false
	for _, g := range rows {
		mine := ""
		if g.Self {
			mine = "  (you)"
		}
		fmt.Fprintf(w, "  %-22s %s%s\n", g.Name, g.ID[:32]+"…", mine)
		if g.Self {
			// The user's own machines that joined the room, one by one, since the user
			// acts on each (D-194).
			for _, p := range reach {
				if p.Own {
					fmt.Fprintf(w, "    your machine %s: %s\n", p.Name, ownReachLine(p, daemonUp))
				}
			}
			continue
		}
		line := reachLine(byID[g.ID], daemonUp)
		fmt.Fprintf(w, "    %s\n", line)
		if strings.HasPrefix(line, "NOT REACHABLE") {
			unreachable = true
		}
	}
	if unreachable {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Nothing passes between you and a colleague this machine cannot reach, so their turns")
		fmt.Fprintln(w, commandText("do not arrive here, and the room looks quiet. They can run /cogmer:self-status to see"))
		fmt.Fprintln(w, "whether this machine can be reached from theirs, and whether their daemon is running.")
	}
}

// ownReachLine says whether this machine can reach one of the user's own machines.
func ownReachLine(p uiPeer, daemonUp bool) string {
	if daemonUp && p.NoAddress {
		return "no address is recorded for it here, so nothing here tries to reach it"
	}
	return reachLine(&p, daemonUp)
}

// selfFailingFor says for how long this daemon has been unreachable, in a few words.
func selfFailingFor(s *uiSelf) string {
	switch {
	case s.Detached != "":
		return " (not attached to a relay for " + s.Detached + ")"
	case s.FailingFor != "":
		return " (for " + s.FailingFor + ", " + shortReason(s.Error) + ")"
	}
	return ""
}

// reachLine says in a few words whether this machine can reach one colleague.
func reachLine(p *uiPeer, daemonUp bool) string {
	switch {
	case !daemonUp:
		return "can this machine reach them: unknown, because the daemon is not running"
	case p == nil:
		return "no address is recorded for them, so nothing here tries to reach them"
	case p.Online:
		return "reachable"
	case p.Trying != "" && p.Since == reachUnknown:
		return fmt.Sprintf("NOT REACHABLE: this machine has tried for %s and not got through (%s). %s", p.Trying, shortReason(p.Error), heardClause(p))
	case p.Trying != "":
		return fmt.Sprintf("NOT REACHABLE for %s, last reached %s (%s). %s", p.Trying, p.Since, shortReason(p.Error), heardClause(p))
	case p.Since == reachUnknown:
		return "checking whether they can be reached"
	default:
		return "last reached " + p.Since
	}
}

// heardClause says when the colleague last made a request of this machine. One who
// reached this machine a minute ago has a daemon that is running, so the fault is in
// reaching it, which is a different search from a daemon that was never started.
func heardClause(p *uiPeer) string {
	if p.HeardFrom == "" {
		return "They have not contacted this machine in that time either."
	}
	return "They last reached this machine " + p.HeardFrom + " ago."
}

// shortReason trims an error to the part a person can use: a timeout says the
// colleague did not answer, and the rest is a path through the code.
func shortReason(e string) string {
	switch {
	case e == "":
		return "no reason recorded"
	case strings.Contains(e, "deadline exceeded"):
		return "no answer"
	case strings.Contains(e, "refused"):
		return "refused"
	}
	if len(e) > 80 {
		return e[:80] + "…"
	}
	return e
}

func runInvite(args []string) {
	if len(args) == 0 {
		log.Fatal(commandText("usage: /cogmer:room-invite <peer>   (from inside the session that is in the room)"))
	}
	withMembership(func(m *Membership, self *Identity) {
		r := roomToChange(m)
		pid, err := resolvePeer(m, args[0])
		if err != nil {
			log.Fatal(err)
		}
		if m.EventListed(r.RoomID) {
			if err := changeGuestList("invite", r.RoomID, pid); err != nil {
				log.Fatalf("invite: %v", err)
			}
		} else if err := m.Invite(r.RoomID, pid); err != nil {
			log.Fatalf("invite: %v", err)
		}
		// The label, not the derived name. Switching between the two inside one
		// message reads as two different peers (D-094).
		fmt.Printf("%s may now enter %s.\n", firstNonEmpty(m.Label(pid), PeerName(pid)), r.RoomName)

		// Admitted, queued, one step from done — and never a refusal. Wanting to
		// start a room is what makes somebody willing to verify, so this is the
		// moment the check finally has a visible purpose, and blocking here would
		// send them away to do an errand instead of finishing (D-106).
		if !m.IsVerified(pid) {
			who := firstNonEmpty(m.Label(pid), PeerName(pid))
			fmt.Printf("\nThey have not been told yet — you and %s have not finished pairing,\n", who)
			fmt.Printf("and until you do nothing would pass between you in either direction.\n\n")
			fmt.Printf(commandText("  /cogmer:peer-pair %s\n\n"), who)
			fmt.Printf("Two words, on a call, both at once. The invitation goes when you finish.\n")
			return
		}

		var res sendOfferResponse
		if err := postLocal("/offer/send", sendOfferRequest{Peer: pid, RoomID: r.RoomID}, &res); err == nil && res.Delivered {
			fmt.Printf(commandText("Told them. They accept with:  /cogmer:room-join %s\n"), r.RoomName)
			return
		} else if err == nil && res.Error != "" {
			fmt.Printf("\nCould not reach them (%s), so send this instead:\n", res.Error)
		} else {
			fmt.Printf("\nCould not reach them, so send this instead:\n")
		}
		// The fallback carries the same four facts by hand. The address in it is
		// the HOST's, so it is unaffected by whatever made the guest unreachable.
		fmt.Printf(commandText("  /cogmer:room-join %s\n"), invitation(r, AdvertisedEndpoint(), self.Peer()))
	})
}

func runRevoke(args []string) {
	if len(args) == 0 {
		log.Fatal(commandText("usage: /cogmer:room-revoke <peer>   (from inside the session that is in the room)"))
	}
	withMembership(func(m *Membership, _ *Identity) {
		r := roomToChange(m)
		pid, err := resolvePeer(m, args[0])
		if err != nil {
			log.Fatal(err)
		}
		if m.EventListed(r.RoomID) {
			if err := changeGuestList("revoke", r.RoomID, pid); err != nil {
				log.Fatalf("revoke: %v", err)
			}
		} else if err := m.Revoke(r.RoomID, pid); err != nil {
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
// (D-080).
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
// (D-080).
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
	if id.NameChosen {
		// An unchosen name is said where the identity first travels, so it is not
		// said twice.
		fmt.Printf("\n%s\n", nameLine(id))
	}
	// A first pairing string needs the user identity, since the string names the
	// user (D-230, a user identity is made in the daemon's page the first time the
	// user needs one).
	if id.HasUser() || userIdentityGateOff() {
		printPairingInvitation(id)
	} else {
		fmt.Println()
		writeIdentityNeeded(os.Stdout)
	}
	writeOwnMachines(os.Stdout, m, id)

	running, answering := daemonVersionAt(addr())
	fmt.Printf("\n%s", versionReport(pluginVersion(), version, running, answering))
}

// pluginVersion is the version of the plugin that ran this command, or "" when it
// was run from a terminal rather than through the plugin.
func pluginVersion() string {
	root := strings.TrimSpace(os.Getenv("CLAUDE_PLUGIN_ROOT"))
	if root == "" {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// versionReport shows the three versions that can disagree for a while after an
// update: the plugin, the binary it fetched, and the daemon still running. Each
// disagreement is followed by what resolves it, so a person checking whether an
// update has landed is told what to do rather than handed three numbers.
func versionReport(plugin, binary, daemon string, answering bool) string {
	var b strings.Builder
	b.WriteString("Versions:\n")
	if plugin != "" {
		fmt.Fprintf(&b, "  plugin  %s\n", plugin)
	}
	fmt.Fprintf(&b, "  binary  %s\n", binary)
	switch {
	case !answering:
		b.WriteString("  daemon  not running\n")
	case daemon == "":
		b.WriteString("  daemon  from before daemons reported a version\n")
	default:
		fmt.Fprintf(&b, "  daemon  %s\n", daemon)
	}
	switch {
	case plugin != "" && shouldReplace(binary, plugin):
		b.WriteString("\nThe binary is older than this plugin expects. A new session fetches it.\n")
	case plugin != "" && olderThan(plugin, binary):
		b.WriteString("\nThis plugin is older than the binary, which is not moved back. Update the plugin in this host, " +
			"and if cogmer is installed in two places, remove the older copy.\n")
	}
	switch {
	case !answering:
		b.WriteString("\nNo daemon is running. A new session starts one.\n")
	case shouldReplace(daemon, binary):
		b.WriteString("\nThe running daemon is not this binary's version. A new session replaces it.\n")
	}
	return b.String()
}

// runVerify is the two-word check (D-048). Both people run it, at the same time,
// on a call where each can recognize the other's voice.
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

	beginCeremony(peerID, name, mine, "", "", terminal)
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
// pairingString is what one person sends another, once, ever.
//
// It carries the sender's chosen name when there is one, so the receiver has a
// sensible default for the label they must supply (D-093) rather than being asked
// to invent one for somebody whose name they obviously know. A GUESSED name is
// never carried: `Ec2-user` traveling as though somebody picked it is worse than
// carrying nothing, because the derived name is at least honest about being
// machine-made (D-097, a pairing string carries a chosen name, never a guessed
// one).
//
// The name is a claim by whoever sent the string, and a substituted string carries
// a substituted name. That is safe here only because the label is written after the
// two words match (D-093), by which point the string came from the person on the
// call. Do not move the write earlier.
func pairingString(peerID, endpoint, name string) string {
	out := peerID
	if endpoint != "" {
		out += "@" + endpoint
	}
	if n := sanitizeName(name); n != "" {
		out += "#" + n
	}
	return out
}

// sanitizeName reduces a name to something usable as a local label.
//
// It becomes a unique key and an argument to `resolvePeer`, and it arrives from
// another machine, so it is trimmed of anything that would break parsing, span
// lines, or be invisible. Length is capped because a label is read in a list.
func sanitizeName(s string) string {
	var b strings.Builder
	space := false
	for _, r := range s {
		switch {
		case r == '#' || r == '@' || r < 0x20 || r == 0x7f:
			continue // would break the format, or is not printable
		case r == ' ' || r == '\t':
			space = true
		case !unicode.IsPrint(r):
			continue // a line or paragraph separator, a direction override, or invisible
		default:
			if space && b.Len() > 0 {
				b.WriteRune(' ')
			}
			space = false
			b.WriteRune(r)
		}
		if b.Len() >= 32 {
			break
		}
	}
	return strings.TrimSpace(b.String())
}

// parsePairing splits a pairing string. The name is optional and older strings do
// not carry one, so its absence is ordinary rather than an error.
func parsePairing(s string) (peerID, endpoint, name string) {
	if hash := strings.Index(s, "#"); hash >= 0 {
		name = sanitizeName(s[hash+1:])
		s = s[:hash]
	}
	if at := strings.LastIndex(s, "@"); at > 0 {
		return s[:at], s[at+1:], name
	}
	return s, "", name
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
	// Re-running the ceremony on a pairing already complete needs the other
	// person present at the same moment, so it is asked for rather than assumed.
	args, again := takeFlag(args, "--again")
	if len(args) == 0 {
		// A prefix names its target, and what this printed was YOU (D-096). Still
		// not an error: somebody here is at the first half of pairing and needs
		// telling what the two halves are, not a usage line.
		fmt.Println("Pairing takes two things: the string your colleague sends you, and a")
		fmt.Println("name for them.")
		fmt.Println()
		fmt.Println(commandText("  /cogmer:peer-pair <their string> <what you call them>"))
		fmt.Println()
		fmt.Println(commandText("Your own string — the one you send THEM — is in /cogmer:self-status."))
		return
	}
	peerID, endpoint, theirName := parsePairing(args[0])

	// Pairing is between users, so it needs this user's identity first (D-230).
	var gated bool
	withMembership(func(_ *Membership, id *Identity) { gated = needsUserIdentity(id) })
	if gated {
		return
	}

	// A name rather than a pairing string: somebody finishing what they started.
	// Pairing is the act people recognize, and verifying is our word for a step
	// inside it, so the same command has to serve both halves — otherwise the
	// second half has no name anybody would guess, and this one silently hashes
	// "alice" into a peer nobody has ever met.
	if _, err := PublicFromPeerID(peerID); err != nil {
		var pid, display, mine, since string
		var done bool
		withMembership(func(m *Membership, id *Identity) {
			resolved, rerr := resolvePeer(m, args[0])
			if rerr != nil {
				log.Fatal(rerr)
			}
			pid, display, mine = resolved, firstNonEmpty(m.Label(resolved), PeerName(resolved)), id.PeerName
			done, since = m.IsVerified(resolved), m.VerifiedAt(resolved)
		})
		if done && !again {
			reportAlreadyPaired(display, since)
			return
		}
		// No label is passed: they already have one, chosen when they were
		// recorded, and this is not the moment to rename anybody.
		beginCeremony(pid, display, mine, "", "", terminal)
		return
	}

	// Asked before anything is named, because somebody already paired needs no
	// name chosen for them and should not be asked about one.
	var alreadyDone bool
	var doneAt, known string
	withMembership(func(m *Membership, _ *Identity) {
		alreadyDone, doneAt = m.IsVerified(peerID), m.VerifiedAt(peerID)
		known = firstNonEmpty(m.Label(peerID), PeerName(peerID))
	})
	if alreadyDone && !again {
		// The address in the string is deliberately ignored. Pairing is a
		// security act and must not quietly write network state, and there is
		// nothing here to repair: an address matters only when it is used, and
		// every use repairs itself. A host who cannot reach a guest falls back to
		// a line the guest pastes, and the guest's first synchronization carries
		// their current address back (D-108).
		reportAlreadyPaired(known, doneAt)
		return
	}

	// What to call them: what this person said, else what THEY said they are
	// called, else ask. Resolved before anything happens, because a flow with two
	// completion points has to answer what an unfinished second one means, and
	// every answer was worse than asking now (D-093).
	name := ""
	if len(args) > 1 {
		name = sanitizeName(args[1])
	}
	switch {
	case name != "":
	case theirName != "":
		// They said what they are called, so asking again would be asking
		// somebody to invent a name for a person whose name they can see.
		name = theirName
		fmt.Printf("Calling them %s, which is the name they gave.\n", name)
		fmt.Printf("%s", commandText("Add a name of your own if you would rather:  /cogmer:peer-pair <their string> <name>\n\n"))
	default:
		fmt.Printf("This needs a name for them as well — what you will call %s in your own\n",
			PeerName(peerID))
		fmt.Printf(commandText("room and peer list:\n\n  /cogmer:peer-pair %s <name>\n\n"), args[0])
		fmt.Println("The derived name above is computed from their key. It identifies them")
		fmt.Println("exactly, and it will mean nothing to you in three weeks.")
		fmt.Println("They did not include a name of their own in what they sent you.")
		return
	}

	var mine string
	withMembership(func(_ *Membership, id *Identity) { mine = id.PeerName })
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
	beginCeremony(peerID, PeerName(peerID), mine, name, endpoint, terminal)
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
// label is what this person calls the peer, and is empty when re-verifying one they
// already named. It is carried to the daemon and written only if the words match,
// so abandoning leaves nothing claimed (D-093).
func beginCeremony(peerID, name, mine, label, endpoint string, preferTerminal bool) {
	// BOTH paths need the daemon: the terminal one posts to /verify/start just as
	// the page does. Falling back to it when the daemon is down produced a second
	// failure with a different message, which reads as two problems rather than
	// one. Say the one true thing instead.
	if !daemonAlreadyServing(addr()) {
		log.Fatal(commandText("the cogmer daemon is not running, and pairing needs it.\n" +
			"  It starts with an agent session, and /cogmer:self-restart starts it without one."))
	}
	// One pairing record for both surfaces. What a match, a mismatch and an
	// abandonment each mean is then decided in one place rather than twice.
	var res pairNewResponse
	if err := postLocal("/pair/new", pairNewRequest{Peer: peerID, Name: label, Endpoint: endpoint}, &res); err != nil {
		log.Fatalf("pair: %v", err)
	}
	if res.Error != "" {
		fmt.Println(res.Error)
		return
	}
	if !preferTerminal {
		if err := openInBrowser(res.URL); err != nil {
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
	verifyWith(pairIDFromURL(res.URL), peerID, name)
}

// pairIDFromURL takes the id back off the URL the daemon minted, so the terminal
// path confirms against the same pairing the page would have.
func pairIDFromURL(u string) string {
	if i := strings.LastIndex(u, "/pair/"); i >= 0 {
		return u[i+len("/pair/"):]
	}
	return ""
}

// verifyWith drives the ceremony and records the answer. Shared by `pair` and
// `verify`: the first is a first meeting, the second is confirming a key recorded
// some other way, and the ceremony is identical.
func verifyWith(pairID, peerID, name string) {
	var out verifyStartResponse
	if err := postLocal("/verify/start", verifyStartRequest{PairID: pairID, Peer: peerID}, &out); err != nil {
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
	if err := postLocal("/verify/confirm", verifyConfirmRequest{PairID: pairID, Peer: peerID, Matched: matched}, &res); err != nil {
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
	_, ok := daemonVersionAt(address)
	return ok
}

// daemonVersionAt reports the version a cogmer daemon at an address says it runs,
// and whether a cogmer daemon answered at all. A daemon built before /healthz
// carried a version answers with none, which reads as "".
func daemonVersionAt(address string) (string, bool) {
	c := &http.Client{Timeout: 700 * time.Millisecond}
	resp, err := c.Get("http://" + address + "/healthz")
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	var body struct {
		OK      bool   `json:"ok"`
		PeerID  string `json:"peerId"`
		Version string `json:"version"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&body); err != nil {
		return "", false
	}
	return body.Version, body.OK && body.PeerID != ""
}

// shouldReplace says whether something of version running gives way to a binary of version
// mine: only an older one does, and one that reports no version, as a daemon built before
// /healthz carried one does (D-158, D-215). Having no version is a difference, and a difference
// is replaced when there is no order to go by. A plugin that is older than the program on the
// machine is an old copy, such as one a desktop app installed, and must not move the machine
// back. A build from source reports dev and is never replaced, nor replaces one, because a
// maintainer runs it by hand and every new session would otherwise end it. A version that is
// not dotted numbers is replaced when it differs, since there is no order to go by.
func shouldReplace(running, mine string) bool {
	if running == "dev" || mine == "dev" {
		return false
	}
	a, b := versionNumbers(running), versionNumbers(mine)
	if a == nil || b == nil {
		return running != mine
	}
	return versionLess(a, b)
}

// olderThan says whether a is a version older than b, when both are dotted numbers.
func olderThan(a, b string) bool {
	x, y := versionNumbers(a), versionNumbers(b)
	return x != nil && y != nil && versionLess(x, y)
}

// versionNumbers reads a version made of dotted numbers, such as 0.10.2, and returns nil for
// anything else.
func versionNumbers(v string) []int {
	if v == "" {
		return nil
	}
	var out []int
	for _, part := range strings.Split(v, ".") {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || part != strconv.Itoa(n) {
			return nil
		}
		out = append(out, n)
	}
	return out
}

// versionLess orders two versions number by number, so 0.9.1 is older than 0.10.0. A version
// with fewer numbers is read as if the rest were zero.
func versionLess(a, b []int) bool {
	for i := 0; i < len(a) || i < len(b); i++ {
		var x, y int
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			return x < y
		}
	}
	return false
}

// stopOutcome is what stopping found at one listening process.
type stopOutcome struct {
	PID     int
	Name    string
	Stopped bool
	Err     error
}

// stopDaemonsAt stops the cogmer daemon listening on each address, and declines
// anything else listening there (D-123). The daemon in the way is the one holding
// the address, whichever state directory it belongs to.
func stopDaemonsAt(addresses []string) ([]stopOutcome, error) {
	seen := map[int]bool{os.Getpid(): true}
	var out []stopOutcome
	for _, a := range addresses {
		pids, err := listeningPIDs(portOf(a))
		if err != nil {
			return out, err
		}
		for _, pid := range pids {
			if seen[pid] {
				continue
			}
			seen[pid] = true
			o := stopOutcome{PID: pid, Name: processName(pid)}
			// Identified by its executable's name: a build renamed by hand is then
			// not stopped, which is the safe way for this check to be wrong.
			if isCogmerImage(o.Name) {
				o.Err = terminate(pid, 5*time.Second)
				o.Stopped = o.Err == nil
			}
			out = append(out, o)
		}
	}
	return out, nil
}

// runStop is `cogmer stop`. With --for-restart it does not say that nothing runs
// afterwards, because the command that passes it starts a daemon next (D-195).
func runStop(args []string) {
	_, forRestart := takeFlag(args, "--for-restart")
	outcomes, err := stopDaemonsAt([]string{addr(), peerAddr()})
	for _, o := range outcomes {
		switch {
		case o.Stopped:
			fmt.Printf("stopped the cogmer daemon, pid %d\n", o.PID)
		case o.Err != nil:
			fmt.Printf("could not stop the cogmer daemon, pid %d: %v\n", o.PID, o.Err)
		default:
			fmt.Printf("left %s, pid %d, alone: it is not a cogmer daemon\n", firstNonEmpty(o.Name, "an unnamed process"), o.PID)
		}
	}
	if err != nil {
		log.Fatalf("stop: %v", err)
	}
	if len(outcomes) == 0 {
		fmt.Printf("nothing is listening on %s or %s\n", addr(), peerAddr())
	}
	if !forRestart {
		localDaemon{}.SetStopped(true)
	}
	for _, o := range outcomes {
		if o.Stopped && !forRestart {
			// Said here because the command that runs this is typed in a session,
			// and nothing restarts a daemon until a session starts (D-150).
			fmt.Println("No daemon is running now, so nothing is captured or shared until one starts.")
			fmt.Println("A new agent session starts one. This command does not.")
			break
		}
	}
}

// holder is a process listening on an address that this daemon needs.
type holder struct {
	PID  int
	Name string
}

// whoHolds names the processes listening on an address, other than this one. A
// process of another user may not be visible, so an empty answer is not proof that
// nothing holds the address.
func whoHolds(address string) ([]holder, error) {
	pids, err := listeningPIDs(portOf(address))
	if err != nil {
		return nil, err
	}
	var out []holder
	for _, pid := range pids {
		if pid != os.Getpid() {
			out = append(out, holder{PID: pid, Name: processName(pid)})
		}
	}
	return out, nil
}

// describeBlock says what holds an address and what to do about it. The holder is
// found by asking the operating system, as `stop` does (D-123), because the
// address alone cannot say whether the thing in the way is a cogmer daemon left
// running from another state directory or an unrelated program, and the two need
// different responses.
func describeBlock(what, address string, cause error, holders []holder, findErr error, stop string) (detail string, advice []string) {
	move := "Or move this daemon: COGMER_ADDR=127.0.0.1:<port> (hooks and view), COGMER_PEER_ADDR (peer sync)"
	for _, h := range holders {
		if isCogmerImage(h.Name) {
			return fmt.Sprintf("%s cannot be served: %s is held by another cogmer daemon, pid %d (%v)", what, address, h.PID, cause),
				[]string{"Stop it:  " + stop + " stop", move}
		}
	}
	if len(holders) > 0 {
		h := holders[0]
		return fmt.Sprintf("%s cannot be served: %s is held by %s, pid %d, which is not a cogmer daemon (%v)",
				what, address, firstNonEmpty(h.Name, "a process with no name"), h.PID, cause),
			[]string{"`" + stop + " stop` leaves a process that is not a cogmer daemon alone, so end it yourself.", move}
	}
	find := "lsof -nP -iTCP:" + portOf(address) + " -sTCP:LISTEN"
	if runtime.GOOS == "windows" {
		find = "netstat -ano | findstr :" + portOf(address)
	}
	why := "this machine could not say what holds it"
	if findErr != nil {
		why = fmt.Sprintf("this machine could not say what holds it: %v", findErr)
	}
	return fmt.Sprintf("%s cannot be served: %s is in use, and %s (%v)", what, address, why, cause),
		[]string{"Find what holds it:  " + find, move}
}

func reportDaemonBlocked(what, address string, cause error) {
	holders, findErr := whoHolds(address)
	detail, advice := describeBlock(what, address, cause, holders, findErr, invocation())
	recordDaemonState("blocked", detail)
	log.Printf("%s", detail)
	log.Printf("  Nothing is captured or shared until that address is free.")
	for _, line := range advice {
		log.Printf("  %s", line)
	}
}

// recordDaemonState mirrors the install-state format -- state, when, detail -- so
// the shell side has one shape to read rather than two.
func recordDaemonState(state, detail string) {
	_ = ensureStateDir()
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
// Not the bare word "cogmer". The plugin installs into ~/.cogmer/bin,
// which is on nobody's PATH, and §29 forbids editing a shell profile to put it
// there. So every line of ours that said "run cogmer pair" was a line that
// fails with "command not found" for every plugin-installed user -- which is most
// of them, and precisely the ones least equipped to work out why. The binary knows
// where it is; a path it prints about itself cannot go stale.
//
// $HOME is spelled ~ because that is what a person recognizes, and it pastes into
// any shell unchanged.
func invocation() string {
	exe, err := os.Executable()
	if err != nil {
		return "cogmer"
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
	writePairingInvitation(os.Stdout, id, AdvertisedEndpoint())
}

// writePairingInvitation prints what a user sends a colleague, or says why there is
// nothing to send. A string whose address nobody can reach is not printed at all: a
// loopback address in a string that is copied and sent cannot complete a pairing,
// and a note after it still leaves a string to copy.
func writePairingInvitation(w io.Writer, id *Identity, endpoint string) {
	if !id.NameChosen {
		// Said here because this is the moment the name starts traveling: all
		// three paths that hand your identity to somebody else come through this
		// function (D-089 consolidated them), and the name is seen by nobody but
		// other people.
		fmt.Fprintf(w, "\n%s\n", nameLine(id))
	}
	ok, why, note := checkReachable(endpoint)
	if !ok {
		fmt.Fprintf(w, "\nThere is no string to send yet: %s\n", why)
		return
	}
	fmt.Fprintf(w, "\nSend your colleague this — any channel will do, it is not a secret:\n\n  %s\n",
		pairingString(id.Peer(), endpoint, chosenName(id)))
	if note != "" {
		fmt.Fprintf(w, "\n%s\n", note)
	}
	fmt.Fprintf(w, "%s", commandText("\nWhen they send you theirs, run:\n\n  /cogmer:peer-pair <their string> <what you call them>\n"))
	fmt.Fprintf(w, "\nThe name is yours and is used everywhere you see them. Their key already\n")
	fmt.Fprintf(w, "gives them a name, but it is a word pair computed from the key and it will\n")
	fmt.Fprintf(w, "mean nothing to you in three weeks.\n")
	fmt.Fprintf(w, "\nA page then opens in your browser showing two words. Get on a call, both of\n")
	fmt.Fprintf(w, "you do this at the same time, and read the words to each other. They must\n")
	fmt.Fprintf(w, "match. Nothing is recorded until they do.\n")
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
	ok, why, _ := checkReachable(endpoint)
	return ok, why
}

// queryDaemonSelf asks the running daemon whether it can be reached through its
// relay. A variable so that a test can answer for the daemon.
var queryDaemonSelf = fetchSelfCheck

// fetchSelfCheck asks the daemon over its own address. The second result is false
// when no daemon answered. The wait covers a check that has to time out.
func fetchSelfCheck() (SelfCheck, bool) {
	c := &http.Client{Timeout: 12 * time.Second}
	resp, err := c.Get("http://" + addr() + "/self")
	if err != nil {
		return SelfCheck{}, false
	}
	defer resp.Body.Close()
	var sc SelfCheck
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&sc) != nil {
		return SelfCheck{}, false
	}
	return sc, true
}

// checkReachable is pairingReachable with the sentence to print when the answer is
// yes. An overlay address is not reachable because it has the right shape. It is
// reachable when the daemon is attached to its relay and a connection made from
// outside gets through, and only the running daemon can try (D-196).
// failingSentence says how long the daemon has been failing the check, when it has
// been for long enough to have a number.
func failingSentence(sc SelfCheck) string {
	if sc.FailingFor == "" {
		return ""
	}
	return "\n      It has failed this check for " + sc.FailingFor + "."
}

func checkReachable(endpoint string) (ok bool, why, note string) {
	e, err := ParseEndpoint(endpoint)
	if err != nil {
		return false, "that address cannot be read, so nobody can reach you at it.", ""
	}
	if e.Scheme == schemeTailcat {
		sc, up := queryDaemonSelf()
		switch {
		case !up:
			if detail, blocked := daemonBlockedReason(); blocked {
				return false, "the daemon could not start, so nobody can reach you.\n      " + detail, ""
			}
			return false, "the daemon is not running, so nobody can reach you at that address.\n" +
				commandText("      A new agent session starts one, and so does /cogmer:self-restart."), ""
		case !sc.Overlay:
			return false, "this daemon is not running the relay connection, so the address in your string\n" +
				commandText("      leads nowhere. /cogmer:self-restart starts the daemon again."), ""
		case sc.Relay == "":
			return false, "the daemon is running but is not attached to a relay, so colleagues cannot reach you.\n" +
				commandText("      /cogmer:self-restart restarts it, and ~/.cogmer/daemon.log says why if that does not help."), ""
		case !sc.Reached:
			return false, fmt.Sprintf("the daemon is attached to the relay (%s) but a test connection from outside\n"+
				"      did not get through (%s), so colleagues cannot reach you at that address.\n"+
				commandText("      /cogmer:self-restart restarts the daemon.%s"), sc.Relay, shortReason(sc.Error), failingSentence(sc)), ""
		}
		return true, "", fmt.Sprintf("Checked just now: a connection from outside reached this machine through the relay at %s, in %dms.", sc.Relay, sc.TookMs)
	}
	// A bare TCP endpoint has to be a host and a port. ParseEndpoint accepts
	// anything without a scheme as TCP, and isLoopback answers "false" for what it
	// cannot parse -- so without this, a malformed address reads as "not loopback"
	// and therefore as fine.
	if _, _, err := net.SplitHostPort(e.Value); err != nil {
		return false, "that address is not a host and port, so nobody can reach you at it.", ""
	}
	if isLoopback(e.Value) {
		// A daemon that could not bind wrote why, and that is the cause, so it is said
		// before the guess that no session has started.
		if detail, blocked := daemonBlockedReason(); blocked {
			return false, "the daemon could not start, so no address was published.\n      " + detail, ""
		}
		if !endpointRecorded() {
			return false, "no daemon has published an address yet, so that one is a guess.\n" +
				"      The daemon records a real one when it starts, which happens when a\n" +
				"      agent session starts. Start one and run this again.", ""
		}
		return false, "that address is loopback, so nobody else can reach it. No route out\n" +
			"      was negotiated; set COGMER_PEER_ADDR to an address they can reach\n" +
			"      and restart the daemon.", ""
	}
	return true, "", ""
}

// daemonBlockedReason is what the daemon wrote when it could not start, whether it
// could not bind its address (D-177) or could not bring the data to its version.
func daemonBlockedReason() (string, bool) {
	raw, err := os.ReadFile(daemonStateFile())
	if err != nil {
		return "", false
	}
	f := strings.SplitN(strings.TrimSpace(string(raw)), "\t", 3)
	if len(f) == 3 && (f[0] == "blocked" || f[0] == upgradeFailedState) {
		return f[2], true
	}
	return "", false
}

// endpointRecorded distinguishes "a daemon published this" from "we guessed".
func endpointRecorded() bool {
	_, err := os.Stat(filepath.Join(homeDir(), endpointFile))
	return err == nil
}

// nameLine says what other people see, and — only while nothing has chosen it —
// how to change it.
//
// Both, not one or the other. The name is the literal answer to "who am I", so it
// is always shown; saying "you chose this" in the other case is noise about
// something the person already knows. The offer appears exactly while it is true
// and stops at the first deliberate act (D-186).
func nameLine(id *Identity) string {
	if id.NameChosen {
		return fmt.Sprintf("Other people see you as %s.", id.UserDisplayName)
	}
	return fmt.Sprintf("Other people see you as %s — that is this computer's username,\n"+
		commandText("not a name anybody picked. /cogmer:self-name changes it."), id.UserDisplayName)
}

// runName shows or sets what other people call you.
//
// Self scope, not peer scope: `peer-` is for commands about somebody else, and a
// command that sets your own name has no business in that namespace (D-096).
func runName(args []string) {
	if len(args) == 0 {
		withMembership(func(_ *Membership, id *Identity) {
			fmt.Println(nameLine(id))
		})
		return
	}
	id, err := SetDisplayName(strings.Join(args, " "))
	if err != nil {
		log.Fatalf("name: %v", err)
	}
	fmt.Printf("Other people will see you as %s.\n", id.UserDisplayName)
	fmt.Println("Turns you have already sent keep the name they were sent with, and a")
	fmt.Println("colleague who gave you a name of their own still sees theirs.")
}

// chosenName is the name to publish: none at all while it is only a guess.
func chosenName(id *Identity) string {
	if !id.NameChosen {
		return ""
	}
	return id.UserDisplayName
}

// cut10 shortens a timestamp to its date, tolerating one that is absent.
func cut10(s string) string {
	if len(s) < 10 {
		return ""
	}
	return s[:10]
}

// reportAlreadyPaired answers the commonest repeat of this command: somebody
// checking, or arriving here twice.
//
// Running the ceremony again is not harmless and must not be the default. It needs
// the other person at their machine at the same moment, so an unrequested one
// leaves somebody watching a page wait ninety seconds for a colleague who was
// never asked.
func reportAlreadyPaired(who, since string) {
	if since != "" && len(since) >= 10 {
		fmt.Printf("You and %s are already paired — since %s.\n", who, since[:10])
	} else {
		fmt.Printf("You and %s are already paired.\n", who)
	}
	fmt.Printf("\nNothing to do. Pairing holds for every room you ever share.\n")
	fmt.Printf("\nIf you have reason to think their key changed — they said two words that\n")
	fmt.Printf("did not match yours, or you were told to check — compare again with:\n\n")
	fmt.Printf(commandText("  /cogmer:peer-pair %s --again\n"), who)
}
