# D-296 — A stop call that carries no transcript path counts the turns offered at its prompt as delivered

**Date:** 2026-11-25 · **Status:** active · **Areas:** capture, hosts

**Decision.** When the Stop hook of Claude Code sends an empty `transcript_path`, the daemon counts the turns it offered at that prompt as delivered when the stop call arrives, by the prompt identifier the stop carries, which is when D-206 (a Cowork turn counts as delivered when the stop call for its prompt arrives) counts a Cowork turn. A session whose hooks send no transcript has no injected block to find, so confirming from the transcript would leave every turn pending and offer the whole room again at each prompt. This is an exception to D-014 (derive delivery state from transcript evidence) and to D-131 (delivery is trusted without evidence only once the evidence is known broken) for a session that has no transcript, and a path that is given and is refused (D-262, a transcript is opened only when it is a regular jsonl file) confirms nothing.

**Rejected.**
- *Leaving the turns pending when there is no transcript.* Each prompt carries every earlier turn of the room again, and the prompt grows with the room. In a Cowork cloud task on 2026-11-25 the daemon offered 21 events at two prompts in a row (`150deeb`).
- *Reading the transcript from another place in the sandbox.* A search of the task's sandbox found no transcript for its session and no process of Claude Code visible from its shell (`ec44c7a`).

**Limits.** No check of the host's behavior runs in a cloud task, since its behavior check fails there with an authentication error, so nothing detects a change in how the host passes a hook's reply to the model. A turn whose injected context the host dropped, with the stop call arriving, is counted delivered and lost. A prompt held while a daemon starts (D-294, a prompt that finds no daemon listening just after one was started is held for it) offers nothing, so it has nothing to count.

**Revisit when** a Cowork cloud task sends a transcript path, or a hook there can see what the model was given.
