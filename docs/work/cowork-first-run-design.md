# Cowork first-run design

A route by which a Cowork-only user starts work by installing the plugin, with three draft decisions and
the questions they leave. Material for the open item "A Cowork-only user cannot start by installing the
plugin" in `docs/open.md`. The drafts become numbered decisions in `docs/decisions/` once they are agreed,
and this page is deleted with the item.

## What the spikes found

The spikes of `docs/work/cowork-first-run-spikes.md` ran on 2026-11-26 on the Team plan, from Chrome on the
user's Mac and from the user's iPhone.

- No route but the user's own upload puts a file that differs between users on a Claude account. The
  documents describe a marketplace, Anthropic's catalog, Share, Publish to org, organization marketplaces
  and the Plugins API, each giving every recipient the same files, and no link that installs anything
  (`docs/work/first-run-spike-A-20261126T1558Z.txt`).
- A plugin's `userConfig` options are asked for nowhere when it is uploaded, and both reached a cloud task
  unset (`docs/work/first-run-spike-B-20261126T1533Z.txt`).
- With the organization's egress mode "Package managers only", every host of cogmer's was refused until an
  owner listed it. An entry `*.probe.cogmer.ai` reached `a.probe.cogmer.ai` and not `probe.cogmer.ai`, and
  an entry `probe.cogmer.ai` reached only that name. TLS went through unchanged to every host the list let
  through, and only `github.com` and `api.anthropic.com` met Anthropic's interception. The probe site saw
  each request from an address in 160.79.106.0/24, a different one between requests, with no forwarding
  header. Nothing a task wrote reached the next new task; a resumed task kept its files
  (`docs/work/first-run-spike-C-20261126T1533Z.txt`).
- A hook's `systemMessage` and `additionalContext` reached the model and showed nowhere on the page, on the
  Mac or the phone, and the `SessionStart` lines reached the model from the second message on
  (`docs/work/first-run-spike-D-20261126T1533Z.txt`).
- Safari on the user's iPhone, with the passkey in iCloud Keychain and answered with Face ID, returned the
  same 32-byte PRF secret at the passkey's creation and at two sign-ins after it
  (`docs/work/first-run-spike-E-20261126T1650Z.txt`).
- A page at `127.0.0.1` sent the user to a page under `cogmer.ai` and got back, by navigation alone, a
  passkey's signature over a challenge it chose, which it verified against the key it kept at
  registration, in Chrome 152 and Safari 26.6.2 on the Mac, with no hosts-file entry or local certificate.
  A passkey for the site `cogmer.ai` made on one name under it signed on another, and both passkeys made
  synced through iCloud Keychain and signed on the user's iPhone, whose chooser showed two passkeys for
  `cogmer.ai` with nothing to tell them apart (`docs/work/first-run-spike-F-20261126T194604Z.txt`).

So a cloud task can be given a secret of its user's only by the user uploading it, a hook cannot show the
person a line, a passkey's secret is available to a page in the user's browser, and a page on the user's
computer can have a passkey sign for it through a page under `cogmer.ai`.

## The route

