# D-225 — A machine sends each removal and new shared key to every colleague it knows until each acknowledges

**Date:** 2026-10-19 · **Status:** not built · **Areas:** identity, sync

**Decision.** A machine that makes a removal, or certifies a new shared key (D-219, a user is a master key that certifies machines through a shared key), sends the record to every colleague it knows and resends until that colleague acknowledges it. A record counts as delivered only when the colleague has said it has it, since a record sent once to a colleague who is offline is lost.

**Rejected.**
- *Leaving colleagues to learn from the identity version in each event (D-221, an event whose identity version the receiver lacks is withheld from the assistant until fetched).* A colleague learns nothing until a machine that is ahead of them speaks to them, and a stolen machine keeps being accepted until then.

**Revisit when** a colleague is found to learn of a record by an event's version before the machine's resends reach them.
