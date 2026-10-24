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
match on 2026-10-24. The build questions in `docs/open.md` come next, of which four are
decided: a self-removal, removals signed by an older shared key, which machines hold the
shared key, and the suspension of a machine left under an older key, then a revision of the design, which gains a slice for a user's own
machines, and only then the second run.

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
- A machine certificate under an older shared version is kept with `Current` false. What it
  may still do is question 4 in section 7 of the design.

Next, the slices in section 6 of the design, in order: storage, then trust, then
propagation, then removal and recovery beside cloud.

An agent's worktree starts from the remote's default branch unless `worktree.baseRef` is
`"head"` in the Claude Code settings, so each slice needs that setting to start from
`user-identity`. This clone has it in `.claude/settings.local.json`, which is gitignored, so
a fresh clone needs it written again. On 2026-10-23 with Claude Code 2.1.309, an agent
started seconds after the file was created still branched from `main`, and one started a
minute later branched from `user-identity`. Each agent checks that its `HEAD` holds the last
commit of `user-identity` before editing.

The questions recorded are in `docs/open.md`, in the item that opens "Building the user
identity needs answers that no decision gives", and the reasoning for each choice is in
section 7 of the design.

Tests: run `go test -tags ts_debug_websockets ./...` in a cloud session. It passed with no
test skipped, in 139s on 2026-10-24. In a cloud session cogmer asks the relay client to carry
its connection as a WebSocket (D-217, a cloud session's daemon carries its relay connection as
a WebSocket), which only a build with that tag can do, and `scripts/release.sh` sets it. Plain
`go test ./...` builds without the tag, so
`TestTheRealOverlayCheckPassesForALiveServerAndFailsForADeadOne` speaks the relay's own
protocol, which the session's proxy breaks, and it times out after 10s. The test runs the
daemon's own reachability check in the test process, and passed in 23s with the tag.
