# The port spec — genre, sections, and where it lives

A **port spec** describes the behavior of an existing implementation
precisely enough that someone can rebuild it in another language without
reading the original. It is not a design document. It records what the
old code *does*, including the parts that are accidents, because an
accident with a consumer is a contract.

**It lives at `docs/<matter>/port-spec.md`, in the repo, committed
(C7.2).** This clause exists because ste9's port spec was written into an
ephemeral scratchpad, ran to 1,409 lines, and never landed — while
twenty shipped Go comments cite it by section number to this day
(`internal/lexicon/parse.go`: "porting spec, §1.2 / §2.1"). Those
citations now point at nothing. Write the spec in the repo from its
first line, and the problem cannot happen.

## When a port spec is worth writing

Write one when the old implementation is an **oracle** (intake step 3):
it has consumers who depend on its exact behavior, or its semantics come
from a runtime you are leaving behind (Python's `\s` class, a shell's
word splitting, a regex engine's ordering).

Skip it when the posture is evidence-only. There is nothing to be
faithful to.

## Sections

Number them. Code comments cite `§N.M`, and a section that gets renamed
breaks every citation, so treat the numbering as an interface.

1. **Scope and judgment calls.** What is in the port, what is dropped,
   and every place you decided to deviate — each with its rationale.
   This section is the one reviewers read first.
2. **Inputs and their loading.** File formats, field counts, encodings,
   what the oracle does with a malformed row (often: nothing, and it
   crashes — say so).
3. **The data model.** Every structure the algorithm walks, in terms of
   the target language, not the source language.
4. **Core algorithms.** One subsection each, with the oracle's exact
   ordering. Ordering is behavior whenever output is a list.
5. **Emission.** What is printed, in what order, with what separators.
   Byte-level. Include the exit codes.
6. **Configuration and environment.** Every variable the oracle reads,
   and whether the port keeps it.
7. **Host-language semantics you must reproduce.** The subtle section:
   character classes, case folding, sort stability, number formatting.
   Measure these against the real runtime rather than trusting your
   memory of them, and record the measured set.
8. **CLI contract.** Arguments, stdin behavior, multi-file behavior,
   empty input, and what each combination exits with.
9. **Known divergences.** Places the port will deliberately differ. Each
   entry names the trigger, both behaviors, and why the new one is
   better.
10. **Verification.** How the parity gate covers each section above, and
    what it cannot reach.

Sections 1–8 describe the oracle. Sections 9–10 describe the port. Keep
that boundary clean: a spec that mixes them cannot be read as "what the
old thing does" by the next person.

## How to write it

- **Read the oracle, do not run it from memory.** Every claim in the
  spec should be traceable to a line of the original.
- **Measure the runtime where semantics are subtle.** ste9's §7 records
  the exact code-point set Python's `\s` matched, produced by running
  Python, not by reading its documentation.
- **Write it before the port, amend it during.** A spec written
  afterward documents the port, which is the one thing you already have.
- **Cite it from the code.** A comment that says *why* a line is shaped
  oddly, plus `(port spec, §7.4)`, is the whole point of the genre.

## Its companion: the divergence list

Deliberate differences accumulate faster than the spec's §9 can hold
them once the port is running. Split them into
`docs/<matter>/parity-divergences.md` when they do (ste9's shape): one
entry per divergence, each with a repro, both behaviors, and the
rationale. The parity gate cannot find these — by definition they are
outside the corpus it runs — so the list is the only record that they
were decided rather than missed.

## Life after cutover

When the oracle is deleted, the port spec stops being normative and
becomes history. Say so in a line at the top rather than deleting it:
the code comments still cite it, and "this described the Python linter,
retired 2026-xx-xx" answers the reader's question completely.
