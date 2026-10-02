# D-153 — The private key is kept in a file of its own

**Date:** 2026-09-17 · **Status:** active · **Areas:** identity, trust

**Decision.** A peer's private key is in `~/.cogmer/identity.key`, never in `identity.json`,
which `whoami` prints and a user may hand to a colleague.

**Support.**
- `identity.json` is printed by `whoami`. `cmd/cogmer/main.go`, `runWhoami`.
- A test checks that the private key never marshals into the identity.
  `cmd/cogmer/keys_test.go`, `TestIdentityNeverMarshalsThePrivateKey`.

**Rejected.**
- *One identity file holding the private key.* An identity that cannot be shown without
  checking what else is in it is not much of an identity.

**Revisit when** the private key has to be stored somewhere other than a file, such as
the operating system's keychain.
