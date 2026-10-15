package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Codex shows a pasted image only after its prompt has been published, so the image goes in a
// second event that points at the prompt's (D-214).

type codexImage struct {
	n    int
	path string
	data []byte
}

// codexUserLine is a rollout line for a user's message, shaped as Codex 0.159.3 wrote one.
func codexUserLine(turnID string, imgs []codexImage, text string) string {
	var content []map[string]any
	for _, im := range imgs {
		open := "<image name=[Image #" + string(rune('0'+im.n)) + "]"
		if im.path != "" {
			open += ` path="` + im.path + `"`
		}
		content = append(content,
			map[string]any{"type": "input_text", "text": open + ">"},
			map[string]any{"type": "input_image", "image_url": "data:image/png;base64," + base64.StdEncoding.EncodeToString(im.data), "detail": "high"},
			map[string]any{"type": "input_text", "text": "</image>"})
	}
	content = append(content, map[string]any{"type": "input_text", "text": text})
	b, _ := json.Marshal(map[string]any{"timestamp": "2026-10-15T08:38:58Z", "type": "response_item", "payload": map[string]any{
		"type": "message", "role": "user", "content": content,
		"internal_chat_message_metadata_passthrough": map[string]any{"turn_id": turnID}}})
	return string(b) + "\n"
}

// codexInstructions is the user message Codex writes into a turn before the prompt, which
// holds the project's instructions and no image.
func codexInstructions(turnID string) string {
	b, _ := json.Marshal(map[string]any{"type": "response_item", "payload": map[string]any{
		"type": "message", "role": "user",
		"content": []map[string]any{{"type": "input_text", "text": "# AGENTS.md instructions\n\n<INSTRUCTIONS>"}, {"type": "input_text", "text": "<environment_context>"}},
		"internal_chat_message_metadata_passthrough": map[string]any{"turn_id": turnID}}})
	return string(b) + "\n"
}

func writeRollout(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	writeFile(t, path, []byte(strings.Join(lines, "")))
	return path
}

func TestCodexTurnImagesReadsOnlyTheNamedTurnsUserMessage(t *testing.T) {
	disk := filepath.Join(t.TempDir(), "codex-clipboard-abc.png")
	img1, img2 := pngBytes(t, 61), pngBytes(t, 62)
	writeFile(t, disk, img1)
	other := codexUserLine("turn-other", []codexImage{{1, "", pngBytes(t, 99)}}, "[Image #1] an earlier turn")
	assistant := strings.Replace(codexUserLine("turn-1", []codexImage{{1, "", pngBytes(t, 98)}}, "x"), `"role":"user"`, `"role":"assistant"`, 1)
	meta := `{"type":"session_meta","payload":{"id":"s"}}` + "\n"
	mine := codexUserLine("turn-1", []codexImage{{1, disk, img1}, {2, "", img2}}, "[Image #1] [Image #2] compare")
	path := writeRollout(t, meta, other, codexInstructions("turn-1"), assistant, mine)

	imgs, found, at, next := codexTurnImages(path, "turn-1", "[Image #1] [Image #2] compare", 0)
	if !found || len(imgs) != 2 {
		t.Fatalf("found %v, %+v", found, imgs)
	}
	if imgs[0].N != 1 || imgs[0].Path != disk || !bytes.Equal(imgs[0].Data, img1) {
		t.Errorf("first image: %+v", imgs[0])
	}
	if imgs[1].N != 2 || imgs[1].Path != "" || !bytes.Equal(imgs[1].Data, img2) {
		t.Errorf("second image: %+v", imgs[1])
	}
	if info, _ := os.Stat(path); next != info.Size() {
		t.Errorf("next offset %d, file is %d", next, info.Size())
	}
	if raw, _ := os.ReadFile(path); !strings.HasPrefix(string(raw[at:]), mine) {
		t.Errorf("the message is not at offset %d", at)
	}
	if _, found, _, _ := codexTurnImages(path, "turn-absent", "[Image #1] [Image #2] compare", 0); found {
		t.Error("a turn that is not there was found")
	}
	if _, found, _, _ := codexTurnImages(filepath.Join(t.TempDir(), "none.jsonl"), "turn-1", "x", 0); found {
		t.Error("a transcript that is not there was read")
	}
	// A message that names no image is found, so that the watch ends.
	noImage := writeRollout(t, codexInstructions("turn-2"), codexUserLine("turn-2", nil, "[Image #1] but nothing was attached"))
	if imgs, found, _, _ := codexTurnImages(noImage, "turn-2", "[Image #1] but nothing was attached", 0); !found || len(imgs) != 0 {
		t.Errorf("a message with no image: %v %+v", found, imgs)
	}
	// Only the message with the prompt's text counts, whatever else the turn holds.
	if _, found, _, _ := codexTurnImages(noImage, "turn-2", "some other prompt", 0); found {
		t.Error("a message that is not the prompt was found")
	}
	if _, found, _, _ := codexTurnImages(writeRollout(t, codexInstructions("turn-2")), "turn-2", "[Image #1] hi", 0); found {
		t.Error("the instructions of a turn were taken for its prompt")
	}
}

