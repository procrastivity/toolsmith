# Evidence — the conformance run: `toolsmith check` on toolsmith

**Date:** 2026-09-11. **toolsmith:** `d94f4ee` (fresh clone). **Environment:**
`nix develop` (`IN_NIX_SHELL=impure`), go1.24.10, git 2.54.0, linux/x86_64.

## 1. What this record is

This is the conformance half of `assets/playbook/migrate.md` Stage 8's last
bullet ("write `evidence/` records for the parity gate's final run and the
conformance run") and of CONTRACT.md's C7.3 ("`evidence/` holds verification
records: parity runs, dogfood passes, conformance reports — committed
beside the code they verify"). The parity half is
`evidence/2026-09-10-parity-final.md`, kept at cutover (commit `725e94e`),
and this record does not repeat it.

This record evidences one part of the `toolsmith-binary` Matter's seal
(`toolsmith-reboot/HANDOFF.md` §3): "`toolsmith check .` clean on
toolsmith". The seal's other two parts already hold. The oracles and the
gate were deleted in one commit (`725e94e`), and the gate's final run is
in the parity-final record. This record makes the "clean" claim
checkable later, not only asserted.

## 2. Reproduction

```
git clone -q /home/dev/Code/toolsmith <scratch>/toolsmith
git -C <scratch>/toolsmith checkout -q d94f4ee
git -C <scratch>/toolsmith status --short   # empty

cd <scratch>/toolsmith
nix develop --command make check

CGO_ENABLED=0 go build -o <scratch>/bin/toolsmith ./cmd/toolsmith
<scratch>/bin/toolsmith version

mkdir -p <scratch>/xdg-empty
cd <scratch>/toolsmith
XDG_CONFIG_HOME=<scratch>/xdg-empty <scratch>/bin/toolsmith check <scratch>/toolsmith
XDG_CONFIG_HOME=<scratch>/xdg-empty <scratch>/bin/toolsmith check --json <scratch>/toolsmith
```

All of the above ran wrapped in `nix develop --command bash -c '...'` from
`<scratch>/toolsmith`. `go` and `git` are also reachable on this host's
plain `PATH` outside the dev shell (home-manager), but everything below was
run (and, where first tried outside the shell, re-run) inside `nix develop`
for fidelity to that instruction; both paths gave identical output
everywhere they were both tried.

The clone kept `.git` throughout — C6.3's `changelogTracked` check reads
git state (`git -C <repo> ls-files CHANGELOG.md`), so a `.git`-less copy
would silently skip that clause rather than exercise it.

## 3. `make check`

Exit code: `0`.

```
shellcheck .envrc contrib/check-commit-msg contrib/check-gofumpt
golangci-lint run
0 issues.
go test ./...
ok  	github.com/procrastivity/toolsmith/drift	0.016s
ok  	github.com/procrastivity/toolsmith/internal/cli	0.478s
ok  	github.com/procrastivity/toolsmith/internal/manifest	0.007s
ok  	github.com/procrastivity/toolsmith/internal/verbs/check	1.205s
ok  	github.com/procrastivity/toolsmith/internal/verbs/new	0.674s
(all other toolsmith packages: "[no test files]")
CGO_ENABLED=0 go build -ldflags "-X main.version=d94f4ee -X main.commit=d94f4ee -X main.date=2026-09-11T00:53:26Z" -o bin/toolsmith ./cmd/toolsmith
rm -rf tmp/smoke tmp/smoke-xdg-config
mkdir -p tmp/smoke-xdg-config
XDG_CONFIG_HOME=<clone>/tmp/smoke-xdg-config bin/toolsmith new smoke --dir tmp/smoke --no-git
instantiated smoke at tmp/smoke (module github.com/procrastivity/smoke)
cd tmp/smoke && CGO_ENABLED=0 go build ./... && go vet ./... && go test ./...
ok  	github.com/procrastivity/smoke/internal/cli	0.467s
ok  	github.com/procrastivity/smoke/internal/manifest	0.007s
(all other smoke packages: "[no test files]")
```

Lint is clean (`0 issues.`), every package that carries tests passes, and
the smoke step — instantiating a fresh tool from the skeleton and building
and testing it — completes without error. `make check` does not run
`toolsmith check` on anything; §4 and §7 do.

## 4. The self-check

Binary built once, no ldflags (per this record's own build step, distinct
from `make check`'s ldflags'd build above):

```
<scratch>/bin/toolsmith version
toolsmith version dev (commit unknown, built unknown)
```

Self-check, `XDG_CONFIG_HOME` pointed at an empty directory so no host
override (`$XDG_CONFIG_HOME/toolsmith/`) shadows the embedded assets:

```
$ XDG_CONFIG_HOME=<scratch>/xdg-empty <scratch>/bin/toolsmith check <scratch>/toolsmith
no findings — mechanical clauses hold for <scratch>/toolsmith
exit=0
```

stderr was empty. Run again with `--json`:

```
$ XDG_CONFIG_HOME=<scratch>/xdg-empty <scratch>/bin/toolsmith check --json <scratch>/toolsmith
no findings — mechanical clauses hold for <scratch>/toolsmith
exit=0
```

Byte-identical to the plain run. `internal/verbs/check/check.go` never
reads the global `--json` flag, so the verb writes the same text lines
either way. `new` does not read it either. The chassis verbs `manifest`,
`version`, `install`, `uninstall` and `doctor` do. C2.3
says `--json` emits the success payload as one JSON value, so both ported
verbs diverge from C2.3. The retired oracles had no JSON output, and the
port spec does not rule on it. This does not change the clean result
above. `docs/binary/decisions.md` records the divergence.

## 5. What the clean run covers

CONTRACT.md's Conformance section is explicit that a clean run means
"clean against the clauses I can reach", never "conforms to the contract".
`internal/verbs/check/coverage.go`'s `AuditedClauses()` names the 13
clauses this checker can emit a finding for, held equal by tests to both
the package's own `find`/`Finding` call sites and CONTRACT.md's `[check]`
marks. The table sets each `[check]` marker at `d94f4ee` beside what
`internal/verbs/check/audit.go` actually reads. "Anywhere" means a
substring or pattern test over the whole file.

| Clause | What the marker claims | What the checker reads | Extent |
|---|---|---|---|
| C1.1 | `CGO_ENABLED=0` in the Makefile, `ci.yml` and `flake.nix`, anywhere in each | the same three tests | matches |
| C1.2 | that `cmd/<tool>` exists as a main package | that `cmd/` has at least one subdirectory | claims more |
| C1.6 | the whole clause: `flake.nix` (buildGoModule, nixos channel, flake-utils), `.envrc` (soft-fail guard, `use flake`), `postInstall` copying assets and removing `assets.go` | `flake.nix` exists; `.envrc` exists and contains `use flake`; when `assets/` exists, `flake.nix` contains `share/` | claims more |
| C2.1 | `.golangci.yml` bans `fmt.Print(`, `fmt.Println(`, `fmt.Printf(` | `.golangci.yml` exists and contains `forbidigo` and `fmt\.Print` | matches, at substring level |
| C3.1 | the whole clause: identity, `schemaVersion`, verbs, assets, backends | `manifest --json` runs and exits 0, and `schemaVersion` is non-zero | claims more |
| C3.4 | that `manifest_digest` is present and `sha256:`-prefixed | the same | matches |
| C3.6 | `"contract": "toolsmith/v1"` | a `toolsmith/` prefix (T24 records that `toolsmith/banana` passes) | claims more |
| C6.2 | `--match 'v[0-9]*'` agreeing with `tag_pattern` | the Makefile contains `--match 'v[0-9]*'`; `cliff.toml` exists and its `tag_pattern` is `"v[0-9]*"` | matches |
| C6.3 | the whole clause: generated per release by git-cliff, never committed, two make targets | `CHANGELOG.md` is not in the git index | claims more |
| C6.5 | SHA-pinned actions | every `uses:` ref is 40 hex; also that both workflows exist and run `nix develop --command` | matches, and reads more |
| C6.6 | the whole clause: enforced from commit one, `make hooks` installs both hook types | the hook file exists; `.pre-commit-config.yaml` contains `commit-msg`; when a `hooks:` target exists, the Makefile contains `--hook-type commit-msg` | claims more |
| C6.7 | the whole clause: schema v2, `default: none`, six linters, gofumpt | `.golangci.yml` contains `version: "2"` and `default: none` | claims more |
| C7.5 | the whole clause: README states the tool, the install two-step, the design of record | `README.md` exists | claims more |

Eight markers claim more than the checker reads. The Conformance section
calls that the document overclaiming, and `fe7d4d8` narrowed C1.1's marker
for the same reason. The clean run in §4 is clean only against the
"What the checker reads" column. `docs/binary/decisions.md` records the
eight markers, and a Stage 8 step narrows them.

Everything else in CONTRACT.md is prose, audited by reading rather than by
this checker. `docs/binary/decisions.md` is that reading pass for
toolsmith, recorded against CONTRACT.md v1.1. It closes C7.1 and records
the clauses that diverge, none of which this checker can reach:

- **C2.3** — `check` and `new` ignore the global `--json` flag (§4).
- **C2.5** — `new` and `doctor` return error codes outside the five
  documented prefixes. C2.4's exit codes still hold.
- **C4.2** — `registry.All` is a slice literal, so no duplicate
  registration can panic.
- **C4.4** — the claude-code projection carries two hand-written strings
  (`skillDescription`, the `plugin.json` suffix) outside the asset chain.
- **C4.5** — the code produces only `current` of the six drift states.

A clean `toolsmith check` run therefore says nothing about these five.
They are prose divergences with a disposition each in
`docs/binary/decisions.md`, not silent passes of a check that could catch
them.

## 6. Deliberate breaks

Each break was made on a fresh `cp -a` of the clone (never the clone
itself, never the repo), checked, then discarded before the next.

**Break 1 — `.github/workflows/ci.yml` loses `CGO_ENABLED=0` (expected:
C1.1).**

```
$ sed -i 's/CGO_ENABLED=0 GOOS=/GOOS=/' .github/workflows/ci.yml
$ toolsmith check <break-copy>
C1.1: ci.yml does not set CGO_ENABLED=0
exit=1
```

Fired, on stdout, as expected — this is the step-37 extension of C1.1 to
CI.

**Break 2 — `cliff.toml`'s `tag_pattern` changed to `"v*"` (expected:
C6.2).**

