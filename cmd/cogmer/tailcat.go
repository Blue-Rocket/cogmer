package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"tailscale.com/tailcfg"
	"time"

	"github.com/tailscale/tailcat"
	"tailscale.com/types/key"
	"tailscale.com/wgengine/filter"
)

// Reaching a peer on another network (§4).
//
// Two people working from home are behind two routers and neither can bind an
// address the other can reach. An SSH tunnel works around it only by a person
// performing NAT traversal by hand, which for a pair who pair daily is the product
// failing at its first step (D-068).
//
// Tailcat is Tailscale's data plane -- WireGuard, hole punching, DERP -- with no
// control plane, no account and no tailnet. A peer publishes an address, the other
// dials it, DERP carries the introduction, and the path upgrades to direct UDP
// when it can.
//
// **Connectivity is near-certain and directness is not.** Any router permits
// outbound connections, which is what a relay needs, so the floor is a working
// path with relay latency added. What is uncertain is how often the upgrade to
// direct succeeds through consumer routers, and that is a question about latency
// rather than correctness. For turns of about a kilobyte it is unlikely to matter.
//
// It sits UNDER everything and decides nothing (D-019, D-068). A tailcat
// connection reaches the door. A signed request (D-044), the guest list (D-045)
// and a person's verification (D-054) decide whether anyone comes in -- unchanged,
// and applied to this connection exactly as to a TCP one, because both arrive at
// the same handler through the same listener interface.
//
// Two identities, two jobs, never to be conflated: tailcat holds a WireGuard key
// for its tunnel, and a peerId is an Ed25519 key that signs events and is what two
// people compare two words against (D-042).
//
// Confined to this file on purpose. Tailcat promises no API stability -- "the Go
// API, CLI flags and output, and wire format may all change" -- so what that buys
// is that a breaking change has one place to land.

// tailcatPort is the port inside the tunnel. It is not an address on any machine
// and never leaves one, so a fixed number is fine and one fewer thing to publish.
const tailcatPort = 4783

const tailcatRegionFile = "tailcat-region.json"

// A cloud session carries its relay connection as a WebSocket (D-217).
//
// A Claude Code cloud session sends everything through an HTTPS proxy that passes
// ordinary requests and breaks the relay's own protocol upgrade. From one, the relay
// map and the relay hosts answered and a tunnel timed out, and the same tunnel carried
// as a WebSocket reached a daemon on another machine (e415f1d). Tailscale's relay
// client offers WebSocket behind the ts_debug_websockets build tag, which
// scripts/release.sh sets, and behind this variable, which is read at each connection
// and so is set here, before any is made. Outside a cloud session nothing changes, and
// a value the user set is left alone.
const relayWebSocketVar = "TS_DEBUG_DERP_WS_CLIENT"

func init() { relayAsWebSocketInCloud(os.Getenv, os.Setenv) }

// relayAsWebSocketInCloud sets relayWebSocketVar in a cloud session where nothing has
// set it, and reports whether it did.
func relayAsWebSocketInCloud(getenv func(string) string, setenv func(string, string) error) bool {
	if getenv("CLAUDE_CODE_REMOTE") != "true" || getenv(relayWebSocketVar) != "" {
		return false
	}
	return setenv(relayWebSocketVar, "1") == nil
}

// relayMode says how the relay connection is carried, for the daemon's log, so that a
// cloud session's log shows whether the switch above took effect.
func relayMode() string {
	if v := os.Getenv(relayWebSocketVar); v != "" {
		return fmt.Sprintf("as a WebSocket (%s=%s; it takes effect only in a build with the ts_debug_websockets tag)", relayWebSocketVar, v)
	}
	return "as the relay's own protocol"
}

const tailcatKeyFile = "tailcat.key"

// tailcatEnabled reports whether to offer a tailcat endpoint. On unless refused:
// the case it serves is the one that exists, and a peer that does not need it
// pays for a listener nobody dials.
func tailcatEnabled() bool {
	return !strings.EqualFold(strings.TrimSpace(os.Getenv("COGMER_TAILCAT")), "off")
}

func init() { RegisterDialer(schemeTailcat, &tailcatDialer{}) }

// tailcatDialer keeps one client per peer, for as long as the daemon runs.
//
// A client is not a connection. Constructing one starts a userspace WireGuard
// stack, probes the network, chooses a DERP region and completes a handshake --
// seconds of work. Doing that per dial made every sync round pay first-contact
// cost, which at a one-second poll meant it never finished paying: the first
// attempt was still handshaking when the next began. Observed as every sync
// timing out while the peer was plainly reachable.
//
// Cached, the cost is paid once and subsequent dials ride the established tunnel.
type tailcatDialer struct {
	mu      sync.Mutex
	clients map[string]*tailcat.Client
}

