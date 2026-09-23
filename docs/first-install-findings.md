# First install from the public repository

**Run:** 2026-09-22, on a MacBook (darwin/arm64) with Claude Code 2.1.280 and cogmer
v0.7.1, installed from the marketplace in `Blue-Rocket/cogmer`. One test ran with
`claude -p --plugin-dir ./plugin` against a fresh `COGMER_HOME`.

**Result:** Installing worked, but none of the first commands a new person runs gave
a usable answer without a restart and manual cleanup.

## What we found

### The README's install block pastes as one command

Both install commands shared one code block. Pasted together, Claude Code read them
as a single `/plugin marketplace add` whose repository was "Blue-Rocket/cogmer
/plugin install cogmer@blue-rocket", and refused it. Fixed in `b613e9e`.

### A plugin installed mid-session never fetches its binary

The plugin's commands worked in the session that installed it, but its SessionStart
hook never ran there. `~/.cogmer` had no `bin/`, no `install.log` and no
`install-state`. Claude Code's documentation says a plugin installed mid-session is
active from v2.1.221, and SessionStart's trigger values do not include a plugin
reload.

### A command whose shell line fails produces no model turn

When a `!` line in a command exits non-zero, Claude Code abandons the command. In an
interactive session the person sees the output under "Shell command failed for
pattern …", labelled `[stderr]`. With `claude -p` the result is the empty string.
`cli.sh` exits 1 whenever there is no binary, and `/cogmer:room-status` fails the
same way outside a room, because `cli.sh guests` exits 1.

### The download takes 13 to 16 seconds

The darwin/arm64 binary is 29.3MB. It took 16s in the `-p` test and 13s in the
interactive install.

### A daemon from another state directory blocked the real one

A test daemon, with its own `COGMER_HOME` and hooks on 4799, still held the default
peer-sync port 4783. The real daemon exited with "peer sync cannot be served:
127.0.0.1:4783 is held by something that is not a cogmer daemon". It was a cogmer
daemon. Stopping it by pid and starting the real daemon again published a `tc://`
address.

### `whoami` printed a pairing string nobody could use

With no daemon serving, `whoami` printed a pairing string with the address
`127.0.0.1:4783` and a note after it. `/cogmer:self-status` advised holding off, then
gave the string as "provisional" and "safe to send". The same output printed the
username notice twice.

## What this does not show

One machine, one person, and a person who wrote the software. A colleague's first
install is still to come.
