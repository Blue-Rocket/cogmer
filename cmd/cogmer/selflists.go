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
// between the user's machines). An entry is one field of a colleague or a room, such
// as a colleague's label or whether one machine of the user has joined a room, so
// that changes to different fields made on different machines at the same time all
// survive, where an entry holding a whole colleague would let one change overwrite
// the other. Each change to an entry is a signed ListChange whose value is the
// field's new value, with a counter one more than the highest this machine has seen
// for the entry; MergeListChanges settles each entry on the highest counter, then the
// lowest hash, so every machine holding the same changes reaches the same value with
// no clock. A colleague or a room is rebuilt from its fields when the lists are read.
//
// The changes are kept in identity_records with the user's other records, and the
// tables a machine reads on its hot paths (known_peers, offers, room_machines) are
// made to follow the merged lists whenever new changes arrive. A machine with no user
// identity writes no change, and behaves as it did before lists were copied.

// The entries of the user's lists. A colleague's are colleague:<id>:label,
// colleague:<id>:verified, colleague:<id>:forgotten and colleague:<id>:endpoints; a
// room's are room:<id>:name, room:<id>:offer and room:<id>:machine:<machine-key>.
// A peer identifier holds a colon (ed25519:...) and a room identifier does not, so
// a colleague's entry splits at its last colon, since no field of a colleague holds
// one, and a room's at the first colon after room:, since a machine's field does.
const (
	entryColleague = "colleague:"
	entryRoom      = "room:"

	fieldLabel     = "label"
	fieldVerified  = "verified"
	fieldForgotten = "forgotten"
	fieldEndpoints = "endpoints"
	fieldName      = "name"
	fieldOffer     = "offer"
	fieldMachine   = "machine:"
)

// colleagueEntry is a colleague as their fields leave them: the label the user gave
// them, when the user verified them, whether the user forgot them, and the addresses
// a machine of the user had for them, so that another machine of the user can reach a
// colleague it never paired with.
type colleagueEntry struct {
	Label     string
	Verified  string
	Forgotten bool
	Endpoints []string
}

// roomEntry is a room on the user's room list: its name, the user's machines from
// which a session has joined it, and, for a room that was offered and not yet joined,
// who offered it and where they listen (D-105, an offered room enters the user's room
// list as offered, so the user can join from any machine). It holds nothing of the
// room's guests or turns, which stay on the machines that joined (D-228).
type roomEntry struct {
	Name      string
	Machines  []string
	OfferedBy string
	Endpoint  string
}

