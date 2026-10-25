# D-246 — A cloud environment is added from the machines page, which shows the one value to paste into it

**Date:** 2026-10-25 · **Status:** not built · **Areas:** identity, view, commands

**Decision.** A user adds a cloud environment on the daemon's page that `/cogmer:self-machines` opens (D-231, a user's machines are managed from one command that opens the daemon's page), where the machine makes a cloud key, certifies it under the shared key as a machine of kind cloud (D-216, a cloud session signs with a key of its own that the user's shared key vouches for), and shows one value, with a button that copies it, for the user to paste into the environment's variable `COGMER_CLOUD_KEY`. The value holds the cloud key, its chain and the relay address of each of the user's machines, which is the same at every start (D-104, the overlay address holds no secret and is the same at every start), so that a session can reach a machine before it holds the user's lists (D-245, a cloud session takes its user's lists only from one of the user's machines). The value is shown only on the page, never to an assistant, a command or a log, because whoever holds it speaks as the user in every room the user is in, the same reason the recovery phrase never reaches them (D-223, the recovery phrase is typed only into the daemon's browser page). The cloud key appears in the machine list with the user's machines, and is removed there like any machine (D-220, a removal names the last event of a machine that counts).

**Rejected.**
- *A command of its own that prints the value.* A command's output reaches the model and the session's transcript, and every other act on the user's machines is on the page.
- *A value holding only the cloud key, with the session finding a machine some other way.* A session needs an address before it can ask for the user's lists, and nothing else gives it one without a second channel.

**Limits.** A value whose machines have all been removed or have lost their keys reaches none of them, and the user adds the cloud environment again. A machine added after the value was made needs no new value, since any machine the value names gives the session the current list.

**Revisit when** a cloud environment can receive a secret without a person pasting it.
