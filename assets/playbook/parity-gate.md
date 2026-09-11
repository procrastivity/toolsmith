# The parity gate — the old implementation as oracle

A parity gate runs both implementations over the same inputs and
requires byte-identical stdout and identical exit codes. It is how a
port stays honest while it is being written, and it is the reason a
cutover can be a judgment call instead of an audit.

Reference implementation: ste9's `scripts/lint_parity.sh`, driven by
`make lint-parity`. The recipe below is that script, generalized.

---

## 1. Decide what "parity" covers

Name the contract before writing the harness. For ste9 it is: **stdout
bytes plus exit code, on well-formed inputs.** Not stderr, not
tracebacks, not performance.

Everything outside that boundary is free to improve, and the freedom is
the point — a gate that demanded stderr parity would have forced the Go
port to reproduce a Python traceback.

Not every verb's payload is stdout. A verb that writes a directory tree
(a scaffolder, an installer) has no stream to diff — its parity contract
covers the tree it produces instead. §5 and §6 say what that means for
the assertion and the report; the rest of this page still applies
unchanged.

## 2. Build the matrix

The gate is a cross-product of every dimension that changes behavior:

```
for mode in <every mode flag>:
  for input in <every corpus file>:
    run oracle, run port, diff
```

The corpus starts as **the files already in the repo**: fixtures,
reference data, README, the tool's own documents. They cost nothing to
add and they exercise real content.

When corpus members come from other repositories rather than living in
this one, pin each by **commit SHA, not branch name**, and extract with
`git archive` from that pinned commit rather than reading a live working
tree. A branch name drifts out from under you — the next commit someone
else makes on that branch silently moves what "the corpus" means, with
the gate unchanged — and a live working tree can sit on the wrong branch
or be mid-edit. A pinned SHA is what makes a later run reproducible;
verify each pin resolves to the recorded commit before extracting
anything, and abort the whole run if one does not, rather than producing
a partial diff against the wrong input. For a conversion whose parity
window spans weeks, treat the pin as a commit from the start —
re-pinning after the fact, once a run has already been taken as a
baseline, is a scheduled failure.

The cost of `git archive`: the extracted tree carries no `.git`, so any
check that depends on git state (tracked vs. untracked, history) is
unreachable from that corpus member. Cover it with a generated probe
instead (§3).

## 3. Add generated probes for what the corpus cannot reach

Checked-in files are well-behaved, which is exactly their weakness.
Generate the awkward ones into a temp directory at run time:

- CRLF line endings;
- non-breaking and other Unicode spaces (they feed `\s`-class regexes);
- Unicode digits and letters (they feed `\w` and `\d`);
- tabs, both as leading indentation and between words;
- an unclosed fenced block, or whatever the equivalent unterminated
  structure is;
- inputs that trip a known quirk you found while writing the port spec.

Generate them with the same interpreter the oracle needs, rather than
embedding them as shell string literals. The code points stay
unambiguous, and no probe file has to survive the repo's hygiene hooks.

## 4. Cover the CLI contract separately

The file matrix never reaches these. Name the dimensions for *this
verb's* CLI shape — do not reuse the list below wholesale, it is one
shape's list, not the definition. It fits an input-to-stdout linter over
files (ste9's own tool, the reference implementation this page
generalizes from), and every one of these has bitten a port of that
shape:

- **stdin with no file arguments;**
- **two files at once** (line-number offsets depend on whether file one
  ends with a newline);
- **empty input** (usually "no findings", exit 0).

A verb shaped differently needs its own list. A verb that takes a single
directory argument and never reads stdin has no "stdin with no files"
case at all — its dimensions are a non-directory argument, the wrong
argument count, and zero arguments. A verb that writes a tree has its
own set again (§1, §5). Work out what this verb's CLI actually varies
over before assuming the list above.

