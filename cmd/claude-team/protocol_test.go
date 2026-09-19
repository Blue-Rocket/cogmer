package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// The wire format must not mention the agent that produced an event. A session
// identifier says where a turn came from; that it is currently a Claude Code
// session is a fact about the adapter, not about the protocol.
func TestWireFormatNamesNoAgent(t *testing.T) {
	buf, err := json.Marshal(toWire(Event{
		EventID: "e", PeerID: "ed25519:x", RoomID: "r", OriginSessionID: "s",
		EventType: EventUserPrompt, Content: "hi",
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"claude", "Claude", "anthropic"} {
		if strings.Contains(string(buf), leak) {
			t.Errorf("wire format mentions %q: %s", leak, buf)
		}
	}
	if !strings.Contains(string(buf), `"originSessionId"`) {
		t.Errorf("wire format lost the session field: %s", buf)
	}
}

// Round-tripping must not quietly drop a field, which is the failure a separate
// wire type exists to make visible rather than prevent.
func TestWireRoundTripPreservesEverything(t *testing.T) {
	in := Event{
		EventID: "e1", PeerID: "ed25519:abc", PeerSequence: 7, RoomID: "r1",
		Timestamp: "2026-09-17T00:00:00Z", UserID: "david", UserDisplayName: "David",
		MachineID: "m", OriginSessionID: "sess", EventType: EventAssistantMessage,
		Content: "text", Metadata: json.RawMessage(`{"toolCalls":2}`), Signature: "sig",
	}
	out := fromWire(toWire(in))
	a, _ := json.Marshal(in)
	b, _ := json.Marshal(out)
	if string(a) != string(b) {
		t.Errorf("round trip changed the event:\n in: %s\nout: %s", a, b)
	}
	// A field added to the stored event and forgotten in the mapping would show
	// up here as a shorter encoding, which is the whole point of the split.
	if strings.Count(string(b), `":`) != strings.Count(string(a), `":`) {
		t.Error("round trip dropped a field")
	}
}

// A signature covers field values, so a peer on a different protocol version may
// be signing something else entirely. Guessing is worse than refusing.
func TestProtocolVersionIsCarried(t *testing.T) {
	buf, _ := json.Marshal(syncResponse{Protocol: wireVersion, RoomID: "r"})
	if !strings.Contains(string(buf), `"protocol":`) {
		t.Errorf("sync response carries no protocol version: %s", buf)
	}
}

// The scheme must survive storage and the wire, or recording it accomplishes
// nothing: a receiver that does not know which scheme an event was signed under
// cannot verify it after a scheme is added.
func TestSigVersionSurvivesTheRoundTrip(t *testing.T) {
	e := Event{EventID: "e1", PeerID: "ed25519:x", EventType: EventUserPrompt,
		Content: "hi", Signature: "sig", SigVersion: 2}
	if got := fromWire(toWire(e)).SigVersion; got != 2 {
		t.Errorf("wire round trip gave scheme %d, want 2", got)
	}

	// A peer on an older build omits the field entirely. That must read as v2 --
	// the scheme in use before the field existed -- not as "unknown".
	var w wireEvent
	if err := json.Unmarshal([]byte(`{"eventId":"e1","peerId":"ed25519:x","signature":"sig"}`), &w); err != nil {
		t.Fatal(err)
	}
	if got := fromWire(w).SigVersion; got != 0 {
		t.Errorf("an omitted sigVersion became %d; it must stay 0 so it reads as v2", got)
	}
}

// Upgrading must not be a flag day. Two people pair daily; requiring both to
// upgrade in the same moment means the room goes silent until they do, and the
// failure names a version rather than saying what to do.
func TestAnOlderProtocolIsStillRead(t *testing.T) {
	for _, v := range []int{0, minWireVersion, wireVersion} {
		if !speaks(v) {
			t.Errorf("protocol v%d is refused; this build reads v%d to v%d", v, minWireVersion, wireVersion)
		}
	}
	// Zero means a peer predating the field, which spoke v1.
	if minWireVersion > 1 && speaks(0) {
		t.Error("an absent version was read as current rather than as v1")
	}
	// A version from the future is refused, because its events may not mean what
	// this build would take them to mean.
	if speaks(wireVersion + 1) {
		t.Errorf("protocol v%d was accepted by a build that speaks v%d", wireVersion+1, wireVersion)
	}
}

// An endpoint carries how to reach it. A bare host:port still means TCP, because
// that is what every endpoint recorded before the seam existed was, and rewriting
// stored rows to add a prefix would be a migration that buys nothing.
func TestEndpointsCarryTheirTransport(t *testing.T) {
	for _, c := range []struct{ in, scheme, value string }{
		{"198.51.100.7:4783", schemeTCP, "198.51.100.7:4783"},
		{"tcp://198.51.100.7:4783", schemeTCP, "198.51.100.7:4783"},
		{"[2001:db8::1]:4783", schemeTCP, "[2001:db8::1]:4783"},
		{"tc://tcpGFwWCCcd6msMC", schemeTailcat, "tcpGFwWCCcd6msMC"},
	} {
		got, err := ParseEndpoint(c.in)
		if err != nil {
			t.Errorf("%q: %v", c.in, err)
			continue
		}
		if got.Scheme != c.scheme || got.Value != c.value {
			t.Errorf("%q parsed as %s/%s, want %s/%s", c.in, got.Scheme, got.Value, c.scheme, c.value)
		}
	}

	// A TCP endpoint round-trips to the bare form, so what is stored stays
	// readable and a person reading the database sees what they always saw.
	e, _ := ParseEndpoint("tcp://198.51.100.7:4783")
	if e.String() != "198.51.100.7:4783" {
		t.Errorf("tcp endpoint rendered as %q, want the bare form", e.String())
	}
	e, _ = ParseEndpoint("tc://abc")
	if e.String() != "tc://abc" {
		t.Errorf("tailcat endpoint rendered as %q", e.String())
	}

	for _, bad := range []string{"", "   ", "carrier-pigeon://somewhere", "tc://"} {
		if _, err := ParseEndpoint(bad); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}

// A transport this build cannot reach must be refused by name, not dialled as
// though it were an address. The failure otherwise is a TCP connection to a
// two-hundred-character hostname, which reports something unrecognisable.
func TestAnUnreachableTransportIsRefusedByName(t *testing.T) {
	saved := dialers
	dialers = map[string]Dialer{schemeTCP: tcpDialer{}}
	defer func() { dialers = saved }()

	_, err := clientFor("tc://whatever", time.Second)
	if err == nil {
		t.Fatal("a client was built for a transport this build lacks")
	}
	if !strings.Contains(err.Error(), schemeTailcat) {
		t.Errorf("the refusal does not name the transport: %v", err)
	}

	if _, err := clientFor("198.51.100.7:4783", time.Second); err != nil {
		t.Errorf("a plain address was refused: %v", err)
	}
}

// A tailcat address is 237 characters and appears in every reachability message.
// Printed whole it pushes what a person is reading off the screen.
func TestEndpointsAreReadableInLogs(t *testing.T) {
	long := "tc://" + strings.Repeat("x", 232)
	short := shortEndpoint(long)
	if len(short) > 48 {
		t.Errorf("a logged endpoint is %d characters: %s", len(short), short)
	}
	if !strings.Contains(short, "237") {
		t.Errorf("the short form hides how much was elided: %s", short)
	}
	// A plain address is short already and must not be mangled.
	if got := shortEndpoint("198.51.100.7:4783"); got != "198.51.100.7:4783" {
		t.Errorf("a plain address was shortened to %q", got)
	}
	// Something unparseable is passed through rather than swallowed: a log line
	// is where a malformed endpoint should be visible, not hidden.
	if got := shortEndpoint("not an endpoint"); got != "not an endpoint" {
		t.Errorf("an unparseable endpoint became %q", got)
	}
}
