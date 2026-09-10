<!--
Workplan template (toolsmith handoff-kit). Copy to your sidecar as
workplans/<slug>.md, one per Matter. Delete this comment and the
> GUIDANCE blocks once the workplan is real.

A workplan is written before the work and amended during it. When the
Matter seals, the file must read as an accurate description of what was
actually built — that is what the reconciliation Step at the end is for.

Placeholder Matters (work whose phase has not opened) get Intent plus a
seed pointer only. Do not pre-plan their Steps.
-->

# Workplan: `<slug>`

## Intent

> GUIDANCE — one paragraph. What this Matter completes, in terms of
> observable behavior, with the contract clauses and handoff decisions
> it is bound by cited inline. End with the earned shape from the
> register.

<What is true when this Matter is done.> <Clauses it must satisfy: C…>
<Decisions that constrain it: H…> Earned shape: **<Steps only | Brief +
Steps | Brief + Stages>** (register).

## Open calls resolved here

> GUIDANCE — every open call from the seed card, resolved with a short
> rationale, or carried as an explicit early Step. Resolving a call
> here — before the Steps — is what stops it from being re-decided
> three times inside the same Matter.

- **<Call> — resolved as <answer>.** <Rationale in two or three
  sentences, including the alternative rejected.>

---

## Brief: <subject>

> GUIDANCE — include this section only when the register says the Matter
> earns a Brief: shared conventions that several later Matters will
> re-read. Write it as conventions-to-re-read, not as narrative. Open
> with a line saying who must read it and when, and state that the Brief
> wins over a later disagreement, or gets amended in place.

*Read this before touching <the code this governs>. If a later change
disagrees with this Brief, the Brief wins or gets amended in place.*

### <Convention>

<The rule, then why, then the rejected alternative.>

---

## Stages

> GUIDANCE — use Stages only when the Matter has internal sequence worth
> naming, and when other Matters need to depend on part of it. Otherwise
> go straight to Steps. Name each Stage and give it its own Step list.

### Stage `<name>`

<Steps for this Stage.>

## Steps

> GUIDANCE — every Step carries a **Done:** criterion that someone else
> could check without asking you. "Implemented X" is not a criterion;
> "X exists, and <observable consequence>, confirmed by test" is.
> Number them step-01, step-02, … so later notes can cite them.

**step-01 — <short title>.**
Done: <observable criterion, and how it is confirmed.>

**step-02 — <short title>.**
Done: <criterion.>

**step-NN — Reconciliation pass.**
Done: the Brief above re-read against what was actually built in steps
01–<NN-1>; every divergence folded back into the Brief text in place and
marked as an implementation-forced note; any new error code or
vocabulary drafted here flagged as provisional rather than treated as
ratified.

> GUIDANCE — the reconciliation Step is not optional, including for a
> Matter with no Brief (there, it reconciles the Intent). It is the one
> Step that keeps the sidecar true.
>
> Implementation-forced notes go **in the text they contradict**, marked
> "(step-NN)", not in a trailing errata list. A reader hitting the
> outdated sentence must meet the correction there.

## Blocked-by

> GUIDANCE — restate the edges from HANDOFF §5 rather than referring to
> them. A workplan should be readable alone. Say which edges are
> *partial* (satisfiable at locally-complete) and note anything that
> runs concurrently, so the schedule is not read out of the edge list.

`<slug> ← <slug>` (HANDOFF §5) — <what specifically is needed.>

*Downstream, for orientation only (not edges this Matter must satisfy):*
`<slug> ← <slug>`.

## Seal condition

> GUIDANCE — copy the register's condition verbatim. Two copies that
> differ is worse than one copy that is inconvenient to find.

<The condition from HANDOFF §3.>

## Subagent notes

> GUIDANCE — whether anything inside this Matter may be dispatched
> concurrently, which batch it sits in, and what a parallel session
> would need. Default posture: strict Step sequence, one worker.
> Design-work fan-out (two independent spikes) is the usual exception.

<Sequence posture.> <Batch membership, and the reminder that batches
schedule but never order.>
