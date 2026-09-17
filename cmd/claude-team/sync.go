package main

import (
	"bytes"
	"encoding/json"
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
	Room string           `json:"room"`
	Have map[string]int64 `json:"have"`
}

type syncResponse struct {
	Room   string  `json:"room"`
	Events []Event `json:"events"`
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
	if req.Room != d.room {
		http.Error(w, "unknown room", http.StatusNotFound)
		return
	}
	evs, err := d.store.EventsSince(d.room, req.Have)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, syncResponse{Room: d.room, Events: evs})
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
func (d *Daemon) RunSync(peers []string, every time.Duration) {
	if len(peers) == 0 {
		return
	}
	log.Printf("sync: polling %d peer(s) every %s", len(peers), every)
	client := &http.Client{Timeout: 5 * time.Second}
	for {
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
	d.mu.Lock()
	have, err := d.store.SyncState(d.room)
	d.mu.Unlock()
	if err != nil {
		return 0, err
	}
	body, _ := json.Marshal(syncRequest{Room: d.room, Have: have})
	resp, err := client.Post("http://"+addr+"/sync", "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var out syncResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}

	d.markPeerSeen(addr)

	d.mu.Lock()
	defer d.mu.Unlock()
	stored := 0
	for i := range out.Events {
		ev := out.Events[i]
		// §13: events keep their origin through a relay. Nothing here rewrites
		// peerId, peerSequence, or eventId.
		res, err := d.store.Insert(&ev)
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
	if stored > 0 {
		d.notify()
	}
	return stored, nil
}
