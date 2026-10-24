# D-099 — The injected block carries the label and says it is the name to use

**Date:** 2026-09-20 · **Status:** active · **Areas:** capture, identity, trust

**Decision.** The injected block carries the label, the name the user gave that colleague,
beside `speaker` and `peerName`, and omits the field when there is no label. The framing says
the label is the name the user calls that person by and is preferred over both. On this
user's own turn from another session the label is "you", and the block carries a field this
machine sets saying that the turn is the user's own (D-232, a user's own turn from another
session is injected as a report marked as the user's).

**Support.**
- The view says one name and the model would say another, `Ec2-user` or a word pair, in the
  same conversation about the same person. `cmd/cogmer/daemon.go`, `FormatTeamContext`.
- `speaker` is what the peer claims and `peerName` is what can be checked, so the label is
  preferred and not substituted, and dropping either would remove the thing the preference is
  safe because of. `cmd/cogmer/daemon.go`.
- The name of the machine a turn came from is, like `speaker`, what the turn's user asserts,
  and the block marks it so. D-233 (every turn carries the name of the machine it came from, as
  a field the user asserts).
- A field the model is given and told nothing about is a puzzle, so the framing states what it
  is for, and an absent label is an absent field because a blank value invites the model to
  wonder what it means. `cmd/cogmer/transcript_test.go`, `TestTheInjectedBlockCarriesTheNameYouChose`.
- `PeerFacts` groups what the receiving side knows, `IsVerified` and `Label`, as against what
  the peer asserts, which is the distinction the block is built on, so a further fact costs an
  implementation and nothing at a call site. Whether a turn is this user's own is such a fact.
  D-090 (attribution in the injected block is structured, and nothing of ours is joined to a
  peer's text), `cmd/cogmer/daemon.go`, `PeerFacts`.

**Revisit when** a further fact the receiving side knows, such as whether a turn is this
user's own, needs more than an implementation change.
