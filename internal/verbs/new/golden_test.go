// Package new_test's golden suite pins `toolsmith new`'s produced tree and
// checklist stdout (port spec §5.2) across three instantiation cases. Each
// case is proven against a model tree built from the shipped skeleton
// (port spec §4.2 steps 5-6) rather than against a byte snapshot, so a
// deliberate skeleton edit does not require regenerating a fixture.
//
// The corpus is frozen: testdata/golden/new/ holds the expected checklist
// bytes, captured as described in evidence/2026-09-10-parity-goldens.md.
// Regenerate this package's and internal/verbs/check's goldens together:
//
//	go test ./internal/verbs/check ./internal/verbs/new -count=1 -update
//
// `-update` is a flag this package's own TestMain defines; the unscoped
// `go test ./... -update` fails in every other package, which does not
// define it. A diff `-update` produces is a behavior change in the port,
// not routine maintenance, and must be reviewed as one.
package new_test

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	// The go test cache can't see into TestMain's own `go build` of the
	// toolsmith binary, so a change to a verb or to the embedded skeleton
	// would otherwise leave a stale binary uncaught. assets, because
	// expectedTree below reads assets/_skeleton straight off disk, which
	// is a different link of the same asset chain instantiate.go's
	// embedded fallback answers from — both must move together.
	_ "github.com/procrastivity/toolsmith/assets"
	_ "github.com/procrastivity/toolsmith/internal/cli"
)

var updateGolden = flag.Bool("update", false, "write testdata/golden/new/*.stdout from the current run instead of comparing to it")

var binPath string

func TestMain(m *testing.M) {
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Fprintln(os.Stderr, "golden: git not found on PATH:", err)
		os.Exit(1)
	}

	tmp, err := os.MkdirTemp("", "toolsmith-new-golden")
	if err != nil {
		fmt.Fprintln(os.Stderr, "golden:", err)
		os.Exit(1)
	}

	binPath = filepath.Join(tmp, "bin", "toolsmith")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/toolsmith")
	build.Dir = repoRootDir()
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "golden: building toolsmith: %v\n%s", err, out)
		_ = os.RemoveAll(tmp)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}

// repoRootDir resolves the toolsmith repo root from this file's own path
// (internal/verbs/new/golden_test.go, three levels below the root), the
// same precedent as drift/drift_test.go's repoRoot.
func repoRootDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("golden: runtime.Caller could not resolve this file's own path")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
}

// hermeticEnv builds the subprocess environment every golden-case
// subprocess in this package uses: the process environment with every seam
// variable stripped, then rebuilt so git never reads this host's config or
// climbs out of the temp tree, and git identity is fixed rather than
// inherited.
func hermeticEnv(t *testing.T, overrides map[string]string) []string {
	t.Helper()
	var out []string
	for _, e := range os.Environ() {
		key, _, _ := strings.Cut(e, "=")
		switch {
		case strings.HasPrefix(key, "GIT_"),
			strings.HasPrefix(key, "TOOLSMITH_"),
			key == "XDG_CONFIG_HOME",
			key == "CGO_ENABLED",
			key == "LC_ALL":
			continue
		}
		out = append(out, e)
	}
	base := map[string]string{
		"XDG_CONFIG_HOME":         t.TempDir(),
		"LC_ALL":                  "C",
		"CGO_ENABLED":             "1",
		"GIT_CONFIG_GLOBAL":       os.DevNull,
		"GIT_CONFIG_NOSYSTEM":     "1",
		"GIT_AUTHOR_NAME":         "golden",
		"GIT_AUTHOR_EMAIL":        "golden@example.invalid",
		"GIT_COMMITTER_NAME":      "golden",
		"GIT_COMMITTER_EMAIL":     "golden@example.invalid",
		"GIT_CEILING_DIRECTORIES": os.TempDir(),
		// Without maintenance.auto=false and gc.autoDetach=false, a
		// detached `git maintenance run --auto` can still be writing to
		// .git/ after `git commit` returns, racing t.TempDir()'s cleanup
		// on the with-git case. See evidence/2026-09-10-parity-goldens.md
		// §10 for the isolation trials behind this.
		"GIT_CONFIG_COUNT":   "5",
		"GIT_CONFIG_KEY_0":   "core.fsmonitor",
		"GIT_CONFIG_VALUE_0": "false",
		"GIT_CONFIG_KEY_1":   "core.untrackedCache",
		"GIT_CONFIG_VALUE_1": "false",
		"GIT_CONFIG_KEY_2":   "maintenance.auto",
		"GIT_CONFIG_VALUE_2": "false",
		"GIT_CONFIG_KEY_3":   "gc.auto",
		"GIT_CONFIG_VALUE_3": "0",
		"GIT_CONFIG_KEY_4":   "gc.autoDetach",
		"GIT_CONFIG_VALUE_4": "false",
	}
	for k, v := range overrides {
		base[k] = v
	}
	for k, v := range base {
		out = append(out, k+"="+v)
	}
	return out
}

