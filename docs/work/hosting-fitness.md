# Hosting fitness

Material for the open item "Nobody has measured what many hosted machines cost one server" in
`docs/open.md`. The phase is to identify options, measure their costs, and find out whether the daemon's
architecture suits hosting many users' machines (D-285, a user with no computer of their own has a machine
hosted for them; D-287, a user with a computer of their own can also have a machine hosted). Delete this with
the item.

Written 2026-11-24. Prices and limits below are from web searches on that date, and the sources disagree
with each other in places, so each is checked on the provider's own page before it is relied on.

## What is known

Measured or read:

- A Linux daemon on the droplet (Ubuntu 24.04, 8GB, 4 vCPU, 77GB disk, $48 a month) held 44MB resident after
  3.7 days up, in light use. A daemon on a Mac held about 100MB with four real rooms after five days up.
  Neither is a loaded daemon.
- Each daemon polls every peer address it syncs with, in a loop of its own for each address, every
  `COGMER_SYNC_MS` milliseconds, which defaults to 1000, and an address that does not answer is polled less
  often, up to every 15 seconds (`RunSync` in `cmd/cogmer/sync.go`, `syncInterval` in `cmd/cogmer/main.go`,
  read on 2026-11-24).
- The hook and page address must be loopback, or the daemon refuses to start (`cmd/cogmer/main.go`, the check
  that cites §25, read on 2026-11-24). A hosted machine's page therefore needs a proxy in front of it, or a
  change to that rule.
- A daemon serves one user, whose state is in one directory named by `COGMER_HOME`. The open item "Nothing
  decides how hosted machines run or how each one's page gets its address" lists the choices of process for each
  user or one process for many, isolation, always running or on demand, storage, and TLS, each with its cost, and
  this document does not repeat them.
- Release builds exist for `linux/arm64` and `linux/amd64` (`scripts/release.sh`), so ARM servers can run it.

Not known: what a daemon costs under load, how load grows with the number of daemons on one box and with the
colleagues each has, and what the relay carries.

## Servers

