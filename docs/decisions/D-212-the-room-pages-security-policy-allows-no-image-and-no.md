# D-212 — The room page's security policy allows no image and no request to another address

**Date:** 2026-10-14 · **Status:** active · **Areas:** view, trust

**Decision.** The daemon sends the room page with a security policy that tells the browser to load no image
and to contact no address except the daemon. A mistake in the renderer then cannot make the browser fetch
an address a colleague chose, which would tell the colleague when the reader looked.

**Rejected.**
- *Relying on the renderer alone.* One mistake would expose the reader.

**Revisit when** a pasted image is to be shown.
