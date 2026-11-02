# D-281 — A command in a cloud session with no value says the user must first set cogmer up on one of their own computers

**Date:** 2026-11-02 · **Status:** not built · **Areas:** commands, identity

**Decision.** A command that needs the user identity, run where `CLAUDE_CODE_REMOTE` is `true` and the daemon has found no cloud value in the variable or in a plugin file (D-279, a cloud session's daemon reads its cloud value from the plugin files it finds, as well as from the variable), does nothing and says that this session shares nothing until the user has cogmer on one of their own computers and adds a cloud environment from its machines page (D-246, a cloud environment is added from the machines page, which shows the one value to paste into it), and it points to the install section of `README.md`. It never opens the identity page and never mentions the recovery phrase, because a cloud session never takes it (D-223, the recovery phrase is typed only into the daemon's browser page). Where the daemon holds a value it has not yet read, or one that does not read, the command says which, and names both places a value can come from.

**Rejected.**
- *The words for a machine with no identity, which send the user to the identity page.* The page is on the sandbox, which the user cannot reach, and it asks for a phrase that a cloud session never takes.
- *Starting no daemon in a cloud session until a value exists, so that no command has anything to say.* The user then sees nothing, and cannot tell a session that shares nothing from one that works.

**Limits.** The words cannot tell a session whose plugin has not synced yet from one with no plugin, since a plugin's files can arrive after a session's first command (D-279).

**Revisit when** a user is found to run a command in a cloud session with no value and still not know what to do.
