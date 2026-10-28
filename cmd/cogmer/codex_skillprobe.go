package main

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// runCodexSkillProbe installs a plugin whose manifest names a skills folder of its own into
// a Codex home made for the probe, asks Codex's app server which skills it offers, and
// returns their names (B36). It takes no model turn, needs no login, and touches nothing of
// the user's Codex configuration, since the home is a directory of its own.
func runCodexSkillProbe(dir string) []string {
	root := filepath.Join(dir, "codexskillprobe")
	src := filepath.Join(root, "src")
	plug := filepath.Join(src, "plugin")
	files := map[string]string{
		filepath.Join(plug, ".codex-plugin", "plugin.json"):          `{"name":"cogmerskillprobe","version":"0.0.1","description":"a probe","skills":"./codex-skills/"}`,
		filepath.Join(plug, "codex-skills", "seen", "SKILL.md"):      "---\nname: seen\ndescription: probe skill\n---\nprobe\n",
		filepath.Join(src, ".agents", "plugins", "marketplace.json"): `{"name":"cogmerskillprobe","plugins":[{"name":"cogmerskillprobe","source":{"source":"local","path":"./plugin"},"policy":{"installation":"AVAILABLE","authentication":"ON_INSTALL"},"category":"Productivity"}]}`,
	}
	for path, body := range files {
		if os.MkdirAll(filepath.Dir(path), 0o700) != nil || os.WriteFile(path, []byte(body), 0o600) != nil {
			return nil
		}
	}
	return codexSkillsOfPlugin(root, src, "cogmerskillprobe", "cogmerskillprobe")
}

// codexSkillsOfPlugin adds the marketplace in src to a Codex home made inside root, installs
// plugin from it, and returns the skills Codex then offers a session in root. The Codex home is
// a directory of its own, so nothing of the user's Codex configuration is read or changed.
func codexSkillsOfPlugin(root, src, plugin, marketplace string) []string {
	home := filepath.Join(root, "codexhome")
	if os.MkdirAll(home, 0o700) != nil {
		return nil
	}
	env := append(os.Environ(), "CODEX_HOME="+home)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for _, args := range [][]string{
		{"plugin", "marketplace", "add", src},
		{"plugin", "add", plugin + "@" + marketplace},
	} {
		cmd := exec.CommandContext(ctx, codexExecutable(), args...)
		cmd.Dir, cmd.Env = root, env
		if cmd.Run() != nil {
			return nil
		}
	}
	return codexSkillNames(ctx, root, env)
}

// codexSkillNames asks a Codex app server, over its standard input and output, for the
// skills it offers to a session in cwd, and returns their names.
func codexSkillNames(ctx context.Context, cwd string, env []string) []string {
	cmd := exec.CommandContext(ctx, codexExecutable(), "app-server")
	cmd.Dir, cmd.Env = cwd, env
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil
	}
	out, err := cmd.StdoutPipe()
	if err != nil || cmd.Start() != nil {
		return nil
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	send := func(m map[string]any) {
		b, _ := json.Marshal(m)
		_, _ = in.Write(append(b, '\n'))
	}
	lines := bufio.NewScanner(out)
	lines.Buffer(make([]byte, 0, 1<<20), 16<<20)
	call := func(id int, method string, params any) json.RawMessage {
		send(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
		for lines.Scan() {
			var m struct {
				ID     *int            `json:"id"`
				Result json.RawMessage `json:"result"`
			}
			if json.Unmarshal(lines.Bytes(), &m) == nil && m.ID != nil && *m.ID == id {
				return m.Result
			}
		}
		return nil
	}
	if call(1, "initialize", map[string]any{"clientInfo": map[string]any{"name": "cogmer-doctor", "title": nil, "version": version}}) == nil {
		return nil
	}
	send(map[string]any{"jsonrpc": "2.0", "method": "initialized"})
	return skillNames(call(2, "skills/list", map[string]any{"cwds": []string{cwd}, "forceReload": true}))
}

// skillNames reads the names from the result of a skills/list call.
func skillNames(result json.RawMessage) []string {
	var r struct {
		Data []struct {
			Skills []struct {
				Name string `json:"name"`
			} `json:"skills"`
		} `json:"data"`
	}
	if json.Unmarshal(result, &r) != nil {
		return nil
	}
	var names []string
	for _, d := range r.Data {
		for _, s := range d.Skills {
			names = append(names, strings.TrimSpace(s.Name))
		}
	}
	return names
}
