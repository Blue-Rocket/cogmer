package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// Images move between peers on their own (D-213): the event names an image and its hash, and
// a peer that lacks the bytes asks for them by that hash.

const imageRoom, imageRoomName = "room-images", "quiet-harbor"

// shareImage has p write a turn that names an image it holds.
func (p *offlinePeer) shareImage(t *testing.T, data []byte, mutate func(*imageRef)) imageRef {
	t.Helper()
	ref := imageRef{N: 1, SHA256: hashOf(data), Type: "image/png", Size: int64(len(data))}
	if err := p.store.SaveImage(ref.SHA256, ref.Type, data); err != nil {
		t.Fatal(err)
	}
	if mutate != nil {
		mutate(&ref)
	}
	if _, err := p.d.appendLocal(p.store, Room{RoomID: imageRoom}, "s", EventUserPrompt, "[Image #1] look",
		map[string]any{"images": []imageRef{ref}}); err != nil {
		t.Fatal(err)
	}
	return ref
}

// askOnly makes p ask for images at the given peers' addresses.
func (p *offlinePeer) askOnly(peers ...*offlinePeer) {
	p.d.imagePeers = func(Room) []string {
		var out []string
		for _, q := range peers {
			out = append(out, q.addr)
		}
		return out
	}
}

func imagePair(t *testing.T) (a, b *offlinePeer) {
	t.Helper()
	a = newOfflinePeer(t, imageRoom, imageRoomName)
	b = newOfflinePeer(t, imageRoom, imageRoomName)
	introduce(t, a, b, imageRoom)
	return a, b
}

func TestAnImageReachesAColleagueAndIsKept(t *testing.T) {
	a, b := imagePair(t)
	img := pngBytes(t, 11)
	ref := a.shareImage(t, img, nil)

	if n, err := b.d.pullFrom(a.addr); err != nil || n != 1 {
		t.Fatalf("pull: %d, %v", n, err)
	}
	row, ok, _ := b.store.ImageRow(ref.SHA256)
	if !ok || row.State != imageWanted || row.Size != ref.Size || row.Type != "image/png" {
		t.Fatalf("the turn arrived and the room records %+v", row)
	}

	b.askOnly(a)
	if n := b.d.fetchWanted(); n != 1 {
		t.Fatalf("fetched %d images", n)
	}
	path, _ := b.store.imagePath(ref.SHA256)
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, img) {
		t.Fatalf("the colleague's file is not the image: %v", err)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Errorf("the file is readable by others: %v", info.Mode())
	}
	if row, _, _ := b.store.ImageRow(ref.SHA256); row.State != imageHave {
		t.Errorf("the room still wants an image it holds: %+v", row)
	}
	if _, err := os.Stat(path + ".part"); err == nil {
		t.Error("a partial file is left behind")
	}
	if n := b.d.fetchWanted(); n != 0 {
		t.Errorf("an image held is fetched again (%d)", n)
	}
}

// A room outlives any one machine, so the bytes come from whoever has them. The hash in the
// signed event is what makes a peer that did not write the turn safe to ask.
func TestAnImageCanComeFromAPeerThatDidNotWriteTheTurn(t *testing.T) {
	a := newOfflinePeer(t, imageRoom, imageRoomName)
	b := newOfflinePeer(t, imageRoom, imageRoomName)
	c := newOfflinePeer(t, imageRoom, imageRoomName)
	introduce(t, a, b, imageRoom)
	introduce(t, b, c, imageRoom)
	introduce(t, a, c, imageRoom)
	img := pngBytes(t, 12)
	ref := a.shareImage(t, img, nil)
	b.d.pullFrom(a.addr)
	b.askOnly(a)
	b.d.fetchWanted()

	a.up.Store(false) // the author is offline
	if n, err := c.d.pullFrom(b.addr); err != nil || n != 1 {
		t.Fatalf("c pulls the turn from b: %d, %v", n, err)
	}
	c.askOnly(b)
	if n := c.d.fetchWanted(); n != 1 {
		t.Fatalf("c fetched %d images from b", n)
	}
	path, _ := c.store.imagePath(ref.SHA256)
	if got, _ := os.ReadFile(path); !bytes.Equal(got, img) {
		t.Error("c's file is not the image")
	}
}

