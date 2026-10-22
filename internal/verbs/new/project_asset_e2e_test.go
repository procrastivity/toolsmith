package new_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	projectAssetName = "templates/skills/claude-code/judgment.md"
	projectFlowName  = "flows/release.md"
)

type generatedResult struct {
	stdout   string
	stderr   string
	exitCode int
}

type resolvedAsset struct {
	Source  string `json:"source"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

// TestGeneratedProjectAssetsAreRuntimeOnly exercises project-root selection
// through one built generated binary. The reader command exists only in the
// disposable module: it routes the real root flag context into the public
// asset.ResolveIn API without adding a production verb or test API.
func TestGeneratedProjectAssetsAreRuntimeOnly(t *testing.T) {
	const (
		toolName = "assetdemo"
		module   = "example.com/assetdemo"
	)

	parent := t.TempDir()
	moduleRoot := filepath.Join(parent, toolName)
	projectRootA := moduleRoot // Generated without Git metadata.
	projectRootB := filepath.Join(parent, "gitfile-root")
	nestedRoot := filepath.Join(projectRootA, "nested")
	outerRoot := parent

	xdgRoot := filepath.Join(parent, "xdg")
	skillsDir := filepath.Join(parent, "skills")
	env := hermeticEnv(t, map[string]string{
		"XDG_CONFIG_HOME":             xdgRoot,
		"ASSETDEMO_CLAUDE_SKILLS_DIR": skillsDir,
		// Deliberately ignored: only the explicit root flag selects a project.
		"ASSETDEMO_PROJECT_ROOT": projectRootA,
	})

	_, stderr, exitCode := runBin(t, parent, env,
		"new", toolName, "--dir", toolName, "--module", module, "--no-git")
	if exitCode != 0 {
		t.Fatalf("toolsmith new: exit=%d stderr=%q", exitCode, stderr)
	}
	if _, err := os.Lstat(filepath.Join(projectRootA, ".git")); !os.IsNotExist(err) {
		t.Fatalf("generated root A unexpectedly has .git metadata: err=%v", err)
	}

	// Add one existing shipped template and one known shipped flow before
	// building, so the generated binary embeds both names and their defaults.
	writeFixtureFile(t, filepath.Join(moduleRoot, "assets", filepath.FromSlash(projectAssetName)), []byte("embedded judgment fixture\n"))
	writeFixtureFile(t, filepath.Join(moduleRoot, "assets", filepath.FromSlash(projectFlowName)), []byte("embedded release flow\n"))

	// Root A is a non-Git project beneath an inert outer .git directory
	// marker. nestedRoot and B have inert .git-file contents (their gitdir
	// targets are not initialized repositories or worktrees). Explicit
	// selection must use exactly that root, not discover or merge nearby paths.
	writeProjectAsset(t, outerRoot, toolName, projectAssetName, "outer unrelated asset\n")
	if err := os.MkdirAll(nestedRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, filepath.Join(nestedRoot, ".git"), []byte("gitdir: ../.git/worktrees/nested\n"))
	writeProjectAsset(t, nestedRoot, toolName, projectAssetName, "nested .git-file marker asset\n")
	if err := os.MkdirAll(projectRootB, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, filepath.Join(projectRootB, ".git"), []byte("gitdir: ../.git/worktrees/assetdemo\n"))
	writeProjectAsset(t, projectRootB, toolName, projectAssetName, "B .git-file marker asset\n")
	projectAssetA := writeProjectAsset(t, projectRootA, toolName, projectAssetName, "A project asset v1\n")
	projectFlowA := writeProjectAsset(t, projectRootA, toolName, projectFlowName, "A project flow\n")
	canonicalRootA := canonicalFixtureRoot(t, projectRootA)
	canonicalRootB := canonicalFixtureRoot(t, projectRootB)
	canonicalNestedRoot := canonicalFixtureRoot(t, nestedRoot)

	writeFixtureFile(t,
		filepath.Join(xdgRoot, toolName, filepath.FromSlash(projectAssetName)),
		[]byte("XDG fallback judgment\n"))
	installProjectAssetReader(t, moduleRoot, module)

	generatedBinary := filepath.Join(moduleRoot, "bin", toolName)
	if err := os.MkdirAll(filepath.Dir(generatedBinary), 0o755); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", generatedBinary, "./cmd/"+toolName)
	build.Dir = moduleRoot
	build.Env = env
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building generated binary: %v\n%s", err, out)
	}

	// Compile and run the generated module's own suite as well; the lifecycle
	// proof below then uses this same binary throughout its runtime scenarios.
	generatedTests := exec.Command("go", "test", "./...")
	generatedTests.Dir = moduleRoot
	generatedTests.Env = env
	if out, err := generatedTests.CombinedOutput(); err != nil {
		t.Fatalf("generated module go test ./...: %v\n%s", err, out)
	}
	if err := os.Mkdir(filepath.Join(outerRoot, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	manifestBefore := runGenerated(t, generatedBinary, projectRootA, env, "manifest", "--json")
	requireGeneratedSuccess(t, "manifest baseline", manifestBefore)
	var manifest struct {
		ManifestDigest string `json:"manifest_digest"`
	}
	if err := json.Unmarshal([]byte(manifestBefore.stdout), &manifest); err != nil || manifest.ManifestDigest == "" {
		t.Fatalf("manifest baseline lacks a digest: err=%v stdout=%q", err, manifestBefore.stdout)
	}
	for _, selectedRoot := range []string{projectRootA, projectRootB} {
		assertProjectManifestUnchanged(t, generatedBinary, projectRootA, env, selectedRoot, manifestBefore, "before live edit")
	}

	installed := runGenerated(t, generatedBinary, projectRootA, env,
		"--project-root", projectRootA, "--json", "install", "claude-code")
	requireGeneratedSuccess(t, "initial install", installed)
	assertInstallStatus(t, installed.stdout, "installed")
	installedTree := loadTree(t, filepath.Join(skillsDir, toolName))
	stampName := "." + toolName + "-manifest-stamp.json"
	stamp, ok := installedTree[stampName]
	if !ok {
		t.Fatal("initial install did not write its manifest stamp")
	}
	skill, ok := installedTree["SKILL.md"]
	if !ok || !bytes.Contains(skill.data, []byte("XDG fallback judgment")) {
		t.Fatalf("installed skill did not use the stable XDG asset fallback: present=%v", ok)
	}
	if bytes.Contains(skill.data, []byte("A project asset")) {
		t.Fatal("install consumed a project-local runtime asset")
	}
	assertFileBytes(t, projectAssetA, []byte("A project asset v1\n"))
	assertFileBytes(t, projectFlowA, []byte("A project flow\n"))

	// Root invocation uses a relative selection from A. The nested absolute
	// selection must still choose A over the nested .git-file marker.
	got := readGeneratedAsset(t, generatedBinary, projectRootA, env,
		"--project-root", ".", "asset-fixture", "--json", projectAssetName)
	assertResolvedAsset(t, got, "project", projectAssetPath(canonicalRootA, toolName, projectAssetName), "A project asset v1\n")
	got = readGeneratedAsset(t, generatedBinary, nestedRoot, env,
		"asset-fixture", "--project-root", projectRootA, "--json", projectAssetName)
	assertResolvedAsset(t, got, "project", projectAssetPath(canonicalRootA, toolName, projectAssetName), "A project asset v1\n")
	got = readGeneratedAsset(t, generatedBinary, nestedRoot, env,
		"--project-root", "..", "asset-fixture", "--json", projectAssetName)
	assertResolvedAsset(t, got, "project", projectAssetPath(canonicalRootA, toolName, projectAssetName), "A project asset v1\n")
	got = readGeneratedAsset(t, generatedBinary, nestedRoot, env,
		"asset-fixture", "--project-root", ".", "--json", projectAssetName)
	assertResolvedAsset(t, got, "project", projectAssetPath(canonicalNestedRoot, toolName, projectAssetName), "nested .git-file marker asset\n")
	got = readGeneratedAsset(t, generatedBinary, nestedRoot, env,
		"--project-root", projectRootB, "asset-fixture", "--json", projectAssetName)
	assertResolvedAsset(t, got, "project", projectAssetPath(canonicalRootB, toolName, projectAssetName), "B .git-file marker asset\n")

	// The added shipped flow is project-overridable too; B has no local copy,
	// so the same known name reaches the embedded default there.
	got = readGeneratedAsset(t, generatedBinary, projectRootA, env,
		"--project-root", projectRootA, "asset-fixture", "--json", projectFlowName)
	assertResolvedAsset(t, got, "project", projectAssetPath(canonicalRootA, toolName, projectFlowName), "A project flow\n")
	got = readGeneratedAsset(t, generatedBinary, projectRootA, env,
		"--project-root", projectRootB, "asset-fixture", "--json", projectFlowName)
	assertResolvedAsset(t, got, "embedded", "", "embedded release flow\n")

	// A live edit is visible immediately through the same process binary;
	// verbose provenance carries the canonical project path.
	writeFixtureFile(t, projectAssetA, []byte("A project asset v2\n"))
	got, verboseStderr := readGeneratedAssetWithStderr(t, generatedBinary, nestedRoot, env,
		"--project-root", projectRootA, "--verbose", "asset-fixture", "--json", projectAssetName)
	assertResolvedAsset(t, got, "project", projectAssetPath(canonicalRootA, toolName, projectAssetName), "A project asset v2\n")
	if !strings.Contains(verboseStderr, projectAssetPath(canonicalRootA, toolName, projectAssetName)) {
		t.Fatalf("verbose provenance %q does not name the canonical project path", verboseStderr)
	}
	for _, selectedRoot := range []string{projectRootA, projectRootB} {
		assertProjectManifestUnchanged(t, generatedBinary, nestedRoot, env, selectedRoot, manifestBefore, "after live edit")
	}

	// Neither cwd nor the similarly named environment variable opts in to
	// project selection: no flag keeps the legacy XDG -> default -> embedded
	// resolver chain even while a project asset is present in A.
	got = readGeneratedAsset(t, generatedBinary, nestedRoot, env, "asset-fixture", "--json", projectAssetName)
	assertResolvedAsset(t, got, "override",
		filepath.Join(xdgRoot, toolName, filepath.FromSlash(projectAssetName)), "XDG fallback judgment\n")

	// A current reinstall under an explicit project root leaves both its
	// installed bytes/stamp and project-owned bytes untouched.
	currentInstall := runGenerated(t, generatedBinary, nestedRoot, env,
		"--project-root", projectRootA, "--json", "install", "claude-code")
	requireGeneratedSuccess(t, "current reinstall", currentInstall)
	assertInstallStatus(t, currentInstall.stdout, "current")
	installedAfterReinstall := loadTree(t, filepath.Join(skillsDir, toolName))
	compareTrees(t, installedTree, installedAfterReinstall)
	if !bytes.Equal(stamp.data, installedAfterReinstall[stampName].data) {
		t.Fatal("manifest stamp changed across current reinstall")
	}
	assertFileBytes(t, projectAssetA, []byte("A project asset v2\n"))
	assertFileBytes(t, projectFlowA, []byte("A project flow\n"))

	if err := os.Remove(projectAssetA); err != nil {
		t.Fatal(err)
	}
	got = readGeneratedAsset(t, generatedBinary, nestedRoot, env,
		"--project-root", projectRootA, "asset-fixture", "--json", projectAssetName)
	assertResolvedAsset(t, got, "override",
		filepath.Join(xdgRoot, toolName, filepath.FromSlash(projectAssetName)), "XDG fallback judgment\n")

	// These invalid roots would make a runtime ResolveIn request refuse. Admin
	// verbs must not inspect them: manifest identity and the installed tree
	// stay fixed, while install and doctor continue to report current.
	nonexistentRoot := filepath.Join(parent, "does-not-exist")
	unsafeRoot := filepath.Join(parent, "unsafe-root")
	unsafeAsset := projectAssetPath(unsafeRoot, toolName, projectAssetName)
	if err := os.MkdirAll(unsafeAsset, 0o755); err != nil {
		t.Fatal(err) // A directory at the asset leaf is deliberately non-regular.
	}
	for _, selectedRoot := range []string{nonexistentRoot, unsafeRoot} {
		manifestResult := runGenerated(t, generatedBinary, nestedRoot, env,
			"--project-root", selectedRoot, "manifest", "--json")
		requireGeneratedSuccess(t, "manifest with selected root "+selectedRoot, manifestResult)
		if manifestResult.stdout != manifestBefore.stdout {
			t.Errorf("manifest changed with runtime root %q", selectedRoot)
		}

		installResult := runGenerated(t, generatedBinary, nestedRoot, env,
			"--project-root", selectedRoot, "--json", "install", "claude-code")
		requireGeneratedSuccess(t, "install with selected root "+selectedRoot, installResult)
		assertInstallStatus(t, installResult.stdout, "current")

		doctorResult := runGenerated(t, generatedBinary, nestedRoot, env,
			"--project-root", selectedRoot, "doctor", "--json")
		requireGeneratedSuccess(t, "doctor with selected root "+selectedRoot, doctorResult)
		assertDoctorCurrent(t, doctorResult.stdout, filepath.Join(skillsDir, toolName))

		compareTrees(t, installedTree, loadTree(t, filepath.Join(skillsDir, toolName)))
	}
	assertFileBytes(t, projectFlowA, []byte("A project flow\n"))
	assertFileBytes(t, projectAssetPath(projectRootB, toolName, projectAssetName), []byte("B .git-file marker asset\n"))
	assertFileBytes(t, projectAssetPath(nestedRoot, toolName, projectAssetName), []byte("nested .git-file marker asset\n"))
	assertFileBytes(t, projectAssetPath(outerRoot, toolName, projectAssetName), []byte("outer unrelated asset\n"))
}

func installProjectAssetReader(t *testing.T, moduleRoot, module string) {
	t.Helper()
	readerDir := filepath.Join(moduleRoot, "internal", "verbs", "assetfixture")
	if err := os.MkdirAll(readerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	reader := fmt.Sprintf(`package assetfixture

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"%s/internal/asset"
	"%s/internal/cliflags"
	"%s/internal/iostreams"
	"%s/internal/surface"
)

