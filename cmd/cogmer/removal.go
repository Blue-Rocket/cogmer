package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
)

// --- removing a machine, and certifying a new shared key, from the identity page ---
//
// Both acts happen in the daemon's page, where a person sees them and the model does
// not (D-231, a user's machines are managed from one command that opens the view).
// Removing a machine signs a removal that names, room by room, the last of the
// machine's turns that counts (D-220, a removal names the last event of a machine
// that counts, room by room). Certifying a new shared key takes the recovery phrase,
// which is typed into the page and nowhere else (D-223, the recovery phrase is typed
// only into the daemon's browser page), and carries over only the removals the user
// confirms, so that a thief's removals signed with the old key end with it (D-220).
// Every route here is guarded (D-087, the local API requires a header that a web page
// cannot send).

// removalRoom is one room's cutoff as the page proposes it: the last of the machine's
// turns this machine holds there, and what the page proposes, which is that turn when
// the machine was not stolen and 0 when it was (D-220).
type removalRoom struct {
	Room     string `json:"room"`
	RoomName string `json:"roomName"`
	Last     int64  `json:"last"`
	Proposed int64  `json:"proposed"`
}

type removeProposeRequest struct {
	Machine string `json:"machine"`
	Stolen  bool   `json:"stolen"`
}

type removeProposeResponse struct {
	Machine string        `json:"machine,omitempty"`
	Name    string        `json:"name,omitempty"`
	Self    bool          `json:"self,omitempty"`
	Stolen  bool          `json:"stolen"`
	Rooms   []removalRoom `json:"rooms"`
	Error   string        `json:"error,omitempty"`
}

// lastHeldTurns is, for each room this machine holds, the highest sequence of
// machine's that its store holds. The cutoff is the removed machine's own sequence
// in the room (D-029, the sequence lives with the machine), so the page proposes it
// as this machine received it.
func (d *Daemon) lastHeldTurns(machine string) []removalRoom {
	rooms, err := d.members.Rooms()
	if err != nil {
		return nil
	}
	var out []removalRoom
	for _, r := range rooms {
		store, err := d.storeFor(r.RoomID)
		if err != nil {
			continue
		}
		d.mu.Lock()
		last, err := store.HighestSequence(machine)
		d.mu.Unlock()
		if err != nil {
			continue
		}
		out = append(out, removalRoom{Room: r.RoomID, RoomName: r.RoomName, Last: last})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Room < out[j].Room })
	return out
}

// proposeRemoval is what the page shows once the user has said whether the machine
// was lost or stolen: not stolen, the last held turn in each room, so that none of
// the user's own turns are marked; stolen, none of them (D-220).
func (d *Daemon) proposeRemoval(machine string, stolen bool) []removalRoom {
	rooms := d.lastHeldTurns(machine)
	for i := range rooms {
		if !stolen {
			rooms[i].Proposed = rooms[i].Last
		}
	}
	return rooms
}

// removableMachine is the certificate of one of the user's machines that is not
// removed already, which is what the page offers to remove.
func (d *Daemon) removableMachine(machine string) (MachineCert, error) {
	if !d.id.HasUser() {
		return MachineCert{}, errors.New("this machine speaks for no user")
	}
	certs, _ := d.members.displayCerts(d.id.User)
	c, ok := certs[machine]
	if !ok {
		return MachineCert{}, errors.New("that is not one of your machines, or it is removed already")
	}
	return c, nil
}

func (d *Daemon) handleRemovePropose(w http.ResponseWriter, r *http.Request) {
	var req removeProposeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	c, err := d.removableMachine(req.Machine)
	if err != nil {
		writeJSON(w, removeProposeResponse{Error: err.Error()})
		return
	}
	writeJSON(w, removeProposeResponse{
		Machine: req.Machine, Name: c.Name, Self: req.Machine == d.id.PeerID, Stolen: req.Stolen,
		Rooms: d.proposeRemoval(req.Machine, req.Stolen),
	})
}

