package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A cloud task saves what a user attaches in an uploads folder, with no placeholder in the prompt
// and no number in the file's name, and an image is sent with the prompt when its file is fresh and
// this session has not sent the same bytes before (D-304).

func cloudUploads(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CLAUDE_CODE_REMOTE", "true")
	t.Setenv("COGMER_UPLOADS_DIR", dir)
	return dir
}

func TestAnImageAttachedInACloudTaskIsSentWithItsPrompt(t *testing.T) {
	uploads := cloudUploads(t)
	d, room, post := promptRoom(t, false)
	img := pngBytes(t, 21)
	writeFile(t, filepath.Join(uploads, "8d4b5751-image.png"), img)

	post(promptReq{SessionID: "s", PromptID: "p1", Prompt: "what is this?"})

	refs := imageRefs(lastPrompt(t, d, room).Metadata)
	if len(refs) != 1 || refs[0].SHA256 != hashOf(img) || refs[0].Type != "image/png" || refs[0].N != 1 {
		t.Fatalf("the event names %+v", refs)
	}
}

func TestAnImageACloudSessionAlreadySentIsNotSentAgain(t *testing.T) {
	uploads := cloudUploads(t)
	d, room, post := promptRoom(t, false)
	img := pngBytes(t, 22)
	writeFile(t, filepath.Join(uploads, "a-image.png"), img)
	post(promptReq{SessionID: "s", PromptID: "p1", Prompt: "first"})
	post(promptReq{SessionID: "s", PromptID: "p2", Prompt: "second, a moment later"})

	if refs := imageRefs(lastPrompt(t, d, room).Metadata); len(refs) != 0 {
		t.Fatalf("the second prompt sent the same image again: %+v", refs)
	}

	// A different picture saved for the next prompt is sent, as the second the session has sent.
	other := pngBytes(t, 23)
	writeFile(t, filepath.Join(uploads, "b-image.png"), other)
	post(promptReq{SessionID: "s", PromptID: "p3", Prompt: "third"})
	refs := imageRefs(lastPrompt(t, d, room).Metadata)
	if len(refs) != 1 || refs[0].SHA256 != hashOf(other) || refs[0].N != 2 {
		t.Fatalf("the third prompt named %+v", refs)
	}
}

func TestACloudSessionSendsNoImageThatWasSavedEarlierOrIsNotAnImage(t *testing.T) {
	uploads := cloudUploads(t)
	d, room, post := promptRoom(t, false)
	old := filepath.Join(uploads, "old-image.png")
	writeFile(t, old, pngBytes(t, 24))
	then := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(old, then, then); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(uploads, "notes.txt"), []byte("a note the user attached"))
	writeFile(t, filepath.Join(uploads, "fake.png"), []byte("not a picture at all"))

	post(promptReq{SessionID: "s", PromptID: "p1", Prompt: "no picture this time"})

	refs := imageRefs(lastPrompt(t, d, room).Metadata)
	for _, r := range refs {
		if r.SHA256 != "" {
			t.Fatalf("an image that was not sent with this prompt was sent: %+v", refs)
		}
	}
}

func TestAnUploadsFolderIsIgnoredOutsideACloudSession(t *testing.T) {
	uploads := t.TempDir()
	t.Setenv("CLAUDE_CODE_REMOTE", "")
	t.Setenv("COGMER_UPLOADS_DIR", uploads)
	d, room, post := promptRoom(t, false)
	writeFile(t, filepath.Join(uploads, "x-image.png"), pngBytes(t, 25))

	post(promptReq{SessionID: "s", PromptID: "p1", Prompt: "on a computer of the user's own"})

	if refs := imageRefs(lastPrompt(t, d, room).Metadata); len(refs) != 0 {
		t.Fatalf("a folder that only a cloud container holds was read on a computer: %+v", refs)
	}
}
