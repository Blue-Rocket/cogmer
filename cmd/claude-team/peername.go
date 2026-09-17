package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// Peer names are derived, never chosen (§6). A name a peer picks for itself is a
// claim about who it is, and a display name must not be a claim.
//
// Derivation is deliberately the only path: the same peer identifier always
// yields the same name, and a peer cannot declare itself to be someone else.
// It does NOT make a name unforgeable. The space is small enough that an
// identity generated freely can be regenerated until its name matches a chosen
// target, and no name short enough to say aloud can resist that. A name is a
// mnemonic for an identity already verified, never an introduction to a stranger.
//
// Word-list constraints (§6): these names attach to colleagues. No adjective
// that would be unkind applied to a person, and no animal used as an insult --
// combinations are generated, so nobody approves them individually.

var peerAdjectives = []string{
	"amber", "ancient", "autumn", "boreal", "brave", "bright", "brisk", "calm",
	"candid", "cedar", "cheerful", "civil", "clever", "coastal", "copper", "cosmic",
	"crisp", "curious", "dawn", "deft", "diligent", "distant", "dusky", "eager",
	"early", "eastern", "easy", "elder", "fabled", "fearless", "fleet", "fond",
	"frosted", "gallant", "genial", "gentle", "gilded", "glad", "golden", "graceful",
	"hardy", "hidden", "honest", "humble", "inland", "jovial", "keen", "kindly",
	"lively", "lucid", "lunar", "mellow", "merry", "mindful", "misty", "modest",
	"northern", "nimble", "noble", "patient", "placid", "polar", "prudent", "quiet",
	"radiant", "rapid", "ready", "restful", "rugged", "serene", "silver", "solar",
	"spirited", "steady", "stellar", "sunlit", "swift", "tidal", "tranquil", "twilit",
	"upland", "valiant", "verdant", "vivid", "wandering", "warm", "western", "willing",
	"windward", "wise", "woodland", "zealous",
}

var peerAnimals = []string{
	"albatross", "avocet", "badger", "bittern", "bluejay", "bobcat", "caribou", "chamois",
	"chickadee", "cormorant", "crane", "curlew", "dipper", "dormouse", "dunlin", "egret",
	"eider", "elk", "ermine", "falcon", "finch", "gannet", "godwit", "goldcrest",
	"grebe", "hare", "harrier", "heron", "ibis", "jackdaw", "kestrel", "kingfisher",
	"kite", "lapwing", "lark", "lemming", "linnet", "lynx", "mallard", "marten",
	"merlin", "mink", "moorhen", "nightjar", "nuthatch", "osprey", "otter", "ouzel",
	"owlet", "oystercatcher", "petrel", "pika", "pintail", "pipit", "plover", "pochard",
	"ptarmigan", "puffin", "quail", "raven", "redshank", "redstart", "reindeer", "roebuck",
	"sanderling", "sandpiper", "shearwater", "shelduck", "shrew", "siskin", "skylark", "smew",
	"sparrow", "stoat", "stonechat", "swift", "teal", "tern", "thrush", "treecreeper",
	"turnstone", "vole", "wagtail", "warbler", "wheatear", "whimbrel", "wigeon", "woodlark",
	"wren", "yellowhammer",
}

// PeerName derives a stable display name from a peer identifier.
//
// It hashes rather than slicing the identifier directly, so the name is
// well-distributed whatever the identifier's format -- random hex today, a public
// key fingerprint once identity becomes cryptographic (D-020). The derivation
// need not change when that happens.
func PeerName(peerID string) string {
	sum := sha256.Sum256([]byte(peerID))
	a := binary.BigEndian.Uint32(sum[0:4]) % uint32(len(peerAdjectives))
	n := binary.BigEndian.Uint32(sum[4:8]) % uint32(len(peerAnimals))
	return fmt.Sprintf("%s-%s", peerAdjectives[a], peerAnimals[n])
}

// PeerNameSpace is the number of distinct names available. It bounds accidental
// collision, not impersonation -- see the note above.
func PeerNameSpace() int { return len(peerAdjectives) * len(peerAnimals) }