type removeRequest struct {
	Machine string `json:"machine"`
	Stolen  bool   `json:"stolen"`
	// Cutoffs maps a room to the last of the machine's turns there that counts, as
	// the user confirmed or edited it. A room left out takes what the page proposed.
	Cutoffs map[string]int64 `json:"cutoffs"`
}

// cutoffsFor checks the cutoffs the user confirmed against what this machine holds,
// and returns them as a removal names them. A cutoff may be anything from 0 to the
// last turn this machine holds of the machine there: earlier than the proposal for a
// machine that was not stolen, or any held turn for one that was, since a point past
// every turn held would let turns nobody here has seen count (D-220, a cutoff only
// moves earlier). A room whose cutoff is 0 is left out, which a removal reads as 0.
func cutoffsFor(rooms []removalRoom, chosen map[string]int64) ([]RoomCutoff, error) {
	var out []RoomCutoff
	for _, r := range rooms {
		seq, ok := chosen[r.Room]
		if !ok {
			seq = r.Proposed
		}
		if seq < 0 {
			return nil, fmt.Errorf("the cutoff for %s is below 0", r.RoomName)
		}
		if seq > r.Last {
			return nil, fmt.Errorf("the cutoff for %s is past the last of the machine's turns this machine holds there, turn %d", r.RoomName, r.Last)
		}
		if seq > 0 {
			out = append(out, RoomCutoff{Room: r.Room, Sequence: seq})
		}
	}
	return out, nil
}

// handleRemove signs a removal of one of the user's machines with this machine's
// current shared key and keeps it, and the identity loop pushes it to every colleague
// and to the user's other machines until each acknowledges (D-225, a machine sends
// each removal to every colleague it knows until each acknowledges). A machine removing
// itself signs with the shared key when it holds the current one, and otherwise with
// its own key, which may remove only itself; it then drops the shared key, so that a
// machine retired from the user certifies nothing more (D-219).
func (d *Daemon) handleRemove(w http.ResponseWriter, r *http.Request) {
	var req removeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	res, err := d.removeMachine(req)
	if err != nil {
		writeJSON(w, identityResult{Error: err.Error()})
		return
	}
	writeJSON(w, res)
}

func (d *Daemon) removeMachine(req removeRequest) (identityResult, error) {
	if _, err := d.removableMachine(req.Machine); err != nil {
		return identityResult{}, err
	}
	cutoffs, err := cutoffsFor(d.proposeRemoval(req.Machine, req.Stolen), req.Cutoffs)
	if err != nil {
		return identityResult{}, err
	}
	id := d.id
	self := req.Machine == id.PeerID
	home := d.homeDirectory()
	shared, version, held, err := loadSharedKeyIn(home)
	if err != nil {
		return identityResult{}, err
	}
	// Only the current shared key removes another machine (D-220), so a machine
	// holding an older one is told so rather than signing a removal that counts for
	// nothing.
	current := d.members.UserState(id.User, nil).Current
	var rem Removal
	switch {
	case held && version == current:
		rem, err = SignRemoval(shared, LevelShared, version, id.User, req.Machine, cutoffs)
	case self:
		rem, err = SignRemoval(id.private, LevelMachine, 0, id.User, req.Machine, cutoffs)
	default:
		return identityResult{}, errors.New("this machine does not hold your current shared key, so it cannot remove another machine; remove it from a machine that does")
	}
	if err != nil {
		return identityResult{}, err
	}
	raw, err := json.Marshal(rem)
	if err != nil {
		return identityResult{}, err
	}
	if _, _, err := d.members.PutIdentityRecord(RecordRemoval, id.User, raw); err != nil {
		return identityResult{}, err
	}
	if self && held {
		if err := os.Remove(sharedKeyPathIn(home)); err != nil && !os.IsNotExist(err) {
			log.Printf("identity: this machine removed itself and could not drop its shared key: %v", err)
		}
	}
	h := rem.Hash()
	log.Printf("identity: removed machine %s, %d room cutoff(s), removal %.16s", PeerName(req.Machine), len(cutoffs), hex.EncodeToString(h[:]))
	d.notify()
	return identityResult{Status: "removed"}, nil
}

