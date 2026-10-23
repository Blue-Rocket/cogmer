package main

import (
	"crypto/ed25519"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
)

// The recovery phrase, and the master key it derives (D-219, a user is a master
// key that certifies machines through a shared key).
//
// The phrase is the user. Nothing stores it: it is typed into the daemon's page,
// the master key is derived while it is there, and both are dropped (D-223, the
// recovery phrase is typed only into the daemon's browser page). Everything in
// this file is therefore written so that a phrase cannot leak by accident. It logs
// nothing, an error names a position and never a word, and a Phrase refuses to
// print or marshal itself.
//
// The words are the PGP word list that sas.go holds. Sixteen random bytes become
// sixteen words, and two more carry the first two bytes of their SHA-256. The two
// lists alternate by position, so a swapped or dropped word gives a phrase that
// does not parse, and the check words catch a wrong word that happens to fall in
// the right list, missing it with 1 chance in 65,536.

// masterKeyInfo is the HKDF info for the master key's seed. Every user's
// identifier is the public half of the key it derives, so changing it changes who
// every user is. A different derivation is a new version beside this one, never
// an edit of it.
const masterKeyInfo = protocolNamespace + "/master-key/v1"

const (
	phraseBytes = 16
	phraseCheck = 2
	phraseWords = phraseBytes + phraseCheck
)

// Phrase is the 16 bytes a recovery phrase encodes. They have full entropy, which
// is why the master key's seed comes from HKDF and not from a slow function such
// as Argon2: there is nothing for a slow function to protect against guessing.
type Phrase [phraseBytes]byte

// NewPhrase draws a phrase from the system's random source.
func NewPhrase() (Phrase, error) {
	var p Phrase
	if _, err := rand.Read(p[:]); err != nil {
		return Phrase{}, err
	}
	return p, nil
}

// checksum is the two bytes the last two words carry.
func (p Phrase) checksum() [phraseCheck]byte {
	sum := sha256.Sum256(p[:])
	return [phraseCheck]byte{sum[0], sum[1]}
}

// Words renders the phrase as 18 words, even positions from sasEven and odd ones
// from sasOdd, as sasRender alternates them.
func (p Phrase) Words() []string {
	c := p.checksum()
	b := append(append([]byte{}, p[:]...), c[:]...)
	out := make([]string, len(b))
	for i, x := range b {
		if i%2 == 0 {
			out[i] = sasEven[x]
		} else {
			out[i] = sasOdd[x]
		}
	}
	return out
}

// phraseIndex maps each lower-cased word to its byte, one map per list. It relies
// on no word appearing in both lists or twice in one, which a test checks, since
// a word that did would parse to whichever byte came last.
var phraseIndex = func() [2]map[string]byte {
	var idx [2]map[string]byte
	for k, list := range [2]*[256]string{&sasEven, &sasOdd} {
		idx[k] = make(map[string]byte, len(list))
		for i, w := range list {
			idx[k][strings.ToLower(w)] = byte(i)
		}
	}
	return idx
}()

// ParsePhrase reads a phrase the user typed, ignoring case and any run of
// whitespace. Its errors say what is wrong and where, and never quote the input:
// an error is shown, logged and pasted into reports, and a word in it is part of
// a phrase held by whoever reads it.
func ParsePhrase(s string) (Phrase, error) {
	words := strings.Fields(s)
	if len(words) != phraseWords {
		return Phrase{}, fmt.Errorf("the phrase has %d words, want %d", len(words), phraseWords)
	}
	var b [phraseWords]byte
	for i, w := range words {
		x, ok := phraseIndex[i%2][strings.ToLower(w)]
		if !ok {
			if _, other := phraseIndex[1-i%2][strings.ToLower(w)]; other {
				return Phrase{}, fmt.Errorf("word %d is from the list for the other position; a word may be missing or two swapped", i+1)
			}
			return Phrase{}, fmt.Errorf("word %d is not in the list for its position", i+1)
		}
		b[i] = x
	}
	var p Phrase
	copy(p[:], b[:phraseBytes])
	if c := p.checksum(); c[0] != b[phraseBytes] || c[1] != b[phraseBytes+1] {
		return Phrase{}, errors.New("the check words do not match; a word is wrong")
	}
	return p, nil
}

// MasterKey derives the user's master key. The same phrase gives the same key on
// every machine, which is what lets a user who has lost every machine be the same
// user on a new one.
func (p Phrase) MasterKey() ed25519.PrivateKey {
	seed, err := hkdf.Key(sha256.New, p[:], nil, masterKeyInfo, ed25519.SeedSize)
	if err != nil {
		// hkdf.Key fails only for a length beyond 255 hashes, and the length here
		// is a constant 32.
		panic(err)
	}
	return ed25519.NewKeyFromSeed(seed)
}

// UserID is the user's identifier: the master key's public half, written as a
// peer identifier is.
func (p Phrase) UserID() string {
	return PeerIDFromPublic(p.MasterKey().Public().(ed25519.PublicKey))
}

// String hides the phrase. A phrase printed by accident with %v, or in a struct
// someone logs, is a copy held by whoever reads the log, so the only way to the
// words is to ask for them with Words.
func (p Phrase) String() string { return "[recovery phrase]" }

// GoString hides the phrase from %#v for the same reason.
func (p Phrase) GoString() string { return p.String() }

// MarshalJSON refuses, for the same reason as String: a phrase in JSON is a phrase
// in whatever file, response or log that JSON reaches.
func (p Phrase) MarshalJSON() ([]byte, error) {
	return nil, errors.New("a recovery phrase is never marshalled")
}
