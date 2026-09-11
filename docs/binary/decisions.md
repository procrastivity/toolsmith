# `toolsmith-binary` — decisions this Matter made

The design of record is the Phase A plan,
`~/.claude/plans/write-a-detailed-handoff-recursive-swan.md` (§Phasing
Phase B, and §Settled decisions item 6). It is external on purpose and is
never copied into this repo. The planning sidecar is `toolsmith-reboot`
(`HANDOFF.md`, `SEED-CARDS.md`). The Matter register is wip (T22), so the
Brief, Body, Workplan, Stages, Steps and the Matter's findings live there.
This file records what building the Matter forced that CONTRACT.md does
not say. It is the reconciliation pass that `assets/playbook/migrate.md`
Stage 8 and CONTRACT.md's Conformance section ask for.

**Status: reconciliation pass complete, read against CONTRACT.md v1.1 at
`4e645c6`, and amended at step-29.** Six clauses diverge. This file closes
one of them (C7.1). The other five are recorded with a disposition and are
not fixed by this pass. Separately, eight `[check]` markers claimed more
than the checker reads, and step-38 narrowed them (§3).

---

## 1. How the pass ran

Every clause from C1.1 to C7.5 was read literally against toolsmith's own
tree. The skeleton was read only where a clause concerns what `new`
produces, or where a divergence is inherited from it. Each clause got one
verdict: holds, diverges, not applicable, or unverifiable by reading.

The reading ran at `725e94e`. Two steps landed while it ran, and both
change only `toolsmith check`: step-18 (`d8e0f3f`, the C6.2 audit) and
step-37 (`4e645c6`, the C1.1 audit now reads `ci.yml`). The pass treated
both as landed.

- **Not applicable:** C4.8 (no splice, emit-only or variant target, as
  step-25's v1 harness scope records), C4.9 (no data dir, because `new`
  writes into the target repo), C4.10 (no lifecycle hook), C5.4 (no
  encumbered asset).
- **Unverifiable by reading:** C6.1, which is a claim about what people do.
- **Holds, with a note:** C4.7. Doctor's garbage collection is a "may", and
  one harness with a fixed install directory leaves nothing to collect.
- **Holds:** every other clause, except C2.3, which the reading missed.
  (step-29) The conformance run found it, and §2 now records it.

The reading compared each clause with the built tool. It did not compare
each `[check]` marker's extent with what the checker reads, although the
Conformance section asks for that too. (step-29) The conformance run made
that comparison, and §3 records the result.

Each note below names where its divergence came from, because the origin
decides who fixes it:

- **port** — this Matter's own choice while porting a verb.
- **chassis** — inherited from the skeleton. Every tool `new` produces
  carries it, and a fix lands in both trees under the drift gate
  (`drift/drift_test.go`).
- **contract** — the clause claims more than any implementation does.

## 2. Implementation-forced notes

### C2.3 — `check` and `new` ignore `--json` (port)

**The clause:** `--json` emits the success payload as one JSON value.

**What was built:** `manifest`, `version`, `install`, `uninstall` and
`doctor` read the flag. `check` and `new` never read it, so
`toolsmith check --json` prints the same text lines as a plain run
(`evidence/2026-09-11-toolsmith-conformance.md` §4).

**Why:** the retired oracles had no JSON output, and the port reproduced
their streams byte for byte through the parity window. The port spec does
not rule on `--json`.

**Disposition:** backlog `01M26ZTDV24B342XPF699KC3PF`. Each verb needs a
payload shape, or a recorded ruling that a verb may decline `--json`.
`check`'s payload is where T24's derived conformance would report the
audited clause set.

### C2.5 — error codes outside the prefix set (port, chassis)

**The clause:** `Code` is a dotted token under `refusal.*`, `validation.*`,
`not-found.*`, `advisory.*` or `internal.*`, and exit codes map from the
prefix.

