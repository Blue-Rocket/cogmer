package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// --- confidentiality between peers, pinned to the key we already verified (D-101, peer connections are TLS pinned) ---
//
// Events are signed, which is integrity and origin. Signing is not secrecy, and room
// content is a person's prompts and whatever their assistant said back. So every peer
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
	if id.HasUser() {
		raw, err := json.Marshal(id.Chain)
		if err != nil {
			return tls.Certificate{}, err
		}
		// Not critical: a peer on a build that does not read the extension pins
		// the key, as it always did, and ignores the rest.
		tmpl.ExtraExtensions = []pkix.Extension{{Id: chainExtensionOID, Critical: false, Value: raw}}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pub, id.private)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: id.private}, nil
}

// chainExtensionOID names the certificate extension that carries a machine's chain to
// its user (D-101, peer connections are TLS pinned to a key this machine has
// recorded, whose certificates end at a recorded master key). It sits under 2.25,
// the arc of identifiers made from a random UUID (ITU-T X.667), which needs no
// registration. Go's certificate parser refuses an arc above 31 bits, and a
// certificate it cannot parse fails the handshake, so the arc is the top 31 bits of a
// UUID generated for this purpose (f5b07499-3e7e-4c94-b8fd-db678a821587) rather than
// all 128. That is not a UUID identifier in X.667's sense, and nothing else is known
// to use it. It never changes, since a certificate made by one build is read by
// another.
var chainExtensionOID = asn1.ObjectIdentifier{2, 25, 2060991052}

// certChain reads the chain a certificate carries, or nil when it carries none.
func certChain(cert *x509.Certificate) (*Chain, error) {
	for _, ext := range cert.Extensions {
		if ext.Id.Equal(chainExtensionOID) {
			var c Chain
			if err := json.Unmarshal(ext.Value, &c); err != nil {
				return nil, fmt.Errorf("the certificate's chain does not parse: %w", err)
			}
			return &c, nil
		}
	}
	return nil, nil
}

// pinTrust is what a pin is checked against. knows is whether this machine has
// recorded a peer, by its identifier. self is this machine's own user, or empty when
// it has none. live checks a chain presented by a machine for its user against this
// machine's records of the user, refusing a removed or suspended machine, and is
// nil where nothing can judge one, which then refuses every chain.
type pinTrust struct {
	knows func(string) bool
	self  string
	live  func(user, signer string, c Chain) error
}

// pinnedVerifier checks that whoever is on the other end holds a key this machine
// knows, and — when the caller knows who it dialled — that it is the expected one.
//
// A key is known in one of two ways. Its own identifier is a recorded peer, which is
// a machine with no user identity, or one recorded before it had one. Or the
// certificate carries a chain from the key to a user, and that user is a recorded
// peer or this machine's own user, so that a colleague verified once is reached on
// every machine of theirs and the user's own machines reach each other (D-101,
// D-219, a user is a master key that certifies machines through a shared key). A
// chain that does not verify, or whose machine is removed or suspended, refuses the
// connection, whichever way the key would otherwise be known (D-234, a suspended
// machine's connections are refused).
//
// `expect` is empty for a server, which cannot know who is calling until they
// present something, and for a client dialling a room address, because a room
// records where its members listen rather than which member listens there. Identity
// is still bound: the signed request inside the connection says who sent it (D-044, sync requests are signed).
// What this adds is that the binding now happens before a body is read.
func pinnedVerifier(trust pinTrust, expect string) func([][]byte, [][]*x509.Certificate) error {
	knows := trust.knows
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
		chain, err := certChain(cert)
		if err != nil {
			return err
		}
		if chain != nil {
			return pinChain(trust, expect, got, *chain)
		}
		if expect != "" && got != expect {
			return fmt.Errorf("expected %s and reached %s: the address answers for a different key",
				PeerName(expect), PeerName(got))
		}
		// RECORDED, not verified, and the difference is the whole ceremony.
		// Verification is how a recorded peer becomes a verified one, and it
		// happens over this connection — so requiring verification here would
		// make it permanently unreachable and nobody could ever be verified at
		// all. Tightening this to IsVerified looks like hardening and is a
		// deadlock. What keeps an unverified peer harmless is D-054: its events
		// are refused at the layer above, and no transcript crosses.
		if knows != nil && !knows(got) {
			return fmt.Errorf("%s is not a peer this machine knows", PeerName(got))
		}
		return nil
	}
}

// pinChain is the pin for a certificate that carries a chain. The chain must end at
// the key in the certificate, the user it names must be known here or be this
// machine's own, the machine must be neither removed nor suspended, and a named peer
// must be the user or the key itself.
func pinChain(trust pinTrust, expect, got string, c Chain) error {
	user := c.Shared.User
	if _, err := c.Verify(user, got); err != nil {
		return fmt.Errorf("%s presents a chain that does not verify: %w", PeerName(got), err)
	}
	if expect != "" && user != expect && got != expect {
		return fmt.Errorf("expected %s and reached a machine of %s: the address answers for a different user",
			PeerName(expect), PeerName(user))
	}
	known := (trust.self != "" && user == trust.self) || (trust.knows != nil && (trust.knows(user) || trust.knows(got)))
	if !known {
		return fmt.Errorf("%s speaks for %s, who is not a peer this machine knows", PeerName(got), PeerName(user))
	}
	if trust.live == nil {
		return errors.New("nothing here can check a chain")
	}
	if err := trust.live(user, got, c); err != nil {
		return fmt.Errorf("%s may not connect: %w", PeerName(got), err)
	}
	return nil
}

// pinTrust is what this daemon pins against: its known peers, its own user, and its
// records of each user.
func (d *Daemon) pinTrust() pinTrust {
	self := ""
	if d.id.HasUser() {
		self = d.id.User
	}
	return pinTrust{
		knows: d.members.Knows,
		self:  self,
		live: func(user, signer string, c Chain) error {
			_, err := d.members.checkLiveChain(user, signer, c, time.Now())
			return err
		},
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
		VerifyPeerCertificate: pinnedVerifier(d.pinTrust(), ""),
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
		VerifyPeerCertificate: pinnedVerifier(d.pinTrust(), expect),
	}, nil
}
