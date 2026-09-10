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

The file matrix never reaches these, and every one of them has bitten a
port:

- **stdin with no file arguments;**
- **two files at once** (line-number offsets depend on whether file one
  ends with a newline);
- **empty input** (usually "no findings", exit 0).

## 5. Assert three things, not one

1. **Byte equality** of stdout, with the exit code appended to the
   captured stream so a mismatch in either shows up as one diff.
2. **No stderr from the port on a clean run.** A port that writes
   diagnostics where the oracle wrote none has changed its output
   contract, even when stdout matches.
3. **Self-determinism**: run the port three times per case and require
   identical output. Map iteration order, concurrent workers, and
   unstable sorts all surface here, and they surface as flakes
   everywhere else.

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

## 9. Record divergences the gate cannot see

Anything the corpus never reaches is invisible to the gate. Mutation
testing during code review finds these; so does reading the oracle's
error paths. Each one goes in `docs/<matter>/parity-divergences.md` with
a repro and a rationale (see `playbook/port-spec.md`).

## 10. Retire it at cutover

The gate and the oracle are deleted together, in one commit, whose
message records that the gate passed at that point. After cutover,
"parity with the old implementation" stops being the specification —
the tool's own tests are.

Keep the final run as an `evidence/` record (C7.3). It is the durable
answer to "was the port ever actually faithful?", and it costs one file.
