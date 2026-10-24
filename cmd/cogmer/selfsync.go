package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// --- the exchange of records between a user's own machines (D-222, D-228) ---
//
// Any two of a user's machines exchange the user's identity records, the changes to
// the user's copied lists among them, and where each of the user's machines listens.
// Each machine sends every record to each of the user's other machines until that
// machine has acknowledged it, and a record is acknowledged by the answer to the
// request that carried it (D-222, a machine tells every machine it knows of each
// machine it adds, until each has acknowledged). The records only accumulate and
// sign themselves, so any order of arrival settles on the same lists.
//
// A request is signed by the machine that sends it, carries its chain, and travels
// over a connection pinned to that machine's key, like a sync request (D-044, sync
// requests are signed; D-101, peer connections are TLS pinned), and is answered only
// for a machine of this machine's own user that is neither removed nor suspended.

// selfListsTag is the purpose tag of a request to exchange a user's records, so its
// signature can be taken for no other request.
const selfListsTag = protocolNamespace + "/user-lists/v1"

// selfRecord is one identity record as it travels: its kind and its JSON.
type selfRecord struct {
	Kind string          `json:"kind"`
	Body json.RawMessage `json:"body"`
}

// machineAddr is where one of the user's machines was last known to listen.
type machineAddr struct {
	Machine  string `json:"machine"`
	Endpoint string `json:"endpoint"`
}

// selfListsPayload is what each side sends: the records the other has not
// acknowledged, the hashes of every record the sender holds, so the answer carries
// only what the sender lacks, and the user's machines with their addresses.
type selfListsPayload struct {
	Records  []selfRecord  `json:"records,omitempty"`
	Have     []string      `json:"have,omitempty"`
	Machines []machineAddr `json:"machines,omitempty"`
	// HeldShared is the version of the shared key the sender holds, and Shared the
	// current key for a machine of the user that holds an older one, sent over this
	// pinned connection (D-219, every machine the user adds holds the shared key).
	HeldShared uint64         `json:"heldShared,omitempty"`
	Shared     *sharedKeyGift `json:"shared,omitempty"`
}

type selfListsRequest struct {
	User      string          `json:"user"`
	Signer    string          `json:"signer"`
	Chain     *Chain          `json:"chain"`
	Timestamp string          `json:"timestamp"`
	Nonce     string          `json:"nonce"`
	Endpoint  string          `json:"endpoint,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"`
}

// selfListsBytes is what a request's signature covers: the fields that say who asks
// and where it listens, and the hash of the payload, length-prefixed behind the tag.
func selfListsBytes(user, signer, ts, nonce, endpoint string, payload []byte) []byte {
	sum := sha256.Sum256(payload)
	var b []byte
	put := func(s string) {
		b = binary.BigEndian.AppendUint32(b, uint32(len(s)))
		b = append(b, s...)
	}
	put(selfListsTag)
	put(user)
	put(signer)
	put(ts)
	put(nonce)
	put(endpoint)
	put(string(sum[:]))
	return b
}

// selfRecordKinds are the kinds of record a user's machines exchange. A session
// certificate travels in the chain of the event it came with, and is not exchanged.
var selfRecordKinds = []string{RecordShared, RecordMachine, RecordRemoval, RecordList}

// storedRecord is a record with the hash it is stored under.
type storedRecord struct {
	Hash string
	selfRecord
}

func (m *Membership) storedSelfRecords(user string) []storedRecord {
	rows, err := m.db.Query(`SELECT hash, kind, body FROM identity_records WHERE user = ? AND kind IN (?,?,?,?) ORDER BY hash`,
		user, selfRecordKinds[0], selfRecordKinds[1], selfRecordKinds[2], selfRecordKinds[3])
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []storedRecord
	for rows.Next() {
		var r storedRecord
		var body string
		if rows.Scan(&r.Hash, &r.Kind, &body) == nil {
			r.Body = json.RawMessage(body)
			out = append(out, r)
		}
	}
	return out
}

// selfRecords is every record of user that the user's machines exchange.
func (m *Membership) selfRecords(user string) []selfRecord {
	var out []selfRecord
	for _, r := range m.storedSelfRecords(user) {
		out = append(out, r.selfRecord)
	}
	return out
}

