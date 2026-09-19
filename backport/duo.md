# Back-port punch list — `duo`

Fixes flowing back into `duo` (branch `go`, module
`github.com/procrastivity/duo`) now that the conventions it produced are
written down. duo is the contract's largest single donor — the
self-committing `manifest_digest` (C3.4, T7), the six-state drift
vocabulary (C4.5, T6), SHA-pinned actions (C6.5, T16) and doctor's
injected keep-predicate (C4.7) all cite it — so several gaps below are
duo failing to hold a clause duo wrote.

Audited 2026-09-19 at `1437f2f`
(`1437f2f69e71d164b4248ed121803e53e3e1a2ba`). The mechanical half comes from
`toolsmith check` run against an export of that commit — 11 findings,
exit 1, byte-identical to the committed golden
`internal/verbs/check/testdata/golden/check/corpus-duo.stdout`. duo has
no committed conformance report yet; item 14 is that gap. Two clauses
the checker could not reach were closed by hand: the manifest probe
short-circuits on the C3.1 failure, so C3.4 and C3.6 were never
evaluated (C3.4 conforms, C3.6 is item 2), and C6.3 is silently skipped
on a non-git export (conforms by hand — CHANGELOG.md is not tracked).

This file is where toolsmith coordinates the duo convergence; duo files
its own matching entries in its own tracker. Where an item needs a
ruling — duo's owner, or an amendment to CONTRACT.md — it says so
instead of pretending the fix is settled.

Item 1 is the one with a clock on it — `release.yml` runs with a release
token. The rest can be picked up alone, in any order; each item names
its clause.

---

## Mechanical — the checker's 11 findings

**1. GitHub Actions are not SHA-pinned (C6.5).**
Ten of the eleven findings: `ci.yml:25,26,32,33,39,40,53,54` (four jobs,
each `actions/checkout@v4` plus
`DeterminateSystems/nix-installer-action@v16`) and `release.yml:21,26`.
A tag is a mutable pointer into someone else's repository, and
`release.yml` runs with a release token. Fix: pin the SHA, keep the
version as a trailing comment so Dependabot can still update. Otherwise
the workflow structure conforms — four jobs, each through the flake
(`nix develop --command`, with `nix build` as the flake entry point),
and CI never publishes.

`backport/wip.md` item 3 says "duo's main branch already does this
(T16); copy its lines." That means duo's TypeScript `main` branch, not
this Go line — the Go line being fixed here. The source to copy is
still main's, same repository, so the instruction in wip.md stands; it
just doesn't mean "copy the branch next door."

**2. The manifest declares no contract version (C3.6).**
Hand-closed gap — the checker never reached this clause.
`manifest.go:106-122` has no `contract` field; TOOLS.md correctly lists
duo as pre-v1, so nothing is currently lying. Fix: add `"contract":
"toolsmith/v1"` to the struct and to
`contracts/schemas/duo-manifest-v1.schema.json` — additive, since the
schema root is `additionalProperties: true`, breaking no consumer. Same
as `backport/wip.md` item 5: conformance should be readable from the
tool, not asserted from memory.

**3. `toolsmith check` cannot audit duo's manifest (C3.1, via C2.3) — and the fix is toolsmith's, not duo's.**
The eleventh finding: `C3.1: duo manifest --json failed or is not
implemented`. A probe failure, not a manifest defect — `duo manifest
--output json` exits 0 with `schemaVersion: 1`. duo retired the global
`--json` on purpose (`internal/cliflags/cliflags.go:13-23`, bound once
at `root.go:53-54`; unknown flag on `doctor`, exit 2, pinned by
`internal/cli/output_test.go:19-46`; decided at
`docs/cli/decisions.md:14-73`, "Resolved 2026-08-24"). The divergence is
in Not a defect and TOOLS.md; the defect is the consequence — the
failed probe short-circuits the manifest walk, so the checker never
reaches C3.4, C3.6 or anything behind it, three clauses dark over one
flag spelling.

Ruling needed: (a) duo adds `--json` — contradicts a ratified,
test-pinned decision, rejected unless reopened; (b) the checker learns a
fallback probe (`--output json`); (c) the contract admits a declared
alternate spelling (C2.3 amendment). The audit expects (b) or (c); the
ruling is toolsmith's, and duo does not move until it lands.

---

## Non-mechanical

**4. Verb packages are flat in `internal/cli/` (C1.4).**
No `internal/verbs/`. All 22 leaf verbs sit directly in
`internal/cli/*.go` (25 non-test files); the clause wants one package
per verb under `internal/verbs/<verb>/`, each exposing `Command(streams
*iostreams.Streams, ...) *cobra.Command`. Domain packages
(`internal/domain`, `internal/launch`, `internal/store`) already sit
beside, never inside. `internal/cli/root.go:23-76` is a clean sole
registration point with no `init()` (C1.3 conforms), so this is a
relocation, not a rewiring — the largest mechanical edit here, and the
least interesting.

