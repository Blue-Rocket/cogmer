package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

// An image a user pastes into a session travels with its turn (D-213). The turn's event
// names the image in its metadata, which the signature covers, and each receiver fetches the
// bytes by their hash and keeps them as a file.

const (
	// maxImageBytes is the largest image sent. A larger one is named in the turn's event
	// with its size and is not sent.
	maxImageBytes = 5 << 20
	// maxImagesPerTurn bounds the images one turn names.
	maxImagesPerTurn = 8
)

// imageTypes are the types that are stored and shown, found from the bytes. An SVG can carry
// script, so no vector type is among them.
var imageTypes = map[string]bool{"image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true}

// sniffImageType returns the type the first bytes of a file show, and whether it is one cogmer
// stores.
func sniffImageType(head []byte) (string, bool) {
	if len(head) > 512 {
		head = head[:512]
	}
	t := http.DetectContentType(head)
	return t, imageTypes[t]
}

// Reasons an image named in a turn has no bytes to show.
const (
	imageTooLarge = "too-large"
	// imageTooManyPixels is an image within the byte limit whose header puts it past the pixel limit.
	imageTooManyPixels = "too-many-pixels"
	imageMissing       = "missing"
	imageUnsupported   = "unsupported"
)

// imageRef is how a turn's event names one pasted image. The metadata is signed, so the hash
// binds the bytes to the author.
type imageRef struct {
	// N is the number in the prompt's "[Image #N]" placeholder.
	N int `json:"n,omitempty"`
	// SHA256 is the hash of the bytes in lower-case hexadecimal, empty when none were sent.
	SHA256 string `json:"sha256,omitempty"`
	Type   string `json:"type,omitempty"`
	Size   int64  `json:"size,omitempty"`
	// Note is why no bytes were sent: one of the constants above.
	Note string `json:"note,omitempty"`
}

var shaPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func validSHA(s string) bool { return shaPattern.MatchString(s) }

// imageRefs reads the images an event's metadata names. What is read is bounded and checked
// by whoever uses it, since the metadata came from a peer.
func imageRefs(meta json.RawMessage) []imageRef {
	if len(meta) == 0 {
		return nil
	}
	var m struct {
		Images []imageRef `json:"images"`
	}
	if json.Unmarshal(meta, &m) != nil {
		return nil
	}
	if len(m.Images) > maxImagesPerTurn {
		m.Images = m.Images[:maxImagesPerTurn]
	}
	return m.Images
}

var placeholderPattern = regexp.MustCompile(`\[Image #(\d+)\]`)

// imageNumbers is the numbers of the images a prompt names, each once, in the order named.
func imageNumbers(prompt string) []int {
	var out []int
	seen := map[int]bool{}
	for _, m := range placeholderPattern.FindAllStringSubmatch(prompt, -1) {
		n, err := strconv.Atoi(m[1])
		if err != nil || n <= 0 || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
		if len(out) == maxImagesPerTurn {
			break
		}
	}
	return out
}

// preparedImage is an image read from the host's files, ready to be stored and named.
type preparedImage struct {
	ref  imageRef
	data []byte // set only when the bytes are to be sent
}

// prepareImages reads the images a prompt names from the files its host saved them in. An image
// that cannot be sent is still named, with the reason, so a colleague sees that one was there.
func prepareImages(host Host, prompt string, ctx imageContext) []preparedImage {
	var out []preparedImage
	for _, f := range host.PastedImages(prompt, ctx) {
		out = append(out, prepareImage(f))
	}
	return out
}

func prepareImage(f pastedFile) preparedImage {
	ref := imageRef{N: f.N}
	if f.Path == "" {
		ref.Note = imageMissing
		log.Printf("image: the prompt names image #%d and the host saved no file for it", f.N)
		return preparedImage{ref: ref}
	}
	file, err := os.Open(f.Path)
	if err != nil {
		ref.Note = imageMissing
		log.Printf("image: image #%d: %v", f.N, err)
		return preparedImage{ref: ref}
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		ref.Note = imageMissing
		return preparedImage{ref: ref}
	}
	ref.Size = info.Size()
	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	typ, ok := sniffImageType(head[:n])
	if ok {
		ref.Type = typ
	}
	switch {
	case !ok:
		ref.Size = 0
		ref.Note = imageUnsupported
		return preparedImage{ref: ref}
	case info.Size() > maxImageBytes:
		ref.Note = imageTooLarge
		return preparedImage{ref: ref}
	}
	data := make([]byte, 0, info.Size())
	data = append(data, head[:n]...)
	rest := make([]byte, info.Size()-int64(n))
	if _, err := io.ReadFull(file, rest); err != nil {
		ref.Note = imageMissing
		ref.Size = 0
		ref.Type = ""
		return preparedImage{ref: ref}
	}
	data = append(data, rest...)
	if !imageWithinLimits(typ, data) {
		ref.Note = imageTooManyPixels
		ref.SHA256 = ""
		return preparedImage{ref: ref}
	}
	sum := sha256.Sum256(data)
	ref.SHA256 = hex.EncodeToString(sum[:])
	ref.Size = int64(len(data))
	return preparedImage{ref: ref, data: data}
}

// storeImages keeps the bytes of each prepared image and returns the references the turn's
// event carries. The bytes are stored before the event is, so an event never names an image
// its author cannot serve (D-213). An image that could not be stored is named as missing.
func storeImages(store *Store, prepared []preparedImage) []imageRef {
	var refs []imageRef
	for _, p := range prepared {
		ref := p.ref
		if p.data != nil {
			if err := store.SaveImage(ref.SHA256, ref.Type, p.data); err != nil {
				log.Printf("image: image #%d was not stored: %v", ref.N, err)
				ref = imageRef{N: ref.N, Note: imageMissing}
			}
		}
		refs = append(refs, ref)
	}
	return refs
}

// pastedImages reads the images sent with a prompt, for a session that is in a room. A session
// in none shares nothing, and so reads nothing.
func (d *Daemon) pastedImages(host Host, req promptReq) []preparedImage {
	room, ok := d.members.RoomForSession(req.SessionID)
	if !ok {
		return nil
	}
	store, err := d.storeFor(room.RoomID)
	if err != nil {
		return nil
	}
	attached, _ := store.AttachedImageNumbers(req.SessionID)
	return prepareImages(host, req.Prompt, imageContext{
		ScratchpadDir: req.ScratchpadDir, Entrypoint: req.Entrypoint, Attached: attached, Now: time.Now(),
	})
}
