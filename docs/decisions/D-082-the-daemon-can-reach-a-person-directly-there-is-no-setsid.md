# D-082 — The daemon can reach a person directly; there is no setsid hazard

**Date:** 2026-09-20 · **Status:** active (measured; supersedes an earlier false finding)

**Context.** Discovery of the browser view was blocked on a question nobody had
answered: how a first-time user, who cannot be told a loopback address, ever arrives
at it. Two candidate channels were investigated — writing to the terminal directly,
and OS notifications — along with auto-opening the view.

**Auto-open works, and takes focus.** A process started the way the session-start
hook starts the daemon (`nohup … &`, no controlling terminal) can call `open` and the
browser becomes frontmost. Measured end to end: a detached, daemon-shaped process
that waits six seconds and then opens the view brought Chrome to the front and opened
the tab. This is the mechanism discovery now rests on, and it is recorded as **B23**.

**An earlier finding said the opposite, and was wrong.** A test reported that a
detached `open` never took focus, and it was attributed to macOS focus-stealing
prevention. The test ran `setsid nohup open …`, and **macOS ships no `setsid`** —
`command -v setsid` returns 1, there is no binary anywhere. The command failed, its
error went to `/dev/null`, and `&` backgrounded the failure. Nothing was ever
launched. The browser did not open a background tab; it did not open at all.

**So the setsid branch in `start_daemon_if_needed` is not a hazard.** It was briefly
believed to be one — that a machine with Homebrew's `util-linux` on PATH would flip
the branch and silently strand the daemon outside the GUI session. That was tested
directly, using Go's `SysProcAttr{Setsid: true}`, which performs the real `setsid(2)`
on darwin even though the binary is absent:

| | POSIX session inherited | new POSIX session |
|---|---|---|
| `launchctl managername` | `Aqua` | `Aqua` |
| `open <url>` | opens, takes focus | opens, takes focus |

The Mach bootstrap namespace is inherited separately from the POSIX session, so
detaching costs nothing. **No code change was made**, and a comment now records why,
because the wrong inference is an easy one to repeat.

**Terminal writes: possible, rejected.** A hook cannot inherit a tty — `/dev/tty` is
`ENXIO` inside a tool call — but it can walk its parent chain to the `claude` process
and read the tty from `ps`, and the daemon can then write to `/dev/ttysNNN`. Only OSC
sequences are safe, since they set terminal chrome rather than grid cells. Rejected
anyway on three counts: the title is contended with Claude Code, which overwrites it;
a `write(2)` can splice into the middle of a sequence Claude Code is emitting, because
a tty offers no `PIPE_BUF` atomicity guarantee, so the failure mode is intermittent
corruption of somebody's live working session; and nothing there can carry something
clickable. A channel that can only say "something happened" is not worth that risk.

**Notifications: kept, with a stated limit.** `osascript -e 'display notification'`
delivers reliably and persists in Notification Center — a banner missed at the time
was found there afterwards. But it posts under Script Editor's identity, and clicking
it does nothing. More importantly the delivery is **unobservable**: the notification
database is unreadable and the Focus state is TCC-protected, so the daemon cannot
learn whether a notification was shown, suppressed, or dismissed unseen. Against
D-014 that settles its role — fit for announcing, unfit for anything that must be
known to have landed.

**Not adopted: an `.app` bundle.** It was built and tested. It would buy a proper
notification identity, a click that opens the view, and a Spotlight-findable icon.
None of that is needed once plain `open` is known to work, and a bundle would have to
be created at install time and registered with LaunchServices. Revisit if a
notification ever needs to be clickable, or if macOS withholds focus from background
processes — which is exactly what B23 watches for.

**Revisit when:** B23 fails, or a notification needs to carry an action.
