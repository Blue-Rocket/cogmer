# Hosting-fitness runs of 2026-11-26 with adaptive polling and nudges

Measurements 4 and 6 of `docs/work/hosting-fitness.md`, one group of four daemons in which every two share one
room, with no turns and with one turn a minute in each room, ran on 2026-11-26 on the droplet (Ubuntu 24.04,
kernel 6.8.0-149-generic, 4 vCPU, 8GB) on four builds. Every run was valid: no daemon exited, no daemon logged
"is unreachable", every turn written was delivered, and the turn written in every room at the end was delivered.

## What was run

The builds, each made by `harness.py build` from a clean worktree:

| build | what it polls |
|---|---|
| `a12ae84`, the control | every colleague address every second, on a connection kept for each address |
| `5a63da9` | the control's polls, with an address that answers nothing polled at double the last wait, up to 20s, and up to 60s after 30 minutes with no event in its rooms |
| `92b3630` | `5a63da9`, with a wait after a wake started again from 1s, which `5a63da9` did not do |
| `8671eac` | `92b3630`, with a machine that writes an event nudging the room's other machines, which poll it at once |

Every run started from a copy of the state measurement 4's fleet was set up with on 2026-11-25, so the
identities, pairings and rooms were the same, and ran as `python3 -I docs/work/hosting-fitness/harness.py run
<dir> <name>`, with `--turns` for the runs with turns. The summaries are in
`docs/work/hosting-fitness/runs-2026-11-26/`, named by build folder and run.

| run | build | from (UTC) | warm-up | sampled |
|---|---|---|---|---|
| control idle | `a12ae84` | 09:14 | 300s | 890s |
| adaptive idle | `5a63da9` | 09:35 | 300s | 890s |
| control turns | `a12ae84` | 09:56 | 300s | 890s |
| fixed turns | `92b3630` | 10:17 | 120s | 470s |
| fixed idle | `92b3630` | 10:28 | 120s | 290s |
| nudge turns | `8671eac` | 10:40 | 120s | 470s |
| nudge idle | `8671eac` | 10:50 | 120s | 290s |

The box was not quiet for every run. Test suites ran during the control idle run's sampling and a build during
the adaptive idle run's warm-up. From 10:01, four minutes into the control turns run's warm-up, the harness and
its daemons were held to vCPUs 0 and 1 with `taskset`, and every later run was started that way, while builds and
test suites ran on vCPUs 2 and 3. The box was 31% busy during the fixed idle run and 37% during the nudge idle
run, against 1% to 16% in the others.

## What was seen

The figures are for one daemon, as the mean over the four.

| run | CPU s an hour | MB sent an hour | PSS | delivery of a turn, median and 95th percentile | turns |
|---|---|---|---|---|---|
| control idle | 99.7 | 15.28 | 17.7MB | | |
| adaptive idle | 12.8 | 0.76 | 16.8MB | | |
| fixed idle | 14.5 | 0.74 | 16.4MB | | |
| nudge idle | 11.7 | 0.74 | 16.4MB | | |
| control turns | 98.5 | 16.53 | 18.8MB | 0.63s, 0.96s | 90 |
| fixed turns | 38.8 | 2.36 | 17.5MB | 10.09s, 10.14s | 49 |
| nudge turns | 23.4 | 2.92 | 17.3MB | 0.004s, 0.010s | 48 |

Each run is read as three windows, and the harness flags a run in which one daemon's CPU differs between windows
by more than 10%. The largest difference for one daemon was 9% in control idle, 10% in adaptive idle, 5% in
control turns, 26% in fixed turns, 56% in fixed idle, 9% in nudge turns and 13% in nudge idle, and the harness
flagged adaptive idle, fixed turns, fixed idle and nudge idle. The windows of the shorter runs are 97s and 157s,
in which a daemon of the adaptive builds uses about a second of CPU. In fixed idle the first window was 19.1 CPU
seconds an hour and the other two 12.2 and 12.4.

The slowest delivery of the turn written at the end in every room was 0.86s in control idle, 6.35s in adaptive
idle, 0.87s in control turns, 15.36s in fixed turns, 5.96s in fixed idle, 0.11s in nudge turns and 0.12s in nudge
idle.

## What the runs do not show

No run lasted past 30 minutes without an event, so none polled at the 60s cap. Every user had one machine, so no
run had a colleague's machine set aside or a turn relayed by another machine of its writer's user, and no run
lost a nudge. The fleet ran on one box over Docker's bridge, so a poll and a nudge crossed no real network, and
no run had the overlay on. Each build ran each measurement once.
