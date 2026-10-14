package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The Cowork plugin's command files are the Claude Code plugin's with the shell lines
// replaced, and are kept as files in the binary. A change to one of the plugin's
// instructions that was not carried across would tell a Cowork model something else.
// COGMER_UPDATE_COWORK_COMMANDS=1 writes them.
func TestTheCoworkCommandFilesAreThePluginsWithTheirOutputPointedAt(t *testing.T) {
	files, _ := filepath.Glob("../../plugin/commands/*.md")
	if len(files) == 0 {
		t.Skip("plugin commands not readable")
	}
	update := os.Getenv("COGMER_UPDATE_COWORK_COMMANDS") == "1"
	want := map[string]bool{}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".md")
		if _, no := typedCommandsUnavailable[name]; no && name != "self-stop" && name != "self-restart" {
			continue // a command that sets Cowork up is not one a Cowork session can run
		}
		src, _ := os.ReadFile(f)
		expected := coworkCommandFile(name, string(src))
		want[name+".md"] = true
		path := filepath.Join("cowork", "commands", name+".md")
		if update {
			if err := os.WriteFile(path, []byte(expected), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != expected {
			t.Errorf("%s is not the plugin's command with its output pointed at; run with COGMER_UPDATE_COWORK_COMMANDS=1", path)
		}
	}
	entries, _ := os.ReadDir(filepath.Join("cowork", "commands"))
	for _, e := range entries {
		if !want[e.Name()] {
			if update {
				_ = os.Remove(filepath.Join("cowork", "commands", e.Name()))
				continue
			}
			t.Errorf("cowork/commands/%s has no command in the plugin", e.Name())
		}
	}
}

