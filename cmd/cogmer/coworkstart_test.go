package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// runStartIn runs the session-start hook with the installer replaced by a stand-in that
// leaves a mark, in an environment of the given extra variables, and returns what it
// printed and the state folder, where the stand-in leaves its mark.
func runStartIn(t *testing.T, extra ...string) (string, string) {
	t.Helper()
	root, home := cliHarness(t)
	src, err := os.ReadFile("../../plugin/hooks-handlers/session-start.sh")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "hooks-handlers", "session-start.sh")
	if err := os.WriteFile(path, src, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", path)
	cmd.Env = append([]string{"PATH=/usr/bin:/bin", "HOME=" + home, "COGMER_HOME=" + home,
		"CLAUDE_PLUGIN_ROOT=" + root, "COGMER_BIN=/missing", "COGMER_ADDR=127.0.0.1:1"}, extra...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out), home
}

// syncedWith makes a folder of synced plugins holding a cloud.value in a plugin folder of
// each given name, under one account folder.
func syncedWith(t *testing.T, names ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, n := range names {
		dir := filepath.Join(root, "account", n)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "cloud.value"), []byte("cogmer-cloud-v1:x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// A Cowork cloud session with no cloud value fetches nothing, starts nothing and adds
// nothing to the session, since a daemon there would speak for nobody (D-286).
func TestACoworkCloudSessionWithNoCloudValueStartsNothing(t *testing.T) {
	for name, synced := range map[string]string{
		"no synced plugins":           filepath.Join(t.TempDir(), "absent"),
		"no cloud plugin":             syncedWith(t, "cogmer"),
		"a plugin of another name":    syncedWith(t, "cogmer-cloud-notdigits", "my-cogmer-cloud-0123abcd"),
		"a folder name with a suffix": syncedWith(t, "cogmer-cloud-0123abcd-copy"),
	} {
		out, home := runStartIn(t, "CLAUDE_CODE_ENTRYPOINT=remote_cowork", "CLAUDE_CODE_REMOTE=true",
			"COGMER_SYNCED_PLUGINS="+synced)
		// The hook returns before it would start the installer, so a short wait is enough.
		time.Sleep(300 * time.Millisecond)
		if _, err := os.Stat(filepath.Join(home, "started")); err == nil {
			t.Errorf("%s: the installer ran", name)
		}
		if strings.TrimSpace(out) != "" {
			t.Errorf("%s: the hook printed %q", name, out)
		}
	}
}

// Every other session starts as it would without the check: a Cowork cloud session that
// holds a value in either form, and a cloud session of Claude Code, whose entrypoint is
// not Cowork's.
func TestASessionThatMayHaveAUserStartsTheDaemon(t *testing.T) {
	for name, env := range map[string][]string{
		"a synced cloud value":        {"CLAUDE_CODE_ENTRYPOINT=remote_cowork", "COGMER_SYNCED_PLUGINS=" + syncedWith(t, "cogmer", "cogmer-cloud-0123abcd")},
		"a value uploaded again":      {"CLAUDE_CODE_ENTRYPOINT=remote_cowork", "COGMER_SYNCED_PLUGINS=" + syncedWith(t, "cogmer-cloud-0123abcd~g2")},
		"the variable":                {"CLAUDE_CODE_ENTRYPOINT=remote_cowork", "COGMER_CLOUD_KEY=cogmer-cloud-v1:x", "COGMER_SYNCED_PLUGINS=" + t.TempDir()},
		"a Claude Code cloud session": {"CLAUDE_CODE_REMOTE=true", "COGMER_SYNCED_PLUGINS=" + t.TempDir()},
		"a local session":             nil,
	} {
		out, home := runStartIn(t, env...)
		if !waitFor(t, filepath.Join(home, "started")) {
			t.Errorf("%s: the installer did not run", name)
		}
		if !strings.Contains(out, "standing policy") {
			t.Errorf("%s: the hook printed %q", name, out)
		}
	}
}
