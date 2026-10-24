# D-222 — A machine tells every machine it knows of each machine it adds, until each has acknowledged

**Date:** 2026-10-18 · **Status:** active · **Areas:** identity, sync

**Decision.** Any two of a user's machines exchange identity records, their own user's and those of the users they know, the user's lists (D-228, a user's colleagues, verifications, labels and rooms belong to the user and are copied between the user's machines) and the events of each room that both have joined. A machine that adds another gives the new machine its list of the user's machines, and the new machine introduces itself to every machine on that list. Each machine sends its whole list to every machine it knows and merges the lists it receives, resending until every machine on its own list has acknowledged. A list is the machine records that have not been removed, and the records only accumulate, so any order of arrival gives the same list, and machines added at the same time still learn of one another.

**Rejected.**
- *Only the adding machine notifying the others.* If it is lost before it reaches them, they never learn of the new machine, and two machines added at once never learn of each other.
- *Colleagues holding the list.* A colleague holds only the machines that have sent it an event, so a machine that has not yet spoken is on no colleague's list.

**Limits.** A machine's list is out of date until it next hears from another of the user's machines, and a user who picks a machine to remove sees only the machines their own machine knows. A user who is unsure which machine was taken has the master certify a new shared key (D-220, a removal names the last event of a machine that counts), and only the machines that machine knows are certified again, so an unknown machine is suspended until it is certified again (D-234, a machine not certified under the current shared key is suspended until it is certified again).

**Revisit when** a machine is found missing from another machine's list after the two have been in contact.
