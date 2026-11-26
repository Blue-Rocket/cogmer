package main

import (
	"sync"
	"time"
)

// How often an address is polled once it has sent nothing for a while (D-305, an address
// is polled less often while its rooms are quiet). An address that answers with nothing
// is polled at twice the last interval, from the usual one up to quietPollCap, and up to
// dormantPollCap once none of the rooms it is asked about has stored an event for
// dormantAfter. dormantPollCap is also the cap for an address that does not answer once
// its rooms are dormant (failDelay). Both caps are under the 90 seconds a kept connection
// waits idle (clientFor in transport.go), so a slow poll still uses the connection it kept
// (D-303, a machine keeps one connection to each address it polls).
// lateArrival is how long after it was written an event that is stored is logged as late:
// longer than the longest wait between polls and a nudge sent again after it, so an event
// on any path that works arrives in less (D-306, a machine nudges the machines it shares a room with when it writes an event).
const lateArrival = 2 * time.Minute

// lateAfterOr is lateArrival unless a test says otherwise.
func (d *Daemon) lateAfterOr() time.Duration {
	if d.pace.lateAfter != 0 {
		return d.pace.lateAfter
	}
	return lateArrival
}

const (
	quietPollCap   = 20 * time.Second
	dormantPollCap = 60 * time.Second
	dormantAfter   = 30 * time.Minute
)

// pollPace is when each room last stored an event and when it last woke the poll loops,
// and wakes every poll loop when a room should wake them, so that an address polled
// slowly is polled again at once when a room it is asked about wants it.
type pollPace struct {
	mu     sync.Mutex
	stored map[string]time.Time // room id → when it last stored an event
	woken  map[string]time.Time // room id → when it last woke its addresses
	// An address a nudge woke, and the user of each address that has nudged this
	// machine with the address of each user's last nudge (D-306, a machine nudges the
	// machines it shares a room with when it writes an event).
	wokenAddr map[string]time.Time
	nudgerOf  map[string]string // address → its machine's user
	lastNudge map[string]string // user → the address of their last nudge
	failing   map[string]bool   // address → its last poll failed
	gen       uint64            // counts the wakes so far
	changed   chan struct{}     // closed, and replaced, at each wake

	// The caps and the quiet time, which are the constants above and syncBackoffCap
	// unless a test says otherwise. A cap below the poll interval keeps every poll at
	// the interval.
	quietCap, dormantCap, dormantAfter, failCap time.Duration
	lateAfter                                   time.Duration
}

// roomStored records that a room stored an event now, and wakes the addresses it is
// asked of when wake is set. An event written on this machine wakes them, since any
// colleague may answer it. An event received from an address wakes none: the address
// that sent it is polled again at the usual interval by its own loop, and the room's
// other addresses have nothing new because of it.
func (d *Daemon) roomStored(roomID string, wake bool) {
	p := &d.pace
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	if p.stored == nil {
		p.stored = map[string]time.Time{}
	}
	p.stored[roomID] = now
	if wake {
		p.wakeLocked(roomID, now)
	}
}

// roomOpened wakes the addresses a room is asked of, because somebody opened its view and
// is about to read it. It stores nothing, so a room that is only looked at still becomes
// dormant.
func (d *Daemon) roomOpened(roomID string) {
	p := &d.pace
	p.mu.Lock()
	defer p.mu.Unlock()
	p.wakeLocked(roomID, time.Now())
}

func (p *pollPace) wakeLocked(roomID string, now time.Time) {
	if p.woken == nil {
		p.woken = map[string]time.Time{}
	}
	p.woken[roomID] = now
	p.bumpLocked()
}

// nudgedBy records that a machine of user at addr nudged this machine, which makes addr
// the one address of a colleague that is polled, and polls it at once (D-306, a machine nudges the machines it shares a room with when it writes an event). A nudge
// from another machine of this machine's own user polls it and sets none aside, so that
// a user's own machines keep polling one another and pass on what each holds.
func (d *Daemon) nudgedBy(user, addr string) {
	p := &d.pace
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.wokenAddr == nil {
		p.wokenAddr, p.nudgerOf, p.lastNudge = map[string]time.Time{}, map[string]string{}, map[string]string{}
	}
	if user != d.ownUser() {
		p.nudgerOf[addr] = user
		p.lastNudge[user] = addr
	}
	p.wokenAddr[addr] = time.Now()
	p.bumpLocked()
}

