package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

// --- trusting a machine because its chain ends at a user this machine knows ---
//
// A colleague is a user, and a user is a master key that certifies machines through
// a shared key (D-219, a user is a master key that certifies machines through a
// shared key), so a colleague verified once is verified for every machine of
// theirs. Each check below takes the key that signed something, the chain that came
// with it, and this machine's records of the user the chain ends at, and answers the
// one question the caller asks: whether the key speaks for that user now.
//
// Every path here is taken only when a chain is presented. A peer that presents none
// is judged by its own key, as it always was, so a machine with no user identity and
// every colleague of it behave as they did before the user identity existed.

// UserState resolves what this machine holds for user, with the records of extra
// added when it is given, so that a chain that came with an event or a request is
// judged together with whatever this machine already knows of its user: a removal
// held here still removes the machine the chain certifies (D-220, a removal names the
// last event of a machine that counts), and a shared key newer than the chain's still
// suspends it (D-234, a machine not certified under the current shared key is
// suspended).
func (m *Membership) UserState(user string, extra *Chain) *UserState {
	rs, _, _ := m.IdentityRecords(user)
	if extra != nil {
		rs.Shared = append(rs.Shared, extra.Shared)
		rs.Machines = append(rs.Machines, extra.Machine)
	}
	return Resolve(user, rs)
}

// StoreChain keeps the records of a chain that has verified, so that a later check
// of the user's machines has them without the chain (D-239, the certificates travel
// with events, and a store keeps each once). A record already held is left as it was.
func (m *Membership) StoreChain(c Chain) error {
	put := func(kind string, v any) error {
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		_, _, err = m.PutIdentityRecord(kind, c.Shared.User, raw)
		return err
	}
	if err := put(RecordShared, c.Shared); err != nil {
		return err
	}
	if err := put(RecordMachine, c.Machine); err != nil {
		return err
	}
	if c.Session != nil {
		return put(RecordSession, *c.Session)
	}
	return nil
}

// RemovalsSeen is the hex hashes of every valid removal this machine holds for
// user, sorted, which a v4 event names so that a receiver lacking one knows to fetch
// it (D-221, an event whose identity version the receiver lacks is withheld until
// fetched).
func (m *Membership) RemovalsSeen(user string) []string {
	st := m.UserState(user, nil)
	var out []string
	for h := range st.Removals {
		out = append(out, hex.EncodeToString(h[:]))
	}
	sort.Strings(out)
	return out
}

// checkLiveChain is the check for a live act, which is a connection, a sync request
// or a verification step: the chain verifies to user and ends at signer, and the
// machine is neither removed nor suspended in what this machine holds for the user,
// and a session certificate has not expired (D-101, peer connections are TLS pinned
// to a key this machine has recorded, and refuse a removed machine).
func (m *Membership) checkLiveChain(user, signer string, c Chain, now time.Time) (ChainLink, error) {
	if c.Shared.User != user {
		return ChainLink{}, errors.New("the chain ends at another user")
	}
	link, err := m.UserState(user, &c).Check(c, signer)
	if err != nil {
		return ChainLink{}, err
	}
	if err := link.LiveAt(now); err != nil {
		return ChainLink{}, err
	}
	return link, nil
}

// judgeEvent decides whether a v4 event that has verified counts here: its chain
// is checked against what this machine holds for its user (D-224, a conflict between
// identity records goes to the higher level, then a removal, then the lowest hash), and its machine must count for the
// event's room and sequence, so that a removed machine's events past its cutoff and
// every event of a suspended machine are not stored (D-220, a removal names the last
// event of a machine that counts; D-234, a suspended machine's turns appear nowhere). The chain's
// records are kept when they are new, so that the next check has them.
//
// It is called with d.mu held, since it runs in the loop that writes the room.
func (d *Daemon) judgeEvent(ev *Event) error {
	if ev.Chain == nil {
		return errors.New("an event signed for a user carries no chain")
	}
	st := d.members.UserState(ev.PeerID, ev.Chain)
	link, err := st.CheckEvent(*ev.Chain, ev.Signer)
	if err != nil {
		return err
	}
	if err := d.members.StoreChain(*ev.Chain); err != nil {
		return fmt.Errorf("keeping the chain: %w", err)
	}
	if !st.Counts(link.Machine, ev.RoomID, ev.PeerSequence) {
		return errNotCounting
	}
	return nil
}

// errNotCounting is an event whose machine is removed past its cutoff in the
// event's room, or suspended. Such an event is not stored, and not stored is all it
// needs: sync is a pull, so it is never marked held and stays on offer, and a
// suspension lifted later brings it on the next poll (D-234, a suspended machine is
// lifted by certifying it again).
var errNotCounting = errors.New("the machine that signed it is removed past its cutoff or suspended")

// isSelf reports whether an identifier is this machine's own peer: its user, when it
// has one, or its own key. A user's own events from another of the user's machines
// are the user's own (D-232, a user's own turn from another session is injected as a
// turn of theirs), and need no verification of anybody.
func (d *Daemon) isSelf(peer string) bool {
	return peer == d.id.Peer() || peer == d.id.PeerID
}

// authorVerified is the verification gate by user (D-054, verification gates
// synchronization and injection): an event or a request
// passes when its peer is this machine's own user or a colleague verified here,
// whichever machine of theirs it came from.
func (d *Daemon) authorVerified(peer string) bool {
	return d.isSelf(peer) || d.members.IsVerified(peer)
}
