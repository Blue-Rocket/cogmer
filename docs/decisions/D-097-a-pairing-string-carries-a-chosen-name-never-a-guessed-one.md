# D-097 — A pairing string carries a chosen name, never a guessed one

**Date:** 2026-09-20 · **Status:** active (implemented) · **Completes** D-094's revisit

**Context.** D-093 made a label compulsory at pairing, which meant somebody had to
type a name for a colleague whose name they obviously know. D-094 wanted the default
to be the name the peer picked, and D-095 made picking one possible. This connects
them.

**The format gains an optional trailing name:**

```
ed25519:KEY@ENDPOINT#Alice
```

`#` separates it, matching the invitation format's use of the same character for a
trailing identifier. Strings without one are ordinary rather than erroneous, so
anything issued before today still parses. The endpoint is untouched, including the
`tc://` form, because the name is stripped before the address is split.

**A guessed name is never carried.** `chosenName` returns nothing while
`NameChosen` is false (D-095). `Ec2-user` arriving as though somebody picked it is
worse than carrying nothing at all: the derived name is at least honest about being
machine-made, whereas a username presented as a name is a claim about a person that
nobody made.

**The receiver's order is: what they typed, then what the sender said, then ask.**
And when the sender's name is used, it says so, because adopting somebody's
description of themselves as your own label is a small act worth stating rather than
performing silently.

**The name is a claim by whoever sent the string**, and a substituted string carries
a substituted name — so a default of "Alice" might come from Mallory. That is safe
only because the label is written after the two words match (D-093), by which point
the string demonstrably came from the person on the call. **Do not move that write
earlier.** A mismatch removes the row entirely, so a substituted name never reaches
the list.

**It is reduced before use.** The name arrives from another machine and becomes a
unique key and an argument to `resolvePeer`, so `sanitizeName` strips what would
break the format (`#`, `@`), span lines, or be invisible, collapses runs of
whitespace, and caps the length at something readable in a list. Asserted both on
what it produces and on the property that matters more: whatever it does, the
endpoint survives.

**Revisit when:** the string carries a third thing, at which point `#` is a
separator with two jobs and wants a real encoding.
