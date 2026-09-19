package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Verification is a thing both people do, not a thing one does to the other.
//
// Each side runs `claude-team verify <peer>`; each daemon holds a session until
// the other appears. Nothing arrives unsolicited, so this adds no inbound surface
// and no prompt anyone can be trained to dismiss -- both open questions in §12a
// stay closed by construction rather than by a rule someone has to remember.
//
// It is a separate act from joining, and from admission. Admission decides whether
// a peer may enter a room; this decides whether the key on file is the key the
// person you know actually holds. They answer different questions and are recorded
// separately.

const verifyTimeout = 90 * time.Second

type verifySession struct {
	peerID string
	nonce  []byte
	commit []byte

	mu          sync.Mutex
	theirCommit []byte
	theirNonce  []byte
	created     time.Time
}

type verifyMessage struct {
	Step      string `json:"step"` // "commit" or "reveal"
	PeerID    string `json:"peerId"`
	Payload   string `json:"payload"` // base64url: the commitment, or the nonce
	Signature string `json:"signature"`
}

// startVerify opens a session for a peer, replacing any earlier one. A second
// attempt should begin afresh rather than continue a half-finished exchange: a
// nonce reused across attempts is a nonce an attacker has already seen.
func (d *Daemon) startVerify(peerID string) (*verifySession, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	s := &verifySession{
		peerID: peerID, nonce: nonce,
		commit: sasCommitment(d.id.PeerID, nonce), created: time.Now(),
	}
	d.verifyMu.Lock()
	if d.verifying == nil {
		d.verifying = map[string]*verifySession{}
	}
	d.verifying[peerID] = s
	d.verifyMu.Unlock()
	return s, nil
}

func (d *Daemon) verifySessionFor(peerID string) (*verifySession, bool) {
	d.verifyMu.Lock()
	defer d.verifyMu.Unlock()
	s, ok := d.verifying[peerID]
	if ok && time.Since(s.created) > verifyTimeout {
		delete(d.verifying, peerID)
		return nil, false
	}
	return s, ok
}

func (d *Daemon) endVerify(peerID string) {
	d.verifyMu.Lock()
	delete(d.verifying, peerID)
	d.verifyMu.Unlock()
}

