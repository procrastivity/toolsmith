# Upstream feedback — `xmr`

Gaps that the `xmr` conversion found in the contract, the skeleton,
the playbook and the checker. This is the inbound counterpart of
`backport/`: a backport list carries fixes from toolsmith into a tool;
a feedback list carries a tool's findings into toolsmith. Nothing here
is ratified. An item that changes the contract needs a DECISIONS.md
entry first (README, "Evolving the conventions"); an item that changes
only the skeleton, the playbook or the checker can land as an ordinary
fix.

xmr is a private, GitLab-hosted tool built with `toolsmith new`. It
converts three agent plugins into one binary that projects five
merge-request review and test-convention skills into claude-code, pi
and opencode. Its CI is GitLab CI. The items come from its migration
and from its reconcile to contract v1.6 (2026-11-09, toolsmith
`v0.0.3-2-g92fcfcf`). Its repository is private, so each item below
says what to look at in toolsmith, not in xmr.

Each item names its clause or file, so a session can pick one up
alone. Items 2 to 5, and the notes test in item 6, also apply to
toolsmith's own copy of the chassis.

---

## Skeleton and chassis

**1. A fresh skeleton is stale at birth (C3.10, T25).**
`assets/_skeleton/internal/manifest/manifest.go` writes
`ContractReconciledMinor: 2`, and the skeleton e2e test pins 2. The
contract is at v1.6, so `toolsmith check` on a tool that `toolsmith new`
just made reports C3.10 stale. That contradicts `bootstrap.md` §5: "A
fresh tool built from the skeleton should report no findings". The
value also reads as "a human reconciled this tool at v1.2", and no such
reading happened: 2 is toolsmith's own recorded minor, copied into
every new tool. `reconcile.md` tells the first reconcile to find the
provenance of the value and not guess a baseline. xmr did that and
read every history row from v1.1. Options: (a) `new` writes the
current minor, because the skeleton is the copy that toolsmith keeps
reconciled; (b) the skeleton writes a value that the checker reports
as "never reconciled", so the first reconcile is visibly due. Either
way, the playbook should say where a new tool's value comes from.

**2. `TestResolveInPermissionErrors` fails in a root CI job.**
`internal/asset/asset_test.go` (`installedTestCommand`) creates its
prefix with `os.MkdirTemp("", …)`, under `TMPDIR`. As root, the test
re-runs its binary as uid 65534 from that prefix. `nix develop` sets
`TMPDIR` to a root-only 0700 directory, so the child cannot traverse to
the prefix and the exec fails with "permission denied". This shows in
any CI that runs as root inside `nix develop` (xmr's GitLab CI uses the
`nixos/nix` image). toolsmith's own GitHub CI runs on `ubuntu-latest`
as a non-root user, so it does not show there. xmr's fix: when euid is 0, create the prefix
under `/tmp` (mode 1777) instead of `TMPDIR`.

**3. The e2e `hermeticEnv` fails open (C2.7).**
`internal/cli/e2e_test.go` isolates only the seams in its list
(`TOOLNAME_CLAUDE_SKILLS_DIR`). A tool that adds a harness and forgets
to add its seam lets the e2e suite write into the host's real harness
directories. The comment says to add each seam, but nothing enforces
it. xmr also sets `HOME` to a temp dir and clears
`PI_CODING_AGENT_DIR` and `XDG_CONFIG_HOME`, so a missing seam lands in
a temp dir. The skeleton could do the same.

**4. `harness.Projectable` filters by kind only (C4.3).**
C4.3 (v1.4) says that where the C8.3 boundary stands, the projectable
set is the `plumbing` namespace's `plumbing`-kind members. The
skeleton's `internal/harness/harness.go` keeps every `plumbing`-kind
verb, and the skeleton's porcelain verbs (`install`, `doctor`,
`manifest`, `uninstall`, `version`) are `plumbing`-kind. When xmr drew
the boundary, all five porcelain verbs reached every skill's verb
table. xmr's fix: when the namespace exists, also require the
`"plumbing "` name prefix, with a unit test. The skeleton has no
namespace by default, so the fix must keep today's behavior until a
tool adds one.

**5. `version` and `manifest` accept stray positionals (C3.8).**
Neither verb declares `Args`, so `<tool> version extra` exits 0, while
the manifest records no usage for the verb. `toolsmith version extra`
behaves the same. xmr declares `cobra.NoArgs` on both and adds a test
that runs every verb that records no usage with a stray positional and
expects a usage error (exit 2).

**6. The release driver's prose is GitHub-only, and its notes test
passes a blank file.**
`contrib/release` names `.github/workflows/release.yml`, "the GitHub
release" and `${WEB_URL}/actions`. The Makefile's `release-notes`
comment names "the GitHub release body". A GitLab tool has to find and
rewrite each one. Separately, `contrib/release` tests the notes with
`[[ -s dist/RELEASE_NOTES.md ]]`. A notes file that holds only a
newline passes, and the tag message and the release body are blank.
xmr tests for content with `grep -q '[^[:space:]]'`. The second part
applies to GitHub tools too.

