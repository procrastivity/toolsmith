package new

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/procrastivity/toolsmith/internal/toolsmitherr"
)

// TestDestPathTable pins the five renames contrib/new-tool.sh performs and,
// just as importantly, pins that nothing else moves. The near-miss cases at
// the bottom are the ones a "replace toolname anywhere in the path" rule
// would get wrong (port spec §4.2 step 5).
func TestDestPathTable(t *testing.T) {
	cases := []struct{ in, want string }{
		{"go.mod.tmpl", "go.mod"},
		{"go.sum.tmpl", "go.sum"},
		{"cmd/toolname/main.go", "cmd/acme/main.go"},
		{"internal/toolnameerr/toolnameerr.go", "internal/acmeerr/acmeerr.go"},
		{"internal/toolnameerr/other.go", "internal/acmeerr/other.go"},

		{"README.md", "README.md"},
		{"assets/assets.go", "assets/assets.go"},
		{"assets/templates/toolname.md", "assets/templates/toolname.md"},
		{"cmd/toolnamex/main.go", "cmd/toolnamex/main.go"},
	}
	for _, c := range cases {
		if got := destPath(c.in, "acme"); got != c.want {
			t.Errorf("destPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestSubstituteOrderWithCustomModule is the case the oracle's own comment
// calls load-bearing: with a custom --module, replacing the bare
// placeholder first would strand github.com/procrastivity/<name> in every
// file. The default module hides the bug, because both orders produce the
// same string there — so the test uses a custom one.
func TestSubstituteOrderWithCustomModule(t *testing.T) {
	p := params{name: "acme", module: "example.com/x/acme", upperName: "ACME"}
	in := []byte("import \"github.com/procrastivity/toolname/internal/toolnameerr\"\nconst env = \"TOOLNAME_CONFIG\"\n")
	want := "import \"example.com/x/acme/internal/acmeerr\"\nconst env = \"ACME_CONFIG\"\n"
	if got := string(substitute(in, p)); got != want {
		t.Errorf("substitute() = %q, want %q", got, want)
	}
}

func TestSubstituteDefaultModule(t *testing.T) {
	p := params{name: "acme", module: "github.com/procrastivity/acme", upperName: "ACME"}
	in := []byte("github.com/procrastivity/toolname and bare toolname and TOOLNAME\n")
	want := "github.com/procrastivity/acme and bare acme and ACME\n"
	if got := string(substitute(in, p)); got != want {
		t.Errorf("substitute() = %q, want %q", got, want)
	}
}

// TestValidateOrderAndCodes walks contrib/new-tool.sh's validation order
// (port spec §4.2 step 2) and pins the error codes exitcode.FromError reads
// to pick the exit code (port spec §9.3): only the existing-target guard is
// a refusal.
func TestValidateOrderAndCodes(t *testing.T) {
	existing := t.TempDir()

	cases := []struct {
		label    string
		name     string
		dir      string
		wantCode string
	}{
		{"empty name", "", "", "new.missing-name"},
		{"uppercase", "Acme", "", "new.invalid-name"},
		{"leading digit", "1acme", "", "new.invalid-name"},
		{"hyphen", "ac-me", "", "new.invalid-name"},
		{"the placeholder itself", "toolname", "", "new.placeholder-name"},
		{"existing target", "acme", existing, "refusal.target-exists"},
	}
	for _, c := range cases {
		_, err := validate(c.name, c.dir, "", true)
		var terr *toolsmitherr.Error
		if err == nil {
			t.Errorf("%s: validate() succeeded, want %s", c.label, c.wantCode)
			continue
		}
		if !asToolsmithErr(err, &terr) || terr.Code != c.wantCode {
			t.Errorf("%s: validate() = %v, want code %s", c.label, err, c.wantCode)
		}
	}
}

// TestValidateResolvesDefaultsBeforeRefusing is the ordering the oracle's
// line 72/73 pair encodes: --dir defaults are applied first, so a caller
// who omits --dir gets the same refusal as one who passed the same path
// explicitly.
func TestValidateResolvesDefaultsBeforeRefusing(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	if err := os.Mkdir("acme", 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := validate("acme", "", "", true)
	var terr *toolsmitherr.Error
	if err == nil || !asToolsmithErr(err, &terr) || terr.Code != "refusal.target-exists" {
		t.Fatalf("validate() with a defaulted --dir = %v, want refusal.target-exists", err)
	}
}

func TestValidateDefaults(t *testing.T) {
	chdir(t, t.TempDir())
	p, err := validate("acme", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if p.targetDir != "acme" {
		t.Errorf("targetDir = %q, want %q", p.targetDir, "acme")
	}
	if p.module != "github.com/procrastivity/acme" {
		t.Errorf("module = %q, want the procrastivity default", p.module)
	}
	if p.upperName != "ACME" {
		t.Errorf("upperName = %q, want ACME", p.upperName)
	}
	if !p.doGit {
		t.Error("doGit = false, want true when --no-git is not passed")
	}
}

// TestSkeletonHooksAreExecutable pins the contract destMode depends on. The
// embedded asset link reports every file as 0444, so the executable bit is
// reconstituted from a shebang; that is only correct while every file the
// skeleton needs executable actually starts with one. If a future skeleton
// ships an executable without a shebang, this test is what says so.
func TestSkeletonHooksAreExecutable(t *testing.T) {
	root := filepath.Join("..", "..", "..", "assets", skeletonPrefix)

	var missing []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode().Perm()&0o111 == 0 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(string(data), "#!") {
			missing = append(missing, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range missing {
		t.Errorf("%s is executable in the skeleton but has no shebang, so an instantiation from the embedded asset link would write it non-executable", path)
	}
}

func asToolsmithErr(err error, out **toolsmitherr.Error) bool {
	terr, ok := err.(*toolsmitherr.Error)
	if ok {
		*out = terr
	}
	return ok
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
