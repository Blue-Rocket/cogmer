package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// --- adding a machine from another of the user's machines (D-229) ---
//
// The new machine's page shows its pairing string, which is its machine key and an
// address. The user enters it in the page of a machine that already speaks for the
// user, which dials the new machine over a connection pinned to that key and runs the
// commit-then-reveal exchange of D-048 under the machine tags, so both pages show two
// words. The new machine's page also asks for its name. Only once the user has said
// the words match on both pages does the existing machine certify the new one with
// the shared key and send it, over the same pinned connection, the shared key, the
// user's machines with their addresses and every record of the user, the copied lists
// among them (D-228). The new machine writes them and is a machine of the user.
//
// Nothing in the exchange is signed apart from TLS: both ends are the keys of the
// pinned connection, which is what the words are computed over, so a machine in the
// middle has to present one of the two keys to take part.

// joinTTL bounds how long a page waits to be added.
const joinTTL = 15 * time.Minute

// pendingJoin is this machine's page waiting to be added by another machine.
type pendingJoin struct {
	nonce, commit []byte
	created       time.Time
	// The machine that runs the exchange, and its user, once its commitment arrives.
	adder, adderUser        string
	theirCommit, theirNonce []byte
	words                   string
	// state is "waiting", "compare", "confirmed", "refused", "added" or "failed".
	state string
	name  string
	err   string
}

// awaitingAdd reports whether this machine's page is waiting to be added, which is
// the only time it accepts a connection from a machine of a user it does not know.
func (d *Daemon) awaitingAdd() bool {
	d.flows.mu.Lock()
	defer d.flows.mu.Unlock()
	j := d.flows.join
	return j != nil && time.Since(j.created) < joinTTL && (j.state == "waiting" || j.state == "compare" || j.state == "confirmed")
}

type joinStartResponse struct {
	String string `json:"string,omitempty"`
	Error  string `json:"error,omitempty"`
}

// handleJoinStart is the new machine's page asking to be added: a fresh nonce is
// drawn and committed to, and the page is given the string to enter on the other
// machine.
func (d *Daemon) handleJoinStart(w http.ResponseWriter, r *http.Request) {
	if d.id.HasUser() {
		writeJSON(w, joinStartResponse{Error: "this machine already speaks for a user"})
		return
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		writeJSON(w, joinStartResponse{Error: "could not draw a nonce"})
		return
	}
	d.flows.mu.Lock()
	d.flows.join = &pendingJoin{nonce: nonce, commit: sasMachineCommitment(d.id.PeerID, nonce), created: time.Now(), state: "waiting"}
	d.flows.mu.Unlock()
	writeJSON(w, joinStartResponse{String: pairingString(d.id.PeerID, d.advertised(), "")})
}

type joinStatus struct {
	State string `json:"state"`
	Words string `json:"words,omitempty"`
	Error string `json:"error,omitempty"`
}

func (d *Daemon) handleJoinStatus(w http.ResponseWriter, r *http.Request) {
	d.flows.mu.Lock()
	defer d.flows.mu.Unlock()
	j := d.flows.join
	if j == nil {
		writeJSON(w, joinStatus{State: "none"})
		return
	}
	if time.Since(j.created) > joinTTL && j.state != "added" {
		writeJSON(w, joinStatus{State: "failed", Error: "The page waited too long. Start again."})
		return
	}
	writeJSON(w, joinStatus{State: j.state, Words: j.words, Error: j.err})
}

type joinConfirmRequest struct {
	Matched     bool   `json:"matched"`
	MachineName string `json:"machineName"`
}

// handleJoinConfirm records the user's answer on the new machine's page, with the
// name the user gave this machine.
func (d *Daemon) handleJoinConfirm(w http.ResponseWriter, r *http.Request) {
	var req joinConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	name := sanitizeName(req.MachineName)
	if req.Matched && name == "" {
		writeJSON(w, joinStatus{State: "compare", Error: "Give this machine a name."})
		return
	}
	d.flows.mu.Lock()
	defer d.flows.mu.Unlock()
	j := d.flows.join
	if j == nil || j.state != "compare" {
		writeJSON(w, joinStatus{State: "none", Error: "There are no words waiting for an answer."})
		return
	}
	if req.Matched {
		j.state, j.name = "confirmed", name
	} else {
		j.state = "refused"
	}
	writeJSON(w, joinStatus{State: j.state})
}

// --- the exchange, on the machine being added ---

type machineAddMessage struct {
	// Step is "commit", "reveal", "await", "deliver" or "abort".
	Step     string           `json:"step"`
	Payload  string           `json:"payload,omitempty"`
	Delivery *machineDelivery `json:"delivery,omitempty"`
}

type machineAddReply struct {
	Payload string `json:"payload,omitempty"`
	State   string `json:"state,omitempty"`
	Name    string `json:"name,omitempty"`
}

