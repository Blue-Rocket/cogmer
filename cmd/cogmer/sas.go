package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"sort"
	"strings"
)

// The short authentication string (D-048).
//
// Two people on a call compare two words. That is sound -- despite being far
// shorter than the 43-character identifier §25 otherwise requires -- only because
// of what the words are derived from. Two properties carry it, and removing either
// leaves a ceremony that looks identical and protects nothing:
//
//   - COMMITMENT. Each side sends a hash of its nonce before either reveals one.
//     An attacker relaying between the two must fix what he presents to each side
//     before learning what the other will present, so he cannot search for a pair
//     of substituted keys whose words agree. He is reduced to one blind guess.
//
//   - FRESHNESS. The words derive from per-exchange nonces, not from the standing
//     identifiers alone. There is nothing to precompute against, because the target
//     does not exist until the exchange is under way.
//
// Together those turn an offline search into a single online guess at 1 in 65,536,
// whose failure the two people HEAR. That last part is not a nicety: a short string
// with silent retries is weak, so a mismatch must refuse and must not present
// itself as a transient error worth repeating.
//
// The words come from the PGP biometric word list, which exists for this job:
// phonetically distinct over a bad telephone line, and split into two lists so
// that the words alternate. Alternation makes a transposition detectable -- said in
// the wrong order, the pair is not a valid rendering of anything.
//
// They are deliberately NOT the peer-name or room-name vocabularies. A SAS sits on
// screen beside a derived peer name, and one mnemonic that could be mistaken for
// the other is how a person compares the wrong thing.

const (
	sasTag    = protocolNamespace + "/sas/v1"
	sasCommit = protocolNamespace + "/sas-commit/v1"
	// sasWords is how many words are compared, one byte each.
	sasWords = 2
)

var sasEven = [256]string{
	"aardvark", "absurd", "accrue", "acme",
	"adrift", "adult", "afflict", "ahead",
	"aimless", "Algol", "allow", "alone",
	"ammo", "ancient", "apple", "artist",
	"assume", "Athens", "atlas", "Aztec",
	"baboon", "backfield", "backward", "banjo",
	"beaming", "bedlamp", "beehive", "beeswax",
	"befriend", "Belfast", "berserk", "billiard",
	"bison", "blackjack", "blockade", "blowtorch",
	"bluebird", "bombast", "bookshelf", "brackish",
	"breadline", "breakup", "brickyard", "briefcase",
	"Burbank", "button", "buzzard", "cement",
	"chairlift", "chatter", "checkup", "chisel",
	"choking", "chopper", "Christmas", "clamshell",
	"classic", "classroom", "cleanup", "clockwork",
	"cobra", "commence", "concert", "cowbell",
	"crackdown", "cranky", "crowfoot", "crucial",
	"crumpled", "crusade", "cubic", "dashboard",
	"deadbolt", "deckhand", "dogsled", "dragnet",
	"drainage", "dreadful", "drifter", "dropper",
	"drumbeat", "drunken", "Dupont", "dwelling",
	"eating", "edict", "egghead", "eightball",
	"endorse", "endow", "enlist", "erase",
	"escape", "exceed", "eyeglass", "eyetooth",
	"facial", "fallout", "flagpole", "flatfoot",
	"flytrap", "fracture", "framework", "freedom",
	"frighten", "gazelle", "Geiger", "glitter",
	"glucose", "goggles", "goldfish", "gremlin",
	"guidance", "hamlet", "highchair", "hockey",
	"indoors", "indulge", "inverse", "involve",
	"island", "jawbone", "keyboard", "kickoff",
	"kiwi", "klaxon", "locale", "lockup",
	"merit", "minnow", "miser", "Mohawk",
	"mural", "music", "necklace", "Neptune",
	"newborn", "nightbird", "Oakland", "obtuse",
	"offload", "optic", "orca", "payday",
	"peachy", "pheasant", "physique", "playhouse",
	"Pluto", "preclude", "prefer", "preshrunk",
	"printer", "prowler", "pupil", "puppy",
	"python", "quadrant", "quiver", "quota",
	"ragtime", "ratchet", "rebirth", "reform",
	"regain", "reindeer", "rematch", "repay",
	"retouch", "revenge", "reward", "rhythm",
	"ribcage", "ridgepole", "rockfish", "rocker",
	"ruffled", "sailboat", "sawdust", "scallion",
	"scenic", "scorecard", "Scotland", "seabird",
	"select", "sentence", "shadow", "shamrock",
	"showgirl", "skullcap", "skydive", "slingshot",
	"slowdown", "snapline", "snapshot", "snowcap",
	"snowslide", "solo", "southward", "soybean",
	"spaniel", "spearhead", "spellbind", "spheroid",
	"spigot", "spindle", "spyglass", "stagehand",
	"stagecoach", "stallion", "standard", "stapler",
	"steamship", "sterling", "stockman", "stopwatch",
	"stormy", "sugar", "surmount", "suspense",
	"sweatband", "swelter", "tactics", "talon",
	"tapeworm", "tempest", "tiger", "tissue",
	"tonic", "topmost", "tracker", "transit",
	"trauma", "treadmill", "Trojan", "trouble",
	"tumor", "tunnel", "tycoon", "uncut",
	"unearth", "unwind", "uproot", "upset",
	"upshot", "vapor", "village", "virus",
	"Vulcan", "waffle", "wallet", "watchword",
	"wayside", "willow", "woodlark", "Zulu",
}

