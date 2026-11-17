package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A machine the user holds is asked to erase its keys before it is removed, so that a
// removal of it leaves no shared key on it (D-252).

// keysAt lists which of the files a retired machine erases are present in home.
func keysAt(home string) map[string]bool {
	out := map[string]bool{}
	for _, f := range []string{sharedKeyPathIn(home), filepath.Join(home, "identity.key"), filepath.Join(home, "identity.json")} {
		_, err := os.Stat(f)
		out[filepath.Base(f)] = err == nil
	}
	return out
}

// holdKeys makes sure each file a retired machine erases exists, so a test can tell
// that it was erased.
func holdKeys(t *testing.T, m *machine) {
	t.Helper()
	home := m.d.homeDirectory()
	for _, f := range []string{sharedKeyPathIn(home), filepath.Join(home, "identity.key"), filepath.Join(home, "identity.json")} {
		if _, err := os.Stat(f); err != nil {
			if err := os.WriteFile(f, []byte("held"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// retireRequest is a request that m signs for target under tag, as askToRetire builds
// one.
func retireRequest(t *testing.T, m *machine, tag, target string) selfListsRequest {
	t.Helper()
	payload, _ := json.Marshal(retirePayload{Machine: target})
	chain := *m.d.id.Chain
	req := selfListsRequest{
		User: m.d.id.User, Signer: m.d.id.PeerID, Chain: &chain,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Nonce:     base64.RawURLEncoding.EncodeToString([]byte(time.Now().String())),
		Payload:   payload,
	}
	req.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(m.d.id.private,
		selfRequestBytes(tag, req.User, req.Signer, req.Timestamp, req.Nonce, req.Endpoint, req.Payload)))
	return req
}

func postRetire(t *testing.T, to *machine, req selfListsRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	to.d.PeerRoutes().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/self/retire", bytes.NewReader(body)))
	return w
}

func TestAMachineErasesItsKeysWhenAnotherOfTheUsersAsksIt(t *testing.T) {
	a, b := twoMachines(t)
	exited := make(chan struct{}, 1)
	b.d.afterRetire = func() { exited <- struct{}{} }
	holdKeys(t, b)
	if err := a.d.askToRetire(b.d.id.PeerID); err != nil {
		t.Fatalf("the machine was not asked: %v", err)
	}
	for f, held := range keysAt(b.d.homeDirectory()) {
		if held {
			t.Errorf("the machine still holds %s", f)
		}
	}
	select {
	case <-exited:
	case <-time.After(2 * time.Second):
		t.Error("the machine did not end after it erased its keys")
	}
}

func TestAMachineErasesNothingForARequestItShouldNotObey(t *testing.T) {
	a, b := twoMachines(t)
	// a stranger: a machine of another user, with a chain of its own
	c := newMachine(t)
	var other Phrase
	for i := range other {
		other[i] = byte(0x50 + i)
	}
	createUser(t, c, other, "Eve", "elsewhere")
	exited := 0
	b.d.afterRetire = func() { exited++ }
	holdKeys(t, b)
	self := b.d.id.PeerID

	cases := map[string]struct {
		req  selfListsRequest
		want int
	}{
		"a request signed for another route":   {retireRequest(t, a, selfListsTag, self), http.StatusUnauthorized},
		"a request that names another machine": {retireRequest(t, a, selfRetireTag, a.d.id.PeerID), http.StatusBadRequest},
		"a machine asking itself":              {retireRequest(t, b, selfRetireTag, self), http.StatusUnauthorized},
		"a machine of another user":            {retireRequest(t, c, selfRetireTag, self), http.StatusUnauthorized},
	}
	// the cloud key itself, which holds no shared key, with a chain that is live
	value := addCloudEnv(t, a, "cloud-test")
	setup, err := parseCloudValue(value)
	if err != nil {
		t.Fatal(err)
	}
	cloudAsk := retireRequest(t, a, selfRetireTag, self)
	cloudAsk.Signer = PeerIDFromPublic(setup.key.Public().(ed25519.PublicKey))
	cloudAsk.Chain = &Chain{Shared: setup.shared, Machine: setup.cloud}
	cloudAsk.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(setup.key,
		selfRequestBytes(selfRetireTag, cloudAsk.User, cloudAsk.Signer, cloudAsk.Timestamp, cloudAsk.Nonce, cloudAsk.Endpoint, cloudAsk.Payload)))
	// a cloud session, whose chain is live and ends in a session key
	session := newCloudMachine(t, value)
	sessionAsk := retireRequest(t, session, selfRetireTag, self)
	cases["a cloud session"] = struct {
		req  selfListsRequest
		want int
	}{sessionAsk, http.StatusUnauthorized}
	cases["a cloud key"] = struct {
		req  selfListsRequest
		want int
	}{cloudAsk, http.StatusUnauthorized}
	noChain := retireRequest(t, a, selfRetireTag, self)
	noChain.Chain = nil
	cases["a request with no chain"] = struct {
		req  selfListsRequest
		want int
	}{noChain, http.StatusUnauthorized}
	for name, c := range cases {
		if w := postRetire(t, b, c.req); w.Code != c.want {
			t.Errorf("%s: answered %d, want %d", name, w.Code, c.want)
		}
		for f, held := range keysAt(b.d.homeDirectory()) {
			if !held {
				t.Errorf("%s: the machine erased %s", name, f)
			}
		}
	}
	if exited != 0 {
		t.Errorf("the machine ended after a request it refused")
	}
	// and the right one is obeyed
	if w := postRetire(t, b, retireRequest(t, a, selfRetireTag, self)); w.Code != http.StatusOK || exited != 1 {
		t.Errorf("a request from a machine of the user is answered %d, ended %d times", w.Code, exited)
	}
}

func TestThePageAsksAMachineToEraseItsKeysAndSaysSoWhenItCannot(t *testing.T) {
	a, b := twoMachines(t)
	b.d.afterRetire = func() {}
	holdKeys(t, b)
	ask := func(machine string) retireAskResponse {
		var res retireAskResponse
		a.local(t, "/machines/retire/ask", retireAskRequest{Machine: machine}, &res)
		return res
	}
	if res := ask(b.d.id.PeerID); res.Status != "erased" || res.Error != "" {
		t.Fatalf("the page was answered %+v", res)
	}
	// this machine cannot be asked, and a machine that is not the user's cannot either
	if res := ask(a.d.id.PeerID); res.Error == "" {
		t.Error("the page asked the machine it runs on to erase its keys")
	}
	if res := ask(testIdentity(t).PeerID); res.Error == "" {
		t.Error("the page asked a machine that is not the user's")
	}
	// a machine that does not answer is reported as one that did not
	b.d.afterRetire = func() {}
	if err := a.d.members.SetMachineEndpoint(b.d.id.PeerID, a.d.id.User, "127.0.0.1:1"); err != nil {
		t.Fatal(err)
	}
	if res := ask(b.d.id.PeerID); res.Error == "" || res.Status != "" {
		t.Errorf("a machine nobody could reach was reported as %+v", res)
	}
}
