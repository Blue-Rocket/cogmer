package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// --- cloud sessions: a key of their own, and the user's lists from a machine ---
//
// A cloud environment is added on the machines page of one of the user's machines,
// which makes a cloud key, certifies it under the shared key as a machine of kind
// cloud, and shows one value for the user to paste into the environment's variable
// COGMER_CLOUD_KEY (D-246, a cloud environment is added from the machines page, which
// shows the one value to paste into it). The machine keeps the certificate and not the
// key, so the value is shown once, in the response to the page's request, and nowhere
// else: never in a log, a command's output or anything given to an assistant, since
// whoever holds it speaks as the user in every room the user is in.
//
// A daemon whose environment holds the value is a cloud session's. It makes a session
// key, certifies it with the cloud key for a day, keeps it in session.key, and certifies
// a new one before the old one expires (D-216, a cloud session signs with a key of its
// own that the user's shared key vouches for; D-247, a cloud session's key is certified
// for twenty-four hours, and the session's daemon certifies a new one before it
// expires). It learns its user's colleagues, rooms and machines only by fetching the
// user's records from one of the user's machines, which answers with a header it signs
// (D-245, a cloud session takes its user's lists only from one of the user's
// machines), and asks again every few minutes. A value that does not read, or no
// machine that answers, leaves the session sharing nothing, with nothing shown, while
// the daemon keeps trying (§3.1, first, do no harm).

// cloudKeyVar is the environment variable a cloud environment holds its value in.
const cloudKeyVar = "COGMER_CLOUD_KEY"

// cloudValuePrefix starts every value, and names its version: a value of another
// form is a new prefix beside this one.
const cloudValuePrefix = "cogmer-cloud-v1:"

// cloudMachine is one of the user's machines as a value or a fetched answer names it:
// its key, the name the user gave it, and where it listens.
type cloudMachine struct {
	Machine  string `json:"machine"`
	Name     string `json:"name,omitempty"`
	Endpoint string `json:"endpoint"`
}

// cloudValueWire is the value's JSON: the cloud key's seed, its two certificates, the
// name the user's colleagues see, and the user's machines with their relay addresses,
// which are the same at every start (D-104, the overlay address holds no secret and is
// the same at every start).
type cloudValueWire struct {
	Key         string         `json:"key"`
	Shared      SharedKeyCert  `json:"shared"`
	Cloud       MachineCert    `json:"cloud"`
	DisplayName string         `json:"displayName,omitempty"`
	Machines    []cloudMachine `json:"machines"`
}

// cloudSetup is a value that has read and verified.
type cloudSetup struct {
	key         ed25519.PrivateKey
	shared      SharedKeyCert
	cloud       MachineCert
	user        string
	displayName string
	machines    []cloudMachine
}

// String keeps the key out of anything that formats a setup.
func (c *cloudSetup) String() string { return "a cloud setup for " + PeerName(c.user) }

// encodeCloudValue writes the value the page shows.
func encodeCloudValue(key ed25519.PrivateKey, shared SharedKeyCert, cloud MachineCert, display string, machines []cloudMachine) (string, error) {
	raw, err := json.Marshal(cloudValueWire{
		Key:         base64.RawURLEncoding.EncodeToString(key.Seed()),
		Shared:      shared,
		Cloud:       cloud,
		DisplayName: display,
		Machines:    machines,
	})
	if err != nil {
		return "", err
	}
	return cloudValuePrefix + base64.RawURLEncoding.EncodeToString(raw), nil
}