**5. No E2E suite through the built binary (C2.7).**
duo tests the root command in-process
(`internal/cli/session_test.go:21-29`). `internal/selftest` exists, its
own comment saying it exercises the error-envelope machinery "end-to-end
through the built binary" — but nothing sets `DUO_SELFTEST`,
unexercised scaffolding, a seam built and never connected. Fix:
`TestMain` runs `go build`, helpers return `{stdout, stderr, exitCode}`,
hermetic env forces every `DUO_*_DIR` seam to a temp dir.

**6. `OutputSchema` is declared and never set (C3.7).**
`manifest/verbs.go:21-29` — `SetOutputSchema` has zero callers, field
always omitted. Same shape as `backport/wip.md` item 10; decide both
together. Both keep and drop conform — the undecided third state is the
defect, reading as an unfinished feature. duo's `verbs.go:21-23` comment
— "No verb calls this yet … not filled speculatively" — already reads
as the keep answer; the remaining work is confirming it, the same way
wip's item lands.

Ruling needed: duo's owner — keep (one line saying it's a reserved
slot) or drop (delete setter and field).

**7. The manifest records no positional-argument usage (C3.8).**
No `usage` field anywhere; the manifest's `Arg` describes flags only,
and the code comment concedes the gap — positionals are "not
introspectable generically… not described here". Concrete cost: a
consumer reading only the manifest cannot see that `session launch`
takes a preset, or `prompt send` a session id. C3.8 wants it recorded
verbatim from the verb's own declaration, never parsed into structure
(T23) — teeth for the harness projection, which reads the manifest and
nothing else.

