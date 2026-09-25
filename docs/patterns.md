# Patterns

Design for the case you have, and never build a ceiling you cannot remove. Work
that grows with the number of participants is a cost and is acceptable. A format that
cannot carry more than the present number is permanent once records exist in it, so
it has to be argued for.

Do not abstract for a consumer that does not exist. An interface written for a
second consumer nobody has examined encodes guesses about it, and the guesses are
most likely wrong where the two consumers differ most. Name the future case if it
helps, and let naming it license nothing.

Keep centralized components few, and name every one. Each is a place where the system
can fail, be observed, or be controlled by whoever operates it, and a component that
nobody has named is one that nobody has weighed.

A change to a stored format migrates existing stores. A statement that creates a
table only when it is missing leaves an older table unchanged, so a column added later
is absent from every store created before it, and the failure appears at the first
query rather than at open.

Add complexity only when a test shows the need. A mechanism kept for a problem
nobody has observed costs its full price now for a benefit that may never arrive.

An integration fails open. When a hook or plugin fails, the host it extends
behaves as it would without it: no error shown, no output added, no delay, no
lost context. Losing the integration's feature is acceptable; degrading the host is
not.

Frame untrusted input by classification, never by a static delimiter. Text from
another party is labelled as data from that party, and the boundary around it is a
value the text cannot know, so the text cannot close the boundary and speak as if from
outside it. The framing is restated after the untrusted text as well as before it.

Loopback is not a security boundary. A web page open in the same machine's
browser can send requests to a local service. Every request that changes state
requires a header a page cannot send, so an unmarked request fails closed.

A secret never shares a file with anything that gets printed or shared. A file
that is shown to people is eventually shown whole, so the secret lives in a file of
its own, and a test asserts it never appears in the other.

Authentication, admission and verification are three questions, and each is
answered separately. Proving a key's possession, being on a list, and having confirmed
the key belongs to the person meant all pass equally for a key substituted in transit
until the last is answered. Records from a party that is not yet verified are held,
never dropped.

A signed or immutable record is never rewritten. Its identifier, author and
sequence number stay as first written, and a change to what is signed adds a new
signing scheme beside the old ones, because a record signed under an old scheme must
still verify.

Persist before publishing, and reserve an identifier before publishing anything
that uses it. The reverse can announce something with no durable record of it, which
a restart then contradicts.

Key on stable identifiers, never on names people choose. Names collide, get
changed and get guessed, and a record keyed on one attaches to the wrong thing when
that happens.

Confirm an effect by observing it, never at the moment it is sent. A reply can be
lost after the effect happened, or the effect can fail after the reply was sent, and
marking success at send time turns either into a silent, permanent error.