**7. C5.5 does not name the refusal's code family.**
C5.5 says that an invalid root, a symlink, a nonregular file or a read
error "refuses", but names no code. "Refuses" can read as C2.4's
refusal (`refusal.*`, exit 3). toolsmith's own `doc` verb uses
`refusal.project-asset` (exit 3). The skeleton ships `ResolveIn` but
no verb that calls it, so a new tool copies no precedent. xmr's
`plumbing asset` reports these cases as `not-found.asset` (exit 1): it
refuses and never falls through, but the family says "not found".
Options: C5.5 names the family, or the skeleton ships the code that
toolsmith's `doc` uses.

**12. The stale-artifact advisory names only the harness and the file
(C4.6, C4.7).**
(Added after items 8 to 11, so it keeps the next free number.)
`staleFileFindings` in `assets/_skeleton/internal/checks/checks.go`
builds the `advisory.stale-harness-artifact` message from the harness
name and the file path only: "found: <harness> harness artifact <path>
<reason> since last install; …". The skeleton's registry has one
target per harness, so each line is unique. xmr splits each harness
into five targets, one per skill, and the main file of every skill is
`SKILL.md`. After an upgrade, `xmr doctor` printed five identical
lines per harness, fifteen in all. The per-target state lines above
them name the skill, but the findings do not. The `modified`,
`unowned` and `incompatible` findings do not have this problem,
because `harness.Risk` names the install dir. xmr's fix: put the
target's label (the skill name) in the message, for example "found:
claude-code xmr-review artifact SKILL.md changed since last install;
…", with a test that installs two skills in one harness and expects
two different findings, each with its skill name. Options: the
skeleton gives `staleFileFindings` a target name now, or
`assets/playbook/new-harness-target.md` says to add one when a harness gets
more than one target. toolsmith's own `internal/checks/checks.go` has
the same message.

---

## Contract

**8. Skill-only shapes (C8.1).**
C8.1 describes a shape in two forms: a top-level `llm` verb and a
projected skill. All five xmr shapes are interactive and agentic: they
stop for the user's go-ahead, edit code, run project tests and commit.
A one-shot `llm` verb cannot do that work, and an embedded agent loop
is out of scope. So each shape ships the flow asset and the skill form
only. The flow asset still gives one definition with live delivery, and
every deterministic step is still a plumbing verb (C8.2). xmr recorded
"C8.1 holds with a note". C8 could say explicitly that a shape may ship
the skill form only.

**9. GitLab CI: a second witness for the GitLab-equivalence draft
(C6.5, C6.4, C1.1).**
xmr is the second tool with GitLab CI, after ste9 (see TOOLS.md). Its
`toolsmith check` reports the same two standing findings:
`no .github/workflows/ci.yml` and `no .github/workflows/release.yml`.
The clauses' intent holds in `.gitlab-ci.yml`: lint, test, `nix build`
and cross-compile jobs through `nix develop`; every image
digest-pinned; publish jobs gated on `$CI_COMMIT_TAG`; the tag pipeline
re-runs lint and test before anything publishes; release assets named
explicitly; `CGO_ENABLED: "0"` in top-level `variables`. One file does
the work of `ci.yml` and `release.yml`, and the release notes come from
the annotated tag message that `contrib/release` writes.

The draft that ste9 wrote covers this. What xmr adds:

- **C1.1 reads clean, not unaudited.** `cgoEnabled()` audits
  `.github/workflows/ci.yml` only if it exists, and relies on C6.5 to
  report a missing file. On a GitLab tool the CI leg of C1.1 is never
  read, and no C1.1 finding says so. If a GitLab mapping retires the
  two C6.5 findings without the C1.1 leg, a GitLab tool loses its CI
  `CGO_ENABLED` audit silently. The draft's checker work order already
  includes this leg; keep it in the scope that gets ratified.
- **The draft's number is taken.** ste9 drafted the mapping as T37,
  and the ste9 row in TOOLS.md still says so. DECISIONS.md has since
  given T37 to the curl installer and T38 to C5.5, so the mapping needs
  a new number when it is ratified.

---

## Playbook

**10. `migrate.md` and `bootstrap.md` still say to paste the
`vendorHash` (T36).**
`migrate.md` Stage 3 says "`nix build` fails once and prints the real
`vendorHash`. Paste it in." `bootstrap.md` §3 says the same. Since T36,
the skeleton ships `vendorHash = null`, and the checklist that `new`
prints already says "there is no hash to paste".

**11. git-cliff in a linked worktree.**
`make release-notes TAG=v0.1.0`, run in a linked git worktree inside
the main checkout (`.claude/worktrees/<name>`), wrote notes that held
one newline. git-cliff logged "Including changes from the current
directory: ../../../.claude/worktrees/…": it took the main checkout as
the repository and kept only commits that touch the worktree's path.
This was not tested further, because a release runs from the main
checkout. `release-and-hygiene.md` could say to cut a release from the
main checkout. Item 6's content test would refuse those notes.
