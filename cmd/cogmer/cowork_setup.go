package main

import (
	"archive/zip"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// Setting a machine up for Claude Cowork (D-205). Two things have to agree: the entry in
// Claude Desktop's configuration that starts the bridge, and the plugin installed in Cowork
// whose hooks call it, which name the entry. One command writes both from one name, so a
// name that Anthropic's device bridge builds from the entry's cannot be mistyped in one of
// them.

// coworkServerName is the name of the bridge's entry in Claude Desktop's configuration. A
// hook reaches its tools as <entry>__<tool> on the server Cowork calls remote-devices.
const coworkServerName = "cogmer"

// coworkDeviceServer is the server through which Cowork presents a Desktop entry's tools to a
// session (observed, and not documented).
const coworkDeviceServer = "remote-devices"

//go:embed cowork/commands/*.md
var coworkCommandFiles embed.FS

// desktopConfigPath is where Claude Desktop reads its configuration. COGMER_DESKTOP_CONFIG
// names another file, for a test.
func desktopConfigPath() (string, error) {
	if p := os.Getenv("COGMER_DESKTOP_CONFIG"); p != "" {
		return p, nil
	}
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), nil
	case "windows":
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "Claude", "claude_desktop_config.json"), nil
		}
	}
	return "", fmt.Errorf("cogmer does not know where Claude Desktop keeps its configuration on this system")
}

// addDesktopServer adds the bridge's entry to Claude Desktop's configuration, and leaves
// everything else in the file as it is. It writes nothing it cannot read back, keeps a copy of
// the file it replaces, and says what it did. A file that is not a JSON object, or whose
// mcpServers is not one, is refused and left alone.
func addDesktopServer(path, name, command string, args []string) (result string, backup string, err error) {
	raw, readErr := os.ReadFile(path)
	cfg := map[string]json.RawMessage{}
	switch {
	case readErr == nil && len(bytes.TrimSpace(raw)) > 0:
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return "", "", fmt.Errorf("%s is not a JSON object (%v), so nothing was changed", path, err)
		}
	case readErr != nil && !os.IsNotExist(readErr):
		return "", "", readErr
	}
	servers := map[string]json.RawMessage{}
	if cur, ok := cfg["mcpServers"]; ok {
		if err := json.Unmarshal(cur, &servers); err != nil {
			return "", "", fmt.Errorf("the mcpServers in %s is not an object (%v), so nothing was changed", path, err)
		}
	}
	entry, _ := json.Marshal(map[string]any{"command": command, "args": args})
	if cur, ok := servers[name]; ok && jsonEqual(cur, entry) {
		return "the entry for " + name + " was already there", "", nil
	}
	existed := false
	if _, ok := servers[name]; ok {
		existed = true
	}
	servers[name] = entry
	cfg["mcpServers"], _ = json.Marshal(servers)
	out, err := marshalSorted(cfg)
	if err != nil {
		return "", "", err
	}
	if readErr == nil {
		backup = fmt.Sprintf("%s.cogmer-backup-%s", path, time.Now().UTC().Format("20060102T150405Z"))
		if err := os.WriteFile(backup, raw, 0o600); err != nil {
			return "", "", fmt.Errorf("could not keep a copy of %s, so nothing was changed: %v", path, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", "", err
	}
	tmp := path + ".cogmer-tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return "", "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", "", err
	}
	if existed {
		return "the entry for " + name + " was updated", backup, nil
	}
	return "the entry for " + name + " was added", backup, nil
}

func jsonEqual(a, b []byte) bool {
	var x, y any
	return json.Unmarshal(a, &x) == nil && json.Unmarshal(b, &y) == nil && fmt.Sprint(x) == fmt.Sprint(y)
}

// marshalSorted writes a configuration with its keys in order and indented, which is how
// a person reads it, and which json does for a map.
func marshalSorted(cfg map[string]json.RawMessage) ([]byte, error) {
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// coworkHookFile is the plugin's hooks, which name the bridge's entry. Only a prompt and a stop
// are hooked: a call at launch never reaches a server (B32).
func coworkHookFile(entry string) ([]byte, error) {
	hook := func(tool string, input map[string]any) map[string]any {
		return map[string]any{"hooks": []any{map[string]any{
			"type": "mcp_tool", "server": coworkDeviceServer, "tool": entry + "__" + tool, "input": input,
			// longer than the daemon takes to run a command, which a user typed and waits for
			"timeout": 30,
		}}}
	}
	return json.MarshalIndent(map[string]any{
		"description": "Captures this session's turns into the room it belongs to and injects colleagues' turns, through cogmer on this computer.",
		"hooks": map[string]any{
			"UserPromptSubmit": []any{hook("prompt", map[string]any{
				"session_id": "${session_id}", "prompt": "${prompt}", "prompt_id": "${prompt_id}", "cwd": "${cwd}"})},
			"Stop": []any{hook("stop", map[string]any{
				"session_id": "${session_id}", "prompt_id": "${prompt_id}",
				"last_assistant_message": "${last_assistant_message}", "transcript_path": "${transcript_path}", "cwd": "${cwd}"})},
		},
	}, "", "  ")
}

// coworkPluginFiles is every file of the plugin installed in Cowork, by path.
func coworkPluginFiles(entry, ver string) (map[string][]byte, error) {
	files := map[string][]byte{}
	manifest, _ := json.MarshalIndent(map[string]any{
		"name":        "cogmer",
		"version":     ver,
		"description": "Share a conversation between Claude Cowork, Claude Code and Codex sessions. Turns replicate directly between your computers, peer to peer, with no server holding the conversation.",
		"author":      map[string]any{"name": "David Foote"},
	}, "", "  ")
	files[".claude-plugin/plugin.json"] = manifest
	hooks, err := coworkHookFile(entry)
	if err != nil {
		return nil, err
	}
	files["hooks/hooks.json"] = hooks
	entries, err := coworkCommandFiles.ReadDir("cowork/commands")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		b, err := coworkCommandFiles.ReadFile("cowork/commands/" + e.Name())
		if err != nil {
			return nil, err
		}
		files["commands/"+e.Name()] = b
	}
	return files, nil
}

