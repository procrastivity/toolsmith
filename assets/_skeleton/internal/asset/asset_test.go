package asset_test

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	rootassets "github.com/procrastivity/toolname/assets"
	"github.com/procrastivity/toolname/internal/asset"
)

const runtimeName = "templates/skills/claude-code/judgment.md"

func writeAsset(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func projectPath(root, name string) string {
	return filepath.Join(root, ".toolname", "assets", filepath.FromSlash(name))
}

func canonical(t *testing.T, path string) string {
	t.Helper()
	got, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func assertResolved(t *testing.T, got asset.Resolved, err error, source asset.Source, path, data string) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != source || got.Path != path || string(got.Bytes()) != data {
		t.Fatalf("resolved = (%s, %q, %q), want (%s, %q, %q)", got.Source, got.Path, got.Bytes(), source, path, data)
	}
}

// Run the same public-API test from a real <prefix>/bin layout, without
// replacing the resolver's executable hook or touching the host installation.
func installedTestCommand(t *testing.T) (string, *exec.Cmd) {
	t.Helper()
	prefix, err := os.MkdirTemp("", "asset-installed-")
	if err != nil {
		t.Fatal(err)
	}
	prefix = canonical(t, prefix)
	t.Cleanup(func() {
		if err := os.RemoveAll(prefix); err != nil {
			t.Error(err)
		}
	})
	if err := os.Chmod(prefix, 0o755); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(prefix, "bin", "toolname")
	writeAsset(t, bin, string(data))
	if err := os.Chmod(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "-test.run=^"+t.Name()+"$", "-test.v")
	cmd.Dir = prefix
	cmd.Env = append(os.Environ(), "ASSET_TEST_CHILD="+t.Name(), "ASSET_TEST_PREFIX="+prefix)
	return prefix, cmd
}

func TestResolveInPrecedence(t *testing.T) {
	if os.Getenv("ASSET_TEST_CHILD") != t.Name() {
		_, cmd := installedTestCommand(t)
		out, err := cmd.CombinedOutput()
		t.Logf("installed test: %s", out)
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	local := projectPath(root, runtimeName)
	xdg := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "toolname", filepath.FromSlash(runtimeName))
	share := filepath.Join(os.Getenv("ASSET_TEST_PREFIX"), "share", "toolname", "assets", filepath.FromSlash(runtimeName))
	writeAsset(t, local, "project A")
	writeAsset(t, xdg, "user B")
	writeAsset(t, share, "shipped C")
	got, err := asset.ResolveIn(root, runtimeName)
	assertResolved(t, got, err, asset.SourceProject, canonical(t, local), "project A")
	got, err = asset.Resolve(runtimeName)
	assertResolved(t, got, err, asset.SourceOverride, xdg, "user B")
	got, err = asset.ResolveIn("", runtimeName)
	assertResolved(t, got, err, asset.SourceOverride, xdg, "user B")

	// Local presence shadows per file, even with empty or unparseable bytes.
	for _, content := range []string{"", "[invalid: yaml"} {
		writeAsset(t, local, content)
		got, err = asset.ResolveIn(root, runtimeName)
		assertResolved(t, got, err, asset.SourceProject, canonical(t, local), content)
	}
	for _, tc := range []struct {
		remove string
		source asset.Source
		path   string
		data   string
	}{
		{local, asset.SourceOverride, xdg, "user B"},
		{xdg, asset.SourceDefault, share, "shipped C"},
	} {
		if err := os.Remove(tc.remove); err != nil {
			t.Fatal(err)
		}
		got, err = asset.ResolveIn(root, runtimeName)
		assertResolved(t, got, err, tc.source, tc.path, tc.data)
		got, err = asset.Resolve(runtimeName)
		assertResolved(t, got, err, tc.source, tc.path, tc.data)
	}
	if err := os.Remove(share); err != nil {
		t.Fatal(err)
	}
	embedded, err := rootassets.FS.ReadFile(runtimeName)
	if err != nil {
		t.Fatal(err)
	}
	got, err = asset.ResolveIn(root, runtimeName)
	assertResolved(t, got, err, asset.SourceEmbedded, "", string(embedded))
	got, err = asset.Resolve(runtimeName)
	assertResolved(t, got, err, asset.SourceEmbedded, "", string(embedded))

	// The installed distribution may ship names absent from this binary.
	const installedOnly = "templates/installed-only.txt"
	writeAsset(t, filepath.Join(os.Getenv("ASSET_TEST_PREFIX"), "share", "toolname", "assets", installedOnly), "installed")
	writeAsset(t, projectPath(root, installedOnly), "local installed name")
	got, err = asset.ResolveIn(root, installedOnly)
	assertResolved(t, got, err, asset.SourceProject, canonical(t, projectPath(root, installedOnly)), "local installed name")
}

func TestResolveInIndependentExactRoots(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a, b := t.TempDir(), t.TempDir()
	writeAsset(t, projectPath(a, runtimeName), "repository A")
	writeAsset(t, projectPath(b, runtimeName), "repository B")
	// A sibling file does not make this a whole-tree override.
	xdg := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "toolname", "agent-guidance.md")
	writeAsset(t, xdg, "user guidance")
	for _, root := range []string{a, b, a} {
		want := "repository A"
		if root == b {
			want = "repository B"
		}
		got, err := asset.ResolveIn(root, runtimeName)
		assertResolved(t, got, err, asset.SourceProject, canonical(t, projectPath(root, runtimeName)), want)
		got, err = asset.ResolveIn(root, "agent-guidance.md")
		assertResolved(t, got, err, asset.SourceOverride, xdg, "user guidance")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(cwd, a)
	if err != nil {
		t.Fatal(err)
	}
	got, err := asset.ResolveIn(relative, runtimeName)
	assertResolved(t, got, err, asset.SourceProject, canonical(t, projectPath(a, runtimeName)), "repository A")
	nested := filepath.Join(a, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	writeAsset(t, filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "toolname", filepath.FromSlash(runtimeName)), "no parent discovery")
	got, err = asset.ResolveIn(nested, runtimeName)
	assertResolved(t, got, err, asset.SourceOverride, filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "toolname", filepath.FromSlash(runtimeName)), "no parent discovery")
}

