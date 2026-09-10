# Evidence — step-34: the parity corpus, captured as goldens

**Date:** 2026-09-10. **toolsmith:** `43478c2` (main, working tree). Every
proof below ran on the uncommitted worktree on top of this commit.
**Environment:** `nix develop`, go1.24.10, git 2.54.0, linux/amd64.

Matter `toolsmith-binary`, Stage 7, step-34: capture the parity corpus as
goldens before step-36 deletes the oracle and the gate. A test then
replaces the gate. This record proves the capture. The goldens match the
real oracle and the real port on the full corpus. Three deliberate breaks
each fail exactly the case they should touch.

The capture ran in two phases. Phase 1 proved the design at `5eadc84` and
captured `corpus-toolsmith` against it. Phase 2 (this record) moved onto
`43478c2`.

Commit `43478c2` split the root Makefile into `Makefile` and
`toolsmith.mk`. It converged `.pre-commit-config.yaml`, `cliff.toml`,
`flake.nix`, and `assets/assets.go` with the skeleton, and added
shellcheck to three skeleton files. Phase 2 re-captured `corpus-toolsmith`
at the new pin. It root-caused and fixed a test-harness flake found in
phase 1 (§10), and it re-ran every proof on the merged tree. §12 lists
what changed between phases.

## 1. What was captured, and where

The suite lives in the two verb packages it replaces the gate for. No
shared package exists.

- `internal/verbs/check/golden_test.go`, `oracle_test.go`,
  `testdata/corpus/{wip,duo,ste9,toolsmith}/`,
  `testdata/golden/check/*.{stdout,stderr}` (11 cases, 22 files).
- `internal/verbs/new/golden_test.go`, `oracle_test.go`,
  `testdata/golden/new/*.stdout` (3 cases).

Corpus pins (unchanged from `contrib/parity-check`):

