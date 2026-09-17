package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The separation is the security property, not a convention. A hook endpoint
// reachable from another machine is equivalent to that machine being the local
// developer: it can publish into the room and read the conversation back.
func mustDaemon(t *testing.T) *Daemon {
	t.Helper()
	d, _ := testDaemon(t)
	return d
}

func TestPeerListenerDoesNotServeHooks(t *testing.T) {
	d := mustDaemon(t)
	peer := d.PeerRoutes()

	for _, path := range []string{"/hook/prompt", "/hook/stop", "/events"} {
		rec := httptest.NewRecorder()
		peer.ServeHTTP(rec, httptest.NewRequest("POST", path, strings.NewReader("{}")))
		if rec.Code != 404 {
			t.Errorf("peer listener serves %s (status %d); it must not", path, rec.Code)
		}
	}
}

// Conversely, synchronization is not offered on loopback, so the two surfaces
// cannot drift into being the same thing by accident.
func TestLocalListenerDoesNotServeSync(t *testing.T) {
	d := mustDaemon(t)
	rec := httptest.NewRecorder()
	d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest("POST", "/sync", strings.NewReader("{}")))
	if rec.Code != 404 {
		t.Errorf("local listener serves /sync (status %d)", rec.Code)
	}
}

func TestBothListenersReportHealth(t *testing.T) {
	d := mustDaemon(t)
	for name, mux := range map[string]*http.ServeMux{
		"local": d.LocalRoutes(), "peer": d.PeerRoutes(),
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
		if rec.Code != 200 {
			t.Errorf("%s listener health = %d", name, rec.Code)
		}
	}
}

func TestLoopbackDetection(t *testing.T) {
	for addr, want := range map[string]bool{
		"127.0.0.1:4783": true,
		"localhost:4783": true,
		"[::1]:4783":     true,
		"0.0.0.0:4783":   false,
		"192.168.1.5:80": false,
	} {
		if got := isLoopback(addr); got != want {
			t.Errorf("isLoopback(%q) = %v, want %v", addr, got, want)
		}
	}
}