// --- certifying a new shared key ---

// heldRemoval is one removal this machine holds of its user, as the page lists it
// before a new shared key is certified: the machine it removes, the key that signed
// it, and whether the new key can carry it, which only a removal signed by a shared
// key needs, since one the master signed counts under any key and one a machine
// signed of itself counts while its certificate does.
type heldRemoval struct {
	Hash        string       `json:"hash"`
	Machine     string       `json:"machine"`
	MachineName string       `json:"machineName"`
	SignedBy    string       `json:"signedBy"`
	Carried     bool         `json:"carried"`
	Counts      bool         `json:"counts"`
	Cutoffs     []RoomCutoff `json:"cutoffs"`
}

type keyMachine struct {
	Machine string `json:"machine"`
	Name    string `json:"name"`
	Serial  uint64 `json:"serial"`
	Self    bool   `json:"self,omitempty"`
	Removed bool   `json:"removed,omitempty"`
	Current bool   `json:"current"`
}

type newKeyPreview struct {
	Version  uint64        `json:"version"`
	Removals []heldRemoval `json:"removals"`
	Machines []keyMachine  `json:"machines"`
	Error    string        `json:"error,omitempty"`
}

// signerText says which key signed a removal, in the words the page shows.
func signerText(r Removal) string {
	switch r.SignerLevel {
	case LevelMaster:
		return "your recovery phrase"
	case LevelShared:
		return fmt.Sprintf("shared key version %d", r.SignerVersion)
	case LevelMachine:
		return "the machine itself"
	}
	return "an unknown key"
}

// newKeyVersion is one above every shared-key version this machine has a record of,
// valid or not, so a new key never reuses a version a thief may have filled.
func newKeyVersion(rs RecordSet, st *UserState) uint64 {
	v := st.Current + 1
	for _, c := range rs.Shared {
		if c.Version < maxSharedVersion && c.Version+1 > v {
			v = c.Version + 1
		}
	}
	return v
}

func (d *Daemon) newKeyPreview() newKeyPreview {
	id := d.id
	if !id.HasUser() {
		return newKeyPreview{Error: "this machine speaks for no user"}
	}
	rs, _, err := d.members.IdentityRecords(id.User)
	if err != nil {
		return newKeyPreview{Error: err.Error()}
	}
	st := Resolve(id.User, rs)
	out := newKeyPreview{Version: newKeyVersion(rs, st)}
	seen := map[[32]byte]bool{}
	for _, r := range rs.Removals {
		h := r.Hash()
		if seen[h] || r.User != id.User {
			continue
		}
		seen[h] = true
		_, counts := st.Removals[h]
		name := ""
		if ms, ok := st.Machines[r.Machine]; ok {
			name = ms.Cert.Name
		}
		out.Removals = append(out.Removals, heldRemoval{
			Hash: hex.EncodeToString(h[:]), Machine: r.Machine, MachineName: name,
			SignedBy: signerText(r), Carried: r.SignerLevel == LevelShared, Counts: counts, Cutoffs: r.Cutoffs,
		})
	}
	sort.Slice(out.Removals, func(i, j int) bool { return out.Removals[i].Hash < out.Removals[j].Hash })
	for k, ms := range st.Machines {
		out.Machines = append(out.Machines, keyMachine{Machine: k, Name: ms.Cert.Name, Serial: ms.Cert.Serial,
			Self: k == id.PeerID, Removed: ms.Removed, Current: ms.Current})
	}
	sort.Slice(out.Machines, func(i, j int) bool {
		if out.Machines[i].Serial != out.Machines[j].Serial {
			return out.Machines[i].Serial < out.Machines[j].Serial
		}
		return out.Machines[i].Machine < out.Machines[j].Machine
	})
	return out
}

func (d *Daemon) handleNewKeyPreview(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, d.newKeyPreview())
}

