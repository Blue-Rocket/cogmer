package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// A nudge from a guest makes the address recorded for its machine the one address of its
// user that is polled, and wakes that address (D-305, a machine nudges the machines it
// shares a room with when it writes an event).
func TestANudgeMakesItsMachinesAddressTheOnePolledForItsUser(t *testing.T) {
	a := newOfflinePeer(t, "room-nudge", "nudge")
	b := newOfflinePeer(t, "room-nudge", "nudge")
	introduce(t, a, b, "room-nudge")
	if err := a.d.members.RecordMachineAddress(b.d.id.PeerID, b.d.id.PeerID, "b-listens-here"); err != nil {
		t.Fatal(err)
	}
	before := time.Now()
	if _, err := b.d.sendNudge(a.addr, "room-nudge"); err != nil {
		t.Fatal(err)
	}
	a.d.pace.mu.Lock()
	last := a.d.pace.lastNudge[b.d.id.PeerID]
	a.d.pace.mu.Unlock()
	if last != "b-listens-here" {
		t.Fatalf("after b's nudge, the address polled for b is %q", last)
	}
	if !a.d.addrWokenSince("b-listens-here", before) {
		t.Error("b's nudge did not wake b's address")
	}
}

// A nudge from a machine this machine knows and has verified, but that is not a guest of
// the room, changes nothing.
func TestANudgeFromAMachineThatIsNotAGuestIsRefused(t *testing.T) {
	a := newOfflinePeer(t, "room-nudge", "nudge")
	b := newOfflinePeer(t, "room-nudge", "nudge")
	stranger := newOfflinePeer(t, "room-nudge", "nudge")
	introduce(t, a, b, "room-nudge")
	// Known and verified both ways, so the handshake passes, and never invited.
	for _, pair := range []struct{ from, to *offlinePeer }{{a, stranger}, {stranger, a}} {
		if err := pair.from.d.members.Allow(pair.to.d.id.PeerID, ""); err != nil {
			t.Fatal(err)
		}
		if err := pair.from.d.members.MarkVerified(pair.to.d.id.PeerID); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.d.members.RecordMachineAddress(stranger.d.id.PeerID, stranger.d.id.PeerID, "stranger"); err != nil {
		t.Fatal(err)
	}
	if final, err := stranger.d.sendNudge(a.addr, "room-nudge"); err == nil || !final {
		t.Error("a nudge from a machine that is not a guest was taken")
	}
	if a.d.setAside("stranger") || len(a.d.pace.lastNudge) != 0 {
		t.Error("a refused nudge was recorded")
	}
}

// A nudge cannot be presented as a sync request, nor a sync request as a nudge, since
// each is signed over a different scope of the room's identifier.
func TestANudgeAndASyncRequestCannotStandForEachOther(t *testing.T) {
	a := newOfflinePeer(t, "room-nudge", "nudge")
	b := newOfflinePeer(t, "room-nudge", "nudge")
	introduce(t, a, b, "room-nudge")
	post := func(path, scope string) int {
		cred, err := signRequestAs(b.d.id, scope, "")
		if err != nil {
			t.Fatal(err)
		}
		body, _ := json.Marshal(syncRequest{Protocol: wireVersion, RoomID: scope,
			PeerID: cred.PeerID, Timestamp: cred.Timestamp, Nonce: cred.Nonce, Signature: cred.Signature})
		req, _ := http.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		w := httptest.NewRecorder()
		a.d.PeerRoutes().ServeHTTP(w, req)
		return w.Code
	}
	if code := post("/sync", nudgeScope+"room-nudge"); code == http.StatusOK {
		t.Error("a nudge's signature was taken as a sync request")
	}
	if code := post("/nudge", "room-nudge"); code == http.StatusNoContent {
		t.Error("a sync request's signature was taken as a nudge")
	}
	if code := post("/nudge", nudgeScope+"room-nudge"); code != http.StatusNoContent {
		t.Errorf("a nudge from a guest was answered %d", code)
	}
}

// Writing an event nudges the room's other machines.
func TestWritingAnEventNudgesTheRoomsOtherMachines(t *testing.T) {
	a := newOfflinePeer(t, "room-nudge", "nudge")
	b := newOfflinePeer(t, "room-nudge", "nudge")
	introduce(t, a, b, "room-nudge")
	if err := b.d.members.RecordMachineAddress(a.d.id.PeerID, a.d.id.PeerID, a.addr); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.RecordMachineAddress(b.d.id.PeerID, b.d.id.PeerID, "b-listens-here"); err != nil {
		t.Fatal(err)
	}
	b.say(t, "room-nudge", "s", EventUserPrompt, "said on b")
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		a.d.pace.mu.Lock()
		last := a.d.pace.lastNudge[b.d.id.PeerID]
		a.d.pace.mu.Unlock()
		if last == "b-listens-here" {
			return
		}
	}
	t.Error("a prompt written on b did not nudge a")
}

