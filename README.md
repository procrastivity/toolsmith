# toolsmith

The contract, playbook, and skeleton for procrastivity-style CLI tools:
agent-friendly Go binaries that install themselves into agent harnesses
(Claude Code, Codex, Pi, and kin) as generated, stamped, drift-checked
skills.

The shape was built once, deliberately, for `wip`; proven again by
`duo`; and re-derived at real cost by `ste9`. This repo exists so the
next tool (`clast` is first) copies instead of re-derives — and so the
conventions themselves have one home where their evolution is tracked
by decision.

## The idea in four commitments

1. Ship each tool as a **single static Go binary**.
2. Ship **tunable assets next to the binary**, not embedded in it
   (override → shipped default → embedded fallback).
3. The binary emits a **manifest** — its machine-readable
   self-description — and every harness integration is **generated
   from it**, stamped, and drift-checked. Nothing harness-side is
   hand-authored except per-harness judgment prose.
4. Install is two-step: put the binary on PATH, then
   `<tool> install <harness>` projects it into each harness.

The full normative text is [CONTRACT.md](CONTRACT.md), with numbered
clauses the conformance checker and tool code comments cite.

## Two entry modes

- **Migrating something that exists** — a Claude skill, a plugin, a
  marketplace entry, an output style, scripts around a prompt. Start at
  [playbook/intake.md](playbook/intake.md): it classifies what you
  have, maps each part to a contract slot, and tells you what to ask
  the owner when it is not sure. Then
  [playbook/migrate.md](playbook/migrate.md) runs the staged
  conversion.
- **Bootstrapping something new** — no legacy, no parity oracle. Go
  straight to [playbook/bootstrap.md](playbook/bootstrap.md) and
  `contrib/new-tool.sh`.

## What is in here

| Path | What |
|---|---|
| [CONTRACT.md](CONTRACT.md) | The versioned cross-tool contract (normative). |
| [DECISIONS.md](DECISIONS.md) | Register of decisions that shaped the contract (T-numbers). |
| [TOOLS.md](TOOLS.md) | Fleet register: every tool, its contract version, its status. |
| [playbook/](playbook/) | Intake, migrate, bootstrap, port-spec, parity-gate, new-harness-target, release-and-hygiene. |
| [handoff-kit/](handoff-kit/) | Templates for running a conversion as its own planning sidecar (HANDOFF, seed cards, workplans) — the wip-reboot process, productized. |
| [skeleton/](skeleton/) | A compiling Go module with `toolname` placeholders: the chassis, manifest, one worked harness target, release tooling, CI, hygiene. |
| [contrib/new-tool.sh](contrib/new-tool.sh) | Instantiates the skeleton: copy + mechanical rename + first commit + checklist. |
| [contrib/check-contract](contrib/check-contract) | Audits any tool repo against the contract's mechanical clauses; findings by clause ID. |
| [backport/](backport/) | Punch lists of fixes flowing back into existing tools (wip first). |
| [evidence/](evidence/) | Conformance runs and skeleton smoke-test records. |

## Using it

```
nix develop            # or direnv allow
make check             # shellcheck + instantiate the skeleton and build/test it
make hooks             # pre-commit, both stages

contrib/new-tool.sh clast --dir ~/Code/clast-go
contrib/check-contract ~/Code/wip
```

A conversion normally runs as an agent-seeded session: create a sidecar
planning repo from [handoff-kit/](handoff-kit/), seed the session with
CONTRACT.md + the playbook page for your entry mode + the handoff, and
let the workplans drive. The skeleton and scripts do the mechanical
part either way.

## Evolving the conventions

Change CONTRACT.md only through a DECISIONS.md entry (T-number), and
version per T20. When a tool teaches the contract something (the way
ste9 taught it splice targets and duo taught it the six drift states),
the lesson lands here as a decision plus a clause — not as tribal
memory in one repo's comments.
