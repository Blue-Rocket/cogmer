# D-247 — A cloud session's key is certified for twenty-four hours, and the session's daemon certifies a new one before it expires

**Date:** 2026-10-25 · **Status:** active · **Areas:** identity, daemon

**Decision.** The cloud key certifies a session's key for 24 hours (D-216, a cloud session signs with a key of its own that the user's shared key vouches for), and the session's daemon, which holds the cloud key from the environment, certifies a new session key for the same session before the old one expires, so that a session key copied out of a container stops working within a day while a session that runs longer keeps speaking for its user without interruption.

**Rejected.**
- *A certificate lasting as long as a session might.* Nobody knows how long that is, and a copied session key would work for as long.
- *A certificate lasting an hour or less.* A daemon that misses one renewal, while asleep or cut off, would have its session's turns refused until it renews, and gains nothing, since the cloud key in the same container can certify a new session key at any time.

**Revisit when** a cloud session can sign with a key it cannot read.
