package main

import (
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"
)

//go:embed pair.html
var pairPageSrc string

var pairTemplate = template.Must(template.New("pair").Parse(pairPageSrc))

// --- the pairing ceremony, in the view (D-088) ---
//
// D-055 is unchanged by this and must stay unchanged: two words, from a live
// commit/reveal exchange, compared aloud on a call, by both people at once, with no
// fallback. What moves is the surface. A terminal was never the point -- it was the
// only thing available that was not the model (D-080), and the view is the other.
//
// EVERY PAIRING GETS ITS OWN URL, and that is load-bearing rather than tidy:
//
//   A pairing page is a live ceremony with a deadline, not a dashboard. Two of them
//   must never share a page, and a page left open from an earlier attempt must never
//   quietly become a different one -- the whole security property is that the person
//   knows which key they are vouching for.
//
//   It is also what makes a second pairing visible. A fixed address whose contents
//   we rewrote would, at best, change a tab the person is not looking at. A fresh
//   address is a fresh tab, which the browser raises.
//
// The id is 128 random bits because it is the capability: holding it is what lets a
// page start this exchange, and the guard (D-087) is what stops another site asking.

const pairTTL = 15 * time.Minute

type pendingPair struct {
	peerID  string
	created time.Time
}

type pairRegistry struct {
	mu   sync.Mutex
	byID map[string]*pendingPair
}

// newPairing records an intent to pair and returns the id its page lives at.
func (d *Daemon) newPairing(peerID string) string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		// Never guess an id. A predictable one would let another page start a
		// verification the person never asked for.
		return ""
	}
	id := base64.RawURLEncoding.EncodeToString(raw)

	d.pairs.mu.Lock()
	defer d.pairs.mu.Unlock()
	if d.pairs.byID == nil {
		d.pairs.byID = map[string]*pendingPair{}
	}
	for k, v := range d.pairs.byID {
		if time.Since(v.created) > pairTTL {
			delete(d.pairs.byID, k)
		}
	}
	d.pairs.byID[id] = &pendingPair{peerID: peerID, created: time.Now()}
	return id
}

// peerForPairing resolves a pairing id. Expiry is reported as absence, because to
// the person there is no difference worth explaining: the link no longer works and
// the answer is to start again.
func (d *Daemon) peerForPairing(id string) (string, bool) {
	d.pairs.mu.Lock()
	defer d.pairs.mu.Unlock()
	p, ok := d.pairs.byID[id]
	if !ok {
		return "", false
	}
	if time.Since(p.created) > pairTTL {
		delete(d.pairs.byID, id)
		return "", false
	}
	return p.peerID, true
}

type pairNewRequest struct {
	Peer string `json:"peer"`
}

type pairNewResponse struct {
	URL   string `json:"url,omitempty"`
	Error string `json:"error,omitempty"`
}

// handlePairNew mints the page a pairing happens on. Guarded, because it is the
// daemon agreeing to hold a pairing open.
func (d *Daemon) handlePairNew(w http.ResponseWriter, r *http.Request) {
	var req pairNewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !d.members.Knows(req.Peer) {
		writeJSON(w, pairNewResponse{Error: fmt.Sprintf(
			"%s is not a peer this machine knows", PeerName(req.Peer))})
		return
	}
	id := d.newPairing(req.Peer)
	if id == "" {
		writeJSON(w, pairNewResponse{Error: "could not generate a pairing link"})
		return
	}
	writeJSON(w, pairNewResponse{URL: "http://" + addr() + "/pair/" + id})
}

// handlePairPage serves the ceremony. A GET a person navigated to, so it is not
// guarded -- there is nothing here another site could read anyway, and a page it
// cannot obtain the id for it cannot reach.
func (d *Daemon) handlePairPage(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/pair/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	peerID, ok := d.peerForPairing(id)
	if !ok {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(pairExpiredPage())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = pairTemplate.Execute(w, struct {
		PairID   string
		PeerName string
	}{PairID: id, PeerName: PeerName(peerID)})
}

// pairExpiredPage says the one useful thing. A bare 404 would leave somebody
// staring at a browser error for a link this program gave them ninety seconds ago.
func pairExpiredPage() []byte {
	return []byte(`<!doctype html><meta charset=utf-8>
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>pairing link expired</title>
<style>:root{--bg:#faf9f7;--fg:#1c1b19;--dim:#6b6660}
@media (prefers-color-scheme:dark){:root{--bg:#16150f;--fg:#e8e4dc;--dim:#8f8a82}}
body{margin:0;background:var(--bg);color:var(--fg);
font:15px/1.6 ui-sans-serif,-apple-system,"Segoe UI",Roboto,sans-serif}
main{max-width:34rem;margin:0 auto;padding:4rem 1.5rem}
h1{font-size:15px;letter-spacing:.14em;text-transform:uppercase}
p{color:var(--dim)}code{font:13px ui-monospace,Menlo,monospace}</style>
<main><h1>This pairing link has expired</h1>
<p>Pairing links last a few minutes, because they hold a ceremony open.
Start a new one with <code>/peer-pair</code> and this page will be replaced.</p>
<p>Nothing was recorded, and nobody was verified.</p></main>`)
}
