// Package check_test's golden suite pins `toolsmith check`'s stdout and
// stderr (port spec §5.1) across a frozen corpus of real repos and a set
// of generated probes, one per condition the corpus alone does not reach
// (port spec §10). It also pins the clean-run stderr rule: no stderr
// except the multiple-cmd/-entries note (§10). A fake `go` on PATH
// answers every case's manifest request, so the CGO_ENABLED override
// (port spec §6) is checked without a real toolchain or any buildable Go
// source.
//
// The corpus is frozen: testdata/corpus/ and testdata/golden/check/ hold
// fixed inputs and expected outputs, captured as described in
// evidence/2026-09-10-parity-goldens.md. Regenerate this package's and
// internal/verbs/new's goldens together:
//
//	go test ./internal/verbs/check ./internal/verbs/new -count=1 -update
//
// `-update` is a flag this package's own TestMain defines; the unscoped
// `go test ./... -update` fails in every other package, which does not
// define it. A diff `-update` produces is a behavior change in the port,
// not routine maintenance, and must be reviewed as one.
package check_test

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	// The go test cache can't see into TestMain's own `go build` of the
	// toolsmith binary, so a change to a verb or to the embedded skeleton
	// would otherwise leave a stale binary uncaught. Blank-importing both
	// makes the cache key depend on them. internal/cli because
	// skeleton-smoke below drives `toolsmith new`, which internal/cli
	// wires onto the root command; assets because that `new` invocation's
	// output is the embedded skeleton.
	_ "github.com/procrastivity/toolsmith/assets"
	_ "github.com/procrastivity/toolsmith/internal/cli"
)

var updateGolden = flag.Bool("update", false, "write testdata/golden/check/*.stdout|stderr from the current run instead of comparing to it")

var (
	binPath   string
	fakeGoDir string
)

func TestMain(m *testing.M) {
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Fprintln(os.Stderr, "golden: git not found on PATH:", err)
		os.Exit(1)
	}

	tmp, err := os.MkdirTemp("", "toolsmith-check-golden")
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

	fakeGoDir = filepath.Join(tmp, "fakebin")
	if err := os.MkdirAll(fakeGoDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "golden:", err)
		os.Exit(1)
	}
	// The fake `go` every case's manifest-verb step is pointed at
	// (port spec §6). It logs its own invocation — proving the audit's
	// own CGO_ENABLED=0 override, not the test env's — and answers with
	// a canned document rather than compiling anything, so the suite
	// needs no buildable Go source in any corpus repo or probe.
	const fakeGoScript = "#!/bin/sh\n" +
		"printf '%s CGO_ENABLED=%s\\n' \"$*\" \"${CGO_ENABLED:-unset}\" >>\"$FAKE_GO_LOG\"\n" +
		"[ -n \"${FAKE_GO_STDOUT:-}\" ] || exit 99\n" +
		"cat \"$FAKE_GO_STDOUT\"\n" +
		"exit \"${FAKE_GO_EXIT:-0}\"\n"
	if err := os.WriteFile(filepath.Join(fakeGoDir, "go"), []byte(fakeGoScript), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "golden:", err)
		os.Exit(1)
	}

	code := m.Run()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}

// repoRootDir resolves the toolsmith repo root from this file's own path
// (internal/verbs/check/golden_test.go, three levels below the root),
// the same precedent as drift/drift_test.go's repoRoot.
func repoRootDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("golden: runtime.Caller could not resolve this file's own path")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
}

