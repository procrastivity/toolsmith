# Reconcile a tool with later contract minors

`toolsmith check` verifies marked mechanical extents. C3.10 reports whether a
tool's recorded `contractReconciledMinor` is current; it does **not** decide
whether the tool semantically conforms. A current integer is a record of a
review, not a verdict. Use this procedure when C3.10 reports a stale minor, or
when an existing tool needs a contract review for another reason.

## Establish the review range

Read the target's actual `manifest --json` and note
`contractReconciledMinor`, its current version, and the commit being reviewed.
Read the current version and clause-history table in `toolsmith doc CONTRACT.md`.
If the field is missing, invalid, or ahead of the current contract, investigate
its provenance; do not guess a baseline. Enumerate every intervening history
row and every clause it added. Read each full clause and nearby clarifications,
including clarifications that did not add a history row. The history table is
a navigation aid, not a rule that every listed change applies to every tool;
any history-table applicability or conformance assurance is a hypothesis to
verify against actual behavior, never a verdict to inherit. For example,
toolsmith's `backport/duo.md` item 17 documents a hand-authored skill that
contradicts a v1.3 history note's assurance that no existing tool shipped that
shape.

## Inspect and record each clause

For every applicable clause, inspect the real binary and source/repository; if
the target ships a skeleton, inspect a freshly instantiated skeleton too.
Record exactly one verdict per clause:

- **holds** — evidence demonstrates the applicable requirement;
- **holds with a note** — it holds, with a material qualification recorded;
- **diverges** — it does not hold; record a disposition: fix now, tracked
  follow-up, or explicit owner decision;
- **not applicable** — explain why the clause does not apply to this target.

Do not turn unresolved divergences into holds. Distinguish the scope covered
by `[check]` markers and mechanical `check --json` findings from the rest of
each clause. A clean check is useful evidence, not a semantic auto-verdict.
Capture exact code paths, commands, and relevant output rather than conclusions
alone.

Use a committed `docs/<matter>/decisions.md` (or the receiving repository's
equivalent review record) for verdicts and dispositions. Cite the contract
version and commit reviewed. Put actual reproduction, binary invocations,
checks, and output in `evidence/` as required by C7.1 and C7.3.

| Clause | Applicability / verdict | Evidence (path, command, output) | Divergence disposition / owner |
| --- | --- | --- | --- |
| Cx.y | applicable / holds, holds with a note, diverges, or not applicable (why) | exact path or command and observed result | fix now, tracked follow-up, owner decision, or — |

### Example enumeration (not verdicts)

For a hypothetical review from v1.2 to v1.5, enumerate the intervening rows:

- v1.3: C8.1–C8.4 (T30, T31)
- v1.4: C8.5, C3.9 (T32)
- v1.5: C3.10 (T25)
- Also read the C4.3 clarification even though it introduced no new history
  row, and assess applicability independently.

These are clauses to investigate, not findings that the hypothetical tool
holds or diverges. In the toolsmith checkout, `docs/contract-v1-2-reconcile/decisions.md`
§2 and `evidence/2026-09-11-contract-v1.2.md` are worked examples. Installed
users without that checkout still have this complete guide and the contract
available through `toolsmith doc`.

## Verify before recording a new minor

Run relevant tests, the tool's self-check, and fresh-skeleton checks where
appropriate. Only after the review record and supporting evidence exist should
you update the target's `contractReconciledMinor` and any assertions or fixtures
that pin it. Then rerun `toolsmith check`. A clean check alone never authorizes
the integer bump; if divergence remains, record its disposition and state any
unresolved conformance claim accurately.
