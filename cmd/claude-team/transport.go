package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// How a peer is reached, and nothing about whether it may speak.
//
// §4 calls an endpoint a bootstrap hint that is opaque to the protocol: whatever
// the transport in use can reach. That was true of the specification and not of
// the code, where `peerAddr()` was both the address the daemon bound and the one it
// told peers to use. Those are the same string only when nothing sits between two
// machines, which for two people working from home is never (D-063). An SSH tunnel
// papered over it in Phases 2 and 5 by making both ends loopback, and Phase 5
// caught the paper tearing: an invitation advertised the host's own 127.0.0.1, the
// bad address propagated to the other peer, and it retried it once a second for the
// length of the run.
//
// So an endpoint now carries how to reach it, and binding is a separate question
// from advertising.
//
//	tcp://198.51.100.7:4783   a direct address, and the bare form host:port
//	tc://<tailcat address>    a path negotiated through DERP, then direct if it can be
//
// A transport carries bytes. Every decision about who may speak sits above it and
// is unchanged: requests are signed (D-044), refused unless the peer is a guest
// (D-045), and refused unless a person verified that peer (D-054). Reaching the
// door is not admission, whichever door it is.

const (
	schemeTCP     = "tcp"
	schemeTailcat = "tc"
)

// Endpoint is an address together with the transport that understands it.
type Endpoint struct {
	Scheme string
	Value  string
}

// ParseEndpoint reads an endpoint. A bare host:port is TCP, because that is what
// every endpoint recorded before this was, and rewriting stored rows to add a
// prefix would be a migration that buys nothing.
func ParseEndpoint(s string) (Endpoint, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Endpoint{}, fmt.Errorf("empty endpoint")
	}
	if i := strings.Index(s, "://"); i > 0 {
		scheme, value := s[:i], s[i+3:]
		switch scheme {
		case schemeTCP, schemeTailcat:
			if value == "" {
				return Endpoint{}, fmt.Errorf("endpoint %q names a transport and no address", s)
			}
			return Endpoint{Scheme: scheme, Value: value}, nil
		default:
			return Endpoint{}, fmt.Errorf("no transport called %q", scheme)
		}
	}
	return Endpoint{Scheme: schemeTCP, Value: s}, nil
}

func (e Endpoint) String() string {
	if e.Scheme == schemeTCP {
		return e.Value // the bare form, so what is stored stays readable
	}
	return e.Scheme + "://" + e.Value
}

// Dialer opens a connection to one endpoint. Implementations are registered rather
// than switched on, so that a transport can be absent from a build without every
// call site knowing.
type Dialer interface {
	Dial(ctx context.Context, e Endpoint) (net.Conn, error)
}

type tcpDialer struct{}

func (tcpDialer) Dial(ctx context.Context, e Endpoint) (net.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, "tcp", e.Value)
}

var dialers = map[string]Dialer{schemeTCP: tcpDialer{}}

// RegisterDialer adds a transport. Called from an init so that removing a
// transport removes its dependency with it.
func RegisterDialer(scheme string, d Dialer) { dialers[scheme] = d }

// clientFor builds an HTTP client that reaches exactly one peer.
//
// One client per endpoint rather than one client with clever routing: a tailcat
// address is two hundred characters and has no business being url-encoded into a
// hostname. The URL becomes http://peer/… and the dialer ignores it entirely,
// which is honest about where the routing decision is actually made.
func clientFor(endpoint string, timeout time.Duration) (*http.Client, error) {
	e, err := ParseEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	d, ok := dialers[e.Scheme]
	if !ok {
		return nil, fmt.Errorf("this build cannot reach a %s endpoint", e.Scheme)
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return d.Dial(ctx, e)
			},
			// A peer is one host, so a pool of one keeps a dead peer from holding
			// connections open against every room that polls it.
			MaxIdleConnsPerHost: 1,
			IdleConnTimeout:     30 * time.Second,
		},
	}, nil
}

// peerURL is the URL used with a client from clientFor. The host is a placeholder:
// the dialer decides where this goes.
func peerURL(path string) string { return "http://peer" + path }

// --- what we tell other peers, which is not what we bind ---

// endpointFile is where the daemon records how it can be reached, so that
// `whoami` and `invite` -- separate processes, which cannot ask a transport
// anything -- print the same endpoint the daemon is actually listening on.
const endpointFile = "endpoint"

// AdvertisedEndpoint is what goes into an invitation, a pairing string, and the
// Endpoint field of a sync request.
//
// It is deliberately NOT peerAddr(). That is the address the daemon binds, and a
// bound address is only reachable by somebody else when nothing sits in between.
func AdvertisedEndpoint() string {
	if v := strings.TrimSpace(os.Getenv("CLAUDE_TEAM_PEER_ENDPOINT")); v != "" {
		return v
	}
	if b, err := os.ReadFile(filepath.Join(homeDir(), endpointFile)); err == nil {
		if v := strings.TrimSpace(string(b)); v != "" {
			return v
		}
	}
	// Nothing recorded: the daemon is not running, or is reachable only directly.
	// Saying the bound address is the best guess available and is frequently
	// wrong, which is why `whoami` warns when it is loopback.
	return peerAddr()
}

// recordEndpoint publishes how this daemon can be reached. Written atomically,
// because `whoami` may read it while the daemon is starting.
func recordEndpoint(e string) error {
	dir := homeDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp := filepath.Join(dir, endpointFile+".tmp")
	if err := os.WriteFile(tmp, []byte(e+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, endpointFile))
}

// Short renders an endpoint for a log line or a status display.
//
// A tailcat address is 237 characters, and a peer's address appears in every
// reachability message. Printed whole it pushes the part a person is reading off
// the screen, and several of them make a log unreadable -- observed immediately on
// the first two-machine run.
//
// The prefix is enough to tell two peers apart, and anything needing the whole
// value is machinery rather than a person.
func (e Endpoint) Short() string {
	if e.Scheme == schemeTCP {
		return e.Value
	}
	const keep = 12
	if len(e.Value) <= keep {
		return e.String()
	}
	// The length reported is of the whole endpoint, because that is the string a
	// person would otherwise be looking at or pasting.
	return fmt.Sprintf("%s://%s… (%d chars)", e.Scheme, e.Value[:keep], len(e.String()))
}

// shortEndpoint is Short for an endpoint that has not been parsed, which is how
// most call sites hold one.
func shortEndpoint(s string) string {
	e, err := ParseEndpoint(s)
	if err != nil {
		return s
	}
	return e.Short()
}