// hermeticEnv builds the subprocess environment every golden-case
// subprocess in this package uses: the process environment with every seam
// variable stripped, then rebuilt so PATH finds the fake `go` first, git
// never reads this host's config or climbs out of the temp tree, and git
// identity is fixed rather than inherited.
func hermeticEnv(t *testing.T, overrides map[string]string) []string {
	t.Helper()
	var out []string
	for _, e := range os.Environ() {
		key, _, _ := strings.Cut(e, "=")
		switch {
		case strings.HasPrefix(key, "GIT_"),
			strings.HasPrefix(key, "TOOLSMITH_"),
			strings.HasPrefix(key, "FAKE_GO_"),
			key == "XDG_CONFIG_HOME",
			key == "CGO_ENABLED",
			key == "LC_ALL",
			key == "PATH":
			continue
		}
		out = append(out, e)
	}
	base := map[string]string{
		"PATH":                    fakeGoDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"XDG_CONFIG_HOME":         t.TempDir(),
		"LC_ALL":                  "C",
		"CGO_ENABLED":             "1", // deliberately ambient-wrong; audit.go must override it to 0 itself.
		"GIT_CONFIG_GLOBAL":       os.DevNull,
		"GIT_CONFIG_NOSYSTEM":     "1",
		"GIT_AUTHOR_NAME":         "golden",
		"GIT_AUTHOR_EMAIL":        "golden@example.invalid",
		"GIT_COMMITTER_NAME":      "golden",
		"GIT_COMMITTER_EMAIL":     "golden@example.invalid",
		"GIT_CEILING_DIRECTORIES": os.TempDir(),
		// This package's cases never commit, so the git-maintenance race
		// internal/verbs/new's hermeticEnv guards against has not shown
		// up here. Carried for consistency, and against a future case
		// that does commit. See evidence/2026-09-10-parity-goldens.md
		// §10.
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

// runBin runs the built toolsmith binary with dir as its working
// directory and env as its full environment.
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

// blob is stdout with the exit code appended as its own trailing line, so
// one comparison covers either mismatch.
func blob(stdout string, exit int) string {
	return stdout + fmt.Sprintf("exit=%d\n", exit)
}

// normalizeRepo replaces every occurrence of dir (and, if it differs, its
// symlink-resolved form — t.TempDir() on some platforms returns a
// symlinked path) with the literal token $REPO, so a golden file survives
// being captured on a different host or in a different temp root.
func normalizeRepo(t *testing.T, s, dir string) string {
	t.Helper()
	out := strings.ReplaceAll(s, dir, "$REPO")
	if resolved, err := filepath.EvalSymlinks(dir); err == nil && resolved != dir {
		out = strings.ReplaceAll(out, resolved, "$REPO")
	}
	return out
}

func goldenPath(t *testing.T, rel string) string {
	t.Helper()
	return filepath.Join(repoRootDir(), "internal", "verbs", "check", "testdata", "golden", rel)
}

// goldenCompare compares got against testdata/golden/<rel>, or writes it
// there under -update.
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

// copyFixture materializes a corpus/probe fixture tree at dst, restoring
// any path segment stored as "dot-X" back to ".X" — the storage trick
// that keeps a checked-in ".envrc" or ".github/" out of this repo's own
// hooks. This repo's own .pre-commit-config.yaml selects shellcheck's
// target files by `types: [shell]`, and a bare .envrc counts as shell; the
// pinned wip fixture's own .envrc carries no shellcheck directive and
// would fail that hook if committed as itself.
func copyFixture(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		destRel := restoreDots(rel)
		destPath := filepath.Join(dst, destRel)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copying fixture %s -> %s: %v", src, dst, err)
	}
}

func restoreDots(rel string) string {
	parts := strings.Split(rel, "/")
	for i, p := range parts {
		if strings.HasPrefix(p, "dot-") {
			parts[i] = "." + strings.TrimPrefix(p, "dot-")
		}
	}
	return strings.Join(parts, "/")
}

func readCorpusFile(t *testing.T, name, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRootDir(), "internal", "verbs", "check", "testdata", "corpus", name, rel))
	if err != nil {
		t.Fatalf("reading corpus fixture %s/%s: %v", name, rel, err)
	}
	return string(data)
}

// checkCase is one parity case: a repo fixture, the tool name Audit
// should discover in it (empty if none — the manifest-verb subprocess
// must never run), and the canned manifest response the fake `go`
// answers with when it does.
type checkCase struct {
	name           string
	setup          func(t *testing.T, dir string)
	targetRel      string // "" means dir itself is the audited repo
	tool           string
	manifestStdout string
	manifestExit   int
	invariant      func(t *testing.T, stdout string)
}

