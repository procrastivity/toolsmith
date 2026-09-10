# Adding a harness target

One sitting per target. The skeleton ships exactly one worked target
(claude-code, T18) because the second target teaches nothing the first
did not — what it costs is research about the harness, not design.

Generalized from wip's Devin handoff, which added a sixth target to a
tool that already had five.

---

## 1. Research first, and pin the facts

Do this before touching code, and write the answers down in the handoff
or the Matter document. A target added from half-remembered facts about
a harness gets rewritten.

| Question | Why it decides something |
|---|---|
| Which directory does the harness load from? | The install path. Prefer the **tool-owned** directory over any shared one. |
| Is there a per-project location too? | Ignore it. `<tool> install` is host-global. |
| What file does it want? | `SKILL.md` alone, or a skill directory plus a plugin descriptor. |
| Does it need a runtime shim? | Only for lifecycle hooks (C4.10), and the shim re-expresses no verbs. |
| Does it edit a foreign file? | Then it is a splice target (C4.8), not a plain projection. |
| What does the harness's own CLI report? | The live verification in step 4. |

Pin each fact to a **version** of the harness you probed, and date it.
Harness layouts move, and "settled 2026-08-28 against CLI 3000.6.2"
tells the next reader exactly how much to trust the row.

Two findings that look like blockers and are not:

- **Overlap with an existing target.** Several harnesses read each
  other's directories. Install into the tool-owned path anyway: the
  stamp, uninstall, and doctor all need to be per-harness, and a shared
  directory makes all three ambiguous.
- **The harness prefers another copy after install.** Record which base
  directory it reports, and move on. It is a finding, not a defect.

## 2. Write the package

Copy the closest existing target's package to
`internal/harness/<harness>/`, keeping its tests. The shape is fixed:

```go
const Name = "<harness>"
const SkillsDirEnv = "<TOOL>_<HARNESS>_SKILLS_DIR"  // test seam only (C2.6)

func SkillsDir() (string, error)
func InstallDir() (string, error)
func Available() bool                                 // root config dir exists
func Generate(manifest.Manifest) (map[string][]byte, error)
func Install(manifest.Manifest) (string, error)
func Uninstall() (string, error)
```

Rules the copy must keep:

- **Policy-free** (C4.3). The package renders and writes. Refusal,
  `--force`, and the current-tree check live in the verbs.
- **Generate only what the harness reads.** A `SKILL.md`-only harness
  gets no plugin descriptor.
- **`Available()` keys off the harness's root config directory**, not
  its skills subdirectory — a fresh install may not have created the
  latter yet.
- **One judgment asset**, at
  `assets/templates/skills/<harness>/judgment.md`. Copy the nearest
  one and add at most a line about how this harness exposes the skill.
  Do not invent harness-specific cadence.

## 3. Wire it up

In the skeleton's layout, a new target touches four places:

1. `internal/harness/registry/registry.go` — one row in `All`. Every
   reader (install, uninstall, doctor, help text, unknown-harness
   errors) goes through this table, so there is no switch to update.
2. `assets/templates/skills/<harness>/judgment.md` — the new asset.
3. `internal/cli/e2e_test.go` — add the new `<TOOL>_<HARNESS>_SKILLS_DIR`
   seam to `hermeticEnv`'s list, then add a round-trip test.
4. The target's own package tests.

Keep the registry's existing order and append; the first row is the
tool's primary harness, and reordering rewrites every help string for
no gain.

**If the tool predates the registry table**, the same target also
touches per-verb switch statements, a per-harness doctor check, and
several expected-harness lists in tests. That duplication is a defect to
fix, not a pattern to follow — see `backport/wip.md`.

## 4. Verify, live and in the suite

```
make check                       # suite green, including the new round trip
<tool> install <harness>
<harness-cli> <its skills list>  # the tool appears
<tool> doctor                    # clean
<tool> uninstall <harness>       # removes exactly the stamped tree
```

Then hand-edit one generated file and re-run install: it must refuse
(C4.6). The unit tests cover this, so a green suite plus one live
install is enough.

## 5. Stay in scope

A harness target is a projection. It is not the place to adopt the
harness's session model, its permission system, its conversation store,
or its hook vocabulary. Write the out-of-scope list into the handoff
before starting, and leave anything interesting you found as a recorded
finding.
