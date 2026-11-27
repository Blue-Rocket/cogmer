# Hosting fitness

Material for the open item "A hosted machine's cost is almost all polling" in
`docs/open.md`. The phase measures what the daemon's present architecture costs to host, for one user and for
many on one server, as the baseline that other architectures for mass hosting are compared against (D-285, a
user with no computer of their own has a machine hosted for them; D-287, a user with a computer of their own can
also have a machine hosted). It is not a load test: no run looks for the point where a server fails. Delete this
with the item.

Written 2026-11-24 and rewritten 2026-11-25. Prices and limits below are from web searches on 2026-11-24, and
the sources disagree with each other in places, so each is checked on the provider's own page before it is
relied on.

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

Measured: what a daemon costs with colleagues and turns, and which part of the design each part of that cost
comes from, in the results below. Not known: what a colleague on another machine costs in bytes on the network,
what the overlay costs with turns, how the store and memory grow over hours of turns, what an OS user or a
container adds, and what an ARM server does.

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

The users each server holds, and the cost of each user a month, are in the last table of the results below.

## The user the costs are for

The reference user has 3 rooms, each with one paired colleague, and each room gets one new turn a minute,
written by either member. That is 3 turns a minute for the user and 4320 a day. The three colleagues are three
different users, so the daemon polls 3 addresses with one room each. A turn is a prompt of 500 bytes and a
reply of 4000. The user set these on 2026-11-25.

At the default interval the reference user's daemon makes 3 polls a second and answers 3, one from each
colleague's machine.

## What the cost is made of

The cost of one hosted user is modeled as a sum of terms, each caused by one part of the present design. An
architecture that changes that part changes that term, so each is measured on its own.

| term | caused by, in the code read on 2026-11-25 |
|---|---|
| fixed, for each daemon | one process for each user, and its loops that run with no colleagues: images every 2 seconds, identity records every 5, the user's own lists every 10 (`runDaemon` in `cmd/cogmer/main.go`), and the overlay's self-check every 12 to 15 minutes (`RunSelfWatch` in `cmd/cogmer/ui.go`) |
| standing, for each colleague address | the loop for the address (`RunSync` in `cmd/cogmer/sync.go`) and, with the overlay on, a WireGuard stack for the address kept as long as the daemon runs (`tailcatDialer` in `cmd/cogmer/tailcat.go`) |
| each poll of a colleague address | `pullFrom` in `cmd/cogmer/sync.go` builds a new HTTP client for each poll (`clientFor` in `cmd/cogmer/transport.go`), so each poll opens a connection and makes a TLS handshake, then sends one request for each room the address is a guest's address in (`roomsToAsk`); the colleague's daemon answers each, and polls back the same way |
| each further room in one poll | one more request and answer on the same connection |
| each turn | capturing, storing and signing it on the writing machine, and fetching and storing it on the other |
| a start | opening the stores, the overlay's first contact through the relay, and migrating stores after an upgrade |

The reference user's cost for an hour is then the fixed term, plus 3 standing terms, plus 3 × 3600 polls an
hour at the default interval, plus 180 turns. Memory is the fixed and standing terms; CPU and bytes are all of
them.

The measured terms are reported in units that carry to other designs: memory in MB of PSS (resident memory with each shared page divided among the processes that share it), CPU in
milliseconds for each poll and each turn and in seconds an hour for the fixed and standing terms, bytes for
each poll and each turn, and handshakes an hour. A change the numbers point to, such as a longer interval for
a peer that has sent nothing, a long poll, or one connection that carries the polls of many daemons, is a decision
for its own entry after the measurements, not before.

## Results

Measurements 0 to 9 ran on 2026-11-25 on the droplet, on a build of `b91c152`, and every run was valid. The
record of each run, with how it was run and where it was not clean, is
`f98fa3d:docs/work/hosting-fitness-runs-2026-11-25.md`. The terms below are derived from those runs for one
daemon, and each counts both the polls a daemon makes and the polls it answers.

