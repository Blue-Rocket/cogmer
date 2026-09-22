package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// pluginManifestName is the namespace Claude Code prefixes to every one of our
// slash commands (D-118). It is read rather than duplicated as a Go constant: a
// constant would be used by nothing but this test, and the manifest is what
// Claude Code actually reads.
func pluginManifestName(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../plugin/.claude-plugin/plugin.json")
	if err != nil {
		t.Skipf("plugin manifest not readable: %v", err)
	}
	var m struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse plugin manifest: %v", err)
	}
	if m.Name == "" {
		t.Fatal("plugin manifest has no name; there is then no command namespace at all")
	}
	return m.Name
}

// D-085: a line telling somebody to run a command that does not exist is worse
// than no line, because it fails for exactly the people least able to diagnose
// it. Since D-118 an unprefixed /room-create is that line, and so is one carrying
// a namespace that is not the manifest's -- renaming the plugin without renaming
// these would break every instruction the binary prints, silently.
func TestEveryCommandReferenceCarriesTheManifestNamespace(t *testing.T) {
	ns := pluginManifestName(t)

	// A command reference, not a file path like commands/room-join.md.
	bare := regexp.MustCompile(`[^\w/-]/(?:room|peer|self)-[a-z]+`)
	namespaced := regexp.MustCompile(`/([\w-]+):(?:room|peer|self)-[a-z]+`)

	roots := map[string][]string{
		".":            {".go", ".html"},
		"../../plugin": {".md", ".sh", ".json"},
	}
	for root, exts := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			var want bool
			for _, e := range exts {
				if strings.HasSuffix(path, e) {
					want = true
				}
			}
			if !want || strings.HasSuffix(path, "commandname_test.go") {
				return nil
			}
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			for _, hit := range bare.FindAllString(string(src), -1) {
				t.Errorf("%s names %q with no namespace; Claude Code offers no unprefixed "+
					"form, so nobody can run it", path, strings.TrimSpace(hit))
			}
			for _, m := range namespaced.FindAllStringSubmatch(string(src), -1) {
				if m[1] != ns {
					t.Errorf("%s names %q but the plugin manifest namespace is %q",
						path, m[0], ns)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
}
