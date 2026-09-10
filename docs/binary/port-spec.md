# Port spec: `toolsmith check` and `toolsmith new`

**Status:** normative. Both oracles below (`contrib/check-contract`,
`contrib/new-tool.sh`) are still installed and authoritative; this spec
describes their exact behavior so the Go verbs can be built without
re-reading the shell. When the oracles are deleted at cutover (migrate
Stage 7), this document stops being normative and becomes history — say
so at the top rather than deleting it (`assets/playbook/port-spec.md`,
"Life after cutover").

Matter: `toolsmith-binary`, Stage 4 ("port the verbs, one at a time").
Companion divergence list: `docs/binary/parity-divergences.md` (not
written by this step).

Two oracles are in scope. Every numbered section below carries two
subsections, `§N.1` for `contrib/check-contract` (138 lines, becomes
`toolsmith check`) and `§N.2` for `contrib/new-tool.sh` (136 lines,
becomes `toolsmith new`), wherever their behavior differs enough to need
separate treatment. Sections 1–8 describe what the oracles do. Sections
9–10 describe the port. That boundary is deliberate — keep it clean.

`check` is ported before `new` (workplan, Stage 4); its oracle is the
richer of the two and the corpus already exists.

---

## 1. Scope and judgment calls

**In scope:** the two verbs named above, ported to Go under
`internal/verbs/check/` and `internal/verbs/new/` (C1.4 — the package
layout follows the contract, one package per verb, never the old
`contrib/` file layout). Both verbs are `plumbing` kind
(`surface.Annotate(cmd, surface.Plumbing)`, C3.2): deterministic output,
no LLM shaping, so the manifest walk and the harness projection both see
them correctly from the first commit that registers them.

**Parity posture:** oracle, for both verbs (Brief, Workplan Stage 4).
**Parity contract**, named before any harness code is written
(`assets/playbook/parity-gate.md` §1): **stdout bytes plus exit code**
for `check`, on repos where the oracle's grep is correct (see §9.1);
**directory tree plus exit code** for `new`, because `new`'s only stdout
is a trailing checklist, not a data payload (see §9.2). Not stderr as a
byte-for-byte requirement, not performance — except the two additional
assertions this Matter's workplan layers on top of parity-gate §1
(parity-gate §5): no stderr from the port on a clean run, and
self-determinism across three runs per case. Both apply to `check` and
`new` alike.

**The rule the rulings follow.** Three of the calls below, and §9.7,
divide the same way. **Reproduce an accident that mislabels; diverge from
an accident that misreports conformance.** A wrong clause label or a
silent coverage gap still leaves the reader a true statement about the
repo, and reproducing it is cheap, so the oracle stays fixed and the
correction is scheduled (§9.6). A false finding tells the reader a
conformant repo is non-conformant, and byte-reproducing it would harden
the defect into the tool that replaces the oracle, so the port is
corrected and the parity contract narrows to exclude the inputs that
trip it (§9.1, §9.7). Exit codes are a third case: they carry no
message at all, so the contract's table wins wherever no consumer
observes the oracle's (§9.3).

**Judgment calls made while writing this spec**, each expanded where
cited:

1. **The C6.2 else-branch mislabels its finding, and hides a second
   defect** (`contrib/check-contract:88`; detailed in §4.1 and §9.6).
   With `Makefile` present and `cliff.toml` missing, the branch reports
   the condition under the label `C6.3` rather than `C6.2`. With
   `Makefile` missing and `cliff.toml` present, it reports nothing at
   all, so C6.2 goes unaudited. *Ruling: reproduce both, and schedule the
   fix.* The gate draws its authority from the oracle staying fixed for
   the duration of the port. A port that improves its own reference
   mid-flight cannot prove faithfulness, and both defects sit inside the
   stdout bytes the contract covers. The fix is cheap and safe after
   cutover, when the tool's own tests become the specification
   (`assets/playbook/parity-gate.md` §10), so it lands as a scheduled
   Stage 8 item and a `docs/binary/parity-divergences.md` entry, never as
   a wish. See §9.6.
2. **`check`'s CLI arity.** The oracle requires exactly one positional
   argument (`contrib/check-contract:13`); the Brief's verb table lists
   `check [path]` — bracketed, i.e. optional. *Recommend*: default the
   path to `.` when omitted, and preserve the oracle's semantics
   otherwise (must resolve to a directory, or exit `2`). The parity gate
   always passes an explicit path, so this never affects byte parity;
   it only affects the CLI contract measured in §8.1/§9.
3. **`new`'s failure-path exit codes follow CONTRACT C2.4, not the
   oracle.** Every failure in `new-tool.sh` exits `1` through `die()`
   (§5.2, §8.2), with no usage/refusal split, even though the shapes
   plainly differ: a bad flag, a bad name, an existing target. *Ruling:
   diverge.* The port exits `2` for argument-parsing failures (Cobra's
   own path) and `3` (`exitcode.Refusal`) for the existing-target
   refusal. The parity contract for `new` narrows to say so: **the
   produced tree, the happy-path stdout bytes, and exit `0`**; the
   failure-path exit codes are excluded from parity. Reason: `new`'s exit
   code has exactly two consumers, `README.md:64` and the `smoke` target
   at `Makefile:33`, and both call it on the happy path only.
   Reproducing a flat `1` would mean suppressing Cobra's usage path to
   preserve an accident nobody observes, while C2.4's table is a clause
   the tool must satisfy from its first commit. Naming the parity
   boundary is ours to do (`assets/playbook/parity-gate.md` §1); this is
   where we put it. Recorded as a divergence in §9.3.
