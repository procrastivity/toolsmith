package new_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/procrastivity/toolsmith/internal/exitcode"
	"github.com/procrastivity/toolsmith/internal/iostreams"
	"github.com/procrastivity/toolsmith/internal/toolsmitherr"
	newverb "github.com/procrastivity/toolsmith/internal/verbs/new"
)

// run drives new's Command in-process and maps whatever it returns to the
// exit code internal/cli.Execute would produce, so the tests below can
// assert C2.4's table directly.
func run(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	var out, errBuf bytes.Buffer
	cmd := newverb.Command(&iostreams.Streams{Out: &out, Err: &errBuf})
	cmd.SetArgs(append([]string{}, args...)) // never nil: Cobra reads os.Args on nil
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	runErr := cmd.Execute()
	if runErr == nil {
		return out.String(), errBuf.String(), 0
	}
	var terr *toolsmitherr.Error
	if errors.As(runErr, &terr) {
		return out.String(), errBuf.String(), exitcode.FromError(terr)
	}
	// Cobra's own argument-parsing failures.
	return out.String(), errBuf.String(), exitcode.Usage
}

// TestChecklistBytes pins the whole of new's stdout contract (port spec
// §5.2) — the blank second line and every line's indentation included.
// The target directory appears exactly as it was passed, never
// absolutized, because the oracle prints $target_dir unmodified.
func TestChecklistBytes(t *testing.T) {
	chdir(t, t.TempDir())

	stdout, stderr, code := run(t, "acme", "--dir", "out/acme", "--no-git")
	if code != 0 || stderr != "" {
		t.Fatalf("exit=%d stderr=%q, want a clean run", code, stderr)
	}

	want := `instantiated acme at out/acme (module github.com/procrastivity/acme)

Checklist — the judgment steps the rename cannot do:
  1. grep -rn 'TODO(acme)' — fill every marker: root Short, README,
     flake meta.description, the judgment and agent-guidance assets, the
     skill description.
  2. cd out/acme && CGO_ENABLED=0 go build ./... && go test ./...
     (should already pass; it did in the skeleton).
  3. nix build — it fails once and prints the real vendorHash; paste it
     into flake.nix.
  4. make hooks — installs both pre-commit stages.
  5. Decide the verb surface; register verbs in internal/cli/root.go,
     one package each, every constructor ending in surface.Annotate.
  6. For a migration (not a fresh tool): follow toolsmith's
     assets/playbook/migrate.md — port spec, parity gate, cutover.
  7. Run toolsmith check out/acme and clear any findings.
  8. Add the tool to toolsmith's TOOLS.md.
`
	if stdout != want {
		t.Errorf("checklist mismatch\n--- got ---\n%s\n--- want ---\n%s", stdout, want)
	}
}

// TestInstantiatedTree asserts what the renames and the content pass are
// for: the placeholder is gone from every path and every file, the module
// files are back under their real names, and the two hooks are executable.
func TestInstantiatedTree(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	if _, _, code := run(t, "acme", "--dir", "acme", "--no-git"); code != 0 {
		t.Fatalf("exit=%d, want 0", code)
	}

	for _, rel := range []string{
		"go.mod", "go.sum",
		"cmd/acme/main.go",
		"internal/acmeerr/acmeerr.go",
		".envrc", ".gitignore", ".golangci.yml", ".pre-commit-config.yaml",
		".github/workflows/ci.yml", ".github/workflows/release.yml",
	} {
		if _, err := os.Stat(filepath.Join("acme", rel)); err != nil {
			t.Errorf("expected %s in the instantiated tree: %v", rel, err)
		}
	}
	for _, rel := range []string{"go.mod.tmpl", "go.sum.tmpl", "cmd/toolname", "internal/toolnameerr"} {
		if _, err := os.Stat(filepath.Join("acme", rel)); err == nil {
			t.Errorf("%s survived instantiation", rel)
		}
	}

	for _, rel := range []string{"contrib/check-commit-msg", "contrib/check-gofumpt"} {
		info, err := os.Stat(filepath.Join("acme", rel))
		if err != nil {
			t.Errorf("expected %s: %v", rel, err)
			continue
		}
		if info.Mode().Perm()&0o111 == 0 {
			t.Errorf("%s is not executable in the instantiated tree (mode %v)", rel, info.Mode().Perm())
		}
	}

	// No placeholder spelling may survive anywhere — in a path or in a
	// file. This is the assertion the oracle's "exhaustive by construction"
	// claim rests on, and the one that caught the unrenamed
	// toolnameerr.go the header comment describes.
	err := filepath.WalkDir("acme", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(path, "toolname") || strings.Contains(path, "TOOLNAME") {
			t.Errorf("placeholder survives in path %s", path)
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte("toolname")) || bytes.Contains(data, []byte("TOOLNAME")) {
			t.Errorf("placeholder survives in the contents of %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestExitCodes pins port spec §9.3: Cobra's path exits 2, the
// existing-target guard exits 3, everything else exits 1 — and no failure
// path writes anything to stdout.
func TestExitCodes(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	if err := os.Mkdir("taken", 0o755); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		label string
		args  []string
		want  int
	}{
		{"unknown flag", []string{"--bogus", "acme"}, exitcode.Usage},
		{"missing flag value", []string{"acme", "--dir"}, exitcode.Usage},
		{"no name", nil, exitcode.Usage},
		{"two positionals", []string{"a", "b"}, exitcode.Usage},
		{"bad name shape", []string{"Acme"}, exitcode.UserFail},
		{"the placeholder name", []string{"toolname"}, exitcode.UserFail},
		{"existing target", []string{"acme", "--dir", "taken"}, exitcode.Refusal},
	}
	for _, c := range cases {
		stdout, _, code := run(t, c.args...)
		if code != c.want {
			t.Errorf("%s: exit=%d, want %d", c.label, code, c.want)
		}
		if stdout != "" {
			t.Errorf("%s: wrote %q to stdout; every failure path writes nothing there", c.label, stdout)
		}
	}
}

// chdir is testing.T.Chdir, which needs go1.24; this module is go1.23
// (go.mod). Drop this helper for t.Chdir when the module's Go version moves.
func chdir(t *testing.T, dir string) {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
}
