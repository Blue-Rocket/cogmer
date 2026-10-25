package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Cloud sessions (D-216, D-245, D-246, D-247): the value a machine's page shows, the
// session key a cloud session's daemon certifies and renews, and the user's lists it
// fetches from one of the user's machines under a header that machine signs.

// addCloudEnv adds a cloud environment named name on a through its page, and returns
// the value the page showed.
func addCloudEnv(t *testing.T, a *machine, name string) string {
	t.Helper()
	var res cloudAddResponse
	a.local(t, "/machines/cloud/add", cloudAddRequest{Name: name}, &res)
	if res.Error != "" || !strings.HasPrefix(res.Value, cloudValuePrefix) {
		t.Fatalf("the page added no cloud environment: %+v", res.Error)
	}
	return res.Value
}

// cloudMachine starts a fresh daemon as a cloud session's, from value.
func newCloudMachine(t *testing.T, value string) *machine {
	t.Helper()
	c := newMachine(t)
	c.d.startCloud(value)
	if c.d.cloud == nil {
		t.Fatal("the daemon did not become a cloud session's")
	}
	return c
}

// secretForms is every way a private key could be written into a file or a log.
func secretForms(key ed25519.PrivateKey) []string {
	var out []string
	for _, enc := range []*base64.Encoding{base64.RawURLEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.StdEncoding} {
		out = append(out, enc.EncodeToString(key.Seed()), enc.EncodeToString(key))
	}
	return append(out, hex.EncodeToString(key.Seed()))
}

// filesHolding lists the files under dir that hold any of forms, other than skip.
func filesHolding(t *testing.T, dir string, forms []string, skip ...string) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return nil
		}
		for _, s := range skip {
			if filepath.Base(p) == s {
				return nil
			}
		}
		raw, _ := os.ReadFile(p)
		for _, f := range forms {
			if bytes.Contains(raw, []byte(f)) {
				out = append(out, p)
				break
			}
		}
		return nil
	})
	return out
}

func TestACloudValueRoundTripsAndTheMachineThatMadeItKeepsNoKey(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	a, _ := twoMachines(t)
	value := addCloudEnv(t, a, "web")
	setup, err := parseCloudValue(value)
	if err != nil {
		t.Fatalf("the page's value does not read: %v", err)
	}
	if setup.user != a.d.id.User || setup.cloud.Name != "web" || setup.cloud.Kind != KindCloud || setup.displayName != "Dana" {
		t.Errorf("the value reads as %s, named %q, kind %q, display %q", setup, setup.cloud.Name, setup.cloud.Kind, setup.displayName)
	}
	if len(setup.machines) != 2 {
		t.Errorf("the value names %d machines, want both of the user's", len(setup.machines))
	}
	again, err := encodeCloudValue(setup.key, setup.shared, setup.cloud, setup.displayName, setup.machines)
	if err != nil || again != value {
		t.Errorf("the value does not encode back to itself: %v", err)
	}
	// Listed, and propagated, like any machine (D-246).
	certs, _ := a.d.members.displayCerts(a.d.id.User)
	if c := certs[setup.cloud.Machine]; c.Kind != KindCloud || c.Name != "web" {
		t.Errorf("the cloud key is listed as %+v", c)
	}
	var st identityState
	a.local(t, "/identity/state", struct{}{}, &st)
	found := false
	for _, m := range st.Machines {
		found = found || (m.Machine == setup.cloud.Machine && m.Kind == KindCloud)
	}
	if !found {
		t.Error("the machines page does not list the cloud environment")
	}
	// The machine keeps the certificate and not the key, and no log holds it.
	if files := filesHolding(t, a.home, secretForms(setup.key)); len(files) > 0 {
		t.Errorf("the machine that made the value keeps its key in %v", files)
	}
	for _, f := range append(secretForms(setup.key), value, strings.TrimPrefix(value, cloudValuePrefix)) {
		if strings.Contains(logs.String(), f) {
			t.Fatal("the daemon's log holds the cloud key")
		}
	}
	// A machine never gives a cloud key the shared key (D-219).
	if g := a.d.giftFor(setup.cloud.Machine, 0); g != nil {
		t.Error("the machine would give the cloud key its shared key")
	}
}

