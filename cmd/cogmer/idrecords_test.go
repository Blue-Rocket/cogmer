package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Fixed seeds, so the frozen identity vectors can be checked against the keys
// that made them.
const (
	seedMasterHex  = "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1"
	seedSharedHex  = "b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2"
	seedMachineHex = "c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3c3"
	seedCloudHex   = "d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4d4"
	seedSessionHex = "e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5e5"
	seedOtherHex   = "f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6f6"
)

var frozenExpiry = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

type idKey struct {
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
	id   string
}

func keyFromSeed(t *testing.T, seedHex string) idKey {
	t.Helper()
	seed, err := hex.DecodeString(seedHex)
	if err != nil {
		t.Fatal(err)
	}
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)
	return idKey{priv, pub, PeerIDFromPublic(pub)}
}

type idKeys struct{ master, shared, machine, cloud, session, other idKey }

func testIDKeys(t *testing.T) idKeys {
	return idKeys{
		master:  keyFromSeed(t, seedMasterHex),
		shared:  keyFromSeed(t, seedSharedHex),
		machine: keyFromSeed(t, seedMachineHex),
		cloud:   keyFromSeed(t, seedCloudHex),
		session: keyFromSeed(t, seedSessionHex),
		other:   keyFromSeed(t, seedOtherHex),
	}
}

func (k idKeys) sharedCert() SharedKeyCert { return CertifySharedKey(k.master.priv, 3, k.shared.pub) }

func (k idKeys) machineCert() MachineCert {
	return CertifyMachine(k.shared.priv, k.master.id, 3, k.machine.pub, KindMachine, "laptop Ünïcode", 7)
}

func (k idKeys) cloudCert() MachineCert {
	return CertifyMachine(k.shared.priv, k.master.id, 3, k.cloud.pub, KindCloud, "cloud", 8)
}

func (k idKeys) sessionCert() SessionCert {
	return CertifySession(k.cloud.priv, k.master.id, k.session.pub, frozenExpiry)
}

func (k idKeys) removal(t *testing.T) Removal {
	t.Helper()
	r, err := SignRemoval(k.shared.priv, LevelShared, 3, k.master.id, k.machine.id,
		[]RoomCutoff{{"room-b", 9}, {"room-a", 5}})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func (k idKeys) listChange() ListChange {
	return SignListChange(k.machine.priv, k.master.id, "colleague:"+k.other.id, `{"label":"Ana Ünïcode","verified":true}`, 4)
}

func (k idKeys) machineChain() Chain { return Chain{Shared: k.sharedCert(), Machine: k.machineCert()} }

func (k idKeys) cloudChain() Chain {
	s := k.sessionCert()
	return Chain{Shared: k.sharedCert(), Machine: k.cloudCert(), Session: &s}
}

func roundTrip[T any](t *testing.T, v T) T {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestEveryIdentityRecordVerifiesAfterAJSONRoundTrip(t *testing.T) {
	k := testIDKeys(t)

	sc := k.sharedCert()
	if sc.Scheme != 1 {
		t.Errorf("a shared-key certificate is signed under scheme %d, want 1", sc.Scheme)
	}
	if got := roundTrip(t, sc); got.Verify() != nil || got.Hash() != sc.Hash() {
		t.Errorf("shared-key certificate after JSON: verify %v, hash changed %v", got.Verify(), got.Hash() != sc.Hash())
	}

	mc := k.machineCert()
	if got := roundTrip(t, mc); got.VerifyBy(k.shared.pub) != nil || got.Hash() != mc.Hash() {
		t.Errorf("machine certificate after JSON: verify %v", got.VerifyBy(k.shared.pub))
	}

	ss := k.sessionCert()
	if ss.Expires != "2026-11-01T00:00:00Z" {
		t.Errorf("session expiry is written %q, want RFC 3339 in UTC", ss.Expires)
	}
	if got := roundTrip(t, ss); got.VerifyBy(k.cloud.pub) != nil || got.Hash() != ss.Hash() {
		t.Errorf("session certificate after JSON: verify %v", got.VerifyBy(k.cloud.pub))
	}

	r := k.removal(t)
	if got := roundTrip(t, r); got.VerifyBy(k.shared.pub) != nil || got.Hash() != r.Hash() {
		t.Errorf("removal after JSON: verify %v", got.VerifyBy(k.shared.pub))
	}

	for name, c := range map[string]Chain{"machine": k.machineChain(), "cloud": k.cloudChain()} {
		signer := k.machine.id
		if name == "cloud" {
			signer = k.session.id
		}
		if _, err := roundTrip(t, c).Verify(k.master.id, signer); err != nil {
			t.Errorf("%s chain after JSON: %v", name, err)
		}
	}
}

func TestTheHashOfARecordIsItsIdentityAndDiffersByRecord(t *testing.T) {
	k := testIDKeys(t)
	hashes := map[[32]byte]string{}
	for name, h := range map[string][32]byte{
		"shared": k.sharedCert().Hash(), "machine": k.machineCert().Hash(), "cloud": k.cloudCert().Hash(),
		"session": k.sessionCert().Hash(), "removal": k.removal(t).Hash(),
	} {
		if h == ([32]byte{}) {
			t.Errorf("%s hashes to zero", name)
		}
		if prev, ok := hashes[h]; ok {
			t.Errorf("%s and %s share a hash", name, prev)
		}
		hashes[h] = name
	}
	if k.sharedCert().Hash() != k.sharedCert().Hash() {
		t.Error("signing one certificate twice gave two hashes")
	}
}

// Each case changes one signed field and keeps the record well formed, so the
// signature is what must refuse it. A field left out of the signing bytes passes.
func wantSignatureRefusal(t *testing.T, what string, err error) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: a tampered record verified, so that field is not signed", what)
	} else if !strings.Contains(err.Error(), "does not match") {
		t.Errorf("%s: refused for %q, not by its signature", what, err)
	}
}

