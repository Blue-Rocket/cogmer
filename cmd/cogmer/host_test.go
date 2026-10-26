package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A host is declared by the entry point that cogmer wrote for it and resolved where
// the request or the command arrives. The daemon serves every host at once and holds
// none (D-203).

func env(pairs ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return func(k string) string { return m[k] }
}

func TestTheHostOfACommandIsDeclaredOrTheOnlyOneInTheEnvironment(t *testing.T) {
	cases := []struct {
		name string
		env  func(string) string
		want string // "" for no host
		err  string
	}{
		{"a terminal holds nothing", env(), "", ""},
		{"the declaration wins over the variables", env("COGMER_HOST", "codex", "CLAUDE_CODE_SESSION_ID", "c", "CODEX_THREAD_ID", "x"), "codex", ""},
		{"a declared Claude wins in a shell Codex started", env("COGMER_HOST", "claude", "CLAUDE_CODE_SESSION_ID", "c", "CODEX_THREAD_ID", "x"), "claude", ""},
		{"only Claude's variable", env("CLAUDE_CODE_SESSION_ID", "c"), "claude", ""},
		{"only Codex's variable", env("CODEX_THREAD_ID", "x"), "codex", ""},
		{"both variables and no declaration", env("CLAUDE_CODE_SESSION_ID", "c", "CODEX_THREAD_ID", "x"), "", "cannot tell which host"},
		{"a host nobody supports", env("COGMER_HOST", "emacs"), "", "no host cogmer supports"},
		{"a blank declaration is no declaration", env("COGMER_HOST", "  ", "CODEX_THREAD_ID", "x"), "codex", ""},
	}
	for _, c := range cases {
		h, err := hostFromEnv(c.env)
		switch {
		case c.err != "":
			if err == nil || !strings.Contains(err.Error(), c.err) {
				t.Errorf("%s: err = %v, want one saying %q", c.name, err, c.err)
			}
		case err != nil:
			t.Errorf("%s: %v", c.name, err)
		case c.want == "" && h != nil:
			t.Errorf("%s: resolved to %s, want no host", c.name, h.Name())
		case c.want != "" && (h == nil || h.Name() != c.want):
			t.Errorf("%s: resolved to %v, want %s", c.name, h, c.want)
		}
	}
	var amb *AmbiguousHostError
	if _, err := hostFromEnv(env("CLAUDE_CODE_SESSION_ID", "c", "CODEX_THREAD_ID", "x")); !errors.As(err, &amb) || len(amb.Hosts) != 2 {
		t.Errorf("an ambiguous environment does not name both hosts: %v", err)
	}
}

func TestAHookThatDeclaresNoHostIsClaudeCodesAndAnUnknownOneIsRefused(t *testing.T) {
	if h, ok := hostByName(""); !ok || h.Name() != "claude" {
		t.Error("a hook that declares no host, as every hook installed before the second host does, is not Claude Code's")
	}
	if h, ok := hostByName("codex"); !ok || h.Name() != "codex" {
		t.Error("codex is not found by name")
	}
	if _, ok := hostByName("nope"); ok {
		t.Error("an unsupported host was accepted")
	}
	seen := map[string]bool{}
	for _, h := range hosts {
		if seen[h.Name()] || seen[h.SessionEnv()] {
			t.Errorf("two hosts share the name or session variable of %s", h.Name())
		}
		seen[h.Name()], seen[h.SessionEnv()] = true, true
	}
}

func TestACommandIsWrittenTheWayItsHostInvokesIt(t *testing.T) {
	const text = "run /cogmer:room-create, then /cogmer:self-status"
	if got := (claudeHost{}).CommandText(text); got != text {
		t.Errorf("Claude Code's commands were rewritten: %q", got)
	}
	if got := (codexHost{}).CommandText(text); got != "run $cogmer room-create, then $cogmer self-status" {
		t.Errorf("Codex's commands are %q", got)
	}
	t.Setenv("CLAUDE_CODE_SESSION_ID", "")
	t.Setenv("CODEX_THREAD_ID", "")
	t.Setenv("COGMER_HOST", "codex")
	if got := commandText(text); strings.Contains(got, "/cogmer:") {
		t.Errorf("a command run from Codex names a slash command: %q", got)
	}
	t.Setenv("COGMER_HOST", "")
	if got := commandText(text); got != text {
		t.Errorf("a terminal's commands were rewritten: %q", got)
	}
}

