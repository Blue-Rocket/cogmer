package main

import (
	"encoding/json"
	"errors"
	"log"
	"sort"
	"strings"
	"time"
)

// --- a user's lists, copied between the user's machines (D-228) ---
//
// A user's colleagues, their verifications and labels, and the user's room list
// belong to the user, so every machine of the user holds a copy (D-228, a user's
// colleagues, verifications, labels and rooms belong to the user and are copied
// between the user's machines). Each change to an entry is a signed ListChange whose
// value is the entry as the change leaves it, with a counter one more than the
// highest this machine has seen for the entry; MergeListChanges settles each entry on
// the highest counter, then the lowest hash, so every machine holding the same
// changes reaches the same value with no clock.
//
// The changes are kept in identity_records with the user's other records, and the
// tables a machine reads on its hot paths (known_peers, offers, room_machines) are
// made to follow the merged lists whenever new changes arrive. A machine with no user
// identity writes no change, and behaves as it did before lists were copied.

const (
	entryColleague = "colleague:"
	entryRoom      = "room:"
)

// colleagueEntry is a colleague as a change leaves them: the label the user gave
// them, when the user verified them, whether the user forgot them, and the addresses
// this machine had for them, so that another machine of the user can reach a
// colleague it never paired with.
type colleagueEntry struct {
	Label     string   `json:"label,omitempty"`
	Verified  string   `json:"verified,omitempty"`
	Forgotten bool     `json:"forgotten,omitempty"`
	Endpoints []string `json:"endpoints,omitempty"`
}

// roomEntry is a room on the user's room list: its name, the user's machines from
// which a session has joined it, and, for a room that was offered and not yet joined,
// who offered it and where they listen (D-105, an offered room enters the user's room
// list as offered, so the user can join from any machine). It holds nothing of the
// room's guests or turns, which stay on the machines that joined (D-228).
type roomEntry struct {
	Name      string   `json:"name"`
	Machines  []string `json:"machines,omitempty"`
	OfferedBy string   `json:"offeredBy,omitempty"`
	Endpoint  string   `json:"endpoint,omitempty"`
}

// trustedListChanges is every change to user's lists signed by a machine the user's
// records certify and do not remove. ListChange says which key signed it, and only
// the user's records say whether that key speaks for the user.
func (m *Membership) trustedListChanges(user string) []ListChange {
	rs, changes, err := m.IdentityRecords(user)
	if err != nil {
		return nil
	}
	st := Resolve(user, rs)
	var out []ListChange
	for _, c := range changes {
		if ms, ok := st.Machines[c.Signer]; ok && !ms.Removed {
			out = append(out, c)
		}
	}
	return out
}

// UserLists is the user's lists as every machine holding the same changes settles
// them, by entry.
func (m *Membership) UserLists(user string) map[string]ListChange {
	return MergeListChanges(user, m.trustedListChanges(user))
}

// listCounter is the highest counter this machine has seen for an entry, from any
// signer, so that its next change is one more and outranks every change it has seen.
func (m *Membership) listCounter(user, entry string) uint64 {
	_, changes, err := m.IdentityRecords(user)
	if err != nil {
		return 0
	}
	var top uint64
	for _, c := range changes {
		if c.Entry == entry && c.Counter > top {
			top = c.Counter
		}
	}
	return top
}

// writeListChange signs and keeps one change to an entry of the user's lists. The
// machine's other machines receive it at their next exchange of lists (D-222, a
// machine sends its records to every machine of the user until each has
// acknowledged).
func (m *Membership) writeListChange(id *Identity, entry string, v any) error {
	if !id.HasUser() || id.private == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c := SignListChange(id.private, id.User, entry, string(raw), m.listCounter(id.User, entry)+1)
	body, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, _, err = m.PutIdentityRecord(RecordList, id.User, body)
	return err
}

