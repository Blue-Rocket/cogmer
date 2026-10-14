package main

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type pullLog struct {
	mu    sync.Mutex
	calls map[string]int
}

func (p *pullLog) note(addr string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.calls == nil {
		p.calls = map[string]int{}
	}
	p.calls[addr]++
}

func (p *pullLog) count(addr string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls[addr]
}

func syncDaemon(t *testing.T, targets string, pull func(string) (int, error)) *Daemon {
	t.Helper()
	m := testMembership(t)
	t.Setenv("COGMER_PEERS", targets)
	d := &Daemon{id: testIdentity(t), members: m, pull: pull}
	t.Cleanup(d.closeStores)
	return d
}

// One peer that does not answer held every poll for as long as its request's timeout, so a
// colleague who was reachable received each turn half a minute after it was written.
func TestAPeerThatDoesNotAnswerDoesNotDelayOneThatDoes(t *testing.T) {
	var seen pullLog
	release := make(chan struct{})
	defer close(release)
	d := syncDaemon(t, "hung-peer,healthy-peer", func(addr string) (int, error) {
		seen.note(addr)
		if addr == "hung-peer" {
			<-release // a request that waits out its timeout
			return 0, errors.New("timeout")
		}
		return 0, nil
	})
	go d.RunSync(nil, 5*time.Millisecond)
	time.Sleep(400 * time.Millisecond)
	if got := seen.count("healthy-peer"); got < 20 {
		t.Errorf("the peer that answered was polled %d times in 400ms with another waiting on a timeout; one loop would have polled it once", got)
	}
	if got := seen.count("hung-peer"); got != 1 {
		t.Errorf("the peer that does not answer has %d polls in flight", got)
	}
}

func TestAPeerThatFailsIsPolledLessOftenAndAtTheUsualIntervalOnceItAnswers(t *testing.T) {
	var seen pullLog
	var answering atomic.Bool
	d := syncDaemon(t, "peer", func(addr string) (int, error) {
		seen.note(addr)
		if answering.Load() {
			return 0, nil
		}
		return 0, errors.New("refused")
	})
	go d.RunSync(nil, 10*time.Millisecond)
	time.Sleep(500 * time.Millisecond)
	failing := seen.count("peer")
	if failing > 12 || failing < 3 {
		t.Errorf("a peer that fails was polled %d times in 500ms at an interval of 10ms, want a few, backing off", failing)
	}
	answering.Store(true)
	time.Sleep(700 * time.Millisecond) // lets the longest wait pass
	before := seen.count("peer")
	time.Sleep(300 * time.Millisecond)
	if got := seen.count("peer") - before; got < 15 {
		t.Errorf("a peer that answers again was polled %d times in 300ms, want the usual interval", got)
	}
	// and a new run of failures starts from the short wait again, not from where the last one ended
	answering.Store(false)
	before = seen.count("peer")
	time.Sleep(150 * time.Millisecond)
	if got := seen.count("peer") - before; got < 3 {
		t.Errorf("a peer that failed again was polled %d times in 150ms, so its earlier failures were still counted", got)
	}
}

func TestAnAddressThatIsNoLongerWorthAskingIsNotPolled(t *testing.T) {
	var seen pullLog
	d := syncDaemon(t, "stays,goes", func(addr string) (int, error) { seen.note(addr); return 0, nil })
	go d.RunSync(nil, 5*time.Millisecond)
	time.Sleep(150 * time.Millisecond)
	if seen.count("goes") == 0 || seen.count("stays") == 0 {
		t.Fatal("the peers were not polled at all")
	}
	t.Setenv("COGMER_PEERS", "stays")
	time.Sleep(100 * time.Millisecond) // the targets are read again, and the loop is told to stop
	gone := seen.count("goes")
	time.Sleep(150 * time.Millisecond)
	if after := seen.count("goes"); after > gone+1 {
		t.Errorf("an address that was removed was polled %d more times", after-gone)
	}
	if seen.count("stays") < 20 {
		t.Errorf("the address that stayed was polled %d times", seen.count("stays"))
	}
	t.Setenv("COGMER_PEERS", "stays,goes")
	time.Sleep(150 * time.Millisecond)
	if seen.count("goes") <= gone+1 {
		t.Error("an address that came back was not polled again")
	}
}
