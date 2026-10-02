# D-184 — A pairing's three endings leave different state

**Date:** 2026-09-20 · **Status:** active · **Areas:** pairing, identity

**Decision.** An abandoned pairing leaves the key unverified, with the derived name as its
label, and resumable. A match leaves the key verified with the label the user chose. A
mismatch leaves nothing when this pairing created the row, and removes only what the pairing
created.

**Support.**
- Abandoning and mismatching once left identical state, so a detected interception left the
  attacker's key on the list under the name meant for the real person, and the uniqueness of
  a name then blocked pairing with the real colleague until somebody worked out that they had
  to `forget` first. `cmd/cogmer/pairview_test.go`, `TestTheThreeEndingsOfAPairing`.
- Re-verifying a colleague of years and seeing different words is an alarm about an existing
  relationship and not a reason to discard it and every admission it holds, so a mismatch
  removes only a row this pairing created. D-073 (forgetting a peer discards their admissions
  with them), `cmd/cogmer/verify.go`.

**Rejected.**
- *Leaving a mismatched key recorded under the colleague's name.* The attack that was detected
  becomes a second, unrelated-looking problem at the worst moment.
- *Discarding an existing relationship on a mismatch.* It destroys a pairing to answer an
  alarm.
