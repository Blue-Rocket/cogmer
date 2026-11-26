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

So a cloud task can be given a secret of its user's only by the user uploading it, a hook cannot show the
person a line, and a passkey's secret is available to a page in the user's browser.

## The route

1. The user installs the cogmer plugin on their account (D-290, cogmer reaches a Cowork cloud session as
   the marketplace plugin installed on the user's Claude account).
2. In a task, the user runs `/cogmer:start`. The task's daemon starts with no secret of the user's, makes
   its session key, and the command prints the link to the cloud page (D-302, a cloud session's room is
   read on a static page) with a token made for this task.
3. The user opens the link and answers the passkey prompt. The page derives the user's cloud key from the
   passkey's PRF secret, certifies the task's session key with it, and sends the task the certificate, the
   cloud key's own chain and what the task needs to reach the user's machines.
4. The task speaks for the user in their rooms, as a task with an uploaded value does today. A new task
   needs steps 2 and 3 again, since nothing persists between tasks.

On a Team plan an owner still allows the hosts the task's daemon reaches. A user who would rather have
tasks start with no tap keeps the uploaded value (D-278, the machines page offers the cloud value as a
plugin for the user to upload once).

## Draft 1: a cloud task's session key is certified in the cloud page, from a passkey

Decision. A cloud task holds no secret of its user's when it starts. Its daemon makes a session key,
and the cloud page, opened from the link the task prints, asks the user's passkey for its PRF secret,
derives the user's cloud key from it, and sends the task a certificate for the session key that expires
in 24 hours (D-247, a cloud session's key is certified for twenty-four hours), with the cloud key's chain.
The cloud key is certified once, when the passkey is registered, by a machine that holds the user's
shared key, as a cloud key is certified today (D-246, a cloud environment is added from the machines
page; D-216, a cloud session signs with a key of its own that the user's shared key vouches for). Nothing
but an upload puts a per-user value in a task (spikes A and B), and a passkey's secret was the same at its creation and at each sign-in on the one phone tried (spike E), so the page can make the same cloud key every time without anything storing it.
The cloud key never enters the task, so the model's shell can copy only a session key that expires within
a day, where today it can copy the cloud key (D-216's Limits).

Rejected.
- *Asking for the cloud value as a plugin option at install.* Nothing asked for an option, and options
  reached a task unset (spike B).
- *A link that installs the user's value plugin.* No such link exists (spike A).
- *The page sending the task the cloud key.* The task's shell could then copy a key that speaks for the
  user until it is removed, as it can today.

Limits. The host that serves the page serves the script that holds the cloud key while it signs, so
that host has to be trusted, as D-302's Limits say of the token. A passkey belongs to the site that made it,
so the page's site cannot change without the user making a new passkey. Each new task needs the page
once, so a task that nobody opens shares nothing, and a task that runs past 24 hours needs the page again.

Revisit when a Claude account can hold a per-user value that a cloud task's hook reads, or a browser a
Cowork user has is found to return no PRF secret.

## Draft 1, signing variant: the passkey signs the task's session key

Draft 1 derives a cloud key from the passkey's PRF secret in the page, so the page's script holds a key
that certifies for as long as it is certified. If whoever serves the page serves a script that keeps the
key, it holds a key that speaks for the user in every cloud task until the key is removed, and if the
master key is derived the same way, it holds the user. In this variant the page derives nothing. The
passkey signs the task's session key and expiry as the challenge of an ordinary WebAuthn assertion, with
the private key that never leaves the authenticator, and the task's certificate is that assertion.

Decision. A cloud task's session key is certified by an assertion of the user's passkey whose challenge is
the session key and the certificate's expiry. The passkey's public key is certified once, when the passkey
is registered, by a machine that holds the user's shared key, as a cloud key is certified today (D-246,
D-216), and a receiver checks the assertion's signature with that key, that its site is `cogmer.ai`, that
  the authenticator confirmed the user with Face ID, a fingerprint or a PIN, and that the page it came from is the cloud page, before it accepts the session key.
A script that turns hostile gets, at each prompt the user answers, one assertion over a key it chose, which
speaks for the user until it expires, and never a key it can use again or the user's master key.

Rejected.
- *Draft 1 as written, deriving a cloud key from the PRF secret.* A script that keeps the derived key
  certifies for as long as the key is certified, where an assertion certifies one key until it expires.

Limits. A passkey signs with P-256 and the user's records are Ed25519, so a certificate in this form is a
new version of the records that every receiver has to read before any task uses it (D-058, signature
schemes are kept, never replaced). The prompt names the site and the account and never what is signed,
and the page chooses the challenge, so the user cannot see which key they certify, as the open item on
holding a hosted user's master key says of a passkey that signs a record's hash. A hostile script that the
user answers can therefore certify a key of its own for one expiry at a time. A passkey that is lost takes
the certified key with it, and the user registers a new one from a machine that holds the shared key.

Revisit when a passkey's prompt can show the user what it signs.

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
  the user's machines (D-245), and today the cloud value carries their relay addresses. The page could
  carry them if it can get them: from a hosted machine at a fixed name under `cogmer.ai`, from the
  browser's storage of an earlier setup, which Safari clears after seven days of browser use without a visit to the site (https://webkit.org/blog/10218/full-third-party-cookie-blocking-and-more/, read on 2026-11-26), or as
  public records kept beside the passkey.
- What a Cowork-only user's identity rests on. The cloud key is certified by the shared key, which a
  machine holds. A user with a computer certifies it on their machines page. A user with none needs a
  hosted machine (D-285, a user with no computer of their own has a machine hosted for them), or the page
  derives the master and shared keys from the passkey too, which is the PRF option of the open item on
  holding a hosted user's master key, with its costs.
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
  - The two words, with a hosted machine (D-285) answering for a user who has no computer, so the live
    exchange runs whenever the two people are on a call. Against anyone else it is as strong as today's,
    and against the hosted machine's operator it is no stronger than trusting the operator, who runs the
    machine that shows the user the words.
  - A password-authenticated key exchange, such as SPAKE2 or CPace, in place of the two words: one person
    reads a short code to the other or sends it once, and the exchange shows that both hold the same code,
    allowing one guess for each attempt. It trusts whatever carried the code, and both sides take part at
    once, so a user with no computer still needs a hosted machine. Replacing the two words with it keeps one
    way to verify, as D-055 has it, and D-048 would be rewritten.
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
- How Draft 1 answers D-247 (a cloud session's key is certified for twenty-four hours, and the session's
  daemon certifies a new one before it expires). With no cloud key the daemon cannot renew, and D-247
  rejects a short certificate because a daemon that misses a renewal has its turns refused. A task whose
  certificate has expired could stop sharing and say so in the view and at its next command, or hold its
  turns unsigned until the page is opened again and sign them then. Which, and how long the certificate
  lasts, is to be chosen.
- How an uploaded value and the passkey keep one cloud key (D-280, a user has one cloud key at a time). A
  user who keeps the uploaded value would otherwise hold a second cloud key beside the one the passkey
  derives. Either page that can use the passkey can derive the key and offer the value's plugin, as the machines
  page does today (D-278). The cloud page can, once it is served under `cogmer.ai` (Draft 3). The machines
  page can if it is served at a name under `cogmer.ai` mapped to the loopback address in the hosts file,
  such as `local.cogmer.ai`, with a certificate the browser trusts, which the open item on holding a hosted
  user's master key describes with its costs. At `127.0.0.1` it can use no passkey.
- Which earlier decisions Drafts 1 and 3 change in part. D-216 (a cloud session signs with a key of its own
  that the user's shared key vouches for) names the cloud key as the one secret given to the environment,
  and D-302 (a cloud session's room is read on a static page) serves the page from GitHub Pages, so each
  is rewritten to what still holds when the drafts are adopted, as W-38 in `docs/writing.md` has it.
