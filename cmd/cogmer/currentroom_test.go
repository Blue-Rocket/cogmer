package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// There is no current room (D-080): a room belongs to the session that is in it. Help
// text, messages and comments that speak of one describe a mechanism that does not
// exist, and a reader acts on them. The words are allowed only to say there is none.
func TestNothingSpeaksOfACurrentRoom(t *testing.T) {
	stale := regexp.MustCompile(`(?i)current[- ]room|currentRoom|room is current`)
	var files []string
	for _, glob := range []string{"*.go", "*.html", "../../plugin/commands/*.md", "../../plugin/hooks-handlers/*.sh", "../../plugin/*.sh"} {
		m, _ := filepath.Glob(glob)
		files = append(files, m...)
	}
	if len(files) < 20 {
		t.Fatalf("found only %d files to read, so the check is not reading what it should", len(files))
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if stale.MatchString(line) && !strings.Contains(strings.ToLower(line), "no current room") {
				t.Errorf("%s:%d speaks of a current room, which D-080 removed: %s", f, i+1, strings.TrimSpace(line))
			}
		}
	}
}
