# Hosting-fitness runs of 2026-11-27 on the current build

Measurement 12 of `docs/work/hosting-fitness.md`, the fixed term at N = 4 and the reference user at N = 20 with
and without turns, ran on 2026-11-27 on a droplet made for the measurements (DigitalOcean Regular, 4 vCPU, 8GB,
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
