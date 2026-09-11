# Parity divergences: `toolsmith check` and `toolsmith new`

**Status:** history. Cutover (`toolsmith-binary` Stage 7) deleted both
oracles (`contrib/check-contract`, `contrib/new-tool.sh`) and the parity
gate (`contrib/parity-check`); `make parity` no longer runs. This
document no longer governs `toolsmith check` and `toolsmith new` — their
own tests do.

Matter: `toolsmith-binary`, Stage 4, Step 16. Companion to
`docs/binary/port-spec.md`, whose §9 rules each of these calls. This
document is the list `assets/playbook/parity-gate.md` §9 asks for: one
entry per divergence, with a repro, both behaviors, and the rationale.

**Why the list exists.** The gate proves that the port agrees with the
oracle on the corpus. It cannot prove that a disagreement was decided.
Every entry below is a place where the two implementations differ on
purpose. Without this record, a later reader cannot tell a ruling from
an oversight.

---

## 1. How to read this list

Three classes, and the difference between them matters:

- **Diverged (§2).** The port does not do what the oracle does. The
  parity contract narrows to say so.
- **Reproduced on purpose (§3).** The oracle is wrong, and the port
  copies the defect anyway for the duration of the parity window. These
  are not divergences. They are here because a reader who finds the
  defect in the port must be able to see that it was carried, not
  introduced.
- **Outside the gate's reach (§4).** Neither implementation is wrong.
  No test covers the behavior at all, so the record is the only
  evidence anyone looked.

The rule the rulings follow is stated in port spec §1, and it is quoted
here because every entry in §2 and §3 turns on it:

> Reproduce an accident that mislabels; diverge from an accident that
> misreports conformance.

A wrong clause label still leaves the reader a true statement about the
repo. A false finding tells the reader that a conformant repo is not
conformant. Byte-reproducing the second kind would harden a defect into
the tool that replaces the oracle.

---

## 2. Divergences

### D1 — `check` parses the manifest; the oracle greps it

Port spec §9.1. Gate: `probe-prettyjson`, asserted on the port side
only.

**Oracle.** `contrib/check-contract:74-75` runs
`go run ./cmd/<tool> manifest --json` and greps the raw bytes for
`"manifest_digest":"sha256:` and `"contract":"toolsmith/`. Both patterns
assume compact JSON with no space after the colon.

**Port.** `internal/verbs/check` decodes the document with
`encoding/json` into `internal/manifest.Manifest` and reads the field
values.

**Repro.** Give a tool a `manifest --json` that pretty-prints with
`json.MarshalIndent(m, "", "  ")`. The oracle reports C3.4 and C3.6
against a manifest whose fields are present and correct. The port
reports nothing. `generate_prettyjson_probe` in `contrib/parity-check`
builds exactly this repo.

**Rationale.** The oracle's findings are false. Reproducing them
byte-for-byte would make the replacement tool wrong in the same way.
The parity contract for `check` therefore reads "stdout bytes plus exit
code, **on repos where the oracle's grep is correct**".

**What the gate does with it.** The port's silence is asserted. The
oracle's output on the same probe is printed as information and is not
compared. The oracle is frozen for the parity window, so pinning its
side would catch nothing that the port-side assertion does not already
catch.

### D2 — `check` strips the carriage return before matching a pinned action

Port spec §9.7. Gate: `probe-crlf`, asserted on the port side only.

**Oracle.** `contrib/check-contract:102-107` reads each `uses:` line
with `while IFS= read -r line` over `grep` output, which keeps a
trailing carriage return. It extracts the ref with `${line##*@}` and
`${ref%% *}` (lines 103-104), then tests it against `^[0-9a-f]{40}$`
(line 106). Neither extraction strips a `\r`, because a carriage return
is not a space. On a CRLF-terminated workflow file the ref is 40 hex
characters followed by `\r`, so the test fails.

**Port.** The port strips the carriage return before it matches, so a
pinned action reads as pinned whatever the file's line endings are. No
finding message carries a stray `\r`.