| option | spec | price | note |
|---|---|---|---|
| Hetzner CAX21 | 4 vCPU, 8GB, 80GB, ARM | about €10.49 a month | https://trackvps.com/hetzner-cloud-pricing; about 30% up on new orders from 2026-06-15 |
| Hetzner CAX31 | 8 vCPU, 16GB, ARM | about €20.99 a month | same source; ARM only in Germany and Finland; the tier was restock-gated |
| Contabo Cloud VPS M | 6 vCPU, 16GB, 400GB | about $10.50 to $14.99 a month | https://findstack.com/resources/16gb-ram-vps; sources disagree, performance unknown |
| Leaseweb VPS 2 | 6 vCPU, 16GB | about $8.85 a month | one source |
| DigitalOcean 16GB | not recorded | about $96 a month | one comparison |
| Vultr 16GB | not recorded | about $80 a month | one comparison |
| Oracle Always Free Ampere A1 | 2 OCPU, 12GB, or 4 and 24GB | free | reports say it was cut to half in 2026, and instances with 95th-percentile CPU under 20% for seven days count as idle (https://docs.oracle.com/iaas/Content/FreeTier/resourceref.htm), which an idle daemon fleet would meet |
| the droplet | 4 vCPU, 8GB | $48 a month | the box the measurements start on |

On memory alone, at 45MB to 100MB for each daemon, an 8GB box holds roughly 70 to 150 daemons, and the open
question is what limits it first.

## What would limit a box

In the order I expect, which the measurements test:

1. Polling. N hosted machines each with k colleague machines make about N×k requests a second at the default
   interval. Where the colleagues' machines are other hosted machines on the same box, the requests need not
   leave it. Where they are not, each is a request through the relay or a direct connection.
2. Idle connections. Each daemon holds its own overlay connection, and each costs keepalive traffic.
3. Relay capacity. The public relays are free and rate-limited (D-192, the public relay is an accepted
   dependency). A relay cogmer runs would be cogmer's bandwidth to pay for.
4. Memory with room count and room size. One SQLite file for each room.
5. Upgrades. Every store migrates when its daemon starts, so an upgrade of a box starts every daemon.
6. Isolation overhead. An OS user for each daemon costs nothing, and a container or microVM for each costs
   memory I have not measured.

## Measurements

Each on one box, against the number of daemons: 1, 10, 50, 100, and more until something fails. Each daemon has
its own `COGMER_HOME` and its own pair of loopback ports and a throwaway user identity, made with the identity
page's own requests as `docs/work/host-observer/probe.py` does.

| measurement | what it answers |
|---|---|
| resident memory, CPU and bytes sent and received for each daemon, idle, with no peers | the fixed cost |
| the same with k = 1, 3 and 10 colleagues, in rooms, with `COGMER_SYNC_MS` at 1000, 5000 and 15000 | how load grows with peers and the poll interval |
| co-hosted daemons paired with each other, with the overlay on, and the path each uses | whether co-hosted machines avoid the relay |
| turns written at a steady rate, with delivery time to the colleague | what the interval costs in latency |
| startup time and resident memory against room count and room size | the cost of a heavy user |
| every daemon restarted at once | the cost of an upgrade or a reboot |
| disk used for each user after a day | storage |
| the same measurements through an OS user for each daemon and through a container for each | the cost of isolation |

Each result goes in a commit that says when, on what machine and build, and what was seen (W-91 in
`docs/writing.md`, a commit that records evidence says how it was run), and the item is rewritten from them.

## Order

1. The polling and idle measurements on the droplet, which is paid for already. They show quickly whether load
   grows with N×k as the code suggests.
2. The co-hosted and interval measurements, which show whether the interval or a direct path is the lever.
3. A cheaper server of another provider, repeating step 1, to see whether the droplet's numbers carry to an
   ARM box.
4. Isolation overhead.

A change the numbers point to, such as a longer interval for a peer that has sent nothing, a long poll, or a
connection that carries the peers of many daemons, is a decision for its own entry after the measurements, not
before.

## How to measure

What a harness does for each measurement in the table above, from the code and the probe scripts as read on
2026-11-24. Nothing here has been run. Anything the measurements could change is a question at the end, and no
topology or fix is chosen here.

Each daemon on the box is the program run as `cogmer daemon` in a process of its own, with three variables set:
`COGMER_HOME` names a directory made for it, and `COGMER_ADDR` and `COGMER_PEER_ADDR` name two loopback ports.
The hook and page address must be loopback or the daemon stops (`runDaemon` in `cmd/cogmer/main.go`). The
peer address may be any address, and here it is loopback too. `COGMER_PREFLIGHT=off` keeps the behavior checks out of the numbers. The
harness also removes every variable that starts with `CLAUDE_CODE_`, `CODEX_`, `COGMER_` or `CLAUDECODE` from
what it inherits, as `env` in `docs/work/host-observer/colleague.py` does, because the harness runs inside a
Claude Code session whose variables would otherwise reach every daemon.

The ports come from 20000 to 21999: daemon i, counting from 0, listens for hooks and the page on 20000 + 2i
and for colleagues' machines on 20001 + 2i, which leaves room for 1000 daemons. On 2026-11-24 at 21:10 UTC
`ss -ltnp` showed listeners on 22, 80, 53 (the resolver, on 127.0.0.53 and 127.0.0.54), 4782 and 4783 (the
claude account's daemon, pid 176631), 8767 on the droplet's public address, and 34417 and 45813 on loopback,
and nothing from 20000 to 21999. The kernel hands out outgoing ports from 32768 to 60999
(`/proc/sys/net/ipv4/ip_local_port_range`, read the same day), so the chosen range never meets a port the
daemons' own connections take. The probe scripts' defaults, 47792 and 47793, are inside that range, and the
harness does not use them. The harness runs `ss -ltnp` before each run and stops if any port in the range is
taken.

The build is a development build of the commit under test:

```
CGO_ENABLED=0 go build -tags ts_debug_websockets -ldflags "-s -w" -o <run dir>/cogmer ./cmd/cogmer
```

It carries no version, so it reports `dev` and refuses the account's real `~/.cogmer` whatever `COGMER_HOME`
says (D-265, a development or test build never opens the account's real state directory). It differs from a
release build only in the version, since `scripts/release.sh` builds with the same tag and the same `-s -w`, so one run at 1 and 10
daemons with a release build of the same commit checks that the numbers carry over. Every daemon runs the one
file, so the kernel shares the program's pages among them.

D-265's refusal keeps a test daemon away from the real directory and not from the real daemon. A command that runs
without `COGMER_ADDR` talks to 127.0.0.1:4782 (`addr` in `cmd/cogmer/main.go`), which is the claude account's
daemon, and that daemon would act on the user's real state. The harness therefore passes all three variables
to every command, never names 4782 or 4783, and after each start checks with `ss -ltnp` that the process it
started holds both of its ports. A check of the page alone is not enough: a development daemon that finds its
port already served by another daemon logs "a cogmer daemon is already serving" and exits, because
`shouldReplace` in `cmd/cogmer/main.go` never replaces when either side is `dev`, and the page that answers is
the other daemon's. The harness starts each daemon in a new session with its output sent to `daemon.log` in
its `COGMER_HOME`, the file that `keepDaemonLogBounded` in `cmd/cogmer/daemonlog.go` keeps under 10MB, and
stops it with `cogmer stop` under the same variables, or with SIGTERM to the process.

`COGMER_TAILCAT=off` turns off the overlay, the encrypted network that reaches a machine on another network
through a relay (`tailcatEnabled` in `cmd/cogmer/tailcat.go`). With it off, the address a daemon gives in its
pairing string is its bound peer address (`AdvertisedEndpoint` in `cmd/cogmer/transport.go`), so pairing and
sync between daemons on the box go over loopback TCP and nothing reaches a relay. With it on, which is the
default, the pairing string carries the overlay address, and every sync request goes through an overlay
client. Each run says which it used.

### Identities, pairing and rooms

Each daemon gets a throwaway user identity through the identity page's own two requests, as `make_identity`
in `docs/work/host-observer/probe.py` does: a POST of `{}` to `/identity/phrase`, which answers with the
words, the positions to confirm and a `createId`, then a POST to `/identity/create` with the `createId`, the
confirmed words by position, a `displayName` and a `machineName`. Both go to the daemon's `COGMER_ADDR` with
the headers `Content-Type: application/json` and `X-Cogmer: 1`. The recovery phrase is never shown or kept.

Pairing two daemons needs the following, from `runPair` and `beginCeremony` in `cmd/cogmer/main.go` and the
verification in `cmd/cogmer/verify.go`:

- Both daemons running, each with a user identity, since `runPair` stops without one (D-230, pairing is
  between users).
- Each side's pairing string, which `cogmer whoami` prints on the line after "Send your colleague this"
  (`pairing_string` in `probe.py`).
- Both sides running `cogmer pair <the other's string> <name> --terminal` at the same time, within the 90s of
  `verifyTimeout`. Each prints its two words on the line before "say those two words aloud", and the harness
  compares the two and writes `y` to both when they match, as `pair` in `probe.py` does.

A name must be given, or `runPair` asks for one. Only one side needs an address the other can reach. No
pairing has been timed on the droplet. With the overlay on, the first contact between two daemons builds a
tunnel through the relay, which the comment in `pullFrom` in `cmd/cogmer/sync.go` describes as taking
seconds. The harness times the first ten pairings and estimates the rest from them before it starts a large
fleet. It pairs disjoint pairs at the same time. One daemon's pairings run one after another until a run
shows that a daemon can hold two ceremonies at once, which the code read did not settle.

Rooms are formed as `host` and `join` in `colleague.py` do, with `COGMER_HOST=claude` and a
`CLAUDE_CODE_SESSION_ID` of each daemon's own: the host runs `cogmer create`, which prints "created
<room>", and `cogmer invite <name>` for each guest, which prints "Told them", and each guest runs `cogmer join
<room>`, which prints "joined <room>". A daemon with k colleagues is one of a room of k + 1 daemons, every one
of which polls every other, so N daemons form N / (k + 1) rooms. Each room asks for k pairings if every
guest needs to be paired only with the host, and for k(k + 1)/2 if every two members must be paired. At
k = 10 and 99 daemons that is 90 pairings or 495.

Once a fleet is set up, the harness stops it and keeps a copy of every `COGMER_HOME`, so each repeat starts
from the same state without pairing again.

### What is read from each daemon, and how often

A sampler, one process for the whole fleet, writes one line for each daemon at each sample, with the time,
and records its own CPU time too, since the sampler also loads the box. CPU time is fields 14 and 15 of
`/proc/<pid>/stat`, user and system time in ticks of 1/100s (`getconf CLK_TCK` gave 100 on 2026-11-24), read
every 10 seconds. Resident memory is `VmRSS` and `Threads` in `/proc/<pid>/status`, also every 10 seconds.
Every 60 seconds the sampler reads `Pss` from `/proc/<pid>/smaps_rollup`, which divides each shared page among
the processes that share it. RSS counts the shared program pages in every daemon, so a box's capacity is
judged by PSS and by the drop in `MemAvailable` in `/proc/meminfo` from before the fleet starts, and RSS only
for the one daemon. `read_bytes` and `write_bytes` in `/proc/<pid>/io` give the disk traffic. The box's own
`/proc/stat`, including its steal time, and its load average are read with each sample, because the droplet's
vCPUs are shared with other customers.

Linux keeps no count of network bytes for each process. `/proc/<pid>/net/dev` gives the counters of the
network namespace, which every process on the box shares. What the harness uses instead:

- `ss -tinpH`, at each sample, lists every TCP socket with the process that owns it, its remote address and
  its `bytes_sent` and `bytes_received`, and needs no root for the account's own processes. The harness sums
  these for each daemon, split by remote address: one in 127.0.0.0/8 is a co-hosted daemon over plain TCP or
  a command talking to the page, and any other is the relay or, at a first start, the relay map. Bytes on a
  relay socket carry every colleague's traffic through that relay together, with the overlay's own
  keepalives, and cannot be divided among them. UDP, which the overlay's direct path uses, is not listed.
  A socket opened and closed between two samples is missed. `pullFrom` in `cmd/cogmer/sync.go` builds a new
  HTTP client for every poll, so each poll opens a new connection with a TLS handshake of its own, and the
  connection a poll leaves idle closes after 30 seconds (`clientFor` in `cmd/cogmer/transport.go`). A poll's
  socket therefore lives at least 30 seconds, and a sample every 10 seconds sees it. The self-check that `RunSelfWatch` in
  `cmd/cogmer/ui.go` makes every 12 to 15 minutes opens a new overlay client and closes it, and is missed.
- The box's counters for `lo` and the public interface in `/proc/net/dev`, read before and during each run.
  They include the claude account's daemon, this Claude Code session and Docker, so the run's share is the
  difference from a baseline taken with the fleet stopped. They count UDP, which `ss` does not show.
- An eBPF count of bytes by process and by kind of address on `tcp_sendmsg`, `tcp_recvmsg`, `udp_sendmsg`
  and `udp_recvmsg` would cover UDP for each daemon. `bpftrace` is at `/usr/bin/bpftrace` on the droplet, but
  it needs root, and `sudo` there asks for a password, so this needs the user.

Which path a pair of co-hosted daemons uses under the overlay is read from these counters: bytes on relay
sockets that grow with turns mean the relay carries them, and UDP bytes on `lo` or the public interface that
grow while relay bytes do not mean a direct path. cogmer logs nothing that names the path a client took, in
the code read.

### Each measurement

Each runs at N = 1, 10, 50 and 100 daemons and then upward in steps of 50 until something fails, unless it
says otherwise. Each point is run three times, and the result is the median with the spread. Three runs that
disagree by more than the difference being measured are run again. The plan sets no pass or fail for any
measurement. In every one, a daemon that exits, a port refused, or a turn not delivered within ten intervals
is recorded as the point where the fleet failed, and the threshold for what a box should hold is a question
below.

1. Fixed cost, idle with no colleagues. Varied: N, and the overlay on and off. Held: no rooms, the default
   interval, the same build. Each run waits 5 minutes after the last daemon starts and then samples for 30
   minutes, which covers two of the overlay's self-checks, made every 12 to 15 minutes. The answer is memory (PSS and the drop
   in `MemAvailable`), CPU seconds an hour and bytes an hour, each for one daemon, plotted against N. A
   straight line means a fixed cost for each daemon, and its slope is that cost. The daemon's other loops
   run even when idle: images every 2 seconds, identity records every 5 and the user's own lists every 10
   (`runDaemon` in `cmd/cogmer/main.go`).
2. Polling. Varied: k at 1, 3 and 10, and `COGMER_SYNC_MS` at 1000, 5000 and 15000. Held: N at 10 and 50 (and
   100 where measurement 1 says it fits), the overlay off, no turns. Each run waits 5 minutes after the rooms form
   and samples for 15. The answer is CPU and bytes for one daemon against k / interval, the polls it makes
   in a second. A poll of one colleague's machine is one request for each room the two share (`pullFrom` and
   `pullRoom` in `cmd/cogmer/sync.go`), so with one room for each daemon the requests a second on the box
   are N × k / interval. A straight line in that number means the code's model holds, and its slope is the
   cost of one request, which includes a new connection and its TLS handshake. At 15000 a machine that does not answer is polled as often as one that does, since
   `syncPeer` caps its backoff at 15 seconds.
3. Co-hosted daemons under the overlay. Varied: the overlay on or off. Held: N at 2 and 10, k = 1, the
   default interval, one turn a minute in each room. Each run samples for 15 minutes after a 5-minute wait.
   The answer is the bytes on relay sockets against UDP bytes, against the same run with the overlay off,
   which says whether co-hosted machines avoid the relay. Each overlay client is a WireGuard stack of its own
   in the process, kept for as long as the daemon runs (`tailcatDialer` in `cmd/cogmer/tailcat.go`), so this
   run also records memory against k under the overlay, which measurement 1 does not reach.
4. Turns at a steady rate. Varied: the interval at 1000, 5000 and 15000. Held: N = 10, k = 1, the overlay off,
   then once on. Each room gets 30 turns, one every 20 seconds, written through `cogmer hook prompt` and
   `cogmer hook stop` with a hand-made transcript, as `say` in `colleague.py` does. Each turn is timed from
   the return of the stop hook on the writing daemon to the turn's appearance in `/events?room=<room>` on
   the reading one, which the harness asks for the room's events every 100 ms. The answer is the median and the 95th percentile of
   that time against the interval.
5. Startup against what a user holds. Varied: rooms at 1, 10 and 50, and turns in each at 100, 1000 and 10000.
   Held: one daemon, the overlay off. Five starts each. The answer is the time from the start of the process
   to the first answer at `/healthz`, and PSS one minute after. Writing 10000 turns through the hooks is
   slow, and whether a faster way gives the stores the same shape is a question below.
6. Every daemon restarted at once. Varied: N, and whether the program is the same or a newer one over state
   an older program wrote (an upgrade, which migrates every store). Held: k = 3, the default interval. Three
   restarts each. The answer is the time until every daemon answers, the time until a turn written just
   before the restart reaches every colleague, and the peak CPU and `MemAvailable` drop while they start.
7. Disk. One run of 24 hours at N = 10 and k = 3, with one turn every 5 minutes in each room. The answer is the
   size of each `COGMER_HOME` with `du`, which includes its log, after 24 hours, and the PSS of each daemon
   over the day, which shows whether memory grows past the first hour.
8. Isolation. Measurements 1 and 2 at N = 10 and 50, repeated with each daemon run as an OS user of its own and with
   each in a container of its own, built `FROM scratch` as the container tests' images are. The answer is
   the difference in memory and CPU for one daemon from the plain processes. An OS user for each daemon
   needs root to make the users. Docker runs here under `sg docker`.

### What can go wrong in the measurement

- A port taken by something else, or a command run without its `COGMER_ADDR`, reaches another daemon and
  looks like success. The checks in the first part of this section cover it.
- A failure makes the fleet look cheaper. A poll that fails is followed by a longer wait, up to 15 seconds
  (`syncPeer` in `cmd/cogmer/sync.go`), so a fleet whose polls are failing makes fewer requests and spends
  less. Every run checks that turns are still delivered, by writing one turn in each room at the end and
  timing it as measurement 4 does, and counts the lines "is unreachable" and "still unreachable after" in each
  daemon's log, which `markPeerUnreachable` in `cmd/cogmer/ui.go` writes.
- With the overlay on, a daemon that cannot reach the relay logs "tailcat unavailable" and serves only over
  TCP (`runDaemon` in `cmd/cogmer/main.go`), so part of an overlay run can be an overlay-off run without
  anything failing. The harness reads each daemon's log for the line "relay connection carried" before it
  starts sampling.
- The public relays are free and rate-limited (D-192, the public relay is an accepted dependency). A large
  fleet from one address may meet the limit, which shows as the failures above and not as cost, and the load
  falls on a service somebody else runs.
- A first start does work later starts do not: it makes the overlay key, fetches the relay map and chooses a
  region (`loadTailcatRegion` in `cmd/cogmer/tailcat.go`), migrates the stores, and grows the Go heap. The
  5-minute wait before sampling leaves it out of the fixed cost. Measurement 5 times a start, and measurement
  6 a start that migrates every store. On a Mac,
  macOS checks a program file the first time it runs, which made a test's hook take 4.88s and 6.25s against a
  2s limit under the parallel runner (`116b639`). Linux has no such check.
- The droplet's other load. On 2026-11-24 the box ran the `claude` OS account's daemon (45MB resident), this Claude
  Code session (407MB), `codegraph` (40MB), Docker, and at times the test suite, which uses all four vCPUs.
  Runs are made with no suite or container test running, and the baseline is taken with the fleet stopped,
  before and after each run. The steal time in `/proc/stat` says when another customer's load took CPU.
- Daemons started together poll in step, once each interval, so CPU comes in bursts. Samples every 10 seconds
  average them, and the peak is read from measurement 6 and from one run at one-second samples.
- `ss` misses UDP and short connections, and the box's counters include everything else on it, as the
  sampling part says.
- Each poll leaves a socket open for 30 seconds and one in TIME_WAIT after it closes, so at k = 10 and
  one-second polls a daemon holds about 300 open sockets to its colleagues and about 300 from their polls of
  it, with more in TIME_WAIT. The open-file limit for the account's
  processes was 1048576 on 2026-11-24 (`ulimit -n`, and `/proc/176631/limits` for the running daemon), so the
  harness records the count of sockets in each state with `ss -s` at each sample, and that count is a result
  too.

### Order

The four steps of the plan's order, each split:

1. Plan's step 1, on the droplet. First a run of two daemons that forms a room and delivers one turn, with
   the overlay off and then on, which checks the harness itself. Then measurement 1 with the overlay off,
   then on at N = 1 and 10, then measurement 2 at the default interval.
2. Plan's step 2. Measurement 3, the rest of measurement 2's intervals, and measurement 4.
3. Measurements 5, 6 and 7, which the plan's order does not place. They come here because each depends on the
   fleet set-up that the first two steps here test.
4. Plan's step 3, the ARM server, repeating measurement 1 and measurement 2 at the default interval.
5. Plan's step 4, measurement 8.

Results that change the order:

- If the overlay costs much more for each idle daemon than the polling does, idle relay connections are the
  first limit, and measurement 3 moves ahead of the rest of measurement 2.
- If CPU and bytes stay small at k = 10, one-second polls and the largest N that fits in memory, polling is
  not the first limit, and memory with room count (measurement 5) and isolation (measurement 8) move up.
- If 100 daemons do not fit in memory, the ramp stops at the largest N that does, and measurement 6 runs
  there.
- If a pairing takes long enough that a fleet of 100 takes more than an hour to set up, the fleet's copy of
  every `COGMER_HOME` becomes the start of every later run, and no run pairs again.
- If one-second polls through the public relay fail at the first N tried, the overlay runs stay at small N
  until the questions below about the relay are answered.

### Questions the code did not answer

- Does a room of k + 1 daemons need every two members paired, or only each guest with the host? The TLS pin
  accepts a key the daemon knows (`pinTrust` in `cmd/cogmer/peertls.go`), and whether a room's guest list
  makes the other guests known was not read.
- Can one daemon hold more than one pairing at once?
- How long does one pairing take on the droplet, over loopback and through the overlay?
- Does each overlay client hold a relay connection of its own, so that relay connections grow with k as well
  as with N?
- What a box should hold to count as fit: a number of daemons for each GB of memory, a CPU share at the
  largest N, a delivery time. The plan names none.
- Should runs at large N go through the public relays at all, or only with the overlay off, and the overlay
  cost be measured at small N and against a relay cogmer runs?
- A hosted machine's colleagues are mostly on other networks, and on one box every simulated colleague is a
  daemon on the box too. Is a second box, holding the colleagues, needed to measure the hosted side alone?
- Can a store be filled with 10000 turns faster than through the hooks and still have the shape real turns
  give it?
- Will the user run the eBPF count and make OS users, both of which need root?
