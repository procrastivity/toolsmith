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
| T18 | The skeleton ships exactly one worked harness target (claude-code). Additional targets follow playbook/new-harness-target.md; six near-verbatim copies in wip showed shipping all of them teaches nothing extra. |
| T19 | Phase B — the `toolsmith` binary (`new`, `check`, `manifest`, `doctor`) with the skeleton as its embedded asset tree — is deferred until Phase A has run one real conversion. The shell scripts are the parity oracle it must match before they retire. |
| T20 | Contract versioning: additive clauses bump the minor, breaking changes bump the major. Tools pin the version they conform to in their manifest. |
| T21 | The conformance checker reports findings by clause ID, flat, exhaustively, exit 0/1 — the same shape as a tool's `doctor` (C4.7). |
