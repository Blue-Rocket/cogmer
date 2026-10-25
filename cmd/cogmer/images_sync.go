package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Images move on their own, apart from the events that name them (D-213). An event is a few
// kilobytes of JSON on a request that a poll makes every second, and an image is up to
// maxImageBytes of raw bytes, so a slow image must not hold up a turn. A peer that has an image
// serves it on /blob, and a peer that lacks one asks for it there, whichever peer wrote the turn.

// registerImages records the images an event that arrived from a peer names, so the fetcher
// knows what is wanted. What the metadata says is checked first, since it came from a peer:
// a hash that is no hash, a size past the cap and a type that is not stored are not wanted.
func registerImages(store *Store, ev *Event) {
	for _, ref := range imageRefs(ev.Metadata) {
		if ref.Note != "" || !validSHA(ref.SHA256) || ref.Size <= 0 || ref.Size > maxImageBytes || !imageTypes[ref.Type] {
			continue
		}
		if err := store.WantImage(ref.SHA256, ref.Type, ref.Size); err != nil {
			log.Printf("image: could not record that %.12s is wanted: %v", ref.SHA256, err)
		}
	}
}

// blobRequest asks a peer for one image of a room, from a byte offset so that an attempt
// that stopped part way continues where it stopped.
type blobRequest struct {
	Protocol int    `json:"protocol"`
	RoomID   string `json:"roomId"`
	SHA256   string `json:"sha256"`
	Offset   int64  `json:"offset"`
	// Credentials, as a sync request carries them.
	PeerID    string `json:"peerId"`
	Timestamp string `json:"timestamp"`
	Nonce     string `json:"nonce"`
	Signature string `json:"signature"`
	Endpoint  string `json:"endpoint,omitempty"`
	Signer    string `json:"signer,omitempty"`
	Chain     *Chain `json:"chain,omitempty"`
}

