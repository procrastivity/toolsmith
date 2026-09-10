# Migrate — an existing tool becomes a Go binary

Entry condition: `assets/playbook/intake.md` produced an intake sheet. You know
the shape, the planned verb surface, which parts are oracles, and which
questions the owner still owes you.

The sequence below is proven twice: wip (evidence-only posture, orphan
branch, sixteen Matters) and ste9 (oracle posture, byte-parity linter,
splice installer). Stages are ordered by what they unblock, not by
importance. Nothing here forbids running two stages at once — stage 5
and stage 6 are meant to overlap.

---

## Stage 0 — decide the entry posture (before any code moves)

Two calls, both from intake, both written down:

- **Oracle or evidence-only**, per part (intake step 3). This decides
  whether stage 4 carries a parity gate at all.
- **Where the old line lives during the conversion.** The default is:
  the old implementation stays installed and keeps working. It is not
  a branch you abandon; it is the thing the user reaches for until the
  new one is better.

## Stage 1 — settle packaging before writing verbs

Read CONTRACT.md end to end and record the tool-specific answers it
leaves open:

- module path and binary name;
- the error package name (`<tool>err`) and env prefix (`<TOOL>_`);
- harnesses for v1 (default: claude-code only — the rest are backlog);
- assets: what ships, what is encumbered (C5.4), what must be
  materialized as real files (C4.9);
- splice targets and lifecycle hooks, if intake found any (C4.8, C4.10);
- state, if there is any that is neither config nor asset.

These answers belong in the conversion's sidecar, not in your head — see
`assets/handoff-kit/`. A conversion that skips the sidecar re-derives its own
decisions three sessions later.

## Stage 2 — cut the branch, leave the old line running

Cut an **orphan branch** in the existing repo (wip's H2). The new line
gets fresh history and conventional commits from commit one (C6.6); the
default branch keeps serving the old implementation until cutover.

Why an orphan branch and not a new repo: the tool keeps one name, one
issue history, and one release stream. Why not a feature branch: the new
history is not a continuation of the old one, and a merge base you never
intend to merge across is a lie that git will keep asking you about.

Record the cutover as a **deferred backlog entry** now, so nobody treats
the branch flip as an implicit stage 7 deliverable.

## Stage 3 — chassis first, from the skeleton

```
contrib/new-tool.sh <tool> --dir <path-to-worktree>
```

Then work the checklist the script prints. Two of its items are
judgment, not mechanics:

- **`nix build` fails once and prints the real `vendorHash`.** Paste it
  in. This is expected, not a defect.
- **Fill every `TODO(<tool>)` marker.** The skeleton ships them exactly
  where a human must decide something: the root command's one-line
  summary, the README, the flake description, the judgment asset, the
  skill description.

Stop when `make check` is green and `<tool> version`, `<tool> manifest
--json`, `<tool> install claude-code`, `<tool> doctor` all work with an
empty verb surface. That is a working tool with nothing to do yet, and
it is the correct place to be.

## Stage 4 — port the verbs, one at a time

For every part intake marked **oracle**:

1. Write the port spec first (`assets/playbook/port-spec.md`). It lives at
   `docs/<matter>/port-spec.md`, committed, with numbered sections that
   code comments cite (C7.2).
2. Stand up the parity gate before the second verb lands
   (`assets/playbook/parity-gate.md`). A gate written after the port is a test
   of what you built, not of what you promised.
3. Port behavior, not structure. The Go package layout follows the
   contract (C1.4), never the old file layout.

For every part marked **evidence-only**: design fresh against the
contract, and consult the old implementation only for the "feel
signals" intake recorded (verb names, lint posture, dev-loop verbs).

Two recurring traps:

- **stdout ownership.** A verb whose output bytes are under a parity
  contract cannot let the error renderer write to its stderr. That is
  what `exitcode.Silent` is for (C2.4). Decide per verb, at port time.
- **Kind annotation.** Every verb ends its constructor with
  `surface.Annotate(cmd, surface.<Kind>)` (C3.2). The manifest walk
  hard-errors without it, so this fails loudly — but it fails at
  runtime, so run `<tool> manifest --json` after every verb lands.

## Stage 5 — let the manifest populate

The manifest is not a stage. It is a consequence: it grows as verbs
register, and it needs no work beyond the annotation each verb already
carries. Check it after each verb (`<tool> manifest --json`) and treat
any surprise as a defect in the verb, not in the manifest.

## Stage 6 — the install target, late and in parallel

Start the harness projection once the verb surface is roughly the shape
it will ship in, and run it alongside the remaining verbs. It needs only
the chassis (C4), so it is never blocked on the port finishing.

The skeleton's claude-code target is already worked; what you write is
the judgment prose (`assets/templates/skills/claude-code/judgment.md`)
and the skill description. Both are the only hand-authored strings in
the generated tree (C4.4). Say when to reach for the tool, not what its
verbs are — the verb table is generated.

Extra harnesses: `assets/playbook/new-harness-target.md`, one sitting each.
Splice targets, emit-only targets, and per-target variants (C4.8) get
byte-golden tests and a one-time backup, without exception.

## Stage 7 — cutover, as a judgment call

The criterion is **"the new one is the one I reach for,"** never a
parity checklist and never a feature count. Parity gates tell you the
port is faithful; they do not tell you the tool is better.

When it is time:

1. flip the default branch, rename the old branch to `<name>-final`;
2. uninstall the old line's harness integration by its own uninstaller,
   then `<tool> install <harness>` for each target;
3. delete the oracle and the parity gate in the same commit, and say in
   the message that the gate passed at the point of deletion;
4. remove the old install machinery (install.sh, marketplace entry,
   plugin.json) — it is replaced, not ported (intake step 1).

## Stage 8 — reconcile and register

The last step of every conversion:

- run `contrib/check-contract <repo>` and clear the findings, or record
  why a finding stands;
- re-read CONTRACT.md against what you actually built, and fold each
  divergence back into the sidecar's decisions as an
  implementation-forced note;
- if the conversion taught the contract something new, that is a
  DECISIONS.md entry in toolsmith plus a clause — not a comment in your
  repo (README, "Evolving the conventions");
- add the tool to toolsmith's TOOLS.md with its contract version;
- write `evidence/` records for the parity gate's final run and the
  conformance run (C7.3).
