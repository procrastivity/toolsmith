# Evidence — Stage 4 closes green: `make check`, the manifest, the gate

**Date:** 2026-09-10. **toolsmith:** `61d2d14` (main, clean tree).
**Environment:** `nix develop`, go 1.24.10, linux/amd64.

Stage 4 of `toolsmith-binary` ported both verbs against their shell
oracles. This record is Step 17's green run: the three things that have
to hold at once before the Stage seals, captured at one commit.

The parity gate is retired at cutover (Stage 7,
`assets/playbook/parity-gate.md` §10), and a final run is kept then. This
is not that run — it is the one that says Stage 4 finished, with both
oracles still installed.

---

## 1. `make check`

```
exit=0
```

Unchanged in shape from `evidence/2026-09-10-toolsmith-selfcheck.md`:
shellcheck over the five shell files, `go vet` and `go test` over the
tree, then `make smoke` instantiating `tmp/smoke` and building it. The
gate is deliberately **not** part of `make check` — the corpus lives on
this host only, so it can never run in CI, and both implementations shell
out to `go run` once per repo.

## 2. `toolsmith manifest --json`

Both ported verbs appear, correctly annotated, with their flags reflected
as args:

| verb | kind | args |
|---|---|---|
| `check` | `plumbing` | — |
| `doctor` | `plumbing` | — |
| `install` | `plumbing` | `force` |
| `manifest` | `plumbing` | — |
| `new` | `plumbing` | `dir`, `module`, `no-git` |
| `uninstall` | `plumbing` | — |
| `version` | `plumbing` | — |

```
contract: toolsmith/v1     schemaVersion: 1     assets: 32
manifest_digest: sha256:58cd5e11eab33110a32123a1a03a0e5b6b864a8f3d5b71ab67e3617af6ac0075
```

`plumbing` on both is the C3.2 call recorded in port spec §1: deterministic
output, no LLM shaping, so the manifest walk and the harness projection
see them correctly from the commit that registers them.

## 3. `toolsmith check .`

```
no findings — mechanical clauses hold for /home/dev/Code/toolsmith
exit=0
```

This is Stage 8's seal condition, met early and recorded here because it
is free to check. It is not the seal — Stage 8 also requires the oracle
deleted and the findings landed in `DECISIONS.md` and `CONTRACT.md`.

## 4. `make parity`

```
passed: 52
failed: 0
parity: all cases passed
```

| group | cases | what it asserts |
|---|---|---|
| corpus parity | 14 | stdout bytes plus exit code across wip, duo, ste9, smoke, toolsmith; determinism over three runs each; recorded baselines for wip (14 findings) and duo (11) |
| generated probes, parity | 9 | the C6.2 mislabel and coverage hole, the tracked-CHANGELOG branch, and the multiple-`cmd/`-entries repo — all four unreachable from the fixed corpus |
| generated probes, port-only | 2 | the §9.1 pretty-printed manifest and §9.7 CRLF exclusions, asserted on the port side; the oracle's divergent output printed as information |
| CLI contract, `check` | 3 | non-directory argument, wrong argument count, and the zero-argument divergence |
| `new`, tree parity | 19 | identical file sets (50 files), byte-identical contents, identical executable bits, identical checklist stdout, and — on the git case — the same branch, commit message and committed tree hash |
| CLI contract, `new` | 5 | the C2.4 exit-code divergence pinned on both sides |

Every deliberate difference between the two implementations is written up
in `docs/binary/parity-divergences.md` (Step 16). Six of its eight entries
are asserted by the gate above rather than merely recorded.

### The corpus pin drifted, and the gate caught it

The first run of this step aborted:

```
parity: corpus repo "ste9" (/home/dev/Code/ste9): ref "go-port" resolves
to 1957b5c..., pin expects 2185ca4
```

Nothing about either implementation had changed. ste9's branch had moved
one commit ahead. `verify_pin` refused to extract a corpus it could not
vouch for, which is `assets/playbook/parity-gate.md` §7 working — a bad
precondition cannot produce a meaningful case-by-case report, so it fails
the whole run rather than adding a row to it.

The precondition itself was wrong, though: the corpus was pinned by branch
name and checked against a SHA prefix, which holds only until someone
commits to an unrelated repo. Fixed at `61d2d14` — the pins are full
commit SHAs, and the branch name survives as a comment. The run above is
from after that fix.
