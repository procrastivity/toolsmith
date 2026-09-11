# Evidence — the `doc` verb dogfood run

**Date:** 2026-09-11. **toolsmith:** `0f2f7f1` (clean tree, `bin/toolsmith`
symlinked from `~/.local/bin/toolsmith`). **Environment:** bubblewrap
0.11.2, linux/x86_64, go1.24.10.

## 1. What this record is

This evidences the `doc-verb` Matter's seal condition: "from a directory
outside any toolsmith clone, with HOME pointed at a scratch directory, the
binary on PATH lists and prints CONTRACT.md, every playbook page and every
kit file; the installed skill tells the reader to do so." It is the
`doc-verb` counterpart to `evidence/2026-09-11-toolsmith-conformance.md`
and follows CONTRACT.md's C7.3: "`evidence/` holds verification records:
parity runs, dogfood passes, conformance reports — committed beside the
code they verify." DECISIONS.md T28 records why the verb exists. The
claude-code skill promised the playbook "without cloning anything", but
no verb read it, only the nix package wrote it to disk, and the contract
did not ship at all. `clast-conversion` found the gap when it set out to
dogfood the installed binary. This record shows that `toolsmith doc`
closes it.

## 2. Method

All work ran under `$S`, a fresh scratch directory:

```
/tmp/claude-1000/-home-dev-Code-toolsmith/041141c1-6245-4c2b-b8d2-dca7aaecaabc/scratchpad/dogfood
```

Part A (setup) ran with the toolsmith repo visible, to produce reference
values only — it read no doc content through the binary.

```
$ toolsmith version
toolsmith version 0f2f7f1 (commit 0f2f7f1, built 2026-09-11T08:17:45Z)
$ readlink -f ~/.local/bin/toolsmith
/home/dev/Code/toolsmith/bin/toolsmith
$ cp /home/dev/Code/toolsmith/bin/toolsmith $S/bin/toolsmith
$ sha256sum /home/dev/Code/toolsmith/bin/toolsmith $S/bin/toolsmith
aa4cb15d...e68aab  /home/dev/Code/toolsmith/bin/toolsmith
aa4cb15d...e68aab  <S>/bin/toolsmith
```

The copied binary is byte-identical to the repo build. Reference
checksums for CONTRACT.md and every file under `assets/playbook/` and
`assets/handoff-kit/` were written to `$S/ref.sha256` as
`<sum>  <docname>`, dropping the `assets/` prefix.

Part B (the probe) ran with the repo hidden, using bubblewrap:

```
bwrap --dev-bind / / --tmpfs /home/dev/Code \
  --setenv HOME $S/home --setenv XDG_CONFIG_HOME $S/home/.config \
  --chdir $S --setenv PATH $S/bin:/usr/bin:/bin -- <cmd>
```

`bwrap` was available on `PATH` (0.11.2, from the nix profile) and ran
without needing a fallback to `unshare` or `strace`. `--tmpfs
/home/dev/Code` replaces the whole toolsmith parent directory with an
empty, unbacked filesystem inside the sandbox's mount namespace, so no
inode under it — not the repo, not any sibling clone — is reachable by
any path lookup the process makes, regardless of what the process asks
for. This is stronger than blocking specific paths: it proves unreachability
by removing the directory itself, not by hoping nothing tries to open it.
Confirmed before any doc command ran:

```
$ bwrap ... -- bash -c 'ls /home/dev/Code 2>&1; which toolsmith'
(ls produced no output — the directory exists and is empty)
/tmp/.../dogfood/bin/toolsmith
```

`ls /home/dev/Code` returned nothing (empty tmpfs, not an error — the
directory exists but is empty, which is what a bind-mounted tmpfs looks
like) and `which toolsmith` resolved to the copied binary, not the repo
build. Every command in §3 onward ran through this same `bwrap`
invocation with `HOME` and `XDG_CONFIG_HOME` under `$S/home`, so no
config carried over from the real host home either.

## 3. Results

### version and doctor, before install

```
$ toolsmith version
toolsmith version 0f2f7f1 (commit 0f2f7f1, built 2026-09-11T08:17:45Z)
$ toolsmith doctor
claude-code: missing at <S>/home/.claude/skills/toolsmith
no issues found
exit=0
```

### install claude-code, and the skill's `doc` coverage

```
$ toolsmith install claude-code
installed claude-code skill at <S>/home/.claude/skills/toolsmith
exit=0
$ toolsmith doctor
claude-code: current at <S>/home/.claude/skills/toolsmith
no issues found
exit=0
```

The verb-table row for `doc` in `SKILL.md`:

```
| `toolsmith doc [path]` | list the contract, playbook and handoff-kit this binary carries, or print one |
```

Every `SKILL.md` line mentioning `toolsmith doc`:

```
`toolsmith doc` lists them and `toolsmith doc <name>` prints one, so you
read them without cloning anything. A conversion starts at
`toolsmith doc playbook/intake.md`.
| `toolsmith doc [path]` | list the contract, playbook and handoff-kit this binary carries, or print one |
Read them with `toolsmith doc` (for example,
`toolsmith doc CONTRACT.md`), not from a clone of the toolsmith repo. The
```

`grep -c 'assets/playbook' SKILL.md` → `0`. The skill never names an
`assets/` path; it names docs the way a reader would ask for them
(`playbook/intake.md`, `CONTRACT.md`).

