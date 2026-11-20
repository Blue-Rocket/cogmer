# D-289 — The view's front page lists every room with its activity, and shows the one picked beside the list

**Date:** 2026-11-20 · **Status:** active · **Areas:** view, rooms

**Decision.** `/` serves one page whatever rooms exist. It lists the rooms from a stream the daemon sends whenever a room changes and every 20 seconds, the most recently active first, each with the newest turn's writer and time, the colleagues' turns since the reader last looked in this browser, the turns no session here has been given (D-204, the view keeps the reader's place and says what the model holds), the sessions here in the room and how many colleagues are reachable. A room picked from the list opens beside it in a frame of its own page at `/room/<name>` (D-077, every room has its own address in the view), and a room's frame is hidden, not closed, while the reader reads another room, so each room holds its place. A user in several rooms asked for one page at one address that shows which rooms are active.

**Rejected.**
- *Redirecting `/` into the room when there is only one.* The same address then opens a room on one day and a list on the next, so a page a user keeps open stops being the one they kept.
- *Counting what is new on the daemon.* What a reader has looked at belongs to one browser, and the daemon would have to be told of every look.
