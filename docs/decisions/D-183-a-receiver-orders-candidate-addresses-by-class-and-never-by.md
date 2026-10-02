# D-183 — A receiver orders candidate addresses by class and never by a configured sort

**Date:** 2026-09-20 · **Status:** not built · **Areas:** transport

**Decision.** The receiver orders the addresses it will try. A remembered winner comes first,
carrying the time it worked and discarded on failure, and the rest are ordered by class: the
same host, the same network, public and relayed. The first few are tried at once. There is no
configurable sort, and a requirement that arrives as an order is met as a filter.

**Support.**
- The receiver holds the information, so any order the sender expresses is a preference and
  not knowledge, and the receiver would honor it on the word of the peer whose address it is.
  RFC 8305 (Happy Eyeballs version 2), https://www.rfc-editor.org/rfc/rfc8305.
- Trying the first few at once makes a dead candidate cost a round trip and not a timeout, and
  a learned winner dominates any static order after the first success, which makes a
  configured order dead weight that is wrong in the cases it was added for. D-091 (identity is
  advertised and location is discovered).
- Every real requirement that arrives as an order is a filter, such as never using a relay for
  a site that will not route through third-party infrastructure, or never exposing a direct
  address, and a preference order cannot express never.
  `f288913:docs/decisions/D-091-identity-is-advertised-location-is-discovered.md`.

**Rejected.**
- *A configurable sort.* It is wrong where it is needed.
- *An order the sender expresses.* It is a preference, not knowledge.
- *A remembered winner kept as a record.* It is a hint, which is why it carries its time and
  is discarded on failure.
