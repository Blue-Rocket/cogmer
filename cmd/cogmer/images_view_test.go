package main

import (
	"bytes"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// The view shows a pasted image from the daemon's own address (D-212, D-213), and says so when
// the image is not there.

// anImage puts a pasted image in the room through the prompt hook, as a session would, and
// returns the hash and bytes.
func anImage(t *testing.T, post func(promptReq), shade uint8) (string, []byte) {
	t.Helper()
	scratchpad, images := sessionFiles(t)
	img := pngBytes(t, shade)
	writeFile(t, filepath.Join(images, "1.png"), img)
	post(promptReq{SessionID: "s", PromptID: "p" + string(rune('a'+shade)), Prompt: "[Image #1] here", ScratchpadDir: scratchpad})
	return hashOf(img), img
}

func TestTheViewShowsAHeldImageAndSaysWhenOneIsComing(t *testing.T) {
	d, room, post := promptRoom(t, false)
	sha, _ := anImage(t, post, 31)
	store, _ := d.storeFor(room.RoomID)
	coming := hashOf([]byte("not yet here"))
	if err := store.WantImage(coming, "image/png", 100); err != nil {
		t.Fatal(err)
	}
	for _, refs := range [][]imageRef{
		{{N: 1, SHA256: coming, Type: "image/png", Size: 100}},
		{{N: 1, Note: imageTooLarge, Size: 8 << 20, Type: "image/png"}},
		{{N: 1, Note: imageMissing}},
		{{N: 1, Note: imageUnsupported}},
	} {
		if _, err := d.appendLocal(store, room, "s", EventUserPrompt, "x", map[string]any{"images": refs}); err != nil {
			t.Fatal(err)
		}
	}
	st, err := d.viewSnapshot(room)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`<img class="pasted" src="/image/` + room.RoomName + `/` + sha + `" alt="pasted image 1">`,
		"image 1 is on its way",
		"image 1 is 8.0 MB, which is more than cogmer sends",
		"image 1 could not be read on its sender's machine",
		"image 1 is not a kind of image cogmer sends",
	}
	if len(st.Events) != len(want) {
		t.Fatalf("%d events", len(st.Events))
	}
	for i, w := range want {
		if !strings.Contains(st.Events[i].HTML, w) {
			t.Errorf("turn %d lacks %q: %s", i, w, st.Events[i].HTML)
		}
	}
	if strings.Contains(st.Events[1].HTML, "<img") {
		t.Error("an image that is not here is shown as an image")
	}
	// The export keeps the text and says an image was there.
	exp := exportMarkdown(func() uiState { s, _ := d.snapshot(room); return s }(), time.Date(2026, 10, 15, 12, 0, 0, 0, time.Local))
	if !strings.Contains(exp, "*[image 1, image/png,") || !strings.Contains(exp, "*[image 1 was not sent: too-large]*") {
		t.Errorf("the export does not say what images a turn had:\n%s", exp)
	}
}

var imgElement = regexp.MustCompile(`<img [^>]*>`)

// What a peer wrote in an event's metadata is never echoed into the page and never becomes an
// address: only a hash that is a hash, in an address the daemon builds, is shown.
func TestNothingAPeerWritesInAnImageNameBecomesMarkup(t *testing.T) {
	held := map[string]imageRow{}
	good := hashOf([]byte("good"))
	held[good] = imageRow{SHA256: good, State: imageHave}
	html := imagesHTML(`a"><script>alert(1)</script>`, []imageRef{
		{N: 1, SHA256: good, Type: "image/png"},
		{N: 2, SHA256: `"><script>alert(2)</script>`, Type: "image/png"},
		{N: 3, SHA256: "../../identity.key"},
		{N: 4, Note: `<script>alert(3)</script>`},
		{N: 5, SHA256: good, Type: `"><img src=//evil/x>`, Note: `"><script>`},
		{N: -1, SHA256: good},
	}, held)
	if strings.Contains(html, "<script") || strings.Contains(html, "evil") || strings.Contains(html, "identity.key") {
		t.Errorf("a peer's text reaches the page: %s", html)
	}
	for _, el := range imgElement.FindAllString(html, -1) {
		if !regexp.MustCompile(`^<img class="pasted" src="/image/[A-Za-z0-9%._~-]+/[0-9a-f]{64}" alt="pasted image -?\d+">$`).MatchString(el) {
			t.Errorf("an image element the daemon did not build: %s", el)
		}
	}
	if len(imgElement.FindAllString(html, -1)) != 2 {
		t.Errorf("images shown: %s", html)
	}
}