```
$ sed -i 's/tag_pattern = "v\[0-9\]\*"/tag_pattern = "v*"/' cliff.toml
$ toolsmith check <break-copy>
C6.2: cliff.toml tag_pattern is not "v[0-9]*"
exit=1
```

Fired, on stdout, as expected — this is the step-18 correction.

**Break 3 — a `CHANGELOG.md` created and `git add -f`'d, not committed
(expected: C6.3).**

```
$ echo "# Changelog" > CHANGELOG.md
$ git add -f CHANGELOG.md
$ toolsmith check <break-copy>
C6.3: CHANGELOG.md is tracked; the contract generates it per release instead
exit=1
```

Fired, on stdout, as expected. `changelogTracked` reads
`git ls-files CHANGELOG.md`, which lists a staged-but-uncommitted file, so
no commit was needed to trigger it.

All three breaks fired the expected clause and nothing else, both outside
and (re-run, to match this record's stated method) inside `nix develop`,
with identical output either way. No break failed to fire.

## 7. A freshly instantiated tool

From an empty scratch dir, `<scratch>/bin/toolsmith new demo --dir demo`
was run with `XDG_CONFIG_HOME` pointed at an empty directory, as step 4
above. The first attempt failed before reaching `check` at all:

```
$ XDG_CONFIG_HOME=<empty> toolsmith new demo --dir demo
Author identity unknown
*** Please tell me who you are.
...
fatal: unable to auto-detect email address (got 'dev@ubuntu-8gb-hel1-1.(none)')
toolsmith: new: git commit -q -m chore: instantiate demo from the toolsmith skeleton failed in demo: exit status 128
exit=1
```

This is an environment finding, not a toolsmith defect: on this host,
git's global identity (`user.name`/`user.email`) is supplied by
home-manager into `$XDG_CONFIG_HOME/git/config`, not `~/.gitconfig`
(`~/.gitconfig` does not exist). Pointing `XDG_CONFIG_HOME` at an empty
directory — required so no host override shadows toolsmith's embedded
assets — also removes git's global identity for any subprocess `new`
spawns, since `new`'s internal `git init`/`git commit` steps inherit the
process environment and read no config of their own. The retry set
`GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL`, `GIT_COMMITTER_NAME`,
`GIT_COMMITTER_EMAIL` for that one subprocess invocation only — no config
file written or touched:

```
$ XDG_CONFIG_HOME=<empty> GIT_AUTHOR_NAME=... GIT_AUTHOR_EMAIL=... \
  GIT_COMMITTER_NAME=... GIT_COMMITTER_EMAIL=... toolsmith new demo --dir demo
instantiated demo at demo (module github.com/procrastivity/demo)
exit=0
```

`toolsmith check demo` on the instantiated tree:

```
$ XDG_CONFIG_HOME=<empty> toolsmith check demo
no findings — mechanical clauses hold for <scratch>/demo-scratch/demo
exit=0
```

No C3.1 finding: `go run ./cmd/demo manifest --json` built and ran inside
`check`'s own subprocess. Verified directly, as this step requires:

```
$ cd demo && go run ./cmd/demo manifest --json
{"tool":{"name":"demo","version":"dev","commit":"unknown","date":"unknown"},"schemaVersion":1,"contract":"toolsmith/v1","verbs":[...5 verbs...],"assets":[...3 assets...],"manifest_digest":"sha256:e5db7dfdb6570a018e08c39cfb1e2fffd9834e1914ccf0dabf2c5e85a1ee8219"}
exit=0
```

Module resolution needed no network: this host's Go module cache already
holds every dependency the skeleton pulls in (from building toolsmith and
`tmp/smoke` earlier in this same session), and no `GOPROXY`/`GOFLAGS`
override was set. The demo `check` run is genuinely clean, not clean
because C3.1 masked itself — the manifest step actually ran and actually
produced the expected document shape.

## Where this stands

`make check` passes. `toolsmith check` finds nothing on toolsmith's own
tree at `d94f4ee`, nor on a freshly instantiated tool. All three
deliberate breaks fire their clause and no other, so the clean run comes
from a checker that can report.

The run is clean against 13 clauses, at the extent §5's "What the checker
reads" column states. Eight of those markers claim more than that. The
five prose divergences in §5 are outside the checker's reach.

Two facts surfaced that do not change the result. `check` and `new`
ignore `--json` (§4). On this host, git's identity lives under
`$XDG_CONFIG_HOME`, so pointing that variable at an empty directory also
removes git's identity from `new`'s commit (§7).