func TestAMalformedCloudValueIsRefusedWithoutQuotingIt(t *testing.T) {
	a, _ := twoMachines(t)
	value := addCloudEnv(t, a, "web")
	setup, err := parseCloudValue(value)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, cloudValuePrefix))
	rewrite := func(change func(*cloudValueWire)) string {
		var w cloudValueWire
		if err := json.Unmarshal(body, &w); err != nil {
			t.Fatal(err)
		}
		change(&w)
		raw, _ := json.Marshal(w)
		return cloudValuePrefix + base64.RawURLEncoding.EncodeToString(raw)
	}
	_, other, _ := ed25519.GenerateKey(nil)
	for name, bad := range map[string]string{
		"empty":            "",
		"another prefix":   "cogmer-cloud-v9:" + strings.TrimPrefix(value, cloudValuePrefix),
		"not base64url":    cloudValuePrefix + "%%%",
		"not JSON":         cloudValuePrefix + base64.RawURLEncoding.EncodeToString([]byte("[1,2]")),
		"cut short":        value[:len(value)-20],
		"another key":      rewrite(func(w *cloudValueWire) { w.Key = base64.RawURLEncoding.EncodeToString(other.Seed()) }),
		"a renamed cert":   rewrite(func(w *cloudValueWire) { w.Cloud.Name = "elsewhere" }),
		"a machine's cert": rewrite(func(w *cloudValueWire) { w.Cloud.Kind = KindMachine }),
		"no machines":      rewrite(func(w *cloudValueWire) { w.Machines = nil }),
		"an unknown field": cloudValuePrefix + base64.RawURLEncoding.EncodeToString(append(append([]byte{}, body[:len(body)-1]...), []byte(`,"extra":1}`)...)),
	} {
		_, err := parseCloudValue(bad)
		if err == nil {
			t.Errorf("%s: a malformed value was taken", name)
			continue
		}
		for _, f := range append(secretForms(setup.key), strings.TrimPrefix(bad, cloudValuePrefix)) {
			if len(f) > 8 && strings.Contains(err.Error(), f) {
				t.Errorf("%s: the refusal quotes the value: %v", name, err)
			}
		}
	}

	// A daemon given a value that does not read shares nothing, says so without the
	// value, and never takes the phrase.
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	c := newMachine(t)
	c.d.startCloud(value[:len(value)-20])
	if c.d.cloud != nil || c.d.id.HasUser() {
		t.Fatal("a daemon took a malformed value")
	}
	if strings.Contains(logs.String(), value[len(cloudValuePrefix):len(cloudValuePrefix)+30]) {
		t.Error("the log quotes the malformed value")
	}
	var ph phraseResponse
	c.local(t, "/identity/phrase", struct{}{}, &ph)
	if ph.Error == "" || len(ph.Words) > 0 {
		t.Error("a cloud session's page drew a recovery phrase")
	}
}