type newKeyRequest struct {
	Phrase string `json:"phrase"`
	// Keep is the hashes of the removals signed by a shared key that the user made
	// and confirms, which are signed again under the new key.
	Keep []string `json:"keep"`
	// Machines is the keys of the user's machines to certify again under the new key.
	// This machine is certified whether or not it is named.
	Machines []string `json:"machines"`
}

type newKeyResult struct {
	Status    string `json:"status,omitempty"`
	Version   uint64 `json:"version,omitempty"`
	Machines  int    `json:"machines,omitempty"`
	Removals  int    `json:"removals,omitempty"`
	Suspended int    `json:"suspended,omitempty"`
	Error     string `json:"error,omitempty"`
}

// handleNewKey certifies a new shared key from the phrase typed into the page: the
// master key certifies it one version above any held, the new key signs again each
// removal the user confirmed and certifies again each machine the user kept, this
// machine takes the new key and its new chain, and the master key and the phrase are
// dropped before the response is written (D-220; D-223). Each other machine certified
// again is sent the new key over the pinned connection of the next exchange of the
// user's records (D-219, every machine the user adds holds the shared key), and a
// machine not certified again is suspended by the rule Resolve applies (D-234, a
// machine not certified under the current shared key is suspended).
func (d *Daemon) handleNewKey(w http.ResponseWriter, r *http.Request) {
	var req newKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	p, err := ParsePhrase(req.Phrase)
	req.Phrase = ""
	if err != nil {
		writeJSON(w, newKeyResult{Error: "The phrase does not read: " + err.Error() + "."})
		return
	}
	res, err := d.certifyNewKey(p, req.Keep, req.Machines)
	p = Phrase{}
	if err != nil {
		writeJSON(w, newKeyResult{Error: err.Error()})
		return
	}
	writeJSON(w, res)
}

// pendingRecord is a record made while a new key is certified, kept once all are made.
type pendingRecord struct {
	kind string
	body any
}

func (d *Daemon) certifyNewKey(p Phrase, keep, machines []string) (newKeyResult, error) {
	id := d.id
	if !id.HasUser() {
		return newKeyResult{}, errors.New("this machine speaks for no user; recover instead")
	}
	if p.UserID() != id.User {
		return newKeyResult{}, errors.New("that phrase is not the phrase of the user this machine speaks for")
	}
	rs, _, err := d.members.IdentityRecords(id.User)
	if err != nil {
		return newKeyResult{}, err
	}
	st := Resolve(id.User, rs)
	version := newKeyVersion(rs, st)

	master := p.MasterKey()
	sharedPub, shared, err := ed25519.GenerateKey(nil)
	if err != nil {
		zero(master)
		return newKeyResult{}, err
	}
	sharedCert := CertifySharedKey(master, version, sharedPub)
	zero(master)

	// The removals the user confirmed, signed again so that they count under the new
	// key; one the user did not confirm is left to count for nothing (D-220).
	kept := map[string]bool{}
	for _, h := range keep {
		kept[h] = true
	}
	records := []pendingRecord{{RecordShared, sharedCert}}
	carried, done := 0, map[[32]byte]bool{}
	for _, r := range rs.Removals {
		h := r.Hash()
		if done[h] || r.SignerLevel != LevelShared || !kept[hex.EncodeToString(h[:])] {
			continue
		}
		done[h] = true
		again, err := SignRemoval(shared, LevelShared, version, id.User, r.Machine, r.Cutoffs)
		if err != nil {
			return newKeyResult{}, err
		}
		records = append(records, pendingRecord{RecordRemoval, again})
		carried++
	}

	// The machines the user kept, under their own keys, names, kinds and serials. A
	// removed machine is certified again too when kept, since a suspended machine's
	// turns count for nothing and a removed one's count up to its cutoff.
	want := map[string]bool{id.PeerID: true}
	for _, m := range machines {
		want[m] = true
	}
	var own MachineCert
	certified, suspended := 0, 0
	for k, ms := range st.Machines {
		if !want[k] {
			suspended++
			continue
		}
		pub, err := PublicFromPeerID(k)
		if err != nil {
			continue
		}
		c := CertifyMachine(shared, id.User, version, pub, ms.Cert.Kind, ms.Cert.Name, ms.Cert.Serial)
		if k == id.PeerID {
			own = c
		}
		records = append(records, pendingRecord{RecordMachine, c})
		certified++
	}
	if own.Machine == "" {
		return newKeyResult{}, errors.New("this machine's own certificate is not among your records")
	}
	for _, rec := range records {
		raw, err := json.Marshal(rec.body)
		if err != nil {
			return newKeyResult{}, err
		}
		if _, _, err := d.members.PutIdentityRecord(rec.kind, id.User, raw); err != nil {
			return newKeyResult{}, err
		}
	}
	if err := d.becomeUser(id.User, "", Chain{Shared: sharedCert, Machine: own}, shared, version); err != nil {
		return newKeyResult{}, err
	}
	log.Printf("identity: certified shared key version %d, %d machine(s) certified again, %d removal(s) carried", version, certified, carried)
	d.notify()
	return newKeyResult{Status: "certified", Version: version, Machines: certified, Removals: carried, Suspended: suspended}, nil
}

