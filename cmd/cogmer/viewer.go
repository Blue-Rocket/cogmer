package main

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// A cloud session's daemon serves the room to a page in the user's own browser over a second port
// of the overlay (D-301). The overlay encrypts what it carries, so the port needs no TLS, which a
// browser could not run over the connection its tailcat module returns. The overlay admits any
// dialer (D-157), so each connection opens with a token that only the user's cloud key can make.
const (
	// viewPort is the overlay port of the page's routes, beside tailcatPort.
	viewPort = 4784
	// viewTokenLabel separates this token from every other value made from the cloud key.
	viewTokenLabel = "cogmer-browser-view-v1"
)

// viewToken is the token a page sends, made from the cloud key of the session, or "" while this
// daemon holds no cloud key. The same cloud key makes the same token in every task and after every
// wake, so a link a user holds stays good, and nothing is stored.
func (d *Daemon) viewToken() string {
	c := d.cloud
	if c == nil {
		return ""
	}
	mac := hmac.New(sha256.New, c.setup.key.Seed())
	mac.Write([]byte(viewTokenLabel))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// viewMux is the routes a page reads the room through: the room as the view draws it, the rooms, an
// image of a turn and the export, and nothing that changes anything.
func (d *Daemon) viewMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", d.health)
	mux.HandleFunc("/snapshot", d.handleSnapshot)
	mux.HandleFunc("/rooms", d.handleRooms)
	mux.HandleFunc("/image/", d.handleImage)
	mux.HandleFunc("/export", d.handleExport)
	return mux
}

// The port speaks lines, not HTTP, so that the page needs no HTTP client, and what it parses is a
// length (D-301). The connection opens with the token on a line. Each request after it is a path on
// a line, always a read. Each answer is a line of the status, the media type and the length of the
// body, then the body.
const (
	viewMaxLine  = 4096
	viewIdle     = 2 * time.Minute
	viewMaxBody  = 32 << 20
	viewMaxConns = 16
)

// ServeView serves the page's connections on l until it closes, at most viewMaxConns at a time.
func (d *Daemon) ServeView(l net.Listener) {
	slots := make(chan struct{}, viewMaxConns)
	for {
		c, err := l.Accept()
		if err != nil {
			return
		}
		select {
		case slots <- struct{}{}:
			go func() {
				defer func() { <-slots }()
				d.serveViewConn(c)
			}()
		default:
			c.Close()
		}
	}
}

// viewReply is what a handler writes, held until it is sent with its length.
type viewReply struct {
	header http.Header
	status int
	body   bytes.Buffer
	over   bool
}

func (r *viewReply) Header() http.Header { return r.header }
func (r *viewReply) WriteHeader(s int)   { r.status = s }
func (r *viewReply) Write(b []byte) (int, error) {
	if r.body.Len()+len(b) > viewMaxBody {
		r.over = true
		return 0, errors.New("the answer is too large for the page")
	}
	return r.body.Write(b)
}

func readViewLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadSlice('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(line), "\r\n"), nil
}

func writeViewReply(c net.Conn, status int, mediaType string, body []byte) error {
	_ = c.SetWriteDeadline(time.Now().Add(viewIdle))
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	if _, err := fmt.Fprintf(c, "%d %s %d\n", status, mediaType, len(body)); err != nil {
		return err
	}
	_, err := c.Write(body)
	return err
}

// serveViewConn serves one connection: it refuses one that does not open with the token, and answers
// each path after that from the routes above.
func (d *Daemon) serveViewConn(c net.Conn) {
	defer c.Close()
	r := bufio.NewReaderSize(c, viewMaxLine)
	_ = c.SetReadDeadline(time.Now().Add(viewIdle))
	given, err := readViewLine(r)
	want := d.viewToken()
	if err != nil || want == "" || !hmac.Equal([]byte(given), []byte(want)) {
		_ = writeViewReply(c, http.StatusUnauthorized, "text/plain", []byte("this page needs the link's token\n"))
		return
	}
	mux := d.viewMux()
	for {
		_ = c.SetReadDeadline(time.Now().Add(viewIdle))
		path, err := readViewLine(r)
		if err != nil {
			return
		}
		if !strings.HasPrefix(path, "/") {
			if writeViewReply(c, http.StatusBadRequest, "text/plain", []byte("a request is a path\n")) != nil {
				return
			}
			continue
		}
		req, err := http.NewRequest(http.MethodGet, path, nil)
		if err != nil {
			if writeViewReply(c, http.StatusBadRequest, "text/plain", []byte("a request is a path\n")) != nil {
				return
			}
			continue
		}
		reply := &viewReply{header: http.Header{}, status: http.StatusOK}
		mux.ServeHTTP(reply, req)
		if reply.over {
			reply = &viewReply{header: http.Header{"Content-Type": {"text/plain"}}, status: http.StatusRequestEntityTooLarge}
			reply.body.WriteString("the answer is too large for the page\n")
		}
		mediaType := strings.TrimSpace(strings.SplitN(reply.header.Get("Content-Type"), ";", 2)[0])
		if writeViewReply(c, reply.status, mediaType, reply.body.Bytes()) != nil {
			return
		}
	}
}

