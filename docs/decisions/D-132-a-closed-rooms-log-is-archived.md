# D-132 — A closed room's log is archived

**Date:** 2026-09-16 · **Status:** not built

**Decision.** When a room closes, its event log is archived. An archived room can be
read and searched, and is never rejoined, synchronized or injected.

**Support.**
- A room keeps each user's actual prompts, Claude's responses, who said each, in what
  order and in which session. §3.4 (preserve actual conversation).
- Membership ends when a room closes. D-015 (rooms are scoped to sessions).

**Rejected.**
- *Discarding the log when a room closes.* It would lose conversation the
  specification requires to be kept.

**Revisit when** an archived room needs to be joined again.
