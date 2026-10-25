package main

import (
	"crypto/ed25519"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// --- the identity page: where a user identity is made, recovered and given machines ---
//
// The user identity is made in a page the daemon serves, the first time the user does
// something that needs it (D-230, a user identity is made in the daemon's page the
// first time the user needs one). The page is the one place the recovery phrase is
// shown or typed (D-223, the recovery phrase is typed only into the daemon's browser
// page): a command's arguments and output pass through the model, and the page does
// not.
//
// So the phrase and the master key are held here only while a request needs them.
// The phrase is shown once, in the response to the request that drew it, and never
// written to a log, a file, an error or any other response; the master key is derived
// while a request runs and zeroed before it returns. Every route below is guarded
// (D-087, the local API requires a header that a web page cannot send), since each
// either writes the identity or reads what only the user's own page should.

//go:embed identity.html
var identityPageSrc string

// createTTL bounds how long a drawn phrase waits for the user to confirm it. A phrase
// not confirmed in that time is dropped, and the page draws another.
const createTTL = 30 * time.Minute

// selfFlows holds the identity page's work in progress.
type selfFlows struct {
	mu       sync.Mutex
	creating map[string]*pendingCreate
	join     *pendingJoin
	adds     map[string]*pendingAdd
	// acked is, for each of the user's other machines, the hashes of the records it
	// has acknowledged, so a record is sent to each until it has (D-222, a machine
	// sends its list to every machine it knows until each has acknowledged). It is in
	// memory, so a restarted daemon sends everything once more.
	acked map[string]map[string]bool
}

// pendingCreate is a phrase drawn and shown, waiting for the user to type back the
// words at the positions confirm names.
type pendingCreate struct {
	phrase  Phrase
	confirm []int
	created time.Time
}

func (d *Daemon) advertised() string {
	if d.advertise != "" {
		return d.advertise
	}
	return AdvertisedEndpoint()
}

func (d *Daemon) homeDirectory() string {
	if d.home != "" {
		return d.home
	}
	return homeDir()
}

func (d *Daemon) selfLocalRoutes(mux *http.ServeMux) {
	// The page itself is a GET a person navigated to, and holds nothing until its
	// script asks the guarded routes below.
	mux.HandleFunc("/identity", d.handleIdentityPage)
	mux.HandleFunc("/identity/state", guardLocal(d.handleIdentityState))
	mux.HandleFunc("/identity/phrase", guardLocal(d.handleIdentityPhrase))
	mux.HandleFunc("/identity/create", guardLocal(d.handleIdentityCreate))
	mux.HandleFunc("/identity/recover", guardLocal(d.handleIdentityRecover))
	mux.HandleFunc("/identity/join/start", guardLocal(d.handleJoinStart))
	mux.HandleFunc("/identity/join/status", guardLocal(d.handleJoinStatus))
	mux.HandleFunc("/identity/join/confirm", guardLocal(d.handleJoinConfirm))
	mux.HandleFunc("/machines/add/start", guardLocal(d.handleAddStart))
	mux.HandleFunc("/machines/add/confirm", guardLocal(d.handleAddConfirm))
	mux.HandleFunc("/machines/rename", guardLocal(d.handleMachineRename))
	// Removing a machine, and certifying a new shared key with the phrase typed into
	// the page (D-220; D-223, the recovery phrase is typed only into the daemon's
	// browser page).
	mux.HandleFunc("/machines/remove/propose", guardLocal(d.handleRemovePropose))
	mux.HandleFunc("/machines/remove", guardLocal(d.handleRemove))
	mux.HandleFunc("/machines/newkey/preview", guardLocal(d.handleNewKeyPreview))
	mux.HandleFunc("/machines/newkey", guardLocal(d.handleNewKey))
}

func (d *Daemon) selfPeerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/self/add", d.handleMachineAdd)
	mux.HandleFunc("/self/lists", d.handleSelfLists)
}

func (d *Daemon) handleIdentityPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(identityPageSrc))
}

// --- what the page shows ---

