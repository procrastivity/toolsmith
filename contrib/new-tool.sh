#!/usr/bin/env bash
# contrib/new-tool.sh — instantiate the skeleton as a new tool.
#
# Usage:
#   contrib/new-tool.sh <name> [--dir <target-dir>] [--module <module-path>] [--no-git]
#
#   <name>     the tool's name: lowercase letters and digits, starting with
#              a letter (it becomes the binary name, package names, env
#              prefix, and paths).
#   --dir      where to create the tool (default: ../<name> relative to the
#              toolsmith repo).
#   --module   Go module path (default: github.com/procrastivity/<name>).
#   --no-git   skip git init + first commit (the Makefile smoke test uses
#              this).
#
# The mechanical rename is exhaustive by construction: the skeleton uses
# exactly three placeholder spellings — `toolname` (identifiers, paths,
# module segment), `TOOLNAME` (env prefix), and nothing else — so three
# substitutions plus two directory renames produce a compiling tool. The
# judgment steps that remain are printed as a checklist at the end.
set -euo pipefail

die() {
  echo "new-tool: $*" >&2
  exit 1
}

name=""
target_dir=""
module=""
do_git=1

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dir)
      [[ $# -ge 2 ]] || die "--dir needs a value"
      target_dir="$2"
      shift 2
      ;;
    --module)
      [[ $# -ge 2 ]] || die "--module needs a value"
      module="$2"
      shift 2
      ;;
    --no-git)
      do_git=0
      shift
      ;;
    -*)
      die "unknown flag $1"
      ;;
    *)
      [[ -z "$name" ]] || die "only one <name> argument is accepted (got \"$name\" and \"$1\")"
      name="$1"
      shift
      ;;
  esac
done

[[ -n "$name" ]] || die "usage: contrib/new-tool.sh <name> [--dir <target-dir>] [--module <module-path>] [--no-git]"
[[ "$name" =~ ^[a-z][a-z0-9]*$ ]] || die "name must be lowercase letters and digits, starting with a letter (it becomes Go package names): got \"$name\""
[[ "$name" != "toolname" ]] || die "\"toolname\" is the placeholder itself; pick a real name"

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_dir="$(dirname "$script_dir")"
skeleton_dir="$repo_dir/skeleton"
[[ -d "$skeleton_dir" ]] || die "skeleton not found at $skeleton_dir"

[[ -n "$target_dir" ]] || target_dir="$repo_dir/../$name"
[[ ! -e "$target_dir" ]] || die "$target_dir already exists; refusing to write into it"

[[ -n "$module" ]] || module="github.com/procrastivity/$name"

upper_name="$(tr '[:lower:]' '[:upper:]' <<<"$name")"

mkdir -p "$target_dir"
cp -R "$skeleton_dir/." "$target_dir/"

# Directory renames first, so the content pass sees final paths.
mv "$target_dir/cmd/toolname" "$target_dir/cmd/$name"
mv "$target_dir/internal/toolnameerr" "$target_dir/internal/${name}err"

# Content pass: module path first (it contains `toolname` as a segment,
# and must not be re-hit by the generic rename), then the two placeholder
# spellings. LC_ALL=C keeps sed byte-oriented and predictable.
while IFS= read -r -d '' f; do
  LC_ALL=C sed -i \
    -e "s|github.com/procrastivity/toolname|$module|g" \
    -e "s|toolname|$name|g" \
    -e "s|TOOLNAME|$upper_name|g" \
    "$f"
done < <(find "$target_dir" -type f -print0)

if [[ $do_git -eq 1 ]]; then
  git -C "$target_dir" init -q -b main
  git -C "$target_dir" add -A
  git -C "$target_dir" commit -q -m "chore: instantiate $name from the toolsmith skeleton"
fi

cat <<EOF
instantiated $name at $target_dir (module $module)

Checklist — the judgment steps the rename cannot do:
  1. grep -rn 'TODO($name)' — fill every marker: root Short, README,
     flake meta.description, the judgment and agent-guidance assets, the
     skill description.
  2. cd $target_dir && CGO_ENABLED=0 go build ./... && go test ./...
     (should already pass; it did in the skeleton).
  3. nix build — it fails once and prints the real vendorHash; paste it
     into flake.nix.
  4. make hooks — installs both pre-commit stages.
  5. Decide the verb surface; register verbs in internal/cli/root.go,
     one package each, every constructor ending in surface.Annotate.
  6. For a migration (not a fresh tool): follow toolsmith's
     playbook/migrate.md — port spec, parity gate, cutover.
  7. Run contrib/check-contract $target_dir from the toolsmith repo and
     clear any findings.
  8. Add the tool to toolsmith's TOOLS.md.
EOF