func (d *tailcatDialer) clientFor(addr string) *tailcat.Client {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.clients == nil {
		d.clients = map[string]*tailcat.Client{}
	}
	if c, ok := d.clients[addr]; ok {
		return c
	}
	c := tailcat.NewClient(tailcat.Addr(addr))
	d.clients[addr] = c
	return c
}

func (d *tailcatDialer) Dial(ctx context.Context, e Endpoint) (net.Conn, error) {
	conn, err := d.clientFor(e.Value).DialTCPPort(ctx, tailcatPort)
	if err != nil {
		return nil, fmt.Errorf("tailcat: %w", err)
	}
	return conn, nil
}

// Reset discards the client held for an address. A client is built once and kept (see above),
// so one that has got into a state it cannot leave would fail for as long as the daemon runs.
func (d *tailcatDialer) Reset(e Endpoint) bool {
	d.mu.Lock()
	c, ok := d.clients[e.Value]
	delete(d.clients, e.Value)
	d.mu.Unlock()
	if ok {
		go func() { _ = c.Close() }()
	}
	return ok
}

// pingOutcome is one relay ping: how long it took, or why it failed.
type pingOutcome struct {
	Took time.Duration
	Err  error
}

// tailcatDiagnosis is what a diagnosis found.
type tailcatDiagnosis struct {
	Region     string      // the relay the address pins, as the address names it
	Held       pingOutcome // a relay ping on the connection this daemon holds
	Fresh      pingOutcome // a relay ping on a new connection
	Path       string      // what a path-discovery ping found, or why it failed
	HeldAbsent bool        // no connection was held, so the first is the same as the second
}

func (o pingOutcome) String() string {
	if o.Err != nil {
		return fmt.Sprintf("failed after %s (%v)", o.Took.Round(100*time.Millisecond), o.Err)
	}
	return fmt.Sprintf("answered in %s", o.Took.Round(time.Millisecond))
}

// describe writes a diagnosis as one line of facts and one of what they suggest. The reading
// is an inference from the three pings, and says so.
func (t tailcatDiagnosis) describe() string {
	var b strings.Builder
	fmt.Fprintf(&b, "the address pins %s. Relay ping, connection held by this daemon: %s. Relay ping, a new connection: %s. Path: %s.",
		t.Region, t.Held, t.Fresh, t.Path)
	var reading string
	switch {
	case t.Held.Err != nil && t.Fresh.Err == nil:
		reading = "a new connection reaches the peer and the one held does not, so the held one is stuck and is replaced"
	case t.Held.Err != nil && t.Fresh.Err != nil:
		reading = "the peer does not answer through that relay, which means it is not running, is attached to another relay, or its replies do not return"
	case t.Held.Err == nil && t.Fresh.Err != nil:
		reading = "the held connection reaches the peer and a new one does not, which is unusual and worth repeating"
	default:
		reading = "the relay path works, so the failure is above it, in the tunnel's TCP connection or the peer's server"
	}
	fmt.Fprintf(&b, " Reading: %s.", reading)
	return b.String()
}

// Diagnose pings the peer three ways at once, so that the answer takes as long as the
// slowest and not their sum: through the relay on the connection held, through the relay on a
// new connection, and by path discovery, which says whether traffic would go direct.
func (d *tailcatDialer) Diagnose(ctx context.Context, e Endpoint) string {
	diag := tailcatDiagnosis{Region: "no relay that could be read from it"}
	if info, err := tailcat.ParseAddr(tailcat.Addr(e.Value)); err != nil {
		diag.Region = fmt.Sprintf("an address that could not be read (%v)", err)
	} else if len(info.Region) > 0 && info.Region[0] != nil {
		diag.Region = fmt.Sprintf("relay region %d (%s)", info.Region[0].RegionID, info.Region[0].RegionCode)
	} else if info.RegionID != 0 {
		diag.Region = fmt.Sprintf("relay region %d", info.RegionID)
	}
	held := d.clientFor(e.Value)
	fresh := tailcat.NewClient(tailcat.Addr(e.Value))
	defer func() { go func() { _ = fresh.Close() }() }()

	ping := func(c *tailcat.Client) pingOutcome {
		start := time.Now()
		res, err := c.Ping(ctx)
		if err != nil {
			return pingOutcome{Took: time.Since(start), Err: err}
		}
		return pingOutcome{Took: res.Latency}
	}
	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); diag.Held = ping(held) }()
	go func() { defer wg.Done(); diag.Fresh = ping(fresh) }()
	go func() {
		defer wg.Done()
		res, err := held.DiscoPing(ctx)
		switch {
		case err != nil:
			diag.Path = fmt.Sprintf("no answer (%v)", err)
		case res.Err != "":
			diag.Path = fmt.Sprintf("no answer (%s)", res.Err)
		case res.Endpoint != "":
			diag.Path = fmt.Sprintf("direct, to %s in %.0fms", res.Endpoint, res.LatencySeconds*1000)
		default:
			diag.Path = fmt.Sprintf("through relay region %d (%s) in %.0fms", res.DERPRegionID, res.DERPRegionCode, res.LatencySeconds*1000)
		}
	}()
	wg.Wait()
	return diag.describe()
}

