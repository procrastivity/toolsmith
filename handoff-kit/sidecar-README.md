# The conversion sidecar

A conversion runs in **two repos**: the tool's own repo, where code
lands, and a **sidecar** that holds the planning record. This page says
why the split exists and how to set one up.

The pattern is wip's: `wip` held the Go rewrite, `wip-reboot` held the
model, the plan, the handoff, the seed cards, and one workplan per
Matter. It survived a dozen sessions across weeks. The templates beside
this file are those formats, emptied.

## Why a separate repo

- **Planning documents outlive sessions and outnumber commits.** They
  get amended constantly, in ways that would drown the tool's history.
- **The tool's repo should read as the tool.** A newcomer cloning it
  finds code, a README, and `docs/<matter>/` — not sixteen workplans for
  work that is finished.
- **The design of record must be citable from outside.** Code comments
  and `docs/<matter>/decisions.md` cite the sidecar by name; the sidecar
  never gets copied into the tool repo, so there is exactly one copy to
  amend.

What does live in the tool repo: `docs/<matter>/decisions.md` (what
building this Matter forced), `docs/<matter>/port-spec.md` (C7.2), and
`evidence/` (C7.3). Those are records the code needs beside it. Plans
are not.

## Setting one up

```
mkdir ../<tool>-reboot && cd ../<tool>-reboot && git init
cp <toolsmith>/handoff-kit/HANDOFF-template.md    HANDOFF.md
cp <toolsmith>/handoff-kit/SEED-CARDS-template.md SEED-CARDS.md
mkdir workplans     # only when the conversion has no tracker — see below
```

Then add whatever design documents this conversion needs. Typically:

| File | What it holds |
|---|---|
| `MODEL.md` or `DESIGN.md` | The design of record: the thing being built, its concepts, its invariants. |
| `PLAN.md` | The work broken into phases, before it is broken into Matters. |
| `intake.md` | The intake sheet from `playbook/intake.md`. |
| `HANDOFF.md` | The bridge from planning to execution. Its §1 constraints are binding on every later session. |
| `SEED-CARDS.md` | Per-Matter reading lists, so a session reads exactly what it needs. |
| `workplans/<slug>.md` | One per Matter: intent, Brief if earned, Steps with done-criteria, edges, seal. Present only when there is no tracker — see "Where the workplans live". |
| `appendix-*.md` | Deep dives too long for the model: packaging, a subsystem, a protocol. |

Small conversions collapse this: a single `HANDOFF.md` plus two or three
workplans is a legitimate sidecar. The shape scales down, and the
temptation to skip it entirely is what produces the third session that
re-decides the exit-code table.

## Where the workplans live (T22)

This kit describes a **model** and offers one **medium** for it. Keep the
two apart, because only the model is fixed.

The model is: a Matter has one intent and one seal condition; shape is
earned, never default; ordering comes only from the edge list. That holds
for every conversion.

The medium is where the model is written down, and there are two:

- **No tracker.** `workplans/<slug>.md`, one file per Matter, is the
  register. Create the directory and write the files. This is the
  default, and the kit's vocabulary — Matter, Stage, Step, seal — is
  yours to use as plain words.
- **A tracker with a workplan slot.** The tracker is the register, and
  the directory is not created. Do not keep a second copy in the
  sidecar: two copies of a workplan diverge by the third session, which
  is the same failure this whole pattern exists to prevent.

`HANDOFF.md` and `SEED-CARDS.md` are files either way. They are the
bridge and the reading lists; no tracker holds them well, and a receiving
session needs them before it has a Matter to open.

Whichever medium you pick, **say so in `HANDOFF.md` §1**, so a session
arriving cold knows where to look and does not create the other one.

That vocabulary came from wip, where this pattern was first run, and wip
is one such tracker. The words are not a dependency on it.

## Point the tool repo at it

Add to the tool repo's `CLAUDE.local.md` (which `.gitignore` covers):

```
Planning docs for this rewrite live in <absolute path to the sidecar>.
<DESIGN>.md is the design of record; workplans/ holds one workplan per
Matter; HANDOFF.md §1 constraints are binding. They are external on
purpose — never copy or commit them into this repo.
```

## Running a session from it

Seed each working session with the smallest correct set:

1. toolsmith's CONTRACT.md;
2. the playbook page for the current stage;
3. `HANDOFF.md` §1 and §2 (constraints and locked decisions);
4. the seed card for the Matter at hand;
5. that Matter's workplan, from whichever medium §1 names.

The seed card exists precisely so that item 5 does not expand into "read
the whole sidecar". A session that reads everything spends its context
on material it will not use, and re-litigates decisions §2 already
locked.

## Keeping it honest

- **Amend, do not append contradictions.** A decision that changes gets
  rewritten in place with a note, not shadowed by a later entry.
- **Implementation-forced notes land in the workplan**, in the Brief
  text they contradict, marked as such. That is what makes a workplan
  readable a month later as a description of what was actually built.
- **The last Step of every Matter is a reconciliation pass.** Re-read
  the Brief against the code and fold in the divergences.
- **Seal, then stop editing.** A sealed Matter's workplan is history.
  New work gets a new Matter.
