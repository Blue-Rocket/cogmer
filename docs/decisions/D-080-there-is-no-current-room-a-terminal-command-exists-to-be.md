# D-080 — There is no current room; a terminal command exists to be tested or to work when the plugin cannot

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** Asked for the reasons a terminal command should exist at all, given
that plugin commands always run in a session which may or may not have a room —
leaving testing as the only obvious remaining purpose. Enumerating the surface
honestly produced five reasons, and the important result was what is **absent**
from them.

**The reasons that survive.**

1. **It cannot pass through a model.** `pair`, `verify`. Already decided (D-053, as
   `d37c77c:docs/decisions.md` held it, and §29): interactive, blocking on another person, and the compared words must reach
   a person's eyes unaltered.
2. **It must work when the plugin path is broken.** `doctor`, `behaviors`,
   `version`, reading `install.log`. A diagnostic that only works when things work
   is not a diagnostic.
3. **The daemon's own lifecycle.** §29 requires it be discoverable and stoppable,
   and it runs when no session exists.
4. **Machine scope rather than room scope.** `whoami`, `peers`, `forget`, `allow` —
   this machine's identity and relationships, which involve no room.
5. **Testing**, by emulating a session.

**What is absent: no room-scoped operation has a terminal-only reason to exist.**
Every one of them fulfils a slash command or is being tested. So the machine-level
pointer had no remaining user — the browser view was its last consumer that could
not be handed a session, and per-room URLs (D-077) removed that.

**Decision.** `SetCurrentRoom` and `CurrentRoom` are deleted. A room-scoped command
resolves one way: the room the invoking session is in. A terminal that wants one
emulates a session, which is what every test here already does.

**The exception, and why it is one.** `conflicts` takes a room by name. It is a
diagnostic of category 2, wanted precisely when a room is misbehaving and possibly
with no healthy session to ask — and naming a room in order to **read** it is not an
exercise of standing, which is what makes this safe here and unsafe in `invite`
(D-078, D-079).

**Slash commands now cover every command that can have one.** Added `/room-revoke`,
which had no session-side route at all and so was unreachable through the supported
path since D-079; `/room-conflicts`; `/peer-list`; `/peer-forget`. `/room-pair`
became `/peer-pair`, because pairing is machine scope and the prefix was saying
otherwise.

**Two prefixes, because there are two scopes** (§12): `room-` acts on one room,
`peer-` on relationships that outlast every room. Deliberately excluded, with
reasons: `pair`/`verify` (category 1 — `/peer-pair` signposts them and does not
attempt them), `doctor`/`behaviors` (category 2 — a slash command for them would
imply the plugin path is a reasonable way to diagnose the plugin path),
`daemon`/`hook`/`probe-hook` (machinery), `seed` (a fixture), `allow` (the
unverified form, for scripts).

**A test now asserts every command file names a real subcommand**, because the
plugin and binary version together and v0.2.0 already shipped a mismatch — a
`/room-status` that called `where` and got usage back.

**Revisit when** something other than a session can hold membership, or when a
category-2 diagnostic needs to change something rather than show it. The line drawn
here is between showing and changing, and it is the second that requires standing.
