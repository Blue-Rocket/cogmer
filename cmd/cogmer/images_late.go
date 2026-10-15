package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Some hosts show a pasted image only after the prompt it belongs to has been published (D-214).
// Codex writes the image into the session's transcript after the prompt hook has returned, and
// nothing at prompt time ties a file to the session. The prompt's event goes out at once. The
// daemon then reads the transcript until the image appears and publishes a second signed event
// that carries the image's hash and points at the prompt's event, since the prompt's own event is
// signed before the bytes exist and cannot change.

// lateImage is an image a host showed after its prompt: the number in the placeholder, the file
// the host saved it in if it names one, and the bytes the transcript holds.
type lateImage struct {
	N    int
	Path string
	Data []byte
}

// lateImageHost is a host whose images are read from its transcript after the prompt.
type lateImageHost interface {
	// LateImages reads the transcript from a byte offset for the user's message of one turn, which
	// is the one whose text is the prompt. It returns the images the message holds, whether the
	// message was found, the offset where the message starts, and the offset to continue from.
	LateImages(path, turnID, prompt string, offset int64) (imgs []lateImage, found bool, at, next int64)
}

func (codexHost) LateImages(path, turnID, prompt string, offset int64) ([]lateImage, bool, int64, int64) {
	return codexTurnImages(path, turnID, prompt, offset)
}

var codexImageOpen = regexp.MustCompile(`^<image name=\[Image #(\d+)\](?: path="([^"]*)")?>$`)

// codexTurnImages reads the images of one turn's user message from a Codex rollout. A turn has
// more than one message from the user, such as the project's instructions, so the message that
// counts is the one with the prompt's text. It holds, for each image, a text item naming it and
// the file it came from, then the image as a data URL, then a closing text item. Only lines that
// are complete are read, so an offset that stops in the middle of a line that Codex is still
// writing is not passed.
func codexTurnImages(path, turnID, prompt string, offset int64) (imgs []lateImage, found bool, at, next int64) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false, 0, offset
	}
	defer f.Close()
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, false, 0, offset
	}
	r := bufio.NewReaderSize(f, 1<<20)
	next = offset
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			return nil, false, 0, next // an incomplete last line is read again later
		}
		start := next
		next += int64(len(line))
		var rec struct {
			Payload struct {
				Type    string `json:"type"`
				Role    string `json:"role"`
				Content []struct {
					Type     string `json:"type"`
					Text     string `json:"text"`
					ImageURL string `json:"image_url"`
				} `json:"content"`
				Meta struct {
					TurnID string `json:"turn_id"`
				} `json:"internal_chat_message_metadata_passthrough"`
			} `json:"payload"`
		}
		if json.Unmarshal(line, &rec) != nil || rec.Payload.Type != "message" || rec.Payload.Role != "user" || rec.Payload.Meta.TurnID != turnID {
			continue
		}
		var out []lateImage
		var cur *lateImage
		isPrompt := false
		for _, c := range rec.Payload.Content {
			switch c.Type {
			case "input_text":
				if strings.TrimSpace(c.Text) == strings.TrimSpace(prompt) {
					isPrompt = true
				}
				if m := codexImageOpen.FindStringSubmatch(c.Text); m != nil {
					n, _ := strconv.Atoi(m[1])
					cur = &lateImage{N: n, Path: m[2]}
				}
			case "input_image":
				if cur == nil {
					continue
				}
				cur.Data = decodeDataURL(c.ImageURL)
				out = append(out, *cur)
				cur = nil
			}
		}
		if !isPrompt {
			continue
		}
		if cur != nil { // named and given no bytes
			out = append(out, *cur)
		}
		return out, true, start, next
	}
}

// decodeDataURL returns the bytes of a base64 data URL, or nothing if it is not one or is too
// large to be an image that is sent.
func decodeDataURL(u string) []byte {
	head, body, ok := strings.Cut(u, ",")
	if !ok || !strings.HasPrefix(head, "data:") || !strings.HasSuffix(head, ";base64") {
		return nil
	}
	if len(body) > maxImageBytes/3*4+8 {
		return nil
	}
	b, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return nil
	}
	return b
}

// prepareData is prepareImage for bytes that are already in memory.
func prepareData(n int, data []byte) preparedImage {
	ref := imageRef{N: n, Size: int64(len(data))}
	typ, ok := sniffImageType(data)
	switch {
	case !ok:
		ref.Size = 0
		ref.Note = imageUnsupported
		return preparedImage{ref: ref}
	case len(data) > maxImageBytes:
		ref.Type = typ
		ref.Note = imageTooLarge
		return preparedImage{ref: ref}
	}
	sum := sha256.Sum256(data)
	ref.Type, ref.SHA256 = typ, hex.EncodeToString(sum[:])
	return preparedImage{ref: ref, data: data}
}

