package main

import (
	"encoding/json"
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

// Claude Code loads the skills/ directory of a plugin by itself, and the skill that gives
// Codex the commands is a second route to them for a Claude Code model, which D-119 keeps
// closed. The skill therefore sits in a directory Claude Code does not load, which only the
// Codex manifest names. Nothing else would notice a skill added to skills/ later, since the
// loading happens in somebody else's program.
func TestTheCodexSkillIsOutsideTheDirectoryClaudeCodeLoads(t *testing.T) {
	root := "../../plugin"
	if _, err := os.Stat(filepath.Join(root, "skills")); err == nil {
		t.Error("plugin/skills exists, and Claude Code loads every skill in it, including a route to the commands")
	}
	var stray []string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "SKILL.md" && !strings.Contains(filepath.ToSlash(p), "/codex-skills/") {
			stray = append(stray, p)
		}
		return nil
	})
	if len(stray) > 0 {
		t.Errorf("a SKILL.md outside plugin/codex-skills: %v", stray)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".codex-plugin", "plugin.json"))
	if err != nil {
		t.Skip("the Codex manifest is not readable")
	}
	var m struct {
		Skills string `json:"skills"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m.Skills != "./codex-skills/" {
		t.Errorf("the Codex manifest names its skills at %q, want ./codex-skills/", m.Skills)
	}
	if _, err := os.Stat(filepath.Join(root, "codex-skills", "cogmer", "SKILL.md")); err != nil {
		t.Errorf("the Codex manifest names a directory with no cogmer skill: %v", err)
	}
}
