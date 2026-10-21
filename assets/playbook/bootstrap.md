# Bootstrap — a new tool, no legacy

Entry condition: nothing exists worth treating as an oracle. There is no
port spec, no parity gate, and no cutover. What remains is the chassis,
a verb surface you design, and the same hygiene every tool gets.

If something *does* exist — a skill directory, scripts, a prompt with
install steps — you are migrating, not bootstrapping. Start at
`playbook/intake.md`.

---

## 1. Name the verb surface before you name the tool

Write the five or six commands you expect a user or an agent to type,
with their output shape, before any code exists. First decide whether
the tool needs the C8.3 `plumbing` namespace. A tool that ships shapes
must draw this boundary; shapes need deterministic document verbs
under `plumbing`. A tool with no shapes and no verbs serving both
audiences can keep a flat command tree — the boundary is not a
requirement for every tool.

- **Which kind does each verb have?** C3.2's kind describes its
  determinism/execution contract: `plumbing` means deterministic
  output, stable JSON and exit codes, no LLM; `llm` means CLI-side LLM
  shaping; `control-plane` means a live MCP surface. This is separate
  from command-tree placement (the audience axis in C8.3). Deterministic
  does not mean read-only: a plumbing verb may change state.
- **Who needs each verb?** For a tool with the C8.3 boundary, ask:
  does a human type it unprompted; does a skill or script call it for
  its own operation; and, if both, is the deliverable the same?
  Human-only verbs belong in the top-level porcelain tree; automation-
  only verbs belong under `plumbing`. If both audiences need it and
  the deliverable is the same, register an alias pair: one
  implementation/constructor in both trees, with identical flags,
  output, schema, and kind. Pin byte-identical stdout, stderr, and exit
  code; help may differ only in the command path, and failure
  diagnostics use the canonical plumbing spelling. The porcelain
  manifest row carries `alias-of: "<canonical qualified name>"`
  (C3.9), e.g. `"plumbing next"`; the canonical row and all other
  verbs omit it. Each spelling remains a full leaf with independently
  emitted and checked flags, usage, and schema.
  If both audiences need it but the deliverable differs, register a
  split pair: the top-level member is the finished, human-sized form;
  the plumbing member is its deterministic substrate and keeps the full
  audit contract. `--json` remains one contract and must be byte-
  identical across the pair. Split pairs carry no `alias-of` metadata.
  A pair is not an execution gate: both spellings execute. Where the
  boundary stands, C4.3 projects exactly the verbs that are both in the
  `plumbing` namespace and of `plumbing` kind; top-level-only verbs do
  not project. If there is no C8.3 boundary, C4.3's namespace condition
  does not apply.
- **Does the tool own state?** Config and assets are not state (C5.3).
  Real state — a store, a log, per-project facts — is a design decision
  the contract does not make for you, and it belongs in the tool's own
  directory, never in the asset chain.

A tool whose verbs are all `llm` or `control-plane` still gets a binary
(T3). It just projects an empty verb table, and that is a legitimate
shape: the binary owns manifest, install, and doctor from day one.

## 2. Instantiate

```
toolsmith new <tool> --dir <path>
```

For an existing mostly-bare checkout, run `toolsmith new` from inside it;
use `toolsmith new --name <tool>` if its directory name is not the intended
command (for example `LDS` → `lds`). Do not combine `--name` with a positional
name or `--dir`. Public names may have single hyphens between lowercase
alphanumeric segments: `docker-extras` keeps its hyphen in the executable,
module default, config/share paths and harness projection; Go/Nix identifiers
drop it (`dockerextras`) and environment prefixes use underscores
(`DOCKER_EXTRAS`). In-place mode preserves existing planning files and Git
state; reconcile preserved README.md, .gitignore and LICENSE yourself.

Work the printed checklist. `nix build` fails once and prints the real
`vendorHash`; paste it into `flake.nix`. Fill every `TODO(<tool>)`
marker — the skeleton puts one exactly where a human must decide
something.

Stop when `make check` is green and the four skeleton verbs work against
an empty surface. `<tool> install claude-code` into a hermetic
`<TOOL>_CLAUDE_SKILLS_DIR` should produce a stamped tree, `<tool> doctor`
should be clean, and a re-install should report `current`.

## 3. Grow the verb surface

One package per verb under `internal/verbs/<verb>/`, registered in
`internal/cli/root.go`, each constructor ending in
`surface.Annotate(cmd, surface.<Kind>)` (C1.3, C1.4, C3.2). Every verb
writes through the injected streams; forbidigo enforces it (C2.1).

Add an error code to the tool's error package as each refusal appears,
never in advance (C2.5). The exit-code table stays closed (C2.4).

## 4. Ship the first release

Follow `playbook/release-and-hygiene.md`. The short version: tag
`v0.1.0`, push the tag, and let the workflow do everything else. Do this
before the tool feels finished — a release process first exercised at
v1.0 is a release process debugged in public.

## 5. Register the tool

Add a row to toolsmith's TOOLS.md with the contract version the manifest
declares, and run `toolsmith check` against the repo. A fresh
tool built from the skeleton should report no findings; anything it does
report is either a real gap or a checker defect worth fixing here.