var sasOdd = [256]string{
	"adroitness", "adviser", "aftermath", "aggregate",
	"alkali", "almighty", "amulet", "amusement",
	"antenna", "applicant", "Apollo", "armistice",
	"article", "asteroid", "Atlantic", "atmosphere",
	"autopsy", "Babylon", "backwater", "barbecue",
	"belowground", "bifocals", "bodyguard", "bookseller",
	"borderline", "bottomless", "Bradbury", "bravado",
	"Brazilian", "breakaway", "Burlington", "businessman",
	"butterfat", "Camelot", "candidate", "cannonball",
	"Capricorn", "caravan", "caretaker", "celebrate",
	"cellulose", "certify", "chambermaid", "Cherokee",
	"Chicago", "clergyman", "coherence", "combustion",
	"commando", "company", "component", "concurrent",
	"confidence", "conformist", "congregate", "consensus",
	"consulting", "corporate", "corrosion", "councilman",
	"crossover", "crucifix", "cumbersome", "customer",
	"Dakota", "decadence", "December", "decimal",
	"designing", "detector", "detergent", "determine",
	"dictator", "dinosaur", "direction", "disable",
	"disbelief", "disruptive", "distortion", "document",
	"embezzle", "enchanting", "enrollment", "enterprise",
	"equation", "equipment", "escapade", "Eskimo",
	"everyday", "examine", "existence", "exodus",
	"fascinate", "filament", "finicky", "forever",
	"fortitude", "frequency", "gadgetry", "Galveston",
	"getaway", "glossary", "gossamer", "graduate",
	"gravity", "guitarist", "hamburger", "Hamilton",
	"handiwork", "hazardous", "headwaters", "hemisphere",
	"hesitate", "hideaway", "holiness", "hurricane",
	"hydraulic", "impartial", "impetus", "inception",
	"indigo", "inertia", "infancy", "inferno",
	"informant", "insincere", "insurgent", "integrate",
	"intention", "inventive", "Istanbul", "Jamaica",
	"Jupiter", "leprosy", "letterhead", "liberty",
	"maritime", "matchmaker", "maverick", "Medusa",
	"megaton", "microscope", "microwave", "midsummer",
	"millionaire", "miracle", "misnomer", "molasses",
	"molecule", "Montana", "monument", "mosquito",
	"narrative", "nebula", "newsletter", "Norwegian",
	"October", "Ohio", "onlooker", "opulent",
	"Orlando", "outfielder", "Pacific", "pandemic",
	"Pandora", "paperweight", "paragon", "paragraph",
	"paramount", "passenger", "pedigree", "Pegasus",
	"penetrate", "perceptive", "performance", "pharmacy",
	"phonetic", "photograph", "pioneer", "pocketful",
	"politeness", "positive", "potato", "processor",
	"provincial", "proximate", "puberty", "publisher",
	"pyramid", "quantity", "racketeer", "rebellion",
	"recipe", "recover", "repellent", "replica",
	"reproduce", "resistor", "responsive", "retraction",
	"retrieval", "retrospect", "revenue", "revival",
	"revolver", "sandalwood", "sardonic", "Saturday",
	"savagery", "scavenger", "sensation", "sociable",
	"souvenir", "specialist", "speculate", "stethoscope",
	"stupendous", "supportive", "surrender", "suspicious",
	"sympathy", "tambourine", "telephone", "therapist",
	"tobacco", "tolerance", "tomorrow", "torpedo",
	"tradition", "travesty", "trombonist", "truncated",
	"typewriter", "ultimate", "undaunted", "underfoot",
	"unicorn", "unify", "universe", "unravel",
	"upcoming", "vacancy", "vagabond", "vertigo",
	"Virginia", "visitor", "vocalist", "voyager",
	"warranty", "Waterloo", "whimsical", "Wichita",
	"Wilmington", "Wyoming", "yesteryear", "Yucatan",
}

