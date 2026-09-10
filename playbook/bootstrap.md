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
with their output shape, before any code exists. Two questions decide
most of the design:

- **Which verbs are `plumbing`?** Deterministic output, stable JSON,
  stable exit codes, no LLM. Only these project into a harness (C4.3),
  so this is also the answer to "what does the installed skill offer?"
- **Does the tool own state?** Config and assets are not state (C5.3).
  Real state — a store, a log, per-project facts — is a design decision
  the contract does not make for you, and it belongs in the tool's own
  directory, never in the asset chain.

A tool whose verbs are all `llm` or `control-plane` still gets a binary
(T3). It just projects an empty verb table, and that is a legitimate
shape: the binary owns manifest, install, and doctor from day one.

## 2. Instantiate

```
contrib/new-tool.sh <tool> --dir <path>
```

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
declares, and run `contrib/check-contract` against the repo. A fresh
tool built from the skeleton should report no findings; anything it does
report is either a real gap or a checker defect worth fixing here.
