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
