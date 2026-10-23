# Brief: build the user identity that sits across machines

This brief starts a session that builds the design in D-219 to D-227 and D-216. It serves the open item "A user's machines are separate identities to colleagues..." in `docs/open.md` and is deleted with it.

Build the user identity that sits across machines, in the cogmer repo, on a branch.

Branch: create `user-identity` from `main` and work only there. Never push to main. Never run `scripts/release.sh` or `scripts/publish.sh`, and never edit `plugin/VERSION`, `plugin/checksums.txt` or a manifest version.

Read first: `CLAUDE.md`, `docs/writing.md`, and the decisions skill. Then read these decisions, which are the design: D-219 (a user is a master key that certifies machines through a shared key), D-220 (a removal names the last event of a machine that counts), D-221 (an event whose identity version the receiver lacks is withheld from the assistant until fetched), D-222 (a machine tells every machine it knows of each machine it adds), D-223 (the recovery phrase is typed only into the daemon's browser page), D-224 (a conflict between identity records goes to the higher level, then to a removal, then to the lowest hash), D-225 (a machine sends each removal and new shared key to every colleague it knows until each acknowledges), D-226 (a cloud session takes the highest valid snapshot of its user's colleagues and rooms from any source), D-227 (the daemon pushes a new snapshot to the user's repo when the user's list changes, once the user has opted in) and D-216 (a cloud session signs with a key of its own that the user's shared key vouches for). D-044, D-101, D-141, D-143 and D-152 are already written for the design and differ from the code. The open item in `docs/open.md` lists what is unbuilt and undecided.

The work is two runs, so that the cost of the first is known before the second starts. Do the first run only. Start the second only when the person who started this session tells you to.

## Rules that matter here

- A session is never worse for having cogmer: fail open, add no delay, show no error.
- The code is pure Go with no cgo, and uses `modernc.org/sqlite` only.
- Signature schemes are kept, never edited (D-058). A new signed form is a new scheme version beside v3, and records that others hold must keep verifying.
- Every write to a room's database is made under `d.mu`. A change to a stored format migrates every existing store, with `addColumnIfMissing`, and private keys live in files of their own (D-153).
- The recovery phrase never passes through an assistant, a command argument or a log.
- A room surface holds only its own room. Copy the two-room tests in `cmd/cogmer/roomscope_test.go` for anything new that serves a room.
- Only three users exist and they upgrade together, so no migration path between identity models is needed. Old events must still verify.
- Never commit past a failing test. `go test ./...` takes about 2.5 minutes, so run targeted tests while working and the whole suite before each commit. Prove that a new test can fail by reverting its fix once.
- When a slice makes a decision true of the code, change the decision's status to active, and change the specification (section 6 and the view's list) in the same commit (W-85). Write documents with the file tools, run the writing-review skill on them, and keep to `docs/writing.md`.

## The first run: the design and the foundation

You are the orchestrator. Use at most two agents at a time, each in its own worktree branched from `user-identity`.

1. Write `docs/work/user-identity-design.md`: the record types, the wire forms, the Go interfaces between the slices, the test plan, and the slices of the second run in the order of their dependencies. Only `docs/open.md` may cite it.
2. Build the foundation: key derivation from the phrase, the certificate and removal records, chain verification, the precedence rule (D-224) and the cutoff rule (D-220). Write them as pure functions with tests and no daemon. One agent can take the records and the chain, and another the precedence and the cutoff. Review each against the decisions before you merge it.

The foundation is not behavior a user can reach, so no decision becomes active in this run.

The first run is done when the design document is committed, the foundation is merged on `user-identity`, `go test ./...` passes, the branch is pushed, and `docs/work/user-identity-status.md` and your report say what is built, what is left and which questions you recorded. Then stop.

## The second run: the slices

Read `docs/work/user-identity-status.md` and the design document first. Use at most three agents at a time, each in its own worktree branched from `user-identity`, one slice each, and merge the slices in the order of their dependencies. Review each merge against the decisions before the next.

- Storage and migration of identity records.
- Trust: pairing and verifying a user, D-101 pinning, D-143 admission, D-044 requests and D-152 events checked by chain.
- Propagation: the identity version in event metadata, a fetch route like `/blob`, the withhold rule, the push with acknowledgement and own-machine list sync.
- Removal and recovery flows and their browser pages (D-223), and marking and withholding a removed machine's events.
- Cloud: the cloud key, the snapshot and its sources, the setup command, and fail-open behavior under `CLAUDE_CODE_REMOTE`.

Each agent writes its tests and reports what it built, what it left and any decision it had to guess. A slice's decisions become active only when its behavior is reachable and the specification changes with it.

The second run is done when every slice is merged or skipped under the rules below, `go test ./...` passes, the branch is pushed, and the status file and your report say what is built, what is deferred, which decisions you changed and why, and the state of the tests.

## Checkpoints and stopping

These apply to both runs.

- After every merge, push the branch and update `docs/work/user-identity-status.md` with what is merged, what is in progress, what is next, the questions recorded and the state of the tests. Never leave more than one slice's work unpushed, since the pushed branch and the status file are the only record if the session is interrupted.
- Stop a slice, record why in the status file and go on with the others when it needs one of the undecided questions answered, or when a failing test still fails after three attempts to fix it.
- Stop the run and report when finishing would reverse a decision, since a reversal is the person's to make. Stop and report also when two agents' work conflicts in a way the design document does not settle, instead of choosing silently.
- Stop the run and report before any step that would touch `main`, release, or send anything to another machine or user, or that needs a tool this environment lacks, such as a real Claude Code or Codex host.

## Not decided

Do not guess silently. Record each question in `docs/open.md` and take the choice that fits the decisions:

- the recovery phrase's word list and key derivation;
- the wire form of certificates, and of an event signed by a machine on behalf of a user;
- whether an event's author must be a guest of its room. The lean is to refuse an author who never was one, which needs a record of past guests;
- the room view's screen for choosing a cutoff;
- recovery on a fresh machine;
- social recovery, which is out of scope.

In the cloud slice, check these with a probe in a cloud session, and record them:

- Where a cloud session places a repo added to it, and whether the repo is there when the session-start hook runs.
- Whether a cloud environment hides an environment variable's value from other people who can edit the environment.
- Whether a cloud session can open a connection and finish a TLS handshake with a machine through the relay.
