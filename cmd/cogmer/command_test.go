package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestCommandHelperProcess is the program a command runs in the tests below: it reports
// how it was started, and does what its first argument asks.
func TestCommandHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_COMMAND_HELPER") != "1" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	last := ""
	if len(args) > 0 {
		last = args[len(args)-1]
	}
	if last == "silent" {
		os.Exit(0)
	}
	fmt.Printf("ARGS:%s\nHOST:%s\nSESSION:%s\nLEAK:%s/%s\n", strings.Join(args, " "),
		os.Getenv("COGMER_HOST"), os.Getenv("COGMER_COWORK_SESSION_ID"),
		os.Getenv("CLAUDE_CODE_SESSION_ID"), os.Getenv("CODEX_THREAD_ID"))
	switch last {
	case "refuse":
		fmt.Println("this room cannot be joined")
		os.Exit(1)
	case "sleep":
		time.Sleep(10 * time.Second)
	}
	os.Exit(0)
}

// helperRunner runs the helper above in place of the binary's subcommands.
func helperRunner() commandRunner {
	return commandRunner{
		exe: os.Args[0], pre: []string{"-test.run=TestCommandHelperProcess", "--"},
		timeout: 5 * time.Second,
		environ: func() []string {
			return append(os.Environ(), "GO_WANT_COMMAND_HELPER=1",
				"CLAUDE_CODE_SESSION_ID=the-session-that-started-the-daemon", "CODEX_THREAD_ID=another")
		},
	}
}

func TestATypedCommandIsReadFromTheFirstLineOfThePrompt(t *testing.T) {
	for _, c := range []struct {
		prompt string
		name   string
		args   string
		ok     bool
	}{
		{"/cogmer:room-create --show-only", "room-create", "--show-only", true},
		{"  /cogmer:self-status  ", "self-status", "", true},
		{"/cogmer:room-join  a-room   --show-only", "room-join", "a-room --show-only", true},
		{"/cogmer:peer-pair abc\nand then this second line", "peer-pair", "abc", true},
		{"/cogmer:", "", "", false},
		{"/cogmer: room-create", "room-create", "", true}, // the first word after the prefix
		{"please run /cogmer:room-create", "", "", false},
		{"/other:thing", "", "", false},
		{"hello", "", "", false},
		{"", "", "", false},
	} {
		got, ok := parseTypedCommand(c.prompt, "/cogmer:")
		if ok != c.ok || got.name != c.name || strings.Join(got.args, " ") != c.args {
			t.Errorf("%q read as %q %q %v, want %q %q %v", c.prompt, got.name, strings.Join(got.args, " "), ok, c.name, c.args, c.ok)
		}
	}
	if _, ok := parseTypedCommand("/cogmer:room-create", ""); ok {
		t.Error("a host with no prefix had a prompt read as a command")
	}
	if got, ok := parseTypedCommand("$cogmer room-join a-room", "$cogmer "); !ok || got.name != "room-join" || strings.Join(got.args, " ") != "a-room" {
		t.Errorf("a Codex command read as %v %v", got, ok)
	}
	for _, not := range []string{"$cogmerx room-join", "$cogmer", "tell $cogmer room-join"} {
		if _, ok := parseTypedCommand(not, "$cogmer "); ok {
			t.Errorf("%q was read as a Codex command", not)
		}
	}
}

