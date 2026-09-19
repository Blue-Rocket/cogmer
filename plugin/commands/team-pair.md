---
description: How to pair with a colleague (runs in a terminal, not here)
allowed-tools: Bash(claude-team whoami:*)
---

## This session's pairing string

!`claude-team whoami`

## Your task

Pairing cannot be done from here, and this command does not attempt it. Tell the
user to run it in a terminal, on a call with their colleague, **at the same time as
each other**:

```
claude-team pair <their pairing string>
```

Both of them will see two words. They say them aloud and confirm they match. Explain
why it must be a call rather than a message: the check is that the voice is the
colleague, and a second written channel proves nothing that the first did not.

Give them the pairing string from the output above to send to their colleague. It is
safe to send by any means — it is a public key and an address, and holding it admits
nobody.

If the output warns that the address is loopback, say that their colleague cannot
reach them until that is changed.
