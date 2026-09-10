# Evidence — the parity gate's final run, kept at cutover

**Date:** 2026-09-10. **toolsmith:** `641dc79` (main) plus the step-35
comment-only sweep, uncommitted. **Environment:** `nix develop`
(`IN_NIX_SHELL=impure`), go1.24.10, git 2.54.0, linux/amd64.

## 1. What this record is

This is the parity gate's final run, kept at cutover per
`assets/playbook/parity-gate.md` §10 and C7.3. It is the durable answer
to "was the port faithful at the moment the oracles were deleted?" The
cutover commit deletes `contrib/check-contract`, `contrib/new-tool.sh`,
and `contrib/parity-check` in the same commit that adds this file.

## 2. The run at the point of deletion

The gate ran on the tree the cutover commit is built from: `641dc79`
plus the step-35 sweep, uncommitted. The sweep touched eleven files:

```
 docs/binary/parity-divergences.md      |  25 +++---
 docs/binary/port-spec.md               |  14 ++--
 drift/drift_test.go                    |  40 +++++-----
 internal/verbs/check/audit.go          | 134 ++++++++++++++++++---------------
 internal/verbs/check/check.go          |  53 +++++++------
 internal/verbs/check/golden_test.go    |   4 +-
 internal/verbs/new/golden_test.go      |   4 +-
 internal/verbs/new/instantiate.go      |  50 ++++++------
 internal/verbs/new/instantiate_test.go |   6 +-
 internal/verbs/new/new.go              |  44 +++++------
 internal/verbs/new/new_test.go         |   9 +--
 11 files changed, 199 insertions(+), 184 deletions(-)
```

That sweep changed comments, doc prose, and drift `reason:` strings
only. It touched no behavior. The gate output below is the proof: every
case still passes.