// Every command a Cowork session can type is out of the model's reach, as the plugin's are
// (D-119), and none of them runs a shell line, which a Cowork session has no daemon behind.
func TestNoCoworkCommandIsModelInvocableOrRunsAShell(t *testing.T) {
	entries, err := coworkCommandFiles.ReadDir("cowork/commands")
	if err != nil || len(entries) < 10 {
		t.Fatalf("the Cowork commands are %d: %v", len(entries), err)
	}
	for _, e := range entries {
		b, _ := coworkCommandFiles.ReadFile("cowork/commands/" + e.Name())
		front, body, ok := splitFrontmatter(string(b))
		if !ok || !strings.Contains(front, "disable-model-invocation: true") {
			t.Errorf("%s is model-invocable or has no frontmatter", e.Name())
		}
		if strings.Contains(front, "allowed-tools") || strings.Contains(body, "!`") || strings.Contains(body, "cli.sh") {
			t.Errorf("%s still names a shell line the sandbox cannot run", e.Name())
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		if !strings.Contains(body, `Output of /cogmer:`+name) || !strings.Contains(body, "## Your task") {
			t.Errorf("%s does not point at its output and say what to do with it", e.Name())
		}
	}
}

func TestTheCoworkHooksNameTheEntryAndOnlyAPromptAndAStop(t *testing.T) {
	raw, err := coworkHookFile("cogmer")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Type    string         `json:"type"`
				Server  string         `json:"server"`
				Tool    string         `json:"tool"`
				Input   map[string]any `json:"input"`
				Timeout int            `json:"timeout"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	var events []string
	for ev := range f.Hooks {
		events = append(events, ev)
	}
	sort.Strings(events)
	if strings.Join(events, ",") != "Stop,UserPromptSubmit" {
		t.Errorf("the plugin hooks %v; a call at launch never reaches the bridge", events)
	}
	for ev, want := range map[string]string{"UserPromptSubmit": "cogmer__prompt", "Stop": "cogmer__stop"} {
		h := f.Hooks[ev][0].Hooks[0]
		if h.Type != "mcp_tool" || h.Server != "remote-devices" || h.Tool != want {
			t.Errorf("%s calls %+v, want the tool %s on remote-devices", ev, h, want)
		}
		if h.Timeout < int(typedCommandTimeout.Seconds()) {
			t.Errorf("%s waits %ds, less than a command may take", ev, h.Timeout)
		}
	}
	// the fields the bridge takes, written as the placeholders a hook fills in
	prompt := f.Hooks["UserPromptSubmit"][0].Hooks[0].Input
	stop := f.Hooks["Stop"][0].Hooks[0].Input
	for _, k := range []string{"session_id", "prompt", "prompt_id"} {
		if prompt[k] != "${"+k+"}" {
			t.Errorf("the prompt hook passes %s as %v", k, prompt[k])
		}
	}
	for _, k := range []string{"session_id", "prompt_id", "last_assistant_message"} {
		if stop[k] != "${"+k+"}" {
			t.Errorf("the stop hook passes %s as %v", k, stop[k])
		}
	}
	// every field a hook passes is one the bridge relays, so none is dropped without a sign
	for tool, input := range map[string]map[string]any{"prompt": prompt, "stop": stop} {
		allowed := map[string]bool{}
		for _, f := range bridgeEndpoints[tool].fields {
			allowed[f] = true
		}
		for k := range input {
			if !allowed[k] {
				t.Errorf("the %s hook passes %s, which the bridge does not relay", tool, k)
			}
		}
	}
	// the tool names are the ones the bridge offers
	var offered []string
	for _, tool := range bridgeTools {
		offered = append(offered, tool["name"].(string))
	}
	for _, want := range []string{"prompt", "stop"} {
		found := false
		for _, o := range offered {
			found = found || o == want
		}
		if !found {
			t.Errorf("the plugin calls a tool %q the bridge does not offer", want)
		}
	}
}

func TestACoworkEntryIsAddedToTheDesktopConfigurationAndNothingElseChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claude_desktop_config.json")
	original := `{
  "globalShortcut": "Ctrl+Space",
  "mcpServers": {
    "clickup": {"command": "npx", "args": ["-y", "clickup-mcp"], "env": {"TOKEN": "secret"}}
  }
}
`
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	what, backup, err := addDesktopServer(path, "cogmer", "/home/u/.cogmer/bin/cogmer", []string{"bridge"})
	if err != nil || !strings.Contains(what, "was added") {
		t.Fatalf("%q %v", what, err)
	}
	if b, _ := os.ReadFile(backup); string(b) != original {
		t.Errorf("the copy of the old file is %q", b)
	}
	var cfg struct {
		Shortcut string                    `json:"globalShortcut"`
		Servers  map[string]map[string]any `json:"mcpServers"`
	}
	raw, _ := os.ReadFile(path)
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("the file is not JSON: %v\n%s", err, raw)
	}
	if cfg.Shortcut != "Ctrl+Space" || cfg.Servers["clickup"]["command"] != "npx" ||
		cfg.Servers["clickup"]["env"].(map[string]any)["TOKEN"] != "secret" {
		t.Errorf("something else in the file changed: %s", raw)
	}
	if cfg.Servers["cogmer"]["command"] != "/home/u/.cogmer/bin/cogmer" || cfg.Servers["cogmer"]["args"].([]any)[0] != "bridge" {
		t.Errorf("the entry is %v", cfg.Servers["cogmer"])
	}
	// the same call again changes nothing and keeps no second copy
	what, backup2, err := addDesktopServer(path, "cogmer", "/home/u/.cogmer/bin/cogmer", []string{"bridge"})
	if err != nil || !strings.Contains(what, "already there") || backup2 != "" {
		t.Errorf("a second run: %q %q %v", what, backup2, err)
	}
	// a different path updates the entry
	what, _, err = addDesktopServer(path, "cogmer", "/elsewhere/cogmer", []string{"bridge"})
	if err != nil || !strings.Contains(what, "was updated") {
		t.Errorf("a changed path: %q %v", what, err)
	}
}

func TestADesktopConfigurationThatIsNotUsableIsLeftAlone(t *testing.T) {
	for name, content := range map[string]string{
		"not json":                       "{ this is not json",
		"an array":                       `[1,2]`,
		"servers that are not an object": `{"mcpServers": [1]}`,
	} {
		dir := t.TempDir()
		path := filepath.Join(dir, "claude_desktop_config.json")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := addDesktopServer(path, "cogmer", "/x/cogmer", []string{"bridge"}); err == nil {
			t.Errorf("%s: the file was accepted", name)
		}
		if b, _ := os.ReadFile(path); string(b) != content {
			t.Errorf("%s: the file was changed to %q", name, b)
		}
		entries, _ := os.ReadDir(dir)
		if len(entries) != 1 {
			t.Errorf("%s: left %d files behind", name, len(entries)-1)
		}
	}
	// no file at all is a configuration with nothing in it
	path := filepath.Join(t.TempDir(), "new", "claude_desktop_config.json")
	if _, backup, err := addDesktopServer(path, "cogmer", "/x/cogmer", []string{"bridge"}); err != nil || backup != "" {
		t.Errorf("a missing file: %q %v", backup, err)
	}
	if b, _ := os.ReadFile(path); !strings.Contains(string(b), `"cogmer"`) {
		t.Errorf("the new file is %q", b)
	}
}

func TestTheCoworkPluginIsOneZipWithThePluginAtItsRoot(t *testing.T) {
	files, err := coworkPluginFiles("cogmer", "9.9.9")
	if err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "cogmer-cowork.zip")
	if err := writeZip(zipPath, files); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(zipPath)
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	got := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		got[f.Name] = string(b)
	}
	for _, need := range []string{".claude-plugin/plugin.json", "hooks/hooks.json", "commands/room-status.md", "commands/self-restart.md"} {
		if _, ok := got[need]; !ok {
			t.Errorf("the zip lacks %s", need)
		}
	}
	var manifest struct{ Name, Version string }
	_ = json.Unmarshal([]byte(got[".claude-plugin/plugin.json"]), &manifest)
	if manifest.Name != "cogmer" || manifest.Version != "9.9.9" {
		t.Errorf("manifest %+v: the commands are typed with the plugin's name in front, so it must be cogmer", manifest)
	}
	if strings.Contains(got["hooks/hooks.json"], "cogmer-probe") {
		t.Error("the plugin names the probe")
	}
	// the same plugin is the same bytes
	if err := writeZip(zipPath, files); err != nil {
		t.Fatal(err)
	}
	if second, _ := os.ReadFile(zipPath); !bytes.Equal(first, second) {
		t.Error("writing the same plugin twice gave different bytes")
	}
}
