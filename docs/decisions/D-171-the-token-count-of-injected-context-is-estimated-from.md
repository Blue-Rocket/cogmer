# D-171 — The token count of injected context is estimated from characters

**Date:** 2026-09-18 · **Status:** active · **Areas:** capture

**Decision.** The token count of injected context is estimated as the character count
divided by four, and no tokenizer measures it.

**Support.**
- A tokenizer would have to track whichever model the session uses, which cogmer does
  not choose, and the figure is for judgment and not for arithmetic. §21 (context window
  management), `cmd/cogmer/daemon.go`, `estimatedTokens`.

**Rejected.**
- *Measuring the count with a tokenizer.* It would have to follow a model cogmer does
  not control.
