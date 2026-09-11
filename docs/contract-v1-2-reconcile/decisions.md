# `contract-v1-2-reconcile` — decisions this Matter made

The design of record is the Phase A plan,
`~/.claude/plans/write-a-detailed-handoff-recursive-swan.md`. It is external
on purpose and is never copied into this repo. The planning input for this
Matter is `~/Code/toolsmith-reboot/HANDOFF-contract-v1.2-reconcile.md`, in
the same sidecar. The Matter register is wip (T22), so the Body, Workplan,
Stages, Steps and findings live there. This file records what building the
Matter forced that CONTRACT.md does not say (C7.1).

**Status: reconciliation pass complete, read against CONTRACT.md v1.2 at
`44cba9d`, and amended at step-19.** The five divergences this Matter took
on are closed. The pass and the conformance run found three more that
predate this Matter. Each is on the backlog, and none blocks the seal (§2.3).

---

## 1. The six drift states (C4.5, step-13)

C4.5 names six states, and until this Matter the chassis produced only
`current` (`docs/binary/decisions.md` §2). The chassis now derives all six
for each harness target, in one function, `harness.Status`, from the three
C4.6 comparisons plus the stamp's `schemaVersion`.

### 1.1 The states, in the order they are checked

`Status` checks the conditions in this order, and the first match wins:

| Order | Condition | State |
|---|---|---|
| 1 | A stamp exists but cannot be parsed, or its `schemaVersion` differs from the binary's `manifest.SchemaVersion` | `incompatible` |
| 2 | The directory holds files and has no stamp | `unowned_conflict` |
| 3 | The files on disk differ from the stamp | `modified` |
| 4 | The binary's generated files differ from the stamp | `stale` |
| 5 | The directory is absent or empty, and there is no stamp | `missing` |
| 6 | None of the above | `current` |

The order puts the unsafe states first. A tree can be both `modified` and
`stale`: a user edited a file, and the binary also changed. `modified` wins,
because `--force` on that tree destroys the user's edit, and that is the
fact the caller must see.

`missing` follows duo's meaning: no projection is present. A tree whose
stamp is gone but whose files remain is `unowned_conflict`, because nothing
proves the tool wrote those files.

A stamp that parses, carries the binary's `schemaVersion` and has no `files`
map is not `incompatible`. The comparisons then decide: files on disk make it
`modified`, and an empty tree makes it `stale`.

### 1.2 How a stamp error maps

`manifest.ReadStamp` returns three kinds of result, and `Status` maps each:

- **No stamp file.** Not an error. Conditions 2 and 5 decide.
- **The file exists but its JSON does not parse.** `ReadStamp` wraps the
  error in a sentinel, `manifest.ErrStampUnparseable`, and `Status` maps it
  to `incompatible`. A stamp this binary cannot read is the same fact as a
  stamp from another schema: the binary cannot tell what it owns.
- **The file cannot be read** (permissions, an I/O error). `Status` returns
  the error. That is not a drift state, and it must not look like one.

A `schemaVersion` of zero, which a stamp without the field decodes to, is
`incompatible`, because it differs from the binary's version.

### 1.3 One function for two kinds of caller

`Status(dir string, files map[string][]byte) (State, error)` takes the
binary's generated files for condition 4. `uninstall` has no reason to build
a manifest, because removing a tree does not depend on what the binary would
generate. When `files` is nil, `Status` skips condition 4, so a stale tree
reads as `current`, and the comment on `Status` states that.

Rejected: a second exported function for the disk-only question. Its
`current` would mean something narrower than `Status`'s `current`, under the
same name.

`IsCurrent(dir, files)` became `Status(dir, files) == current` at step-14.
Step-15 deleted it, because `install` then called `Status` itself and no
caller was left. `RefuseHandEdited` is removed, and its two cases are now
two states. `harness.Risk` words the fact behind each unsafe state once, for
both `install`'s refusal and `doctor`'s finding (step-16).

### 1.4 Refusals

The single `refusal.unstamped-harness-target` covered two different risks,
and only its message told them apart. It splits into one code per refusing
state:

| State | Code | `install` message names | `uninstall` message names |
|---|---|---|---|
| `unowned_conflict` | `refusal.unowned-harness-target` | `--force`, which overwrites files the tool never wrote | removing the tree by hand |
| `modified` | `refusal.modified-harness-target` | `--force`, which destroys the edits | removing the tree by hand |
| `incompatible` | `refusal.incompatible-harness-target` | `--force`, which replaces the stamp | removing the tree by hand |

`uninstall` has no `--force`, so its messages keep today's "remove it by
hand" hint (owner amendment to R5).

