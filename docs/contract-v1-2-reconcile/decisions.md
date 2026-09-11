# `contract-v1-2-reconcile` — decisions this Matter made

The design of record is the Phase A plan,
`~/.claude/plans/write-a-detailed-handoff-recursive-swan.md`. It is external
on purpose and is never copied into this repo. The planning input for this
Matter is `~/Code/toolsmith-reboot/HANDOFF-contract-v1.2-reconcile.md`, in
the same sidecar. The Matter register is wip (T22), so the Body, Workplan,
Stages, Steps and findings live there. This file records what building the
Matter forced that CONTRACT.md does not say (C7.1).

**Status: the C4.5 design is recorded (step-13). The reconciliation pass
against v1.2 is not yet run.**

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

`IsCurrent(dir, files)` becomes `Status(dir, files) == current`.
`RefuseHandEdited` is removed. Its two cases are now two states.

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

## Where this stands

The C4.5 design is recorded. Steps 14 to 17 build it in both trees. The
reconciliation pass against v1.2 (step-18) extends this file.
