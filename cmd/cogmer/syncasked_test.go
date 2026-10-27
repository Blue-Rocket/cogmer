package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"
)

// askedLog is what one listening address was sent: the identifiers of the rooms the
// requests named.
type askedLog struct {
	mu    sync.Mutex
	rooms []string
}

func (a *askedLog) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req syncRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		a.mu.Lock()
		a.rooms = append(a.rooms, req.RoomID)
		a.mu.Unlock()
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
	})
}

func (a *askedLog) got() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.rooms...)
}

// A pull from an address names only the rooms whose guest listens there, so a peer is never
// told the identifier of a room it is not in. The choice is made by the code a real pull
// runs, and what is asserted is what each listening server received (D-103).
func TestAPullNamesToEachListeningAddressOnlyTheRoomsItsOwnerIsAGuestOf(t *testing.T) {
	t.Setenv("COGMER_PEERS", "")
	m := testMembership(t)
	self := testIdentity(t)
	if err := m.Allow(self.PeerID, "self"); err != nil {
		t.Fatal(err)
	}
	d := &Daemon{id: self, members: m, stores: map[string]*Store{}}
	t.Cleanup(d.closeStores)
	roomA, _ := m.CreateRoom(self.PeerID, false)
	roomB, _ := m.CreateRoom(self.PeerID, false)

	asked := map[string]*askedLog{}
	guestOf := map[string][]string{"alice": {roomA.RoomID}, "bob": {roomB.RoomID}}
	for _, name := range []string{"alice", "bob", "owl"} {
		srv := testIdentity(t)
		log := &askedLog{}
		asked[name] = log
		addr := servePeerTLSHandler(t, &Daemon{id: srv, members: m}, log.handler())
		if err := m.Allow(srv.PeerID, name); err != nil {
			t.Fatal(err)
		}
		if err := m.SetPeerEndpoint(srv.PeerID, addr); err != nil {
			t.Fatal(err)
		}
		for _, r := range guestOf[name] {
			if err := m.Invite(r, srv.PeerID); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, addr := range d.syncTargets() {
		if _, err := d.pullFrom(addr); err != nil {
			t.Fatalf("pull from %s: %v", addr, err)
		}
	}
	for name, want := range map[string][]string{"alice": {roomA.RoomID}, "bob": {roomB.RoomID}, "owl": nil} {
		got := asked[name].got()
		if len(got) != len(want) || (len(want) == 1 && got[0] != want[0]) {
			t.Errorf("%s was sent requests naming %v, want %v", name, got, want)
		}
	}
}

// Every guest's server holds each request until the test ends, standing for an address that
// waits out its timeout, so a round that asked the addresses one after another would reach
// the first and never the rest. Whatever order the guests come in, each must be asked while
// the others wait.
func TestEveryGuestIsAskedWhileTheOthersWaitOnTheirTimeouts(t *testing.T) {
	t.Setenv("COGMER_PEERS", "")
	m := testMembership(t)
	self := testIdentity(t)
	if err := m.Allow(self.PeerID, "self"); err != nil {
		t.Fatal(err)
	}
	room, _ := m.CreateRoom(self.PeerID, false)
	release := make(chan struct{})
	defer close(release)
	d := &Daemon{id: self, members: m, stores: map[string]*Store{}}
	t.Cleanup(d.closeStores)
	asked := &askedLog{}
	for _, name := range []string{"one", "two", "three", "four"} {
		srv := testIdentity(t)
		if err := m.Allow(srv.PeerID, name); err != nil {
			t.Fatal(err)
		}
		if err := m.Invite(room.RoomID, srv.PeerID); err != nil {
			t.Fatal(err)
		}
		addr := servePeerTLSHandler(t, &Daemon{id: srv, members: m}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			asked.mu.Lock()
			asked.rooms = append(asked.rooms, name)
			asked.mu.Unlock()
			<-release
		}))
		if err := m.SetPeerEndpoint(srv.PeerID, addr); err != nil {
			t.Fatal(err)
		}
	}
	go d.RunSync(nil, 20*time.Millisecond)
	deadline := time.Now().Add(3 * time.Second)
	for len(asked.got()) < 4 {
		if time.Now().After(deadline) {
			t.Fatalf("only %v were asked while their peers waited, want all four", asked.got())
		}
		time.Sleep(10 * time.Millisecond)
	}
}
