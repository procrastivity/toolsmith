# Evidence — Stage 7 adjudicated run: `toolsmith check` for real

**Date:** 2026-09-10. **toolsmith (checker under test):** `7b69b4a` (main).
**Oracle:** `/home/dev/Code/toolsmith/contrib/check-contract` at the same
commit. **Environment:** linux/amd64, go toolchain from `PATH` (not `nix
develop`), binary built once at the start of this run.

This is not another parity-corpus run. `evidence/2026-09-10-conformance.md`
and `evidence/2026-09-10-parity-green.md` already show the port matching
its oracle byte-for-byte. What neither of those records can show is
whether that agreement is *correct* — the parity corpus takes its expected
findings from the oracle, so a defect the oracle shares is invisible to
it. This run re-derives every verdict from `CONTRACT.md`'s `[check]`
sub-parts and the repo under test, not from either tool's output, and
specifically hunts for clauses that should have fired and didn't.

The format follows ste9's first-run report on its own linter. That report
served as a model only, and its findings belong to ste9.

---

## 0. Targets and reproduction

Four clones under this session's scratch dir. Each keeps `.git`, because
C6.3's `changelogTracked` reads git state:

| target | source | branch | HEAD (verified) |
|---|---|---|---|
| wip | `/home/dev/Code/wip` | `go` | `5be050755492c5b6d0a8767c9f071ce971fd8dff` |
| duo | `/home/dev/Code/duo` | `go` | `7763ef1e4bca70bf99fac6de64eda056cef4b541` |
| ste9 | `/home/dev/Code/ste9` | `go-port` | `5b9d1022bb0aeed5d67c193c01a4cea2d4884f98` |
| toolsmith | `/home/dev/Code/toolsmith` | `main` | `7b69b4a419296be435304f2fc9f1e44931f2cab5` |

