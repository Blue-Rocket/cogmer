package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// imageRow is where one pasted image stands in a room on this machine.
type imageRow struct {
	SHA256   string
	Type     string
	Size     int64
	State    string // imageHave or imageWanted
	Attempts int
	NextTry  string
}

const (
	imageHave   = "have"
	imageWanted = "wanted"
)

// imagePath is where an image's file is. The name is the hash and nothing a peer wrote, so no
// name can reach outside the directory.
func (s *Store) imagePath(sha string) (string, error) {
	if !validSHA(sha) {
		return "", fmt.Errorf("%q is not an image hash", sha)
	}
	return filepath.Join(s.imageDir, sha), nil
}

// SaveImage stores the bytes of an image this machine holds, and records that it has them.
// The file is written whole and then moved into place, so a crash leaves no half of an image
// under its hash.
func (s *Store) SaveImage(sha, mediaType string, data []byte) error {
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != sha {
		return errors.New("the bytes are not the image the hash names")
	}
	path, err := s.imagePath(sha)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.imageDir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.imageDir, "save-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return err
	}
	return s.markImage(sha, mediaType, int64(len(data)))
}

func (s *Store) markImage(sha, mediaType string, size int64) error {
	_, err := s.db.Exec(`
		INSERT INTO images (sha256, media_type, size, state) VALUES (?, ?, ?, ?)
		ON CONFLICT(sha256) DO UPDATE SET state = excluded.state, attempts = 0, next_try = ''`,
		sha, mediaType, size, imageHave)
	return err
}

// CompleteImage records that a fetched file is stored. The caller has checked its hash.
func (s *Store) CompleteImage(sha string) error {
	_, err := s.db.Exec(`UPDATE images SET state = ?, attempts = 0, next_try = '' WHERE sha256 = ?`, imageHave, sha)
	return err
}

// WantImage records that an event names an image whose bytes are not here, and leaves an image
// already held as it is.
func (s *Store) WantImage(sha, mediaType string, size int64) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO images (sha256, media_type, size, state) VALUES (?, ?, ?, ?)`,
		sha, mediaType, size, imageWanted)
	return err
}

// ImageRows is every image the room knows of, by hash.
func (s *Store) ImageRows() (map[string]imageRow, error) {
	rows, err := s.db.Query(`SELECT sha256, media_type, size, state, attempts, next_try FROM images`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]imageRow{}
	for rows.Next() {
		var r imageRow
		if err := rows.Scan(&r.SHA256, &r.Type, &r.Size, &r.State, &r.Attempts, &r.NextTry); err != nil {
			return nil, err
		}
		out[r.SHA256] = r
	}
	return out, rows.Err()
}

// ImageRow is one image's row, and whether the room knows of it.
func (s *Store) ImageRow(sha string) (imageRow, bool, error) {
	var r imageRow
	err := s.db.QueryRow(`SELECT sha256, media_type, size, state, attempts, next_try FROM images WHERE sha256 = ?`, sha).
		Scan(&r.SHA256, &r.Type, &r.Size, &r.State, &r.Attempts, &r.NextTry)
	if errors.Is(err, sql.ErrNoRows) {
		return imageRow{}, false, nil
	}
	return r, err == nil, err
}

// WantedImages is the images still to fetch whose next attempt is due.
func (s *Store) WantedImages(now time.Time) ([]imageRow, error) {
	rows, err := s.db.Query(`SELECT sha256, media_type, size, state, attempts, next_try FROM images
		WHERE state = ? AND (next_try = '' OR next_try <= ?) ORDER BY rowid`,
		imageWanted, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []imageRow
	for rows.Next() {
		var r imageRow
		if err := rows.Scan(&r.SHA256, &r.Type, &r.Size, &r.State, &r.Attempts, &r.NextTry); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ImageFailed records a failed attempt and when the next is due.
func (s *Store) ImageFailed(sha string, next time.Time) error {
	_, err := s.db.Exec(`UPDATE images SET attempts = attempts + 1, next_try = ? WHERE sha256 = ?`,
		next.UTC().Format(time.RFC3339Nano), sha)
	return err
}

// AttachedImageNumbers is the numbers of the images a session has already sent with its prompts,
// from the events themselves, so that nothing is kept that a lost database would also lose.
func (s *Store) AttachedImageNumbers(sessionID string) (map[int]bool, error) {
	rows, err := s.db.Query(`SELECT metadata FROM events
		WHERE origin_session_id = ? AND event_type = ? AND metadata LIKE '%"images"%'`, sessionID, EventUserPrompt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]bool{}
	for rows.Next() {
		var meta sql.NullString
		if err := rows.Scan(&meta); err != nil {
			return nil, err
		}
		for _, ref := range imageRefs(json.RawMessage(meta.String)) {
			out[ref.N] = true
		}
	}
	return out, rows.Err()
}
