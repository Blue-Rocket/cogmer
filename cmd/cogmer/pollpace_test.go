package main

import (
	"bufio"
	"context"
	"errors"
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
	d.roomStored("room-pace", true)
	time.Sleep(50 * time.Millisecond)
	if after := polls.count(); after <= before {
		t.Fatalf("the room stored an event and its address was not polled within 50ms (%d polls before, %d after)", before, after)
	}
	time.Sleep(60 * time.Millisecond) // the wake's poll, and waits of 20 and 40ms after it
	if gap := polls.lastGap(); gap > 100*time.Millisecond {
		t.Errorf("after the wake the address waited %v between polls; want the waits to start again from 20ms", gap)
	}
}

// An event in a room an address is not asked about leaves that address's wait as it was.
func TestAnEventInAnotherRoomLeavesAnAddressPolledSlowly(t *testing.T) {
	d, polls := paceDaemon(t, time.Second, time.Second, time.Hour, func() int { return 0 })
	go d.RunSync(nil, 20*time.Millisecond)
	time.Sleep(900 * time.Millisecond)
	before := polls.count()
	d.roomStored("room-other", true)
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
	d.roomStored("room-pace", true)
	time.Sleep(30 * time.Millisecond)
	if gap := polls.lastGap(); gap > 20*time.Millisecond {
		t.Errorf("after the room stored an event and the address sent more, the last wait was %v; want the usual interval", gap)
	}
}

// An event a session on this machine writes into a room wakes the room's addresses, and
// one a poll brings from another machine is recorded without waking them.
func TestAnEventWrittenHereWakesARoomsAddressesAndOneReceivedDoesNot(t *testing.T) {
	a := newOfflinePeer(t, "room-pace", "pace")
	b := newOfflinePeer(t, "room-pace", "pace")
	introduce(t, a, b, "room-pace")
	srv := serveCounted(t, b.d, "127.0.0.1:0")
	minuteAgo := time.Now().Add(-time.Minute)
	b.say(t, "room-pace", "s", EventUserPrompt, "written on b")
	if !b.d.addrWokenSince(a.addr, minuteAgo) {
		t.Error("a prompt written on this machine did not wake its room's addresses")
	}
	if _, err := a.d.pullFrom(srv.addr); err != nil {
		t.Fatal(err)
	}
	a.d.pace.mu.Lock()
	stored := a.d.pace.stored["room-pace"]
	a.d.pace.mu.Unlock()
	if !stored.After(minuteAgo) {
		t.Error("a poll that stored an event did not record that its room stored one")
	}
	if a.d.addrWokenSince(srv.addr, minuteAgo) {
		t.Error("an event received from one machine woke the room's addresses")
	}
}

// An event received from one address leaves another address of the same room polled
// slowly, since that address has nothing new because of it.
func TestAnEventReceivedFromOneAddressLeavesAnotherPolledSlowly(t *testing.T) {
	polls := map[string]*pollTimes{"peer": {}, "other": {}}
	d := syncDaemon(t, "peer,other", func(addr string) (int, error) {
		if p := polls[addr]; p != nil { // a loop outliving this test may poll another test's addresses
			p.note()
		}
		return 0, nil
	})
	d.pace.quietCap, d.pace.dormantCap, d.pace.dormantAfter = time.Second, time.Second, time.Hour
	d.askRooms = func(string, []Room) []Room { return []Room{{RoomID: "room-pace"}} }
	go d.RunSync(nil, 20*time.Millisecond)
	time.Sleep(900 * time.Millisecond)
	before := polls["other"].count()
	d.roomStored("room-pace", false)
	time.Sleep(50 * time.Millisecond)
	if after := polls["other"].count(); after != before {
		t.Errorf("an event received from another address polled this one %d more times", after-before)
	}
}

