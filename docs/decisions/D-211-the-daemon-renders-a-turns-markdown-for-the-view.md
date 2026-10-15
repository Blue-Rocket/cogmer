# D-211 — The daemon renders a turn's Markdown for the view

**Date:** 2026-10-14 · **Status:** active · **Areas:** view, trust

**Decision.** We use Go code in the daemon to render the Markdown in each turn into HTML, because that is the
safest place to do it, and the page shows the result. Links open only for web addresses, and an image appears
as its address and is never loaded.

**Rejected.**
- *Rendering in the page, by hand or with a script.* Each rule would depend on code that tests cannot run.
- *Dropping raw HTML, as goldmark does.* A turn about markup would lose the markup it names.
- *Showing every link as text.* A reader would copy addresses by hand.

**Revisit when** a reader needs a link that is not `http` or `https`, or an image that a turn's Markdown names.