// ownUser is the user this machine speaks for, which is the machine itself when it has no
// user identity.
func (d *Daemon) ownUser() string {
	if d.id == nil {
		return ""
	}
	if d.id.HasUser() {
		return d.id.User
	}
	return d.id.PeerID
}

// setAside says whether addr is a colleague's machine that has nudged this machine and
// whose user has nudged since from another address that still answers, so that addr is
// not polled until it nudges again (D-306, a machine nudges the machines it shares a room with when it writes an event). When the address that nudged last stops
// answering, the user's other machines are polled again, so that a turn written on one
// of them whose nudge was lost is not left there.
func (d *Daemon) setAside(addr string) bool {
	p := &d.pace
	p.mu.Lock()
	defer p.mu.Unlock()
	user, ok := p.nudgerOf[addr]
	if !ok {
		return false
	}
	last := p.lastNudge[user]
	return last != addr && !p.failing[last]
}

// pollFailed and pollAnswered record whether addr's last poll failed, and wake the poll
// loops when that changes, since a machine set aside is polled while its user's last
// nudger fails.
func (d *Daemon) pollFailed(addr string)   { d.setFailing(addr, true) }
func (d *Daemon) pollAnswered(addr string) { d.setFailing(addr, false) }

func (d *Daemon) setFailing(addr string, failing bool) {
	p := &d.pace
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.failing[addr] == failing {
		return
	}
	if p.failing == nil {
		p.failing = map[string]bool{}
	}
	if failing {
		p.failing[addr] = true
	} else {
		delete(p.failing, addr)
	}
	p.gen++
	if p.changed != nil {
		close(p.changed)
		p.changed = nil
	}
}

func (p *pollPace) bumpLocked() {
	p.gen++
	if p.changed != nil {
		close(p.changed)
		p.changed = nil
	}
}

// paceWatch is a channel closed at the next wake, and how many wakes there have been.
func (d *Daemon) paceWatch() (<-chan struct{}, uint64) {
	p := &d.pace
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.changed == nil {
		p.changed = make(chan struct{})
	}
	return p.changed, p.gen
}

// askedRooms is the rooms addr is asked about.
func (d *Daemon) askedRooms(addr string) ([]Room, error) {
	rooms, err := d.members.Rooms()
	if err != nil {
		return nil, err
	}
	ask := d.roomsToAsk
	if d.askRooms != nil {
		ask = d.askRooms
	}
	return ask(addr, rooms), nil
}

// addrWokenSince says whether addr was nudged after t, or a room that addr is asked about
// woke its addresses after t.
func (d *Daemon) addrWokenSince(addr string, t time.Time) bool {
	return d.addrNudgedSince(addr, t) || d.addrSince(addr, t, func(p *pollPace) map[string]time.Time { return p.woken })
}

// addrNudgedSince says whether addr nudged this machine after t.
func (d *Daemon) addrNudgedSince(addr string, t time.Time) bool {
	d.pace.mu.Lock()
	defer d.pace.mu.Unlock()
	return d.pace.wokenAddr[addr].After(t)
}

// addrDormant says whether no room that addr is asked about has stored an event for the
// quiet time, in a loop that started at started.
func (d *Daemon) addrDormant(addr string, started time.Time) bool {
	quiet := d.pace.dormantAfter
	if quiet == 0 {
		quiet = dormantAfter
	}
	since := time.Now().Add(-quiet)
	return started.Before(since) &&
		!d.addrSince(addr, since, func(p *pollPace) map[string]time.Time { return p.stored })
}

func (d *Daemon) addrSince(addr string, t time.Time, times func(*pollPace) map[string]time.Time) bool {
	asked, err := d.askedRooms(addr)
	if err != nil {
		return true // unknown, so the address is not slowed
	}
	p := &d.pace
	p.mu.Lock()
	defer p.mu.Unlock()
	m := times(p)
	for _, r := range asked {
		if m[r.RoomID].After(t) {
			return true
		}
	}
	return false
}

// pollState is what the loop polling one address knows of its own polls.
type pollState struct {
	started  time.Time   // when the loop started
	last     pollOutcome // what the last poll did
	failures int         // polls in a row that failed
	empty    int         // polls in a row that answered with nothing
}

type pollOutcome int

const (
	notPolled pollOutcome = iota
	pollFailedNow
	pollBroughtEvents
	pollBroughtNothing
)