A user whose first session after installing is a cloud session needs a machine of their own to hold
their identity, serve their lists to each task (D-245, a cloud session takes its user's lists only from one
of the user's machines) and keep the turns their tasks write, since a task keeps nothing past its sandbox
(spike C). cogmer provides hosted machines (D-285, a user with no computer of their own has a machine
hosted for them), and onboarding suggests one to such a user beside the choice of a computer of their own.
Whether the hosted machine is the default is not decided.

1. The user installs the cogmer plugin on their account (D-290, cogmer reaches a Cowork cloud session as
   the marketplace plugin installed on the user's Claude account).
2. In a task, the user runs `/cogmer:start`. The task's daemon starts with no secret of the user's, makes
   its session key, and the command prints the link to the cloud page (D-302, a cloud session's room is
   read on a static page) with a token made for this task.
3. The first time, the page asks where the user's identity will live, suggesting a hosted machine:
   - A hosted machine: the page registers the user's passkey, and the hosted machine is made with the
     user's identity and certifies the passkey, in one step.
   - A computer of their own: the page gives one line to paste into Claude Code or Codex on the computer,
     which installs cogmer there and opens its machines page, where the identity is made and the passkey
     registered through the page under `cogmer.ai` (spike F).
4. The user answers the passkey prompt, and the page certifies the task's session key by an assertion of
   the passkey (Draft 1) and sends the task the certificate and what the task needs to reach the user's
   machines. A user at their own computer can instead approve the task on its machines page, which
   certifies it with no passkey. The user's machines renew the certificate from then on.
5. The task speaks for the user in their rooms, as a task with an uploaded value does today. A new task
   needs steps 2 and 4 again, since nothing persists between tasks.

On a Team plan an owner still allows the hosts the task's daemon reaches. A user who would rather have
tasks start with no tap keeps the uploaded value (D-278, the machines page offers the cloud value as a
plugin for the user to upload once).

## Draft 1: a cloud task's session key is certified by the user's passkey and renewed by the user's machines

Decision. A cloud task holds no secret of its user's when it starts. Its daemon makes a session key, and
the cloud page, opened from the link the task prints, has the user's passkey sign the session key and an
expiry three days later as the challenge of an ordinary WebAuthn assertion, which is the task's first
certificate. The passkey's public key is certified once, when the passkey is registered, by a machine that
holds the user's shared key, as a cloud key is certified today (D-246, a cloud environment is added from the
machines page; D-216, a cloud session signs with a key of its own that the user's shared key vouches for).
A receiver checks the assertion's signature with that key, that its site is `cogmer.ai`, that the
authenticator confirmed the user with Face ID, a fingerprint or a PIN, and that the page it came from is
the cloud page, before it accepts the session key. From then on the task renews its certificate every two
to four hours from whichever of the user's durable machines it reaches, hosted or their own computer, each
of which holds the shared key and certifies the session key for three days more. A task that reaches none
before its certificate ends stops publishing, keeps capturing its turns, and says that sharing is paused
until one of the user's machines answers, and when one does, or the user taps the page's link again, it
signs the held turns and sends them. A script on the page that turned hostile gets, at each prompt the
user answers, one certificate for a key it chose, and never a key it can use again or the user's master
key; the cloud page passes the identifier of the user's registered passkey, so the user is not asked to
choose among passkeys for `cogmer.ai`, which a chooser showed with nothing to tell apart (spike F).

Three days lets a task survive its user's machines all being away for a weekend, and caps how long a
session key copied out of a sandbox speaks for the user, unless a removal of it reaches colleagues first
(D-250, a removal of a cloud key names a cutoff for each of its rooms). It departs from D-247 (a cloud
session's key is certified for twenty-four hours, and the session's daemon certifies a new one before it
expires), whose renewal is done by a cloud key in the task, since the task holds no cloud key here.

Rejected.
- *Deriving a cloud key from the passkey's PRF secret in the page.* A script that keeps the derived key
  certifies for as long as the key is certified, where an assertion certifies one key until it expires, and
  carrying a PRF secret between pages puts it in an address.
- *Asking for the cloud value as a plugin option at install.* Nothing asked for an option, and options
  reached a task unset (spike B).
- *A link that installs the user's value plugin.* No such link exists (spike A).
- *A certificate of a day, renewed by the passkey.* The user would answer a prompt every day for every task
  that runs that long, or the task would pause.

Limits. A passkey signs with P-256 and the user's records are Ed25519, so a certificate in this form is a
new version of the records that every receiver has to read before any task uses it (D-058, signature
schemes are kept, never replaced). D-058 leaves the scheme's version outside the signature, which it says
is harmless because every scheme is Ed25519 and none is weaker to be downgraded to, so a P-256 scheme
beside them needs that reason restated. The prompt names the site and the account and never what is
signed, and the page chooses the challenge, so the user cannot see which key they certify, as the open item
on holding a hosted user's master key says of a passkey that signs a record's hash. A hostile script that
the user answers can therefore certify a key of its own for one certificate at a time. A task that ends
while paused loses the turns it held. A passkey that is lost takes the certified key with it, and the user
registers a new one from a machine that holds the shared key.

Revisit when a passkey's prompt can show the user what it signs, or users find tasks paused often enough
that three days is too short.

## Draft 2: a cloud task with nothing set up shares nothing until the user runs `/cogmer:start`

Decision. A Cowork cloud task with no cloud value starts nothing at its start (D-286, a Cowork cloud
session with no cloud value fetches and starts nothing), and `/cogmer:start` starts its daemon, which
serves the page's port with a token made at random for the task, and prints the cloud page's link. A hook's
messages reach the model and never the person (spike D), so a line that tells the user where to start can
reach them only through the model, and a command the user types is a turn the user asked for. D-286's
Limits already let a command start a daemon in such a task.

Rejected.
- *A hook that asks the model to tell the user how to start.* The model said the link when asked (spike
  D), but the line would be in the model's context in every task of every user who had not set up, and
  would make the model raise cogmer in a conversation the user started for something else, which §3.1
  (first, do no harm) rules out.

Limits. A user who does not know the command shares nothing, so the plugin's description and the
README carry it.

Revisit when a host shows a hook's message to the person in a cloud task.

## Draft 3: the cloud page is served from a name under cogmer.ai

Decision. The cloud page is served from a name under `cogmer.ai`, such as `view.cogmer.ai`, rather
than from GitHub Pages, so that the passkey it uses belongs to `cogmer.ai` and serves the hosted machine's
page and the self-hosted page under that domain, as the open item on holding a hosted user's master key
proposes. A passkey belongs to one site for life and a page may name any parent of its own name as the
site, so one passkey under `cogmer.ai` serves every page under it, and one made for
`blue-rocket.github.io` would serve that site only.

Rejected.
- *Keeping the page on GitHub Pages.* Its passkeys would belong to `blue-rocket.github.io`, which
  `Blue-Rocket` controls on GitHub, and could never be used by a page under `cogmer.ai`.

Limits. cogmer runs the host, and its trust is the trust D-302's Limits give the page's host.

Revisit when cogmer gives up the domain.

## Questions the route leaves

- Where the task learns how to reach its user's machines. A task takes its user's lists only from one of
  the user's machines (D-245, a cloud session takes its user's lists only from one of the user's machines),
  and today the cloud value carries their relay addresses. A machine that approves or renews a task gives
  it the addresses then, so the question is the first certificate, which the page makes:
  - At the user's own computer, the cloud page sends the user to the machines page at `127.0.0.1` with the
    task's address and token after `#`, and the machine dials the task and gives it a certificate, the
    lists and the addresses, so the page carries nothing back.
  - With a hosted machine, the passkey's user handle, up to 64 bytes set at registration and returned at
    every sign-in on every device, names the hosted machine, whose name under `cogmer.ai` the page reaches
    from any device.
  - A user whose only machine is a computer, away from it, gets the first certificate from the passkey and
    the lists and a renewal when the computer answers. A machine's overlay address is about 186
    characters (`docs/work/cloud-run-20261025T171545Z.txt`), more than a user handle holds.
  The passkey's `largeBlob` extension, which stores up to a few kilobytes with a passkey, could carry the
  computer's addresses, and is not designed around. On 2026-11-26 the sources read said that iCloud
  Keychain and Google Password Manager on desktop Chrome store a blob, that Windows Hello, Samsung Pass and
  eight third-party password managers declined it at registration, that Microsoft Password Manager failed
  the write, and that Chrome on Android does not pass it on
  (https://www.corbado.com/blog/webauthn-l3-credential-manager-support, tested in August 2026). None says
  whether a blob syncs with the passkey to the user's other devices, which is the case it would serve.
- What the identity of a user with no computer rests on. It rests on a hosted machine, which onboarding
  suggests (D-285, a user with no computer of their own has a machine hosted for them): the machine holds
  the shared key, certifies the passkey and renews the user's tasks. Deriving the master and shared keys
  from the passkey is set aside, since the passkey signs and carries no secret.
- How the page reaches a task whose daemon holds no key yet. Today the daemon serves the page on a second
  port of the overlay, which admits by a token made from the cloud key (D-301, a cloud session's daemon
  serves its room on a second overlay port), and Draft 2 makes the token random for the task. The page's
  link carries it, and the link is in the conversation, as D-301's Limits say of today's.
- How two Cowork-only users verify each other. A live two-word comparison needs both daemons running at
  once (D-048, verification is a live commit-then-reveal exchange that derives two words), and such a
  user's daemon runs only while a task does. An invitation link sent over a channel both people trust
  could verify the inviter's key to the invitee and prove the invitee to the inviter by a one-time value it
  carries. That is a second way to verify, which D-055 (a peer is verified in one way, the two-word
  comparison) rejects because "a fallback is the path people reach for when the other is inconvenient", and
  its revisit condition, two users who must verify and cannot have their daemons reach each other, is the
  case to show first. Three ways remain, each with what it trusts:
  - The two words, with a hosted machine (D-285, a user with no computer of their own has a machine hosted for them) answering for a user who has no computer, so the live
    exchange runs whenever the two people are on a call. Against anyone else it is as strong as today's,
    and against the hosted machine's operator it is no stronger than trusting the operator, who runs the
    machine that shows the user the words.
  - A password-authenticated key exchange, such as SPAKE2 or CPace, in place of the two words: one person
    reads a short code to the other or sends it once, and the exchange shows that both hold the same code,
    allowing one guess for each attempt. It trusts whatever carried the code, and both sides take part at
    once, so a user with no computer still needs a hosted machine. Replacing the two words with it keeps one
    way to verify, as D-055 has it, and changes which way that is, so D-055 and D-048 (verification is a
    live commit-then-reveal exchange that derives two words) would both be rewritten.
  - The invitation link above, which trusts the channel that carried it and everyone it was passed to.
- Which hosts a Team owner allows. Two entries, `cogmer.ai` and `*.cogmer.ai`, would cover everything
  under the domain (spike C), if the task reached cogmer's own relay or hosted machines rather than
  tailcat's relays. A Pro or Max plan was not tried.
- Whether the Mac's browsers, and the same passkey reached through iCloud Keychain, return the same secret.
  Only Safari on an iPhone was tried.
- How Draft 2's token answers D-301 (a cloud session's daemon serves its room on a second overlay port).
  D-301 has a daemon with no cloud key serve nothing, and rejects a token made fresh at each start, since
  a sleeping task wakes often and each wake would break the link, and a token stored for each task, since
  it needs a file and a rule for when it changes. A token derived from the task's session key, as today's
  is from the cloud key, would last across wakes with no file of its own, since a resumed task kept its
  files (spike C). Draft 2 then has to say it departs from D-301 in serving a daemon that holds no cloud
  key.
- Whether Draft 2 is an exception to D-281 (a command in a cloud session with no value says the user must
  first set cogmer up on one of their own computers). In a Cowork cloud task, `/cogmer:start` would start
  the daemon and print the page's link instead, and D-286's Limits, which Draft 2 relies on, say the
  command says what D-281 has it say. A Claude Code cloud session could keep D-281's answer.
- Whether a passkey that certifies tasks counts under D-280 (a user has one cloud key at a time). The
  passkey is not a cloud key but a key of another kind that certifies a task's session key, and a user who
  keeps the uploaded value keeps a cloud key beside it. Either page can run the passkey step: the machines
  page stays at `127.0.0.1` and sends the user to a page under `cogmer.ai` with its challenge, taking back
  the signature by navigation alone, with no hosts-file entry or local certificate (spike F).
- Whether a new task needs a tap. A new task starts with nothing, so it needs the user's passkey or a
  secret stored where tasks read it, which only an upload puts there (spikes A to C). The upload could hold
  a token that only the user's own machines honor, in place of today's cloud key, which every colleague
  honors: a new task presents it to one of the user's machines and gets a three-day certificate, and the
  machine can show that a task started, limit how often the token is used, and revoke it at once, where a
  cloud key that is copied speaks for the user until a removal reaches each colleague (D-225, a machine
  sends each removal and new shared key to every colleague it knows until each acknowledges).
- Which earlier decisions Drafts 1 and 3 change in part. D-216 (a cloud session signs with a key of its own
  that the user's shared key vouches for) names the cloud key as the one secret given to the environment,
  and D-302 (a cloud session's room is read on a static page) serves the page from GitHub Pages, so each
  is rewritten to what still holds when the drafts are adopted, as W-38 in `docs/writing.md` has it.
