package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// A machine that writes an event into a room tells the room's other machines to poll it
// now, and each of them polls, of a colleague's machines in the room, only the one that
// told it last (D-306, a machine nudges the machines it shares a room with when it writes
// an event). The nudge carries nothing of the room. It is signed as a sync request is,
// over the room's identifier with nudgeScope before it, so that a nudge cannot be
// presented as a sync request or a sync request as a nudge.
const nudgeScope = "nudge/"

// nudgeTimeout bounds a nudge, which is worth nothing once a poll would have found the event.
const nudgeTimeout = 5 * time.Second

// nudger sends nudges, at most one at a time to each address. A nudge asked for while one
// is on its way to the address is sent once that one ends, and further asks in the
// meantime add nothing, since one poll fetches every event the address holds.
//
// A nudge counts as delivered only when the receiver answers it or polls this machine
// after it was asked for. Until then it is sent again, at nudgeRetryMin and twice as long
// each time up to nudgeRetryMax, for up to nudgeGiveUp, so that a nudge lost on its way
// does not leave a turn on this machine unasked for (D-306, a machine nudges the machines it shares a room with when it writes an event).
type nudger struct {
	mu      sync.Mutex
	sending map[string]bool      // address → a nudge is on its way
	again   map[string]string    // address → the room of a nudge to send when that one ends
	polled  map[string]time.Time // address → when the machine there last polled this one

	// The retry waits, which are the constants below unless a test says otherwise.
	retryMin, retryMax, giveUp time.Duration
}

const (
	nudgeRetryMin = time.Second
	nudgeRetryMax = time.Minute
	nudgeGiveUp   = time.Hour
)

// notePolled records that the machine at addr polled this machine now, which delivers
// every nudge to addr asked for before it.
func (n *nudger) notePolled(addr string) {
	if addr == "" {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.polled == nil {
		n.polled = map[string]time.Time{}
	}
	n.polled[addr] = time.Now()
}

func (n *nudger) polledSince(addr string, t time.Time) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.polled[addr].After(t)
}

func (n *nudger) waits() (lo, hi, giveUp time.Duration) {
	n.mu.Lock()
	defer n.mu.Unlock()
	lo, hi, giveUp = n.retryMin, n.retryMax, n.giveUp
	if lo == 0 {
		lo = nudgeRetryMin
	}
	if hi == 0 {
		hi = nudgeRetryMax
	}
	if giveUp == 0 {
		giveUp = nudgeGiveUp
	}
	return lo, hi, giveUp
}

// nudgeRoom tells every machine this machine polls for a room to poll it now.
func (d *Daemon) nudgeRoom(roomID string) { d.nudgeRoomExcept(roomID, "") }

// nudgeRoomExcept nudges every machine this machine polls for a room but the one at
// except, which is where the events came from.
func (d *Daemon) nudgeRoomExcept(roomID, except string) {
	if d.expelledFrom(roomID) {
		return
	}
	for _, addr := range d.members.RoomPeers(roomID, d.id.PeerID) {
		if addr != except {
			d.nudgeAddr(addr, roomID)
		}
	}
}

func (d *Daemon) nudgeAddr(addr, roomID string) {
	n := &d.nudges
	n.mu.Lock()
	if n.sending == nil {
		n.sending, n.again = map[string]bool{}, map[string]string{}
	}
	if n.sending[addr] {
		n.again[addr] = roomID
		n.mu.Unlock()
		return
	}
	n.sending[addr] = true
	n.mu.Unlock()
	go func() {
		for {
			d.deliverNudge(addr, roomID)
			n.mu.Lock()
			next, ok := n.again[addr]
			delete(n.again, addr)
			if !ok {
				delete(n.sending, addr)
				n.mu.Unlock()
				return
			}
			n.mu.Unlock()
			roomID = next
		}
	}()
}

