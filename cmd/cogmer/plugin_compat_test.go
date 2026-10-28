//go:build !windows

package main

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// One program in ~/.cogmer/bin serves every host, and the installer and the daemon replace it
// only with a newer one (D-215, the installer and the daemon replace a program only with a
// newer version), so a host whose plugin has not updated runs that plugin's hook lines, handler
// scripts and command files against the current program. A release that renames a subcommand
// an older plugin calls, or changes something its scripts read, leaves that host's hooks
// failing open and its commands answering with a usage text, and nothing shows an error.
//
// These tests run the current program through the plugin of each of the last three releases,
// read from git by tag. They skip where the tags are not there, as in a shallow clone.

const compatReleases = 3

// releasePlugins extracts the plugin directory of each of the last n release tags into a
// directory of its own, and returns tag to directory, newest first.
func releasePlugins(t *testing.T, n int) (tags []string, dirs map[string]string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on this machine")
	}
	root := filepath.Join("..", "..")
	out, err := exec.Command("git", "-C", root, "tag", "--list", "v*", "--sort=-v:refname").Output()
	if err != nil {
		t.Skip("this is not a git checkout")
	}
	for _, tag := range strings.Fields(string(out)) {
		if len(tags) == n {
			break
		}
		tags = append(tags, tag)
	}
	if len(tags) == 0 {
		t.Skip("this checkout has no release tags")
	}
	dirs = map[string]string{}
	for _, tag := range tags {
		archive, err := exec.Command("git", "-C", root, "archive", "--format=tar", tag, "plugin").Output()
		if err != nil {
			t.Skipf("the plugin of %s is not in this checkout: %v", tag, err)
		}
		dir := t.TempDir()
		tr := tar.NewReader(bytes.NewReader(archive))
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, filepath.FromSlash(h.Name))
			if !strings.HasPrefix(path, dir+string(os.PathSeparator)) {
				t.Fatalf("%s holds a path outside itself: %s", tag, h.Name)
			}
			switch h.Typeflag {
			case tar.TypeDir:
				_ = os.MkdirAll(path, 0o755)
			case tar.TypeReg:
				_ = os.MkdirAll(filepath.Dir(path), 0o755)
				b, _ := io.ReadAll(tr)
				if err := os.WriteFile(path, b, os.FileMode(h.Mode)&0o777|0o600); err != nil {
					t.Fatal(err)
				}
			}
		}
		dirs[tag] = filepath.Join(dir, "plugin")
	}
	return tags, dirs
}

var (
	cliCall   = regexp.MustCompile(`cli\.sh"? ([a-z-]+)`)
	skillCall = regexp.MustCompile("(?m)^\\| `[a-z-]+` \\| `([a-z-]+)")
)

