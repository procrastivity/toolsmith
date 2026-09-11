Every verb writes only its payload to stdout — pass `--json` for a single
parseable value, `-v/--verbose` for extra diagnostic lines on stderr. On
failure stdout is empty; read the error from stderr or the `--json` error
envelope, never from exit code alone.

Everything under a harness's generated skill directory (this file
included) is written by `toolsmith install <harness>` from the binary's
own manifest — never hand-edit it. If it looks stale or wrong, re-run
`toolsmith install <harness>` (it refuses to clobber a hand-edited target
without `--force`) or run `toolsmith doctor` first to see what drifted.

CONTRACT.md and the playbook are the source of truth for how this tool
and the tools it produces are shaped; nothing generated here overrides
them. Read them with `toolsmith doc` (for example,
`toolsmith doc CONTRACT.md`), not from a clone of the toolsmith repo. The
binary's copy of the contract is the one its `check` audits against.