// deliverNudge sends a nudge until the receiver answers it, polls this machine, refuses
// it, is no longer polled for the room, or nudgeGiveUp passes.
func (d *Daemon) deliverNudge(addr, roomID string) {
	asked := time.Now()
	wait, most, giveUp := d.nudges.waits()
	for tries := 1; ; tries++ {
		final, err := d.sendNudge(addr, roomID)
		if err == nil {
			return
		}
		if final {
			log.Printf("sync: %s refused a nudge: %v", shortEndpoint(addr), err)
			return
		}
		if tries == 1 {
			log.Printf("sync: could not nudge %s, and will try again until it answers or asks: %v", shortEndpoint(addr), err)
		}
		if time.Since(asked) > giveUp {
			log.Printf("sync: gave up nudging %s after %s: %v", shortEndpoint(addr), giveUp, err)
			return
		}
		time.Sleep(wait)
		if d.nudges.polledSince(addr, asked) || !d.pollsFor(addr, roomID) {
			return
		}
		if wait *= 2; wait > most {
			wait = most
		}
	}
}

// pollsFor says whether addr is still an address this machine polls for the room.
func (d *Daemon) pollsFor(addr, roomID string) bool {
	if d.expelledFrom(roomID) {
		return false
	}
	for _, a := range d.members.RoomPeers(roomID, d.id.PeerID) {
		if a == addr {
			return true
		}
	}
	return false
}

// sendNudge sends one nudge on the connection kept for polling the address (D-303, a
// machine keeps one connection to each address it polls). A machine on a version that
// takes no nudges answers 404, which is not an error. final says that sending it again
// would be refused again: the receiver refused it, or this machine cannot sign one.
func (d *Daemon) sendNudge(addr, roomID string) (final bool, err error) {
	client, err := d.syncClient(addr)
	if err != nil {
		return false, err
	}
	scoped := nudgeScope + roomID
	// No address: the receiver polls the one it recorded for this machine.
	cred, err := signRequestAs(d.id, scoped, "")
	if err != nil {
		return true, err
	}
	body, _ := json.Marshal(syncRequest{
		Protocol: wireVersion, RoomID: scoped,
		PeerID: cred.PeerID, Timestamp: cred.Timestamp, Nonce: cred.Nonce, Signature: cred.Signature,
		Endpoint: cred.Endpoint, Signer: cred.Signer, Chain: cred.Chain,
	})
	ctx, cancel := context.WithTimeout(context.Background(), nudgeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, peerURL("/nudge"), bytes.NewReader(body))
	if err != nil {
		return true, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<10))
	switch {
	case resp.StatusCode == http.StatusNoContent, resp.StatusCode == http.StatusNotFound:
		return false, nil
	case resp.StatusCode >= 500:
		return false, fmt.Errorf("the nudge was not taken: %s", resp.Status)
	default:
		return true, fmt.Errorf("the nudge was refused: %s", resp.Status)
	}
}

// handleNudge takes a nudge from a machine that wrote an event into a room this machine
// shares with it. The nudge is checked as a sync request is: its signature, timestamp and
// nonce (D-044, sync requests are signed), its signer against the connection (D-143,
// admission is proved by a signed request bound to the connection), and its user against
// the room's guests. The machine's recorded address is then polled at once, and becomes
// the one address of its user this machine polls.
func (d *Daemon) handleNudge(w http.ResponseWriter, r *http.Request) {
	var req syncRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBody)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !speaks(req.Protocol) {
		http.Error(w, "unsupported protocol", http.StatusBadRequest)
		return
	}
	roomID, ok := strings.CutPrefix(req.RoomID, nudgeScope)
	if !ok {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	room, ok := d.members.RoomByID(roomID)
	if !ok || d.expelledFrom(room.RoomID) {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	machine := req.requestMachine()
	if err := connectedAs(r, machine); err != nil {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	if err := d.verifyRequest(req, room.RoomID); err != nil {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	if addr, _ := d.members.MachineEndpoint(machine); addr != "" {
		d.nudgedBy(req.PeerID, addr)
	}
	w.WriteHeader(http.StatusNoContent)
}
