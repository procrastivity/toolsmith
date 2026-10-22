// Package cliflags carries --json, --verbose, and the optional
// --project-root selection from root's PersistentPreRunE through context.
// The first two are C2.3's global flags; project-root only activates
// project-aware runtime asset reads.
package cliflags

import "context"

type contextKey struct{}

// Flags is the set of global flag values threaded through context. A tool
// that grows more global flags (wip added --as-role with an env fallback)
// extends this struct and root's PersistentPreRunE together — nowhere else.
type Flags struct {
	JSON        bool
	Verbose     bool
	ProjectRoot string
}

// WithFlags returns a context carrying f, for verbs to read via FromContext.
func WithFlags(ctx context.Context, f Flags) context.Context {
	return context.WithValue(ctx, contextKey{}, f)
}

// FromContext returns the Flags stored by WithFlags, or the zero value if
// none were stored.
func FromContext(ctx context.Context) Flags {
	f, _ := ctx.Value(contextKey{}).(Flags)
	return f
}
