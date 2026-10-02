# D-077 — Every room has its own address in the view

**Date:** 2026-09-20 · **Status:** active · **Areas:** view, rooms

**Decision.** The view serves each room at `/room/<name>`. `/` lists the rooms and
redirects only when there is one. The events endpoint and the event stream take the
room as a parameter, and `watchLine` prints the room's own address.

**Support.**
- A list of one is a question nobody needs asked, so the redirect happens only then.
  `cmd/cogmer/ui.go`, `handleUI`.
- Two tabs stream two rooms and neither changes identity when somebody creates a third,
  and the line printed at creation is a stable link and not a window whose contents can
  move. `cmd/cogmer/daemon.go`, `handleEvents`, `cmd/cogmer/main.go`, `watchLine`, and
  `cmd/cogmer/ui_test.go`.
- The address carries the room, so the view needs no value that picks one for it. D-080
  (there is no current room).

**Rejected.**
- *One address that shows whichever room is current.* Its contents change identity when
  the current room changes.
