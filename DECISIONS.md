# Decision register — the toolsmith conventions

Operative form only: one line of decision, one line of rationale.
Amendments are folded into the operative text with an italic
parenthetical; closures rewrite the entry. Full context lives in the
session records and source repos, never here. This register is how the
conventions evolve: a change to CONTRACT.md cites the T-number that
authorized it.

| # | Decision |
|---|---|
| T1 | The repo is named `toolsmith`. "Chassis" stays the name for the code layer inside each tool (the conventions every verb inherits). |
| T2 | Tools share the contract, not a library. Chassis code is copied per tool; the written contract keeps copies aligned. No cross-repo version coupling. |
| T3 | Every conversion target becomes a Go binary. No lighter "asset pack" tier; prose-heavy tools carry their prose as assets next to a small binary. |
| T4 | The skeleton is a compiling Go module using `toolname` / `TOOLNAME` / `toolnameerr` as placeholders, instantiated by `contrib/new-tool.sh`. A compiling skeleton can be smoke-tested green; a text-template one rots silently. |
| T5 | `exitcode.Silent` is part of the chassis contract (C2.4). Ratified from ste9: a byte-parity verb cannot let the error renderer own its stderr. |
| T6 | Drift vocabulary is duo's six states — `current, missing, stale, modified, unowned_conflict, incompatible` (C4.5). wip's per-file added/removed/changed is the floor and maps in. |
| T7 | The manifest carries a self-committing `manifest_digest` (C3.4), adopted from duo: it makes the manifest a comparable identity. |
| T8 | The manifest declares its contract version (`"contract": "toolsmith/v1"`, C3.6), so conformance is read from the tool, not asserted from memory. |
| T9 | "Manifest" is reserved for the verb-surface self-description; the install-state file is the "stamp" (C3.5). ste9 used "manifest" for install state and proved the collision. |
| T10 | Splice targets are a ratified install-target class (C4.8): marker blocks for text, gjson/sjson surgical edits for JSON, one-time backup, byte-golden tests. From ste9's CLAUDE.md/settings.json work. |
| T11 | Emit-only targets are a ratified install-target class (C4.8): render for pasting, install nothing, stamp nothing. From ste9's desktop target. |
| T12 | Per-target variant flags are permitted where targets genuinely differ (C4.8), shaped `--<asset> <target>=<variant>,...`. From ste9's register variants. |
| T13 | A materialized data dir is a ratified artifact class (C4.9): write-side, stamped, drift-checked, distinct from the read-side asset chain. From ste9's greppable references. |
| T14 | Port specs live at `docs/<matter>/port-spec.md`, committed (C7.2). ste9's port spec stranded in an ephemeral scratchpad while shipped comments cited it. |
| T15 | `make hooks` installs both pre-commit stages (C6.6). wip shipped the one-stage gap; ste9 fixed it; the skeleton ships the fix. |
| T16 | GitHub Actions are SHA-pinned in the skeleton workflows (C6.5). From duo main. |
| T17 | Reusable (workflow_call) GitHub workflows are rejected for v1: cross-repo coupling cuts against T2. Revisit only if copy drift demonstrably hurts. |
| T18 | The skeleton ships exactly one worked harness target (claude-code). Additional targets follow assets/playbook/new-harness-target.md; six near-verbatim copies in wip showed shipping all of them teaches nothing extra. |
| T19 | Phase B — the `toolsmith` binary (`new`, `check`, `manifest`, `doctor`) with the skeleton as its embedded asset tree — is deferred until Phase A has run one real conversion (*amended: that conversion is toolsmith's own, tracked as the `toolsmith-binary` Matter; clast remains the acceptance test of intake, the orphan branch, and cutover, tracked as `clast-conversion`; neither Matter blocks the other*). The shell scripts are the parity oracle it must match before they retire, and only a self-conversion exercises that retirement — clast never touches them, and no clast source is recorded on any host. |
| T20 | Contract versioning: additive clauses bump the minor, breaking changes bump the major. Tools pin the version they conform to in their manifest (*amended by T24: the minor lives in CONTRACT.md's clause history, and the version a tool declares carries the major alone*). |
| T21 | The conformance checker reports findings by clause ID, flat, exhaustively, exit 0/1 — the same shape as a tool's `doctor` (C4.7). |
| T22 | `assets/handoff-kit/` states its model separately from its medium. `HANDOFF.md` and `SEED-CARDS.md` are always files in the sidecar; the Matter register and the workplans are model, and live in `workplans/<slug>.md` when the conversion has no tracker or in the tracker when it has one, with the sidecar naming which. The kit borrowed wip's vocabulary and wip's filesystem together — a conversion run without a tracker needs the directory, and one run with a tracker needs the tracker to be the single copy. |
| T23 | The manifest records each verb's positional-argument usage verbatim from its declaration (`usage`, C3.8), never parsed into structure. Found in toolsmith's own conversion: the first two verbs with a meaningful positional rendered in the generated claude-code skill as bare verb names, so an agent reading the projection could not tell that `new` requires one. The angle/bracket convention the declaration uses is enforced by nothing, so splitting it into fields would be inventing structure — C3.7's prohibition applied to arguments. |
| T24 | The contract's minor version lives in CONTRACT.md's clause history; the string a tool declares (C3.6) carries the major alone. Conformance is **derived** — the checker reports which clauses it audited and how they fared — rather than self-declared, and the checker's coverage is stated as partial rather than implied to be total. Measured, not assumed: nothing reads the version digits today — the oracle greps `"contract":"toolsmith/` and the port tests `strings.HasPrefix(m.Contract, "toolsmith/")`, so `toolsmith/banana` passes C3.6 as readily as `toolsmith/v1`. A declared minor would add digits no consumer reads while manufacturing the cross-repo version coupling T2 exists to prevent, and it would stay a self-report that can be wrong in both directions. Deriving instead is what T8 was actually reaching for: a self-declared version is still asserted from memory, just the tool author's. Precedent: `SchemaVersion` already answered this same question the same way, bumping only on incompatible change because every harness bakes it into its SKILL.md. |
| T25 | A tool records the contract minor it was last reconciled against, so the checker can name the clauses added since and call the reading pass stale. This is the one consumer a minor version has: it verifies that the human pass `migrate.md` Stage 8 mandates actually ran and is current, not that the tool conforms. Self-reported like any version, but it reports a process fact ("I read the contract at v1.1") rather than a semantic claim ("I conform"), so it is hard to get accidentally wrong and it fails as visible staleness rather than silent falsehood. |
