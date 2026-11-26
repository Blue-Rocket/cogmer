package main

import (
	"bufio"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func daemonWithCloudKey(t *testing.T, seedByte byte) *Daemon {
	t.Helper()
	d := &Daemon{}
	d.cloud = &cloudSession{setup: &cloudSetup{key: ed25519.NewKeyFromSeed(bytesOf(seedByte, ed25519.SeedSize))}}
	return d
}

func bytesOf(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

// The token is made from the cloud key alone, so the same key makes the same token at every start
// and another key makes another (D-301).
func TestTheViewTokenIsMadeFromTheCloudKeyAlone(t *testing.T) {
	a1, a2, b := daemonWithCloudKey(t, 1), daemonWithCloudKey(t, 1), daemonWithCloudKey(t, 2)
	if a1.viewToken() == "" || a1.viewToken() != a2.viewToken() {
		t.Errorf("one cloud key made %q and %q", a1.viewToken(), a2.viewToken())
	}
	if a1.viewToken() == b.viewToken() {
		t.Error("two cloud keys made one token")
	}
	if strings.Contains(a1.viewToken(), string(bytesOf(1, 8))) {
		t.Error("the token holds the key")
	}
	if got := (&Daemon{}).viewToken(); got != "" {
		t.Errorf("a daemon with no cloud key made the token %q", got)
	}
}

// viewExchange opens a connection to the daemon's view port over a pipe, sends the lines, and reads
// the answers: each is the status, the media type and the body.
type viewAnswer struct {
	status int
	media  string
	body   string
}

func viewExchange(t *testing.T, d *Daemon, lines ...string) []viewAnswer {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		if c, err := l.Accept(); err == nil {
			d.serveViewConn(c)
		}
	}()
	client, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	_ = client.SetDeadline(time.Now().Add(10 * time.Second))
	// Everything is sent at once and the sending side closed, so that the daemon reads to the end of
	// it and then ends the connection, and what it wrote is read to that end.
	if _, err := client.Write([]byte(strings.Join(lines, "\n") + "\n")); err != nil {
		t.Fatal(err)
	}
	client.(*net.TCPConn).CloseWrite()
	r := bufio.NewReader(client)
	var out []viewAnswer
	for {
		head, err := r.ReadString('\n')
		if err != nil {
			break
		}
		f := strings.Fields(head)
		if len(f) != 3 {
			t.Fatalf("an answer's first line is %q", head)
		}
		status, _ := strconv.Atoi(f[0])
		n, _ := strconv.Atoi(f[2])
		body := make([]byte, n)
		if _, err := io.ReadFull(r, body); err != nil {
			t.Fatalf("a body of %d bytes did not arrive: %v", n, err)
		}
		out = append(out, viewAnswer{status, f[1], string(body)})
	}
	<-done
	return out
}

// A connection that does not open with the token is refused and closed, and one that does is served
// only the routes a page reads through, as reads.
func TestOnlyAConnectionThatOpensWithTheTokenIsServedAndOnlyTheViewsRoutes(t *testing.T) {
	d, _, _ := promptRoom(t, false)
	d.cloud = &cloudSession{setup: &cloudSetup{key: ed25519.NewKeyFromSeed(bytesOf(1, ed25519.SeedSize))}}
	good := d.viewToken()

	for name, token := range map[string]string{
		"no token":           "",
		"another token":      daemonWithCloudKey(t, 2).viewToken(),
		"a prefix":           good[:10],
		"the token and more": good + "x",
	} {
		got := viewExchange(t, d, token, "/healthz")
		if len(got) != 1 || got[0].status != http.StatusUnauthorized {
			t.Errorf("%s: answered %+v, want one refusal", name, got)
		}
	}

	for name, c := range map[string]struct {
		path string
		want int
	}{
		"the health check":   {"/healthz", http.StatusOK},
		"the rooms":          {"/rooms", http.StatusOK},
		"a hook route":       {"/hook/prompt", http.StatusNotFound},
		"the peer routes":    {"/sync", http.StatusNotFound},
		"the page itself":    {"/", http.StatusNotFound},
		"a room that is not": {"/snapshot?room=none-such", http.StatusNotFound},
		"not a path":         {"rooms", http.StatusBadRequest},
	} {
		got := viewExchange(t, d, good, c.path)
		if len(got) != 1 || got[0].status != c.want {
			t.Errorf("%s: %s answered %+v, want %d", name, c.path, got, c.want)
		}
	}

	// Several requests are answered on one connection, in order.
	got := viewExchange(t, d, good, "/healthz", "/rooms", "/nowhere")
	if len(got) != 3 || got[0].status != 200 || got[1].status != 200 || got[2].status != 404 {
		t.Errorf("three requests on one connection: %+v", got)
	}
	if got[0].media != "application/json" {
		t.Errorf("the media type is %q", got[0].media)
	}
}

func TestADaemonWithNoCloudKeyServesNothingToAPage(t *testing.T) {
	d, _, _ := promptRoom(t, false)
	for _, token := range []string{"", "x"} {
		got := viewExchange(t, d, token, "/healthz")
		if len(got) != 1 || got[0].status != http.StatusUnauthorized {
			t.Errorf("with %q a daemon with no key answered %+v", token, got)
		}
	}
}

// What a handler writes that is larger than a page can take is refused, not cut short, and a line
// that is too long ends the connection.
func TestAnAnswerTooLargeForThePageAndALineTooLongAreRefused(t *testing.T) {
	d, _, _ := promptRoom(t, false)
	d.cloud = &cloudSession{setup: &cloudSetup{key: ed25519.NewKeyFromSeed(bytesOf(1, ed25519.SeedSize))}}
	reply := &viewReply{header: http.Header{}, status: 200}
	if _, err := reply.Write(make([]byte, viewMaxBody)); err != nil {
		t.Fatalf("a body at the limit was refused: %v", err)
	}
	if _, err := reply.Write([]byte("x")); err == nil || !reply.over {
		t.Error("a body past the limit was taken")
	}
	got := viewExchange(t, d, d.viewToken(), "/"+strings.Repeat("a", viewMaxLine*2))
	if len(got) != 0 {
		t.Errorf("a line longer than the limit was answered: %+v", got)
	}
}

// The room comes back as the view draws it, with the token, from the same snapshot the stream sends.
func TestAPageThatCarriesTheTokenReadsTheRoomAsTheViewDrawsIt(t *testing.T) {
	d, room, post := promptRoom(t, false)
	d.cloud = &cloudSession{setup: &cloudSetup{key: ed25519.NewKeyFromSeed(bytesOf(7, ed25519.SeedSize))}}
	post(promptReq{Host: "claude", SessionID: "s", PromptID: "p1", Prompt: "a turn for the page"})

	got := viewExchange(t, d, d.viewToken(), "/snapshot?room="+room.RoomName)
	if len(got) != 1 || got[0].status != http.StatusOK {
		t.Fatalf("/snapshot answered %+v", got)
	}
	if !strings.Contains(got[0].body, "a turn for the page") {
		t.Errorf("the page was given %s", got[0].body)
	}
}

// A cloud session's commands get the link from the daemon, which holds the token, and the link
// carries the address, the token and the room in its fragment (D-301).
func TestTheDaemonMakesTheLinkAPageOpensTheRoomFrom(t *testing.T) {
	d, room, _ := promptRoom(t, false)
	d.cloud = &cloudSession{setup: &cloudSetup{key: ed25519.NewKeyFromSeed(bytesOf(3, ed25519.SeedSize))}}
	post := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/view/link", strings.NewReader(`{"room":"`+room.RoomName+`"}`))
		r.Header.Set(localGuardHeader, localGuardValue)
		w := httptest.NewRecorder()
		d.LocalRoutes().ServeHTTP(w, r)
		return w
	}
	if w := post(); w.Code != http.StatusServiceUnavailable {
		t.Errorf("a daemon not reachable through the relay made a link: %d", w.Code)
	}
	d.viewAddr = "tco2ADDRESS"
	w := post()
	if w.Code != http.StatusOK {
		t.Fatalf("/view/link answered %d: %s", w.Code, w.Body.String())
	}
	var out struct{ URL string }
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	u, err := url.Parse(out.URL)
	if err != nil || !strings.HasPrefix(out.URL, viewPageURL+"#") {
		t.Fatalf("the link is %q (%v)", out.URL, err)
	}
	frag, _ := url.ParseQuery(u.Fragment)
	if frag.Get("a") != "tco2ADDRESS" || frag.Get("k") != d.viewToken() || frag.Get("r") != room.RoomName {
		t.Errorf("the link's fragment holds %v", frag)
	}
	if u.RawQuery != "" {
		t.Errorf("the link sends something to the host of the page: %q", u.RawQuery)
	}

	// A web page cannot ask for it.
	r := httptest.NewRequest(http.MethodPost, "/view/link", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	d.LocalRoutes().ServeHTTP(rec, r)
	if rec.Code != http.StatusForbidden {
		t.Errorf("a request with no local header was answered %d", rec.Code)
	}
}

// A cloud host that does not set CLAUDE_CODE_REMOTE still gets the page, because the daemon that read
// a cloud key drops the key from its environment and then serves the port as a cloud session's.
func TestADaemonThatReadACloudKeyServesThePageWithoutTheHostVariable(t *testing.T) {
	t.Setenv("CLAUDE_CODE_REMOTE", "")
	t.Setenv(cloudKeyVar, "")
	d := &Daemon{}
	if d.servesView() {
		t.Fatal("a daemon with no cloud key and no cloud host variable serves the page")
	}
	d.cloud = &cloudSession{setup: &cloudSetup{key: ed25519.NewKeyFromSeed(bytesOf(1, ed25519.SeedSize))}}
	if !d.servesView() {
		t.Fatal("a daemon that speaks as a cloud session serves no page")
	}
	d.cloud = nil
	t.Setenv("CLAUDE_CODE_REMOTE", "true")
	if !d.servesView() {
		t.Fatal("a daemon in a cloud container serves no page")
	}
}

// A page that holds a room's state says so, and the daemon answers in a few bytes that it is unchanged,
// with only the age of the lists, which moves while a room sits quiet (D-307).
func TestARoomPolledWithTheHashItHoldsIsAnsweredAsUnchangedInAFewBytes(t *testing.T) {
	d, room, post := promptRoom(t, false)
	d.cloud = &cloudSession{setup: &cloudSetup{key: ed25519.NewKeyFromSeed(bytesOf(1, ed25519.SeedSize))}}
	good := d.viewToken()
	post(promptReq{SessionID: "s", PromptID: "p1", Prompt: strings.Repeat("a long turn with many words. ", 50)})
	path := "/snapshot?room=" + room.RoomName

	first := viewExchange(t, d, good, path)
	if len(first) != 1 || first[0].status != 200 {
		t.Fatalf("the first poll: %+v", first)
	}
	var state struct {
		Etag   string `json:"etag"`
		Events []any  `json:"events"`
	}
	if err := json.Unmarshal([]byte(first[0].body), &state); err != nil || state.Etag == "" || len(state.Events) == 0 {
		t.Fatalf("the first poll names no hash or no turn: %v %q", err, first[0].body)
	}

	again := viewExchange(t, d, good, path+"&since="+state.Etag)
	if len(again) != 1 || again[0].status != 200 {
		t.Fatalf("the poll with the hash: %+v", again)
	}
	if !strings.Contains(again[0].body, `"unchanged":true`) || len(again[0].body) > 200 {
		t.Errorf("a room that has not changed was answered with %d bytes: %.120s", len(again[0].body), again[0].body)
	}

	post(promptReq{SessionID: "s", PromptID: "p2", Prompt: "a new turn"})
	changed := viewExchange(t, d, good, path+"&since="+state.Etag)
	if len(changed) != 1 || strings.Contains(changed[0].body, `"unchanged"`) || !strings.Contains(changed[0].body, "a new turn") {
		t.Errorf("a room that changed was answered as unchanged or without the new turn: %.160s", changed[0].body)
	}

	other := viewExchange(t, d, good, path+"&since=0000000000000000")
	if len(other) != 1 || strings.Contains(other[0].body, `"unchanged"`) {
		t.Errorf("a hash the daemon does not hold was answered as unchanged")
	}
}

// The age of the lists is the one part of a quiet room's state that moves, so it is left out of the
// hash and sent beside the answer.
func TestTheAgeOfTheListsDoesNotChangeTheHashOfARoom(t *testing.T) {
	d, room, post := promptRoom(t, false)
	d.cloud = &cloudSession{setup: &cloudSetup{key: ed25519.NewKeyFromSeed(bytesOf(1, ed25519.SeedSize))}}
	good := d.viewToken()
	post(promptReq{SessionID: "s", PromptID: "p1", Prompt: "hello"})
	path := "/snapshot?room=" + room.RoomName
	etagOf := func() string {
		got := viewExchange(t, d, good, path)
		var s struct{ Etag string }
		if err := json.Unmarshal([]byte(got[0].body), &s); err != nil {
			t.Fatal(err)
		}
		return s.Etag
	}
	set := func(age time.Duration) {
		d.cloud.mu.Lock()
		d.cloud.header = &ListsHeader{Time: time.Now().Add(-age).UTC().Format(time.RFC3339)}
		d.cloud.from = "Tonka Truck"
		d.cloud.mu.Unlock()
	}
	set(time.Minute)
	before := etagOf()
	set(time.Hour)
	if after := etagOf(); after != before {
		t.Errorf("the hash moved from %s to %s when only the lists' age did", before, after)
	}
}

// The static page is sent the newest turns of a room, and asks for the turns before the oldest it holds
// in pages of the same size, until none are left (D-308).
func pagedRoom(t *testing.T, turns, perPage, minTurns, bytes int) (*Daemon, Room, string) {
	t.Helper()
	oldTurns, oldBytes, oldMin := pageTurns, pageBytes, pageMinTurns
	pageTurns, pageBytes, pageMinTurns = perPage, bytes, minTurns
	t.Cleanup(func() { pageTurns, pageBytes, pageMinTurns = oldTurns, oldBytes, oldMin })
	d, room, post := promptRoom(t, false)
	d.cloud = &cloudSession{setup: &cloudSetup{key: ed25519.NewKeyFromSeed(bytesOf(1, ed25519.SeedSize))}}
	for i := 1; i <= turns; i++ {
		post(promptReq{SessionID: "s", PromptID: fmt.Sprintf("p%d", i), Prompt: fmt.Sprintf("turn %02d", i)})
	}
	return d, room, d.viewToken()
}

type pageAnswer struct {
	Events []struct {
		Content string `json:"content"`
		Order   string `json:"o"`
	} `json:"events"`
	Total      int  `json:"total"`
	HasEarlier bool `json:"hasEarlier"`
	Earlier    bool `json:"earlier"`
	Paged      bool `json:"paged"`
}

func askPage(t *testing.T, d *Daemon, token, path string) pageAnswer {
	t.Helper()
	got := viewExchange(t, d, token, path)
	var a pageAnswer
	if len(got) != 1 || got[0].status != 200 || json.Unmarshal([]byte(got[0].body), &a) != nil {
		t.Fatalf("%s answered %+v", path, got)
	}
	return a
}

func TestTheNewestTurnsOfARoomAreSentAndTheEarlierOnesAreAskedForByCursor(t *testing.T) {
	d, room, token := pagedRoom(t, 12, 5, 2, 1<<30)
	path := "/snapshot?room=" + room.RoomName

	first := askPage(t, d, token, path)
	if !first.Paged || first.Earlier || len(first.Events) != 5 || first.Total != 12 || !first.HasEarlier {
		t.Fatalf("the newest page: %+v", first)
	}
	if first.Events[0].Content != "turn 08" || first.Events[4].Content != "turn 12" || first.Events[0].Order == "" {
		t.Fatalf("the newest page holds %q to %q, order %q", first.Events[0].Content, first.Events[4].Content, first.Events[0].Order)
	}

	var all []string
	for _, e := range first.Events {
		all = append(all, e.Content)
	}
	cursor, more := first.Events[0].Order, first.HasEarlier
	for pages := 0; more; pages++ {
		if pages > 5 {
			t.Fatal("the earlier pages did not end")
		}
		p := askPage(t, d, token, path+"&before="+url.QueryEscape(cursor))
		if !p.Earlier || len(p.Events) == 0 || len(p.Events) > 5 {
			t.Fatalf("an earlier page: %+v", p)
		}
		var texts []string
		for _, e := range p.Events {
			texts = append(texts, e.Content)
		}
		all = append(texts, all...)
		cursor, more = p.Events[0].Order, p.HasEarlier
	}
	if len(all) != 12 {
		t.Fatalf("the pages hold %d turns, want 12: %v", len(all), all)
	}
	for i, c := range all {
		if want := fmt.Sprintf("turn %02d", i+1); c != want {
			t.Fatalf("turn %d is %q, want %q; the pages are out of order or repeat a turn: %v", i+1, c, want, all)
		}
	}
}

func TestAPageEndsAtTheByteCapOnceItHasItsMinimumOfTurns(t *testing.T) {
	// Each short turn counts for twice its text and 300 more, 320, so 2000 is reached at 7 turns.
	d, room, token := pagedRoom(t, 30, 100, 3, 2000)
	if got := askPage(t, d, token, "/snapshot?room="+room.RoomName); len(got.Events) != 7 || !got.HasEarlier {
		t.Errorf("small turns under a cap of 2000 gave %d, want 7", len(got.Events))
	}
}

// The minimum holds for turns that are each over the cap.
func TestAPageHoldsItsMinimumOfTurnsThatAreEachOverTheByteCap(t *testing.T) {
	d, room, post := promptRoom(t, false)
	pageTurns, pageBytes, pageMinTurns = 100, 2000, 3
	t.Cleanup(func() { pageTurns, pageBytes, pageMinTurns = 20, 300<<10, 10 })
	d.cloud = &cloudSession{setup: &cloudSetup{key: ed25519.NewKeyFromSeed(bytesOf(1, ed25519.SeedSize))}}
	for i := 0; i < 10; i++ {
		post(promptReq{SessionID: "s", PromptID: fmt.Sprintf("p%d", i), Prompt: strings.Repeat("x", 5000)})
	}
	if got := askPage(t, d, d.viewToken(), "/snapshot?room="+room.RoomName); len(got.Events) != 3 {
		t.Errorf("turns over the cap gave %d, want the minimum of 3", len(got.Events))
	}
}

func TestAPageRoomWithFewTurnsHasNoEarlierPage(t *testing.T) {
	d, room, token := pagedRoom(t, 3, 20, 10, 300<<10)
	got := askPage(t, d, token, "/snapshot?room="+room.RoomName)
	if len(got.Events) != 3 || got.HasEarlier || got.Total != 3 {
		t.Errorf("a room of 3 turns: %+v", got)
	}
	none := askPage(t, d, token, "/snapshot?room="+room.RoomName+"&before="+url.QueryEscape(got.Events[0].Order))
	if len(none.Events) != 0 || none.HasEarlier {
		t.Errorf("a request before the first turn answered %+v", none)
	}
}

// A page that opens a room wakes the addresses the room is asked of (D-305), and a page that asks again
// with the hash it holds, or for earlier turns, does not, since it is already reading and its polls
// would keep a quiet room polled every second.
func TestThePagesRouteWakesTheRoomOnlyWhenAPageOpensIt(t *testing.T) {
	d, room, post := promptRoom(t, false)
	d.cloud = &cloudSession{setup: &cloudSetup{key: ed25519.NewKeyFromSeed(bytesOf(1, ed25519.SeedSize))}}
	good := d.viewToken()
	post(promptReq{SessionID: "s", PromptID: "p1", Prompt: "hello"})
	path := "/snapshot?room=" + room.RoomName
	woken := func() bool {
		d.pace.mu.Lock()
		defer d.pace.mu.Unlock()
		_, ok := d.pace.woken[room.RoomID]
		delete(d.pace.woken, room.RoomID)
		return ok
	}
	woken()
	viewExchange(t, d, good, path)
	if !woken() {
		t.Error("a page opening the room did not wake its addresses")
	}
	viewExchange(t, d, good, path+"&since=0123456789abcdef")
	if woken() {
		t.Error("a page asking again with the hash it holds woke the room's addresses")
	}
	viewExchange(t, d, good, path+"&before=zzz")
	if woken() {
		t.Error("a page asking for earlier turns woke the room's addresses")
	}
}
