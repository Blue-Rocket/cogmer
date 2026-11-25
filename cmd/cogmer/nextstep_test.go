package main

import (
	"strings"
	"testing"
)

// One line says what a user does next on the way to a first shared room, from the state this
// machine can read, and says nothing once they are there.
func TestTheNextStepFollowsHowFarAUserHasGot(t *testing.T) {
	for name, c := range map[string]struct {
		j    journey
		want string // a phrase the line holds, or "" for no line
	}{
		"no identity yet, which the command that needs one handles": {journey{}, ""},
		"no peer":                 {journey{HasUser: true}, "/cogmer:peer-pair <their string>"},
		"a peer not verified":     {journey{HasUser: true, Peers: 1, Unverified: "Bob"}, "finish pairing with Bob"},
		"a peer, no room":         {journey{HasUser: true, Peers: 1, Verified: 1}, "/cogmer:room-create"},
		"a room of one":           {journey{HasUser: true, Peers: 1, Verified: 1, InRoom: true}, "/cogmer:room-invite"},
		"a room with a colleague": {journey{HasUser: true, Peers: 1, Verified: 1, InRoom: true, Others: true}, ""},
	} {
		got := nextStep(c.j)
		if c.want == "" && got != "" {
			t.Errorf("%s: a line was given: %q", name, got)
		}
		if c.want != "" && !strings.Contains(got, c.want) {
			t.Errorf("%s: %q does not hold %q", name, got, c.want)
		}
	}
}

func TestTheNextStepIsReadFromThisMachinesState(t *testing.T) {
	m := testMembership(t)
	id := testIdentity(t)
	bob := testIdentity(t)
	t.Setenv("COGMER_IDENTITY_PAGE", "off")
	t.Setenv("COGMER_HOST", "claude")
	t.Setenv("CLAUDE_CODE_SESSION_ID", "s")

	if got := nextStepLine(m, id); !strings.Contains(got, "pair with a colleague") {
		t.Errorf("a machine with no peer: %q", got)
	}
	if err := m.Allow(bob.PeerID, "Bob"); err != nil {
		t.Fatal(err)
	}
	if got := nextStepLine(m, id); !strings.Contains(got, "finish pairing with Bob") {
		t.Errorf("a peer whose words were not compared: %q", got)
	}
	if err := m.MarkVerified(bob.PeerID); err != nil {
		t.Fatal(err)
	}
	if got := nextStepLine(m, id); !strings.Contains(got, "/cogmer:room-create") {
		t.Errorf("a verified peer and no room: %q", got)
	}
	room, err := m.CreateRoom(id.PeerID, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.BindSession("s", room.RoomID); err != nil {
		t.Fatal(err)
	}
	if got := nextStepLine(m, id); !strings.Contains(got, "/cogmer:room-invite") {
		t.Errorf("a room of the user alone: %q", got)
	}
	if err := m.Invite(room.RoomID, bob.PeerID); err != nil {
		t.Fatal(err)
	}
	if got := nextStepLine(m, id); got != "" {
		t.Errorf("a room with a colleague in it was told a next step: %q", got)
	}
}

// A machine with no user identity is told by the command that needs one, so no step is named.
func TestNoNextStepIsNamedBeforeTheUserIdentityExists(t *testing.T) {
	m := testMembership(t)
	t.Setenv("COGMER_IDENTITY_PAGE", "")
	if got := nextStepLine(m, testIdentity(t)); got != "" {
		t.Errorf("a machine with no user identity was told %q", got)
	}
}
