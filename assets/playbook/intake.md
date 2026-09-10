# Intake — classify the thing you were pointed at

You were pointed at "something agent-skill shaped": a repo, a directory,
a marketplace entry. Before any code moves, classify it, map its parts to
contract slots, and write down what you cannot decide alone. The output
of intake is a filled **intake sheet** (bottom of this page) — it seeds
the packaging appendix and the handoff.

Rule of engagement: when classification is ambiguous, **ask the owner**
— the questions list per shape below is what to ask. Do not guess a
tool's purpose from its prompts.

## Step 1 — inventory what exists

Walk the source and list every part into one of these bins:

| Bin | What it looks like | Where it lands in the contract |
|---|---|---|
| Verb-shaped scripts | bash/python invoked with subcommands or flags; produces output an agent parses | Go verbs (`internal/verbs/`), kind `plumbing` (C3.2) |
| Judgment prose | "when to use this", workflow guidance inside SKILL.md | The per-harness judgment asset — the only hand-written projection content (C4.4) |
| Reference assets | dictionaries, templates, partials, examples the skill greps or reads | Shipped assets (C5); a materialized data dir if they must be greppable files at runtime (C4.9) |
| Prompt/LLM steps | instructions that require model judgment to execute | Kind `llm` verbs, or stay prose — they never project (C4.3) |
| Lifecycle hooks | SessionStart/Stop hooks in plugin.json or settings | The thinnest shim that invokes the binary; never re-expresses verbs (C4.10) |
| Foreign-file edits | install steps that append to CLAUDE.md, edit settings.json, set an output style | Splice targets (C4.8) — marker blocks, gjson/sjson, byte-golden tests |
| Config | per-user tunables | Tool config (C5.2); per-project facts go to the tool's own store, never XDG |
| Install machinery | install.sh, marketplace manifest, plugin.json | Replaced wholesale by `<tool> install <harness>` (C4.1); the plugin/marketplace layer is not ported, it is regenerated |
| MCP server | a stdio server with tools | Kind `control-plane`; keep the server, but the CLI verb surface stays primary (duo is the worked example) |

## Step 2 — recognize the shape

- **Bare skill** (SKILL.md, maybe references/): smallest case. The port
  is mostly asset extraction plus a small verb surface. Ask: is there
  any deterministic behavior at all, or is this judgment prose plus
  reference files? Either way it still gets a binary (T3) — the binary
  owns manifest/install/doctor even when it owns no other verb yet.
- **Skill + scripts** (ste9's shape): the scripts are a hidden verb
  surface with an existing output contract. Decide per script: port
  (users/agents call it) or keep as dev-only tooling (ste9 kept
  extract.py). Ported scripts with consumers get a **parity gate**
  (assets/playbook/parity-gate.md) and a **port spec** (assets/playbook/port-spec.md).
- **Plugin with hooks/commands**: hooks map to C4.10 shims; slash
  commands usually collapse into the generated skill's verb table.
- **Marketplace of several skills**: decide the tool boundary first —
  one binary per coherent verb surface, not one per skill. Several
  skills can ride one binary (ste9 ships three skills + an output style
  from one binary).
- **Output style / register**: a splice-and-settings install (C4.8),
  with per-target variants if harnesses differ.
- **TS/MCP tool**: the contract's conventions apply even where Go
  doesn't (duo main). Decide language honestly: an MCP-centric tool may
  stay TS; everything else converts (T3).

## Step 3 — decide what the old implementation is

Two distinct postures, both proven:

- **Oracle** (ste9): the old code has byte-level consumers or subtle
  semantics. Keep it running as the parity oracle until the gate passes;
  write the port spec from it; delete it only after cutover.
- **Evidence only** (wip): the old design is not consulted at all; only
  "feel signals" (verb names you liked, lint posture) carry over. The
  rewrite starts from a fresh model on an orphan branch.

Pick one per part, explicitly. A repo can mix them (ste9: linter =
oracle; installer = evidence).

## Step 4 — questions for the owner

Ask whatever the walk left ambiguous. The recurring ones:

1. Who consumes each script's output today, and does any consumer
   depend on exact bytes? (decides parity gates and `exitcode.Silent`)
2. Which parts are dev-only vs shipped? (decides what gets ported at all)
3. Do any assets have licensing constraints? (decides embed split,
   private releases — C5.4)
4. Which harnesses matter for v1? (default: claude-code only; the rest
   are backlog — wip's H9 posture)
5. Does anything need to differ per harness? (decides variant flags)
6. Is there state (not config, not assets)? Where does it live, and
   does it need a migration?
7. What is the cutover judgment? (default: "the new one is the one I
   reach for" — never a parity checklist)

## The intake sheet

Record the answers as the first file of the conversion sidecar:

```
# Intake: <source> -> <tool>
Shape: <bare skill | skill+scripts | plugin | marketplace | output style | MCP>
Verb surface (planned): <verbs, each with kind>
Oracle parts: <part -> oracle | evidence-only>
Assets: <list; flag encumbered/large ones; greppable-at-runtime ones>
Splice targets: <foreign files the install must edit, if any>
Hooks: <lifecycle events needed, if any>
Harnesses v1: <default claude-code>
Open questions: <what the owner has not yet answered>
```

Then continue with assets/playbook/migrate.md (existing tool) or
assets/playbook/bootstrap.md (nothing worth treating as an oracle).
