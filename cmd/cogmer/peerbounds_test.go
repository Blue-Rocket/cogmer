package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

// pushAs sends one record to d's push route as id would.
func pushAs(t *testing.T, d *Daemon, id *Identity, kind string, record any) *httptest.ResponseRecorder {
	t.Helper()
	raw, _ := json.Marshal(record)
	c, err := signRequestAs(id, identityPushScope, "")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(identityPushRequest{Protocol: wireVersion, Kind: kind, Record: raw, RoomID: identityPushScope,
		PeerID: c.PeerID, Timestamp: c.Timestamp, Nonce: c.Nonce, Signature: c.Signature, Signer: c.Signer, Chain: c.Chain})
	w := httptest.NewRecorder()
	d.PeerRoutes().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/identity/push", bytes.NewReader(body)))
	return w
}

// A pushed record no key signed is refused before it is kept, and a machine keeps a
// bounded number of pushed records of one user, so a colleague cannot fill its disk.
func TestThePushRouteKeepsOnlySignedRecordsAndABoundedNumber(t *testing.T) {
	ana, ben := newTestUser(t), newTestUser(t)
	a := newUserPeer(t, trustRoom, trustRoomName, ana.machine(t, "laptop"), ana)
	b := newUserPeer(t, trustRoom, trustRoomName, ben.machine(t, "desktop"), ben)
	introduceUsers(t, a, b, trustRoom)

	forged, err := SignRemoval(ben.shared, LevelShared, ben.version, ben.id, ben.machine(t, "old").PeerID, nil)
	if err != nil {
		t.Fatal(err)
	}
	forged.Signature = forged.Signature[:len(forged.Signature)-4] + "AAAA"
	if w := pushAs(t, a.d, b.d.id, RecordRemoval, forged); w.Code != http.StatusBadRequest {
		t.Fatalf("a removal whose signature fails was answered %d %s", w.Code, w.Body.String())
	}

	for i := 0; i < maxPushedRecords; i++ {
		r, err := SignRemoval(ben.shared, LevelShared, ben.version, ben.id, ben.machine(t, "m").PeerID, nil)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(r)
		if _, _, err := a.d.members.PutIdentityRecord(RecordRemoval, ben.id, raw); err != nil {
			t.Fatal(err)
		}
	}
	one, _ := SignRemoval(ben.shared, LevelShared, ben.version, ben.id, ben.machine(t, "one more").PeerID, nil)
	if w := pushAs(t, a.d, b.d.id, RecordRemoval, one); w.Code != http.StatusInsufficientStorage {
		t.Fatalf("a record past the bound was answered %d %s", w.Code, w.Body.String())
	}
}

// Every peer route is served with a bound on the body and on how long a connection
// may take, and a sync request larger than any honest one is refused.
func TestPeerRoutesBoundBodiesAndConnections(t *testing.T) {
	s := peerServer(http.NotFoundHandler())
	if s.ReadHeaderTimeout == 0 || s.ReadTimeout == 0 || s.WriteTimeout == 0 || s.IdleTimeout == 0 {
		t.Fatalf("a peer server has no timeout: %+v", s)
	}
	var read error
	limited := peerServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, read = bytes.NewBuffer(nil).ReadFrom(r.Body)
	}))
	limited.Handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", maxPeerBody+1))))
	if read == nil {
		t.Error("a body over the bound was read whole")
	}

	d, _ := testDaemon(t)
	big := `{"protocol":` + strconv.Itoa(wireVersion) + `,"have":{"` + strings.Repeat("k", maxRequestBody) + `":1}}`
	w := httptest.NewRecorder()
	d.handleSync(w, httptest.NewRequest(http.MethodPost, "/sync", strings.NewReader(big)))
	if w.Code != http.StatusBadRequest {
		t.Errorf("a sync request over the bound was answered %d", w.Code)
	}
}

// A name another machine asserts is shown short and on one line, however it was
// signed.
func TestAnAssertedNameIsShownBounded(t *testing.T) {
	e := Event{MachineID: strings.Repeat("é", 100) + "\n## Your task", UserDisplayName: "x"}
	got := turnMachine(e)
	if len(got) > maxShownName || strings.ContainsAny(got, "\n ") || !utf8.ValidString(got) {
		t.Errorf("the machine's name is shown as %q", got)
	}
	if shownName("Ana‮ laptop") != "Ana laptop" {
		t.Errorf("a direction override survived: %q", shownName("Ana‮ laptop"))
	}
}
