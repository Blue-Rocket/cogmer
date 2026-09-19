# Phase 5 — Offline and Reconnection

**Run:** 2026-09-18, MacBook (darwin/arm64) and a DigitalOcean droplet
(linux/amd64), peers meeting through an SSH tunnel with both peer APIs on
loopback. Fresh identities and a fresh room on both sides.

**Result: convergence after a real partition, automatic and in identical order.**
Nothing was triggered by hand — the sync loop did it.

## What was run

| | |
|---|---|
| Paired | `concert Orlando` on both machines, over the internet |
| Connected | one turn each; both peers held 2 events |
| Partitioned | tunnel killed; Mac wrote 2 turns, droplet wrote 3 |
| While partitioned | Mac held 4, droplet 5 — both kept working (§26) |
| Healed | tunnel restored; both held **7**, same order, within one poll |
| Conflicts | none on either side |
| Clock skew | 1 second |

The reconnection is visible in both logs as one exchange each way:

```
mac      sync: received 3 event(s) from 127.0.0.1:4901
droplet  sync: received 2 event(s) from 127.0.0.1:4902
```

Two events each side had already seen were not re-sent, because the watermark is
the highest **contiguous** sequence per peer and it had advanced before the
partition.

## Two defects found, both fixed

Neither was reachable from a single machine, and neither is exotic.

### Whoever ran `pair` first always failed

The droplet typed four seconds before the Mac, sent its commitment, and got `401
Unauthorized` — the Mac had not yet recorded it, so `handleVerify` refused an
unknown peer. `RunVerification` retried only on 409 and treated 401 as final, so
the earlier caller failed immediately while the later one waited ninety seconds and
timed out. **Both people fail, and the one who followed the instruction first fails
faster.**

Fixed by answering 409 for "I do not know you yet", which is a state that resolves
the moment the other person types. 401 is now reserved for a signature that does
not verify — the only answer here that waiting cannot fix.

### The side that finished first stranded the other

Deeper, and it survived the first fix. Both sides drove their own exchange; the
side that completed tore its session down in a `defer`, so the other side's
commitment arrived to nothing and waited out the timeout. Symmetric in appearance,
and not symmetric at all: it fails whenever two people type a few seconds apart,
which is always.

Fixed by letting either side complete from **inbound** as well as outbound. A peer
whose session already holds the other's revealed nonce has everything it needs and
stops driving. Only one side has to run the exchange; both learn the words.

`TestTheFirstToTypeWaitsForTheOther` reproduces the ordering in-process.

## Two defects found and not fixed

### An endpoint is advertised as the address it binds

`peerAddr()` is used both to bind the listener and to tell other peers where to
reach this one. Those are the same string only when nothing sits in between. Behind
the tunnel the droplet's invitation read `127.0.0.1:4783` — its own loopback — and
the Mac had to be given `127.0.0.1:4901` by hand. The advertised address also
propagated: the Mac spent the run politely trying `127.0.0.1:4783` every second and
logging the refusal.

§4 already says an endpoint is a bootstrap hint that need only be correct once, and
that an invitation may carry more than one. The implementation cannot express
either. **Behind NAT this is the ordinary case, not an edge case**, so it belongs
with discovery rather than after it.

### `log`, `conflicts`, `seed` and `whoami` still read the pre-D-046 room

They call `openLocal()`, which resolves a single room from `config.json` and
defaults to `default`. During a live room with seven events, `claude-team log`
reported `ROOM DEFAULT -- 0 events`, and `conflicts` reported on a room nobody was
in. The daemon has served many rooms since D-046; these four commands never caught
up, and the failure is silent — an empty room reads as a quiet one.

## What this run did not test

NAT traversal: the peers met through a tunnel, as in Phase 2. Long partitions:
minutes, not hours, so nothing exercised a room's dormancy. A third peer, which
remains deferred.