The other states:

- `install` writes on `missing` and `stale`, and reports `current` without
  writing. `install --force` skips `Status`, as today.
- `uninstall` removes the tree on `current` (and so on a stale tree, per
  §1.3). On `missing` it keeps `not-found.harness-not-installed`. An empty
  directory with no stamp is now `missing`, where it was a refusal.
- `install` with no harness argument records each refused target's own
  code in its results. Its closing error was also
  `refusal.unstamped-harness-target`. It becomes
  `refusal.harness-targets-refused`, because it summarizes refusals of any
  of the three kinds.

### 1.5 What doctor reports

`doctor` reports each registered target's state, in both output modes:

- In `--json`, a `targets` array of `{"harness","dir","state"}` sits beside
  `findings`.
- In text mode, one `<harness>: <state> at <dir>` line per target comes
  ahead of the findings or the `no issues found` line.

The owner ruled which states fail the run:

| State | Finding | Fails the run |
|---|---|---|
| `current`, `missing` | none | no |
| `stale` | today's `advisory.stale-harness-artifact`, one per drifted file | no |
| `modified` | `advisory.modified-harness-target` | no |
| `unowned_conflict` | `advisory.unowned-harness-target` | no |
| `incompatible` | `refusal.incompatible-harness-target` | yes |

A finding's code names the refusal `install` would return, with `advisory.`
in place of `refusal.` when the state does not fail the run (C4.7). The
per-file stale findings stay for any target whose stamp parses, including a
`modified` one, because the binary-versus-stamp question is independent of
the disk (C4.6). For an `incompatible` target, doctor skips them: today an
unparseable stamp makes doctor exit with an error, and after this change it
is a finding.

### 1.6 Out of scope

- C4.5's seventh dimension, a harness session that loaded an older
  projection, needs knowledge of live sessions. No chassis code has that.
- C4.3 places refusal policy in the verbs, and `claudecode.Uninstall`
  refuses inside the per-harness package. This Matter splits the codes
  where they are and does not move them. The reconciliation pass reads C4.3.

## 2. The reconciliation pass against v1.2 (step-18)

The pass read every clause this Matter touched, literally, against
toolsmith's tree and against the skeleton that `new` ships. Each clause got
one verdict: holds, diverges, or holds with a note. The clauses this Matter
did not touch keep their verdicts from `docs/binary/decisions.md` §1.

### 2.1 Holds

- **C2.3.** `check` and `new` now read `--json`, as `manifest`, `version`,
  `install`, `uninstall` and `doctor` already did, so every verb honors the
  flag. No verb redeclares it. `-v/--verbose` stays bound once at root, and
  `manifest` and `version` read it.
- **C2.4.** The exit table is unchanged. `exitcode.Silent` stays in the
  chassis, and no toolsmith verb returns it any more.
- **C2.5.** Every `toolsmitherr.New` code in toolsmith is under `refusal.`,
  `validation.`, `not-found.` or `internal.`, or has the sanctioned
  `<verb>.findings-present` form (`check`, `doctor`). Doctor's finding codes
  are `advisory.` or `refusal.`.
- **C4.2.** `namesOf` panics on a duplicate harness name.
- **C4.4.** The only hand-written prose in the claude-code projection is
  `description.txt`, `judgment.md` and `agent-guidance.md`, all resolved
  through the asset chain. The `plugin.json` suffix and the `SKILL.md`
  headings are fixed template text, which C4.4 counts as generated (T27).
- **C4.6.** The three comparisons stay distinct inside `harness.Status`, as
  conditions 3 and 4 and the `current` case, and its comment names each.
  `install`'s refusal names `--force`. A version bump with byte-identical
  output is still `current`, because `toolVersion` is never compared.
- **C7.1.** This file exists, with the fixed opening stanza, a bolded
  status line and a closer.

### 2.2 Holds, with a note

- **C4.5.** All six states are derived (§1). The seventh dimension, a
  session that loaded an older projection, is not built, because no chassis
  code knows about live sessions (§1.6).
- **C4.7.** `doctor` reports exhaustively, as flat `{code, message}` findings,
  and only `incompatible` fails the run. `uninstall` removes the stamped tree
  only when the disk matches the stamp, and refuses on foreign content, on
  edits and on an unusable stamp. Two readings to note:
  - On an empty directory with no stamp, it returns
    `not-found.harness-not-installed`, not a refusal, because nothing is
    installed.
  - It removes a stale tree. C4.7's "refuses on … drift" is read as drift
    between disk and stamp, which is the only drift that puts a user's
    content at risk.

