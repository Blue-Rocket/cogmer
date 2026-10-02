# D-111 — Section 3.8 is a test about hosts and not about Claude Code

**Date:** 2026-09-21 · **Status:** active · **Areas:** hosts, project

**Decision.** §3.8 reads "The host is launched and used unchanged". It defines the host as the
application a session runs in, names Claude Code as the host today, and states every clause of
the test about the host. A host that offers no way in is out of reach, and never a reason to wrap
it.

**Support.**
- The same clauses put a proposal out as before: a different launch command, installing
  something the host does not document as an extension mechanism, or depending on how the host
  renders. It rules out the PTY wrapper and permits a view outside the session. §3.8 (the host is
  launched and used unchanged), D-034 (no terminal wrapper; a view sits beside the session rather
  than around it).
- The constraint is a product boundary, so a host that cannot be extended is a host this system
  does not serve, and a host with some other way in needs a different design and not a smaller
  adapter. D-110 (host order: Claude Code, CoWork soon after, ChatGPT Desktop much later).
- The reach argument, that Claude Code is a terminal program, a desktop application and an editor
  extension and extending it through its own mechanisms reaches all three, is a fact about one
  product and remains stated as one. A different host has different surfaces, which changes which
  mechanisms exist and nothing about the reasoning. §3.8.

**Rejected.**
- *Leaving §3.8 naming Claude Code with a note that it applies to other hosts.* It is consulted
  as a test, and a test carrying a footnote about which product it governs is applied
  inconsistently.
- *Rewriting the whole specification in host-general language.* §3.8 is the framing test, and the
  rest describes Claude Code's actual mechanisms and is correct as written.
- *Waiting for CoWork.* The rewrite costs the same later.

**Limits.** What the install clause asks is D-113 (the install test names a type, and the type
must be documented).

**Revisit when** a second host is supported, since §3.8's Claude Code examples then want a
companion and not a replacement, or a host appears whose extension points reach a person directly.
