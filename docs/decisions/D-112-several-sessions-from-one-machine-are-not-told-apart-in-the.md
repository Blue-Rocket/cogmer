# D-112 — Several sessions from one machine are not told apart in the block

**Date:** 2026-09-21 · **Status:** active · **Areas:** capture

**Decision.** The injected block does not mark which session of one machine a turn came from.

**Support.**
- One machine can have two live sessions in a room, since a session is keyed to its room because
  sessions are members and a peer is admitted separately, and both carry the same `peerId`, the
  same derived name and the same label. `UndeliveredFor` excludes the origin session and not the
  peer, so the two see each other's turns. §3.6 (session-scoped rooms), `cmd/cogmer/store.go`.
- A discriminator derived from `originSessionId`, which is on the event and not shown, reports the
  topology of processes and not the work, so it says that turn 7 came from a different session than
  turn 6 and nothing about whether that matters, and the reader cannot act on it. It also invites an
  inference it cannot support, two marked streams read as two topics or two people, which is a false
  affordance here as a data field. `cmd/cogmer/store.go`, `Event`.
- The view leaves the derived name off a user's own turns, where it identifies nothing the user
  does not know, and an identifier the machine can derive and the reader cannot use is the same
  kind of thing. D-021 (peer names are word pairs derived from the identity, never chosen), D-094
  (the name you chose leads in the view, with the derived name beside it).
- A replacement after a crash is sequential, and unrelated work corrects itself, since a session
  in no room captures nothing and joining is deliberate, which leaves knowingly split work,
  where the person knows why and the reader does not. D-064 (a session's room is the one somebody
  chose inside it, never a machine default).

**Rejected.**
- *A per-block discriminator derived from `originSessionId`.* It distinguishes without informing.

**Limits.** What would carry information is a purpose the user supplies, such as "this is the
frontend half", which is a feature with a real cost and is not justified without evidence that
anybody runs split sessions.

**Revisit when** there is evidence that two live sessions from one machine is an ordinary thing to
do, and not only a thing the schema permits.
