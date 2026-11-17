# D-248 — A cloud session asks its user's colleagues for a room at the addresses its user's lists hold

**Date:** 2026-10-25 · **Status:** active · **Areas:** sync, transport

**Decision.** A cloud session that joins a room records each colleague its user's lists give an address for as a place to ask for that room, and asks only those the room's guest list admits, so that it syncs its turns with its colleagues directly and not only through one of its user's machines (D-216, a cloud session signs with a key of its own that the user's shared key vouches for). A colleague's machine that the session asks then knows the session's key and address from the signed request, and asks the session in turn (D-160, the address in a sync request is signed). A colleague who is no guest of the room is never asked, and never learns the room's identifier from the session.

**Rejected.**
- *Reaching colleagues only through the user's machine the session fetched its lists from.* Every turn would wait on that machine being awake, which is the dependence D-216 rejects.

**Revisit when** a user's own machines are found to need the same direct route to colleagues for a room joined by name.
