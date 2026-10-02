# D-104 — The overlay address holds no secret and is the same at every start

**Date:** 2026-09-21 · **Status:** active · **Areas:** transport, trust

**Decision.** The overlay runs with its pre-shared key disabled, so its published address
carries three public keys and a relay and is the same at every start.

**Support.**
- An overlay address encodes three keys and a relay. Two of the keys derive from the node key,
  which is persisted, and the third is a pre-shared key that the library generates afresh at
  every start unless told not to, so every restart, plugin update or wake from sleep would
  invalidate every pairing string and invitation the machine had issued. A restart measured
  twice produced identical endpoints with the key disabled and different ones with it enabled.
  `cmd/cogmer/tailcat.go`, `DisablePresharedKey`, and
  `f288913:docs/decisions/D-104-the-overlay-address-is-public-and-stable.md`.
- An address need be correct only once, and that holds on the overlay path, which is the path
  everybody uses, only while the address is stable. D-018 (identity and reachability are
  separate).
- The library treats an address that contains a pre-shared key as secret, and the pairing
  string is pasted into chat and read aloud, because the two-word comparison exists so that it
  need not be confidential, so a secret in it would bring back the bootstrap problem the
  ceremony removes. §25 (security).
- The key's purpose is a hedge on recorded traffic, which needs it to remain secret, and it
  travels inside the address that must be published, so no arrangement of one node's address
  holds both. A second address would need a second node key, which is the identity.
  `f288913:docs/decisions/D-104-the-overlay-address-is-public-and-stable.md`.

**Rejected.**
- *Keeping the pre-shared key in the address.* A restart would strand every string issued, and
  a secret would sit in a string that has to be public.

**Limits.** Disabling the key gives up the only quantum-resistant element in the stack, since
the TLS layer keys on the same elliptic curve. Whether to rebuild it at this layer is not
decided here. Who may open a tunnel is D-157 (the overlay listener admits any dialer).

**Revisit when** the library offers a per-peer pre-shared key.