// handleBlob serves the bytes of an image to a peer that is a verified guest of the room, and
// to nobody else. It answers only for images this machine holds.
func (d *Daemon) handleBlob(w http.ResponseWriter, r *http.Request) {
	var req blobRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !speaks(req.Protocol) {
		http.Error(w, "unsupported protocol", http.StatusBadRequest)
		return
	}
	room, ok := d.members.RoomByID(req.RoomID)
	// An expelled user's machine serves the room to nobody (D-241).
	if !ok || d.expelledFrom(room.RoomID) {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	cred := syncRequest{PeerID: req.PeerID, RoomID: req.RoomID, Timestamp: req.Timestamp,
		Nonce: req.Nonce, Signature: req.Signature, Endpoint: req.Endpoint, Signer: req.Signer, Chain: req.Chain}
	if err := connectedAs(r, cred.requestMachine()); err != nil {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	if err := d.verifyRequest(cred, room.RoomID); err != nil {
		log.Printf("image: refused a request for %s: %v", room.RoomName, err)
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	d.markPeerHeardFrom(req.PeerID)
	if !validSHA(req.SHA256) {
		http.Error(w, "not an image hash", http.StatusBadRequest)
		return
	}
	store, err := d.storeFor(room.RoomID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Only an image this room knows of is served, though the directory holds this room's
	// images alone. A file that is not there, such as one still being fetched, is not found.
	if _, found, err := store.ImageRow(req.SHA256); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if !found || d.imageOnlyHidden(store, room.RoomID, req.SHA256) {
		// An image only a suspended machine's turns name is served no more than they are
		// (D-234).
		http.NotFound(w, r)
		return
	}
	path, err := store.imagePath(req.SHA256)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || req.Offset < 0 || req.Offset > info.Size() {
		http.Error(w, "offset outside the image", http.StatusBadRequest)
		return
	}
	if _, err := f.Seek(req.Offset, io.SeekStart); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", fmt.Sprint(info.Size()-req.Offset))
	_, _ = io.Copy(w, f)
}

// imageTimeout is how long one attempt to fetch an image may take. It grows with the bytes
// still to come, since a limit that suits a turn would end a large image on a slow relay
// before it arrived and start it again from nothing.
func imageTimeout(remaining int64) time.Duration {
	return 20*time.Second + time.Duration(remaining/(50<<10))*time.Second
}

// imageBackoff is the wait after the nth failed attempt to fetch an image.
func imageBackoff(attempts int) time.Duration {
	wait := 5 * time.Second
	for i := 1; i < attempts && wait < 10*time.Minute; i++ {
		wait *= 2
	}
	if wait > 10*time.Minute {
		wait = 10 * time.Minute
	}
	return wait
}

// RunImages fetches the images that rooms are waiting for, on a loop of its own.
func (d *Daemon) RunImages(every time.Duration) {
	for {
		d.fetchWanted()
		time.Sleep(every)
	}
}

// fetchWanted makes one pass over every room's wanted images whose next attempt is due, and
// returns how many it fetched.
func (d *Daemon) fetchWanted() int {
	rooms, err := d.members.Rooms()
	if err != nil {
		return 0
	}
	fetched := 0
	for _, room := range rooms {
		store, err := d.storeFor(room.RoomID)
		if err != nil {
			continue
		}
		wanted, err := store.WantedImages(time.Now())
		if err != nil {
			continue
		}
		for _, row := range wanted {
			if d.fetchImage(room, store, row) {
				fetched++
			}
		}
	}
	return fetched
}

// imagePeers is where a room's images can be asked for: every guest's address. A peer other
// than the author may hold the bytes, and the hash in the signed event is what makes asking
// any of them safe.
func (d *Daemon) imagePeersFor(room Room) []string {
	if d.imagePeers != nil {
		return d.imagePeers(room)
	}
	return d.members.RoomPeers(room.RoomID, d.id.PeerID)
}

func (d *Daemon) fetchImage(room Room, store *Store, row imageRow) bool {
	var lastErr error
	for _, addr := range d.imagePeersFor(room) {
		err := d.fetchImageFrom(addr, room, store, row)
		if err == nil {
			log.Printf("image: fetched %.12s (%d bytes) for %s from %s", row.SHA256, row.Size, room.RoomName, shortEndpoint(addr))
			d.notify()
			return true
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no peer to ask")
	}
	if row.Attempts%5 == 0 {
		log.Printf("image: could not fetch %.12s for %s (attempt %d): %v", row.SHA256, room.RoomName, row.Attempts+1, lastErr)
	}
	d.mu.Lock()
	_ = store.ImageFailed(row.SHA256, time.Now().Add(imageBackoff(row.Attempts+1)))
	d.mu.Unlock()
	return false
}

// fetchImageFrom asks one peer for an image, continues a partial file if an earlier attempt
// left one, and keeps the result only if its hash and its type are the ones the signed event
// named.
func (d *Daemon) fetchImageFrom(addr string, room Room, store *Store, row imageRow) error {
	final, err := store.imagePath(row.SHA256)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(store.imageDir, 0o700); err != nil {
		return err
	}
	part := final + ".part"
	var offset int64
	if info, err := os.Stat(part); err == nil {
		if info.Size() > row.Size {
			_ = os.Remove(part)
		} else {
			offset = info.Size()
		}
	}
	remaining := row.Size - offset
	if remaining > 0 {
		if err := d.requestBlob(addr, room, row, part, offset, remaining); err != nil {
			return err
		}
	}
	if err := finishImage(row, part, final); err != nil {
		return err
	}
	// The write is made under the lock every writer to a room's database takes.
	d.mu.Lock()
	defer d.mu.Unlock()
	return store.CompleteImage(row.SHA256)
}

func (d *Daemon) requestBlob(addr string, room Room, row imageRow, part string, offset, remaining int64) error {
	conf, err := d.clientConfig("")
	if err != nil {
		return err
	}
	client, err := clientFor(addr, imageTimeout(remaining), conf)
	if err != nil {
		return err
	}
	cred, err := signRequestAs(d.id, room.RoomID, AdvertisedEndpoint())
	if err != nil {
		return err
	}
	body, _ := json.Marshal(blobRequest{Protocol: wireVersion, RoomID: room.RoomID, SHA256: row.SHA256, Offset: offset,
		PeerID: cred.PeerID, Timestamp: cred.Timestamp, Nonce: cred.Nonce, Signature: cred.Signature,
		Endpoint: cred.Endpoint, Signer: cred.Signer, Chain: cred.Chain})
	resp, err := client.Post(peerURL("/blob"), "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return errors.New("the peer does not hold the image")
	case http.StatusUnauthorized:
		return errors.New("the peer does not admit this machine to the room")
	default:
		return fmt.Errorf("the peer answered %s", resp.Status)
	}
	if resp.ContentLength != remaining {
		return fmt.Errorf("the peer offers %d bytes where %d remain", resp.ContentLength, remaining)
	}
	f, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, io.LimitReader(resp.Body, remaining))
	if err != nil {
		return err // the bytes that arrived stay, and the next attempt continues from them
	}
	return nil
}

// finishImage checks a complete partial file against what the signed event named, and moves it
// into place only if it is that image. Anything else is deleted, since continuing from bytes
// that are wrong would fail the same way each time.
func finishImage(row imageRow, part, final string) error {
	data, err := os.ReadFile(part)
	if err != nil {
		return err
	}
	if int64(len(data)) != row.Size {
		return fmt.Errorf("%d bytes where %d were named", len(data), row.Size)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != row.SHA256 {
		_ = os.Remove(part)
		return errors.New("the bytes are not the image the signed event names")
	}
	if typ, ok := sniffImageType(data); !ok || typ != row.Type {
		_ = os.Remove(part)
		return fmt.Errorf("the bytes are %q, not the %s the event names", typ, row.Type)
	}
	if err := os.Chmod(part, 0o600); err != nil {
		return err
	}
	return os.Rename(part, final)
}
