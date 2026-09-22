package main

import "testing"

// Verifying one peer must not dial another. The sweep this replaced handed
// RunVerification every address the daemon knew, so pairing with Alice opened a
// connection to Bob -- and since D-103 put every address in known_peers, Bob's
// could never have answered anyway: clientConfig pins the dial to Alice's key.
//
// The assertion is that Bob's address is ABSENT, not that Alice's is present. A
// test that only checked for Alice would pass unchanged if the sweep came back.
func TestVerifyDialsOnlyTheNamedPeer(t *testing.T) {
	m := testMembership(t)
	alice, bob := testIdentity(t), testIdentity(t)
	if alice.PeerID == bob.PeerID {
		t.Fatal("two identities collided; the rest of this test proves nothing")
	}
	for _, p := range []struct{ id, name, addr string }{
		{alice.PeerID, "alice", "alice.example:4783"},
		{bob.PeerID, "bob", "bob.example:4783"},
	} {
		if err := m.Allow(p.id, p.name); err != nil {
			t.Fatal(err)
		}
		if err := m.SetPeerEndpoint(p.id, p.addr); err != nil {
			t.Fatal(err)
		}
	}
	d := &Daemon{id: testIdentity(t), members: m}

	got := d.verifyTargets(alice.PeerID)
	for _, a := range got {
		if a == "bob.example:4783" {
			t.Fatalf("verifying alice dialled bob: %v", got)
		}
	}
	if len(got) != 1 || got[0] != "alice.example:4783" {
		t.Fatalf("want only alice's own address, got %v", got)
	}
}

// COGMER_PEERS survives the narrowing because those addresses name no peer,
// so one of them may be the peer we want. Anything attributed to somebody else
// does not.
func TestVerifyKeepsUnattributedConfiguredAddresses(t *testing.T) {
	m := testMembership(t)
	alice := testIdentity(t)
	if err := m.Allow(alice.PeerID, "alice"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COGMER_PEERS", "configured.example:4783")
	d := &Daemon{id: testIdentity(t), members: m}

	got := d.verifyTargets(alice.PeerID)
	if len(got) != 1 || got[0] != "configured.example:4783" {
		t.Fatalf("want the configured address when the peer has none, got %v", got)
	}
}