func TestATamperedFieldOfASharedKeyCertFailsVerification(t *testing.T) {
	// The user is also the key Verify checks against, so changing it changes the
	// key as well and fails whether or not it is signed. The frozen vectors are
	// what hold it in the signing bytes.
	k := testIDKeys(t)
	for name, change := range map[string]func(*SharedKeyCert){
		"version":    func(c *SharedKeyCert) { c.Version = 4 },
		"shared key": func(c *SharedKeyCert) { c.SharedKey = k.other.id },
	} {
		c := k.sharedCert()
		change(&c)
		wantSignatureRefusal(t, name, c.Verify())
	}
}

func TestATamperedFieldOfAMachineCertFailsVerification(t *testing.T) {
	k := testIDKeys(t)
	for name, change := range map[string]func(*MachineCert){
		"user":           func(c *MachineCert) { c.User = k.other.id },
		"shared version": func(c *MachineCert) { c.SharedVersion = 4 },
		"machine":        func(c *MachineCert) { c.Machine = k.other.id },
		"kind":           func(c *MachineCert) { c.Kind = KindCloud },
		"name":           func(c *MachineCert) { c.Name = "laptop Ünïcodf" },
		"serial":         func(c *MachineCert) { c.Serial = 8 },
	} {
		c := k.machineCert()
		change(&c)
		wantSignatureRefusal(t, name, c.VerifyBy(k.shared.pub))
	}
}

func TestATamperedFieldOfASessionCertFailsVerification(t *testing.T) {
	k := testIDKeys(t)
	for name, tc := range map[string]struct {
		change func(*SessionCert)
		by     ed25519.PublicKey
	}{
		"user":    {func(c *SessionCert) { c.User = k.other.id }, k.cloud.pub},
		"cloud":   {func(c *SessionCert) { c.CloudKey = k.other.id }, k.other.pub},
		"session": {func(c *SessionCert) { c.Session = k.other.id }, k.cloud.pub},
		"expires": {func(c *SessionCert) { c.Expires = "2026-11-02T00:00:00Z" }, k.cloud.pub},
	} {
		c := k.sessionCert()
		tc.change(&c)
		wantSignatureRefusal(t, name, c.VerifyBy(tc.by))
	}
}