**Repro.** Write `.github/workflows/ci.yml` with CRLF line endings and a
genuinely pinned `- uses: actions/checkout@<40 hex>` line. The oracle
emits `C6.5: ci.yml: action not SHA-pinned: ...<CR>`. The port emits
nothing for that line. `generate_crlf_probe` builds it.

**Rationale.** Same class as D1. A formatting artifact of the input
becomes a conformance verdict. Workflow files with CRLF line endings are
outside the parity contract, alongside D1's pretty-printed manifests.

`assets/playbook/parity-gate.md` §3 names CRLF as a required generated
probe, so the gate carries this exclusion explicitly rather than hitting
it as a failure.

### D3 — `new` follows CONTRACT C2.4 on the failure paths

Port spec §1 judgment call 3, and §9.3. Gate: `run_new_cli_cases`
asserts the divergence itself.

**Oracle.** Every failure in `contrib/new-tool.sh` exits `1` through one
`die()` helper (lines 26-29). A bad flag, a bad name and an existing
target all produce the same code.

**Port.** `2` for argument-parsing failures (Cobra's own usage path),
`3` (`exitcode.Refusal`) for the existing-target refusal, `1` for the
rest.

**Repro.** Run `contrib/new-tool.sh --bogus x` and
`toolsmith new --bogus x`. Both write a message to stderr and produce no
stdout. The exit codes are `1` and `2`.

**Rationale.** The oracle's flat `1` is an artifact of one helper, not a
decision, and its only two callers (`README.md:64`, `Makefile:33`) call
`new` on the happy path. No consumer observes the code. C2.4's table is
a clause the tool owes from its first commit, and reproducing the
accident would mean suppressing Cobra's usage path. Naming the parity
boundary belongs to us (`assets/playbook/parity-gate.md` §1), so the
contract for `new` is the produced tree, the happy-path checklist bytes,
and exit `0`. Failure-path exit codes are excluded.

**What the gate does with it.** It asserts the divergence rather than
ignoring it. Each negative case pins the oracle at `1` and the port at
its expected code, so a silent drift on either side still fails.

### D4 — `check` defaults its path argument to `.`

Port spec §1 judgment call 2. Gate: the zero-argument case in
`run_cli_cases`.

**Oracle.** `contrib/check-contract:13` requires exactly one positional
argument. Zero arguments print a usage line and exit `2`.

**Port.** `cobra.MaximumNArgs(1)`, and the path defaults to `.` when the
argument is absent (`internal/verbs/check/check.go:38-42`). Everything
else is preserved: the path must resolve to a directory, or the verb
exits `2`.

**Repro.** Run `contrib/check-contract` with no arguments in a clean
repo, then `toolsmith check` in the same directory. The oracle exits
`2`. The port audits the current directory and exits `0`.

**Rationale.** The Brief's verb table writes `check [path]`, bracketed.
The bracket is the contract, and the oracle's arity is the accident.
This never touches byte parity, because the gate always passes an
explicit path.

### D5 — `new`'s `--dir` default is `./<name>`

Ruled at Step 15. The port spec does not cover it: §8.2 records the
oracle's default as a fact, and §9 has no entry. Gate: outside the
contract by construction.

**Oracle.** `contrib/new-tool.sh:72` defaults the target to
`<toolsmith-repo>/../<name>`, a sibling of the repository the script
lives in. It resolves the repository from the script's own location
(lines 66-70).

**Port.** The target defaults to `<name>` under the current directory,
resolved in `validate()` rather than in the flag declaration
(`internal/verbs/new/new.go:83-87`).

**Repro.** Run each implementation with no `--dir` from a directory
outside the toolsmith repository. The oracle writes beside the toolsmith
checkout. The port writes under the current directory.

**Rationale.** A binary has no repository to be a sibling of. The
oracle's default is a property of a script that lives inside a checkout,
and it does not survive the port.

**Why it is outside the parity contract.** The gate always passes
`--dir` explicitly, and it must. The oracle prints `$target_dir` back
into the checklist verbatim rather than absolutizing it, so two
different absolute paths would make the checklist differ for a reason
that has nothing to do with the port. Both implementations therefore run
from their own scratch parent with the same relative `--dir` string.

### D6 — `new` reads the skeleton through the asset chain

Ruled at Step 15. Gate: the tree comparison, plus
`TestSkeletonHooksAreExecutable`.

**Oracle.** It resolves `$repo/assets/_skeleton` relative to the
script's own location and copies it with `cp -R`
(`contrib/new-tool.sh:66-70`, line 80).

**Port.** It goes through C5.1's override, default, embedded chain.
`internal/asset` gained `Tree(prefix)` for this, because the existing
`Resolve` is per-file. C5.2's shadow-by-name applies to a tree as a
unit: an override directory replaces the shipped one wholesale, never
merged file by file.

**Consequence, and it is the part worth knowing.** `embed.FS` reports
every file as mode `0444`, so the embedded link cannot carry the
executable bit at all. The two disk links carry real modes. The port
therefore reconstitutes the bit from a shebang when the embedded link
answered, and copies the real mode otherwise
(`internal/verbs/new/instantiate.go:205-216`).

**Repro.** Instantiate from a binary with no override and no
materialized default. The two hooks the skeleton ships
(`contrib/check-commit-msg`, `contrib/check-gofumpt`) come out
executable because both start with `#!`.

**Rationale.** The chain is what C5 requires of a binary. The shebang
rule is an assumption about the skeleton, so a test pins it:
`TestSkeletonHooksAreExecutable` fails if the skeleton ever ships an
executable without a shebang.

### D7 — `new` sends git's streams to stderr

Ruled at Step 15. Gate: the no-stderr assertion covers the port side.

**Oracle.** `contrib/new-tool.sh:112-114` lets `git init`, `git add` and
`git commit` inherit the script's own stdout. All three use `-q`.

**Port.** Both of git's streams go to the error stream
(`internal/verbs/new/instantiate.go:234-236`).

**Repro.** Make one of the git calls talkative, for example by removing
`-q`. The oracle's extra bytes land on stdout, inside the verb's
contract. The port's land on stderr.

**Rationale.** C2.1 makes stdout the verb's own. On a successful run
`-q` means git writes nothing anywhere, so the difference is invisible
in practice. Port spec §5.2 already records that git's silence is not
hermetically guaranteed, and this is where the guarantee is made
structural instead.

### D8 — the tree comparison asserts the executable bit, not the mode

This is a contract boundary rather than a behavior difference, and it is
listed because a reader of the gate will otherwise take the silence for
an oversight.

**Asserted.** The set of relative paths, every file's bytes, and the
executable bit.

**Not asserted.** The full permission mode. The oracle instantiates with
`cp -R`, so its tree carries whatever the developer's umask left on the
checked-out working copy — `664` and `775` on this host. That is a fact
about the checkout, not about the tool, and the port cannot reproduce it
from an embedded asset tree in any case. Comparing full modes would
assert the umask. Comparing the executable bit asserts that an
instantiated tree's hooks still run.

**`.git` is excluded** from the file comparison for the same class of
reason: two independent `git init` runs never produce identical object
stores. What the gate compares instead is what the oracle's three git
calls are for — the branch name, the commit message, and the tree hash
the commit points at. A tree hash is content-derived, so equal hashes
mean both implementations committed the same tree.

---

## 3. Reproduced on purpose

### R1 — the C6.2 else-branch: a mislabel and a coverage hole

Port spec §1 judgment call 1, and §9.6. Gate: `probe-c62-mislabel` and
`probe-c62-hole`, both under full parity.

**The defect.** `contrib/check-contract:86-89` guards the C6.2 tag
comparison on `Makefile` **and** `cliff.toml` both being present, and its
`else` branch tests only `cliff.toml`.

**Repro A, the mislabel.** A repo with a `Makefile` and no `cliff.toml`
reports `C6.3: no cliff.toml (changelog is not derivable from tags)`.
The condition is real. The clause label is wrong: it is a C6.2 concern
reported as C6.3.

**Repro B, the coverage hole.** A repo with a `cliff.toml` and no
`Makefile` reports nothing from this block at all, so the C6.2
`--match 'v[0-9]*'` and `tag_pattern` agreement is never audited.

**Ruling: reproduce both, and schedule the fix.** The gate draws its
authority from the oracle staying fixed for the duration of the port. A
port that improves its own reference mid-flight cannot prove
faithfulness, and both defects sit inside the stdout bytes the contract
covers. Neither one misreports conformance: the mislabel still tells the
reader something true about the repo, and the hole tells the reader
nothing at all.

**Where the fix lives.** Stage 8, `step-18` on this Matter: correct the
label and close the hole after cutover retires the gate. It is
scheduled, not deferred indefinitely.

### R2 — exit `0` with stderr on a repo with several `cmd/` entries

Port spec §9.4. Gate: `probe-multicmd`, under full parity, plus a named
carve-out in the no-stderr assertion.

**The behavior.** A repo with two or more `cmd/<x>` entries that is
otherwise contract-clean makes `check-contract` exit `0` after writing
the "multiple cmd/ entries" note to stderr.

**The tension.** This Matter's workplan layers "no stderr from the port
on a clean run" on top of the parity contract
(`assets/playbook/parity-gate.md` §5). The note breaks the assertion as
an absolute.

**Ruling: keep the note, carve out the assertion.** The note is real
information about an ambiguous repo. Dropping it to satisfy a gate
assertion would trade behavior for convenience. The port writes the same
note to the same stream, and the gate's assertion reads "no stderr on a
clean run, except the multiple-`cmd/`-entries note"
(`contrib/parity-check:162`).

This is not a divergence. It is here so that the carve-out reads as a
decision rather than as a hole in the assertion.

---

## 4. Outside the gate's reach

Nothing below is a difference between the two implementations. Each one
is a behavior that no case in `contrib/parity-check` exercises, so this
record is the only evidence that anyone looked at it.

### U1 — the `set -e` abort paths

Port spec §7. The oracle runs under `set -euo pipefail`, so a
permission-denied repository directory or a disk that fills mid-`new`
aborts the script at the failing command. These are fault-injection
cases. A fixed corpus of real repos cannot produce them, and building a
fault harness costs more than the paths are worth during a parity window
that ends at cutover.

### U2 — a toolchain fault rather than a clean failure

Both implementations shell out to `go run ./cmd/<tool> manifest --json`.
The gate covers what happens when that command fails. It does not cover
what happens when the command hangs, because neither implementation sets
a timeout and the gate would hang with it.

### U3 — the locale-dependent sort

Port spec §7. Every name in the corpus is ASCII, so the sort order the
oracle inherits from the shell's locale and the order the port produces
cannot be told apart. A non-ASCII tool name would separate them. None
exists to test with.

---

## 5. At cutover

`assets/playbook/parity-gate.md` §10 retired the gate in the same commit
that deleted the oracles. This document survived that commit, with a
status line saying it is history.

Two entries had work attached to them at that point:

- **R1** — `step-18` corrects the C6.2 label and closes the coverage
  hole. Both changes are only safe now that the gate no longer requires
  the port to match the oracle. The correction landed at step-18:
  `tagNamespace` now audits Makefile and cliff.toml independently, so a
  Makefile-only repo reports its own `C6.2` finding plus
  `C6.2: no cliff.toml (no tag_pattern for git describe --match to agree
  with)` in place of the old mislabeled `C6.3` line, and a
  cliff.toml-only repo now reports `C6.2: cliff.toml tag_pattern is not
  "v[0-9]*"` when it fails to match, closing the hole. The two generated
  probes are renamed for what they exercise rather than for the defect
  they used to expose: `probe-c62-mislabel` is `probe-c62-no-cliff`, and
  `probe-c62-hole` is `probe-c62-no-makefile`.
- **D3, D4** — the divergent exit codes and the defaulted path became
  plain behavior, described by the tool's own tests rather than by a
  boundary drawn against an oracle.

Everything else in §2 stays true now that the oracles are gone, because
each entry describes what the port does and why. The oracle's half
becomes the historical note.