// A machine that does not answer is polled up to every syncBackoffCap while its rooms are
// active, and up to the dormant cap once they are dormant.
func TestAMachineThatDoesNotAnswerIsPolledAtTheDormantCapOnceItsRoomsAreDormant(t *testing.T) {
	polls := &pollTimes{}
	d := syncDaemon(t, "peer", func(string) (int, error) { polls.note(); return 0, errors.New("refused") })
	d.pace.failCap, d.pace.dormantCap, d.pace.dormantAfter = 40*time.Millisecond, 300*time.Millisecond, 400*time.Millisecond
	d.askRooms = func(string, []Room) []Room { return []Room{{RoomID: "room-pace"}} }
	go d.RunSync(nil, 5*time.Millisecond)
	time.Sleep(350 * time.Millisecond)
	if gap := polls.lastGap(); gap > 80*time.Millisecond {
		t.Errorf("before its rooms were dormant, a machine that does not answer waited %v; want the cap of 40ms", gap)
	}
	time.Sleep(1200 * time.Millisecond)
	if gap := polls.lastGap(); gap < 250*time.Millisecond {
		t.Errorf("once its rooms were dormant, a machine that does not answer waited %v; want the dormant cap of 300ms", gap)
	}
}

// Opening a room's view wakes its addresses, and records no event, so a room that is only
// looked at still becomes dormant.
func TestOpeningARoomsViewWakesItsAddresses(t *testing.T) {
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
	woken, stored := a.d.pace.woken["room-pace"], a.d.pace.stored["room-pace"]
	a.d.pace.mu.Unlock()
	if !woken.After(time.Now().Add(-time.Minute)) {
		t.Error("opening the room's view did not wake its addresses")
	}
	if !stored.IsZero() {
		t.Error("opening the room's view recorded that the room stored an event")
	}
}

// A colleague whose machine answered its last poll is online while it is polled slowly
// in a quiet room, and one whose polls fail after it answered, or that has not answered
// for longer than the longest wait between polls, is not (D-304; D-194, a colleague this
// machine cannot reach is named in the room status and the view).
func TestAColleaguePolledSlowlyInAQuietRoomIsOnline(t *testing.T) {
	d := syncDaemon(t, "", func(string) (int, error) { return 0, nil })
	now := time.Now()
	d.peerSeen = map[string]*peerState{
		"quiet":   {lastSeen: now.Add(-40 * time.Second)},
		"failing": {lastSeen: now.Add(-40 * time.Second), downSince: now.Add(-20 * time.Second)},
		"gone":    {lastSeen: now.Add(-100 * time.Second)},
	}
	online := map[string]bool{}
	for _, p := range d.statusOf([]string{"quiet", "failing", "gone"}) {
		online[p.addr] = p.Online
	}
	if !online["quiet"] {
		t.Error("a machine that answered its last poll 40s ago, in a quiet room, is shown as not online")
	}
	if online["failing"] {
		t.Error("a machine whose polls have failed for 20s since it last answered is shown as online")
	}
	if online["gone"] {
		t.Error("a machine that has not answered for longer than the longest wait between polls is shown as online")
	}
}

