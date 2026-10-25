# A cloud session in a room with a real machine, through the real relay

This is working material for the open item in `docs/open.md` that opens "No cloud session has
shared a room with a paired machine", and is deleted with it. It is a run to follow, in two
parts: part 1 on the maintainer's machine, wonka, and part 2 in a new cloud session. The
container tests show the same flow with direct addresses. This run shows it through the public
relay, between wonka and a Claude Code cloud container.

## Rules for whoever follows this

- Never point the branch build at the real `~/.cogmer`. Its first start migrates the state
  directory and writes data version 1, and the installed plugin, 0.10.5, reads no data version,
  so its hooks would go on running against migrated state and break the user's real rooms.
- Never put the branch build in `~/.cogmer/bin`. The installer replaces a `dev` build there with
  the version its plugin expects at the next session start.
- Never read, print, log or pass on the cloud environment's value. Whoever holds it speaks as the
  user (D-246, a cloud environment is added from the machines page, which shows the one value to
  paste into it). The person copies it from the page into the environment's settings. An
  assistant does not request `/machines/cloud/add` itself, read the page's response, or ask for
  the value to be pasted into a chat.
- The test identity is a throwaway. Its recovery phrase needs to be kept only until the run is
  done, and is typed only into the page.

## Part 1: on wonka

Run these from a clone of the repository. The same variables go on every command, so define
them once in the shell:

```
export T_HOME="$HOME/.cogmer-test"
export T_ENV="COGMER_HOME=$T_HOME COGMER_ADDR=127.0.0.1:4792 COGMER_PEER_ADDR=127.0.0.1:4793 COGMER_HOST=claude COGMER_PREFLIGHT=off"
```

1. Build the branch outside `~/.cogmer`:

   ```
   git fetch origin user-identity && git checkout user-identity
   go build -o /tmp/cogmer-test ./cmd/cogmer
   /tmp/cogmer-test version
   ```

   `version` prints `dev`.

2. Start a test daemon beside the real one, with its own state directory and ports, in a
   terminal of its own or detached with its output to a file:

   ```
   env $T_ENV /tmp/cogmer-test daemon > /tmp/cogmer-test.log 2>&1 &
   ```

   Check that it answers, and that the real daemon on 4782 is untouched:

   ```
   curl -s http://127.0.0.1:4792/healthz
   curl -s http://127.0.0.1:4782/healthz
   ```

   Its relay key is in `$T_HOME`, so it has a relay address of its own. Wait until
   `/tmp/cogmer-test.log` shows "relay connection carried", then check that
   `env $T_ENV /tmp/cogmer-test whoami` gives an address beginning `tc://`.

3. The person opens `http://127.0.0.1:4792/identity` in a browser, answers that cogmer has not
   been used on another machine, keeps the recovery phrase until the run is done, confirms the
   words it asks for, gives the name colleagues see, and names this machine `wonka-test`.

4. The person, on the same page, uses "Add a cloud environment", names it `cloud-test`, and copies
   the value with the page's button. This machine keeps only the cloud key's certificate, so the
   value exists only on the page and in what the person pastes.

5. Create a room on the test daemon, as a session would:

   ```
   env $T_ENV CLAUDE_CODE_SESSION_ID=wonka-test-1 /tmp/cogmer-test create
   env $T_ENV CLAUDE_CODE_SESSION_ID=wonka-test-1 /tmp/cogmer-test rooms
   ```

   Note the room's name for part 2.

6. The person pastes the value into the cloud environment's settings as the variable
   `COGMER_CLOUD_KEY` (the cloud environment menu in a session's title bar, then Edit), and
   starts a new cloud session on this repository. A variable reaches only sessions started after
   it is saved.

## Part 2: in the new cloud session

The plugin in a cloud session runs the released program, which knows no cloud key, so this part
drives a daemon built from the branch by hand, as the container tests do.

1. Check out `user-identity`, build it with `go build -tags ts_debug_websockets -o
   /tmp/cogmer-cloud ./cmd/cogmer`, and confirm `COGMER_CLOUD_KEY` is set without printing it
   (`test -n "$COGMER_CLOUD_KEY" && echo set`).

2. Start the daemon with a throwaway state directory and ports that the plugin's own daemon is
   not using:

   ```
   export C_ENV="COGMER_HOME=/tmp/cogmer-cloud-home COGMER_ADDR=127.0.0.1:4792 COGMER_PEER_ADDR=127.0.0.1:4793 COGMER_HOST=claude COGMER_PREFLIGHT=off"
   env $C_ENV /tmp/cogmer-cloud daemon > /tmp/cogmer-cloud.log 2>&1 &
   ```

   `CLAUDE_CODE_REMOTE` is `true` there, so the daemon carries its relay connection as a WebSocket
   (D-217, a cloud session's daemon carries its relay connection as a WebSocket).

3. Wait for the daemon to fetch the user's lists from `wonka-test`, which its log and the view's
   lists age show (D-245, a cloud session takes its user's lists only from one of the user's
   machines). `env $C_ENV /tmp/cogmer-cloud whoami` names the test user, without printing any key.

4. Join the room by name from part 1, with a session identifier of its own, and send one turn
   through the hooks: run `hook prompt` with a made-up transcript, then `hook stop`, the way
   `m.turn` in `cmd/cogmer/containers_test.go` does.

5. Record what happened: whether the lists arrived and how long it took, whether the join
   succeeded, and the daemon log's lines on the relay and on the fetch.

## What to check on wonka afterwards

The turn from part 2 appears in the test daemon's view at `http://127.0.0.1:4792`, in the room
from part 1, marked as the user's own turn from the cloud environment `cloud-test` (D-232, a
user's own turn from another session is injected as a report marked as the user's). A turn sent
from wonka, with `hook prompt` and `hook stop` under a session in the room, reaches the cloud
daemon's next `hook prompt`.

Then, on wonka, remove `cloud-test` on the page as not stolen, and check that the turn already
shown still counts and that the cloud session's next turn is refused (D-250, a removal of a cloud
key names a cutoff for each of its session keys in each room).

## Cleaning up

On wonka, stop the test daemon with `env $T_ENV /tmp/cogmer-test stop`, and delete `$T_HOME` and
`/tmp/cogmer-test`. Delete `COGMER_CLOUD_KEY` from the cloud environment's settings. The results
go in a commit message under W-91 in `docs/writing.md`, and this file is deleted with its open
item.