wip and duo match the parity gate's pins exactly. ste9's `go-port` has
moved from `2185ca4` (the gate's pin) to `5b9d1022b` — **new ground**: no
prior recorded baseline exists for this SHA, unlike wip/duo whose findings
are already recorded in `evidence/2026-09-10-conformance.md`.

Reproduction:
```sh
S=<scratch>/step-33
go build -o $S/toolsmith ./cmd/toolsmith        # from /home/dev/Code/toolsmith @ 7b69b4a
git clone --branch go      --single-branch /home/dev/Code/wip  $S/wip
git clone --branch go      --single-branch /home/dev/Code/duo  $S/duo
git clone --branch go-port --single-branch /home/dev/Code/ste9 $S/ste9
git clone /home/dev/Code/toolsmith $S/toolsmith-repo

$S/toolsmith check $S/<target>
/home/dev/Code/toolsmith/contrib/check-contract $S/<target>
```

Both commands were run twice more per target to confirm determinism
(stdout/stderr/exit identical across runs on all four). No `make` target
and no `tmp/smoke` were touched — the binary above is this run's only
build artifact, built once.

---

## 1. Per-target results

### 1.1 wip @ `5be0507`

**Port** (`exit=1`, 14 findings):
```
C3.4: manifest --json carries no manifest_digest
C3.6: manifest --json declares no toolsmith contract version
C6.5: ci.yml: action not SHA-pinned: - uses: actions/checkout@v4        (x4, one per CI job)
C6.5: ci.yml: action not SHA-pinned: - uses: DeterminateSystems/nix-installer-action@v16  (x4)
C6.5: release.yml: action not SHA-pinned: - uses: actions/checkout@v4
C6.5: release.yml: action not SHA-pinned: - uses: DeterminateSystems/nix-installer-action@v16
C6.6: make hooks does not install the commit-msg stage
C7.5: no README.md
14 finding(s) for <wip clone>
```
**Oracle**: byte-identical (diffed programmatically — 0 lines differ in
stdout, stderr, or exit). Matches the recorded `evidence/2026-09-10-conformance.md`
baseline (same 14 lines, same 5 distinct conditions).

**Verdict table**

| finding | verdict | cause | evidence |
|---|---|---|---|
| C3.4 no manifest_digest | **true** | — | `go run ./cmd/wip manifest --json` \| grep for `"manifest_digest"` and `"contract"`: neither key appears in the document at all |
| C3.6 no contract version | **true** | — | same run, same absence |
| C6.5 x10 (checkout@v4, nix-installer-action@v16, ci.yml x4 jobs + release.yml) | **true**, all 10 | — | `grep -n uses: .github/workflows/{ci,release}.yml` shows exactly 10 `uses:` lines, all `@v4`/`@v16` tags, none a 40-hex SHA |
| C6.6 make hooks missing commit-msg | **true** | — | `Makefile:25-26` is `hooks:\n\tpre-commit install` with no `--hook-type` at all; `--hook-type commit-msg` appears nowhere in the file |
| C7.5 no README.md | **true** | — | `ls wip/README.md` → no such file |

No false positives. Clean clauses double-checked by reading and all
genuinely hold: `CGO_ENABLED=0` in both Makefile and `flake.nix`;
`flake.nix`+`.envrc` (`use flake`) present and `assets/`→`share/`
postInstall present; `.golangci.yml` has `forbidigo` and the literal
`fmt\.Print` ban string, schema `version: "2"` and `default: none`;
`Makefile` and `cliff.toml` both present and both match `--match
'v[0-9]*'` / `tag_pattern = "v[0-9]*"` (C6.2 holds, not just "no C6.2
finding" — the else-branch mislabel of port-spec §9.6 does not trigger
here because both files exist); `CHANGELOG.md` not tracked;
`contrib/check-commit-msg` present, `.pre-commit-config.yaml` contains
`commit-msg`; single `cmd/wip` entry (no multiple-cmd/ note); manifest's
`schemaVersion` is `1` (no C3.1).

**False negatives found: none.**

### 1.2 duo @ `7763ef1`

**Port** (`exit=1`, 11 findings), **oracle**: byte-identical. Matches
`evidence/2026-09-10-conformance.md`'s recorded 11.
```
C3.1: `duo manifest --json` failed or is not implemented
C6.5: [10 unpinned-action lines, ci.yml x4 jobs + release.yml]
11 finding(s)
```

| finding | verdict | cause | evidence |
|---|---|---|---|
| C3.1 manifest failed | **true** | — | `go run ./cmd/duo manifest --json` → `duo: unknown flag: --json`, exit status 2. duo exposes `--output json` instead of the contract's global `--json` (C2.3) — a real surface divergence, already called out in the recorded evidence file |
| C6.5 x10 | **true**, all 10 | — | same method as wip: 10 `uses:` lines, all tag refs, none 40-hex |

Clean clauses re-verified by reading: `CGO_ENABLED=0` in Makefile and
flake.nix; flake/.envrc/share postInstall all present and correct;
`.golangci.yml` forbidigo+schema both correct; `Makefile`+`cliff.toml`
both present and both match (C6.2 genuinely holds); CHANGELOG.md not
tracked; `contrib/check-commit-msg` present, pre-commit config has
`commit-msg`, **and** `Makefile:26` actually contains `--hook-type
commit-msg` in the `hooks:` recipe (duo's `go` branch already carries the
T16/C6.6 fix that wip's `go` branch lacks — a real, not merely
unreported, difference between the two repos); README.md present.

One thing worth naming even though it produces no misverdict here: per
`evidence/2026-09-10-conformance.md`'s own note, the C3.1 failure masks
C3.4 and C3.6 entirely — `manifestVerb` (`audit.go:220-231`) returns after
the single C3.1 finding without ever reading the document, so whether duo
carries a real `manifest_digest` or contract string goes unasked. This is
a shared design limit (the oracle's `2>/dev/null`-guarded `go run ... ||
find_it` idiom has the identical fold-to-one-finding behavior), not new,
and not misleading on its own terms — but see §3 for why it's worth
flagging as a class of risk.

**False negatives found: none.**

### 1.3 ste9 @ `5b9d102` (new ground — past the gate's pin)

**Port** (`exit=1`, 10 findings), **oracle**: byte-identical. No prior
recorded baseline exists for this SHA; this run establishes one.
```
C3.1: `ste9 manifest --json` failed or is not implemented
C6.5: [8 unpinned-action lines, ci.yml x4 jobs]
C6.5: no .github/workflows/release.yml
10 finding(s)
```

| finding | verdict | cause | evidence |
|---|---|---|---|
| C3.1 manifest failed | **true** | — | `ste9 --help` lists `completion, help, install, lint, status, uninstall, version` — **there is no `manifest` command at all**, not a flag mismatch like duo's. `go run ./cmd/ste9 manifest --json` → `unknown command "manifest" for "ste9"`, exit 2 |
| C6.5 x8 (checkout@v4, nix-installer-action@v16, x4 CI jobs) | **true** | — | `grep uses: .github/workflows/ci.yml`: 8 lines, all tag refs |
| C6.5 no release.yml | **true** | — | `ls ste9/.github/workflows/` → only `ci.yml` exists |

Clean clauses re-verified: `CGO_ENABLED=0` in Makefile+flake.nix; flake
share postInstall present (note: ste9's postInstall uses `find ... -name
'*.go' -delete` rather than wip/duo's `rm -f assets/assets.go` — a
different mechanism, same effect, not audited beyond the `share/`
substring so this variance is invisible to the checker either way, which
is correct per C1.6's marked sub-part); `.golangci.yml` correct;
`Makefile`+`cliff.toml` both present and matching; CHANGELOG.md not
tracked; `contrib/check-commit-msg` present, pre-commit `commit-msg`
present, **and** `Makefile:26` has `--hook-type commit-msg` (ste9 also
carries the C6.6 fix); README.md present.

ste9 is mid-flight per `CONTRACT.md`'s own framing ("ste9 (in flight)")
and its manifest verb genuinely does not exist yet — this is a true
positive about ste9's current state, not a defect in the checker. The
`status`/no-`doctor` naming divergence visible in `--help` is outside the
13 audited clauses and not reported by either implementation; correctly
so, since neither C4 nor any `[check]` clause in the audited set governs
verb naming.

**False negatives found: none.**

### 1.4 toolsmith @ `7b69b4a` (self-check)

**Port**: `no findings — mechanical clauses hold for <path>`, `exit=0`.
**Oracle**: byte-identical, `exit=0`.

This is the highest-stakes target — it's the repo asking to be cut over
— so every clean clause was re-verified directly rather than trusted:

| clause | checked condition | result |
|---|---|---|
| C1.1 | `Makefile` has `CGO_ENABLED=0` (×3 call sites); `flake.nix:32` has `env.CGO_ENABLED = 0;` | holds |
| C1.2 | exactly one `cmd/` entry, `toolsmith` | holds |
| C1.6 | `flake.nix`, `.envrc` (`use flake`) present; `assets/` exists and `flake.nix` contains `share/` (three `$out/share/toolsmith/...` lines) | holds |
| C2.1 | `.golangci.yml` has `forbidigo` and the literal `fmt\.Print` ban | holds |
| C3.1/C3.4/C3.6 | `go run ./cmd/toolsmith manifest --json` succeeds; parsed document has `schemaVersion:1`, `contract:"toolsmith/v1"`, `manifest_digest:"sha256:cdaa63ec…"` | holds |
| C6.2 | `Makefile` has `--match 'v[0-9]*'`; `cliff.toml:44` has `tag_pattern = "v[0-9]*"` | holds |
| C6.3 | `CHANGELOG.md` not in `git ls-files` | holds |
| C6.5 | all 10 `uses:` lines (`ci.yml` ×4 jobs, `release.yml`) pin a 40-hex SHA with a `# vX.Y.Z` trailing comment; both files run through `nix develop --command` | holds |
| C6.6 | `contrib/check-commit-msg` present; `.pre-commit-config.yaml` has `commit-msg`; `Makefile` `hooks:` recipe has `--hook-type commit-msg` | holds |
| C6.7 | `.golangci.yml` has `version: "2"` and `default: none` | holds |
| C7.5 | `README.md` present | holds |

**Verdict: genuinely clean, not just clean-looking.** Every one of the 13
audited clauses' `[check]` sub-parts was independently confirmed against
the repo's actual bytes, not inferred from the checker's own silence.

**False negatives found: none.**

---

## 2. False-negative table (across all four targets)

| clause | target | what fails | why the checker missed it |
|---|---|---|---|
| — | — | *(no instance found)* | Every clause that produced no finding on any target was independently confirmed true by reading the repo (§1 tables above). Every clause that did produce a finding was independently confirmed to be a real condition, not an artifact. |

One real gap turned up. It is a coverage overclaim, not a misfire.
C1.1's `[check]` marker sat on "`CGO_ENABLED=0` in the Makefile, CI, and
the Nix package — all three". But `cgoEnabled` (mirroring
`contrib/check-contract:40-48`) reads only `Makefile` and `flake.nix`, and
it never opens a workflow file. The oracle has the identical hole, so
parity kept it and the parity gate could never see it.

No target exposes it today. All four set `CGO_ENABLED=0` in `ci.yml`,
confirmed with `git grep` against each pinned commit. A repo whose
Makefile and flake.nix are correct but whose CI drops the setting passes
C1.1 on both implementations.

The first draft of this record filed the gap as a documented boundary.
It cited CONTRACT.md's Conformance section, which named C1.1 as partly
checkable. That reading was wrong. The partial-check note meant that
C1.1's opening prose sentence is unchecked, while the marker itself
claimed all three places. Review corrected the verdict. The commit that
lands this record narrows the marker to the Makefile and the Nix package,
and tightens the Conformance wording so it cannot be read that way again.
Extending the check to CI changes `check`'s output, so it waits for
cutover and the goldens, like the C6.2 fix.

The step-32 marks gate did not catch this either. That gate compares
clause IDs, and C1.1 is correctly marked as audited. It cannot see how
much of a clause a marker claims.

No other candidate false negative survived a read of the corresponding
repo state. Candidates considered and ruled out: C1.2 not verifying
`cmd/<tool>/main.go` is an actual `package main` (all four targets build
successfully, so unexercised); C6.7 not checking the specific enabled
linters or the `gofumpt` formatter beyond `version: "2"`/`default: none`
(also named implicitly by the same "partly checkable" framing, and all
four `.golangci.yml` files in fact do carry the full list, so moot here
regardless); the C6.6 whole-file (not recipe-scoped) `--hook-type
commit-msg` search (port-spec §7, reproduced-as-measured) — on all three
non-clean targets the string's presence/absence matched the recipe body's
actual content, so the known scoping bug never changed a verdict.

---

## 3. Message quality

Findings read as actionable in every case observed this run: each names
the file (`ci.yml`, `release.yml`, `Makefile`, `.golangci.yml`,
`README.md`) and the specific missing/wrong content, and the C6.5 lines
quote the offending `uses:` line verbatim so a reader can find it with
one search.

One finding is accurate but incomplete in a way worth flagging for the
owner: **`C3.1: \`<tool> manifest --json\` failed or is not
implemented`** collapses two different real situations — duo's is a flag
mismatch (`--output json` vs `--json`), ste9's is a missing command
entirely — into one message, and per §1.2, it silently forecloses C3.4
and C3.6 for that repo (`checkManifestDoc`, `audit.go:276-291`, is never
reached). A reader who fixes the `--json` flag on duo and reruns will see
*new* findings appear (C3.4 and/or C3.6, if duo's document doesn't carry
those fields under its own vocabulary — see the C3.1 note in
`evidence/2026-09-10-conformance.md`) that a naive read of "1 finding,
now fixed" would not anticipate. This is not a new defect — the oracle
has the identical fold — but it is a real trap for whoever acts on the
message, and worth a one-line addition to the C3.1 message itself
("...; C3.4/C3.6 not evaluated") if this checker is ever revised post-
cutover.

---

## 4. Summary

**Defects found in this run: no false positives and no false negatives on
these four targets, and one coverage overclaim in the contract itself
(C1.1, §2).** Every finding on every target is a true positive. Every
silent clause on every target was independently confirmed to hold. The port matches its oracle byte-for-byte on
stdout, stderr, and exit code across all four targets, including
`ste9 @ 5b9d102`, which is past the parity gate's pin and had no prior
recorded baseline — this run's ste9 result is new evidence, not a
re-confirmation.

This is a materially different kind of clean than a parity-gate green:
the gate can only say "matches the oracle"; this run independently
re-derived each of the 13 audited clauses' truth value from
`CONTRACT.md` and the repo bytes, specifically to catch the class of bug
the gate structurally cannot see (ste9's own D1 — a false negative on
its own canonical example — is exactly that class, in a different tool).
None turned up here.

The one item worth the owner's attention is **the C1.1 overclaim (§2)**.
It does not block cutover. The oracle shares it, no target exposes it,
and the contract's marker now states what the checker actually reads. It
is dormant by luck rather than by design, so extending the check to CI is
scheduled after cutover.

No environment limits were hit — `go` was on `PATH`, all four `go.mod`
trees resolved without network access, and every manifest invocation
either succeeded (wip, toolsmith) or failed for an in-scope reason
(duo's flag mismatch, ste9's missing command) rather than an
infrastructure one.