// Codex is still writing the line the daemon reads, so a line that is not complete is read again
// from its start and not taken as it is.
func TestCodexTurnImagesWaitsForALineToBeComplete(t *testing.T) {
	line := codexUserLine("turn-1", []codexImage{{1, "", pngBytes(t, 63)}}, "[Image #1]")
	path := writeRollout(t, line[:len(line)/2])
	imgs, found, _, next := codexTurnImages(path, "turn-1", "[Image #1]", 0)
	if found || imgs != nil || next != 0 {
		t.Fatalf("half a line: %v %v %d", found, imgs, next)
	}
	writeFile(t, path, []byte(line))
	imgs, found, _, _ = codexTurnImages(path, "turn-1", "[Image #1]", next)
	if !found || len(imgs) != 1 {
		t.Errorf("the whole line: %v %+v", found, imgs)
	}
}

func TestADataURLIsReadOnlyWhenItIsOneAndSmallEnough(t *testing.T) {
	img := pngBytes(t, 64)
	b64 := base64.StdEncoding.EncodeToString(img)
	if got := decodeDataURL("data:image/png;base64," + b64); !bytes.Equal(got, img) {
		t.Error("a data URL is not read")
	}
	for name, u := range map[string]string{
		"a web address":   "https://example.com/x.png",
		"not base64":      "data:image/png,abc",
		"bad base64":      "data:image/png;base64,!!!",
		"empty":           "",
		"no comma":        "data:image/png;base64",
		"far too large":   "data:image/png;base64," + strings.Repeat("A", maxImageBytes*2),
		"a file address":  "file:///etc/passwd",
		"another scheme":  "javascript:alert(1)",
		"missing prefix":  "image/png;base64," + b64,
		"a text data URL": "data:text/plain;charset=utf-8,hello",
	} {
		if got := decodeDataURL(u); got != nil {
			t.Errorf("%s is read as %d bytes", name, len(got))
		}
	}
}

func TestALateImageIsReadFromItsFileAndFromTheTranscriptWhenTheFileIsGone(t *testing.T) {
	disk := filepath.Join(t.TempDir(), "codex-clipboard-x.png")
	onDisk, inTranscript := pngBytes(t, 65), pngBytes(t, 66)
	writeFile(t, disk, onDisk)
	if p := prepareLate(lateImage{N: 1, Path: disk, Data: inTranscript}); p.ref.SHA256 != hashOf(onDisk) {
		t.Errorf("the file is not preferred: %+v", p.ref)
	}
	if p := prepareLate(lateImage{N: 1, Path: filepath.Join(t.TempDir(), "gone.png"), Data: inTranscript}); p.ref.SHA256 != hashOf(inTranscript) {
		t.Errorf("the transcript's bytes are not used when the file is gone: %+v", p.ref)
	}
	if p := prepareLate(lateImage{N: 3}); p.ref.Note != imageMissing || p.ref.N != 3 {
		t.Errorf("an image with no file and no bytes: %+v", p.ref)
	}
	if p := prepareLate(lateImage{N: 1, Data: []byte("not an image")}); p.ref.Note != imageUnsupported || p.data != nil {
		t.Errorf("bytes that are no image: %+v", p.ref)
	}
	if p := prepareLate(lateImage{N: 1, Data: append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, maxImageBytes)...)}); p.ref.Note != imageTooLarge || p.data != nil {
		t.Errorf("a large image: %+v", p.ref)
	}
}