// writeZip writes files as a zip with the plugin at its root, in a fixed order, and with
// a fixed time so the same plugin is the same bytes.
func writeZip(path string, files map[string][]byte) error {
	var names []string
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, n := range names {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: n, Method: zip.Deflate, Modified: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
		if err != nil {
			return err
		}
		if _, err := w.Write(files[n]); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// coworkCommandFile is a command file of the Claude Code plugin as a Cowork session reads it.
// Its output is not printed by a shell line but arrives as context from the hook, so the part
// of the file that holds the shell lines is replaced by a pointer to it, and the instructions
// for relaying what it says are the plugin's own.
func coworkCommandFile(name, src string) string {
	front, body, ok := splitFrontmatter(src)
	if !ok {
		return src
	}
	var kept []string
	for _, l := range strings.Split(front, "\n") {
		if strings.HasPrefix(l, "allowed-tools:") {
			continue
		}
		kept = append(kept, l)
	}
	task := body
	if i := strings.Index(body, "## Your task"); i >= 0 {
		task = body[i:]
	}
	var b strings.Builder
	b.WriteString("---\n" + strings.Join(kept, "\n") + "\n---\n\n")
	fmt.Fprintf(&b, "## What happened\n\nThe output of this command is in your context, under the heading \"Output of /cogmer:%s\". Nothing else produced it.\n\n", name)
	b.WriteString("If that heading is not in your context, cogmer did not answer: this session captured and shared nothing, and the setup that connects Cowork to cogmer may need doing again. Say so plainly and stop.\n\n")
	b.WriteString(task)
	return b.String()
}

func splitFrontmatter(s string) (front, body string, ok bool) {
	if !strings.HasPrefix(s, "---\n") {
		return "", "", false
	}
	rest := s[4:]
	i := strings.Index(rest, "\n---\n")
	if i < 0 {
		return "", "", false
	}
	return rest[:i], strings.TrimLeft(rest[i+5:], "\n"), true
}

// runCoworkSetup is `cogmer cowork-setup`. It adds the bridge to Claude Desktop's
// configuration and writes the plugin to install in Cowork, both from one name, and says
// what to do next.
func runCoworkSetup(args []string) {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("cowork-setup: %v", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	config, err := desktopConfigPath()
	if err != nil {
		log.Fatalf("cowork-setup: %v", err)
	}
	did, backup, err := addDesktopServer(config, coworkServerName, exe, []string{"bridge"})
	if err != nil {
		log.Fatalf("cowork-setup: %v", err)
	}
	files, err := coworkPluginFiles(coworkServerName, version)
	if err != nil {
		log.Fatalf("cowork-setup: %v", err)
	}
	zipPath := filepath.Join(homeDir(), "cowork", "cogmer-cowork.zip")
	if err := writeZip(zipPath, files); err != nil {
		log.Fatalf("cowork-setup: %v", err)
	}

	fmt.Println("Cowork is set up on this machine in two parts.")
	fmt.Println()
	fmt.Printf("1. Claude Desktop starts cogmer's bridge: %s in\n   %s\n", did, config)
	if backup != "" {
		fmt.Printf("   A copy of the file as it was is at\n   %s\n", backup)
	}
	fmt.Println()
	fmt.Printf("2. A plugin for Cowork was written to\n   %s\n", zipPath)
	fmt.Println()
	fmt.Println("What to do next:")
	fmt.Println("  1. Quit Claude Desktop completely and open it again.")
	fmt.Println("  2. In Cowork, open Customize, then Plugins, and upload that file.")
	fmt.Println("  3. Start a new Cowork session and type /cogmer:self-status. If it answers, cogmer works there.")
	fmt.Println()
	fmt.Println("Run this command again if you rename this computer or move cogmer, and upload the new file.")
}