func Command(streams *iostreams.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "asset-fixture <name>",
		Short: "read a shipped asset for generated runtime tests",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cliflags.FromContext(cmd.Context())
			resolved, err := asset.ResolveIn(flags.ProjectRoot, args[0])
			if err != nil {
				return err
			}
			if flags.Verbose {
				if _, err := fmt.Fprintf(streams.Err, "asset-fixture: source=%%s path=%%q\n", resolved.Source, resolved.Path); err != nil {
					return err
				}
			}
			payload := map[string]string{
				"source": resolved.Source.String(),
				"path": resolved.Path,
				"content": string(resolved.Bytes()),
			}
			encoded, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(streams.Out, string(encoded))
			return err
		},
	}
	surface.Annotate(cmd, surface.Plumbing)
	return cmd
}
`, module, module, module, module)
	writeFixtureFile(t, filepath.Join(readerDir, "assetfixture.go"), []byte(reader))

	rootPath := filepath.Join(moduleRoot, "internal", "cli", "root.go")
	root, err := os.ReadFile(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	rootSource := string(root)
	versionImport := fmt.Sprintf("\tversionverb %q", module+"/internal/verbs/version")
	fixtureImport := fmt.Sprintf("\tassetfixtureverb %q", module+"/internal/verbs/assetfixture")
	rootSource = replaceOnce(t, rootSource, versionImport, versionImport+"\n"+fixtureImport)
	versionRegistration := "root.AddCommand(versionverb.Command(streams, build))"
	rootSource = replaceOnce(t, rootSource, versionRegistration,
		versionRegistration+"\n\troot.AddCommand(assetfixtureverb.Command(streams))")
	writeFixtureFile(t, rootPath, []byte(rootSource))
}

func replaceOnce(t *testing.T, source, old, replacement string) string {
	t.Helper()
	if count := strings.Count(source, old); count != 1 {
		t.Fatalf("generated root source has %d occurrences of %q, want exactly one", count, old)
	}
	return strings.Replace(source, old, replacement, 1)
}

func writeProjectAsset(t *testing.T, root, toolName, name, content string) string {
	t.Helper()
	path := projectAssetPath(root, toolName, name)
	writeFixtureFile(t, path, []byte(content))
	return path
}

func projectAssetPath(root, toolName, name string) string {
	return filepath.Join(root, "."+toolName, "assets", filepath.FromSlash(name))
}

func canonicalFixtureRoot(t *testing.T, root string) string {
	t.Helper()
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("canonicalizing generated fixture root %q: %v", root, err)
	}
	return canonical
}

func assertProjectManifestUnchanged(t *testing.T, binary, dir string, env []string, selectedRoot string, baseline generatedResult, stage string) {
	t.Helper()
	result := runGenerated(t, binary, dir, env, "--project-root", selectedRoot, "manifest", "--json")
	requireGeneratedSuccess(t, "manifest with valid project root "+selectedRoot, result)
	if result.stdout != baseline.stdout || result.stderr != baseline.stderr {
		t.Errorf("manifest output changed with valid project root %q %s: stdout equal=%v stderr equal=%v",
			selectedRoot, stage, result.stdout == baseline.stdout, result.stderr == baseline.stderr)
	}
}

func writeFixtureFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGenerated(t *testing.T, binary, dir string, env []string, args ...string) generatedResult {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := generatedResult{stdout: stdout.String(), stderr: stderr.String()}
	if err == nil {
		return result
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.exitCode = exitErr.ExitCode()
		return result
	}
	t.Fatalf("running generated binary %s %v: %v", binary, args, err)
	return result
}

func requireGeneratedSuccess(t *testing.T, label string, result generatedResult) {
	t.Helper()
	if result.exitCode != 0 {
		t.Fatalf("%s: exit=%d stdout=%q stderr=%q", label, result.exitCode, result.stdout, result.stderr)
	}
}

func readGeneratedAsset(t *testing.T, binary, dir string, env []string, args ...string) resolvedAsset {
	t.Helper()
	result, stderr := readGeneratedAssetWithStderr(t, binary, dir, env, args...)
	if stderr != "" {
		t.Fatalf("asset-fixture stderr=%q, want empty", stderr)
	}
	return result
}

func readGeneratedAssetWithStderr(t *testing.T, binary, dir string, env []string, args ...string) (resolvedAsset, string) {
	t.Helper()
	result := runGenerated(t, binary, dir, env, args...)
	requireGeneratedSuccess(t, "asset-fixture "+strings.Join(args, " "), result)
	var resolved resolvedAsset
	if err := json.Unmarshal([]byte(result.stdout), &resolved); err != nil {
		t.Fatalf("asset-fixture stdout is not JSON: %v stdout=%q", err, result.stdout)
	}
	return resolved, result.stderr
}

func assertResolvedAsset(t *testing.T, got resolvedAsset, source, path, content string) {
	t.Helper()
	if got.Source != source || got.Path != path || got.Content != content {
		t.Fatalf("resolved asset = %+v, want source=%q path=%q content=%q", got, source, path, content)
	}
}

func assertInstallStatus(t *testing.T, output, want string) {
	t.Helper()
	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("install output is not JSON: %v stdout=%q", err, output)
	}
	if result.Status != want {
		t.Fatalf("install status=%q, want %q; stdout=%q", result.Status, want, output)
	}
}

func assertDoctorCurrent(t *testing.T, output, wantDir string) {
	t.Helper()
	var result struct {
		Findings []json.RawMessage `json:"findings"`
		Targets  []struct {
			Harness string `json:"harness"`
			Dir     string `json:"dir"`
			State   string `json:"state"`
		} `json:"targets"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("doctor output is not JSON: %v stdout=%q", err, output)
	}
	if len(result.Findings) != 0 || len(result.Targets) != 1 || result.Targets[0].Harness != "claude-code" ||
		result.Targets[0].Dir != wantDir || result.Targets[0].State != "current" {
		t.Fatalf("doctor state = %+v, want no findings and claude-code current at %q", result, wantDir)
	}
}

func assertFileBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s bytes = %q, want %q", path, got, want)
	}
}