func TestATamperedFieldOfARemovalFailsVerification(t *testing.T) {
	k := testIDKeys(t)
	master, err := SignRemoval(k.master.priv, LevelMaster, 0, k.master.id, k.machine.id, nil)
	if err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		base   Removal
		change func(*Removal)
		by     ed25519.PublicKey
	}{
		"user":           {k.removal(t), func(r *Removal) { r.User = k.other.id }, k.shared.pub},
		"machine":        {k.removal(t), func(r *Removal) { r.Machine = k.other.id }, k.shared.pub},
		"signer level":   {master, func(r *Removal) { r.SignerLevel = LevelMachine }, k.master.pub},
		"signer":         {k.removal(t), func(r *Removal) { r.Signer = k.other.id }, k.other.pub},
		"signer version": {k.removal(t), func(r *Removal) { r.SignerVersion = 4 }, k.shared.pub},
		"cutoff room":    {k.removal(t), func(r *Removal) { r.Cutoffs[0].Room = "room-a2" }, k.shared.pub},
		"cutoff seq":     {k.removal(t), func(r *Removal) { r.Cutoffs[1].Sequence = 10 }, k.shared.pub},
		"cutoff dropped": {k.removal(t), func(r *Removal) { r.Cutoffs = r.Cutoffs[:1] }, k.shared.pub},
		"cutoff added": {k.removal(t), func(r *Removal) {
			r.Cutoffs = append(r.Cutoffs, RoomCutoff{"room-c", 1})
		}, k.shared.pub},
	} {
		r := tc.base
		r.Cutoffs = append([]RoomCutoff{}, tc.base.Cutoffs...)
		tc.change(&r)
		wantSignatureRefusal(t, name, r.VerifyBy(tc.by))
	}
}

func TestARecordCheckedAgainstTheWrongKeyFails(t *testing.T) {
	k := testIDKeys(t)
	forged := CertifySharedKey(k.other.priv, 3, k.shared.pub)
	forged.User = k.master.id
	if forged.Verify() == nil {
		t.Error("a shared-key certificate signed by another key verified as the master's")
	}
	if k.machineCert().VerifyBy(k.other.pub) == nil {
		t.Error("a machine certificate verified against a key that did not sign it")
	}
	if k.sessionCert().VerifyBy(k.other.pub) == nil {
		t.Error("a session certificate verified against a key that did not sign it")
	}
	if k.removal(t).VerifyBy(k.other.pub) == nil {
		t.Error("a removal verified against a key that did not sign it")
	}
}

func TestAnIdentityRecordUnderAnUnknownSchemeIsRefusedAsAVersionProblem(t *testing.T) {
	k := testIDKeys(t)
	for _, scheme := range []int{0, 2} {
		sc, mc, ss, r, lc := k.sharedCert(), k.machineCert(), k.sessionCert(), k.removal(t), k.listChange()
		sc.Scheme, mc.Scheme, ss.Scheme, r.Scheme, lc.Scheme = scheme, scheme, scheme, scheme, scheme
		for name, err := range map[string]error{
			"shared-key certificate": sc.Verify(),
			"machine certificate":    mc.VerifyBy(k.shared.pub),
			"session certificate":    ss.VerifyBy(k.cloud.pub),
			"removal":                r.VerifyBy(k.shared.pub),
			"list change":            lc.VerifyBy(k.machine.pub),
		} {
			if err == nil || !strings.Contains(err.Error(), "upgrade") || strings.Contains(err.Error(), "does not match") {
				t.Errorf("a %s under scheme %d: got %v, want a refusal that says to upgrade", name, scheme, err)
			}
		}
		if sc.Hash() != ([32]byte{}) {
			t.Errorf("a record under scheme %d has a hash, which this build cannot compute", scheme)
		}
	}
}

