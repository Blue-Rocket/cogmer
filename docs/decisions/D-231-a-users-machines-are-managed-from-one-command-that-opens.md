# D-231 — A user's machines are managed from one command that opens the view

**Date:** 2026-10-24 · **Status:** not built · **Areas:** commands, view, identity

**Decision.** `/cogmer:self-machines` opens the daemon's page, where the user adds, removes and renames machines and recovers, and `/cogmer:self-status` also lists the user's machines and whether each can be reached from this machine. Recovery takes the phrase only in the page (D-223, the recovery phrase is typed only into the daemon's browser page), and adding a machine shows its words there (D-229, a machine is added from another of the user's machines, with the two-word check), so every act on the user's machines happens where a person sees it without the model (D-086, the terminal is not a user experience, and the view is the surface).

**Rejected.**
- *A command for each act, under `self-`.* Each would only open the same page, so the plugin would list several commands that do one thing.
- *A `machine-` prefix.* A prefix names what a command is about (D-096, a command prefix names its target, and `/self-status` is where you are), and a user's machines are the user's, which `self-` names.

**Revisit when** an act on the user's machines is found that needs neither the phrase nor a comparison and that users want without opening the page.
