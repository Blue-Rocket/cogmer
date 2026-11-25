# D-300 — A prompt is judged and recorded without the text its host puts before what the user typed

**Date:** 2026-11-25 · **Status:** active · **Areas:** capture, hosts, commands

**Decision.** The daemon removes the blocks of `<system-reminder>…</system-reminder>` text at the start of a prompt before it decides whether the prompt is a command (D-209, a command typed in a session is no turn of the room) and before it records the prompt. A command typed in a session of a host that puts such text first is therefore held back from the room, with what the model says about it, and the turn a colleague's session is given holds what the user typed. A prompt that is nothing but such blocks is left as it came. A Cowork cloud task puts a reminder holding the user's timezone and the time before the prompts it sends, and a command typed after it was recorded as a turn of the room, with its reply (`51d548e`).

**Rejected.**
- *Finding the command prefix anywhere in the prompt.* A prompt that only mentions a command would be held back from the room.
- *Matching the reminder's own sentence word for word.* The text differs between hosts and versions, and only its form, a block that begins and ends with the tag, is common to them.

**Limits.** Only blocks at the start of a prompt are removed, and only in the `<system-reminder>` form. A prompt that is nothing but a reminder is recorded as it came, which puts the reminder's text in the room. No behavior check covers that a host adds the text, since no probe runs in a cloud task (D-297, the behavior checks are not run in a cloud container).

**Revisit when** a host marks what it adds to a prompt apart from what the user typed.
