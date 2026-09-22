# What leaves the machine

**Date:** 2026-09-21 · **Version:** `plugin/VERSION` 0.6.0, `wireVersion` 2 ·
**Method:** read every outbound path in `cmd/claude-team` — every `http` client,
every `Listen`, every marshalled wire type — plus the release scripts and the
embedded view.

This is a mirror, not a promise, and not a disclosure. It is what the software
does today, established by reading it. A **disclosure** is a different artifact: a
statement addressed to the person whose data it is, in something they read. This
document is the evidence such a statement would be written from, and it is not
addressed to anybody outside this repository.

A line here becomes a commitment only by being written into the specification as a
requirement, and none has been.

The third column is the point. **What holds a property in place** is a different
question from whether it holds today, and it is the one that decides how easily the
property is lost. A thing held by architecture survives people forgetting about it.
A thing held by nothing is one convenient feature away from gone, and nobody will
notice the moment it goes.

## What leaves

| what | to whom | what holds the limits |
|---|---|---|
| Conversation content — the turns in a room | each peer in that room | The design: everyone in a room holds the whole log. TLS 1.3 pinned to the peer's own key (D-101), so it is readable only by the peer it is addressed to. |
| Conversation content, again | each person's **model provider**, under their own account | Inherent. Injection puts a colleague's turns into your session, so their words reach your provider on your subscription. Stated in §28 and logged as review finding A3. This is the largest disclosure in the system by volume and the one least visible as a disclosure. |
| Room membership — which peers you hold events from | every peer you sync with | `syncRequest.Have` is `peer_id → sequence` from `SyncState`, scoped to one room. Anti-entropy needs it (§9). Room-scoped, never the machine-level list. |
| The address you advertise | peers, signed | A pull protocol needs somewhere to pull from. |
| Traffic metadata: which nodes talk, when, how much | the DERP relay, when NAT requires one | The relay carries ciphertext and cannot read it (D-101). That it is operated by somebody else is open and named in §31's Phase 13 question. D-104 pins a region, so the same operator sees it consistently. |
| An IP address and a version | the host in `plugin/release-url.txt`, once, at install | **A pre-GA tag, and nothing else.** Wherever `publish.sh` points — today a personally-operated droplet over plain HTTP at a bare IP, whose logs belong to this project. Integrity is held by `plugin/checksums.txt`, not by the transport, so tampering is caught but the version fetched is visible in transit. The only disclosure neither to a peer nor to a provider, and the earliest one. Retired when a public release host exists (D-115). |

## What does not leave

| what | status today | what holds it |
|---|---|---|
| The private key | never leaves `~/.claude-team/identity.key` (0600) | Architecture, and a test: it is absent from `identity.json`, which `whoami` prints, and a test asserts it never marshals. |
| Any account or registration | none exists | Architecture. An identity is a keypair generated on the machine that uses it (§6); nobody issues it and nobody records it. |
| The machine-level list of known peers | in no response, no sync request, no invitation — an invitation carries the room's name, id, endpoint and the inviting peer's id | **Nothing.** True because no payload happens to include it. No decision says it must not, and no test would catch it. |
| Telemetry, usage reporting, a version check | none exists; every outbound destination is loopback or a peer | **Nothing.** Not a decision — nobody decided against it, and whether to add one is open. |
| Anything the browser view fetches | the embedded page has no absolute URL and no external asset | **Nothing.** True of the current file only. |

## What this shows about the posture

**Three of the four things that do not leave are held by nothing at all.** They are
true by accident of what has not been built, not by a choice anybody made or a
check anybody wrote. That is invisible from every other document, because an
absence has no assertion — the same reason the MCP display finding lives in
`CLAUDE.md` rather than in a test.

Two properties are genuinely load-bearing and would be hard to lose by accident:
the private key never leaving, and a relay never seeing plaintext. Both are held by
architecture and one is held by a test as well.

The weakest point is not a leak but an asymmetry: **the one non-peer disclosure
goes to a host this project operates.** Everything else discloses to a peer the
person chose, or to a provider they already pay. It is tagged pre-GA rather than
defended (D-115), and the tag has an operative trigger rather than a milestone,
since GA is defined nowhere: the droplet exists because a module path must match a
repository URL and there is no repository, which is Phase 13's blocker. The same
event retires both.

## There is nowhere to disclose any of this

Three surfaces reach a person: `plugin/README.md` at 64 lines, which is what
somebody installing reads; the browser view at 145 lines, which D-039 records as
the one thing consulted rather than forgotten; and command output at a terminal.
**None of them says anything about what is transmitted** — the word privacy appears
in none of the three.

`README.md` is not a fourth: it carries build instructions and is addressed to
somebody compiling the thing. Nor are the fourteen files in `plugin/commands/`,
which are prompts the model reads, not documentation a person reads — D-086 settled
that about `peer-pair.md` specifically and it is true of all of them.

So there is no user-facing documentation, and a privacy posture written into the
specification or into this document reaches nobody it concerns. If that changes,
the view is the surface the project has already established people look at.

## What is not established here

Whether any of this should be permanent. Which lines are worth stating as
requirements, and which are simply true this month. Whether a version check would
be worth its line in the first table. And what a person should be told, as
distinct from what is true — which cannot be settled until there is somewhere to
tell them.
