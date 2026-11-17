# D-214 — An image a host shows after the prompt is sent in a second event

**Date:** 2026-10-15 · **Status:** not built · **Areas:** capture, sync

**Decision.** When a host shows a pasted image only after its prompt has been published, as Codex does, the
daemon watches the session's transcript, and when the image appears it publishes a second signed event that
carries the image's hash and points at the prompt's event. A colleague's view shows the picture under the
prompt's turn only when the second event was written by the machine that published the prompt, or, for a
cloud session, by the session that did (D-216, a cloud session signs with a key of its own that the user's
shared key vouches for), and the injected context leaves it out. The hash has to be in a signed event,
and the prompt's event is signed before the bytes exist.

**Rejected.**
- *Amending the prompt's event.* A signed event cannot change.
- *Holding the prompt's event until the image appears.* The prompt would be published later, and a crash in
  the gap would lose it.
- *Taking the newest image file in the temp folder at prompt time.* Nothing ties the file to the session, so
  an image from another session, or one the user removed, could be sent.
- *Accepting the second event from any machine of the prompt's user.* Only the machine that published the
  prompt reads the session's transcript, so another machine's image for that prompt is one it never saw
  there.

**Revisit when** a host gives the image at prompt time.
