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

## Install

Install the binary with the release installer:

```sh
curl -fsSL https://github.com/procrastivity/toolsmith/releases/latest/download/toolsmith-install.sh | sh
```

It installs `toolsmith` to `~/.local/bin` by default; ensure that directory
is on `PATH`. Set `TOOLSMITH_VERSION` in the environment to pin a release
tag instead of using the latest release. Currently supported binaries are
Linux amd64 and macOS arm64.

This command installs only the binary. Once it is on `PATH`, project its
skills into detected harnesses, or target one harness, separately:

```sh
toolsmith install
toolsmith install <harness>
```

`curl | sh` executes the fetched installer before any checksum check. The
installer verifies the selected binary against `SHA256SUMS` fetched from
the same release before writing the destination; this does not authenticate
the installer script or guarantee a benign origin.

## Two entry modes

- **Migrating something that exists** — a Claude skill, a plugin, a
  marketplace entry, an output style, scripts around a prompt. Start at
  [assets/playbook/intake.md](assets/playbook/intake.md): it classifies what you
  have, maps each part to a contract slot, and tells you what to ask
  the owner when it is not sure. Then
  [assets/playbook/migrate.md](assets/playbook/migrate.md) runs the staged
  conversion.
- **Bootstrapping something new** — no legacy, no parity oracle. Go
  straight to [assets/playbook/bootstrap.md](assets/playbook/bootstrap.md) and
  `toolsmith new`.

## What is in here

| Path | What |
|---|---|
| [CONTRACT.md](CONTRACT.md) | The versioned cross-tool contract (normative). |
| [DECISIONS.md](DECISIONS.md) | Register of decisions that shaped the contract (T-numbers). |
| [TOOLS.md](TOOLS.md) | Fleet register: every tool, its contract version, its status. |
| [assets/playbook/](assets/playbook/) | Intake, migrate, bootstrap, port-spec, parity-gate, new-harness-target, release-and-hygiene. |
| [assets/handoff-kit/](assets/handoff-kit/) | Templates for running a conversion as its own planning sidecar (HANDOFF, seed cards, workplans) — the wip-reboot process, productized. |
| [assets/_skeleton/](assets/_skeleton/) | A compiling Go module with `toolname` placeholders: the chassis, manifest, one worked harness target, release tooling, CI, hygiene. |
| [cmd/toolsmith](cmd/toolsmith) | The binary: its `new` verb instantiates the skeleton, its `check` verb audits a repo against the contract's mechanical clauses. |
| [backport/](backport/) | Punch lists of fixes flowing back into existing tools (wip first). |
| [evidence/](evidence/) | Conformance runs and skeleton smoke-test records. |

## Using it

```
nix develop            # or direnv allow
make check             # shellcheck + instantiate the skeleton and build/test it
make hooks             # pre-commit, both stages
make build             # bin/toolsmith

bin/toolsmith new clast --dir ~/Code/clast-go
cd ~/Code/clast && toolsmith new             # mostly-bare existing project, in place
cd ~/Code/LDS && toolsmith new --name lds     # keep an uppercase or neutral checkout name
bin/toolsmith new docker-extras --dir ~/Code/docker-extras
bin/toolsmith check ~/Code/wip
bin/toolsmith doc                       # list the contract, playbook and kit
bin/toolsmith doc playbook/intake.md    # print one
```

With no positional name, `new` uses the current directory's basename as
the tool name, unless `--name` overrides it in place. `--name` cannot be
combined with a positional name or `--dir`; a positional name selects a fresh target.
Names use lowercase ASCII letters and digits in hyphen-separated segments,
starting with a letter (no leading, trailing or repeated hyphens). For
`docker-extras`, the executable, module default, config/share paths and
harness artifacts retain the hyphen; Go package and Nix binding identifiers
use `dockerextras`, and environment variables use `DOCKER_EXTRAS`.
In-place mode leaves existing `README.md`, `.gitignore`, and `LICENSE`
untouched and refuses other skeleton path collisions before writing. It
does not initialize Git, stage, or commit in this mode; review the generated files and reconcile
the retained files yourself. For an existing implementation rather than
a mostly-bare project, use `toolsmith doc playbook/migrate.md`.

The binary carries CONTRACT.md, the playbook and the handoff-kit. A
conversion therefore needs no clone of this repo: `toolsmith doc` lists
them, and `toolsmith doc <name>` prints one (T28).

A conversion normally runs as an agent-seeded session: create a sidecar
planning repo from `toolsmith doc handoff-kit/sidecar-README.md`, seed
the session with CONTRACT.md + the playbook page for your entry mode +
the handoff, and let the workplans drive. The skeleton and binary do the
mechanical part either way.

## Evolving the conventions

Change CONTRACT.md only through a DECISIONS.md entry (T-number), and
version per T20. When a tool teaches the contract something (the way
ste9 taught it splice targets and duo taught it the six drift states),
the lesson lands here as a decision plus a clause — not as tribal
memory in one repo's comments.
