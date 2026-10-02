# D-085 — A line telling somebody to run `cogmer` is a line that fails

**Date:** 2026-09-20 · **Status:** active (implemented)

**Context.** Reviewing documentation for the assumption that every user is a
person. The tone was the smaller problem.

**The instructions did not work.** `/peer-pair` told people to run
`cogmer pair <string>` in a terminal; `/room-join` told them to run
`cogmer verify`. The installer puts the binary in `~/.cogmer/bin`, which is
deliberately not on PATH — §29 forbids editing a shell profile to put it there, and
`cli.sh` exists precisely because slash commands hit this. So those lines produce
`command not found` for **every plugin-installed user**: not an edge case, but the
default install, and the people least equipped to diagnose it.

**The binary prints its own invocation.** `invocation()` resolves `os.Executable()`,
spells `$HOME` as `~`, and is used wherever a person is told to go and type
something. A path the binary reports about itself cannot go stale, and it is
correct for a from-source install and a plugin install alike.

**The command docs now relay rather than restate.** They are told explicitly not to
shorten it, and `/peer-pair` walks through opening a terminal, pasting, and pressing
return — because somebody being asked to compare two spoken words with a colleague
may never have opened one.

**Not done: making pairing work from a slash command.** It was raised, and it needs
the view to have somewhere to show the words. That belongs with D-083, not here.

**Revisit when:** the install location changes, or pairing gains a surface that is
not a terminal.