// parseCloudValue reads a value and checks its certificates. No error quotes any part
// of the value, since it holds a private key.
func parseCloudValue(s string) (*cloudSetup, error) {
	s = strings.TrimSpace(s)
	body, ok := strings.CutPrefix(s, cloudValuePrefix)
	if !ok {
		return nil, fmt.Errorf("it does not start with %q", cloudValuePrefix)
	}
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return nil, errors.New("its body is not base64url")
	}
	var v cloudValueWire
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return nil, errors.New("its body is not the JSON of a cloud value")
	}
	seed, err := base64.RawURLEncoding.DecodeString(v.Key)
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, errors.New("it holds no usable cloud key")
	}
	key := ed25519.NewKeyFromSeed(seed)
	if err := v.Shared.Verify(); err != nil {
		return nil, fmt.Errorf("its shared-key certificate: %w", err)
	}
	sharedPub, _ := PublicFromPeerID(v.Shared.SharedKey)
	c := v.Cloud
	switch {
	case c.User != v.Shared.User:
		return nil, errors.New("its certificates are for two users")
	case c.SharedVersion != v.Shared.Version:
		return nil, errors.New("its cloud certificate is under another shared version than its shared-key certificate")
	case c.Kind != KindCloud:
		return nil, errors.New("its certificate is not a cloud key's")
	case c.Machine != PeerIDFromPublic(key.Public().(ed25519.PublicKey)):
		return nil, errors.New("its certificate is for another key than the one it holds")
	}
	if err := c.VerifyBy(sharedPub); err != nil {
		return nil, fmt.Errorf("its cloud certificate: %w", err)
	}
	out := &cloudSetup{key: key, shared: v.Shared, cloud: c, user: c.User, displayName: sanitizeName(v.DisplayName)}
	for _, m := range v.Machines {
		if _, err := PublicFromPeerID(m.Machine); err != nil || strings.TrimSpace(m.Endpoint) == "" {
			return nil, errors.New("it names a machine with no key or no address")
		}
		out.machines = append(out.machines, cloudMachine{Machine: m.Machine, Name: sanitizeName(m.Name), Endpoint: strings.TrimSpace(m.Endpoint)})
	}
	if len(out.machines) == 0 {
		return nil, errors.New("it names none of the user's machines to fetch the user's lists from")
	}
	return out, nil
}

// --- adding a cloud environment, on the machines page ---

type cloudAddRequest struct {
	Name string `json:"name"`
}

