// Package uninstall implements the `toolsmith uninstall <harness>` verb: it
// removes exactly the stamped tree a prior `toolsmith install <harness>`
// wrote, and refuses — rather than silently proceeding — if it finds
// unstamped content at the target path (C4.7).
package uninstall

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/procrastivity/toolsmith/internal/cliflags"
	"github.com/procrastivity/toolsmith/internal/harness/registry"
	"github.com/procrastivity/toolsmith/internal/iostreams"
	"github.com/procrastivity/toolsmith/internal/surface"
)

import "github.com/procrastivity/toolsmith/internal/toolsmitherr"

// Command constructs the `toolsmith uninstall <harness>` verb.
func Command(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall <harness>",
		Short: "remove a previously installed harness self-projection",
		Long: "remove a previously installed harness self-projection.\n\n" +
			"Available harnesses: " + strings.Join(registry.Names, ", ") + ".",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				_, err := fmt.Fprintf(streams.Out, "available harnesses: %s\nusage: toolsmith uninstall <harness>\n", strings.Join(registry.Names, ", "))
				return err
			}
			harnessName := args[0]
			if !slices.Contains(registry.Names, harnessName) {
				quoted := make([]string, len(registry.Names))
				for i, name := range registry.Names {
					quoted[i] = fmt.Sprintf("%q", name)
				}
				return toolsmitherr.New("validation.unknown-harness",
					fmt.Sprintf("unknown harness %q — only %s is supported", harnessName, strings.Join(quoted, ", or ")))
			}

			flags := cliflags.FromContext(cmd.Context())

			// harnessName was already validated against registry.Names
			// above, so Lookup is guaranteed to find it here.
			h, _ := registry.Lookup(harnessName)
			dir, err := h.Uninstall()
			if err != nil {
				return err
			}

			if flags.JSON {
				payload := struct {
					Harness string `json:"harness"`
					Dir     string `json:"dir"`
				}{Harness: harnessName, Dir: dir}
				b, err := json.Marshal(payload)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(streams.Out, string(b))
				return err
			}

			_, err = fmt.Fprintf(streams.Out, "uninstalled %s skill from %s\n", harnessName, dir)
			return err
		},
	}
	surface.Annotate(cmd, surface.Plumbing)
	return cmd
}