// viewPageURL is where the page that reads the room is served, a static page that holds no
// secret, and the link to it carries what the page needs in its fragment, which a browser does not
// send to the host of the page.
const viewPageURL = "https://blue-rocket.github.io/cogmer/view/"

// viewLinkURL is the link that opens the room in a browser: the address of the daemon, the
// token, and the room when there is one.
func viewLinkURL(addr, token, room string) string {
	v := url.Values{}
	v.Set("a", addr)
	v.Set("k", token)
	if room != "" {
		v.Set("r", room)
	}
	return viewPageURL + "#" + v.Encode()
}

// handleViewLink answers the commands of a cloud session with the link to its room. The link holds
// the token, so the route is the local daemon's alone, as the others that read a secret are.
func (d *Daemon) handleViewLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Room string `json:"room"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	token := d.viewToken()
	if token == "" || d.viewAddr == "" {
		http.Error(w, "this daemon serves no page: it holds no cloud key, or it is not reachable through the relay", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, map[string]string{"url": viewLinkURL(d.viewAddr, token, req.Room)})
}

// cloudWatchLine is the line that says where to read the room from a cloud session, whose own
// page no browser outside its container reaches. It asks the daemon for the link, and says why
// when the daemon cannot make one.
func cloudWatchLine(room string) string {
	body, _ := json.Marshal(map[string]string{"room": room})
	resp, err := localPost(&http.Client{Timeout: 3 * time.Second}, "http://"+addr()+"/view/link", body)
	if err != nil {
		return "the room's page is in this cloud workspace and no browser of yours reaches it, and the link for yours could not be made because the daemon did not answer"
	}
	defer resp.Body.Close()
	var out struct {
		URL string `json:"url"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&out) != nil || out.URL == "" {
		return "the room's page is in this cloud workspace and no browser of yours reaches it, and the link for yours could not be made yet; try again in a moment"
	}
	return fmt.Sprintf("Read this room in your browser: %s\nAnyone who has this link can read the room while this session's daemon is awake.", out.URL)
}

// handleSnapshot is one message of the view's stream, for a page that asks for it again in place
// of holding a stream open.
func (d *Daemon) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	room, ok := d.roomFromPath("/room/" + r.URL.Query().Get("room"))
	if !ok {
		http.Error(w, "no such room", http.StatusNotFound)
		return
	}
	st, err := d.viewSnapshot(room)
	if err != nil {
		http.Error(w, "could not read the room", http.StatusInternalServerError)
		return
	}
	lists := st.Lists
	st.Lists = nil
	raw, err := json.Marshal(st)
	if err != nil {
		http.Error(w, "could not read the room", http.StatusInternalServerError)
		return
	}
	sum := sha256.Sum256(raw)
	st.Etag = hex.EncodeToString(sum[:8])
	if since := r.URL.Query().Get("since"); since != "" && since == st.Etag {
		// The page holds this state, and only the age of the lists moves while a room is quiet.
		writeJSON(w, map[string]any{"unchanged": true, "lists": lists})
		return
	}
	st.Lists = lists
	writeJSON(w, st)
}

// servesView says whether this daemon serves the page's port, which a cloud session's daemon does.
func (d *Daemon) servesView() bool { return d.inCloud() }

// inCloud says whether this daemon is a cloud session's: it runs in a cloud container, or it speaks
// as a cloud session, which a cloud host that does not set CLAUDE_CODE_REMOTE still does, since the
// daemon drops the cloud key from its own environment once it has read it.
func (d *Daemon) inCloud() bool {
	return inCloudContainer() || d.cloud != nil
}

// cloudUploadsDir is the folder a cloud container saves a user's attachments in (D-304), which
// COGMER_UPLOADS_DIR names where a test needs another.
func cloudUploadsDir() string {
	if dir := os.Getenv("COGMER_UPLOADS_DIR"); dir != "" {
		return dir
	}
	return "/mnt/user-data/uploads"
}
