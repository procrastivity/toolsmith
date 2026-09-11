# Back-port punch list — `wip`

Fixes flowing back into `wip` (branch `go`, module
`github.com/procrastivity/wip`) now that the conventions it produced are
written down. wip is the reference implementation, so a gap here is a
gap in the fleet's most-copied source.

Audited 2026-09-10 at `5be0507`. The mechanical half comes from
`contrib/check-contract` (14 findings, `evidence/2026-09-10-conformance.md`);
the rest from reading.

Nothing below is urgent. Each item names its clause, so a session can
pick one up alone.

---

## Mechanical — the checker's 14 findings

**1. No README.md (C7.5).**
wip ships without one. A newcomer cloning the `go` branch gets no
statement of what the tool is, the two-step install, or where the design
of record lives. The skeleton ships a README template so absence is the
anomaly; wip predates it.

**2. `make hooks` misses the commit-msg stage (C6.6).**
`.pre-commit-config.yaml` declares the conventional-commit hook with
`stages: [commit-msg]`, and `make hooks` runs
`pre-commit install` without `--hook-type commit-msg`. The hook is
configured and never fires. Fix:

```make
hooks:
	pre-commit install --hook-type pre-commit --hook-type commit-msg
```

ste9 found this first (T15). Every commit wip made under the "enforced
from commit one" decision (H4) was in fact unenforced locally.

**3. GitHub Actions are not SHA-pinned (C6.5).**
Twelve uses across `ci.yml` (four jobs) and `release.yml`:
`actions/checkout@v4` and
`DeterminateSystems/nix-installer-action@v16`. A tag is a mutable
pointer into someone else's repository, and `release.yml` runs with a
release token. Pin the SHA, keep the version as a trailing comment so
Dependabot can still update it. duo's main branch already does this
(T16); copy its lines.

**4. The manifest carries no `manifest_digest` (C3.4, T7).**
Adopt duo's self-committing digest: sha256 over the manifest's own
canonical JSON with the digest field blanked. It makes the manifest a
comparable identity, which is what lets a drift check compare without
re-walking. The skeleton's `internal/manifest/digest.go` is the copyable
implementation.

**5. The manifest declares no contract version (C3.6, T8).**
Add `"contract": "toolsmith/v1"`. Conformance should be readable from
the tool, not asserted from memory — this is what `check-contract` and
TOOLS.md want to read.

---

## Non-mechanical

**6. Per-harness `install.go` is copied four times verbatim.**
`internal/harness/{codex,devin,opencode,pi}/install.go` are
byte-identical once the harness name is normalized — 93 lines each.
`amp` and `claudecode` differ only where their generated file sets
differ. The registry table (C4.2) already removed the switch statements
from the verbs, and `guards.CheckStaleHarnessArtifacts` already walks
`registry.All`, so the duplication that remains is entirely inside the
per-harness packages.

Fix: hoist the shared `Install`/`Uninstall` bodies into
`internal/harness` as functions parameterized by `{Name, InstallDir,
Generate}` — the same three fields the registry row already carries —
and leave each package holding only its paths, its `Available` probe,
and its `Generate`. Keeping the packages policy-free (C4.3) is what
makes this safe: there is no per-harness policy in those bodies to lose.

Related, smaller: the six single-harness
`CheckStale<Name>HarnessArtifact` wrappers in
`internal/guards/staleharness.go` now exist only for their own tests.
Either point those tests at the parameterized helper, or keep the
wrappers and say in one comment that they are test affordances.

**7. Adopt the six-state drift vocabulary (C4.5, T6).**
wip reports drift as per-file `added/removed/changed`. That is the floor
and it maps in, but the tool-level states —
`current | missing | stale | modified | unowned_conflict | incompatible`
— are what a caller wants to switch on, and they are what duo, the
skeleton, and this contract all speak. The per-file report stays; it
becomes the detail under `modified`.

**8. `docs/install-target-devin.md` is untracked and stale.**
It is a completed handoff (the Devin target shipped) sitting outside
git. Its grep list names `claude-code, codex, pi, opencode` — it
predates both `devin` landing and `amp`, so following it today misses a
call site. Decide one of:

- delete it — `assets/playbook/new-harness-target.md` in toolsmith now covers
  the genre, and the specific Devin facts live in the shipped package;
- or commit it under `docs/<matter>/` as a dated research record, with
  its own header saying the target shipped and the wire-up list is
  historical.

Either way it stops being an untracked file that reads like current
instructions.

**9. `.wip/` is excluded per-clone, not by decision (C6.8).**
`.wip/` appears in `.git/info/exclude` and in no committed file. That
exclude is invisible to collaborators and to a fresh clone, and it sits
in tension with wip's own doctor check, which refuses to render into a
tracked `.wip/`. The posture may well be right; it needs to be a
recorded decision in the repo, not an artifact of one machine's git
config.

**10. `OutputSchema` is declared and never set (C3.7).**
`manifest.SetOutputSchema` has no callers; the field is always omitted.
The clause permits the field and forbids filling it speculatively, so
both answers conform. Decide explicitly:

- **keep** — write one line saying it is a reserved slot awaiting the
  first verb with a JSON payload worth describing; or
- **drop** — delete the setter and the field, and let the first real
  consumer reintroduce it.

An undecided third state is the only wrong answer, because it reads to
the next person as an unfinished feature.

**11. The skill description is an inline format string in six targets (C4.4, T27).**
Each of wip's six harness targets writes its `SKILL.md` frontmatter
description from the same Go format string, `"description: Track and drive
%s work — Matters, Stages, Steps — through its verb surface.\n"`:
`internal/harness/{claudecode,codex,opencode,pi,devin,amp}/*.go`. The
description decides whether an agent loads the skill, so it is tunable
prose (C5.1), and C4.4 sends hand-written prose through the asset chain.
T27 does not cover it: T27 exempts fixed template text around manifest
fields, and this sentence is prose with the tool name inside it.

toolsmith's chassis moved its copy to
`assets/templates/skills/claude-code/description.txt`
(contract-v1-2-reconcile step-10). A shared asset for all six targets fits
wip better than six copies, which item 6's copied `install.go` already
shows the cost of.

claude-code's `plugin.json` suffix, `"%s's own generated plumbing-verb
skill."`, conforms under T27 as it is.

---

## Not a defect

Recorded so nobody "fixes" them:

- **Nix stamps the commit, make stamps the tag.** Documented in
  `flake.nix` and deliberately not reconciled (C1.7). A VERSION file
  would be a second copy of a fact.
- **CHANGELOG.md absent from the index.** Correct (C6.3).
- **`WIP_*_SKILLS_DIR` env vars.** Test seams, and comments say so
  (C2.6).
