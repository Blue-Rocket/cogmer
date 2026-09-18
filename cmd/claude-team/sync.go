package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Phase 2 anti-entropy, reduced to what the §30 experiment needs.
//
// Deliberately omitted, and none of it changes what the experiment measures:
// invitations and guest lists (§12) — peers are named by CLAUDE_TEAM_PEERS;
// signatures (§25) — a peer is trusted to report its own identity. Either would
// entrench a format before D-023 has settled one.
//
// Polling is NOT among the omissions: it is the decided default (D-031). Peer
// propagation is half a second; arrival into a teammate's Claude waits for their
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
	// Endpoint is where the caller listens. Synchronisation is a pull, so a peer
	// that never says where it is can be read from and never read.
	Endpoint string `json:"endpoint,omitempty"`
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
	room, ok := d.members.RoomByID(req.RoomID)
	if !ok {
		// Do not distinguish "no such room" from "not a guest": both answers are
		// the same to anyone entitled to neither, and one of them is a disclosure.
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	if err := d.verifyRequest(req, room.RoomID); err != nil {
		log.Printf("sync: refused a request for %s — %v", room.RoomName, err)
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	// A guest that tells us where it listens becomes reachable, which is what
	// makes the exchange mutual rather than one peer reading another.
	if req.Endpoint != "" {
		_ = d.members.AddRoomPeer(room.RoomID, req.Endpoint)
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
	raw := os.Getenv("CLAUDE_TEAM_PEERS")
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

// RunSync polls each configured peer. A failed poll is not an error condition:
// peers are expected to be absent, and the next round recovers whatever was
// missed (§26).
func (d *Daemon) RunSync(_ []string, every time.Duration) {
	client := &http.Client{Timeout: 5 * time.Second}
	for {
		peers := d.syncTargets() // re-read: a room joined since last round has peers
		for _, addr := range peers {
			if n, err := d.pullFrom(client, addr); err != nil {
				log.Printf("sync: %s unreachable (%v)", addr, err)
			} else if n > 0 {
				log.Printf("sync: received %d event(s) from %s", n, addr)
			}
		}
		time.Sleep(every)
	}
}

func (d *Daemon) pullFrom(client *http.Client, addr string) (int, error) {
	rooms, err := d.members.Rooms()
	if err != nil {
		return 0, err
	}
	total := 0
	for _, r := range rooms {
		n, err := d.pullRoom(client, addr, r)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

// syncTargets is every address worth asking: those configured for this machine,
// and those each room was joined through. A room carries its own peers because
// reachability is a property of a room's membership, not of the daemon.
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
			for _, a := range d.members.RoomPeers(r.RoomID) {
				add(a)
			}
		}
	}
	return out
}

func (d *Daemon) pullRoom(client *http.Client, addr string, room Room) (int, error) {
	store, err := d.storeFor(room.RoomID)
	if err != nil {
		return 0, err
	}
	d.mu.Lock()
	have, err := store.SyncState(room.RoomID)
	d.mu.Unlock()
	if err != nil {
		return 0, err
	}
	ts, nonce, sig, err := signRequest(d.id, room.RoomID, peerAddr())
	if err != nil {
		return 0, err
	}
	body, _ := json.Marshal(syncRequest{
		Protocol: wireVersion, RoomID: room.RoomID, Have: have,
		PeerID: d.id.PeerID, Timestamp: ts, Nonce: nonce, Signature: sig,
		Endpoint: peerAddr(),
	})
	resp, err := client.Post("http://"+addr+"/sync", "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		// Not a guest there, or no such room. Both are ordinary when a peer hosts
		// rooms we are not in, so this is not an error worth repeating every second.
		return 0, nil
	}

	var out syncResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}
	// A peer speaking a protocol we do not is a condition to report, not to
	// guess at. Silently accepting unknown-shaped events is how a field means
	// two things.
	if out.Protocol != 0 && out.Protocol != wireVersion {
		return 0, fmt.Errorf("peer speaks protocol v%d, this daemon speaks v%d", out.Protocol, wireVersion)
	}

	d.markPeerSeen(addr)

	d.mu.Lock()
	defer d.mu.Unlock()
	stored, rejected := 0, 0
	for i := range out.Events {
		ev := fromWire(out.Events[i])

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

		// Nothing below rewrites peerId, peerSequence, or eventId; doing so would
		// invalidate the signature, which is now how that rule is enforced.
		res, err := store.Insert(&ev)
		if err != nil {
			return stored, err
		}
		switch res {
		case InsertStored:
			stored++
		case InsertConflict:
			log.Printf("sync: CONFLICT from %s — peer %s sequence %d is already held by a different event; run `claude-team conflicts`",
				addr, PeerName(ev.PeerID), ev.PeerSequence)
		}
	}
	if rejected > 0 {
		log.Printf("sync: %d event(s) from %s failed verification and were not stored", rejected, addr)
	}
	if stored > 0 {
		d.notify()
	}
	return stored, nil
}