| term | CPU | memory (PSS) | bytes sent | disk written | from |
|---|---|---|---|---|---|
| fixed, for each daemon | 4.1s an hour | 12.0MB | 0 | 0 | measurement 1 at N = 4 |
| standing, for each colleague address | about 0 | 5.1MB at 1s polls, 2.1MB at 5s | 0 | 0 | measurement 4 against 1 and 5 |
| each poll of a colleague address | 14.4ms to 15.1ms | not separated | 6.4KB to 6.6KB | 0 | measurements 2, 4 and 5 |
| each further room in a poll | 5.2ms | not separated | 1.5KB | 0 | measurement 3 against 2 |
| each turn in one of the daemon's rooms | 99ms | not separated | 7.6KB | 110KB | measurement 6 against 4 |
| the overlay, for each daemon | 2.1s an hour | 7.2MB | 0.29MB an hour to the relay | 17KB an hour | measurement 1 |
| the overlay, for each colleague address | 4.7ms more for each poll | 42MB | not measured | 0 | measurement 7 against 4 |

The polls are almost the whole cost. In measurement 4 they took 156 of the daemon's 160 CPU seconds an hour,
and polling every 5 seconds instead of every second cut the CPU from 160s to 35s an hour, in proportion to
the polls, which leaves the standing term near 0. Each poll opens a new connection with a TLS handshake
(`clientFor` in `cmd/cogmer/transport.go`). The code suggests that the handshake is most of a poll's 6.4KB,
which was not measured on its own.

The model's check passed. Measurement 4 was predicted at the fixed term plus 3 times measurement 2's colleague
cost, 154 CPU seconds an hour, and measured 160.5, within 4%. The cost of one user did not grow with the users
on the box: a daemon took 181 CPU seconds an hour at N = 20 and 178 at N = 4, and the box's CPU less the
baseline and the harness's own came to 189 seconds an hour for each user in both.

A turn of 4.5KB writes about 110KB to disk on each of its two daemons, about 25 times its size. A start of 20
daemons at once had every daemon answering within 0.52s and a turn delivered within 0.94s, and took about
250MB of `MemAvailable`.

The reference user, an hour:

| | without the overlay | with the overlay |
|---|---|---|
| the daemon's CPU | 178s | about 230s, measurement 7 plus measurement 6's turns |
| the box's CPU for each user | 189s | about 228s, estimated the same way |
| memory (PSS) | 28.7MB | 161MB, measured with no turns |
| bytes the daemon sends | 67.5MB | not measured |
| disk written | 19.8MB | not measured |

A user whose colleagues are on other machines sends those 67.5MB an hour over the network, about 49GB in a
month of 730 hours, and the price of that traffic is in no table here.

The table below gives the users each server holds, taking the droplet's numbers for every server. Users by CPU are the vCPUs × 3600
divided by the box's CPU for each user, which is every vCPU fully busy, a ceiling rather than a level to run
at. Users by memory are the memory less 1GB, divided by the user's PSS, since the droplet's 8GB left about 7.0GB
in `MemAvailable`.

| option | users by CPU | users by memory | users | each user a month | users, overlay on | each user a month, overlay on |
|---|---|---|---|---|---|---|
| Hetzner CAX21 | 76 | 243 | 76 | about €0.14 | 43 | about €0.24 |
| Hetzner CAX31 | 152 | 522 | 152 | about €0.14 | 93 | about €0.23 |
| Contabo Cloud VPS M | 114 | 522 | 114 | about $0.09 to $0.13 | 93 | about $0.11 to $0.16 |
| Leaseweb VPS 2 | 114 | 522 | 114 | about $0.08 | 93 | about $0.10 |
| the droplet | 76 | 243 | 76 | about $0.63 | 43 | about $1.12 |

DigitalOcean 16GB and Vultr 16GB are left out because their vCPUs are not recorded, and Oracle's free tier
because an OCPU is not a vCPU, and Oracle's rule on idle instances depends on how busy an instance runs. With the overlay
off, CPU limits every server in the table. With it on, memory limits every server, though on the 6-vCPU
ones it allows 93 users against CPU's 94.

## How to measure

What the harness, `docs/work/hosting-fitness/harness.py`, does for each measurement, from the code and the
probe scripts as read on 2026-11-24 and 2026-11-25. Its own check, two daemons that pair, form a room and
deliver a turn each way, passed on 2026-11-25 with the overlay off and on: each pairing took 1.1s, and each
turn arrived in 0.6s to 0.9s.