type cloudAddResponse struct {
	Value   string `json:"value,omitempty"`
	Name    string `json:"name,omitempty"`
	Machine string `json:"machine,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleCloudAdd makes a cloud key, certifies it under this machine's shared key as a
// machine of kind cloud with the name the user gave, keeps the certificate with the
// user's records, so that it is listed, copied and removed like any machine's, and
// answers with the value (D-246). The key is in this response and nowhere else.
func (d *Daemon) handleCloudAdd(w http.ResponseWriter, r *http.Request) {
	var req cloudAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	name := sanitizeName(req.Name)
	if name == "" {
		writeJSON(w, cloudAddResponse{Error: "Give the cloud environment a name."})
		return
	}
	value, machine, err := d.addCloud(name)
	if err != nil {
		writeJSON(w, cloudAddResponse{Error: err.Error()})
		return
	}
	writeJSON(w, cloudAddResponse{Value: value, Name: name, Machine: machine})
}

// addCloud certifies a new cloud key named name and returns the value that holds it.
func (d *Daemon) addCloud(name string) (value, machine string, err error) {
	id := d.id
	if !id.HasUser() || d.isCloudSession() {
		return "", "", errors.New("only one of your own machines adds a cloud environment")
	}
	shared, version, ok, err := loadSharedKeyIn(d.homeDirectory())
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", errors.New("this machine holds no shared key, so it cannot add a cloud environment")
	}
	rs, _, _ := d.members.IdentityRecords(id.User)
	st := Resolve(id.User, rs)
	sharedID := PeerIDFromPublic(shared.Public().(ed25519.PublicKey))
	if version != st.Current || st.SharedKeys[version] != sharedID {
		return "", "", errors.New("this machine's shared key is not your current one, so a key it certifies would be suspended")
	}
	var sharedCert *SharedKeyCert
	for i, c := range rs.Shared {
		if c.Version == version && c.SharedKey == sharedID {
			sharedCert = &rs.Shared[i]
			break
		}
	}
	if sharedCert == nil {
		return "", "", errors.New("this machine holds no certificate of its shared key")
	}
	pub, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		return "", "", err
	}
	defer zero(key)
	cert := CertifyMachine(shared, id.User, version, pub, KindCloud, name, d.members.highestSerial(id.User)+1)
	body, err := json.Marshal(cert)
	if err != nil {
		return "", "", err
	}
	if _, _, err := d.members.PutIdentityRecord(RecordMachine, id.User, body); err != nil {
		return "", "", err
	}
	value, err = encodeCloudValue(key, *sharedCert, cert, id.UserDisplayName, d.ownMachinesForCloud())
	if err != nil {
		return "", "", err
	}
	return value, cert.Machine, nil
}

// ownMachinesForCloud is each of the user's machines a cloud session can fetch the
// user's lists from: certified as a machine under the current shared key, not removed,
// with an address, this one included.
func (d *Daemon) ownMachinesForCloud() []cloudMachine {
	id := d.id
	certs, current := d.members.displayCerts(id.User)
	var out []cloudMachine
	for key, c := range certs {
		if c.Kind != KindMachine || !current[key] {
			continue
		}
		ep := ""
		if key == id.PeerID {
			ep = d.advertised()
		} else {
			ep, _ = d.members.MachineEndpoint(key)
		}
		if ep != "" {
			out = append(out, cloudMachine{Machine: key, Name: c.Name, Endpoint: ep})
		}
	}
	sortCloudMachines(out)
	return out
}

func sortCloudMachines(ms []cloudMachine) {
	for i := 1; i < len(ms); i++ {
		for j := i; j > 0 && ms[j].Machine < ms[j-1].Machine; j-- {
			ms[j], ms[j-1] = ms[j-1], ms[j]
		}
	}
}

// --- the session key ---

func sessionKeyPathIn(dir string) string { return filepath.Join(dir, "session.key") }

// loadSessionKeyIn reads a cloud session's key, reporting false when there is none or
// it does not read.
func loadSessionKeyIn(dir string) (ed25519.PrivateKey, bool) {
	raw, err := os.ReadFile(sessionKeyPathIn(dir))
	if err != nil {
		return nil, false
	}
	seed, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, false
	}
	return ed25519.NewKeyFromSeed(seed), true
}

// saveSessionKeyIn writes a cloud session's key in a file of its own, readable by this
// account alone, through a temporary file renamed over the old one, so that a crash
// leaves one whole key (D-153, the private key is kept in a file of its own).
func saveSessionKeyIn(dir string, key ed25519.PrivateKey) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "session.key.*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(base64.RawURLEncoding.EncodeToString(key.Seed()) + "\n"); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), sessionKeyPathIn(dir))
}

// --- the cloud session's daemon ---

const (
	// sessionCertTTL is how long the cloud key certifies a session key for (D-247).
	sessionCertTTL = 24 * time.Hour
	// sessionRenewBefore is how long before its certificate expires a session key is
	// replaced, which leaves a daemon that misses a check many more to renew in.
	sessionRenewBefore = 2 * time.Hour
	// cloudListsEvery is how often a session asks for its user's lists again, so a
	// removal made after it started reaches it within minutes (D-245).
	cloudListsEvery = 3 * time.Minute
	// cloudListsRetry is how soon a session that reached none of its user's machines
	// tries again.
	cloudListsRetry = 15 * time.Second
)

// cloudSession is what a cloud session's daemon holds beside its identity.
type cloudSession struct {
	setup *cloudSetup

	ttl, renewBefore  time.Duration
	listsEvery, retry time.Duration
	mu                sync.Mutex
	nextFetch         time.Time
	header            *ListsHeader // the last header taken
	from              string       // the name of the machine that signed it
	fetchedAt         time.Time
	failingSince      time.Time
	failure           string
	answered          bool // whether any fetch has succeeded since the daemon started
}

// isCloudSession reports whether this daemon serves a cloud session: one whose
// environment holds a cloud value, whether or not it read. Such a daemon never takes
// the recovery phrase (D-223, a cloud session never asks for the phrase).
func (d *Daemon) isCloudSession() bool { return d.cloud != nil || d.cloudEnv }

// startCloud makes this daemon a cloud session's from the value raw. A value that
// does not read leaves the daemon with no identity, sharing nothing, and says why in
// the log without quoting the value.
func (d *Daemon) startCloud(raw string) {
	d.cloudEnv = true
	setup, err := parseCloudValue(raw)
	if err != nil {
		log.Printf("cloud: %s does not read as a cloud value (%v), so this session shares nothing. Add a cloud environment again from /cogmer:self-machines on one of your machines.", cloudKeyVar, err)
		return
	}
	if d.id.HasUser() && d.id.Chain.Session == nil {
		log.Printf("cloud: this machine already speaks for %s as a machine of its own, so %s is ignored", PeerName(d.id.User), cloudKeyVar)
		return
	}
	d.cloud = &cloudSession{setup: setup, ttl: sessionCertTTL, renewBefore: sessionRenewBefore,
		listsEvery: cloudListsEvery, retry: cloudListsRetry}
	for _, rec := range []struct {
		kind string
		v    any
	}{{RecordShared, setup.shared}, {RecordMachine, setup.cloud}} {
		body, _ := json.Marshal(rec.v)
		if _, _, err := d.members.PutIdentityRecord(rec.kind, setup.user, body); err != nil {
			log.Printf("cloud: the cloud key's certificates were not kept: %v", err)
		}
	}
	for _, m := range setup.machines {
		if err := d.members.SetMachineEndpoint(m.Machine, setup.user, m.Endpoint); err != nil {
			log.Printf("cloud: could not record where %s listens: %v", firstNonEmpty(m.Name, PeerName(m.Machine)), err)
		}
	}
	if err := d.ensureSessionKey(time.Now()); err != nil {
		log.Printf("cloud: no session key could be certified, so this session shares nothing: %v", err)
		d.cloud = nil
	}
}

// ensureSessionKey keeps the session key while its certificate has more than
// renewBefore to run and is from this cloud key, and otherwise certifies a new one.
func (d *Daemon) ensureSessionKey(now time.Time) error {
	c := d.cloud
	id := d.id
	if id.HasUser() && id.Chain.Session != nil && id.Chain.Session.CloudKey == c.setup.cloud.Machine && id.User == c.setup.user {
		if exp, err := time.Parse(time.RFC3339, id.Chain.Session.Expires); err == nil && exp.Sub(now) > c.renewBefore {
			return d.refreshCloudChain()
		}
	}
	return d.renewSession(now)
}

// renewSession certifies a new session key for this session and makes it the key the
// daemon signs with. It is a new key and not a new certificate for the old one, since
// a session's chain travels with every event it signs, so a key copied out of the
// container could take each new certificate from the next event and never stop
// working (D-247, a session key copied out of a container stops working within a day).
// The key is written before the identity that names it, and the new key numbers its
// own events from 1 (D-249, each key that signs in a room counts its own sequence
// from 1).
func (d *Daemon) renewSession(now time.Time) error {
	c := d.cloud
	pub, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		return err
	}
	shared, cloud := d.cloudCerts()
	cert := CertifySession(c.setup.key, c.setup.user, pub, now.Add(c.ttl))
	chain := Chain{Shared: shared, Machine: cloud, Session: &cert}
	signer := PeerIDFromPublic(pub)
	if _, err := chain.Verify(c.setup.user, signer); err != nil {
		return fmt.Errorf("the session's chain does not verify: %w", err)
	}
	home := d.homeDirectory()
	if err := saveSessionKeyIn(home, key); err != nil {
		return err
	}
	if err := d.members.StoreChain(chain); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	next := Identity{
		PeerID: signer, PeerName: PeerName(signer), private: key,
		UserID: d.id.UserID, UserDisplayName: firstNonEmpty(c.setup.displayName, d.id.UserDisplayName), NameChosen: true,
		MachineID: cloud.Name, User: c.setup.user, Chain: &chain,
	}
	if err := saveIdentityIn(home, &next); err != nil {
		return err
	}
	d.id = &next
	d.tls.reset()
	log.Printf("cloud: this session speaks for %s as %s, with a key certified until %s", PeerName(c.setup.user), cloud.Name, cert.Expires)
	return nil
}

// cloudCerts is the shared-key and cloud certificates the session's chain carries: the
// cloud key's certificate as the user's records settle it, renamed or certified again
// under a newer shared key, or the value's own while the records hold nothing newer.
func (d *Daemon) cloudCerts() (SharedKeyCert, MachineCert) {
	s := d.cloud.setup
	rs, _, _ := d.members.IdentityRecords(s.user)
	st := Resolve(s.user, rs)
	ms, ok := st.Machines[s.cloud.Machine]
	if !ok || ms.Removed || ms.Cert.Kind != KindCloud {
		return s.shared, s.cloud
	}
	for _, c := range rs.Shared {
		if c.Version == ms.Cert.SharedVersion && c.SharedKey == st.SharedKeys[c.Version] {
			return c, ms.Cert
		}
	}
	return s.shared, s.cloud
}

// refreshCloudChain carries the session's key over to the cloud key's certificate as
// the user's records now settle it, when that is not the one the chain carries.
func (d *Daemon) refreshCloudChain() error {
	id := d.id
	if !id.HasUser() || id.Chain.Session == nil {
		return nil
	}
	shared, cloud := d.cloudCerts()
	if cloud.Hash() == id.Chain.Machine.Hash() && shared.Hash() == id.Chain.Shared.Hash() {
		return nil
	}
	sess := *id.Chain.Session
	chain := Chain{Shared: shared, Machine: cloud, Session: &sess}
	if _, err := chain.Verify(id.User, id.PeerID); err != nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	next := *d.id
	next.Chain, next.MachineID = &chain, cloud.Name
	if err := saveIdentityIn(d.homeDirectory(), &next); err != nil {
		return err
	}
	d.id = &next
	d.tls.reset()
	return nil
}

// RunCloud renews the session key before it expires and fetches the user's lists on a
// cycle, each on a loop of its own and never on a hook's path.
func (d *Daemon) RunCloud(every time.Duration) {
	for {
		d.cloudRound(time.Now())
		time.Sleep(every)
	}
}

// cloudRound is one turn of RunCloud: a session key due for renewal is renewed, and
// the lists are fetched when they are due.
func (d *Daemon) cloudRound(now time.Time) {
	c := d.cloud
	if c == nil {
		return
	}
	if err := d.ensureSessionKey(now); err != nil {
		log.Printf("cloud: the session key was not renewed, and is tried again: %v", err)
	}
	c.mu.Lock()
	due := !now.Before(c.nextFetch)
	c.mu.Unlock()
	if !due {
		return
	}
	from, err := d.fetchUserLists()
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		if c.failingSince.IsZero() {
			c.failingSince = now
			log.Printf("cloud: none of your machines answered for your lists, so this session waits for one: %v", err)
		}
		c.failure = err.Error()
		c.nextFetch = now.Add(c.retry)
		return
	}
	// Said once after a start and once after a failure, so the log shows that the
	// session has its lists and from where, and stays quiet while nothing changes.
	if !c.answered || !c.failingSince.IsZero() {
		log.Printf("cloud: your lists arrived from %s", from)
	}
	c.answered = true
	c.failingSince, c.failure = time.Time{}, ""
	c.nextFetch = now.Add(c.listsEvery)
}

// cloudTargets is where to ask for the user's lists: the machines the value names, and
// every machine of the user since learned that is certified as a machine and current.
func (d *Daemon) cloudTargets() []cloudMachine {
	s := d.cloud.setup
	seen := map[string]bool{}
	var out []cloudMachine
	add := func(m cloudMachine) {
		if m.Endpoint != "" && !seen[m.Endpoint] {
			seen[m.Endpoint] = true
			out = append(out, m)
		}
	}
	certs, current := d.members.displayCerts(s.user)
	ms, _ := d.members.UserMachines(s.user)
	for _, m := range ms {
		if c, ok := certs[m.MachineKey]; ok && c.Kind == KindMachine && current[m.MachineKey] {
			add(cloudMachine{Machine: m.MachineKey, Name: c.Name, Endpoint: m.Endpoint})
		}
	}
	for _, m := range s.machines {
		if c, ok := certs[m.Machine]; ok && (c.Kind != KindMachine || !current[m.Machine]) {
			continue
		}
		add(m)
	}
	return out
}

// fetchUserLists asks each of the user's machines in turn until one answers, and
// gives the name of the machine that did.
func (d *Daemon) fetchUserLists() (string, error) {
	var errs []string
	for _, m := range d.cloudTargets() {
		name := firstNonEmpty(m.Name, PeerName(m.Machine))
		if err := d.fetchListsFrom(m); err != nil {
			d.markPeerUnreachable(m.Endpoint, err)
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		d.markPeerSeen(m.Endpoint)
		return name, nil
	}
	if len(errs) == 0 {
		return "", errors.New("no machine of yours has an address")
	}
	return "", errors.New(strings.Join(errs, "; "))
}

// selfFetchTag is the purpose tag of a cloud session's request for its user's lists.
const selfFetchTag = protocolNamespace + "/user-lists-fetch/v1"

type selfFetchPayload struct {
	Have []string `json:"have,omitempty"`
}

// selfFetchResponse is a machine's answer: the header it signs, the chain by which it
// speaks for the user, the user's records the asker lacks, and the user's machines
// with their addresses.
type selfFetchResponse struct {
	Header   ListsHeader    `json:"header"`
	Chain    Chain          `json:"chain"`
	Records  []selfRecord   `json:"records,omitempty"`
	Machines []cloudMachine `json:"machines,omitempty"`
}

// fetchListsFrom asks one machine for the user's lists, and takes its answer only when
// its header verifies, is signed by a machine of the user that is current and not
// removed, and is no older than the last one taken from that machine.
func (d *Daemon) fetchListsFrom(m cloudMachine) error {
	id := d.id
	if !id.HasUser() {
		return errors.New("this session has no identity")
	}
	var p selfFetchPayload
	for _, r := range d.members.storedSelfRecords(id.User) {
		p.Have = append(p.Have, r.Hash)
	}
	payload, err := json.Marshal(p)
	if err != nil {
		return err
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	chain := *id.Chain
	req := selfListsRequest{
		User: id.User, Signer: id.PeerID, Chain: &chain,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Nonce: base64.RawURLEncoding.EncodeToString(raw),
		Endpoint: d.advertised(), Payload: payload,
	}
	req.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(id.private,
		selfRequestBytes(selfFetchTag, req.User, req.Signer, req.Timestamp, req.Nonce, req.Endpoint, req.Payload)))
	conf, err := d.clientConfig(id.User)
	if err != nil {
		return err
	}
	client, err := clientFor(m.Endpoint, 30*time.Second, conf)
	if err != nil {
		return err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	resp, err := client.Post(peerURL("/self/fetch"), "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		text, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(text)))
	}
	var answer selfFetchResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxPeerBody)).Decode(&answer); err != nil {
		return err
	}
	connected := ""
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		if k, ok := resp.TLS.PeerCertificates[0].PublicKey.(ed25519.PublicKey); ok {
			connected = PeerIDFromPublic(k)
		}
	}
	return d.takeLists(answer, connected)
}

// takeLists checks a machine's answer and applies it. connected is the key on the
// other end of the connection, which must be the machine that signed the header.
func (d *Daemon) takeLists(a selfFetchResponse, connected string) error {
	id := d.id
	h := a.Header
	if h.User != id.User {
		return errors.New("the answer is for another user")
	}
	if connected != "" && connected != h.Machine {
		return errors.New("the header is signed by a machine other than the one connected")
	}
	link, err := a.Chain.Verify(id.User, h.Machine)
	if err != nil {
		return fmt.Errorf("the answering machine's chain: %w", err)
	}
	if link.Kind != KindMachine {
		return errors.New("the answer is not from one of the user's machines")
	}
	pub, err := PublicFromPeerID(h.Machine)
	if err != nil {
		return err
	}
	if err := h.VerifyBy(pub); err != nil {
		return fmt.Errorf("the header: %w", err)
	}
	if held := d.members.listsHeaderHeld(h.Machine); h.Version < held {
		return fmt.Errorf("the header is version %d, older than version %d already taken from that machine", h.Version, held)
	}
	for _, r := range a.Records {
		if !knownSelfKind(r.Kind) {
			continue
		}
		if _, _, err := d.members.PutIdentityRecord(r.Kind, id.User, r.Body); err != nil {
			log.Printf("cloud: a record of your lists was not kept: %v", err)
		}
	}
	// Checked as a machine checks one of its user's machines, against the records the
	// answer brought with it: a machine removed or suspended there answers for nobody.
	if _, err := d.members.UserState(id.User, &a.Chain).Check(a.Chain, h.Machine); err != nil {
		return fmt.Errorf("the answering machine: %w", err)
	}
	if err := d.members.recordListsHeader(h); err != nil {
		return err
	}
	certs, current := d.members.displayCerts(id.User)
	for _, m := range a.Machines {
		if c, ok := certs[m.Machine]; !ok || c.Kind != KindMachine || !current[m.Machine] || m.Endpoint == "" {
			continue
		}
		if cur, _ := d.members.MachineEndpoint(m.Machine); cur != m.Endpoint {
			if err := d.members.SetMachineEndpoint(m.Machine, id.User, m.Endpoint); err != nil {
				log.Printf("cloud: could not record where %s listens: %v", firstNonEmpty(m.Name, PeerName(m.Machine)), err)
			}
		}
	}
	if err := d.members.ApplyUserLists(id); err != nil {
		log.Printf("cloud: your lists were not all applied: %v", err)
	}
	_ = d.refreshCloudChain()
	c := d.cloud
	c.mu.Lock()
	hh := h
	c.header, c.from, c.fetchedAt = &hh, firstNonEmpty(certs[h.Machine].Name, PeerName(h.Machine)), time.Now()
	c.mu.Unlock()
	d.notify()
	return nil
}

// listsView is what the room view says of a cloud session's lists: which machine they
// came from and how old they are, or that none has answered yet.
type listsView struct {
	From    string `json:"from,omitempty"`
	Age     string `json:"age,omitempty"`
	Waiting string `json:"waiting,omitempty"`
}

func (d *Daemon) cloudListsView() *listsView {
	c := d.cloud
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	v := &listsView{}
	if c.header != nil {
		v.From = c.from
		if t, err := time.Parse(time.RFC3339, c.header.Time); err == nil {
			v.Age = roughDuration(time.Since(t))
		}
	}
	if !c.failingSince.IsZero() {
		v.Waiting = roughDuration(time.Since(c.failingSince))
	}
	return v
}

// --- the machine's side: answering a cloud session ---

// handleSelfFetch answers a key whose chain leads to this machine's own user with the
// user's records it lacks, the user's machines and their addresses, and a header this
// machine signs (D-245). A key of any other user is refused, as is a machine removed
// or suspended in what this machine holds.
func (d *Daemon) handleSelfFetch(w http.ResponseWriter, r *http.Request) {
	id := d.id
	if !id.HasUser() || id.Chain.Session != nil {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	var req selfListsRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 32<<20)).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := d.checkSelfRequest(selfFetchTag, req, r); err != nil {
		log.Printf("lists: refused a fetch from %s — %v", PeerName(req.Signer), err)
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	var p selfFetchPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	d.markPeerHeardFrom(req.Signer)
	version, err := d.members.listsVersion(id.User)
	if err != nil {
		http.Error(w, "the lists could not be read", http.StatusInternalServerError)
		return
	}
	have := map[string]bool{}
	for _, h := range p.Have {
		have[h] = true
	}
	out := selfFetchResponse{Header: SignListsHeader(id.private, id.User, version, time.Now()), Chain: *id.Chain}
	for _, rec := range d.members.storedSelfRecords(id.User) {
		if !have[rec.Hash] {
			out.Records = append(out.Records, rec.selfRecord)
		}
	}
	out.Machines = d.ownMachinesForCloud()
	writeJSON(w, out)
}

// cloudSchema is membership.db's record of the version of the user's lists: on a
// machine, the counter it signs into each header, with the digest of the records it
// counted, and on a cloud session, the highest version taken from each machine.
const cloudSchema = `
CREATE TABLE IF NOT EXISTS lists_version (
  user    TEXT PRIMARY KEY,
  version INTEGER NOT NULL,
  digest  TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS lists_headers (
  machine TEXT PRIMARY KEY,
  version INTEGER NOT NULL,
  time    TEXT NOT NULL
);
`

// listsVersion is the version of user's lists this machine holds: one more than the
// last whenever the records held have changed since it was counted, recorded before it
// is signed into any header, so that a restart never signs a lower version for lists
// that changed (the pattern "persist anything before publishing it").
func (m *Membership) listsVersion(user string) (uint64, error) {
	sum := sha256.New()
	for _, r := range m.storedSelfRecords(user) {
		sum.Write([]byte(r.Hash))
	}
	digest := hex.EncodeToString(sum.Sum(nil))
	tx, err := m.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var version uint64
	var held string
	err = tx.QueryRow(`SELECT version, digest FROM lists_version WHERE user = ?`, user).Scan(&version, &held)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	if err == nil && held == digest {
		return version, nil
	}
	version++
	if _, err := tx.Exec(`INSERT INTO lists_version (user, version, digest) VALUES (?,?,?)
		ON CONFLICT(user) DO UPDATE SET version = excluded.version, digest = excluded.digest`, user, version, digest); err != nil {
		return 0, err
	}
	return version, tx.Commit()
}

// sessionKeysOf is every session key that a session certificate this machine holds
// says cloud certified for user, sorted, each certificate checked against cloud. The
// certificates are kept from the chains of events and requests, so a session whose
// turns never reached this machine is not among them.
func (m *Membership) sessionKeysOf(user, cloud string) []string {
	pub, err := PublicFromPeerID(cloud)
	if err != nil {
		return nil
	}
	rows, err := m.db.Query(`SELECT body FROM identity_records WHERE user = ? AND kind = ?`, user, RecordSession)
	if err != nil {
		return nil
	}
	defer rows.Close()
	seen := map[string]bool{}
	var out []string
	for rows.Next() {
		var body string
		var c SessionCert
		if rows.Scan(&body) != nil || json.Unmarshal([]byte(body), &c) != nil {
			continue
		}
		if c.User != user || c.CloudKey != cloud || seen[c.Session] || c.VerifyBy(pub) != nil {
			continue
		}
		seen[c.Session] = true
		out = append(out, c.Session)
	}
	sort.Strings(out)
	return out
}

// machineOfSigner is the key user's records certify for signer: the cloud key of a
// session certificate this machine holds for signer, kept from the chain of an event or
// a request, and signer itself otherwise.
func (m *Membership) machineOfSigner(user, signer string) string {
	rows, err := m.db.Query(`SELECT body FROM identity_records WHERE user = ? AND kind = ?`, user, RecordSession)
	if err != nil {
		return signer
	}
	defer rows.Close()
	for rows.Next() {
		var body string
		var c SessionCert
		if rows.Scan(&body) == nil && json.Unmarshal([]byte(body), &c) == nil && c.Session == signer && c.CloudKey != "" {
			return c.CloudKey
		}
	}
	return signer
}

// listsHeaderHeld is the highest header version a cloud session has taken from
// machine, 0 for none.
func (m *Membership) listsHeaderHeld(machine string) uint64 {
	var v uint64
	_ = m.db.QueryRow(`SELECT version FROM lists_headers WHERE machine = ?`, machine).Scan(&v)
	return v
}

// recordListsHeader keeps the version of a header taken, never lowering it.
func (m *Membership) recordListsHeader(h ListsHeader) error {
	_, err := m.db.Exec(`INSERT INTO lists_headers (machine, version, time) VALUES (?,?,?)
		ON CONFLICT(machine) DO UPDATE SET version = excluded.version, time = excluded.time
		WHERE excluded.version >= lists_headers.version`, h.Machine, h.Version, h.Time)
	return err
}
