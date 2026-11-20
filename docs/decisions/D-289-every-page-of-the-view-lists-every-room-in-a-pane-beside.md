# D-289 — Every page of the view lists every room in a pane beside the room it shows

**Date:** 2026-11-20 · **Status:** active · **Areas:** view, rooms

**Decision.** Every page of the view has a rooms pane, listed from a stream the daemon sends whenever a room changes and every 20 seconds, the most recently active room first, each with the newest turn's writer and time, the colleagues' turns since the reader last looked in this browser, the turns no session here has been given (D-204, the view keeps the reader's place and says what the model holds), the sessions here in the room and how many colleagues are reachable. A room in the pane links to its own page at `/room/<name>` (D-077, every room has its own address in the view), and `/` is that page with no room picked, whatever rooms exist. A user in several rooms asked to see which rooms are active from where they read a room.

**Rejected.**
- *A separate page that lists the rooms and shows the one picked in a frame.* It is one more page to keep open, and the room page would have to let another page frame it.
- *Redirecting `/` into the room when there is only one.* The same address then opens a room on one day and a list on the next.
- *Counting what is new on the daemon.* What a reader has looked at belongs to one browser, and the daemon would have to be told of every look.