Each daemon on the box is the program run as `cogmer daemon` in a process of its own, with three variables set:
`COGMER_HOME` names a directory made for it, `COGMER_ADDR` names a loopback port for hooks and the page, and
`COGMER_PEER_ADDR` names a port on 172.17.0.1 for colleagues' machines. The hook and page address must be
loopback or the daemon stops (`runDaemon` in `cmd/cogmer/main.go`). The peer address cannot be loopback, since
`cogmer whoami` then gives no pairing string (`checkReachable` in `cmd/cogmer/main.go`), so it is Docker's
bridge address on the droplet, which only the box and its containers reach. Linux carries traffic to an
address of the box's own over `lo`, and a connection between two daemons can leave from any address the box
holds, including its public one, as Linux picks the source. `COGMER_PREFLIGHT=off` keeps the behavior
checks out of the numbers. The harness also removes every variable that starts with `CLAUDE_CODE_`, `CODEX_`,
`COGMER_` or `CLAUDECODE` from what it inherits, as `env` in `docs/work/host-observer/colleague.py` does,
because the harness runs inside a Claude Code session whose variables would otherwise reach every daemon.

The ports come from 20000 to 21999: daemon i, counting from 0, listens for hooks and the page on 20000 + 2i
and for colleagues' machines on 20001 + 2i. On 2026-11-24 at 21:10 UTC `ss -ltnp` showed listeners on 22, 80,
53 (the resolver, on 127.0.0.53 and 127.0.0.54), 4782 and 4783 (the claude account's daemon, pid 176631), 8767
on the droplet's public address, and 34417 and 45813 on loopback, and nothing from 20000 to 21999. The kernel
hands out outgoing ports from 32768 to 60999 (`/proc/sys/net/ipv4/ip_local_port_range`, read the same day), so
the chosen range never meets a port the daemons' own connections take. The probe scripts' defaults, 47792 and
47793, are inside that range, and the harness does not use them. The harness runs `ss -ltnp` before each run
and stops if any port in the range is taken.

The build is a development build of the commit under test:

```
CGO_ENABLED=0 go build -tags ts_debug_websockets -ldflags "-s -w" -o <run dir>/cogmer ./cmd/cogmer
```

It carries no version, so it reports `dev` and refuses the account's real `~/.cogmer` whatever `COGMER_HOME`
says (D-265, a development or test build never opens the account's real state directory). It differs from a
release build only in the version, since `scripts/release.sh` builds with the same tag and the same `-s -w`.
Every daemon runs the one file, so the kernel shares the program's pages among them.

D-265's refusal keeps a test daemon away from the real directory and not from the real daemon. A command that
runs without `COGMER_ADDR` talks to 127.0.0.1:4782 (`addr` in `cmd/cogmer/main.go`), which is the claude
account's daemon, and that daemon would act on the user's real state. The harness therefore passes all three
variables to every command, never names 4782 or 4783, and after each start checks with `ss -ltnp` that the
process it started holds both of its ports. A check of the page alone is not enough: a development daemon that
finds its port already served by another daemon logs "a cogmer daemon is already serving" and exits, because
`shouldReplace` in `cmd/cogmer/main.go` never replaces when either side is `dev`, and the page that answers is
the other daemon's. The harness starts each daemon in a new session with its output sent to `daemon.log` in its
`COGMER_HOME`, the file that `keepDaemonLogBounded` in `cmd/cogmer/daemonlog.go` keeps under 10MB, and stops it
with `cogmer stop` under the same variables, or with SIGTERM to the process.

`COGMER_TAILCAT=off` turns off the overlay, the encrypted network that reaches a machine on another network
through a relay (`tailcatEnabled` in `cmd/cogmer/tailcat.go`). With it off, the address a daemon gives in its
pairing string is its bound peer address (`AdvertisedEndpoint` in `cmd/cogmer/transport.go`), so pairing and
sync between daemons on the box go over TCP within the box and nothing reaches a relay. With it on, which is the
default, the pairing string carries the overlay address, and every sync request goes through an overlay
client. Each run says which it used.

### The fleet

Every daemon in a run is a reference user, or a smaller case of one, and its colleagues are the other daemons
in the run. A daemon's process does its own polls and answers its colleagues' polls of it, so its cost is the
hosted user's whole cost whether its colleagues are hosted on the same server or elsewhere. What differs for a
colleague elsewhere is the path the bytes take, which the overlay runs measure.

The reference case is built from groups of four daemons in which every two share one room, which gives each
daemon 3 rooms with 3 different colleagues. A group needs 6 pairings and 6 rooms, and N daemons need N / 4
groups. The 6 pairings run in three rounds of two disjoint pairs, A with B and C with D, then A with C and B
with D, then A with D and B with C, so no daemon is ever in two pairings at once.

Each daemon gets a throwaway user identity through the identity page's own two requests, as `make_identity` in
`docs/work/host-observer/probe.py` does: a POST of `{}` to `/identity/phrase`, which answers with the words, the
positions to confirm and a `createId`, then a POST to `/identity/create` with the `createId`, the confirmed
words by position, a `displayName` and a `machineName`. Both go to the daemon's `COGMER_ADDR` with the headers
`Content-Type: application/json` and `X-Cogmer: 1`. The recovery phrase is never shown or kept.

Pairing two daemons needs the following, from `runPair` and `beginCeremony` in `cmd/cogmer/main.go` and the
verification in `cmd/cogmer/verify.go`:

- Both daemons running, each with a user identity, since `runPair` stops without one (D-230, pairing is
  between users).
- Each side's pairing string, which `cogmer whoami` prints on the line after "Send your colleague this"
  (`pairing_string` in `probe.py`).
- Both sides running `cogmer pair <the other's string> <name> --terminal` at the same time, within the 90s of
  `verifyTimeout`. Each prints its two words on the line before "say those two words aloud", and the harness
  compares the two and writes `y` to both when they match, as `pair` in `probe.py` does. A name must be
  given, or `runPair` asks for one.

With the overlay on, the first contact between two daemons builds a tunnel through the relay, which the
comment in `pullFrom` describes as taking seconds. The harness records how long each pairing takes.

Each room is formed as `host` and `join` in `colleague.py` do, with `COGMER_HOST=claude` and a
`CLAUDE_CODE_SESSION_ID` of its own for each daemon in each room: one member runs `cogmer create`, which
prints "created <room>", and `cogmer invite <name>`, which prints "Told them", and the other runs `cogmer join
<room>`, which prints "joined <room>". Every room has two members, so a room never needs a pairing beyond its
two members'.

A turn is written as `say` in `colleague.py` does: `cogmer hook prompt` with the prompt, a hand-made
transcript holding the prompt, then `cogmer hook stop` with the reply as `last_assistant_message`, under the
room's session. Every turn has the reference user's size, a prompt of 500 bytes and a reply of 4000. Turns alternate between the room's two members, and the rooms of a run start
their turns at times spread over the minute, so the fleet's turns do not arrive together.

Once a fleet is set up, the harness stops it and keeps a copy of every `COGMER_HOME`, so a later run starts from
the same state without pairing again.

### What is read from each daemon, and how often

A sampler, one process for the whole fleet, writes one line for each daemon at each sample, with the time,
and records its own CPU time too, since the sampler also loads the box. CPU time is fields 14 and 15 of
`/proc/<pid>/stat`, user and system time in ticks of 1/100s (`getconf CLK_TCK` gave 100 on 2026-11-24), read
every 10 seconds. `VmRSS` and `Threads` come from `/proc/<pid>/status` at the same times. Every 60 seconds the
sampler reads `Pss` from `/proc/<pid>/smaps_rollup`, which divides each shared page among the processes that
share it. RSS counts the shared program pages in every daemon, so memory for each user is PSS, checked against
the drop in `MemAvailable` in `/proc/meminfo` from before the fleet starts. `read_bytes` and `write_bytes` in
`/proc/<pid>/io` give the disk traffic. The box's own `/proc/stat`, including its steal time, and its load
average are read with each sample, because the droplet's vCPUs are shared with other customers.

Linux keeps no count of network bytes for each process. `/proc/<pid>/net/dev` gives the counters of the
network namespace, which every process on the box shares. What the harness uses instead:

- `ss -tinpH`, at each sample, lists every TCP socket with the process that owns it, its remote address and
  its `bytes_sent` and `bytes_received`, and needs no root for the account's own processes. The harness sums
  these for each daemon, split by remote address: an address the box holds is a colleague's daemon or a
  command talking to the page, and any other is the relay or, at a first start, the relay map. A socket opened
  and closed between two samples is missed. Each poll's connection is left idle and closes after 30 seconds
  (`IdleConnTimeout` in `clientFor`), so a sample every 10 seconds sees it after its exchange is done. The
  overlay's self-check opens a client and closes it, and is missed.
- The box's counters for `lo` and the public interface in `/proc/net/dev`, read before and during each run.
  They include the claude account's daemon, any Claude Code session on the box and Docker, so the run's share
  is the difference from a baseline taken with the fleet stopped. They count UDP, which `ss` does not show, and
  with the overlay on, the box's UDP bytes divided by N is each daemon's share, since every daemon in a run
  does the same work.
- `ss -s` at each sample gives the count of sockets in each state. Each poll leaves a socket open for 30
  seconds and one in TIME_WAIT after it closes, so a reference user's daemon at one-second polls holds about
  90 open sockets to its colleagues and about 90 from their polls of it, with more in TIME_WAIT. The count is
  a result too. The open-file limit for the account's processes was 1048576 on 2026-11-24 (`ulimit -n`).

Under the overlay, `ss` shows how many relay connections each daemon holds, which says whether each overlay
client holds one of its own. Bytes on relay sockets that grow with turns mean the relay carries them, and UDP
bytes that grow while relay bytes do not mean a direct path. cogmer logs nothing that names the path a client
took, in the code read.

### Each measurement

Each run waits 5 minutes after its last daemon starts or its last room forms, and then samples for 15 minutes,
except where it says otherwise. The 15 minutes are read as three windows of 5, and a run whose figure for one
daemon differs between windows by more than 10% is run again. Each is run once otherwise, because each
answers a term, not a limit. The overlay is off unless the run says otherwise. The plan sets no pass or fail
for a cost. A run is invalid, and its numbers are not used, when a daemon exits, a port is refused, a turn is
not delivered within ten intervals, or a daemon's log has a line "is unreachable".

0. The box with no fleet, for 15 minutes. Its CPU, `MemAvailable` and interface counters are the baseline
   every later run is read against, and it is taken again before each later run.
1. Fixed cost. Daemons with identities and no rooms, at N = 1 and N = 4, then at N = 4 with the overlay on.
   The answer is memory, CPU seconds an hour and bytes an hour for one daemon. The run at N = 4 gives PSS with
   the program's pages shared, which is the figure for a server of many daemons. The overlay run's 15 minutes
   cover at least one self-check.
2. One colleague in one room. N = 2, paired, one room, no turns. The answer, less measurement 1, is the
   standing term and the cost of a poll for one colleague address: CPU and bytes for each poll, counting both
   the poll a daemon makes and the one it answers, and the handshakes an hour.
3. One colleague in three rooms. N = 2, paired once, three rooms, no turns. The answer, less measurement 2, is
   the cost of each further room in a poll.
4. The reference user, idle. One group of four, no turns. The model predicts the fixed term plus 3 times
   measurement 2's colleague terms. A result more than 10% from that means the model is missing a term, and
   one more run, three daemons in which every two share a room, says whether the missing term grows with the
   colleagues.
5. The poll interval. Measurement 4 with `COGMER_SYNC_MS` at 5000. Measurement 4 has five times the polls of
   this run with the same colleagues, so the two divide the colleague cost into the standing term and the cost
   of a poll. At any interval a machine that does not answer is polled at most every 15 seconds, since
   `syncPeer` in `cmd/cogmer/sync.go` caps its backoff there.
6. The reference user with turns. Measurement 4 with one turn a minute in each room. The answer, less
   measurement 4, is the cost of a turn, as CPU, bytes and disk for each of the room's two daemons. Each turn
   is timed from the return of the stop hook on the writing daemon to the turn's appearance in
   `/events?room=<room>` on the other, which the harness asks for the room's events every 100 ms, and the
   median and 95th percentile of that time are recorded with the cost, since the interval sets both.
7. The overlay. Measurements 4 and 6 with the overlay on. The answer, less measurements 4 and 6, is what the
   overlay adds for each colleague: the memory of its WireGuard stacks, the CPU of each poll through it, the
   bytes on relay sockets against UDP bytes, and the relay connections each daemon holds. The 12 polls a
   second of a group of four go through the public relays, and no larger fleet runs with the overlay on.
8. Many reference users on one server. Measurement 6 at N = 20, five groups. The answer is whether one daemon
   costs at N = 20 what it costs at N = 4. A difference of more than 10% means something shared grows with N,
   such as scheduling or the box's network stack, and runs at N = 8 and 12 say how it grows.
9. A start of every daemon. The 20 daemons of measurement 8, stopped and started at once, three times. The
   answer is the time until every daemon answers at `/healthz`, the time until a turn written just after the
   start reaches its colleague, and the peak CPU and `MemAvailable` drop during the start.
10. Growth with what a user holds. One group of four, filled by `harness.py fill`, which writes turns into
    every room as fast as the hooks take them, up to checkpoints given in days of the reference user's
    turns, 1440 a day in each room and 4320 a day for each user. At each checkpoint it waits for a turn in
    every room to be delivered, reads the size of each daemon's room databases, their logs and its
    membership database, and samples the daemons idle for 5 minutes. The answer is disk for each turn, and
    whether memory, the cost of an idle poll and the time a hook takes grow with the store. Turns written
    at the reference rate would reach a quarter of a day in 6 hours, so the fill replaces waiting with
    writing. A user's store at a year would hold about 1.6 million turns, which no run here reaches.
11. Isolation. Measurement 6 repeated with each daemon run as an OS user of its own and then with each in a
    container of its own, built `FROM scratch` as the container tests' images are. The answer is what each adds
    for one daemon. OS users need root to make, and the user makes them. Docker runs here under `sg docker`.

12. A user on the current build, for choosing between a process for each user and one process for many
    (the open item "Nothing decides how hosted machines run or how each one's page gets its address").
    Measurement 1 at N = 4 and measurement 6 at N = 20, with the overlay off, on the head of
    `release-0.11.0`, which keeps a connection for each address, polls quiet rooms less often and nudges
    (D-303, D-305 and D-306). The answer is the cost of a process for each user, as the box's CPU and PSS
    for one user, and the fixed term on that build, which is the most one process for many users could save
    for each user, since everything else a user costs it pays too. On `b91c152` the fixed term was 12.0MB
    and 4.1 CPU seconds an hour, and on `8671eac` a daemon of the reference user in a group of four held
    17.3MB and took 23.4 CPU seconds an hour (`d1d5db9`), so one process for many users would save at most
    about two thirds of a user's memory and a sixth of its CPU, which this measurement checks.
13. A machine started when it is needed. One group of four whose stores hold a day of the reference user's
    turns, made by `harness.py fill`. One daemon is stopped, a colleague writes a turn in their shared room,
    and the stopped daemon is started; this is done 10 times. The answer is the time from the start until
    the daemon answers at `/healthz`, until it answers a colleague's poll, and until the turn written while
    it was stopped is in its store, with its PSS a minute after the start and the disk it holds while
    stopped. A machine stopped while its user is idle costs only that disk, so these say what starting on
    demand saves and what a colleague waits for. Nothing yet starts a daemon when traffic arrives, so the
    harness starts it at a known moment, which measures the daemon's part and not a waking relay's. The
    harness's `restart` starts every daemon at once, so it needs a way to stop and start one.
14. A page left open on a hosted machine. The page a hosted user reads will reach the machine through
    cogmer's own relay, which is not built, so this measures the two parts such a page would use that exist.
    The first is the daemon's work for each change it shows. The loopback page holds a stream open, and on
    every change in any of the daemon's rooms the daemon builds and sends the whole snapshot of each open
    stream's room (`handleStream` in `cmd/cogmer/ui.go`, woken by `notify`), which is the work a page told
    of a change would cause. It is measured as a Go benchmark of `viewSnapshot` on stores that
    `harness.py fill` makes, at 300, 3,000 and 30,000 turns in a room, and the number of changes an hour is
    counted on a daemon of the reference user in measurement 12's group of four, with one stream open. The
    second is the relay's cost for each held page connection, measured with the relay spike in
    `docs/work/relay-spike/` holding 1, 100 and 1,000 idle connections to its responder, as the relay
    process's PSS and CPU an hour. The answer is the CPU an hour of an open page against the room's size, and
    the relay's memory and CPU for each open page, which a hosted user's cost takes for each hour a page is
    open. It says whether a page should be sent only what changed rather than the whole room, since in a long
    room a stream that sends the whole room on every change could cost more than the rest of the daemon.

The results become one table of the terms with their units, the reference user's cost an hour from them, and,
for each server in the table above, the users it holds by memory and by CPU and the cost of one user a month.
The droplet's numbers stand for the x86 servers. The ARM servers are priced from the same numbers until one is
measured, which is a question below.

Each result goes in a commit that says when, on what machine and build, and what was seen (W-91 in
`docs/writing.md`, a commit that records evidence says how it was run), and the item is rewritten from them.

### What can go wrong in the measurement

- A port taken by something else, or a command run without its `COGMER_ADDR`, reaches another daemon and
  looks like success. The checks above cover it.
- A failure makes a daemon look cheaper. A poll that fails is followed by a longer wait, up to 15 seconds
  (`syncPeer` in `cmd/cogmer/sync.go`), so a daemon whose polls fail makes fewer of them and spends less. Every
  run checks that turns are delivered, by writing one turn in each room at the end and timing it as
  measurement 6 does, and counts the lines "is unreachable" and "still unreachable after" in each daemon's log,
  which `markPeerUnreachable` in `cmd/cogmer/ui.go` writes.
- With the overlay on, a daemon that cannot reach the relay logs "tailcat unavailable" and serves only over TCP
  (`runDaemon` in `cmd/cogmer/main.go`), so an overlay run can be an overlay-off run without anything failing.
  The harness reads each daemon's log for the line "relay connection carried" before it starts sampling.
- A first start does work later starts do not: it makes the overlay key, fetches the relay map and chooses a
  region (`loadTailcatRegion` in `cmd/cogmer/tailcat.go`), migrates the stores, and grows the Go heap. The
  5-minute wait before sampling leaves it out of the other terms, and measurement 9 measures it. On a Mac, macOS
  checks a program file the first time it runs, which made a test's hook take 4.88s and 6.25s against a 2s
  limit under the parallel runner (`116b639`). Linux has no such check.
- The droplet's other load. On 2026-11-24 the box ran the `claude` OS account's daemon (45MB resident), a
  Claude Code session (407MB), `codegraph` (40MB) and Docker, and from about 23:00 UTC that day a second Claude
  Code session worked on the box too. A test suite or container test uses all four vCPUs. Runs are made with no suite,
  container test or build running in any session, and the baseline is taken before and after each run. The
  steal time in `/proc/stat` says when another customer's load took CPU.
- Daemons started together poll in step, once each interval, so CPU comes in bursts. A sample every 10
  seconds averages them, which is right for a cost, and measurement 9 gives the peak.
- The terms are small. One poll may cost well under one tick of CPU, so CPU for a term comes from the total
  over a run divided by the polls, never from one sample.

### Order

1. Measurement 12, about an hour, and measurement 10, at a quarter, a half, one and two days of turns,
   which takes about an hour and makes the stores measurement 13 starts from.
2. Measurement 13, once the harness can stop and start one daemon, and measurement 14 on the stores of
   measurement 10, with a store of 30,000 turns filled for it, and on the relay spike.
3. Measurement 11, which needs the harness to run a daemon as an OS user of its own and in a container of its
   own, which it cannot yet do.
4. An ARM server, once the question below is answered.

### Questions

- Is there a pair of builds where the newer migrates the older's stores, so that a start after an upgrade can
  be measured? Measurement 9 measures a start with nothing to migrate.
- Should an ARM server be rented for the measurements, since the cheapest servers in the table are ARM and
  the droplet is not?
