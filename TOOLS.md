# Fleet register

One row per tool in the family. "Contract" is the string that the tool's
manifest declares (C3.6). Copy it from `<tool> manifest --json`, not from
memory. `pre-v1` marks a tool whose manifest declares no contract string.
The string names the major version only (T24). It is not a conformance
claim. Conformance comes from a checker run. Update this table when a tool
converts, seals, or drifts.

Conformance runs live in [evidence/](evidence/).
[The Stage 7 adjudicated run](evidence/2026-09-10-stage-7-adjudicated-run.md)
audited wip, duo, ste9, and toolsmith on 2026-09-10.

| Tool | Repo | Language | Contract | Status |
|---|---|---|---|---|
| wip | `~/Code/wip` (branch `go`) | Go | pre-v1 | Reference implementation. The contract was extracted from it; convergence items live in [backport/wip.md](backport/wip.md). |
| duo | `~/Code/duo` (branch `go`) | Go | pre-v1 | Second conforming implementation. Contributed the digest, the six-state drift vocabulary, and the contracts-dir pattern. Install/uninstall verbs spec'd (`docs/vnext/duo-vnext-installation-contract.md`) but not yet built. Its manifest diverges: `--output json` rather than the global `--json`, with its own document vocabulary (`schema`, `product`, `operations`, `projectability`). Convergence items live in [backport/duo.md](backport/duo.md). |
| ste9 | `~/Code/ste9` | Go | `toolsmith/v1` | Adoption, tracked as the `toolsmith-adoption` Matter: the bespoke Go migration has landed. Ships `install`/`uninstall`/`status`/`doctor`/`lint`/`version`/`manifest`; CI is a nix-based GitLab pipeline — every job inside `nix develop` (branch proof: pipeline 899647). `toolsmith check` at `c103760` reports two known-standing C6.5 findings — the checker only audits the GitHub workflow layout, and ste9's CI is `.gitlab-ci.yml` — whose GitLab-equivalence mapping is drafted as T37. `evidence/` and the reconstructed `docs/toolsmith-v1/port-spec.md` (C7.2/T14) are committed. Contributed C2.4 Silent, C4.8 splice/emit/variants, C4.9 data dir, C7.2 port-spec home. Pending drafts: T37 (GitLab-equivalence) and T38 (the C4.4 central-record-shape proposal) — DECISIONS.md ends at T36. Retro folds into CONTRACT v1.x when it seals. |
| clast | `~/Code/clast` (branch `main`) | Go | `toolsmith/v1` | First *foreign* conversion, tracked as the `clast-conversion` Matter. Converted from bash, evidence-only (no parity gate, no oracle — clast HANDOFF H4), planned in the `clast-reboot` sidecar. Cutover 2026-09-14: the Go line is `main`, the bash line is frozen at `bash-final`, and `toolsmith check` reported no findings on the tip (`9f81211`). Exercises the SessionStart-hook splice (C4.8) and a three-skill claude-code projection. npm returns as a release channel for the binary (clast BDS-207). |
| toolsmith | this repo | Go | `toolsmith/v1` | Self-conversion, tracked as the `toolsmith-binary` Matter: the contract's first dogfood (T19). The Stage 7 adjudicated run found `toolsmith check` clean on this repo at `7b69b4a`, over the clauses that the checker audits (T24). Cutover deleted `contrib/check-contract`, `contrib/new-tool.sh`, and `contrib/parity-check`; the gate passed at the point of deletion ([evidence/2026-09-10-parity-final.md](evidence/2026-09-10-parity-final.md)). The Matter is not sealed. |