func TestASharedVersionOfZeroIsRefused(t *testing.T) {
	k := testIDKeys(t)
	if err := CertifySharedKey(k.master.priv, 0, k.shared.pub).Verify(); err == nil {
		t.Error("a shared-key certificate for version 0 verified")
	}
	if err := CertifyMachine(k.shared.priv, k.master.id, 0, k.machine.pub, KindMachine, "m", 1).VerifyBy(k.shared.pub); err == nil {
		t.Error("a machine certificate under version 0 verified")
	}
	if _, err := SignRemoval(k.shared.priv, LevelShared, 0, k.master.id, k.machine.id, nil); err == nil {
		t.Error("a shared key's removal under version 0 was signed")
	}
}

func TestSignRemovalSortsCutoffsAndRefusesARoomTwice(t *testing.T) {
	k := testIDKeys(t)
	r := k.removal(t)
	if r.Cutoffs[0].Room != "room-a" || r.Cutoffs[1].Room != "room-b" {
		t.Errorf("cutoffs are %v, want sorted by room", r.Cutoffs)
	}
	again, _ := SignRemoval(k.shared.priv, LevelShared, 3, k.master.id, k.machine.id,
		[]RoomCutoff{{"room-a", 5}, {"room-b", 9}})
	if again.Hash() != r.Hash() {
		t.Error("one set of cutoffs given in two orders gave two hashes")
	}
	if _, err := SignRemoval(k.shared.priv, LevelShared, 3, k.master.id, k.machine.id,
		[]RoomCutoff{{"room-a", 5}, {"room-a", 2}}); err == nil || !strings.Contains(err.Error(), "twice") {
		t.Errorf("a room given twice: got %v", err)
	}
	for _, level := range []Level{0, 4} {
		if _, err := SignRemoval(k.master.priv, level, 0, k.master.id, k.machine.id, nil); err == nil {
			t.Errorf("a removal at level %d was signed", level)
		}
	}
	self, err := SignRemoval(k.machine.priv, LevelMachine, 0, k.master.id, k.machine.id, nil)
	if err != nil || self.VerifyBy(k.machine.pub) != nil {
		t.Errorf("a self-removal naming no room: %v", err)
	}
}

func TestAMalformedRemovalIsRefusedBeforeItsSignature(t *testing.T) {
	k := testIDKeys(t)
	for name, tc := range map[string]struct {
		change func(*Removal)
		want   string
	}{
		"unsorted":      {func(r *Removal) { r.Cutoffs[0], r.Cutoffs[1] = r.Cutoffs[1], r.Cutoffs[0] }, "not sorted"},
		"room twice":    {func(r *Removal) { r.Cutoffs[1].Room = r.Cutoffs[0].Room }, "twice"},
		"unknown level": {func(r *Removal) { r.SignerLevel = 9 }, "not a level"},
		"no room":       {func(r *Removal) { r.Cutoffs[0].Room = "" }, "names no room"},
		"negative":      {func(r *Removal) { r.Cutoffs[0].Sequence = -1 }, "negative"},
	} {
		r := k.removal(t)
		tc.change(&r)
		if err := r.VerifyBy(k.shared.pub); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v, want an error saying %q", name, err, tc.want)
		}
	}
}

func TestAWellFormedChainYieldsWhatItSaysAboutItsKey(t *testing.T) {
	k := testIDKeys(t)
	link, err := k.machineChain().Verify(k.master.id, k.machine.id)
	if err != nil {
		t.Fatal(err)
	}
	want := ChainLink{User: k.master.id, SharedVersion: 3, Machine: k.machine.id, Kind: KindMachine,
		Name: "laptop Ünïcode", Serial: 7, Signer: k.machine.id}
	if link != want {
		t.Errorf("machine chain link is %+v, want %+v", link, want)
	}
	link, err = k.cloudChain().Verify(k.master.id, k.session.id)
	if err != nil {
		t.Fatal(err)
	}
	if link.Machine != k.cloud.id || link.Signer != k.session.id || link.Expires != "2026-11-01T00:00:00Z" {
		t.Errorf("cloud chain link is %+v", link)
	}
}

