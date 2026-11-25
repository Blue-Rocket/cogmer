# Hosting-fitness runs of 2026-11-25

Measurements 0 to 9 of `docs/work/hosting-fitness.md` ran on 2026-11-25 from 10:29 to 16:37 UTC on the
droplet: Ubuntu 24.04, kernel 6.8.0-149-generic, 4 vCPU (reported as "DO-Regular"), 8GB. The build was commit
`b91c152`, built clean at 10:47 UTC by `harness.py build` as a development build with the `ts_debug_websockets`
tag. Every run was valid: no daemon exited, no daemon logged "is unreachable", and the turn written in every
room at the end of each run was delivered. Each file in `docs/work/hosting-fitness/runs-2026-11-25/` is the
`summary.json` the harness wrote for one run.

## What was run

Each run below is `python3 -I docs/work/hosting-fitness/harness.py` with the arguments shown, on a fleet made
by `fleet <dir> <shape> <n>` from the same build. A run waited 5 minutes and then sampled for 890 seconds. The
overlay was off unless the run says it was on.

| file | measurement | fleet | run |
|---|---|---|---|
| `m0-baseline.json` | 0, the box with no fleet | none | `baseline` |
| `m1-solo1.json` | 1, fixed cost | solo, N = 1 | `run` |
| `m1-solo4.json` | 1, fixed cost | solo, N = 4 | `run` |
| `m1-solo4-overlay.json` | 1, fixed cost with the overlay | solo, N = 4, `--overlay` | `run --overlay` |
| `m2-pair1.json` | 2, one colleague in one room | pair1, N = 2 | `run` |
| `m3-pair3.json` | 3, one colleague in three rooms | pair3, N = 2 | `run` |
| `m4-group4.json` | 4, the reference user, idle | group4, N = 4 | `run` |
| `m5-group4-sync5000.json` | 5, polls every 5 seconds | the group4 fleet of measurement 4 | `run --sync-ms 5000` |
| `m6-group4-turns.json` | 6, the reference user with turns | the group4 fleet of measurement 4 | `run --turns` |
| `m7-group4-overlay.json` | 7, the reference user with the overlay, idle | group4, N = 4, `--overlay` | `run --overlay` |
| `m8-group4x20-turns.json` | 8, 20 reference users with turns | group4, N = 20 | `run --turns` |
| `m9-restart.json` | 9, a start of every daemon | the fleet of measurement 8 | `restart --times 3` |

Each pairing took 0.1s to 1.3s, and 40 of the 44 took 1.0s or more.

## What was seen

The figures are for one daemon, as the mean over the daemons of the run. CPU is user and system time from
`/proc/<pid>/stat`. PSS is the mean of the readings from `/proc/<pid>/smaps_rollup`, taken every 60 seconds. Bytes are what the daemon's TCP
sockets to the box's own addresses sent, from `ss`. Disk is `write_bytes` from `/proc/<pid>/io`.

| run | CPU s an hour | PSS MB | polls made an hour | bytes sent an hour | disk written an hour | open sockets, mean |
|---|---|---|---|---|---|---|
| m1 solo, N = 1 | 4.0 | 29.7 | 0 | 0 | 0 | 0 |
| m1 solo, N = 4 | 4.1 | 12.0 | 0 | 0 | 0 | 0 |
| m1 solo, N = 4, overlay | 6.2 | 19.2 | 0 | 0 | 17KB | 1 |
| m2 pair1 | 54.1 | 25.2 | 3414 | 22.4MB | 0 | 59 |
| m3 pair3 | 90.0 | 26.2 | 3448 | 32.4MB | 0 | 59 |
| m4 group4 | 160.5 | 27.4 | 10365 | 66.1MB | 0 | 178 |
| m5 group4, 5s polls | 35.1 | 18.2 | 2146 | 14.0MB | 0 | 36 |
| m6 group4, turns | 178.3 | 28.7 | 10372 | 67.5MB | 19.8MB | 178 |
| m7 group4, overlay | 211.7 | 160.8 | not seen | not seen | 17KB | 4 |
| m8 group4, N = 20, turns | 181.2 | 23.7 | 10289 | 67.0MB | 19.5MB | 177 |

The box's CPU an hour, from `/proc/stat`, was 59s in the baseline, 762s in m4, 858s in m6 and 4064s in m8. The
harness's own CPU, which counts the `ss` calls and the hook processes that write turns, was 25s, 45s and 221s
an hour in those runs. The box's CPU less the baseline and the harness's, divided by the daemons, was 170s an
hour in m4, 189s in m6 and 189s in m8.

Turns: in m6, 90 of 90 were delivered, with a median of 0.43s and a 95th percentile of 0.95s. In m8, 450 of
450 were delivered, with a median of 0.53s and a 95th percentile of 0.97s. The slowest delivery of the final
check was 0.73s in m4 and 2.97s in m5.

With the overlay on, `ss` saw no sockets between daemons, since the overlay's connections are inside the
daemon's process. Each daemon held 1 socket to an address outside the box in measurement 1, with no rooms, and 4 in m7, and those
sockets sent 0.26MB to 0.29MB an hour and received about 0.05MB. The box's `lo` counter carried 110.9MB an
hour for each daemon in m7 against 82.4MB in m4. PSS in m7 moved between 146MB and 185MB over the run, with no
trend.

Measurement 9 started the 20 daemons of m8 together three times. Every daemon answered at `/healthz` within
0.43s, 0.39s and 0.52s, a turn written just after the start was delivered within 0.94s, 0.93s and 0.78s,
`MemAvailable` dropped by 258MB, 252MB and 246MB, and the box's busiest sample was 65%, 67% and 90% busy.

## Where the runs were not clean

- Three runs report `windows_disagree`, a figure for one daemon that differed by more than 10% between the
  three 5-minute windows. In m1 with the overlay one daemon's CPU read 5.76, 6.12 and 6.83s an hour, a
  difference of about 9 ticks of 1/100s in a window. In m7 every daemon's first window was the lowest, at
  195s to 207s an hour against 211s to 226s later. In m8 the largest difference was 12%. None was run again.
- During m3, from 12:32 to 12:53 UTC, the droplet's public interface sent 120.7MB an hour, against 1.3MB to
  6.1MB in every other run. No daemon held a socket outside the box in that run, so the traffic was another
  process's. The daemons' own figures in m3 are in line with m2's.
- In m8 the box's steal time was 3.2%, against 0.03% to 0.7% in the other runs.

## What the runs do not show

- What a colleague on another machine costs in bytes on the network. With the overlay off, every byte went
  over `lo`, and with it on, the paths stayed within the box.
- The overlay with turns, and the overlay at N = 20. Neither ran.
- Disk a turn adds to the store, and growth of memory with the store, which measurement 10 measures.
- A start after an upgrade, which would migrate the stores.
- Isolation by OS user or container, which measurement 11 measures.
- Any server but the droplet.