// What a peer sends is kept only if it is the image the author's signature names.
func TestAPeerThatSendsADifferentImageIsRefused(t *testing.T) {
	a, b := imagePair(t)
	img := pngBytes(t, 13)
	ref := a.shareImage(t, img, nil)
	b.d.pullFrom(a.addr)
	b.askOnly(a)

	// The peer's copy is replaced by other bytes of the same length, under the same name.
	apath, _ := a.store.imagePath(ref.SHA256)
	swapped := append([]byte{}, img...)
	swapped[len(swapped)-1] ^= 0xFF
	if err := os.WriteFile(apath, swapped, 0o600); err != nil {
		t.Fatal(err)
	}
	if n := b.d.fetchWanted(); n != 0 {
		t.Fatalf("a swapped image was accepted (%d)", n)
	}
	bpath, _ := b.store.imagePath(ref.SHA256)
	for _, p := range []string{bpath, bpath + ".part"} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("%s is left after a refused image", p)
		}
	}
	row, _, _ := b.store.ImageRow(ref.SHA256)
	if row.State != imageWanted || row.Attempts != 1 || row.NextTry == "" {
		t.Errorf("the room records %+v", row)
	}
}

func TestAnImageOfAnotherTypeThanTheEventNamesIsRefused(t *testing.T) {
	a, b := imagePair(t)
	a.shareImage(t, pngBytes(t, 14), func(r *imageRef) { r.Type = "image/gif" })
	b.d.pullFrom(a.addr)
	b.askOnly(a)
	if n := b.d.fetchWanted(); n != 0 {
		t.Errorf("an image whose bytes are not the named type was accepted (%d)", n)
	}
}

// An event comes from a peer, so what it asks this machine to fetch is checked first.
func TestAnEventCannotAskForAnImageThatIsNotWanted(t *testing.T) {
	a, b := imagePair(t)
	img := pngBytes(t, 15)
	for _, mutate := range []func(*imageRef){
		func(r *imageRef) { r.Size = maxImageBytes + 1 },
		func(r *imageRef) { r.Size = 0 },
		func(r *imageRef) { r.Type = "image/svg+xml" },
		func(r *imageRef) { r.Type = "text/html" },
		func(r *imageRef) { r.SHA256 = "../../etc/passwd" },
		func(r *imageRef) { r.SHA256 = strings.Repeat("A", 64) },
		func(r *imageRef) { r.Note = imageTooLarge },
	} {
		a.shareImage(t, img, mutate)
	}
	b.d.pullFrom(a.addr)
	rows, _ := b.store.ImageRows()
	if len(rows) != 0 {
		t.Errorf("the room wants %+v", rows)
	}
}

func TestAFetchContinuesWhereItStopped(t *testing.T) {
	a, b := imagePair(t)
	img := append(pngBytes(t, 16), make([]byte, 40000)...) // larger than one read
	ref := a.shareImage(t, img, nil)
	b.d.pullFrom(a.addr)
	b.askOnly(a)
	path, _ := b.store.imagePath(ref.SHA256)
	if err := os.MkdirAll(b.store.imageDir, 0o700); err != nil {
		t.Fatal(err)
	}
	half := len(img) / 2
	if err := os.WriteFile(path+".part", img[:half], 0o600); err != nil {
		t.Fatal(err)
	}
	if n := b.d.fetchWanted(); n != 1 {
		t.Fatalf("fetched %d", n)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, img) {
		t.Error("the continued file is not the image")
	}
	// A partial file that cannot belong to the image is started again.
	a2, b2 := imagePair(t)
	img2 := pngBytes(t, 17)
	ref2 := a2.shareImage(t, img2, nil)
	b2.d.pullFrom(a2.addr)
	b2.askOnly(a2)
	p2, _ := b2.store.imagePath(ref2.SHA256)
	_ = os.MkdirAll(b2.store.imageDir, 0o700)
	_ = os.WriteFile(p2+".part", append(img2, 1, 2, 3), 0o600)
	if n := b2.d.fetchWanted(); n != 1 {
		t.Fatalf("a partial file longer than the image blocks the fetch (%d)", n)
	}
}