// RecordColleague writes the change that leaves a colleague as this machine now holds
// them: labeled, verified, or forgotten. Called after pairing, labeling, verifying or
// forgetting, so that the user's other machines know the colleague the same way
// (D-228; D-073, forgetting a peer on one machine forgets them on every machine).
func (m *Membership) RecordColleague(id *Identity, peer string) error {
	v := colleagueEntry{Forgotten: true}
	if m.Knows(peer) {
		v = colleagueEntry{Label: m.Label(peer), Verified: m.VerifiedAt(peer), Endpoints: m.PeerEndpointsOf(peer)}
	}
	return m.writeListChange(id, entryColleague+peer, v)
}

// roomEntryOf is a room as the user's lists hold it, and whether they hold it.
func (m *Membership) roomEntryOf(user, roomID string) (roomEntry, bool) {
	c, ok := m.UserLists(user)[entryRoom+roomID]
	if !ok {
		return roomEntry{}, false
	}
	var v roomEntry
	if json.Unmarshal([]byte(c.Value), &v) != nil {
		return roomEntry{}, false
	}
	return v, true
}

// RecordRoomOnList writes the change that puts a room on the user's room list, or
// changes it there: change edits the entry as the lists hold it now.
func (m *Membership) RecordRoomOnList(id *Identity, roomID, name string, change func(*roomEntry)) error {
	if !id.HasUser() {
		return nil
	}
	v, _ := m.roomEntryOf(id.User, roomID)
	if name != "" {
		v.Name = name
	}
	if change != nil {
		change(&v)
	}
	v.Machines = uniqueSorted(v.Machines)
	return m.writeListChange(id, entryRoom+roomID, v)
}

// withMachine and withoutMachine are the changes for a machine joining and leaving.
func withMachine(machine string) func(*roomEntry) {
	return func(v *roomEntry) {
		v.Machines = append(v.Machines, machine)
		v.OfferedBy, v.Endpoint = "", ""
	}
}

func withoutMachine(machine string) func(*roomEntry) {
	return func(v *roomEntry) {
		var keep []string
		for _, k := range v.Machines {
			if k != machine {
				keep = append(keep, k)
			}
		}
		v.Machines = keep
	}
}

