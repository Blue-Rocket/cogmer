# D-121 — Releases are served from the repository's GitHub releases

**Date:** 2026-09-22 · **Status:** active · **Areas:** release, transport

**Decision.** Release assets are served from the repository's GitHub releases, `plugin/release-url.txt`
names that address, and `publish.sh` fetches every asset back over the public URL and hashes it
against `checksums.txt` before reporting success.

**Support.**
- GitHub serves a release's assets at `{owner}/{repo}/releases/download/{tag}/{asset}`, and
  `install.sh` builds `${base}/v${want}/${asset}`, which is the same shape, so the installer needs no
  change and the move is `release-url.txt` and `publish.sh`. `plugin/hooks-handlers/install.sh`,
  `scripts/publish.sh`.
- The host is one nobody here operates and the transport is HTTPS, and trust is unchanged, since the
  sha256 in `checksums.txt` is what authorizes a binary. D-066 (the binary is fetched and verified,
  never shipped in the plugin), D-115 (a centralized component must trace to a disclosed tradeoff
  that benefits the person).
- A release that was created but serves something else fails at the person who published it and not at
  the person installing. `scripts/publish.sh`.

**Rejected.**
- *Serving release assets from a personally-operated droplet over plain HTTP.* It is a component that
  failed D-115's benefit test, serving the maintainer and not the person installing, and it is an
  availability dependency that a release host nobody here operates is not.
- *A DNS name in front of the droplet.* It hides the address without removing the component, which is
  still a machine one person operates with plain HTTP or a certificate to keep alive.
- *Publishing to both the droplet and the release.* Two hosts for one pin, so the first question after
  a failed download is which one served it.

**Revisit when** GitHub stops serving assets at `/{tag}/{asset}` under the download base. `install.sh`
builds that URL by concatenation, so a changed convention arrives as a 404 and a source build, and not
as a message saying the convention changed.
