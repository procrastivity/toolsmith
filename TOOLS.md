# Fleet register

One row per tool in the family. "Contract" is what the tool's manifest
declares (C3.6) — `pre-v1` marks a tool built before the field existed.
Update this table when a tool converts, seals, or drifts.

Conformance runs live in [evidence/](evidence/); the latest audited wip
and duo on 2026-09-10.

| Tool | Repo | Language | Contract | Status |
|---|---|---|---|---|
| wip | `~/Code/wip` (branch `go`) | Go | pre-v1 | Reference implementation. The contract was extracted from it; convergence items live in [backport/wip.md](backport/wip.md). |
| duo | `~/Code/duo` (branch `go`) | Go | pre-v1 | Second conforming implementation. Contributed the digest, the six-state drift vocabulary, and the contracts-dir pattern. Install/uninstall verbs spec'd (`docs/vnext/duo-vnext-installation-contract.md`) but not yet built. Its manifest diverges: `--output json` rather than the global `--json`, with its own document vocabulary (`schema`, `product`, `operations`, `projectability`). |
| ste9 | `~/Code/ste9` | Go (mid-migration) | pre-v1 | Bespoke migration in flight (chassis + lint port done, install verbs next). Contributed C2.4 Silent, C4.8 splice/emit/variants, C4.9 data dir, C7.2 port-spec home. Retro folds into CONTRACT v1.x when it seals. |
| clast | — | bash (pre-conversion) | — | First planned dogfood of this playbook. Reads Claude Code session JSONL; exercises the lifecycle-hook story (C4.10). |
| toolsmith | this repo | docs + scripts | — | Phase B makes it a tool conforming to its own contract (T19). |