// Of a user's machines that have nudged, only the one that nudged last is polled, and one
// set aside is polled again at once when it nudges. A machine that has never nudged, such
// as one on a version that sends no nudges, is polled as before.
func TestOnlyTheMachineOfAUserThatNudgedLastIsPolled(t *testing.T) {
	polls := map[string]*pollTimes{"laptop": {}, "desktop": {}, "old": {}}
	d := syncDaemon(t, "laptop,desktop,old", func(addr string) (int, error) {
		if p := polls[addr]; p != nil {
			p.note()
		}
		return 0, nil
	})
	d.nudgedBy("ana", "laptop")
	d.nudgedBy("ana", "desktop")
	go d.RunSync(nil, 5*time.Millisecond)
	time.Sleep(150 * time.Millisecond)
	if n := polls["laptop"].count(); n != 0 {
		t.Errorf("the machine set aside was polled %d times", n)
	}
	if polls["desktop"].count() < 10 || polls["old"].count() < 10 {
		t.Fatalf("the machine that nudged last was polled %d times and the one that never nudged %d", polls["desktop"].count(), polls["old"].count())
	}
	d.nudgedBy("ana", "laptop")
	time.Sleep(30 * time.Millisecond)
	if polls["laptop"].count() == 0 {
		t.Fatal("the machine set aside nudged and was not polled")
	}
	desktop := polls["desktop"].count()
	time.Sleep(100 * time.Millisecond)
	if n := polls["desktop"].count() - desktop; n > 1 {
		t.Errorf("after the laptop nudged, the desktop was polled %d more times", n)
	}
}

// However often an address nudges, it is polled at most every interval.
func TestAnAddressIsPolledAtMostEveryIntervalHoweverOftenItNudges(t *testing.T) {
	polls := &pollTimes{}
	d := syncDaemon(t, "peer", func(addr string) (int, error) {
		if addr == "peer" {
			polls.note()
		}
		return 0, nil
	})
	d.pace.quietCap, d.pace.dormantCap = time.Hour, time.Hour
	go d.RunSync(nil, 50*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	start := polls.count()
	for i := 0; i < 100; i++ {
		d.nudgedBy("ana", "peer")
		time.Sleep(3 * time.Millisecond)
	}
	if n := polls.count() - start; n > 8 {
		t.Errorf("100 nudges in 300ms polled the address %d times at an interval of 50ms", n)
	}
}

// A turn written on a colleague's machine that is set aside, whose nudge is lost, is not
// stranded while the colleague's machine that nudged last does not answer: the reader
// polls the colleague's other machines again, and takes the turn within a few waits
// (D-305).
func TestATurnOnAMachineSetAsideIsNotStrandedWhenTheLastNudgerIsDown(t *testing.T) {
	var laptopHolds atomic.Bool
	laptopHolds.Store(true) // the turn written on the laptop, whose nudge was lost
	var got atomic.Bool
	d := syncDaemon(t, "laptop,desktop", func(addr string) (int, error) {
		switch addr {
		case "desktop":
			return 0, errors.New("the desktop is closed")
		case "laptop":
			if laptopHolds.Load() && !got.Load() {
				got.Store(true)
				return 1, nil
			}
		}
		return 0, nil
	})
	d.pace.failCap, d.pace.dormantCap = 20*time.Millisecond, 20*time.Millisecond
	d.nudgedBy("ana", "laptop")
	d.nudgedBy("ana", "desktop") // the desktop nudged last, so the laptop is set aside
	go d.RunSync(nil, 5*time.Millisecond)
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if got.Load() {
			return
		}
	}
	t.Error("the turn on the machine set aside was not taken in 2s while the machine that nudged last did not answer")
}

