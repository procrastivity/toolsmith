# Evidence — conformance runs against wip and duo

**Date:** 2026-09-10. **Checker:** `contrib/check-contract` at toolsmith
`f9e287e`.

Two purposes: seed `backport/wip.md`, and validate the checker itself
against real repos that predate the contract. A checker that reports
nothing on a tool known to have gaps is broken; so is one that reports
gaps a clean skeleton also trips
(`evidence/2026-09-10-skeleton-smoke.md` covers the second half).

---

## wip — `/home/dev/Code/wip`, branch `go`, `5be0507`

```
C3.4: manifest --json carries no manifest_digest
C3.6: manifest --json declares no toolsmith contract version
C6.5: ci.yml: action not SHA-pinned: - uses: actions/checkout@v4
C6.5: ci.yml: action not SHA-pinned: - uses: DeterminateSystems/nix-installer-action@v16
C6.5: ci.yml: action not SHA-pinned: - uses: actions/checkout@v4
C6.5: ci.yml: action not SHA-pinned: - uses: DeterminateSystems/nix-installer-action@v16
C6.5: ci.yml: action not SHA-pinned: - uses: actions/checkout@v4
C6.5: ci.yml: action not SHA-pinned: - uses: DeterminateSystems/nix-installer-action@v16
C6.5: ci.yml: action not SHA-pinned: - uses: actions/checkout@v4
C6.5: ci.yml: action not SHA-pinned: - uses: DeterminateSystems/nix-installer-action@v16
C6.5: release.yml: action not SHA-pinned: - uses: actions/checkout@v4
C6.5: release.yml: action not SHA-pinned: - uses: DeterminateSystems/nix-installer-action@v16
C6.6: make hooks does not install the commit-msg stage
C7.5: no README.md
14 finding(s) for /home/dev/Code/wip
exit=1
```

Five distinct conditions (the twelve C6.5 lines are one condition across
four CI jobs and the release workflow). Every one was predicted before
the checker existed, which is the result this run was for. All five are
written up with fixes in `backport/wip.md`.

No false positives: wip passes every other mechanical clause, including
`CGO_ENABLED=0` in all three places, the `--match`/`tag_pattern`
agreement, CHANGELOG.md absent from the index, and the forbidigo rules.

## duo — `/home/dev/Code/duo`, branch `go`, `7763ef1`

The repo's working tree sits on another branch, so the `go` branch was
extracted with `git archive` into a scratch tree and audited there. The
working tree (`amp-exclusive-writer`, `36ce118`) reports the same 11
findings — the Go sources are shared.

```
C3.1: `duo manifest --json` failed or is not implemented
C6.5: [ten action-not-SHA-pinned lines, ci.yml x4 jobs and release.yml]
11 finding(s)
exit=1
```

The C3.1 finding is real and is the interesting one. duo emits its
manifest under `--output json`, not the contract's global `--json`
(C2.3), and the document underneath uses its own vocabulary throughout:

```json
{"schema":"duo.manifest/v1","product":{...},
 "manifest_digest":"sha256:349d7ac…","operations":[{"projectability":…}]}
```

against the contract's `schemaVersion` + `contract`, `tool`, and
`verbs` with `kind`. This is a genuine surface divergence between two
tools that otherwise share a design, found mechanically. It is duo's
call how to resolve it; the contract's answer is `--json`.

Note what the finding hides: because the invocation fails, the checker
never reads the document, so C3.4 and C3.6 go unreported. duo in fact
carries `manifest_digest` — it is the tool T7 adopted it from — and it
declares no toolsmith contract version. One flag mismatch masks three
clauses, which is worth knowing when reading any C3.1 finding.

The C6.5 findings are against this branch. duo's `main` already
SHA-pins its actions, which is where T16 came from — so the two branches
disagree, and this run says which one is behind.

## What these runs do not say

The checker audits the `[check]`-marked clauses only. A clean run is not
conformance: the chassis discipline, the install model's semantics, the
asset chain, and the docs conventions are all audited by reading. Both
repos have more contract surface than these findings touch.