// decidePoll, at the real interval and caps, gives each rule of when an address is polled
// next (D-304, D-305).
func TestDecidePollGivesEachRuleOfWhenAnAddressIsPolledNext(t *testing.T) {
	p := pollPacing{every: time.Second, quietCap: quietPollCap, failCap: syncBackoffCap, dormantCap: dormantPollCap}
	asleep := func() bool { return true }
	awake := func() bool { return false }
	unasked := func() bool { t.Helper(); t.Error("dormancy was asked before a wait reached its cap"); return false }
	s := func(last pollOutcome, failures, empty int) pollState {
		return pollState{last: last, failures: failures, empty: empty}
	}
	for _, c := range []struct {
		name     string
		s        pollState
		f        pollFacts
		wait     time.Duration
		wakeable bool
		aside    bool
	}{
		{"a machine set aside is not polled", s(pollBroughtNothing, 0, 3), pollFacts{setAside: true, dormant: unasked}, 0, false, true},
		{"the first poll is at once", s(notPolled, 0, 0), pollFacts{dormant: unasked}, 0, false, false},
		{"a poll that brought events is followed after the interval", s(pollBroughtEvents, 0, 0), pollFacts{dormant: unasked}, time.Second, false, false},
		{"one empty answer doubles the wait, and a wake ends it", s(pollBroughtNothing, 0, 1), pollFacts{dormant: unasked}, 2 * time.Second, true, false},
		{"four empty answers wait 16s, below the quiet cap", s(pollBroughtNothing, 0, 4), pollFacts{dormant: unasked}, 16 * time.Second, true, false},
		{"five empty answers stop at the quiet cap", s(pollBroughtNothing, 0, 5), pollFacts{dormant: awake}, quietPollCap, true, false},
		{"dormant rooms raise the cap to a minute", s(pollBroughtNothing, 0, 9), pollFacts{dormant: asleep}, dormantPollCap, true, false},
		{"the first failure waits the interval, and a wake does not end it", s(pollFailedNow, 1, 0), pollFacts{dormant: unasked}, time.Second, false, false},
		{"failures stop at the failure cap", s(pollFailedNow, 9, 0), pollFacts{dormant: awake}, syncBackoffCap, false, false},
		{"failures in dormant rooms wait up to a minute", s(pollFailedNow, 9, 0), pollFacts{dormant: asleep}, dormantPollCap, false, false},
	} {
		d := decidePoll(c.s, c.f, p)
		if d.setAside != c.aside || d.wait != c.wait || d.wakeable != c.wakeable || d.nudgeable != (c.wakeable || c.s.last == pollFailedNow && !c.aside) {
			t.Errorf("%s: got set aside %v, wait %v, wakeable %v; want %v, %v, %v",
				c.name, d.setAside, d.wait, d.wakeable, c.aside, c.wait, c.wakeable)
		}
		if !c.aside && d.least != time.Second {
			t.Errorf("%s: two polls may be %v apart; want the interval", c.name, d.least)
		}
	}
}

// A poll's outcome and a wake change the loop's state as decidePoll expects.
func TestAPollsOutcomeAndAWakeChangeTheLoopsState(t *testing.T) {
	var s pollState
	s.polled(0, nil)
	s.polled(0, nil)
	if s.last != pollBroughtNothing || s.empty != 2 {
		t.Fatalf("two empty answers left %+v", s)
	}
	s.woken()
	if s.empty != 0 {
		t.Errorf("a wake left %d empty answers counted", s.empty)
	}
	s.polled(0, errors.New("refused"))
	s.polled(0, errors.New("refused"))
	if s.last != pollFailedNow || s.failures != 2 || s.empty != 0 {
		t.Errorf("two failures left %+v", s)
	}
	s.polled(3, nil)
	if s.last != pollBroughtEvents || s.failures != 0 || s.empty != 0 {
		t.Errorf("a poll that brought events left %+v", s)
	}
}

// A wake that comes while a poll is under way ends the wait after that poll, as one that
// comes during the wait does.
func TestAWakeDuringAPollEndsTheWaitAfterIt(t *testing.T) {
	var d *Daemon
	var polls *pollTimes
	var wakeDuring atomic.Bool
	d, polls = paceDaemon(t, time.Second, time.Second, time.Hour, func() int {
		if wakeDuring.Swap(false) {
			d.roomStored("room-pace", true) // a prompt written here while the poll is under way
		}
		return 0
	})
	go d.RunSync(nil, 20*time.Millisecond)
	time.Sleep(900 * time.Millisecond) // the waits have grown past 300ms
	wakeDuring.Store(true)
	for deadline := time.Now().Add(2 * time.Second); wakeDuring.Load() && time.Now().Before(deadline); time.Sleep(time.Millisecond) {
	}
	woke := polls.count()
	time.Sleep(150 * time.Millisecond)
	if polls.count() == woke {
		t.Error("a wake during a poll did not end the wait after it")
	}
}
