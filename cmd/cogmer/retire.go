package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// --- asking a machine the user holds to erase its keys before it is removed (D-252) ---
//
// A machine removed as retired still holds the shared key unless it erased it, and
// whoever then has the machine can use the key to add machines of their own. So the
// machine that removes it first asks it, over its pinned connection while its
// certificate is still valid, to erase its keys and to say that it did, and signs the
// removal only after the answer (D-251, removing a machine that someone other than the
// user could still use always certifies a new shared key). A removed machine is refused
// at its connection and so is never asked after the removal (D-101, peer connections
// are TLS pinned to a key this machine has recorded).

// selfRetireTag is the purpose tag of a request to erase a machine's keys, so that its
// signature can be taken for no other request.
const selfRetireTag = protocolNamespace + "/retire/v1"

// retirePayload names the machine the request is for, which the machine that receives
// it checks is itself, so that a request signed for one machine erases no other.
type retirePayload struct {
	Machine string `json:"machine"`
}

type retireAnswer struct {
	Status string `json:"status"`
}

// retireAskTimeout is how long the removing machine waits for the other to answer.
const retireAskTimeout = 10 * time.Second

// handleSelfRetire answers another of the user's machines that asks this one to erase
// its keys. Only a machine of this machine's user that holds the shared key may ask, so
// a cloud session, which holds none, cannot (D-219, a user is a master key that
// certifies machines through a shared key).
func (d *Daemon) handleSelfRetire(w http.ResponseWriter, r *http.Request) {
	if !d.id.HasUser() || d.isCloudSession() {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	var req selfListsRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := d.checkRetireRequest(req, r); err != nil {
		log.Printf("retire: refused a request from %s — %v", PeerName(req.Signer), err)
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	var p retirePayload
	if json.Unmarshal(req.Payload, &p) != nil || p.Machine != d.id.PeerID {
		http.Error(w, "the request names another machine", http.StatusBadRequest)
		return
	}
	d.retireSelf(req.Signer)
	writeJSON(w, retireAnswer{Status: "erased"})
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	if d.afterRetire != nil {
		d.afterRetire()
		return
	}
	// A test that has not said what happens next must not end its own process.
	if inTestBinary() {
		return
	}
	// The answer is on its way, and the process holds the keys in memory until it ends.
	go func() {
		time.Sleep(time.Second)
		os.Exit(0)
	}()
}

// checkRetireRequest establishes that a machine of this machine's own user that holds
// the shared key, and is not this machine, signed the request now. A cloud key alone
// has no live chain, since only its session keys sign, and a cloud session is refused by name.
func (d *Daemon) checkRetireRequest(req selfListsRequest, r *http.Request) error {
	if req.Chain != nil && req.Chain.Session != nil {
		return errors.New("a cloud session holds no shared key and cannot ask a machine to erase its keys")
	}
	if req.Signer == d.id.PeerID {
		return errors.New("a machine does not ask itself")
	}
	return d.checkSelfRequest(selfRetireTag, req, r)
}

// retireSelf erases this machine's private keys: its own, and its copy of the shared
// key. It keeps the rest of the state directory, which holds no key, for the user to
// delete or keep.
func (d *Daemon) retireSelf(askedBy string) {
	home := d.homeDirectory()
	for _, f := range []string{sharedKeyPathIn(home), filepath.Join(home, "identity.key"), filepath.Join(home, "identity.json")} {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			log.Printf("retire: could not erase %s: %v", filepath.Base(f), err)
		}
	}
	log.Printf("retire: erased this machine's keys at the request of %s", PeerName(askedBy))
}

// askToRetire asks machine, one of the user's, to erase its keys, and returns nil only
// when it answered that it had. A machine that does not answer is one that someone
// other than the user could still use (D-252).
func (d *Daemon) askToRetire(machine string) error {
	id := d.id
	if !id.HasUser() || id.Chain == nil || id.Chain.Session != nil {
		return errors.New("this machine holds no shared key, so it cannot ask another to erase its keys")
	}
	addr, _ := d.members.MachineEndpoint(machine)
	if addr == "" {
		return errors.New("this machine does not know where it listens")
	}
	payload, err := json.Marshal(retirePayload{Machine: machine})
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
		selfRequestBytes(selfRetireTag, req.User, req.Signer, req.Timestamp, req.Nonce, req.Endpoint, req.Payload)))
	conf, err := d.clientConfig(id.User)
	if err != nil {
		return err
	}
	client, err := clientFor(addr, retireAskTimeout, conf)
	if err != nil {
		return err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	resp, err := client.Post(peerURL("/self/retire"), "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		text, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(text)))
	}
	var ans retireAnswer
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<10)).Decode(&ans); err != nil {
		return err
	}
	if ans.Status != "erased" {
		return fmt.Errorf("it answered %q, not that it had erased its keys", ans.Status)
	}
	return nil
}

type retireAskRequest struct {
	Machine string `json:"machine"`
}

type retireAskResponse struct {
	Status string `json:"status,omitempty"`
	Error  string `json:"error,omitempty"`
}

// handleRetireAsk is the page's request that the machine the user is removing erase its
// keys first. The page treats any answer but "erased" as a machine that someone other
// than the user could still use.
func (d *Daemon) handleRetireAsk(w http.ResponseWriter, r *http.Request) {
	var req retireAskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	c, err := d.removableMachine(req.Machine)
	if err != nil {
		writeJSON(w, retireAskResponse{Error: err.Error()})
		return
	}
	switch {
	case req.Machine == d.id.PeerID:
		writeJSON(w, retireAskResponse{Error: "that is the machine you are using, which erases its own key when it removes itself"})
	case c.Kind == KindCloud:
		writeJSON(w, retireAskResponse{Error: "a cloud key holds no shared key to erase"})
	default:
		if err := d.askToRetire(req.Machine); err != nil {
			log.Printf("retire: %s was not asked to erase its keys: %v", PeerName(req.Machine), err)
			writeJSON(w, retireAskResponse{Error: "It did not answer that it had erased its keys."})
			return
		}
		writeJSON(w, retireAskResponse{Status: "erased"})
	}
}
