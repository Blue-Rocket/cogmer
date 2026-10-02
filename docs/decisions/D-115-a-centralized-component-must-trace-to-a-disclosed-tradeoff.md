# D-115 — A centralized component must trace to a disclosed tradeoff that benefits the person

**Date:** 2026-09-21 · **Status:** active (standing constraint)

**Context.** §32 lists central services among things not to build, as items on a
non-goals list ending "these can be evaluated after the core experiment". That is a
list, not a rule. It cannot govern a component that turns out to be necessary, and
three centralized or third-party components already exist with nothing governing
them.

The outlook recorded here is not privacy maximalism and not an objection to
centralization as such. It is that the count should be as small as the problem
allows, and that whatever remains should be explicable.

**Decision.** A compromise to privacy or to decentralization is acceptable when all
three hold.

1. **It traces.** A recorded decision says what was given up and why, so the
   reasoning can be found rather than inferred from the code.
2. **It is disclosed.** The person whose data it concerns is told, in something
   they actually read — not in this specification, which they will not.
3. **It benefits them.** The gain accrues to the person using this, not to the
   people building it. A compromise buying us convenience, operational simplicity
   or data is not one of these, however small.

Failing a test is not a prohibition on building the thing. It says the thing is not
yet defensible, and names which part is missing.

**Applied to what exists**, which is what makes this operative rather than
aspirational.

- **The DERP relay.** *Traces* — D-062, as
  `f288913:docs/decisions/D-062-tailcat-evaluated-for-phase-15-a-good-fit-adopted-behind-an.md` held it, evaluated it, and Phase 13's question
  asks openly whether depending on a relay nobody here operates is acceptable.
  *Benefits the person* — two people behind strict NAT cannot reach each other
  otherwise, so the alternative is not a purer system but no collaboration. *Not
  disclosed.* **Two of three.**
- **The model provider.** *Traces* — §28 states that injection carries a
  colleague's conversation to another person's provider under their own account,
  logged as review finding A3. *Benefits the person* — it is the product. *Not
  disclosed*, and it is the least obvious of the three: a colleague's words reach
  **your** provider on **your** subscription, which nobody would assume. **Two of
  three.**
- **The release host — pre-GA, and tagged as such.** *Traces* only partly: D-066
  decided the binary is fetched and verified rather than shipped, and the choice
  of host is not a decision at all but a default in `publish.sh`, a
  personally-operated droplet reached over plain HTTP at a bare IP
  (`plugin/release-url.txt`). Integrity is held by `plugin/checksums.txt` rather
  than by the transport, so tampering is caught; the transport still shows an
  observer which version was fetched. *Benefits us rather than the person*: it was
  what was available. *Not disclosed.* **One of three — and not a standing
  compromise.**

  It exists because there is nowhere else to publish from: a module path must match
  its repository URL, there is no repository, and that is blocked on the name
  (Phase 13). **GA is defined nowhere in this project**, so the tag needs an
  operative trigger rather than a milestone: this is retired when a public release
  host exists, which is the same event that unblocks Phase 13. `publish.sh` already
  anticipates it — "when this moves to a public release host, this script is what
  gets replaced and `release.sh` is what does not."

The asymmetry is worth naming. The relay is **forced** — NAT leaves no alternative.
The release host is **temporary** — it is what pre-GA looks like, not a tradeoff
anybody would defend, and a public host serves the person better on every axis
including this one. Under this rule those are different kinds of thing: one is
defensible as it stands, the other is defensible only as long as the tag is true.

**The disclosure test cannot currently be satisfied by anything.** There is no
user-facing documentation: `plugin/README.md`, the browser view and terminal output
are the only surfaces reaching a person, none mentions what is transmitted, and the
files in `plugin/commands/` are prompts rather than documentation (D-086). So this
rule presently creates an obligation nothing can meet, which is a fact about the
project rather than a defect in the rule. `84a0751:docs/what-leaves-findings.md` is the
evidence any disclosure would be written from.

**What this does not do.** It does not forbid a version check, which is the
question that produced it. A check plausibly benefits the person, since running a
stale build with a known defect is a harm — so the rule conditions it rather than
refusing it: decide it in the open, disclose it, and be able to say what the person
gets. It is a test, not a veto.

**Rejected.**

- *Treat centralization as a prohibition.* §32 already reads that way and is
  already wrong about the relay, which is both central-ish and necessary. A
  prohibition that reality overrules teaches people to ignore the document.
- *Rely on §32's list.* A list of things not to build cannot govern a thing that
  must be built, and it explicitly expires after the core experiment.
- *Require disclosure only where a person could notice the compromise.* The
  model-provider case is the one nobody would notice and the one most worth being
  told.

**Revisit when** a fourth centralized component is proposed, or when any of the
three above passes all three tests — at which point the question worth asking is
whether the rule was applied or merely satisfied on paper.
