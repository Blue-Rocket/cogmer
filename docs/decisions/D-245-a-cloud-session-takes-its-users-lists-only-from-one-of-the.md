# D-245 — A cloud session takes its user's lists only from one of the user's machines

**Date:** 2026-10-25 · **Status:** not built · **Areas:** identity, sync, trust

**Decision.** A cloud session learns its user's colleagues, rooms and machines by fetching them from one of the user's machines it can reach through the relay, and from no other source, so that the user's lists travel by one channel and no daemon runs git with the user's credentials unattended. What it fetches is the user's own signed records, the identity records and the list changes (D-228, a user's machines copy the user's lists between them as signed changes), with a header that one of the user's machines signs, giving a version and a time, and the session checks each record as a machine does, refuses a header whose version is lower than one it holds, and shows the header's age in the view. A machine answers such a fetch only for a key whose chain leads to its own user (D-216, a cloud session signs with a key of its own that the user's shared key vouches for), and the session asks again every few minutes, so a removal made after it started reaches it. A session that reaches none of the user's machines starts with no sharing, and keeps trying.

**Rejected.**
- *A private repo holding a snapshot, pushed by the daemon whenever the user's lists change.* It is a second channel for the user's lists and needs the user's git credentials used unattended by the daemon, to cover a session started while every machine of the user is asleep, a case the user accepts for now.
- *A snapshot record of its own, separate from the records the user's machines exchange.* It would be a second form of the user's lists, which could disagree with the first.

**Limits.** A session started while every machine of the user is asleep or offline shares nothing until one wakes. The fetched records list colleagues' keys and addresses and the user's machines' names and addresses, and anything that runs in the session, including the model's shell, can read them.

**Revisit when** users start cloud sessions with every machine asleep often enough that waiting for one to wake costs them the room.