// handleVerify answers a peer's step in the exchange.
//
// A session must already exist, which is what keeps this from being an inbound
// channel: a peer whose user has not asked to verify gets "not expecting" and
// nothing is displayed to anyone. The caller must also be a peer this machine
// knows -- verification confirms a key already on file, and there is no key to
// confirm for a stranger.
func (d *Daemon) handleVerify(w http.ResponseWriter, r *http.Request) {
	var msg verifyMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	payload, err := base64.RawURLEncoding.DecodeString(msg.Payload)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	sig, err := base64.RawURLEncoding.DecodeString(msg.Signature)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !checkVerify(msg.PeerID, msg.Step, payload, sig) {
		http.Error(w, "signature does not match the peer id presenting it", http.StatusUnauthorized)
		return
	}
	// Two ways of not being ready, and both are ordinary rather than wrong.
	//
	// 409 rather than 401 for "I do not know you": two people pair within seconds
	// of each other, so whoever types first always arrives before the other has
	// recorded them. Answering 401 made that a hard failure, which meant the first
	// person to type always lost. Found on a two-machine run, where the ordering is
	// real rather than arranged.
	//
	// 401 is now reserved for a signature that does not verify, which is the only
	// answer here that waiting cannot fix.
	if !d.members.Knows(msg.PeerID) {
		http.Error(w, "not a peer this machine knows yet", http.StatusConflict)
		return
	}
	s, ok := d.verifySessionFor(msg.PeerID)
	if !ok {
		http.Error(w, "not expecting a verification", http.StatusConflict)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	switch msg.Step {
	case "commit":
		s.theirCommit = payload
		reply(w, d, s, "commit", s.commit)
	case "reveal":
		// The commitment must have arrived first and must open. Accepting a reveal
		// from a peer that never committed would let it choose its nonce after
		// seeing ours, which is the whole of the attack this protocol prevents.
		if s.theirCommit == nil {
			http.Error(w, "reveal before commit", http.StatusBadRequest)
			return
		}
		if !sasOpens(s.theirCommit, msg.PeerID, payload) {
			http.Error(w, "revealed nonce does not open the commitment", http.StatusBadRequest)
			return
		}
		s.theirNonce = payload
		reply(w, d, s, "reveal", s.nonce)
	default:
		http.Error(w, "unknown step", http.StatusBadRequest)
	}
}

func reply(w http.ResponseWriter, d *Daemon, s *verifySession, step string, payload []byte) {
	writeJSON(w, verifyMessage{
		Step: step, PeerID: d.id.PeerID,
		Payload:   base64.RawURLEncoding.EncodeToString(payload),
		Signature: base64.RawURLEncoding.EncodeToString(signVerify(d.id.private, step, d.id.PeerID, payload)),
	})
}

// RunVerification drives the exchange to a string of words, or to an error.
//
// Commit, commit, reveal, reveal -- in that order, and the order IS the security.
// Both nonces are fixed before either is known, so a relaying attacker must choose
// what he shows each side while blind to what the other will show.
// Both sides run this at once, and only ONE of them needs to drive.
//
// Each side starts a session and then does two things in the same loop: tries to
// drive the exchange outward, and watches whether the other side has already
// driven it inward. Whichever completes first yields the words.
//
// Driving from both ends looks symmetric and is not: the side that finishes tears
// its session down, so a peer that started a moment later finds nothing to talk to
// and waits until it times out. That is not a rare race -- it is what happens
// whenever two people type a few seconds apart, which is always.
//
// Commit, commit, reveal, reveal -- in that order, and the order IS the security.
// Both nonces are fixed before either is known, so a relaying attacker must choose
// what he shows each side while blind to what the other will show.
func (d *Daemon) RunVerification(peerID string, addrs []string) (string, error) {
	s, err := d.startVerify(peerID)
	if err != nil {
		return "", err
	}
	defer d.endVerify(peerID)

	client := &http.Client{Timeout: 10 * time.Second}
	deadline := time.Now().Add(verifyTimeout)

	for {
		// Inbound: the other side ran the whole exchange against this session, so
		// the nonce it revealed is already here and nothing further is needed.
		s.mu.Lock()
		theirs := s.theirNonce
		s.mu.Unlock()
		if theirs != nil {
			return SAS(d.id.PeerID, s.nonce, peerID, theirs), nil
		}

		// Outbound: one round against each address we know of, keeping the one that
		// answers signed by the identity we asked for. A reply from a different peer
		// is a wrong peer, not a wrong address, and verifyStep refuses it.
		var lastErr error
		for _, addr := range addrs {
			theirCommit, err := d.verifyStep(client, addr, peerID, "commit", s.commit)
			if err != nil {
				lastErr = err
				continue
			}
			theirNonce, err := d.verifyStep(client, addr, peerID, "reveal", s.nonce)
			if err != nil {
				return "", err
			}
			if !sasOpens(theirCommit, peerID, theirNonce) {
				// The peer revealed something other than what it committed to.
				// That is not a mismatch to read aloud -- it is a protocol
				// violation, and the only thing that does it is an attempt to
				// choose a nonce after seeing ours.
				return "", errors.New("the peer's revealed nonce does not match what it committed to; " +
					"this is not a network fault and must not be retried")
			}
			return SAS(d.id.PeerID, s.nonce, peerID, theirNonce), nil
		}
		if lastErr != nil && !errors.Is(lastErr, errPeerNotReady) && !isReachErr(lastErr) {
			return "", lastErr
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("%s did not run `claude-team pair` or `claude-team verify` within %s",
				PeerName(peerID), verifyTimeout)
		}
		time.Sleep(time.Second)
	}
}

var errPeerNotReady = errors.New("peer is not expecting a verification")

// isReachErr distinguishes "that address is not answering" from a refusal, so a
// candidate list containing dead addresses does not abort the attempt.
func isReachErr(err error) bool {
	var ue *url.Error
	return errors.As(err, &ue)
}

func (d *Daemon) verifyStep(client *http.Client, addr, expect, step string, payload []byte) ([]byte, error) {
	body, _ := json.Marshal(verifyMessage{
		Step: step, PeerID: d.id.PeerID,
		Payload:   base64.RawURLEncoding.EncodeToString(payload),
		Signature: base64.RawURLEncoding.EncodeToString(signVerify(d.id.private, step, d.id.PeerID, payload)),
	})
	resp, err := client.Post("http://"+addr+"/verify", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return nil, errPeerNotReady
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("verification refused: %s", resp.Status)
	}
	var msg verifyMessage
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		return nil, err
	}
	out, err := base64.RawURLEncoding.DecodeString(msg.Payload)
	if err != nil {
		return nil, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(msg.Signature)
	if err != nil {
		return nil, err
	}
	if msg.PeerID != expect {
		return nil, fmt.Errorf("that address answered as %s, not %s", PeerName(msg.PeerID), PeerName(expect))
	}
	if !checkVerify(msg.PeerID, msg.Step, out, sig) {
		return nil, errors.New("the peer's reply is not signed by the key it claims")
	}
	return out, nil
}

// --- the local side: a person asks for a verification, and answers it ---

type verifyStartRequest struct {
	Peer string `json:"peer"`
}

type verifyStartResponse struct {
	Peer  string `json:"peer"`
	Name  string `json:"name"`
	Words string `json:"words"`
	Error string `json:"error,omitempty"`
}

type verifyConfirmRequest struct {
	Peer    string `json:"peer"`
	Matched bool   `json:"matched"`
}

// handleVerifyStart runs the exchange for a peer named by this machine's own user.
// It is on the LOCAL routes, not the peer routes: nobody else may cause a
// verification to begin here.
func (d *Daemon) handleVerifyStart(w http.ResponseWriter, r *http.Request) {
	var req verifyStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !d.members.Knows(req.Peer) {
		writeJSON(w, verifyStartResponse{Error: fmt.Sprintf(
			"%s is not a peer this machine knows; `claude-team allow <identifier>` records one first",
			PeerName(req.Peer))})
		return
	}
	words, err := d.RunVerification(req.Peer, d.syncTargets())
	if err != nil {
		writeJSON(w, verifyStartResponse{Error: err.Error()})
		return
	}
	writeJSON(w, verifyStartResponse{Peer: req.Peer, Name: PeerName(req.Peer), Words: words})
}

// handleVerifyConfirm records the answer a person gave.
//
// Only a match is recorded. A mismatch deliberately stores nothing: there is no
// "failed verification" state worth keeping, because the useful response is to
// stop and say what was seen, and a stored failure invites a UI that offers to
// retry -- which is precisely what must not be offered.
func (d *Daemon) handleVerifyConfirm(w http.ResponseWriter, r *http.Request) {
	var req verifyConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !req.Matched {
		writeJSON(w, map[string]string{"status": "not recorded"})
		return
	}
	if err := d.members.MarkVerified(req.Peer); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"status": "verified"})
}
