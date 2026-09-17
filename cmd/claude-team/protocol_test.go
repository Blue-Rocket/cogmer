package main

import (
	"encoding/json"
	"strings"
	"testing"
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
	buf, _ := json.Marshal(syncResponse{Protocol: wireVersion, Room: "r"})
	if !strings.Contains(string(buf), `"protocol":`) {
		t.Errorf("sync response carries no protocol version: %s", buf)
	}
}
