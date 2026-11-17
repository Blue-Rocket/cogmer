# D-253 — A cloud session says its turns are not being shared when every machine it asks refuses it

**Date:** 2026-10-26 · **Status:** active · **Areas:** identity, daemon, view

**Decision.** When every machine on a cloud session's list of the user's machines refuses its requests, the session's daemon says to the user, at the session's next prompt and on its room page, that every machine refused and that the session's turns are not being shared. A refusal is a machine's answer, so a session whose machines all refuse it is in a different state from one whose machines do not answer, which it reports as machines it cannot reach. A removed cloud key meets this refusal (D-101, peer connections are TLS pinned to a key this machine has recorded), and so does a suspended one (D-234, a machine not certified under the current shared key is suspended until it is certified again), and the words do not say which.

**Rejected.**
- *A signed statement of the removal, answered to a removed key.* It needs a route that a removed key can reach, which a thief who holds the key reaches too.

**Limits.** The session cannot tell a removal from a suspension (D-234, a machine not certified under the current shared key is suspended until it is certified again), and a turn it records before it says so is held only in its own room store and reaches nobody.

**Revisit when** a user is found to leave a removed session running for want of a clearer message.
