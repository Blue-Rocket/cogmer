# D-212 — The room page's security policy allows only the daemon's own images and requests

**Date:** 2026-10-14 · **Status:** active · **Areas:** view, trust

**Decision.** The daemon sends the room page with a security policy that tells the browser to load images
from the daemon alone and to contact no address except the daemon. A mistake in the renderer then cannot make
the browser fetch an address a colleague chose, which would tell the colleague when the reader looked. The
daemon shows a pasted image from its own address, builds each image element itself from a hash it has
checked, and serves the image with the type its bytes show and with `nosniff`.

**Rejected.**
- *Relying on the renderer alone.* One mistake would expose the reader.

**Revisit when** an image is to be loaded from anywhere but the daemon.