// The table is the plugin's command files seen from the daemon, and a command added to one
// and not the other would work in Claude Code and fail in Cowork, or the reverse.
func TestTheTypedCommandsAreTheOnesThePluginHas(t *testing.T) {
	files, _ := filepath.Glob("../../plugin/commands/*.md")
	if len(files) == 0 {
		t.Skip("plugin commands not readable")
	}
	call := regexp.MustCompile(`cli\.sh" ([a-z-]+)`)
	mainSrc, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	subcommand := func(sub string) bool {
		return bytes.Contains(mainSrc, []byte("\tcase \""+sub+"\":"))
	}
	seen := map[string]bool{}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".md")
		seen[name] = true
		src, _ := os.ReadFile(f)
		var subs []string
		have := map[string]bool{}
		for _, m := range call.FindAllStringSubmatch(string(src), -1) {
			if !have[m[1]] {
				have[m[1]] = true
				subs = append(subs, m[1])
			}
		}
		if _, no := typedCommandsUnavailable[name]; no {
			if _, also := typedCommands[name]; also {
				t.Errorf("%s is both runnable and unavailable", name)
			}
			continue
		}
		steps, ok := typedCommands[name]
		if !ok {
			t.Errorf("the plugin has the command %s and the daemon cannot run it from a Cowork prompt", name)
			continue
		}
		var want []string
		for _, st := range steps {
			want = append(want, st.sub)
			if !subcommand(st.sub) {
				t.Errorf("%s runs the subcommand %q, which main.go has no case for", name, st.sub)
			}
		}
		if strings.Join(subs, ",") != strings.Join(want, ",") {
			t.Errorf("%s runs %v for Claude Code and %v from a Cowork prompt", name, subs, want)
		}
	}
	var missing []string
	for name := range typedCommands {
		if !seen[name] {
			missing = append(missing, name)
		}
	}
	for name := range typedCommandsUnavailable {
		if !seen[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("the daemon knows commands the plugin does not have: %v", missing)
	}
}