**What was built:** `new` returns `new.missing-name`, `new.invalid-name`,
`new.placeholder-name`, `new.git-failed` and `new.no-skeleton`
(`internal/verbs/new/instantiate.go`). `doctor` returns
`doctor.findings-present` (`internal/verbs/doctor/doctor.go`), in toolsmith
and in the skeleton.

**Effect:** C2.4 still holds. `exitcode` maps `refusal.` to 3 and
`internal.` to 4, and every other prefix falls to 1, which is the code each
of these cases wants. C2.5's vocabulary does not hold: a consumer of the
`{"error":{code,message}}` envelope cannot classify these codes by prefix.

**Why:** no recorded reason for `new`. Port spec §9.3 rules on `new`'s exit
codes, not on its code tokens. `doctor.findings-present` came with the
chassis. None of the five prefixes names "the run reported failing
findings", so that case may be a gap in the clause, not in the code.

**Disposition:** backlog `01M26Z2ZFSG03M9NQ2YNKZHBNH`. The three name cases
read as `validation.*`. `new.git-failed` and `new.no-skeleton` need a
ruling, because `internal.*` would move their exit code to 4.
`doctor.findings-present` needs a sixth prefix by decision, or a mapping
into the five.

### C4.2 — a duplicate registration does not panic (chassis)

**The clause:** registration panics on duplicates, because registration is
static program construction, not user input.

**What was built:** `registry.All` is a slice literal
(`internal/harness/registry/registry.go`). There is no registration step,
no duplicate check and no panic. With one row, nothing can collide yet.

**Why:** no recorded reason in this repo.

**Disposition:** already on the backlog as `01M25VVQQCDCQHJ65P7FSA13TT`.

### C4.4 — two hand-written strings outside the asset chain (port, chassis)

**The clause:** the only hand-written content in any projection is the
per-harness judgment prose and the shared guidance, and both ride as
assets.

**What was built:** the claude-code projection also carries
`skillDescription`, a Go constant, and the fixed suffix that
`renderPluginJSON` writes into `plugin.json`
(`internal/harness/claudecode/claudecode.go`). The skeleton carries the
same two strings. The package comment calls this "a tension with the
clause rather than a conformance to it".

**Why:** each string is one line, and step-24 revised both against the
shipped verb surface and kept them inline. The package comment cites "the
Stage 6 finding on the toolsmith-binary Matter". That finding is not among
the Matter-level findings wip renders, so the citation does not resolve
from this repo.

**Disposition:** backlog `01M26Z2ZHH2EVJN7YDM6839AQ6`. Either amend C4.4 to
admit a one-line description and a fixed `plugin.json` suffix, or move both
strings to assets.

### C4.5 — only `current` of the six drift states is produced (chassis, contract)

**The clause:** drift is described as `current | missing | stale |
modified | unowned_conflict | incompatible`. wip's per-file
added/removed/changed report is the floor, and it maps into the six states.

**What was built:** the floor exists (`manifest.Drift`), and four call
sites use it. Only `install` reports a state, and the only state it
reports is `current`. Other drift surfaces as doctor advisories, or as
`refusal.unstamped-harness-target`. That one code covers both a hand-edited
target and an unowned one, and only the message tells them apart. Nothing
detects `missing` (a stamp gone after install) or `incompatible`. The
`Drifted` comment in `internal/manifest/drift.go` says the callers derive
the six states from the floor, and they do not.

**Why:** the chassis carries wip's floor. T6 adopted the six-state
vocabulary from duo without narrowing it, so the clause describes duo's
model, not the chassis this contract ships.

**Disposition:** backlog `01M26Z2ZKRXB5KWE08M756FD1Y`. Either implement the
states in the chassis, or amend C4.5 to require the floor and name the six
states as the target. Either way, correct the `drift.go` comment in both
trees.

### C7.1 — this file did not exist (contract, playbook)

**The clause:** `docs/<matter>/decisions.md` records what building a
Matter forced.

**What was built:** until this file, `docs/binary/` held only the port spec
and the divergences list. The Matter's decisions lived only as wip
findings.

