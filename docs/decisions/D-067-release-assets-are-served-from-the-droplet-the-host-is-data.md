# D-067 — Release assets are served from the droplet; the host is data, not code

**Date:** 2026-09-18 · **Status:** active (implemented, v0.1.0 published)

**Context.** D-066 built verified acquisition and left it inert: no repository
existed to release to. Worse than absent — **there is no git remote at all**, so
`github.com/Blue-Rocket/cogmer` is a module path inherited from `go.mod` rather
than somewhere anything lives. And GitHub release assets inherit repository
visibility, so on a private repository the download URL returns 404 to an
unauthenticated client while `go install` fails for the same reason. Both paths
dead, for the same cause.

**Decision.** Serve the assets from the droplet that already exists, and treat
GitHub Releases as where this goes when the code is public.

**The host is data.** `plugin/release-url.txt` sits beside `plugin/checksums.txt`
and `plugin/VERSION`, all three committed together. Moving hosts is then one commit
that changes the URL and the hashes at the same time. A wrong host costs a failed
download; hashes that did not move with it would cost a refusal nobody could
explain.

That also decides where the seam goes. `scripts/release.sh` builds and hashes and
is host-agnostic; `scripts/publish.sh` puts the files somewhere and is not. When
this moves to a public release host, publish.sh is replaced and release.sh is
untouched.

**Plain HTTP, and the checksum is why.** The binaries are not secret, and what
authorises running one is a sha256 pinned in the plugin rather than the transport.
A tampered response is refused exactly as a tampered file is — demonstrated in
D-066. HTTPS would still be better and is one of the things a public release host
would hand us for free; until then the pin is load-bearing rather than
belt-and-braces, and should not be quietly removed as redundant.

**publish.sh verifies what the host actually serves**, fetching each asset back and
hashing it against the pinned value. A publish that succeeded while the host served
something else is the one failure nobody would think to look for, and it is
invisible until a colleague's install refuses a binary for reasons on their machine
rather than yours.

**Verified end to end, from a clean state with no environment overrides.** A fresh
install fetched and hashed and landed 11 MB; a second run added nothing to the log;
bumping the version and republishing was picked up and replaced the binary in
place. nginx serves one directory with autoindex and nothing dynamic.

**What this costs.** The distribution host is now a machine you run. If it is down
when a colleague installs, they get no binary and retry in an hour — degraded
rather than broken, which is the right direction, but it is an availability
dependency that a release host would not be. It is also one more thing to remember
exists.

**Revisit when** the code goes public. GitHub Releases then gives HTTPS, a CDN,
provenance, and no host to run — and the change is `release-url.txt`, the checksums
regenerated for the same bytes, and a different publish.sh.
