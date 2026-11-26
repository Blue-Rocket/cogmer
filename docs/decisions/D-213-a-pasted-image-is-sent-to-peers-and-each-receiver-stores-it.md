# D-213 — A pasted image is sent to peers and each receiver stores it as a file

**Date:** 2026-10-15 · **Status:** not built · **Areas:** sync, storage, capture

**Decision.** An image a user pastes into a session travels with its turn to the room's other machines, and each receiver stores
the image as a file, with a reference to the file in the database. The signed event carries a hash of the
image, and the image moves as raw bytes in a request of its own, so a receiver can fetch the image from any
machine that holds it, check it, and keep a slow image from delaying a turn. A screenshot is about 250 KB, many times the size
of a turn's text, so the database holds a reference and never the bytes, and an image over 5 MB is sent as a note that
gives its size.

**Rejected.**
- *Storing the image in the database.* The database would grow by the size of every image.
- *Storing only a note that an image was attached.* The colleague would see that something was left out and
  could not see the image.
- *Fetching an image only from its author.* A colleague could not see an image while its author was offline.
- *Compressing or re-encoding the image.* A PNG is compressed when it is made, and re-encoding changes what the author pasted.