func uniqueSorted(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

// RoomOnList finds a room on the user's room list by its name or identifier, so a
// session on any of the user's machines can join it by name (D-228). A name two rooms
// share finds neither, as FindRoom refuses an ambiguous name.
func (m *Membership) RoomOnList(user, nameOrID string) (string, roomEntry, bool) {
	var hits []string
	var found roomEntry
	for entry, c := range m.UserLists(user) {
		if !strings.HasPrefix(entry, entryRoom) {
			continue
		}
		var v roomEntry
		if json.Unmarshal([]byte(c.Value), &v) != nil {
			continue
		}
		id := strings.TrimPrefix(entry, entryRoom)
		if id == nameOrID || v.Name == nameOrID {
			hits = append(hits, id)
			found = v
		}
	}
	if len(hits) != 1 {
		return "", roomEntry{}, false
	}
	return hits[0], found, true
}

// JoinFromList records here a room found on the user's room list, so that this
// machine fetches it. The user's machines that joined it are where its events are
// fetched from (D-228, its machine then fetches the room's guest list and events from
// a machine that has joined), and until the room's guest list arrives among them the
// user is recorded as its guest, which makes every machine of the user a place to ask
// and lets a machine of the user that has joined answer. A room offered and not yet
// joined by any of the user's machines is fetched from the machine that offered it,
// as an offer accepted here would be.
func (m *Membership) JoinFromList(id *Identity, roomID string, v roomEntry, showOnly bool) error {
	if err := m.RecordRoom(roomID, v.Name, showOnly); err != nil {
		return err
	}
	if err := m.Invite(roomID, id.User); err != nil {
		return err
	}
	for _, k := range v.Machines {
		if k == id.PeerID {
			continue
		}
		if err := m.RecordRoomMachine(roomID, k, id.User); err != nil {
			return err
		}
	}
	if v.OfferedBy != "" {
		if !m.Knows(v.OfferedBy) {
			if err := m.Allow(v.OfferedBy, ""); err != nil {
				return err
			}
		}
		if err := m.SetPeerEndpoint(v.OfferedBy, v.Endpoint); err != nil {
			return err
		}
		if err := m.Invite(roomID, v.OfferedBy); err != nil {
			return err
		}
		if err := m.RecordRoomMachine(roomID, v.OfferedBy, v.OfferedBy); err != nil {
			return err
		}
	}
	return nil
}

// ApplyUserLists makes this machine's tables follow the user's lists as they have
// settled: each colleague known, labeled, verified or forgotten as the lists say, an
// offered room waiting here as an offer, and each room joined here reaching the user's
// machines that joined it. A machine that joined a room the settled list does not say
// it joined, which happens when two machines joined at once and one change won, says
// so again with a change of its own.
func (m *Membership) ApplyUserLists(id *Identity) error {
	if !id.HasUser() {
		return nil
	}
	var errs []error
	for entry, c := range m.UserLists(id.User) {
		switch {
		case strings.HasPrefix(entry, entryColleague):
			peer := strings.TrimPrefix(entry, entryColleague)
			if _, err := PublicFromPeerID(peer); err != nil || peer == id.User {
				continue
			}
			var v colleagueEntry
			if json.Unmarshal([]byte(c.Value), &v) != nil {
				continue
			}
			if err := m.applyColleague(peer, v); err != nil {
				errs = append(errs, err)
			}
		case strings.HasPrefix(entry, entryRoom):
			var v roomEntry
			if json.Unmarshal([]byte(c.Value), &v) != nil {
				continue
			}
			if err := m.applyRoom(id, strings.TrimPrefix(entry, entryRoom), v); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func (m *Membership) applyColleague(peer string, v colleagueEntry) error {
	if v.Forgotten {
		if m.Knows(peer) {
			return m.Forget(peer)
		}
		return nil
	}
	if !m.Knows(peer) || m.Label(peer) != v.Label {
		if err := m.Allow(peer, v.Label); err != nil {
			// A label this machine already gives another key is the alarm D-074
			// describes, and is left for the user to see rather than resolved here.
			var taken *NameTakenError
			if !errors.As(err, &taken) {
				return err
			}
			log.Printf("lists: %q names another key on this machine, so %s keeps the name it has here", v.Label, PeerName(peer))
			if !m.Knows(peer) {
				if err := m.Allow(peer, ""); err != nil {
					return err
				}
			}
		}
	}
	if (v.Verified != "") != m.IsVerified(peer) {
		if err := m.setVerifiedAt(peer, v.Verified); err != nil {
			return err
		}
	}
	if len(m.PeerEndpointsOf(peer)) == 0 {
		for _, e := range v.Endpoints {
			if err := m.SetPeerEndpoint(peer, e); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Membership) applyRoom(id *Identity, roomID string, v roomEntry) error {
	if _, joined := m.RoomByID(roomID); !joined {
		if v.OfferedBy != "" && len(v.Machines) == 0 {
			if _, ok := m.FindOffer(roomID); !ok {
				return m.RecordOffer(Offer{RoomID: roomID, RoomName: v.Name, Host: v.OfferedBy, Endpoint: v.Endpoint})
			}
		}
		return nil
	}
	in := false
	for _, k := range v.Machines {
		if k == id.PeerID {
			in = true
			continue
		}
		if err := m.RecordRoomMachine(roomID, k, id.User); err != nil {
			return err
		}
	}
	if !in && m.SessionsInRoom(roomID) > 0 {
		return m.RecordRoomOnList(id, roomID, v.Name, withMachine(id.PeerID))
	}
	return nil
}

// setVerifiedAt records a verification made on another of the user's machines, at
// the time it was made there, or clears one the lists say no longer stands.
func (m *Membership) setVerifiedAt(peer, at string) error {
	var val any
	if at != "" {
		if _, err := time.Parse(time.RFC3339, at); err != nil {
			return nil
		}
		val = at
	}
	_, err := m.db.Exec(`UPDATE known_peers SET verified_at = ? WHERE peer_id = ?`, val, peer)
	return err
}