// calledSubcommands is every subcommand of the program that a plugin's command files and
// its Codex skill call.
func calledSubcommands(t *testing.T, plugin string) []string {
	t.Helper()
	set := map[string]bool{}
	files, _ := filepath.Glob(filepath.Join(plugin, "commands", "*.md"))
	for _, f := range files {
		b, _ := os.ReadFile(f)
		for _, m := range cliCall.FindAllStringSubmatch(string(b), -1) {
			set[m[1]] = true
		}
	}
	for _, skill := range []string{"skills/cogmer/SKILL.md", "codex-skills/cogmer/SKILL.md"} {
		if b, err := os.ReadFile(filepath.Join(plugin, filepath.FromSlash(skill))); err == nil {
			for _, m := range skillCall.FindAllStringSubmatch(string(b), -1) {
				set[m[1]] = true
			}
		}
	}
	var out []string
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// compatEnv is the environment the older plugin's scripts run in: a state directory and
// addresses of their own, and the current program named the way the scripts find one.
func compatEnv(t *testing.T, bin string) []string {
	t.Helper()
	home := t.TempDir()
	return append(os.Environ(), "HOME="+home, "COGMER_HOME="+filepath.Join(home, stateDirName), "COGMER_BIN="+bin,
		"COGMER_ADDR="+freeAddr(t), "COGMER_PEER_ADDR="+freeAddr(t), "COGMER_TAILCAT=off", "COGMER_HOST=claude",
		"COGMER_PREFLIGHT=off", "CLAUDE_CODE_SESSION_ID=s-1")
}

// stopStrays ends a daemon a command started, if one did, at the addresses the test made.
func stopStrays(t *testing.T, bin string, env []string) {
	t.Helper()
	t.Cleanup(func() {
		cmd := exec.Command(bin, "stop")
		cmd.Env = env
		_, _ = cmd.CombinedOutput()
	})
}

// subcommandsRunWithNoArguments are the subcommands an older plugin's commands call that can
// be run with none and change nothing outside the state directory the test made. `cowork-setup`
// writes the configuration of a desktop app, and `restart` starts a daemon.
func runnableWithNoArguments(sub string) bool { return sub != "cowork-setup" && sub != "restart" }

const usageHeader = "a conversation shared between Claude Code and Codex sessions"

// Every subcommand an older plugin's commands call is one the current program knows, and the
// plugin's own script runs it to something other than the program's usage text.
func TestTheCurrentProgramAnswersEverySubcommandOfRecentPlugins(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	tags, dirs := releasePlugins(t, compatReleases)
	bin := buildVersion(t, t.TempDir(), testBuildVersion)
	mainSrc, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, tag := range tags {
		plugin := dirs[tag]
		t.Run(tag, func(t *testing.T) {
			subs := calledSubcommands(t, plugin)
			if len(subs) < 8 {
				t.Fatalf("found only %v in the plugin of %s, so the scan reads the wrong thing", subs, tag)
			}
			env := compatEnv(t, bin)
			stopStrays(t, bin, env)
			for _, sub := range subs {
				if sub != "restart" && !bytes.Contains(mainSrc, []byte("\tcase \""+sub+"\":")) {
					t.Errorf("the plugin of %s calls the subcommand %q, which the current program has no case for", tag, sub)
					continue
				}
				if !runnableWithNoArguments(sub) {
					continue
				}
				cmd := exec.Command("bash", filepath.Join(plugin, "cli.sh"), sub)
				cmd.Env = env
				out, _ := cmd.CombinedOutput()
				if strings.Contains(string(out), usageHeader) {
					t.Errorf("the plugin of %s runs %q and the current program answers with its usage text:\n%s", tag, sub, out)
				}
				if strings.TrimSpace(string(out)) == "" {
					t.Errorf("the plugin of %s runs %q and gets no answer", tag, sub)
				}
			}
		})
	}
}

// Each hook line of an older plugin, run through its own handler against the current program,
// exits 0 and says nothing when no daemon answers, as a hook must (D-011, hooks fail open:
// exit 0, empty stdout).
func TestTheHookLinesOfRecentPluginsFailOpenAgainstTheCurrentProgram(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	tags, dirs := releasePlugins(t, compatReleases)
	bin := buildVersion(t, t.TempDir(), testBuildVersion)
	for _, tag := range tags {
		plugin := dirs[tag]
		t.Run(tag, func(t *testing.T) {
			ran := 0
			for _, hookFile := range []string{"hooks/hooks.json", "hooks/codex-hooks.json"} {
				raw, err := os.ReadFile(filepath.Join(plugin, filepath.FromSlash(hookFile)))
				if err != nil {
					continue
				}
				var cfg struct {
					Hooks map[string][]struct {
						Hooks []struct {
							Command string `json:"command"`
						} `json:"hooks"`
					} `json:"hooks"`
				}
				if err := json.Unmarshal(raw, &cfg); err != nil {
					t.Fatalf("%s of %s: %v", hookFile, tag, err)
				}
				for event, groups := range cfg.Hooks {
					for _, g := range groups {
						for _, h := range g.Hooks {
							// The session-start line fetches a binary from the network when the
							// versions differ, which the installer's own tests cover.
							if !strings.Contains(h.Command, "run.sh") {
								continue
							}
							line := strings.NewReplacer("${CLAUDE_PLUGIN_ROOT}", plugin, "${PLUGIN_ROOT}", plugin).Replace(h.Command)
							env := compatEnv(t, bin)
							env = append(env, "COGMER_ADDR=127.0.0.1:1")
							cmd := exec.Command("bash", "-c", line)
							cmd.Env = env
							cmd.Stdin = strings.NewReader(`{"session_id":"s-1","turn_id":"t-1","prompt":"hello","transcript_path":"/nonexistent","last_assistant_message":"x"}`)
							var stdout bytes.Buffer
							cmd.Stdout = &stdout
							if err := cmd.Run(); err != nil || stdout.Len() != 0 {
								t.Errorf("the %s line of %s (%s) disturbed the session: %v, stdout %q", event, tag, hookFile, err, stdout.String())
							}
							ran++
						}
					}
				}
			}
			if ran < 2 {
				t.Errorf("ran %d hook lines of %s, so the scan reads the wrong thing", ran, tag)
			}
		})
	}
}

// What an older plugin's scripts read from the program is still written as they read it: the
// daemon's mark of why it could not start, and the version in the health answer.
func TestWhatRecentPluginsReadFromTheProgramIsStillWrittenAsTheyReadIt(t *testing.T) {
	tags, dirs := releasePlugins(t, compatReleases)
	for _, tag := range tags {
		plugin := dirs[tag]
		t.Run(tag, func(t *testing.T) {
			common := filepath.Join(plugin, "hooks-handlers", "common.sh")
			src, err := os.ReadFile(common)
			if err != nil {
				t.Fatal(err)
			}

			// the mark: `state<TAB>seconds<TAB>detail`, read by daemon_state
			if strings.Contains(string(src), "daemon_state()") {
				t.Setenv("COGMER_HOME", filepath.Join(t.TempDir(), stateDirName))
				recordDaemonState("blocked", "the address is held")
				cmd := exec.Command("bash", "-c", `. "$1"; daemon_state`, "_", common)
				cmd.Env = os.Environ()
				out, err := cmd.Output()
				if err != nil || !regexp.MustCompile(`^blocked \d+ the address is held$`).Match(out) {
					t.Errorf("daemon_state of %s reads %q from the mark the program writes (%v)", tag, out, err)
				}
			}

			// the version in the health answer, read by a sed expression on the daemon's reply
			if strings.Contains(string(src), `"version":"\([^"]*\)"`) {
				a, _ := twoMachines(t)
				rec := httptestGet(t, a.d.health)
				cmd := exec.Command("sed", "-n", `s/.*"version":"\([^"]*\)".*/\1/p`)
				cmd.Stdin = strings.NewReader(rec)
				out, _ := cmd.Output()
				if strings.TrimSpace(string(out)) == "" {
					t.Errorf("the sed expression of %s finds no version in the health answer %q", tag, rec)
				}
			}
		})
	}
}

// httptestGet is the body a handler answers a GET with.
func httptestGet(t *testing.T, h http.HandlerFunc) string {
	t.Helper()
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	return rec.Body.String()
}
