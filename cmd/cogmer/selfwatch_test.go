package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// A daemon whose overlay stops being reachable looks like a healthy one until
// somebody tries it, which in one case was sixteen minutes and could have been
// weeks. The watcher asks on a timer and says when the answer changes (D-197). It
// reports and restarts nothing.

func scripted(results ...SelfCheck) (func(context.Context) SelfCheck, *int32) {
	var n int32
	return func(context.Context) SelfCheck {
		i := int(atomic.AddInt32(&n, 1)) - 1
		if i >= len(results) {
			i = len(results) - 1
		}
		return results[i]
	}, &n
}

var (
	failedCheck = SelfCheck{Overlay: true, Relay: "sfo", Error: "context deadline exceeded"}
	passedCheck = SelfCheck{Overlay: true, Relay: "sfo", Reached: true, TookMs: 300}
)

func TestOneFailedCheckIsConfirmedBeforeItCounts(t *testing.T) {
	d, _ := testDaemon(t)
	d.selfCheck, _ = scripted(failedCheck, passedCheck)
	buf := captureLog(t)
	d.watchDial(make(chan struct{}), time.Millisecond)
	if s := d.selfStatus(); s == nil || s.Failing {
		t.Errorf("a failure that passed on the second look was recorded: %+v", s)
	}
	if strings.Contains(buf.String(), "cannot be reached") {
		t.Errorf("a failure that passed on the second look was reported:\n%s", buf.String())
	}
}

func TestAConfirmedFailureIsReportedOnceAndItsEndIsSaid(t *testing.T) {
	d, _ := testDaemon(t)
	d.selfCheck, _ = scripted(failedCheck, failedCheck, failedCheck, failedCheck, passedCheck)
	buf := captureLog(t)
	stop := make(chan struct{})

	d.watchDial(stop, time.Millisecond)
	s := d.selfStatus()
	if s == nil || !s.Failing || !strings.Contains(s.Error, "deadline exceeded") {
		t.Fatalf("two failures in a row were not recorded: %+v", s)
	}
	d.watchDial(stop, time.Millisecond)
	if n := strings.Count(buf.String(), "cannot be reached through its relay"); n != 1 {
		t.Errorf("the failure was reported %d times, want once until the slow repeat:\n%s", n, buf.String())
	}
	for _, want := range []string{"colleagues cannot reach it", "/cogmer:self-restart"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("the report lacks %q:\n%s", want, buf.String())
		}
	}

	d.watchDial(stop, time.Millisecond)
	if !strings.Contains(buf.String(), "relay: reachable again after") {
		t.Errorf("the end of the failure was not said:\n%s", buf.String())
	}
	if s := d.selfStatus(); s == nil || s.Failing {
		t.Errorf("a daemon that passed is still failing: %+v", s)
	}
}

func TestAMissingRelayIsNoticedAfterSeveralChecks(t *testing.T) {
	d, _ := testDaemon(t)
	d.selfCheck, _ = scripted(passedCheck)
	var relay atomic.Value
	relay.Store("")
	d.relayNow = func() string { return relay.Load().(string) }
	d.watch.cheapEvery = 10 * time.Millisecond
	buf := captureLog(t)

	d.watchRelay()
	d.watchRelay()
	if s := d.selfStatus(); s.Failing {
		t.Fatalf("two checks with no relay counted: %+v", s)
	}
	if strings.Contains(buf.String(), "has not been attached") {
		t.Fatalf("a relay missing for two checks was reported, which a daemon that has just started would trigger:\n%s", buf.String())
	}
	time.Sleep(45 * time.Millisecond)
	d.watchRelay()
	if s := d.selfStatus(); !s.Failing || s.Detached == "" {
		t.Errorf("no relay for several checks was not recorded: %+v", s)
	}
	if !strings.Contains(buf.String(), "has not been attached to a relay") {
		t.Errorf("it was not said:\n%s", buf.String())
	}
	relay.Store("sfo")
	d.watchRelay()
	if !strings.Contains(buf.String(), "attached to the relay again") || d.selfStatus().Failing {
		t.Errorf("attaching again was not said or still counts as failing:\n%s", buf.String())
	}
}

func TestTheWatcherAsksOnATimerAndStops(t *testing.T) {
	d, _ := testDaemon(t)
	check, calls := scripted(passedCheck)
	d.selfCheck = check
	d.relayNow = func() string { return "sfo" }
	d.watch.every, d.watch.cheapEvery, d.watch.confirm = 40*time.Millisecond, 10*time.Millisecond, time.Millisecond
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() { d.RunSelfWatch(stop); close(done) }()

	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(calls) < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := atomic.LoadInt32(calls); n < 2 {
		t.Errorf("the watcher asked %d times in two seconds, so nothing asks on a timer", n)
	}
	close(stop)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("the watcher kept running after it was told to stop")
	}
}

func TestADaemonWithNoOverlayHasNothingToWatch(t *testing.T) {
	d, _ := testDaemon(t)
	done := make(chan struct{})
	go func() { d.RunSelfWatch(make(chan struct{})); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("a daemon with no overlay started a watcher")
	}
	if d.selfStatus() != nil {
		t.Error("a daemon with no overlay reports a reachability status")
	}
}

// What a person is shown: the room status leads with it, and the check says how
// long.
func TestAnUnreachableMachineIsSaidFirstInTheRoomStatusAndInSelfStatus(t *testing.T) {
	d, _ := testDaemon(t)
	d.selfCheck, _ = scripted(failedCheck)
	d.watch.every, d.watch.cheapEvery = time.Hour, time.Minute
	d.noteSelfResult(failedCheck)
	d.watch.mu.Lock()
	d.watch.failSince = time.Now().Add(-40 * time.Minute)
	d.watch.mu.Unlock()

	rec := httptest.NewRecorder()
	d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/peers", nil))
	if !strings.Contains(rec.Body.String(), `"failing":true`) || !strings.Contains(rec.Body.String(), `"failingFor":"40m"`) {
		t.Fatalf("/peers does not carry the watcher's status: %s", rec.Body.String())
	}

	var out strings.Builder
	renderGuests(&out, "moonlit-oasis", []guestRow{{Name: "wonka", ID: "ed25519:" + strings.Repeat("a", 43), Self: true}}, nil,
		&uiSelf{Failing: true, FailingFor: "40m", Error: "context deadline exceeded"}, true)
	text := out.String()
	for _, want := range []string{"THIS MACHINE CANNOT BE REACHED", "for 40m", "no answer", "do not receive", "/cogmer:self-restart"} {
		if !strings.Contains(text, want) {
			t.Errorf("the room status lacks %q:\n%s", want, text)
		}
	}
	if strings.Index(text, "THIS MACHINE") > strings.Index(text, "wonka") {
		t.Errorf("the unreachable machine is said after the guests, not before:\n%s", text)
	}

	overlayAnswers(t, SelfCheck{Overlay: true, Relay: "sfo", Error: "context deadline exceeded", FailingFor: "40m"}, true)
	_, why, _ := checkReachable("tc://tcpSOMELONGNEGOTIATEDVALUE")
	if !strings.Contains(why, "failed this check for 40m") {
		t.Errorf("self-status does not say how long: %q", why)
	}
}
