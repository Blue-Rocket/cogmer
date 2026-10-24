package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

// Anti-entropy synchronization (§10): a peer asks each other peer for the events it
// lacks, and receives them.
//
// Polling is the decided default (D-031). Peer
// propagation is half a second; arrival into a colleague's session waits for their
// next prompt, which is minutes during a long turn. Push would improve the fast
// half of a path whose slow half is the turn. Pull also makes reconnection free —
// an absent peer recovers by asking, so nobody tracks what it missed.

type syncRequest struct {
	Protocol int `json:"protocol"`
	// RoomID, never the room name. A name is for people and collides by design
	// (D-017): two rooms on one daemon may share one, and a wire field that
	// resolves ambiguously resolves to whichever row came back first. The guest
	// receives the roomId on admission, so it always has this to send.
	RoomID string           `json:"roomId"`
	Have   map[string]int64 `json:"have"`
	// Credentials proving the caller holds the key its identifier names.
	PeerID    string `json:"peerId"`
	Timestamp string `json:"timestamp"`
	Nonce     string `json:"nonce"`
	Signature string `json:"signature"`
	// Endpoint is where the caller listens. Synchronization is a pull, so a peer
	// that never says where it is can be read from and never read.
	Endpoint string `json:"endpoint,omitempty"`
	// Signer and Chain are present under the v5 scheme, from a machine that speaks
	// for the user PeerID names: the machine's key, which signed, and the chain from
	// it to the user, which is not signed and proves itself (D-044, sync requests are signed; D-219, a user is a master key that certifies machines through a shared key).
	Signer string `json:"signer,omitempty"`
	Chain  *Chain `json:"chain,omitempty"`
}

type syncResponse struct {
	Protocol int         `json:"protocol"`
	RoomID   string      `json:"roomId"`
	Events   []wireEvent `json:"events"`
}