// ownMachineAddrs is where each of the user's machines listens, this one included.
func (d *Daemon) ownMachineAddrs() []machineAddr {
	out := []machineAddr{{Machine: d.id.PeerID, Endpoint: d.advertised()}}
	ms, _ := d.members.UserMachines(d.id.User)
	for _, m := range ms {
		if m.MachineKey != d.id.PeerID && m.Endpoint != "" {
			out = append(out, machineAddr{Machine: m.MachineKey, Endpoint: m.Endpoint})
		}
	}
	return out
}

// recordMachineAddrs keeps the addresses of the user's machines another of them sent.
// An address is kept for a machine this machine has none for, since the sender's
// knowledge is no fresher than this machine's own (D-229, each exchange of machine
// lists also repairs the addresses it carries).
func (d *Daemon) recordMachineAddrs(addrs []machineAddr) {
	for _, a := range addrs {
		if a.Machine == d.id.PeerID || a.Endpoint == "" {
			continue
		}
		if _, err := PublicFromPeerID(a.Machine); err != nil {
			continue
		}
		if cur, _ := d.members.MachineEndpoint(a.Machine); cur != "" {
			continue
		}
		if err := d.members.SetMachineEndpoint(a.Machine, d.id.User, a.Endpoint); err != nil {
			log.Printf("lists: could not record where %s listens: %v", PeerName(a.Machine), err)
		}
	}
}

// ingestSelfRecords keeps the records another of the user's machines sent and makes
// this machine follow them: its own chain when it was renamed or certified again
// elsewhere, and its tables when the lists changed.
func (d *Daemon) ingestSelfRecords(p selfListsPayload) {
	user := d.id.User
	for _, r := range p.Records {
		if !knownSelfKind(r.Kind) {
			continue
		}
		if _, _, err := d.members.PutIdentityRecord(r.Kind, user, r.Body); err != nil {
			log.Printf("lists: a record was not kept: %v", err)
		}
	}
	d.recordMachineAddrs(p.Machines)
	d.refreshOwnChain()
	if err := d.members.ApplyUserLists(d.id); err != nil {
		log.Printf("lists: the user's lists were not all applied: %v", err)
	}
	if len(p.Records) > 0 {
		d.notify()
	}
}

func knownSelfKind(k string) bool {
	for _, s := range selfRecordKinds {
		if s == k {
			return true
		}
	}
	return false
}

// handleSelfLists answers another of the user's machines: it keeps what was sent,
// and answers with the records the sender does not hold.
func (d *Daemon) handleSelfLists(w http.ResponseWriter, r *http.Request) {
	if !d.id.HasUser() {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	var req selfListsRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 32<<20)).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := d.checkSelfListsRequest(req, r); err != nil {
		log.Printf("lists: refused a request from %s — %v", PeerName(req.Signer), err)
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	var p selfListsPayload
	if err := json.Unmarshal(req.Payload, &p); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Endpoint != "" {
		if err := d.members.RecordMachineAddress(req.Signer, d.id.User, req.Endpoint); err != nil {
			log.Printf("lists: could not record where %s listens: %v", PeerName(req.Signer), err)
		}
	}
	d.markPeerHeardFrom(req.Signer)
	d.ingestSelfRecords(p)
	d.acceptGift(p.Shared, req.Signer)
	have := map[string]bool{}
	for _, h := range p.Have {
		have[h] = true
	}
	var out selfListsPayload
	for _, rec := range d.members.storedSelfRecords(d.id.User) {
		if !have[rec.Hash] {
			out.Records = append(out.Records, rec.selfRecord)
		}
	}
	out.Machines = d.ownMachineAddrs()
	out.HeldShared = d.heldSharedVersion()
	out.Shared = d.giftFor(req.Signer, p.HeldShared)
	writeJSON(w, out)
}