// A nudge that does not reach its receiver is sent again until it does.
func TestANudgeThatDoesNotArriveIsSentAgainUntilTheReceiverAnswers(t *testing.T) {
	a := newOfflinePeer(t, "room-nudge", "nudge")
	b := newOfflinePeer(t, "room-nudge", "nudge")
	introduce(t, a, b, "room-nudge")
	if err := a.d.members.RecordMachineAddress(b.d.id.PeerID, b.d.id.PeerID, "b-listens-here"); err != nil {
		t.Fatal(err)
	}
	if err := b.d.members.RecordMachineAddress(a.d.id.PeerID, a.d.id.PeerID, a.addr); err != nil {
		t.Fatal(err)
	}
	b.d.nudges.retryMin, b.d.nudges.retryMax = 10*time.Millisecond, 20*time.Millisecond
	a.up.Store(false)
	b.d.nudgeAddr(a.addr, "room-nudge")
	time.Sleep(60 * time.Millisecond)
	a.up.Store(true)
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		a.d.pace.mu.Lock()
		last := a.d.pace.lastNudge[b.d.id.PeerID]
		a.d.pace.mu.Unlock()
		if last == "b-listens-here" {
			return
		}
	}
	t.Error("a nudge sent while its receiver did not answer never arrived once it answered")
}

// A nudge is not sent again once its receiver has polled the machine that owed it, since
// that poll took what the nudge was for.
func TestANudgeIsNotSentAgainOnceItsReceiverHasPolled(t *testing.T) {
	a := newOfflinePeer(t, "room-nudge", "nudge")
	b := newOfflinePeer(t, "room-nudge", "nudge")
	introduce(t, a, b, "room-nudge")
	if err := a.d.members.RecordMachineAddress(b.d.id.PeerID, b.d.id.PeerID, "b-listens-here"); err != nil {
		t.Fatal(err)
	}
	if err := b.d.members.RecordMachineAddress(a.d.id.PeerID, a.d.id.PeerID, a.addr); err != nil {
		t.Fatal(err)
	}
	b.d.nudges.retryMin, b.d.nudges.retryMax = 20*time.Millisecond, 20*time.Millisecond
	a.up.Store(false)
	b.d.nudgeAddr(a.addr, "room-nudge")
	time.Sleep(30 * time.Millisecond)
	t.Setenv("COGMER_PEER_ENDPOINT", a.addr) // what a signs as its address when it polls
	if _, err := a.d.pullFrom(b.addr); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	a.up.Store(true)
	time.Sleep(100 * time.Millisecond)
	a.d.pace.mu.Lock()
	last := a.d.pace.lastNudge[b.d.id.PeerID]
	a.d.pace.mu.Unlock()
	if last != "" {
		t.Error("a nudge was sent again after its receiver had polled the machine that owed it")
	}
}

// A machine's own user's other machines are never set aside, so a user's machines keep
// polling one another and pass on what each holds.
func TestAMachinesOwnUsersOtherMachinesAreNeverSetAside(t *testing.T) {
	d := syncDaemon(t, "", func(string) (int, error) { return 0, nil })
	own := d.ownUser()
	d.nudgedBy(own, "my-laptop")
	d.nudgedBy(own, "my-desktop")
	if d.setAside("my-laptop") || d.setAside("my-desktop") {
		t.Error("a machine of this machine's own user was set aside")
	}
	d.nudgedBy("ana", "ana-laptop")
	d.nudgedBy("ana", "ana-desktop")
	if !d.setAside("ana-laptop") {
		t.Error("a colleague's machine that did not nudge last was not set aside")
	}
}

// A nudge is not sent again to an address this machine no longer polls for the room, such
// as one its machine moved away from.
func TestANudgeIsNotSentAgainToAnAddressNoLongerPolledForTheRoom(t *testing.T) {
	a := newOfflinePeer(t, "room-nudge", "nudge")
	b := newOfflinePeer(t, "room-nudge", "nudge")
	introduce(t, a, b, "room-nudge")
	if err := a.d.members.RecordMachineAddress(b.d.id.PeerID, b.d.id.PeerID, "b-listens-here"); err != nil {
		t.Fatal(err)
	}
	if err := b.d.members.RecordMachineAddress(a.d.id.PeerID, a.d.id.PeerID, a.addr); err != nil {
		t.Fatal(err)
	}
	b.d.nudges.retryMin, b.d.nudges.retryMax = 20*time.Millisecond, 20*time.Millisecond
	a.up.Store(false)
	b.d.nudgeAddr(a.addr, "room-nudge")
	time.Sleep(30 * time.Millisecond)
	if err := b.d.members.RecordMachineAddress(a.d.id.PeerID, a.d.id.PeerID, "a-moved-here"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	a.up.Store(true)
	time.Sleep(100 * time.Millisecond)
	a.d.pace.mu.Lock()
	last := a.d.pace.lastNudge[b.d.id.PeerID]
	a.d.pace.mu.Unlock()
	if last != "" {
		t.Error("a nudge was sent again to an address no longer polled for the room")
	}
}
