package main

import "encoding/json"

// The wire format is defined here and nowhere else.
//
// It is deliberately a separate type from the stored Event, even though the two
// presently carry the same fields. A database row and a protocol message answer to
// different pressures: a column can be added for local bookkeeping without
// telling any peer, and a wire field cannot be changed without every peer agreeing.
// Sharing one struct means the next convenient column silently becomes protocol.
//
// Nothing here mentions the agent that produced an event. A session identifier is
// an opaque string naming where a turn came from; that it is currently a Claude
// Code session is a fact about the adapter, not about the protocol.
const wireVersion = 2

// minWireVersion is the oldest a peer may speak and still be read.
//
// A hard equality check makes every protocol change a flag day: both people must
// upgrade at the same moment or the room goes silent, and the failure names a
// version rather than saying what to do. That is tolerable while one person builds
// both sides and intolerable the moment somebody else runs this (D-065).
//
// Raise this only when an older version can no longer be understood — which is a
// statement about the events on the wire, not about tidiness.
const minWireVersion = 1

// speaks reports whether a version is one this build can read. Zero means a peer
// that predates the field, which spoke v1.
func speaks(v int) bool {
	if v == 0 {
		v = 1
	}
	return v >= minWireVersion && v <= wireVersion
}

type wireEvent struct {
	EventID         string          `json:"eventId"`
	PeerID          string          `json:"peerId"`
	PeerSequence    int64           `json:"peerSequence"`
	RoomID          string          `json:"roomId"`
	Timestamp       string          `json:"timestamp"`
	UserID          string          `json:"userId"`
	UserDisplayName string          `json:"userDisplayName"`
	MachineID       string          `json:"machineId"`
	OriginSessionID string          `json:"originSessionId"`
	EventType       string          `json:"eventType"`
	Content         string          `json:"content"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	Signature       string          `json:"signature,omitempty"`
	// SigVersion travels with the event because the receiver must know which
	// scheme to verify under. Omitted means v2 -- the scheme in use before this
	// field existed, which is what a peer on an older build still sends.
	SigVersion int `json:"sigVersion,omitempty"`
}

func toWire(e Event) wireEvent {
	return wireEvent{
		EventID: e.EventID, PeerID: e.PeerID, PeerSequence: e.PeerSequence,
		RoomID: e.RoomID, Timestamp: e.Timestamp, UserID: e.UserID,
		UserDisplayName: e.UserDisplayName, MachineID: e.MachineID,
		OriginSessionID: e.OriginSessionID, EventType: e.EventType,
		Content: e.Content, Metadata: e.Metadata, Signature: e.Signature,
		SigVersion: e.SigVersion,
	}
}

func fromWire(w wireEvent) Event {
	return Event{
		EventID: w.EventID, PeerID: w.PeerID, PeerSequence: w.PeerSequence,
		RoomID: w.RoomID, Timestamp: w.Timestamp, UserID: w.UserID,
		UserDisplayName: w.UserDisplayName, MachineID: w.MachineID,
		OriginSessionID: w.OriginSessionID, EventType: w.EventType,
		Content: w.Content, Metadata: w.Metadata, Signature: w.Signature,
		SigVersion: w.SigVersion,
	}
}
