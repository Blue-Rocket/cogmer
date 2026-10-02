# D-013 — The probe points ambient hooks at a closed port

**Date:** 2026-09-16 · **Status:** active · **Areas:** behaviors

**Decision.** The probe runs its Claude session with `COGMER_ADDR` set to a closed port,
so that any `cogmer` hook from the user's own settings fails open inside the probe and
records nothing.

**Support.**
- The user's own hooks also fire inside the probe's session, which would otherwise
  publish the probe's synthetic conversation into a live room.
  `cmd/cogmer/probe.go`.
- A hook whose daemon cannot be reached exits 0 with empty output. B11.

**Rejected.**
- *Loading no ambient settings, with `--setting-sources ""`.* It was never tried, and a
  surprise in how Claude Code parses the flag would break the probe entirely.

**Revisit when** a hook from the user's settings is found to act inside the probe.
