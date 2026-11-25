# What a day of turns takes in a daemon's store, read on 2026-11-25

The store of daemon `u00` from the third fill in `docs/work/hosting-fitness-runs-2026-11-25-fill.md`, two daemons
sharing one room filled with 1441 turns on the build of `998deae` with the change to `cmd/cogmer/store.go` that
`a14972e` commits, was read on 2026-11-25 after both daemons had stopped. The room database and the membership
database with its log were copied to a scratch directory, and `docs/work/hosting-fitness/storesize.py` read the
copies with Python's `sqlite3` module (SQLite 3.45.1), using its `dbstat` table for the space of each table and
index. Its output is `docs/work/hosting-fitness/runs-2026-11-25/store-pair1-day1-fixed.txt`.

## What was seen

The files after the daemons stopped: a room database of 12,115,968 bytes with no log beside it, and a membership
database of 172,032 bytes with a log of 4,120,032 bytes. While the daemons ran, the room database's log was 4.2MB
to 4.4MB (`a6ac576`).

Space in the room database, in 4096-byte pages:

| table or index | bytes |
|---|---|
| `events` | 9,854,976 |
| `pending_injection` | 1,073,152 |
| `idx_events_room_signer` | 356,352 |
| `sqlite_autoindex_events_2`, the index of `UNIQUE(signer, peer_sequence)` | 208,896 |
| `idx_events_signer`, on `(signer, peer_sequence)` | 208,896 |
| `idx_events_room` | 180,224 |
| `sqlite_autoindex_events_1` | 143,360 |
| the other tables and indexes | 53,248 |
| free pages | 36,864 |

The `events` table held 2884 events: 1441 prompts with 500 bytes of content and 96 of metadata, 1441 replies with
4000 bytes of content and 28 of metadata, and 2 invitations. Averaged over the events, the columns held 2248 bytes
of content, 571 of `chain`, 86 of signature, 62 of metadata, 51 each of `peer_id` and `signer`, 43 of `room_id`,
34 of `event_id`, 30 of timestamp, and 51 in the other columns together. Every event in the room has the same
`room_id`, and each signer's events carry its chain.

`pending_injection` held 20 rows, all for one session, and each row's `event_ids` was 51,838 bytes, about 1440
identifiers. `session_delivered` held no rows. The harness writes each transcript by hand without the context a
prompt was given, so no offered turn was ever counted as delivered, and each prompt offered every turn the other
daemon had written. `PrunePending` keeps at most 20 rows for each session (`maxPendingPerSession` in
`cmd/cogmer/daemon.go`).

## What the reading does not show

- How well a turn's text compresses. The harness fills each turn with random lowercase letters and spaces, which
  compress far less than prose.
- `pending_injection` in a session whose host confirms delivery, which is every session the harness does not
  write.
- Whether a log grows past about 4MB when a reader holds a checkpoint back, and the log of a room database that
  was not closed cleanly.
- More than one room, or more than a day of turns.
