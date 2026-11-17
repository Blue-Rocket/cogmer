# D-249 — Each key that signs in a room counts its own sequence from 1

**Date:** 2026-10-25 · **Status:** active · **Areas:** sync, storage

**Decision.** The count of sequences a machine has issued in a room belongs to the key that signed them, and a reservation by another key starts that key's count at 1 in the same transaction that records the key, so that a cloud session that certifies a new session key (D-247, a cloud session's key is certified for twenty-four hours, and the session's daemon certifies a new one before it expires) numbers the new key's events from 1, as each key numbers its own events (D-216, a cloud session signs with a key of its own). A count stored with no key, from before counts named their key, is taken as the count of the key that next reserves, which on a machine is the machine's own key (D-029, losing a room database does not end membership; the sequence lives with the machine).

**Rejected.**
- *One count per room across a session's keys.* A new key's first event would follow the old key's last number, and a receiver counts only an unbroken run from 1 for each key, so it would hold the gap open and be offered the new key's events at every poll.
- *Resetting the counts when the new key is written.* A crash between writing the key and resetting would leave the old count under the new key, and resetting first would let the old key, still in use until the swap, reissue a number it had given.

**Revisit when** a machine signs in one room with two keys at once.
