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

// Placeholder spellings the skeleton uses (port spec §4.2 step 6).
// Longer spellings must precede their substrings in substitute's replacer.
const (
	modulePlaceholder = "github.com/procrastivity/toolname"
	namePlaceholder   = "toolname"
	envPlaceholder    = "TOOLNAME"
)

// Hyphens separate public-name segments; derived Go identifiers omit them,
// while environment prefixes replace them with underscores.
var namePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

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
	goName    string
	upperName string
	doGit     bool
	inPlace   bool
}

// validateInPlace uses the explicit name or derives it from the directory. A
// directory with a Go module is already a tool, not a mostly-bare project.
// No git commands run in this mode: existing history and staged work belong
// to the caller, and initialising a new repo would commit their planning files.
func validateInPlace(name, module string) (params, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return params{}, internalErr("finding current directory: %v", err)
	}
	if name == "" {
		name = filepath.Base(cwd)
	}
	if err := validateName(name); err != nil {
		return params{}, err
	}
	if _, err := os.Lstat("go.mod"); err == nil {
		return params{}, toolsmitherr.New("refusal.target-exists", "go.mod already exists; this is not a bare project")
	} else if !os.IsNotExist(err) {
		return params{}, internalErr("checking go.mod: %v", err)
	}
	if module == "" {
		module = "github.com/procrastivity/" + name
	}
	return params{name: name, targetDir: ".", module: module, goName: strings.ReplaceAll(name, "-", ""), upperName: strings.ToUpper(strings.ReplaceAll(name, "-", "_")), inPlace: true}, nil
}

// Only these non-code files are safe to retain instead of the skeleton's
// versions. All other collisions fail before any file is written.
func preserveInPlace(rel string) bool {
	switch rel {
	case "README.md", ".gitignore", "LICENSE":
		return true
	}
	return false
}

func preflightInPlace(p params, skeleton fs.FS) error {
	return fs.WalkDir(skeleton, ".", func(rel string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		dest := filepath.Join(p.targetDir, filepath.FromSlash(destPath(rel, p.name, p.goName)))
		info, err := os.Lstat(dest)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return internalErr("checking %q: %v", dest, err)
		}
		if d.IsDir() && info.IsDir() {
			return nil
		}
		if !d.IsDir() && preserveInPlace(rel) && info.Mode().IsRegular() {
			return nil
		}
		return toolsmitherr.New("refusal.target-exists", fmt.Sprintf("%s already exists; refusing to overwrite a skeleton path", dest))
	})
}

// validate reproduces contrib/new-tool.sh's validation order exactly (port
// spec §4.2 step 2): name non-empty, name shape, the placeholder name,
// then — after both defaults are resolved — the existing-target refusal.
// Resolving the defaults before the existence check is what makes a caller
// who relies on the default --dir get the same refusal as one who passed
// it explicitly.
func validate(name, targetDir, module string, noGit bool) (params, error) {
	if err := validateName(name); err != nil {
		return params{}, err
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
		goName:    strings.ReplaceAll(name, "-", ""),
		upperName: strings.ToUpper(strings.ReplaceAll(name, "-", "_")),
		doGit:     !noGit,
	}, nil
}

