# Cowork first-run design

A route by which a Cowork-only user starts work by installing the plugin, with five draft decisions and
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
2. A task that never creates or joins a room needs nothing, and starts nothing (D-286, a Cowork cloud
   session with no cloud value fetches and starts nothing). When the user creates or joins a room in a
   task, the command starts the task's daemon with no secret of the user's, which makes its session key,
   and prints the link to the cloud page (D-302, a cloud session's room is read on a static page) with a
   token made for this task. `/cogmer:start` does the same with no room, for a user who would set up first.
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
5. The task speaks for the user in their rooms, as a task with an uploaded value does today. A resumed task
   keeps its key and certificate (spike C). A new task that creates or joins a room taps again, at the
   moment the user asks it to share, and a task that never does is never asked.

On a Team plan an owner still allows the hosts the task's daemon reaches, which are `cogmer.ai` and `*.cogmer.ai` once cogmer runs its own relay there, as the open item on a relay change records; a Pro or Max plan was not tried. A user who would rather have
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
the cloud page, before it accepts the session key. The
task sends the passkey's certificate with its events, as a machine sends its own (D-219, a user is a master
key that certifies machines through a shared key), and the page gives it to the task from its memory or
from the directory of Draft 4, with the addresses of the user's machines. From then on the task renews its certificate every two
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

D-216 (a cloud session signs with a key of its own that the user's shared key vouches for) rejects a cloud
key that expires, because renewing it needs a machine that holds the shared key, which a user away from
their machines cannot reach. This draft makes no cloud key expire: it certifies the task's session key with
no cloud key between, and a user away from their machines certifies a task again with a passkey prompt
from any device the passkey reaches, a hosted machine is always reachable, three days covers a weekend
away, and a task whose certificate ends pauses and holds its turns until either happens. Nor does the task
relay its exchanges through the user's machines, which D-216 also rejects, since it asks them only to renew.
D-216 is rewritten in place, as W-38 in `docs/writing.md` has it, to name two ways a session key is
certified, by the cloud key or by the passkey with the user's machines renewing, and to say why its
rejection of an expiring cloud key does not reach the second.

A passkey is not a cloud key, and D-280 (a user has one cloud key at a time) does not count it. A user may
register several, such as one in iCloud Keychain and one in Windows Hello, each certified on its own, and
may keep one uploaded cloud key beside them. D-280 refuses a second cloud key because the plugin that
carries it reaches every cloud session of the account, so a second would hold the same authority as the
first with no use. A passkey is in no session, signs only when its user answers a prompt, and certifies
one task's key at a time, so a second passkey serves another device and not the same sessions twice.

Rejected.
- *Counting a passkey as the user's one cloud key.* A user whose devices keep passkeys with different
  providers could register only one, and keeping an uploaded value would take the passkey's place, while
  the risk D-280 limits is a secret copied into every session, which a passkey is not.
- *Retiring the uploaded cloud key now, with the passkey and the token for unattended tasks in its place.*
  It removes what users have before the passkey route is built and tried.
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

## Draft 2: a cloud task asks for the user's passkey only when a command needs the user's identity

Decision. A Cowork cloud task with no cloud value starts nothing at its start (D-286, a Cowork cloud
session with no cloud value fetches and starts nothing). A command that needs the user's identity, which in
such a task is creating or joining a room, or `/cogmer:start`, starts the task's daemon, which serves the
page's port with a token made for the task, and prints the cloud page's link, so the user taps their
passkey at the moment they ask the task to share and never in a task that does not. A hook's messages reach
the model and never the person (spike D), so a line that tells the user where to start can reach them only
through the model, and a command the user types is a turn the user asked for. D-286's Limits already let a
command start a daemon in such a task.

The command does the same in every cloud session with no identity and no cloud value, Claude Code's as
well as Cowork's, in place of the words of D-281 (a command in a cloud session with no value says the user
must first set cogmer up on one of their own computers), which this replaces. The page's first question
offers a computer of the user's own, with the line to paste into Claude Code or Codex there, so a user who
has a computer is told no less than D-281 told them, and D-286's Limits (a Cowork cloud session with no
cloud value fetches and starts nothing) are rewritten to say the command prints the link.

