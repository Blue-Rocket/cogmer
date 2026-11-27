package noise

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"testing"
)

func TestTheTwoSidesAgreeAndAWrongKeyFails(t *testing.T) {
	static, _ := ecdh.X25519().GenerateKey(rand.Reader)
	in := NewInitiator(static.PublicKey())
	m1, err := in.First()
	if err != nil {
		t.Fatal(err)
	}
	m2, rRecv, rSend, err := Responder(static, m1)
	if err != nil {
		t.Fatal(err)
	}
	iSend, iRecv, err := in.Second(m2)
	if err != nil {
		t.Fatal(err)
	}
	p, err := rRecv.Open(iSend.Seal([]byte("token\n")))
	if err != nil || !bytes.Equal(p, []byte("token\n")) {
		t.Fatalf("initiator to responder: %q %v", p, err)
	}
	p, err = iRecv.Open(rSend.Seal([]byte("200\n")))
	if err != nil || !bytes.Equal(p, []byte("200\n")) {
		t.Fatalf("responder to initiator: %q %v", p, err)
	}

	other, _ := ecdh.X25519().GenerateKey(rand.Reader)
	wrong := NewInitiator(other.PublicKey())
	m1, _ = wrong.First()
	if _, _, _, err := Responder(static, m1); err == nil {
		t.Fatal("a first message made for another key was accepted")
	}
}
