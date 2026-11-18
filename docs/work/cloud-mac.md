# A rented Mac for the host probes

How to run probes 2 and 3 of `docs/work/host-probes.md` on a Mac rented in the cloud, with the
apps driven by scripts rather than by a person at the screen. Material for the same open items as
that runbook. Delete it with them.

The probes need Claude Desktop and the ChatGPT app, and the user travels without a Mac. A Mac
mini M4 rented from Scaleway is bare metal, so Cowork can start its virtual machine there. Scaleway
rents it for 24 hours at least, for licensing reasons, and can delete it automatically when the 24
hours end (https://www.scaleway.com/en/docs/apple-silicon/faq.md, found on 2026-11-18). It is
reached over SSH and over VNC, on a port the console shows, which is not always 5900
(https://scaleway.com/en/docs/apple-silicon/troubleshooting/cant-connect-using-vnc, found on
2026-11-18).

## What a script can drive

Claude Desktop cannot be driven through Chrome's debugging protocol. Claude Desktop 5.11492.5 for
Linux, read on the droplet on 2026-11-18 without being installed, prints that it is "refusing to start" because "a debugging or network-override switch is present
on the command line", and exits, when it is started with `--remote-debugging-port`, `--remote-debugging-pipe` or one of 14 network-override
switches. It starts with them only when `CLAUDE_CDP_AUTH` holds a token signed by a key built into
the app, tied to the app's data folder and younger than 5 minutes. Its Electron fuses also turn
off `--inspect` and refuse a changed app package. The Mac in probe 1 ran the same version, and
whether its build has the same check is not confirmed.

Two routes remain, tried in this order:

1. macOS accessibility scripting. A script presses buttons and reads text by their accessible
   names, through System Events or the accessibility API. The ChatGPT app for Mac is native, and
   Claude Desktop and the Codex app are Electron apps, whose page content Chromium exposes to
   accessibility clients. How much each app labels is not known.
2. Claude computer use. A loop sends a screenshot taken with `screencapture` to the API's
   computer-use tool and performs the clicks it returns. It reaches any screen and costs API
   tokens for each step. Its screenshots also serve as the screen text that a probe's record
   asks for.

Both routes need the Accessibility permission, and computer use also needs Screen Recording.
macOS lets no script grant either, so a person grants them once in System Settings over VNC. Which
program needs them depends on what starts the script: a script started over SSH runs under a
different program than one started in Terminal. The first session finds out which.

## First session, at the screen

This is the one session that needs VNC from a phone or laptop.

1. In the Scaleway console, create the Mac mini M4 with automatic deletion after 24 hours, and
   add an SSH key. Record the macOS version.
2. Over SSH, install Xcode's command-line tools, Go, Node, the Codex CLI and Claude Code, clone the
   repository and check out `host-probes`.
3. Over VNC, install Claude Desktop and the ChatGPT app, and sign in to each. Record the versions.
4. Grant Accessibility, and Screen Recording if computer use will be needed, to the program found
   in the previous section.
5. Open Accessibility Inspector, from Xcode's developer tools, on each app, and record whether the
   message box, the send button, the model's reply and the plugin settings have accessible names.
   That decides how much of each probe route 1 can drive.
6. Start Claude Code in the checkout with Remote Control, so that the rest runs from the phone.

## Each probe after that

Claude Code on the Mac runs `probe.py` and drives the app, and the user watches and steers through
Remote Control. A step the script cannot do, such as a trust prompt with no accessible name, falls to
computer use, and a step that neither can do is done over VNC and recorded as such in the run's
record.

The probe's record names the macOS version, the app versions and, for each step, whether a script,
computer use or a person did it. Screenshots go with the record in `docs/work/`.

Each probe ends with `probe.py report` and `probe.py after`, as `docs/work/host-probes.md` says.
When the last probe is done, sign out of both apps, so that no signed-in session of the user's
accounts is left on a machine somebody else will rent. Revoke the SSH key added for the rental, and
delete the rental in the console if it was not set to delete itself.