func TestDoctorChecksTheHostNamedTheHostInUseOrEveryHostInstalled(t *testing.T) {
	both := func(h Host) bool { return h.Name() != "cowork" }
	onlyCodex := func(h Host) bool { return h.Name() == "codex" }
	none := func(Host) bool { return false }
	names := func(hs []Host) string {
		var out []string
		for _, h := range hs {
			out = append(out, h.Name())
		}
		return strings.Join(out, ",")
	}

	if hs, err := doctorTargets("codex", claudeHost{}, both); err != nil || names(hs) != "codex" {
		t.Errorf("a named host is not the one checked: %v %v", names(hs), err)
	}
	if hs, err := doctorTargets("cowork", nil, both); err != nil || names(hs) != "cowork" {
		t.Errorf("a named Cowork host was not the one checked: %v %v", names(hs), err)
	}
	if (coworkHost{}).Installed() {
		t.Error("Cowork counts as installed with no setup that says it is in use, so a terminal doctor would check it")
	}
	if hs, err := doctorTargets("", codexHost{}, both); err != nil || names(hs) != "codex" {
		t.Errorf("inside Codex, doctor checks %v %v", names(hs), err)
	}
	if hs, err := doctorTargets("", nil, both); err != nil || names(hs) != "claude,codex" {
		t.Errorf("at a terminal with both installed, doctor checks %v %v", names(hs), err)
	}
	if hs, err := doctorTargets("", nil, onlyCodex); err != nil || names(hs) != "codex" {
		t.Errorf("at a terminal with only Codex installed, doctor checks %v %v", names(hs), err)
	}
	if _, err := doctorTargets("", nil, none); err == nil || !strings.Contains(err.Error(), "no supported host is installed") {
		t.Errorf("a machine with no host got %v", err)
	}
	if _, err := doctorTargets("emacs", nil, both); err == nil || !strings.Contains(err.Error(), "unknown host") {
		t.Errorf("an unknown host name got %v", err)
	}
}

