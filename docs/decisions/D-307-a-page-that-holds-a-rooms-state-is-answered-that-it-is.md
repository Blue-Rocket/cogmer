# D-307 — A page that holds a room's state is answered that it is unchanged

**Date:** 2026-11-26 · **Status:** active · **Areas:** view, transport

**Decision.** The static page that reads a cloud session's room (D-302, a cloud session's room is read on a static page, the daemon's own view served from GitHub Pages) asks for the newest page of a room (D-308, the static page is sent a page of a room's turns and asks for the earlier ones) with the hash of the one it holds. When the page has not changed, the daemon answers `{"unchanged": true}` and the current age of the lists, a few bytes, and answers the page when it has. The hash covers the state without the lists, whose age moves while a room sits quiet, so the page puts the age into the state it holds. A page can come to 300 KB and the page asks every three seconds, which is up to 360 MB an hour for a page left open, most of it a page the browser has.

**Rejected.**
- *A long poll or a push.* The port speaks one request and one answer in lines (D-301, a cloud session's daemon serves its room on a second overlay port, in lines, with a token made from the cloud key), and a held request needs a way to tell a slept daemon from a quiet room.
- *A sequence number in place of a hash.* A number is true only of what it counts, and the state holds more than the room's events, such as who is online and the guest list. A hash covers whatever the page shows.
- *Leaving the lists' age in the hash.* The hash would differ at every poll and no answer would be short.

**Limits.** The daemon still builds the whole snapshot to hash it, which cost 36 ms for 300 turns in a run on 2026-11-26, so the saving is bytes sent and not the daemon's work. A change that moves only a field outside the hash, other than the lists' age, is not seen by a page until the room changes, so a field that moves on its own belongs in the answer beside the age.

**Revisit when** a quiet room's snapshot costs the daemon more than the bytes cost the page.
