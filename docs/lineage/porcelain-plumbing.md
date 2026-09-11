# Porcelain and plumbing: how the concept narrowed, and what it costs

**Status:** lineage record, 2026-09-11. Research produced during the
clast conversion's Pass 2 (design of record:
`~/Code/clast-reboot/SURFACE.md`); recorded here because the finding
is about this contract, not about clast. This document is evidence
for a future T-decision that gives C3.2 the decision record it never
had; it changes nothing by itself.

---

## 1. The original concept

clast (the bash tool this all descends from) had a two-layer design
whose vocabulary the ecosystem later reused with a different
meaning. In the original:

- **Plumbing** — the deterministic substrate: stable JSON, stable
  exit codes, callable by anything.
- **Porcelain** — the *public user API*: a named workflow **shape**
  ("wake", "brief", "retro") that ships in **two delivery forms of
  the same flow**:
  1. a CLI verb — the script coordinates plumbing and talks to an
     LLM endpoint itself (`clast wake` from a plain terminal or
     cron);
  2. a harness skill — the agent *is* the LLM and executes the same
     flow by calling the same plumbing (`/wake` inside Claude Code).

The invariant: the two forms mimic each other. Old clast states it
verbatim — `clast/docs/guides/run-without-claude-code.md:3-12`
("**`clast wake`** — interactive day curation. The standalone
equivalent of `/wake`. … reuse the same prompt templates as the
plugin skills, so the output stays in sync") and
`clast/skills/wake/SKILL.md:166` ("This mirrors `clast wake --auto`
in the CLI porcelain, and behaves the same way").

Two axes are in play, and they are orthogonal **in both
directions**:

- **Audience** — porcelain (what a human types; what they should see
  in bare `--help`) vs plumbing (what skills and scripts call).
- **Determinism** — deterministic output contract vs LLM-shaped.

Every LLM verb is porcelain, but not every porcelain is LLM: `init`,
`install`, a jot-a-note verb, or a pretty human-sized `status` are
deterministic *and* public API. wip demonstrates the pain of
conflating the axes from the other side: its intended human surface
was `status` and `next`, its actual surface is ~39 top-level verbs,
and its `status` output is agent-grade detail rather than a
human-sized view.

## 2. How the concept narrowed — three steps

The narrowing happened **before wip's implementation**, at the first
codification, and each later step hardened it.

### Step 1 — the framing inversion (`wip-reboot/procrastivity-cli-architecture.md`)

The ancestral architecture doc adopts "porcelain" in the *git* sense
— a hand-authored surface layered over a core — and frames the whole
problem as drift between hand-written porcelains to be *eliminated*
by generation (`:5`, `:158-163`: "free of the drift that comes from
maintaining multiple hand-written porcelains over one core").

Its kind vocabulary (`:134-136`) is two-valued: `plumbing`
(deterministic, JSON/exit-code, no LLM) vs `porcelain-llm` ("the
CLI-only judgment verbs like wip's `ask`/`intake`"). At `:145-148`
it derives the projection rule: only `plumbing` projects into a
harness, "because inside a harness, the model already *is* the LLM."

So at the moment of first codification, "porcelain" already denoted
**a verb that calls an LLM**, not **a workflow shape with two
delivery forms**. The two forms were recast as a drift problem to
delete, when in clast they were both legitimately wanted.

### Step 2 — the word is dropped from the kind (wip D47/D53)

`wip-reboot/MODEL.md:514` (D47): "Porcelain capability classes are
**surfaces** (1 deterministic · 2 LLM-shaping · 3 control-plane) …
CLI = surfaces 1–2; agent porcelain = 1–3." `porcelain-llm` loses
the word and becomes `llm`; "porcelain" is reassigned to the
*transport* axis, where "agent porcelain" means the live MCP
surface. D53 (`MODEL.md:520`) makes the three surfaces the manifest
verb kinds, with only `plumbing` projecting. Every later refinement
(D84/D87/D88/D89; `wip/evidence/dogfood/surface-3-decisions.md`)
argues inside the transport meaning. The user-API meaning is no
longer in play.

The phrase that survives to today is minted in
`wip/internal/surface/surface.go:1-5`: the kinds are "the
**mechanical enforcement point for the porcelain/plumbing split**,
never inferred from folder name or convention."

### Step 3 — the contract inherits the mechanism without the rationale (C3.2)

duo copies wip's package comment verbatim
(`duo/internal/surface/surface.go:1-5`); toolsmith copies it again,
swapping `D53` for `C3.2` (`toolsmith/internal/surface/surface.go:1-5`,
and the skeleton shipped to every future tool,
`assets/_skeleton/internal/surface/surface.go:1-5`).

C3.2 arrived in CONTRACT.md as part of the extracted v1.0 body,
**with no T-number** — DECISIONS.md has no entry for verb kinds,
despite its own rule that contract changes cite the T-number that
authorized them. Outside `git status --porcelain` in tests, the word
"porcelain" appears in the toolsmith repo exactly twice:
`CONTRACT.md:123` and the surface.go comment. A reader meets "the
porcelain/plumbing split" as an undefined term of art whose only
visible referent is the kind list beside it. **C3.2 became the
definition of a concept it was originally only the enforcement point
of.**

## 3. The vocabulary today is vestigial

Kind usage across the four Go tools (surveyed 2026-09-11):

| Tool | `plumbing` | `llm` | `control-plane` | Notes |
|---|---|---|---|---|
| wip | 60 | 0 | 0 | `ask`/`intake` from the architecture doc were never built |
| duo | 22 | 0 | 0 | parallel `human_llm_porcelain` projectability class: **zero rows** (`duo/internal/registry/table.go`) |
| ste9 | — | — | — | never adopted kinds at all (no `internal/surface`) |
| toolsmith | 8 | 0 | 0 | |

Ninety annotated verbs; every one `plumbing`. The `llm` and
`control-plane` kinds are exercised only by test fixtures — fake
verbs named `ask` and `watch` in `wip/internal/harness/*_test.go`
and `manifest_test.go`. The only tool that has ever had a real
LLM-shaped user API is clast, the tool the concept came from — and
its Go conversion had to re-derive the two-forms idea from scratch
as a tool-local decision because the contract does not carry it.

## 4. The structural gap

For a workflow with two delivery forms, the contract files the
halves under unrelated clauses:

- the CLI form is a verb of kind `llm` — which C4.3 **excludes from
  projection**;
- the skill form is hand-written judgment prose under C4.4 —
  stamped, but opaque to the manifest.

No object in the manifest, the contract, or `harness.Projectable`
records that `wake`-the-verb and `wake`-the-skill are the same
workflow. The skill cannot cite the verb; the manifest cannot link
them; drift between them is invisible to exactly the machinery the
architecture doc built to stop drift.

## 5. Evidence that the gap bites: the old clast parity audit

Audited 2026-09-11 against `procrastivity/clast` v0.0.8. Every gap
below shipped and persisted.

- **wake** — parity *partial*. Porcelain-only: slug derivation via
  raw-transcript `ai-title` (`wake.bash:132-157` — reads the
  transcript file directly; **no plumbing command exposes it**, so
  the skill ships no slug rule at all); 2000-char/turn truncation
  (`:441-452`); scan window `-14d`/`CLAST_WAKE_SINCE` vs the
  skill's hardcoded `-30d`; recorded-time header with its
  timezone-correctness argument (`:391-418`); LLM retry menu; model
  timing. Skill-only: the promotion flow
  (decision/common-issue/workflow) and per-project ordering — the
  porcelain has neither.
- **brief** — parity *broken*. The skill never reads
  `brief-system.md`/`brief-user.md`; it inlines a **stale prompt
  copy** predating the workspace-grouping work. The porcelain's
  deterministic gathering (`brief.bash:36-124`: label//branch group
  keys, current-workspace hoisting, 3-per-group/8-total caps, and
  the comment at `:38-43` explaining why a flat `--limit` is wrong)
  is absent from the skill — whose flat `--limit 5` is precisely
  the failure mode that comment warns about.
- **retro** — **no skill exists.** The porcelain adds over plumbing:
  a `-7d` default window (plumbing defaults to the whole corpus — a
  naive skill would fan one model call per session across all
  history), a fingerprinted summary cache with `--refresh`
  (`retro.bash:83-111`), the fold-back transform, and a
  summary-aware renderer plumbing cannot produce.
- Of six prompt templates, only wake's two are shared between forms;
  four are porcelain-only. The two forms even load prompts by
  different mechanisms (search path vs `$CLAUDE_PLUGIN_ROOT`
  hardcode).

The common cause across every item: **flow logic lived in only one
delivery form**, and nothing could see that it was missing from the
other.

## 6. Implications (proposed, not ratified)

Being proven first in the clast conversion
(`~/Code/clast-reboot/SURFACE.md`), then candidates for contract
ratification:

1. **The shape concept.** A named workflow with one canonical flow
   definition and two delivery forms (an `llm` verb and a projected
   skill). The flow lives as an asset (`assets/flows/<shape>.md`),
   served live through the tool's asset-resolution verb, so the
   skill reads it at use time and the Go verb implements it citing
   steps — one source, no re-install to update, drift becomes
   visible.
2. **The audience axis returns as surface structure, not as a
   kind.** One binary; bare `--help` shows only porcelain; the
   substrate lives under a `plumbing` namespace (`clast plumbing …`).
   This also dissolves the shape-name collision: `clast retro` is
   the shape, `clast plumbing retro` is its deterministic document —
   no more invented names like `retro-data`.
3. **The push-down rule.** Every deterministic step a shape needs
   must be reachable through plumbing; a shape's verb form may hold
   private state (a cache) but never private *logic* the skill form
   cannot reproduce.
4. **wip back-port sketch.** Bare `wip` becomes `status`, `next`,
   `init`, `install`, `plumbing`; the other ~35 verbs move under the
   namespace; the human `status` can become a small pretty porcelain
   over the agent-grade plumbing one.

## Where this stands

C3.2 keeps working as the determinism/projectability axis — nothing
here weakens it. What is missing is the audience axis and the object
linking a shape's two forms. The clast conversion is the proving
ground; when its surface lands, the follow-up Matter here should:
give C3.2 its retroactive decision record, ratify a shape clause and
the namespace pattern (minor-version bump per T20/T24), and decide
whether wip adopts the namespace. Until then, this document is the
record of why.
