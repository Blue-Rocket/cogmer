# Cowork first-run spikes

Six spikes that answer what stands between a Cowork-only user and starting work by installing the
plugin alone. Material for the open item "A Cowork-only user cannot start by installing the plugin" in
`docs/open.md`. Delete it with that item.

Written on 2026-11-26 for a Claude Code session on the user's Mac that drives claude.ai in Chrome with
the Claude in Chrome extension. Claude Desktop cannot be driven that way (`docs/work/cloud-mac.md`), so
every Cowork task here is started in claude.ai in Chrome. Cowork tasks started there run in the cloud
(`docs/work/cowork-account-install-20261120T1305Z.txt`).

## What a Cowork-only user does today, and what each spike could remove

| step today | spike that could remove or shrink it |
|---|---|
| install the cogmer plugin on the Claude account (D-290, cogmer reaches a Cowork cloud session as the marketplace plugin installed on the user's Claude account) | none: this is the one step meant to stay |
| on a Team plan, an owner allows the relay's five hosts (`ce962eb`) | C: what a task reaches by default, on which plans, and whether a wildcard entry is honored |
| a machine to hold the identity, which for a user with no computer is a hosted machine (D-285, not built) | E: whether a passkey can hold the master key in the browsers such a user has |
| make the cloud value and upload it as a plugin zip (D-246, D-278) | A and B: whether anything but an upload puts a per-user value in a task |
| verify each colleague with two words, both at once (D-054) | none here: a decision about verifying by an invitation link |
| join rooms with commands | none here |
| the first task says nothing when nothing is set up (D-286) | D: what a person sees of a hook's message in a cloud task |

## Rules for the session that runs them

- Ask the user before anything that changes the account or the organization: uploading, installing or
  removing a plugin, adding a marketplace, and changing a network setting. Say what will change and how it
  is undone, and wait for a yes.
- Type only the markers below into a plugin's options or a prompt. Never type, paste or print a cloud
  value, a recovery phrase or any credential, and never read a value of the user's real `cogmer-cloud-`
  plugin.
- Do not solve a CAPTCHA. If one appears, stop and ask the user.
- A passkey prompt needs the user's Touch ID or phone. Ask the user to answer it.
- Record each run in a file `docs/work/first-run-spike-<letter>-<UTC time>.txt` on branch `host-probes`:
  when it ran, the account's plan, the browser and its version, what was done in enough detail to repeat
  it, what was seen, quoted where possible, and what the run does not show (W-91 in `docs/writing.md`).
  Commit each with the full suite passing, as `CLAUDE.md` says.
- Leave the account as it was found: remove the probe plugin and undo any setting changed, once the user
  agrees.

The markers are `frs-secret-1` for the sensitive option and `frs-plain-1` for the plain one, raised to
`-2` and so on when an option is entered again.

## The probe plugin

`docs/work/first-run-spike/` is a plugin named `first-run-spike`, version 0.1.0. It holds no secret,
sends nothing of the user's anywhere, and records the names of environment variables but never their
values.

- Its `SessionStart` and `UserPromptSubmit` hooks answer at once with a `systemMessage` beginning
  `FRS-SYS-<event>` and an `additionalContext` beginning `FRS-CTX-<event>`, each with the link
  `https://example.com/first-run-spike`.
- At `SessionStart` it also writes `/tmp/first-run-spike.txt` in the background: whether each `userConfig`
  option arrived, with its length and first four characters; the variable names; the marker an earlier
  task left in each place a file might persist, before writing a new one; the synced plugin folders by
  name; the mount points; and, for thirteen hosts, curl's exit code and HTTP status with the certificate's
  subject, issuer and SHA-256 fingerprint, and for the probe site's three names what the site saw of the
  request.
- `/first-run-spike:report` has the model print that file whole.

Make its zip on the Mac from the repository root:

```sh
(cd docs/work/first-run-spike && zip -r ../first-run-spike.zip . -x '.DS_Store')
claude plugin validate docs/work/first-run-spike
```

For the certificates the Mac sees, the baseline for spike C, run the hook's record on the Mac. It runs
only its network part usefully there and writes the same file:

```sh
HOME=$(mktemp -d) sh docs/work/first-run-spike/hooks/spike.sh --run; cat /tmp/first-run-spike.txt
```

## The probe site

`https://probe.cogmer.ai`, `https://a.probe.cogmer.ai` and `https://b.probe.cogmer.ai` are served by
Caddy 2.8.4 on the droplet at 164.92.115.202, with certificates from Let's Encrypt, from
`docs/work/probe-site/Caddyfile`. The names are A records at Porkbun added on 2026-11-26, and the rest of
cogmer.ai is the registrar's parking page.

- `/echo` answers with what the site saw of the request: the name asked for, the name in the TLS
  handshake, the TLS version and protocol, the address it came from, the user agent, and the `Via`,
  `X-Forwarded-For` and `Forwarded` headers. A request from the droplet itself shows its own address and
  no proxy header.
- `/prf/` is the passkey page of spike E. It creates a passkey for its own name and shows whether a PRF
  secret came back, its length and the first eight bytes of its SHA-256 hash, and sends nothing anywhere.
- The site keeps no access log. Caddy runs as the account `claude` from `~/probe-site`, detached, and
  stops if the droplet restarts. To start it again there:

```sh
cd ~/probe-site && PROBE_SITE=$PWD/site setsid nohup ./caddy run --config Caddyfile --adapter caddyfile >> caddy.log 2>&1 < /dev/null &
```

To take it down when the item is done: stop Caddy, delete the two A records at Porkbun, and run
`sudo setcap -r /home/claude/probe-site/caddy`.

## A. Which routes put a plugin on an account

The question: can anything other than the user uploading a zip put a plugin with a per-user file on the
user's Claude account, ideally from one link with one confirmation? If yes, the cloud value needs no
upload.

1. Read and record, with each URL and the date read:
   - https://code.claude.com/docs/en/plugins/install.md, especially "Add from claude.ai";
   - https://claude.com/docs/plugins/org-sync, on distributing plugins through organization settings;
   - the claude.ai help center articles on plugins in Cowork and in claude.ai;
   - any Anthropic page on a link, button or API that adds a marketplace or installs a plugin.
   For each route, record who can use it (a user, an owner, an admin), whether it can carry a file that
   differs between users, and whether the docs say it reaches a Cowork cloud task.
2. In Chrome, open claude.ai's plugin settings and record the address of each page and dialog on the way
   to adding a marketplace and installing a plugin from it. Then open each address in a new tab and record
   whether it opens the same dialog. An address that opens an install dialog is a candidate for the link
   a cogmer page would give. Change nothing on the account in this step.
3. If the account is an organization's, record what the organization settings offer for plugins, with
   the user present, without changing anything.

A route counts only if it can carry a value that differs between users. An organization-wide plugin
carries the same files to everyone, so it could carry cogmer but not one user's cloud value.

## B. Whether a plugin's options reach a cloud task

The question: when a plugin declares `userConfig` options, does claude.ai or Cowork ask the user for them
at install, and does a cloud task's hook receive them as `CLAUDE_PLUGIN_OPTION_<KEY>`? If yes, the cogmer
plugin could ask for the cloud value at install, which folds the upload into installing the plugin.

The plugins reference says Claude Code prompts for `userConfig` when a plugin is enabled, stores a
sensitive value in the platform's secure store, and exports every option to hook processes as
`CLAUDE_PLUGIN_OPTION_<KEY>` (https://code.claude.com/docs/en/plugins-reference, read on 2026-11-26).
It says nothing about claude.ai, Cowork or a cloud task.

1. With the user's agreement, upload `first-run-spike.zip` as a personal plugin. Record whether a dialog
   asks for either option, its wording, and whether the sensitive one is masked. If one asks, enter the
   markers.
2. Start a new Cowork task in Chrome and send `hello`. Then run `/first-run-spike:report` and record the
   section "B. userConfig options".
3. If no dialog asked, look in the plugin's settings on claude.ai for a place to set the options, set them
   if there is one, start another task and record again.
4. Record whether an uploaded plugin behaves differently here from a marketplace plugin. Testing a
   marketplace plugin needs a marketplace holding the probe, which means publishing a repository. Ask the
   user before creating one, and record the question as open if they decline.

## C. What a cloud task's hooks reach and keep

The questions: which hosts a task reaches with no allowlist change, on each plan the user has; whether a
wildcard entry such as `*.probe.cogmer.ai` is honored; whether the proxy passes TLS through or presents
its own certificate, and what a server of cogmer's sees of a task's request; and what, if anything,
persists from one task to the next where a hook can read it.

Earlier runs on the Team plan found that the organization's "Allow network egress" setting with "Package
managers only" refused the relay and `example.com` with "CONNECT tunnel failed, response 403", and that
adding the five relay hosts let them through, while `example.com` was still refused. In all three runs
`github.com` answered 400 with a body that links to Anthropic's documentation, which reads as the proxy
answering the request itself, and a network setting applies when a task is created (`docs/work/cowork-
network-probe-20261031T053000Z.txt`). A task began with an empty synced-plugins folder before the account
install (`docs/work/cowork-account-install-20261120T1305Z.txt`). Whether a Pro or Max plan has such a
setting is not known.

1. With the probe plugin installed, start a Cowork task, send `hello`, run `/first-run-spike:report`, and
   record sections C1 to C3 and the plan. A network setting applies when a task is created, so start a
   new task after each change.
2. Compare each certificate's issuer and fingerprint with the Mac's baseline, taken the same day, since
   a certificate is renewed every few weeks. An issuer that is the proxy's, or a fingerprint that
   differs, means the proxy terminates TLS. For the probe site's names the report also holds the site's
   `/echo`: the address it came from is where the task's traffic leaves Anthropic's network, a `Via` or
   forwarding header names a proxy, and a site that answered at all was reached as a server a hosted
   machine could be.
3. Persistence: start a second, new task and run the report again. A place whose marker the second task
   found persists between tasks. Then return to the first task after at least ten minutes, send a prompt,
   and record what a resumed task found.
4. Wildcards, on the Team plan, with an owner present and agreeing: add one allowlist entry
   `*.probe.cogmer.ai`, start a new task, and record whether `a.probe.cogmer.ai` and `b.probe.cogmer.ai`
   are reached and whether `probe.cogmer.ai` is. Then replace the entry with `probe.cogmer.ai` alone and
   record the same. Remove the entry afterwards and record that it was removed.
5. If the user has a Pro or Max account as well, repeat steps 1 to 3 there, and record what its settings
   show about network access.

## D. What a person sees of a hook's message in a cloud task

The question: can the first task of a user with nothing set up show them one line with a link to start
setup? D-286 (a Cowork cloud session with no cloud value fetches and starts nothing) has that task print
nothing today. In Claude Code on a computer, a `systemMessage` is shown to the person and never given to
the model (`11b1294`, cited in `CLAUDE.md`).

1. In the task of spike B or C, before running the report, record, with a screenshot, what the page shows
   after the task starts and after `hello`: whether `FRS-SYS-SessionStart` or `FRS-SYS-UserPromptSubmit`
   appears, where, in what style, and whether its link can be clicked.
2. Ask the model "What did the first-run spike say?" and record whether it quotes `FRS-CTX-` lines and the
   link.
3. Do the same from the Claude app on the user's phone, if they will start a task there, since a
   Cowork-only user may start from one.

## E. Whether a passkey can hold the master key

The question: do the browsers a Cowork-only user has return a passkey's PRF secret, the same each time,
from which a page could derive the master key? The open item on holding a hosted user's master key lists
this option and its costs.

1. Open `https://probe.cogmer.ai/prf/`, the probe site's page, which runs entirely in the browser.
2. On the Mac, in Chrome and in Safari: create a passkey on that page, then sign in with it twice. Record whether the create step reported PRF enabled, whether each sign-in returned a
   secret, and whether the two hashes were the same. Record the length and the hash, never more.
3. If the passkey syncs, through iCloud Keychain or Google Password Manager, sign in with it on the
   user's phone, and record whether the secret there matches.
4. Record each browser's name and version, which the page prints first, and the phone's system version.
5. Ask the user to delete the passkeys named "PRF probe" for `probe.cogmer.ai` from the Mac and the phone
   afterwards.

## F. Whether a page on the user's computer can use a passkey through a page under cogmer.ai

The question: can the daemon's page at `127.0.0.1` send the user to a signing page under `cogmer.ai`, and
get back the passkey's signature over a challenge it chose, by navigation alone, with no hosts-file entry,
no local certificate and no request from one page to the other; and does a passkey made for the site
`cogmer.ai` on one name under it sign on another. A passkey belongs to one site, a site is a domain name and
never an address, and a page may name a parent of its own name as the site, so a page at `127.0.0.1` can use
none itself (the open item on holding a hosted user's master key).

The probe site's `/sign/` page (`docs/work/probe-site/sign/index.html`), served on all three names, makes a
passkey whose site is `cogmer.ai` and signs a challenge given in its address after `#`, only when its button
is pressed, and returns only to an address on `127.0.0.1` or `localhost`. The page
`docs/work/probe-site/local/index.html` stands in for the daemon's page: it sends the user to `/sign/`, keeps
the public key the registration returns, and checks a signature that comes back as a daemon would, printing
one line for each check.

1. On the Mac, from the repository root:

   ```sh
   python3 -m http.server 4799 --bind 127.0.0.1 --directory docs/work/probe-site/local
   ```

2. In Chrome, open `http://127.0.0.1:4799/`, press "Register a passkey on probe.cogmer.ai", then "Register
   a passkey" on the page it opens, and have the user answer the prompt. Record where the browser offered to
   save the passkey, the line the stand-in prints when it comes back, and any prompt or warning about the
   local network or about leaving or returning to the page.
3. Press "Sign on probe.cogmer.ai", then "Sign with my passkey", and record every line the stand-in prints.
   Then do the same with "Sign on a.probe.cogmer.ai". Every check is meant to read `true`, and the last line,
   the signature over a changed challenge, `false`.
4. Do steps 2 and 3 in Safari, which keeps its own store for the stand-in, so it registers again.
5. If a passkey was saved to iCloud Keychain, open `https://a.probe.cogmer.ai/sign/#mode=self` on the user's
   iPhone, press the button, and record whether the passkey made on the Mac answered there.
6. Stop the server, and ask the user to delete the passkeys named "cogmer signing probe" for `cogmer.ai`.

## What to bring back

For each spike, a recorded run and a sentence of what it found. Then the open item is rewritten with what
the spikes found and the routes that remain, and the decisions the chosen route needs are written:
hosted machines (D-285), verification by an invitation link against D-054 (verification gates
synchronization and injection), and a first-run line against D-286.
