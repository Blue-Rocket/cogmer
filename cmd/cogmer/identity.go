package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	// guessed. The guess is $USER capitalized, which is "David" on a laptop and
	// "Ec2-user" in a container -- and the difference matters because the name is
	// seen only by OTHER people. Without the flag there is no way to tell "David
	// because I meant it" from "David because the OS said so", and comparing
	// against the guess would pester the person whose username really is their
	// name (D-095).
	NameChosen bool   `json:"nameChosen"`
	MachineID  string `json:"machineId"`
	// User is the identifier of this machine's user, the master public key, and
	// Chain the certificates by which this machine speaks for the user (D-219, a
	// user is a master key that certifies machines through a shared key). Both are
	// empty until the user identity is made on this machine. Every part of them is
	// public, so they belong in this file, which whoami prints; the shared key's
	// private half is in shared.key and never here (D-153, the private key is kept
	// in a file of its own).
	User  string `json:"user,omitempty"`
	Chain *Chain `json:"chain,omitempty"`
}

// HasUser reports whether this machine speaks for a user: identity.json holds the
// user and the chain that ties this machine's key to the user's master key (D-219,
// a user is a master key that certifies machines through a shared key). Every path
// the user identity adds is taken only when this is true, so a machine without one
// behaves as it did before the user identity existed.
func (id *Identity) HasUser() bool { return id != nil && id.User != "" && id.Chain != nil }

// Peer is the identifier this machine is known by to colleagues: its user's when it
// has one, and otherwise its own key's, which is what PeerID always holds. PeerID
// stays the machine key, since the key is what signs (D-239, an event a machine
// signs for its user names the user and the signing key).
func (id *Identity) Peer() string {
	if id.HasUser() {
		return id.User
	}
	return id.PeerID
}

// Config separates room membership (shareable) from identity (personal), per §28.
type Config struct {
	Room string `json:"room"`
}

// stateDirName is the one place the product name reaches the filesystem, so a
// rename is one line rather than a search.
const stateDirName = ".cogmer"

// homeDir is where identity, membership, rooms and the fetched binary live.
//
// COGMER_HOME overrides it. That is not only for tests: the plugin's
// installer already honored the variable while the binary ignored it, so a
// person who set it got a binary in one place and its state in another, and
// nothing said so.
func homeDir() string {
	dir := resolvedHomeDir()
	if err := refusedHome(dir); err != nil {
		// A development build never reaches here from main, which refuses first
		// (guardRealHome). In a test, a refusal is a test that forgot to name a
		// directory of its own, and failing it loudly is what protects the real one.
		panic(err)
	}
	return dir
}

// resolvedHomeDir is the state directory homeDir names, before the check that a
// development build may use it.
func resolvedHomeDir() string {
	if v := strings.TrimSpace(os.Getenv("COGMER_HOME")); v != "" {
		return v
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(h, stateDirName)
}

// ensureStateDir creates the state directory with mode 0700 and tightens one that
// exists with group or other bits, so the databases, logs and keys inside it are not
// readable by other accounts whatever the umask was when something first made it. A
// state directory made by a shell script under a umask of 022, or by an older program,
// is 0755, and nothing but this call would change that. Windows has no such bits, so
// it only creates the directory there.
func ensureStateDir() error { return ensureDir(homeDir()) }

// ensureRoomsDir does the same for the folder of room databases.
func ensureRoomsDir() error {
	if err := ensureStateDir(); err != nil {
		return err
	}
	return ensureDir(filepath.Join(homeDir(), "rooms"))
}

func ensureDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		return nil
	}
	fi, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if fi.Mode().Perm()&0o077 != 0 {
		return os.Chmod(dir, 0o700)
	}
	return nil
}

func randomID() string {
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// LoadIdentity reads ~/.cogmer/identity.json, creating a default on first run.
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
		// A cloud session signs with its session key, kept in session.key, and the
		// file names that key as the end of its chain (D-216, a cloud session signs with
		// a key of its own). The daemon writes both; until it has written a key that
		// matches, the session speaks for nobody, and the file is left for the daemon.
		if id.Chain != nil && id.Chain.Session != nil {
			id.private = priv
			if sk, ok := loadSessionKeyIn(homeDir()); ok && PeerIDFromPublic(sk.Public().(ed25519.PublicKey)) == id.Chain.Session.Session {
				id.private = sk
				id.PeerID = id.Chain.Session.Session
			} else {
				id.PeerID, id.User, id.Chain = derived, "", nil
			}
			id.PeerName = PeerName(id.PeerID)
			return &id, nil
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
	if err := ensureStateDir(); err != nil {
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
