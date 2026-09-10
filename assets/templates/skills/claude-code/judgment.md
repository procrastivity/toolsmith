Reach for toolsmith when you are turning something that already exists —
a Claude skill, a plugin, a marketplace entry, an output style, scripts
around a prompt — into a single static Go binary, or when you are
starting a new procrastivity-style CLI tool from nothing. toolsmith
instantiates the chassis, audits a tool repo against the contract, and
carries the migration playbook and handoff-kit as assets you can read
straight out of the binary's install — check the verb table below for
which of that is available as a verb today versus still run by hand from
the repo. It does not build the tool for you: it hands you a compiling
skeleton and the sequence to work from, and tells you where you have
drifted from the contract.

Do not reach for it to make one-off edits inside a tool's generated
harness projection — that projection is regenerated from the tool's own
manifest, not from toolsmith.