**8. The install surface is unbuilt (C4.1, C4.2, C4.3, C4.6).**
One item, deliberately. `root.go` registers `version, manifest, doctor,
session, conversation, prompt, config, workspace, provider` — no
`install`, no `uninstall`, no projection verbs. No install verb means no
harness registry (`internal/harness/registry` doesn't exist), no
per-harness packages (`harness_targets` hard-coded `[]` at
`manifest.go:157`), and no chassis home for C4.6's three comparisons,
which exist only inside the Devin adapter. One absence, one cause;
splitting into four items would produce work nobody could pick up
independently. The spec exists —
`docs/vnext/duo-vnext-installation-contract.md`, TOOLS.md says "spec'd
but not yet built" — a pointer to planned work, not a discovery.
Largest item here, least likely to be picked up by a passing session.

**9. Two stamp implementations, and the contract-shaped one is dead (C4.4).**
`internal/manifest/stamp.go` defines `.duo-manifest-stamp.json` exactly
per C4.4, but `WriteStamp`, `ReadStamp`, `ChecksumFiles`, `Drift` have
no callers outside their package. The live Devin projection stamps
`.duo-generated.json` instead (schema `duo.projection-stamp/v1`,
`internal/runtime/devin/hooks.go:32,91-110`) with a richer shape —
`manifest_digest`, `installation_id`, `source_assets`,
`target.tested_version_range` — backed by
`contracts/schemas/duo-projection-stamp-v1.schema.json`; C3.5 holds
either way, both files are called stamps, never "manifest".

Ruling needed: duo's owner picks the real stamp. Projection stamp wins
(the audit's expectation, since it's the one that runs) → delete
`stamp.go` and `drift.go`, and consider amending C4.4's field list to
admit the richer shape. Contract shape wins → the Devin adapter moves
onto it. Not acceptable: both — a dead API matching the contract is how
a tool looks conformant while behaving otherwise.

**10. The six drift states live only in the Devin adapter (C4.5).**
`internal/runtime/devin/hooks.go:63-79` derives all six — `current |
missing | stale | modified | unowned_conflict | incompatible`. The
chassis `drift.go:11-14` still speaks wip's three-way
`added/removed/changed` floor: duo contributed the six states to the
contract (T6), and duo's own chassis doesn't speak them. Same item as
`backport/wip.md` #7 from the opposite direction — wip never had the six
states, duo has them and left them in an adapter. Fix: hoist to the
chassis; the per-file report survives as the detail under `modified`.

**11. `doctor` has no verdict (C4.7).**
Scoped to doctor — uninstall belongs to item 8. duo's doctor emits a
nested typed report and always exits 0; verified live —
`devin_projection.status` can read `unowned_conflict` or `incompatible`
and the run still exits 0, no `doctor.findings-present` code (the C2.5
form v1.2 sanctions for exactly this). The GC half conforms and is
duo's own donation — the injected `KeepHarnessDir` predicate is the
clause's "(from duo)" source; don't touch it. Fix sketch: keep the typed
report as stdout payload (C2.5 keeps findings on stdout), derive flat
`{code, message}` findings and the verdict from it, exit 1 via
`doctor.findings-present` when non-advisory findings are present —
reported exhaustively, never first-found.

**12. Four tunable payloads bypass the asset chain (C5.1).**
The chain itself is textbook —
`internal/asset/asset.go:96-134`, override → share → embed — which
makes the exceptions conspicuous. Four payloads are
embedded in runtime packages instead:
`internal/runtime/claude/closeonexit.go:18`,
`internal/runtime/pi/closeonexit.go:19`,
`internal/runtime/pi/extension.go:17`, and
`internal/runtime/pi/inject.go:22`. The session-end hook's
`TERMINAL_REASONS` comment invites editing — tunable by its own
admission, C5.1's exact trigger.

Ruling needed: these payloads ride the projection stamp's
`source_assets` digest list, so moving them under `assets/` must keep
the digest story intact. A user override that changes the stamp may be
correct (drift becomes visible, which is what stamps are for) but it's
a behavior change, not a refactor. Cross-reference item 9 — the answer
depends on which stamp wins.

**13. `decisions.md` files carry the stanza and neither bookend (C7.1).**
Ten `docs/<topic>/decisions.md` files carry the fixed opening stanza
naming the external design of record; zero carry the bolded `Status:`
line, zero carry the "Where this stands" closer. Consistent across all
ten, so this is a template gap, not rot — one pass fixes it. The
directory-layout half of C7.1 (`docs/<topic>/` vs `docs/<matter>/`) is
NOT in this item — stated policy at `docs/README.md:10-14`, sitting in
Not a defect pending a ruling. Keep those apart: the stanza fix is
settled work, not blocked behind a layout debate.

**14. No conformance report in `evidence/` (C7.3).**
Conforms in substance — `evidence/` holds rich committed verification
records — but no conformance report exists. Cheapest item on the list:
the run behind this punch list is the report. Cite the golden
`internal/verbs/check/testdata/golden/check/corpus-duo.stdout` and this
file as the inputs.

**15. No comment cites a contract clause (C7.4).**
duo's comments are the best in the fleet on the why and the rejected
alternative — the citation half is missing: zero clause citations
anywhere outside `docs/vnext/`, no C-IDs, no `CONTRACT.md`, no
"toolsmith". Fix is small since comments already paraphrase the
clauses; add the ID where the paraphrase already is: `flake.nix:14-22`
is C1.7, `exitcode.go:1-3` is C2.4, `internal/asset/asset.go:1-6` is
C5.1/C5.3, `Makefile:28-33` is C1.1.

**16. README omits the install two-step (C7.5).**
Conforms mechanically and on two of three points — README.md exists,
states what duo is and where the design of record lives. Can't state
the install two-step because there's no install surface (item 8).
Blocked on item 8; interim worth doing now: one line saying the
projection surface is spec'd and unbuilt, so a newcomer isn't left
inferring it from silence.

**17. The delegation-loop skill is hand-authored (C8.1).**
C8.1 is conditional and duo triggers it: duo ships an LLM-shaped
workflow as `skills/duo-delegation-loop/SKILL.md`, 155 hand-authored,
hand-committed lines with a four-sentence description frontmatter.
`README.md:16-17` calls it "the interim delegation-loop skill" — no
`assets/flows/<shape>.md`, no `llm`-kind verb, no projection,
contradicting the contract's own principle: "the harness layer is not a
thing you author. It is a thing the binary emits."

Two nuances: (1) C8.2 rides along, satisfied by luck — the flow is
fully push-down-satisfied, every step a real plumbing verb, no private
logic, unenforceable without C8.1's flow asset, so compliance is real
but unpinned. (2) CONTRACT.md's v1.3 clause-history row claims "No
existing tool ships a shape... so all conform with no change" — it
didn't account for this skill and is wrong as written; correcting it is
toolsmith's work, whether or not duo restages the skill.

Fix: (a) move the prose to `assets/flows/delegation-loop.md` riding the
C5.1 chain; (b) project the skill from the manifest, blocked on item 8.
(a) alone is worth doing now.

---

## Not a defect

Recorded so nobody "fixes" them:

**`make hooks` installs both hook types (C6.6) — resolved, no work
here.** A prior finding recorded the bare `pre-commit install` form on
`go`. It is stale: commit `949aa0b` (2026-09-05) added `--hook-type
pre-commit --hook-type commit-msg`, two weeks before the 2026-09-18
confirmation that repeated the finding. No clone carries the bare form.
The confirmation most plausibly carried over from `backport/wip.md` item
2, which is the same defect in wip and is still open there. BDS-214
names C6.6 and duo carries a matching entry in its own tracker, so it is
recorded here rather than dropped — closed, with the fixing commit, so
neither tracker has to re-derive it.

**`--output text|json` instead of `--json` (C2.3).** Deliberate,
decided, test-pinned and fixture-fixed: `docs/cli/decisions.md:14-73`
retired the global `--json` on 2026-08-24 (dogfood Step 24); the
spelling is held by a committed, embedded, digest-checked fixture
(`contracts/fixtures/duo-external-v1/projection-cases.json`), bound to
the registry by `TestProjectionCasesMatchRegistry`. TOOLS.md already
records it as a divergence — nobody should "restore" `--json` to make
the checker happy, the checker is what moves. Cross-reference item 3,
which carries the ruling.

**The error-code vocabulary (C2.5).** The shape conforms:
`duoerr.Error{Code, Message}`, one `Render`,
`SilenceUsage`/`SilenceErrors` at root, `exitcode.FromError` mapping
everything outside `refusal.*`/`internal.*` to exit 1 — and every code
duo emits is dotted (132 `duoerr.New` sites, zero undotted).
`invalid.request`, `object.not_found`, `session.target_exited` and kin
are the `duo.external/v1` public wire vocabulary, fixed by
`contracts/schemas/duo-external-v1.schema.json` and its fixtures;
renaming them to the contract's hyphenated examples is a wire break,
and whether snake_case after the dot needs a ruling is duo's owner's
call. The undotted strings a grep does find — the twelve `Code*`
constants in `wire.go:22-56` and two test literals at
`fakeserver_test.go:308,384`, all under
`internal/host/herdr` — are the external Herdr host's own codes passed
through, not `duoerr.Error.Code`; C2.5 does not reach them, and
dotting them would break fidelity with the real server. Recorded so
nobody does a find-and-replace.

**`docs/<topic>/` rather than `docs/<matter>/` (C7.1).**
`docs/README.md:10-14` states per-topic directories as policy, on
purpose; C7.1 says per-Matter — a real disagreement between duo's
stated policy and the contract, wanting a ruling: either duo adopts
per-Matter or the contract admits per-topic. Not wanted: a silent
rename by a session that noticed the mismatch. Item 13 fixes the stanza
bookends, settled either way.

**No port spec (C7.2).** Record why, not just n/a: C7.2 binds a port —
work reproducing a behavioral oracle, where shipped comments cite a spec
by section. duo's Go line is not that: `main` is a separate, live
TypeScript product, not an oracle the Go line reproduces. duo's shipped
comments do cite a spec by section —
`duo-vnext-installation-contract.md` §1.1–§1.3, at
`internal/cli/config.go:6`, `internal/cli/doctor.go:86`,
`internal/config/migrate.go:19` and kin — but that spec is duo's own
design of record, committed at `docs/vnext/`, so the stranded-spec
failure C7.2 exists to prevent cannot occur here. The clause exists
because ste9 stranded 1,409 lines of spec in a scratchpad while shipped
comments cited it by §-number; duo has no such situation. Conditional:
if duo starts citing main's behavior as an oracle in Go comments, C7.2
activates and this entry expires.

- Nix stamps the commit, make stamps the tag. Documented at
  `flake.nix:14-22` and deliberately not reconciled (C1.7). A VERSION
  file would be a second copy of a fact.
- CHANGELOG.md absent from the index. Correct (C6.3). The checker skips
  this on a non-git export; verified by hand.
- `.wip/` ignored by a committed, commented `.gitignore` rule
  (`:15-16`). Correct (C6.8) — and it is the posture `backport/wip.md`
  item 9 asks wip to adopt. duo got this right; do not touch it.
- `--version` alongside the `version` verb (`root.go:27-31`). A
  deliberate departure, noted in place.
- `Details` on `duoerr.Error`. Documented, JSON-only, additive to
  C2.5's `{Code, Message}` shape.
- `internal/registry` is not C4.2's harness registry. It is the
  operation registry, a different object with a colliding name. Nobody
  should point C4.2 at it and call item 8 done.
- `nix build` not wrapped in `nix develop --command` (`ci.yml:41`).
  Correct — it is the flake entry point, not a command run inside the
  dev shell.
- `CGO_ENABLED=0` appears once in `ci.yml`. Conforms: C1.1's **[check]**
  scopes to the setting "anywhere in the file, not at every build it
  governs."
