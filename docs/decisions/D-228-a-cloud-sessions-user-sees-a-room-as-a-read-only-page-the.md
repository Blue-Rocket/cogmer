# D-228 — A cloud session's user sees a room as a read-only page the session hands to the app

**Date:** 2026-10-20 · **Status:** not built · **Areas:** view, trust

**Decision.** A user in a Claude Code cloud session sees a room as a page the daemon writes to a file. The session hands the file to the Claude app to show, and the page changes only when the daemon writes it again. The view's address is loopback inside the container (D-030, two listeners, hooks on loopback), so a file the session sends to the app is the one route to the user that needs nothing listening outside the container. Pairing and every other act that changes state happen in the view on one of the user's machines, since a cloud session acts for a user its colleagues verified there (D-216, a cloud session signs with a key of its own that the user's shared key vouches for).

**Rejected.**
- *Exposing the container's view through a public tunnel, with a secret in the address.* The view would listen beyond loopback, every route that changes state, the pairing confirmation among them, would answer anybody holding the link, and the tunnel would be a centralized component that must trace to a disclosed tradeoff (D-115).
- *Confirming a pairing from the cloud session by telling the assistant the words matched.* The words and the answer would both pass through the model, which is what showing the ceremony in the page avoids (D-088, the pairing ceremony lives in the view).

**Limits.** The page is a snapshot, so a colleague's turn that arrives after it was written is absent until it is written again. This entry does not decide the page's form, which command writes it, or where in the container the file is written.

**Revisit when** a cloud session can serve a page that the user's browser reaches while the view listens only on loopback, when a user in a cloud session needs to act on a room rather than read it, or the Claude app on a phone shows no page that a cloud session sends.
