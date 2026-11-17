package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// --- how a stored turn stands for the reader, and who the room's people are ---
//
// A turn this machine stored can stop counting after it was stored: a removal of its
// machine arrives with a cutoff before it (D-220, a removal names the last event of a
// machine that counts), a revocation of its author arrives with one (D-236, a
// revocation names the last turn of each of the revoked user's machines that counts),
// or the author certifies a new shared key without its machine (D-234, a machine not
// certified under the current shared key is suspended). Each is decided again from
// the records every time the room is read, so a suspension lifted by certifying the
// machine again brings its turns back, and nothing stored is deleted.

type standing int

const (
	// standingCounts is a turn that counts.
	standingCounts standing = iota
	// standingRemovedMachine is a turn past its machine's removal cutoff, shown marked
	// and never given to an assistant (D-220).
	standingRemovedMachine
	// standingRemovedGuest is a turn past its author's revocation cutoff, stored before
	// the revocation arrived, shown marked and never given to an assistant (D-236).
	standingRemovedGuest
	// standingHidden is a turn of a suspended machine, which appears nowhere (D-234).
	standingHidden
)

// markText is what the view and the export say of a marked turn.
func (s standing) markText() string {
	switch s {
	case standingRemovedMachine:
		return "from a removed machine"
	case standingRemovedGuest:
		return "from a removed guest"
	}
	return ""
}

// standings judges a room's turns, resolving each signing machine's user once.
type standings struct {
	members *Membership
	room    string
	list    *GuestList
	states  map[string]*UserState
}

func (d *Daemon) standingsFor(roomID string) *standings {
	return &standings{members: d.members, room: roomID, list: d.guestList(roomID), states: map[string]*UserState{}}
}

// of is how one stored event stands. An event of a machine with no user identity is
// judged by the guest list alone, as it always was.
func (s *standings) of(e Event) standing {
	signer := e.signerKey()
	if e.SigVersion == 4 {
		key := e.PeerID + " " + signer
		st, ok := s.states[key]
		if !ok {
			st = s.members.UserState(e.PeerID, e.Chain)
			s.states[key] = st
		}
		ms, ok := st.Machines[e.machineKey()]
		if !ok || !ms.Current {
			return standingHidden
		}
		if !ms.counts(signer, s.room, e.PeerSequence) {
			return standingRemovedMachine
		}
	}
	if s.list != nil && !isGuestEventType(e.EventType) && !s.list.Counts(e.PeerID, signer, e.PeerSequence) {
		return standingRemovedGuest
	}
	return standingCounts
}

// onlyShown drops the events of suspended machines, which appear nowhere (D-234).
func (s *standings) onlyShown(evs []Event) []Event {
	out := evs[:0:0]
	for _, e := range evs {
		if s.of(e) != standingHidden {
			out = append(out, e)
		}
	}
	return out
}

// onlyServed drops the events of suspended machines from what this machine serves to
// other machines, as the view and the export drop them: a receiver serves none of a
// suspended machine's events, and serves them again once the machine is certified
// again, since they are kept (D-234, a machine not certified under the current shared
// key is suspended until it is certified again). Sync counts an unbroken run, so a
// requester simply asks again for what was left out.
func (d *Daemon) onlyServed(roomID string, evs []Event) []Event {
	return d.standingsFor(roomID).onlyShown(evs)
}

// imageOnlyHidden reports whether every event of a room that names an image is a
// suspended machine's, so that the image is not served while its turns are not
// (D-234). An image no event names is judged as before, by the room's image rows.
func (d *Daemon) imageOnlyHidden(store *Store, roomID, sha string) bool {
	evs, err := store.ListRoom(roomID)
	if err != nil {
		return false
	}
	judge := d.standingsFor(roomID)
	named := false
	for _, e := range evs {
		for _, r := range imageRefs(e.Metadata) {
			if r.SHA256 != sha {
				continue
			}
			named = true
			if judge.of(e) != standingHidden {
				return false
			}
		}
	}
	return named
}

// turnMachine is the name of the machine a turn came from, as its user gave it: the
// name in the machine's certificate for an event signed for a user, and the name the
// event carries otherwise. It is the user's claim, and is shown and injected as a
// field of its own, never joined to a name (D-233, every turn carries the name of the
// machine it came from, as a field the user asserts).
func turnMachine(e Event) string {
	if e.SigVersion == 4 && e.Chain != nil && e.Chain.Machine.Name != "" {
		return shownName(e.Chain.Machine.Name)
	}
	return shownName(e.MachineID)
}

