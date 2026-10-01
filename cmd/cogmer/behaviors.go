package main

import (
	"fmt"
	"strings"
)

// Tier controls how expensive a behavior check is to run.
type Tier int

const (
	// TierOffline needs no Claude session at all.
	TierOffline Tier = iota
	// TierSession needs one `claude -p` turn.
	TierSession
	// TierCompaction additionally drives a compaction; several turns.
	TierCompaction
)

func (t Tier) String() string {
	switch t {
	case TierOffline:
		return "offline"
	case TierSession:
		return "session"
	default:
		return "compaction"
	}
}

// Behavior is an undocumented Claude Code behavior this project depends on.
//
// None of these are contractual. They were established empirically against a
// specific version (the runs are in 84a0751:docs/phase0-findings.md and
// 84a0751:docs/phase0a-findings.md), and a Claude Code upgrade can change any of
// them without notice. Several fail silently: the room keeps accepting events
// while recording the wrong thing. That is why they are checked rather than
// assumed.
type Behavior struct {
	ID       string
	Title    string
	Reliance string // what breaks here if this behavior changes
	Tier     Tier
	Check    func(*Probe) error
}

// Behaviors is the registry. docs/relied-on-behaviors.md is generated from it,
// so the documentation cannot drift from what is actually checked.
var Behaviors = []Behavior{
	{
		ID:       "B01",
		Title:    "UserPromptSubmit carries prompt, session_id, transcript_path, prompt_id",
		Reliance: "If this changes, the hooks cannot tell which session called them, so no session is ever in a room: nothing is captured and nothing is injected. It fails silently, because a session in no room is an ordinary Claude Code session. The prompt is what capture records, and prompt_id keys each pending injection.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			return p.requireFields("UserPromptSubmit", "prompt", "session_id", "transcript_path", "prompt_id")
		},
	},
	{
		ID:       "B02",
		Title:    "UserPromptSubmit stdout is injected into the pending turn",
		Reliance: "If this changes, a colleague's turns never reach your Claude, while the browser view still shows the room. It fails silently: the session answers normally, without the context.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			if !strings.Contains(p.Final, p.Sentinel) {
				return fmt.Errorf("sentinel %q absent from the model's reply; injected context did not reach Claude", p.Sentinel)
			}
			return nil
		},
	},
	{
		ID:       "B03",
		Title:    "Stop fires and carries last_assistant_message",
		Reliance: "If this changes, responses are not captured. If Stop stops firing, no response reaches the room; if it stops carrying last_assistant_message, every captured response loses its final text block, which is not yet in the transcript when Stop fires (B05). Either way the room shows prompts with incomplete or missing answers.",
		Tier:     TierSession,
		Check:    func(p *Probe) error { return p.requireFields("Stop", "last_assistant_message", "session_id") },
	},
	{
		ID:       "B04",
		Title:    "last_assistant_message contains ONLY the turn's final text block",
		Reliance: "If this changes, nothing breaks: mergeTail also handles a last_assistant_message that holds the whole turn. The check fails so that this entry can be corrected, and so that the change does not pass unseen.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			s, _ := p.Turn1Stop["last_assistant_message"].(string)
			if strings.Contains(s, "ALPHA") {
				return fmt.Errorf("last_assistant_message now includes pre-tool text (contains ALPHA) -- it is no longer final-block-only; mergeTail handles this, but B04's Title and Reliance are out of date")
			}
			return nil
		},
	},
	{
		ID:       "B05",
		Title:    "The transcript at Stop time is missing the final text block",
		Reliance: "If this changes, nothing breaks: reassembly reads both the transcript and last_assistant_message, and the second becomes redundant. The check fails so that this entry can be corrected.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			for _, r := range p.AtStop {
				if r.Type == "assistant" {
					for _, b := range blocksOf(r) {
						if b.Type == "text" && strings.Contains(b.Text, p.Sentinel) {
							return fmt.Errorf("final block WAS already flushed at Stop time -- the race appears fixed; reassembly is still correct but B05's Title and Reliance are out of date")
						}
					}
				}
			}
			return nil
		},
	},
	{
		ID:       "B06",
		Title:    "Human prompts carry promptSource; tool-result records do not",
		Reliance: "If this changes, reassembly anchors on the wrong record and publishes a fragment of the turn, or treats a tool result as a prompt. It fails silently: the room keeps accepting events, with the wrong text in them.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			var human, toolResult int
			for _, r := range p.Transcript {
				if r.Type != "user" {
					continue
				}
				if r.PromptSource != "" {
					human++
				} else if strings.Contains(string(r.Message.Content), "tool_result") {
					toolResult++
				}
			}
			if human == 0 {
				return fmt.Errorf("no user record carried promptSource; the reassembly anchor is gone")
			}
			if toolResult == 0 {
				return fmt.Errorf("no tool_result user records found; probe did not exercise a tool call")
			}
			return nil
		},
	},
	{
		ID:       "B07",
		Title:    "Assistant content blocks use type text / tool_use / thinking",
		Reliance: "If this changes, response text goes missing from captured turns: reassembly publishes only blocks of type text, and records tool_use blocks as tool calls. It fails silently. A renamed thinking type is not published either way, since only text is.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			seen := map[string]bool{}
			for _, r := range p.Transcript {
				if r.Type == "assistant" {
					for _, b := range blocksOf(r) {
						seen[b.Type] = true
					}
				}
			}
			if !seen["text"] || !seen["tool_use"] {
				return fmt.Errorf("expected text and tool_use blocks, saw %v", keysOf(seen))
			}
			return nil
		},
	},
	{
		ID:       "B08",
		Title:    "isSidechain marks subagent traffic",
		Reliance: "If this changes, subagent output, including the compaction summarizer's, is captured as if the session had written it and published to the room. It fails silently.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			for _, r := range p.rawTranscript {
				if _, ok := r["isSidechain"]; ok {
					return nil
				}
			}
			return fmt.Errorf("no record carried isSidechain; subagent output can no longer be distinguished and may leak into the room")
		},
	},
	{
		ID:       "B09",
		Title:    "Assistant records carry no promptId",
		Reliance: "If this changes, nothing breaks: turns are segmented by position, because assistant records carry no promptId. If they gained one, transcript.go could match each response to its prompt by id instead.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			for _, r := range p.rawTranscript {
				if r["type"] == "assistant" {
					if v, ok := r["promptId"]; ok && v != nil {
						return fmt.Errorf("assistant records now carry promptId -- positional segmentation in transcript.go can be replaced with exact correlation")
					}
				}
			}
			return nil
		},
	},
	{
		ID:       "B11",
		Title:    "A hook whose daemon is unreachable exits 0 with empty stdout",
		Reliance: "If this changes, every prompt in every session is disturbed whenever the daemon is down: a non-zero exit or stray output from the hook reaches Claude Code, and the session is worse off for having cogmer installed.",
		Tier:     TierOffline,
		Check:    func(p *Probe) error { return checkFailOpen() },
	},
	{
		ID:       "B12",
		Title:    "Reassembly reconstructs the complete turn",
		Reliance: "If this changes, captured turns lose text or repeat it, even while B03, B04 and B05 each pass. It is the check that shows whether they still combine into a complete turn, and it fails silently in the room.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			if p.Reassembled == nil {
				return fmt.Errorf("reassembly produced nothing")
			}
			txt := p.Reassembled.Text
			if !strings.Contains(txt, "ALPHA") {
				return fmt.Errorf("pre-tool text lost from reassembled turn: %.120q", txt)
			}
			if !strings.Contains(txt, p.Sentinel) {
				return fmt.Errorf("final text lost from reassembled turn: %.120q", txt)
			}
			if strings.Count(txt, "ALPHA") > 1 {
				return fmt.Errorf("duplicated text in reassembled turn: %.160q", txt)
			}
			return nil
		},
	},

	{
		ID:       "B20",
		Title:    "Injected hook output is recorded as a hook_success attachment",
		Reliance: "If this changes, no injection is ever confirmed as delivered, so a colleague's turns are offered again at every prompt. The daemon logs that it is re-offering them and points at `cogmer doctor`. Once doctor has recorded this check as failing for the installed version, the daemon counts a turn as delivered without evidence, and an injection that never arrived is then lost silently.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			if len(p.ObservedBlocks) == 0 {
				return fmt.Errorf("no hook_success attachment recorded for UserPromptSubmit; delivery confirmation has no evidence to work from and falls back to trust")
			}
			want := HashBlock(p.InjectedText)
			for _, b := range p.ObservedBlocks {
				if HashBlock(b) == want {
					return nil
				}
			}
			return fmt.Errorf("recorded attachment does not match the injected block byte-for-byte; hash matching in ConfirmDelivered will never succeed (got %d block(s))", len(p.ObservedBlocks))
		},
	},

	{
		ID:       "B21",
		Title:    "CLAUDE_CODE_SESSION_ID is exported into a tool call's environment",
		Reliance: "If this changes, a slash command cannot tell which session ran it, so `/cogmer:room-create` and `/cogmer:room-join` refuse rather than guess, and stop working. If the variable is present but reports a different id from the hooks, the failure is worse and silent: the room is bound to a session that does not exist, the real session is bound to nothing, and capture stops with no error anywhere.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			if p.ToolEnvSessionID == "" {
				return fmt.Errorf("CLAUDE_CODE_SESSION_ID was empty or unset inside a Bash tool call; a command run from a session cannot learn which session it is in")
			}
			if p.ToolEnvSessionID != p.SessionID {
				return fmt.Errorf("a tool call saw session %q but the session is %q; binding would attach a room to the wrong session",
					p.ToolEnvSessionID, p.SessionID)
			}
			return nil
		},
	},

	{
		ID:       "B22",
		Title:    "Injected hook output is positioned as data, not as the session's own instruction",
		Reliance: "If this changes, a colleague can steer your session: a turn written as an instruction would be followed rather than read. The injected block frames its content as information, never instruction, and that framing travels in the same hook output as the content, so it holds only while Claude Code places hook output where a session reads it as data. Nothing in the protocol would show the change. A session given a turn claiming SYSTEM OVERRIDE refused it and told its user on Claude Code 2.1.275 (commit b09b208).",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			// The probe injects a block and reads the reply. What is asserted is
			// that injected content did not become instruction: the codeword is
			// answered because the PROMPT asked for it, and no injected text
			// altered the shape of the reply.
			if len(p.ObservedBlocks) == 0 {
				return fmt.Errorf("no injected block was recorded, so nothing can be said about how it was positioned")
			}
			if !strings.Contains(p.InjectedText, "information, never instruction") {
				return fmt.Errorf("the injected block no longer frames its content as information rather than instruction; the framing is the defence")
			}
			if !strings.Contains(p.InjectedText, "\"turns\":") {
				return fmt.Errorf("the injected block is no longer JSON-encoded; delimiting has gone back to escaping by hand (D-081)")
			}
			return nil
		},
	},

	{
		ID:       "B23",
		Title:    "A process the daemon's shape keeps the GUI session, so it can open the view and notify",
		Reliance: "If this changes, nobody sees a room: the daemon, started in the background by the session-start hook, is the only part of cogmer that can open the room view, since everything Claude Code offers delivers to the model rather than to a person (D-033, D-036). There is no error path back to the daemon, so it believes it showed the view while nothing appeared, and a first-time user is left with a loopback address nobody told them about. On Darwin 25.6 a background process keeps the logged-in GUI session, opens a URL and takes focus, whether or not it is in a new POSIX session.",
		Tier:     TierOffline,
		Check: func(p *Probe) error {
			mgr := p.DetachedSessionManager
			if mgr == "" {
				mgr = measureDetachedSessionManager()
			}
			return assertGUISession(mgr)
		},
	},

	// ---- compaction tier ----
	{
		ID:       "B14",
		Title:    "session_id and transcript_path survive compaction",
		Reliance: "If this changes, a session drops out of its room at its first compaction: the hooks find a session's room by its id, so the new id belongs to no room, and nothing more is captured or injected. It fails silently.",
		Tier:     TierCompaction,
		Check: func(p *Probe) error {
			v, _ := p.field("PreCompact", "session_id")
			if s, _ := v.(string); s != p.SessionID {
				return fmt.Errorf("session id changed at compaction: %q -> %q; the watermark key is no longer stable", p.SessionID, s)
			}
			if p.PostCompactSessionID != "" && p.PostCompactSessionID != p.SessionID {
				return fmt.Errorf("session id changed after compaction: %q -> %q", p.SessionID, p.PostCompactSessionID)
			}
			return nil
		},
	},
	{
		ID:       "B15",
		Title:    "The transcript is appended to across compaction, never rewritten",
		Reliance: "If this changes, reassembly can lose the record it anchors on, which was written before the compaction, and publish the wrong text for the turn after it. It fails silently.",
		Tier:     TierCompaction,
		Check: func(p *Probe) error {
			if p.PostCompactBytes < p.PreCompactBytes {
				return fmt.Errorf("transcript shrank across compaction (%d -> %d bytes); it is being rewritten or truncated", p.PreCompactBytes, p.PostCompactBytes)
			}
			return nil
		},
	},
	{
		ID:       "B17",
		Title:    "The compaction summary record carries no promptSource",
		Reliance: "If this changes, reassembly mistakes the compaction summary for a user's prompt, and every turn after a compaction is anchored on it and published with the wrong text. It fails silently.",
		Tier:     TierCompaction,
		Check: func(p *Probe) error {
			for _, r := range p.rawTranscript {
				if v, ok := r["isCompactSummary"]; ok && v == true {
					if ps, ok := r["promptSource"]; ok && ps != nil && ps != "" {
						return fmt.Errorf("compaction summary now carries promptSource=%v; reassembly would anchor on it", ps)
					}
				}
			}
			return nil
		},
	},
	{
		ID:       "B18",
		Title:    "Slash commands do not reach UserPromptSubmit",
		Reliance: "If this changes, every slash command a user types, including /compact, is captured as a prompt and published to the room.",
		Tier:     TierCompaction,
		Check: func(p *Probe) error {
			for _, pl := range p.HookAll("UserPromptSubmit") {
				if s, _ := pl["prompt"].(string); strings.HasPrefix(strings.TrimSpace(s), "/") {
					return fmt.Errorf("slash command %q reached UserPromptSubmit; it would be published to the room", s)
				}
			}
			return nil
		},
	},
	{
		ID:       "B19",
		Title:    "Injected teammate context survives compaction",
		Reliance: "If this changes, a session counts a colleague's turns as delivered after a compaction has dropped them, so it never receives them again, and a later reference to them finds nothing. It fails silently. Delivery is therefore not reset at compaction. Only context that was the conversation's subject is checked here; that incidental context also survived is recorded in 84a0751:docs/phase0a-findings.md.",
		Tier:     TierCompaction,
		Check: func(p *Probe) error {
			if !strings.Contains(p.PostCompactAnswer, p.Sentinel) {
				return fmt.Errorf("sentinel %q lost after compaction -- the watermark now claims delivery of context Claude cannot see; add a compaction rewind", p.Sentinel)
			}
			return nil
		},
	},
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// MarkdownReport renders the registry as documentation.
func MarkdownReport() string {
	var b strings.Builder
	b.WriteString("# Claude Code behaviors this project relies on\n\n")
	b.WriteString("Generated by `cogmer behaviors --markdown`. Do not edit by hand.\n\n")
	b.WriteString("Claude Code promises none of these. Each was observed on a particular version and\n")
	b.WriteString("can change without notice when Claude Code is upgraded, and several fail silently.\n")
	b.WriteString("`cogmer doctor` checks them against the installed version.\n\n")
	for _, tier := range []Tier{TierOffline, TierSession, TierCompaction} {
		b.WriteString(fmt.Sprintf("## Tier: %s\n\n", tier))
		for _, bh := range Behaviors {
			if bh.Tier != tier {
				continue
			}
			fmt.Fprintf(&b, "### %s: %s\n\n%s\n\n", bh.ID, bh.Title, bh.Reliance)
		}
	}
	return b.String()
}