func TestACloudSessionCertifiesAKeyOfItsOwnAndRenewsItBeforeItExpires(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	a, _ := twoMachines(t)
	value := addCloudEnv(t, a, "web")
	setup, _ := parseCloudValue(value)
	c := newCloudMachine(t, value)
	id := c.d.id
	if !id.HasUser() || id.User != a.d.id.User || id.Chain.Session == nil || id.MachineID != "web" {
		t.Fatalf("the cloud session's identity is %+v", id)
	}
	link, err := id.Chain.Verify(id.User, id.PeerID)
	if err != nil || link.Machine != setup.cloud.Machine || link.Signer != id.PeerID {
		t.Fatalf("the session's chain: %+v %v", link, err)
	}
	exp, _ := time.Parse(time.RFC3339, link.Expires)
	if d := time.Until(exp); d < 23*time.Hour || d > 24*time.Hour {
		t.Errorf("the session key is certified for %s, want 24h (D-247)", d)
	}
	if id.PeerID == setup.cloud.Machine {
		t.Error("the session signs with the cloud key itself")
	}
	// The session key is in session.key alone, and a command reads it from there.
	c.use(t)
	t.Setenv("COGMER_HOME", c.d.home)
	loaded, err := LoadIdentity()
	if err != nil || loaded.PeerID != id.PeerID || !loaded.HasUser() || !bytes.Equal(loaded.private, id.private) {
		t.Fatalf("a command does not read the session's identity: %v", err)
	}
	whoami, _ := json.Marshal(loaded)
	secrets := append(secretForms(id.private), secretForms(setup.key)...)
	for _, f := range secrets {
		if bytes.Contains(whoami, []byte(f)) {
			t.Fatal("the identity a command prints holds a private key")
		}
	}
	if files := filesHolding(t, c.home, secrets, "session.key"); len(files) > 0 {
		t.Errorf("a private key of the cloud session is in %v", files)
	}
	if files := filesHolding(t, c.home, secretForms(setup.key)); len(files) > 0 {
		t.Errorf("the cloud key is written to %v", files)
	}

	// Kept while it has more than two hours to run, and replaced by a new key once it
	// has less, with a new 24 hours.
	c.d.cloud.nextFetch = time.Now().Add(48 * time.Hour)
	c.d.cloudRound(time.Now().Add(20 * time.Hour))
	if c.d.id.PeerID != id.PeerID {
		t.Fatal("a session key with four hours to run was replaced")
	}
	c.d.cloudRound(time.Now().Add(23 * time.Hour))
	renewed := c.d.id
	if renewed.PeerID == id.PeerID || renewed.Chain.Session.Session != renewed.PeerID {
		t.Fatal("a session key with an hour to run was not replaced by a new key")
	}
	exp2, _ := time.Parse(time.RFC3339, renewed.Chain.Session.Expires)
	if exp2.Sub(exp) < 22*time.Hour {
		t.Errorf("the new key is certified until %s, the old until %s", exp2, exp)
	}
	if k, ok := loadSessionKeyIn(c.d.home); !ok || PeerIDFromPublic(k.Public().(ed25519.PublicKey)) != renewed.PeerID {
		t.Error("session.key does not hold the new key")
	}
	for _, f := range append(secretForms(id.private), append(secretForms(renewed.private), secretForms(setup.key)...)...) {
		if strings.Contains(logs.String(), f) {
			t.Fatal("the daemon's log holds a private key")
		}
	}
}

func TestEachSessionKeyNumbersItsOwnEventsFromOne(t *testing.T) {
	m := newMachine(t)
	if err := m.d.members.RecordRoom("room-a", "misty-canyon", false); err != nil {
		t.Fatal(err)
	}
	for want := int64(1); want <= 3; want++ {
		if got, err := m.d.members.ReserveSequence("room-a", "ed25519:first"); err != nil || got != want {
			t.Fatalf("the first key reserved %d (%v), want %d", got, err, want)
		}
	}
	if got, _ := m.d.members.ReserveSequence("room-a", "ed25519:second"); got != 1 {
		t.Errorf("a new key reserved %d, want 1, so that it leaves no gap behind it", got)
	}
	if got := m.d.members.IssuedSequenceFor("room-a", "ed25519:first"); got != 0 {
		t.Errorf("the count is still the first key's: %d", got)
	}
}

