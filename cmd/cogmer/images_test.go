package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A pasted image travels with its turn (D-213). The tests here follow one from the files the
// host saves it in to the event that names it.

// pngBytes is a small image that really is one.
func pngBytes(t *testing.T, shade uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			img.Set(x, y, color.RGBA{shade, uint8(x * 30), uint8(y * 30), 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func hashOf(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// sessionFiles is the directory Claude Code keeps a session's files in: a scratchpad, and
// the images directory beside it.
func sessionFiles(t *testing.T) (scratchpad, images string) {
	t.Helper()
	root := t.TempDir()
	scratchpad = filepath.Join(root, "scratchpad")
	images = filepath.Join(root, "images")
	for _, d := range []string{scratchpad, images} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return scratchpad, images
}

func writeFile(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestOnlyRasterImagesAreStored(t *testing.T) {
	jpeg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 20)...)
	gif := append([]byte("GIF89a"), make([]byte, 20)...)
	webp := append([]byte("RIFF\x00\x00\x00\x00WEBPVP8 "), make([]byte, 20)...)
	cases := []struct {
		name string
		in   []byte
		want string
		ok   bool
	}{
		{"png", pngBytes(t, 1), "image/png", true},
		{"jpeg", jpeg, "image/jpeg", true},
		{"gif", gif, "image/gif", true},
		{"webp", webp, "image/webp", true},
		{"svg, which can carry script", []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"></svg>`), "", false},
		{"html", []byte("<!doctype html><script>alert(1)</script>"), "", false},
		{"text", []byte("hello"), "", false},
		{"nothing", nil, "", false},
	}
	for _, c := range cases {
		got, ok := sniffImageType(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("%s: %q, %v", c.name, got, ok)
		}
	}
}

func TestAnImageIsFoundByItsPlaceholderEachOnce(t *testing.T) {
	got := imageNumbers("[Image #2] and [Image #1] and again [Image #2], not [Image #0] or [image #3] or [Image #x]")
	if len(got) != 2 || got[0] != 2 || got[1] != 1 {
		t.Errorf("numbers = %v", got)
	}
	var b strings.Builder
	for i := 1; i <= 30; i++ {
		fmt.Fprintf(&b, "[Image #%d] ", i)
	}
	if got := imageNumbers(b.String()); len(got) != maxImagesPerTurn {
		t.Errorf("a prompt naming 30 images yields %d, want %d", len(got), maxImagesPerTurn)
	}
}

func TestClaudeCodeSavesAPastedImageBesideTheScratchpad(t *testing.T) {
	scratchpad, images := sessionFiles(t)
	writeFile(t, filepath.Join(images, "1.png"), pngBytes(t, 1))
	writeFile(t, filepath.Join(images, "2.jpg"), []byte{0xFF, 0xD8, 0xFF})
	got := (claudeHost{}).PastedImages("[Image #1] [Image #2] [Image #3]", imageContext{ScratchpadDir: scratchpad})
	if len(got) != 3 || filepath.Base(got[0].Path) != "1.png" || filepath.Base(got[1].Path) != "2.jpg" || got[2].Path != "" || got[2].N != 3 {
		t.Errorf("found %+v", got)
	}
	if got := (claudeHost{}).PastedImages("[Image #1]", imageContext{}); got != nil {
		t.Errorf("a prompt with no scratchpad directory finds %+v", got)
	}
	if got := (claudeHost{}).PastedImages("no image here", imageContext{ScratchpadDir: scratchpad}); got != nil {
		t.Errorf("a prompt naming nothing finds %+v", got)
	}
	if got := (codexHost{}).PastedImages("[Image #1]", imageContext{ScratchpadDir: scratchpad}); got != nil {
		t.Errorf("Codex finds %+v", got)
	}
}

func TestPreparingAnImageSaysWhyOneIsNotSent(t *testing.T) {
	_, images := sessionFiles(t)
	good := pngBytes(t, 3)
	writeFile(t, filepath.Join(images, "1.png"), good)
	writeFile(t, filepath.Join(images, "2.png"), append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, maxImageBytes)...))
	writeFile(t, filepath.Join(images, "3.png"), []byte("not an image at all"))
	p := func(n int, path string) preparedImage { return prepareImage(pastedFile{N: n, Path: path}) }

	ok := p(1, filepath.Join(images, "1.png"))
	if ok.ref.SHA256 != hashOf(good) || ok.ref.Type != "image/png" || ok.ref.Size != int64(len(good)) || ok.ref.Note != "" || !bytes.Equal(ok.data, good) {
		t.Errorf("an image: %+v", ok.ref)
	}
	big := p(2, filepath.Join(images, "2.png"))
	if big.ref.Note != imageTooLarge || big.data != nil || big.ref.SHA256 != "" || big.ref.Size <= maxImageBytes || big.ref.Type != "image/png" {
		t.Errorf("a large image: %+v", big.ref)
	}
	text := p(3, filepath.Join(images, "3.png"))
	if text.ref.Note != imageUnsupported || text.data != nil || text.ref.SHA256 != "" || text.ref.Type != "" {
		t.Errorf("a file that is no image: %+v", text.ref)
	}
	gone := p(4, filepath.Join(images, "none.png"))
	if gone.ref.Note != imageMissing || gone.data != nil {
		t.Errorf("a file that is not there: %+v", gone.ref)
	}
	unnamed := p(5, "")
	if unnamed.ref.Note != imageMissing || unnamed.ref.N != 5 {
		t.Errorf("an image no file was saved for: %+v", unnamed.ref)
	}
	dir := p(6, images)
	if dir.ref.Note != imageMissing {
		t.Errorf("a directory: %+v", dir.ref)
	}
}

// promptRoom is a daemon with a session in a room, and a way to post a prompt as the hook does.
func promptRoom(t *testing.T, showOnly bool) (*Daemon, Room, func(promptReq)) {
	t.Helper()
	m := testMembership(t)
	id := testIdentity(t)
	room, err := m.CreateRoom(id.PeerID, showOnly)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.BindSession("s", room.RoomID); err != nil {
		t.Fatal(err)
	}
	d := &Daemon{id: id, members: m}
	t.Cleanup(d.closeStores)
	t.Cleanup(d.background.Wait)
	post := func(req promptReq) {
		t.Helper()
		b, _ := json.Marshal(req)
		r := httptest.NewRequest(http.MethodPost, "/hook/prompt", bytes.NewReader(b))
		r.Header.Set(localGuardHeader, localGuardValue)
		w := httptest.NewRecorder()
		d.LocalRoutes().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("/hook/prompt: %d %s", w.Code, w.Body.String())
		}
	}
	return d, room, post
}

func lastPrompt(t *testing.T, d *Daemon, room Room) Event {
	t.Helper()
	store, err := d.storeFor(room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	evs, err := store.ListRoom(room.RoomID)
	if err != nil || len(evs) == 0 {
		t.Fatalf("no events: %v", err)
	}
	return evs[len(evs)-1]
}

func TestAPastedImageIsStoredAndNamedInTheEventsSignedMetadata(t *testing.T) {
	d, room, post := promptRoom(t, false)
	scratchpad, images := sessionFiles(t)
	img := pngBytes(t, 7)
	writeFile(t, filepath.Join(images, "1.png"), img)

	post(promptReq{SessionID: "s", PromptID: "p", Prompt: "[Image #1] what is this?", ScratchpadDir: scratchpad})

	ev := lastPrompt(t, d, room)
	refs := imageRefs(ev.Metadata)
	if len(refs) != 1 || refs[0].SHA256 != hashOf(img) || refs[0].Type != "image/png" || refs[0].Size != int64(len(img)) || refs[0].N != 1 {
		t.Fatalf("the event names %+v", refs)
	}
	if ev.Content != "[Image #1] what is this?" {
		t.Errorf("the turn's text changed: %q", ev.Content)
	}
	store, _ := d.storeFor(room.RoomID)
	path, err := store.imagePath(refs[0].SHA256)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, img) {
		t.Fatalf("the stored file is not the image: %v", err)
	}
	if info, _ := os.Stat(path); modesMeanSomething() && info.Mode().Perm() != 0o600 {
		t.Errorf("the file is readable by others: %v", info.Mode())
	}
	if info, _ := os.Stat(store.imageDir); modesMeanSomething() && info.Mode().Perm() != 0o700 {
		t.Errorf("the directory is open to others: %v", info.Mode())
	}
	if row, ok, _ := store.ImageRow(refs[0].SHA256); !ok || row.State != imageHave {
		t.Errorf("the room does not record that it holds the image: %+v", row)
	}
	if err := ev.Verify(); err != nil {
		t.Errorf("the event does not verify: %v", err)
	}
}

// The hash is what stops a peer from showing a different picture under the author's name.
func TestTheImageHashIsCoveredByTheSignature(t *testing.T) {
	d, room, post := promptRoom(t, false)
	scratchpad, images := sessionFiles(t)
	writeFile(t, filepath.Join(images, "1.png"), pngBytes(t, 9))
	post(promptReq{SessionID: "s", PromptID: "p", Prompt: "[Image #1]", ScratchpadDir: scratchpad})
	ev := lastPrompt(t, d, room)
	ref := imageRefs(ev.Metadata)[0]
	if err := ev.Verify(); err != nil {
		t.Fatal(err)
	}
	swapped := strings.Replace(string(ev.Metadata), ref.SHA256, hashOf([]byte("another picture")), 1)
	if swapped == string(ev.Metadata) {
		t.Fatal("the hash is not in the metadata")
	}
	ev.Metadata = json.RawMessage(swapped)
	if err := ev.Verify(); err == nil {
		t.Error("an event naming a different image verifies")
	}
}

func TestAPromptNamesOnlyTheImagesThatItHolds(t *testing.T) {
	scratchpad, images := sessionFiles(t)
	writeFile(t, filepath.Join(images, "1.png"), pngBytes(t, 1))
	writeFile(t, filepath.Join(images, "2.png"), append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, maxImageBytes)...))
	writeFile(t, filepath.Join(images, "3.png"), []byte("plain text"))

	d, room, post := promptRoom(t, false)
	post(promptReq{SessionID: "s", PromptID: "a", Prompt: "no image in this one", ScratchpadDir: scratchpad})
	if m := string(lastPrompt(t, d, room).Metadata); strings.Contains(m, "images") {
		t.Errorf("a prompt naming no image carries an images field: %s", m)
	}

	post(promptReq{SessionID: "s", PromptID: "b", Prompt: "[Image #1] [Image #2] [Image #3] [Image #4]", ScratchpadDir: scratchpad})
	refs := imageRefs(lastPrompt(t, d, room).Metadata)
	if len(refs) != 4 {
		t.Fatalf("refs = %+v", refs)
	}
	if refs[0].SHA256 == "" || refs[0].Note != "" {
		t.Errorf("the image that could be sent: %+v", refs[0])
	}
	for i, want := range []string{"", imageTooLarge, imageUnsupported, imageMissing} {
		if refs[i].Note != want {
			t.Errorf("image %d note = %q, want %q", i+1, refs[i].Note, want)
		}
		if want != "" && refs[i].SHA256 != "" {
			t.Errorf("image %d has no bytes to send and names a hash: %+v", i+1, refs[i])
		}
	}
	store, _ := d.storeFor(room.RoomID)
	rows, _ := store.ImageRows()
	if len(rows) != 1 {
		t.Errorf("the room holds %d images, want the one that was sent", len(rows))
	}
}

