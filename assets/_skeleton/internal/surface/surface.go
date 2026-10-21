// Package surface names the three manifest kinds every verb declares on its
// registration (CONTRACT.md C3.2). A kind describes determinism/execution,
// not audience or command-tree placement (C8.3); the manifest walk requires
// the annotation and never infers it from a folder name or convention.
package surface

import "github.com/spf13/cobra"

// Kind is a verb's manifest surface kind.
type Kind string

const (
	// Plumbing verbs have a deterministic output contract: stable JSON and
	// exit codes, no LLM. This kind does not imply read-only behavior.
	// Where a tool draws the C8.3 boundary, C4.3 projects only verbs that are
	// both in the plumbing namespace and of plumbing kind.
	Plumbing Kind = "plumbing"
	// LLM verbs do CLI-side LLM shaping; no MCP. Absent from harness
	// projections because inside a harness the model already is the LLM.
	LLM Kind = "llm"
	// ControlPlane verbs require a live MCP surface; they exist only in the
	// agent porcelain.
	ControlPlane Kind = "control-plane"
)

// annotationKind is the cobra.Command.Annotations key a verb's kind is
// recorded under. The registration lives on the command itself so the
// manifest generator can walk the command tree without a parallel registry.
const annotationKind = "toolname.surface-kind"

// Annotate records kind as cmd's explicit, machine-readable surface kind.
func Annotate(cmd *cobra.Command, kind Kind) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[annotationKind] = string(kind)
}

// Of returns the kind recorded on cmd by Annotate, and whether one was found.
func Of(cmd *cobra.Command) (Kind, bool) {
	k, ok := cmd.Annotations[annotationKind]
	return Kind(k), ok
}
