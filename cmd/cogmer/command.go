package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// A command typed in a Cowork session arrives at the daemon as the text of the prompt
// (D-207). The sandbox the session runs in has no daemon and no binary to call, so the
// daemon runs the command itself, as the subcommands of this binary that the plugin's
// command files call for the other hosts, and hands the output back as context for the
// model to relay. The output is not a turn of the room.

// typedCommandRunner is a host whose commands the daemon runs from the text of the prompt.
// The other hosts run their own commands in the session, and the daemon only holds the
// prompt back from the room (D-209).
type typedCommandRunner interface {
	RunsTypedCommands() bool
}

// commandStep is one subcommand of the binary, with the heading its output carries when a
// command runs several.
type commandStep struct {
	title string
	sub   string
}

// typedCommands maps each command a user types to the subcommands that carry it out. The
// plugin's command files call the same ones through plugin/cli.sh, and a test compares
// the two so that a command added to one is not missing from the other.
var typedCommands = map[string][]commandStep{
	"room-create":    {{"", "create"}},
	"room-join":      {{"", "join"}},
	"room-leave":     {{"", "leave"}},
	"room-invite":    {{"", "invite"}},
	"room-revoke":    {{"", "revoke"}},
	"room-status":    {{"Rooms", "rooms"}, {"Guests of this session's room", "guests"}, {"Where to watch it", "where"}},
	"room-log":       {{"", "log"}},
	"room-list":      {{"", "rooms"}},
	"room-conflicts": {{"", "conflicts"}},
	"peer-list":      {{"", "peers"}},
	"peer-forget":    {{"", "forget"}},
	"peer-pair":      {{"", "pair"}},
	"self-name":      {{"", "name"}},
	"self-status":    {{"", "whoami"}},
}

// typedCommandsUnavailable are the commands that cannot run from the daemon, with what a
// user is told. Each acts on the daemon that would be running it.
var typedCommandsUnavailable = map[string]string{
	"self-stop":    "stops the daemon that would be running it",
	"self-restart": "restarts the daemon that would be running it",
}

const (
	// typedCommandTimeout is the longest a command may run. The bridge waits a little
	// longer than this for the daemon's answer.
	typedCommandTimeout = 20 * time.Second
	// typedCommandMaxOutput bounds what is handed to the model.
	typedCommandMaxOutput = 64 << 10
	// typedCommandMaxArgs bounds the arguments of a command.
	typedCommandMaxArgs = 4 << 10
	// commandPromptsKept bounds how many prompts that were commands the daemon remembers.
	commandPromptsKept = 256
)

type typedCommand struct {
	name string
	args []string
}

// parseTypedCommand reads a prompt that begins with the host's command prefix. The first
// line is the command and its arguments, split on white space and never interpreted by a
// shell.
func parseTypedCommand(prompt, prefix string) (typedCommand, bool) {
	text := strings.TrimSpace(prompt)
	if prefix == "" || !strings.HasPrefix(text, prefix) {
		return typedCommand{}, false
	}
	if i := strings.IndexAny(text, "\r\n"); i >= 0 {
		text = text[:i]
	}
	fields := strings.Fields(strings.TrimPrefix(text, prefix))
	if len(fields) == 0 {
		return typedCommand{}, false
	}
	return typedCommand{name: fields[0], args: fields[1:]}, true
}

// commandRunner runs one subcommand of the binary as a child of the daemon.
type commandRunner struct {
	exe     string   // this binary
	pre     []string // arguments before the subcommand; a test uses them to reach a helper
	timeout time.Duration
	environ func() []string // os.Environ when nil
}

// childEnvironment is what a command runs with: the daemon's environment without any
// host's session, then the host and the session this command was typed in. A daemon that
// a Claude Code or Codex session started holds that session's variable, and a command
// that read it would act on the wrong session (D-167).
func childEnvironment(base []string, host Host, rawSession string) []string {
	drop := map[string]bool{"COGMER_HOST": true}
	for _, h := range hosts {
		if env := h.SessionEnv(); env != "" {
			drop[env] = true
		}
	}
	var out []string
	for _, kv := range base {
		if name, _, ok := strings.Cut(kv, "="); ok && drop[name] {
			continue
		}
		out = append(out, kv)
	}
	out = append(out, "COGMER_HOST="+host.Name())
	if env := host.SessionEnv(); env != "" {
		out = append(out, env+"="+rawSession)
	}
	return out
}