// codexRoom is a daemon with a Codex session in a room, and a way to post a prompt as the hook does.
func codexRoom(t *testing.T) (*Daemon, Room, func(promptReq)) {
	t.Helper()
	m := testMembership(t)
	id := testIdentity(t)
	room, err := m.CreateRoom(id.PeerID, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.BindSession("codex:cs", room.RoomID); err != nil {
		t.Fatal(err)
	}
	d := &Daemon{id: id, members: m, imageWatchEvery: 5 * time.Millisecond, imageWatchWithin: 400 * time.Millisecond}
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

func roomEvents(t *testing.T, d *Daemon, room Room) []Event {
	t.Helper()
	store, _ := d.storeFor(room.RoomID)
	evs, err := store.ListRoom(room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	return evs
}

func TestACodexImageFollowsItsPromptInASecondEvent(t *testing.T) {
	d, room, post := codexRoom(t)
	img := pngBytes(t, 67)
	rollout := writeRollout(t, `{"type":"session_meta","payload":{}}`+"\n", codexInstructions("t1"))
	post(promptReq{Host: "codex", SessionID: "cs", TurnID: "t1", Prompt: "[Image #1] what is this?", TranscriptPath: rollout})

	// The prompt is published at once, with no image. Codex writes its record afterwards.
	evs := roomEvents(t, d, room)
	if len(evs) != 1 || evs[0].EventType != EventUserPrompt || len(imageRefs(evs[0].Metadata)) != 0 {
		t.Fatalf("after the hook: %+v", evs)
	}
	f, err := os.OpenFile(rollout, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(codexUserLine("t1", []codexImage{{1, "", img}}, "[Image #1] what is this?")); err != nil {
		t.Fatal(err)
	}
	f.Close()
	d.background.Wait()

	evs = roomEvents(t, d, room)
	if len(evs) != 2 || evs[1].EventType != EventUserImage {
		t.Fatalf("events: %+v", evs)
	}
	refs := imageRefs(evs[1].Metadata)
	if len(refs) != 1 || refs[0].SHA256 != hashOf(img) || refs[0].N != 1 || refs[0].Type != "image/png" {
		t.Fatalf("the second event names %+v", refs)
	}
	if imageEventTarget(evs[1].Metadata) != evs[0].EventID {
		t.Errorf("the second event points at %q, not the prompt %q", imageEventTarget(evs[1].Metadata), evs[0].EventID)
	}
	if err := evs[1].Verify(); err != nil {
		t.Errorf("the second event does not verify: %v", err)
	}
	store, _ := d.storeFor(room.RoomID)
	if row, ok, _ := store.ImageRow(hashOf(img)); !ok || row.State != imageHave {
		t.Errorf("the image is not stored: %+v", row)
	}
	// The view shows one turn, with the picture under it.
	st, err := d.viewSnapshot(room)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Events) != 1 || !strings.Contains(st.Events[0].HTML, `<img class="pasted" src="/image/`+room.RoomName+`/`+hashOf(img)+`"`) {
		t.Errorf("the view: %d turns, %s", len(st.Events), st.Events[0].HTML)
	}
}

func TestACodexPromptThatNamesNoImageStartsNoWatch(t *testing.T) {
	d, room, post := codexRoom(t)
	rollout := writeRollout(t, codexUserLine("t1", []codexImage{{1, "", pngBytes(t, 68)}}, "an image Codex has, that the prompt does not name"))
	post(promptReq{Host: "codex", SessionID: "cs", TurnID: "t1", Prompt: "no placeholder here", TranscriptPath: rollout})
	d.background.Wait()
	if evs := roomEvents(t, d, room); len(evs) != 1 {
		t.Errorf("%d events", len(evs))
	}
}

func TestAWatchEndsWhenTheTranscriptNeverShowsTheImages(t *testing.T) {
	d, room, post := codexRoom(t)
	rollout := writeRollout(t, codexUserLine("another-turn", []codexImage{{1, "", pngBytes(t, 69)}}, "x"))
	started := time.Now()
	post(promptReq{Host: "codex", SessionID: "cs", TurnID: "t1", Prompt: "[Image #1] nothing comes", TranscriptPath: rollout})
	d.background.Wait()
	if evs := roomEvents(t, d, room); len(evs) != 1 {
		t.Errorf("%d events; another turn's image was attached", len(evs))
	}
	if took := time.Since(started); took > 5*time.Second {
		t.Errorf("the watch ran for %v", took)
	}
}

// Only a host that shows images late is watched.
func TestAClaudePromptStartsNoWatchOfItsTranscript(t *testing.T) {
	d, room, post := promptRoom(t, false)
	rollout := writeRollout(t, codexUserLine("t1", []codexImage{{1, "", pngBytes(t, 70)}}, "x"))
	post(promptReq{SessionID: "s", PromptID: "p", TurnID: "t1", Prompt: "[Image #1] hi", TranscriptPath: rollout})
	d.background.Wait()
	if evs := roomEvents(t, d, room); len(evs) != 1 {
		t.Errorf("%d events", len(evs))
	}
}

func TestOnlyThePromptsAuthorAddsImagesToItsTurn(t *testing.T) {
	ref := imageRef{N: 1, SHA256: hashOf([]byte("a")), Type: "image/png", Size: 1}
	meta := func(target string) json.RawMessage {
		b, _ := json.Marshal(map[string]any{"images": []imageRef{ref}, "forEvent": target})
		return b
	}
	st := uiState{Events: []uiEvent{{EventID: "p1", PeerID: "author"}, {EventID: "p2", PeerID: "author"}}}
	mergeImageEvents(&st, []Event{
		{PeerID: "author", EventType: EventUserImage, Metadata: meta("p1")},
		{PeerID: "somebody-else", EventType: EventUserImage, Metadata: meta("p2")}, // not the author
		{PeerID: "author", EventType: EventUserImage, Metadata: meta("no-such-event")},
		{PeerID: "author", EventType: EventUserImage, Metadata: json.RawMessage(`not json`)},
	})
	if len(st.Events[0].Images) != 1 {
		t.Errorf("the author's image is not shown: %+v", st.Events[0].Images)
	}
	if len(st.Events[1].Images) != 0 {
		t.Errorf("an image from another peer is shown under the author's turn: %+v", st.Events[1].Images)
	}
	for i := 0; i < 3*maxImagesPerTurn; i++ {
		mergeImageEvents(&st, []Event{{PeerID: "author", EventType: EventUserImage, Metadata: meta("p1")}})
	}
	if len(st.Events[0].Images) > maxImagesPerTurn {
		t.Errorf("a turn shows %d images", len(st.Events[0].Images))
	}
	if imageEventTarget(json.RawMessage(`{"forEvent": 5}`)) != "" || imageEventTarget(nil) != "" {
		t.Error("a malformed target is read")
	}
}

// An image event is no turn: it is not offered to a colleague's session, and the view does not
// count it as one.
func TestAnImageEventIsNotATurn(t *testing.T) {
	d, room := testDaemon(t)
	other := testIdentity(t).PeerID
	store, _ := d.storeFor(room.RoomID)
	if _, err := store.Append(1, &Identity{PeerID: other, UserDisplayName: "X"}, room.RoomID, "theirs", EventUserPrompt, "a turn", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Append(2, &Identity{PeerID: other, UserDisplayName: "X"}, room.RoomID, "theirs", EventUserImage, "[pasted image 1]", map[string]any{"forEvent": "x"}); err != nil {
		t.Fatal(err)
	}
	got, err := store.UndeliveredFor(room.RoomID, "mine")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].EventType != EventUserPrompt {
		t.Errorf("a session is offered %+v", got)
	}
	st, _ := d.snapshot(room)
	if len(st.Events) != 1 {
		t.Errorf("the view counts %d turns", len(st.Events))
	}
}

func TestACodexHookWithNoTranscriptStartsNoWatch(t *testing.T) {
	d, room, post := codexRoom(t)
	post(promptReq{Host: "codex", SessionID: "cs", TurnID: "t1", Prompt: "[Image #1] but no transcript to read"})
	d.background.Wait()
	if evs := roomEvents(t, d, room); len(evs) != 1 {
		t.Errorf("%d events", len(evs))
	}
}

// Prompts with the same words are told apart by their turns, and by where their messages are.
func TestIdenticalPromptsGetTheirOwnImages(t *testing.T) {
	const words = "[Image #1] what is this?"
	a, b := pngBytes(t, 71), pngBytes(t, 72)
	inTurns := writeRollout(t,
		codexInstructions("t1"), codexUserLine("t1", []codexImage{{1, "", a}}, words),
		codexInstructions("t2"), codexUserLine("t2", []codexImage{{1, "", b}}, words))
	for turn, want := range map[string][]byte{"t1": a, "t2": b} {
		imgs, found, _, _ := codexTurnImages(inTurns, turn, words, 0)
		if !found || len(imgs) != 1 || !bytes.Equal(imgs[0].Data, want) {
			t.Errorf("turn %s: %v %+v", turn, found, imgs)
		}
	}
}

func TestTwoPromptsWithTheSameWordsInOneTurnTakeOneMessageEach(t *testing.T) {
	const words = "[Image #1] what is this?"
	d, room, post := codexRoom(t)
	a, b := pngBytes(t, 73), pngBytes(t, 74)
	rollout := writeRollout(t, codexInstructions("t1"),
		codexUserLine("t1", []codexImage{{1, "", a}}, words),
		codexUserLine("t1", []codexImage{{1, "", b}}, words))
	post(promptReq{Host: "codex", SessionID: "cs", TurnID: "t1", Prompt: words, TranscriptPath: rollout})
	post(promptReq{Host: "codex", SessionID: "cs", TurnID: "t1", Prompt: words, TranscriptPath: rollout})
	d.background.Wait()

	var got []string
	for _, e := range roomEvents(t, d, room) {
		if e.EventType == EventUserImage {
			for _, r := range imageRefs(e.Metadata) {
				got = append(got, r.SHA256)
			}
		}
	}
	if len(got) != 2 || got[0] == got[1] {
		t.Fatalf("the two prompts took %v; each message goes to one prompt", got)
	}
	for _, want := range [][]byte{a, b} {
		if got[0] != hashOf(want) && got[1] != hashOf(want) {
			t.Errorf("an image went to no prompt: %s", hashOf(want)[:12])
		}
	}
}

func TestAMessageIsClaimedOnce(t *testing.T) {
	d := &Daemon{}
	if !d.claimLateMessage("/a", 10) || d.claimLateMessage("/a", 10) {
		t.Error("a message can be claimed twice, or never")
	}
	if !d.claimLateMessage("/a", 11) || !d.claimLateMessage("/b", 10) {
		t.Error("a claim on one message blocks another")
	}
	for i := int64(0); i < 5000; i++ {
		d.claimLateMessage("/c", i)
	}
	if len(d.lateClaimed) > 4096 {
		t.Errorf("%d claims are kept", len(d.lateClaimed))
	}
}
