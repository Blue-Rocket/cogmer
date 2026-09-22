package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
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

// --- an invitation delivered over the channel pairing established (D-105) ---
//
// A host admits a guest by writing a guest-list row, which is the admission and is
// the host's judgement alone. This carries the news of it, so that forming a room
// costs one command rather than a command plus a string pasted into a chat window.
//
// Delivering an offer admits nobody. The row already exists on the host's side;
// this only says so. Joining stays the guest's own act, and reading the room still
// waits on verification (D-054). An offer arriving is data being stored, never a
// turn being taken, so §3.7 is untouched.

// Namespaced like every other signing tag, and arbitrary on purpose: nothing
// cryptographic may depend on the product's name (D-069).
const offerTag = protocolNamespace + "/offer/v1"

type offerRequest struct {
	Protocol  int    `json:"protocol"`
	PeerID    string `json:"peerId"` // the host, who is doing the admitting
	RoomID    string `json:"roomId"`
	RoomName  string `json:"roomName"`
	Endpoint  string `json:"endpoint"` // where the host listens
	Timestamp string `json:"timestamp"`
	Nonce     string `json:"nonce"`
	Signature string `json:"signature"`
}

// offerBytes is a signing scheme of its own rather than a reuse of the sync one.
// Two messages that mean different things must not be interchangeable under one
// signature, or a captured offer could be replayed as something else (D-058).
func offerBytes(peerID, roomID, roomName, endpoint, ts, nonce string) []byte {
	var b strings.Builder
	put := func(s string) {
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(s)))
		b.Write(n[:])
		b.WriteString(s)
	}
	put(offerTag)
	put(peerID)
	put(roomID)
	put(roomName)
	put(endpoint)
	put(ts)
	put(nonce)
	return []byte(b.String())
}

func (d *Daemon) signOffer(r Room, endpoint string) (offerRequest, error) {
	if d.id.private == nil {
		return offerRequest{}, errors.New("no private key: this peer cannot prove who it is")
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return offerRequest{}, err
	}
	nonce := base64.RawURLEncoding.EncodeToString(raw)
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	return offerRequest{
		Protocol: wireVersion, PeerID: d.id.PeerID,
		RoomID: r.RoomID, RoomName: r.RoomName, Endpoint: endpoint,
		Timestamp: ts, Nonce: nonce,
		Signature: base64.RawURLEncoding.EncodeToString(
			ed25519.Sign(d.id.private, offerBytes(d.id.PeerID, r.RoomID, r.RoomName, endpoint, ts, nonce))),
	}, nil
}

// handleOffer receives an invitation. It stores it and does nothing else: the room
// is not recorded, not synchronized and not shown as joined, because none of that
// has been agreed to yet.
func (d *Daemon) handleOffer(w http.ResponseWriter, r *http.Request) {
	var req offerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !speaks(req.Protocol) {
		http.Error(w, "unsupported protocol", http.StatusBadRequest)
		return
	}
	pub, err := PublicFromPeerID(req.PeerID)
	if err != nil {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	sig, err := base64.RawURLEncoding.DecodeString(req.Signature)
	if err != nil || !ed25519.Verify(pub,
		offerBytes(req.PeerID, req.RoomID, req.RoomName, req.Endpoint, req.Timestamp, req.Nonce), sig) {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	t, err := time.Parse(time.RFC3339Nano, req.Timestamp)
	if err != nil {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	if age := time.Since(t); age > authTolerance || age < -authTolerance {
		http.Error(w, "stale", http.StatusUnauthorized)
		return
	}
	if !d.replay.admit(req.Nonce, time.Now().UTC()) {
		http.Error(w, "replayed", http.StatusUnauthorized)
		return
	}
	// Only from somebody this machine has verified. A host withholds until then
	// (D-106), so an offer arriving early is either an older build or a peer
	// ignoring the rule — and accepting it would produce the silent room the
	// withholding exists to prevent.
	if !d.members.IsVerified(req.PeerID) {
		http.Error(w, "not verified", http.StatusForbidden)
		return
	}
	if err := d.members.RecordOffer(Offer{
		RoomID: req.RoomID, RoomName: req.RoomName, Host: req.PeerID, Endpoint: req.Endpoint,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("OFFER %s from %s — accept with /cogmer:room-join %s",
		req.RoomName, PeerName(req.PeerID), req.RoomName)
	d.notify()
	writeJSON(w, map[string]string{"status": "offered"})
}

// deliverOffer tells one peer about one room. Returns an error the caller can act
// on, because a host that cannot reach a guest must be told at the time and handed
// the string instead.
func (d *Daemon) deliverOffer(peerID string, r Room) error {
	addr := d.members.PeerEndpoint(peerID)
	if addr == "" {
		return fmt.Errorf("no address recorded for %s", PeerName(peerID))
	}
	req, err := d.signOffer(r, AdvertisedEndpoint())
	if err != nil {
		return err
	}
	conf, err := d.clientConfig(peerID)
	if err != nil {
		return err
	}
	client, err := clientFor(addr, 10*time.Second, conf)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(req)
	resp, err := client.Post(peerURL("/offer"), "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<12))
		return fmt.Errorf("%s: %s", resp.Status, bytes.TrimSpace(msg))
	}
	return nil
}

// deliverPending sends whatever was waiting for a peer, which is every room they
// have been admitted to. Called when a verification completes: the sequence a host
// expects — invite, then verify — produces the room at the end of it rather than a
// silent one in the middle (D-106).
func (d *Daemon) deliverPending(peerID string) {
	for _, r := range d.members.RoomsAdmitting(peerID) {
		if err := d.deliverOffer(peerID, r); err != nil {
			log.Printf("offer: could not tell %s about %s (%v); they can still be sent the invitation line",
				PeerName(peerID), r.RoomName, err)
		}
	}
}

type sendOfferRequest struct {
	Peer   string `json:"peer"`
	RoomID string `json:"roomId"`
}

type sendOfferResponse struct {
	Delivered bool   `json:"delivered"`
	Error     string `json:"error,omitempty"`
}

// handleSendOffer lets a typed command ask the daemon to deliver, because the
// daemon is what holds the tunnel and the peer certificates. A failure is returned
// rather than logged: the host must be told at the time so they can send the line
// by hand instead (D-105).
func (d *Daemon) handleSendOffer(w http.ResponseWriter, r *http.Request) {
	var req sendOfferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	room, ok := d.members.RoomByID(req.RoomID)
	if !ok {
		writeJSON(w, sendOfferResponse{Error: "no such room"})
		return
	}
	if err := d.deliverOffer(req.Peer, room); err != nil {
		writeJSON(w, sendOfferResponse{Error: err.Error()})
		return
	}
	writeJSON(w, sendOfferResponse{Delivered: true})
}
