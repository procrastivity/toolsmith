package manifest

import (
	"io/fs"
	"strings"
	"testing"

	rootassets "github.com/procrastivity/toolsmith/assets"
)

// TestIsShippedAssetTable_SkeletonPayload pins isShippedAsset against
// toolsmith's own shipped tree: the skeleton the new verb writes to disk,
// under assets/_skeleton/. Only toolsmith ships this payload — a generated
// tool's own skeleton copy carries no _skeleton/ of its own — so these rows
// live here rather than in the shared assets_test.go, which converges
// byte-for-byte with the skeleton's copy of this package.
func TestIsShippedAssetTable_SkeletonPayload(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"_skeleton/cmd/toolname/main.go", true},
		{"_skeleton/internal/cli/root.go", true},
		{"_skeleton/internal/cli/e2e_test.go", true},
		{"_skeleton/assets/assets.go", true},

		{"_skeleton/go.mod.tmpl", true},
		{"playbook/migrate.md", true},
	}
	for _, c := range cases {
		if got := isShippedAsset(c.path); got != c.want {
			t.Errorf("isShippedAsset(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

// TestWalkEmbedded_ReachesSkeletonPayload asserts that at least one
// _skeleton/ Go source reaches the manifest's asset list. The chassis
// cannot compile without its own sources, so a zero here means the walk
// stopped reaching the skeleton payload, not that the skeleton changed
// shape — exactly the Stage 5 regression (commit 9cba3cd): a blanket "ends
// in .go" exclusion dropped every one of the skeleton's Go sources from the
// checksum list C3.3 keeps and from the manifest_digest computed over it
// (C3.4).
func TestWalkEmbedded_ReachesSkeletonPayload(t *testing.T) {
	got, err := walkEmbedded()
	if err != nil {
		t.Fatalf("walkEmbedded: %v", err)
	}
	listed := make(map[string]struct{}, len(got))
	for _, a := range got {
		listed[a.Path] = struct{}{}
	}

	var skeletonGo int
	err = fs.WalkDir(rootassets.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if _, ok := listed[path]; ok && strings.HasPrefix(path, "_skeleton/") && strings.HasSuffix(path, ".go") {
			skeletonGo++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking embedded fallback: %v", err)
	}

	if skeletonGo == 0 {
		t.Errorf("no _skeleton Go source reached the asset list; the payload `new` writes is unchecksummed")
	}
}