type machineView struct {
	Machine string `json:"machine"`
	Name    string `json:"name"`
	Serial  uint64 `json:"serial"`
	Self    bool   `json:"self,omitempty"`
	// Current is false for a machine not certified under the user's current shared
	// key, which is suspended until it is certified again (D-234).
	Current bool `json:"current"`
	// Reach is "reachable", "not reachable", "not tried" or "checking", as this
	// machine last found it.
	Reach  string `json:"reach,omitempty"`
	Trying string `json:"trying,omitempty"`
}

type identityState struct {
	HasUser     bool          `json:"hasUser"`
	User        string        `json:"user,omitempty"`
	UserName    string        `json:"userName,omitempty"`
	DisplayName string        `json:"displayName,omitempty"`
	Hostname    string        `json:"hostname"`
	Machine     string        `json:"machine"`
	CanCertify  bool          `json:"canCertify"`
	Machines    []machineView `json:"machines,omitempty"`
}

func hostname() string {
	h, _ := os.Hostname()
	return sanitizeName(h)
}

func (d *Daemon) handleIdentityState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, d.identityState())
}

func (d *Daemon) identityState() identityState {
	id := d.id
	st := identityState{Hostname: hostname(), Machine: id.PeerID}
	if !id.HasUser() {
		return st
	}
	_, _, have, _ := loadSharedKeyIn(d.homeDirectory())
	st.HasUser, st.User, st.UserName, st.DisplayName, st.CanCertify = true, id.User, PeerName(id.User), id.UserDisplayName, have
	st.Machines = d.machineViews()
	return st
}

// machineViews is the user's machines as this machine knows them, by serial, with
// whether each can be reached from here (D-231, a user's machines are managed from
// one command that opens the view).
func (d *Daemon) machineViews() []machineView {
	id := d.id
	certs, current := d.members.displayCerts(id.User)
	var out []machineView
	for key, c := range certs {
		v := machineView{Machine: key, Name: c.Name, Serial: c.Serial, Self: key == id.PeerID, Current: current[key]}
		if !v.Self {
			v.Reach = "not tried"
			if ep, _ := d.members.MachineEndpoint(key); ep != "" {
				p := d.statusOf([]string{ep})[0]
				switch {
				case p.Online:
					v.Reach = "reachable"
				case p.Trying != "":
					v.Reach, v.Trying = "not reachable", p.Trying
				default:
					v.Reach = p.Since
				}
			}
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Serial != out[j].Serial {
			return out[i].Serial < out[j].Serial
		}
		return out[i].Machine < out[j].Machine
	})
	return out
}

// displayCerts is, for each machine of user not removed, the certificate Resolve
// settles on, whose name and serial are shown and carried; Resolve takes the
// highest serial under a version, so the latest renaming is the name (D-231,
// renaming a machine certifies it again under its key with the new name). current
// says which machines are certified under the user's current shared key.
func (m *Membership) displayCerts(user string) (map[string]MachineCert, map[string]bool) {
	out, current := map[string]MachineCert{}, map[string]bool{}
	rs, _, err := m.IdentityRecords(user)
	if err != nil {
		return out, current
	}
	for k, ms := range Resolve(user, rs).Machines {
		if !ms.Removed {
			out[k], current[k] = ms.Cert, ms.Current
		}
	}
	return out, current
}

// highestSerial is the highest serial any certificate of user's this machine holds
// carries, valid or not, so the next is one more (design 3, a serial is one more than
// the highest the issuing machine knows for the user).
func (m *Membership) highestSerial(user string) uint64 {
	rs, _, _ := m.IdentityRecords(user)
	var top uint64
	for _, c := range rs.Machines {
		if c.Serial > top {
			top = c.Serial
		}
	}
	return top
}

// --- creating a user: the phrase, its confirmation, and the first machine ---

