package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// --- how a user's identity records reach the machines that need them ---
//
// Two paths, for two questions. A receiver holding an event that names a shared-key
// version or a removal it lacks asks for the records by hash from any machine,
// since the master or the shared key signed them and the receiver checks them itself,
// and withholds the event from every assistant until it has them (D-221, an event
// whose identity version the receiver lacks is withheld from the assistant until
// fetched). And a machine holding a removal or a shared-key certificate of its own
// user sends it to every colleague it knows until one machine of each says it has it,
// and a machine that receives one copies it to its own user's other machines until
// each says so (D-225, a machine sends each removal and new shared key to every
// colleague it knows until each acknowledges; D-222, a machine tells every machine
// it knows of each machine it adds, until each has acknowledged).
//
// Both run on a loop of their own, never on a hook's path, so a session waits on
// neither (§3.1, first, do no harm).

// propagationSchema is membership.db's record of what is wanted and what is owed. It
// is kept there, beside the identity records, and not in memory, so that a restart
// resends and fetches again what it had not finished (the pattern "persist anything
// before publishing it").
const propagationSchema = `
-- A record an event named that this machine lacks. want is the hex hash of a removal,
-- or "shared:<version>" for a shared-key version above the one held. room_id, signer
-- and source say where the event was and who sent it, so the signer and the machine it
-- came from are asked first.
CREATE TABLE IF NOT EXISTS identity_wants (
  user     TEXT NOT NULL,
  want     TEXT NOT NULL,
  room_id  TEXT NOT NULL DEFAULT '',
  signer   TEXT NOT NULL DEFAULT '',
  source   TEXT NOT NULL DEFAULT '',
  attempts INTEGER NOT NULL DEFAULT 0,
  next_try TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (user, want)
);
-- A record owed to a target until the target acknowledges it. target is a colleague's
-- user identifier, for whom one machine acknowledging counts, or one of this user's
-- own machines, each of which must (D-225).
CREATE TABLE IF NOT EXISTS identity_push (
  hash        TEXT NOT NULL,
  kind        TEXT NOT NULL,
  target      TEXT NOT NULL,
  target_kind TEXT NOT NULL,
  attempts    INTEGER NOT NULL DEFAULT 0,
  next_try    TEXT NOT NULL DEFAULT '',
  acked_at    TEXT,
  PRIMARY KEY (hash, target)
);`

const (
	pushToUser    = "user"
	pushToMachine = "machine"

	// The signed request carries a room identifier, and these stand in its place for
	// the two routes that serve no room, so that a signature made for one route, or
	// for a room's sync, cannot be presented at another (D-044, sync requests are
	// signed). Neither can name a room, since a room's identifier is a hash.
	identityFetchScope = protocolNamespace + "/identity-fetch"
	identityPushScope  = protocolNamespace + "/identity-push"

	// maxFetchHashes bounds one fetch request.
	maxFetchHashes = 256
)

// pushAttemptLimit is how many failed attempts a record owed to a target gets before it is no
// longer sent: a week of attempts at the longest backoff. A machine that has been away longer
// learns of the record from a machine that is ahead of it when it returns (D-221, an event
// whose identity version the receiver lacks is withheld from the assistant until fetched),
// and a record owed to a machine that can never answer, such as a cloud session that ended,
// would otherwise be sent every ten minutes for good (D-268, a record is owed to no removed
// machine and for no more than a week of failed attempts).
const pushAttemptLimit = 7 * 24 * 6

// propagationBackoff is the wait after the nth failed attempt to fetch or to push,
// doubling from 5s to 10 minutes, so a colleague offline for a day costs a request
// every ten minutes and one back after an hour is reached within ten.
func propagationBackoff(attempts int) time.Duration {
	wait := 5 * time.Second
	for i := 1; i < attempts && wait < 10*time.Minute; i++ {
		wait *= 2
	}
	return min(wait, 10*time.Minute)
}

// --- the withhold rule (D-221) ---

