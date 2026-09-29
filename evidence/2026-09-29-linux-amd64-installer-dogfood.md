# Evidence — published Linux amd64 curl installer dogfood

**Date:** 2026-09-29. **Release:** `v0.0.2`. **Source checkout:**
`installer` at `482c8005406a6e8fab96781ee810f8ca03b340ef` (clean before
this evidence file). **Container:** `debian:bookworm-slim`, Linux amd64,
image digest `sha256:3783cc01769c7b2b1b83a5c5ad96c815348e28ed7da68e2e3687004faa906251`
(image ID `sha256:db9f02c6bde9fa90cc8074c92754b2b046947392f1a857726e77f041febb7b82`).

## Release observed

GitHub's [`releases/latest` API](https://api.github.com/repos/procrastivity/toolsmith/releases/latest)
reported tag `v0.0.2`, published `2026-09-29T18:40:44Z`. The release
contains [`toolsmith-install.sh`](https://github.com/procrastivity/toolsmith/releases/download/v0.0.2/toolsmith-install.sh),
[`SHA256SUMS`](https://github.com/procrastivity/toolsmith/releases/download/v0.0.2/SHA256SUMS),
and [`toolsmith-linux-amd64`](https://github.com/procrastivity/toolsmith/releases/download/v0.0.2/toolsmith-linux-amd64).
The latest installer asset digest reported by GitHub is
`sha256:6e03997efc4aad0eb49fc37abeb1ae791213e513ef3771d5dae783e83ee8a092`.

## Reproduction

This starts a new, disposable amd64 container, installs only the tools
needed to run the published shell installer, confirms `toolsmith` is not
already on `PATH`, then runs the real latest-release `curl | sh` command.
It independently fetches the release `SHA256SUMS` and compares its
`toolsmith-linux-amd64` entry to the hash of the installed file itself.

```sh
docker run --rm --platform linux/amd64 \
  debian@sha256:3783cc01769c7b2b1b83a5c5ad96c815348e28ed7da68e2e3687004faa906251 \
  sh -ec '
    apt-get update -qq >/dev/null 2>&1
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends \
      bash ca-certificates coreutils curl >/dev/null 2>&1
    rm -rf /var/lib/apt/lists/*
    ! command -v toolsmith >/dev/null 2>&1
    mkdir -p /opt/toolsmith/bin
    export TOOLSMITH_INSTALL_DIR=/opt/toolsmith/bin
    bash -o pipefail -c "curl -fsSL https://github.com/procrastivity/toolsmith/releases/latest/download/toolsmith-install.sh | sh"
    export PATH=/opt/toolsmith/bin:$PATH
    installed=$(command -v toolsmith)
    test "$installed" = /opt/toolsmith/bin/toolsmith
    test -x "$installed"
    curl -fsSL https://github.com/procrastivity/toolsmith/releases/latest/download/SHA256SUMS -o /tmp/SHA256SUMS
    expected=$(awk '\''$2 == "toolsmith-linux-amd64" {print $1}'\'' /tmp/SHA256SUMS)
    actual=$(sha256sum "$installed" | awk '\''{print $1}'\'')
    test -n "$expected"
    test "$actual" = "$expected"
    printf "asset=toolsmith-linux-amd64\nexpected_sha256=%s\ninstalled_sha256=%s\n" "$expected" "$actual"
    version=$(toolsmith version)
    printf "version_output=%s\n" "$version"
    case "$version" in "toolsmith version v0.0.2 "*) ;; *) exit 1 ;; esac
  '
```

## Result

Exit code: **0**. The installer printed `toolsmith-linux-amd64: OK` and
installed `/opt/toolsmith/bin/toolsmith` as executable. The independent
comparison matched:

```text
asset=toolsmith-linux-amd64
expected_sha256=870e2312f8b91aa8c4a91de8ca5813c3e6481294652b4502fd5aa95b0d052188
installed_sha256=870e2312f8b91aa8c4a91de8ca5813c3e6481294652b4502fd5aa95b0d052188
version_output=toolsmith version v0.0.2 (commit 482c800, built 2026-09-29T18:40:31Z)
```

The installer, binary, and checksum file were fetched over the public
GitHub release URLs inside the container; no local script or binary was
mounted or substituted.