// --- the new shared key reaching the user's other machines ---

// sharedKeyGift is the shared key one of the user's machines sends another over the
// pinned connection of an exchange of the user's records, so that every machine the
// user keeps holds the current key (D-219, every machine the user adds holds the
// shared key). It travels only between two machines of the user, each of which has
// proved its key on the connection and its chain in the request.
type sharedKeyGift struct {
	Version uint64 `json:"version"`
	Key     string `json:"key"`
}

// giftFor is the current shared key for machine, when this machine holds it and
// machine is a current, unremoved machine of the user that holds an older one.
func (d *Daemon) giftFor(machine string, theirs uint64) *sharedKeyGift {
	if !d.id.HasUser() {
		return nil
	}
	shared, version, ok, err := loadSharedKeyIn(d.homeDirectory())
	if err != nil || !ok || theirs >= version {
		return nil
	}
	st := d.members.UserState(d.id.User, nil)
	if version != st.Current || st.SharedKeys[version] != PeerIDFromPublic(shared.Public().(ed25519.PublicKey)) {
		return nil
	}
	ms, ok := st.Machines[machine]
	if !ok || !ms.Current || ms.Removed {
		return nil
	}
	return &sharedKeyGift{Version: version, Key: base64.RawURLEncoding.EncodeToString(shared)}
}

// heldSharedVersion is the version of the shared key this machine holds, 0 for none.
func (d *Daemon) heldSharedVersion() uint64 {
	_, v, ok, err := loadSharedKeyIn(d.homeDirectory())
	if err != nil || !ok {
		return 0
	}
	return v
}

// acceptGift keeps a shared key another of the user's machines sent, when it is the
// winning key of the user's current version, newer than the one held, and this machine
// is certified under it and not removed.
func (d *Daemon) acceptGift(g *sharedKeyGift, from string) {
	if g == nil || !d.id.HasUser() || g.Version <= d.heldSharedVersion() {
		return
	}
	st := d.members.UserState(d.id.User, nil)
	if g.Version != st.Current {
		return
	}
	raw, err := base64.RawURLEncoding.DecodeString(g.Key)
	if err != nil || len(raw) != ed25519.PrivateKeySize {
		return
	}
	key := ed25519.PrivateKey(raw)
	if PeerIDFromPublic(key.Public().(ed25519.PublicKey)) != st.SharedKeys[g.Version] {
		log.Printf("lists: %s sent a shared key that is not the user's key of version %d; it was not kept", PeerName(from), g.Version)
		return
	}
	if ms, ok := st.Machines[d.id.PeerID]; !ok || !ms.Current || ms.Removed {
		return
	}
	if err := saveSharedKeyIn(d.homeDirectory(), key, g.Version); err != nil {
		log.Printf("lists: could not keep the shared key of version %d: %v", g.Version, err)
		return
	}
	log.Printf("lists: this machine now holds the shared key of version %d, from %s", g.Version, PeerName(from))
}
