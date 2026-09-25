# Patterns

Design for the case you have, and never build a ceiling you cannot remove. Work
that grows with the number of participants is a cost and is acceptable. A format that
cannot carry more than the present number is permanent once records exist in it, so
it has to be argued for.

Do not abstract for a consumer that does not exist. An interface written for a
second consumer nobody has examined encodes guesses about it, and the guesses are
most likely wrong where the two consumers differ most.

Keep centralized components few. Each is a place where the system can fail, be
observed, or be controlled by whoever operates it.

When you change a stored format, migrate every store that already exists. A statement
that creates a table only when it is missing leaves an older table unchanged, so a
column added later is absent from every store created before it, and the failure
appears at the first query that names the column rather than when the store opens.

Add complexity only when a test shows the need. A mechanism kept for a problem
nobody has observed costs its full price now for a benefit that may never arrive.

Never let an extension break its host. When a plugin or hook fails, leave the host
behaving as it would without it: no error shown, no output added, no delay, no lost
context. Otherwise installing the extension leaves its user worse off than not
installing it.

Wrap untrusted text in a boundary its writer cannot forge. Mark where it starts and
ends with a value that does not occur anywhere in the text, such as a random value
checked against it. A marker that the text contains ends the wrapper early, so that
whatever follows is read as if it came from outside.

Do not trust a request because it comes from the same machine. Any web page open in
that machine's browser can send requests to a service listening there, so a local
service that accepts requests unchecked will act for, and answer, any site its user
visits. Require every request to carry a combination of things that an external page
cannot supply.

Keep a secret in a file of its own, apart from any file that gets printed or shared,
and test that the secret never appears in a file that does. A file that is shown to
people is eventually shown whole.

Answer authentication, admission and verification separately. Proving that a party
holds a key, finding the key on a list, and confirming that the key belongs to the
intended person are three checks. A key substituted in transit passes the first two,
and only the third catches it.

Keep the code for every format a record has ever been written in, each paired with a
version number that every record carries. When the structure of a record changes,
add a new format under a new version, and never edit or remove an older one. Records
that others hold cannot be rewritten, so each can be read, or its signature or hash
checked, only with the format its version names.

Persist anything before publishing it. Otherwise a restart can lose what was
announced, and the system then contradicts its own announcement.

Never derive the next identifier from the records it numbers when those records can
be lost while their issuer carries on. Keep the counter where it survives that loss,
and record each number before using it. A counter derived from the records restarts
when they are lost, and reissues identifiers that others already hold for different
things.

Key a record on an identifier that never changes, and keep anything that can change,
such as a name, as a field of the record. When a key changes, records keyed on the
old value point at nothing, or at whatever takes that value next.

Identify a party by something only that party can prove it holds, such as a key
pair, never by a name it gives. Anyone can give any name, so a name lets one party
pass as another.

Treat something as delivered only once the receiver has said it has it, whether by
acknowledging it or by asking for what comes after it. A message can be lost after it
is sent, and a sender that records delivery at send time never sends that message
again.
