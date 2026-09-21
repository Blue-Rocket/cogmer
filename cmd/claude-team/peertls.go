package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// --- confidentiality between peers, pinned to the key we already verified (D-101) ---
//
// Events are signed, which is integrity and origin. Signing is not secrecy, and room
// content is a developer's prompts and whatever their Claude said back. So every peer
// connection is TLS 1.3, over every transport, including the one that is already
// encrypted -- uniform, so that confidentiality never depends on which path a dial
// happened to take.
//
// There is no PKI here and nothing to manage. The burden people mean by
// "certificates" is a certificate authority, issuance, trust stores, expiry and
// revocation, and pinning removes all of it: a `peerId` IS an Ed25519 public key
// (D-042), already confirmed to be that person's by the two-word comparison (D-055).
// The certificate is a container for a key we hold, and verification is one
// comparison against it. Dates and chains are never consulted.
//
// The alternative -- deriving a key exchange from the same Ed25519 keys ourselves --
// would not avoid certificates so much as replace a reviewed TLS 1.3 implementation
// with a handshake, nonce discipline, replay window and rekeying of our own. The
// primitives would be sound and the composition is where this goes wrong silently.

// peerCert builds the self-signed certificate this daemon presents, both as a server
// and as a client. Held in memory: it carries no secret the identity file does not
// already hold, and regenerating it on restart costs nothing because nobody pins the
// certificate -- they pin the key inside it.
func peerCert(id *Identity) (tls.Certificate, error) {
	pub, ok := id.private.Public().(ed25519.PublicKey)
	if !ok {
		return tls.Certificate{}, errors.New("identity key is not ed25519")
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: id.PeerID},
		// Set because the format requires them, and never read: a pinned key is
		// the whole check, so an expiry would only be a way for this to stop
		// working for a reason unrelated to identity.
		NotBefore:   time.Now().Add(-time.Hour),
		NotAfter:    time.Now().AddDate(100, 0, 0),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pub, id.private)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: id.private}, nil
}

// pinnedVerifier checks that whoever is on the other end holds a key this machine
// knows, and — when the caller knows who it dialled — that it is the expected one.
//
// `expect` is empty for a server, which cannot know who is calling until they
// present something, and for a client dialling a room address, because a room
// records where its members listen rather than which member listens there. Identity
// is still bound: the signed request inside the connection says who sent it (D-044).
// What this adds is that the binding now happens before a body is read.
func pinnedVerifier(knows func(string) bool, expect string) func([][]byte, [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return errors.New("no certificate presented, so there is nothing to pin to")
		}
		cert, err := x509.ParseCertificate(rawCerts[0])
		if err != nil {
			return fmt.Errorf("unreadable certificate: %w", err)
		}
		pub, ok := cert.PublicKey.(ed25519.PublicKey)
		if !ok {
			return errors.New("certificate does not carry an ed25519 key, so it names no peer")
		}
		got := PeerIDFromPublic(pub)
		if expect != "" && got != expect {
			return fmt.Errorf("expected %s and reached %s: the address answers for a different key",
				PeerName(expect), PeerName(got))
		}
		if knows != nil && !knows(got) {
			return fmt.Errorf("%s is not a peer this machine knows", PeerName(got))
		}
		return nil
	}
}

// peerTLS caches the certificate, which is derived from a key that never changes.
type peerTLS struct {
	once sync.Once
	cert tls.Certificate
	err  error
}

func (p *peerTLS) certificate(id *Identity) (tls.Certificate, error) {
	p.once.Do(func() { p.cert, p.err = peerCert(id) })
	return p.cert, p.err
}

// serverConfig is what the peer listeners present.
//
// A client certificate is REQUIRED, not requested: an anonymous peer has nothing to
// pin and no business here, and refusing at the handshake is cheaper and clearer
// than refusing after a body has been read.
func (d *Daemon) serverConfig() (*tls.Config, error) {
	cert, err := d.tls.certificate(d.id)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates:          []tls.Certificate{cert},
		MinVersion:            tls.VersionTLS13,
		ClientAuth:            tls.RequireAnyClientCert,
		VerifyPeerCertificate: pinnedVerifier(d.members.Knows, ""),
	}, nil
}

// clientConfig is what this daemon presents when dialling a peer.
//
// InsecureSkipVerify disables the checks that assume a certificate authority —
// chain, hostname, dates — none of which exist here. It does not disable
// VerifyPeerCertificate, which is the check that matters and is stricter than what
// it replaces: the key must be one this machine knows.
func (d *Daemon) clientConfig(expect string) (*tls.Config, error) {
	cert, err := d.tls.certificate(d.id)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates:          []tls.Certificate{cert},
		MinVersion:            tls.VersionTLS13,
		InsecureSkipVerify:    true,
		VerifyPeerCertificate: pinnedVerifier(d.members.Knows, expect),
	}, nil
}
