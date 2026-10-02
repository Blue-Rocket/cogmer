# D-117 — The product is named `cogmer`, and nothing is carried forward

**Date:** 2026-09-22 · **Status:** active (implemented)

**Context.** The name is settled: the product, the plugin and the repository are
`cogmer`. D-069 did the expensive half of this a name ago, by breaking the couplings
that would have been permanent once real events existed. What was left was the half
that is merely annoying.

**Decision.** The name reaches the module path, the binary, the command directory,
the plugin manifest, the state directory, the local API header, the environment
variable prefix and the release path. Nowhere else.

**There is no compatibility surface, because there is nothing to be compatible
with.** The only installation that ever existed was on the author's machine, and it
was deleted rather than migrated. That single fact removes work that would
otherwise have been mandatory, and it is worth stating plainly because every
instinct here says otherwise:

- The superseded event signing scheme is **deleted**, not retained. `signingBytes`
  now knows one scheme and refuses every other version, including the zero value
  that used to mean "stored before the column existed". A test pins that refusal,
  because silently guessing a scheme verifies bytes whose provenance nobody
  checked.
- There is **no state directory migration**. A directory under the former name is
  not looked for, moved, or mentioned.

**This does not weaken D-058.** Schemes are still added and never edited, and the
`signingBytes` switch that makes that possible is intact and still tested. What
changed is that the set of schemes worth keeping turned out to be empty. Once
anybody else holds a room, it will not be empty again, and deleting a scheme stops
being available.

**The organisation casing is fixed at the same time.** D-069 recorded that the
inherited module path disagreed with the organisation's actual name, which is why
`go install` failed in D-066's test. A path that had never resolved was free to
leave alone; one about to name a real repository is not.

**Nothing cryptographic moved, which was the point of D-069.**
`protocolNamespace` is still `peer-room` and still arbitrary on purpose.

**The former name is struck from this log rather than preserved.** Where it was
incidental — a path, a command, an environment variable, a module path — the
current name is substituted as though it had always been there. Where the name
itself was the subject, as in D-069's account of namespaces carrying a product
name, the prose now says "the product name" and names nothing. A dead name in a
decision is a thing a reader must hold and resolve, and it buys no understanding:
the reasoning was never about which name it was.

**The command prefixes did not change**, for reasons that are now different from
the ones that put them there. See D-118.

**What this unblocks.** The repository, and through it Phase 13 — which is now
blocked on the repository existing rather than on the name. Also D-096's overview
command, though not in the form D-096 planned for it; D-118 says why.

**Revisit when** somebody other than the author holds a room. At that point the
latitude this entry used — deleting a scheme, deleting state — is gone, and D-058's
rule applies with nothing to soften it.
