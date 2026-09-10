<!--
HANDOFF template (toolsmith handoff-kit). Copy to your sidecar as
HANDOFF.md and fill it in. Delete this comment and every > GUIDANCE
block before the document is used.

What this document is: the bridge between a design session and the
sessions that write workplans and code. It ratifies *shapes* — which
Matters exist, what each earns, the dependency edges. It does not
contain Step lists; those are the workplans' job.

Rule of thumb: if a later session would otherwise have to re-decide it,
it belongs here. If a later session should decide it with the code in
front of them, it belongs in a workplan as an open call.
-->

# <tool> — planning handoff

**Status:** <ratified | draft> <date>. <One sentence: what this document
bridges.>

**Seed set for a receiving session:** <the exact documents to read, in
order>. `SEED-CARDS.md` maps each Matter to exactly what to read for it.

**Your job (receiving session):** write the workplans — Briefs, Stages,
Steps — for every Matter in §3. The shapes are ratified here; do not
re-litigate them. The contents are yours to write. Where a seed card
lists settled inputs, encode them; where it lists open calls, resolve
each with a short rationale or carry it as an explicit early Step.

---

## 1. Operating constraints

> GUIDANCE — these are binding on every later session, so keep them few
> and keep them checkable. Recurring ones worth stating explicitly:
> what process machinery does *not* exist yet; where ordering comes from
> (the edge list, never list position); what may run concurrently; where
> the workplans live (T22 — the sidecar or a tracker, never both);
> whether early plans may be amended.

1. **<Constraint>.** <Why, in one or two sentences.>
2. **Ordering comes only from the edge list (§5).** Sibling order in any
   list in this document is presentation-only. Never infer sequence from
   position.
3. **Where the workplans live (T22):** <`workplans/<slug>.md` in this
   sidecar, one file per Matter | the tracker, at <name and locator
   scheme>>. Name one and only one, so a session arriving cold does not
   create the other. Either way a workplan contains intent (one
   paragraph); a Brief only if §3 says the Matter earns one; Stages
   and/or Steps with per-Step done-criteria; `blocked-by` edges
   restated; the seal condition; subagent notes where relevant.
4. **Early-planning risk is accepted.** If an early Matter's
   implementation invalidates a later workplan, amend that workplan.
   The plan is not fixed.

## 2. Decisions locked in this session

> GUIDANCE — one row per decision, with a one-line rationale. These are
> the answers later sessions must not reopen. Number them (H1, H2, …) so
> workplans and code comments can cite them.

| # | Decision | Rationale (one line) |
|---|---|---|
| H1 | <Decision.> | <Why.> |
| H2 | <Decision.> | <Why.> |

> GUIDANCE — decisions almost every conversion needs to lock:
> the branch posture (orphan branch, old line stays installed);
> conventional commits from commit one; the CLI framework and module
> path; which harnesses v1 targets; whether release engineering is
> deferred; what the cutover criterion is; which parts of the old
> implementation may be consulted at all.

## 3. Matter register

> GUIDANCE — a Matter is a unit of work with one intent and one seal
> condition. "Earned shape" says whether it gets a Brief (shared
> conventions several later Matters re-read) or Steps alone. Slugs are
> locators; the order below is presentation-only.

<N> Matters: <breakdown, e.g. two scaffold, six core, two placeholders>.

| Slug | Intent | Earned shape | Seal condition |
|---|---|---|---|
| `<slug>` | <One sentence: what this Matter completes.> | <Steps only \| **Brief** + Steps \| **Brief** + Stages> | <Observable condition. Something you can run.> |

## 4. Deferred backlog

> GUIDANCE — work that is real, named, and deliberately not now. Record
> the origin so the reason survives. A deferred entry with a constraint
> attached ("nothing in X may make Y harder") is how you keep a door
> open without walking through it.

1. `<slug>` — origin: <which decision deferred it>. <One line of scope.>
   <Constraint carried, if any.>

## 5. Edge list (`blocked-by`)

> GUIDANCE — the only source of ordering. Matter grain by default; name
> a Stage only where the finer edge is real. Cross-Matter Step-to-Step
> edges are a smell — if you need one, document it here explicitly so
> nobody "fixes" it later.

```
<matter>            ← <matter>, <matter>
<matter>/<stage>    ← <matter>
```

Unblocked at t=0: `<slug>`, `<slug>`.

## 6. Batches

> GUIDANCE — scheduling groups only. Membership implies nothing;
> ordering always derives from §5. Useful when several Matters can be
> dispatched to parallel sessions.

- **<name>** (<suggested concurrency>): `<slug>`, `<slug>`.

## 7. Inheritance from the old implementation

> GUIDANCE — for a migration only. Name exactly what may carry over and
> what may not. When the posture is evidence-only, this section is short
> and its whole purpose is to stop a later session from "just checking
> how the old one did it". When a part is an oracle, say so here and
> point at the port spec.

Carried over: <dev-loop verb names, lint posture, config files, …>.
Oracle parts: <part → what the parity gate covers>.
Never consulted: <the old design, its layout, its data format>.

## 8. Intake remainder

> GUIDANCE — taste-level items the planning session could not settle and
> the implementing session should just decide. Listing them here stops
> them from being mistaken for oversights.

- <Item.>
- <Item.>