func TestAReceiverRefusesAnExpiredSessionAndARemovedCloudKey(t *testing.T) {
	a, _ := twoMachines(t)
	value := addCloudEnv(t, a, "web")
	c := newCloudMachine(t, value)
	chain := *c.d.id.Chain
	user, sess := c.d.id.User, c.d.id.PeerID
	if _, err := a.d.members.checkLiveChain(user, sess, chain, time.Now()); err != nil {
		t.Fatalf("a live session is refused: %v", err)
	}
	if _, err := a.d.members.checkLiveChain(user, sess, chain, time.Now().Add(25*time.Hour)); err == nil {
		t.Error("an expired session is accepted for a live act")
	}
	// An event the session signed still verifies after its certificate expires,
	// since a relayed event is the session's whenever it arrives (D-058).
	ev := &Event{EventID: "e1", PeerID: user, PeerSequence: 1, RoomID: "r", Timestamp: "2026-10-25T00:00:00Z", EventType: "PROMPT", Content: "hi"}
	ev.SignV4(c.d.id.private, chain, nil)
	if err := ev.Verify(); err != nil {
		t.Fatalf("a session's event does not verify: %v", err)
	}

	setup, _ := parseCloudValue(value)
	var res identityResult
	a.local(t, "/machines/remove", removeRequest{Machine: setup.cloud.Machine, Stolen: true}, &res)
	if res.Status != "removed" {
		t.Fatalf("removing the cloud key: %+v", res)
	}
	if _, err := a.d.members.checkLiveChain(user, sess, chain, time.Now()); err == nil || !strings.Contains(err.Error(), "removed") {
		t.Errorf("a session of a removed cloud key is accepted: %v", err)
	}
	if st := a.d.members.UserState(user, &chain); st.Counts(setup.cloud.Machine, "r", 1) {
		t.Error("a stolen cloud key's session event still counts")
	}
}

func TestACloudSessionTakesItsUsersListsFromAMachine(t *testing.T) {
	a, b := twoMachines(t)
	carol := testIdentity(t).PeerID
	if err := a.d.members.Allow(carol, "carol"); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.MarkVerified(carol); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.RecordColleague(a.d.id, carol); err != nil {
		t.Fatal(err)
	}
	value := addCloudEnv(t, a, "web")
	c := newCloudMachine(t, value)
	if c.d.members.IsVerified(carol) {
		t.Fatal("the session knows carol before it fetched anything")
	}
	if err := c.d.fetchUserLists(); err != nil {
		t.Fatalf("fetching the lists: %v", err)
	}
	if !c.d.members.IsVerified(carol) || c.d.members.Label(carol) != "carol" {
		t.Errorf("the session does not know carol as the user does: verified %v, label %q",
			c.d.members.IsVerified(carol), c.d.members.Label(carol))
	}
	if ep, _ := c.d.members.MachineEndpoint(b.d.id.PeerID); ep != b.addr {
		t.Errorf("the session learned %q as the address of the user's second machine, want %q", ep, b.addr)
	}
	v := c.d.cloudListsView()
	if v == nil || v.From == "" || v.Age == "" {
		t.Errorf("the view says nothing of the lists' age: %+v", v)
	}
	held := c.d.members.listsHeaderHeld(c.d.cloud.header.Machine)
	if held == 0 {
		t.Fatal("the session kept no header version")
	}

	// A header lower than one already taken from that machine is refused.
	first := *c.d.cloud.header
	if first.Machine != a.d.id.PeerID && first.Machine != b.d.id.PeerID {
		t.Fatalf("the header is signed by %s", first.Machine)
	}
	signer := a
	if first.Machine == b.d.id.PeerID {
		signer = b
	}
	older := SignListsHeader(signer.d.id.private, signer.d.id.User, held-1, time.Now())
	if err := c.d.takeLists(selfFetchResponse{Header: older, Chain: *signer.d.id.Chain}, signer.d.id.PeerID); err == nil {
		t.Error("an older header was taken")
	}
	// A cloud session writes nothing to the user's lists, and a machine of the user
	// takes no change from one.
	if err := c.d.members.writeListChange(c.d.id, "colleague:"+carol+":label", "mallory"); err != nil {
		t.Fatal(err)
	}
	if _, changes, _ := c.d.members.IdentityRecords(c.d.id.User); func() bool {
		for _, ch := range changes {
			if ch.Signer == c.d.id.PeerID {
				return true
			}
		}
		return false
	}() {
		t.Error("the cloud session wrote a change to the user's lists")
	}
	if err := c.d.exchangeWith(a.d.id.PeerID, a.addr); err == nil {
		t.Error("a machine exchanged the user's lists with a cloud session")
	}
}