### `toolsmith doc` — the full list

```
$ toolsmith doc
CONTRACT.md
handoff-kit/HANDOFF-template.md
handoff-kit/SEED-CARDS-template.md
handoff-kit/sidecar-README.md
handoff-kit/workplan-template.md
playbook/bootstrap.md
playbook/intake.md
playbook/migrate.md
playbook/new-harness-target.md
playbook/parity-gate.md
playbook/port-spec.md
playbook/release-and-hygiene.md
```

Count: **12**. That is CONTRACT.md, 4 handoff-kit files and 7 playbook
pages. The protocol for this run said 11, which was a miscount in the
protocol: the playbook had 7 pages before this Matter started. The list
matches the repo's `assets/playbook/` and `assets/handoff-kit/`
directories file for file.

### Checksums and source, every listed doc

| doc name | sha256 match against `$S/ref.sha256` | source |
|---|---|---|
| CONTRACT.md | match | embedded |
| handoff-kit/HANDOFF-template.md | match | embedded |
| handoff-kit/SEED-CARDS-template.md | match | embedded |
| handoff-kit/sidecar-README.md | match | embedded |
| handoff-kit/workplan-template.md | match | embedded |
| playbook/bootstrap.md | match | embedded |
| playbook/intake.md | match | embedded |
| playbook/migrate.md | match | embedded |
| playbook/new-harness-target.md | match | embedded |
| playbook/parity-gate.md | match | embedded |
| playbook/port-spec.md | match | embedded |
| playbook/release-and-hygiene.md | match | embedded |

All 12 sums matched the reference file (`diff` between the sorted
reference and probe checksum files produced no output). All 12 sources
report `embedded` via `toolsmith doc -v <name>`, for example:

```
$ toolsmith doc -v CONTRACT.md 2>&1 >/dev/null
doc: serving CONTRACT.md from embedded
```

No share tree exists next to `$S/bin` (`ls $S/bin` shows only the
`toolsmith` binary), so `embedded` is the only source available in this
sandbox, and the binary reported it correctly for every doc.

### The kit's own instruction, followed literally

`toolsmith doc handoff-kit/sidecar-README.md`'s "Setting one up" block:

```
mkdir ../<tool>-reboot && cd ../<tool>-reboot && git init
toolsmith doc handoff-kit/HANDOFF-template.md    > HANDOFF.md
toolsmith doc handoff-kit/SEED-CARDS-template.md > SEED-CARDS.md
mkdir workplans     # only when the conversion has no tracker — see below
```

Followed (skipping the `git init`, since this is a probe, not a new
conversion) from `$S`:

```
$ toolsmith doc handoff-kit/HANDOFF-template.md > HANDOFF.md
$ toolsmith doc handoff-kit/SEED-CARDS-template.md > SEED-CARDS.md
$ wc -l HANDOFF.md SEED-CARDS.md
139 HANDOFF.md
 62 SEED-CARDS.md
```

Both files are non-empty and both commands succeeded with the repo
unreachable, exactly as the block instructs a new conversion to run them.

### The front door

```
$ toolsmith doc playbook/intake.md | sed -n 1,14p
# Intake — classify the thing you were pointed at
...
Every page this playbook names is read with `toolsmith doc <name>` (for
example, `toolsmith doc playbook/migrate.md`); `toolsmith doc` on its
own lists them all.
...
```

Lines 8–10 tell the reader how to read every other page the playbook
names: through `toolsmith doc`, not a clone.

### Negative cases

| command | stdout | stderr | exit |
|---|---|---|---|
| `toolsmith doc _skeleton/Makefile` | empty | `toolsmith: doc: no doc named "_skeleton/Makefile"; run 'toolsmith doc' to list them` | 1 |
| `toolsmith doc ../CONTRACT.md` | empty | `toolsmith: doc: no doc named "../CONTRACT.md"; run 'toolsmith doc' to list them` | 1 |
| `toolsmith doc --json nope` | empty | `{"error":{"code":"not-found.doc","message":"no doc named \"nope\"; run 'toolsmith doc' to list them"}}` | 1 |

All three exit 1 with empty stdout. The plain-text cases carry the same
message on stderr without a machine code; the `--json` case carries the
`not-found.doc` code, as expected. `_skeleton/Makefile` (a real chassis
file, but not a doc) and `../CONTRACT.md` (a path escape attempt) are
both rejected the same way a made-up name is — the doc set is a fixed
list, not a filesystem lookup.

### Uninstall

```
$ toolsmith uninstall claude-code
uninstalled claude-code skill from <S>/home/.claude/skills/toolsmith
exit=0
```

## 4. What this does not prove

It proves nothing about a nix-installed share tree. This run had no share
tree next to the binary, so every doc came from the `embedded` link. The
only evidence for the `default` link is an unrecorded check during
`doc-verb` step-02: `./result/bin/toolsmith doc -v playbook/intake.md`
after `nix build` reported `default` at the store path. The e2e suite
covers the override link.

It does not prove that the pages are enough to carry out a conversion.
The `clast-conversion` intake will test that, by reading these same pages
through this same verb.

The doc count is not fixed. The binary carries whatever
`assets/playbook/` and `assets/handoff-kit/` hold at build time, so a new
page appears in the list with no code change.
