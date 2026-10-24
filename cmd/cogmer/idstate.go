package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
)

// What a user's identity records add up to. Every receiver holding the same
// records for a user computes the same state from them, whatever order they
// arrived in, with no clock and no server, because nothing here reads anything
// but the records (D-224, a conflict between identity records goes to the higher
// level, then to a removal, then to the lowest hash). That is what lets records
// travel from any peer in any order and still settle (D-222, a machine tells
// every machine it knows of each machine it adds).
//
// A record's hash is used here only once the record has verified. A record under
// a scheme this build does not know hashes to zero, which is the lowest hash and
// would win every tie it entered.

// Claim is what D-224 needs to know of a record to rank it against another.
type Claim struct {
	Level   Level
	Removal bool
	Hash    [32]byte
}

// Prevails reports whether a wins over b: the higher level, then a removal over
// an addition, then the lower hash. The hash is the last resort because it is
// the one thing every receiver sees alike and no signer can choose usefully, so
// two identical claims prevail over neither.
func Prevails(a, b Claim) bool {
	if a.Level != b.Level {
		return a.Level > b.Level
	}
	if a.Removal != b.Removal {
		return a.Removal
	}
	return bytes.Compare(a.Hash[:], b.Hash[:]) < 0
}

// RecordSet is every identity record a receiver holds for one user.
type RecordSet struct {
	Shared   []SharedKeyCert `json:"shared"`
	Machines []MachineCert   `json:"machines"`
	Removals []Removal       `json:"removals"`
}

type MachineState struct {
	Cert    MachineCert
	Current bool // certified under the user's current shared version
	Removed bool
	// Cutoffs maps a room to the machine's last counting sequence there, and is
	// set only for a removed machine. A room absent counts nothing.
	Cutoffs map[string]int64
}

// Discarded is a record that failed a check or lost a conflict, kept with its
// reason so that the room page can show it and a test can name it.
type Discarded struct {
	Hash   [32]byte
	Kind   string // "shared", "machine" or "removal"
	Reason string
}

type UserState struct {
	User       string
	Current    uint64            // 0 when no shared-key certificate is valid
	SharedKeys map[uint64]string // the winning shared key of each version
	Machines   map[string]*MachineState
	Removals   map[[32]byte]Removal // every valid removal, by hash
	Discarded  []Discarded
}

// shortHash names a record in a reason. Eight bytes tell records apart for a
// person reading the room page; the full hash is in Discarded.Hash.
func shortHash(h [32]byte) string { return hex.EncodeToString(h[:8]) }

