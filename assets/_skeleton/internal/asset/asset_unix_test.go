//go:build linux || darwin

package asset_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	rootassets "github.com/procrastivity/toolname/assets"
	"github.com/procrastivity/toolname/internal/asset"
)

func TestResolveInPermissionErrors(t *testing.T) {
	if os.Getenv("ASSET_TEST_CHILD") != t.Name() {
		prefix, cmd := installedTestCommand(t)
		deniedFile := projectPath(filepath.Join(prefix, "file"), runtimeName)
		deniedDir := filepath.Join(prefix, "directory", ".toolname")
		deniedRoot := filepath.Join(prefix, "inaccessible")
		xdg := filepath.Join(prefix, "xdg", "toolname", runtimeName)
		writeAsset(t, deniedFile, "unreadable local file")
		writeAsset(t, projectPath(filepath.Join(prefix, "directory"), runtimeName), "unsearchable directory")
		writeAsset(t, filepath.Join(deniedRoot, "selected", "placeholder"), "unsearchable root parent")
		writeAsset(t, xdg, "unreadable XDG file")
		if err := os.Mkdir(filepath.Join(prefix, "empty"), 0o755); err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{deniedFile, deniedDir, deniedRoot, xdg} {
			// Restore access before the temporary installation's cleanup.
			t.Cleanup(func() {
				if err := os.Chmod(path, 0o755); err != nil {
					t.Error(err)
				}
			})
			if err := os.Chmod(path, 0); err != nil {
				t.Fatal(err)
			}
		}
		// chmod does not deny reads to root. Run the public API unprivileged
		// even when CI runs this test suite as root; do not skip that case.
		if os.Geteuid() == 0 {
			cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 65534, Gid: 65534}}
		}
		out, err := cmd.CombinedOutput()
		t.Logf("permission test: %s", out)
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	if os.Geteuid() == 0 {
		t.Fatal("permission checks must run unprivileged")
	}
	t.Logf("permission assertions running as euid=%d", os.Geteuid())
	prefix := os.Getenv("ASSET_TEST_PREFIX")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(prefix, "xdg"))
	for _, root := range []string{filepath.Join(prefix, "file"), filepath.Join(prefix, "directory"), filepath.Join(prefix, "inaccessible", "selected")} {
		if _, err := asset.ResolveIn(root, runtimeName); !errors.Is(err, fs.ErrPermission) {
			t.Errorf("root %q: %v, want permission error without fallback", root, err)
		}
	}
	embedded, err := rootassets.FS.ReadFile(runtimeName)
	if err != nil {
		t.Fatal(err)
	}
	got, err := asset.Resolve(runtimeName)
	assertResolved(t, got, err, asset.SourceEmbedded, "", string(embedded))
	got, err = asset.ResolveIn(filepath.Join(prefix, "empty"), runtimeName)
	assertResolved(t, got, err, asset.SourceEmbedded, "", string(embedded))
}

func TestResolveInRejectsFIFO(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	path := projectPath(root, runtimeName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := asset.ResolveIn(root, runtimeName); !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("FIFO: %v, want invalid without attempting a blocking read", err)
	}
}
