package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// The manifest's version is what Claude Code pins an installed plugin to: a person
// receives a new plugin only when that string moves. Everything else the plugin
// carries rides on that delivery -- the commands, the hooks, plugin/VERSION naming
// the binary to install, and checksums.txt, which is the only thing authorising a
// downloaded binary to run. So a manifest left at an older version does not look
// like a stale number; it looks like a release nobody receives, and the symptom is
// a person still on the old binary with no error anywhere.
//
// scripts/release.sh writes both files from its one argument. This test is what
// catches the other path: a version bumped by hand, in one file, without it.
func TestTheManifestVersionIsTheReleasedVersion(t *testing.T) {
	released, err := os.ReadFile("../../plugin/VERSION")
	if err != nil {
		t.Skipf("plugin/VERSION not readable: %v", err)
	}
	want := strings.TrimSpace(string(released))
	if want == "" {
		t.Fatal("plugin/VERSION is empty; nothing then names the binary to install")
	}

	data, err := os.ReadFile("../../plugin/.claude-plugin/plugin.json")
	if err != nil {
		t.Skipf("plugin manifest not readable: %v", err)
	}
	var m struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse plugin manifest: %v", err)
	}
	if m.Version != want {
		t.Errorf("plugin manifest says version %q, plugin/VERSION says %q; an installed "+
			"plugin is pinned to the manifest's string, so the release does not reach "+
			"anybody until they agree — rerun scripts/release.sh %s", m.Version, want, want)
	}
}
