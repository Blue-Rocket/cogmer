---
name: decisions
description: Find the decisions that govern an area before proposing a change to how anything works, and to answer why something is the way it is. Use before changing, simplifying or redesigning pairing, rooms, admission, identity, sync, transport, capture, the daemon, the view, commands, release or any other part of cogmer.
---

# Decisions

`docs/decisions/` holds one file for each decision. Each records what is decided, the facts it
rests on with their sources, the alternatives rejected and why, and what it does not cover. The
rejected alternatives are what stop a change proposing again something that was ruled out.

## Find the decisions for an area

1. Read `docs/decisions/areas.md`. It lists the areas, each with what it covers, and a decision
   may belong to several.
2. List the active decisions in an area. The date line of each file ends with its areas.

   ```
   rg -l 'Status:\*\* (active|not built).*Areas:.*\bpairing\b' docs/decisions
   ```

   Replace `pairing` with the area. A file name carries the number and the title, so the list is
   readable without opening anything.
3. Read each file found, and read **Rejected.** and **Limits.** closely. A change that revives a
   rejected alternative needs a reason the file does not already answer.
4. A subject that spans areas is found by its words in the titles, with
   `ls docs/decisions | rg -i 'word'`.
5. To follow a citation such as `D-054`, run `ls docs/decisions | rg D-054`. A file whose status is
   withdrawn, moved or removed names what replaced it.

If no decision covers the area, read the specification section for it, and read `docs/open.md`
for what is unfinished or undecided.

## Before proposing a change

Say which decisions govern the area and whether the change reverses one. A reversal is recorded:
the earlier entry becomes a tombstone, and the reason goes in the replacing decision's
**Rejected.**, as `docs/writing.md` W-37 says.