func TestAFailedFetchIsTriedAgainLaterAndNotAtOnce(t *testing.T) {
	a, b := imagePair(t)
	img := pngBytes(t, 18)
	ref := a.shareImage(t, img, nil)
	b.d.pullFrom(a.addr)
	b.askOnly(a)

	a.up.Store(false)
	if n := b.d.fetchWanted(); n != 0 {
		t.Fatalf("fetched from an unreachable peer (%d)", n)
	}
	row, _, _ := b.store.ImageRow(ref.SHA256)
	if row.Attempts != 1 {
		t.Fatalf("attempts = %d", row.Attempts)
	}
	a.up.Store(true)
	if n := b.d.fetchWanted(); n != 0 {
		t.Errorf("an image is tried again before its wait is over (%d)", n)
	}
	if _, err := b.store.db.Exec(`UPDATE images SET next_try = ?`, time.Now().Add(-time.Second).UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	if n := b.d.fetchWanted(); n != 1 {
		t.Errorf("the image is not fetched once the wait is over (%d)", n)
	}
}

// blobAs posts a request for an image as the given identity would.
func blobAs(t *testing.T, d *Daemon, id *Identity, roomID, sha string, offset int64) *httptest.ResponseRecorder {
	t.Helper()
	ts, nonce, sig, err := signRequest(id, roomID, "")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(blobRequest{Protocol: wireVersion, RoomID: roomID, SHA256: sha, Offset: offset,
		PeerID: id.PeerID, Timestamp: ts, Nonce: nonce, Signature: sig})
	r := httptest.NewRequest(http.MethodPost, "/blob", bytes.NewReader(body))
	w := httptest.NewRecorder()
	d.PeerRoutes().ServeHTTP(w, r)
	return w
}

func TestOnlyAVerifiedGuestIsServedAnImage(t *testing.T) {
	a, b := imagePair(t)
	img := pngBytes(t, 19)
	ref := a.shareImage(t, img, nil)

	if w := blobAs(t, a.d, b.d.id, imageRoom, ref.SHA256, 0); w.Code != 200 || !bytes.Equal(w.Body.Bytes(), img) {
		t.Fatalf("a guest is served %d", w.Code)
	} else if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("the bytes could be sniffed as a page")
	}
	if w := blobAs(t, a.d, b.d.id, imageRoom, ref.SHA256, 10); w.Code != 200 || !bytes.Equal(w.Body.Bytes(), img[10:]) {
		t.Errorf("an offset is not honored (%d)", w.Code)
	}
	if w := blobAs(t, a.d, b.d.id, imageRoom, ref.SHA256, int64(len(img))+1); w.Code != 400 {
		t.Errorf("an offset past the end gives %d", w.Code)
	}
	if w := blobAs(t, a.d, b.d.id, imageRoom, ref.SHA256, -1); w.Code != 400 {
		t.Errorf("a negative offset gives %d", w.Code)
	}
	stranger := testIdentity(t)
	if w := blobAs(t, a.d, stranger, imageRoom, ref.SHA256, 0); w.Code != 401 || w.Body.Len() > 40 {
		t.Errorf("a stranger gets %d with %d bytes", w.Code, w.Body.Len())
	}
	// A guest nobody has verified is refused as it is for turns (D-054).
	unverified := testIdentity(t)
	if err := a.d.members.Allow(unverified.PeerID, ""); err != nil {
		t.Fatal(err)
	}
	if err := a.d.members.Invite(imageRoom, unverified.PeerID); err != nil {
		t.Fatal(err)
	}
	if w := blobAs(t, a.d, unverified, imageRoom, ref.SHA256, 0); w.Code != 401 {
		t.Errorf("an unverified guest gets %d", w.Code)
	}
	if w := blobAs(t, a.d, b.d.id, "another-room", ref.SHA256, 0); w.Code != 401 {
		t.Errorf("a room that does not exist gives %d", w.Code)
	}
}

