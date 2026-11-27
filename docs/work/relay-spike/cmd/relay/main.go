// Command relay joins a browser's WebSocket to a daemon's, for the relay spike. A daemon keeps a control
// connection at /serve?key=K; a browser dials /dial?key=K; the relay tells the daemon "dial <id>", the
// daemon opens /accept?id=<id>, and the relay copies binary messages both ways. It reads nothing it
// carries, which the browser and daemon encrypt end to end, and keeps no log of who connects.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type relay struct {
	mu      sync.Mutex
	daemons map[string]*websocket.Conn
	waiting map[string]chan *websocket.Conn
}

func main() {
	addr := flag.String("listen", "127.0.0.1:8091", "address to listen on")
	flag.Parse()
	r := &relay{daemons: map[string]*websocket.Conn{}, waiting: map[string]chan *websocket.Conn{}}
	http.HandleFunc("/serve", r.serve)
	http.HandleFunc("/accept", r.accept)
	http.HandleFunc("/dial", r.dial)
	log.Printf("relay spike listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func accept(w http.ResponseWriter, req *http.Request) (*websocket.Conn, error) {
	c, err := websocket.Accept(w, req, &websocket.AcceptOptions{InsecureSkipVerify: true}) // any origin: the page is elsewhere
	if err == nil {
		c.SetReadLimit(1 << 22)
	}
	return c, err
}

func (r *relay) serve(w http.ResponseWriter, req *http.Request) {
	key := req.URL.Query().Get("key")
	c, err := accept(w, req)
	if err != nil {
		return
	}
	r.mu.Lock()
	if old := r.daemons[key]; old != nil {
		old.Close(websocket.StatusGoingAway, "replaced")
	}
	r.daemons[key] = c
	r.mu.Unlock()
	ctx := req.Context()
	for { // the daemon sends nothing here; reading notices when it goes
		if _, _, err := c.Read(ctx); err != nil {
			break
		}
	}
	r.mu.Lock()
	if r.daemons[key] == c {
		delete(r.daemons, key)
	}
	r.mu.Unlock()
}

func (r *relay) accept(w http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")
	r.mu.Lock()
	ch := r.waiting[id]
	delete(r.waiting, id)
	r.mu.Unlock()
	if ch == nil {
		http.Error(w, "no such dial", http.StatusNotFound)
		return
	}
	c, err := accept(w, req)
	if err != nil {
		return
	}
	done := make(chan struct{})
	ch <- c
	<-done // never closed: the dial handler owns the connection until it ends, and the request ends with it
}

func (r *relay) dial(w http.ResponseWriter, req *http.Request) {
	key := req.URL.Query().Get("key")
	r.mu.Lock()
	d := r.daemons[key]
	r.mu.Unlock()
	if d == nil {
		http.Error(w, "that daemon is not connected", http.StatusNotFound)
		return
	}
	b, err := accept(w, req)
	if err != nil {
		return
	}
	defer b.CloseNow()
	var raw [8]byte
	rand.Read(raw[:])
	id := hex.EncodeToString(raw[:])
	ch := make(chan *websocket.Conn, 1)
	r.mu.Lock()
	r.waiting[id] = ch
	r.mu.Unlock()
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	err = d.Write(ctx, websocket.MessageText, []byte("dial "+id))
	cancel()
	if err != nil {
		return
	}
	var dc *websocket.Conn
	select {
	case dc = <-ch:
	case <-time.After(10 * time.Second):
		r.mu.Lock()
		delete(r.waiting, id)
		r.mu.Unlock()
		b.Close(websocket.StatusTryAgainLater, "the daemon did not answer")
		return
	}
	defer dc.CloseNow()
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	go pipe(ctx, stop, b, dc)
	pipe(ctx, stop, dc, b)
}

func pipe(ctx context.Context, stop func(), from, to *websocket.Conn) {
	defer stop()
	for {
		typ, msg, err := from.Read(ctx)
		if err != nil {
			return
		}
		if err := to.Write(ctx, typ, msg); err != nil {
			return
		}
	}
}