func TestEveryBrokenLinkOfAChainIsRefusedAndNamed(t *testing.T) {
	k := testIDKeys(t)
	otherMaster := keyFromSeed(t, "0707070707070707070707070707070707070707070707070707070707070707")
	session := k.sessionCert()
	for name, tc := range map[string]struct {
		chain  func() Chain
		user   string
		signer string
		want   string
	}{
		"another user's chain": {k.machineChain, otherMaster.id, k.machine.id, "shared-key certificate"},
		"shared cert forged": {func() Chain {
			c := k.machineChain()
			c.Shared.SharedKey = k.other.id
			return c
		}, k.master.id, k.machine.id, "shared-key certificate"},
		"shared version 0": {func() Chain {
			return Chain{Shared: CertifySharedKey(k.master.priv, 0, k.shared.pub),
				Machine: CertifyMachine(k.shared.priv, k.master.id, 0, k.machine.pub, KindMachine, "m", 1)}
		}, k.master.id, k.machine.id, "shared-key certificate"},
		"machine cert for another user": {func() Chain {
			c := k.machineChain()
			c.Machine = CertifyMachine(k.shared.priv, otherMaster.id, 3, k.machine.pub, KindMachine, "m", 1)
			return c
		}, k.master.id, k.machine.id, "machine certificate"},
		"machine cert under another version": {func() Chain {
			c := k.machineChain()
			c.Machine = CertifyMachine(k.shared.priv, k.master.id, 2, k.machine.pub, KindMachine, "m", 1)
			return c
		}, k.master.id, k.machine.id, "machine certificate is under shared version 2"},
		"machine cert by another shared key": {func() Chain {
			c := k.machineChain()
			c.Machine = CertifyMachine(k.other.priv, k.master.id, 3, k.machine.pub, KindMachine, "m", 1)
			return c
		}, k.master.id, k.machine.id, "machine certificate"},
		"unknown kind": {func() Chain {
			c := k.machineChain()
			c.Machine = CertifyMachine(k.shared.priv, k.master.id, 3, k.machine.pub, "laptop", "m", 1)
			return c
		}, k.master.id, k.machine.id, "machine certificate"},
		"machine key that is not the signer": {k.machineChain, k.master.id, k.other.id, "not at the key that signed"},
		"machine key with a session cert": {func() Chain {
			c := k.machineChain()
			c.Session = &session
			return c
		}, k.master.id, k.session.id, "session certificate under a machine key"},
		"cloud key with no session cert": {func() Chain {
			c := k.cloudChain()
			c.Session = nil
			return c
		}, k.master.id, k.cloud.id, "session certificate missing"},
		"session cert for another user": {func() Chain {
			c := k.cloudChain()
			s := CertifySession(k.cloud.priv, otherMaster.id, k.session.pub, frozenExpiry)
			c.Session = &s
			return c
		}, k.master.id, k.session.id, "session certificate"},
		"session cert from another cloud key": {func() Chain {
			c := k.cloudChain()
			s := CertifySession(k.other.priv, k.master.id, k.session.pub, frozenExpiry)
			c.Session = &s
			return c
		}, k.master.id, k.session.id, "session certificate"},
		"session cert forged": {func() Chain {
			c := k.cloudChain()
			s := *c.Session
			s.Session = k.other.id
			c.Session = &s
			return c
		}, k.master.id, k.other.id, "session certificate"},
		"session key that is not the signer": {k.cloudChain, k.master.id, k.cloud.id, "not at the key that signed"},
	} {
		_, err := tc.chain().Verify(tc.user, tc.signer)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v, want an error naming %q", name, err, tc.want)
		}
	}
}