// handleSync answers with what the caller is missing. It is a pull: a peer asks,
// rather than being told, so a peer that was absent recovers by asking again
// rather than by anyone having to remember what it missed.
func (d *Daemon) handleSync(w http.ResponseWriter, r *http.Request) {
	var req syncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// A request names the room it wants. A daemon serving several cannot infer one,
	// and inferring one was how a single-room daemon avoided asking. It is looked
	// up by id alone: FindRoom also accepts a name, and accepting one here would
	// let a caller address a room by an identifier that is not unique.
	if !speaks(req.Protocol) {
		http.Error(w, fmt.Sprintf("this daemon reads protocol v%d to v%d, not v%d",
			minWireVersion, wireVersion, req.Protocol), http.StatusBadRequest)
		return
	}
	room, ok := d.members.RoomByID(req.RoomID)
	if !ok {
		// Do not distinguish "no such room" from "not a guest": both answers are
		// the same to anyone entitled to neither, and one of them is a disclosure.
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	// A machine whose user was expelled from the room serves it to nobody, since the
	// room's other guests hold its history (D-241, an expelled user's machines stop
	// serving the room).
	if d.expelledFrom(room.RoomID) {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	if err := d.verifyRequest(req, room.RoomID); err != nil {
		log.Printf("sync: refused a request for %s — %v", room.RoomName, err)
		var notGuest *notGuestError
		if errors.As(err, &notGuest) {
			d.refuseNonGuest(w, room, req.PeerID)
			return
		}
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	// A guest that tells us where it listens becomes reachable, which is what
	// makes the exchange mutual rather than one peer reading another.
	//
	// Recorded against the peer, not the room. verifyRequest has just established
	// req.PeerID cryptographically and the signature covers the endpoint, so this
	// is an authenticated claim by the only party entitled to make it — and it is
	// the sole way a peer that moved teaches anybody where it went (D-103, an address belongs to a machine).
	//
	// The address is the machine's that signed, stored against that machine and
	// its user, since a user with several machines has several (D-103, an address belongs to a machine).
	if req.Endpoint != "" {
		if err := d.members.RecordMachineAddress(req.requestMachine(), req.PeerID, req.Endpoint); err != nil {
			log.Printf("sync: could not record where %s listens: %v", PeerName(req.requestMachine()), err)
		}
	}
	d.markPeerHeardFrom(req.PeerID)
	// A machine that asked for the room and was admitted has joined it, so it is a
	// sync target from now on, and a machine of a guest that never asked is not
	// (D-103, a room's sync targets are the machines from which a session has joined
	// it).
	if err := d.members.RecordRoomMachine(room.RoomID, req.requestMachine(), req.PeerID); err != nil {
		log.Printf("sync: could not record that %s joined %s: %v", PeerName(req.requestMachine()), room.RoomName, err)
	}

	store, err := d.storeFor(room.RoomID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	evs, err := store.EventsSince(room.RoomID, req.Have)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]wireEvent, 0, len(evs))
	for _, e := range evs {
		out = append(out, toWire(e))
	}
	writeJSON(w, syncResponse{Protocol: wireVersion, RoomID: room.RoomID, Events: out})
}

func peerList() []string {
	raw := os.Getenv("COGMER_PEERS")
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// syncBackoffCap is the longest a peer that does not answer is left between two polls.
const syncBackoffCap = 15 * time.Second

// RunSync polls each address worth asking, every one in a loop of its own. A failed poll
// is not an error condition: peers are expected to be absent, and the next round recovers
// whatever was missed (§26).
//
// The loops are separate because a peer that does not answer holds its poll for as long as
// the request's timeout, and one loop for every peer made that wait the interval between
// polls of all of them: with one peer unreachable for a day, a colleague who was reachable
// received each turn half a minute after it was written. A peer that does not answer is
// also polled less often, up to syncBackoffCap, and at the usual interval again once it
// answers.
func (d *Daemon) RunSync(_ []string, every time.Duration) {
	stops := map[string]chan struct{}{}
	for {
		want := map[string]bool{}
		for _, addr := range d.syncTargets() { // re-read: a room joined since last round has peers
			want[addr] = true
			if _, running := stops[addr]; !running {
				stop := make(chan struct{})
				stops[addr] = stop
				go d.syncPeer(addr, every, stop)
			}
		}
		for addr, stop := range stops {
			if !want[addr] {
				close(stop)
				delete(stops, addr)
			}
		}
		time.Sleep(every)
	}
}

// syncPeer polls one address until it is told to stop.
func (d *Daemon) syncPeer(addr string, every time.Duration, stop <-chan struct{}) {
	pull := d.pull
	if pull == nil {
		pull = d.pullFrom
	}
	failures := 0
	for {
		select {
		case <-stop:
			return
		default:
		}
		n, err := pull(addr)
		delay := every
		if err != nil {
			d.markPeerUnreachable(addr, err)
			failures++
			for i := 1; i < failures && delay < syncBackoffCap; i++ {
				delay *= 2
			}
			if delay > syncBackoffCap {
				delay = syncBackoffCap
			}
		} else {
			failures = 0
			if n > 0 {
				log.Printf("sync: received %d event(s) from %s", n, shortEndpoint(addr))
			}
		}
		select {
		case <-stop:
			return
		case <-time.After(delay):
		}
	}
}

func (d *Daemon) pullFrom(addr string) (int, error) {
	// Generous, because first contact with a peer on another network includes
	// building a tunnel: probing the network, choosing a relay and handshaking.
	// Later rounds ride the cached client and return in milliseconds, so this
	// bounds the first attempt rather than the ordinary one.
	// A room records where its members listen, not which member listens where, so
	// there is no single key to expect here. Any known peer is accepted, and the
	// signed request inside binds identity exactly (D-044, sync requests are signed).
	conf, cerr := d.clientConfig("")
	if cerr != nil {
		return 0, cerr
	}
	client, err := clientFor(addr, 30*time.Second, conf)
	if err != nil {
		return 0, err
	}
	rooms, err := d.members.Rooms()
	if err != nil {
		return 0, err
	}
	// Every room is attempted, even after one fails. Returning on the first error
	// starves every room after it for as long as the failure lasts, and the
	// failures that last are exactly the ones worth surviving: a corrupt store, a
	// peer on another protocol version, a room whose host is down while others are
	// up. One bad room must not make a peer silent.
	total := 0
	var failed []string
	ask := d.roomsToAsk
	if d.askRooms != nil {
		ask = d.askRooms
	}
	for _, r := range ask(addr, rooms) {
		if d.expelledFrom(r.RoomID) {
			continue // D-241: an expelled user's machine asks nobody for the room
		}
		n, err := d.pullRoom(client, addr, r)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", r.RoomName, err))
			continue
		}
		total += n
	}
	if len(failed) > 0 {
		return total, errors.New(strings.Join(failed, "; "))
	}
	return total, nil
}

// syncTargets is every address worth asking: those configured for this machine,
// and the guests of each room, at the address recorded for each (D-103, an address belongs to a machine). A room
// carries its own peers because reachability is a property of a room's membership,
// not of the daemon.
//
// A peer that is known and is in no room is not here. Asking its address sends it a
// signed request naming a room it is not a guest of, which tells a stranger the room's
// identifier and this machine's address every second for as long as the record lasts,
// and a pairing that was never finished leaves a record. Verification and an
// invitation reach the peer by their own routes, which do not need it polled.
func (d *Daemon) syncTargets() []string {
	seen := map[string]bool{}
	var out []string
	add := func(a string) {
		if a != "" && !seen[a] {
			seen[a] = true
			out = append(out, a)
		}
	}
	for _, a := range peerList() {
		add(a)
	}
	if rooms, err := d.members.Rooms(); err == nil {
		for _, r := range rooms {
			// A room this machine's user was expelled from is dropped from the
			// targets (D-241, an expelled user's machines stop asking for the room).
			if d.expelledFrom(r.RoomID) {
				continue
			}
			for _, a := range d.members.RoomPeers(r.RoomID, d.id.PeerID) {
				add(a)
			}
		}
	}
	return out
}

// roomTargets is the addresses this daemon tries for one room: those configured for the
// machine, which name no peer and are asked about every room, and the addresses of the
// room's guests.
func (d *Daemon) roomTargets(roomID string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(a string) {
		if a != "" && !seen[a] {
			seen[a] = true
			out = append(out, a)
		}
	}
	if d.expelledFrom(roomID) {
		return nil
	}
	for _, a := range peerList() {
		add(a)
	}
	for _, a := range d.members.RoomPeers(roomID, d.id.PeerID) {
		add(a)
	}
	return out
}

// roomsToAsk is the rooms to pull from an address: those it is a guest's address in,
// and every room for an address configured for this machine, which names no peer. A room
// is never pulled from an address that is not one of its guests', so a guest of one
// room is not told about another.
func (d *Daemon) roomsToAsk(addr string, rooms []Room) []Room {
	for _, a := range peerList() {
		if a == addr {
			return rooms
		}
	}
	var out []Room
	for _, r := range rooms {
		if d.expelledFrom(r.RoomID) {
			continue
		}
		for _, a := range d.members.RoomPeers(r.RoomID, d.id.PeerID) {
			if a == addr {
				out = append(out, r)
				break
			}
		}
	}
	return out
}

func (d *Daemon) pullRoom(client *http.Client, addr string, room Room) (int, error) {
	// addr is carried for reporting and for marking the peer seen. Where the
	// request actually goes was decided when the client was built.
	store, err := d.storeFor(room.RoomID)
	if err != nil {
		return 0, err
	}
	d.mu.Lock()
	d.reviewRefusals(store, room.RoomID)
	have, err := store.SyncState(room.RoomID)
	d.mu.Unlock()
	if err != nil {
		return 0, err
	}
	cred, err := signRequestAs(d.id, room.RoomID, AdvertisedEndpoint())
	if err != nil {
		return 0, err
	}
	body, _ := json.Marshal(syncRequest{
		Protocol: wireVersion, RoomID: room.RoomID, Have: have,
		PeerID: cred.PeerID, Timestamp: cred.Timestamp, Nonce: cred.Nonce, Signature: cred.Signature,
		Endpoint: cred.Endpoint, Signer: cred.Signer, Chain: cred.Chain,
	})
	resp, err := client.Post(peerURL("/sync"), "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		// The room's guests revoked this machine's user, and the refusal carries the
		// room's guest-list events, the revocations that did among them (D-241, an
		// expelled user is told, through the refusal that carries the revocation).
		var refusal revocationRefusal
		if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&refusal); err != nil || refusal.RoomID != room.RoomID {
			return 0, nil
		}
		d.mu.Lock()
		defer d.mu.Unlock()
		d.keepGuestEvents(store, room, addr, refusal.GuestEvents)
		return 0, nil
	}
	if resp.StatusCode == http.StatusUnauthorized {
		// Not a guest there, or no such room. Both are ordinary when a peer hosts
		// rooms we are not in, so this is not an error worth repeating every second.
		return 0, nil
	}

	var out syncResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}
	// A peer speaking a protocol we cannot read is a condition to report, not to
	// guess at. Silently accepting unknown-shaped events is how a field means two
	// things. But an OLDER version we still understand is read, so that upgrading
	// is not a flag day for everyone at once (D-065).
	if !speaks(out.Protocol) {
		return 0, fmt.Errorf("peer speaks protocol v%d; this daemon reads v%d to v%d — upgrade the older side",
			out.Protocol, minWireVersion, wireVersion)
	}

	d.markPeerSeen(addr)

	d.mu.Lock()
	defer d.mu.Unlock()
	stored, rejected, unverified, notCounting, notGuest := 0, 0, 0, 0, 0
	// Changes to the guest list are taken first and the list computed again from
	// them before any other event is judged, so that a turn arriving with the
	// invitation that admits its author is stored in the same poll (D-235, a room's
	// one guest list decides whose events count).
	events := make([]Event, 0, len(out.Events))
	for i := range out.Events {
		events = append(events, fromWire(out.Events[i]))
	}
	sort.SliceStable(events, func(i, j int) bool {
		return isGuestEventType(events[i].EventType) && !isGuestEventType(events[j].EventType)
	})
	guestStored, listRead := false, false
	var list *GuestList
	readList := func() {
		if listRead {
			return
		}
		listRead = true
		if guestStored {
			if _, err := d.refreshGuestList(store, room.RoomID); err != nil {
				log.Printf("sync: could not compute the guest list of %s: %v", room.RoomName, err)
			}
			// An invitation that lifts a revocation lets the events refused past its
			// cutoff count, so they are asked for at the next poll of any machine
			// (D-236).
			d.reviewRefusals(store, room.RoomID)
		}
		list = d.guestList(room.RoomID)
	}
	defer readList()
	for i := range events {
		ev := events[i]
		guestChange := isGuestEventType(ev.EventType)
		if !guestChange {
			readList()
		}

		// §13 says events keep their origin through a relay, and until now that
		// was a rule with nothing enforcing it: an event arriving from one peer
		// claiming to originate with another was indistinguishable from one the
		// sender composed. The signature is made by the originating peer over the
		// event's own fields, so a relayer can carry it and cannot author it.
		//
		// Rejected rather than quarantined: a bad signature is not ambiguous the
		// way a sequence conflict is. There is no benign reading of it.
		if err := ev.Verify(); err != nil {
			log.Printf("sync: REJECTED event %.12s from %s — %v", ev.EventID, addr, err)
			rejected++
			continue
		}

		// A change to the guest list holds no conversation, so it is stored whenever it
		// verifies and its machine counts for its user, whoever its author is to this
		// machine: every guest computes the list from the same events, and
		// ResolveGuests decides which of them count (D-235, a room's one guest list).
		// Only a machine that speaks for a user writes one, so one under an older
		// scheme, or whose content does not read as its kind, is refused.
		if guestChange {
			if _, ok := guestEventFrom(ev); !ok {
				log.Printf("sync: REJECTED event %.12s from %s — a guest-list change that is not one signed for a user", ev.EventID, addr)
				rejected++
				continue
			}
			if err := d.judgeEvent(&ev); errors.Is(err, errNotCounting) {
				notCounting++
				d.refuse(store, &ev)
				continue
			} else if err != nil {
				log.Printf("sync: REJECTED event %.12s from %s — %v", ev.EventID, addr, err)
				rejected++
				continue
			}
			res, err := store.Insert(&ev)
			if err != nil {
				return stored, err
			}
			switch res {
			case InsertStored:
				stored++
				guestStored = true
			case InsertConflict:
				log.Printf("sync: CONFLICT from %s — %s sequence %d is already held by a different event; run `cogmer conflicts`",
					addr, PeerName(ev.signerKey()), ev.PeerSequence)
			}
			continue
		}

		// A valid signature proves the event came from the key it names. It does
		// not prove that key belongs to the person whose name is on it, and an
		// unverified origin is exactly the case where those differ. Refused at the
		// ORIGIN rather than at the sender, so a relayed event (§13) is judged by
		// who wrote it rather than by who carried it.
		//
		// Not stored, not quarantined, and not counted as seen: sync is a pull
		// against a watermark, so refusing simply leaves the events on offer. When
		// the two people verify, the next poll brings the whole backlog.
		//
		// The gate is by user: an event a machine signed for its user names the
		// user as its peer, so a colleague verified once is verified for every
		// machine of theirs, and the user's own events from another of the user's
		// machines pass as the user's own (D-054, verification gates synchronization and
		// injection; D-219, a user is a master key
		// that certifies machines through a shared key).
		if !d.authorVerified(ev.PeerID) {
			unverified++
			continue
		}

		// An event signed for a user is checked against what this machine holds
		// for that user: a chain whose shared-key certificate lost is refused, and
		// a removed machine's events past its cutoff and a suspended machine's
		// events are not stored (D-220, a removal names the last event of a machine
		// that counts; D-234, a machine not certified under the current shared key
		// is suspended). Neither is quarantined, and both stay on offer.
		if ev.SigVersion == 4 {
			if err := d.judgeEvent(&ev); errors.Is(err, errNotCounting) {
				notCounting++
				d.refuse(store, &ev)
				continue
			} else if err != nil {
				log.Printf("sync: REJECTED event %.12s from %s — %v", ev.EventID, addr, err)
				rejected++
				continue
			}
		}

		// In a room that keeps its guest list as events, an event counts only when
		// its author is on the list, or was revoked and wrote it within the cutoff of
		// the machine that signed it (D-235, a room's one guest list decides whose
		// events count; D-236, a revocation names the last turn that counts). One that
		// does not is left on offer, as an unverified author's is, so that an
		// invitation lifting the revocation brings it by ordinary sync. A room whose
		// list is room_guests has no such check, as before.
		if list != nil && !list.Counts(ev.PeerID, ev.signerKey(), ev.PeerSequence) {
			notGuest++
			d.refuse(store, &ev)
			continue
		}

		// Nothing below rewrites peerId, peerSequence, or eventId; doing so would
		// invalidate the signature, which is now how that rule is enforced.
		res, err := store.Insert(&ev)
		if err != nil {
			return stored, err
		}
		switch res {
		case InsertStored:
			stored++
			registerImages(store, &ev)
			// Stored and shown, and withheld from every assistant until the records it
			// names are fetched, which is off this path (D-221, an event whose identity
			// version the receiver lacks is withheld from the assistant until fetched).
			d.noteLacking(ev, addr)
		case InsertConflict:
			log.Printf("sync: CONFLICT from %s — %s sequence %d is already held by a different event; run `cogmer conflicts`",
				addr, PeerName(ev.signerKey()), ev.PeerSequence)
		}
	}
	if unverified > 0 {
		// Said once per poll and made actionable: this is not a fault, it is the
		// gate doing its job, and the only thing that clears it is two people on a
		// call. Silence here would look like an empty room.
		log.Printf("sync: %d event(s) held back — their origin peer is UNVERIFIED. "+
			"Both of you run /cogmer:peer-pair, which opens the two-word check in a browser; "+
			"their turns arrive in full once you have.", unverified)
	}
	if rejected > 0 {
		log.Printf("sync: %d event(s) from %s failed verification and were not stored", rejected, addr)
	}
	if notCounting > 0 {
		log.Printf("sync: %d event(s) from %s were not stored — the machine that signed them is removed past its cutoff or suspended",
			notCounting, addr)
	}
	if notGuest > 0 {
		log.Printf("sync: %d event(s) from %s were not stored — their author is not on %s's guest list, or wrote them past a revocation's cutoff",
			notGuest, addr, room.RoomName)
	}
	readList()
	if stored > 0 {
		d.notify()
	}
	return stored, nil
}

// refuse records that this machine refused an event whose machine is past a cutoff
// or suspended, or whose author is past a revocation's cutoff, so that the sequence is
// not offered again on every poll (D-236, a revocation names the last turn that
// counts). Called with d.mu held.
func (d *Daemon) refuse(store *Store, ev *Event) {
	if err := store.RecordRefusal(ev); err != nil {
		log.Printf("sync: could not record that %.12s was refused: %v", ev.EventID, err)
	}
}
