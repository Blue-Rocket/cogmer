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
// specific version (see docs/phase0-findings.md and docs/phase0a-findings.md),
// and a Claude Code upgrade can change any of them without notice. Several fail
// SILENTLY -- the room keeps accepting events while quietly recording the wrong
// thing -- which is why they are checked rather than assumed.
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
		Reliance: "Prompt capture and session keying. Without session_id the delivery watermark cannot be keyed, and every session would re-inject the whole room.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			return p.requireFields("UserPromptSubmit", "prompt", "session_id", "transcript_path", "prompt_id")
		},
	},
	{
		ID:       "B02",
		Title:    "UserPromptSubmit stdout is injected into the pending turn",
		Reliance: "The entire cross-session context feature. If this stops working, teammates become invisible to each other's Claude and only the human-readable room still functions.",
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
		Reliance: "Half of turn reassembly. Without it the final text block of every turn is lost, because it is not yet on disk when Stop fires.",
		Tier:     TierSession,
		Check:    func(p *Probe) error { return p.requireFields("Stop", "last_assistant_message", "session_id") },
	},
	{
		ID:       "B04",
		Title:    "last_assistant_message contains ONLY the turn's final text block",
		Reliance: "mergeTail's append semantics. If Claude Code widens this to the whole turn, naive appending duplicates every pre-tool text block. mergeTail already tolerates the change; this check exists to report it rather than let it pass unnoticed.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			s, _ := p.Turn1Stop["last_assistant_message"].(string)
			if strings.Contains(s, "ALPHA") {
				return fmt.Errorf("last_assistant_message now includes pre-tool text (contains ALPHA) -- it is no longer final-block-only; mergeTail handles this, but docs/phase0-findings.md §3 is out of date")
			}
			return nil
		},
	},
	{
		ID:       "B05",
		Title:    "The transcript at Stop time is missing the final text block",
		Reliance: "Why reassembly unions two sources instead of just reading the transcript. If this race disappears, the union becomes redundant but stays correct.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			for _, r := range p.AtStop {
				if r.Type == "assistant" {
					for _, b := range blocksOf(r) {
						if b.Type == "text" && strings.Contains(b.Text, p.Sentinel) {
							return fmt.Errorf("final block WAS already flushed at Stop time -- the race appears fixed; reassembly is still correct but docs/phase0-findings.md §3 is out of date")
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
		Reliance: "Turn segmentation and prompt filtering. If tool-result records gained a promptSource, reassembly would anchor on the wrong record and publish a fragment of the turn.",
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
		Reliance: "Response extraction and the rule that thinking is never published to the room.",
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
		Reliance: "Keeping subagent output -- including the compaction summarizer's -- out of the shared room.",
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
		Reliance: "Why segmentation is positional. If assistant records gained a promptId, segmentation could become exact -- an improvement worth taking, not a break.",
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
		ID:       "B10",
		Title:    "PostToolUse carries tool_name and tool_response",
		Reliance: "Tool-activity metadata on stored events.",
		Tier:     TierSession,
		Check: func(p *Probe) error {
			return p.requireFields("PostToolUse", "tool_name", "tool_response", "tool_use_id")
		},
	},
	{
		ID:       "B11",
		Title:    "A hook whose daemon is unreachable exits 0 with empty stdout",
		Reliance: "The guarantee that Claude Code keeps working when collaboration is down. A non-zero exit or stray stdout here would corrupt every prompt.",
		Tier:     TierOffline,
		Check:    func(p *Probe) error { return checkFailOpen() },
	},
	{
		ID:       "B12",
		Title:    "Reassembly reconstructs the complete turn",
		Reliance: "End-to-end proof that B03/B04/B05 still compose. This is the check that matters if the individual ones drift.",
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
		Reliance: "Delivery confirmation (D-014). The daemon marks teammate events delivered only once it observes the injected block in the transcript. If this record disappears or changes shape, delivery falls back to trusting that the turn carried it -- degraded, not broken, but the guarantee weakens silently.",
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
		Reliance: "Commands run from inside a session knowing which session they are in. A slash command shells out to this binary, and the binary reads this variable to bind a room to the session that asked for it. If the variable disappears, a command invoked from a session cannot tell itself apart from one typed at a terminal, and the safe response -- refusing rather than guessing -- means `/team-create` and `/team-join` stop working. If it is present but reports a DIFFERENT id than the hooks report, the failure is worse and silent: the room binds to a session that does not exist, the real session binds to nothing, and capture stops with no error anywhere.",
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
		Reliance: "The whole defence against a hostile teammate turn. Room content is injected as hook stdout, and every word framing that content as data rather than instruction travels in the same blob as the content itself. Anthropic's guidance says instructions placed alongside untrusted content may be discounted, because a model is right to be sceptical of that position -- so if this attachment is treated as tool-result-like, our framing is discounted with the payload it frames. Measured to hold on 2.1.275: a session given a turn claiming SYSTEM OVERRIDE quoted it, named it an injection attempt, refused it, and told its user, including refusing the instruction not to mention it. If that stops holding, the room becomes a channel for one peer to steer another's session, and nothing in the protocol would show it.",
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
		Reliance: "How a person ever sees a room at all. The daemon is started detached by the session-start hook and is the only component that can put something in front of somebody: it opens the room view when a first pairing needs looking at, and raises a notification afterwards. Claude Code cannot do either, because every extension point it offers delivers to the model rather than to the person (D-033, D-036). This works only while a background-launched process still belongs to the logged-in GUI session -- measured on Darwin 25.6, where such a process reports the Aqua session manager, opens a URL, and takes focus, whether or not it was placed in a new POSIX session. If macOS ever withholds GUI-session access from background processes, there is no error path back to the daemon: it will go on believing it showed somebody the view while nothing at all appeared on screen, and a first-time user is left holding a loopback address nobody ever told them.",
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
		ID:       "B13",
		Title:    "PreCompact fires and reports its trigger",
		Reliance: "Detecting compaction at all. Without it, compaction is only visible by reading the transcript after the fact.",
		Tier:     TierCompaction,
		Check:    func(p *Probe) error { return p.requireFields("PreCompact", "trigger", "session_id", "transcript_path") },
	},
	{
		ID:       "B14",
		Title:    "session_id and transcript_path survive compaction",
		Reliance: "The delivery watermark is keyed on session ID. A new ID per compaction would silently re-inject the entire room every time.",
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
		Reliance: "Reassembly anchors on a record written before the boundary. A rewritten transcript could remove the anchor.",
		Tier:     TierCompaction,
		Check: func(p *Probe) error {
			if p.PostCompactBytes < p.PreCompactBytes {
				return fmt.Errorf("transcript shrank across compaction (%d -> %d bytes); it is being rewritten or truncated", p.PreCompactBytes, p.PostCompactBytes)
			}
			return nil
		},
	},
	{
		ID:       "B16",
		Title:    "Compaction writes a compact_boundary marker and a summary record",
		Reliance: "Detecting a compaction from the transcript alone, without having caught the hook.",
		Tier:     TierCompaction,
		Check: func(p *Probe) error {
			var boundary, summary bool
			for _, r := range p.rawTranscript {
				if r["type"] == "system" && r["subtype"] == "compact_boundary" {
					boundary = true
				}
				if v, ok := r["isCompactSummary"]; ok && v == true {
					summary = true
				}
			}
			if !boundary || !summary {
				return fmt.Errorf("compact_boundary=%v isCompactSummary=%v; compaction is no longer self-describing in the transcript", boundary, summary)
			}
			return nil
		},
	},
	{
		ID:       "B17",
		Title:    "The compaction summary record carries no promptSource",
		Reliance: "Reassembly must not mistake the summary for a human prompt. If it did, every post-compaction turn would anchor on the summary and publish the wrong text.",
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
		Reliance: "Keeping /compact and other slash commands out of the room. If this changes, every slash command becomes a published conversation event.",
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
		Reliance: "Why no watermark rewind exists. If this fails, sessions are marked as having incorporated context they can no longer see, and the referent is lost silently. Remediation is documented in docs/phase0a-findings.md §5.",
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
	b.WriteString("Generated by `claude-team behaviors --markdown`. Do not edit by hand.\n\n")
	b.WriteString("None of these are contractual. Each was established empirically and can change\n")
	b.WriteString("without notice on a Claude Code upgrade; several fail silently. `claude-team doctor`\n")
	b.WriteString("re-verifies them against the installed version.\n\n")
	for _, tier := range []Tier{TierOffline, TierSession, TierCompaction} {
		b.WriteString(fmt.Sprintf("## Tier: %s\n\n", tier))
		for _, bh := range Behaviors {
			if bh.Tier != tier {
				continue
			}
			fmt.Fprintf(&b, "### %s — %s\n\n**Relied on for:** %s\n\n", bh.ID, bh.Title, bh.Reliance)
		}
	}
	return b.String()
}