// identityLacks reports whether an event signed for a user names a shared-key version
// above the one this state holds or a removal it does not hold. A removal named in a
// form that is no hash can never be fetched, and is lacking for good, which holds the
// event: a signer that names one has said nothing a receiver can check.
func identityLacks(st *UserState, ev Event) bool {
	if ev.SigVersion != 4 {
		return false
	}
	return st.Lacks(ev.IdentityVersion, removalHashes(ev.RemovalsSeen))
}

// removalHashes reads the hex hashes an event names. One that does not read as a hash
// becomes the zero hash, which no valid removal has, so it stays lacking.
func removalHashes(hexes []string) [][32]byte {
	out := make([][32]byte, 0, len(hexes))
	for _, h := range hexes {
		var a [32]byte
		if b, err := hex.DecodeString(h); err == nil && len(b) == 32 {
			copy(a[:], b)
		}
		out = append(out, a)
	}
	return out
}

// onlyIdentityKnown drops the events signed for a user whose records this machine
// lacks, and those whose machine does not count in what it holds, which a removal
// fetched after the event was stored can make so (D-221; D-220, a removal names the
// last event of a machine that counts). A dropped event stays stored, is shown in the
// view, and is marked there as not yet given to the assistant, since it was never
// delivered. It is given at the first prompt after the records arrive, with no limit
// on how long that takes. One resolution per signing machine, from membership.db and
// the event's own chain, which a machine's own events were never stored from, so a
// prompt waits on no network.
func (d *Daemon) onlyIdentityKnown(evs []Event, roomID string) []Event {
	states := map[string]*UserState{}
	out := evs[:0:0]
	for _, e := range evs {
		if e.SigVersion != 4 {
			out = append(out, e)
			continue
		}
		key := e.PeerID + " " + e.signerKey()
		st, ok := states[key]
		if !ok {
			st = d.members.UserState(e.PeerID, e.Chain)
			states[key] = st
		}
		if identityLacks(st, e) || !st.Counts(e.machineKey(), e.signerKey(), roomID, e.PeerSequence) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// wantsFor is what to fetch for an event the state lacks records for.
func wantsFor(st *UserState, ev Event) []string {
	var out []string
	if ev.IdentityVersion > st.Current {
		out = append(out, "shared:"+strconv.FormatUint(ev.IdentityVersion, 10))
	}
	for i, h := range removalHashes(ev.RemovalsSeen) {
		if _, ok := st.Removals[h]; !ok && h != ([32]byte{}) {
			out = append(out, strings.ToLower(ev.RemovalsSeen[i]))
		}
	}
	return out
}

// WantIdentity records the records an event names that this machine lacks, with
// where the event was and who sent it. A want already recorded is left as it is.
func (m *Membership) WantIdentity(user string, wants []string, roomID, signer, source string) error {
	for _, w := range wants {
		if _, err := m.db.Exec(`INSERT OR IGNORE INTO identity_wants (user, want, room_id, signer, source)
			VALUES (?,?,?,?,?)`, user, w, roomID, signer, source); err != nil {
			return err
		}
	}
	return nil
}

type identityWant struct {
	User, Want, RoomID, Signer, Source string
	Attempts                           int
}

// dueIdentityWants is every want whose next attempt is due, by user.
func (m *Membership) dueIdentityWants(now time.Time) (map[string][]identityWant, error) {
	rows, err := m.db.Query(`SELECT user, want, room_id, signer, source, attempts FROM identity_wants
		WHERE next_try = '' OR next_try <= ? ORDER BY user, want`, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]identityWant{}
	for rows.Next() {
		var w identityWant
		if err := rows.Scan(&w.User, &w.Want, &w.RoomID, &w.Signer, &w.Source, &w.Attempts); err != nil {
			return nil, err
		}
		out[w.User] = append(out[w.User], w)
	}
	return out, rows.Err()
}

func (m *Membership) dropIdentityWant(user, want string) error {
	_, err := m.db.Exec(`DELETE FROM identity_wants WHERE user = ? AND want = ?`, user, want)
	return err
}

func (m *Membership) identityWantFailed(w identityWant, now time.Time) error {
	_, err := m.db.Exec(`UPDATE identity_wants SET attempts = ?, next_try = ? WHERE user = ? AND want = ?`,
		w.Attempts+1, now.Add(propagationBackoff(w.Attempts+1)).UTC().Format(time.RFC3339Nano), w.User, w.Want)
	return err
}

// noteLacking records what a just-stored event names that this machine lacks.
func (d *Daemon) noteLacking(ev Event, source string) {
	if ev.SigVersion != 4 {
		return
	}
	st := d.members.UserState(ev.PeerID, nil)
	if !identityLacks(st, ev) {
		return
	}
	if err := d.members.WantIdentity(ev.PeerID, wantsFor(st, ev), ev.RoomID, ev.signerKey(), source); err != nil {
		log.Printf("identity: could not record what %.12s names and this machine lacks: %v", ev.EventID, err)
	}
}

// scanWithheld records a want for every stored event whose records this machine
// lacks, so that an event stored before this build, or a want lost with membership.db,
// is fetched all the same. It runs once when the loop starts.
func (d *Daemon) scanWithheld() {
	rooms, err := d.members.Rooms()
	if err != nil {
		return
	}
	for _, room := range rooms {
		store, err := d.storeFor(room.RoomID)
		if err != nil {
			continue
		}
		evs, err := store.ListRoom(room.RoomID)
		if err != nil {
			continue
		}
		for _, e := range evs {
			d.noteLacking(e, "")
		}
	}
}

// RunIdentity fetches the records events name and pushes the records owed to
// colleagues and to this user's other machines, on a loop of its own.
func (d *Daemon) RunIdentity(every time.Duration) {
	d.scanWithheld()
	for {
		now := time.Now()
		d.fetchIdentityWants(now)
		d.pushIdentityRecords(now)
		time.Sleep(every)
	}
}

// --- the fetch route ---

type identityFetchRequest struct {
	Protocol int `json:"protocol"`
	// Hashes names records by the hex of their hash, of any kind and any user.
	Hashes []string `json:"hashes,omitempty"`
	// SharedOf and SharedAbove ask for every shared-key certificate of a user above a
	// version, which an event names by number and not by hash.
	SharedOf    string `json:"sharedOf,omitempty"`
	SharedAbove uint64 `json:"sharedAbove,omitempty"`
	// Credentials, as a sync request carries them, with identityFetchScope as the room.
	RoomID    string `json:"roomId"`
	PeerID    string `json:"peerId"`
	Timestamp string `json:"timestamp"`
	Nonce     string `json:"nonce"`
	Signature string `json:"signature"`
	Endpoint  string `json:"endpoint,omitempty"`
	Signer    string `json:"signer,omitempty"`
	Chain     *Chain `json:"chain,omitempty"`
}

// identityRecordWire is one record as it travels: its kind and its JSON (D-238,
// identity records are signed over length-prefixed fields and carried as JSON).
type identityRecordWire struct {
	Kind   string          `json:"kind"`
	Record json.RawMessage `json:"record"`
}

type identityFetchResponse struct {
	Protocol int                  `json:"protocol"`
	Records  []identityRecordWire `json:"records"`
}

// mayExchangeIdentity is who may fetch records from this machine or push them to it:
// this machine's own user, from another of the user's machines, and any user or peer
// this machine knows. A stranger is refused, so that a machine is not a directory of
// whom its user knows for anyone who reaches it.
func (d *Daemon) mayExchangeIdentity(peer string) bool {
	return d.isSelf(peer) || d.members.Knows(peer)
}

// verifyIdentityRequest establishes who signed a request to one of the identity
// routes, and that this machine exchanges records with them.
func (d *Daemon) verifyIdentityRequest(scope string, cred syncRequest, r *http.Request) error {
	if cred.RoomID != scope {
		return errors.New("the request is not signed for this route")
	}
	if err := connectedAs(r, cred.requestMachine()); err != nil {
		return err
	}
	// No room: verifyRequest skips admission and verification for an empty room,
	// and the scope stands in the signed room field.
	if err := d.verifyRequest(cred, ""); err != nil {
		return err
	}
	if !d.mayExchangeIdentity(cred.PeerID) {
		return fmt.Errorf("%s is not a peer this machine knows", PeerName(cred.PeerID))
	}
	return nil
}

// handleIdentityFetch answers with the records this machine holds of those asked for,
// whichever user they are of, as /blob does for an image (D-213, a pasted image is
// sent to peers). A record proves itself by its signatures, so serving one discloses
// nothing a chain in an event does not.
func (d *Daemon) handleIdentityFetch(w http.ResponseWriter, r *http.Request) {
	var req identityFetchRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !speaks(req.Protocol) {
		http.Error(w, "unsupported protocol", http.StatusBadRequest)
		return
	}
	cred := syncRequest{PeerID: req.PeerID, RoomID: req.RoomID, Timestamp: req.Timestamp, Nonce: req.Nonce,
		Signature: req.Signature, Endpoint: req.Endpoint, Signer: req.Signer, Chain: req.Chain}
	if err := d.verifyIdentityRequest(identityFetchScope, cred, r); err != nil {
		log.Printf("identity: refused a fetch: %v", err)
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	if len(req.Hashes) > maxFetchHashes {
		http.Error(w, "too many records in one request", http.StatusBadRequest)
		return
	}
	out := identityFetchResponse{Protocol: wireVersion, Records: []identityRecordWire{}}
	for _, h := range req.Hashes {
		var hash [32]byte
		b, err := hex.DecodeString(h)
		if err != nil || len(b) != 32 {
			continue
		}
		copy(hash[:], b)
		kind, body, ok, err := d.members.IdentityRecordByHash(hash)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ok {
			out.Records = append(out.Records, identityRecordWire{Kind: kind, Record: body})
		}
	}
	if req.SharedOf != "" {
		rs, _, err := d.members.IdentityRecords(req.SharedOf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, c := range rs.Shared {
			if c.Version > req.SharedAbove {
				raw, _ := json.Marshal(c)
				out.Records = append(out.Records, identityRecordWire{Kind: RecordShared, Record: raw})
			}
		}
	}
	writeJSON(w, out)
}

// --- fetching ---

// identitySources is where to ask for a user's records, in order: the machine that
// signed each event that named them, the machine it arrived from, the admitted
// machines of each room it was in, the user's own machines, then this user's own
// other machines, which hold what this user's machines copy between them (D-221,
// asking only the sender lets a thief withhold the records; D-222).
func (d *Daemon) identitySources(user string, wants []identityWant) []string {
	seen := map[string]bool{}
	var out []string
	add := func(a string) {
		if a != "" && !seen[a] {
			seen[a] = true
			out = append(out, a)
		}
	}
	for _, w := range wants {
		if e, _ := d.members.MachineEndpoint(w.Signer); w.Signer != d.id.PeerID {
			add(e)
		}
		add(w.Source)
	}
	for _, w := range wants {
		if w.RoomID != "" {
			for _, a := range d.members.RoomPeers(w.RoomID, d.id.PeerID) {
				add(a)
			}
		}
	}
	for _, u := range []string{user, d.id.User} {
		if u == "" {
			continue
		}
		ms, _ := d.members.UserMachines(u)
		for _, m := range ms {
			if m.MachineKey != d.id.PeerID {
				add(m.Endpoint)
			}
		}
	}
	return out
}

// fetchIdentityWants makes one pass over the wants that are due, and returns how many
// it satisfied. A want is satisfied when its removal is held, or a shared-key version
// at least the one named is current; a removal held that names a newer shared key than
// the one current wants that key in turn.
func (d *Daemon) fetchIdentityWants(now time.Time) int {
	due, err := d.members.dueIdentityWants(now)
	if err != nil {
		return 0
	}
	satisfied := 0
	for user, wants := range due {
		remaining := wants
		for _, addr := range d.identitySources(user, wants) {
			records, err := d.requestIdentity(addr, user, remaining)
			if err != nil {
				continue
			}
			d.keepFetched(user, remaining, records)
			remaining = d.settleWants(user, remaining)
			if len(remaining) == 0 {
				break
			}
		}
		satisfied += len(wants) - len(remaining)
		for _, w := range remaining {
			if w.Attempts%5 == 0 {
				log.Printf("identity: could not fetch %s of %s (attempt %d)", w.Want, PeerName(user), w.Attempts+1)
			}
			_ = d.members.identityWantFailed(w, now)
		}
	}
	if satisfied > 0 {
		log.Printf("identity: fetched %d record(s) that events named", satisfied)
		d.notify()
	}
	return satisfied
}

// keepFetched stores the records a machine answered with that were asked for: a
// removal whose hash was named, or a shared-key certificate of the user, since a
// machine could otherwise fill this one's store with anything. PutIdentityRecord
// computes each hash itself, so a record sent under another's hash is not that one.
func (d *Daemon) keepFetched(user string, wants []identityWant, records []identityRecordWire) {
	named := map[string]bool{}
	for _, w := range wants {
		named[w.Want] = true
	}
	for _, rec := range records {
		switch rec.Kind {
		case RecordRemoval:
			_, hash, err := identityRecordHash(rec.Kind, rec.Record)
			if err != nil || !named[hex.EncodeToString(hash[:])] {
				continue
			}
		case RecordShared:
		default:
			continue
		}
		if _, _, err := d.members.PutIdentityRecord(rec.Kind, user, rec.Record); err != nil {
			log.Printf("identity: a fetched %s record of %s was not kept: %v", rec.Kind, PeerName(user), err)
		}
	}
}

// settleWants drops the wants the records now held satisfy and returns the rest.
func (d *Daemon) settleWants(user string, wants []identityWant) []identityWant {
	st := d.members.UserState(user, nil)
	var rest []identityWant
	for _, w := range wants {
		done := false
		if v, ok := strings.CutPrefix(w.Want, "shared:"); ok {
			n, err := strconv.ParseUint(v, 10, 64)
			done = err == nil && st.Current >= n
		} else if b, err := hex.DecodeString(w.Want); err == nil && len(b) == 32 {
			var h [32]byte
			copy(h[:], b)
			_, body, held, _ := d.members.IdentityRecordByHash(h)
			done = held
			// A removal signed by a shared key this machine does not hold yet is held
			// and not valid, and the key it names is wanted next.
			var r Removal
			if held && json.Unmarshal(body, &r) == nil && r.SignerLevel == LevelShared && r.SignerVersion > st.Current {
				_ = d.members.WantIdentity(user, []string{"shared:" + strconv.FormatUint(r.SignerVersion, 10)},
					w.RoomID, w.Signer, w.Source)
			}
		}
		if done {
			_ = d.members.dropIdentityWant(user, w.Want)
			continue
		}
		rest = append(rest, w)
	}
	return rest
}

// requestIdentity asks one machine for the records of user that wants name.
func (d *Daemon) requestIdentity(addr, user string, wants []identityWant) ([]identityRecordWire, error) {
	req := identityFetchRequest{Protocol: wireVersion}
	for _, w := range wants {
		if strings.HasPrefix(w.Want, "shared:") {
			req.SharedOf = user
			req.SharedAbove = d.members.UserState(user, nil).Current
			continue
		}
		req.Hashes = append(req.Hashes, w.Want)
	}
	var out identityFetchResponse
	if err := d.postSigned(addr, "/identity/fetch", identityFetchScope, &req, func(c requestCreds) {
		req.RoomID, req.PeerID, req.Timestamp, req.Nonce, req.Signature = identityFetchScope, c.PeerID, c.Timestamp, c.Nonce, c.Signature
		req.Endpoint, req.Signer, req.Chain = c.Endpoint, c.Signer, c.Chain
	}, &out); err != nil {
		return nil, err
	}
	return out.Records, nil
}

// postSigned signs a request for scope, posts it to path at addr and decodes the
// answer. fill writes the credentials into the request before it is encoded.
func (d *Daemon) postSigned(addr, path, scope string, body any, fill func(requestCreds), out any) error {
	conf, err := d.clientConfig("")
	if err != nil {
		return err
	}
	client, err := clientFor(addr, 20*time.Second, conf)
	if err != nil {
		return err
	}
	cred, err := signRequestAs(d.id, scope, AdvertisedEndpoint())
	if err != nil {
		return err
	}
	fill(cred)
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := client.Post(peerURL(path), "application/json", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %s", shortEndpoint(addr), resp.Status)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}

// --- the push route ---

type identityPushRequest struct {
	Protocol int             `json:"protocol"`
	Kind     string          `json:"kind"`
	Record   json.RawMessage `json:"record"`
	// Credentials, as a sync request carries them, with identityPushScope as the room.
	RoomID    string `json:"roomId"`
	PeerID    string `json:"peerId"`
	Timestamp string `json:"timestamp"`
	Nonce     string `json:"nonce"`
	Signature string `json:"signature"`
	Endpoint  string `json:"endpoint,omitempty"`
	Signer    string `json:"signer,omitempty"`
	Chain     *Chain `json:"chain,omitempty"`
}

type identityPushResponse struct {
	Protocol int `json:"protocol"`
	// Ack is the hex hash of the record, which this machine now holds. The sender
	// counts the record delivered only on it (the pattern "treat something as
	// delivered only once the receiver has said it has it").
	Ack string `json:"ack"`
}

// pushedKinds are the records a machine pushes: a removal and a shared-key
// certificate, which change whose events count (D-225).
func pushedKind(kind string) bool { return kind == RecordRemoval || kind == RecordShared }

// maxPushedRecords is how many pushed records, removals and shared-key certificates
// together, this machine keeps for one user. A user makes one of either only when a
// machine is lost or removed, so a count near this is a machine pushing records to
// fill this one's disk.
const maxPushedRecords = 1024

// pushedRecordSigned refuses a pushed record whose signature does not verify against
// the key it names: the master key for a shared-key certificate, and the signer for a
// removal. Whether that key speaks for the user is Resolve's question, and needs the
// user's other records; a record no key signed is refused before it is kept.
func pushedRecordSigned(kind string, body []byte) error {
	switch kind {
	case RecordShared:
		var c SharedKeyCert
		if err := json.Unmarshal(body, &c); err != nil {
			return err
		}
		return c.Verify()
	case RecordRemoval:
		var r Removal
		if err := json.Unmarshal(body, &r); err != nil {
			return err
		}
		pub, err := PublicFromPeerID(r.Signer)
		if err != nil {
			return err
		}
		return r.VerifyBy(pub)
	}
	return fmt.Errorf("%q is not a record that is pushed", kind)
}

// pushedRecordsFull reports whether this machine keeps as many pushed records of user
// as maxPushedRecords allows and does not already hold the one hash names.
func (m *Membership) pushedRecordsFull(user string, hash [32]byte) bool {
	var held, n int
	if err := m.db.QueryRow(`SELECT COUNT(*) FROM identity_records WHERE hash = ?`, hex.EncodeToString(hash[:])).Scan(&held); err != nil || held > 0 {
		return false
	}
	if err := m.db.QueryRow(`SELECT COUNT(*) FROM identity_records WHERE user = ? AND kind IN (?, ?)`,
		user, RecordRemoval, RecordShared).Scan(&n); err != nil {
		return true
	}
	return n >= maxPushedRecords
}

// handleIdentityPush stores a removal or a shared-key certificate a colleague or
// another of this user's machines sent, acknowledges it by its hash, and, when it came
// from a colleague and is new here, owes it to each of this user's other machines
// (D-225, one machine of the colleague acknowledging counts, and it copies the record
// to the colleague's other machines). A record about a user this machine does not know
// is refused, since it has nobody's events to judge by it.
func (d *Daemon) handleIdentityPush(w http.ResponseWriter, r *http.Request) {
	var req identityPushRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !speaks(req.Protocol) {
		http.Error(w, "unsupported protocol", http.StatusBadRequest)
		return
	}
	cred := syncRequest{PeerID: req.PeerID, RoomID: req.RoomID, Timestamp: req.Timestamp, Nonce: req.Nonce,
		Signature: req.Signature, Endpoint: req.Endpoint, Signer: req.Signer, Chain: req.Chain}
	if err := d.verifyIdentityRequest(identityPushScope, cred, r); err != nil {
		log.Printf("identity: refused a push: %v", err)
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	if !pushedKind(req.Kind) {
		http.Error(w, "not a record that is pushed", http.StatusBadRequest)
		return
	}
	user, hash, err := identityRecordHash(req.Kind, req.Record)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !d.mayExchangeIdentity(user) {
		http.Error(w, "a record of a user this machine does not know", http.StatusForbidden)
		return
	}
	if err := pushedRecordSigned(req.Kind, req.Record); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if d.members.pushedRecordsFull(user, hash) {
		http.Error(w, "this machine holds as many pushed records of that user as it keeps", http.StatusInsufficientStorage)
		return
	}
	_, isNew, err := d.members.PutIdentityRecord(req.Kind, user, req.Record)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isNew && !d.isSelf(req.PeerID) {
		if err := d.members.oweToOwnMachines(hex.EncodeToString(hash[:]), req.Kind, d.id, cred.requestMachine()); err != nil {
			log.Printf("identity: could not record that the user's other machines are owed %s: %v", req.Kind, err)
		}
	}
	if isNew {
		log.Printf("identity: %s of %s pushed by %s", req.Kind, PeerName(user), PeerName(req.PeerID))
		d.notify()
	}
	writeJSON(w, identityPushResponse{Protocol: wireVersion, Ack: hex.EncodeToString(hash[:])})
}

// --- pushing ---

// notRemoved is an SQL condition, with its arguments, that leaves out each machine of user
// that a removal removes, since a removed machine is told nothing (D-252, a reachable machine
// is asked to erase its keys before it is removed). It is empty when none is removed.
func (m *Membership) notRemoved(user, column string) (string, []any) {
	st := m.UserState(user, nil)
	var args []any
	for key, ms := range st.Machines {
		if ms.Removed {
			args = append(args, key)
		}
	}
	if len(args) == 0 {
		return "", nil
	}
	return " AND " + column + " NOT IN (?" + strings.Repeat(",?", len(args)-1) + ")", args
}

// oweToOwnMachines records that each of this user's other machines is owed a record,
// apart from the machine it came from.
func (m *Membership) oweToOwnMachines(hash, kind string, id *Identity, from string) error {
	if !id.HasUser() {
		return nil
	}
	cond, extra := m.notRemoved(id.User, "machine_key")
	_, err := m.db.Exec(`INSERT OR IGNORE INTO identity_push (hash, kind, target, target_kind)
		SELECT ?, ?, machine_key, ? FROM machines WHERE user = ? AND machine_key <> ? AND machine_key <> ?`+cond,
		append([]any{hash, kind, pushToMachine, id.User, id.PeerID, from}, extra...)...)
	return err
}

// owePushes records, for every removal and shared-key certificate of this machine's
// own user, that every colleague who speaks for a user, and each of this user's other
// machines, is owed it. It is run before each pass, so a record from any path, a
// colleague paired later and a machine added later are all owed what they lack, and
// a row once acknowledged stays so and is not sent again.
func (m *Membership) owePushes(id *Identity) error {
	if !id.HasUser() {
		return nil
	}
	if _, err := m.db.Exec(`INSERT OR IGNORE INTO identity_push (hash, kind, target, target_kind)
		SELECT r.hash, r.kind, k.peer_id, ? FROM identity_records r, known_peers k
		WHERE r.user = ? AND r.kind IN (?, ?) AND k.peer_id <> ? AND k.peer_id <> ?
		  AND k.peer_id IN (SELECT user FROM identity_records)`,
		pushToUser, id.User, RecordRemoval, RecordShared, id.User, id.PeerID); err != nil {
		return err
	}
	cond, extra := m.notRemoved(id.User, "mc.machine_key")
	if _, err := m.db.Exec(`INSERT OR IGNORE INTO identity_push (hash, kind, target, target_kind)
		SELECT r.hash, r.kind, mc.machine_key, ? FROM identity_records r, machines mc
		WHERE r.user = ? AND r.kind IN (?, ?) AND mc.user = ? AND mc.machine_key <> ?`+cond,
		append([]any{pushToMachine, id.User, RecordRemoval, RecordShared, id.User, id.PeerID}, extra...)...); err != nil {
		return err
	}
	// What was owed to a machine before a removal removed it is owed no more.
	for _, key := range extra {
		if _, err := m.db.Exec(`DELETE FROM identity_push WHERE target = ? AND target_kind = ? AND acked_at IS NULL`, key, pushToMachine); err != nil {
			return err
		}
	}
	return nil
}

type owedPush struct {
	Hash, Kind, Target, TargetKind string
	Attempts                       int
}

func (m *Membership) duePushes(now time.Time) ([]owedPush, error) {
	rows, err := m.db.Query(`SELECT hash, kind, target, target_kind, attempts FROM identity_push
		WHERE acked_at IS NULL AND attempts < ? AND (next_try = '' OR next_try <= ?) ORDER BY target, hash`,
		pushAttemptLimit, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []owedPush
	for rows.Next() {
		var p owedPush
		if err := rows.Scan(&p.Hash, &p.Kind, &p.Target, &p.TargetKind, &p.Attempts); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (m *Membership) pushAcked(p owedPush, now time.Time) error {
	_, err := m.db.Exec(`UPDATE identity_push SET acked_at = ? WHERE hash = ? AND target = ?`,
		now.UTC().Format(time.RFC3339), p.Hash, p.Target)
	return err
}

func (m *Membership) pushFailed(p owedPush, now time.Time) error {
	_, err := m.db.Exec(`UPDATE identity_push SET attempts = ?, next_try = ? WHERE hash = ? AND target = ?`,
		p.Attempts+1, now.Add(propagationBackoff(p.Attempts+1)).UTC().Format(time.RFC3339Nano), p.Hash, p.Target)
	return err
}

// pushTargets is where a target is reached: every machine of a colleague this machine
// knows an address for, any one of which acknowledging counts, or the one machine.
func (d *Daemon) pushTargets(p owedPush) []string {
	if p.TargetKind == pushToMachine {
		e, _ := d.members.MachineEndpoint(p.Target)
		if e == "" {
			return nil
		}
		return []string{e}
	}
	ms, _ := d.members.UserMachines(p.Target)
	var out []string
	for _, m := range ms {
		if m.Endpoint != "" {
			out = append(out, m.Endpoint)
		}
	}
	return out
}

// pushIdentityRecords makes one pass over the records owed, and returns how many
// were acknowledged. The rows are written before anything is sent, so a restart
// resends what was not acknowledged.
func (d *Daemon) pushIdentityRecords(now time.Time) int {
	if err := d.members.owePushes(d.id); err != nil {
		log.Printf("identity: could not record what is owed: %v", err)
	}
	due, err := d.members.duePushes(now)
	if err != nil {
		return 0
	}
	acked := 0
	for _, p := range due {
		if d.pushOne(p) {
			acked++
			_ = d.members.pushAcked(p, now)
			continue
		}
		if p.Attempts%5 == 0 {
			log.Printf("identity: %s %.12s has not reached %s yet (attempt %d)", p.Kind, p.Hash, PeerName(p.Target), p.Attempts+1)
		}
		_ = d.members.pushFailed(p, now)
	}
	return acked
}

// pushOne sends one record to its target's machines in turn until one acknowledges it.
func (d *Daemon) pushOne(p owedPush) bool {
	var h [32]byte
	b, err := hex.DecodeString(p.Hash)
	if err != nil || len(b) != 32 {
		return false
	}
	copy(h[:], b)
	kind, body, ok, err := d.members.IdentityRecordByHash(h)
	if err != nil || !ok {
		return false
	}
	for _, addr := range d.pushTargets(p) {
		req := identityPushRequest{Protocol: wireVersion, Kind: kind, Record: body}
		var out identityPushResponse
		err := d.postSigned(addr, "/identity/push", identityPushScope, &req, func(c requestCreds) {
			req.RoomID, req.PeerID, req.Timestamp, req.Nonce, req.Signature = identityPushScope, c.PeerID, c.Timestamp, c.Nonce, c.Signature
			req.Endpoint, req.Signer, req.Chain = c.Endpoint, c.Signer, c.Chain
		}, &out)
		if err == nil && out.Ack == p.Hash {
			return true
		}
	}
	return false
}
