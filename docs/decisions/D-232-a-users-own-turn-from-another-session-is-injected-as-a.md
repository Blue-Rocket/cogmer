# D-232 — A user's own turn from another session is injected as a report marked as the user's

**Date:** 2026-10-24 · **Status:** active · **Areas:** capture, trust, identity

**Decision.** A turn this user wrote in another session, on another of their machines or on this one, is injected into this user's sessions in the room as a colleague's turn is: a report of what was said elsewhere, never words addressed to this session (D-149, the injected block frames its content by classification, not by authority). It carries the label "you", and a field this machine sets saying that the turn is this user's own from another session. The turn is injected at all because a user's sessions in one room share context with one another as they do with a colleague's, whichever of the user's machines they run on. This machine sets the field from the user identifier that the turn's certificate chain proves (D-219, a user is a master key that certifies machines through a shared key), so no text in a turn can claim it.

**Rejected.**
- *Framing the turn as words addressed to this session.* The user wrote it to another session, about that session's work, so this session would act on a request that another session is answering. Anything that gets a turn signed under the user, such as a stolen machine before its removal (D-220, a removal names the last event of a machine that counts, room by room), would then instruct every session of the user's.
- *Framing it as a colleague's turn with nothing marking it.* The model would read the user's own words as another person's, and would name the user to the user by their derived name.
- *Not injecting the user's own turns from another machine.* A session would miss what the user did in the same room on another machine, and the user would have to retell it.

**Revisit when** a session is found treating the user's own turn from another machine as an instruction to it, or as a colleague's.
