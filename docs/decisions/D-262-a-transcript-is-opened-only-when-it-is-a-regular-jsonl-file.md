# D-262 — A transcript is opened only when it is a regular .jsonl file under a size limit

**Date:** 2026-10-27 · **Status:** active · **Areas:** capture, daemon

**Decision.** The path a hook sends as `transcript_path` is statted before it is opened, and cogmer opens it only when it is a regular file, its name ends in `.jsonl` and it is no larger than 1 GiB. Every reader of a host's transcript goes through the one function that checks, for Claude Code, for Codex and for the late images of a Codex turn. Claude Code names a transcript `<session id>.jsonl` and Codex names one `rollout-<time>-<thread id>.jsonl`, so neither is refused. A refused path fails open (D-011, hooks fail open: exit 0, empty stdout): the turn is not captured, the Stop request is answered with nothing stored, and the daemon writes one log line that says why and never holds the path.

**Rejected.**
- *Requiring the path to sit under the host's transcript folder.* A cloud session and Cowork keep their transcripts somewhere cogmer cannot name in advance, so a real transcript would be refused there.
- *Opening the path and checking the file afterwards.* Opening a FIFO blocks until something writes to it, which holds the request open.

**Revisit when** a host names its transcript without a `.jsonl` ending, or a session's transcript passes 1 GiB.