func TestAShowOnlyRoomSharesAPastedImageToo(t *testing.T) {
	d, room, post := promptRoom(t, true)
	scratchpad, images := sessionFiles(t)
	img := pngBytes(t, 5)
	writeFile(t, filepath.Join(images, "1.png"), img)
	post(promptReq{SessionID: "s", PromptID: "p", Prompt: "[Image #1]", ScratchpadDir: scratchpad})
	refs := imageRefs(lastPrompt(t, d, room).Metadata)
	if len(refs) != 1 || refs[0].SHA256 != hashOf(img) {
		t.Errorf("refs = %+v", refs)
	}
}

func TestTheImageTableIsAddedToARoomThatPredatesIt(t *testing.T) {
	setHome(t, t.TempDir())
	dir := filepath.Join(homeDir(), "rooms")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	old := schema[:strings.Index(schema, "-- Where each pasted image")]
	db, err := sql.Open("sqlite", filepath.Join(dir, "old.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(old); err != nil {
		t.Fatal(err)
	}
	db.Close()
	s, err := OpenStore("old")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.WantImage(hashOf([]byte("x")), "image/png", 1); err != nil {
		t.Errorf("a room created before images cannot record one: %v", err)
	}
}

func TestAnImageNameCannotLeaveItsDirectory(t *testing.T) {
	s := &Store{imageDir: t.TempDir()}
	for _, name := range []string{"../x", "a/b", "..", "", strings.Repeat("g", 64), strings.Repeat("A", 64), strings.Repeat("a", 63)} {
		if _, err := s.imagePath(name); err == nil {
			t.Errorf("%q is accepted as an image name", name)
		}
	}
	if _, err := s.imagePath(strings.Repeat("a", 64)); err != nil {
		t.Error(err)
	}
}

// An event comes from a peer, so how many images it names is bounded where it is read.
func TestAPeersEventCannotNameManyImages(t *testing.T) {
	var refs []imageRef
	for i := 0; i < 500; i++ {
		refs = append(refs, imageRef{N: i + 1, SHA256: hashOf([]byte{byte(i), byte(i >> 8)}), Type: "image/png", Size: 10})
	}
	meta, _ := json.Marshal(map[string]any{"images": refs})
	if got := imageRefs(meta); len(got) != maxImagesPerTurn {
		t.Errorf("an event naming 500 images yields %d", len(got))
	}
	if got := imageRefs(json.RawMessage(`{"images": "not a list"}`)); got != nil {
		t.Errorf("malformed metadata yields %+v", got)
	}
	if got := imageRefs(nil); got != nil {
		t.Errorf("no metadata yields %+v", got)
	}
}

// The README says a colleague's model is given the turn's text and not the image, so the event's
// metadata, where the image is named, is not part of what is injected.
func TestWhatIsInjectedHoldsTheTextAndNotTheImage(t *testing.T) {
	d, room, post := promptRoom(t, false)
	scratchpad, images := sessionFiles(t)
	img := pngBytes(t, 41)
	writeFile(t, filepath.Join(images, "1.png"), img)
	post(promptReq{SessionID: "s", PromptID: "p", Prompt: "[Image #1] what is this?", ScratchpadDir: scratchpad})
	ev := lastPrompt(t, d, room)
	block := FormatTeamContext([]Event{ev}, nil)
	if !strings.Contains(block, "[Image #1] what is this?") {
		t.Errorf("the turn's text is not injected: %s", block)
	}
	for _, leak := range []string{hashOf(img), "image/png", "sha256", "images"} {
		if strings.Contains(block, leak) {
			t.Errorf("the injected block holds %q", leak)
		}
	}
}

// age sets how long ago a file was saved.
func age(t *testing.T, path string, ago time.Duration) {
	t.Helper()
	at := time.Now().Add(-ago)
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
}

// The terminal saves an image when it is pasted, so a file is there for an image the user then
// took out of the prompt. Sending it would share what the user chose not to send.
func TestTheTerminalSendsOnlyTheImagesThePromptNames(t *testing.T) {
	scratchpad, images := sessionFiles(t)
	writeFile(t, filepath.Join(images, "1.png"), pngBytes(t, 1))
	writeFile(t, filepath.Join(images, "2.png"), pngBytes(t, 2))
	for _, entry := range []string{"cli", "", "claude-vscode", "sdk-ts"} {
		ctx := imageContext{ScratchpadDir: scratchpad, Entrypoint: entry, Now: time.Now()}
		got := (claudeHost{}).PastedImages("[Image #2] only this one", ctx)
		if len(got) != 1 || got[0].N != 2 {
			t.Errorf("entry point %q: found %+v", entry, got)
		}
		if got := (claudeHost{}).PastedImages("the placeholder was deleted", ctx); got != nil {
			t.Errorf("entry point %q sends an image the prompt does not name: %+v", entry, got)
		}
	}
}

// The desktop app saves an image as the prompt is sent and puts no placeholder in the text, so
// what it saved just now, and has not sent before, was sent with this prompt.
func TestTheDesktopAppSendsWhatItSavedJustNowAndNotSentBefore(t *testing.T) {
	scratchpad, images := sessionFiles(t)
	for _, n := range []string{"1.png", "2.jpg", "3.png", "4.png", "notes.txt", "5.png.bak", "x.png"} {
		writeFile(t, filepath.Join(images, n), pngBytes(t, 1))
	}
	if err := os.Mkdir(filepath.Join(images, "6.png"), 0o755); err != nil {
		t.Fatal(err)
	}
	age(t, filepath.Join(images, "1.png"), time.Second)
	age(t, filepath.Join(images, "2.jpg"), 10*time.Second)
	age(t, filepath.Join(images, "3.png"), 5*time.Minute) // saved for an earlier prompt
	age(t, filepath.Join(images, "4.png"), time.Second)   // sent already
	ctx := imageContext{ScratchpadDir: scratchpad, Entrypoint: desktopEntrypoint, Now: time.Now(), Attached: map[int]bool{4: true}}
	got := (claudeHost{}).PastedImages("what is this", ctx)
	if len(got) != 2 || got[0].N != 1 || got[1].N != 2 {
		t.Fatalf("found %+v", got)
	}
	if len(got) == 2 && filepath.Base(got[1].Path) != "2.jpg" {
		t.Errorf("path %s", got[1].Path)
	}
	// A file dated in the future is not this prompt's either.
	age(t, filepath.Join(images, "2.jpg"), -5*time.Minute)
	if got := (claudeHost{}).PastedImages("x", ctx); len(got) != 1 {
		t.Errorf("a file saved in the future is read: %+v", got)
	}
	// And no more than a turn may name.
	many := t.TempDir()
	sp := filepath.Join(many, "scratchpad")
	_ = os.MkdirAll(filepath.Join(many, "images"), 0o755)
	for i := 1; i <= 20; i++ {
		writeFile(t, filepath.Join(many, "images", fmt.Sprintf("%d.png", i)), pngBytes(t, 1))
	}
	if got := (claudeHost{}).PastedImages("x", imageContext{ScratchpadDir: sp, Entrypoint: desktopEntrypoint, Now: time.Now()}); len(got) != maxImagesPerTurn {
		t.Errorf("a prompt takes %d images", len(got))
	}
}

func TestADesktopPromptSendsEachImageOnce(t *testing.T) {
	d, room, post := promptRoom(t, false)
	scratchpad, images := sessionFiles(t)
	first := pngBytes(t, 51)
	writeFile(t, filepath.Join(images, "1.png"), first)
	post(promptReq{SessionID: "s", PromptID: "a", Prompt: "what is this", ScratchpadDir: scratchpad, Entrypoint: desktopEntrypoint})
	refs := imageRefs(lastPrompt(t, d, room).Metadata)
	if len(refs) != 1 || refs[0].SHA256 != hashOf(first) || refs[0].N != 1 {
		t.Fatalf("the first prompt names %+v", refs)
	}
	post(promptReq{SessionID: "s", PromptID: "b", Prompt: "and a question with no image", ScratchpadDir: scratchpad, Entrypoint: desktopEntrypoint})
	if m := string(lastPrompt(t, d, room).Metadata); strings.Contains(m, "images") {
		t.Errorf("an image sent once is sent again: %s", m)
	}
	second := pngBytes(t, 52)
	writeFile(t, filepath.Join(images, "2.png"), second)
	post(promptReq{SessionID: "s", PromptID: "c", Prompt: "another", ScratchpadDir: scratchpad, Entrypoint: desktopEntrypoint})
	refs = imageRefs(lastPrompt(t, d, room).Metadata)
	if len(refs) != 1 || refs[0].SHA256 != hashOf(second) || refs[0].N != 2 {
		t.Errorf("the third prompt names %+v", refs)
	}
}

// What a terminal user pasted and removed is not sent even when the prompt that follows has no
// placeholder, and the daemon cannot be told otherwise by a file's being there.
func TestATerminalPromptWithoutAPlaceholderSendsNoImage(t *testing.T) {
	d, room, post := promptRoom(t, false)
	scratchpad, images := sessionFiles(t)
	writeFile(t, filepath.Join(images, "1.png"), pngBytes(t, 53))
	post(promptReq{SessionID: "s", PromptID: "a", Prompt: "no image", ScratchpadDir: scratchpad, Entrypoint: "cli"})
	if m := string(lastPrompt(t, d, room).Metadata); strings.Contains(m, "images") {
		t.Errorf("an image the prompt does not name is sent: %s", m)
	}
	store, _ := d.storeFor(room.RoomID)
	if rows, _ := store.ImageRows(); len(rows) != 0 {
		t.Errorf("the room holds %d images", len(rows))
	}
}

func TestTheHookReportsWhichProgramRunsTheSession(t *testing.T) {
	p := map[string]any{"prompt": "x"}
	addEntrypoint(p, func(k string) string {
		if k != "CLAUDE_CODE_ENTRYPOINT" {
			t.Errorf("reads %s", k)
		}
		return "claude-desktop"
	})
	if p["entrypoint"] != "claude-desktop" {
		t.Errorf("payload %v", p)
	}
	q := map[string]any{}
	addEntrypoint(q, func(string) string { return "" })
	if _, ok := q["entrypoint"]; ok {
		t.Errorf("an empty entry point is sent: %v", q)
	}
}
