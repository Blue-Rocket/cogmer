# Hosting-fitness runs of 2026-11-27 on the current build

Measurements 12, 10 and 13 of `docs/work/hosting-fitness.md`, the fixed term at N = 4, the reference user at N = 20 with
and without turns, a group of four filled to two days of turns, and ten starts of a stopped daemon, ran on 2026-11-27 on a droplet made for the measurements (DigitalOcean Regular, 4 vCPU, 8GB,
Ubuntu 24.04, kernel 6.8.0-149-generic, in SFO3) on the build `36162d5`, the head of `release-0.11.0`, which keeps
a connection for each address, polls quiet rooms less often and nudges (D-303, D-305 and D-306). Every run was
valid: no daemon exited, no daemon logged "is unreachable", every turn written was delivered, and the turn written
in every room at the end was delivered.

## What was run

The build was made by `harness.py build` from a clean worktree on the droplet the sessions run on, and copied to
the measurement droplet with the harness, whose checksum matched. Nothing else ran on the measurement droplet: it
was made for the runs, its daily package timers were stopped, and the harness ran as an ordinary user. The peer
address 172.17.0.1 was put on a dummy interface, since the droplet has no Docker. Each run was `python3 -I
harness.py run <dir> <name>`, with `--turns` for the run with turns, and waited 300s before sampling 890s. The
summaries are in `docs/work/hosting-fitness/runs-2026-11-27/`.

| run | from (UTC) | daemons |
|---|---|---|
| baseline | 21:03 | none |
| fixed | 21:18 | 4, identities and no rooms |
| turns | 21:39 | 20, five groups of four, every two in a group sharing a room, one turn a minute in each room |
| idle | 22:00 | the same 20, no turns |

The box was 0.29% busy in the baseline, and steal was under 0.7% in every run.

## What was seen

The figures for a daemon are the mean over the daemons. The box's CPU for each user is the box's CPU less the
baseline's and the harness's own, divided by the users.

| run | daemon CPU s an hour | box CPU s an hour for each user | PSS | MB sent an hour | delivery of a turn, median and 95th percentile |
|---|---|---|---|---|---|
| fixed | 4.36 | | 12.12MB | 0 | |
| turns | 25.68 | 27.3 | 13.35MB | 2.89 | 0.007s, 0.014s, 450 of 450 turns |
| idle | 13.44 | 15.7 | 11.66MB | 0.78 | |

`MemAvailable` was 264MB lower at the start of the turns run than in the baseline, 13.2MB for each user. A daemon
with turns wrote 21.5MB an hour to disk. Its resident memory was 38.5MB and its PSS 13.35MB, because the 20
processes share the program's pages.

Each run is read as three windows. The largest difference in one daemon's CPU between windows was 17% in the
fixed run, 20% in the turns run and 21% in the idle run, and the median 13%, 13% and 16%, so the harness flagged
all three. A window is 297s, in which a fixed daemon uses about 0.36 CPU seconds.

The slowest delivery of the turn written at the end in every room was 0.62s in the turns run and 0.14s in the
idle run.

The fixed run had four daemons and the others twenty, so the PSS of a fixed daemon at N = 20, where more
processes share the program's pages, is lower than 12.12MB and was not measured. Every user had one machine, the
overlay was off, and the fleet ran on one box, so no poll or nudge crossed a real network. Each run was made once.

## A store that grows, and a machine started when needed

Measurement 10 made a group of four with `harness.py fleet <dir> group4 4`, whose first attempt at 22:21 failed when
one daemon's `create` returned "founding the room: database is locked (5) (SQLITE_BUSY)", and whose second
succeeded. From 22:35 `harness.py fill` wrote turns into the six rooms, six writers at once, to a quarter, a half,
one and two days of the reference user's turns, 1440 a day in each room, and at each checkpoint waited for a turn
in every room to be delivered and sampled the daemons idle for 300s. No turn failed. The results are in
`growth-fill.json`.

| days | turns for each daemon | hook's time, median and 95th percentile | idle CPU s an hour | idle PSS | room databases | all files | slowest delivery |
|---|---|---|---|---|---|---|---|
| 0.25 | 1,083 | 0.162s, 0.310s | 12.29 | 43.0MB | 8.5MB | 24.8MB | 0.88s |
| 0.5 | 2,163 | 0.208s, 0.354s | 14.83 | 55.7MB | 17.0MB | 33.5MB | 1.00s |
| 1 | 4,323 | 0.369s, 0.577s | 16.11 | 67.9MB | 34.3MB | 51.2MB | 1.81s |
| 2 | 8,643 | 0.712s, 1.154s | 20.83 | 89.7MB | 68.7MB | 86.5MB | 3.23s |

The hook's time is the time `cogmer hook prompt` and `cogmer hook stop` took together for one turn while the six
rooms were written at once. The harness flagged the idle samples at a half, one and two days for windows that
differed by more than 10%.

Measurement 13 then ran `harness.py wake <dir> m13 --as-left` on those stores: ten times, one daemon was stopped,
a colleague wrote a turn in a room they share, the daemon stayed stopped for 30s and was started. The results are
in `wake.json`.

| | median | least | most |
|---|---|---|---|
| until the daemon answered at `/healthz` | 0.089s | 0.058s | 0.105s |
| until its peer port listened | 0.092s | 0.058s | 0.105s |
| until the turn written while it was stopped was in its store | 0.96s | 0.84s | 1.81s |
| until a turn it wrote reached the colleague | 0.68s | 0.59s | 1.39s |
| PSS a minute after the start | 74.4MB | 43.8MB | 86.6MB |
| disk while stopped | 74.8MB | 74.7MB | 74.8MB |

The baseline taken after the runs found the box 0.32% busy.