### 2.3 Diverges

None of these divergences comes from this Matter's changes. All three are
chassis code that the toolsmith-binary pass read as holding.

- **C2.2 (chassis).** Bare `install` writes its per-target results to
  stdout and then returns `refusal.harness-targets-refused` when any target
  refused. So stdout is not empty on that failure, and C2.2's one exception
  is the `<verb>.findings-present` verdict, which a refusal is not. The code
  was `refusal.unstamped-harness-target` before step-15, with the same
  shape. wip has it too. **Disposition:** backlog
  `01M279NRSG375S17HV10Z1GXK8`, which needs a ruling on what a partial run
  is.
- **C4.3 (chassis).** `install` keeps refusal policy in the verb. But
  `claudecode.Uninstall` still decides its refusal inside the per-harness
  package, because the registry row's `Uninstall func()` takes no policy
  input. Step-15 split the codes where they were and did not move them
  (§1.6). **Disposition:** backlog `01M279NRVB1R4QE5Q9BW2NT204`.
- **C2.4 (chassis), found at step-19.** `cli.Execute` assumes that an error
  which is not a `toolsmitherr.Error` came from Cobra's argument parsing, and
  exits 2. Verbs also return plain errors, such as I/O and asset failures,
  and those exit 2 as usage where C2.4 means 4. The conformance run hit it
  through a malformed description override, the one path this Matter added,
  and step-21 fixed that path. **Disposition:** backlog
  `01M27ACVAM9D3SZSAXZDMZTKQX`.

## 3. `[check]` markers beside what the checker reads

This Matter changed no `[check]` marker, no audited clause and no code in
`internal/verbs/check/audit.go` or `coverage.go` (`git diff c48235e` on both
is empty). The marks gate (`coverage_test.go`) is green, so the 13 marked
clause IDs still equal `AuditedClauses()`. That gate compares IDs, not
extents, so the extents are set here:

| Clause | What the marker claims | What the checker reads | Extent |
|---|---|---|---|
| C1.1 | `CGO_ENABLED=0` anywhere in the Makefile, `ci.yml` and `flake.nix` | the same three tests | matches |
| C1.2 | that `cmd/` holds a `<tool>` directory | that `cmd/` has at least one subdirectory | matches |
| C1.6 | `flake.nix` exists, `.envrc` contains `use flake`, `share/` when `assets/` exists | the same | matches |
| C2.1 | `.golangci.yml` carries the forbidigo rules banning the three `fmt` print calls | `.golangci.yml` contains `forbidigo` and `fmt\.Print` | matches, at substring level |
| C3.1 | `manifest --json` runs, exits 0, and has a non-zero `schemaVersion` | the same | matches |
| C3.4 | `manifest_digest` is present and `sha256:`-prefixed | the same | matches |
| C3.6 | the `toolsmith/` prefix (T24) | the same | matches |
| C6.2 | `--match 'v[0-9]*'` agreeing with `cliff.toml`'s `tag_pattern` | the Makefile's `--match`, and `cliff.toml`'s `tag_pattern` | matches |
| C6.3 | `CHANGELOG.md` is not in the git index | the same | matches |
| C6.5 | SHA-pinned actions | every `uses:` ref is 40 hex, and both workflows exist and run `nix develop --command` | matches, and reads more |
| C6.6 | the hook file, `commit-msg` in the pre-commit config, `--hook-type commit-msg` when a `hooks:` target exists | the same | matches |
| C6.7 | `version: "2"` and `default: none` | the same | matches |
| C7.5 | `README.md` exists | the same | matches |

The eight markers that `evidence/2026-09-11-toolsmith-conformance.md` §5
found claiming more were narrowed at `c48235e`. No marker claims more than
the checker reads.

## 4. The contract minor this pass read against

**v1.2.** The clause added after v1.1 is C2.5's `<verb>.findings-present`
form (T26). T27's C4.4 sentence is a clarification and has no row. The
previous pass, `docs/binary/decisions.md`, read against v1.1.

## Where this stands

toolsmith and its skeleton hold C2.3, C2.5, C4.2, C4.4 and C4.5 at
contract v1.2. Those are the five divergences toolsmith-binary left open,
and each note in `docs/binary/decisions.md` §2 carries its resolution.
Every clause this Matter touched is read against v1.2 in §2. Three older
chassis divergences, C2.2 in bare `install`, C4.3 in `uninstall` and C2.4
in `cli.Execute`, are recorded with backlog entries and are not fixed here.
The conformance run against v1.2 is `evidence/2026-09-11-contract-v1.2.md`.
