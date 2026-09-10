// Package claudecode renders the tool's manifest into a Claude Code skill
// directory plus a .claude-plugin/plugin.json in the same directory — the
// "skills-dir as plugin" mechanism: no marketplace manifest, no separate
// registry entry. Everything in the generated tree traces back to the
// manifest except two verbatim, hand-authored strings: the per-harness
// judgment paragraph and the shared agent-guidance paragraph, both
// resolved through the asset chain (C4.4).
package claudecode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/procrastivity/toolname/internal/asset"
	"github.com/procrastivity/toolname/internal/harness"
	"github.com/procrastivity/toolname/internal/manifest"
)

// Name is this harness's install-target name, as passed to `toolname
// install <harness>` / `toolname uninstall <harness>`.
const Name = "claude-code"

// skillName is the directory name the generated skill is installed under.
const skillName = "toolname"

// skillDescription is the SKILL.md frontmatter description.
// TODO(toolname): replace with one line saying what this tool's verb
// surface does — Claude reads it to decide when to load the skill.
const skillDescription = "Drive toolname through its plumbing verb surface."

// judgmentAsset is the one hand-authored prose paragraph this harness
// owns, resolved through the asset chain.
const judgmentAsset = "templates/skills/claude-code/judgment.md"

// guidanceAsset is the shared agent-guidance paragraph, seeded at the top
// level of the shipped asset tree and projected here verbatim — the only
// other hand-written string in the generated tree.
const guidanceAsset = "agent-guidance.md"

// SkillsDirEnv is an environment variable that, when set, overrides
// SkillsDir()'s result — a test seam only (C2.6), never read for any other
// purpose.
const SkillsDirEnv = "TOOLNAME_CLAUDE_SKILLS_DIR"

var userHomeDir = os.UserHomeDir

// rootDir returns Claude Code's own root config directory under home,
// ~/.claude — the directory both SkillsDir and Available key off of.
func rootDir(home string) string {
	return filepath.Join(home, ".claude")
}

// SkillsDir returns the directory Claude Code loads skills from,
// ~/.claude/skills.
func SkillsDir() (string, error) {
	if dir := os.Getenv(SkillsDirEnv); dir != "" {
		return dir, nil
	}
	home, err := userHomeDir()
	if err != nil {
		return "", fmt.Errorf("claudecode: locating home directory: %w", err)
	}
	return filepath.Join(rootDir(home), "skills"), nil
}

// Available reports whether this harness appears to be present on this
// host: its root config directory (~/.claude) exists, or — when
// SkillsDirEnv overrides SkillsDir — that override directory exists. The
// root, not the skills subdirectory, is the signal: a fresh harness
// install may not have created its skills directory yet. A home-dir
// lookup failure counts as not available rather than an error — the bare
// install run treats an unavailable harness as a skip, and a probe should
// never abort that run.
func Available() bool {
	dir := os.Getenv(SkillsDirEnv)
	if dir == "" {
		home, err := userHomeDir()
		if err != nil {
			return false
		}
		dir = rootDir(home)
	}
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

// InstallDir returns the directory the generated skill is written to and
// read back from: SkillsDir()/toolname.
func InstallDir() (string, error) {
	dir, err := SkillsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, skillName), nil
}

// Generate renders m into the generated tree's content, keyed by path
// relative to InstallDir(). It filters m's verbs to the plumbing subset
// (harness.Projectable, C4.3) before rendering — the manifest's own full
// verb list is never projected as-is.
func Generate(m manifest.Manifest) (map[string][]byte, error) {
	verbs := harness.Projectable(m.Verbs)

	judgment, err := asset.Resolve(judgmentAsset)
	if err != nil {
		return nil, fmt.Errorf("claudecode: resolving judgment template: %w", err)
	}
	guidance, err := asset.Resolve(guidanceAsset)
	if err != nil {
		return nil, fmt.Errorf("claudecode: resolving agent-guidance partial: %w", err)
	}

	pluginJSON, err := renderPluginJSON(m)
	if err != nil {
		return nil, fmt.Errorf("claudecode: rendering plugin.json: %w", err)
	}

	return map[string][]byte{
		"SKILL.md":                   renderSkillMD(m, verbs, judgment.Bytes(), guidance.Bytes()),
		".claude-plugin/plugin.json": pluginJSON,
	}, nil
}

func renderSkillMD(m manifest.Manifest, verbs []manifest.Verb, judgment, guidance []byte) []byte {
	var b strings.Builder

	fmt.Fprintf(&b, "---\n")
	fmt.Fprintf(&b, "name: %s\n", skillName)
	fmt.Fprintf(&b, "description: %s\n", skillDescription)
	fmt.Fprintf(&b, "---\n\n")

	fmt.Fprintf(&b, "# %s\n\n", m.Tool.Name)
	fmt.Fprintf(&b, "Generated from %s %s (schema %d) — never hand-edit; re-run `%s install %s` after upgrading.\n\n",
		m.Tool.Name, m.Tool.Version, m.SchemaVersion, m.Tool.Name, Name)

	b.Write(judgment)
	fmt.Fprintf(&b, "\n\n")

	fmt.Fprintf(&b, "## Verbs\n\n")
	if len(verbs) == 0 {
		fmt.Fprintf(&b, "(none registered yet)\n\n")
	} else {
		fmt.Fprintf(&b, "| Verb | Description |\n|---|---|\n")
		for _, v := range verbs {
			desc := v.Description
			if desc == "" {
				desc = "(no description)"
			}
			invocation := strings.TrimSpace(m.Tool.Name + " " + v.Name + " " + v.Usage)
			fmt.Fprintf(&b, "| `%s` | %s |\n", invocation, desc)
		}
		fmt.Fprintf(&b, "\n")
	}

	b.Write(guidance)
	fmt.Fprintf(&b, "\n")

	return []byte(b.String())
}

// pluginManifest is the minimal .claude-plugin/plugin.json shape (the
// skills-dir-as-plugin mechanism). Hooks/agents/MCP sections are left
// absent until a real need exists — the mechanism costs nothing to leave
// empty. A lifecycle hook, when one is earned, rides here as the smallest
// shim that invokes the binary (C4.10).
type pluginManifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

func renderPluginJSON(m manifest.Manifest) ([]byte, error) {
	p := pluginManifest{
		Name:        skillName,
		Description: fmt.Sprintf("%s's own generated plumbing-verb skill.", m.Tool.Name),
		Version:     m.Tool.Version,
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
