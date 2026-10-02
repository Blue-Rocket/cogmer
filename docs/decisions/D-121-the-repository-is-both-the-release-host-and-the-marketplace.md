# D-121 — The repository is both the release host and the marketplace

**Date:** 2026-09-22 · **Status:** active (implemented)

**Context.** Phase 13 was blocked on one thing: `go.mod` names a repository that
does not exist. Creating it turns out to settle two other open items in the same
act, because both of them existed only because there was no repository.

**The release host.** D-115 counts three centralized components and this was the one
failing its benefit test outright — a personally-operated droplet, plain HTTP, bare
IP, serving us rather than the person installing. GitHub serves a release's assets
at `{owner}/{repo}/releases/download/{tag}/{asset}`, and `install.sh` already builds
`${base}/v${want}/${asset}`. Those are the same shape, so the installer does not
change at all: `release-url.txt` and `publish.sh` are the whole of it, which is what
that split was for. What is gained is HTTPS and a host nobody here operates. What is
not gained is trust — the sha256 in `checksums.txt` is still what authorizes a
binary, exactly as it was when the transport was plain HTTP.

**The address is also removed from the history**, not just the tip. A public
repository would otherwise advertise a personal machine in two commits' diffs, and
the one quoted log line in this file naming it is redacted for the same reason.

**The marketplace.** D-041 said the install is one line, `claude plugin install
cogmer`. It is two: a plugin is installed from a marketplace, and a marketplace is
added by name. The repository carries `.claude-plugin/marketplace.json` beside the
`plugin/` directory it lists, so `/plugin marketplace add Blue-Rocket/cogmer` and
`/plugin install cogmer@blue-rocket` name the same repository and a person has one
name to know rather than two. D-041's count is superseded; nothing else in it is —
the plugin still carries the hooks and the session-start hook still starts the
daemon.

**Rejected.**
- *Serving release assets from a personally-operated droplet over plain HTTP.* It is a
  component that failed D-115's benefit test outright, serving the maintainer and not the
  person installing, and it is an availability dependency that a release host nobody here
  operates is not.
- *A DNS name in front of the droplet.* It hides the address without removing the
  component: still a machine one person operates, still plain HTTP or a certificate
  to keep alive, still failing D-115's benefit test. The address was the symptom.
- *A separate marketplace repository.* One more name for a person to know and one
  more thing to keep in step with every release, whose only entry would be a plugin
  that lives somewhere else.
- *Publishing to both the droplet and the release.* Two hosts for one pin, and the
  first question after any failed download becomes which one served it.

**The check.** `publish.sh` fetches every asset back over the public URL and hashes
it against `checksums.txt` before reporting success, so a release that was created
but serves something else fails at the person who published it rather than at the
person installing.

**Revisit when** GitHub stops serving assets at `/{tag}/{asset}` under the download
base. `install.sh` builds that URL by concatenation, so a changed convention arrives
as a 404 and a source build, not as a message saying the convention changed.
