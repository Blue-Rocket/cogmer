# D-058 — Signature schemes are kept, never replaced

**Date:** 2026-09-18 · **Status:** active · **Areas:** identity, sync

**Decision.** An event records the scheme it was signed under and is verified under that
scheme. A new scheme is a new function beside the old ones, and an old one is never
edited. An event whose scheme is missing or unknown is refused as a version problem and
not as a forgery.

**Support.**
- Events are immutable and cannot be re-signed, and they are relayed between peers and
  refetched as recovery, so after a format change old events keep arriving and would be
  rejected if they were verified under the new bytes. §7 (event model), §13 (transitive
  synchronization), D-029 (losing a room database does not end membership).
- The version is not covered by the signature. An attacker who alters it only makes
  verification fail, since every scheme is Ed25519 over length-prefixed fields and there
  is no weaker scheme to be downgraded to. `cmd/cogmer/store.go`, `Event`.
- A refusal worded as a forgery sends a person hunting an attacker, and one worded as a
  version problem sends them to install a build, and the two need opposite responses. A
  test requires the message to say "upgrade" and not "does not match the peer id".
  `cmd/cogmer/keys_test.go`, `TestAnUnknownSchemeIsRefusedAsAVersionProblem`.
- Signing records the scheme, so nothing relies on a zero value meaning one. `cmd/cogmer/keys_test.go`,
  `TestSigningRecordsItsScheme` and `TestAnEventWithNoRecordedSchemeIsRefused`.
- Every other signing test signs a fresh record and verifies it, so an edit to the signing bytes
  changes signer and verifier together and none fails. Records signed once from a fixed key and
  checked in are what an edit cannot change: two events under v3, and a sync request, an offer, a
  verification step, the two pairing words and a commitment. Editing any signing function in turn
  failed at least one of them. `cmd/cogmer/testdata/vectors.json`, `cmd/cogmer/vectors_test.go`,
  `TestAnEventSignedUnderV3BeforeAnyEditStillVerifies`.

**Rejected.**
- *Editing the signing bytes in place and verifying every event under them.* Every
  historical event would stop verifying, and none can be re-signed.

**Limits.** If a scheme is found weak, that version is refused. Its number is not signed.