type phraseResponse struct {
	CreateID string   `json:"createId,omitempty"`
	Words    []string `json:"words,omitempty"`
	// Confirm is the positions, counted from 1, of the words the user types back.
	Confirm []int  `json:"confirm,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleIdentityPhrase draws the phrase of a new user and shows it. This response
// is the only one that ever carries the words.
func (d *Daemon) handleIdentityPhrase(w http.ResponseWriter, r *http.Request) {
	if d.id.HasUser() {
		writeJSON(w, phraseResponse{Error: "this machine already speaks for a user"})
		return
	}
	draw := d.newPhrase
	if draw == nil {
		draw = NewPhrase
	}
	p, err := draw()
	if err != nil {
		writeJSON(w, phraseResponse{Error: "could not draw a phrase from the system's random source"})
		return
	}
	confirm, err := confirmPositions(3)
	if err != nil {
		writeJSON(w, phraseResponse{Error: "could not choose words to confirm"})
		return
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		writeJSON(w, phraseResponse{Error: "could not draw an identifier for this page"})
		return
	}
	cid := base64.RawURLEncoding.EncodeToString(raw)
	d.flows.mu.Lock()
	if d.flows.creating == nil {
		d.flows.creating = map[string]*pendingCreate{}
	}
	for k, v := range d.flows.creating {
		if time.Since(v.created) > createTTL {
			v.phrase = Phrase{}
			delete(d.flows.creating, k)
		}
	}
	d.flows.creating[cid] = &pendingCreate{phrase: p, confirm: confirm, created: time.Now()}
	d.flows.mu.Unlock()
	out := phraseResponse{CreateID: cid, Words: p.Words()}
	for _, c := range confirm {
		out.Confirm = append(out.Confirm, c+1)
	}
	writeJSON(w, out)
}

// confirmPositions picks n distinct positions of the phrase, in order.
func confirmPositions(n int) ([]int, error) {
	seen := map[int]bool{}
	var out []int
	for len(out) < n {
		var b [2]byte
		if _, err := rand.Read(b[:]); err != nil {
			return nil, err
		}
		p := int(binary.BigEndian.Uint16(b[:])) % phraseWords
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	sort.Ints(out)
	return out, nil
}

type createRequest struct {
	CreateID string `json:"createId"`
	// Words maps a position, counted from 1, to the word the user typed for it.
	Words       map[int]string `json:"words"`
	DisplayName string         `json:"displayName"`
	MachineName string         `json:"machineName"`
}

type identityResult struct {
	Status string `json:"status,omitempty"`
	User   string `json:"user,omitempty"`
	Error  string `json:"error,omitempty"`
}

// handleIdentityCreate makes the user once the words typed back match the phrase:
// the master key certifies shared key 1, the shared key certifies this machine, and
// the master key and the phrase are dropped (D-230; D-219, a user is a master key that
// certifies machines through a shared key). A wrong word refuses, naming its position
// and never a word.
func (d *Daemon) handleIdentityCreate(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	display, machine := sanitizeName(req.DisplayName), sanitizeName(req.MachineName)
	if display == "" {
		writeJSON(w, identityResult{Error: "Give the name your colleagues will see."})
		return
	}
	if machine == "" {
		writeJSON(w, identityResult{Error: "Give this machine a name."})
		return
	}
	d.flows.mu.Lock()
	pc, ok := d.flows.creating[req.CreateID]
	d.flows.mu.Unlock()
	if !ok || time.Since(pc.created) > createTTL {
		writeJSON(w, identityResult{Error: "This phrase is no longer waiting. Start again, and a new phrase is drawn."})
		return
	}
	words := pc.phrase.Words()
	for _, pos := range pc.confirm {
		typed := strings.TrimSpace(req.Words[pos+1])
		if !strings.EqualFold(typed, words[pos]) {
			writeJSON(w, identityResult{Error: fmt.Sprintf(
				"Word %d does not match the phrase. Check what you wrote down, since recovery needs every word.", pos+1)})
			return
		}
	}
	d.flows.mu.Lock()
	delete(d.flows.creating, req.CreateID)
	d.flows.mu.Unlock()
	user, err := d.makeUser(pc.phrase, display, machine)
	pc.phrase = Phrase{}
	if err != nil {
		writeJSON(w, identityResult{Error: err.Error()})
		return
	}
	writeJSON(w, identityResult{Status: "created", User: user})
}

// makeUser derives the master key from p, certifies shared key 1 and this machine as
// serial 1, writes shared.key and identity.json, and zeroes the master key.
func (d *Daemon) makeUser(p Phrase, display, machine string) (string, error) {
	master := p.MasterKey()
	defer zero(master)
	user := PeerIDFromPublic(master.Public().(ed25519.PublicKey))
	sharedPub, shared, err := ed25519.GenerateKey(nil)
	if err != nil {
		return "", err
	}
	chain := Chain{
		Shared:  CertifySharedKey(master, 1, sharedPub),
		Machine: CertifyMachine(shared, user, 1, d.id.private.Public().(ed25519.PublicKey), KindMachine, machine, 1),
	}
	return user, d.becomeUser(user, display, chain, shared, 1)
}

// becomeUser makes this machine one of user's: the shared key in shared.key, the
// chain in identity.json and membership.db, and the daemon presenting the chain from
// its next handshake. It is the last step of creating a user, adding a machine and
// recovering.
func (d *Daemon) becomeUser(user, display string, chain Chain, shared ed25519.PrivateKey, version uint64) error {
	if _, err := chain.Verify(user, d.id.PeerID); err != nil {
		return fmt.Errorf("the certificates made for this machine do not verify: %w", err)
	}
	home := d.homeDirectory()
	if err := saveSharedKeyIn(home, shared, version); err != nil {
		return err
	}
	if err := d.members.StoreChain(chain); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	next := *d.id
	next.User, next.Chain = user, &chain
	if display != "" {
		next.UserDisplayName, next.NameChosen = display, true
	}
	// A turn carries the machine's name as the user gave it (D-233, every turn carries
	// the name of the machine it came from).
	next.MachineID = chain.Machine.Name
	if err := saveIdentityIn(home, &next); err != nil {
		return err
	}
	d.id = &next
	d.tls.reset()
	return nil
}

// storeOwnChain keeps this machine's own chain with its user's records. Creating a
// user, adding a machine and recovering keep it in becomeUser; this is for a daemon
// starting with a chain already in identity.json.
func (d *Daemon) storeOwnChain() {
	if !d.id.HasUser() {
		return
	}
	if err := d.members.StoreChain(*d.id.Chain); err != nil {
		log.Printf("identity: this machine's own certificates were not kept: %v", err)
	}
}

// saveIdentityIn writes identity.json into dir. The private key is never in it.
func saveIdentityIn(dir string, id *Identity) error {
	buf, err := json.MarshalIndent(id, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "identity.json"), buf, 0o600)
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// --- recovering from the phrase ---

type recoverRequest struct {
	Phrase      string `json:"phrase"`
	DisplayName string `json:"displayName"`
	MachineName string `json:"machineName"`
}

// handleIdentityRecover takes the phrase, derives the master key, certifies a new
// shared key one version above the highest this machine can learn of, certifies this
// machine under it, and drops the master key (D-223; D-234, a machine not certified
// under the current shared key is suspended until it is certified again). A machine
// that holds no record of the user has learned of no version, and takes 2, since the
// user's first machine took 1. Every machine not certified again is suspended by the
// rule Resolve already applies, so recovery names none of them.
func (d *Daemon) handleIdentityRecover(w http.ResponseWriter, r *http.Request) {
	var req recoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	p, err := ParsePhrase(req.Phrase)
	req.Phrase = ""
	if err != nil {
		writeJSON(w, identityResult{Error: "The phrase does not read: " + err.Error() + "."})
		return
	}
	defer func() { p = Phrase{} }()
	user := p.UserID()
	id := d.id
	if id.HasUser() && id.User != user {
		writeJSON(w, identityResult{Error: "This machine already speaks for a different user, and the phrase is not theirs."})
		return
	}
	display := sanitizeName(req.DisplayName)
	if id.HasUser() {
		display = firstNonEmpty(display, id.UserDisplayName)
	}
	machine := sanitizeName(req.MachineName)
	if machine == "" && id.HasUser() {
		machine = id.Chain.Machine.Name
	}
	if display == "" {
		writeJSON(w, identityResult{Error: "Give the name your colleagues will see."})
		return
	}
	if machine == "" {
		writeJSON(w, identityResult{Error: "Give this machine a name."})
		return
	}
	rs, _, _ := d.members.IdentityRecords(user)
	version := uint64(2)
	for _, c := range rs.Shared {
		if c.Version < maxSharedVersion && c.Version+1 > version {
			version = c.Version + 1
		}
	}
	master := p.MasterKey()
	defer zero(master)
	sharedPub, shared, err := ed25519.GenerateKey(nil)
	if err != nil {
		writeJSON(w, identityResult{Error: err.Error()})
		return
	}
	chain := Chain{
		Shared: CertifySharedKey(master, version, sharedPub),
		Machine: CertifyMachine(shared, user, version, id.private.Public().(ed25519.PublicKey), KindMachine,
			machine, d.members.highestSerial(user)+1),
	}
	zero(master)
	if err := d.becomeUser(user, display, chain, shared, version); err != nil {
		writeJSON(w, identityResult{Error: err.Error()})
		return
	}
	writeJSON(w, identityResult{Status: "recovered", User: user})
}

// --- renaming a machine ---

type renameRequest struct {
	Machine string `json:"machine"`
	Name    string `json:"name"`
}

// handleMachineRename certifies a machine of the user again, under its key and the
// current shared key, with a new name and the next serial (D-231). The renamed
// machine takes the certificate as its own chain when it next exchanges lists with
// this one, and its turns carry the new name from then (D-233).
func (d *Daemon) handleMachineRename(w http.ResponseWriter, r *http.Request) {
	var req renameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	name := sanitizeName(req.Name)
	if name == "" {
		writeJSON(w, identityResult{Error: "Give the machine a name."})
		return
	}
	if err := d.certifyMachine(req.Machine, name); err != nil {
		writeJSON(w, identityResult{Error: err.Error()})
		return
	}
	d.refreshOwnChain()
	writeJSON(w, identityResult{Status: "renamed"})
}

// certifyMachine signs a certificate for one of the user's machines with this
// machine's shared key, and keeps it with the user's records.
func (d *Daemon) certifyMachine(machine, name string) error {
	id := d.id
	if !id.HasUser() {
		return errors.New("this machine speaks for no user")
	}
	pub, err := PublicFromPeerID(machine)
	if err != nil {
		return err
	}
	shared, version, ok, err := loadSharedKeyIn(d.homeDirectory())
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("this machine holds no shared key, so it cannot certify a machine")
	}
	kind := KindMachine
	if certs, _ := d.members.displayCerts(id.User); certs[machine].Kind != "" {
		kind = certs[machine].Kind
	}
	cert := CertifyMachine(shared, id.User, version, pub, kind, name, d.members.highestSerial(id.User)+1)
	body, err := json.Marshal(cert)
	if err != nil {
		return err
	}
	_, _, err = d.members.PutIdentityRecord(RecordMachine, id.User, body)
	return err
}

// refreshOwnChain takes as this machine's chain the certificate its user's records
// now give it, when that is not the chain it presents: after a renaming, or a
// certification under a newer shared key made on another of the user's machines.
func (d *Daemon) refreshOwnChain() {
	id := d.id
	if !id.HasUser() {
		return
	}
	certs, _ := d.members.displayCerts(id.User)
	cert, ok := certs[id.PeerID]
	if !ok || cert.Hash() == id.Chain.Machine.Hash() {
		return
	}
	rs, _, _ := d.members.IdentityRecords(id.User)
	st := Resolve(id.User, rs)
	var shared *SharedKeyCert
	for i, c := range rs.Shared {
		if c.Version == cert.SharedVersion && c.SharedKey == st.SharedKeys[c.Version] {
			shared = &rs.Shared[i]
			break
		}
	}
	if shared == nil {
		return
	}
	chain := Chain{Shared: *shared, Machine: cert}
	if _, err := chain.Verify(id.User, id.PeerID); err != nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	next := *d.id
	next.Chain = &chain
	next.MachineID = cert.Name
	if err := saveIdentityIn(d.homeDirectory(), &next); err != nil {
		return
	}
	d.id = &next
	d.tls.reset()
}
