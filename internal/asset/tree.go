package asset

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	rootassets "github.com/procrastivity/toolsmith/assets"
)

// Tree resolves a whole asset subtree — not a single file — by the same
// override -> default -> embedded chain Resolve uses (C5.1), and returns an
// fs.FS rooted at prefix together with the link that satisfied it.
//
// Shadow by name applies to the tree as a unit (C5.2): an override
// directory at prefix fully replaces the shipped default's, and the two are
// never merged file by file. A caller that wants per-file shadowing wants
// Resolve, not this.
//
// The embedded link loses one thing the two disk links keep: embed.FS
// reports every file as mode 0444, so a caller writing the tree out has to
// reconstitute the executable bit rather than copy it (see
// internal/verbs/new).
func Tree(prefix string) (fs.FS, Source, error) {
	if dir, err := OverrideDir(); err == nil {
		if sub, ok := dirFSIfExists(filepath.Join(dir, prefix)); ok {
			return sub, SourceOverride, nil
		}
	}
	return DefaultTree(prefix)
}

// DefaultTree resolves prefix by the default -> embedded chain only,
// skipping the override link Tree checks first. It exists for a caller
// that enumerates the *names* in a tree (internal/verbs/doc lists
// playbook/ and handoff-kit/) rather than resolving one asset's content:
// shadow by name (C5.2) replaces a file at a name the shipped tree already
// has, so the set of names on offer must never grow just because an
// override directory happens to contain an extra file — "overrides never
// add names." A per-file override is still honored where it matters, at
// content-resolution time, through Resolve.
func DefaultTree(prefix string) (fs.FS, Source, error) {
	if dir, err := DefaultDir(); err == nil {
		if sub, ok := dirFSIfExists(filepath.Join(dir, prefix)); ok {
			return sub, SourceDefault, nil
		}
	}

	sub, err := fs.Sub(rootassets.FS, prefix)
	if err != nil {
		return nil, SourceEmbedded, fmt.Errorf("asset: %q not found in the default directory, and the embedded fallback has no such tree: %w", prefix, err)
	}
	// fs.Sub succeeds for a prefix that does not exist, so confirm the
	// subtree is really there before reporting it resolved.
	if _, err := fs.Stat(sub, "."); err != nil {
		return nil, SourceEmbedded, fmt.Errorf("asset: %q not found in the default directory, and the embedded fallback has no such tree", prefix)
	}
	return sub, SourceEmbedded, nil
}

func dirFSIfExists(path string) (fs.FS, bool) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return nil, false
	}
	return os.DirFS(path), true
}
