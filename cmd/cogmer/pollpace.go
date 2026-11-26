package main

import (
	"sync"
	"time"
)

// How often an address is polled once it has sent nothing for a while (D-304, an address
// is polled less often while its rooms are quiet). An address that answers with nothing
// is polled at twice the last interval, from the usual one up to quietPollCap, and up to
// dormantPollCap once none of the rooms it is asked about has stored an event for
// dormantAfter. Both caps are under the 90 seconds a kept connection waits idle
// (clientFor in transport.go), so a slow poll still uses the connection it kept (D-303, a
// machine keeps one connection to each address it polls).
const (
	quietPollCap   = 20 * time.Second
	dormantPollCap = 60 * time.Second
	dormantAfter   = 30 * time.Minute
)

// pollPace is when each room last stored an event, from a session on this machine or from
// another machine, and wakes every poll loop when a room stores one, so that an address
// polled slowly is polled again at once when a room it is asked about becomes active.
type pollPace struct {
	mu      sync.Mutex
	active  map[string]time.Time // room id → when it last stored an event
	gen     uint64               // counts the times a room became active
	changed chan struct{}        // closed, and replaced, when a room becomes active

	// The caps and the quiet time, which are the constants above unless a test says
	// otherwise. A cap below the poll interval keeps every poll at the interval.
	quietCap, dormantCap, dormantAfter time.Duration
}

// roomActive records that a room stored an event now and wakes the poll loops.
func (d *Daemon) roomActive(roomID string) {
	p := &d.pace
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.active == nil {
		p.active = map[string]time.Time{}
	}
	p.active[roomID] = time.Now()
	p.gen++
	if p.changed != nil {
		close(p.changed)
		p.changed = nil
	}
}

// paceWatch is a channel closed when a room next becomes active, and how many times rooms
// have become active so far.
func (d *Daemon) paceWatch() (<-chan struct{}, uint64) {
	p := &d.pace
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.changed == nil {
		p.changed = make(chan struct{})
	}
	return p.changed, p.gen
}

// addrActiveSince says whether a room that addr is asked about has stored an event after t.
func (d *Daemon) addrActiveSince(addr string, t time.Time) bool {
	rooms, err := d.members.Rooms()
	if err != nil {
		return true // unknown, so the address is not slowed
	}
	ask := d.roomsToAsk
	if d.askRooms != nil {
		ask = d.askRooms
	}
	asked := ask(addr, rooms)
	p := &d.pace
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, r := range asked {
		if p.active[r.RoomID].After(t) {
			return true
		}
	}
	return false
}

// quietDelay is how long to wait before polling addr again after empty answers in a row,
// for a loop that started at started.
func (d *Daemon) quietDelay(addr string, every time.Duration, empty int, started time.Time) time.Duration {
	p := &d.pace
	quietCap, dormantCap, quiet := p.quietCap, p.dormantCap, p.dormantAfter
	if quietCap == 0 {
		quietCap = quietPollCap
	}
	if dormantCap == 0 {
		dormantCap = dormantPollCap
	}
	if quiet == 0 {
		quiet = dormantAfter
	}
	delay := every
	for i := 0; i < empty && delay < dormantCap; i++ {
		delay *= 2
	}
	if delay < quietCap {
		return delay
	}
	limit := quietCap
	since := time.Now().Add(-quiet)
	if started.Before(since) && !d.addrActiveSince(addr, since) {
		limit = dormantCap
	}
	if delay > limit {
		delay = limit
	}
	if delay < every {
		delay = every
	}
	return delay
}
