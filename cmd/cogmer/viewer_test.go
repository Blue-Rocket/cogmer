package main

import (
	"bufio"
	"crypto/ed25519"
	"encoding/json"
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
