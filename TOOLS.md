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
| duo | `~/Code/duo` (branch `go`) | Go | pre-v1 | Second conforming implementation. Contributed the digest, the six-state drift vocabulary, and the contracts-dir pattern. Install/uninstall verbs spec'd (`docs/vnext/duo-vnext-installation-contract.md`) but not yet built. Its manifest diverges: `--output json` rather than the global `--json`, with its own document vocabulary (`schema`, `product`, `operations`, `projectability`). |
| ste9 | `~/Code/ste9` | Go (mid-migration) | pre-v1 | Bespoke migration in flight (chassis + lint port done, install verbs next). Contributed C2.4 Silent, C4.8 splice/emit/variants, C4.9 data dir, C7.2 port-spec home. Retro folds into CONTRACT v1.x when it seals. |
| clast | — | bash (pre-conversion) | — | First *foreign* conversion, tracked as the `clast-conversion` Matter: the acceptance test of assets/playbook/intake.md, migrate Stage 2, and Stage 7. Reads Claude Code session JSONL; exercises the lifecycle-hook story (C4.10). Not started — no source is located on this host, so the owner still owes us where it lives and what shape it has. |
| toolsmith | this repo | Go | `toolsmith/v1` | Self-conversion, tracked as the `toolsmith-binary` Matter: the contract's first dogfood (T19). The Stage 7 adjudicated run found `toolsmith check` clean on this repo at `7b69b4a`, over the clauses that the checker audits (T24). Cutover deleted `contrib/check-contract`, `contrib/new-tool.sh`, and `contrib/parity-check`; the gate passed at the point of deletion ([evidence/2026-09-10-parity-final.md](evidence/2026-09-10-parity-final.md)). The Matter is not sealed. |