// polled records what a poll did.
func (s *pollState) polled(events int, err error) {
	switch {
	case err != nil:
		s.last, s.failures, s.empty = pollFailedNow, s.failures+1, 0
	case events > 0:
		s.last, s.failures, s.empty = pollBroughtEvents, 0, 0
	default:
		s.last, s.failures = pollBroughtNothing, 0
		s.empty++
	}
}

// woken records that a wake ended the wait, after which the waits start again from the
// usual interval.
func (s *pollState) woken() { s.empty = 0 }

// pollFacts is what decidePoll needs to know beyond the loop's own polls.
type pollFacts struct {
	setAside bool        // another machine of the address's user nudged last and answers
	dormant  func() bool // no room the address is asked about has stored an event lately
}

// pollPacing is the interval and the caps a decision uses.
type pollPacing struct {
	every, quietCap, failCap, dormantCap time.Duration
}

// pollDecision is when an address is polled next.
type pollDecision struct {
	setAside  bool          // not polled until a nudge or a failure changes the facts
	wait      time.Duration // how long to wait before the next poll
	wakeable  bool          // whether a wake of the address or its rooms ends the wait early
	nudgeable bool          // whether a nudge from the address ends the wait early
	least     time.Duration // the least time between two polls, however soon a wake comes
}

// decidePoll is the one place that decides when an address is polled next, from what its
// loop's polls did and what is known of its rooms and its user's machines (D-305, an
// address is polled less often while its rooms are quiet; D-306, a machine nudges the
// machines it shares a room with when it writes an event).
//
//   - A colleague's machine that another of the colleague's machines has displaced as the
//     last to nudge is set aside, and not polled at all.
//   - The first poll is at once.
//   - After a failure, the wait doubles from the interval up to failCap, or up to
//     dormantCap once the address's rooms are dormant. A nudge from the address ends it,
//     since the nudge shows the machine there is running, and a wake of its rooms does
//     not.
//   - After a poll that brought events, the wait is the interval.
//   - After a poll that brought nothing, the wait doubles from the interval up to
//     quietCap, or up to dormantCap once the rooms are dormant, and a wake ends it.
//   - However soon a wake comes, two polls are at least the interval apart.
//
// dormant is asked only once a wait reaches the cap for rooms that are not dormant, since
// answering it reads the room records.
func decidePoll(s pollState, f pollFacts, p pollPacing) pollDecision {
	if f.setAside {
		return pollDecision{setAside: true}
	}
	d := pollDecision{least: p.every}
	switch s.last {
	case notPolled:
	case pollFailedNow:
		d.wait = backoff(p.every, s.failures-1, p.failCap, p.dormantCap, f.dormant)
		d.nudgeable = true
	case pollBroughtEvents:
		d.wait = p.every
	case pollBroughtNothing:
		d.wait = backoff(p.every, s.empty, p.quietCap, p.dormantCap, f.dormant)
		d.wakeable, d.nudgeable = true, true
	}
	return d
}

// backoff doubles every n times, up to activeCap, or up to dormantCap once dormant says
// so.
func backoff(every time.Duration, n int, activeCap, dormantCap time.Duration, dormant func() bool) time.Duration {
	top := max(activeCap, dormantCap)
	delay := every
	for i := 0; i < n && delay < top; i++ {
		delay *= 2
	}
	if delay <= activeCap {
		return max(delay, every)
	}
	limit := activeCap
	if dormant() {
		limit = dormantCap
	}
	return max(min(delay, limit), every)
}

// pacing is the interval and caps this daemon decides with, which are the constants
// unless a test says otherwise.
func (d *Daemon) pacing(every time.Duration) pollPacing {
	p := pollPacing{every: every, quietCap: d.pace.quietCap, failCap: d.pace.failCap, dormantCap: d.pace.dormantCap}
	if p.quietCap == 0 {
		p.quietCap = quietPollCap
	}
	if p.failCap == 0 {
		p.failCap = syncBackoffCap
	}
	if p.dormantCap == 0 {
		p.dormantCap = dormantPollCap
	}
	return p
}

// pollFactsFor is what is known now of addr for a loop that started at started.
func (d *Daemon) pollFactsFor(addr string, started time.Time) pollFacts {
	return pollFacts{
		setAside: d.setAside(addr),
		dormant:  func() bool { return d.addrDormant(addr, started) },
	}
}
