package main

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// pollTimes records when each poll of one address began.
type pollTimes struct {
	mu sync.Mutex
	at []time.Time
}

func (p *pollTimes) note() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.at = append(p.at, time.Now())
}

func (p *pollTimes) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.at)
}

// lastGap is the time between the last two polls.
func (p *pollTimes) lastGap() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.at) < 2 {
		return 0
	}
	return p.at[len(p.at)-1].Sub(p.at[len(p.at)-2])
}

// paceDaemon polls one address, "peer", asked about room "room-pace", with the caps and
// quiet time given, and pull answering for it.
func paceDaemon(t *testing.T, quietCap, dormantCap, dormant time.Duration, pull func() int) (*Daemon, *pollTimes) {
	t.Helper()
	polls := &pollTimes{}
	d := syncDaemon(t, "peer", func(string) (int, error) { polls.note(); return pull(), nil })
	d.pace.quietCap, d.pace.dormantCap, d.pace.dormantAfter = quietCap, dormantCap, dormant
	d.askRooms = func(string, []Room) []Room { return []Room{{RoomID: "room-pace"}} }
	return d, polls
}

// An address that answers with nothing is polled at twice the last interval, up to the
// cap, and not every interval (D-304, an address is polled less often while its rooms are
// quiet).
func TestAnAddressThatAnswersWithNothingIsPolledLessOften(t *testing.T) {
	d, polls := paceDaemon(t, 80*time.Millisecond, 80*time.Millisecond, time.Hour, func() int { return 0 })
	go d.RunSync(nil, 5*time.Millisecond)
	time.Sleep(700 * time.Millisecond)
	if n := polls.count(); n > 15 || n < 5 {
		t.Errorf("an address that sent nothing was polled %d times in 700ms at an interval of 5ms and a cap of 80ms; want about 11", n)
	}
	if gap := polls.lastGap(); gap < 60*time.Millisecond || gap > 150*time.Millisecond {
		t.Errorf("the last wait was %v; want the cap of 80ms", gap)
	}
}

// An address that sends events is polled at the usual interval.
func TestAnAddressThatSendsEventsIsPolledAtTheUsualInterval(t *testing.T) {
	d, polls := paceDaemon(t, 80*time.Millisecond, 80*time.Millisecond, time.Hour, func() int { return 1 })
	go d.RunSync(nil, 5*time.Millisecond)
	time.Sleep(300 * time.Millisecond)
	if n := polls.count(); n < 30 {
		t.Errorf("an address that sent an event at every poll was polled %d times in 300ms at an interval of 5ms", n)
	}
}

// An address polled slowly is polled at once when a room it is asked about stores an
// event, from a session on this machine or from another address, and at the usual
// interval after that until it is quiet again.
func TestAnAddressPolledSlowlyIsPolledAtOnceWhenItsRoomStoresAnEvent(t *testing.T) {
	d, polls := paceDaemon(t, time.Second, time.Second, time.Hour, func() int { return 0 })
	go d.RunSync(nil, 20*time.Millisecond)
	time.Sleep(900 * time.Millisecond) // waits of 40, 80, 160 and 320ms, then one of 640ms
	before := polls.count()
	d.roomActive("room-pace")
	time.Sleep(50 * time.Millisecond)
	if after := polls.count(); after <= before {
		t.Fatalf("the room stored an event and its address was not polled within 50ms (%d polls before, %d after)", before, after)
	}
}

// An event in a room an address is not asked about leaves that address's wait as it was.
func TestAnEventInAnotherRoomLeavesAnAddressPolledSlowly(t *testing.T) {
	d, polls := paceDaemon(t, time.Second, time.Second, time.Hour, func() int { return 0 })
	go d.RunSync(nil, 20*time.Millisecond)
	time.Sleep(900 * time.Millisecond)
	before := polls.count()
	d.roomActive("room-other")
	time.Sleep(50 * time.Millisecond)
	if after := polls.count(); after != before {
		t.Errorf("an event in a room the address is not asked about polled it %d more times", after-before)
	}
}

// Once no room an address is asked about has stored an event for the quiet time, its
// polls back off to the longer cap, and an event in one of those rooms brings them back to
// the usual interval.
func TestAnAddressWhoseRoomsAreDormantIsPolledAtTheLongerCap(t *testing.T) {
	var events atomic.Int32
	d, polls := paceDaemon(t, 40*time.Millisecond, 300*time.Millisecond, 400*time.Millisecond, func() int {
		if events.Load() > 0 {
			events.Add(-1)
			return 1
		}
		return 0
	})
	go d.RunSync(nil, 5*time.Millisecond)
	time.Sleep(1500 * time.Millisecond)
	if gap := polls.lastGap(); gap < 250*time.Millisecond {
		t.Errorf("with nothing stored for longer than the quiet time, the last wait was %v; want the longer cap of 300ms", gap)
	}
	events.Store(3)
	d.roomActive("room-pace")
	time.Sleep(30 * time.Millisecond)
	if gap := polls.lastGap(); gap > 20*time.Millisecond {
		t.Errorf("after the room stored an event and the address sent more, the last wait was %v; want the usual interval", gap)
	}
}

// A room becomes active when a session on this machine writes an event into it and when
// a poll stores an event from another machine, which are the two ways a room stores one.
func TestARoomBecomesActiveWhenItStoresAnEventWrittenHereOrReceived(t *testing.T) {
	a := newOfflinePeer(t, "room-pace", "pace")
	b := newOfflinePeer(t, "room-pace", "pace")
	introduce(t, a, b, "room-pace")
	srv := serveCounted(t, b.d, "127.0.0.1:0")
	active := func(p *offlinePeer) bool { return p.d.addrActiveSince(srv.addr, time.Now().Add(-time.Minute)) }
	if active(a) {
		t.Fatal("a room that has stored nothing is active")
	}
	b.say(t, "room-pace", "s", EventUserPrompt, "written on b")
	if !b.d.pace.active["room-pace"].After(time.Now().Add(-time.Minute)) {
		t.Error("a prompt written on this machine left its room inactive")
	}
	if _, err := a.d.pullFrom(srv.addr); err != nil {
		t.Fatal(err)
	}
	if !active(a) {
		t.Error("a poll that stored an event left its room inactive")
	}
}

// Opening a room's view makes the room active, so its addresses are polled at once.
func TestOpeningARoomsViewMakesTheRoomActive(t *testing.T) {
	a := newOfflinePeer(t, "room-pace", "pace")
	srv := httptest.NewServer(a.d.LocalRoutes())
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/stream?room=pace", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("the view's stream answered %s", resp.Status)
	}
	if _, err := bufio.NewReader(resp.Body).ReadString('\n'); err != nil {
		t.Fatal(err)
	}
	a.d.pace.mu.Lock()
	at := a.d.pace.active["room-pace"]
	a.d.pace.mu.Unlock()
	if !at.After(time.Now().Add(-time.Minute)) {
		t.Error("opening the room's view left the room inactive")
	}
}
