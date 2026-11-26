//go:build docker

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// A pasted image between a cloud session and a colleague's machine (D-213), in containers: the cloud
// session's hooks send no transcript path, as a Cowork cloud task's do, and the image is the file
// the host saved beside the session's scratchpad.

// pasteTurn has session on m say text, which names image 1, with the image saved where Claude Code
// saves it, and returns the sha-256 of the image. noTranscript sends the hooks with no transcript
// path, as a cloud task does.
func pasteTurn(t *testing.T, m *box, session, text string, png []byte, noTranscript bool) {
	t.Helper()
	scratch := "/h/" + session + "/scratchpad"
	if _, err := m.drive("write", "/h/"+session+"/images/1.png", png); err != nil {
		t.Fatalf("%s: saving the image: %v", m.name, err)
	}
	transcript := "/h/transcripts/" + session + ".jsonl"
	if noTranscript {
		transcript = ""
	}
	promptID := randomHex(8)
	prompt, _ := json.Marshal(map[string]any{
		"session_id": session, "prompt": text, "prompt_id": promptID, "cwd": "/h", "scratchpad_dir": scratch,
		"transcript_path": transcript, "hook_event_name": "UserPromptSubmit",
	})
	if _, errOut, err := m.exec(nil, prompt, "/cogmer", "hook", "prompt"); err != nil || strings.Contains(errOut, "daemon unreachable") {
		t.Fatalf("%s: the prompt hook failed: %v %s", m.name, err, errOut)
	}
	stop, _ := json.Marshal(map[string]any{
		"session_id": session, "prompt_id": promptID, "transcript_path": transcript, "cwd": "/h",
		"last_assistant_message": "I see the picture", "hook_event_name": "Stop",
	})
	if _, errOut, err := m.exec(nil, stop, "/cogmer", "hook", "stop"); err != nil || strings.Contains(errOut, "daemon unreachable") {
		t.Fatalf("%s: the stop hook failed: %v %s", m.name, err, errOut)
	}
}

// imagesOf is the images the view of room on m names for the turn whose text is text.
func imagesOf(t *testing.T, m *box, room, text string) []imageRef {
	t.Helper()
	raw, err := m.drive("get", "/events?room="+room, nil)
	if err != nil {
		t.Fatalf("%s: reading the view: %v", m.name, err)
	}
	var out struct {
		Events []struct {
			Content  string          `json:"content"`
			Metadata json.RawMessage `json:"metadata"`
		} `json:"events"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("%s: the view did not read: %v", m.name, err)
	}
	for _, e := range out.Events {
		if e.Content == text {
			return imageRefs(e.Metadata)
		}
	}
	return nil
}

// seesImage waits until m's view names image 1 of the turn text with the hash of png, and holds
// the bytes, which its image route then serves.
func seesImage(t *testing.T, m *box, room, text string, png []byte) {
	t.Helper()
	sum := sha256.Sum256(png)
	want := hex.EncodeToString(sum[:])
	eventually(t, 90*time.Second, m.name+" holding the image of "+text, func() (bool, string) {
		refs := imagesOf(t, m, room, text)
		if len(refs) != 1 || refs[0].SHA256 != want {
			return false, "the turn names " + toJSON(refs)
		}
		got, err := m.drive("get", "/image/"+room+"/"+want, nil)
		if err != nil || got != string(png) {
			return false, "the image route served " + toJSON(len(got)) + " bytes, " + toJSON(err)
		}
		return true, ""
	})
}

func toJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func TestContainerAPastedImageCrossesBetweenACloudSessionAndAColleague(t *testing.T) {
	c := newCluster(t)
	r := pairedRoom(t, c, harnessImage(t, imageWork), true)
	cloud, _ := addCloudBox(t, c, r.alice, "alice-web")
	cloudS := "session-cloud-1"
	cloudJoinByName(t, cloud, cloudS, r.name, 60*time.Second)

	// The cloud session pastes a picture, with the hooks of a cloud task: no transcript path.
	fromCloud := pngBytes(t, 11)
	pasteTurn(t, cloud, cloudS, "look at [Image #1] from the cloud", fromCloud, true)
	seesImage(t, r.bob, r.name, "look at [Image #1] from the cloud", fromCloud)
	seesImage(t, r.alice, r.name, "look at [Image #1] from the cloud", fromCloud)

	// And bob pastes one, which the cloud session's daemon takes and keeps.
	fromBob := pngBytes(t, 77)
	pasteTurn(t, r.bob, r.bobS, "look at [Image #1] from bob", fromBob, false)
	seesImage(t, cloud, r.name, "look at [Image #1] from bob", fromBob)
}

// A picture attached in a cloud task is saved in its uploads folder, with no placeholder in the
// prompt and no transcript path in the hooks (D-304), and reaches a colleague.
func TestContainerAnImageAttachedInACloudTaskReachesAColleague(t *testing.T) {
	c := newCluster(t)
	r := pairedRoom(t, c, harnessImage(t, imageWork), true)
	var added cloudAddResponse
	r.alice.mustPost("/machines/cloud/add", cloudAddRequest{Name: "alice-web"}, &added)
	cloud := c.startWith("alice-web", harnessImage(t, imageWork), cloudKeyVar+"="+added.Value, "COGMER_UPLOADS_DIR=/mnt/user-data/uploads")
	cloudS := "session-cloud-1"
	cloudJoinByName(t, cloud, cloudS, r.name, 60*time.Second)

	attached := pngBytes(t, 99)
	if _, err := cloud.drive("write", "/mnt/user-data/uploads/8d4b5751-image.png", attached); err != nil {
		t.Fatal(err)
	}
	text := "describe the attached picture"
	prompt, _ := json.Marshal(map[string]any{
		"session_id": cloudS, "prompt": text, "prompt_id": randomHex(8), "cwd": "/h",
		"transcript_path": "", "hook_event_name": "UserPromptSubmit",
	})
	if _, errOut, err := cloud.exec(nil, prompt, "/cogmer", "hook", "prompt"); err != nil || strings.Contains(errOut, "daemon unreachable") {
		t.Fatalf("the prompt hook failed: %v %s", err, errOut)
	}
	seesImage(t, r.bob, r.name, text, attached)
}