// loadTailcatKey keeps the tunnel key across restarts, so that a peer's published
// address stays the one others recorded.
//
// Separate from identity.key, which holds the peer identity. A single file for
// both would invite treating them as one thing, and they are not: losing this one
// costs a stale endpoint, and losing that one costs the identity every event was
// signed with.
func loadTailcatKey() (key.NodePrivate, error) {
	path := filepath.Join(homeDir(), tailcatKeyFile)
	if b, err := os.ReadFile(path); err == nil {
		var k key.NodePrivate
		if err := k.UnmarshalText([]byte(strings.TrimSpace(string(b)))); err == nil {
			return k, nil
		}
		// Unreadable rather than absent. Minting a new one changes the published
		// address, which is recoverable, where refusing to start is not.
	}

	k := tailcat.NewPrivateKey().Private
	text, err := k.MarshalText()
	if err != nil {
		return k, err
	}
	if err := os.MkdirAll(homeDir(), 0o700); err != nil {
		return k, err
	}
	// 0600, like identity.key: holding it is holding the tunnel.
	if err := os.WriteFile(path, append(text, '\n'), 0o600); err != nil {
		return k, err
	}
	return k, nil
}

// connListener adapts tailcat's callback shape to a net.Listener, so that one
// http.Serve over one set of routes serves both transports. Nothing about a
// request should depend on which carried it, and this is what makes that true
// rather than merely intended.
type connListener struct {
	conns  chan net.Conn
	closed chan struct{}
	once   sync.Once
	addr   net.Addr
}

func (l *connListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *connListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return nil
}

func (l *connListener) Addr() net.Addr { return l.addr }

type tailcatAddr string

func (a tailcatAddr) Network() string { return schemeTailcat }
func (a tailcatAddr) String() string  { return string(a) }

// StartTailcat begins listening and returns the endpoint to advertise.
func StartTailcat() (net.Listener, string, *tailcat.Server, error) {
	k, err := loadTailcatKey()
	if err != nil {
		return nil, "", nil, err
	}
	l := &connListener{conns: make(chan net.Conn), closed: make(chan struct{})}

	region, rerr := loadTailcatRegion()
	if rerr != nil {
		return nil, "", nil, rerr
	}
	s := &tailcat.Server{
		Key:    k,
		Region: region,
		// No pre-shared key, so the published address contains no secret and is
		// identical on every start (D-104). With one, the library mints a fresh
		// key at each Start and the address changes, which silently invalidates
		// every pairing string and invitation this machine has issued. The key
		// also made the address confidential, and §12 requires it to be
		// pasteable — the two cannot both hold, because the key travels inside
		// the thing that must be published.
		DisablePresharedKey: true,
		// Any dialer may open a tunnel, because a dialer's key is generated at its
		// first use and no colleague's pairing string can carry it. The TLS pin
		// refuses a key this machine has not recorded before a request is read
		// (D-157), so a stranger gets as far as the handshake and no further.
		// Silenced: tailcat narrates its startup at a volume suited to a CLI, and
		// this is a daemon whose log a person reads to learn about peers.
		Logf: func(string, ...any) {},
		// Two gates, because they fail differently and the difference matters when
		// something is wrong. The packet filter drops without a reply, so a client
		// dialing anything else times out; OnTCP returning nil sends a reset, which
		// says "not here" promptly.
		ServedTCPPorts: []filter.PortRange{{First: tailcatPort, Last: tailcatPort}},
		OnTCP: func(port uint16) func(net.Conn) {
			if port != tailcatPort {
				return nil
			}
			return func(c net.Conn) {
				select {
				case l.conns <- c:
				case <-l.closed:
					c.Close()
				}
			}
		},
	}
	if err := s.Start(); err != nil {
		return nil, "", nil, err
	}
	addr := string(s.TailcatAddr())
	if addr == "" {
		s.Close()
		return nil, "", nil, errors.New("tailcat started but published no address")
	}
	l.addr = tailcatAddr(addr)
	return l, Endpoint{Scheme: schemeTailcat, Value: addr}.String(), s, nil
}

