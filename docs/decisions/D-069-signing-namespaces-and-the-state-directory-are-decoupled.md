# D-069 — Signing namespaces and the state directory are decoupled from the name

**Date:** 2026-09-19 · **Status:** active (implemented)

**Context.** Asked to push the repository, which meant choosing a name — and the
name was not settled. Mapping where the product name was load-bearing turned up two
couplings that are free to break today and permanent after the first real room.

**The signing namespaces.** Every signature covered a domain-separation string with
the product name inside it: the event tag, the sync-request tag, and the SAS and
verify tags. D-058 settled what a change to those costs — schemes
are **added, never edited**, because an event is immutable and can never be
re-signed and D-029 makes refetching history a recovery path. So a rename after
real events exist would mean carrying the old namespace forever, for a name nobody
uses.

They should never have carried a product name. Domain separation needs **stability
and uniqueness**; it does not need meaning. `protocolNamespace` is now `peer-room`,
documented as arbitrary on purpose so there is never a reason to change it.

**D-058's mechanism got its first real use, which is the point of having built it.**
`signingBytesV3` carried the new namespace while the superseded scheme stayed
untouched beside it, serving every event an older build had signed. (D-117 later
deleted that scheme, on the ground that no such event existed anywhere; the
mechanism that would have carried it is unchanged.)

The other three tags moved outright rather than gaining a version. They protect a
live exchange — a sync request, a verification — and nothing signed under them
outlives it, so there is no history to keep faith with.

**The state directory.** `~/.cogmer` is where the name reaches the filesystem.
It is now a single constant, so a rename is one line rather than a search.

**And a latent bug found by looking.** `install.sh` honoured `COGMER_HOME`
while the binary ignored it, so anyone setting it got a binary in one place and its
state in another, with nothing saying so. It was invisible because the only thing
that set it was my own testing, which set `HOME` as well and so never noticed.

**What is deliberately not done.** The module path still says
`github.com/Blue-Rocket/cogmer`, and the organisation is actually
`Blue-Rocket` — a path that has never resolved, which is why `go install` failed in
D-066's test. It is left alone because a module path must match the repository URL,
and that needs the name. Nothing external imports it, so it costs nothing to wait.

**Revisit when** the name is settled. What changes then: module path, binary name,
plugin name, state directory, release URL path, and the `/team-*` commands only if
the prefix is wanted differently — they carry no product name already. What does
**not** change, by construction, is anything cryptographic.