The page's token is the HMAC-SHA256 of D-301's fixed label under the task's session key, as today's is
under the cloud key (D-301, a cloud session's daemon serves its room on a second overlay port), so it is
the same after every wake, since a resumed task keeps its files (spike C), needs no file of its own, and
opens only this task. Before the task holds a certificate, the token admits one write as well as the
reads: the first certificate for the task's own session key. The task takes a first certificate once, and
afterwards only renewals that chain to the same user, so someone who reads the link in the conversation
before the user taps can claim the task, and the page then shows the user that the task is claimed under
another name. D-301 is rewritten to serve a daemon that holds no cloud key and to take that one write.

Rejected.
- *A hook that asks the model to tell the user how to start.* The model said the link when asked (spike
  D), but the line would be in the model's context in every task of every user who had not set up, and
  would make the model raise cogmer in a conversation the user started for something else, which §3.1
  (first, do no harm) rules out.
- *Printing the link in Cowork tasks only, and keeping D-281's words in a Claude Code cloud session.* The
  command would tell the hosts apart for no gain, since the page's own-computer choice says what D-281
  says, and a Claude Code user who has no computer at hand would be left without the passkey route.
- *A random token for each task, made at each start or stored in a file.* D-301 rejects both: the first
  breaks the user's link at each wake, and the second needs a file and a rule for when it changes.
- *No token until the task is certified.* Anyone who reached the address could then hand the task a
  certificate, where the token limits that to whoever can read the conversation.
- *Certifying every task at its start.* Most tasks never join a room, and each would need a tap or a secret
  stored for it.

Limits. A user who does not know the commands shares nothing, so the plugin's description and the README
carry them. A task that creates or joins a room with nobody there to tap shares nothing until someone does,
unless the user has uploaded the token in the question on unattended tasks.

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

D-302 (a cloud session's room is read on a static page) is rewritten in place, as W-38 in `docs/writing.md`
has it: the page's host becomes cogmer's, its case under D-115 (a centralized component must trace to a
disclosed tradeoff that benefits the person) is made again for a host cogmer runs and disclosed in
`plugin/README.md`, and its security policy names cogmer's relay map in place of `tailcat.dev`.

Limits. cogmer runs the host, and its trust is the trust D-302's Limits give the page's host.

Revisit when cogmer gives up the domain.

## Draft 4: the cloud page remembers the user's machines, and a directory cogmer runs answers when it cannot

Decision. The cloud page keeps, in its own storage on each device, the addresses of the user's machines,
which it reads from any task it is open on that has reached one of them, and gives them to a new task with
its first certificate, so a task started away from the user's computer reaches the computer when it answers.
Where the page holds none, on a new device, after its storage is cleared, or when an address has changed,
it asks a directory that cogmer runs under `cogmer.ai`, which holds for each user a record of their
machines' addresses with each machine's certificate chain and the passkey's certificate, signed by one of their machines and stored
under the user's identifier. The directory takes a write only when the chain leads to that identifier, as
any machine checks a record of the user's, and a reader checks the record the same way, so the directory
can withhold a record or serve an old one and never send a reader to a machine that is not the user's. It
holds no name, so a user is found only by someone who already has their identifier, as D-142 (identities
are exchanged through pairing strings, with no directory) has it, and it keeps no log of requests and only
the latest record of each user. The page reads the user's identifier from the passkey's user handle, which
holds up to 64 bytes and comes back at every sign-in on every device, and the user's colleagues hold it
from pairing, so a colleague also finds a machine the user added since they last spoke. A user with a
hosted machine needs neither, since the user handle names the hosted machine.

Rejected.
- *A directory that cannot tell whose record it holds.* Storing records under a hash, sealing them, and
  refusing other writers without the identifier needs a write key derived from the shared key, a new
  identity record to publish it and a pointer to follow when the shared key changes, to hide from cogmer
  what the relays already see: which machines reach which (D-192, the public relay is an accepted
  dependency).
- *A mailbox at `cogmer.ai` that carries the certificate from a small passkey page to the task.* It kept
  the cloud page's module out of the passkey step, and the module's cost is paid once on each device: an
  iPhone's first open took 19.6 seconds to its first data and its second 1.6 (`docs/open.md`, the item on
  the page that reads a cloud session's room).
- *A directory that maps a name or an email address to a user's key.* Anyone could ask it whether someone
  uses cogmer, and it reverses D-142, which the open item on reaching a user by a handle weighs on its own.
- *Carrying the computer's addresses in the passkey's `largeBlob`.* Most passkey providers decline it, and
  none is documented to sync it (the question on where a task learns how to reach its user's machines).

Limits. The directory learns each user's identifier, machines and addresses, and who looks a user up and
when, which for a cloud task is an address of Anthropic's (spike C) and for a colleague is their own. It is
a component cogmer runs, which D-115 (a centralized component must trace to a disclosed tradeoff that
benefits the person) admits only for a benefit to the user, recorded and disclosed: here, tasks that reach
the user's computer from any device and colleagues who reach the user's new machines, said in the README's
account of what leaves the user's machine, with its promise to log nothing. The page's memory is per
device and per browser, and a private tab keeps none.

Revisit when the passkey providers most users hold carry data with a passkey to every device, so that a
task could take its user's addresses from the passkey.

## Draft 5: an invitation link and its reply verify two colleagues over a channel they trust, with no call

Decision. A user invites a colleague they have not paired with by sending a link over a channel the two
of them already trust, such as a chat they share or an email, and the invitation completes only when the
colleague sends a reply link back over that channel. The invitation is `https://cogmer.ai/i#` followed by
the inviting user's pairing string (D-097, a pairing string carries a chosen name and never a guessed one),
the room's identifier and name, and a random value naming this invitation, all after `#`, so cogmer's
server never receives them, and none of them is a secret. The reply is `https://cogmer.ai/r#` followed by
the colleague's pairing string and the value naming the invitation.

Opening either link acts on it. The page the link opens loads the overlay module the cloud page loads
(D-302, a cloud session's room is read on a static page), finds the opener's machines in its memory or in
the directory of Draft 4 under the identifier the opener's passkey names, and asks the opener to confirm,
as "Join *room* from Alice?" for an invitation or "Add Ben to *room*?" for a reply. The confirmation is a
passkey assertion whose challenge is a fresh value from the opener's machine with a hash of the link, and the
page sends it through a relay to a port of the overlay on which the machine accepts only a link signed so,
checked against the passkey the machine certified (Draft 1). It works from any device the passkey reaches,
the opener's phone among them, while one of the opener's machines answers. An opener who has no passkey,
at the computer that runs cogmer, is sent by the page to the machines page at `127.0.0.1` with the link
after `#`, as the passkey challenge was in spike F, and confirms there with a button. Where neither works,
the page offers "Open in Claude", which opens a new session with the line already in its prompt,
`/cogmer:room-join` followed by the invitation or `/cogmer:peer-pair` followed by the reply, so the opener
only presses send, and shows the same line to copy beside it. On the user's iPhone,
`https://claude.ai/code/new?q=` opened a new Claude Code session in Safari with the prompt filled in and
unsent, under Claude's caution about running a prompt from a link, and a cogmer command sent from it ran
(`docs/work/first-run-spike-G-20261127T1407Z.txt`), and tapped in Notes, Mail or Messages, where an invitation
arrives, it opened a new Claude Code session in the Claude app with the prompt filled in and no caution
shown, so the page says before the button that Claude may show a caution. Where the Claude desktop app is installed, the button can open a Cowork session the same
way through `claude://cowork/new?q=`, which Anthropic documents and which was not tried. The page tells
someone without cogmer to install the plugin first. The link carries no secret, so putting it in the
address claude.ai receives costs nothing. A link never acts without its opener's tap or button, since anyone can make one and any page in
the opener's browser can send it to `127.0.0.1`.

The colleague's confirmation records the inviting user as a verified colleague and the room as offered, and
gives the colleague the reply to send back. The inviting user's confirmation of the reply, on a machine
that finds an invitation of its own with that value, records the colleague as verified and puts them on the
room's guest list (D-235, a room's one guest list decides whose events count), and the offer then reaches
the colleague's machines (D-105, an invitation is delivered to a paired guest as an offer over the paired
channel). Each machine finds the other's machines in the directory of Draft 4 by the identifier when the
string carries no address it can reach.

The invitation grants nothing to whoever holds it. Anyone who uses it learns the inviting user's identifier,
which is safe to know (D-023, a peer identifier must be safe to know), and gets a reply that the inviting
user acts on only when it arrives from the colleague they invited. A reply to an invitation the machine did
not make is refused, and a second reply to one invitation is shown to the inviting user beside the first
rather than recorded.

The inviting user names the colleague the invitation is for when making it, and the name stays on their
machines with the invitation and becomes the colleague's label when the reply is accepted, which is the
label D-093 (the label is collected with the pairing string and written only when the words match) collects
while pairing. Confirming a reply shows the name the inviting user gave, the name the replier chose (D-186,
a user chooses the name colleagues see when the user identity is made in the daemon's page) and how long
ago the invitation was made, and asks whether the reply reached them from that colleague in the chat or
email the invitation went to. It says so in a sentence when the two names differ, when another reply to the
invitation came first, when the invitation is old, and when the key is already a colleague's under another
name. Refusing a reply records nothing of it, leaves the invitation open for the colleague it was meant for,
and offers to cancel it.

While a reply is awaited, the room's page on the inviting user's machines shows "Invited Ben by link,
waiting for a reply" with a button that cancels the invitation, and the colleague's machine lists the room
among its offers as "Waiting for Alice to open your reply" with the reply there to copy again, as D-188 (a
waiting offer is shown with the rooms and a withheld one beside the peer who can release it) shows a
waiting offer. An invitation expires 30 days after it is made: its page then says that it has expired and
to ask the inviting user for a new one, and a reply that arrives after it is refused. The colleague keeps
the inviting user as verified, since the link reached them in time.

A link establishes a colleague who is new and never replaces the key of one the user has. A reply or an
invitation whose key differs from the key of a colleague with the same name meets D-074 (a name means one
master key, and a collision is where a new master key surfaces): nothing is recorded, and the page says
that a new key cannot be told from somebody else's sent in that colleague's name, and that the key is
replaced only by forgetting the old one and comparing the two words with the colleague on a call. A message
saying "I set up again, here is my new link" is the one a channel carries unchanged for an attacker, and it
reaches a user whose guard is down because they know the colleague, so the ceremony that does not trust the
channel guards it.

Two colleagues who have paired already need neither link. The two words remain for a
user who would rather not trust the channel. The verified field of a colleague's entry (D-228, a user's
colleagues, verifications, labels and rooms belong to the user and are copied between the user's machines)
records whether the words or links verified them, the colleague list shows "verified by link" for the
second, and comparing the words later records the words. Nothing gates on which: synchronization, names
and the view's markers treat both alike (D-054, verification gates synchronization and injection). Every
colleague verified before this is recorded as verified by the words, the only way there was.

Rejected.
- *A link carrying a one-time secret, whose first use verifies and admits.* It is useful to anyone who
  reads it before the colleague does, an administrator of the chat or a person it was forwarded to, which
  D-026 (there is no join token at all) rejects, and a theft is found only when the colleague uses it after.
- *The reply as two words read back over the channel.* Words are short enough that someone who reads the
  reply can make a key whose words match before the inviting user opens it, where a reply carrying the
  whole identifier cannot be matched.
- *Recording only that a colleague is verified, not how.* Which colleagues a channel verified is known
  only at the moment it does, so a user who learns that a chat account was taken over could not find the
  colleagues to verify again with the words.
- *A room that admits only guests verified by the words.* It adds a setting to every room and a refusal to
  explain to a colleague, for a need nobody has shown.
- *An invitation that never expires, or one that expires in seven days.* One that never expires lets a link
  found months later bring a reply the inviting user may accept without remembering whom they invited, and
  seven days ends an invitation before a colleague back from two weeks away has seen it.
- *A link that replaces a colleague's key.* One message in the colleague's name would take over a colleague
  the user already trusts, which is worse than a first contact gone wrong.
- *An invitation that the colleague's key opens, with the colleague sending their pairing string first.* The
  inviting user still has to trust that the first message came from the colleague, so the check moves to
  the start without growing stronger, and the colleague has to act before they can be invited.
- *The two words alone, with a hosted machine answering for a user who has no computer.* The exchange
  needs both people on a call at once.
- *A password-authenticated key exchange, such as SPAKE2 or CPace, with a code sent once.* It trusts the
  channel that carried the code, as the links do, and needs both machines to take part at the same moment.

Limits. The first command of a new session can arrive before the session has downloaded cogmer, and it
then says to wait and run the command again, which a person opening their first invitation meets at once
(spike G). On a phone the link opened Safari when tapped in the Claude app, and no documented link opens a
Cowork task on a phone. A verification by links trusts that the channel delivers what each person sent and shows who sent
it, so a channel that substitutes both links is not found out, as with anything else said on that channel.
The colleague waits for the inviting user to open the reply before reading the room, though neither has to
be present when the other acts. Opening a link on a device first loads the overlay module, which took 19.6
seconds to first data on an iPhone's first open and 1.6 on its second (`docs/open.md`, the item on the page
that reads a cloud session's room), and the opener's machine has to be awake. Every machine whose user has
a passkey listens on the overlay for a signed link, a listener that admits any dialer (D-157, the overlay
listener admits any dialer) and acts on nothing but a link its user's passkey signed.

Revisit when a reply is found to have come from someone other than the colleague invited, or when people
come to send the links over channels that do not show who sent them.

What it changes. D-055 (a peer is verified in one way, the two-word comparison) gives way to two ways, and
its rejected fallback, "the path people reach for when the other is inconvenient", is what the links will
be: most colleagues will be verified by links. D-163 (only a person's confirmation records a verification,
and a mismatch records nothing) takes the colleague's use of the invitation and the inviting user's opening
of the reply as the two confirmations. D-301 (a cloud session's daemon serves its room on a second overlay
port, in lines, with a token made from the cloud key) serves only reads and only in a daemon that speaks as
a cloud session, and rejects serving a port in every daemon because it would be open for every user; a
port that does nothing without the user's passkey signature answers that rejection differently, and D-301
is rewritten to hold both. D-026 (there is no join token at all), D-048 (verification is a live
commit-then-reveal exchange that derives two words), D-051 (stranger pairing is not a supported case) and
D-074 (a name means one master key, and a collision is where a new master key surfaces) stand, and the
links rely on D-074 to send a changed key to the words.

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
  - A user whose only machine is a computer, away from it, gets the first certificate from the passkey, and
    the page gives the task the computer's addresses from its memory, or from the directory, by the user's
    identifier, which the passkey's user handle holds (Draft 4). A machine's overlay address is about 186 characters
    (`docs/work/cloud-run-20261025T171545Z.txt`), more than a user handle holds itself.
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
- Whether the Mac's browsers, and the same passkey reached through iCloud Keychain, return the same secret.
  Only Safari on an iPhone was tried.
- How a task that joins a room with nobody there to tap is certified, such as a scheduled task whose prompt
  holds a cogmer command. An upload is the only way to store a secret where a new task reads it (spikes A
  to C), so such a user opts in by uploading a token that only their own machines honor, in place of
  today's cloud key, which every colleague honors: the task presents it to one of the user's machines and
  gets a three-day certificate, and the machine can show that a task started, limit how often the token is
  used, and revoke it at once, where a cloud key that is copied speaks for the user until a removal
  reaches each colleague (D-225, a machine sends each removal and new shared key to every colleague it
  knows until each acknowledges). Onboarding does not offer it; the machines page does, for a user who
  needs it.
