package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The person is shown one line when the install finishes and when cogmer needs repair (D-282,
// the person is shown one line when the install finishes and when cogmer needs repair).

func TestTheInstallNoticeIsTakenOnceAndNamesTheVersionAndTheNextCommand(t *testing.T) {
	t.Setenv("COGMER_HOME", t.TempDir())
	if takeInstallNotice() != "" {
		t.Fatal("a notice came with no mark")
	}
	if err := os.WriteFile(installNoticePath(), []byte("0.10.6\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	line := takeInstallNotice()
	if !strings.Contains(line, "cogmer 0.10.6 is installed") || !strings.Contains(line, "/cogmer:self-status") {
		t.Errorf("the notice reads %q", line)
	}
	if _, err := os.Stat(installNoticePath()); err == nil {
		t.Error("the mark stayed after it was taken")
	}
	if takeInstallNotice() != "" {
		t.Error("the notice was shown twice")
	}
	if err := os.WriteFile(installNoticePath(), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if line := takeInstallNotice(); !strings.HasPrefix(line, "cogmer is installed") {
		t.Errorf("an empty mark gave %q", line)
	}
}

func TestOnlyAHostThatShowsALineTakesTheNotice(t *testing.T) {
	t.Setenv("COGMER_HOME", t.TempDir())
	if err := os.WriteFile(installNoticePath(), []byte("0.10.6\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, h := range []Host{codexHost{}, coworkHost{}} {
		if noticeFor(h) != "" {
			t.Errorf("%s took a notice it cannot show", h.Name())
		}
		if _, err := os.Stat(installNoticePath()); err != nil {
			t.Fatalf("%s removed the mark: %v", h.Name(), err)
		}
	}
	if line := noticeFor(claudeHost{}); !strings.Contains(line, "0.10.6") {
		t.Errorf("Claude Code took %q", line)
	}
	if _, err := os.Stat(installNoticePath()); err == nil {
		t.Error("Claude Code left the mark")
	}
}

func TestAPromptHookCarriesTheNoticeInTheFormAHostShows(t *testing.T) {
	// Claude Code, with a notice and with context.
	var both struct {
		System string `json:"systemMessage"`
		Hook   struct {
			Event   string `json:"hookEventName"`
			Context string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	raw := promptHookOutput(claudeHost{}, "NOTICE", "CONTEXT")
	if err := json.Unmarshal(raw, &both); err != nil || both.System != "NOTICE" || both.Hook.Context != "CONTEXT" || both.Hook.Event != "UserPromptSubmit" {
		t.Errorf("Claude Code's output is %q: %v", raw, err)
	}
	// With a notice and no context, a message and nothing else.
	var only map[string]any
	if err := json.Unmarshal(promptHookOutput(claudeHost{}, "NOTICE", ""), &only); err != nil || len(only) != 1 || only["systemMessage"] != "NOTICE" {
		t.Errorf("a notice with no context is %v: %v", only, err)
	}
	// With no notice, the context in the form the host has always read.
	if got := promptHookOutput(claudeHost{}, "", "CONTEXT"); string(got) != "CONTEXT\n" {
		t.Errorf("Claude Code's plain context is %q", got)
	}
	if got := promptHookOutput(claudeHost{}, "", ""); len(got) != 0 {
		t.Errorf("nothing to say wrote %q", got)
	}
	// A host that shows nothing gets the context alone, and never a systemMessage.
	for _, h := range []Host{codexHost{}, coworkHost{}} {
		got := promptHookOutput(h, "NOTICE", "CONTEXT")
		if bytes.Contains(got, []byte("systemMessage")) || bytes.Contains(got, []byte("NOTICE")) || !bytes.Contains(got, []byte("CONTEXT")) {
			t.Errorf("%s's output is %q", h.Name(), got)
		}
		if got := promptHookOutput(h, "NOTICE", ""); len(got) != 0 {
			t.Errorf("%s wrote %q for a notice alone", h.Name(), got)
		}
	}
}

// startHook runs the session-start hook as it is, with a stand-in program for cogmer that
// reports a version and, for the install notice, prints a line and removes the mark as the real
// one does. It returns the JSON the hook printed.
func startHook(t *testing.T, home, bin string, args ...string) map[string]any {
	t.Helper()
	root, _ := cliHarness(t)
	src, err := os.ReadFile("../../plugin/hooks-handlers/session-start.sh")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "hooks-handlers", "session-start.sh")
	if err := os.WriteFile(path, src, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", append([]string{path}, args...)...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + home, "COGMER_HOME=" + home, "CLAUDE_PLUGIN_ROOT=" + root,
		"COGMER_BIN=" + bin, "COGMER_ADDR=127.0.0.1:1"}
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(out, &v); err != nil {
		t.Fatalf("the hook did not print one JSON value: %v\n%s", err, out)
	}
	return v
}

func noticeStand(t *testing.T) string {
	t.Helper()
	return standInProgram(t, t.TempDir(), "cogmer",
		`case "$1" in version) echo 9.9.9;; notice) echo "STUB NOTICE"; rm -f "$COGMER_HOME/install-notice";; esac`,
		"echo 9.9.9")
}

func systemMessage(v map[string]any) string {
	s, _ := v["systemMessage"].(string)
	return s
}

func TestTheSessionStartHookShowsAFaultInOneLineAndNothingWhileTheInstallRuns(t *testing.T) {
	_, home := cliHarness(t)
	writeInstallState(t, home, "failed", 5*time.Minute, "no verified download and no usable Go toolchain")
	v := startHook(t, home, "/missing")
	if got := systemMessage(v); !strings.Contains(got, "cogmer failed to install: no verified download and no usable Go toolchain") ||
		!strings.Contains(got, "tries again in about an hour") {
		t.Errorf("a failed install is shown as %q", got)
	}
	// The model's note is still there, so a session asked why answers from the fact.
	if h, _ := v["hookSpecificOutput"].(map[string]any); !strings.Contains(h["additionalContext"].(string), "standing policy") ||
		!strings.Contains(h["additionalContext"].(string), "cogmer failed to install") {
		t.Errorf("the note for the model is %v", h)
	}

	_, home = cliHarness(t)
	writeInstallState(t, home, "installing", 5*time.Second, "fetching v0.10.6")
	if got := systemMessage(startHook(t, home, "/missing")); got != "" {
		t.Errorf("an install in progress is shown as %q", got)
	}

	_, home = cliHarness(t)
	writeInstallState(t, home, "installing", 20*time.Minute, "fetching v0.10.6")
	if got := systemMessage(startHook(t, home, "/missing")); !strings.Contains(got, "did not finish") {
		t.Errorf("a stalled install is shown as %q", got)
	}
}

func TestTheSessionStartHookShowsABlockedDaemonAndAFailedUpgradeAndNothingWhenAllIsWell(t *testing.T) {
	bin := noticeStand(t)
	for state, want := range map[string]string{
		"blocked":        "cogmer's daemon could not start: port 4782 is held by another program. Nothing is captured or shared",
		"upgrade-failed": "cogmer could not upgrade its data: the copy of the databases was not complete. Nothing is captured or shared",
	} {
		_, home := cliHarness(t)
		detail := map[string]string{"blocked": "port 4782 is held by another program", "upgrade-failed": "the copy of the databases was not complete"}[state]
		mark := state + "\t" + itoaNow() + "\t" + detail + "\n"
		if err := os.WriteFile(filepath.Join(home, "daemon-state"), []byte(mark), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := systemMessage(startHook(t, home, bin)); !strings.Contains(got, want) {
			t.Errorf("a %s daemon is shown as %q", state, got)
		}
	}
	_, home := cliHarness(t)
	if got := systemMessage(startHook(t, home, bin)); got != "" {
		t.Errorf("a machine with nothing wrong is shown %q", got)
	}
}

func itoaNow() string { return strconv.FormatInt(time.Now().Unix(), 10) }

func TestTheSessionStartHookAnnouncesAFirstInstallOnceAndNeverToCodex(t *testing.T) {
	bin := noticeStand(t)
	_, home := cliHarness(t)
	if err := os.WriteFile(filepath.Join(home, "install-notice"), []byte("9.9.9\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Codex's hook line passes codex, and Codex is not shown a line.
	if got := systemMessage(startHook(t, home, bin, "codex")); got != "" {
		t.Errorf("Codex was given the line %q", got)
	}
	if _, err := os.Stat(filepath.Join(home, "install-notice")); err != nil {
		t.Fatal("Codex's hook took the mark")
	}
	if got := systemMessage(startHook(t, home, bin)); got != "STUB NOTICE" {
		t.Errorf("the finished install is shown as %q", got)
	}
	if got := systemMessage(startHook(t, home, bin)); got != "" {
		t.Errorf("the finished install was shown twice: %q", got)
	}

	// A fault is shown before the announcement, and the mark waits.
	_, home = cliHarness(t)
	if err := os.WriteFile(filepath.Join(home, "install-notice"), []byte("9.9.9\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "daemon-state"), []byte("blocked\t"+itoaNow()+"\tport in use\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := systemMessage(startHook(t, home, bin)); !strings.Contains(got, "could not start") {
		t.Errorf("a fault was shown as %q", got)
	}
	if _, err := os.Stat(filepath.Join(home, "install-notice")); err != nil {
		t.Error("a fault consumed the announcement")
	}
}
