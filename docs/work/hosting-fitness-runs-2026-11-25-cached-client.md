# Hosting-fitness run of 2026-11-25 with a client kept for each colleague

Measurement 4 of `docs/work/hosting-fitness.md`, one group of four daemons in which every two share one room
and no turns are written, ran twice on 2026-11-25 on the droplet (Ubuntu 24.04, kernel 6.8.0-149-generic, 4
vCPU, 8GB). The first run, from 17:28 UTC, was on a build in which each daemon keeps one HTTP client for each
colleague address across polls. The second, from 17:49 UTC, was a control on the unchanged build of `b91c152`
that measurement 4 ran on that morning. Both runs were valid: no daemon exited, no daemon logged "is unreachable", and the turn written
in every room at the end was delivered.

## What was run

The changed build is `f0023cd` with the edit in `docs/work/hosting-fitness/runs-2026-11-25/m4c-cached-client.diff`,
built by `harness.py build` from a worktree holding that edit. In the unchanged code `pullFrom` in
`cmd/cogmer/sync.go` calls `clientFor` in `cmd/cogmer/transport.go` at every poll, which builds a new HTTP
client, so each poll opens a new connection and makes a TLS handshake. The edit stores the first client built
for an address in a map on the daemon and uses it for every later poll of that address. Before the run, the
tests matching `Sync`, `Peer`, `Pull` or `Room` passed on the edit with `go test -short`.

Both runs started from a copy of the state measurement 4's fleet was set up with, so the identities,
pairings and rooms were the same. Each ran as `python3 -I docs/work/hosting-fitness/harness.py run <dir>
<name>`, waited 5 minutes and sampled for 890 seconds. The summaries are `m4c-group4-cached-client.json` and
`m4c-group4-control.json` in `docs/work/hosting-fitness/runs-2026-11-25/`.

## What was seen

The figures are for one daemon, as the mean over the four. The harness counts a poll when a daemon opens a new
connection to a colleague, so with the client kept its count is of handshakes, and it does not count the polls
made on a connection already open.

| | control | client kept |
|---|---|---|
| CPU s an hour | 168.8 | 89.8 |
| the box's CPU an hour for each daemon, less the baseline and the harness's own | 179 | 98 |
| bytes sent an hour | 65.7MB | 15.3MB |
| new connections an hour | 10297 | 0 |
| open sockets, mean | 177.8 | 6.0 |
| PSS | 27.6MB | 17.5MB |
| the slowest delivery of the final check | 0.94s | 1.04s |

The control's figures are within 6% of measurement 4's of the morning, which were 160.5 CPU seconds an hour,
66.1MB an hour and 27.4MB. The control's windows disagreed: every daemon's first 5-minute window was the
lowest, at 156s to 163s an hour against 168s to 177s in the later two. The kept-client run's windows agreed
within 10%. Steal time was 0.56% in the kept-client run and 0.76% in the control.

## What the run does not show

- How many polls the kept-client run made. The interval was the same and every room delivered, but a poll on
  an open connection is not counted, so the cost of one poll with the client kept is the CPU and bytes divided
  by the polls of the control, on the assumption that the counts were equal.
- What the remaining 90 CPU seconds an hour are spent on.
- The overlay, turns, or more than four daemons with the client kept.
- What happens to a kept connection when a colleague's machine is removed or this machine's identity changes,
  which the edit does not handle.