// checkCases is the case table: the four pinned corpus repos, one live
// instantiation, and a generated probe for each port spec condition the
// corpus repos never reach on their own. Adding a probe without a case
// here is a gap this suite cannot see — TestGoldenCheckStaleFiles only
// catches the opposite drift.
func checkCases(t *testing.T) []checkCase {
	t.Helper()
	var cases []checkCase

	for _, name := range []string{"wip", "duo", "ste9", "toolsmith"} {
		exitStr := strings.TrimSpace(readCorpusFile(t, name, "manifest.exit"))
		exit, err := strconv.Atoi(exitStr)
		if err != nil {
			t.Fatalf("corpus %s manifest.exit = %q: %v", name, exitStr, err)
		}
		cases = append(cases, checkCase{
			name: "corpus-" + name,
			setup: func(t *testing.T, dir string) {
				copyFixture(t, filepath.Join(repoRootDir(), "internal", "verbs", "check", "testdata", "corpus", name, "repo"), dir)
			},
			tool:           name,
			manifestStdout: readCorpusFile(t, name, "manifest.stdout"),
			manifestExit:   exit,
		})
	}

	cases = append(cases, checkCase{
		name: "skeleton-smoke",
		setup: func(t *testing.T, dir string) {
			env := hermeticEnv(t, nil)
			stdout, stderr, exit := runBin(t, dir, env, "new", "smoke", "--dir", "smoke", "--no-git")
			if exit != 0 {
				t.Fatalf("instantiating skeleton-smoke: exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
			}
		},
		targetRel:      "smoke",
		tool:           "smoke",
		manifestStdout: manifestJSONLiteral,
		manifestExit:   0,
	})

	cases = append(cases, checkCase{
		name:  "probe-c62-mislabel",
		setup: generateC62MislabelProbe,
	})
	cases = append(cases, checkCase{
		name:  "probe-c62-hole",
		setup: generateC62HoleProbe,
	})
	cases = append(cases, checkCase{
		name:  "probe-changelog-tracked",
		setup: generateChangelogTrackedProbe,
	})
	cases = append(cases, checkCase{
		name:           "probe-multicmd",
		setup:          generateMulticmdProbe,
		tool:           "aaa",
		manifestStdout: manifestJSONLiteral,
		manifestExit:   0,
	})
	cases = append(cases, checkCase{
		name:  "probe-crlf",
		setup: generateCRLFProbe,
		invariant: func(t *testing.T, stdout string) {
			t.Helper()
			if strings.Contains(stdout, "action not SHA-pinned") {
				t.Errorf("port spec §9.7: a correctly SHA-pinned CRLF action was reported as unpinned:\n%s", stdout)
			}
		},
	})
	cases = append(cases, checkCase{
		name:           "probe-prettyjson",
		setup:          generatePrettyjsonProbe,
		tool:           "prettytool",
		manifestStdout: prettyManifestJSON,
		manifestExit:   0,
		invariant: func(t *testing.T, stdout string) {
			t.Helper()
			for _, clause := range []string{"C3.1:", "C3.4:", "C3.6:"} {
				if strings.Contains(stdout, clause) {
					t.Errorf("port spec §9.1: a pretty-printed manifest tripped a false %s finding:\n%s", clause, stdout)
				}
			}
		},
	})

	return cases
}

// manifestJSONLiteral is a compact, contract-conformant manifest document:
// schemaVersion, contract, and manifest_digest all present with the exact
// prefixes the field checks require (port spec §9.1).
const manifestJSONLiteral = `{"schemaVersion":1,"contract":"toolsmith/v1","manifest_digest":"sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcd"}`

// prettyManifestJSON carries the same fields as manifestJSONLiteral,
// MarshalIndent-spaced — port spec §9.1's condition, a pretty-printed
// manifest whose fields a grep-based check would miss.
const prettyManifestJSON = `{
  "schemaVersion": 1,
  "contract": "toolsmith/v1",
  "manifest_digest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcd"
}`

const probeSHA = "0123456789abcdef0123456789abcdef01234567"

// cleanRunStderr is port spec §10's clean-run rule: no stderr, except the
// §9.4 multiple-cmd/ note.
var cleanRunStderr = regexp.MustCompile(`\A(note: multiple cmd/ entries \([^)]*\); auditing as "[^"]*"\n)?\z`)

func generateC62MislabelProbe(t *testing.T, dir string) {
	t.Helper()
	mustWriteFile(t, dir, "Makefile", "")
}

func generateC62HoleProbe(t *testing.T, dir string) {
	t.Helper()
	mustWriteFile(t, dir, "cliff.toml", "")
}

func generateChangelogTrackedProbe(t *testing.T, dir string) {
	t.Helper()
	mustWriteFile(t, dir, "CHANGELOG.md", "# Changelog\n")
	env := hermeticEnv(t, nil)
	runGit(t, dir, env, "init", "-q")
	runGit(t, dir, env, "add", "CHANGELOG.md")
}

func generateMulticmdProbe(t *testing.T, dir string) {
	t.Helper()
	mustMkdirAll(t, filepath.Join(dir, "cmd", "aaa"))
	mustMkdirAll(t, filepath.Join(dir, "cmd", "bbb"))
	mustMkdirAll(t, filepath.Join(dir, ".github", "workflows"))
	mustMkdirAll(t, filepath.Join(dir, "contrib"))

	mustWriteFile(t, dir, "cmd/aaa/PLACEHOLDER", "")
	mustWriteFile(t, dir, "cmd/bbb/PLACEHOLDER", "")

	mustWriteFile(t, dir, "go.mod", "module parity-probe-multicmd\n\ngo 1.23\n")
	mustWriteFile(t, dir, "Makefile", "# probe fixture -- not a real Makefile, only grepped by contrib/check-contract\n"+
		"CGO_ENABLED=0\n"+
		"VERSION := $(shell git describe --tags --match 'v[0-9]*' --always --dirty)\n"+
		"\n"+
		"hooks:\n"+
		"\tpre-commit install --hook-type pre-commit --hook-type commit-msg\n")
	mustWriteFile(t, dir, "flake.nix", "{ CGO_ENABLED = 0; }\n")
	mustWriteFile(t, dir, ".envrc", "use flake\n")
	mustWriteFile(t, dir, ".golangci.yml", "version: \"2\"\n"+
		"linters:\n"+
		"  default: none\n"+
		"  enable:\n"+
		"    - forbidigo\n"+
		"linters-settings:\n"+
		"  forbidigo:\n"+
		"    forbid:\n"+
		"      - 'fmt\\.Print('\n")
	mustWriteFile(t, dir, "cliff.toml", "tag_pattern = \"v[0-9]*\"\n")
	for _, wf := range []string{"ci", "release"} {
		mustWriteFile(t, dir, ".github/workflows/"+wf+".yml",
			fmt.Sprintf("name: %s\njobs:\n  build:\n    steps:\n      - run: nix develop --command true\n      - uses: actions/checkout@%s\n", wf, probeSHA))
	}
	mustWriteFile(t, dir, "contrib/check-commit-msg", "")
	mustWriteFile(t, dir, ".pre-commit-config.yaml", "repos:\n  - hooks:\n      - id: commit-msg\n")
	mustWriteFile(t, dir, "README.md", "probe fixture\n")
}

func generateCRLFProbe(t *testing.T, dir string) {
	t.Helper()
	mustMkdirAll(t, filepath.Join(dir, ".github", "workflows"))
	content := "name: ci\r\njobs:\r\n  build:\r\n    steps:\r\n      - run: nix develop --command true\r\n      - uses: actions/checkout@" + probeSHA + "\r\n"
	mustWriteFile(t, dir, ".github/workflows/ci.yml", content)
}

func generatePrettyjsonProbe(t *testing.T, dir string) {
	t.Helper()
	mustMkdirAll(t, filepath.Join(dir, "cmd", "prettytool"))
	mustWriteFile(t, dir, "cmd/prettytool/PLACEHOLDER", "")
	mustWriteFile(t, dir, "go.mod", "module parity-probe-prettyjson\n\ngo 1.23\n")
}

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, dir string, env []string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}

