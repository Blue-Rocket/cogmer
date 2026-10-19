# D-218 — The export is the whole room as text-only Markdown

**Date:** 2026-10-16 · **Status:** active · **Areas:** view

**Decision.** The room view's Export gives the whole room as one Markdown file with no pictures, so that a room
can be kept as a reference after it is abandoned. A reader who wants part of a room selects and copies the
text, and a reader who wants the pictures saves the whole page with the browser.

**Rejected.**
- *Controls to choose which turns are exported.* A reader can select and copy any part of the text.
- *Pictures in the export, as a zip or an HTML page.* The browser saves the page as it shows it, pictures
  included.
- *Other forms, such as JSON or HTML.* Markdown reads as it is and can be given to a model.