func (r commandRunner) run(ctx context.Context, host Host, rawSession, sub string, args []string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	argv := append(append(append([]string{}, r.pre...), sub), args...)
	cmd := exec.CommandContext(ctx, r.exe, argv...)
	base := os.Environ()
	if r.environ != nil {
		base = r.environ()
	}
	cmd.Env = childEnvironment(base, host, rawSession)
	cmd.Stdin = nil
	var out bytes.Buffer
	w := &limitedWriter{w: &out, left: typedCommandMaxOutput}
	cmd.Stdout, cmd.Stderr = w, w // one writer, so the two streams are not written at once
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	return out.String(), err
}

// limitedWriter keeps the first bytes written and discards the rest.
type limitedWriter struct {
	w    *bytes.Buffer
	left int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	n := len(p)
	if l.left <= 0 {
		return n, nil
	}
	if len(p) > l.left {
		p = p[:l.left]
	}
	l.w.Write(p)
	l.left -= len(p)
	return n, nil
}

// runTypedCommand runs what a user typed and returns the text for the model. Whatever
// goes wrong is said in that text, since a command a person typed has to answer.
func (d *Daemon) runTypedCommand(host Host, rawSession string, cmd typedCommand) string {
	heading := "## Output of " + typedCommandName(host, cmd.name)
	if why, ok := typedCommandsUnavailable[cmd.name]; ok {
		return fmt.Sprintf("%s\n\nThis command is not available here. It %s. Run it from a Claude Code or Codex session on this machine.",
			heading, why)
	}
	steps, ok := typedCommands[cmd.name]
	if !ok {
		return fmt.Sprintf("%s\n\ncogmer has no command named %q.", heading, cmd.name)
	}
	if len(strings.Join(cmd.args, " ")) > typedCommandMaxArgs {
		return heading + "\n\nThe arguments are too long."
	}
	runner := d.commands
	if runner.exe == "" {
		exe, err := os.Executable()
		if err != nil {
			return heading + "\n\nThis daemon cannot find its own program to run the command."
		}
		runner.exe = exe
	}
	if runner.timeout == 0 {
		runner.timeout = typedCommandTimeout
	}
	var parts []string
	for _, st := range steps {
		// Only a command with one step takes arguments. room-status runs three subcommands
		// that take none, as its command file does.
		var args []string
		if len(steps) == 1 {
			args = cmd.args
		}
		out, err := runner.run(context.Background(), host, rawSession, st.sub, args)
		out = strings.TrimSpace(out)
		switch {
		case err != nil && out == "":
			out = fmt.Sprintf("The command failed: %v.", err)
		case err != nil:
			// What a failing subcommand printed is the explanation, and the exit status adds
			// nothing a person can use.
		case out == "":
			out = "The command printed nothing."
		}
		if st.title != "" {
			out = "### " + st.title + "\n\n" + out
		}
		parts = append(parts, out)
	}
	return heading + "\n\n" + strings.Join(parts, "\n\n")
}

// typedCommandName writes a command's name as the user typed it.
func typedCommandName(host Host, name string) string {
	return host.CommandPrefix() + name
}

// commandPrompts remembers the prompts that were commands, so that the stop call of the
// same turn does not publish the model's account of one: it can hold a pairing string, an
// invitation or a room's guests, which the user typed a command to see and did not mean
// to share.
type commandPrompts struct {
	mu    sync.Mutex
	order []string
	seen  map[string]bool
}

func (c *commandPrompts) note(session, prompt string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.seen == nil {
		c.seen = map[string]bool{}
	}
	key := session + "\x00" + prompt
	if c.seen[key] {
		return
	}
	c.seen[key] = true
	c.order = append(c.order, key)
	if len(c.order) > commandPromptsKept {
		delete(c.seen, c.order[0])
		c.order = c.order[1:]
	}
}

func (c *commandPrompts) has(session, prompt string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.seen[session+"\x00"+prompt]
}
