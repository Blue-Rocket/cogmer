# D-083 — The view first opens at a first pairing, not at room creation

**Date:** 2026-09-20 · **Status:** active · **Areas:** view, pairing

**Decision.** The view first opens when a user first pairs with a colleague, and not when a
room is first created.

**Support.**
- Pairing comes before a room in the user's experience. It is a machine-scope act that
  outlasts every room, and an unverified peer is refused before any room content moves.
  D-053 (`pair` is the machine-scope act, and `invite` is the room-scope act), D-054
  (verification gates synchronization and injection).
- The first unfamiliar step is reading two words aloud with a colleague, and that step needs
  something on screen where a terminal is the wrong place for something a user must find and
  compare under time pressure. D-055 (a peer is verified in one way, the two-word
  comparison).
- Pairing opens the view, and creating a room does not. `cmd/cogmer/main.go`, `beginCeremony`.

**Rejected.**
- *Opening the view at the first `create` or `join`.* A room is the second thing a user meets,
  and the first step would happen with nothing on screen.

**Limits.** D-088 (the pairing ceremony lives in the view, and every pairing gets its own
URL) holds that the view drives the ceremony and opens again at each pairing.
