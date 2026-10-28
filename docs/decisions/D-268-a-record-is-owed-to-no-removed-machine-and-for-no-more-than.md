# D-268 — A record is owed to no removed machine and for no more than a week of failed attempts

**Date:** 2026-10-28 · **Status:** active · **Areas:** identity, sync

**Decision.** A machine sends a removal or a shared-key certificate to a target only while the target is not a machine of the user's that a removal removes (D-252, a reachable machine is asked to erase its keys before it is removed), and for at most 1,008 failed attempts, which is a week at the longest wait between attempts, ten minutes. A removal that removes a machine drops what was owed to it and owes it nothing after, and a record that is owed for a week without an acknowledgement is not sent again. A target away for longer learns of the record when it returns, from a machine that is ahead of it (D-221, an event whose identity version the receiver lacks is withheld from the assistant until fetched).

**Rejected.**
- *Sending until acknowledged, however long it takes.* A cloud session that has ended and a machine that erased its keys never acknowledge, and each costs a request every ten minutes for as long as the daemon runs.
- *A limit in time from the first attempt.* The table holds no time for the first attempt, and the attempts at the longest wait measure the same week.

**Limits.** A machine that is away for more than a week is not sent what it missed when it returns, and learns it by its first exchange with a machine that is ahead of it.

**Revisit when** a machine is found to lack a removal or a shared-key certificate after a week away.