// loadTailcatRegion pins the relay this node publishes.
//
// The address names a rendezvous: where this node can be found so an introduction
// can happen, not where it is. Left unpinned the library re-chooses by latency at
// every start, with a random fallback when the probe fails, so the published
// address could differ after a restart on another network — and a changed address
// strands everybody holding the old one (D-187).
//
// Pinned, a machine that travels keeps an address its colleagues can still use, at
// the cost of a relay that may no longer be the nearest. That is the right trade:
// the relay carries an introduction and the path upgrades to direct afterwards, so
// the penalty is a slower handshake rather than a slower session.
func loadTailcatRegion() (*tailcfg.DERPRegion, error) {
	path := filepath.Join(homeDir(), tailcatRegionFile)
	if b, err := os.ReadFile(path); err == nil {
		var r tailcfg.DERPRegion
		if json.Unmarshal(b, &r) == nil && r.RegionID != 0 {
			return &r, nil
		}
	}
	ci := &tailcat.ConnInfo{RegionID: -1}
	if err := ci.Expand(context.Background(), tailcat.ExpandForServer); err != nil {
		return nil, err
	}
	if len(ci.Region) == 0 {
		return nil, errors.New("no relay region could be chosen")
	}
	r := ci.Region[0]
	if buf, err := json.MarshalIndent(r, "", "  "); err == nil {
		_ = os.MkdirAll(homeDir(), 0o700)
		_ = os.WriteFile(path, buf, 0o600)
	}
	return r, nil
}

// SelfCheck says whether this daemon can be reached through its relay. The pairing
// string carries an overlay address whatever state the server is in, so without
// asking, a daemon that never attached to its relay would hand out an address that
// leads nowhere and report it as reachable (D-196).
type SelfCheck struct {
	// Overlay is false when this daemon runs no overlay server.
	Overlay bool `json:"overlay"`
	// Relay is the region the server is attached to, empty until it is.
	Relay string `json:"relay,omitempty"`
	// Reached is whether a connection made from outside, through the relay, got to
	// this server's port.
	Reached bool   `json:"reached"`
	TookMs  int64  `json:"tookMs,omitempty"`
	Error   string `json:"error,omitempty"`
	// FailingFor is how long the daemon has been failing this check, empty while it
	// passes. The daemon fills it in, since only it remembers earlier answers.
	FailingFor string `json:"failingFor,omitempty"`
}

// newRelayProbe returns what the server reports as its relay right now, which is
// empty when it is not attached. It opens no connection, so it is cheap enough to
// ask every minute.
func newRelayProbe(srv *tailcat.Server) func() string {
	return func() string { return srv.Status().Self.Relay }
}

// newSelfCheck returns the check for a running server. It dials the server's own
// published address with a client of its own, which is what a colleague does, so a
// pass means a colleague can get this far. The dial opens a TCP connection and
// closes it, and sends nothing.
func newSelfCheck(srv *tailcat.Server, endpoint string) func(context.Context) SelfCheck {
	return func(ctx context.Context) SelfCheck {
		sc := SelfCheck{Overlay: true}
		// The attachment takes a few seconds after the server starts, and the check
		// waits for it as long as it is allowed to, since a daemon that was just
		// started has not failed to attach.
		for {
			if sc.Relay = srv.Status().Self.Relay; sc.Relay != "" {
				break
			}
			select {
			case <-ctx.Done():
				sc.Error = "not attached to a relay"
				return sc
			case <-time.After(100 * time.Millisecond):
			}
		}
		e, err := ParseEndpoint(endpoint)
		if err != nil {
			sc.Error = err.Error()
			return sc
		}
		start := time.Now()
		c := tailcat.NewClient(tailcat.Addr(e.Value))
		defer c.Close()
		conn, err := c.DialTCPPort(ctx, tailcatPort)
		sc.TookMs = time.Since(start).Milliseconds()
		if err != nil {
			sc.Error = err.Error()
			return sc
		}
		_ = conn.Close()
		sc.Reached = true
		return sc
	}
}
