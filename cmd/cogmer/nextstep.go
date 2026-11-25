package main

import "strings"

// journey is how far a user is from a first shared room, as far as this machine can tell.
type journey struct {
	HasUser    bool
	Peers      int    // peers this machine knows
	Verified   int    // of them, those whose two words were compared
	Unverified string // the name of one that was not, when none was
	InRoom     bool   // this session is in a room
	Others     bool   // somebody besides this user may enter that room
}

// nextStep is the one line that says what a user does next on the way to a first shared room, or
// nothing once they are there. It is the same line from every command that shows it, so that a
// user told what to do next by one command is not told something else by another. A user with no
// identity is told by the command that needs one, which opens the page that makes it.
func nextStep(j journey) string {
	switch {
	case !j.HasUser:
		return ""
	case j.Peers == 0:
		return "Next step: pair with a colleague. They send you their string, you send them yours (it is in " +
			"/cogmer:self-status), and you run /cogmer:peer-pair <their string> <what you call them>."
	case j.Verified == 0:
		return "Next step: finish pairing with " + j.Unverified + ". Run /cogmer:peer-pair with their string " +
			"again, and compare the two words with them on a call."
	case !j.InRoom:
		return "Next step: put this session in a room with /cogmer:room-create, or with /cogmer:room-join " +
			"<invitation> if a colleague invited you."
	case !j.Others:
		return "Next step: invite a colleague you have paired with, with /cogmer:room-invite <name>. They join " +
			"with /cogmer:room-join."
	}
	return ""
}

// nextStepLine is nextStep for this machine and this session, in the host's way of naming a
// command, or "" when there is nothing to say or the state cannot be read. A cloud container
// has none: its session reaches its user's colleagues and rooms through the user's own
// machines, so the steps are taken on one of those and its own state says nothing of them.
func nextStepLine(m *Membership, id *Identity) string {
	if inCloudContainer() || id == nil || !(id.HasUser() || userIdentityGateOff()) {
		return ""
	}
	j := journey{HasUser: true}
	known, err := m.KnownPeers()
	if err != nil {
		return ""
	}
	j.Peers = len(known)
	for _, p := range known {
		if p.VerifiedAt != "" {
			j.Verified++
		} else if j.Unverified == "" {
			j.Unverified = p.Name
		}
	}
	if room, ok := m.RoomForSession(sessionID()); ok {
		j.InRoom = true
		guests, err := m.Guests(room.RoomID)
		if err != nil {
			return ""
		}
		// The list holds the founder, as a user or as a machine's key.
		for _, g := range guests {
			if g != id.Peer() && g != id.PeerID {
				j.Others = true
			}
		}
	}
	return strings.TrimSpace(commandText(nextStep(j)))
}
