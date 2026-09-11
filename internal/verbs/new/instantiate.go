package new

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/procrastivity/toolsmith/internal/asset"
	"github.com/procrastivity/toolsmith/internal/toolsmitherr"
)

// skeletonPrefix names the shipped skeleton subtree inside the asset tree.
// The leading underscore is not decoration: a directory containing a file
// named exactly go.mod cannot be embedded ("cannot embed directory X: in
// different module" — assets/assets.go's "all:" prefix does not help).
// Renaming go.mod away is not sufficient by itself either: once it's gone,
// the tree's .go files become ordinary packages of *this* module, and
// `go build ./...`, `go vet ./...`, and `go test ./...` break on 23
// unresolved imports the skeleton was never meant to resolve here. The "_"
// prefix is what fixes that half — the go tool skips underscore-prefixed
// directories when expanding "./..." — and go.mod.tmpl / go.sum.tmpl
// (renamed back to go.mod / go.sum by destPath below) is what fixes the
// embed half. Do not "fix" either one by reverting the name or the suffix;
// a future reader who does will reintroduce both failures. assets.go's
// "all:" prefix is also what makes the skeleton's dotfiles (.envrc,
// .github/, .gitignore, …) embed at all — a bare pattern silently omits
// them — and TestInstantiatedTree guards that they survive instantiation.
const skeletonPrefix = "_skeleton"

// placeholder spellings the skeleton uses, and nothing else (port spec
// §4.2 step 6). modulePlaceholder is a superset of namePlaceholder — it
// contains it as a substring — which is why the substitutions below are
// ordered and must stay ordered.
const (
	modulePlaceholder = "github.com/procrastivity/toolname"
	namePlaceholder   = "toolname"
	envPlaceholder    = "TOOLNAME"
)

// namePattern is the oracle's name-shape check `^[a-z][a-z0-9]*$` (port
// spec §4.2 step 2). The shape is load-bearing beyond taste: the name
// becomes Go package names, and it is what makes the ASCII uppercase
// mapping for the env prefix total (port spec §3.2).
var namePattern = regexp.MustCompile(`^[a-z][a-z0-9]*$`)

// params is the validated parameter set, built once before any filesystem
// mutation begins — the oracle validates everything (port spec §4.2
// steps 2-3) before its first mkdir (§4.2 step 4), and port spec §3.2
// keeps that order.
type params struct {
	name string
	// targetDir is kept exactly as the caller wrote it, never absolutized.
	// The oracle prints this value back in the checklist's first line
	// (port spec §5.2), so `--dir tmp/smoke` must report `tmp/smoke`, not
	// its absolute form.
	targetDir string
	module    string
	upperName string
	doGit     bool
}

// validate reproduces contrib/new-tool.sh's validation order exactly (port
// spec §4.2 step 2): name non-empty, name shape, the placeholder name,
// then — after both defaults are resolved — the existing-target refusal.
// Resolving the defaults before the existence check is what makes a caller
// who relies on the default --dir get the same refusal as one who passed
// it explicitly.
func validate(name, targetDir, module string, noGit bool) (params, error) {
	if name == "" {
		return params{}, toolsmitherr.New("validation.missing-name", "a tool name is required")
	}
	if !namePattern.MatchString(name) {
		return params{}, toolsmitherr.New("validation.invalid-name",
			fmt.Sprintf("name must be lowercase letters and digits, starting with a letter (it becomes Go package names): got %q", name))
	}
	if name == namePlaceholder {
		return params{}, toolsmitherr.New("validation.placeholder-name",
			fmt.Sprintf("%q is the placeholder itself; pick a real name", namePlaceholder))
	}

	if targetDir == "" {
		// The oracle defaults to `<toolsmith-repo>/../<name>`, a sibling
		// of the repository the script lives in. A binary has no
		// repository to be a sibling of, so the port defaults to
		// `<name>` under the current directory instead. Recorded as D5
		// in docs/binary/parity-divergences.md.
		targetDir = name
	}
	if module == "" {
		module = "github.com/procrastivity/" + name
	}

	if _, err := os.Lstat(targetDir); err == nil {
		// Lstat, not Stat: the oracle's guard is `[[ ! -e "$target_dir" ]]`,
		// which is true for a dangling symlink too, and refusing to write
		// through one is the same answer.
		return params{}, toolsmitherr.New("refusal.target-exists",
			fmt.Sprintf("%s already exists; refusing to write into it", targetDir))
	}

	return params{
		name:      name,
		targetDir: targetDir,
		module:    module,
		// the oracle's `tr '[:lower:]' '[:upper:]'` (port spec §4.2
		// step 3). Total for this input because namePattern already
		// confined name to ASCII lowercase and digits (port spec §3.2).
		upperName: strings.ToUpper(name),
		doGit:     !noGit,
	}, nil
}

// instantiate writes the skeleton out as p.name. It follows the oracle's
// operation order (port spec §4.2) with one structural difference: the
// oracle copies the tree and then renames paths and rewrites contents in
// place, while this writes each file once, straight to its final path with
// its final contents. The result is the same tree; there is no intermediate
// state on disk for a reader to see, and no second pass over files that
// were just written.
func instantiate(p params, skeleton fs.FS, embedded bool) error {
	return fs.WalkDir(skeleton, ".", func(rel string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(skeleton, rel)
		if err != nil {
			return internalErr("reading skeleton file %q: %v", rel, err)
		}

		dest := filepath.Join(p.targetDir, filepath.FromSlash(destPath(rel, p.name)))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return internalErr("creating %q: %v", filepath.Dir(dest), err)
		}

		mode, err := destMode(skeleton, rel, data, embedded)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dest, substitute(data, p), mode); err != nil {
			return internalErr("writing %q: %v", dest, err)
		}
		return nil
	})
}

