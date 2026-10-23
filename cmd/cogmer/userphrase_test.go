package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// A frozen phrase, its words and the user identifier it derives, produced once
// from this code and checked against an HKDF and Ed25519 computed outside Go.
// Every user's identifier depends on the word mapping and on the derivation, so a
// change to either makes every phrase already written down name somebody else.
//
// Never regenerate these to make a test pass. A failure here means the mapping or
// the derivation changed, and the fix is a new version beside the old one.
const (
	frozenPhraseHex   = "00112233445566778899aabbccddeeff"
	frozenPhraseWords = "aardvark Babylon blockade concurrent crumpled equipment framework inception newborn nebula reward publisher spigot tambourine tycoon Yucatan retouch whimsical"
	frozenUserID      = "ed25519:wTGxpC6hYgmAhR0i0_oXXTVhzx-1Upxf4ysmmzsLnTo"
)

func frozenPhrase(t *testing.T) Phrase {
	t.Helper()
	b, err := hex.DecodeString(frozenPhraseHex)
	if err != nil {
		t.Fatal(err)
	}
	var p Phrase
	copy(p[:], b)
	return p
}

func TestTheFrozenPhraseRendersAndDerivesWhatItAlwaysHas(t *testing.T) {
	p := frozenPhrase(t)
	if got := strings.Join(p.Words(), " "); got != frozenPhraseWords {
		t.Fatalf("the words for the frozen phrase changed to %q", got)
	}
	if got := p.UserID(); got != frozenUserID {
		t.Fatalf("the user identifier for the frozen phrase changed to %s", got)
	}
	back, err := ParsePhrase(frozenPhraseWords)
	if err != nil || back != p {
		t.Fatalf("the frozen words no longer parse to the frozen bytes: %v", err)
	}
}

func TestThePhraseListsShareNoWordAndRepeatNone(t *testing.T) {
	seen := map[string]string{}
	for name, list := range map[string]*[256]string{"sasEven": &sasEven, "sasOdd": &sasOdd} {
		for i, w := range list {
			k := strings.ToLower(w)
			if where, dup := seen[k]; dup {
				t.Errorf("%s[%d] %q is also %s; parsing cannot tell the two apart", name, i, w, where)
			}
			seen[k] = fmt.Sprintf("%s[%d]", name, i)
		}
	}
}

func TestAPhraseRoundTripsThroughItsWords(t *testing.T) {
	for range 200 {
		p, err := NewPhrase()
		if err != nil {
			t.Fatal(err)
		}
		w := p.Words()
		if len(w) != 18 {
			t.Fatalf("got %d words, want 18", len(w))
		}
		// Typed loosely: mixed case and uneven whitespace still parse.
		typed := "  " + strings.ToUpper(strings.Join(w[:9], "\t")) + " \n " + strings.ToLower(strings.Join(w[9:], "   ")) + "\n"
		back, err := ParsePhrase(typed)
		if err != nil {
			t.Fatalf("a phrase's own words did not parse: %v", err)
		}
		if back != p {
			t.Fatal("a phrase's words parsed to different bytes")
		}
	}
}

func TestNewPhraseDrawsADifferentPhraseEachTime(t *testing.T) {
	seen := map[Phrase]bool{}
	for range 50 {
		p, err := NewPhrase()
		if err != nil {
			t.Fatal(err)
		}
		if seen[p] {
			t.Fatal("NewPhrase returned a phrase it had already returned")
		}
		seen[p] = true
	}
}

func TestTheMasterKeyIsTheSameEveryTimeAndDiffersByPhrase(t *testing.T) {
	p := frozenPhrase(t)
	a, b := p.MasterKey(), p.MasterKey()
	if !bytes.Equal(a, b) {
		t.Fatal("one phrase derived two master keys")
	}
	if len(a) != ed25519.PrivateKeySize {
		t.Fatalf("master key is %d bytes", len(a))
	}
	q := p
	q[15] ^= 1
	if bytes.Equal(q.MasterKey(), a) || q.UserID() == p.UserID() {
		t.Fatal("two phrases derived one master key")
	}
}

// Every error from ParsePhrase names a position and quotes nothing, because an
// error is shown, logged and pasted, and a word in it is part of a phrase.
func TestAPhraseThatDoesNotParseSaysWhereAndQuotesNoWord(t *testing.T) {
	good := strings.Fields(frozenPhraseWords)
	with := func(f func(w []string) []string) string {
		w := append([]string{}, good...)
		return strings.Join(f(w), " ")
	}
	cases := []struct {
		name, input, want string
	}{
		{"a word in no list", with(func(w []string) []string { w[4] = "zebra"; return w }), "word 5 "},
		{"an even word at an odd position", with(func(w []string) []string { w[3] = "Algol"; return w }), "word 4 "},
		{"a swapped pair", with(func(w []string) []string { w[6], w[7] = w[7], w[6]; return w }), "word 7 "},
		{"a missing word", with(func(w []string) []string { return append(w[:2], w[3:]...) }), "has 17 words, want 18"},
		{"an extra word", with(func(w []string) []string { return append(w, "aardvark") }), "has 19 words, want 18"},
		{"a wrong word in the right list", with(func(w []string) []string { w[0] = "absurd"; return w }), "check words do not match"},
		{"a bad check word", with(func(w []string) []string { w[17] = "adviser"; return w }), "check words do not match"},
		{"empty", "", "has 0 words, want 18"},
	}
	for _, c := range cases {
		_, err := ParsePhrase(c.input)
		if err == nil {
			t.Errorf("%s: parsed", c.name)
			continue
		}
		msg := strings.ToLower(err.Error())
		if !strings.Contains(msg, c.want) {
			t.Errorf("%s: error %q does not say %q", c.name, err, c.want)
		}
		for _, w := range strings.Fields(c.input) {
			if strings.Contains(msg, strings.ToLower(w)) {
				t.Errorf("%s: the error quotes a word of the input", c.name)
			}
		}
	}
}

// A phrase printed or marshalled by accident is a copy held by whoever reads the
// output, so neither path may yield its words or its bytes.
func TestAPhraseNeverPrintsOrMarshalsItself(t *testing.T) {
	p := frozenPhrase(t)
	words := p.Words()
	holder := struct{ P Phrase }{p}
	for _, out := range []string{
		fmt.Sprint(p), fmt.Sprintf("%v", p), fmt.Sprintf("%+v", holder), fmt.Sprintf("%#v", p),
		fmt.Sprintf("%s", p), fmt.Sprintf("%q", p),
	} {
		if strings.Contains(out, frozenPhraseHex) || strings.Contains(out, "0x0") {
			t.Errorf("printing shows the phrase's bytes: %q", out)
		}
		for _, w := range words {
			if strings.Contains(strings.ToLower(out), strings.ToLower(w)) {
				t.Errorf("printing shows a word of the phrase: %q", out)
			}
		}
		if !strings.Contains(out, "[recovery phrase]") {
			t.Errorf("printing does not show the redacted text: %q", out)
		}
	}
	// %x hex-encodes what String returns, so it shows the redacted text in hex.
	if out := fmt.Sprintf("%x", p); strings.Contains(out, frozenPhraseHex) {
		t.Errorf("%%x shows the phrase's bytes: %q", out)
	}
	if b, err := json.Marshal(p); err == nil {
		t.Errorf("a phrase marshalled to %s", b)
	}
	if b, err := json.Marshal(holder); err == nil {
		t.Errorf("a struct holding a phrase marshalled to %s", b)
	}
}
