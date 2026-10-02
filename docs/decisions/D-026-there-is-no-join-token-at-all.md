# D-026 — There is no join token at all

**Date:** 2026-09-16 · **Status:** active · **Areas:** admission

**Decision.** There is no join token, code or invitation secret, and nothing a user can
hold admits them to a room. Admission is an entry on a guest list, proved by possession
of a key, or a present host's approval of a request.

**Support.**
- A host who knows a guest can admit them and leave, and the guest joins whenever it
  likes, with no host present and no token. D-024 (admission is a guest list).
- A token would be needed only by a host who is absent and has never recorded the guest,
  and such a host has to act before that guest can enter under any scheme, since
  somebody has to issue the token. §12 (forming a room).
- A token admits whoever holds it, so it has to be kept secret in transit, cannot be
  checked afterwards, and enrolls whoever intercepts it under a name the room's members
  will treat as familiar. §12.

**Rejected.**
- *A code for inviting in advance, when the host will not be present.* The host has to
  act anyway, and acting once through the guest list buys the same.
- *A join code as a fallback for strangers who have not paired.* A host who is present
  can approve a request instead (D-144), and a code carries every weakness of a bearer
  credential.
- *Codes as an optional convenience.* A credential that exists will be used, and its
  weaknesses do not become optional with it.

**Limits.** Pairing with someone entirely unknown needs a host present to approve it.

**Revisit when** users need to invite someone they have never paired with while the
host is away.