func validateName(name string) error {
	if name == "" {
		return toolsmitherr.New("validation.missing-name", "a tool name is required")
	}
	if !namePattern.MatchString(name) {
		return toolsmitherr.New("validation.invalid-name",
			fmt.Sprintf("name must be lowercase ASCII letters and digits in hyphen-separated segments, starting with a letter: got %q", name))
	}
	if name == namePlaceholder {
		return toolsmitherr.New("validation.placeholder-name",
			fmt.Sprintf("%q is the placeholder itself; pick a real name", namePlaceholder))
	}
	return nil
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

		dest := filepath.Join(p.targetDir, filepath.FromSlash(destPath(rel, p.name, p.goName)))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return internalErr("creating %q: %v", filepath.Dir(dest), err)
		}

		mode, err := destMode(skeleton, rel, data, embedded)
		if err != nil {
			return err
		}
		if p.inPlace {
			if preserveInPlace(rel) {
				if info, err := os.Lstat(dest); err == nil {
					if info.Mode().IsRegular() {
						return nil
					}
					return toolsmitherr.New("refusal.target-exists", fmt.Sprintf("%s already exists; refusing to overwrite a skeleton path", dest))
				} else if !os.IsNotExist(err) {
					return internalErr("checking %q: %v", dest, err)
				}
			}
			file, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
			if err != nil {
				return internalErr("creating %q: %v", dest, err)
			}
			if _, err := file.Write(substitute(data, p)); err != nil {
				_ = file.Close()
				return internalErr("writing %q: %v", dest, err)
			}
			if err := file.Close(); err != nil {
				return internalErr("closing %q: %v", dest, err)
			}
		} else if err := os.WriteFile(dest, substitute(data, p), mode); err != nil {
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
func destPath(rel, name, goName string) string {
	switch rel {
	case "go.mod.tmpl":
		return "go.mod"
	case "go.sum.tmpl":
		return "go.sum"
	case "internal/toolnameerr/toolnameerr.go":
		return path.Join("internal", goName+"err", goName+"err.go")
	}
	if rest, ok := strings.CutPrefix(rel, "cmd/toolname/"); ok {
		return path.Join("cmd", name, rest)
	}
	if rest, ok := strings.CutPrefix(rel, "internal/toolnameerr/"); ok {
		return path.Join("internal", goName+"err", rest)
	}
	return rel
}

// substitute replaces the longest placeholders first, in one pass. A name
// such as toolname-err may itself contain a placeholder spelling; a series
// of ReplaceAll calls would corrupt the newly inserted identifier. Likewise
// a custom module path must not be rewritten after insertion.
func substitute(data []byte, p params) []byte {
	return []byte(strings.NewReplacer(
		modulePlaceholder, p.module,
		"toolnameerr", p.goName+"err",
		"toolpkg", p.goName,
		namePlaceholder, p.name,
		envPlaceholder, p.upperName,
	).Replace(string(data)))
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

// lookPath is a test seam over exec.LookPath (C2.6).
var lookPath = exec.LookPath

// initGitRepo creates the target, runs `git init` in it, and checks that
// git can resolve both an author and a committer identity there, before
// any skeleton file is written. It exists so that a missing identity fails
// up front, not at the commit after the whole tree is on disk
// (evidence/2026-09-11-toolsmith-conformance.md §7).
//
// The identity check runs inside the new repository (`git -C`). Rejected:
// checking from the process's working directory before `git init`. That
// directory's local config can supply an identity the new repository will
// not see (a false pass), and an `includeIf "gitdir:..."` identity matches
// only inside the new repository (a false refusal). Also rejected: checking
// GIT_COMMITTER_IDENT alone, which passes when only the committer is set
// while `git commit` fails on the author.
//
// On a failure, initGitRepo removes the target. That is safe because
// validate refused an existing target, so this call created it.
func initGitRepo(p params, errOut *bytes.Buffer) error {
	if _, err := lookPath("git"); err != nil {
		return toolsmitherr.New("not-found.git",
			"git is not on PATH; install it, or pass --no-git")
	}

	if err := os.MkdirAll(p.targetDir, 0o755); err != nil {
		return internalErr("creating %q: %v", p.targetDir, err)
	}

	initCmd := exec.Command("git", "-C", p.targetDir, "init", "-q", "-b", "main")
	initCmd.Stdout = errOut
	initCmd.Stderr = errOut
	if err := initCmd.Run(); err != nil {
		_ = os.RemoveAll(p.targetDir)
		return toolsmitherr.New("internal.git-failed",
			fmt.Sprintf("git init failed in %s: %v", p.targetDir, err))
	}

	for _, v := range []string{"GIT_AUTHOR_IDENT", "GIT_COMMITTER_IDENT"} {
		var stderr bytes.Buffer
		cmd := exec.Command("git", "-C", p.targetDir, "var", v)
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			_ = os.RemoveAll(p.targetDir)
			return toolsmitherr.New("not-found.git-identity",
				fmt.Sprintf("git has no commit identity for %s (%s); set user.name and user.email, or pass --no-git",
					p.targetDir, lastLine(stderr.String())))
		}
	}
	return nil
}

// lastLine returns the last non-blank line of git's stderr. git explains an
// identity failure over several lines and ends with the specific cause, and
// the error message must stay one line (C2.5's human line).
func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// commitGit makes the first commit in the repository initGitRepo created.
// Both of git's streams go to stderr, never to the verb's stdout: the
// oracle (port spec §4.2 step 8) lets git inherit the script's stdout, but
// C2.1 makes stdout the verb's own, and -q means a successful run writes
// nothing to either stream anyway. Port spec §5.2 already records that
// git's silence is not hermetically guaranteed.
//
// A failure here is internal.git-failed (exit 4): initGitRepo already
// proved that git runs and has an identity in this repository, so the
// failure is unexpected (C2.5).
func commitGit(p params, errOut *bytes.Buffer) error {
	steps := [][]string{
		{"add", "-A"},
		{"commit", "-q", "-m", "chore: instantiate " + p.name + " from the toolsmith skeleton"},
	}
	for _, args := range steps {
		cmd := exec.Command("git", append([]string{"-C", p.targetDir}, args...)...)
		cmd.Stdout = errOut
		cmd.Stderr = errOut
		if err := cmd.Run(); err != nil {
			return toolsmitherr.New("internal.git-failed",
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
