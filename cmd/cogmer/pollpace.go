package main

import (
	"sync"
	"time"
)

// How often an address is polled once it has sent nothing for a while (D-304, an address
// is polled less often while its rooms are quiet). An address that answers with nothing
// is polled at twice the last interval, from the usual one up to quietPollCap, and up to
// dormantPollCap once none of the rooms it is asked about has stored an event for
// dormantAfter. dormantPollCap is also the cap for an address that does not answer once
// its rooms are dormant (failDelay). Both caps are under the 90 seconds a kept connection
// waits idle (clientFor in transport.go), so a slow poll still uses the connection it kept
// (D-303, a machine keeps one connection to each address it polls).
const (
	quietPollCap   = 20 * time.Second
	dormantPollCap = 60 * time.Second
	dormantAfter   = 30 * time.Minute
)

// pollPace is when each room last stored an event and when it last woke the poll loops,
// and wakes every poll loop when a room should wake them, so that an address polled
// slowly is polled again at once when a room it is asked about wants it.
type pollPace struct {
	mu      sync.Mutex
	stored  map[string]time.Time // room id → when it last stored an event
	woken   map[string]time.Time // room id → when it last woke its addresses
	gen     uint64               // counts the wakes so far
	changed chan struct{}        // closed, and replaced, at each wake

	// The caps and the quiet time, which are the constants above and syncBackoffCap
	// unless a test says otherwise. A cap below the poll interval keeps every poll at
	// the interval.
	quietCap, dormantCap, dormantAfter, failCap time.Duration
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

// addrWokenSince says whether a room that addr is asked about woke its addresses after t.
func (d *Daemon) addrWokenSince(addr string, t time.Time) bool {
	return d.addrSince(addr, t, func(p *pollPace) map[string]time.Time { return p.woken })
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

// quietDelay is how long to wait before polling addr again after empty answers in a row,
// for a loop that started at started.
func (d *Daemon) quietDelay(addr string, every time.Duration, empty int, started time.Time) time.Duration {
	quietCap, dormantCap := d.paceCaps()
	return d.backoff(addr, every, empty, quietCap, dormantCap, started)
}

// failDelay is how long to wait before polling addr again after failures in a row. A
// machine that does not answer is polled up to every syncBackoffCap, and up to every
// dormantPollCap once its rooms are dormant, so a machine that is off for days is asked
// once a minute (D-304).
func (d *Daemon) failDelay(addr string, every time.Duration, failures int, started time.Time) time.Duration {
	failCap := d.pace.failCap
	if failCap == 0 {
		failCap = syncBackoffCap
	}
	_, dormantCap := d.paceCaps()
	return d.backoff(addr, every, failures-1, failCap, dormantCap, started)
}

func (d *Daemon) paceCaps() (quietCap, dormantCap time.Duration) {
	quietCap, dormantCap = d.pace.quietCap, d.pace.dormantCap
	if quietCap == 0 {
		quietCap = quietPollCap
	}
	if dormantCap == 0 {
		dormantCap = dormantPollCap
	}
	return quietCap, dormantCap
}

// backoff doubles every n times, up to activeCap, or up to dormantCap once addr is dormant.
func (d *Daemon) backoff(addr string, every time.Duration, n int, activeCap, dormantCap time.Duration, started time.Time) time.Duration {
	top := activeCap
	if dormantCap > top {
		top = dormantCap
	}
	delay := every
	for i := 0; i < n && delay < top; i++ {
		delay *= 2
	}
	if delay <= activeCap {
		return max(delay, every)
	}
	limit := activeCap
	if d.addrDormant(addr, started) {
		limit = dormantCap
	}
	return max(min(delay, limit), every)
}
