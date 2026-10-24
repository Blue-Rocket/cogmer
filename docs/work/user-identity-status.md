# Status: building the user identity

This is working material for the open item "A user's machines are separate identities to
colleagues, and a cloud session has no identity of its own to sign with" in `docs/open.md`,
and is deleted with it. It records where the build on the `user-identity` branch stands, so
that an interrupted session can resume from the pushed branch. The design is
`docs/work/user-identity-design.md`, and the brief is `docs/work/user-identity-brief.md`.

The first run is done. The second run starts only when the person who started the session
says so.

Before the second run, the maintainer decided on 2026-10-23 that the user, and not the
machine, is the primary entity, and answered ten questions it raised, recorded in
`docs/work/user-primary-inventory.md`. The decisions and the specification were rewritten to
match on 2026-10-24, and the seven build questions were decided the same day, in D-224 and
D-234 to D-239 among others. The design was revised the same day: its section 6 lists the second
run's slices, now seven, and its section 8 proposes a choice for each detail the decisions
leave open, for the maintainer to confirm before the slice that needs it.

The second run started on 2026-10-24. Merged in it so far: slice 0, the foundation
additions, which are `CheckEvent`, the signed `ListChange` record and `MergeListChanges`,
`RoomIDFor`, and `ResolveGuests` in `guests.go`, which computes a room's list in rounds so
that an expelled guest cannot admit an accomplice (D-235); and slice 1, storage, which adds
`identity_records` keyed on each record's own hash, `machines` with an address per machine,
the guest-list cache, `signer` on `events`, `shared.key`, and the user and chain in
`identity.json`. Slice 2a, the transport and signing half of trust, is merged: event scheme v4, sync
requests under v5 with the chain, the chain in the TLS certificate (OID `2.25.2060991052`),
pairing over users' identifiers, and addresses per machine.

Slice 2b, the guest list as room events, is merged: founding, inviting and revoking append
guest events, a founded room's list is computed by `ResolveGuests` and cached, an event is
stored only if its author counts on the list, a room's sync targets are the admitted users'
machines that joined it (`room_machines`), and forgetting leaves every room's list alone. A
room without a founding event behaves as before. Slice 4, propagation, is merged:
the identity fetch route, the withhold rule (D-221), pushing removals and new shared keys with
acknowledgement (D-225), refusals that stop a cut-off machine's events being offered again,
and the refusal that carries the revocation (D-241), with `d.Expulsion(roomID)` for slice 5.
Slice 3, a user's own machines, is merged: the identity page (D-230) with the phrase and
its confirmation, adding a machine with the machine words (D-229), recovery, `self-machines`
(D-231) and machines in `self-status`, copying a user's lists between their machines (D-228),
joining a room by name from another machine, and each machine storing its own chain. A user
identity can now be made, so `plugin/README.md` says what it sends. Slice 5, removal, recovery and what the user sees, is merged: the removal page with its
lost-or-stolen question (D-220), certifying a new shared key with the removals and machines
the user confirms (D-220, D-234), suspended machines' turns hidden from the view, export and
injection, marked turns past a cutoff, machine names and the user's own turns in the view and
the injected block (D-232, D-233), reachability per colleague and per own machine (D-194), and
the notice of an expulsion (D-241). Slice 6, cloud sessions, is next, and the open item for
setting up a cloud session holds the three facts it depends on.

What slice 3 leaves open:

- `Resolve` picks among one machine's certificates under one version by lowest hash, so a
  rename wins only about half the time there; names are read through `displayCerts`, by
  highest version and then highest serial, and a rename takes a new serial.
- A `ListChange` holds a colleague's whole entry, so a label changed on one machine and a
  verification made on another at the same time can lose the verification.
- Recovery takes the highest version this machine holds plus one, or 2, since slice 4's fetch
  route was not on its base; it should ask colleagues for the user's versions.
- `COGMER_IDENTITY_PAGE=off` turns off the page's gate, for tests that create and join with no
  daemon, and ships in the binary as `COGMER_PREFLIGHT=off` does.
- A machine with rooms from before it made a user identity is not migrated: its colleagues'
  guest lists hold its machine key, and under D-240 those rooms are created again.

What the later slices need to know from slice 2b:

- A cut-off machine's later events leave a gap in its sequence, and sync counts only an
  unbroken run, so they are offered again on every poll until the revocation is lifted.
  Nothing is lost, but slice 4 should stop offering a sequence a receiver has refused.
- The view shows a revoked guest's turns stored before the revocation arrived, and an
  assistant is never given those past the cutoff; slice 5 marks them in the view.