func TestEachHostHandsContextToItsModelInItsOwnForm(t *testing.T) {
	const context = "quote: \" <team-conversation>\ncomplete block"
	if got := string((claudeHost{}).PromptHookOutput(context)); got != context+"\n" {
		t.Errorf("Claude Code's hook output is %q, want the context on standard output", got)
	}
	var decoded struct {
		Out struct {
			Event   string `json:"hookEventName"`
			Context string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	out := (codexHost{}).PromptHookOutput(context)
	if err := json.Unmarshal(out, &decoded); err != nil || decoded.Out.Event != "UserPromptSubmit" || decoded.Out.Context != context {
		t.Errorf("Codex's hook output is not valid JSON carrying the context unchanged: %s (%v)", out, err)
	}
	if strings.Contains(string(out), "\n") && !strings.HasSuffix(string(out), "}\n") {
		t.Errorf("Codex's hook output has a stray newline: %q", out)
	}
}

func TestTheHostFlagIsTakenFromAHookLine(t *testing.T) {
	rest, host := takeValue([]string{"--host", "codex", "prompt"}, "--host")
	if host != "codex" || len(rest) != 1 || rest[0] != "prompt" {
		t.Errorf("hook --host codex prompt read as %v and %q", rest, host)
	}
	rest, host = takeValue([]string{"prompt"}, "--host")
	if host != "" || len(rest) != 1 {
		t.Errorf("a hook with no host read as %v and %q", rest, host)
	}
	rest, host = takeValue([]string{"prompt", "--host"}, "--host")
	if host != "" || len(rest) != 2 {
		t.Errorf("a trailing flag with no value read as %v and %q", rest, host)
	}
}

// A hook that names a host cogmer does not support leaves the host alone: no output
// and a zero exit (§3.1).
func TestAHookForAnUnsupportedHostFailsOpen(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	bin := buildVersion(t, t.TempDir(), testBuildVersion)
	cmd := exec.Command(bin, "hook", "--host", "emacs", "prompt")
	cmd.Stdin = strings.NewReader(`{"session_id":"s","prompt":"x"}`)
	cmd.Env = append(os.Environ(), "COGMER_ADDR=127.0.0.1:1")
	out, err := cmd.Output()
	if err != nil || len(out) != 0 {
		t.Errorf("a hook for an unsupported host disturbed the session: %v %q", err, out)
	}
}

// The daemon holds no host. The version it knows of each is looked up in the
// background, so that a hook is not delayed by running another program, and it is
// "unknown" until the first lookup returns, which must fall on the safe side.
func TestAHostVersionIsLookedUpInTheBackgroundAndRefreshed(t *testing.T) {
	d := &Daemon{}
	clock := time.Unix(1_000_000, 0)
	d.versions.now = func() time.Time { return clock }
	d.versions.ttl = time.Minute
	release := make(chan struct{}, 8)
	version := "1.0"
	lookups := 0
	d.versions.lookup = func(Host) string { <-release; lookups++; return version }

	if got := d.hostVersion(claudeHost{}); got != "" {
		t.Fatalf("a version never looked up read as %q, want unknown", got)
	}
	// Asked again while the lookup runs: still unknown, and not a second lookup.
	if got := d.hostVersion(claudeHost{}); got != "" {
		t.Fatalf("a version still being looked up read as %q", got)
	}
	release <- struct{}{}
	d.background.Wait()
	if got := d.hostVersion(claudeHost{}); got != "1.0" || lookups != 1 {
		t.Fatalf("after the lookup: %q after %d lookups, want 1.0 after one", got, lookups)
	}

	// The host is upgraded while the daemon runs. Within the ttl the old version
	// stands, and after it the old version is returned once more while the new one is
	// fetched.
	version = "2.0"
	clock = clock.Add(30 * time.Second)
	if got := d.hostVersion(claudeHost{}); got != "1.0" {
		t.Errorf("within the ttl the version changed to %q", got)
	}
	clock = clock.Add(2 * time.Minute)
	if got := d.hostVersion(claudeHost{}); got != "1.0" {
		t.Errorf("a stale version was not returned while the new one is fetched: %q", got)
	}
	release <- struct{}{}
	d.background.Wait()
	if got := d.hostVersion(claudeHost{}); got != "2.0" {
		t.Errorf("an upgrade after the daemon started was not seen: %q", got)
	}

	// Each host has its own entry.
	version = "9.9"
	d.hostVersion(codexHost{})
	release <- struct{}{}
	d.background.Wait()
	if c, x := d.hostVersion(claudeHost{}), d.hostVersion(codexHost{}); c != "2.0" || x != "9.9" {
		t.Errorf("hosts share a version: claude %q, codex %q", c, x)
	}
}

// A turn offered to a session with no transcript evidence is counted as delivered only
// for a host whose behavior check is recorded as failing on a version known to be
// running. An unknown version leaves it pending, and Codex always does.
func TestDeliveryWithoutEvidenceIsTrustedOnlyWhereThatHostsCheckFailed(t *testing.T) {
	// Each case has a state directory of its own, with the verification records in it.
	pending := func(t *testing.T, session string) (*Store, stopReq) {
		s := testStore(t)
		saveVerification(verification{Version: "1.0", Failures: map[string]string{"B20": "no attachment"}})
		saveVerification(verification{Version: "codex:1.0", Failures: map[string]string{"B20": "no attachment"}})
		seedTeammate(t, s, 1)
		events, _ := s.UndeliveredFor("test", session)
		if err := s.RecordPending("turn", session, events, "complete block"); err != nil {
			t.Fatal(err)
		}
		return s, stopReq{SessionID: session, PromptID: "turn", TranscriptPath: write(t, "")}
	}
	undelivered := func(s *Store, session string) int {
		left, _ := s.UndeliveredFor("test", session)
		return len(left)
	}
	daemonSeeing := func(t *testing.T, version string) *Daemon {
		d := &Daemon{}
		d.versions.lookup = func(Host) string { return version }
		t.Cleanup(d.background.Wait)
		return d
	}

	t.Run("claude, check failed on the running version", func(t *testing.T) {
		d := daemonSeeing(t, "1.0")
		s, req := pending(t, "claude-session")
		d.confirmDelivery(s, req, claudeHost{})
		if undelivered(s, "claude-session") != 1 {
			t.Error("a version not looked up yet was trusted, which loses an injection that never arrived")
		}
		d.background.Wait()
		d.confirmDelivery(s, req, claudeHost{})
		if undelivered(s, "claude-session") != 0 {
			t.Error("a version whose check is recorded as failing was not trusted")
		}
	})

	t.Run("claude, check has not failed on the running version", func(t *testing.T) {
		d := daemonSeeing(t, "3.0")
		s, req := pending(t, "claude-session")
		d.confirmDelivery(s, req, claudeHost{})
		d.background.Wait()
		d.confirmDelivery(s, req, claudeHost{})
		if undelivered(s, "claude-session") != 1 {
			t.Error("a turn was trusted on a version whose check has not failed")
		}
	})

	t.Run("codex, even where the same check is recorded as failing", func(t *testing.T) {
		d := daemonSeeing(t, "1.0")
		s, req := pending(t, "codex:session")
		req.Host = "codex"
		d.confirmDelivery(s, req, codexHost{})
		d.background.Wait()
		d.confirmDelivery(s, req, codexHost{})
		if undelivered(s, "codex:session") != 1 {
			t.Error("a Codex turn was trusted without evidence")
		}
	})
}

// The commands of the Claude plugin call cli.sh directly and Codex's go through
// codex-cli.sh, so cli.sh declares the host unless a wrapper already did.
func TestCliDeclaresTheHostUnlessAWrapperAlreadyDid(t *testing.T) {
	root, home := cliHarness(t)
	if err := os.MkdirAll(filepath.Join(home, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	stand := "#!/usr/bin/env bash\necho \"host=$COGMER_HOST\"\n"
	if err := os.WriteFile(filepath.Join(home, "bin", "cogmer"), []byte(stand), 0o755); err != nil {
		t.Fatal(err)
	}
	run := func(extra ...string) string {
		cmd := exec.Command("bash", filepath.Join(root, "cli.sh"), "whoami")
		cmd.Env = append([]string{"PATH=/usr/bin:/bin", "HOME=" + home, "COGMER_HOME=" + home, "CLAUDE_PLUGIN_ROOT=" + root}, extra...)
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(out))
	}
	if got := run(); got != "host=claude" {
		t.Errorf("a command run with nothing declared reached the binary as %q, want host=claude", got)
	}
	if got := run("COGMER_HOST=codex"); got != "host=codex" {
		t.Errorf("a wrapper's declaration was overwritten: %q", got)
	}
}

// A command that is about to write binds the session that ran it, and with no way to
// name that session it would leave a room with nobody in it. So when the host cannot
// be told it refuses before it writes anything, and says so.
func TestCreateAndJoinRefuseBeforeWritingWhenTheHostCannotBeTold(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	dir := t.TempDir()
	bin := buildVersion(t, dir, testBuildVersion)
	home := filepath.Join(dir, "home")
	run := func(extra []string, args ...string) (string, int) {
		cmd := exec.Command(bin, args...)
		cmd.Env = append([]string{"PATH=/usr/bin:/bin", "HOME=" + dir, "COGMER_HOME=" + home,
			"COGMER_TAILCAT=off", "COGMER_PREFLIGHT=off", "COGMER_IDENTITY_PAGE=off", "COGMER_ADDR=127.0.0.1:1"}, extra...)
		out, err := cmd.CombinedOutput()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
		return string(out), code
	}
	both := []string{"CLAUDE_CODE_SESSION_ID=a", "CODEX_THREAD_ID=b"}
	unknown := []string{"COGMER_HOST=emacs"}
	invitation := "misty-canyon/00000000-0000-4000-8000-000000000000@127.0.0.1:9#" + testIdentity(t).PeerID

	for name, extra := range map[string][]string{"both hosts' variables": both, "a host nobody supports": unknown} {
		for _, args := range [][]string{{"create"}, {"create", "--show-only"}, {"join", invitation}} {
			out, code := run(extra, args...)
			if code == 0 || !strings.Contains(out, "Nothing was created or recorded") {
				t.Errorf("%s: %v was not refused: exit %d\n%s", name, args, code, out)
			}
		}
		if out, _ := run(nil, "rooms"); !strings.Contains(out, "no rooms") {
			t.Errorf("%s: a refused command left a room behind:\n%s", name, out)
		}
		if out, _ := run(nil, "peers"); !strings.Contains(out, "No peers yet") {
			t.Errorf("%s: a refused join recorded a peer:\n%s", name, out)
		}
	}

	// The same commands go through when the host is clear: declared, one variable, or a
	// terminal with neither.
	for name, extra := range map[string][]string{
		"declared":      {"COGMER_HOST=claude", "CLAUDE_CODE_SESSION_ID=a", "CODEX_THREAD_ID=b"},
		"one variable":  {"CLAUDE_CODE_SESSION_ID=a2"},
		"at a terminal": nil,
	} {
		if out, code := run(extra, "create"); code != 0 || !strings.Contains(out, "created") {
			t.Errorf("%s: create was refused: exit %d\n%s", name, code, out)
		}
	}
}

// What a command prints about the next command is written the way the host invokes it.
// A user cannot run "cogmer invite" at a terminal, so naming it that way sends them
// to something that does not work.
func TestCommandsNameTheNextCommandTheWayTheHostInvokesIt(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	dir := t.TempDir()
	bin := buildVersion(t, dir, testBuildVersion)
	run := func(home string, extra []string, args ...string) string {
		cmd := exec.Command(bin, args...)
		cmd.Env = append([]string{"PATH=/usr/bin:/bin", "HOME=" + dir, "COGMER_HOME=" + home,
			"COGMER_TAILCAT=off", "COGMER_PREFLIGHT=off", "COGMER_IDENTITY_PAGE=off", "COGMER_ADDR=127.0.0.1:1"}, extra...)
		out, _ := cmd.CombinedOutput()
		return string(out)
	}
	for _, c := range []struct {
		host   string
		env    []string
		invite string
		join   string
	}{
		{"claude", []string{"COGMER_HOST=claude", "CLAUDE_CODE_SESSION_ID=a"}, "/cogmer:room-invite <name>", "/cogmer:room-join <invitation>"},
		{"codex", []string{"COGMER_HOST=codex", "CODEX_THREAD_ID=b"}, "$cogmer room-invite <name>", "$cogmer room-join <invitation>"},
	} {
		home := filepath.Join(dir, c.host)
		if out := run(home, c.env, "create"); !strings.Contains(out, c.invite) || strings.Contains(out, "cogmer invite") {
			t.Errorf("%s: create does not name the invite command as %q:\n%s", c.host, c.invite, out)
		}
		if out := run(home, c.env, "join"); !strings.Contains(out, c.join) || strings.Contains(out, "cogmer join") {
			t.Errorf("%s: join with no argument does not name its usage as %q:\n%s", c.host, c.join, out)
		}
	}
}

// Outside a host the binary is for tests. A user is told to run the command through the
// plugin, and is not told how to declare a host by hand.
func TestTheRefusalOfAnAmbiguousHostDoesNotSendAUserToTheEnvironment(t *testing.T) {
	_, err := hostFromEnv(env("CLAUDE_CODE_SESSION_ID", "c", "CODEX_THREAD_ID", "x"))
	if err == nil {
		t.Fatal("an ambiguous environment was not refused")
	}
	msg := err.Error()
	if !strings.Contains(msg, "plugin") {
		t.Errorf("the refusal does not say to use the plugin's command: %q", msg)
	}
	for _, leak := range []string{"COGMER_HOST", "environment variable", "emulate"} {
		if strings.Contains(msg, leak) {
			t.Errorf("the refusal tells a user about %q: %q", leak, msg)
		}
	}
}
