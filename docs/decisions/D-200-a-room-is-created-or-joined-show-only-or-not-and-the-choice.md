# D-200 — A room is created or joined show-only or not, and the choice never changes for that room

**Date:** 2026-10-12 · **Status:** active · **Areas:** rooms, commands, trust

**Decision.** `create` and `join` take `--show-only`. A room created or first joined with it gives none of
its conversation to this machine's Claude sessions: the prompt hook captures the session's prompt and
offers nothing. The setting is stored with the room on this machine. A request for the other setting for a
room recorded earlier is refused, and the store refuses any change to it.

**Support.**
- Injection carries a colleague's conversation into a user's session, and so to that user's model provider
  under their account, which a user may need not to happen. §28 (configuration).
- A setting that could change would give a session what it was promised it would not receive, or leave in its
  context what could not be withdrawn, and what a session has been told cannot be withdrawn. D-056 (a
  session's room is fixed when its first hook fires, and never changes), §12a.
- The store refuses an update that changes the setting, so no code path can change it by mistake.
  `cmd/cogmer/membership.go`, `rooms_show_only_is_fixed`, `cmd/cogmer/showonly_test.go`,
  `TestTheSettingCannotChangeForARoom`.
- A `Room` built without the setting injects, so code that builds one cannot turn injection off. A
  membership database from before the setting reads every room as injecting.
  `cmd/cogmer/membership.go`, `Room`, `cmd/cogmer/showonly_test.go`,
  `TestAMembershipDatabaseFromBeforeTheSettingKeepsInjecting`.
- The prompt of a session in a show-only room is captured, since the room's members read it in the view, and
  nothing is read as pending or offered. Breaking the check, the trigger, the migration, the refusal and
  the reading of the setting each failed a test. `cmd/cogmer/daemon.go`, `handlePrompt`,
  `cmd/cogmer/showonly_test.go`, `TestAShowOnlyRoomGivesNothingToTheSession`.

**Rejected.**
- *A setting that can be changed afterwards.* A session told that nothing would arrive would then receive
  something, and a session that had received something could not be made not to have.
- *A room-level setting carried in the invitation and applied to every member.* Only a member's own
  machine decides what its sessions receive, so a host could request it and could not guarantee it.

**Limits.** The setting protects the person who makes it. A colleague's machine gives their sessions this
user's turns unless they chose show-only too, and the command says so when the user chooses it. The user's own
turns are shared with the room as in any other.

**Revisit when** somebody needs a room that shares nothing of their own either, or a host needs to ask members
for show-only.