func TestAnExpiredSessionStillVerifiesButIsNotLive(t *testing.T) {
	k := testIDKeys(t)
	link, err := k.cloudChain().Verify(k.master.id, k.session.id)
	if err != nil {
		t.Fatalf("a chain whose session expired in the past must still verify for relayed events: %v", err)
	}
	if err := link.LiveAt(frozenExpiry.Add(-time.Second)); err != nil {
		t.Errorf("a session a second before its expiry: %v", err)
	}
	for _, at := range []time.Time{frozenExpiry, frozenExpiry.Add(time.Hour)} {
		if link.LiveAt(at) == nil {
			t.Errorf("a session at %s, at or after its expiry, is live", at)
		}
	}
	link.Expires = "next tuesday"
	if link.LiveAt(frozenExpiry.Add(-time.Hour)) == nil {
		t.Error("a malformed expiry was read as live")
	}
	mlink, _ := k.machineChain().Verify(k.master.id, k.machine.id)
	if err := mlink.LiveAt(time.Now()); err != nil {
		t.Errorf("a machine key has no expiry, yet: %v", err)
	}
}

func TestNoPrivateKeyAppearsInARecordOrAnError(t *testing.T) {
	k := testIDKeys(t)
	var text strings.Builder
	for _, v := range []any{k.sharedCert(), k.machineCert(), k.sessionCert(), k.removal(t), k.cloudChain()} {
		raw, _ := json.Marshal(v)
		text.Write(raw)
	}
	_, err := k.cloudChain().Verify(k.master.id, k.cloud.id)
	text.WriteString(err.Error())
	for _, key := range []idKey{k.master, k.shared, k.machine, k.cloud, k.session} {
		for _, enc := range []*base64.Encoding{base64.RawURLEncoding, base64.RawStdEncoding} {
			if strings.Contains(text.String(), enc.EncodeToString(key.priv.Seed())) {
				t.Error("a private key's seed appears in a record or an error")
			}
		}
	}
}

// Frozen identity records, produced once from the seeds above and checked in as
// testdata/identity_vectors.json, for the reason vectors_test.go gives: a test
// that signs and then verifies cannot see a change to the signing bytes, and a
// record a colleague already holds can.
//
// Never regenerate the file to make a test pass. A failure here means the bytes a
// scheme signs changed, and the fix is a new scheme beside the old one.

type frozenIdentity struct {
	Keys struct {
		Master, Shared, Machine, Cloud, Session string
	} `json:"keys"`
	Shared       SharedKeyCert     `json:"shared"`
	Machine      MachineCert       `json:"machine"`
	Cloud        MachineCert       `json:"cloud"`
	Session      SessionCert       `json:"session"`
	Removal      Removal           `json:"removal"`
	MachineChain Chain             `json:"machineChain"`
	CloudChain   Chain             `json:"cloudChain"`
	Hashes       map[string]string `json:"hashes"`
	ListChange   struct {
		Record ListChange `json:"record"`
		Hash   string     `json:"hash"`
	} `json:"listChange"`
}

