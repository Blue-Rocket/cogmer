# Why an old signing recipe is never edited, and how a test proves it

Written 2026-09-24 against `cmd/cogmer/keys.go` and `cmd/cogmer/keys_test.go` at
commit 43c6989, when the only signing scheme is v3 and no test holds an event signed
in the past.

## The cast

- **Ana** and **Ben** are colleagues in a cogmer room. **Carol** joins the room later.
- **The maintainer** is whoever changes cogmer's code.
- **Event E1** is one turn Ana wrote, stored as a record with these fields: an event
  ID, Ana's peer ID, a sequence number, the room ID and the text of the turn.

## Background: how a signature works

1. Before signing E1, Ana's cogmer turns E1's fields into one string of bytes, always
   in the same order, with the same separators. That string is the *signing bytes*.
   The rule for building it is the *recipe*.
2. Ana's private key signs the signing bytes and produces a signature. The signature
   travels with E1.
3. When Ben's cogmer receives E1, it rebuilds the signing bytes from E1's fields
   using its own copy of the recipe. It then checks the signature against Ana's
   public key.
4. The check passes only if Ben's recipe produces **byte for byte** the same string
   that Ana's recipe produced. If one byte differs, the check fails, and a check
   failing is exactly what a forged or altered event looks like.

cogmer calls a recipe a *scheme* and numbers it. The current one is v3, and every
event records which scheme signed it.

## Scenario 1: the maintainer edits the recipe in place

1. **Monday.** Ana runs cogmer 0.7.1. Ana's cogmer signs E1 with the v3 recipe,
   which has five fields. Ben's cogmer receives E1, checks it, and stores it.
2. **Tuesday.** The maintainer wants events to also record which model wrote a turn.
   The maintainer adds a `model` field to the v3 recipe itself, so v3 now has six
   fields. cogmer 0.8 ships with that change.
3. **Wednesday.** Carol joins the room with 0.8. Carol's cogmer fetches the room's
   history from Ben's, including E1.
4. Carol's cogmer sees that E1 is marked v3 and rebuilds E1's signing bytes with the
   recipe 0.8 calls v3, the six-field one. The result differs from the five-field
   bytes Ana signed on Monday.
5. The signature check fails. Carol's cogmer rejects E1 with "signature does not
   match the peer id", which reads as if someone forged Ana's turn.
6. Nobody can repair E1. Ben's cogmer can't re-sign it, because only Ana's key can
   sign as Ana. Even Ana's cogmer can't sign E1 again without producing a different
   record, and every event in the room already signed under the old recipe has the
   same problem.

The same failure hits Ben if Ben's database is lost. cogmer's recovery is to refetch
the room's history from the other colleagues. The refetched events arrive, fail the
check, and are rejected.

## Scenario 2: the maintainer adds a new recipe alongside

1. Tuesday is different. The maintainer leaves v3 untouched at five fields, and adds
   a v4 recipe with six.
2. From 0.8 on, new events are signed with v4 and marked v4.
3. Carol's cogmer fetches E1 and sees it's marked v3, so it uses the five-field v3
   recipe. The bytes match what Ana signed, and the check passes.
4. New events are checked with v4, and those pass too.

**Result:** old and new events both verify. That's the rule: add a new scheme beside
the old ones, and never edit one that has already been used.

## Why the tests wouldn't catch Scenario 1

Every signing test in cogmer does two things in a row:

1. It signs a brand-new event with today's recipe.
2. It checks that event with today's recipe.

In Scenario 1, after the maintainer's edit, "today's recipe" is the six-field one on
both sides, so the bytes match and the test passes. The test never sees an event
signed **before** the edit, so it can't notice that such events stopped verifying.
Carol is the first to find out, after release.

## What a saved event adds

1. Today, sign one event with the five-field v3 recipe, and save that event and its
   signature in the repository as a fixed file.
2. Add a test that checks the saved event with whatever the v3 recipe is at the time
   the test runs.
3. If the maintainer later edits v3, the test rebuilds the saved event's bytes with
   the edited recipe. They differ from what was signed, and the test fails on the
   maintainer's machine before release, instead of on Carol's machine after it.

The saved event stands in for E1: it's a record signed in the past that today's code
must still accept.

## The general pattern

cogmer is one case of something that happens whenever records leave your hands. Once
other machines hold copies, you can't go back and change those copies. So when you
change the format:

- write new records in a new version;
- keep your code able to read every old version;
- keep a saved sample of each old version in your tests, so you find out the moment
  your code stops reading one.
