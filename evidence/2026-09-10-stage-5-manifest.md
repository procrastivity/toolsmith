# Evidence — Stage 5: the manifest populated, and what that exposed

**Date:** 2026-09-10. **toolsmith:** `07b5d6b` (main, clean tree).
**Environment:** `nix develop`, go 1.24.10, linux/amd64.

`assets/playbook/migrate.md` Stage 5 says the manifest "is not a stage. It
is a consequence: it grows as verbs register, and it needs no work beyond
the annotation each verb already carries. Check it after each verb
(`<tool> manifest --json`) and treat any surprise as a defect in the verb,
not in the manifest."

The check was run against C3.1–C3.8 clause by clause rather than eyeballed,
and it turned up two surprises. Both were defects in the manifest. This
record is what held, what did not, and the green run at the fix commit.

Three commits: `20a3225`, `9cba3cd`, `07b5d6b`.

---

## 1. What the audit found

### 1.1 The asset walk omitted exactly the payload `new` writes

`internal/manifest/assets.go`'s `isShippedAsset` excluded every path ending
in `.go`. Its comment justified that for one file — `assets/assets.go`,
whose own `//go:embed` directive matches itself — but the predicate was a
blanket suffix test.

Measured at `9bcc163`, before the fix:

```
assets listed:                  32
files on disk under assets/:    65
omitted:                        assets.go  +  32 _skeleton/**/*.go
`toolsmith new demo` writes:    32 .go files
```

The omitted set was `assets.go` (correctly excluded) plus exactly the Go
sources of the chassis. `cmd/toolname/main.go` and all of
`_skeleton/internal/**` — the entire thing `new` puts on disk — were
invisible to the manifest.

C3.3 lists assets with checksums "so drift and tampering are detectable",
and C3.4's `manifest_digest` is computed over that list. Editing
`_skeleton/internal/cli/root.go` moved neither. A tool instantiated from a
tampered skeleton was undetectable by the mechanism built to detect it.

The fix narrows the exclusion to what the comment describes: the assets
package's own top-level Go source, never a `.go` file in a subdirectory.

One latent inconsistency had to be fixed with it. `walkDiskDir` passed the
**absolute** walk path to the predicate while `walkEmbedded` passed the
**embed-relative** one. A suffix test cannot tell those apart, so the bug
hid it; a top-level-only test can, so both branches now pass the relative
form.

The defect predates Stage 4 — `skeleton/` moved under `assets/` at Stage 3
step-08 — but it was inert until `new` shipped, because nothing else made
the skeleton a payload. Which is the argument for Stage 5 existing at all.

### 1.2 Positional arguments had no field, and the projection showed it

`check [path]` and `new <name>` are the first verbs whose positional
argument carries meaning. The manifest had no field for one, so the
generated claude-code skill rendered:

```
| `toolsmith new`   | instantiate the chassis skeleton as a new tool |
| `toolsmith check` | audit a tool repo against the mechanical ...    |
```

An agent reading the projection — the manifest's main consumer — could not
tell that `new` requires a name. `Verb.Usage` now records the positional
portion of the verb's own cobra `Use` line verbatim. Recorded, never
parsed: see C3.8 and T23 for why structure would be speculation.

### 1.3 Both fixes had to be applied twice, by hand

`assets/_skeleton/internal/` is a `toolsmith` → `toolname` substitution of
toolsmith's own `internal/`, with three deliberate divergences (the
`Contract` const, the `skillDescription` TODO, the env prefix). Nothing
checks that. A chassis fix that lands in one tree and not the other is
silent: `make smoke` still builds green while shipping the old bug.
Backlogged, not built here — it is C6 hygiene, not a manifest consequence.

### 1.4 What held

| clause | verdict |
|---|---|
| C3.1 | tool identity, `schemaVersion`, `contract`, verbs, assets all present; no `backends` field, and none is owed — a harness is an install target (C4.2), not a provider seam |
| C3.2 | every verb carries a kind; the walk's hard error on a missing one is now covered by a test, which the e2e suite cannot reach |
| C3.3 | all 66 checksums verified against the files on disk after the fix; no mismatch, nothing listed that is not there |
| C3.4 | verified in substance, not just by prefix — see §3 |
| C3.5 | holds — the install-state file is the stamp; its filename contains "manifest" only as a reference to what it was stamped from, which the code says in place |
| C3.6 | `"contract": "toolsmith/v1"` |
| C3.7 | `outputSchema` absent on all seven verbs — the field exists, unfilled |

## 2. `toolsmith manifest --json`

```
contract: toolsmith/v1     schemaVersion: 1     assets: 66
manifest_digest: sha256:0f47ddd15db6b6ce4f105cc515853da96bf83828511d7c7b3360f82ce37aeabb
```

| verb | kind | usage | args |
|---|---|---|---|
| `check` | `plumbing` | `[path]` | — |
| `doctor` | `plumbing` | — | — |
| `install` | `plumbing` | `<harness>` | `force` |
| `manifest` | `plumbing` | — | — |
| `new` | `plumbing` | `<name>` | `dir`, `module`, `no-git` |
| `uninstall` | `plumbing` | — | — |
| `version` | `plumbing` | — | — |

Assets went 32 → 66: the 34 skeleton Go sources that were missing (32 at
the time of the finding, plus the two test files added with the fix).
`schemaVersion` does not move — an additive field is not an incompatible
shape change, which is what that integer is for.

## 3. The digest, gated rather than asserted

C3.4's only prior assertion was that `manifest_digest` starts with
`sha256:`. Nothing checked that it committed to anything. That property is
what makes the asset list load-bearing, so it is now gated two ways:

- **self-commitment** — the emitted digest is a sha256 over the exact bytes
  the caller received with the digest field blanked, checked by blanking
  the raw stdout rather than re-encoding a decoded document, so the test
  reproduces what an external consumer can do and stays independent of the
  producing code's own marshalling.
- **determinism** — three runs of the same binary emit byte-identical
  documents, so neither walk may vary with map or filesystem iteration
  order.

Both are also in the skeleton's e2e suite, so a new tool inherits them.

## 4. The green run

```
make check              exit=0
make smoke              exit=0
make parity             passed: 52   failed: 0
toolsmith check .       no findings — mechanical clauses hold      exit=0
toolsmith install claude-code                                       exit=0
toolsmith doctor        no issues found                             exit=0
```

The parity gate is unchanged in shape from
`evidence/2026-09-10-parity-green.md`; its `new` tree cases now compare 52
files rather than 50, because both implementations write the skeleton's
two new test files. Nothing in this Stage touched
either implementation's behaviour on a corpus repo.

The instantiated tool inherits both fixes: `tmp/smoke`'s own manifest
reports `install <harness>` and `uninstall <harness>` with usage, and
excludes only its own `assets.go`.

## 5. Carried forward

- **To Stage 8.** C3.8 landed without a contract version bump. T20 says an
  additive clause bumps the minor; `CONTRACT.md` reads "v1" with no minor,
  and C3.6 pins the declared string to the major. Giving the contract a
  minor means deciding whether the declared string carries it — a T-number
  of its own, and Stage 8's business, not a Stage 5 audit's.
- **To the playbook.** Stage 5 tells you to treat a surprise as a defect in
  the verb. Both surprises here were defects in the manifest, surfaced by
  verbs that introduced a new *kind* of thing to describe — a written
  payload, a positional argument. The page should say that finding such a
  gap is the stage's deliverable, not a sign the port went wrong.
  `clast-conversion` re-reads this.
- **Backlogged.** The skeleton/`internal` drift gate (§1.3), and C4.2's
  duplicate-registration panic, which the registry does not implement.
