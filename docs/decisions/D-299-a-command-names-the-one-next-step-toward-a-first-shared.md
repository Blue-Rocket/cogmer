# D-299 — A command names the one next step toward a first shared room, from the state of the machine

**Date:** 2026-11-25 · **Status:** active · **Areas:** commands

**Decision.** `/cogmer:self-status` and `/cogmer:self-doctor` include one line that begins "Next step", and the output of `/cogmer:peer-pair` names the step after its two words match. The line is chosen from what this machine holds: with no peer it says to pair, with peers and none of them verified it says to finish pairing, with a verified peer and a session in no room it says to create or join a room, and with a room in which nobody else may enter it says to invite a colleague. Once a colleague may enter the session's room it says nothing, and a machine with no user identity gets no line, since the command that needs the identity points to the page that makes it (D-230, a user identity is made in the daemon's page the first time the user needs one). Each command file tells the model to give the line as the last sentence of its answer, since a command's output reaches the user only through the model. One function chooses the line, so that every command tells a user the same next step.

**Rejected.**
- *Each command naming a next step only for its own case.* A user who has run only `/cogmer:self-status` is told how to pair and not that a room comes after, and which step they are told depends on the command they happened to run.

**Limits.** A cloud container is told no step, since its session reaches its user's colleagues and rooms through the user's own machines and its own state holds none of them. The state is this machine's: a colleague who has not yet paired from their side is not known, so a user is told to invite one who may not be ready. The line reaches the user only if the model relays it, and a model can leave it out.

**Revisit when** a command's output can reach the user without passing through the model.
