# D-223 — The recovery phrase is typed only into the daemon's browser page

**Date:** 2026-10-18 · **Status:** not built · **Areas:** identity, view, trust

**Decision.** The daemon's page in the browser is the only place that takes the recovery phrase, and no command, argument, prompt or terminal question does. Anything typed into an assistant goes into the model's context and to the service that runs it, so a phrase typed there is a copy held by others, and a command's arguments pass through the assistant too. A cloud session never asks for the phrase, since its container is not a place the user controls.

**Rejected.**
- *A command that takes the phrase as an argument.* The argument is part of what the user types to the assistant.
- *A terminal question that applies when no browser opens.* The output of a command an assistant runs is read by the model, which is why the pairing ceremony shows its words in the page (D-088, the pairing ceremony lives in the view), and in a cloud container nobody can answer a terminal question.

**Revisit when** a user has to recover on a machine with no browser.