// Resolve computes the state of user's identity from rs. A record that does not
// verify yet may verify once another arrives, such as a machine certificate
// whose shared-key certificate is still in transit, so the caller reruns Resolve
// over the whole set whenever it grows rather than carrying a verdict forward.
func Resolve(user string, rs RecordSet) *UserState {
	s := &UserState{
		User:       user,
		SharedKeys: map[uint64]string{},
		Machines:   map[string]*MachineState{},
		Removals:   map[[32]byte]Removal{},
	}
	discard := func(h [32]byte, kind, reason string) {
		s.Discarded = append(s.Discarded, Discarded{Hash: h, Kind: kind, Reason: reason})
	}

	// 1. Shared-key certificates. Two for one version with different keys are
	// both from the master, so D-224 comes down to the lowest hash.
	sharedByVersion := map[uint64]map[[32]byte]SharedKeyCert{}
	for _, c := range rs.Shared {
		if c.User != user {
			discard(c.Hash(), "shared", "certificate is for another user")
			continue
		}
		if err := c.Verify(); err != nil {
			discard(c.Hash(), "shared", err.Error())
			continue
		}
		if sharedByVersion[c.Version] == nil {
			sharedByVersion[c.Version] = map[[32]byte]SharedKeyCert{}
		}
		keepLeastSigned(sharedByVersion[c.Version], c.Hash(), c, func(c SharedKeyCert) string { return c.Signature })
	}
	for v, certs := range sharedByVersion {
		win := bestHash(certs, func([32]byte) Claim { return Claim{Level: LevelMaster} })
		s.SharedKeys[v] = certs[win].SharedKey
		if v > s.Current {
			s.Current = v
		}
		for h := range certs {
			if h != win {
				discard(h, "shared", fmt.Sprintf(
					"lost to shared-key certificate %s for version %d under D-224 (a conflict goes to the higher level, then a removal, then the lowest hash)",
					shortHash(win), v))
			}
		}
	}

	// 2. Machine certificates, each checked against the winning shared key of
	// its version, so one signed by a key that lost in step 1 fails here.
	certsByMachine := map[string]map[[32]byte]MachineCert{}
	for _, c := range rs.Machines {
		if c.User != user {
			discard(c.Hash(), "machine", "certificate is for another user")
			continue
		}
		key, ok := s.SharedKeys[c.SharedVersion]
		if !ok {
			discard(c.Hash(), "machine", fmt.Sprintf("no valid shared-key certificate for version %d", c.SharedVersion))
			continue
		}
		pub, _ := PublicFromPeerID(key) // a winning key has verified, so it parses
		if err := c.VerifyBy(pub); err != nil {
			discard(c.Hash(), "machine", fmt.Sprintf("not signed by the winning shared key of version %d: %v",
				c.SharedVersion, err))
			continue
		}
		if certsByMachine[c.Machine] == nil {
			certsByMachine[c.Machine] = map[[32]byte]MachineCert{}
		}
		keepLeastSigned(certsByMachine[c.Machine], c.Hash(), c, func(c MachineCert) string { return c.Signature })
	}
	for machine, certs := range certsByMachine {
		// The highest version is the user's latest word on the machine. Under one
		// version every certificate is a shared-level addition, so D-224 decides
		// by hash alone.
		var win [32]byte
		found := false
		for h, c := range certs {
			if !found {
				win, found = h, true
				continue
			}
			w := certs[win]
			if c.SharedVersion > w.SharedVersion ||
				(c.SharedVersion == w.SharedVersion && Prevails(Claim{Level: LevelShared, Hash: h}, Claim{Level: LevelShared, Hash: win})) {
				win = h
			}
		}
		cert := certs[win]
		s.Machines[machine] = &MachineState{Cert: cert, Current: cert.SharedVersion == s.Current}
		for h := range certs {
			if h != win {
				discard(h, "machine", fmt.Sprintf("superseded by certificate %s for the same machine", shortHash(win)))
			}
		}
	}

	// 3. Removals. The signature is checked before the signer's authority, so a
	// removal under a scheme this build does not know is reported as needing an
	// upgrade rather than as an unauthorized signer.
	for _, r := range rs.Removals {
		h := r.Hash()
		if r.User != user {
			discard(h, "removal", "removal is for another user")
			continue
		}
		pub, err := PublicFromPeerID(r.Signer)
		if err != nil {
			discard(h, "removal", fmt.Sprintf("signer: %v", err))
			continue
		}
		if err := r.VerifyBy(pub); err != nil {
			discard(h, "removal", err.Error())
			continue
		}
		if err := s.mayRemove(r); err != nil {
			discard(h, "removal", err.Error())
			continue
		}
		s.Removals[h] = r
	}

	// 4. Every valid removal takes effect. A removal and the certificate it
	// removes are not in conflict, since the removal presupposes the certificate
	// it ends; ranking them by level would keep every machine that removed itself
	// (D-220, a removal names the last event of a machine that counts). A removal
	// of a machine with no certificate here stays in Removals, so the machine is
	// removed when its certificate arrives and Resolve runs again.
	for _, r := range s.Removals {
		m, ok := s.Machines[r.Machine]
		if !ok || m.Removed {
			continue
		}
		m.Removed = true
		m.Cutoffs = removalCutoffs(r.Machine, s.Removals)
	}

	sort.Slice(s.Discarded, func(i, j int) bool {
		a, b := s.Discarded[i], s.Discarded[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if c := bytes.Compare(a.Hash[:], b.Hash[:]); c != 0 {
			return c < 0
		}
		return a.Reason < b.Reason
	})
	return s
}

// mayRemove decides whether a verified removal's signer has the authority it
// claims. The master may remove any machine, and so may the shared key of the
// current version. A removal signed by an older shared key does not count, so
// that a thief holding a stolen machine's old shared key cannot mark the user's
// events for good: certifying a new shared key ends what the old one can do, and
// the machine that certifies it signs again, under the new key, the removals it
// holds (D-220, a removal names the last event of a machine that counts). A
// machine or cloud key may remove only itself, of any version, and only once
// certified, which is also what leaves a session key, which has no machine
// certificate, unable to remove anything.
func (s *UserState) mayRemove(r Removal) error {
	switch r.SignerLevel {
	case LevelMaster:
		if r.Signer != s.User {
			return errors.New("removal at the master level is signed by a key other than the user's master key")
		}
	case LevelShared:
		if key, ok := s.SharedKeys[r.SignerVersion]; !ok || key != r.Signer {
			return fmt.Errorf("removal at the shared level is not signed by the winning shared key of version %d",
				r.SignerVersion)
		}
		if r.SignerVersion != s.Current {
			return fmt.Errorf("removal is signed by the shared key of version %d, and only the current version %d may remove",
				r.SignerVersion, s.Current)
		}
	case LevelMachine:
		if r.Signer != r.Machine {
			return errors.New("a machine may remove only itself")
		}
		if _, ok := s.Machines[r.Signer]; !ok {
			return errors.New("removal is signed by a key with no valid machine certificate")
		}
	default:
		return fmt.Errorf("removal signer level %d is not a level", r.SignerLevel)
	}
	return nil
}

// removalCutoffs gives each room the lowest sequence any removal of machine
// allows there, a room a removal leaves out counting as 0, so that a cutoff only
// moves earlier whoever signs next (D-220, a removal names the last event of a
// machine that counts). A room whose lowest is 0 is left out, which reads the
// same as a cutoff of 0.
func removalCutoffs(machine string, removals map[[32]byte]Removal) map[string]int64 {
	var mine []Removal
	rooms := map[string]bool{}
	for _, r := range removals {
		if r.Machine != machine {
			continue
		}
		mine = append(mine, r)
		for _, c := range r.Cutoffs {
			rooms[c.Room] = true
		}
	}
	out := map[string]int64{}
	for room := range rooms {
		low := int64(-1)
		for _, r := range mine {
			var seq int64
			for _, c := range r.Cutoffs {
				if c.Room == room {
					seq = c.Sequence
				}
			}
			if low < 0 || seq < low {
				low = seq
			}
		}
		if low > 0 {
			out[room] = low
		}
	}
	return out
}

// keepLeastSigned stores a verified record under its hash. Two copies with one
// hash have the same signed fields, and if their signature strings differ, the
// lesser is kept so that the state does not depend on which copy came first.
func keepLeastSigned[T any](m map[[32]byte]T, h [32]byte, c T, sig func(T) string) {
	if old, ok := m[h]; ok && sig(old) <= sig(c) {
		return
	}
	m[h] = c
}

// bestHash returns the hash whose claim prevails over every other in m. Prevails
// is a total order on distinct hashes, so the result does not depend on the
// order the map yields them in.
func bestHash[T any](m map[[32]byte]T, claim func([32]byte) Claim) [32]byte {
	var win [32]byte
	found := false
	for h := range m {
		if !found {
			win, found = h, true
			continue
		}
		c, w := claim(h), claim(win)
		c.Hash, w.Hash = h, win
		if Prevails(c, w) {
			win = h
		}
	}
	return win
}

// Counts reports whether machine's event with this sequence in room counts. An
// unknown machine's event does not, since nothing yet says the user vouches for
// it.
func (s *UserState) Counts(machine, room string, sequence int64) bool {
	m, ok := s.Machines[machine]
	if !ok {
		return false
	}
	// A machine not certified under the current shared key is suspended: none
	// of its events count until the user certifies it again, which only the
	// user can do once a new key exists, so a machine a thief added with the old
	// key stays out of every transcript for good.
	if !m.Current {
		return false
	}
	if !m.Removed {
		return true
	}
	cutoff, ok := m.Cutoffs[room]
	return ok && sequence <= cutoff
}

// List gives the certificates of the machines not removed, the list a machine
// sends the user's other machines (D-222, a machine tells every machine it knows
// of each machine it adds). It is sorted by serial, the number shown beside a
// machine's name, then by key, since two machines can share a serial.
func (s *UserState) List() []MachineCert {
	var out []MachineCert
	for _, m := range s.Machines {
		if !m.Removed {
			out = append(out, m.Cert)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Serial != out[j].Serial {
			return out[i].Serial < out[j].Serial
		}
		return out[i].Machine < out[j].Machine
	})
	return out
}

// Check verifies a chain that came with something signer signed, and refuses it
// when this state knows better: its shared-key certificate lost its version, or
// its machine has been removed. A chain under a version this state does not hold
// yet, or for a machine it has not heard of, is accepted, because the chain
// carries its own proof; whether to hold the event until the records arrive is
// Lacks's question (D-221, an event whose identity version the receiver lacks is
// withheld until fetched). A removal held for a machine whose certificate is not
// in the state still refuses the chain, since adding the chain's certificate to
// the records would make the machine removed. A machine-level removal cannot be
// such a removal, as it needs the certificate to be valid at all.
//
// Check is for a live act: a connection, a sync request or an admission proof.
// An event of a removed machine still counts up to its cutoff, so an event is
// judged by Counts after its chain verifies, and never refused by Check for a
// removal (D-220, a removal names the last event of a machine that counts).
//
// A machine certified only under an older version than this state's current
// one is suspended and refused, whatever chain it presents, until the user
// certifies it under the current key. A chain under a version this state does
// not hold yet is accepted, since it is newer than what this state knows.
//
// Check reads no clock: a live act also calls LiveAt on the link.
func (s *UserState) Check(c Chain, signer string) (ChainLink, error) {
	link, err := c.Verify(s.User, signer)
	if err != nil {
		return ChainLink{}, err
	}
	if key, ok := s.SharedKeys[c.Shared.Version]; ok && key != c.Shared.SharedKey {
		return ChainLink{}, fmt.Errorf("the chain's shared-key certificate for version %d lost to another under D-224 (a conflict goes to the higher level, then a removal, then the lowest hash)",
			c.Shared.Version)
	}
	if link.SharedVersion < s.Current {
		if m, ok := s.Machines[link.Machine]; !ok || !m.Current {
			return ChainLink{}, fmt.Errorf("machine %.24s is not certified under the current shared key, version %d, so it is suspended",
				link.Machine, s.Current)
		}
	}
	// Every removed machine has a removal here, so this also covers
	// MachineState.Removed.
	for _, r := range s.Removals {
		if r.Machine == link.Machine {
			return ChainLink{}, fmt.Errorf("machine %.24s has been removed", link.Machine)
		}
	}
	return link, nil
}

// Lacks reports whether an event naming this shared version and these removals
// names records this state does not hold, which is the test for withholding it
// from the assistant until they are fetched (D-221, an event whose identity
// version the receiver lacks is withheld until fetched).
func (s *UserState) Lacks(version uint64, removals [][32]byte) bool {
	if version > s.Current {
		return true
	}
	for _, h := range removals {
		if _, ok := s.Removals[h]; !ok {
			return true
		}
	}
	return false
}
