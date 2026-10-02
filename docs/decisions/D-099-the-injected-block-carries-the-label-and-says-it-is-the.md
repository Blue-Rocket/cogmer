# D-099 — The injected block carries the label and says it is the name to use

**Date:** 2026-09-20 · **Status:** active · **Areas:** capture, identity, trust

**Decision.** The injected block carries the label, the name the user gave that colleague,
beside `speaker` and `peerName`, and omits the field when there is no label. The framing says
the label is the name the user calls that person by and is preferred over both.

**Support.**
- The view says one name and the model would say another, `Ec2-user` or a word pair, in the
  same conversation about the same person. `cmd/cogmer/daemon.go`, `FormatTeamContext`.
- `speaker` is what the peer claims and `peerName` is what can be checked, so the label is
  preferred and not substituted, and dropping either would remove the thing the preference is
  safe because of. `cmd/cogmer/daemon.go`.
- A field the model is given and told nothing about is a puzzle, so the framing states what it
  is for, and an absent label is an absent field because a blank value invites the model to
  wonder what it means. `cmd/cogmer/transcript_test.go`, `TestTheInjectedBlockCarriesTheNameYouChose`.
- `PeerFacts` groups what the receiving side knows, `IsVerified` and `Label`, as against what
  the peer asserts, which is the distinction the block is built on, so a further fact costs an
  implementation and nothing at a call site. D-090 (attribution in the injected block is
  structured, and nothing of ours is joined to a peer's text), `cmd/cogmer/daemon.go`,
  `PeerFacts`.

**Revisit when** a fourth peer fact appears, which should be an implementation change alone.
