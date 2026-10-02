# D-169 — A lost room state is reported when the room is opened

**Date:** 2026-09-18 · **Status:** active · **Areas:** sync, rooms

**Decision.** When a room is opened, `reportLostState` compares the highest sequence the
store holds for this peer with the highest this peer recorded as issued. If the room holds
less, the daemon says that state was lost, that membership and sequence position are
intact, that history is being refetched and depends on a member being reachable, and that
colleagues' turns may be injected a second time.

**Support.**
- A running daemon serves a deleted database from its open file handle, so opening a room
  is the only moment the loss can be seen, and otherwise it surfaces at a restart hours
  later as a room that has gone quiet. `cmd/cogmer/daemon.go`, `reportLostState`.
- The specification requires the check on every open and the report in terms a user can
  act on. §8 (event identity and ordering).

**Rejected.**
- *Recovering silently.* A recovery nobody is told about looks the same as nothing having
  gone wrong.
