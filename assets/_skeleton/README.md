# toolname

TODO(toolname): one paragraph on what this tool is and who it is for.

Built on the [toolsmith contract](https://github.com/procrastivity/toolsmith)
(`toolsmith/v1` — the manifest declares it): a single static Go binary
that projects itself into agent harnesses as generated, stamped skills.

## Install

```
curl -fsSL https://github.com/OWNER/toolname/releases/latest/download/toolname-install.sh | sh
nix profile install github:OWNER/toolname   # alternative: install the binary with Nix
```

The installer puts the `toolname` binary in `~/.local/bin` by default.
Installing the binary is separate from projecting its skills into a
harness; once the binary is on `PATH`, run:

```
toolname install                            # project into every detected harness
```

`toolname install <harness>` targets one harness; `toolname uninstall
<harness>` removes exactly what install wrote; `toolname doctor` reports
stale or drifted projections.

## Optional project-local runtime assets

The root command accepts `--project-root PATH`, but that context does not
make every verb read project files. Only a named runtime reader that opts
in uses it, for known shipped names under
`<root>/.<tool>/assets/<name>`; a missing file falls through to the normal
asset chain, while invalid roots and unsafe or unreadable files refuse.
The project directory is caller-trusted, and the check is not a sandbox
against concurrent writers. Keep config, `Tree`, manifest, install,
doctor, stamps, and embedded `CONTRACT.md` on their existing paths. See
C5.5 in the toolsmith contract.

For a command whose runtime reader opts in, select the root explicitly:

```sh
toolname --project-root /work/acme <runtime-command>
```

The generated chassis has no project-aware runtime reader by default; the
placeholder command must be implemented to call `asset.ResolveIn`.

## Develop

```
direnv allow      # or: nix develop
make check        # lint + test
make hooks        # pre-commit, both stages
```

Version stamps come from release tags (`git describe --match 'v[0-9]*'`);
pushing an annotated `vX.Y.Z` tag is the only human release action.
CHANGELOG.md is generated per release, never committed.

## Design of record

TODO(toolname): name where the design of record lives (a sidecar planning
repo, docs/ — whatever this tool uses), so a reader knows where decisions
come from.
