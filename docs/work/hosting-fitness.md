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
