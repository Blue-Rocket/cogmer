package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Claude Code loads plugin/commands/*.md identically to skills/<name>/SKILL.md,
// so a command is a model-invocable skill unless its frontmatter says otherwise
// (D-119). Three things follow, and the third is why this is a test rather than a
// note: the model can run a command nobody asked for, including ones that bind a
// session to a room irreversibly (D-016) or withdraw a colleague's admission; the
// descriptions occupy every session's context whether or not anything invokes
// them, which §3.1 counts as harm by itself; and neither failure is visible from
// inside this repository, because the loading happens in somebody else's program.
//
// The defect this guards is not the files that exist now. It is the command added
// later by somebody who does not know the frontmatter is load-bearing, which
// nothing else would catch.
func TestNoCommandIsModelInvocable(t *testing.T) {
	dir := "../../plugin/commands"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("plugin commands not readable: %v", err)
	}

	var seen int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		seen++
		src, rerr := os.ReadFile(filepath.Join(dir, e.Name()))
		if rerr != nil {
			t.Fatalf("%s: %v", e.Name(), rerr)
		}
		front, ok := frontmatter(string(src))
		if !ok {
			t.Errorf("%s has no frontmatter, so it cannot carry disable-model-invocation", e.Name())
			continue
		}
		if !strings.Contains(front, "disable-model-invocation: true") {
			t.Errorf("%s is model-invocable: Claude can run it without anybody asking, and its "+
				"description costs every session context budget. Add "+
				"disable-model-invocation: true to its frontmatter", e.Name())
		}
	}
	if seen == 0 {
		t.Fatal("no command files found; this test would pass no matter what the frontmatter said")
	}
}

// frontmatter returns the text between the opening and closing --- fences.
func frontmatter(src string) (string, bool) {
	if !strings.HasPrefix(src, "---\n") {
		return "", false
	}
	rest := src[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}
