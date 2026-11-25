# D-295 — A turn with no transcript path is recorded from the Stop hook's last message

**Date:** 2026-11-25 · **Status:** active · **Areas:** capture, hosts

**Decision.** When the Stop hook of Claude Code sends an empty `transcript_path`, the daemon records the turn's reply as the hook's `last_assistant_message`, trimmed, and nothing else: the final text block, with no tool calls. A path that is given and is not a regular `.jsonl` file of at most 1 GiB is still refused, and its turn is not captured (D-262, a transcript is opened only when it is a regular jsonl file). A turn recorded this way has none of what the model said before a tool call, which is the part of a turn the transcript supplies (D-003, reassemble turns by unioning two incomplete sources), so a colleague sees the closing text of each reply.

**Support.**
- The hook has the last message and no transcript. In a Cowork cloud task on Claude Code 2.1.331, a probe plugin recorded `transcript_path: ''` at SessionStart, UserPromptSubmit and Stop, and `last_assistant_message` among the fields of Stop, while the daemon logged each reply as refused for a name that did not end in `.jsonl`, since an empty path has no extension. `cmd/cogmer/transcript.go`.
- The hook reports only the turn's final text block, so with no transcript the reply is partial and never wrong about what it holds (B04, `last_assistant_message` contains only the turn's final text block).

**Rejected.**
- *Continuing to refuse an empty path.* Every reply from a Cowork cloud task is then missing from the room, while every prompt is there.
- *Searching the sandbox for the session's transcript.* A search for a file named for the session found none, and the one transcript under `~/.claude/projects` was the behavior check's.
- *Recording nothing for a reply when it is only partial.* A colleague who reads a prompt with no answer cannot tell whether the model answered.

**Limits.** Text before a tool call and the count of tool calls are not recorded for such a turn. No transcript shows what the model was given, so delivery cannot be confirmed by it (D-014, delivery state is derived from transcript evidence), and what that does to a colleague's turns offered to such a session is not known.

**Revisit when** the Cowork cloud runtime sends a transcript path, or a check of that host's hooks records the path as present.