func runBin(t *testing.T, dir string, env []string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir
	cmd.Env = env
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("running %s %v: %v", binPath, args, err)
		}
	}
	return out.String(), errBuf.String(), exitCode
}

func blob(stdout string, exit int) string {
	return stdout + fmt.Sprintf("exit=%d\n", exit)
}

func goldenPath(t *testing.T, rel string) string {
	t.Helper()
	return filepath.Join(repoRootDir(), "internal", "verbs", "new", "testdata", "golden", rel)
}

func goldenCompare(t *testing.T, rel, got string) {
	t.Helper()
	path := goldenPath(t, rel)
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden %s: %v", path, err)
	}
	if string(want) != got {
		t.Errorf("%s mismatch:\n--- want (golden) ---\n%s\n--- got ---\n%s", rel, want, got)
	}
}

// treeEntry is one file of an instantiated tree, as either loadTree or
// expectedTree sees it: content bytes and whether the user-executable bit
// is set. The tree comparison asserts the executable bit, never the full
// mode (port spec §9.2) — an embedded asset link cannot carry more than
// that bit in any case (internal/verbs/new/instantiate.go's destMode).
type treeEntry struct {
	data []byte
	exec bool
}

// loadTree reads every regular file under dir into a map keyed by its
// slash-separated path relative to dir. .git is excluded: two independent
// `git init` runs never produce identical object stores, so the git facts
// are asserted separately, by name.
func loadTree(t *testing.T, dir string) map[string]treeEntry {
	t.Helper()
	out := map[string]treeEntry{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(dir, p)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if rel == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 || !d.Type().IsRegular() {
			t.Fatalf("loadTree: %s is not a regular file (mode %v)", p, d.Type())
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = treeEntry{data: data, exec: info.Mode().Perm()&0o111 != 0}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", dir, err)
	}
	return out
}

// destPathModel and substituteModel are a deliberately separate
// reimplementation of instantiate.go's destPath/substitute (port spec
// §4.2 steps 5-6): expectedTree below is a model built from first
// principles against the shipped skeleton on disk, not a snapshot, so it
// stays a real check on the production code rather than restating it.
func destPathModel(rel, name string) string {
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

func substituteModel(data []byte, module, name, upper string) []byte {
	out := bytes.ReplaceAll(data, []byte("github.com/procrastivity/toolname"), []byte(module))
	out = bytes.ReplaceAll(out, []byte("toolname"), []byte(name))
	return bytes.ReplaceAll(out, []byte("TOOLNAME"), []byte(upper))
}

// expectedTree builds the model tree instantiating name (module module)
// from skeletonDir ought to produce. The executable bit is read straight
// off skeletonDir's own files, not reconstituted from a shebang, which is
// what makes TestSkeletonHooksAreExecutable (instantiate_test.go)
// load-bearing for this model too.
func expectedTree(t *testing.T, skeletonDir, name, module string) map[string]treeEntry {
	t.Helper()
	upper := strings.ToUpper(name)
	out := map[string]treeEntry{}
	err := filepath.WalkDir(skeletonDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(skeletonDir, p)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		dest := destPathModel(rel, name)
		out[dest] = treeEntry{
			data: substituteModel(data, module, name, upper),
			exec: info.Mode().Perm()&0o111 != 0,
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking skeleton %s: %v", skeletonDir, err)
	}
	return out
}

// compareTrees reports every difference between want and got exhaustively
// — every only-in-want path, every only-in-got path, every content
// mismatch, every executable-bit mismatch — then fails once, rather than
// stopping at the first mismatch found.
func compareTrees(t *testing.T, want, got map[string]treeEntry) {
	t.Helper()
	var onlyWant, onlyGot, contentDiff, execDiff []string
	for p := range want {
		if _, ok := got[p]; !ok {
			onlyWant = append(onlyWant, p)
		}
	}
	for p := range got {
		if _, ok := want[p]; !ok {
			onlyGot = append(onlyGot, p)
		}
	}
	for p, w := range want {
		g, ok := got[p]
		if !ok {
			continue
		}
		if !bytes.Equal(w.data, g.data) {
			contentDiff = append(contentDiff, p)
		}
		if w.exec != g.exec {
			execDiff = append(execDiff, fmt.Sprintf("%s (want exec=%v got exec=%v)", p, w.exec, g.exec))
		}
	}
	sort.Strings(onlyWant)
	sort.Strings(onlyGot)
	sort.Strings(contentDiff)
	sort.Strings(execDiff)
	if len(onlyWant)+len(onlyGot)+len(contentDiff)+len(execDiff) > 0 {
		t.Errorf("tree mismatch:\n  only in model: %v\n  only in produced tree: %v\n  content differs: %v\n  executable bit differs: %v",
			onlyWant, onlyGot, contentDiff, execDiff)
	}
}

func gitOutput(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = env
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out.String())
	}
	return out.String()
}

// assertGitFacts checks what the tree comparison deliberately excludes:
// the branch, the commit count and subject, and that nothing is left
// uncommitted. Two independent `git init` runs never produce the same
// object store, which is why the tree comparison excludes .git — it is
// not a reason to skip checking what the three git calls are for.
func assertGitFacts(t *testing.T, dir string, env []string, name string) {
	t.Helper()
	branch := strings.TrimSpace(gitOutput(t, dir, env, "rev-parse", "--abbrev-ref", "HEAD"))
	if branch != "main" {
		t.Errorf("branch = %q, want main", branch)
	}
	count := strings.TrimSpace(gitOutput(t, dir, env, "rev-list", "--count", "HEAD"))
	if count != "1" {
		t.Errorf("commit count = %q, want 1", count)
	}
	subject := strings.TrimSpace(gitOutput(t, dir, env, "log", "-1", "--format=%s"))
	want := "chore: instantiate " + name + " from the toolsmith skeleton"
	if subject != want {
		t.Errorf("commit subject = %q, want %q", subject, want)
	}
	status := gitOutput(t, dir, env, "status", "--porcelain", "--untracked-files=all", "--ignored")
	if status != "" {
		t.Errorf("git status --porcelain --untracked-files=all --ignored not empty:\n%s", status)
	}
}

// assertAnchors checks two invariants port spec §4.2 steps 5-6 make
// skeleton-independent: go.mod's first line, and that no placeholder
// spelling survives anywhere in a path or in a file's contents.
func assertAnchors(t *testing.T, tree map[string]treeEntry, module string) {
	t.Helper()
	goMod, ok := tree["go.mod"]
	if !ok {
		t.Fatalf("go.mod missing from the produced tree")
	}
	firstLine, _, _ := strings.Cut(string(goMod.data), "\n")
	wantFirst := "module " + module
	if firstLine != wantFirst {
		t.Errorf("go.mod line 1 = %q, want %q", firstLine, wantFirst)
	}
	for p, e := range tree {
		if strings.Contains(p, "toolname") || strings.Contains(p, "TOOLNAME") {
			t.Errorf("placeholder survives in path %s", p)
		}
		if bytes.Contains(e.data, []byte("toolname")) || bytes.Contains(e.data, []byte("TOOLNAME")) {
			t.Errorf("placeholder survives in the contents of %s", p)
		}
	}
}

// newCase is one instantiation case: the positional name, the full
// argument list (--dir always explicit — the oracle's own default is a
// sibling of its own repository, port spec §8.2, which a binary has no
// equivalent of), and the module and doGit facts the test asserts
// against.
type newCase struct {
	name   string
	tool   string
	args   []string
	module string
	doGit  bool
}

var newCases = []newCase{
	{
		name:   "default",
		tool:   "demo",
		args:   []string{"demo", "--dir", "demo", "--no-git"},
		module: "github.com/procrastivity/demo",
		doGit:  false,
	},
	{
		// The case the oracle's own comment calls load-bearing: the only
		// one where getting substitute()'s ordering wrong (bare
		// "toolname" before the module path) produces a different tree
		// (port spec §4.2 step 6).
		name:   "custom-module",
		tool:   "acme",
		args:   []string{"acme", "--dir", "acme", "--module", "example.com/x/acme", "--no-git"},
		module: "example.com/x/acme",
		doGit:  false,
	},
	{
		name:   "with-git",
		tool:   "gitdemo",
		args:   []string{"gitdemo", "--dir", "gitdemo"},
		module: "github.com/procrastivity/gitdemo",
		doGit:  true,
	},
}

// TestGoldenNew instantiates each case three times, into three fresh
// parent directories, and requires all three to agree — on the checklist
// bytes, and on the produced tree — before comparing the first run to the
// model tree and to (or, under -update, writing) the checklist golden.
func TestGoldenNew(t *testing.T) {
	skeletonDir := filepath.Join(repoRootDir(), "assets", "_skeleton")

	for _, tc := range newCases {
		t.Run(tc.name, func(t *testing.T) {
			var blobs []string
			var trees []map[string]treeEntry
			var envs [][]string
			var dirs []string

			for i := range 3 {
				parent := t.TempDir()
				env := hermeticEnv(t, nil)
				stdout, stderr, exit := runBin(t, parent, env, append([]string{"new"}, tc.args...)...)
				if exit != 0 {
					t.Fatalf("run %d: exit=%d stdout=%q stderr=%q", i, exit, stdout, stderr)
				}
				if stderr != "" {
					t.Fatalf("run %d: stderr=%q, want empty on a clean run (port spec §5.2)", i, stderr)
				}
				blobs = append(blobs, blob(stdout, exit))
				target := filepath.Join(parent, tc.tool)
				trees = append(trees, loadTree(t, target))
				envs = append(envs, env)
				dirs = append(dirs, target)
			}

			for i := 1; i < 3; i++ {
				if blobs[i] != blobs[0] {
					t.Errorf("determinism: run %d's checklist differs from run 0", i)
				}
				compareTrees(t, trees[0], trees[i])
			}

			goldenCompare(t, filepath.Join("new", tc.name+".stdout"), blobs[0])
			compareTrees(t, expectedTree(t, skeletonDir, tc.tool, tc.module), trees[0])
			assertAnchors(t, trees[0], tc.module)

			if tc.name == "custom-module" {
				for p, e := range trees[0] {
					if bytes.Contains(e.data, []byte("github.com/procrastivity/acme")) {
						t.Errorf("wrong substitution order: %s still contains the stranded default module path", p)
					}
				}
			}

			if tc.doGit {
				assertGitFacts(t, dirs[0], envs[0], tc.tool)
			}
		})
	}
}

// TestGoldenNewCLI covers the failure shapes newCases's tree matrix cannot
// reach: bad flags, a bad name shape, the placeholder name, a missing
// name, and an existing target, each pinned against the exit code table
// CONTRACT C2.4 defines (port spec §9.3).
func TestGoldenNewCLI(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"unknown flag", []string{"--bogus", "acme"}, 2},
		{"bad name shape", []string{"Acme"}, 1},
		{"the placeholder name", []string{"toolname"}, 1},
		{"no name", nil, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			env := hermeticEnv(t, nil)
			stdout, _, exit := runBin(t, dir, env, append([]string{"new"}, c.args...)...)
			if exit != c.want {
				t.Errorf("exit = %d, want %d", exit, c.want)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
		})
	}
	t.Run("existing target", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "taken"), 0o755); err != nil {
			t.Fatal(err)
		}
		env := hermeticEnv(t, nil)
		stdout, _, exit := runBin(t, dir, env, "new", "acme", "--dir", "taken")
		if exit != 3 {
			t.Errorf("exit = %d, want 3 (exitcode.Refusal)", exit)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
	})
}

// TestGoldenNewStaleFiles requires every golden file under
// testdata/golden/new/ to name a live case.
func TestGoldenNewStaleFiles(t *testing.T) {
	names := map[string]bool{}
	for _, tc := range newCases {
		names[tc.name] = true
	}
	dir := goldenPath(t, "new")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".stdout")
		if !names[base] {
			t.Errorf("stale golden file %s: no case named %q in newCases", e.Name(), base)
		}
	}
	var sorted []string
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)
	for _, n := range sorted {
		if _, err := os.Stat(filepath.Join(dir, n+".stdout")); err != nil {
			t.Errorf("case %q has no golden %s.stdout", n, n)
		}
	}
}
