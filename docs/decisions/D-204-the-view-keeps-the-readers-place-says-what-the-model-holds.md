# D-204 — The view keeps the reader's place, says what the model holds, and exports the room

**Date:** 2026-10-13 · **Status:** active · **Areas:** view

**Decision.** The room view draws each turn once and moves it into place when it arrives, so an arrival
leaves a tool list the reader opened and text the reader selected as they were. It counts the colleagues'
turns that arrive while the reader is away from the newest turn or the tab is hidden, shows the count on a
button that jumps to them and in the tab's title, and clears it on reaching the bottom. It marks a
colleague's turn that no session on this machine is confirmed to hold, except in a show-only room. It shows
when each colleague last wrote a turn. It offers the room as a Markdown download.

**Support.**
- The view is the only way a user sees a colleague's turns, so a turn the reader did not notice arrived
  without the reader knowing. D-039 (one renderer until it has been used), D-086 (the terminal is not a user
  experience; the view is the surface).
- A reachable colleague whose turns have stopped arriving looked the same as a quiet one, since the dot
  reports reachability and not turns. The time of the last turn is what separates them, and it is sent as
  an absolute time, which cannot go stale between redraws. `cmd/cogmer/ui.go`, `turnTime`,
  `cmd/cogmer/viewaffordances_test.go`, `TestThePeerListCarriesWhenEachColleagueLastWroteATurn`.
- Whether a turn reached the model is the delivery state, a set of events a session has been confirmed to
  hold, and the marker reads it across every session on the machine. §19 (incremental context injection),
  `cmd/cogmer/store.go`, `DeliveredEventIDs`, `cmd/cogmer/viewaffordances_test.go`,
  `TestTheViewSaysWhichColleagueTurnsNoSessionHolds`.
- A show-only room gives nothing to a session, so every turn in it would carry the marker, which would
  say nothing. D-200 (a room can be show-only), `TestAShowOnlyRoomMarksNoTurnAsNotInASession`.
- The export is written by the daemon and not by the page, so a test reads it, it carries the date a
  turn's clock leaves out, and the response is a download with `nosniff`, so a colleague's text is never
  rendered as a page. `cmd/cogmer/ui.go`, `exportMarkdown`, `cmd/cogmer/viewaffordances_test.go`,
  `TestTheExportNamesEachTurnAndKeepsItsContent`, `TestTheExportRouteDownloadsTheNamedRoom`.
- The export route is a read of what `/stream` sends, so it is not guarded. D-087 (the local API requires a
  header that a web page cannot send), `cmd/cogmer/daemon.go`, `TestUIIsNotServedToPeers`.
- The page's script ran in a browser DOM under Node with jsdom on 2026-10-13: an opened tool list survived
  a later arrival, a late turn sorted into place, the count rose and cleared on scrolling, and a changed
  label rewrote one line. `cmd/cogmer/ui.html`.

**Rejected.**
- *A line across the feed where the session has read up to.* Delivery is a set with holes, since a lost
  injection leaves one, so no single place in the feed is the boundary, and each turn is marked instead.
  `cmd/cogmer/store.go`, `session_delivered`.
- *A last-turn time written as an age, such as "3m ago".* It is wrong the next time the page redraws
  without a new event.
- *A browser notification when a turn arrives.* D-039 holds it until the view is shown not to be
  consulted. The count appears only on a page the reader has open.

**Limits.** No test runs the page's script, so these rules hold by `cmd/cogmer/ui.html` and the guard
`TestThePageKeepsTheReadersPlace`. With two sessions on a machine, a turn one of them holds is not marked.
Searching, copying a turn and hiding turns that only ran tools are not built, and each waits for use to
ask for it (D-039).
