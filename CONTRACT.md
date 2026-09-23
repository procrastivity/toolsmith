# The toolsmith contract — v1.4

**Status: normative.** This document is the cross-tool contract for
procrastivity-style CLI tools. A tool that conforms declares
`"contract": "toolsmith/v1"` in its manifest (C3.6). Amend this document
only by decision (DECISIONS.md).

**Versioning (T20, as amended by T24).** Additive clauses bump this
document's minor version; breaking changes bump the major. The version a
tool *declares* carries the major only — `toolsmith/v1`, never
`toolsmith/v1.1` — so an additive clause never puts a shipped tool into
a "behind" state it must cut a release to leave. That would be exactly
the cross-repo version coupling T2 rejects, in exchange for a claim
nothing verifies. Conformance is instead **derived**: the checker reports
which clauses it audited and how they fared, so a tool's level is a
measured fact rather than a self-report (T24). The minor lives here, in
the clause history below, and is what a staleness check compares a
tool's last reconciliation against.

The contract descends from wip's packaging commitments and is now proven
by three implementations: `wip` (the reference), `duo` (go branch), and
`ste9` (in flight). Clauses adopted from a later tool cite it.

One principle underneath everything: **the system-wide binary is the
source of truth, and every agent-harness integration is a derived
artifact generated from it.** The harness layer is not a thing you
author. It is a thing the binary emits.

Clauses are numbered for citation — from tool code comments, from
`toolsmith check` findings, and from workplans. Sub-clauses
marked **[check]** are mechanically verifiable.

---

## C1 — Language and build

- **C1.1** The tool is a single static Go binary built with cobra.
  `CGO_ENABLED=0` in the Makefile, CI, and the Nix package — all three.
  **[check]** covers all three: the Makefile, `.github/workflows/ci.yml`,
  and `flake.nix`, each for the setting anywhere in the file, not at
  every build it governs. This is load-bearing, not a default:
  it is what makes the single-binary/cross-compile promise hold
  regardless of later dependency choices. If the tool needs SQLite, this
  constrains the driver to a pure-Go implementation (modernc.org/sqlite),
  never mattn/go-sqlite3.
- **C1.2** **[check]** (that `cmd/` holds a `<tool>` directory, not
  that it is a main package) `cmd/<tool>/main.go` does nothing beyond
  constructing streams and build info, calling the root command, and
  mapping the result to an exit code (~30 lines).
- **C1.3** `internal/cli/root.go` is the single registration point. It
  is the only place any verb package gets imported. No `init()` side
  effects anywhere else.
- **C1.4** One package per verb under `internal/verbs/<verb>/`, each
  exposing `Command(streams *iostreams.Streams, ...) *cobra.Command`.
  Domain packages sit beside `verbs/`, never inside it.
- **C1.5** Dependencies stay deliberately tiny. Adding one is a
  decision, not a convenience.
- **C1.6** **[check]** (that `flake.nix` exists, that `.envrc`
  contains `use flake`, and, when `assets/` exists, that `flake.nix`
  mentions `share/`) The repo carries `flake.nix` (buildGoModule, nixos
  release channel, flake-utils) and `.envrc` (nix guard that soft-fails
  with an installer hint, then `use flake`). The flake's `postInstall`
  copies `assets/` to `$out/share/<tool>/assets` and removes `assets.go`
  from the installed tree.
- **C1.7** Nix stamps the commit; make stamps the tag. The divergence
  is documented in flake.nix and never "fixed" with a VERSION file: the
  tag is the single source of truth, and a second copy would go stale
  in silence.

## C2 — Chassis

- **C2.1** Every verb writes through an injected `iostreams.Streams`
  writer pair. The discipline is structural (threaded in) and
  mechanical: **[check]** `.golangci.yml` carries the forbidigo rules
  banning `fmt.Print(`, `fmt.Println(`, `fmt.Printf(`.
- **C2.2** stdout carries exactly one thing: the payload. Diagnostics
  go to stderr. On failure, stdout is empty, except for the findings
  verdict that C2.5 sanctions.
- **C2.3** Two global flags, bound once at root and never redeclared by
  a verb: `--json` (emit the success payload as one JSON value) and
  `-v/--verbose` (extra diagnostic lines on stderr).
