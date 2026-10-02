# D-066 — The binary is fetched and verified, never shipped in the plugin

**Date:** 2026-09-18 · **Status:** active (implemented); inert until a release exists

**Context.** §29 requires that a participant installs one thing. The hooks call a
compiled binary, and nothing put it on the machine. Three candidates: ship binaries
in the plugin repository, fetch them at first run, or build from source.

**What the ecosystem does — stated more carefully than it first was.** Of 53
plugins in the official marketplace, not one ships a binary. That is true and it
proves much less than it sounds like, because **almost none of them has one**: four
run `bun run --cwd ${CLAUDE_PLUGIN_ROOT}` over TypeScript sitting in the plugin
directory, two are npm packages, one is Python, one runs the user's own PHP project.
The runner there is an interpreter for code already present, not a distribution
strategy. Eight of nine never faced this problem.

The signal that does survive is the ninth. `terraform` is the only plugin with a
genuinely compiled server, and it runs `docker run hashicorp/terraform-mcp-server:0.4.0`
— a registry, content-addressed and version-pinned, rather than the repository that
describes it. Where a compiled artifact exists, it comes from somewhere with content
addressing.

**The arithmetic.** Five targets at 11 MB stripped is 56 MB of binaries. Every
install would download all of them to obtain the one it can run, because four fifths
are for platforms that machine is not — against a plugin that is otherwise about
100 KB of text. Tailcat would roughly double it (D-062, as
`f288913:docs/decisions/D-062-tailcat-evaluated-for-phase-15-a-good-fit-adopted-behind-an.md` held it).

A git repository also keeps every version of every file forever, and binaries do not
delta-compress, so ten releases would accumulate half a gigabyte of history. Whether
an installer pays that depends on clone depth, which was asserted here before it was
checked and is **not** established: the official marketplace turns out not to be a
git clone at all — it arrives as a content-addressed archive with a `.gcs-sha` — and
no git-sourced plugin was installed locally to measure. The 56 MB per install stands
on its own and needs none of it.

The runner pattern is also unavailable to us, and for a reason already recorded:
`npx` needs Node, and the README says plainly that Claude Code ships as a native
binary and a teammate may have none. Choosing Go with pure-Go SQLite was precisely
to avoid a runtime dependency; reintroducing one as the *delivery* mechanism would
undo it at the last step.

**What a registry would have given us, and what we gave up.** Version resolution for
free, the platform matrix handled by somebody else, provenance and revocation, and a
fetch path corporate networks already proxy. Checksum pinning is a thin hand-rolled
substitute for the third of those. Set against it: `npx pkg@latest` re-authorizes
whatever the registry serves on every single launch, with no pin at all. The trade
is not convenience against rigour in one direction — we lost provenance
infrastructure and gained a pin they do not have.

And the framing that resolves it: GitHub Releases with checksums **is** the Go
convention, which is what goreleaser exists to do. We follow our language's norm and
they follow theirs. The divergence is downstream of choosing Go, which was decided
for a reason that still holds.

**Decision.** Fetch at first run into `~/.cogmer/bin`, and **run nothing that
cannot be verified**. `plugin/checksums.txt` is committed to the plugin repository
and is the only thing that authorizes execution. A download whose hash is not
listed is deleted rather than run, and `scripts/release.sh` generates the file from
the bytes it just built, because a hash typed rather than computed authorizes
something nobody has seen.

That is the same shape as the rest of this design: the thing that arrives is
checked against something that came by a different path. The check is weaker here —
release assets and the plugin repository are both on GitHub — but altering a
committed file leaves a trace in history where a swapped release asset would not,
and that is a real difference rather than a comforting one.

**Three lessons taken from the only comparable bootstrap in the marketplace**, whose
comments record them as bugs it already paid for:

- **Install outside the plugin directory.** A plugin update must not discard a
  working binary and re-fetch it; the two change on different schedules.
- **Several sessions start at once, routinely.** Two concurrent installs writing one
  path is a corrupt binary. `mkdir` is the lock, being atomic everywhere; the losers
  exit silently, because being second is not an error.
- **Never retry a failure every session.** A machine with no network and no Go would
  otherwise spend an attempt on every session forever. One hour's cooldown.

And one of our own: it runs **detached**. §29 requires that starting must not delay
a session, so nothing waits on a download — a session that begins before the binary
exists simply has nothing to inject yet, and the next one has everything.

**Verified by running it, including the case that matters.** A local release server
produced a correct install; a tampered asset was refused and deleted with the hashes
named, nothing was installed, and the process still exited 0 (§3.1). Five concurrent
installs produced exactly one attempt. A second run inside the cooldown added
nothing to the log.

**It is inert today, and deliberately so.** `checksums.txt` carries no entries until
a release is published, which disables downloading entirely — the intended failure
direction, since no binary is better than an unverified one.

**A finding that changes what must happen next.** The source-build fallback assumes
a module path a colleague can reach, and this repository is private: `go install`
fails for anyone but its owner. So for the first real pair, a published release with
pinned checksums is not the preferred path, it is the **only** one. Phase 13 depends
on it — if the binary is hand-delivered, the first-five-minutes test measures
something that will never happen again.

**Revisit when** a release exists and the first colleague installs. What to watch:
whether the detached install completes before they try to use it, and what the
session says in the window where the plugin is present and the binary is not.