// sasRender turns bytes into alternating words, even list first.
func sasRender(b []byte) string {
	var out []string
	for i, x := range b {
		if i%2 == 0 {
			out = append(out, sasEven[x])
		} else {
			out = append(out, sasOdd[x])
		}
	}
	return strings.Join(out, " ")
}

// sasCommitment is what a peer publishes before revealing its nonce.
func sasCommitment(peerID string, nonce []byte) []byte {
	h := sha256.New()
	writeField(h, sasCommit)
	writeField(h, peerID)
	h.Write(nonce)
	return h.Sum(nil)
}

// sasOpens reports whether a revealed nonce matches a commitment already held.
// Constant time is not strictly required here -- the commitment is public once
// sent -- but a comparison that leaks nothing costs nothing.
func sasOpens(commitment []byte, peerID string, nonce []byte) bool {
	return subtle.ConstantTimeCompare(commitment, sasCommitment(peerID, nonce)) == 1
}

// SAS derives the words two peers compare.
//
// Both identities and both nonces go in, in an order both sides compute the same
// way, so that the string says "these two keys, this exchange" rather than merely
// "we share a secret". Sorting by identifier is what makes it symmetric: neither
// side is the initiator as far as the digest is concerned.
func SAS(peerA string, nonceA []byte, peerB string, nonceB []byte) string {
	type side struct {
		id    string
		nonce []byte
	}
	s := []side{{peerA, nonceA}, {peerB, nonceB}}
	sort.Slice(s, func(i, j int) bool { return s[i].id < s[j].id })

	h := sha256.New()
	writeField(h, sasTag)
	for _, x := range s {
		writeField(h, x.id)
		writeField(h, string(x.nonce))
	}
	return sasRender(h.Sum(nil)[:sasWords])
}

// writeField length-prefixes, so that no two different field sequences can
// produce the same bytes. The same reasoning as signingBytes in keys.go.
func writeField(h interface{ Write([]byte) (int, error) }, s string) {
	var n [4]byte
	binary.BigEndian.PutUint32(n[:], uint32(len(s)))
	h.Write(n[:])
	h.Write([]byte(s))
}

// verifyBytes is what a verification message's signature covers. Each message is
// signed by the key it claims, so the exchange also proves possession -- the SAS
// binds the identities, the signature shows each side holds the one it named.
func verifyBytes(step, peerID string, payload []byte) []byte {
	h := sha256.New()
	writeField(h, protocolNamespace+"/verify/v1")
	writeField(h, step)
	writeField(h, peerID)
	h.Write(payload)
	return h.Sum(nil)
}

func signVerify(priv ed25519.PrivateKey, step, peerID string, payload []byte) []byte {
	return ed25519.Sign(priv, verifyBytes(step, peerID, payload))
}

func checkVerify(peerID, step string, payload, sig []byte) bool {
	pub, err := PublicFromPeerID(peerID)
	if err != nil {
		return false
	}
	return ed25519.Verify(pub, verifyBytes(step, peerID, payload), sig)
}