- **C2.4** The exit-code table is closed: `0` success, `1` user-level
  failure, `2` usage, `3` refusal, `4` internal. A new code gets added
  only when a real case does not fit — never speculatively. One
  sanctioned extension exists: `exitcode.Silent` — a non-zero exit that
  writes nothing to stderr, for verbs whose output bytes are owned by a
  parity contract (ratified from ste9, T5).
- **C2.5** One structured error type: `<tool>err.Error{Code, Message}`
  where `Code` is a dotted machine token (`refusal.*`, `validation.*`,
  `not-found.*`, `advisory.*`, `internal.*`). Exit codes map from the
  code prefix, not from Go types. Exactly one form outside the five
  prefixes is sanctioned: `<verb>.findings-present`, exit 1, for a verb
  whose run completed and whose own findings are the verdict (`doctor`
  under C4.7, the conformance checker under T21). Such a verb's findings
  stay on stdout as its payload, and the error is the verdict about them
  (T26). One `Render` produces the human line
  (`<tool>: <verb>: <message>`) and the `{"error":{code,message}}`
  envelope from the same value — no second code path that could
  diverge. Root sets `SilenceUsage`/`SilenceErrors`; cobra never prints.
- **C2.6** Test seams are package-level var indirection
  (`var userHomeDir = os.UserHomeDir`), never interfaces invented for
  tests. Env vars of the form `<TOOL>_*_DIR` are test seams only, and
  say so in a comment; production code never reads them for any other
  purpose.
- **C2.7** E2E tests build and drive the real binary: `TestMain` runs
  `go build`, helpers return `{stdout, stderr, exitCode}`, and a
  hermetic env forces every `<TOOL>_*_SKILLS_DIR` seam to a temp dir so
  the suite can never touch the host's real installs.

## C3 — The manifest

- **C3.1** **[check]** (that `manifest --json` runs, exits 0,
  and carries a non-zero `schemaVersion`) `<tool> manifest --json`
  emits the tool's machine-readable self-description: tool identity
  (name, version, commit, date), `schemaVersion`, verbs, assets, and
  any registered backends.
- **C3.2** Every verb carries a surface kind, recorded as a cobra
  annotation at the bottom of its constructor
  (`surface.Annotate(cmd, surface.Plumbing)`): `plumbing`
  (deterministic output contract — stable JSON, stable exit codes, no
  LLM), `llm` (CLI-side LLM shaping), `control-plane` (needs a live MCP
  surface). The manifest walk hard-errors on a verb with no kind. This
  is the mechanical enforcement point of the porcelain/plumbing split —
  never inferred from folder names. The kinds carry the determinism
  axis alone; T29 records what the phrase once also meant, and why.
- **C3.3** Assets are listed with sha256 checksums so drift and
  tampering are detectable.
- **C3.4** **[check]** (that the field is present and `sha256:`-prefixed)
  The manifest carries a self-committing `manifest_digest`: a
  sha256 over its own canonical JSON with the digest field blanked
  (adopted from duo, T7). The digest turns the manifest into a
  comparable identity, which makes cheap drift checks possible without
  re-walking.
- **C3.5** The word **manifest** is reserved for this self-description.
  The install-state file a tool writes into a harness directory is the
  **stamp** (C4.4) — never "manifest" (T9; ste9 collided here first).
- **C3.6** **[check]** (that the field carries the `toolsmith/`
  prefix, T24) The manifest declares the contract version it conforms
  to: `"contract": "toolsmith/v1"` (T8). `toolsmith check` and TOOLS.md
  read conformance from the tool, not from memory.
- **C3.7** Output schemas per verb are permitted (`OutputSchema`) but
  never filled speculatively.
- **C3.8** Each verb records its positional-argument usage verbatim from
  its own declaration (`usage`; absent when it takes none) — recorded,
  never parsed into structure (T23). A consumer that reads only the
  manifest, the harness projection included, must be able to see that a
  verb takes an argument at all.
- **C3.9** When a verb registers in both trees as an alias pair (C8.5),
  the manifest records the link on the porcelain member only:
  `alias-of: "<canonical qualified name>"` — `"plumbing next"` on the
  `next` entry. The field is absent on every other verb, the canonical
  member included, and on both members of a split pair: a different
  deliverable is not an alias. Each member registers as a full leaf —
  flags, usage, and schema emitted and checked independently — so the
  porcelain spelling is a real manifest row, never a hidden cobra
  alias. Projection keys on the plumbing tree (C4.3), so only the
  canonical member reaches a skill table. Adopted from wip (D112);
  ratified by T32.

