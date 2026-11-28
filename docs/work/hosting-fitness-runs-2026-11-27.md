# Hosting-fitness runs of 2026-11-27 and 2026-11-28 on the current build

Measurements 12, 10 and 13 of `docs/work/hosting-fitness.md`, the fixed term at N = 4, the reference user at N = 20 with
and without turns, a group of four filled to two days of turns, and ten starts of a stopped daemon, ran on a droplet made for the measurements (DigitalOcean Regular, 4 vCPU, 8GB,
Ubuntu 24.04, kernel 6.8.0-149-generic, in SFO3) on the build `36162d5`, the head of `release-0.11.0`, which keeps
a connection for each address, polls quiet rooms less often and nudges (D-303, D-305 and D-306). Measurement 12 ran
on 2026-11-27, and measurements 10 and 13 on 2026-11-28. Every run of measurement 12 was valid: no daemon exited, no
daemon logged "is unreachable", every turn written was delivered, and the turn written in every room at the end was
delivered.

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

The box was 0.29% busy in the baseline, and steal was under 0.7% in every run. A baseline taken at the end of
2026-11-27 found the box 0.32% busy.

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

Measurement 10 made a group of four with `harness.py fleet <dir> group4 4` at 11:52 UTC on 2026-11-28, which
succeeded at the first attempt. From 11:52 to 12:31 `harness.py fill` wrote turns into the six rooms, six writers at
once, to a quarter, a half, one and two days of the reference user's turns, 1440 a day in each room, and at each
checkpoint waited for a turn in every room to be delivered and sampled the daemons idle for 300s. Each turn was
written as Claude Code writes one: `cogmer hook prompt`, then the user's prompt and the block the hook gave appended
to the session's transcript, which holds every turn of that session, then `cogmer hook stop`. The results are in
`growth-fill.json`.

The fill delivered every turn and confirmed it. No turn failed and no daemon exited. Every daemon logged "delivery:
confirmed", about 2,280 times by two days, and recorded about 8,600 events as delivered to its sessions (§19,
incremental context injection), and at two days no session had more than five events of its colleague not yet
confirmed. One daemon logged "is unreachable" once, at 11:52:46, for a colleague whose daemon started at 11:52:47.

| days | turns for each daemon | prompt hook, median and 95th percentile | stop hook, median and 95th percentile | idle CPU s an hour | idle PSS | room databases | all files | slowest delivery |
|---|---|---|---|---|---|---|---|---|
| 0.25 | 1,083 | 0.055s, 0.108s | 0.067s, 0.126s | 12.97 | 38.0MB | 7.5MB | 23.7MB | 0.95s |
| 0.5 | 2,163 | 0.065s, 0.136s | 0.097s, 0.185s | 15.23 | 57.2MB | 15.4MB | 31.8MB | 1.30s |
| 1 | 4,323 | 0.080s, 0.141s | 0.143s, 0.233s | 17.22 | 63.6MB | 31.5MB | 48.1MB | 2.53s |
| 2 | 8,643 | 0.115s, 0.199s | 0.247s, 0.386s | 21.16 | 111.4MB | 63.0MB | 80.2MB | 4.15s |

A hook's time is the time `cogmer hook prompt` or `cogmer hook stop` took for one turn while the six rooms were
written at once on the four vCPUs. The stop hook reads the session's transcript, and each transcript held 6.3MB at
two days. The harness flagged the idle samples at every checkpoint for windows that differed by more than 10%.

Measurement 13 then ran `harness.py wake <dir> m13 --as-left` on those stores, from 12:31 to 12:47: ten times, one
daemon was stopped, a colleague wrote a turn in a room they share, the daemon stayed stopped for 30s and was
started. The results are in `wake.json`.

| | median | least | most |
|---|---|---|---|
| until the daemon answered at `/healthz` | 0.112s | 0.062s | 0.151s |
| until its peer port listened | 0.112s | 0.062s | 0.221s |
| until the turn written while it was stopped was in its store | 1.88s | 0.86s | 2.61s |
| until a turn it wrote reached the colleague | 0.77s | 0.62s | 1.41s |
| PSS a minute after the start | 60.5MB | 42.1MB | 85.4MB |
| disk while stopped | 68.7MB | 68.6MB | 68.8MB |

The PSS a minute after the start fell into two groups: 42.1MB to 47.9MB in five starts, and 73.0MB to 85.4MB in
the other five.
