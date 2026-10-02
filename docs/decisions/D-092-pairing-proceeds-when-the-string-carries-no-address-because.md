# D-092 — Pairing proceeds when the string carries no address, because only one side needs to dial

**Date:** 2026-09-20 · **Status:** active · **Areas:** pairing, transport

**Decision.** `pair` starts the exchange when a pairing string has no address, and says which
of the two people has to dial.

**Support.**
- `RunVerification` checks for an inbound exchange before it dials, so when the other side
  runs the whole exchange against this session the revealed nonce is here and the
  same two words come out. D-059 (either side completes a verification from inbound, so one
  side can drive it), `cmd/cogmer/sas_test.go`, `TestOnlyOneSideNeedsAnAddress`.
- A colleague behind a NAT that cannot be traversed is the side that publishes nothing, and
  refusing to start sent the user to fetch an address they did not need. `cmd/cogmer/main.go`,
  `runPair`.
- An address reaches this machine once, at `pair` time, as the `@` tail of the string a
  colleague sends, and `RemoteAddr` is read nowhere, so nothing is learned from a connection.
  `cmd/cogmer/main.go`, `parsePairing`, `cmd/cogmer/sync.go`, `syncTargets`.

**Limits.** The instruction sites that point at `/peer-pair` and not at `verify` name the
command in full, and the operator surfaces keep the short form because the reader is
at a prompt. They are the daemon log, the stderr of `doctor` and the header of the generated
behaviors document. D-085 (a line telling a user to run cogmer uses the path the binary
reports for itself).

**Revisit when** a pairing string carries a set of addresses, and "no address" becomes "no
address that worked", which is a different message. D-091 (identity is advertised and
location is discovered).