```
CGO_ENABLED=0 go build -ldflags "-X main.version=641dc79-dirty -X main.commit=641dc79 -X main.date=2026-09-10T23:27:10Z" -o bin/toolsmith ./cmd/toolsmith
contrib/parity-check
== resolving corpus pins ==
pin ok: wip @ 5be050755492c5b6d0a8767c9f071ce971fd8dff = 5be050755492c5b6d0a8767c9f071ce971fd8dff (matches pin 5be050755492c5b6d0a8767c9f071ce971fd8dff)
pin ok: duo @ 7763ef1e4bca70bf99fac6de64eda056cef4b541 = 7763ef1e4bca70bf99fac6de64eda056cef4b541 (matches pin 7763ef1e4bca70bf99fac6de64eda056cef4b541)
pin ok: ste9 @ 2185ca443d2373fdea0acdf333f28655690dd7e2 = 2185ca443d2373fdea0acdf333f28655690dd7e2 (matches pin 2185ca443d2373fdea0acdf333f28655690dd7e2)
toolsmith: working tree, /home/dev/Code/toolsmith (never extracted, per the corpus table)

== extracting pinned corpus ==

== regenerating smoke corpus (make smoke) ==

== corpus parity ==
  ok  - parity[wip]: stdout+exit byte-identical
  ok  - determinism[wip]: three port runs identical
  ok  - baseline[wip]: 14 findings (matches evidence/2026-09-10-conformance.md)
  ok  - parity[duo]: stdout+exit byte-identical
  ok  - determinism[duo]: three port runs identical
  ok  - baseline[duo]: 11 findings (matches evidence/2026-09-10-conformance.md)
  ok  - parity[ste9]: stdout+exit byte-identical
  ok  - determinism[ste9]: three port runs identical
  ok  - parity[smoke]: stdout+exit byte-identical
  ok  - stderr[smoke]: clean run, no stderr
  ok  - determinism[smoke]: three port runs identical
  ok  - parity[toolsmith]: stdout+exit byte-identical
  ok  - stderr[toolsmith]: clean run, no stderr
  ok  - determinism[toolsmith]: three port runs identical

== generated probes: parity (port spec §10) ==
  ok  - parity[probe-c62-mislabel]: stdout+exit byte-identical
  ok  - determinism[probe-c62-mislabel]: three port runs identical
  ok  - parity[probe-c62-hole]: stdout+exit byte-identical
  ok  - determinism[probe-c62-hole]: three port runs identical
  ok  - parity[probe-changelog-tracked]: stdout+exit byte-identical
  ok  - determinism[probe-changelog-tracked]: three port runs identical
  ok  - parity[probe-multicmd]: stdout+exit byte-identical
  ok  - stderr[probe-multicmd]: clean run, only the §9.4 multiple-cmd/-entries note
  ok  - determinism[probe-multicmd]: three port runs identical

== generated probes: port-only (port spec §9.1, §9.7) ==
  ok  - port-only[crlf]: CRLF-terminated, correctly-pinned action recognized as pinned (§9.7 exclusion)
  info - oracle on the same CRLF probe (expected to diverge per §9.7, not asserted):
         C1.2: no cmd/<tool> main package found
         C1.1: no Makefile
         C1.6: no flake.nix
         C1.6: no .envrc
         C2.1: no .golangci.yml
         C3.1: cannot run the manifest verb (need go, go.mod, and cmd/<tool>)
         C6.3: no cliff.toml (changelog is not derivable from tags)
         C6.5: ci.yml: action not SHA-pinned: - uses: actions/checkout@0123456789abcdef0123456789abcdef01234567
         C6.5: no .github/workflows/release.yml
         C6.6: no contrib/check-commit-msg hook
         C6.6: no .pre-commit-config.yaml
         C7.5: no README.md
  ok  - port-only[prettyjson]: pretty-printed manifest fields parsed correctly (§9.1 exclusion)
  info - oracle on the same pretty-printed probe (expected to diverge per §9.1, not asserted):
         C1.1: no Makefile
         C1.6: no flake.nix
         C1.6: no .envrc
         C2.1: no .golangci.yml
         C3.4: manifest --json carries no manifest_digest
         C3.6: manifest --json declares no toolsmith contract version
         C6.3: no cliff.toml (changelog is not derivable from tags)
         C6.5: no .github/workflows/ci.yml
         C6.5: no .github/workflows/release.yml
         C6.6: no contrib/check-commit-msg hook
         C6.6: no .pre-commit-config.yaml
         C7.5: no README.md

== CLI contract (parity-gate §4) ==
  ok  - cli: non-directory argument, both exit 2 (port spec §8.1 — exit code under parity, message text is not)
  ok  - cli: wrong argument count (2 positionals), both exit 2
  ok  - cli: zero arguments — oracle exit 2, port defaults to '.' and exits 0 (documented divergence, port spec §1 judgment call 2)

== new: tree parity (port spec §9.2) ==
  ok  - new[default]: checklist bytes + exit identical
  ok  - new[default]: clean run, no stderr
  ok  - new[default]: identical file set (52 files)
  ok  - new[default]: every file byte-identical
  ok  - new[default]: identical executable bits (2 executable)
  ok  - new[default]: three port runs identical
  ok  - new[custom-module]: checklist bytes + exit identical
  ok  - new[custom-module]: clean run, no stderr
  ok  - new[custom-module]: identical file set (52 files)
  ok  - new[custom-module]: every file byte-identical
  ok  - new[custom-module]: identical executable bits (2 executable)
  ok  - new[custom-module]: three port runs identical
  ok  - new[with-git]: checklist bytes + exit identical
  ok  - new[with-git]: clean run, no stderr
  ok  - new[with-git]: identical file set (52 files)
  ok  - new[with-git]: every file byte-identical
  ok  - new[with-git]: identical executable bits (2 executable)
  ok  - new[with-git]: same branch, commit message, and committed tree hash
  ok  - new[with-git]: three port runs identical

== new: CLI contract (port spec §9.3) ==
  ok  - new-cli[unknown flag]: oracle exits 1, port exits 2 (port spec §9.3, exit codes excluded from parity)
  ok  - new-cli[bad name shape]: oracle exits 1, port exits 1 (port spec §9.3, exit codes excluded from parity)
  ok  - new-cli[the placeholder name]: oracle exits 1, port exits 1 (port spec §9.3, exit codes excluded from parity)
  ok  - new-cli[no name]: oracle exits 1, port exits 2 (port spec §9.3, exit codes excluded from parity)
  ok  - new-cli[existing target]: oracle exits 1, port exits 3 as a refusal (port spec §9.3)

== parity gate summary ==
passed: 52
failed: 0
parity: all cases passed

real	0m10.589s
user	0m12.172s
sys	0m6.981s
exit=0
```

## 3. The clean-HEAD run

The same gate ran first this session on the committed tree at `641dc79`,
before the step-35 sweep began. It is the baseline for §2:

```
passed: 52
failed: 0
parity: all cases passed
```

## 4. What replaced the gate

`evidence/2026-09-10-parity-goldens.md` §11 is the coverage table:
every case above maps to a kept, replaced, or dropped test. After this
commit, no automated oracle-vs-port comparison exists — the verbs' own
golden tests are the specification.

## 5. Reproduction

The gate itself is gone from this tree. To re-run it, check out the
cutover commit's parent, `641dc79`:

```
git worktree add /tmp/toolsmith-641dc79 641dc79
cd /tmp/toolsmith-641dc79
make parity
```

`make parity` needs the corpus repos `~/Code/wip`, `~/Code/duo`, and
`~/Code/ste9`. Each must contain the commit that `contrib/parity-check`'s
`PIN_WIP`, `PIN_DUO`, or `PIN_STE9` names at that revision. The gate
extracts each pin with `git archive`, so the checked-out branch does not
matter. The corpus lives on this host only, and the gate never ran in
CI.