func TestACommandRunsForTheSessionItWasTypedInAndNoOther(t *testing.T) {
	d := &Daemon{commands: helperRunner()}
	out := d.runTypedCommand(coworkHost{}, "raw-id", typedCommand{name: "self-status"})
	for _, want := range []string{"## Output of /cogmer:self-status", "ARGS:whoami", "HOST:cowork", "SESSION:raw-id"} {
		if !strings.Contains(out, want) {
			t.Errorf("the output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "the-session-that-started-the-daemon") || strings.Contains(out, "another") {
		t.Errorf("a command was given a session from the environment of the daemon:\n%s", out)
	}
}

func TestOnlyACommandWithOneStepTakesArguments(t *testing.T) {
	d := &Daemon{commands: helperRunner()}
	out := d.runTypedCommand(coworkHost{}, "s", typedCommand{name: "room-create", args: []string{"--show-only"}})
	if !strings.Contains(out, "ARGS:create --show-only") {
		t.Errorf("a command's arguments were not passed on:\n%s", out)
	}
	out = d.runTypedCommand(coworkHost{}, "s", typedCommand{name: "room-status", args: []string{"extra", "$(rm -rf)"}})
	for _, want := range []string{"### Rooms", "ARGS:rooms\n", "### Guests of this session's room", "ARGS:guests\n", "### Where to watch it", "ARGS:where\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("room-status lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "extra") || strings.Contains(out, "rm -rf") {
		t.Errorf("arguments reached a command that takes none:\n%s", out)
	}
	if strings.Index(out, "ARGS:rooms") > strings.Index(out, "ARGS:guests") || strings.Index(out, "ARGS:guests") > strings.Index(out, "ARGS:where") {
		t.Error("the three parts of room-status are out of order")
	}
}

// What a subcommand printed when it refused is the explanation, and a command that printed
// nothing, or could not run, still says so.
func TestACommandThatFailsOrIsSilentStillAnswers(t *testing.T) {
	d := &Daemon{commands: helperRunner()}
	refused := d.runTypedCommand(coworkHost{}, "s", typedCommand{name: "room-join", args: []string{"refuse"}})
	if !strings.Contains(refused, "this room cannot be joined") || strings.Contains(refused, "The command failed") {
		t.Errorf("a refusal was not relayed as the subcommand said it:\n%s", refused)
	}
	d.commands.exe = "/no/such/program"
	missing := d.runTypedCommand(coworkHost{}, "s", typedCommand{name: "self-status"})
	if !strings.Contains(missing, "The command failed") {
		t.Errorf("a command that could not run said nothing:\n%s", missing)
	}
	d.commands = helperRunner()
	quiet := d.runTypedCommand(coworkHost{}, "s", typedCommand{name: "room-join", args: []string{"silent"}})
	if !strings.Contains(quiet, "printed nothing") {
		t.Errorf("a silent command answered with %q", quiet)
	}
}

func TestACommandThatRunsTooLongIsStopped(t *testing.T) {
	d := &Daemon{commands: helperRunner()}
	d.commands.timeout = 300 * time.Millisecond
	start := time.Now()
	out := d.runTypedCommand(coworkHost{}, "s", typedCommand{name: "room-join", args: []string{"sleep"}})
	if took := time.Since(start); took > 3*time.Second {
		t.Errorf("the command was allowed %s with a limit of 300ms", took)
	}
	if !strings.Contains(out, "The command failed") && !strings.Contains(out, "ARGS:join sleep") {
		t.Errorf("a stopped command said nothing:\n%s", out)
	}
}

func TestUnavailableAndUnknownCommandsSayWhyAndRunNothing(t *testing.T) {
	d := &Daemon{commands: commandRunner{exe: "/no/such/program"}}
	for _, c := range []struct{ name, want string }{
		{"self-restart", "not available here"},
		{"self-stop", "not available here"},
		{"self-destruct", `no command named "self-destruct"`},
	} {
		out := d.runTypedCommand(coworkHost{}, "s", typedCommand{name: c.name})
		if !strings.Contains(out, c.want) || strings.Contains(out, "The command failed") {
			t.Errorf("%s answered %q, want it to say %q without trying to run anything", c.name, out, c.want)
		}
	}
}

func TestTheChildOfACommandHoldsNoOtherHostsSession(t *testing.T) {
	base := []string{"PATH=/bin", "HOME=/h", "CLAUDE_CODE_SESSION_ID=a", "CODEX_THREAD_ID=b", "COGMER_HOST=claude",
		"COGMER_COWORK_SESSION_ID=old", "COGMER_HOME=/state"}
	got := strings.Join(childEnvironment(base, coworkHost{}, "new"), "\n")
	for _, bad := range []string{"CLAUDE_CODE_SESSION_ID", "CODEX_THREAD_ID", "COGMER_HOST=claude", "COGMER_COWORK_SESSION_ID=old"} {
		if strings.Contains(got, bad) {
			t.Errorf("the child still holds %q:\n%s", bad, got)
		}
	}
	for _, want := range []string{"PATH=/bin", "HOME=/h", "COGMER_HOME=/state", "COGMER_HOST=cowork", "COGMER_COWORK_SESSION_ID=new"} {
		if !strings.Contains(got, want) {
			t.Errorf("the child lacks %q:\n%s", want, got)
		}
	}
}

// A command typed in Cowork needs no room, is no turn of the room, offers nothing, and
// the model's account of it is not published.
func TestACommandTypedInCoworkIsNotATurnOfTheRoom(t *testing.T) {
	m := testMembership(t)
	id := testIdentity(t)
	room, _ := m.CreateRoom(id.PeerID, false)
	for _, sid := range []string{"cowork:cw", "claude"} {
		if err := m.BindSession(sid, room.RoomID); err != nil {
			t.Fatal(err)
		}
	}
	d := &Daemon{id: id, members: m, commands: helperRunner()}
	d.versions.lookup = func(Host) string { return "3.0" }
	d.hostVersion(coworkHost{})
	d.background.Wait()
	t.Cleanup(d.closeStores)
	store, _ := d.storeFor(room.RoomID)

	post := func(endpoint string, payload any) map[string]any {
		t.Helper()
		b, _ := json.Marshal(payload)
		r := httptest.NewRequest(http.MethodPost, endpoint, bytes.NewReader(b))
		r.Header.Set(localGuardHeader, localGuardValue)
		w := httptest.NewRecorder()
		d.LocalRoutes().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", endpoint, w.Code, w.Body.String())
		}
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return out
	}
	// the colleague: another session of this peer, whose turn Cowork has not seen
	post("/hook/prompt", promptReq{SessionID: "claude", PromptID: "claude-1", Prompt: "FROM CLAUDE"})
	before, _ := store.ListRoom(room.RoomID)
	out := post("/hook/prompt", promptReq{Host: "cowork", SessionID: "cw", PromptID: "c-1", Prompt: "/cogmer:self-status"})
	ctx, _ := out["context"].(string)
	if !strings.Contains(ctx, "ARGS:whoami") || strings.Contains(ctx, "team-conversation") {
		t.Errorf("the command's context = %s", ctx)
	}
	if got := post("/hook/stop", stopReq{Host: "cowork", SessionID: "cw", PromptID: "c-1",
		LastAssistantMessage: "Your pairing string is ed25519:SECRET"}); got["stored"] != false {
		t.Errorf("the model's account of a command was stored: %v", got)
	}
	after, _ := store.ListRoom(room.RoomID)
	if len(after) != len(before) {
		t.Errorf("a command or the account of one became %d turn(s) of the room", len(after)-len(before))
	}
	for _, e := range after {
		if strings.Contains(e.Content, "SECRET") || strings.Contains(e.Content, "/cogmer:") {
			t.Errorf("the room holds %q", e.Content)
		}
	}
	// the colleague's turn was not spent on the command: the next ordinary prompt gets it
	next := post("/hook/prompt", promptReq{Host: "cowork", SessionID: "cw", PromptID: "p-2", Prompt: "an ordinary prompt"})
	if text, _ := next["context"].(string); !strings.Contains(text, "FROM CLAUDE") {
		t.Errorf("the colleague's turn was not offered at the next ordinary prompt: %s", text)
	}
	// and an ordinary prompt's reply is still captured
	if got := post("/hook/stop", stopReq{Host: "cowork", SessionID: "cw", PromptID: "p-2", LastAssistantMessage: "a reply"}); got["stored"] != true {
		t.Errorf("the reply to an ordinary prompt was not captured: %v", got)
	}
	// a session in no room can still run a command
	out = post("/hook/prompt", promptReq{Host: "cowork", SessionID: "alone", PromptID: "c-9", Prompt: "/cogmer:room-list"})
	if ctx, _ := out["context"].(string); !strings.Contains(ctx, "ARGS:rooms") {
		t.Errorf("a session in no room could not run a command: %s", ctx)
	}
}

// A host that runs its own commands has the prompt of one held back from the room and
// nothing run by the daemon, and the model's account of it is not stored (D-209).
func TestACommandTypedInAnotherHostIsHeldBackFromTheRoomAndNotRun(t *testing.T) {
	for _, c := range []struct {
		host, session, prompt, ordinary string
	}{
		{"", "claude", "/cogmer:room-create", "an ordinary prompt"},
		{"codex", "codex:cx", "$cogmer room-create", "an ordinary prompt"},
	} {
		t.Run(c.session, func(t *testing.T) {
			m := testMembership(t)
			id := testIdentity(t)
			room, _ := m.CreateRoom(id.PeerID, false)
			if err := m.BindSession(c.session, room.RoomID); err != nil {
				t.Fatal(err)
			}
			d := &Daemon{id: id, members: m, commands: commandRunner{exe: "/no/such/program"}}
			d.versions.lookup = func(Host) string { return "1.0" }
			t.Cleanup(d.closeStores)
			raw := strings.TrimPrefix(c.session, "codex:")
			post := func(endpoint string, payload any) map[string]any {
				t.Helper()
				b, _ := json.Marshal(payload)
				r := httptest.NewRequest(http.MethodPost, endpoint, bytes.NewReader(b))
				r.Header.Set(localGuardHeader, localGuardValue)
				w := httptest.NewRecorder()
				d.LocalRoutes().ServeHTTP(w, r)
				if w.Code != http.StatusOK {
					t.Fatalf("%s: %d %s", endpoint, w.Code, w.Body.String())
				}
				var out map[string]any
				_ = json.Unmarshal(w.Body.Bytes(), &out)
				return out
			}
			turn := func(prompt, id string) promptReq {
				return promptReq{Host: c.host, SessionID: raw, PromptID: id, TurnID: id, Prompt: prompt}
			}
			out := post("/hook/prompt", turn(c.prompt, "cmd-1"))
			if ctx, _ := out["context"].(string); ctx != "" {
				t.Errorf("a command typed in %s was answered by the daemon: %q", c.session, ctx)
			}
			post("/hook/stop", stopReq{Host: c.host, SessionID: raw, PromptID: "cmd-1", TurnID: "cmd-1",
				LastAssistantMessage: "Here is your invitation: SECRET-INVITE", TranscriptPath: "/nonexistent"})
			store, _ := d.storeFor(room.RoomID)
			if events, _ := store.ListRoom(room.RoomID); len(events) != 0 {
				t.Errorf("a command or the account of one became %d turn(s) of the room: %q", len(events), events[0].Content)
			}
			post("/hook/prompt", turn(c.ordinary, "p-2"))
			if events, _ := store.ListRoom(room.RoomID); len(events) != 1 || events[0].Content != c.ordinary {
				t.Errorf("an ordinary prompt was not captured as before: %v", events)
			}
		})
	}
}

func TestOnlyAPromptThatIsACommandWaitsLongerAtTheBridge(t *testing.T) {
	b := &bridge{deadline: 3 * time.Second, commandDeadline: 25 * time.Second}
	for _, c := range []struct {
		tool string
		args map[string]any
		want time.Duration
	}{
		{"prompt", map[string]any{"prompt": "/cogmer:room-create"}, 25 * time.Second},
		{"prompt", map[string]any{"prompt": "  /cogmer:self-status now"}, 25 * time.Second},
		{"prompt", map[string]any{"prompt": "an ordinary prompt"}, 3 * time.Second},
		{"prompt", map[string]any{"prompt": "run /cogmer:room-create please"}, 3 * time.Second},
		{"prompt", map[string]any{}, 3 * time.Second},
		{"stop", map[string]any{"prompt": "/cogmer:room-create", "last_assistant_message": "/cogmer:x"}, 3 * time.Second},
	} {
		if got := b.deadlineFor(c.tool, c.args); got != c.want {
			t.Errorf("%s %v waits %s, want %s", c.tool, c.args, got, c.want)
		}
	}
	if bridgeCommandDeadline <= typedCommandTimeout {
		t.Error("the bridge gives up on a command before the daemon does")
	}
}

// A command file runs cli.sh through bash, so a plugin copied without the execute bit
// still works, and each shell line is allowed by a pattern naming that exact subcommand.
// A line that does not start with the pattern's prefix is not allowed, and a pattern
// wider than its line allows more than the command runs.
func TestCommandFilesRunCliThroughBashAndAllowOnlyTheirOwnSubcommands(t *testing.T) {
	files, _ := filepath.Glob("../../plugin/commands/*.md")
	if len(files) < 10 {
		t.Skip("plugin commands not readable")
	}
	const prefix = `bash "${CLAUDE_PLUGIN_ROOT}/cli.sh" `
	line := regexp.MustCompile("(?m)^!`(.*)`$")
	for _, f := range files {
		src, _ := os.ReadFile(f)
		front, body, ok := splitFrontmatter(string(src))
		if !ok {
			t.Errorf("%s has no frontmatter", f)
			continue
		}
		allowed := map[string]bool{}
		for _, l := range strings.Split(front, "\n") {
			rest, found := strings.CutPrefix(l, "allowed-tools: ")
			if !found {
				continue
			}
			for _, p := range strings.Split(rest, ", ") {
				allowed[p] = true
			}
		}
		lines := line.FindAllStringSubmatch(body, -1)
		if len(lines) == 0 {
			t.Errorf("%s runs no shell line", f)
		}
		used := map[string]bool{}
		for _, m := range lines {
			cmd, ok := strings.CutPrefix(m[1], prefix)
			if !ok {
				t.Errorf("%s runs %q, which does not run cli.sh through bash", f, m[1])
				continue
			}
			sub, _, _ := strings.Cut(cmd, " ")
			pat := "Bash(" + prefix + sub + ":*)"
			used[pat] = true
			if !allowed[pat] {
				t.Errorf("%s runs %q and does not allow %s", f, m[1], pat)
			}
		}
		for p := range allowed {
			if !used[p] {
				t.Errorf("%s allows %s, which none of its lines runs", f, p)
			}
		}
	}
}
