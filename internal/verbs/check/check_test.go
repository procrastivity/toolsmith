package check_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/procrastivity/toolsmith/internal/exitcode"
	"github.com/procrastivity/toolsmith/internal/iostreams"
	"github.com/procrastivity/toolsmith/internal/verbs/check"
)

// run drives check's Command in-process against args, the way the other
// verb packages' own tests would (internal/cli/e2e_test.go covers the
// same paths again through the real built binary, at the chassis level).
func run(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	var out, err bytes.Buffer
	cmd := check.Command(&iostreams.Streams{Out: &out, Err: &err})
	cmd.SetArgs(append([]string{}, args...)) // never nil: Cobra reads os.Args on nil
	cmd.SetOut(&out)
	cmd.SetErr(&err)
	// internal/cli.NewRootCommand sets these on root, and root renders
	// every error itself (see internal/cli/execute.go); replicate that
	// here since this test drives check's Command standalone, without a
	// root to inherit it from.
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	runErr := cmd.Execute()
	code := 0
	if runErr != nil {
		var serr *exitcode.SilentError
		if errors.As(runErr, &serr) {
			code = serr.Code
		} else {
			// Cobra's own Args-validation errors (too many positional
			// args) and the plain "not a directory" error both reach
			// here; internal/cli.Execute maps both to exit 2 for the
			// real binary. Mirror that mapping directly since this test
			// drives the command in-process, not through Execute.
			code = exitcode.Usage
		}
	}
	return out.String(), err.String(), code
}

// TestCheck_NotADirectory covers §8.1's usage path: a path that does not
// resolve to a directory exits 2, with nothing on stdout.
func TestCheck_NotADirectory(t *testing.T) {
	f := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, code := run(t, f)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
}

// TestCheck_DefaultPath covers judgment call 2 (port spec §1, §8.1): with
// no positional argument, check audits the current directory rather than
// refusing for a wrong argument count the way the oracle does.
func TestCheck_DefaultPath(t *testing.T) {
	repo := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	stdout, _, code := run(t)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (an empty dir has findings); stdout=%q", code, stdout)
	}
	if !bytes.Contains([]byte(stdout), []byte("C7.5: no README.md")) {
		t.Fatalf("stdout = %q, want it to include the C7.5 finding for the empty cwd", stdout)
	}
}

// TestCheck_FindingsPath asserts the exact stdout bytes and exit code
// (port spec §5.1's findings-present row, §9.5's exitcode.Silent(1)) for
// a small, deliberately incomplete repo tree that never invokes `go run`
// (no go.mod), so the test has no toolchain dependency.
func TestCheck_FindingsPath(t *testing.T) {
	repo := t.TempDir()
	mustWrite(t, repo, "README.md", "# x\n")
	mustWrite(t, repo, "Makefile", "CGO_ENABLED=0 go build ./...\n")

	stdout, stderr, code := run(t, repo)

	// filepath.Abs (what check itself uses) doesn't resolve symlinks;
	// t.TempDir() on some platforms returns a symlinked path (e.g.
	// macOS's /tmp -> /private/tmp). Resolve it the same way before
	// comparing so this test isn't platform-specific.
	wantRepo, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	if resolved, err := filepath.EvalSymlinks(wantRepo); err == nil {
		wantRepo = resolved
	}

	want := "" +
		"C1.2: no cmd/<tool> main package found\n" +
		"C1.6: no flake.nix\n" +
		"C1.6: no .envrc\n" +
		"C2.1: no .golangci.yml\n" +
		"C3.1: cannot run the manifest verb (need go, go.mod, and cmd/<tool>)\n" +
		"C6.2: Makefile git describe lacks --match 'v[0-9]*'\n" +
		"C6.2: no cliff.toml (no tag_pattern for git describe --match to agree with)\n" +
		"C6.5: no .github/workflows/ci.yml\n" +
		"C6.5: no .github/workflows/release.yml\n" +
		"C6.6: no contrib/check-commit-msg hook\n" +
		"C6.6: no .pre-commit-config.yaml\n"

	if stdout != want {
		t.Fatalf("stdout =\n%q\nwant\n%q", stdout, want)
	}
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	wantSummary := fmt.Sprintf("11 finding(s) for %s\n", wantRepo)
	if stderr != wantSummary {
		t.Fatalf("stderr = %q, want %q", stderr, wantSummary)
	}
}

// TestCheck_CleanRepo builds a full, conformant fixture tree — including
// a minimal real Go module whose `manifest --json` the port actually
// shells out to (`go run`, per port spec §6) — and asserts the exact
// clean-run stdout line, empty stderr, and exit 0 (port spec §5.1's
// exit-0 row). This is the one test in the package that exercises the
// manifest subprocess path end-to-end rather than through
// checkManifestDoc directly.
func TestCheck_CleanRepo(t *testing.T) {
	if _, err := os.Stat(os.DevNull); err != nil {
		t.Skip("no /dev/null-equivalent host; unexpected environment")
	}
	repo := t.TempDir()

	mustWrite(t, repo, "go.mod", "module checkfixture\n\ngo 1.23\n")
	mustWrite(t, repo, "cmd/checkfixture/main.go", fixtureMain)
	mustWrite(t, repo, "Makefile",
		"CGO_ENABLED=0\nhooks:\n\tpre-commit install --hook-type commit-msg\n\ngit describe --match 'v[0-9]*'\n")
	mustWrite(t, repo, "flake.nix", "# CGO_ENABLED = 0\n# installs assets into share/\n")
	mustWrite(t, repo, "assets/placeholder.txt", "x\n")
	mustWrite(t, repo, ".envrc", "use flake\n")
	mustWrite(t, repo, ".golangci.yml", "version: \"2\"\nlinters:\n  default: none\n  settings:\n    forbidigo:\n      forbid:\n        - pattern: 'fmt\\.Print('\n")
	mustWrite(t, repo, "cliff.toml", `tag_pattern = "v[0-9]*"`+"\n")
	mustWrite(t, repo, ".github/workflows/ci.yml", "jobs:\n  build:\n    steps:\n      - run: nix develop --command make check\n")
	mustWrite(t, repo, ".github/workflows/release.yml", "jobs:\n  build:\n    steps:\n      - run: nix develop --command make build\n")
	mustWrite(t, repo, "contrib/check-commit-msg", "#!/usr/bin/env bash\n")
	mustWrite(t, repo, ".pre-commit-config.yaml", "repos:\n  - hooks:\n      - id: commit-msg\n")
	mustWrite(t, repo, "README.md", "# checkfixture\n")

	stdout, stderr, code := run(t, repo)

	wantRepo, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	if resolved, err := filepath.EvalSymlinks(wantRepo); err == nil {
		wantRepo = resolved
	}

	wantStdout := fmt.Sprintf("no findings — mechanical clauses hold for %s\n", wantRepo)
	if stdout != wantStdout {
		t.Fatalf("stdout = %q, want %q (stderr=%q)", stdout, wantStdout, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty on a clean run", stderr)
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

// fixtureMain is a minimal Go program whose only job is to answer
// `manifest --json` with a conformant document, so TestCheck_CleanRepo's
// fixture repo has a real cmd/<tool> the manifest subprocess step can
// `go run`.
const fixtureMain = `package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) >= 3 && os.Args[1] == "manifest" && os.Args[2] == "--json" {
		fmt.Println(` + "`" + `{"schemaVersion":1,"contract":"toolsmith/v1","manifest_digest":"sha256:deadbeef"}` + "`" + `)
	}
}
`

func mustWrite(t *testing.T, repo, rel, content string) {
	t.Helper()
	full := filepath.Join(repo, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
