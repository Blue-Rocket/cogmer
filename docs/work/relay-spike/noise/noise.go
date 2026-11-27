// Package noise is the Noise_NK_25519_AESGCM_SHA256 handshake the relay spike runs between a browser
// and a daemon, written against the Noise specification (revision 34) with the standard library only.
// The browser knows the daemon's static key from the link and has none of its own.
package noise

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
)

const ProtocolName = "Noise_NK_25519_AESGCM_SHA256"

type symmetric struct {
	ck, h [32]byte
	k     []byte
	n     uint64
}

func newSymmetric() *symmetric {
	s := &symmetric{}
	copy(s.h[:], ProtocolName) // the name is 28 bytes, so it is padded with zeros
	s.ck = s.h
	s.mixHash(nil) // empty prologue
	return s
}

func (s *symmetric) mixHash(data []byte) {
	d := sha256.New()
	d.Write(s.h[:])
	d.Write(data)
	copy(s.h[:], d.Sum(nil))
}

func hkdf2(ck, ikm []byte) (a, b []byte) {
	m := hmac.New(sha256.New, ck)
	m.Write(ikm)
	temp := m.Sum(nil)
	m = hmac.New(sha256.New, temp)
	m.Write([]byte{1})
	a = m.Sum(nil)
	m = hmac.New(sha256.New, temp)
	m.Write(a)
	m.Write([]byte{2})
	b = m.Sum(nil)
	return
}

func (s *symmetric) mixKey(ikm []byte) {
	ck, k := hkdf2(s.ck[:], ikm)
	copy(s.ck[:], ck)
	s.k, s.n = k, 0
}

func seal(k []byte, n uint64, ad, p []byte) []byte {
	b, _ := aes.NewCipher(k)
	g, _ := cipher.NewGCM(b)
	var nonce [12]byte
	binary.BigEndian.PutUint64(nonce[4:], n)
	return g.Seal(nil, nonce[:], p, ad)
}

func open(k []byte, n uint64, ad, c []byte) ([]byte, error) {
	b, _ := aes.NewCipher(k)
	g, _ := cipher.NewGCM(b)
	var nonce [12]byte
	binary.BigEndian.PutUint64(nonce[4:], n)
	return g.Open(nil, nonce[:], c, ad)
}

func (s *symmetric) encryptAndHash(p []byte) []byte {
	c := seal(s.k, s.n, s.h[:], p)
	s.n++
	s.mixHash(c)
	return c
}

func (s *symmetric) decryptAndHash(c []byte) ([]byte, error) {
	p, err := open(s.k, s.n, s.h[:], c)
	if err != nil {
		return nil, err
	}
	s.n++
	s.mixHash(c)
	return p, nil
}

// CipherState carries one direction of the transport after the handshake.
type CipherState struct {
	k []byte
	n uint64
}

func (c *CipherState) Seal(p []byte) []byte { out := seal(c.k, c.n, nil, p); c.n++; return out }
func (c *CipherState) Open(m []byte) ([]byte, error) {
	p, err := open(c.k, c.n, nil, m)
	if err == nil {
		c.n++
	}
	return p, err
}

func (s *symmetric) split() (*CipherState, *CipherState) {
	a, b := hkdf2(s.ck[:], nil)
	return &CipherState{k: a}, &CipherState{k: b}
}

// Responder answers the first message with the second, given the daemon's static key, and returns the
// cipher states for receiving from and sending to the initiator.
func Responder(static *ecdh.PrivateKey, msg1 []byte) (msg2 []byte, recv, send *CipherState, err error) {
	if len(msg1) != 32+16 {
		return nil, nil, nil, errors.New("first message has the wrong length")
	}
	s := newSymmetric()
	s.mixHash(static.PublicKey().Bytes())
	re, err := ecdh.X25519().NewPublicKey(msg1[:32])
	if err != nil {
		return nil, nil, nil, err
	}
	s.mixHash(msg1[:32])
	dh, err := static.ECDH(re)
	if err != nil {
		return nil, nil, nil, err
	}
	s.mixKey(dh)
	if _, err := s.decryptAndHash(msg1[32:]); err != nil {
		return nil, nil, nil, err
	}
	e, _ := ecdh.X25519().GenerateKey(rand.Reader)
	msg2 = append(msg2, e.PublicKey().Bytes()...)
	s.mixHash(e.PublicKey().Bytes())
	dh, err = e.ECDH(re)
	if err != nil {
		return nil, nil, nil, err
	}
	s.mixKey(dh)
	msg2 = append(msg2, s.encryptAndHash(nil)...)
	c1, c2 := s.split()
	return msg2, c1, c2, nil
}

// Initiator is the browser's side, kept here so the test can run both ends and the JavaScript can be
// checked against it.
type Initiator struct {
	s  *symmetric
	e  *ecdh.PrivateKey
	rs *ecdh.PublicKey
}

func NewInitiator(rs *ecdh.PublicKey) *Initiator {
	return &Initiator{s: newSymmetric(), rs: rs}
}

func (i *Initiator) First() ([]byte, error) {
	i.s.mixHash(i.rs.Bytes())
	i.e, _ = ecdh.X25519().GenerateKey(rand.Reader)
	out := append([]byte(nil), i.e.PublicKey().Bytes()...)
	i.s.mixHash(out)
	dh, err := i.e.ECDH(i.rs)
	if err != nil {
		return nil, err
	}
	i.s.mixKey(dh)
	return append(out, i.s.encryptAndHash(nil)...), nil
}

func (i *Initiator) Second(msg2 []byte) (send, recv *CipherState, err error) {
	if len(msg2) != 32+16 {
		return nil, nil, errors.New("second message has the wrong length")
	}
	re, err := ecdh.X25519().NewPublicKey(msg2[:32])
	if err != nil {
		return nil, nil, err
	}
	i.s.mixHash(msg2[:32])
	dh, err := i.e.ECDH(re)
	if err != nil {
		return nil, nil, err
	}
	i.s.mixKey(dh)
	if _, err := i.s.decryptAndHash(msg2[32:]); err != nil {
		return nil, nil, err
	}
	c1, c2 := i.s.split()
	return c1, c2, nil
}