- The command-line tools reach founding, inviting and revoking through a local route,
  `/room/guests`.

What the later slices need to know from slice 2a:

- Slice 2a made a room's sync targets every machine of each guest, which its brief got
  wrong; slice 2b made them the machines from which a session has joined the room (D-103).
- The events table's uniqueness is on the signer and sequence, migrated by
  `rekeyEventsOnSigner`, since two machines of one user both write sequence 1.
- `Sign` stays v3; `SignAs` and `Store.AppendAs` sign v4 when the machine has a user identity.
- Requests, events, offers, verification steps and the certificate carry the chain, which
  shows the user's identifier, the shared-key version and the machine's name and serial to
  whoever the machine connects to. `plugin/README.md`, under "What leaves your machine", says
  so in the slice that makes a user identity reachable, slice 3.
- `known_peers.endpoint` is written beside `machines` until a later slice drops it.

What the later slices need to know from slice 1:

- Every existing `known_peers` address was copied into `machines` with the machine key and
  the user both set to the peer's identifier. Once a colleague's master key is known, slice 2
  rewrites `user` on those rows, which `SetMachineEndpoint` refuses to do, so it needs a
  function of its own.
- `SetPeerEndpoint` still writes only `known_peers.endpoint`, and nothing reads `machines`
  yet. Slice 2 moves the readers and writers to `machines`, then drops the old column.
- Room databases store `signer = peer_id` for every event until v4 events exist.
- `IdentityRecordByHash` reads any stored record, session certificates included, for the
  fetch route in slice 4.

Merged in the first run:

- The design document.
- `userphrase.go`: the 18-word recovery phrase and the master key derived from it. A phrase
  prints and marshals as a redacted text, and no parse error quotes a word.
- `idrecords.go`: the shared-key, machine, session and removal records, their signing bytes
  and hashes, and the chain check. Frozen records are in `testdata/identity_vectors.json`.
  A record under a scheme this build does not know hashes to zero, so nothing uses a hash
  before the record verifies.
- `idstate.go`: `Prevails` (D-224, a conflict goes to the higher level, then to a removal,
  then to the lowest hash), `Resolve`, the cutoff (D-220, a removal names the last event of a
  machine that counts), `Counts`, `List`, `Check` and `Lacks`.

Nothing a user can reach uses the foundation, so every decision of the design stays not
built, and the specification is unchanged.

What the second run's slices need to know from the first:

- `Check` is for a live act and refuses any chain whose machine has a removal. An event is
  judged by `Chain.Verify`, the shared-key conflict and `Counts`, because a removed
  machine's events count up to its cutoff. The trust slice needs a function for events
  that checks the shared-key conflict without refusing for a removal.
- `Check` also refuses a chain for a machine whose certificate is not held yet when a
  removal of that machine is held.
- `Resolve` is rerun over the whole record set whenever the set grows. A machine
  certificate whose shared-key certificate has not arrived is discarded until it has.
- A machine certificate under an older shared version is kept with `Current` false, and the
  machine is suspended: `Counts` gives false for its events and `Check` refuses its chain
  until it is certified under the current version (D-234). Only the current shared key may
  remove a machine (D-220).

Next, the slices in section 6 of the design, in order: foundation additions, storage, trust,
then a user's own machines beside propagation, then removal and what the user sees beside
cloud.

An agent's worktree starts from the remote's default branch unless `worktree.baseRef` is
`"head"` in the Claude Code settings, so each slice needs that setting to start from
`user-identity`. This clone has it in `.claude/settings.local.json`, which is gitignored, so
a fresh clone needs it written again. On 2026-10-23 with Claude Code 2.1.309, an agent
started seconds after the file was created still branched from `main`, and one started a
minute later branched from `user-identity`. Each agent checks that its `HEAD` holds the last
commit of `user-identity` before editing.

The details the decisions leave undecided are in `docs/open.md`, in the item that opens "The
decisions that make the user the peer leave details undecided".

Tests: run `go test -tags ts_debug_websockets ./...` in a cloud session. It passed with no
test skipped, in 139s on 2026-10-24. In a cloud session cogmer asks the relay client to carry
its connection as a WebSocket (D-217, a cloud session's daemon carries its relay connection as
a WebSocket), which only a build with that tag can do, and `scripts/release.sh` sets it. Plain
`go test ./...` builds without the tag, so
`TestTheRealOverlayCheckPassesForALiveServerAndFailsForADeadOne` speaks the relay's own
protocol, which the session's proxy breaks, and it times out after 10s. The test runs the
daemon's own reachability check in the test process, and passed in 23s with the tag.
