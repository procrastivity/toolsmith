# Installer back-port punch list — `clast`

Apply this list in the clast repository; nothing here asserts that clast
has already adopted the installer.

- Adapt `assets/_skeleton/scripts/install.sh` as `scripts/install.sh`.
  First inspect clast's current `Makefile` and release workflow to learn
  the actual binary names and platform matrix; adapt the script's binary,
  repository, environment prefix, and supported assets to those published
  targets rather than assuming toolsmith's current matrix.
- Port `assets/_skeleton/scripts/release_install_test.go` to
  `scripts/release_install_test.go`. Keep fake curl/uname, real SHA256SUMS
  from fixture bytes, ordered URL assertions, executable installed bytes,
  mismatch preservation/absence, and temp cleanup.
- Add `scripts/install.sh` to `SHELLCHECK_FILES` in `Makefile`.
- In `.github/workflows/release.yml`, copy the script to
  `dist/clast-install.sh` after the build creates `dist/`; keep
  `SHA256SUMS` limited to the audited published binaries and explicitly
  list the installer in `gh release create` (no glob). Preserve release
  re-gating and tag flow.
- Document the exact curl URL, binary install, and clast's actual existing
  projection invocation separately in `README.md`; explain latest/pinned,
  owner/base URL, `curl | sh` trust implications, checksum scope, and the
  audited platform matrix in the release/hygiene docs. Don't invent a CLI
  command or an unpublished platform asset.
- Verify with clast's existing `make check` and applicable tests. Add and
  run a disposable fake-curl/fake-uname installer fixture against clast's
  built binary and matching `SHA256SUMS`; assert its audited asset names,
  checksum refusal, and install behavior. Inspect that the release copy
  step and explicit published asset agree with README and checksums. Do
  not assume clast has toolsmith's `new` verb or skeleton/drift test.

This is a copied-tool convention (T37), not a claim that C4.1/C6.4 require
all existing tools to publish installers. The tool owner decides whether
and when to adopt it.
