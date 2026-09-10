# Release and hygiene — the whole story, once

The skeleton ships this already wired. This page says **why** each piece
is shaped the way it is, so nobody "simplifies" a load-bearing detail
back out. Read it once per tool, when you are about to cut a first
release or when a lint rule starts feeling arbitrary.

Normative text: CONTRACT.md C6. Everything below is rationale.

---

## The one rule

**Pushing an annotated `vX.Y.Z` tag is the only human action in a
release.** Everything else derives from the tag. Any process step that
asks a human to also edit a file, also bump a constant, or also
regenerate a document is a place where the two can disagree, and one day
they will.

## Version comes from the tag, and `--match` is not decoration

```make
VERSION := $(shell git describe --tags --match 'v[0-9]*' --always --dirty)
```

`--match 'v[0-9]*'` keeps `git describe` inside the release-tag
namespace. Without it, a milestone tag like `phase-1` wins the describe
and the binary stamps `phase-1-10-gabc1234` — a version that matches no
release. The same pattern is `cliff.toml`'s `tag_pattern`, and the two
must agree, because they answer the same question for different readers.
`toolsmith check` compares them (C6.2).

**Nix stamps the commit; make stamps the tag.** That divergence is
documented in `flake.nix` and deliberately not "fixed". Fixing it means
writing the version into a file, and a second copy of a fact goes stale
in silence. The tag stays the single source of truth (C1.7).

## CHANGELOG.md is generated and never committed

git-cliff builds it from conventional-commit history at release time.
A committed changelog is a copy of the git history that drifts from it,
and reviewers end up arguing about the copy.

Two targets, because they answer different questions:

- `make changelog` → the full history, attached as a release asset;
- `make release-notes TAG=vX.Y.Z` → only that release's slice, header
  stripped, used as the GitHub release body.

`release-notes` probes whether the tag ref exists, then picks
`--current` (CI, where the tag push started the run) or `--unreleased
--tag` (a local dry run before tagging). One target, both situations.

## The release workflow re-gates

`release.yml` runs `nix develop --command make check` before it builds
anything. A tag can sit on a commit that never went through branch CI —
a hotfix tagged locally, a tag moved by hand — and the release path is
the last place to notice.

Assets are **named explicitly, never globbed**. `dist/` also holds
`RELEASE_NOTES.md`, which is the release body, not an asset; a glob
attaches it and the release page gets a duplicate of itself. Naming
assets also means a rename fails the release instead of silently
publishing fewer files.

`SHA256SUMS` rides along so a binary fetched with curl can be verified
without trusting the transport.

## CI never publishes

`ci.yml` runs lint, test, `nix build`, and the cross-compile matrix,
each through `nix develop --command` so local and CI run the same tools
at the same versions. Publishing lives only in the tag-triggered
workflow. One code path writes releases.

**Actions are SHA-pinned** (C6.5). A tag on a third-party action is a
mutable pointer into someone else's repository, and it runs with your
release token. Pin the SHA and keep the version in a trailing comment,
which is what Dependabot expects to update.

Workflows are **copied per tool, not shared by reference** (T17).
Cross-repo workflow coupling cuts against the whole copy-the-contract
posture: a tool must stay buildable and releasable from its own history
alone.

## Conventional commits from commit one

Not because the format is pleasant, but because git-cliff needs it and
retrofitting it means rewriting history you have already published. The
commit-msg hook (`contrib/check-commit-msg`) enforces it locally from
the first commit.

**`make hooks` installs both hook types:**

```
pre-commit install --hook-type pre-commit --hook-type commit-msg
```

The commit-msg stage does **not** install with pre-commit's default, so
a repo that runs plain `pre-commit install` has a commit-msg hook in its
config that never fires. wip shipped exactly that gap for months; ste9
found it; the skeleton ships the fix (T15). The checker tests for it.

The full test suite is deliberately **not** a per-commit hook. Running
tests before pushing is a norm; making it a commit gate trains people to
pass `--no-verify`, which disables the fast checks too.

## Lint posture

golangci-lint schema v2, `default: none`, then an explicit enable list:
`govet staticcheck errcheck unused revive forbidigo`, with gofumpt as
the formatter. Starting from none and adding is a decision per linter;
starting from the default set and disabling is an argument per linter,
repeated at every upgrade.

The forbidigo rules ban `fmt.Print`, `fmt.Println`, and `fmt.Printf`.
That is the mechanical half of the streams discipline (C2.1) — the
structural half is that every verb receives its writer pair at
construction. `Fprint*` variants take an explicit writer and stay
allowed.

## `.gitignore` and tool-generated directories

Cover `/CLAUDE.local.md`, build output, and Nix litter. Beyond that, any
directory the tool generates inside a user's repo (wip's `.wip/`) needs
an **explicitly decided** ignore posture, written down. wip's landed in
`.git/info/exclude`, which is per-clone, invisible to collaborators, and
in tension with its own doctor check that refuses a tracked `.wip/`. A
decision recorded in the repo is fine; an accident of local git config
is not (C6.8).

## Cutting the first release

```
make check                       # green
make release-notes TAG=v0.1.0    # read what the release will say
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

Do this early — at v0.1.0, when nothing depends on it. A release process
first exercised at v1.0 is a release process debugged in public.