// roomOffer is the value of a room's offer field; both empty is no offer.
type roomOffer struct {
	OfferedBy string `json:"offeredBy,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
}

// userLists is the user's lists rebuilt from their settled fields.
type userLists struct {
	colleagues map[string]colleagueEntry
	rooms      map[string]roomEntry
}

// splitEntry parses an entry into its kind, the colleague or room it belongs to, and
// the field. An entry in another form, such as one written by a build that held a
// whole colleague in one entry, gives ok false and is ignored.
func splitEntry(entry string) (kind, id, field string, ok bool) {
	if rest, found := strings.CutPrefix(entry, entryColleague); found {
		i := strings.LastIndex(rest, ":")
		if i <= 0 || i == len(rest)-1 {
			return "", "", "", false
		}
		return entryColleague, rest[:i], rest[i+1:], true
	}
	if rest, found := strings.CutPrefix(entry, entryRoom); found {
		id, field, ok = strings.Cut(rest, ":")
		return entryRoom, id, field, ok && id != "" && field != ""
	}
	return "", "", "", false
}

// rebuildLists makes each colleague and room from its settled fields. A value that
// does not parse leaves its field as if it had never been written.
func rebuildLists(merged map[string]ListChange) userLists {
	l := userLists{colleagues: map[string]colleagueEntry{}, rooms: map[string]roomEntry{}}
	for entry, c := range merged {
		kind, id, field, ok := splitEntry(entry)
		if !ok {
			continue
		}
		switch kind {
		case entryColleague:
			v := l.colleagues[id]
			switch field {
			case fieldLabel:
				// A label another machine wrote is held to the rule a name typed here
				// is, since it becomes this machine's name for the colleague.
				var label string
				json.Unmarshal([]byte(c.Value), &label)
				v.Label = sanitizeName(label)
			case fieldVerified:
				json.Unmarshal([]byte(c.Value), &v.Verified)
			case fieldForgotten:
				json.Unmarshal([]byte(c.Value), &v.Forgotten)
			case fieldEndpoints:
				json.Unmarshal([]byte(c.Value), &v.Endpoints)
			default:
				continue
			}
			l.colleagues[id] = v
		case entryRoom:
			v := l.rooms[id]
			switch {
			case field == fieldName:
				// A name another machine wrote is held to the form a room's name
				// takes, and one of any other form leaves the field unwritten
				// (D-134, a room's name is generated from two word lists).
				var name string
				json.Unmarshal([]byte(c.Value), &name)
				v.Name = sanitizeRoomName(name)
			case field == fieldOffer:
				var o roomOffer
				if json.Unmarshal([]byte(c.Value), &o) == nil {
					v.OfferedBy, v.Endpoint = o.OfferedBy, o.Endpoint
				}
			case strings.HasPrefix(field, fieldMachine):
				var joined bool
				if json.Unmarshal([]byte(c.Value), &joined) == nil && joined {
					v.Machines = uniqueSorted(append(v.Machines, strings.TrimPrefix(field, fieldMachine)))
				}
			default:
				continue
			}
			l.rooms[id] = v
		}
	}
	return l
}

// trustedListChanges is every change to user's lists signed by a machine the user's
// records certify under the current shared key and do not remove. ListChange says
// which key signed it, and only the user's records say whether that key speaks for
// the user; a suspended machine speaks for nobody until it is certified again
// (D-234, a machine not certified under the current shared key is suspended). A cloud
// key writes no change, since a cloud session takes the user's lists from the user's
// machines and gives them none (D-245, a cloud session takes its user's lists only
// from one of the user's machines).
func (m *Membership) trustedListChanges(user string) []ListChange {
	rs, changes, err := m.IdentityRecords(user)
	if err != nil {
		return nil
	}
	st := Resolve(user, rs)
	var out []ListChange
	for _, c := range changes {
		if ms, ok := st.Machines[c.Signer]; ok && ms.Current && !ms.Removed && ms.Cert.Kind == KindMachine {
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

// lists is the user's lists rebuilt into colleagues and rooms.
func (m *Membership) lists(user string) userLists {
	return rebuildLists(m.UserLists(user))
}

// listCounter is the highest counter of a trusted change this machine holds for an
// entry, so that its next change is one more and outranks every change that counts.
// A change from a machine that speaks for nobody, or one at or above maxListCounter,
// counts for nothing, and counting it would let it push this machine's counters
// toward the bound.
func (m *Membership) listCounter(user, entry string) uint64 {
	var top uint64
	for _, c := range m.trustedListChanges(user) {
		if c.Entry == entry && c.Counter < maxListCounter && c.Counter > top {
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
	// A cloud session writes no change to the user's lists, since no machine takes one
	// from it (D-245).
	if !id.HasUser() || id.private == nil || id.Chain.Session != nil {
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

// writeFieldIfChanged writes a change to one field only when the value differs from
// what the settled lists hold for it, so that a machine recording a verification
// leaves untouched a label another machine changed at the same time. A field the
// lists do not hold reads as its zero value, which is what rebuildLists gives it.
func (m *Membership) writeFieldIfChanged(id *Identity, entry string, was, now any) error {
	a, err := json.Marshal(was)
	if err != nil {
		return err
	}
	b, err := json.Marshal(now)
	if err != nil {
		return err
	}
	if string(a) == string(b) {
		return nil
	}
	return m.writeListChange(id, entry, now)
}

// RecordColleague writes a change for each field of a colleague that this machine now
// holds differently from the user's lists: labeled, verified, forgotten, or paired
// again, which writes forgotten false. Called after pairing, labeling, verifying or
// forgetting, so that the user's other machines know the colleague the same way
// (D-228; D-073, forgetting a peer on one machine forgets them on every machine).
// Forgetting writes only the forgotten field, so a later pairing on any machine
// undoes it under the counter.
func (m *Membership) RecordColleague(id *Identity, peer string) error {
	if !id.HasUser() {
		return nil
	}
	was := m.lists(id.User).colleagues[peer]
	pre := entryColleague + peer + ":"
	if !m.Knows(peer) {
		return m.writeFieldIfChanged(id, pre+fieldForgotten, was.Forgotten, true)
	}
	now := colleagueEntry{Label: m.Label(peer), Verified: m.VerifiedAt(peer), Endpoints: m.PeerEndpointsOf(peer)}
	return errors.Join(
		m.writeFieldIfChanged(id, pre+fieldForgotten, was.Forgotten, false),
		m.writeFieldIfChanged(id, pre+fieldLabel, was.Label, now.Label),
		m.writeFieldIfChanged(id, pre+fieldVerified, was.Verified, now.Verified),
		m.writeFieldIfChanged(id, pre+fieldEndpoints, was.Endpoints, now.Endpoints),
	)
}

// roomEntryOf is a room as the user's lists hold it, and whether they hold it.
func (m *Membership) roomEntryOf(user, roomID string) (roomEntry, bool) {
	v, ok := m.lists(user).rooms[roomID]
	return v, ok
}

// RecordRoomOnList writes the changes that put a room on the user's room list, or
// change it there: change edits the room as the lists hold it now, and one change is
// written for each field it alters, a machine's joining or leaving being a field of
// that machine's own (D-228, each room on the list carries the user's machines that
// have joined it).
func (m *Membership) RecordRoomOnList(id *Identity, roomID, name string, change func(*roomEntry)) error {
	if !id.HasUser() {
		return nil
	}
	was, _ := m.roomEntryOf(id.User, roomID)
	now := was
	now.Machines = append([]string(nil), was.Machines...)
	if name != "" {
		now.Name = name
	}
	if change != nil {
		change(&now)
	}
	now.Machines = uniqueSorted(now.Machines)
	pre := entryRoom + roomID + ":"
	errs := []error{
		m.writeFieldIfChanged(id, pre+fieldName, was.Name, now.Name),
		m.writeFieldIfChanged(id, pre+fieldOffer, roomOffer{was.OfferedBy, was.Endpoint}, roomOffer{now.OfferedBy, now.Endpoint}),
	}
	joined := func(list []string) map[string]bool {
		out := map[string]bool{}
		for _, k := range list {
			out[k] = true
		}
		return out
	}
	before, after := joined(was.Machines), joined(now.Machines)
	for _, k := range uniqueSorted(append(append([]string(nil), was.Machines...), now.Machines...)) {
		errs = append(errs, m.writeFieldIfChanged(id, pre+fieldMachine+k, before[k], after[k]))
	}
	return errors.Join(errs...)
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
	for id, v := range m.lists(user).rooms {
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
	if err := m.RecordRoom(roomID, firstNonEmpty(v.Name, roomID), showOnly); err != nil {
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
// machines that joined it. Each machine's joining a room is its own entry, so two
// machines joining at once both stand and nothing here has to say it again.
func (m *Membership) ApplyUserLists(id *Identity) error {
	if !id.HasUser() {
		return nil
	}
	l := m.lists(id.User)
	var errs []error
	for peer, v := range l.colleagues {
		if _, err := PublicFromPeerID(peer); err != nil || peer == id.User {
			continue
		}
		if err := m.applyColleague(peer, v); err != nil {
			errs = append(errs, err)
		}
	}
	for roomID, v := range l.rooms {
		if err := m.applyRoom(id, roomID, v); err != nil {
			errs = append(errs, err)
		}
	}
	errs = append(errs, m.revertUntrusted(id, l))
	return errors.Join(errs...)
}

// revertUntrusted undoes here what a change applied earlier made, once no trusted
// change holds its entry: the change's machine was suspended or removed after it was
// applied. Every change this machine holds is merged, and an entry whose settled
// value the trusted lists lack is taken back where taking it back loses nothing the
// user chose on this machine: a colleague's verification and label, and an offer of
// a room. A colleague forgotten by such a change stays forgotten, since forgetting
// keeps nothing to restore, and a machine recorded as having joined a room stays
// recorded, since a machine that speaks for nobody is refused whatever is recorded.
func (m *Membership) revertUntrusted(id *Identity, trusted userLists) error {
	_, changes, err := m.IdentityRecords(id.User)
	if err != nil {
		return err
	}
	all := rebuildLists(MergeListChanges(id.User, changes))
	var errs []error
	for peer, was := range all.colleagues {
		if _, ok := trusted.colleagues[peer]; ok || peer == id.User || !m.Knows(peer) {
			continue
		}
		if was.Verified != "" && m.IsVerified(peer) {
			errs = append(errs, m.setVerifiedAt(peer, ""))
		}
		if was.Label != "" && m.Label(peer) == was.Label {
			// The derived name, given outright, since Allow keeps a name it is
			// given none for.
			errs = append(errs, m.Allow(peer, PeerName(peer)))
		}
	}
	for roomID, was := range all.rooms {
		if was.OfferedBy == "" || trusted.rooms[roomID].OfferedBy == was.OfferedBy {
			continue
		}
		if o, ok := m.FindOffer(roomID); ok && o.RoomID == roomID && o.Host == was.OfferedBy {
			errs = append(errs, m.DropOffer(roomID))
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
				return m.RecordOffer(Offer{RoomID: roomID, RoomName: firstNonEmpty(v.Name, roomID), Host: v.OfferedBy, Endpoint: v.Endpoint})
			}
		}
		return nil
	}
	for _, k := range v.Machines {
		if k == id.PeerID {
			continue
		}
		if err := m.RecordRoomMachine(roomID, k, id.User); err != nil {
			return err
		}
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