// envForCase builds the case's full subprocess environment, including the
// fake-go response wiring when the case names a tool, and returns the
// fresh log path the fake go writes its own invocation line to.
func envForCase(t *testing.T, tc checkCase) (env []string, logPath string) {
	t.Helper()
	logPath = filepath.Join(t.TempDir(), "fakego.log")
	if err := os.WriteFile(logPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	overrides := map[string]string{"FAKE_GO_LOG": logPath}
	if tc.tool != "" {
		stdoutPath := filepath.Join(t.TempDir(), "manifest.stdout")
		if err := os.WriteFile(stdoutPath, []byte(tc.manifestStdout), 0o644); err != nil {
			t.Fatal(err)
		}
		overrides["FAKE_GO_STDOUT"] = stdoutPath
		overrides["FAKE_GO_EXIT"] = strconv.Itoa(tc.manifestExit)
	}
	return hermeticEnv(t, overrides), logPath
}

func wantFakeGoLog(tc checkCase) string {
	if tc.tool == "" {
		return ""
	}
	return fmt.Sprintf("run ./cmd/%s manifest --json CGO_ENABLED=0\n", tc.tool)
}

// TestGoldenCheck is the parity suite proper: every case in checkCases,
// run three times through the built binary for determinism, its blob and
// stderr compared to (or, under -update, written to) testdata/golden.
func TestGoldenCheck(t *testing.T) {
	for _, tc := range checkCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tc.setup(t, dir)
			target := dir
			if tc.targetRel != "" {
				target = filepath.Join(dir, tc.targetRel)
			}

			var blobs, stderrs []string
			for i := range 3 {
				env, logPath := envForCase(t, tc)
				stdout, stderr, exit := runBin(t, target, env, "check", target)
				blobs = append(blobs, blob(normalizeRepo(t, stdout, target), exit))
				stderrs = append(stderrs, normalizeRepo(t, stderr, target))

				gotLog, err := os.ReadFile(logPath)
				if err != nil {
					t.Fatalf("reading fake-go log: %v", err)
				}
				if want := wantFakeGoLog(tc); string(gotLog) != want {
					t.Errorf("run %d: fake-go log = %q, want %q", i, gotLog, want)
				}
			}
			for i := 1; i < 3; i++ {
				if blobs[i] != blobs[0] {
					t.Errorf("determinism: run %d's stdout+exit differs from run 0\nrun 0: %q\nrun %d: %q", i, blobs[0], i, blobs[i])
				}
				if stderrs[i] != stderrs[0] {
					t.Errorf("determinism: run %d's stderr differs from run 0\nrun 0: %q\nrun %d: %q", i, stderrs[0], i, stderrs[i])
				}
			}
			if strings.HasSuffix(blobs[0], "exit=0\n") && !cleanRunStderr.MatchString(stderrs[0]) {
				t.Errorf("port spec §10: a clean run wrote stderr other than the §9.4 note:\n%s", stderrs[0])
			}

			goldenCompare(t, filepath.Join("check", tc.name+".stdout"), blobs[0])
			goldenCompare(t, filepath.Join("check", tc.name+".stderr"), stderrs[0])

			if tc.invariant != nil {
				tc.invariant(t, blobs[0])
			}
		})
	}
}

