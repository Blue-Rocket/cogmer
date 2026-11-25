# Hosting-fitness fills of 2026-11-25: an idle poll's cost with a day of turns

Three fills ran on 2026-11-25 on the droplet (Ubuntu 24.04, kernel 6.8.0-149-generic, 4 vCPU, 8GB), each on a
copy of the state of measurement 2's fleet: two daemons, paired, sharing one room, with the overlay off. Two
ran on the build of `b91c152`, from 19:13 and 19:18 UTC. The third ran from 19:43 UTC on `998deae` with the
change to `cmd/cogmer/store.go` that the commit after this one makes, built by `harness.py build` from the
working tree. Every fill's final turn was delivered and no daemon exited. The results are
`fill-pair1-day1-a.json`, `fill-pair1-day1-b.json` and `fill-pair1-day1-fixed.json` in
`docs/work/hosting-fitness/runs-2026-11-25/`.

## What was run

Each fill was `python3 -I docs/work/hosting-fitness/harness.py fill <dir> <name> --days 1`, with `--idle 30` for
the first and `--idle 300` for the other two. The harness started both daemons and wrote turns into the room,
alternating between the two daemons, as fast as the hooks took them, until the room held 1440. It then wrote one
more turn and waited for its delivery, waited 60 seconds, read the sizes of each daemon's files, and sampled both
daemons idle for the time given.

While the daemons of the second fill's state were started again, one of them ran under `strace -f -e
trace=openat,unlinkat,unlink` for under a minute. The lines of that trace that name SQLite's temporary files are
`fill-pair1-trace-tempfiles.txt`.

## What was seen

The figures are for one daemon, as the mean over the two.

| | first, on `b91c152` | second, on `b91c152` | third, with the change |
|---|---|---|---|
| time to write 1440 turns | 193s | 187s | 174s |
| a turn's two hooks, median and 95th percentile | 0.13s and 0.22s | 0.13s and 0.21s | 0.11s and 0.21s |
| idle CPU s an hour | 598 | 530 | 76 |
| idle disk written an hour | 29.1GB | 30.0GB | 0 |
| idle PSS | 62.9MB | 60.2MB | 39.9MB |
| delivery of the final turn | 1.56s | 1.65s | 1.26s |

With the room empty, measurement 2's daemons took 54.1 CPU seconds an hour and 25.2MB of PSS, and wrote nothing.

The disk writes were steady: one daemon of the second fill wrote between 247MB and 258MB in each 30 seconds of
its 5 idle minutes. The traced daemon created and removed 11 files named `/var/tmp/etilqs_*`, which is the name
SQLite gives a temporary file.

Each daemon's files grew from about 0.4MB to between 20.5MB and 20.8MB over the 1441 turns: 11.5MB to 12.0MB of
room database, 4.2MB to 4.4MB of its write-ahead log, 4.3MB of membership database, and 0.24MB of other
files.

## What the fills do not show

- More than one room, more than two daemons, or more than a day of turns.
- What the remaining 22 CPU seconds an hour above the empty room's are spent on with the change.
- Whether the write-ahead log or the membership database keeps growing, and what the membership database holds
  for each turn.
- The overlay.