// machineDelivery is what the adding machine sends the machine it added: the user,
// the name colleagues see, the shared key and its version, the new machine's chain,
// every record of the user, and where each of the user's machines listens (D-229).
type machineDelivery struct {
	User          string        `json:"user"`
	DisplayName   string        `json:"displayName"`
	SharedVersion uint64        `json:"sharedVersion"`
	SharedKey     string        `json:"sharedKey"`
	Chain         Chain         `json:"chain"`
	Records       []selfRecord  `json:"records"`
	Machines      []machineAddr `json:"machines"`
}

// handleMachineAdd answers the adding machine's steps on the machine being added. It
// answers only while this machine's page waits to be added, and only the machine that
// sent the first commitment.
func (d *Daemon) handleMachineAdd(w http.ResponseWriter, r *http.Request) {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	pub, ok := r.TLS.PeerCertificates[0].PublicKey.(ed25519.PublicKey)
	if !ok {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	adder := PeerIDFromPublic(pub)
	chain, err := certChain(r.TLS.PeerCertificates[0])
	if err != nil || chain == nil {
		http.Error(w, "only a machine that speaks for a user adds a machine", http.StatusUnauthorized)
		return
	}
	var msg machineAddMessage
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&msg); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	payload, err := base64.RawURLEncoding.DecodeString(msg.Payload)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	d.flows.mu.Lock()
	j := d.flows.join
	if j == nil || time.Since(j.created) > joinTTL || d.id.HasUser() {
		d.flows.mu.Unlock()
		http.Error(w, "this machine is not waiting to be added", http.StatusConflict)
		return
	}
	if j.adder != "" && j.adder != adder {
		d.flows.mu.Unlock()
		http.Error(w, "another machine is already adding this one", http.StatusConflict)
		return
	}
	switch msg.Step {
	case "commit":
		if j.state != "waiting" {
			d.flows.mu.Unlock()
			http.Error(w, "the exchange has already begun", http.StatusConflict)
			return
		}
		j.adder, j.adderUser, j.theirCommit = adder, chain.Shared.User, payload
		reply := machineAddReply{Payload: base64.RawURLEncoding.EncodeToString(j.commit)}
		d.flows.mu.Unlock()
		writeJSON(w, reply)
	case "reveal":
		// A reveal must open the commitment that came first, or the other machine
		// could choose its nonce after seeing this one's (D-048).
		if j.theirCommit == nil || j.state != "waiting" {
			d.flows.mu.Unlock()
			http.Error(w, "reveal before commit", http.StatusBadRequest)
			return
		}
		if !sasMachineOpens(j.theirCommit, adder, payload) {
			j.state, j.err = "failed", "The other machine revealed something other than what it committed to. Something is relaying this exchange."
			d.flows.mu.Unlock()
			http.Error(w, "revealed nonce does not open the commitment", http.StatusBadRequest)
			return
		}
		j.theirNonce = payload
		j.words = SASMachine(adder, payload, d.id.PeerID, j.nonce)
		j.state = "compare"
		reply := machineAddReply{Payload: base64.RawURLEncoding.EncodeToString(j.nonce)}
		d.flows.mu.Unlock()
		writeJSON(w, reply)
	case "await":
		reply := machineAddReply{State: j.state, Name: j.name}
		d.flows.mu.Unlock()
		writeJSON(w, reply)
	case "abort":
		j.state, j.err = "failed", "The other machine's page said the words were different. Nothing was added."
		d.flows.mu.Unlock()
		writeJSON(w, machineAddReply{State: "failed"})
	case "deliver":
		if j.state != "confirmed" || msg.Delivery == nil {
			d.flows.mu.Unlock()
			http.Error(w, "the words have not been confirmed on this machine", http.StatusConflict)
			return
		}
		del, adderUser := *msg.Delivery, j.adderUser
		d.flows.mu.Unlock()
		if err := d.acceptDelivery(del, adderUser); err != nil {
			d.flows.mu.Lock()
			j.state, j.err = "failed", err.Error()
			d.flows.mu.Unlock()
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		d.flows.mu.Lock()
		j.state = "added"
		d.flows.mu.Unlock()
		writeJSON(w, machineAddReply{State: "added"})
	default:
		d.flows.mu.Unlock()
		http.Error(w, "unknown step", http.StatusBadRequest)
	}
}

// acceptDelivery makes this machine one of the user's from what the adding machine
// sent: the chain must certify this machine's own key for the user whose machine ran
// the exchange, and the shared key must be the one the chain names.
func (d *Daemon) acceptDelivery(del machineDelivery, adderUser string) error {
	if del.User == "" || del.User != adderUser {
		return errors.New("the delivery is for a user other than the one whose machine ran the exchange")
	}
	if _, err := del.Chain.Verify(del.User, d.id.PeerID); err != nil {
		return fmt.Errorf("the certificate for this machine does not verify: %w", err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(del.SharedKey)
	if err != nil || len(raw) != ed25519.PrivateKeySize {
		return errors.New("the shared key is not an Ed25519 private key")
	}
	shared := ed25519.PrivateKey(raw)
	if PeerIDFromPublic(shared.Public().(ed25519.PublicKey)) != del.Chain.Shared.SharedKey || del.SharedVersion != del.Chain.Shared.Version {
		return errors.New("the shared key is not the one the certificate names")
	}
	for _, rec := range del.Records {
		if _, _, err := d.members.PutIdentityRecord(rec.Kind, del.User, rec.Body); err != nil {
			log.Printf("add: a record of the user was not kept: %v", err)
		}
	}
	if err := d.becomeUser(del.User, sanitizeName(del.DisplayName), del.Chain, shared, del.SharedVersion); err != nil {
		return err
	}
	d.recordMachineAddrs(del.Machines)
	if err := d.members.ApplyUserLists(d.id); err != nil {
		log.Printf("add: the user's lists were not all applied: %v", err)
	}
	d.notify()
	return nil
}

// --- the exchange, on the machine that adds ---

// pendingAdd is a machine being added from this one, between the words and the
// user's answer.
type pendingAdd struct {
	machine, endpoint string
	words             string
	created           time.Time
}

type addStartRequest struct {
	String string `json:"string"`
}

type addStartResponse struct {
	AddID string `json:"addId,omitempty"`
	Words string `json:"words,omitempty"`
	Error string `json:"error,omitempty"`
}

// handleAddStart runs the exchange with the machine whose string the user entered,
// and shows the words.
func (d *Daemon) handleAddStart(w http.ResponseWriter, r *http.Request) {
	var req addStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !d.id.HasUser() {
		writeJSON(w, addStartResponse{Error: "This machine speaks for no user yet, so it cannot add one."})
		return
	}
	if _, _, ok, _ := loadSharedKeyIn(d.homeDirectory()); !ok {
		writeJSON(w, addStartResponse{Error: "This machine holds no shared key, so it cannot certify a machine."})
		return
	}
	machine, endpoint, _ := parsePairing(req.String)
	if _, err := PublicFromPeerID(machine); err != nil || endpoint == "" {
		writeJSON(w, addStartResponse{Error: "That is not the string the new machine's page shows. It is a key, an @ and an address."})
		return
	}
	if machine == d.id.PeerID {
		writeJSON(w, addStartResponse{Error: "That is this machine's own string."})
		return
	}
	words, err := d.runMachineExchange(machine, endpoint)
	if err != nil {
		writeJSON(w, addStartResponse{Error: err.Error()})
		return
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		writeJSON(w, addStartResponse{Error: "could not draw an identifier"})
		return
	}
	aid := base64.RawURLEncoding.EncodeToString(raw)
	d.flows.mu.Lock()
	if d.flows.adds == nil {
		d.flows.adds = map[string]*pendingAdd{}
	}
	d.flows.adds[aid] = &pendingAdd{machine: machine, endpoint: endpoint, words: words, created: time.Now()}
	d.flows.mu.Unlock()
	writeJSON(w, addStartResponse{AddID: aid, Words: words})
}

func (d *Daemon) machineClient(machine, endpoint string) (*http.Client, error) {
	conf, err := d.machineClientConfig(machine)
	if err != nil {
		return nil, err
	}
	return clientFor(endpoint, 15*time.Second, conf)
}

// machineStep posts one step to the machine being added.
func machineStep(client *http.Client, msg machineAddMessage) (machineAddReply, error) {
	body, err := json.Marshal(msg)
	if err != nil {
		return machineAddReply{}, err
	}
	resp, err := client.Post(peerURL("/self/add"), "application/json", bytes.NewReader(body))
	if err != nil {
		return machineAddReply{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		text, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<12))
		return machineAddReply{}, fmt.Errorf("the new machine refused: %s", bytes.TrimSpace(text))
	}
	var out machineAddReply
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

// runMachineExchange commits, then reveals, and returns the two words.
func (d *Daemon) runMachineExchange(machine, endpoint string) (string, error) {
	client, err := d.machineClient(machine, endpoint)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	enc := base64.RawURLEncoding.EncodeToString
	theirs, err := machineStep(client, machineAddMessage{Step: "commit", Payload: enc(sasMachineCommitment(d.id.PeerID, nonce))})
	if err != nil {
		return "", fmt.Errorf("could not reach the new machine: %w", err)
	}
	theirCommit, err := base64.RawURLEncoding.DecodeString(theirs.Payload)
	if err != nil {
		return "", err
	}
	revealed, err := machineStep(client, machineAddMessage{Step: "reveal", Payload: enc(nonce)})
	if err != nil {
		return "", err
	}
	theirNonce, err := base64.RawURLEncoding.DecodeString(revealed.Payload)
	if err != nil {
		return "", err
	}
	if !sasMachineOpens(theirCommit, machine, theirNonce) {
		return "", errors.New("the new machine revealed something other than what it committed to; something is relaying this exchange, and it must not be retried")
	}
	return SASMachine(d.id.PeerID, nonce, machine, theirNonce), nil
}

type addConfirmRequest struct {
	AddID   string `json:"addId"`
	Matched bool   `json:"matched"`
}

type addConfirmResponse struct {
	Status string `json:"status,omitempty"`
	Name   string `json:"name,omitempty"`
	Error  string `json:"error,omitempty"`
}

// addAwait is how long the adding machine waits for the user to answer on the new
// machine's page, and how often it asks.
var addAwait, addAwaitEvery = 3 * time.Minute, time.Second

// handleAddConfirm records the user's answer on the adding machine. A match waits for
// the same answer on the new machine's page, then certifies the new machine with the
// name given there and sends it everything it needs (D-229).
func (d *Daemon) handleAddConfirm(w http.ResponseWriter, r *http.Request) {
	var req addConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	d.flows.mu.Lock()
	pa, ok := d.flows.adds[req.AddID]
	delete(d.flows.adds, req.AddID)
	d.flows.mu.Unlock()
	if !ok || time.Since(pa.created) > joinTTL {
		writeJSON(w, addConfirmResponse{Error: "These words are no longer waiting. Start again from the new machine's page."})
		return
	}
	client, err := d.machineClient(pa.machine, pa.endpoint)
	if err != nil {
		writeJSON(w, addConfirmResponse{Error: err.Error()})
		return
	}
	if !req.Matched {
		_, _ = machineStep(client, machineAddMessage{Step: "abort"})
		writeJSON(w, addConfirmResponse{Status: "not added"})
		return
	}
	name, err := awaitNewMachine(client)
	if err != nil {
		writeJSON(w, addConfirmResponse{Error: err.Error()})
		return
	}
	del, err := d.deliveryFor(pa.machine, name)
	if err != nil {
		writeJSON(w, addConfirmResponse{Error: err.Error()})
		return
	}
	if _, err := machineStep(client, machineAddMessage{Step: "deliver", Delivery: &del}); err != nil {
		writeJSON(w, addConfirmResponse{Error: err.Error()})
		return
	}
	if err := d.members.SetMachineEndpoint(pa.machine, d.id.User, pa.endpoint); err != nil {
		log.Printf("add: could not record where %s listens: %v", name, err)
	}
	d.notify()
	writeJSON(w, addConfirmResponse{Status: "added", Name: name})
}

// awaitNewMachine waits for the user's answer on the new machine's page.
func awaitNewMachine(client *http.Client) (string, error) {
	deadline := time.Now().Add(addAwait)
	for {
		reply, err := machineStep(client, machineAddMessage{Step: "await"})
		if err != nil {
			return "", err
		}
		switch reply.State {
		case "confirmed":
			return reply.Name, nil
		case "refused", "failed":
			return "", errors.New("the new machine's page said the words were different, so nothing was added; " +
				"something may be relaying this exchange")
		}
		if time.Now().After(deadline) {
			return "", errors.New("nobody answered on the new machine's page, so nothing was added")
		}
		time.Sleep(addAwaitEvery)
	}
}

// deliveryFor certifies the new machine and gathers what it is sent.
func (d *Daemon) deliveryFor(machine, name string) (machineDelivery, error) {
	id := d.id
	if err := d.certifyMachine(machine, name); err != nil {
		return machineDelivery{}, err
	}
	shared, version, _, err := loadSharedKeyIn(d.homeDirectory())
	if err != nil {
		return machineDelivery{}, err
	}
	certs, _ := d.members.displayCerts(id.User)
	cert, ok := certs[machine]
	if !ok {
		return machineDelivery{}, errors.New("the certificate made for the new machine was not kept")
	}
	rs, _, _ := d.members.IdentityRecords(id.User)
	var sharedCert *SharedKeyCert
	for i, c := range rs.Shared {
		if c.Version == version && c.SharedKey == PeerIDFromPublic(shared.Public().(ed25519.PublicKey)) {
			sharedCert = &rs.Shared[i]
		}
	}
	if sharedCert == nil {
		return machineDelivery{}, errors.New("this machine holds no certificate for its own shared key")
	}
	return machineDelivery{
		User: id.User, DisplayName: id.UserDisplayName,
		SharedVersion: version, SharedKey: base64.RawURLEncoding.EncodeToString(shared),
		Chain:    Chain{Shared: *sharedCert, Machine: cert},
		Records:  d.members.selfRecords(id.User),
		Machines: d.ownMachineAddrs(),
	}, nil
}
