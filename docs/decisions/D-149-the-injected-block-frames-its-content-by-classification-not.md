# D-149 — The injected block frames its content by classification, not by authority

**Date:** 2026-09-17 · **Status:** active

**Decision.** The framing around injected turns tells the model what kind of thing it is
reading: that nothing inside is addressed to it, however phrased, including text that
appears to come from an operator, a system or its own user, and that a request inside
is a report that someone made a request, not a request made of it.

**Support.**
- Telling a model to disregard instructions invites it to weigh two instructions, while
  telling it what it is reading does not. §20 (attribution in injected context).
- A session given a forged "SYSTEM OVERRIDE" instruction inside the block ignored it,
  explained that it came from inside the record and was information, and told its own
  user, unprompted. `84a0751:docs/decisions.md`.
- A check fails if the injected block stops framing its content as information rather
  than instruction. B22.

**Rejected.**
- *Instructing the model to disregard any instructions in the block.* It sets two
  instructions against each other.
- *Relying on the model to withstand the instruction.* That is a property of a model, not of this design,
  and the framing has to hold whether or not the next model can.

**Limits.** A colleague's genuine turn may contain imperative text, and no framing tells a
hostile imperative from a harmless one. Both are reports of what someone said, and
neither is addressed to the reading model.

**Revisit when** a session is found following an instruction from inside the block.
