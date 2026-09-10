package manifest

import (
	"io/fs"
	"strings"
	"testing"

	rootassets "github.com/procrastivity/toolsmith/assets"
)

// TestIsShippedAssetTable pins that the .go exclusion reaches exactly the
// assets package's own source and nothing under a subdirectory. The
// _skeleton rows are the regression: a blanket "ends in .go" test dropped
// all 32 of the chassis's Go sources — the whole payload `new` writes —
// out of the checksum list C3.3 keeps so drift and tampering are
// detectable, and so out of the manifest_digest computed over it.
func TestIsShippedAssetTable(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"assets.go", false},
		{"doc.go", false},

		{"_skeleton/cmd/toolname/main.go", true},
		{"_skeleton/internal/cli/root.go", true},
		{"_skeleton/internal/cli/e2e_test.go", true},
		{"_skeleton/assets/assets.go", true},

		{"_skeleton/go.mod.tmpl", true},
		{"playbook/migrate.md", true},
		{"agent-guidance.md", true},
	}
	for _, c := range cases {
		if got := isShippedAsset(c.path); got != c.want {
			t.Errorf("isShippedAsset(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

// TestWalkEmbedded_ListsEveryShippedFile asserts the walk is exhaustive
// against the embedded tree rather than spot-checking it: every file in the
// fallback FS is listed with a checksum, except the package's own top-level
// Go source. An asset that reaches the binary but not the manifest is
// invisible to C3.3 and to the digest, which is the failure this guards.
func TestWalkEmbedded_ListsEveryShippedFile(t *testing.T) {
	got, err := walkEmbedded()
	if err != nil {
		t.Fatalf("walkEmbedded: %v", err)
	}

	listed := make(map[string]string, len(got))
	for _, a := range got {
		if a.SHA256 == "" {
			t.Errorf("asset %q carries no checksum (C3.3)", a.Path)
		}
		if _, dup := listed[a.Path]; dup {
			t.Errorf("asset %q listed twice", a.Path)
		}
		listed[a.Path] = a.SHA256
	}

	var skeletonGo int
	err = fs.WalkDir(rootassets.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		want := isShippedAsset(path)
		if _, ok := listed[path]; ok != want {
			t.Errorf("embedded file %q: listed = %v, want %v", path, ok, want)
		}
		if want && strings.HasPrefix(path, "_skeleton/") && strings.HasSuffix(path, ".go") {
			skeletonGo++
		}
		delete(listed, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walking embedded fallback: %v", err)
	}

	for path := range listed {
		t.Errorf("manifest lists %q, which is not in the embedded tree", path)
	}

	// The chassis cannot compile without its own sources, so a zero here
	// means the skeleton stopped reaching the manifest, not that the
	// skeleton changed shape.
	if skeletonGo == 0 {
		t.Errorf("no _skeleton Go source reached the asset list; the payload `new` writes is unchecksummed")
	}
}

// TestWalkEmbedded_SortedByPath pins the order manifestDigest's determinism
// argument depends on: two Build calls against the same binary must produce
// the same document, so the asset list may not vary with FS iteration order.
func TestWalkEmbedded_SortedByPath(t *testing.T) {
	got, err := walkEmbedded()
	if err != nil {
		t.Fatalf("walkEmbedded: %v", err)
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Path >= got[i].Path {
			t.Fatalf("asset list is not sorted by path: %q before %q", got[i-1].Path, got[i].Path)
		}
	}
}