| repo | pin | branch (hint only) |
|---|---|---|
| wip | `5be050755492c5b6d0a8767c9f071ce971fd8dff` | go |
| duo | `7763ef1e4bca70bf99fac6de64eda056cef4b541` | go |
| ste9 | `2185ca443d2373fdea0acdf333f28655690dd7e2` | go-port |
| toolsmith | `43478c24829c2e3f7b968dc0c6f30a7a051f8b37` | main (this capture's own HEAD, phase 2) |

Each corpus fixture is a projection, not a full checkout. It holds three
kinds of paths.

The first kind is verbatim files. `Audit` and `contrib/check-contract`
read these byte for byte from the pinned commit: `Makefile`, `flake.nix`,
`.envrc`, `.golangci.yml`, `cliff.toml`, `.pre-commit-config.yaml`, and
`.github/workflows/{ci,release}.yml`.

The second kind is existence-only placeholders: `go.mod`, `README.md`,
`contrib/check-commit-msg`, each `cmd/<x>/`, and `assets/` when present.
The third kind is a recorded manifest response. This is the real
`go run ./cmd/<tool> manifest --json` stdout and exit status, captured
once from a `git archive` extraction with the real toolchain.

The capture stores dotfile and dot-directory path segments as `dot-X`,
and restores them to `.X` at test time. A checked-in `.envrc` would
otherwise reach this repo's own pre-commit hooks. `.pre-commit-config.yaml`
selects shellcheck's target files by `types: [shell]`, and a bare
`.envrc` counts as shell. Wip's pinned `.envrc` carries no shellcheck
directive and would fail that hook if committed as itself.

`internal/verbs/check/audit.go` and `contrib/check-contract` were read
side by side to confirm this file list is complete. It is. Every other
check in both implementations gates on existence only (`isFile`/
`has_file`) or a directory listing (`cmd/`), never on a second file's
contents.

### Reproduction

One throw-away shell function, `cap`, captured the corpus, run once per
repo. It never touched a working tree. Every read went through
`git --git-dir=<repo>/.git`, so the four repos' own checked-out branches
never mattered.

```sh
V=(Makefile flake.nix .envrc .golangci.yml cliff.toml .pre-commit-config.yaml \
   .github/workflows/ci.yml .github/workflows/release.yml)
E=(go.mod README.md contrib/check-commit-msg)

# dot-file storage: a leading "." on any path segment becomes "dot-".
st() { printf '%s' "$1" | sed -E 's#(^|/)\.#\1dot-#g'; }

cap() {
  local n="$1" gitdir="$2" pin="$3" tool="$4" o="$C/$n"
  mkdir -p "$o/repo"

  for f in "${V[@]}"; do
    [ "$(git --git-dir="$gitdir" cat-file -t "$pin:$f" 2>/dev/null)" = blob ] || continue
    mkdir -p "$o/repo/$(dirname "$(st "$f")")"
    git --git-dir="$gitdir" show "$pin:$f" >"$o/repo/$(st "$f")"
  done
  for f in "${E[@]}"; do
    [ "$(git --git-dir="$gitdir" cat-file -t "$pin:$f" 2>/dev/null)" = blob ] || continue
    mkdir -p "$o/repo/$(dirname "$f")"; : >"$o/repo/$f"
  done
  for d in $(git --git-dir="$gitdir" ls-tree -d --name-only "$pin" cmd/); do
    mkdir -p "$o/repo/$d"; : >"$o/repo/$d/PLACEHOLDER"
  done
  [ "$(git --git-dir="$gitdir" cat-file -t "$pin:assets" 2>/dev/null)" = tree ] &&
    { mkdir -p "$o/repo/assets"; : >"$o/repo/assets/PLACEHOLDER"; }

  git --git-dir="$gitdir" archive "$pin" | tar -x -C "$FULL/$n"
  (cd "$FULL/$n" && CGO_ENABLED=0 go run "./cmd/$tool" manifest --json \
    >"$o/manifest.stdout" 2>/dev/null)
  echo $? >"$o/manifest.exit"
}

cap wip       <wip repo>/.git  5be050755492c5b6d0a8767c9f071ce971fd8dff  wip
cap duo       <duo repo>/.git  7763ef1e4bca70bf99fac6de64eda056cef4b541  duo
cap ste9      <ste9 repo>/.git 2185ca443d2373fdea0acdf333f28655690dd7e2  ste9
cap toolsmith <this repo>/.git 43478c24829c2e3f7b968dc0c6f30a7a051f8b37  toolsmith
```

Every projected path is a regular blob or a tree at all four pins, never a
symlink. `git ls-tree <pin> -- <path>` was run for every entry in `V`,
`E`, `cmd/<x>`, and `assets/`, at all four pins. Every mode is `100644`,
`100755`, or `040000`. None is `120000`.

## 2. `make parity`

Re-run on the merged tree. `make parity` now resolves through
`toolsmith.mk`'s `include`, unchanged in substance from the pre-merge
target. No verb code changed in this step, only test code and fixtures.

```
passed: 52
failed: 0
parity: all cases passed
```

## 3. `TestOracle*`, with the env var

Both packages' oracle suites run every case through the real oracle
script and the same fake `go`. Each result must equal the port's golden.

Two named divergences are the exception. `probe-crlf` and
`probe-prettyjson` log the oracle's output instead of asserting it, so a
stopped divergence (D1, D2) still fails the test.

Without the env var, both suites SKIP with a message naming the
variable. `TestOracleNew/with-git` and `TestGoldenNew/with-git` each ran
40 times over, with the env var set (§10).

```
TOOLSMITH_GOLDEN_ORACLE=1 go test ./internal/verbs/check/... -run TestOracle -count=1 -v
--- PASS: TestOracleCheck (0.39s)
    (all 11 corpus/probe cases, plus the cli subtest)

TOOLSMITH_GOLDEN_ORACLE=1 go test ./internal/verbs/new/... -run TestOracle -count=1 -v
--- PASS: TestOracleNew (0.52s)
    (all 3 cases)
--- PASS: TestOracleNewCLI (0.02s)
```

## 4. Full-corpus diff, both implementations, all four repos

The strongest proof available. `git archive` extracted each corpus repo
in full, not the projected fixture. This run re-took toolsmith's own
extraction at `43478c2`. Both the real `contrib/check-contract` and the
real `bin/toolsmith check` audited each extraction, and a byte-for-byte
diff (repo path normalized to `$REPO`) compared the result against the
captured golden.

```
ok  - oracle/wip stdout matches golden
ok  - oracle/wip stderr matches golden
ok  - port/wip stdout matches golden
ok  - port/wip stderr matches golden
ok  - oracle/duo stdout matches golden
ok  - oracle/duo stderr matches golden
ok  - port/duo stdout matches golden
ok  - port/duo stderr matches golden
ok  - oracle/ste9 stdout matches golden
ok  - oracle/ste9 stderr matches golden
ok  - port/ste9 stdout matches golden
ok  - port/ste9 stderr matches golden
ok  - oracle/toolsmith stdout matches golden
ok  - oracle/toolsmith stderr matches golden
ok  - port/toolsmith stdout matches golden
ok  - port/toolsmith stderr matches golden

pass=16 fail=0
```

Baseline check, `grep -c '^C'` against the golden stdout
(`evidence/2026-09-10-conformance.md`'s recorded counts):

```
wip: 14
duo: 11
```

Both match. §1 shows the projected fixture is faithful to the full
extraction by construction. This run confirms the golden itself is
faithful to what both implementations do on the real, unprojected repos.

`bin/toolsmith check .` against the merged tree:

```
no findings — mechanical clauses hold for <worktree path>
exit=0
```

Clean. Stage 8's seal condition requires this.

## 5. Deliberate breaks

Run in phase 1, against `audit.go` and `instantiate.go`. Neither file
changed in the phase 2 merge, except one comment in `instantiate.go`.
Phase 2 did not re-run these breaks, and the result stands.

Each break was made, the named test run, the failure excerpt captured,
then reverted. `git diff --stat` on the touched file was empty again
before the next break.

**Finding-message break.** `audit.go`: `"%s: action not SHA-pinned: %s"`
became `"%s: action is not SHA-pinned: %s"`. `go test
./internal/verbs/check/... -run TestGoldenCheck -count=1` failed on
`corpus-wip`, `corpus-duo`, and `corpus-ste9`. Those three corpus cases
carry that finding.

```
--- FAIL: TestGoldenCheck/corpus-wip
--- FAIL: TestGoldenCheck/corpus-duo
--- FAIL: TestGoldenCheck/corpus-ste9
    check/corpus-ste9.stdout mismatch:
    --- want (golden) ---
    ...action not SHA-pinned: ...
    --- got ---
    ...action is not SHA-pinned: ...
```

Reverted. `go test ./internal/verbs/check/... -run TestGoldenCheck
-count=1` passed again.

**Order break.** `instantiate.go`'s `substitute`: the bare-name
replacement moved before the module-path replacement. `go test
./internal/verbs/new/... -run TestGoldenNew -count=1 -v` failed
`custom-module` on both the model comparison and the anchor check.
`default` and `with-git` passed.

```
--- FAIL: TestGoldenNew/custom-module
    tree mismatch: content differs: [cmd/acme/main.go go.mod ... 23 files]
    go.mod line 1 = "module github.com/procrastivity/acme", want "module example.com/x/acme"
    wrong substitution order: ... still contains the stranded default module path
--- PASS: TestGoldenNew/default
--- PASS: TestGoldenNew/with-git
```

Reverted. The suite passed again.

**Exec break.** `instantiate.go`'s `destMode`: the shebang branch
returned `0o644` instead of `0o755`. `go test ./internal/verbs/new/...
-run TestGoldenNew -count=1 -v` failed all three cases on the executable
bit.

```
executable bit differs: [contrib/check-commit-msg (want exec=true got exec=false) contrib/check-gofumpt (want exec=true got exec=false)]
--- FAIL: TestGoldenNew/default
--- FAIL: TestGoldenNew/custom-module
--- FAIL: TestGoldenNew/with-git
```

Reverted. The suite passed again.

## 6. No-churn proof

Re-run on the merged tree, this time with `TOOLSMITH_GOLDEN_ORACLE=1`
set, so both the golden and the oracle suites exercised the model
comparison. This run appended a line to `assets/_skeleton/LICENSE`, and
created `assets/_skeleton/churn-probe.txt`. Neither reached a commit.
`TestGoldenNew`, `TestOracleNew`, and `TestOracleNewCLI` all build their
model tree (`expectedTree`) from the live skeleton on disk, not a
snapshot. An edit to the skeleton should change both sides of the
comparison together, and not fail:

```
TOOLSMITH_GOLDEN_ORACLE=1 go test ./internal/verbs/check/... ./internal/verbs/new/... -count=1
ok  	github.com/procrastivity/toolsmith/internal/verbs/check	1.305s
ok  	github.com/procrastivity/toolsmith/internal/verbs/new	1.262s
```

Both files went back to their prior state. `git status --short` and
`git diff --stat` against the touched files were empty afterward.

## 7. `go test ./drift`

The merged commit (`43478c2`) added a second drift test,
`TestRootChassisMatchesSkeleton`, for the root chassis files that
converged with the skeleton in the same commit. Both tests pass.

```
--- PASS: TestChassisMatchesSkeleton
--- PASS: TestRootChassisMatchesSkeleton
ok  	github.com/procrastivity/toolsmith/drift	0.021s
```

Neither test needs an exemption. The new testdata lives under the two
verb packages' own `testdata/` trees. Neither drift test's walk reaches
there.

## 8. `pre-commit`, and `git status` at the end

`pre-commit run --files <every new file>` passed every hook and changed
nothing, in both phases. Phase 2 ran it against 88 files: the four Go
test files, the corpus fixtures, the golden files, and this evidence file
(87 before it existed).

```
trim trailing whitespace.................................................Passed
fix end of files.........................................................Passed
check yaml...............................................................Passed
check for merge conflicts................................................Passed
shellcheck...........................................(no files to check)Skipped
gofumpt -l...............................................................Passed
golangci-lint run........................................................Passed
```

Final `git status --short` (phase 2, on top of `43478c2`):

```
?? evidence/2026-09-10-parity-goldens.md
?? internal/verbs/check/golden_test.go
?? internal/verbs/check/oracle_test.go
?? internal/verbs/check/testdata/
?? internal/verbs/new/golden_test.go
?? internal/verbs/new/oracle_test.go
?? internal/verbs/new/testdata/
```

No tracked file carries a change. Every new file is new.

## 9. `make check`

Re-run on the merged tree. `make check` now expands `lint`, `test`, and
`toolsmith.mk`'s `smoke`, through `Makefile`'s `include`.

```
exit=0
```

Unchanged in shape from the Stage 4 record
(`evidence/2026-09-10-parity-green.md`): shellcheck, `golangci-lint run`,
`go test ./...`, then `make smoke`. `make parity` stays outside it, for
the reason it always has. The corpus lives on this host only.

## 10. A finding, root-caused: git's auto-maintenance

`t.TempDir()`'s own cleanup intermittently failed on the `with-git` case,
with `unlinkat .../.git/objects: directory not empty`. Phase 1 saw this
and worked around it with a retrying cleanup helper (`tempParent`),
without understanding it. Phase 2 root-caused it and replaced the
workaround with a fix.

A standalone Go program, outside `go test`, drove `contrib/new-tool.sh
gitdemo` from a fresh temp parent with a chosen
`GIT_CONFIG_COUNT`/`_KEY_N`/`_VALUE_N` set. It then called one
non-retrying `os.RemoveAll` right after `git status` returned, the same
shape as `t.TempDir()`'s own cleanup. 40 trials per configuration:

| config under test | failures / 40 |
|---|---|
| no extra git config at all | 13 |
| `core.fsmonitor=false` + `core.untrackedCache=false` only (phase 1's fix) | 13 |
| + `gc.auto=0` only | 13 |
| `maintenance.auto=false` only | 0 |
| `gc.autoDetach=false` only | 0 |

The cause is git's own post-command auto-maintenance. It is not
`fsmonitor`, and not classic loose-object-count auto-gc. `gc.auto`'s
default threshold is 6700, and the `with-git` repo carries about 85
loose objects. `git commit` triggers `git maintenance run --auto` by
default (`maintenance.auto`). That run detaches into the background by
default too (`gc.autoDetach`). The detached process can still be writing
to `.git/` after the parent `git commit` has already returned 0.

The fix sets `maintenance.auto=false` and `gc.autoDetach=false` in
`hermeticEnv`, in both packages, alongside `core.fsmonitor=false`,
`core.untrackedCache=false`, and `gc.auto=0` for further hygiene. This
step removes the `tempParent` helper. Both packages use plain
`t.TempDir()` again. `go test ./internal/verbs/new/... -run 'TestGoldenNew/with-git'
-count=40`, and the same for `TestOracleNew/with-git` with the env var,
each passed 40/40.

This was a test-harness hazard on this host's git version. It was never
a behavior of `toolsmith new` or `contrib/new-tool.sh`.

## 11. Coverage table (gate case → test)

| Gate case | Test | Note |
|---|---|---|
| parity/stderr/determinism for wip, duo, ste9 | kept: `corpus-*` goldens, 3 runs | input is a projection plus a recorded manifest |
| baseline wip 14 / duo 11 | covered by the golden | pinned in the golden and re-confirmed against the full extraction (§4) |
| smoke | kept: `skeleton-smoke`, live instantiation | tracks the skeleton; manifest canned |
| toolsmith (live working tree) | replaced by `corpus-toolsmith` snapshot | dropped: auditing the live repo each run; `toolsmith check .` stays a seal command |
| probes c62-mislabel, c62-hole, changelog-tracked, multicmd | kept: generated + golden | multicmd's manifest canned |
| port-only crlf, prettyjson | kept: golden + invariant | oracle side checked at capture only (§3) |
| cli: file arg, argc | port exit 2 kept | oracle exit parity checked at capture only |
| cli: zero args (D4) | port exit 0 kept | oracle exit 2 checked at capture only |
| clean-run stderr (no stderr except the multiple-cmd/ note) | kept: `cleanRunStderr` invariant in `TestGoldenCheck` | proven to catch a regression: a changed note wording still fails under `-update` |
| new: checklist+exit, no stderr, determinism | kept | |
| new: file set, bytes, exec vs oracle | replaced by model check (`expectedTree`) | proven == oracle at capture (§3) |
| new: branch, subject | kept (`assertGitFacts`) | |
| new: tree hash vs oracle | replaced by "status clean, one commit" + model check | oracle tree-hash comparison kept in `TestOracleNew` (capture-time only) |
| new-cli (5 cases) | port codes kept: `TestGoldenNewCLI` + existing `TestExitCodes` | oracle exit 1 checked at capture only |

Dropped outright:

- Real `go run` of each corpus tool's manifest verb. The invocation shape
  stays pinned by the fake-go log. No test compiles or runs a real tool
  binary.
- `verify_pin` and extraction as a live, re-runnable step.
- Oracle-vs-port comparison after cutover.
- Stderr text on CLI failure cases.
- Following the corpus repos past their pins.
- Audit inputs outside the projection. A future check that reads a new
  file needs a re-capture (§1's confirmation step).
- Live manifests. Nothing automated now runs `check` against toolsmith's
  own real manifest, or against the instantiated skeleton's real
  manifest. `make smoke` builds the smoke tool but does not check it.
  Nothing runs `toolsmith check .` automatically.
- Zero-args, in one sense only: the case moved from the live repo to the
  snapshot. `TestGoldenCheckCLI`'s zero-arguments case now audits a copy
  of `corpus-toolsmith`, not the working tree.

## 12. What changed between phases

1. **Placement.** Two external test packages, `internal/verbs/check` and
   `internal/verbs/new`. No shared package.

2. **Fake `go` on both sides of `TestOracle*`, not only the port's.**
   Neither implementation reads `cmd/<tool>`'s file contents, only its
   directory listing. No probe needs a real, buildable `main.go`. The
   golden and oracle suites share the same, unmodified corpus and probe
   setup functions.

3. **`GIT_CONFIG_COUNT`/`_KEY_N`/`_VALUE_N` in both packages'
   `hermeticEnv`**: `core.fsmonitor=false`, `core.untrackedCache=false`,
   `maintenance.auto=false`, `gc.auto=0`, `gc.autoDetach=false`. §10 has
   the cause. Phase 1 also added a retrying `tempParent` helper. Phase 2
   root-caused the race and removed it.

4. **No real Go source in the generated probes' `cmd/<x>/`
   directories.** A `PLACEHOLDER` file is enough, matching the corpus
   fixtures' own convention.

5. **Found and fixed: a `nil`-slice bug in two tracked helpers**.
   `check_test.go`'s and `new_test.go`'s `run` helpers called
   `cmd.SetArgs(args)` with a `nil` variadic slice on a zero-argument
   call. Cobra treats `SetArgs(nil)` as never called. It reads the real
   process's `os.Args[1:]` instead.

   An ordinary `go test` run never showed this, because nothing extra
   sat in `os.Args`. This suite's own `-update` flag changed that. `go
   test ./internal/verbs/check/... -update` left the literal token
   `-update` in `os.Args`. Cobra's fallback then tried to parse it as a
   `check` flag and failed. It exited 2 where `TestCheck_DefaultPath`
   expected 1.

   `new_test.go`'s `TestExitCodes` had already been passing its "no
   name" case for the wrong reason. Its `nil` args slice let the same
   fallback read the real `os.Args[1:]`. Under a plain `go test` run,
   that happened to be empty. The expected exit code came out right by
   coincidence, not because the test controlled its own arguments.

   Both helpers now call `cmd.SetArgs(append([]string{}, args...))`,
   which is never nil. `go test ./internal/verbs/check
   ./internal/verbs/new -count=1 -update` now passes, and the goldens
   stay byte-identical (checksummed before and after).

## 13. Test runtime

Phase 2, on `43478c2`:

```
go test ./internal/verbs/check/... -count=1   0.86s
go test ./internal/verbs/new/...   -count=1   0.69-1.02s
```

A few seconds combined, well inside `make check`'s existing `go test
./...` step. Unchanged in shape from phase 1's ~0.8s / ~0.7s.
