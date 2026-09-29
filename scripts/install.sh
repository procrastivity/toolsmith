#!/bin/sh
set -eu

die() {
	printf '%s\n' "toolsmith: $*" >&2
	exit 1
}

if [ -z "${TOOLSMITH_INSTALL_DIR:-}" ]; then
	[ -n "${HOME:-}" ] || die 'HOME is not set; set TOOLSMITH_INSTALL_DIR explicitly'
	TOOLSMITH_INSTALL_DIR=$HOME/.local/bin
fi
TOOLSMITH_VERSION=${TOOLSMITH_VERSION:-}
TOOLSMITH_BASE_URL=${TOOLSMITH_BASE_URL:-https://github.com/procrastivity/toolsmith}

while [ "${TOOLSMITH_BASE_URL%/}" != "$TOOLSMITH_BASE_URL" ]; do
	TOOLSMITH_BASE_URL=${TOOLSMITH_BASE_URL%/}
done
[ -n "$TOOLSMITH_BASE_URL" ] || die 'TOOLSMITH_BASE_URL must not be empty'

os=$(uname -s)
arch=$(uname -m)
case "$os:$arch" in
Linux:x86_64 | Linux:amd64) asset=toolsmith-linux-amd64 ;;
Darwin:arm64) asset=toolsmith-darwin-arm64 ;;
MINGW*:* | MSYS*:* | CYGWIN*:* | Windows_NT:*)
	die 'Windows is not supported by this installer. It does not publish a Windows .exe; if you have a compatible .exe from another source, put it in a directory on PATH.'
	;;
Linux:*) die "unsupported Linux architecture: $arch (supported: amd64)" ;;
Darwin:*) die "unsupported macOS architecture: $arch (supported: arm64)" ;;
*) die "unsupported platform: $os/$arch (supported: Linux amd64 and macOS arm64)" ;;
esac

command -v curl >/dev/null 2>&1 || die 'curl is required'
if command -v sha256sum >/dev/null 2>&1; then
	checksum_tool=sha256sum
elif command -v shasum >/dev/null 2>&1; then
	checksum_tool=shasum
else
	die 'sha256sum or shasum is required to verify the download'
fi

if [ -n "$TOOLSMITH_VERSION" ]; then
	download_url=$TOOLSMITH_BASE_URL/releases/download/$TOOLSMITH_VERSION
else
	download_url=$TOOLSMITH_BASE_URL/releases/latest/download
fi

if ! tmp=$(mktemp -d "${TMPDIR:-/tmp}/toolsmith-install.XXXXXX"); then
	die 'could not create a temporary directory'
fi
install_tmp=
cleanup() {
	rm -rf "$tmp"
	if [ -n "$install_tmp" ]; then
		rm -f "$install_tmp"
	fi
}
trap cleanup 0
trap 'exit 1' HUP INT TERM

printf 'Downloading %s...\n' "$asset"
curl -fsSL "$download_url/$asset" -o "$tmp/$asset" || die "download failed for $asset"
curl -fsSL "$download_url/SHA256SUMS" -o "$tmp/SHA256SUMS" || die 'download failed for SHA256SUMS'

checksum=$(grep "  ${asset}\$" "$tmp/SHA256SUMS" || true)
[ -n "$checksum" ] || die "SHA256SUMS has no entry for $asset"
if [ "$checksum_tool" = sha256sum ]; then
	if ! printf '%s\n' "$checksum" | (cd "$tmp" && sha256sum -c -); then
		die "checksum verification failed for $asset"
	fi
else
	if ! printf '%s\n' "$checksum" | (cd "$tmp" && shasum -a 256 -c -); then
		die "checksum verification failed for $asset"
	fi
fi

install_dir=$TOOLSMITH_INSTALL_DIR
case "$install_dir" in
/*) ;;
*) install_dir=./$install_dir ;;
esac
destination=$install_dir/toolsmith
mkdir -p "$install_dir" || die "could not create install directory: $TOOLSMITH_INSTALL_DIR"
[ ! -d "$destination" ] || die "install destination is a directory: $destination"

if ! install_tmp=$(mktemp "$install_dir/.toolsmith.XXXXXX"); then
	die "could not create a temporary file in $TOOLSMITH_INSTALL_DIR"
fi
cp "$tmp/$asset" "$install_tmp" || die 'could not copy the verified binary'
chmod 755 "$install_tmp" || die 'could not make the installed binary executable'
mv -f "$install_tmp" "$destination" || die "could not install to $destination"
install_tmp=

printf 'Installed %s to %s\n' toolsmith "$destination"
printf 'Next: toolsmith install <harness>\n'