// prepareLate reads a late image from the file the host saved, and from the bytes in the
// transcript if the file is gone.
func prepareLate(img lateImage) preparedImage {
	if img.Path != "" {
		if p := prepareImage(pastedFile{N: img.N, Path: img.Path}); p.ref.Note != imageMissing {
			return p
		}
	}
	if len(img.Data) > 0 {
		return prepareData(img.N, img.Data)
	}
	return preparedImage{ref: imageRef{N: img.N, Note: imageMissing}}
}

// watchForLateImages starts reading the session's transcript for the images of a prompt that
// names some, when the host shows them only after the prompt.
func (d *Daemon) watchForLateImages(host Host, room Room, store *Store, req promptReq, prompt *Event) {
	h, ok := host.(lateImageHost)
	if !ok || len(imageNumbers(req.Prompt)) == 0 {
		return
	}
	d.inBackground(func() {
		d.followLateImages(h, room, store, req, prompt.EventID)
	})
}

// followLateImages reads the transcript until the turn's user message appears or time runs out,
// and publishes the images in a second event.
func (d *Daemon) followLateImages(h lateImageHost, room Room, store *Store, req promptReq, promptEvent string) {
	every, within := d.imageWatchEvery, d.imageWatchWithin
	if every <= 0 {
		every = 250 * time.Millisecond
	}
	if within <= 0 {
		within = 10 * time.Second
	}
	deadline := time.Now().Add(within)
	var offset int64
	for {
		imgs, found, at, next := h.LateImages(req.TranscriptPath, req.TurnID, req.Prompt, offset)
		offset = next
		if found {
			// A message goes to one prompt. Two prompts with the same text in one turn would
			// otherwise both take the first message's images.
			if d.claimLateMessage(req.TranscriptPath, at) {
				d.publishLateImages(room, store, req.SessionID, promptEvent, imgs)
				return
			}
			continue
		}
		if time.Now().After(deadline) {
			log.Printf("image: the transcript of session %.8s did not show the images of a prompt within %s", req.SessionID, within)
			return
		}
		time.Sleep(every)
	}
}

// claimLateMessage reports whether this is the first claim on a message of a transcript, which
// is named by where it starts.
func (d *Daemon) claimLateMessage(path string, at int64) bool {
	d.lateMu.Lock()
	defer d.lateMu.Unlock()
	if d.lateClaimed == nil {
		d.lateClaimed = map[string]bool{}
	}
	if len(d.lateClaimed) >= 4096 { // a claim matters for seconds, so forgetting old ones is safe
		d.lateClaimed = map[string]bool{}
	}
	key := path + "\x00" + strconv.FormatInt(at, 10)
	if d.lateClaimed[key] {
		return false
	}
	d.lateClaimed[key] = true
	return true
}

func (d *Daemon) publishLateImages(room Room, store *Store, sessionID, promptEvent string, imgs []lateImage) {
	if len(imgs) > maxImagesPerTurn {
		imgs = imgs[:maxImagesPerTurn]
	}
	// Files are read and hashed before the lock is taken. Every write to a room's database is
	// made under it, because two writers at once can fail one of them with "database is locked",
	// and when that one is a prompt hook the prompt is lost.
	var prepared []preparedImage
	for _, img := range imgs {
		prepared = append(prepared, prepareLate(img))
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	refs := storeImages(store, prepared)
	if len(refs) == 0 {
		return
	}
	// Words for a build that does not know this event type, which shows it as a small turn.
	var names []string
	for _, r := range refs {
		names = append(names, "pasted image "+strconv.Itoa(r.N))
	}
	if _, err := d.appendLocal(store, room, sessionID, EventUserImage, "["+strings.Join(names, ", ")+"]",
		map[string]any{"images": refs, "forEvent": promptEvent}); err != nil {
		log.Printf("image: the images of a prompt were not published: %v", err)
		return
	}
	d.notify()
}

// imageEventTarget is the event an image event points at.
func imageEventTarget(meta json.RawMessage) string {
	var m struct {
		ForEvent string `json:"forEvent"`
	}
	if json.Unmarshal(meta, &m) != nil {
		return ""
	}
	return m.ForEvent
}
