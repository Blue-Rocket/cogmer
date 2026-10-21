package main

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
)

// A peer has three names (§6): the label its user gave it, the derived name computed
// from its key, and the display name it asserts for itself. The peer record holds the
// first, or the derived name when no label was given. The display name travels on
// every event, so the latest one seen in any room is what the list can say it calls
// itself.

// reportedNames returns, for each peer, the display name on its latest event in any
// room on this machine. A peer with no event, or with no name on it, has no entry.
// The rooms are opened read-only, and a room that cannot be read is skipped, since
// the list must answer without them.
func reportedNames(peers []string) map[string]string {
	out := map[string]string{}
	at := map[string]string{}
	files, _ := filepath.Glob(filepath.Join(homeDir(), "rooms", "*.db"))
	for _, file := range files {
		db, err := sql.Open("sqlite", "file:"+file+"?mode=ro&_pragma=busy_timeout(2000)")
		if err != nil {
			continue
		}
		for _, peer := range peers {
			var ts, name string
			err := db.QueryRow(`SELECT timestamp, user_display_name FROM events
				WHERE peer_id = ? AND COALESCE(user_display_name, '') <> ''
				ORDER BY peer_sequence DESC LIMIT 1`, peer).Scan(&ts, &name)
			if err != nil {
				continue
			}
			if name = sanitizeName(name); name != "" && ts > at[peer] {
				out[peer], at[peer] = name, ts
			}
		}
		db.Close()
	}
	return out
}

// formatPeer is a peer's entry in the list: the name it leads with and how it stands,
// then each of its names in words that say what it is, and its key. It leads with the
// label, or with the derived name when no label was given.
func formatPeer(p KnownPeer, reported string) string {
	derived := PeerName(p.PeerID)
	label := p.Name
	if label == derived {
		label = "" // a placeholder offered as a choice would be a lie (D-094)
	}
	state := "verified " + cut10(p.VerifiedAt)
	if p.VerifiedAt == "" {
		state = "not finished"
	}
	given, claim := label, "\""+reported+"\" (their own claim)"
	if given == "" {
		given = "none given"
	}
	if reported == "" {
		claim = "not seen yet"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "  %s  %s\n", firstNonEmpty(label, derived), state)
	fmt.Fprintf(&b, "      what you call them:         %s\n", given)
	fmt.Fprintf(&b, "      derived from their key:     %s\n", derived)
	fmt.Fprintf(&b, "      what they call themselves:  %s\n", claim)
	fmt.Fprintf(&b, "      key:                        %s\n", p.PeerID)
	return b.String()
}