4. **`new-tool.sh` refuses an existing target directory**
   (`contrib/new-tool.sh:73`). This is scope, not a new discovery — it is
   already a recorded finding against `assets/playbook/migrate.md`'s
   Stage 3 wording (workplan, Stage 3: "does not mention the refusal").
   §8.2 records the refusal as-is; nothing here reopens it.

---

## 2. Inputs and their loading

### 2.1 `contrib/check-contract`

One input: a single positional argument, resolved to an absolute path
with `repo="$(cd "$1" && pwd)"` (`contrib/check-contract:17`). Everything
downstream reads plain text files inside that directory tree with
`grep`/`has_file` — there is no parser and no malformed-input handling
beyond "the string wasn't found," which is folded indistinguishably into
"the finding fires." Files read, all optional (`has_file` gates each):
`Makefile`, `flake.nix`, `.envrc`, `.golangci.yml`, `go.mod`,
`cliff.toml`, `CHANGELOG.md` (existence-in-git-index only, never read),
`.github/workflows/ci.yml`, `.github/workflows/release.yml`,
`contrib/check-commit-msg`, `.pre-commit-config.yaml`, `README.md`, and
the directory listing of `cmd/`.

One input is not a file: the oracle also *executes* the repo under
audit — `CGO_ENABLED=0 go run "./cmd/$tool" manifest --json` — and reads
its stdout as a string (`contrib/check-contract:72`). This is the one
place the oracle depends on a toolchain (`go`) and a working build, not
just file contents; `command -v go` gates it (line 70). A crash or
compile failure in the audited tool's `manifest` verb surfaces
identically to "no manifest support": stderr is discarded
(`2>/dev/null`) and the `if` fails, producing the single finding
`C3.1: \`$tool manifest --json\` failed or is not implemented` — the
oracle cannot distinguish "doesn't implement manifest" from "implements
it but it panics" from "go isn't on PATH but go.mod exists." All three
collapse to the same finding text.

### 2.2 `contrib/new-tool.sh`

Inputs are the positional `<name>` argument plus three flags
(`--dir`, `--module`, `--no-git`), and the skeleton tree at
`assets/_skeleton/` (`contrib/new-tool.sh:69`), copied wholesale
(`cp -R "$skeleton_dir/." "$target_dir/"`, line 80) before any rewriting
happens. There is no format checked on the skeleton's contents — the
content pass (§4.2 step 4) runs `sed -i` over every regular file found by
`find "$target_dir" -type f -print0` (line 100) unconditionally,
including files that are not text. The skeleton is assumed hygienic (no
binaries) rather than verified to be.

---

## 3. The data model

### 3.1 `check-contract` → Go

The shell state is minimal: a mutable integer counter (`findings`,
`contrib/check-contract:19`) and, transiently, a bash array of `cmd/`
subdirectory names (`cmds`, line 30). The Go port's equivalent data model
is a single ordered slice built during the walk:

```go
type Finding struct {
    Clause  string // "C1.1", "C3.4", ...
    Message string
}
```

appended in the exact source order enumerated in §4.1 — never sorted,
never grouped by clause — plus a `stderr []string` accumulator for the
two stderr-only lines (the multiple-`cmd/`-entries note and the
failing-run count). `internal/verbs/check/` owns this type; it is not
shared with `internal/manifest`, whose `Manifest` struct (`schemaVersion`
`contract` `manifest_digest` fields, `internal/manifest/manifest.go:73-83`)
is what `check` parses out of the audited repo's own `manifest --json`
run (§9.1) rather than grepping.

### 3.2 `new-tool.sh` → Go

No persistent data model either — a sequence of filesystem operations
parameterized by five resolved values: `name`, `target_dir`, `module`,
`upper_name` (`tr '[:lower:]' '[:upper:]'` over `name`,
`contrib/new-tool.sh:77` — safe because `name` is already constrained to
`^[a-z][a-z0-9]*$`, so the uppercase mapping never meets a non-ASCII
byte), and `do_git`. The Go port's equivalent is a validated parameter
struct built once, up front, before any filesystem mutation begins —
matching the oracle's own order (all validation in §4.2 steps 1–3 happens
before `mkdir -p` at line 79).

---

## 4. Core algorithms

Ordering is behavior whenever output is a list (`assets/playbook/port-spec.md`
§4). Both oracles are single top-to-bottom passes; source order *is*
finding order and *is* operation order. No sorting, no reordering, no
short-circuiting a later check because an earlier one failed, except
where a later check is nested inside an earlier check's own `if` (noted
below).

### 4.1 `check-contract` — finding order

