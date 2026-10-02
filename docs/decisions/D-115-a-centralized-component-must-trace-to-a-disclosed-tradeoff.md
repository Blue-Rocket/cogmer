# D-115 — A centralized component must trace to a disclosed tradeoff that benefits the person

**Date:** 2026-09-21 · **Status:** active · **Areas:** project, trust, transport

**Decision.** A compromise to privacy or to decentralization is acceptable when all three hold. It
traces, in that a recorded decision says what was given up and why. It is disclosed, in that the
person whose data it concerns is told in something they actually read. It benefits that person, in
that the gain accrues to the user and not to those building the system. Failing a test says that the
thing is not yet defensible and names the part that is missing, and it does not forbid building it.

**Support.**
- A list of things not to build cannot govern a component that has to be built, so the count of
  such components should be as small as the problem allows and whatever remains should be
  explicable. §32 (explicit non-goals).
- A compromise that buys the builders convenience, operational simplicity or data is not one that
  benefits the person, however small. W-82 in `docs/writing.md`.
- The relay is forced and not chosen, since two people behind strict NAT cannot reach each other
  otherwise, so the alternative is not a purer system and is no collaboration. D-068 (tailcat is
  the cross-network transport, behind our own dialer).
- The model provider is the least obvious of the three, since a colleague's words reach the
  user's provider on the user's subscription, which nobody would assume. §28 (configuration).


**Rejected.**
- *Treating centralization as a prohibition.* A prohibition that reality overrules teaches people
  to ignore the document, and the relay is both central in part and necessary.
- *Requiring disclosure only where a person could notice the compromise.* The model-provider case
  is the one nobody would notice and the one most worth being told.

**Limits.** The rule does not forbid a version check. Running a stale build with a known defect is
a harm, so a check plausibly benefits the person, and the rule conditions it: decide it in the open,
disclose it, and be able to say what the person gets.

**Revisit when** a fourth centralized component is proposed, or when one of the existing ones
passes all three tests, and the question is whether the rule was applied or only satisfied on
paper.