- **C3.10** **[check]** The manifest records the contract minor its
  latest human reconciliation actually read as the positive integer
  `contractReconciledMinor`. The checker derives the current minor and
  clause additions from this document's history and reports missing,
  invalid, future, or stale readings; stale readings name every later
  version and its added clauses. This records process currency, not
  semantic conformance (T25).

## C4 — The install model

- **C4.1** Distribution is two-step: install the binary system-wide
  first (`nix profile install`, brew, a release binary), then ask the
  installed binary to project itself: `<tool> install <harness>`, with
  `uninstall <harness>` and `doctor` beside it.
- **C4.2** Harness targets live in a registry table
  (`internal/harness/registry`): one row per harness with
  `{Name, InstallDir, Available, Generate, Install, Uninstall}`. One
  list, many readers — install, uninstall, doctor, and help text all
  read the same table, so they can never disagree about what is
  installable. Registration panics on duplicates: registration is
  static program construction, not user input.
- **C4.3** Per-harness packages are policy-free: they render and write.
  Refusal policy (hand-edit protection, force semantics) lives in the
  verbs. Projected artifacts are **written, never symlinked**. Only
  `plumbing` verbs project into a harness — where the C8.3 boundary
  stands, the projectable set is exactly the namespace's
  `plumbing`-kind members: the audience axis and the determinism axis
  conjoined, so a pair reaches a skill table through its canonical
  member alone (T32). The generator applies the filter
  (`harness.Projectable`), the manifest itself never filters.
- **C4.4** Each installed tree carries one stamp file
  (`.<tool>-manifest-stamp.json`): `{toolVersion, schemaVersion,
  files: {path: sha256}}`. Everything generated is stamped; the only
  hand-written content in any projection is the per-harness judgment
  prose and shared guidance, which ride as assets (C5) and are stamped
  like everything else once rendered. Fixed template text in a generator
  (headings, labels, and format strings around manifest fields) is
  generated output, not hand-written content (T27).
- **C4.5** Drift is described with the six-state vocabulary (adopted
  from duo, T6): `current | missing | stale | modified |
  unowned_conflict | incompatible`. wip's three-way
  `added/removed/changed` per-file report is the floor, and maps into
  it. A seventh dimension exists where a harness only loads projections
  at session start: "an active session loaded an older projection;
  restart it."
- **C4.6** Three comparisons, kept distinct and each documented as
  which one it is: disk vs stamp (refuse hand-edited targets — the
  refusal names `--force`), binary vs stamp and disk vs stamp together
  (report `current`, write nothing), binary vs stamp alone (doctor's
  advisory staleness). A version bump whose generated content is
  byte-identical is still `current`.
- **C4.7** `uninstall` removes exactly the stamped tree and refuses on
  a missing stamp or drift. `doctor` reports exhaustively — every
  instance of a condition present at the time of the run, never the
  first found — as flat `{code, message}` findings with no severity
  levels; advisory codes never fail the run. Doctor may also collect
  garbage (orphaned generated trees) behind an injected keep-predicate
  (from duo).
- **C4.8** Ratified target extensions beyond plain file projection:
  - **Splice targets** (from ste9, T10): a tool may edit a foreign file
    in place only via a marker block (`<!-- BEGIN <tool> ... -->`) or,
    for JSON like `settings.json`, via surgical path edits
    (tidwall/gjson+sjson — never a `map[string]any` round-trip, which
    destroys key order and number fidelity), with a one-time `.bak` and
    byte-level golden tests.
  - **Emit-only targets** (from ste9, T11): a target may exist that
    never writes — it renders to stdout or `--dest` for a human to
    paste (e.g. a settings dialog). `Available()` is false; it installs
    nothing and stamps nothing.
  - **Per-target variants** (from ste9, T12): a projection may take
    per-harness variant selection (`--register claude=full,codex=none`
    shaped flags) when targets genuinely differ.
