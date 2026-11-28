package main

import (
	"crypto/sha256"
	"encoding/hex"
)

// testLabel is a name for a peer that no other key in a test shares. A test that
// records several random keys under their derived names fails whenever two of them
// derive the same one, which is about once in 8,280 pairs (PeerNameSpace), because
// a name means one key (D-074, a name means one master key). Forty-eight bits of
// the key's hash make that chance negligible.
func testLabel(peer string) string {
	sum := sha256.Sum256([]byte(peer))
	return "t-" + hex.EncodeToString(sum[:6])
}
