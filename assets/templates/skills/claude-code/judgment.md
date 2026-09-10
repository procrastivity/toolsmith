Reach for toolsmith when you are turning something that already exists —
a Claude skill, a plugin, a marketplace entry, an output style, scripts
around a prompt — into a single static Go binary, or when you are
starting a new procrastivity-style CLI tool from nothing. toolsmith
instantiates the chassis, audits a tool repo against the contract, and
carries the migration playbook and handoff-kit as files under the
installed binary's own asset tree, so you read them without cloning
anything.
It does not build the tool for you: it hands you a compiling skeleton
and the sequence to work from, and tells you where you have drifted
from the contract.

Reach for it again once a tool exists, to keep its own harness
projections honest: re-run it after upgrading, or to diagnose a stale
or hand-edited install before deciding whether to force past it.

Do not reach for it to make one-off edits inside a tool's generated
harness projection — that projection is regenerated from the tool's own
manifest, not from toolsmith.