- **C4.9** A tool may materialize a **data dir**
  (`$XDG_DATA_HOME/<tool>/...`) when assets must exist as real files —
  greppable references, line-addressed specs. The data dir is a
  generated artifact: stamped, drift-checked, owned by install/doctor
  (from ste9, T13). It is the write-side fourth location, distinct from
  the read-side asset chain (C5.1).
- **C4.10** Lifecycle hooks are not verbs. Where a harness needs one
  (SessionStart and kin), the projection carries the smallest shim that
  invokes the binary and re-expresses no verbs.

## C5 — Assets

- **C5.1** Tunable behavior — prompts, templates, judgment prose,
  default config — ships as files, resolved through a three-link chain:
  user override (`$XDG_CONFIG_HOME/<tool>/`) → shipped default
  (`<prefix>/share/<tool>/assets`, resolved relative to the running
  executable, the layout buildGoModule's `$out` already guarantees) →
  embedded last-resort fallback (`//go:embed` of the whole assets
  tree).
- **C5.2** Shadow by name: an override fully replaces a shipped default
  at the same relative path, never merged. The one documented
  exception is config: `config.yaml` merges key-by-key over
  `config.default.yaml`, and the two filenames differ on purpose so
  they are never mistaken for the same file.
- **C5.3** Assets are inputs, never state. The asset package only ever
  reads. Generated artifacts are never parsed back.
- **C5.4** Large or encumbered assets (a copyrighted spec, a megabyte
  lexicon) get their own embed package so a build tag can drop them
  (from ste9). Encumbered content may force a private repo and private
  releases; the contract does not assume publishable assets.

## C6 — Release and hygiene

- **C6.1** Pushing an annotated `vX.Y.Z` tag is the only human action
  in a release. Everything else derives from it.
- **C6.2** **[check]** Version comes from
  `git describe --tags --match 'v[0-9]*'`; the `--match` is
  load-bearing (keeps milestone tags out of the stamp) and must agree
  with cliff.toml's `tag_pattern`.
- **C6.3** **[check]** (that CHANGELOG.md is not in the git index)
  CHANGELOG.md is generated per release by git-cliff and **never
  committed** — nothing in the repo can disagree with the tag it was
  built from. `make changelog` and `make release-notes TAG=` (which
  probes whether the ref exists to pick `--current` vs `--unreleased`)
  are the two targets.
- **C6.4** release.yml triggers on the tag push, re-gates on
  `nix develop --command make check` (a tag can sit on a commit that
  never went through CI), cross-compiles, generates the changelog and
  notes, writes SHA256SUMS, and publishes with assets **named
  explicitly, never globbed** (RELEASE_NOTES.md is the body, not an
  asset).
- **C6.5** ci.yml runs lint, test, `nix build`, and a cross-compile
  matrix, each through `nix develop --command`. CI never publishes.
  GitHub Actions are **SHA-pinned** (from duo main, T16) — **[check]**. Workflows are
  copied per tool, not shared by reference (T17).
- **C6.6** **[check]** (that `contrib/check-commit-msg` exists, that
  `.pre-commit-config.yaml` names `commit-msg`, and, when the Makefile
  has a `hooks:` target, that it passes `--hook-type commit-msg`)
  Conventional commits are enforced from commit one by a commit-msg
  hook (`contrib/check-commit-msg`), and `make hooks` installs **both**
  hook types (`--hook-type pre-commit --hook-type commit-msg`) — wip
  shipped the one-stage gap, ste9 fixed it (T15). The full test suite
  is deliberately not a per-commit hook.
- **C6.7** **[check]** (that the config says `version: "2"` and
  `default: none`) golangci-lint config is schema v2, `default: none`,
  enabling `govet staticcheck errcheck unused revive forbidigo`, with
  gofumpt as the formatter.
- **C6.8** `.gitignore` covers `/CLAUDE.local.md`, build output, and
  Nix litter. Any tool-generated per-clone directory (like `.wip/`)
  gets an explicit, decided ignore posture — not an accident of
  `.git/info/exclude`.

## C7 — Docs and evidence

- **C7.1** `docs/<matter>/decisions.md` records what building a Matter
  forced, with a fixed opening stanza naming the external design of
  record, a bolded `Status:` line, and a "Where this stands" closer.
- **C7.2** A port has a **port spec**, and it lives at
  `docs/<matter>/port-spec.md` — in the repo, committed. Code comments
  cite it by section. ste9's port spec stranded 1,409 lines in an
  ephemeral scratchpad while shipped Go comments cited it by §-number;
  this clause exists so that never happens again (T14).
- **C7.3** `evidence/` holds verification records: parity runs, dogfood
  passes, conformance reports — committed beside the code they verify.
- **C7.4** Comments carry the *why* and the rejected alternative, and
  cite decision IDs (contract clauses, register entries). A comment
  that restates the next line is noise; a comment that marks a
  constraint as load-bearing is the point.
- **C7.5** **[check]** (that README.md exists) The repo has a README.md
  that states what the tool is, the install two-step, and where the
  design of record lives. (wip shipped without one; the skeleton makes
  absence the anomaly.)

## C8 — Shapes

- **C8.1** Where a tool ships an LLM-shaped workflow, it ships it as a
  **shape**: one named workflow, defined **once**, delivered in two
  forms that stay in lockstep by construction. The definition is a
  **flow asset** — `assets/flows/<shape>.md`: numbered steps, plumbing
  calls, prompt references, decision points, and a marked set of
  form-specific presentation details outside the parity promise. The
  two forms:
  - the **verb form** — a top-level `llm`-kind verb: the tool
    implements the flow, citing its steps in comments (C7.4), and
    talks to the LLM endpoint itself;
  - the **skill form** — the projected skill, whose core instruction
    is *read the flow asset and follow it*; the harness agent is the
    LLM and drives the plumbing directly.
  The flow rides the asset chain (C5.1) and reaches the skill form
  live, through an asset-resolving verb at use time — never rendered
  into the projection at install, so an edit or a user override
  reaches both forms with no re-install. The flow asset is the one
  object that links a shape's two forms; without it the halves file
  under unrelated clauses (`llm` verb here, C4.4 judgment prose
  there) and drift between them is invisible (T29). Proven by the
  clast conversion (SURFACE.md V6, owner decision S9); ratified by
  T30.
- **C8.2** **The push-down rule.** Every deterministic step a shape
  needs must be reachable through `plumbing`-kind verbs. A shape's
  verb form may hold private *state* (a cache) but never private
  *logic* the skill form cannot reproduce (clast SURFACE.md V7, T30).
  Old clast's parity audit is the cautionary record: grouping rules,
  scan windows, and default windows lived in one delivery form each,
  and every one drifted or went missing in the other.
- **C8.3** **The `plumbing` namespace.** The audience axis — porcelain,
  the public user API a human types, vs plumbing, the substrate that
  skills and scripts call — is carried by the command tree, never by a
  kind: one binary, bare `--help` listing only the porcelain set, and
  the substrate invoked as `<tool> plumbing <verb>` and listed only by
  `<tool> plumbing --help` (bare `<tool> plumbing` prints that help,
  exit 0). The axes stay orthogonal in both directions: every
  `llm`-kind verb is porcelain, and porcelain also holds deterministic
  verbs (`init`, `doctor`, the install family). The manifest records
  the qualified name (`"plumbing sessions"`) with the verb's own usage
  (C3.8). The label is exactly `plumbing`: the C3.2 kind string and
  the git term of art; shorter labels were challenged and rejected on
  the merits (lineage doc §7). A tool that ships shapes (C8.1) draws
  this boundary; adopting it in an existing flat surface is that
  tool's own decision, in its own register. Proven by the clast
  conversion (SURFACE.md V1–V3, owner decision S8); ratified by T31.
- **C8.4** **Shape names reuse across the boundary.** Every shape has
  a same-named plumbing document verb: `<tool> <shape>` is the shape
  (C8.1's verb form), and `<tool> plumbing <shape>` is its
  deterministic core — the document its flow starts from, readable
  markdown with `--json` as the structure, and step 1 of every flow
  asset. Inside the namespace, shape names are reserved for these
  documents and nothing else. The reuse dissolves invented sibling
  names (draft-1 clast's `retro-data`): the collision only ever
  existed because a flat surface made the good name collide with
  itself (SURFACE.md V2/V8, T31).
- **C8.5** **Verb pairs.** Where the C8.3 boundary stands, a verb may
  register in both trees when both audiences need it. Three questions
  decide placement: does a human type it unprompted; does a skill or
  script call it for its own operation; and, when both are yes, is the
  deliverable the same for both audiences.
  - **Alias pair** (same deliverable): one implementation, one
    constructor registered twice — identical flags, output, schema,
    and kind. A golden test pins sameness: byte-identical stdout,
    stderr, and exit code under both spellings, with help text
    differing only in the command path itself and failure
    diagnostics canonicalizing to the plumbing spelling.
  - **Split pair** (different deliverable): the top-level member is
    the finished, human-sized form; the plumbing member is the
    deterministic substrate it is built on and keeps the full audit
    contract. The pair splits on the human face only — `--json`
    remains one contract, byte-identical across the pair.
  One concept per shared name. Both spellings execute — the boundary
  stays help-listing and projection, never execution; a pair is not
  a gate. A pair's plumbing member keeps it projectable, and
  top-level-only members stay unprojected on purpose. Proven by wip
  (D112): `wip next` and `wip plumbing next` are byte-equal (alias
  pair); `wip status` and `wip plumbing status` differ by design
  while `--json` is byte-identical (split pair). Ratified by T32.

---

## Clause history

The minor version this document carries. Only clauses added *after* T20
established the rule are dated here; everything else is the v1.0 body the
contract was extracted with, and reconstructing a finer history for it
would be invention.

| Version | Clauses added | Note |
|---|---|---|
| v1.0 | C1.1–C7.5 as extracted | The body proven by wip, duo and ste9. |
| v1.1 | C3.8 (T23) | Each verb records its positional-argument usage. The first additive clause after T20; it is what exposed that the document had nowhere to record a minor (T24). |
| v1.2 | C2.5's `<verb>.findings-present` form (T26) | A verb whose findings are its verdict has a sanctioned error code, and its findings stay on stdout. wip, duo, ste9 and every tool `new` produced already emit `doctor.findings-present`, so they conform with no change. |
| v1.3 | C8.1–C8.4 (T30, T31) | Shapes and the audience surface: an LLM workflow is defined once, in a live-served flow asset, with a verb form and a skill form; every deterministic step it needs is reachable through plumbing; the `plumbing` namespace is the audience boundary, with shape names reused across it. Proven by the clast conversion (S8–S10). No existing tool ships a shape, and the namespace binds only tools that do, so all conform with no change. |
| v1.4 | C8.5, C3.9 (T32) | Verb pairs and the `alias-of` manifest field: where the `plumbing` namespace stands, a verb may register in both trees — an alias pair for the same deliverable, a split pair for different ones — and the manifest links an alias pair's porcelain member to its canonical qualified name. C4.3's projectable set is clarified, not changed: the namespace's `plumbing`-kind members. Proven by wip (D112); the clause is permissive, so every tool conforms with no change — wip as shipped, clast's own pairs decided-not-built (SURFACE.md V24 amendment), duo and ste9 shipping no namespace. |
| v1.5 | C3.10 (T25) | The manifest records the minor its latest human reconciliation read, and the checker reports missing, invalid, future, or stale readings with every intervening version and its added clauses. This is process currency, not semantic conformance. |

A tool's declared string does not move with this table (T24) — see
**Versioning** above.

## Conformance

The conformance checker audits the **[check]**-marked clauses
mechanically and reports findings by clause ID. That coverage is
**partial and deliberately so**: most clauses here are prose, and a
checker that pretended otherwise would be worse than one that admits its
boundary. So the checker reports what it audited alongside what it
found — a clean run means "clean against the clauses I can reach", never
"conforms to the contract".

A clause may be *partly* checkable: C1.2's `cmd/<tool>` existence, C3.4's
digest field. The marker says exactly which part it covers. A mark that
claims more than the checker reads is the document overclaiming, even
when the clause ID is correctly marked. The marker sits at the sub-part
it governs, and the mark set is expected to grow as clauses that read as
prose turn out to have mechanical sub-parts. The marks and the checker's
own emitted clause set must agree — if they drift, the document is
lying about its coverage.

The rest is audited by reading: a conversion's final workplan step is a
reconciliation pass against this document, recorded with the minor
version it was read against, so a later run can tell which clauses
landed since.

Tools declare their contract version in the manifest (C3.6) and are
listed with it in TOOLS.md.
