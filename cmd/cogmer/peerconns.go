package main

import (
	"bytes"
	"crypto/x509"
	"net/http"
	"sync"
	"time"
)

// peerConns keeps one HTTP client for each address this machine polls, so that its
// connection is used again by the next poll instead of a new connection and handshake
// every second (D-303, a machine keeps one connection to each address it polls).
//
// A connection's key is checked against this machine's records at the handshake
// (D-101, peer connections are TLS pinned to a key this machine has recorded), so a
// kept connection is checked again before each poll, with the certificate the other
// side last presented on it, against the records as they are then. A client is dropped,
// and the next poll makes a new connection, when that check fails, when this machine's
// own certificate has changed since the client was made, and when the daemon discards a
// connection that is stuck (D-210, the daemon discards a connection that is stuck). A
// connection the other side closed is not kept, since the transport dials again for the
// next request.
type peerConns struct {
	mu sync.Mutex
	m  map[string]*peerConn
}

type peerConn struct {
	client *http.Client
	own    []byte // the leaf of this machine's certificate when the client was made

	mu   sync.Mutex
	peer *x509.Certificate // the leaf the other side last presented
}

// recordingTransport notes on its peerConn what each response's connection presented.
type recordingTransport struct {
	next http.RoundTripper
	pc   *peerConn
}

func (t recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.next.RoundTrip(req)
	if err == nil && resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		t.pc.mu.Lock()
		t.pc.peer = resp.TLS.PeerCertificates[0]
		t.pc.mu.Unlock()
	}
	return resp, err
}

// syncClient is the client to poll addr with: the kept one when it still holds, and a
// new one otherwise.
func (d *Daemon) syncClient(addr string) (*http.Client, error) {
	cert, err := d.tls.certificate(d.id)
	if err != nil {
		return nil, err
	}
	var own []byte
	if len(cert.Certificate) > 0 {
		own = cert.Certificate[0]
	}
	d.peerConns.mu.Lock()
	defer d.peerConns.mu.Unlock()
	if pc := d.peerConns.m[addr]; pc != nil {
		if d.peerConnHolds(pc, own) {
			return pc.client, nil
		}
		pc.client.CloseIdleConnections()
		delete(d.peerConns.m, addr)
	}
	conf, err := d.clientConfig("")
	if err != nil {
		return nil, err
	}
	client, err := clientFor(addr, 30*time.Second, conf)
	if err != nil {
		return nil, err
	}
	pc := &peerConn{client: client, own: own}
	client.Transport = recordingTransport{next: client.Transport, pc: pc}
	if d.peerConns.m == nil {
		d.peerConns.m = map[string]*peerConn{}
	}
	d.peerConns.m[addr] = pc
	return client, nil
}

// peerConnHolds says whether a kept client may be used for another poll: this machine
// presents the certificate it was made with, and the certificate the other side last
// presented would pass the handshake's check now.
func (d *Daemon) peerConnHolds(pc *peerConn, own []byte) bool {
	pc.mu.Lock()
	peer := pc.peer
	pc.mu.Unlock()
	if !bytes.Equal(pc.own, own) {
		return false
	}
	if peer == nil {
		return true // no answer has come on it yet, so no connection is held
	}
	return pinnedVerifier(d.pinTrust(), "")([][]byte{peer.Raw}, nil) == nil
}

// dropSyncClient drops the client kept for addr, so that the next poll makes a new one.
func (d *Daemon) dropSyncClient(addr string) {
	d.peerConns.mu.Lock()
	defer d.peerConns.mu.Unlock()
	if pc := d.peerConns.m[addr]; pc != nil {
		pc.client.CloseIdleConnections()
		delete(d.peerConns.m, addr)
	}
}
