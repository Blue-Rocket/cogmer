//go:build docker

package main

import (
	"strings"
	"testing"
	"time"
)

// Cloud sessions in containers (D-216, D-245, D-246, D-247): a cloud environment added
// on alice's machines page, and a third container started with nothing but the value
// in COGMER_CLOUD_KEY, which takes alice's lists from her machine, joins her room by
// name and speaks in it as alice.

// addCloudBox adds a cloud environment named name on m's machines page and starts a
// container called name with the value the page showed. It returns the container and
// the cloud key the value holds.
func addCloudBox(t *testing.T, c *cluster, m *box, name string) (*box, string) {
	t.Helper()
	var res cloudAddResponse
	m.mustPost("/machines/cloud/add", cloudAddRequest{Name: name}, &res)
	if res.Error != "" || !strings.HasPrefix(res.Value, cloudValuePrefix) {
		t.Fatalf("%s's page added no cloud environment: %s", m.name, res.Error)
	}
	cloud := c.startWith(name, harnessImage(t, imageWork), cloudKeyVar+"="+res.Value)
	return cloud, res.Machine
}

// cloudJoinByName has session on m join room by its name, retrying while m has not yet
// fetched the user's lists, and returns the command's last output.
func cloudJoinByName(t *testing.T, m *box, session, room string, within time.Duration) {
	t.Helper()
	out := ""
	eventually(t, within, m.name+" joining "+room+" by name", func() (bool, string) {
		out, _ = m.cli(session, "join", room)
		return strings.Contains(out, "joined "+room), out
	})
}

func TestContainerCloudSessionSpeaksAsItsUserUntilItsKeyIsRemoved(t *testing.T) {
	c := newCluster(t)
	r := pairedRoom(t, c, harnessImage(t, imageWork), true)
	handOver(t, r.name, r.alice, r.aliceS, r.bob, r.bobS, "alice says the parser is done")

	cloud, cloudKey := addCloudBox(t, c, r.alice, "alice-web")
	cloudS := "session-cloud-1"
	cloudJoinByName(t, cloud, cloudS, r.name, 60*time.Second)

	// The session's turn reaches bob as alice's, from her cloud environment, and
	// reaches alice's machine as her own (D-232, D-233).
	got := handOver(t, r.name, cloud, cloudS, r.bob, r.bobS, "alice in the cloud says the build is green")
	tr := mustSee(t, "bob", got, "alice in the cloud says the build is green")
	if tr.Label != "alice" || !tr.Verified || tr.YourUser {
		t.Fatalf("bob was given the cloud session's turn labeled %q, verified %v, as his own %v", tr.Label, tr.Verified, tr.YourUser)
	}
	if tr.Machine == nil || tr.Machine.Name != "alice-web" {
		t.Fatalf("bob was given the cloud session's turn with machine %+v", tr.Machine)
	}
	eventually(t, 60*time.Second, "the cloud turn reaching alice's machine", func() (bool, string) {
		return r.alice.viewHas(r.name, "alice in the cloud says the build is green"), "not in the view"
	})
	tr = mustSee(t, "alice", r.alice.turn(r.aliceS, "what did I do in the cloud?", "noted"), "alice in the cloud says the build is green")
	if !tr.YourUser || tr.Label != "you" {
		t.Fatalf("alice's machine was given her cloud turn labeled %q, as her own %v", tr.Label, tr.YourUser)
	}
	// And the cloud session is given bob's turn.
	got = handOver(t, r.name, r.bob, r.bobS, cloud, cloudS, "bob says welcome to the cloud")
	mustSee(t, "the cloud session", got, "bob says welcome to the cloud")

	// Nothing the session holds or prints carries the value or its key.
	if out := cloud.mustCLI(cloudS, "whoami"); strings.Contains(out, cloudValuePrefix) {
		t.Fatal("whoami in the cloud session prints the value")
	}
	if strings.Contains(cloud.logs(), cloudValuePrefix) {
		t.Fatal("the cloud session's daemon logs the value")
	}

	// Alice removes the cloud environment on her page as not stolen, so the page
	// proposes the last of the session's turns her machine holds (D-250). Once bob holds
	// the removal, bob still counts the session's earlier turn and refuses its next one.
	var p removeProposeResponse
	r.alice.mustPost("/machines/remove/propose", removeProposeRequest{Machine: cloudKey}, &p)
	if !p.Cloud || len(p.Sessions) != 1 || p.Sessions[0].Proposed == 0 || p.Sessions[0].Started == "" {
		t.Fatalf("alice's page proposed %+v for her cloud environment", p)
	}
	var res identityResult
	r.alice.mustPost("/machines/remove", removeRequest{Machine: cloudKey}, &res)
	if res.Status != "removed" {
		t.Fatalf("alice's page did not remove the cloud key: %+v", res)
	}
	eventually(t, 60*time.Second, "bob refusing the removed cloud key", func() (bool, string) {
		cloud.turn(cloudS, "probe after removal", "noted")
		return strings.Contains(r.bob.logs(), "has been removed"), "bob's log does not say the machine is removed"
	})
	cloud.turn(cloudS, "alice in the cloud after removal", "noted")
	time.Sleep(10 * time.Second)
	if r.bob.viewHas(r.name, "alice in the cloud after removal") {
		t.Fatal("bob took a turn from a removed cloud key")
	}
	if r.alice.viewHas(r.name, "alice in the cloud after removal") {
		t.Fatal("alice's machine took a turn from a removed cloud key")
	}
	for _, who := range []*box{r.bob, r.alice} {
		found := false
		for _, e := range who.view(r.name) {
			if e.Content == "alice in the cloud says the build is green" {
				found = true
				if e.Marked != "" {
					t.Errorf("%s marks the retired environment's earlier turn: %s", who.name, e.Marked)
				}
			}
		}
		if !found {
			t.Errorf("%s no longer shows the retired environment's earlier turn", who.name)
		}
	}
}

func TestContainerCloudSessionWaitsForItsUsersMachineAndCatchesUp(t *testing.T) {
	c := newCluster(t)
	r := pairedRoom(t, c, harnessImage(t, imageWork), true)
	handOver(t, r.name, r.alice, r.aliceS, r.bob, r.bobS, "alice says the parser is done")

	var res cloudAddResponse
	r.alice.mustPost("/machines/cloud/add", cloudAddRequest{Name: "alice-web"}, &res)
	r.alice.disconnect()
	cloud := c.startWith("alice-web", harnessImage(t, imageWork), cloudKeyVar+"="+res.Value)
	cloudS := "session-cloud-1"

	// With none of alice's machines reachable, the session knows no room, and its hooks
	// add nothing and show nothing.
	time.Sleep(5 * time.Second)
	if out, _ := cloud.cli(cloudS, "join", r.name); strings.Contains(out, "joined") {
		t.Fatalf("the cloud session joined a room with no lists:\n%s", out)
	}
	if given := cloud.turn(cloudS, "anything?", "no"); strings.TrimSpace(given) != "" {
		t.Fatalf("a session in no room was given:\n%s", given)
	}

	r.alice.connect()
	cloudJoinByName(t, cloud, cloudS, r.name, 90*time.Second)
	got := handOver(t, r.name, cloud, cloudS, r.bob, r.bobS, "alice in the cloud, once her laptop woke")
	mustSee(t, "bob", got, "alice in the cloud, once her laptop woke")
}
