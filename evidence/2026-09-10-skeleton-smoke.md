# Evidence — skeleton instantiation and install loop

**Date:** 2026-09-10. **toolsmith:** `f9e287e` (main).
**Environment:** `nix develop`, go 1.24.10, linux/amd64.

What this record answers: does `contrib/new-tool.sh` produce a tool that
builds, tests, self-describes, installs, refuses, and uninstalls — with
no hand edits between instantiation and the run? Yes.

## 1. `make check` — instantiate, build, vet, test

```
$ nix develop --command make check
shellcheck contrib/new-tool.sh contrib/check-contract contrib/check-commit-msg \
  skeleton/contrib/check-commit-msg skeleton/contrib/check-gofumpt
rm -rf tmp/smoke
contrib/new-tool.sh smoke --dir tmp/smoke --no-git
instantiated smoke at tmp/smoke (module github.com/procrastivity/smoke)
[checklist printed]
cd tmp/smoke && CGO_ENABLED=0 go build ./... && go vet ./... && go test ./...
ok  	github.com/procrastivity/smoke/internal/cli	0.383s
[20 packages with no test files]
exit=0
```

Shellcheck clean on all five scripts. The instantiated module builds,
vets, and passes its e2e suite with no edits.

## 2. `smoke manifest --json`

```
contract:      toolsmith/v1
schemaVersion: 1
manifest_digest: sha256:7336736196951898c6659884a0cae722714b0bd6fb545eb7b010318dd4ecb63c
verbs:  doctor, install, manifest, uninstall, version
assets: agent-guidance.md, config.default.yaml,
        templates/skills/claude-code/judgment.md
```

C3.4 (digest) and C3.6 (contract declaration) hold in a freshly
instantiated tool, which is the point of putting them in the skeleton
rather than in a migration checklist.

## 3. The install loop

Run against a hermetic `SMOKE_CLAUDE_SKILLS_DIR` (C2.7), so the host's
real skill installs were never touched.

| Step | Result |
|---|---|
| `smoke install claude-code` | `installed claude-code skill at <dir>`, exit 0 |
| tree written | `SKILL.md`, `.claude-plugin/plugin.json`, `.smoke-manifest-stamp.json` |
| `smoke doctor` | `no issues found`, exit 0 |
| `smoke install claude-code --json` (re-run) | `{"harness":"claude-code","dir":"…","status":"current"}`, exit 0 — nothing rewritten |
| hand-edit `SKILL.md`, then `install` | refused, exit 3: "no longer matches what `smoke install claude-code` last wrote (1 file(s) changed since); re-run with `--force`" |
| `uninstall` on the edited tree | refused, exit 3: "remove it by hand if that was intentional" |
| `install --force` | `installed …`, exit 0 |
| `uninstall` on the clean tree | `uninstalled claude-code skill from <dir>`, exit 0 |

This exercises all three comparisons C4.6 keeps distinct: disk-vs-stamp
(the two refusals), binary-and-disk-vs-stamp (`current`), and
binary-vs-stamp (doctor's advisory, silent here because nothing drifted).

## 4. `contrib/check-contract tmp/smoke`

```
no findings — mechanical clauses hold for /home/dev/Code/toolsmith/tmp/smoke
exit=0
```

The checker is clean against the skeleton's own output, which is what
makes its findings against wip and duo readable as facts about those
repos rather than as checker noise
(`evidence/2026-09-10-conformance.md`).

## Not covered here

- `nix build` of an instantiated tool. It fails once by design and
  prints the real `vendorHash` for pasting, so it is a checklist step,
  not an automated gate.
- The release workflow. First exercised by the first tool to tag.
