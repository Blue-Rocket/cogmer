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

	"github.com/tailscale/tailcat"
	"tailscale.com/types/key"
	"tailscale.com/wgengine/filter"
)

// Reaching a peer on another network (§15).
//
// Two people working from home are behind two routers and neither can bind an
// address the other can reach. Phases 2 and 5 papered over this with an SSH
// tunnel, which is a person performing NAT traversal by hand; for a pair who pair
// daily that is the product failing at its first step (D-063).
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
// It sits UNDER everything and decides nothing (D-019, D-062). A tailcat
// connection reaches the door. A signed request (D-044), the guest list (D-045)
// and a person's verification (D-054) decide whether anyone comes in -- unchanged,
// and applied to this connection exactly as to a TCP one, because both arrive at
// the same handler through the same listener interface.
//
// Two identities, two jobs, never to be conflated: tailcat holds a WireGuard key
// for its tunnel, and a peerId is an Ed25519 key that signs events and is what two
// people compare two words against (D-020).
//
// Confined to this file on purpose. Tailcat promises no API stability -- "the Go
// API, CLI flags and output, and wire format may all change" -- so what that buys
// is that a breaking change has one place to land.

// tailcatPort is the port inside the tunnel. It is not an address on any machine
// and never leaves one, so a fixed number is fine and one fewer thing to publish.
const tailcatPort = 4783

const tailcatRegionFile = "tailcat-region.json"

const tailcatKeyFile = "tailcat.key"

// tailcatEnabled reports whether to offer a tailcat endpoint. On unless refused:
// the case it serves is the one that exists, and a peer that does not need it
// pays for a listener nobody dials.
func tailcatEnabled() bool {
	return !strings.EqualFold(strings.TrimSpace(os.Getenv("CLAUDE_TEAM_TAILCAT")), "off")
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
func StartTailcat(allowed []key.NodePublic) (net.Listener, string, *tailcat.Server, error) {
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
		// Who may open a tunnel is a list, not a secret. It is the peers this
		// machine has recorded, which is a question we can already answer, and
		// unlike a shared key it is per peer and revocable.
		AllowedClients: allowed,
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
// strands everybody holding the old one (D-104).
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

// nodeKeyFor reads the tunnel identity out of an address recorded for a peer, so
// the allow-list can be built from the peers this machine already knows.
func nodeKeyFor(endpoint string) (key.NodePublic, bool) {
	e, err := ParseEndpoint(endpoint)
	if err != nil || e.Scheme != schemeTailcat {
		return key.NodePublic{}, false
	}
	ci, err := tailcat.ParseAddr(tailcat.Addr(e.Value))
	if err != nil || ci.ServerPublic.NodePublic.IsZero() {
		return key.NodePublic{}, false
	}
	return ci.ServerPublic.NodePublic, true
}
