# D-104 — The overlay address is public and stable

**Date:** 2026-09-21 · **Status:** active (implemented)

**Context.** Somebody prints their pairing string at a coffee shop, is interrupted,
and sends it from home two hours later. Is the address still good? Following that
found the address is not stable at all, and that it contains a secret.

**The address changes on every restart, and the cause was measured.** A tailcat
address encodes three keys and a relay. Two of the keys derive from the node key,
which is persisted. The third is a **pre-shared key**, and the library generates a
fresh one at every `Start()` unless told not to. Restarting a daemon twice and
comparing the published endpoints:

| | two starts identical |
|---|---|
| pre-shared key disabled | yes — byte for byte |
| pre-shared key enabled | no |

So every reboot, plugin update or bad wake from sleep silently invalidates every
pairing string and every invitation that machine has issued. That makes §4's claim
that an address need only be correct once false for the overlay path, which is the
path everybody uses.

**And it puts a secret in a string this design requires to be public.** The
library's own guidance is to treat an address containing a pre-shared key as secret.
§12 says the opposite, and must: the string is pasted into chat and read aloud, and
the two-word comparison exists precisely so that it need not be confidential. A
secret in that string reintroduces the bootstrap problem the ceremony was built to
remove.

**The two cannot both hold at that layer.** The key's purpose is a post-quantum
hedge on recorded traffic, which requires it to stay secret, and it travels inside
the address, which must be published. No arrangement of one node's address escapes
that. A second address would mean a second node key, which is the identity.

Who may open a tunnel is D-157 (the overlay listener admits any dialer).

**The hedge is genuinely lost, and is recoverable elsewhere.** Nothing else in the
stack is quantum-resistant — the TLS layer keys on the same elliptic curve — so
removing it removes the only such element. It can be rebuilt at our own layer, from
material the pairing exchange already produces, per peer rather than one secret
shared with everybody ever paired with. That is a separate decision and is not
blocked by this one.

**Also pin the region.** The published address embeds a relay chosen at startup by
latency, with a random fallback when the probe fails. Disabling the pre-shared key
made two starts identical on one network; it does not follow that a start on another
network picks the same relay. Persisting the chosen region alongside the node key
makes the address a property of the identity rather than of the last startup.

**Revisit when:** the post-quantum hedge is wanted, or the library offers a per-peer
pre-shared key.
