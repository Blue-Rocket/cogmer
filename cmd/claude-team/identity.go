package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Identity is the stable per-machine peer identity required by spec §6.
//
// MachineID is a display and attribution label only. It is NOT an address and
// nothing may route by it: an OS hostname is frequently not a routable name, and
// is usually not what the transport knows the machine by. On one macOS host this
// returned "macbookpro.lan" while the same machine also answered to "pushover"
// and would carry a third name on a tailnet. Reachability is a transport
// property, discovered rather than derived (§4).
type Identity struct {
	PeerID string `json:"peerId"`
	// PeerName is derived from PeerID, never stored as an independent value --
	// a persisted name could drift from the identity it claims to represent.
	PeerName string `json:"-"`
	// private never appears in identity.json and is never marshalled: whoami
	// prints this struct, and an identity you cannot hand to a colleague without
	// checking what else is in it is not much of an identity.
	private         ed25519.PrivateKey `json:"-"`
	UserID          string             `json:"userId"`
	UserDisplayName string             `json:"userDisplayName"`
	// NameChosen distinguishes a name somebody picked from one this program
	// guessed. The guess is $USER capitalised, which is "David" on a laptop and
	// "Ec2-user" in a container -- and the difference matters because the name is
	// seen only by OTHER people. Without the flag there is no way to tell "David
	// because I meant it" from "David because the OS said so", and comparing
	// against the guess would pester the person whose username really is their
	// name (D-095).
	NameChosen bool   `json:"nameChosen"`
	MachineID  string `json:"machineId"`
}

// Config separates room membership (shareable) from identity (personal), per §28.
type Config struct {
	Room string `json:"room"`
}

// stateDirName is the one place the product name reaches the filesystem. Renaming
// the product is then a one-line change plus a migration, rather than a search.
const stateDirName = ".claude-team"

// homeDir is where identity, membership, rooms and the fetched binary live.
//
// CLAUDE_TEAM_HOME overrides it. That is not only for tests: the plugin's
// installer already honoured the variable while the binary ignored it, so a
// person who set it got a binary in one place and its state in another, and
// nothing said so.
func homeDir() string {
	if v := strings.TrimSpace(os.Getenv("CLAUDE_TEAM_HOME")); v != "" {
		return v
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(h, stateDirName)
}

func randomID() string {
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// LoadIdentity reads ~/.claude-team/identity.json, creating a default on first run.
func LoadIdentity() (*Identity, error) {
	priv, err := loadOrCreateKey()
	if err != nil {
		return nil, err
	}
	derived := PeerIDFromPublic(priv.Public().(ed25519.PublicKey))

	path := filepath.Join(homeDir(), "identity.json")
	data, rerr := os.ReadFile(path)
	if rerr == nil {
		var id Identity
		if err := json.Unmarshal(data, &id); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		// An identifier that is not this key names an identity nothing can verify.
		// Adopt the key's identifier rather than keep a claim we cannot support.
		if id.PeerID != derived {
			id.PeerID = derived
			buf, _ := json.MarshalIndent(id, "", "  ")
			_ = os.WriteFile(path, buf, 0o600)
		}
		id.private = priv
		id.PeerName = PeerName(id.PeerID)
		return &id, nil
	}
	if !os.IsNotExist(rerr) {
		return nil, rerr
	}

	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME") // Windows
	}
	if user == "" {
		// A name a colleague reads on every turn. Neither account variable is
		// set, so we know nothing about them -- say that, rather than guessing
		// an occupation. Matches unknown-machine below: an honest blank.
		user = "someone"
	}
	host, _ := os.Hostname()
	if host == "" {
		host = "unknown-machine"
	}
	id := &Identity{
		PeerID:          derived,
		UserID:          strings.ToLower(user),
		UserDisplayName: strings.ToUpper(user[:1]) + user[1:],
		NameChosen:      false, // guessed, and said so
		MachineID:       host,
	}
	if err := os.MkdirAll(homeDir(), 0o755); err != nil {
		return nil, err
	}
	buf, _ := json.MarshalIndent(id, "", "  ")
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		return nil, err
	}
	id.private = priv
	id.PeerName = PeerName(id.PeerID)
	return id, nil
}

// SetDisplayName records what this person wants other people to call them.
//
// Deliberate even when it changes nothing: somebody who runs this and keeps the
// guessed name has chosen it, and should stop being offered the chance.
//
// Changing it is safe, and that is a consequence rather than a coincidence. Events
// already sent keep the old name, because events are immutable (§7). And the label
// a colleague gave you is theirs (D-094), so your rename cannot alter what anybody
// else calls you -- which is what makes this cosmetic enough to defer and to change.
func SetDisplayName(name string) (*Identity, error) {
	id, err := LoadIdentity()
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name != "" {
		id.UserDisplayName = name
	}
	id.NameChosen = true
	buf, err := json.MarshalIndent(id, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(homeDir(), "identity.json"), buf, 0o600); err != nil {
		return nil, err
	}
	return id, nil
}