// TestGoldenCheckCLI covers the usage-contract paths TestGoldenCheck's
// fixture-driven cases never reach: a non-directory argument, too many
// positional arguments, and the zero-argument default, which audits the
// current directory rather than requiring an explicit path (port spec §1
// judgment call 2).
func TestGoldenCheckCLI(t *testing.T) {
	t.Run("file argument", func(t *testing.T) {
		dir := t.TempDir()
		f := filepath.Join(dir, "not-a-dir")
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		env := hermeticEnv(t, nil)
		stdout, _, exit := runBin(t, dir, env, "check", f)
		if exit != 2 {
			t.Errorf("exit = %d, want 2", exit)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
	})

	t.Run("two positional arguments", func(t *testing.T) {
		dir := t.TempDir()
		env := hermeticEnv(t, nil)
		stdout, _, exit := runBin(t, dir, env, "check", "a", "b")
		if exit != 2 {
			t.Errorf("exit = %d, want 2", exit)
		}
		if stdout != "" {
			t.Errorf("stdout = %q, want empty", stdout)
		}
	})

	t.Run("zero arguments defaults to cwd", func(t *testing.T) {
		dir := t.TempDir()
		copyFixture(t, filepath.Join(repoRootDir(), "internal", "verbs", "check", "testdata", "corpus", "toolsmith", "repo"), dir)

		stdoutPath := filepath.Join(t.TempDir(), "manifest.stdout")
		if err := os.WriteFile(stdoutPath, []byte(readCorpusFile(t, "toolsmith", "manifest.stdout")), 0o644); err != nil {
			t.Fatal(err)
		}
		logPath := filepath.Join(t.TempDir(), "fakego.log")
		if err := os.WriteFile(logPath, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		exitStr := strings.TrimSpace(readCorpusFile(t, "toolsmith", "manifest.exit"))
		env := hermeticEnv(t, map[string]string{
			"FAKE_GO_LOG":    logPath,
			"FAKE_GO_STDOUT": stdoutPath,
			"FAKE_GO_EXIT":   exitStr,
		})

		stdout, stderr, exit := runBin(t, dir, env, "check")
		wantRepo := dir
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			wantRepo = resolved
		}
		want := fmt.Sprintf("no findings — mechanical clauses hold for %s\n", wantRepo)
		if stdout != want {
			t.Errorf("stdout = %q, want %q", stdout, want)
		}
		if exit != 0 {
			t.Errorf("exit = %d, want 0", exit)
		}
		if stderr != "" {
			t.Errorf("stderr = %q, want empty", stderr)
		}
	})
}

// TestGoldenCheckStaleFiles requires every golden file under
// testdata/golden/check/ to name a live case — the direction
// TestGoldenCheck itself cannot check, since a stale golden simply never
// gets read.
func TestGoldenCheckStaleFiles(t *testing.T) {
	names := map[string]bool{}
	for _, tc := range checkCases(t) {
		names[tc.name] = true
	}
	dir := goldenPath(t, "check")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		base := strings.TrimSuffix(strings.TrimSuffix(e.Name(), ".stdout"), ".stderr")
		if !names[base] {
			t.Errorf("stale golden file %s: no case named %q in checkCases", e.Name(), base)
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
		if _, err := os.Stat(filepath.Join(dir, n+".stderr")); err != nil {
			t.Errorf("case %q has no golden %s.stderr", n, n)
		}
	}
}