Pair each case with an exit code only when the port keeps the oracle's
exit codes verbatim. When the port deliberately adopts the contract's
own code table instead (C2.4's `exitcode.Silent` and its neighbors), the
matrix still asserts the streams for that case, but names which code it
excludes and why — a deliberate code change is a recorded divergence
(§9), not a case dropped from the matrix.

## 5. Assert three things, not one

1. **Byte equality** of stdout, with the exit code appended to the
   captured stream so a mismatch in either shows up as one diff.
2. **No stderr from the port on a clean run — as a named exception
   list, not an absolute.** A port that writes diagnostics where the
   oracle wrote none has changed its output contract, even when stdout
   matches. But some oracles are not silent themselves: one that exits 0
   on a clean run while writing a legitimate note to stderr (a warning,
   a deprecation notice) sets a real precedent for the port. Do not
   forbid all stderr in that case — list the exact cases the oracle
   itself writes stderr for on a clean run, and require the port's
   stderr to match only those; anything outside the list still fails.
3. **Self-determinism**: run the port three times per case and require
   identical output. Map iteration order, concurrent workers, and
   unstable sorts all surface here, and they surface as flakes
   everywhere else.

### When the payload is a tree, not a stream

Replace "byte equality of stdout" with a tree comparison: the same set
of relative paths, every file's bytes, and the executable bit — not the
full permission mode. A tool that materializes a tree by copying one
(`cp -R` over a checked-out skeleton, say) inherits whatever the
developer's umask left on the source working copy; that is a fact about
the checkout, not about the port, and an embedded asset tree cannot
reproduce it in any case. Comparing full modes would assert the umask;
comparing the executable bit asserts the thing that actually matters —
that an instantiated tree's hooks still run.

Exclude `.git` from the file comparison itself: two independent `git
init` runs never produce identical object stores, so comparing them
byte-for-byte asserts nothing. Assert instead what the oracle's git
calls are actually for — the branch name, the commit message, and the
tree hash the commit points at. A tree hash is content-derived, so equal
hashes mean both implementations committed the same tree, which is the
claim that matters.

Report a tree mismatch as a path-level diff — paths present on only one
side, paths whose bytes or executable bit differ — rather than a unified
diff of stream bytes. §6's discipline still applies: report every
mismatch, fail once.

## 6. Report exhaustively, fail once

Count passes, print every mismatch with a unified diff, and exit
non-zero at the end naming the first mismatch. Never stop at the first
failure: the second and third mismatch usually explain the first, and a
harness that hides them costs you a full run per fix.

## 7. Wire it into the dev loop

A `make` target that builds the binary first, then runs the gate. Keep
it out of the per-commit hook if it is slow, and keep it in `make check`
if it is not.

The gate needs the oracle's runtime available. Say so in the dev shell,
and have the script fail with a clear message when the runtime or the
built binary is missing, rather than producing an empty pass.

## 8. When output bytes are owned by the gate

A verb under a parity contract cannot let the standard error renderer
write to its stderr, and cannot invent a new exit code. That is the case
`exitcode.Silent` exists for (C2.4): a non-zero exit that prints
nothing. Decide it per verb, at port time, and say in the comment that
the parity contract is the reason.

## 9. Record every divergence, and say what the gate does with it

"The gate cannot see this divergence" is the weakest kind of entry, not
the definition of the list. Most divergences the gate can and should
see — say which of these applies:

- **Asserted directly**, when it can be encoded as a probe: the gate
  pins the port's corrected behavior and prints the oracle's alongside
  it as information (an excluded-by-contract case — a format the oracle
  never produces, say).
- **Asserted on both sides**, when the two implementations deliberately
  disagree: the gate pins the oracle at its old value and the port at
  its new one, so a silent drift on *either* side fails the run.
- **Built into the comparison as a boundary**, where the divergence is
  encoded in what "equal" means rather than recorded separately (§5's
  tree comparison excluding `.git` and full permission modes is this
  kind).
- **Genuinely unreachable**, when no corpus member and no generated
  probe can exercise the path at all.

Prefer the first three over the fourth wherever the divergence can be
encoded that way: a recorded exclusion decays silently over time, while
a probe fails loudly the moment it stops holding. Mutation testing
during code review surfaces candidates for all four kinds; so does
reading the oracle's error paths. Whichever kind a divergence is, record
it in `docs/<matter>/parity-divergences.md` with a repro, a rationale,
and which of the four kinds it is — see `playbook/port-spec.md`'s
"Its companion: the divergence list".

## 10. Retire it at cutover

The gate and the oracle are deleted together, in one commit, whose
message records that the gate passed at that point. After cutover,
"parity with the old implementation" stops being the specification —
the tool's own tests are.

Keep the final run as an `evidence/` record (C7.3). It is the durable
answer to "was the port ever actually faithful?", and it costs one file.