// destPath maps a skeleton-relative path to its path in the instantiated
// tree. It encodes exactly the five renames the oracle performs (port
// spec §4.2 steps 5 and 7), and nothing more.
//
// A general "replace toolname anywhere in the path" rule would produce the
// same answer for today's skeleton, because those are the only paths that
// carry the placeholder. It is not the same rule, though: a skeleton that
// later shipped, say, assets/templates/toolname.md would be renamed by the
// general rule and left alone by the oracle. The explicit table is what the
// oracle does, so the explicit table is what this is.
func destPath(rel, name string) string {
	switch rel {
	case "go.mod.tmpl":
		return "go.mod"
	case "go.sum.tmpl":
		return "go.sum"
	case "internal/toolnameerr/toolnameerr.go":
		return path.Join("internal", name+"err", name+"err.go")
	}
	if rest, ok := strings.CutPrefix(rel, "cmd/toolname/"); ok {
		return path.Join("cmd", name, rest)
	}
	if rest, ok := strings.CutPrefix(rel, "internal/toolnameerr/"); ok {
		return path.Join("internal", name+"err", rest)
	}
	return rel
}

// substitute applies the oracle's three ordered replacements
// (port spec §4.2 step 6).
//
// The order is load-bearing, not cosmetic. The module path contains the
// bare placeholder as a substring, so running the bare replacement first
// would leave the module replacement nothing to match — and for a custom
// --module that strands github.com/procrastivity/<name> in every file
// instead of the module the caller asked for.
//
// Byte-wise, matching the oracle's `LC_ALL=C sed`. Two differences from sed
// are deliberate and both narrow the surface rather than widen it: the
// oracle's pattern is a basic regular expression, so its unescaped dots
// match any byte, and its replacement text gives & and \ their sed
// meanings. Literal bytes are what the substitution is actually for, and no
// module path or skeleton file reaches either corner.
func substitute(data []byte, p params) []byte {
	out := bytes.ReplaceAll(data, []byte(modulePlaceholder), []byte(p.module))
	out = bytes.ReplaceAll(out, []byte(namePlaceholder), []byte(p.name))
	return bytes.ReplaceAll(out, []byte(envPlaceholder), []byte(p.upperName))
}

// destMode decides the written file's permissions.
//
// The oracle uses `cp -R`, which copies the source mode verbatim — so its
// output carries whatever bits the skeleton's working copy happens to have,
// group-write included, which is a fact about the developer's umask rather
// than about the tool. What actually matters to an instantiated tree is the
// executable bit: contrib/check-commit-msg and contrib/check-gofumpt are
// hooks, and a tree that ships them non-executable is broken.
//
// Two disk links of the asset chain carry real modes and are trusted
// directly. The embedded link cannot: embed.FS reports every file as 0444,
// so the bit has to be reconstituted, and a shebang is the signal the
// skeleton's own executables carry. TestSkeletonHooksAreExecutable pins
// that this stays true of the shipped skeleton.
func destMode(skeleton fs.FS, rel string, data []byte, embedded bool) (os.FileMode, error) {
	if !embedded {
		info, err := fs.Stat(skeleton, rel)
		if err != nil {
			return 0, internalErr("stat-ing skeleton file %q: %v", rel, err)
		}
		return info.Mode().Perm(), nil
	}
	if bytes.HasPrefix(data, []byte("#!")) {
		return 0o755, nil
	}
	return 0o644, nil
}

// initGit reproduces the oracle's git-init sequence (port spec §4.2
// step 8). Both of git's streams go to stderr, never to the verb's
// stdout: the oracle lets git inherit the script's stdout, but C2.1 makes
// stdout the verb's own, and -q means a successful run writes nothing to
// either stream anyway. Port spec §5.2 already records that git's
// silence is not hermetically guaranteed.
func initGit(p params, errOut *bytes.Buffer) error {
	steps := [][]string{
		{"init", "-q", "-b", "main"},
		{"add", "-A"},
		{"commit", "-q", "-m", "chore: instantiate " + p.name + " from the toolsmith skeleton"},
	}
	for _, args := range steps {
		cmd := exec.Command("git", append([]string{"-C", p.targetDir}, args...)...)
		cmd.Stdout = errOut
		cmd.Stderr = errOut
		if err := cmd.Run(); err != nil {
			return toolsmitherr.New("new.git-failed",
				fmt.Sprintf("git %s failed in %s: %v", strings.Join(args, " "), p.targetDir, err))
		}
	}
	return nil
}

func internalErr(format string, args ...any) error {
	return toolsmitherr.New("internal.instantiate", fmt.Sprintf(format, args...))
}

// skeletonTree resolves the skeleton subtree through the asset chain and
// reports whether the embedded fallback answered. The oracle's equivalent
// is a single `[[ -d "$repo/assets/_skeleton" ]]` against the repository it
// lives in (port spec §2.2); a binary has no repository, so the chain is
// what stands in for it (C5.1, D6 in docs/binary/parity-divergences.md).
//
// The error is internal.*, not not-found.* (C2.5). asset.Tree returns an
// override or shipped-default directory whenever one exists and falls
// through when it does not, so no user action reaches this branch: it
// fires only when the embedded fallback lacks _skeleton, which a correct
// build cannot produce.
func skeletonTree() (fs.FS, bool, error) {
	tree, source, err := asset.Tree(skeletonPrefix)
	if err != nil {
		return nil, false, toolsmitherr.New("internal.skeleton-missing", "skeleton not found: "+err.Error())
	}
	return tree, source == asset.SourceEmbedded, nil
}
