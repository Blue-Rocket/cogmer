# D-035 — A remote event never makes an interactive session take a turn

**Date:** 2026-09-17 · **Status:** active · **Areas:** capture, trust

**Decision.** A session a user is working in takes a turn when that user asks it to, and
at no other time. A remote event never prompts, resumes or otherwise drives an
interactive session, and any Claude run a peer event causes outside that session
neither borrows its context nor interrupts it.

**Support.**
- A turn that arrived unbidden would spend the context window the user relies on, might
  act on their working tree in the middle of their thought, and would take away their
  ability to reason about what their own session has seen. §3.7 (a remote event never
  drives an interactive session).
- Claude Code edits files and runs commands, so an event that could start a turn on a
  receiving machine would be execution on that machine, authorized by whoever sent the
  event. §3.7.
- Claude Code offers ways to start a run, such as `claude --bg`, so a daemon could drive
  a session, and only this rule prevents it. §3.7.

**Rejected.**
- *Forbidding a remote event from causing any Claude run at all.* It joins two concerns,
  not disturbing a session and not spending a user's resources, which have different
  remedies.

**Limits.** It does not decide whether a peer event may cause a separate Claude run
outside the interactive session.

**Revisit when** a peer event needs to cause a Claude run of its own.
