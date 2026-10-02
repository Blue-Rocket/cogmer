# D-178 — The files in plugin/commands are prompts, so they carry no roadmap or commentary

**Date:** 2026-09-20 · **Status:** active · **Areas:** commands

**Decision.** A file in `plugin/commands/` is a prompt that the model reads and not
documentation that a person reads, so it holds no roadmap, no provisional note and no
commentary, and the record of interim state belongs in the decision log.

**Support.**
- The model reads a command file as instruction, so a remark about what is planned would be
  read as something to act on. `plugin/commands/peer-pair.md`.
- Writing tests do not check these files, because they are instructions to the model.
  `docs/writing.md`, "Enforcement".

**Rejected.**
- *Marking a command file as provisional.* The mark would be read as an instruction.
