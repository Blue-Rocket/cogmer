package main

import (
	"strings"
	"testing"
	"time"
)

// A cloud session says its turns are not being shared when every machine it asks refuses
// it, in words that differ from the silence of machines it cannot reach (D-253, a cloud
// session says its turns are not being shared when every machine it asks refuses it).

// cloudWithOneMachine is a user on one machine, a cloud environment added from it, and
// a cloud session started from the environment's value.
func cloudWithOneMachine(t *testing.T) (a, c *machine, cloudKey string) {
	t.Helper()
	a = newMachine(t)
	createUser(t, a, fixedPhrase(), "Dana", "laptop")
	value := addCloudEnv(t, a, "web")
	setup, err := parseCloudValue(value)
	if err != nil {
		t.Fatal(err)
	}
	return a, newCloudMachine(t, value), setup.cloud.Machine
}

func TestACloudSessionAllOfWhoseMachinesRefuseItSaysSoOncePerSession(t *testing.T) {
	a, c, key := cloudWithOneMachine(t)
	room := foundAndSay(t, a, "FROM THE LAPTOP")
	a.d.exchangeSelfLists()
	c.d.cloudRound(time.Now())
	if v := c.d.cloudListsView(); v == nil || v.From == "" || v.Refused {
		t.Fatalf("a session that took its lists is reported as %+v", v)
	}
	joinByName(t, c, room.RoomName, "s1")
	if got := promptOn(t, c, "s1", "p0", "hello"); strings.Contains(got, "refused") {
		t.Fatalf("a session no machine has refused was told it was: %s", got)
	}

	// The user removes the environment, and the machine that did refuses the session.
	var res identityResult
	a.local(t, "/machines/remove", removeRequest{Machine: key, Stolen: true}, &res)
	if res.Status != "removed" {
		t.Fatalf("removing the cloud key: %+v", res)
	}
	c.d.cloud.mu.Lock()
	c.d.cloud.nextFetch = time.Time{}
	c.d.cloud.mu.Unlock()
	c.d.cloudRound(time.Now())

	v := c.d.cloudListsView()
	if v == nil || !v.Refused {
		t.Fatalf("every machine refused the session, and the view says %+v", v)
	}
	first := promptOn(t, c, "s1", "p1", "anything")
	for _, want := range []string{"<cogmer-notice>", "Every machine of the user refused this session's requests", "not being shared"} {
		if !strings.Contains(first, want) {
			t.Errorf("the next prompt does not say %q: %s", want, first)
		}
	}
	if strings.Contains(first, "removed or suspended") == false {
		t.Errorf("the notice does not say what meets this refusal: %s", first)
	}
	if again := promptOn(t, c, "s1", "p2", "and again"); strings.Contains(again, "refused this session") {
		t.Errorf("the session was told twice in one run of refusals: %s", again)
	}
	if got := c.d.cloudRefusalNotice("another-session"); got == "" {
		t.Error("a second session in the environment was not told")
	}

	// A new run of refusals tells a session again.
	c.d.cloud.mu.Lock()
	c.d.cloud.refused, c.d.cloud.nextFetch = false, time.Time{}
	c.d.cloud.mu.Unlock()
	c.d.cloudRound(time.Now())
	if got := c.d.cloudRefusalNotice("s1"); got == "" {
		t.Error("a new run of refusals did not tell the session again")
	}
}

func TestMachinesThatDoNotAnswerAreNotARefusal(t *testing.T) {
	_, c, _ := cloudWithOneMachine(t)
	// the machines the value names are not listening
	c.d.cloud.setup.machines = []cloudMachine{{Machine: testIdentity(t).PeerID, Name: "gone", Endpoint: "127.0.0.1:1"}}
	c.d.cloudRound(time.Now())
	v := c.d.cloudListsView()
	if v == nil || v.Refused || v.Waiting == "" {
		t.Errorf("silence is reported as %+v, want waiting and not refused", v)
	}
	if got := c.d.cloudRefusalNotice("s1"); got != "" {
		t.Errorf("a session whose machines do not answer was told it was refused: %s", got)
	}
}