func TestResolveInNamesAndRoots(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	for _, name := range []string{"local-only.md", "assets.go", "templates", "CONTRACT.md"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			root := t.TempDir()
			writeAsset(t, projectPath(root, name), "project-only")
			xdg := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "toolname", name)
			writeAsset(t, xdg, "legacy user name")
			if _, err := asset.ResolveIn(root, name); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("ResolveIn(%q): %v, want unknown shipped name", name, err)
			}
			// The new API must not narrow the legacy API's accepted names.
			got, err := asset.Resolve(name)
			assertResolved(t, got, err, asset.SourceOverride, xdg, "legacy user name")
		})
	}
	for _, name := range []string{"", ".", "..", "../agent-guidance.md", "/agent-guidance.md", "templates/../../agent-guidance.md", "templates/../agent-guidance.md", "./agent-guidance.md", "templates//x", "templates/./x", "templates/", `templates\x`, `C:/x`, `//server/x`, "x\x00y"} {
		if _, err := asset.ResolveIn(root, name); !errors.Is(err, fs.ErrInvalid) {
			t.Errorf("ResolveIn(%q): %v, want invalid name", name, err)
		}
	}
	writeAsset(t, filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "toolname", runtimeName), "must not mask invalid root")
	if _, err := asset.ResolveIn(filepath.Join(root, "absent"), runtimeName); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing root: %v", err)
	}
	fileRoot := filepath.Join(root, "file")
	writeAsset(t, fileRoot, "not a directory")
	if _, err := asset.ResolveIn(fileRoot, runtimeName); !errors.Is(err, fs.ErrInvalid) {
		t.Errorf("file root: %v", err)
	}
}

func TestResolveInRejectsSymlinksAndNonregularPaths(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeAsset(t, filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "toolname", runtimeName), "must not mask unsafe project path")
	for _, component := range []string{".toolname", ".toolname/assets", ".toolname/assets/templates", ".toolname/assets/" + runtimeName} {
		for _, kind := range []string{"inside link", "outside link", "broken link", "nonregular"} {
			t.Run(component+"/"+kind, func(t *testing.T) {
				root := t.TempDir()
				path := filepath.Join(root, filepath.FromSlash(component))
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if kind == "nonregular" {
					if component == ".toolname/assets/"+runtimeName {
						if err := os.Mkdir(path, 0o755); err != nil {
							t.Fatal(err)
						}
					} else {
						writeAsset(t, path, "not a directory")
					}
				} else {
					target := filepath.Join(root, "target")
					if kind == "outside link" {
						target = filepath.Join(t.TempDir(), "target")
					}
					if kind != "broken link" {
						contentPath := target
						if component != ".toolname/assets/"+runtimeName {
							remaining := strings.TrimPrefix(".toolname/assets/"+runtimeName, component+"/")
							contentPath = filepath.Join(target, filepath.FromSlash(remaining))
						}
						writeAsset(t, contentPath, "symlink target")
					}
					if err := os.Symlink(target, path); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := asset.ResolveIn(root, runtimeName); !errors.Is(err, fs.ErrInvalid) {
					t.Fatalf("unsafe project path: %v, want invalid", err)
				}
			})
		}
	}
	// Symlinks in the selected root itself are canonicalized, not refused.
	root := t.TempDir()
	writeAsset(t, projectPath(root, runtimeName), "canonical project")
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	got, err := asset.ResolveIn(alias, runtimeName)
	assertResolved(t, got, err, asset.SourceProject, canonical(t, projectPath(root, runtimeName)), "canonical project")
}

func TestSourceCompatibility(t *testing.T) {
	for i, source := range []asset.Source{asset.SourceOverride, asset.SourceDefault, asset.SourceEmbedded, asset.SourceProject} {
		if int(source) != i || source.String() != []string{"override", "default", "embedded", "project"}[i] {
			t.Errorf("source %d = (%d, %q)", i, source, source.String())
		}
	}
}