func loadIdentityVectors(t *testing.T) frozenIdentity {
	t.Helper()
	raw, err := os.ReadFile("testdata/identity_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v frozenIdentity
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestFrozenIdentityRecordsStillVerifyAndHashAsTheyDid(t *testing.T) {
	k := testIDKeys(t)
	v := loadIdentityVectors(t)
	if got := []string{v.Keys.Master, v.Keys.Shared, v.Keys.Machine, v.Keys.Cloud, v.Keys.Session}; !reflect.DeepEqual(got,
		[]string{k.master.id, k.shared.id, k.machine.id, k.cloud.id, k.session.id}) {
		t.Fatalf("the frozen seeds give other identifiers than the vectors name: %v", got)
	}
	const why = "the bytes scheme v1 signs have changed; add a scheme and leave v1 as it was (D-058, signature schemes are kept, never replaced)"
	for name, err := range map[string]error{
		"shared":  v.Shared.Verify(),
		"machine": v.Machine.VerifyBy(k.shared.pub),
		"cloud":   v.Cloud.VerifyBy(k.shared.pub),
		"session": v.Session.VerifyBy(k.cloud.pub),
		"removal": v.Removal.VerifyBy(k.shared.pub),
	} {
		if err != nil {
			t.Errorf("frozen %s no longer verifies: %v; %s", name, err, why)
		}
	}
	if _, err := v.MachineChain.Verify(k.master.id, k.machine.id); err != nil {
		t.Errorf("frozen machine chain: %v; %s", err, why)
	}
	if _, err := v.CloudChain.Verify(k.master.id, k.session.id); err != nil {
		t.Errorf("frozen cloud chain: %v; %s", err, why)
	}
	for name, h := range map[string][32]byte{
		"shared": v.Shared.Hash(), "machine": v.Machine.Hash(), "cloud": v.Cloud.Hash(),
		"session": v.Session.Hash(), "removal": v.Removal.Hash(),
	} {
		if got := hex.EncodeToString(h[:]); got != v.Hashes[name] {
			t.Errorf("frozen %s hashes to %s, was %s; a hash is how peers name a record to one another", name, got, v.Hashes[name])
		}
	}
}

// The frozen hashes came from this code, so they hold the bytes still; this holds
// them to the encoding the design states, written out without the code's helpers.
func TestFrozenHashesAreOfTheEncodingTheDesignStates(t *testing.T) {
	v := loadIdentityVectors(t)
	var b []byte
	str := func(s string) { b = append(binary.BigEndian.AppendUint32(b, uint32(len(s))), s...) }
	num := func(n uint64) { b = binary.BigEndian.AppendUint64(b, n) }

	str("peer-room/shared-key-cert/v1")
	str(v.Keys.Master)
	num(3)
	str(v.Keys.Shared)
	if h := sha256.Sum256(b); hex.EncodeToString(h[:]) != v.Hashes["shared"] {
		t.Error("the frozen shared-key certificate is not a hash of user, version and shared key under its tag")
	}

	b = nil
	str("peer-room/removal/v1")
	str(v.Keys.Master)
	str(v.Keys.Machine)
	num(2)
	str(v.Keys.Shared)
	num(3)
	num(2)
	str("room-a")
	num(5)
	str("room-b")
	num(9)
	if h := sha256.Sum256(b); hex.EncodeToString(h[:]) != v.Hashes["removal"] {
		t.Error("the frozen removal is not a hash of its fields in the order the design gives")
	}
}

func TestAChangeToAFrozenIdentityRecordFailsVerification(t *testing.T) {
	// The check above cannot fail if verification accepts anything.
	k := testIDKeys(t)
	v := loadIdentityVectors(t)
	v.Machine.Name += "x"
	if v.Machine.VerifyBy(k.shared.pub) == nil {
		t.Error("an altered frozen machine certificate verified")
	}
	v.ListChange.Record.Value += "x"
	if v.ListChange.Record.VerifyBy(k.machine.pub) == nil {
		t.Error("an altered frozen list change verified")
	}
}

func TestFrozenListChangeStillVerifiesAndHashesAsItDid(t *testing.T) {
	k := testIDKeys(t)
	v := loadIdentityVectors(t)
	lc := v.ListChange.Record
	if err := lc.VerifyBy(k.machine.pub); err != nil {
		t.Errorf("frozen list change no longer verifies: %v; the bytes scheme v1 signs have changed; add a scheme and leave v1 as it was (D-058, signature schemes are kept, never replaced)", err)
	}
	h := lc.Hash()
	if got := hex.EncodeToString(h[:]); got != v.ListChange.Hash {
		t.Errorf("frozen list change hashes to %s, was %s; a hash is how machines name a record to one another", got, v.ListChange.Hash)
	}
	if lc != k.listChange() {
		t.Error("signing the frozen list change's fields again gives another record")
	}

	// The encoding the design states, written out without the code's helpers.
	var b []byte
	str := func(s string) { b = append(binary.BigEndian.AppendUint32(b, uint32(len(s))), s...) }
	str("peer-room/list-change/v1")
	str(v.Keys.Master)
	str("colleague:" + k.other.id)
	str(`{"label":"Ana Ünïcode","verified":true}`)
	b = binary.BigEndian.AppendUint64(b, 4)
	str(v.Keys.Machine)
	if h := sha256.Sum256(b); hex.EncodeToString(h[:]) != v.ListChange.Hash {
		t.Error("the frozen list change is not a hash of user, entry, value, counter and signer under its tag")
	}
}

func TestAListChangeVerifiesAfterAJSONRoundTrip(t *testing.T) {
	k := testIDKeys(t)
	lc := k.listChange()
	if lc.Scheme != 1 {
		t.Errorf("a list change is signed under scheme %d, want 1", lc.Scheme)
	}
	if got := roundTrip(t, lc); got.VerifyBy(k.machine.pub) != nil || got.Hash() != lc.Hash() {
		t.Errorf("list change after JSON: verify %v, hash changed %v", got.VerifyBy(k.machine.pub), got.Hash() != lc.Hash())
	}
}

func TestATamperedFieldOfAListChangeFailsVerification(t *testing.T) {
	k := testIDKeys(t)
	for name, change := range map[string]func(*ListChange){
		"user":    func(c *ListChange) { c.User = k.other.id },
		"entry":   func(c *ListChange) { c.Entry = "room:" + k.other.id },
		"value":   func(c *ListChange) { c.Value = `{"label":"Ana Ünïcodf","verified":true}` },
		"counter": func(c *ListChange) { c.Counter = 5 },
	} {
		c := k.listChange()
		change(&c)
		wantSignatureRefusal(t, name, c.VerifyBy(k.machine.pub))
	}
}

func TestAListChangeIsRefusedForAnotherSignerNoEntryOrACounterOfZero(t *testing.T) {
	k := testIDKeys(t)
	for name, tc := range map[string]struct {
		change func(*ListChange)
		by     ed25519.PublicKey
		want   string
	}{
		"checked by another key": {func(*ListChange) {}, k.other.pub, "signer other than"},
		"signer renamed":         {func(c *ListChange) { c.Signer = k.other.id }, k.other.pub, "does not match"},
		"no entry":               {func(c *ListChange) { c.Entry = "" }, k.machine.pub, "no entry"},
		"counter 0":              {func(c *ListChange) { c.Counter = 0 }, k.machine.pub, "counter 0"},
	} {
		c := k.listChange()
		tc.change(&c)
		if err := c.VerifyBy(tc.by); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v, want an error saying %q", name, err, tc.want)
		}
	}
	signed := SignListChange(k.machine.priv, k.master.id, "", "v", 1)
	if signed.VerifyBy(k.machine.pub) == nil {
		t.Error("a list change signed with no entry verified")
	}
}

// A room's identifier is computed by every guest's machine from the founding
// self-invitation, so a change to how it is computed founds every existing room
// anew and admits nobody. Never regenerate this value to make the test pass; a
// failure means the computation changed, and the fix is a new version beside it.
func TestARoomIdentifierIsTheFrozenHashOfItsCreatorAndNonce(t *testing.T) {
	k := testIDKeys(t)
	const frozen = "aNpd4CVD4nmqnuipDPasfnGeIEMRLLWm0-2C_Gsi-kg"
	nonce := []byte("founding-nonce-0123456789abcdef")
	if got := RoomIDFor(k.master.id, nonce); got != frozen {
		t.Fatalf("RoomIDFor gives %s, was %s", got, frozen)
	}
	if RoomIDFor(k.other.id, nonce) == frozen {
		t.Error("another creator with the same nonce gives the same room")
	}
	if RoomIDFor(k.master.id, append(nonce, 0)) == frozen {
		t.Error("another nonce gives the same room")
	}
	// Length prefixes keep a creator and a nonce from running together.
	if RoomIDFor("ab", []byte("c")) == RoomIDFor("a", []byte("bc")) {
		t.Error("two splits of one string give one room")
	}
}
