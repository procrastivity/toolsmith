# Evidence — toolsmith's own dev shell and hooks

**Date:** 2026-09-10. **toolsmith:** `1b1182f` (main).
**Environment:** `nix develop`, go 1.24.10, linux/amd64.

The other two records verify what toolsmith *produces*. This one
verifies toolsmith itself: the flake gives a working shell, `make check`
is green at HEAD, and both hook stages fire. A repo that preaches C6
hygiene has to hold it.

## 1. `nix develop --command make check`

```
shellcheck contrib/new-tool.sh contrib/check-contract contrib/check-commit-msg \
  skeleton/contrib/check-commit-msg skeleton/contrib/check-gofumpt
rm -rf tmp/smoke
contrib/new-tool.sh smoke --dir tmp/smoke --no-git
instantiated smoke at tmp/smoke (module github.com/procrastivity/smoke)
[checklist printed]
cd tmp/smoke && CGO_ENABLED=0 go build ./... && go vet ./... && go test ./...
ok  	github.com/procrastivity/smoke/internal/cli	0.361s
[20 packages with no test files]
exit=0
```

Same result as the `f9e287e` run in
`evidence/2026-09-10-skeleton-smoke.md`. The three commits since that
run touched `TOOLS.md`, `backport/`, `evidence/`, and `flake.lock` only
— no skeleton source — so that record's install-loop table still
describes HEAD.

## 2. `contrib/check-contract tmp/smoke`

```
no findings — mechanical clauses hold for /home/dev/Code/toolsmith/tmp/smoke
exit=0
```

Still clean at HEAD, so the wip and duo findings in
`evidence/2026-09-10-conformance.md` remain readable as facts about
those repos.

## 3. `make hooks` — both stages

```
$ nix develop --command make hooks
pre-commit install --hook-type pre-commit --hook-type commit-msg
pre-commit installed at .git/hooks/pre-commit
pre-commit installed at .git/hooks/commit-msg
```

Both files land. One `pre-commit install` with no `--hook-type` writes
only the first, which is exactly the wip gap C6.6 names and
`backport/wip.md` carries a fix for.

The commit-msg hook was then run against two messages:

| Message | Result |
|---|---|
| `this is not conventional` | exit 1 — "commit message must be conventional-commit shaped: `<type>(<scope>)?: <description>`", types listed, offending line echoed |
| `chore: probe hook` | exit 0 — `conventional commit message ... Passed` |

The gate rejects and admits, so C6.6 holds here by execution rather than
by the config file's presence. This closes the plan's verification
item 1.

## Not covered here

- `nix build` of toolsmith. Phase A ships no package output on purpose
  (`flake.nix`); Phase B's binary adds one.
- The pre-commit stage's own hooks beyond what a hook run touches. They
  run per commit in normal use.
