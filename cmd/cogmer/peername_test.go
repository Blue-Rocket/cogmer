package main

import (
	"strings"
	"testing"
)

// The same identity must always yield the same name; otherwise a name cannot
// serve as a mnemonic for a peer at all.
func TestPeerNameIsStable(t *testing.T) {
	for _, id := range []string{"peer-abc123", "peer-000000", ""} {
		if PeerName(id) != PeerName(id) {
			t.Errorf("unstable name for %q", id)
		}
	}
	if PeerName("peer-a") == PeerName("peer-b") {
		t.Error("distinct identities collided on a two-input sample; derivation is not distributing")
	}
}

func TestPeerNameShape(t *testing.T) {
	n := PeerName("peer-852ffe6408339bcada91")
	parts := strings.Split(n, "-")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		t.Fatalf("want adjective-animal, got %q", n)
	}
}

// Derivation must spread across the space rather than clustering, or accidental
// collisions arrive far sooner than the space size suggests.
func TestPeerNameDistribution(t *testing.T) {
	seen := map[string]bool{}
	const n = 2000
	for i := 0; i < n; i++ {
		seen[PeerName("peer-"+strings.Repeat("x", i%7)+string(rune(i)))] = true
	}
	// Birthday bound over 8280 names predicts heavy collision at 2000 draws;
	// require only that we are not collapsing into a handful of buckets.
	if len(seen) < n/3 {
		t.Errorf("derivation clustering badly: %d distinct names from %d identities", len(seen), n)
	}
}

// The lists name colleagues (§6). This guards the constraint that is easiest to
// erode as the lists grow, since combinations are generated and unreviewed.
func TestWordListsAvoidTermsUsedAgainstPeople(t *testing.T) {
	banned := []string{
		"fat", "slow", "dim", "lazy", "dumb", "odd", "ugly", "weak", "mad", "crazy",
		"pig", "rat", "snake", "weasel", "ape", "monkey", "cow", "dog", "chicken", "worm", "slug", "toad",
	}
	for _, list := range [][]string{peerAdjectives, peerAnimals} {
		for _, w := range list {
			for _, b := range banned {
				if w == b {
					t.Errorf("%q is used against people; not suitable for naming a colleague", w)
				}
			}
		}
	}
}

func TestWordListsAreCleanAndUnique(t *testing.T) {
	for _, list := range [][]string{peerAdjectives, peerAnimals} {
		seen := map[string]bool{}
		for _, w := range list {
			if seen[w] {
				t.Errorf("duplicate word %q shrinks the space silently", w)
			}
			seen[w] = true
			if w != strings.ToLower(w) || strings.ContainsAny(w, " -_") {
				t.Errorf("word %q is not a bare lowercase token; it would break name parsing", w)
			}
		}
	}
	if PeerNameSpace() < 4000 {
		t.Errorf("name space %d is small enough that accidental collisions are routine", PeerNameSpace())
	}
}