func getImage(d *Daemon, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	d.LocalRoutes().ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec
}

func TestTheViewIsServedAHeldImageAndNothingElse(t *testing.T) {
	d, room, post := promptRoom(t, false)
	sha, img := anImage(t, post, 32)
	rec := getImage(d, "/image/"+room.RoomName+"/"+sha)
	if rec.Code != 200 || !bytes.Equal(rec.Body.Bytes(), img) {
		t.Fatalf("a held image gives %d", rec.Code)
	}
	h := rec.Header()
	if h.Get("Content-Type") != "image/png" || h.Get("X-Content-Type-Options") != "nosniff" ||
		!strings.Contains(h.Get("Content-Security-Policy"), "sandbox") || h.Get("Referrer-Policy") != "no-referrer" {
		t.Errorf("headers: %v", h)
	}
	store, _ := d.storeFor(room.RoomID)
	if err := store.WantImage(hashOf([]byte("w")), "image/png", 1); err != nil {
		t.Fatal(err)
	}
	// A second room on the same daemon does not serve the first room's image.
	other, err := d.members.CreateRoom(d.id.PeerID, false)
	if err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{
		"another room":       "/image/" + other.RoomName + "/" + sha,
		"no such room":       "/image/nowhere/" + sha,
		"an image not held":  "/image/" + room.RoomName + "/" + hashOf([]byte("w")),
		"a hash not known":   "/image/" + room.RoomName + "/" + hashOf([]byte("unknown")),
		"a traversal":        "/image/" + room.RoomName + "/../../identity.key",
		"an encoded upward":  "/image/" + room.RoomName + "/%2e%2e%2fidentity.key",
		"no hash":            "/image/" + room.RoomName + "/",
		"an upper-case hash": "/image/" + room.RoomName + "/" + strings.ToUpper(sha),
		"a short hash":       "/image/" + room.RoomName + "/" + sha[:63],
	} {
		// The router may answer a path with dots by redirecting to the cleaned path, which is
		// outside /image/. What matters is that no image is served.
		if rec := getImage(d, path); rec.Code == 200 || bytes.Contains(rec.Body.Bytes(), []byte("PNG")) {
			t.Errorf("%s gives %d", name, rec.Code)
		}
	}
	// A row that names a type the page would not show is not served.
	if _, err := store.db.Exec(`UPDATE images SET media_type = 'text/html' WHERE sha256 = ?`, sha); err != nil {
		t.Fatal(err)
	}
	if rec := getImage(d, "/image/"+room.RoomName+"/"+sha); rec.Code != 404 {
		t.Errorf("a type that is not an image gives %d", rec.Code)
	}
}

// The page is built once, so a picture that arrives after its turn has to rewrite the body, and
// only the body.
func TestThePageUpdatesATurnsBodyWhenItsImageArrives(t *testing.T) {
	src := string(uiPage)
	for _, want := range []string{"m!==t.body", "t.md.innerHTML=m", ".imagenote", "img.pasted", "toolsHtml"} {
		if !strings.Contains(src, want) {
			t.Errorf("the page no longer has %q", want)
		}
	}
	if strings.Contains(src, "bodyHtml") {
		t.Error("the page rebuilds the body with its tool list")
	}
}