// maxShownName is the longest a name another machine asserts, a turn's speaker or its
// machine, is shown or given to an assistant.
const maxShownName = 64

// shownName is a name another machine asserts, as it is shown and injected: without
// characters that break a line, hide or reorder text, and cut to maxShownName bytes.
// The event keeps the name as it was signed; only what is shown of it is bounded.
func shownName(s string) string {
	var b strings.Builder
	for _, r := range s {
		if !unicode.IsPrint(r) {
			continue
		}
		if b.Len()+utf8.RuneLen(r) > maxShownName {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}

// receivingFacts is what this machine knows of a turn's author that the author does
// not assert: whether they are verified, the label this user gave them, and whether
// the turn is this user's own (D-099, the block carries the label; D-232, a user's own
// turn from another session is injected as a report marked as the user's). The user
// is the identifier the turn's chain proved, so no text in a turn can claim it.
type receivingFacts struct {
	m    *Membership
	self string
}

func (f receivingFacts) IsVerified(peer string) bool { return peer == f.self || f.m.IsVerified(peer) }
func (f receivingFacts) Label(peer string) string {
	if peer == f.self {
		return "you"
	}
	return f.m.Label(peer)
}
func (f receivingFacts) IsOwn(peer string) bool { return peer == f.self }

// ownFacts is the fact a PeerFacts may also know: whether a peer is the receiving
// user. Facts that do not know it, such as a probe's, mark no turn as the user's own.
type ownFacts interface {
	IsOwn(peerID string) bool
}

// --- the room's guest list, as the view shows it ---

// uiGuest is one user the room's list has ever admitted (D-235, a room's one guest
// list shows who invited or revoked whom).
type uiGuest struct {
	Name     string `json:"name"`
	PeerName string `json:"peerName"`
	PeerID   string `json:"peerId"`
	Self     bool   `json:"self,omitempty"`
	// Verified is false for a guest this user does not know or has not verified, such
	// as one this user forgot, who is shown by the derived name (D-073, forgetting
	// leaves rooms alone).
	Verified bool `json:"verified"`
	// State is "guest", "removed", or "unsettled" for a user an invitation counted
	// for whom the list does not admit, because the room's record of changes never
	// settles (D-235), so the view does not present them as a guest.
	State     string   `json:"state"`
	InvitedBy []string `json:"invitedBy,omitempty"`
	RevokedBy []string `json:"revokedBy,omitempty"`
	Founder   bool     `json:"founder,omitempty"`
}

// guestView is the room's list as its events compute it, for a room that keeps its
// list as events, with who invited and who revoked each guest.
func (d *Daemon) guestView(store *Store, roomID string) []uiGuest {
	if !d.members.EventListed(roomID) {
		return nil
	}
	g, _, err := d.resolveRoomGuests(store, roomID)
	if err != nil {
		return nil
	}
	self := d.id.Peer()
	name := func(u string) string {
		if u == self {
			return "you"
		}
		return firstNonEmpty(d.members.Label(u), PeerName(u))
	}
	var out []uiGuest
	for u, admitted := range g.Admitted {
		v := uiGuest{Name: name(u), PeerName: PeerName(u), PeerID: u, Self: u == self,
			Verified: u == self || d.members.IsVerified(u)}
		for _, c := range g.Changes {
			if c.Subject != u {
				continue
			}
			switch {
			case c.Kind == GuestInvite && c.Author == u:
				// A founding self-invitation, or a founder of a room from before user
				// identities (D-243), since no other self-invitation counts.
				v.Founder = true
			case c.Kind == GuestInvite:
				v.InvitedBy = appendOnce(v.InvitedBy, name(c.Author))
			case c.Kind == GuestRevoke:
				v.RevokedBy = appendOnce(v.RevokedBy, name(c.Author))
			}
		}
		switch {
		case admitted:
			v.State = "guest"
		case len(v.RevokedBy) > 0:
			v.State = "removed"
		default:
			v.State = "unsettled"
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Self != out[j].Self {
			return out[i].Self
		}
		return out[i].PeerID < out[j].PeerID
	})
	return out
}

func appendOnce(list []string, s string) []string {
	for _, x := range list {
		if x == s {
			return list
		}
	}
	return append(list, s)
}

// --- this user's removal from a room, as the page, the status and the prompt say it ---

// removalSummary is what this machine holds of its user's removal from a room, said
// in names this user knows (D-241, an expelled user is told, by whom and from which
// of their turns on).
type removalSummary struct {
	Expelled bool     `json:"expelled"`
	By       []string `json:"by"`
	From     []string `json:"from"`
	// Resuming is a room an admitted guest has offered again since, which is coming
	// back once the invitation that lifts the revocation arrives (D-241).
	Resuming bool `json:"resuming,omitempty"`
	// FromBeforeUsers is a room made before user identities, whose guests' machines
	// refuse this one until they count its user as a guest (D-243, a room from before
	// user identities takes one guest list).
	FromBeforeUsers bool `json:"fromBeforeUsers,omitempty"`
	// key names the revocations the summary is of, so a notice is given once for them.
	key string
}

// removalFrom summarizes the counted revocations of this machine's user from a room,
// and reports false when the list does not expel the user.
func (d *Daemon) removalFrom(roomID string) (removalSummary, bool) {
	ex, ok := d.Expulsion(roomID)
	if !ok || !ex.Expelled {
		return removalSummary{}, false
	}
	g := d.guestList(roomID)
	store, err := d.storeFor(roomID)
	if err != nil || g == nil {
		return removalSummary{}, false
	}
	resolved, _, err := d.resolveRoomGuests(store, roomID)
	if err != nil {
		return removalSummary{}, false
	}
	counted := map[string]bool{}
	for _, c := range resolved.Changes {
		counted[c.EventID] = true
	}
	out := removalSummary{Expelled: true}
	var ids []string
	for _, r := range ex.Revocations {
		if !counted[r.EventID] {
			continue
		}
		ids = append(ids, r.EventID)
		if r.Author == d.id.Peer() {
			out.By = appendOnce(out.By, "you")
		} else {
			out.By = appendOnce(out.By, firstNonEmpty(d.members.Label(r.Author), PeerName(r.Author)))
		}
	}
	if len(ids) == 0 {
		return removalSummary{}, false
	}
	names, _ := d.members.displayCerts(d.id.User)
	cut := g.Cutoffs[d.id.User]
	var machines []string
	for k := range cut {
		machines = append(machines, k)
	}
	sort.Strings(machines)
	for _, k := range machines {
		machine := firstNonEmpty(names[k].Name, PeerName(k))
		if cut[k] == 0 {
			out.From = append(out.From, fmt.Sprintf("every turn from %s", machine))
		} else {
			out.From = append(out.From, fmt.Sprintf("the turns from %s after its turn %d", machine, cut[k]))
		}
	}
	out.From = append(out.From, "every turn from any other machine of yours")
	_, _, out.Resuming = d.members.Resumption(roomID)
	out.key = strings.Join(ids, ",")
	return out, true
}

// sentence is the summary as one sentence, in names this user knows and words cogmer
// wrote; a room's name is left out, since whoever invited the user chose it.
func (s removalSummary) sentence() string {
	return fmt.Sprintf("You were removed from this room by %s. What counts of yours stops at: %s.",
		strings.Join(s.By, " and "), strings.Join(s.From, "; "))
}

// expulsionNotice is what the next prompt of a session in a room its user was removed
// from is told, once per session for one set of revocations (D-241). It comes from
// this machine, so it stands outside the block of other people's turns.
func (d *Daemon) expulsionNotice(room Room, session string) string {
	s, ok := d.removalFrom(room.RoomID)
	if !ok || !d.members.noteExpulsionNotice(session, room.RoomID, s.key) {
		return ""
	}
	return "<cogmer-notice>\n" + s.sentence() +
		" This session keeps recording its own turns on this machine, and they reach the room if a guest invites you back." +
		" This session cannot join another room; a new session can.\n</cogmer-notice>\n"
}

// noteExpulsionNotice records that a session was told of a set of revocations, and
// reports whether it had not been told before.
func (m *Membership) noteExpulsionNotice(session, roomID, revocations string) bool {
	res, err := m.db.Exec(`INSERT OR IGNORE INTO expulsion_notices (session_id, room_id, revocations, told_at) VALUES (?,?,?,?)`,
		session, roomID, revocations, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return false
	}
	n, err := res.RowsAffected()
	return err == nil && n == 1
}

// --- reachability, by person and by the user's own machine ---

// groupReach turns the state of each address into what a person reads: one line for
// each colleague, reachable when any of their machines is, and unreachable only when
// none is, for the shortest time any has been; and one line for each of the user's own
// machines, by its name (D-194, a colleague this machine cannot reach is named, with
// how long). An address configured for the machine, which names nobody, stays a line
// of its own. joined, when not nil, is the user's own machines that joined the room,
// each listed even with no address.
func (d *Daemon) groupReach(ps []uiPeer, joined []string) []uiPeer {
	self := d.id.Peer()
	machines := d.members.EndpointMachines()
	names := map[string]MachineCert{}
	if d.id.HasUser() {
		names, _ = d.members.displayCerts(d.id.User)
	}
	var out []uiPeer
	byUser := map[string][]uiPeer{}
	var order []string
	ownSeen := map[string]bool{}
	for _, p := range ps {
		switch {
		case p.PeerID == "":
			out = append(out, p)
		case p.PeerID == self && d.id.HasUser():
			k := machines[p.addr]
			p.Own, p.Machine = true, k
			p.Name = firstNonEmpty(names[k].Name, PeerName(k))
			ownSeen[k] = true
			out = append(out, p)
		default:
			if _, ok := byUser[p.PeerID]; !ok {
				order = append(order, p.PeerID)
			}
			byUser[p.PeerID] = append(byUser[p.PeerID], p)
		}
	}
	for _, k := range joined {
		if k == d.id.PeerID || ownSeen[k] {
			continue
		}
		out = append(out, uiPeer{Name: firstNonEmpty(names[k].Name, PeerName(k)), PeerID: self, Own: true, Machine: k, NoAddress: true})
	}
	for _, u := range order {
		out = append(out, bestReach(byUser[u]))
	}
	return out
}

// bestReach is one colleague's line from the lines of their machines.
func bestReach(group []uiPeer) uiPeer {
	for _, p := range group {
		if p.Online {
			p.Machines = len(group)
			return p
		}
	}
	best, unreachable := group[0], true
	for _, p := range group {
		if p.Trying == "" {
			best, unreachable = p, false
			break
		}
		if p.tryingFor < best.tryingFor {
			best = p
		}
	}
	if unreachable {
		for _, p := range group {
			if p.HeardFrom != "" && (best.HeardFrom == "" || p.heardAgo < best.heardAgo) {
				best.HeardFrom, best.heardAgo = p.HeardFrom, p.heardAgo
			}
		}
	}
	best.Machines = len(group)
	return best
}

// roomJoinedOwn is the user's own machines recorded as having joined a room.
func (d *Daemon) roomJoinedOwn(roomID string) []string {
	if !d.id.HasUser() {
		return nil
	}
	var out []string
	rows, err := d.members.db.Query(`SELECT machine_key FROM room_machines WHERE room_id = ? AND user = ? ORDER BY machine_key`,
		roomID, d.id.User)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		if rows.Scan(&k) == nil {
			out = append(out, k)
		}
	}
	return out
}

// EndpointMachines maps each recorded address to the machine it was recorded for.
func (m *Membership) EndpointMachines() map[string]string {
	rows, err := m.db.Query(`SELECT endpoint, machine_key FROM machines WHERE endpoint IS NOT NULL AND endpoint <> ''`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var e, k string
		if rows.Scan(&e, &k) == nil {
			out[e] = k
		}
	}
	return out
}

// --- refusals by a room's guests ---

// roomRefusals is, for each room, when a guest's machine last refused this machine's
// request for it and when one last answered it, in memory like the peer states. A
// machine whose user was removed and that holds no revocation learns of it only as
// refusals (D-241, a machine that holds no revocation says the room's guests are
// refusing it).
type roomRefusals struct {
	refused, answered map[string]time.Time
}

func (d *Daemon) noteRoomRefused(roomID string) {
	d.peerMu.Lock()
	defer d.peerMu.Unlock()
	if d.refusals.refused == nil {
		d.refusals.refused = map[string]time.Time{}
	}
	d.refusals.refused[roomID] = time.Now()
}

func (d *Daemon) noteRoomAnswered(roomID string) {
	d.peerMu.Lock()
	defer d.peerMu.Unlock()
	if d.refusals.answered == nil {
		d.refusals.answered = map[string]time.Time{}
	}
	d.refusals.answered[roomID] = time.Now()
}

// roomRefusing reports whether the last word from a room's guests was a refusal.
func (d *Daemon) roomRefusing(roomID string) bool {
	d.peerMu.Lock()
	defer d.peerMu.Unlock()
	r, ok := d.refusals.refused[roomID]
	return ok && r.After(d.refusals.answered[roomID])
}