func TestAMachinesListsVersionRisesOnlyWhenTheListsChange(t *testing.T) {
	a, _ := twoMachines(t)
	v1, err := a.d.members.listsVersion(a.d.id.User)
	if err != nil || v1 == 0 {
		t.Fatalf("version %d, %v", v1, err)
	}
	if v, _ := a.d.members.listsVersion(a.d.id.User); v != v1 {
		t.Errorf("the version rose from %d to %d with nothing changed", v1, v)
	}
	if err := a.d.members.writeListChange(a.d.id, "colleague:"+testIdentity(t).PeerID+":label", "erin"); err != nil {
		t.Fatal(err)
	}
	if v, _ := a.d.members.listsVersion(a.d.id.User); v != v1+1 {
		t.Errorf("the version is %d after a change, want %d", v, v1+1)
	}
}

func TestTheFetchRouteRefusesAnotherUsersChainAndARemovedCloudKey(t *testing.T) {
	a, _ := twoMachines(t)
	value := addCloudEnv(t, a, "web")

	// Another user's machine, whose chain verifies, is refused.
	e := newMachine(t)
	var p Phrase
	for i := range p {
		p[i] = byte(0x10 + i)
	}
	createUser(t, e, p, "Erin", "erin-laptop")
	if err := e.d.fetchListsFrom(cloudMachine{Machine: a.d.id.PeerID, Endpoint: a.addr}); err == nil {
		t.Error("a machine answered another user's machine with its user's lists")
	}

	c := newCloudMachine(t, value)
	target := cloudMachine{Machine: a.d.id.PeerID, Endpoint: a.addr}
	if err := c.d.fetchListsFrom(target); err != nil {
		t.Fatalf("the user's own cloud session was refused: %v", err)
	}
	setup, _ := parseCloudValue(value)
	var res identityResult
	a.local(t, "/machines/remove", removeRequest{Machine: setup.cloud.Machine, Stolen: true}, &res)
	if res.Status != "removed" {
		t.Fatalf("removing the cloud key: %+v", res)
	}
	if err := c.d.fetchListsFrom(target); err == nil {
		t.Error("a machine answered a session of a removed cloud key")
	}
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// The frozen header, for the reason vectors_test.go gives: a header a session already
// holds was signed over these bytes.
func TestFrozenListsHeaderStillVerifiesAndHashesAsItDid(t *testing.T) {
	k := testIDKeys(t)
	raw, err := os.ReadFile("testdata/identity_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Keys        struct{ Master, Machine string } `json:"keys"`
		ListsHeader struct {
			Record ListsHeader `json:"record"`
			Hash   string      `json:"hash"`
		} `json:"listsHeader"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	h := v.ListsHeader.Record
	if err := h.VerifyBy(k.machine.pub); err != nil {
		t.Errorf("the frozen lists header no longer verifies: %v; the bytes scheme v1 signs have changed; add a scheme and leave v1 as it was (D-058, signature schemes are kept, never replaced)", err)
	}
	got := h.Hash()
	if hex.EncodeToString(got[:]) != v.ListsHeader.Hash {
		t.Errorf("the frozen lists header hashes to %x, was %s", got, v.ListsHeader.Hash)
	}
	if h != SignListsHeader(k.machine.priv, k.master.id, 12, frozenExpiry) {
		t.Error("signing the frozen header's fields again gives another header")
	}
	var b []byte
	str := func(s string) { b = append(binary.BigEndian.AppendUint32(b, uint32(len(s))), s...) }
	str("peer-room/lists-header/v1")
	str(v.Keys.Master)
	str(v.Keys.Machine)
	b = binary.BigEndian.AppendUint64(b, 12)
	str("2026-11-01T00:00:00Z")
	if s := sha256Hex(b); s != v.ListsHeader.Hash {
		t.Error("the frozen header is not a hash of user, machine, version and time under its tag")
	}
	// The check above cannot fail if verification accepts anything.
	h.Version++
	if h.VerifyBy(k.machine.pub) == nil {
		t.Error("an altered frozen header verified")
	}
}