**Why:** three texts disagreed about where these notes go.

- C7.1, and `assets/handoff-kit/sidecar-README.md`'s "What does live in
  the tool repo", name `docs/<matter>/decisions.md`.
- `migrate.md` Stage 8 said "the sidecar's decisions".
- The conventions in `sidecar-README.md`, and the reconciliation step in
  `workplan-template.md`, fold the notes into the Brief text in place. A
  tracker whose Brief is create-once, as wip's is, cannot do that, and T22
  lets the tracker be the medium.

**Disposition:** closed by this file. `migrate.md` Stage 8 now names
`docs/<matter>/decisions.md`. The Brief-in-place convention is recorded
as a wip finding against the kit, for `clast-conversion` to read.

## 3. `[check]` markers that claim more than the checker reads (step-29)

The Conformance section says a marker states exactly which part of a
clause the checker covers, and a marker that claims more is the document
overclaiming. `fe7d4d8` narrowed C1.1's marker for that reason. The
conformance run set every marker beside `internal/verbs/check/audit.go`
(`evidence/2026-09-11-toolsmith-conformance.md` §5). Eight still claim
more:

- **C1.2** claims a main package. The checker reads that `cmd/` has a
  subdirectory.
- **C1.6** sits at the head of the clause. The checker reads that
  `flake.nix` exists, that `.envrc` contains `use flake`, and, when
  `assets/` exists, that `flake.nix` contains `share/`.
- **C3.1** sits at the head of the clause. The checker reads that
  `manifest --json` exits 0 with a non-zero `schemaVersion`.
- **C3.6** quotes `toolsmith/v1`. The checker reads a `toolsmith/` prefix,
  as T24 records.
- **C6.3** sits at the head of the clause. The checker reads that
  `CHANGELOG.md` is not in the git index.
- **C6.6** sits at the head of the clause. The checker reads the hook
  file, `commit-msg` in `.pre-commit-config.yaml`, and
  `--hook-type commit-msg` in a Makefile with a `hooks:` target.
- **C6.7** sits at the head of the clause. The checker reads
  `version: "2"` and `default: none`.
- **C7.5** sits at the head of the clause. The checker reads that
  `README.md` exists.

The step-32 gate (`coverage_test.go`) cannot catch this class, because it
compares clause IDs, not extents. The origin is the contract: the marks
were placed at clause level before the Conformance rule said where a mark
sits.

**Disposition:** step-38 narrowed all eight markers to the sub-part the
checker reads, in C3.4's parenthetical shape. No checker behavior
changed.

## 4. The contract minor this pass read against

The Conformance section asks for the reading pass to be "recorded with the
minor version it was read against". For toolsmith, this file is that
record: **v1.1**, whose only clause added after v1.0 is C3.8.

T25's machine half has no clause and no code yet: a tool records that
minor, so that the checker can name the clauses added since and call the
pass stale. It is deferred on the backlog as `01M2696J8K3688E5572G9W46HP`,
beside T24's derived-conformance entry, `01M2696J6ECKMW3M7158GK5SMA`.

## Where this stands

toolsmith holds every clause it can be read against except C2.3, C2.5,
C4.2, C4.4 and C4.5. Each of the five is on the backlog with the choice it
needs. The eight overclaiming markers are narrowed at step-38.

None of this blocks the Matter's seal. The seal asks for three things:
`toolsmith check .` clean on toolsmith, the oracles and the gate deleted in
one commit, and the gate's final run in `evidence/`. The last two hold at
`725e94e` and in `evidence/2026-09-10-parity-final.md`. The first holds at
`d94f4ee`, in `evidence/2026-09-11-toolsmith-conformance.md`.

C2.3 is port-only. The other four notes reach the chassis, so every tool
`new` has produced carries them too. C4.2 needs only code. C2.3, C2.5,
C4.4 and C4.5 each need a ruling first, because each fix is either new
code or a narrower clause.
