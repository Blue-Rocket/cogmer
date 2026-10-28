# D-267 — An image is stored and shown only when its header puts it within a pixel limit

**Date:** 2026-10-28 · **Status:** active · **Areas:** capture, sync, trust

**Decision.** A pasted image is sent, stored and shown only when its header gives a width and a height of at most 16,384 pixels each and at most 40 million pixels together, as well as being at most 5 MB (D-213, a pasted image is sent to peers and each receiver stores it as a file). The sender reads the header before it hashes the image, and an image past the limit is named in the turn's event with a note that says so and no bytes, which the view words as having more pixels than cogmer sends. A receiver reads the header of the bytes it fetched, after the hash and the type have matched, and refuses and discards an image past the limit, since an event names an image by its author's word. An image whose header cannot be read is refused. PNG, GIF and JPEG headers are read by the standard library's header reader, which stops before the picture's data, and a WebP's by a reader of its three header forms.

**Rejected.**
- *Checking only the size in bytes.* A file of 5 MB can hold a picture of hundreds of megapixels, which a browser decodes into hundreds of megabytes and stalls on, and a colleague's turn can ask every other colleague's browser to do it.
- *Resizing an image that is too large.* D-213 rejects re-encoding, since it changes what the author pasted.
- *Checking only on the sender.* A receiver cannot tell a sender that checked from one that did not.

**Limits.** The limit is a judgment: 40 million pixels holds a screenshot of an 8K display and decodes to about 160 MB. The limit applies when an image is sent or fetched, so an image in a store is not read again, and a frame count is not limited, so a long animated image is limited only by its bytes. A header the readers cannot read, such as a JPEG with an unsupported feature, is refused though a browser might show the picture.

**Revisit when** a user's image is refused by the limit and the user needs it shown.