1. **Tool-name discovery** (`contrib/check-contract:28-38`).
   `find "$repo/cmd" -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | sort`,
   three branches:
   - exactly one entry → `tool` is that name, silently, no stdout/stderr;
   - more than one entry → `tool` is `cmds[0]` — the **alphabetically
     first** name after `sort` — and a note goes to **stderr**:
     `note: multiple cmd/ entries (...); auditing as "$tool"` (line 35).
     Every downstream check that needs `$tool` (the manifest run) uses
     this alphabetically-picked name; the other entries are never
     mentioned again.
   - `cmd/` missing, or present but empty → `tool` stays `""`, and the
     guard at line 38 fires `C1.2: no cmd/<tool> main package found`.
     These two input shapes are **indistinguishable in the output** —
     "no `cmd/`" and "`cmd/` exists but is empty" both produce the exact
     same finding text.
   An empty `tool` cascades: the C3 block (step 5 below) falls into its
   own `else` (line 79-81) and adds a **second**, related finding,
   `C3.1: cannot run the manifest verb (need go, go.mod, and cmd/<tool>)`
   — one root cause, two findings, the same pattern the evidence record
   already documents for duo's `--output json` vs `--json` mismatch
   (`evidence/2026-09-10-conformance.md:72-76`, "one flag mismatch masks
   three clauses").
2. **C1.1, `CGO_ENABLED=0`** (`contrib/check-contract:40-48`). Two
   independent sub-checks: Makefile (missing file → its own finding;
   present but lacking the string → a different finding), then, *only if
   `flake.nix` exists*, its `CGO_ENABLED *= *0|env\.CGO_ENABLED *= *0`
   pattern. **A missing `flake.nix` produces no C1.1 finding at all** —
   the `if has_file flake.nix` at line 46 has no `else`, so that half of
   C1.1 is silently skipped. The gap is covered instead, redundantly and
   under a different clause label, by step 3.
3. **C1.6, flake/envrc/share layout** (lines 50-59). `has_file flake.nix`
   is checked again here, this time with an `else` — `no flake.nix` is a
   **C1.6** finding, not C1.1 (so a repo with no `flake.nix` gets exactly
   one finding about it, labeled C1.6, and zero about C1.1's flake half).
   `.envrc` existence and its `use flake` string are checked next. The
   `postInstall`/`share/` sub-check (line 57) only runs when **both**
   `flake.nix` and an `assets/` directory exist in the repo; a repo with
   no `assets/` dir skips this sub-check entirely, with no finding either
   way.
4. **C2.1, forbidigo** (lines 61-67). `.golangci.yml` missing → one
   finding. Present → two independent sub-checks: `forbidigo` enabled at
   all, and the fixed-string `grep -qF 'fmt\.Print'` (line 64, detailed
   in §7).
5. **C3.1/C3.4/C3.6, the manifest verb** (lines 69-81). Guarded on
   `[[ -n "$tool" ]] && command -v go >/dev/null && has_file go.mod`; if
   that fails, one finding (`C3.1`, "cannot run the manifest verb").
   Otherwise `go run ./cmd/$tool manifest --json` is captured
   (stderr discarded); on failure, one finding (`C3.1`, "failed or is not
   implemented") and **C3.4/C3.6 are never checked** — the invocation
   failing masks them exactly as the duo run in the evidence record
   shows. On success, three independent greps against the raw JSON
   bytes, each its own finding on a miss: `"schemaVersion"` (C3.1),
   `"manifest_digest":"sha256:` (C3.4), `"contract":"toolsmith/` (C3.6).
6. **C6.2/C6.3, tag namespace** (lines 83-89). *If both* `Makefile` and
   `cliff.toml` exist: two independent greps, each its own **C6.2**
   finding on a miss (the `--match 'v[0-9]*'` string in Makefile, the
   `tag_pattern` string in cliff.toml). *Otherwise* — either file
   missing — the `else` branch at line 88 runs `has_file cliff.toml ||
   find_it "C6.3" "no cliff.toml..."`. This is the bug named in the
   task and in §9.6: the section is titled and gated as C6.2, but the
   only finding its `else` branch can produce is labeled **C6.3**. Worse,
   if `Makefile` is the file that's missing while `cliff.toml` is
   present, `has_file cliff.toml` is true, so `find_it` is **never
   called** — a repo missing only its Makefile silently produces *no*
   finding from this whole section, under either label.
7. **C6.3, CHANGELOG.md tracked** (lines 91-96). Only runs inside
   `git -C "$repo" rev-parse --git-dir`; a non-git directory skips this
   silently. Checks `git ls-files CHANGELOG.md` is non-empty →
   `C6.3: CHANGELOG.md is tracked...`. Note this is a *second*, textually
   distinct finding also labeled C6.3 (step 6 produces the first) — both
   are legitimately about clause C6.3, but they come from different code
   paths and neither code comment cross-references the other.
8. **C6.5, workflow SHA-pinning** (lines 98-111), `for wf in ci.yml
   release.yml`, in that order, each independently: `nix develop
   --command` presence, then the SHA-pin loop over every line matching
   `^\s*-? *uses:` (detailed in §7). Missing workflow file → one finding,
   and the SHA-pin loop for that file does not run at all (there's
   nothing to `grep` from).
9. **C6.6, conventional commits** (lines 113-122). Three independent
   checks in order: `contrib/check-commit-msg` exists; if
   `.pre-commit-config.yaml` exists, it contains `commit-msg`, else its
   own finding; if `Makefile` has a `^hooks:` target, that target's
   recipe contains `--hook-type commit-msg`.
10. **C6.7, golangci schema** (lines 124-128). Only runs if
    `.golangci.yml` exists (already established at step 4) — two
    independent greps, `version: "2"` and `default: none`.
11. **C7.5, README** (lines 130-131). One check, one finding on miss.
12. **Summary and exit** (lines 133-138) — see §5.1.

### 4.2 `new-tool.sh` — instantiation order

1. **Argument parsing** (lines 36-61): a `while`/`case` loop, flags and
   the one positional argument may interleave in any order (the loop
   doesn't care whether `--dir` comes before or after `<name>`). Unknown
   `-*` flags and a second positional argument are both immediate
   `die()` calls (exit 1, see §5.2, §8.2).
2. **Validation** (lines 63-73), in this exact order: `name` non-empty →
   `name` matches `^[a-z][a-z0-9]*$` → `name != "toolname"` → the skeleton
   directory exists → the *resolved* `target_dir` does not already exist.
   `target_dir` and `module` get their defaults resolved (lines 72, 75)
   **before** the existing-target check, so a caller relying on the
   default `--dir` gets the same refusal as one who passed `--dir`
   explicitly.
3. `upper_name` computed (line 77).
4. **Copy** the whole skeleton tree (lines 79-80).
5. **Directory renames, before the content pass** (lines 82-89): `cmd/toolname`
   → `cmd/$name`, `internal/toolnameerr` → `internal/${name}err`, then the
   file inside it, `toolnameerr.go` → `${name}err.go`. The comment at
   lines 82-86 states why renames come first: the content pass (step 6)
   walks the tree by path, so it must see final paths, and Go doesn't
   care what a file is named — a stray unrenamed `toolnameerr.go` would
   still compile, which is exactly the kind of gap that stays invisible
   until a self-conversion's own tree disagrees with what the
   instantiator produces.
6. **Content pass, three ordered substitutions per file** (lines 91-100),
   `LC_ALL=C sed -i` over every regular file: (a) the full module path
   `github.com/procrastivity/toolname` → `$module`, (b) the bare word
   `toolname` → `$name`, (c) `TOOLNAME` → `$upper_name`. Order (a) before
   (b) is load-bearing, not cosmetic (comment, lines 91-93): (a)'s
   pattern is a superset of (b)'s — it contains the substring `toolname`
   — so if (b) ran first it would already have rewritten every
   occurrence of the module path's `toolname` segment to `$name`,
   leaving nothing for (a) to match. For the *default* module
   (`github.com/procrastivity/$name`) this happens to produce the same
   final string either way; for a **custom** `--module`, running (b)
   first would strand the literal string `github.com/procrastivity/$name`
   in every file instead of the custom module path. §7 expands on why
   `LC_ALL=C` specifically.
7. **Restore real module files** (lines 102-109): `go.mod.tmpl` →
   `go.mod`, `go.sum.tmpl` → `go.sum`. The comment explains the `.tmpl`
   spelling exists only so `assets/_skeleton` itself isn't a second Go
   module that would break `//go:embed` from `assets/assets.go`
   ("cannot embed directory X: in different module").
8. **`--no-git` branch** (lines 111-115): when `do_git=1` (the default),
   `git init -q -b main`, `git add -A`, `git commit -q -m "chore:
   instantiate $name from the toolsmith skeleton"`. `--no-git` skips all
   three — this is the path the self-conversion and the Makefile's
   `smoke` target use (workplan, Stage 3).
9. **Print the checklist** (lines 117-136) — the entire stdout contract,
   verbatim in §5.2.

---

## 5. Emission

### 5.1 `check-contract` — byte-level, per stream, per exit path

Stream ownership is **asymmetric by design**, and under a stdout-bytes
parity contract that asymmetry is the whole contract:

| stream | content | when |
|---|---|---|
| stdout | one `"$clause: $message"` line per finding, `find_it` (`contrib/check-contract:21`), interleaved with the script's control flow in the exact §4.1 order | every finding, any exit path except usage |
| stdout | `no findings — mechanical clauses hold for $repo` (line 134) | only when `findings == 0`, i.e. only on exit 0 |
| stderr | `note: multiple cmd/ entries (...); auditing as "$tool"` (line 35) | only when `cmd/` has 2+ entries — **can co-occur with exit 0** (see below) |
| stderr | `$findings finding(s) for $repo` (line 137) | only when `findings > 0`, i.e. only on exit 1 — never printed on exit 0 |
| stderr | `usage: contrib/check-contract <tool-repo-dir>` (line 14) | only on the usage path, exit 2, and nothing else runs — no findings, no summary line, on this path |

Three exit paths, exactly:

- **exit 0** — `findings == 0`. stdout: zero or more finding lines
  (there are none, by definition of `findings == 0`) then the one clean
  line. stderr: normally empty — **except** when the multiple-`cmd/`
  note fired during tool discovery (§4.1 step 1) while every other check
  still passed. That input shape (2+ `cmd/` entries, otherwise clean)
  makes the oracle exit 0 *with* non-empty stderr, which is in tension
  with this Matter's added "no stderr on a clean run" assertion
  (parity-gate §5) — flagged in §9 and §10; none of the five corpus
  repos has more than one `cmd/` entry (checked directly: `wip`, `duo`,
  `ste9`, `tmp/smoke` each have exactly `cmd/<name>`), so the fixed
  corpus never exercises this tension today.
- **exit 1** — `findings > 0`. stdout: the finding lines, no clean-run
  line. stderr: the count-summary line, and the multiple-`cmd/` note if
  it applies.
- **exit 2** — usage (`$# -ne 1 || ! -d "$1"`, line 13). stdout: nothing.
  stderr: only the usage line. Note the check cannot distinguish "wrong
  argument count" from "the one argument you gave isn't a directory" —
  both produce the identical usage string.

### 5.2 `new-tool.sh` — byte-level, per stream, per exit path

`new` has no stdout data contract at all (§9.2) — its real output is the
instantiated tree. What stdout *does* carry, on success, is exactly one
thing: the trailing checklist, emitted by the unquoted heredoc at
`contrib/new-tool.sh:117-136` (unquoted, so `$name`, `$target_dir`,
`$module` interpolate; everything else is literal). Verbatim, with `X`,
`/scratch/X`, `github.com/procrastivity/X` standing in for the three
variables:

```
instantiated X at /scratch/X (module github.com/procrastivity/X)

Checklist — the judgment steps the rename cannot do:
  1. grep -rn 'TODO(X)' — fill every marker: root Short, README,
     flake meta.description, the judgment and agent-guidance assets, the
     skill description.
  2. cd /scratch/X && CGO_ENABLED=0 go build ./... && go test ./...
     (should already pass; it did in the skeleton).
  3. nix build — it fails once and prints the real vendorHash; paste it
     into flake.nix.
  4. make hooks — installs both pre-commit stages.
  5. Decide the verb surface; register verbs in internal/cli/root.go,
     one package each, every constructor ending in surface.Annotate.
  6. For a migration (not a fresh tool): follow toolsmith's
     assets/playbook/migrate.md — port spec, parity gate, cutover.
  7. Run contrib/check-contract /scratch/X from the toolsmith repo and
     clear any findings.
  8. Add the tool to toolsmith's TOOLS.md.
```

Note the blank line after the first line (line 119 in the source is
empty) — it is part of the byte contract for the tree/checklist
comparison, not incidental formatting.

On every failure path, `die()` (lines 26-29) writes `new-tool: $*` to
**stderr** and exits **1** — there is no stdout output whatsoever on any
failure path, and there is exactly one exit code for every failure
shape: bad flag, missing value, duplicate positional, empty name, bad
name shape, the literal name `toolname`, missing skeleton, or existing
target directory. Git's own output is suppressed with `-q` on all three
invocations (line 112-114) but not hermetically guaranteed silent (see
§10).

---

## 6. Configuration and environment

Neither oracle reads an environment variable to change its behavior.
Each **sets** one, narrowly. `check-contract` prefixes `CGO_ENABLED=0`
onto its single `go run ./cmd/<tool> manifest --json` invocation
(`contrib/check-contract:72`), and runs it with the audited repo as the
working directory (`cd "$repo" && …` in a subshell, so the oracle's own
cwd is untouched); the port must reproduce both, or a repo with cgo
sources audits differently under the two implementations.
`new-tool.sh` sets the other: `LC_ALL=C` prefixed onto the single
`sed -i` invocation only (`contrib/new-tool.sh:95`), not exported for the
rest of the script — everything before and after that line runs under
whatever locale the caller's shell already has. Neither script reads
`$HOME`, `$XDG_*`, or any `<TOOL>_*` seam; both operate purely on their
positional/flag arguments and the filesystem they're pointed at. The Go
port introduces no new environment surface for either verb — nothing in
CONTRACT.md's C2.6 (`<TOOL>_*_DIR` test seams) applies here, since
neither oracle has state to seam around.

---

## 7. Host-language semantics you must reproduce

- **`grep -qF 'fmt\.Print'`** (`contrib/check-contract:64`) is a
  **fixed-string** (`-F`) match for the five literal characters
  `fmt\.Print`, backslash included. It works only because `.golangci.yml`'s
  forbidigo entries are written as regex patterns inside YAML strings
  (e.g. `'fmt\.Print('`), so the raw byte sequence `f m t \ . P r i n t`
  genuinely appears in the file. A `.golangci.yml` that banned the same
  function with an unescaped pattern, a different quoting style, or a
  semantically-equivalent rule spelled another way would not contain
  that exact byte sequence and would trip a false C2.1 finding despite
  correctly banning `fmt.Print*`. This is an accident with a consumer:
  the check is bound to one spelling, not to the ban's presence.
- **The C6.5 SHA-pin loop** (lines 102-107) does string surgery per
  matched `uses:` line, three steps: `ref="${line##*@}"` (strip up to
  and including the last `@`), `ref="${ref%% *}"` (keep only the first
  whitespace-delimited token of what's left), then separately,
  `trimmed="${line#"${line%%[![:space:]]*}"}"` (strip *leading*
  whitespace only, for the message text — this does not affect `ref`).
  Two edge cases, worked through exactly:
  - **A line with no `@`** (e.g. a local composite action,
    `- uses: ./.github/actions/foo`, no version pin at all).
    `${line##*@}` matches nothing (no `@` present) and returns `$line`
    **unchanged**, indentation and dash included. `${ref%% *}` then
    strips from the very first space in that unchanged string — which,
    for an indented line, is the leading whitespace itself — leaving
    `ref=""`. An empty string never matches `^[0-9a-f]{40}$`, so the line
    is reported `not SHA-pinned` (line 106) — correctly, for the wrong
    mechanical reason: it isn't detecting "no `@`," it's falling through
    a generic non-match the way any malformed `ref` would.
  - **A commented-out `uses:` line** (e.g.
    `  # - uses: actions/checkout@v4`). It is never seen at all: the
    filter is `grep -E '^\s*-? *uses:'` (line 107) — leading whitespace,
    an optional single dash, more spaces, then the literal `uses:`
    immediately. A `#` sitting between the whitespace and `uses:` breaks
    that adjacency at every possible position, so the line never matches
    the filter and never enters the loop. The oracle silently skips
    commented-out `uses:` lines — it neither flags nor exempts them on
    purpose; they simply never reach the check.
- **`set -euo pipefail`** is active in both scripts
  (`contrib/check-contract:11`, `contrib/new-tool.sh:24`). `errexit`
  does **not** apply to a command that is the condition of `if`/`while`,
  or that is followed by `||`/`&&` — every `grep -q ... || find_it ...`
  pattern in `check-contract` and the manifest capture
  (`if manifest_json="$(...)"; then`, line 72) rely on exactly this
  exemption to turn a "no match" or "command failed" outcome into a
  controlled finding instead of a script abort. Where the exemption
  *doesn't* apply, a failing command aborts the oracle outright, with
  that command's own exit code — not a crafted usage/finding code:
  - `check-contract:17`, `repo="$(cd "$1" && pwd)"` is a bare assignment,
    not an `if` condition; a directory that passes `[[ -d "$1" ]]` but
    can't actually be `cd`'d into (e.g. execute bit denied) aborts the
    whole script via `errexit`, before a single finding is produced.
  - `new-tool.sh:79-89` (`mkdir -p`, `cp -R`, the three `mv`s) and the
    `sed -i` loop at lines 94-100 are all bare commands. A failure
    partway through — disk full during `cp`, a read-only file blocking
    `sed -i` — aborts the script mid-instantiation with no cleanup: the
    partially-written `target_dir` is left on disk, and a re-run at the
    same `--dir` is refused by the existing-target check (line 73) until
    a human removes it by hand. `git init`/`add`/`commit` (lines
    112-114) are equally unguarded.
- **Tool-name sort** (`contrib/check-contract:30`, `find ... | sort`)
  uses the caller's shell locale, not a fixed byte order; the Go port's
  equivalent (`sort.Strings`, byte-wise) agrees with it for the
  `^[a-z][a-z0-9]*$` names `new-tool.sh` produces, but `check-contract`
  places no format constraint on `cmd/` subdirectory names in the repo
  under audit — a repo with non-ASCII `cmd/` entries could sort
  differently under a non-C locale than under Go's byte-wise sort. Not
  exercised by the corpus (§10); recorded here because it's exactly the
  kind of measured-not-assumed fact §7 exists for.

---

## 8. CLI contract

### 8.1 `check-contract`

Exactly one positional argument, no flags, no stdin behavior (the
oracle never reads stdin), no multi-file behavior (a single directory
argument only — everything inside it is discovered by the checks
themselves, not passed as further arguments). `$# -ne 1 || ! -d "$1"` is
a single combined guard (line 13): zero, two, or more arguments and "one
argument that isn't a directory" are indistinguishable failures, both
exit `2` with the identical usage string. There is no "empty input"
case in the usual linter sense — an empty or near-empty repository
directory still runs every check in §4.1 to completion and typically
produces *many* findings (missing Makefile, missing flake.nix, etc.),
never the "no findings" line.

### 8.2 `new-tool.sh`

One required positional argument (`<name>`), three optional flags
(`--dir`, `--module`, `--no-git`), freely interleaved with the
positional argument in any order (§4.2 step 1). No stdin, no multi-file
behavior. Defaults: `--dir` is `<repo>/../<name>` — a sibling of the
`toolsmith` repository, never inside it — and `--module` is
`github.com/procrastivity/<name>`; neither default nor an explicitly
passed `--module` value is validated against any shape (only `<name>`
gets the `^[a-z][a-z0-9]*$` regex, line 64). The script **refuses an
existing target directory** outright (line 73, `[[ ! -e "$target_dir" ]]
|| die ...`) with no `--force` escape hatch of any kind — this is the
already-recorded finding against `migrate.md` Stage 3 (§1 item 4). Every
failure exits `1` (§5.2) — there is no usage code (`2`)
distinct from a name-validation failure distinct from the existing-target
refusal in the oracle's own output. The port diverges here: §1 judgment
call 3 and §9.3.

---

## 9. Known divergences

**9.1 — C3.4/C3.6 detection: parse, don't grep.** The oracle has no JSON
parser: it runs `go run ./cmd/<tool> manifest --json` and greps the raw
bytes for `"manifest_digest":"sha256:` and `"contract":"toolsmith/`
(`contrib/check-contract:74-75`). Both patterns assume compact JSON with
no space after the colon; `encoding/json`'s `MarshalIndent` (or any
pretty-printer) inserts one, so a pretty-printed manifest yields **false**
C3.4 and C3.6 findings even when the fields are genuinely present and
correct. **Repro**: take any conformant manifest JSON, reformat it with
`json.MarshalIndent(m, "", "  ")` instead of `json.Marshal(m)`, feed it
through the two grep patterns above — both miss. `toolsmith check`
parses the JSON document with `encoding/json` into
`internal/manifest.Manifest` and checks the fields' values directly, so
formatting cannot produce a false finding. Byte-reproducing the oracle's
bug would harden a known defect rather than port a behavior worth
keeping, so **the parity contract excludes repos whose manifest output
trips this — "on repos where the oracle's grep is correct"** (workplan,
Stage 4; §1 of this spec). Today's own manifest verb
(`internal/verbs/manifest/manifest.go:44-49`) emits compact JSON via
`json.Marshal`, so none of the five corpus repos trips this by accident
— it needs a generated probe to exercise at all (§10).

**9.2 — `new` has no stdout contract; the gate compares trees.** `new`'s
real output is a directory tree; per §5.2, the *only* stdout is the
trailing checklist heredoc. `assets/playbook/parity-gate.md` assumes an
input-to-stdout tool throughout §1-§6 — it never contemplates a verb
whose payload isn't on stdout at all. That is a gap in the playbook, not
a defect in this port: §10 records what the gate can and cannot reach
for `new` given that gap.

**9.3 — `new`'s failure-path exit codes.** Ruled in §1, judgment call 3.
Every `new-tool.sh` failure exits `1` uniformly (`die()`,
`contrib/new-tool.sh:26-29`). The port instead follows CONTRACT C2.4:
`2` for argument-parsing failures, `3` (`exitcode.Refusal`) for the
existing-target refusal, `1` for the rest. **Repro**: run
`contrib/new-tool.sh --bogus x` and `toolsmith new --bogus x`; both
print a message on stderr and produce no stdout, and the exit codes are
`1` and `2`. **Why the new one is better**: the oracle's flat `1` is an
artifact of one `die()` helper, not a decision, and its only two callers
(`README.md:64`, `Makefile:33`) invoke `new` on the happy path, so no
consumer observes it. C2.4's table is a contract clause the tool owes
from its first commit, and reproducing the accident would mean
suppressing Cobra's usage path to do it. The parity contract for `new`
is therefore the produced tree, the happy-path stdout bytes, and exit
`0` — failure-path exit codes are outside it, by the boundary-naming
freedom of `assets/playbook/parity-gate.md` §1.

**9.4 — exit `0` with stderr on a multi-`cmd/`-entries repo.** Recorded
in §5.1. A repo with two or more `cmd/<x>` entries that is otherwise
contract-clean makes `check-contract` exit `0` after writing the
"multiple cmd/ entries" note to stderr, which is in tension with this
Matter's "no stderr on a clean run" assertion (parity-gate §5). *Ruling:
keep the note, carve out the assertion.* The note is real information
about an ambiguous repo, and dropping it to satisfy a gate assertion
would trade behavior for convenience. So this is not a divergence: the
port writes the same note to the same stream, and the gate's no-stderr
assertion reads "no stderr on a clean run, except the
multiple-`cmd/`-entries note." Step 14 encodes the carve-out; no corpus
repo currently trips it, so a generated probe is the only way to
exercise it (§10).

**9.5 — `exitcode.Silent` for `check`.** `check` needs it;
`assets/playbook/migrate.md`'s "stdout ownership" trap names this as a
per-verb, port-time decision (Stage 4), and `assets/playbook/parity-gate.md`
§8 ties it directly to C2.4. Reasoning: on the findings-present path
(oracle exit `1`), `check`'s stdout must carry the exact finding lines
and *no* clean-run line (§5.1) — a payload, not an error message — while
C2.2 says "on failure, stdout is empty" and the standard error path
(`toolsmitherr.Render`, C2.5) would both empty stdout and write its own
`toolsmith: check: <message>` line to stderr, which is neither the
finding-count line the oracle writes nor byte-identical to it. So `check`
must write its findings directly to `streams.Out` during `RunE` (as the
oracle's own payload, not as an error), then return
`exitcode.Silent(1)` instead of a `toolsmitherr.Error` so `Execute` exits
`1` without invoking the standard renderer at all. The usage path (`2`,
judgment call 2 in §1: bad/missing `[path]`) does **not** need `Silent`
— both the oracle and the standard chassis path already agree on "stdout
empty, exit 2," and stderr content isn't asserted there, so Cobra's own
usage handling suffices.

**9.6 — the C6.2 else-branch: a mislabel and a coverage hole.** Ruled in
§1, judgment call 1. `contrib/check-contract:86-89` guards the C6.2 tag
comparison on `Makefile` **and** `cliff.toml` both being present, and its
`else` branch tests only `cliff.toml`. Two consequences. **Repro A**
(mislabel): a repo with a `Makefile` and no `cliff.toml` reports
`C6.3: no cliff.toml (changelog is not derivable from tags)` — the
condition is real, the clause label is wrong; it is a C6.2 concern
reported as C6.3. **Repro B** (coverage hole): a repo with a `cliff.toml`
and no `Makefile` reports nothing from this block at all, so the C6.2
`--match 'v[0-9]*'` / `tag_pattern` agreement is never audited. The port
reproduces both for the duration of the parity window, per §1. The
correction is scheduled, not deferred indefinitely: a Stage 8 item plus a
`docs/binary/parity-divergences.md` entry, applied after cutover retires
the gate.
**9.7 — CRLF workflow files: the oracle reports a false C6.5.** Found by
review at Step 13, measured against both implementations. The oracle
reads each `uses:` line with `while IFS= read -r line` over `grep`
output (`contrib/check-contract:107`), which keeps a trailing carriage
return. It then extracts the ref with `${line##*@}` and tests it against
`^[0-9a-f]{40}$` (`contrib/check-contract:106`). On a CRLF-terminated
workflow file the ref is `<40 hex chars>\r`, which fails the test, so a
**correctly SHA-pinned action is reported as not SHA-pinned**. The same
`\r` also rides along in the message text of every real finding from
that file.

**Repro**: write `.github/workflows/ci.yml` with CRLF line endings and a
genuinely pinned `- uses: actions/checkout@11bd7190…683` line. The
oracle emits `C6.5: ci.yml: action not SHA-pinned: - uses:
actions/checkout@11bd7190…683<CR>`; the port emits nothing for that
line.

**Ruling: diverge**, by the rule in §1. This is a false finding of
exactly §9.1's kind — a formatting artifact of the input turned into a
conformance verdict — not §9.6's kind, where the reported condition is
real and only its label is wrong. The port strips the carriage return
before matching, so a pinned action reads as pinned whatever the file's
line endings are, and finding messages carry no stray `\r`. The parity
contract narrows accordingly: **workflow files with CRLF line endings
are outside it**, alongside the pretty-printed manifests of §9.1.

`assets/playbook/parity-gate.md` §3 names CRLF as a required generated
probe, so the gate built at Step 14 must carry this exclusion explicitly
rather than discovering it as a failure.

---

## 10. Verification

The parity gate (built in Step 14, not here) covers this spec as
follows; anything not listed is a gap the gate does not reach.

- **§4.1's twelve-step finding order, §5.1's stream/exit table** — the
  core of what byte-parity on `check` verifies. The five corpus repos
  (`~/Code/wip` branch `go`, `~/Code/duo` branch `go`, `~/Code/ste9`,
  `tmp/smoke`, a freshly instantiated demo — parity-gate §2, "the corpus
  is free") exercise most of §4.1's branches directly, with expected
  findings already recorded as a baseline in
  `evidence/2026-09-10-conformance.md` (wip: 14 findings/5 conditions;
  duo: 11 findings/2 conditions, extracted via `git archive` because its
  working tree sits on another branch). A regression in either
  implementation shows up against that known baseline.
- **What the fixed corpus does *not* reach, and needs a generated probe
  for** (parity-gate §3): the C6.2-else path — both the C6.3 mislabel and
  the C6.2 coverage hole (§4.1 step 6, §9.6) — all five corpus repos
  carry both `Makefile` and `cliff.toml`, so neither the mislabel nor
  the "Makefile missing, cliff.toml present → silently zero findings" gap
  ever fires; the §9.1
  pretty-printed-manifest exclusion — none of the corpus repos' manifest
  verbs pretty-print; the §9.7 CRLF exclusion — every corpus workflow
  file is LF-terminated, and parity-gate §3 requires a CRLF probe that
  must be generated with the exclusion already encoded; the §9.4 exit-0-with-stderr tension — no corpus
  repo has 2+ `cmd/` entries; the §7 `set -e`-abort paths (a
  permission-denied repo directory, a disk-full mid-`new`) — these are
  fault-injection cases, outside what a fixed corpus of real repos can
  produce at all, and belong in `docs/binary/parity-divergences.md`
  rather than the gate's matrix (parity-gate §9).
- **§4.2's instantiation order and §5.2's checklist bytes** — verified
  for `new` by running the oracle and the port from the same `<name>`
  into two scratch directories and diffing the resulting **trees**, not
  stdout (§9.2), plus a separate byte-diff of the captured stdout against
  the exact checklist text in §5.2, with `$name`/`$target_dir`/`$module`
  substituted for whatever the test run used.
- **§8's CLI contract** (both oracles): parity-gate §4's "cover the CLI
  contract separately" — bad flag, duplicate positional, missing name,
  bad name shape, `toolname` refusal, existing target for `new`; wrong
  argument count and non-directory argument for `check`. Note the
  boundary §9.3 draws: for `new`, the gate asserts stdout and the message
  stream on these negative cases but **not** the exit code, because the
  port deliberately exits `2` and `3` where the oracle exits `1`. For
  `check`, exit codes stay under parity on every case.
- **Per parity-gate §5, required in addition to byte/tree equality**: no
  stderr from the port on a clean run (with the §9.4 carve-out: the
  multiple-`cmd/`-entries note is permitted stderr, and no corpus repo
  trips it today) and
  self-determinism — three runs per case, identical output — across
  every corpus case for both verbs.
- **What the gate cannot reach at all**: §7's `set -e`-abort paths;
  anything gated on a toolchain fault (a `go run` that hangs rather than
  fails cleanly); the locale-dependent sort noted in §7, since every name
  in the corpus is already ASCII; and, structurally, anything about
  `new`'s stdout beyond the checklist bytes, because the tree comparison
  is what the gate actually asserts for that verb (§9.2). These stay
  recorded here and in `docs/binary/parity-divergences.md`, not
  discovered by a passing gate.