func TestAnImageIsServedOnlyWhenItIsHeldAndOnlyByItsHash(t *testing.T) {
	a, b := imagePair(t)
	ref := a.shareImage(t, pngBytes(t, 20), nil)
	for _, sha := range []string{hashOf([]byte("never stored")), "../../identity.key", "", strings.Repeat("g", 64)} {
		w := blobAs(t, a.d, b.d.id, imageRoom, sha, 0)
		if w.Code != 404 && w.Code != 400 {
			t.Errorf("%q gives %d", sha, w.Code)
		}
		if strings.Contains(w.Body.String(), "PNG") {
			t.Errorf("%q served an image", sha)
		}
	}
	// An image still being fetched is not held, so it is not served on.
	if err := a.store.WantImage(hashOf([]byte("wanted")), "image/png", 5); err != nil {
		t.Fatal(err)
	}
	if w := blobAs(t, a.d, b.d.id, imageRoom, hashOf([]byte("wanted")), 0); w.Code != 404 {
		t.Errorf("an image not held gives %d", w.Code)
	}
	_ = ref
}

func TestTheTimeAnImageMayTakeGrowsWithItsSize(t *testing.T) {
	small, large := imageTimeout(250<<10), imageTimeout(maxImageBytes)
	if small < 20*time.Second || large <= small || large > 5*time.Minute {
		t.Errorf("timeouts %v and %v", small, large)
	}
	if imageTimeout(maxImageBytes) < 60*time.Second {
		t.Error("a large image on a slow relay would end before it arrived")
	}
	if imageBackoff(1) != 5*time.Second || imageBackoff(2) != 10*time.Second || imageBackoff(100) != 10*time.Minute {
		t.Errorf("backoff %v %v %v", imageBackoff(1), imageBackoff(2), imageBackoff(100))
	}
}

// A peer that does not say how many bytes it sends, or says a number other than what remains, is
// not read, since the signed event names the size.
func TestAPeerThatMisstatesTheLengthIsNotRead(t *testing.T) {
	a, b := imagePair(t)
	img := pngBytes(t, 21)
	ref := a.shareImage(t, img, nil)
	b.d.pullFrom(a.addr)

	for name, serve := range map[string]func(http.ResponseWriter){
		"a shorter length": func(w http.ResponseWriter) {
			w.Header().Set("Content-Length", "10")
			_, _ = w.Write(img[:10])
		},
		"a longer length": func(w http.ResponseWriter) {
			w.Header().Set("Content-Length", "1000000")
			_, _ = w.Write(img)
		},
		"no length": func(w http.ResponseWriter) {
			w.(http.Flusher).Flush()
			_, _ = w.Write(img)
		},
	} {
		serve := serve
		fake := servePeerTLSHandler(t, a.d, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { serve(w) }))
		b.d.imagePeers = func(Room) []string { return []string{fake} }
		if _, err := b.store.db.Exec(`UPDATE images SET next_try = ''`); err != nil {
			t.Fatal(err)
		}
		if n := b.d.fetchWanted(); n != 0 {
			t.Errorf("%s: an image was kept (%d)", name, n)
		}
		path, _ := b.store.imagePath(ref.SHA256)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("%s: a file was stored", name)
		}
		_ = os.Remove(path + ".part")
	}
}