// checkSelfListsRequest establishes that a machine of this machine's own user, on
// the other end of the pinned connection, sent the request now.
func (d *Daemon) checkSelfListsRequest(req selfListsRequest, r *http.Request) error {
	if req.User != d.id.User || req.Chain == nil {
		return errors.New("not a machine of this machine's user")
	}
	pub, err := PublicFromPeerID(req.Signer)
	if err != nil {
		return err
	}
	sig, err := base64.RawURLEncoding.DecodeString(req.Signature)
	if err != nil {
		return err
	}
	if !ed25519.Verify(pub, selfListsBytes(req.User, req.Signer, req.Timestamp, req.Nonce, req.Endpoint, req.Payload), sig) {
		return errors.New("signature does not match the machine presenting it")
	}
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		if k, ok := r.TLS.PeerCertificates[0].PublicKey.(ed25519.PublicKey); !ok || PeerIDFromPublic(k) != req.Signer {
			return errors.New("the request is signed by a machine other than the one connected")
		}
	}
	if _, err := d.members.checkLiveChain(req.User, req.Signer, *req.Chain, time.Now()); err != nil {
		return err
	}
	t, err := time.Parse(time.RFC3339Nano, req.Timestamp)
	if err != nil {
		return err
	}
	if age := time.Since(t); age > authTolerance || age < -authTolerance {
		return errors.New("stale")
	}
	if !d.replay.admit(req.Nonce, time.Now().UTC()) {
		return errors.New("replayed")
	}
	return nil
}

// exchangeSelfLists exchanges records with each of the user's other machines this
// machine has an address for. A machine that does not answer is tried at the next
// round, and its records stay unacknowledged until it does.
func (d *Daemon) exchangeSelfLists() {
	if !d.id.HasUser() {
		return
	}
	ms, err := d.members.UserMachines(d.id.User)
	if err != nil {
		return
	}
	certs, current := d.members.displayCerts(d.id.User)
	for _, m := range ms {
		// Only a machine the user's records certify, and not one suspended, is a
		// machine of the user to send the user's records to.
		if m.MachineKey == d.id.PeerID || m.Endpoint == "" {
			continue
		}
		if _, ok := certs[m.MachineKey]; !ok || !current[m.MachineKey] {
			continue
		}
		if err := d.exchangeWith(m.MachineKey, m.Endpoint); err != nil {
			d.markPeerUnreachable(m.Endpoint, err)
			continue
		}
		d.markPeerSeen(m.Endpoint)
	}
}

// exchangeWith sends one machine the records it has not acknowledged, and keeps what
// it answers with.
func (d *Daemon) exchangeWith(machine, addr string) error {
	id := d.id
	recs := d.members.storedSelfRecords(id.User)
	d.flows.mu.Lock()
	acked := d.flows.acked[machine]
	d.flows.mu.Unlock()
	var p selfListsPayload
	var sent []string
	for _, r := range recs {
		p.Have = append(p.Have, r.Hash)
		if !acked[r.Hash] {
			p.Records = append(p.Records, r.selfRecord)
			sent = append(sent, r.Hash)
		}
	}
	p.Machines = d.ownMachineAddrs()
	p.HeldShared = d.heldSharedVersion()
	// The key is sent until the machine has answered a request that carried it,
	// which is in memory, so a restarted daemon sends it once more.
	if g := d.giftFor(machine, 0); g != nil {
		if k := fmt.Sprintf("shared-key:%d", g.Version); !acked[k] {
			p.Shared = g
			sent = append(sent, k)
		}
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
		selfListsBytes(req.User, req.Signer, req.Timestamp, req.Nonce, req.Endpoint, req.Payload)))
	conf, err := d.clientConfig(id.User)
	if err != nil {
		return err
	}
	client, err := clientFor(addr, 15*time.Second, conf)
	if err != nil {
		return err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	resp, err := client.Post(peerURL("/self/lists"), "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		text, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(text)))
	}
	var answer selfListsPayload
	if err := json.NewDecoder(io.LimitReader(resp.Body, 32<<20)).Decode(&answer); err != nil {
		return err
	}
	d.flows.mu.Lock()
	if d.flows.acked == nil {
		d.flows.acked = map[string]map[string]bool{}
	}
	if d.flows.acked[machine] == nil {
		d.flows.acked[machine] = map[string]bool{}
	}
	for _, h := range sent {
		d.flows.acked[machine][h] = true
	}
	d.flows.mu.Unlock()
	d.ingestSelfRecords(answer)
	d.acceptGift(answer.Shared, machine)
	return nil
}

// RunSelfLists exchanges records with the user's other machines on a slow cycle. A
// change reaches them at the next round with nothing for the user to do (D-228).
func (d *Daemon) RunSelfLists(every time.Duration) {
	for {
		d.exchangeSelfLists()
		time.Sleep(every)
	}
}
