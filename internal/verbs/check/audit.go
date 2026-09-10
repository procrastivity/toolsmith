package check

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	toolmanifest "github.com/procrastivity/toolsmith/internal/manifest"
)

// Finding is one reported condition: a clause tag ("C1.1", "C3.4", ...)
// and its human-readable message. Audit appends these in exactly the
// order contrib/check-contract's find_it calls fire (port spec §3.1,
// §4.1) — never sorted, never grouped by clause.
type Finding struct {
	Clause  string
	Message string
}

// Audit runs every mechanical check from CONTRACT.md against repo (an
// absolute path) and returns its findings in exactly the source order
// contrib/check-contract emits them (port spec §4.1), plus any
// stderr-only notes (port spec §5.1) — today, only the
// multiple-cmd/-entries note (§4.1 step 1) can fire.
//
// Like the oracle, Audit never hard-fails on a missing or unreadable
// file, an absent go/git binary, or a failing subprocess — every one of
// those input shapes folds into a Finding (or, for the git-gated check,
// silently into no finding at all), exactly as contrib/check-contract's
// `set -e`-exempted `grep ... || find_it ...` pattern does. There is
// nothing left for Audit itself to report as a Go error.
func Audit(repo string) (findings []Finding, notes []string) {
	a := &auditor{repo: repo}

	tool := a.toolName()    // §4.1 step 1
	a.cgoEnabled()          // §4.1 step 2
	a.flakeAndEnvrc()       // §4.1 step 3
	a.forbidigo()           // §4.1 step 4
	a.manifestVerb(tool)    // §4.1 step 5
	a.tagNamespace()        // §4.1 step 6, §9.6
	a.changelogTracked()    // §4.1 step 7
	a.workflows()           // §4.1 step 8, §7
	a.conventionalCommits() // §4.1 step 9
	a.golangciSchema()      // §4.1 step 10
	a.readme()              // §4.1 step 11

	return a.findings, a.notes
}

// auditor accumulates findings and stderr notes across one audit pass —
// the Go equivalent of the oracle's mutable `findings` counter (port spec
// §3.1) plus the two lines it writes straight to stderr mid-script.
type auditor struct {
	repo     string
	findings []Finding
	notes    []string
}

func (a *auditor) find(clause, message string) {
	a.findings = append(a.findings, Finding{Clause: clause, Message: message})
}

func (a *auditor) note(message string) {
	a.notes = append(a.notes, message)
}

// isFile mirrors contrib/check-contract's has_file: `[[ -f "$repo/$1" ]]`.
// Any stat failure — not found, permission denied, a parent that isn't a
// directory — reads as false, exactly like the bash test does.
func (a *auditor) isFile(rel string) bool {
	info, err := os.Stat(filepath.Join(a.repo, rel))
	return err == nil && info.Mode().IsRegular()
}

func (a *auditor) isDir(rel string) bool {
	info, err := os.Stat(filepath.Join(a.repo, rel))
	return err == nil && info.IsDir()
}

// read returns rel's contents, or nil if it can't be read. A nil slice
// contains no fixed string and matches no regexp, so a file that passed
// isFile a moment ago but became unreadable before this read (or simply
// can't be read for any other reason) falls through to the same
// "pattern not found" finding a failing `grep` would produce under the
// oracle's `grep ... || find_it ...` idiom — not a Go error.
func (a *auditor) read(rel string) []byte {
	data, err := os.ReadFile(filepath.Join(a.repo, rel))
	if err != nil {
		return nil
	}
	return data
}

// toolName reproduces the oracle's tool-name discovery (port spec §4.1
// step 1). Exactly one cmd/ entry names the tool silently; two or more
// picks the alphabetically first (after sort) and notes the rest to
// stderr; zero (cmd/ missing, or present but empty — indistinguishable)
// leaves the tool name empty and raises C1.2.
func (a *auditor) toolName() string {
	var cmds []string
	if a.isDir("cmd") {
		if entries, err := os.ReadDir(filepath.Join(a.repo, "cmd")); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					cmds = append(cmds, e.Name())
				}
			}
		}
		// port spec §7: sort.Strings is a byte-wise sort. It agrees with
		// the oracle's locale-dependent `find ... | sort` for every name
		// the corpus and contrib/new-tool.sh produce
		// (^[a-z][a-z0-9]*$), but check-contract places no format
		// constraint on cmd/ subdirectory names in the repo under audit
		// — a non-ASCII cmd/ entry could sort differently under a
		// non-C locale. Unexercised by the corpus (port spec §7, §10).
		sort.Strings(cmds)
	}

	var tool string
	switch len(cmds) {
	case 0:
		// tool stays "" — cmd/ missing and cmd/ present-but-empty are
		// indistinguishable in the output, exactly like the oracle.
	case 1:
		tool = cmds[0]
	default:
		tool = cmds[0]
		a.note(fmt.Sprintf("note: multiple cmd/ entries (%s); auditing as \"%s\"", strings.Join(cmds, " "), tool))
	}
	if tool == "" {
		a.find("C1.2", "no cmd/<tool> main package found")
	}
	return tool
}

// cgoFlakePattern is the oracle's C1.1 flake pattern
// `CGO_ENABLED *= *0|env\.CGO_ENABLED *= *0` (port spec §4.1 step 2).
var cgoFlakePattern = regexp.MustCompile(`CGO_ENABLED *= *0|env\.CGO_ENABLED *= *0`)

// cgoEnabled reproduces the oracle's C1.1 checks (port spec §4.1 step 2).
// Two independent sub-checks: Makefile (missing file is its own finding;
// present but lacking the string is a different one), then, only if
// flake.nix exists, its CGO_ENABLED pattern. A missing flake.nix produces
// no C1.1 finding at all here — that gap is covered, under a different
// clause label, by flakeAndEnvrc below (§4.1 step 2/3).
func (a *auditor) cgoEnabled() {
	if a.isFile("Makefile") {
		if !bytes.Contains(a.read("Makefile"), []byte("CGO_ENABLED=0")) {
			a.find("C1.1", "Makefile does not set CGO_ENABLED=0")
		}
	} else {
		a.find("C1.1", "no Makefile")
	}
	if a.isFile("flake.nix") {
		if !cgoFlakePattern.Match(a.read("flake.nix")) {
			a.find("C1.1", "flake.nix does not set CGO_ENABLED = 0")
		}
	}
}

// flakeAndEnvrc reproduces the oracle's C1.6 checks (port spec §4.1
// step 3). flake.nix's existence is checked again here, this time with
// an else — "no flake.nix" is a C1.6 finding, not C1.1. The
// postInstall/share/ sub-check only runs when both flake.nix and an
// assets/ dir exist.
func (a *auditor) flakeAndEnvrc() {
	hasFlake := a.isFile("flake.nix")
	if !hasFlake {
		a.find("C1.6", "no flake.nix")
	}
	if a.isFile(".envrc") {
		if !bytes.Contains(a.read(".envrc"), []byte("use flake")) {
			a.find("C1.6", ".envrc does not 'use flake'")
		}
	} else {
		a.find("C1.6", "no .envrc")
	}
	if hasFlake && a.isDir("assets") {
		if !bytes.Contains(a.read("flake.nix"), []byte("share/")) {
			a.find("C1.6", "flake.nix postInstall does not install assets into $out/share")
		}
	}
}

// forbidigo reproduces the oracle's C2.1 checks (port spec §4.1 step 4).
func (a *auditor) forbidigo() {
	if !a.isFile(".golangci.yml") {
		a.find("C2.1", "no .golangci.yml")
		return
	}
	data := a.read(".golangci.yml")
	if !bytes.Contains(data, []byte("forbidigo")) {
		a.find("C2.1", ".golangci.yml does not enable forbidigo")
	}
	// port spec §7: `grep -qF 'fmt\.Print'` is a fixed-string match for
	// the ten literal bytes f-m-t-\-.-P-r-i-n-t, backslash included. It
	// works only because forbidigo's own ban patterns are written as
	// regex source inside YAML strings (e.g. 'fmt\.Print('), so that
	// exact byte sequence genuinely appears; a differently-spelled ban
	// on the same function would trip a false C2.1 finding here despite
	// correctly banning fmt.Print* (this is an accident with a
	// consumer, not a spec — reproduced as measured).
	if !bytes.Contains(data, []byte(`fmt\.Print`)) {
		a.find("C2.1", ".golangci.yml forbidigo does not ban fmt.Print*")
	}
}

// manifestVerb reproduces the oracle's C3.1/C3.4/C3.6 checks (port spec
// §4.1 step 5). Guarded on a non-empty tool name, `go` on PATH, and
// go.mod present; any failure of that guard, or a non-zero exit from the
// manifest run itself, collapses to the single C3.1 finding the oracle
// can't further distinguish (port spec §4.1 step 1, §6). On a successful
// run, checkManifestDoc applies the C3.1/C3.4/C3.6 field checks (§9.1).
func (a *auditor) manifestVerb(tool string) {
	if tool == "" || !goAvailable() || !a.isFile("go.mod") {
		a.find("C3.1", "cannot run the manifest verb (need go, go.mod, and cmd/<tool>)")
		return
	}
	out, err := runManifestJSON(a.repo, tool)
	if err != nil {
		a.find("C3.1", fmt.Sprintf("`%s manifest --json` failed or is not implemented", tool))
		return
	}
	a.findings = append(a.findings, checkManifestDoc(out)...)
}

func goAvailable() bool {
	_, err := exec.LookPath("go")
	return err == nil
}

// runManifestJSON runs `go run ./cmd/<tool> manifest --json` with the
// audited repo as the working directory and CGO_ENABLED=0 forced in the
// environment (port spec §6), discarding the
// child's stderr exactly as the oracle's `2>/dev/null` does — a crash, a
// compile failure, and "go isn't on PATH but go.mod exists" all surface
// identically to the caller as a non-nil error.
func runManifestJSON(repo, tool string) ([]byte, error) {
	cmd := exec.Command("go", "run", "./cmd/"+tool, "manifest", "--json")
	cmd.Dir = repo

	env := make([]string, 0, len(os.Environ())+1)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "CGO_ENABLED=") {
			continue
		}
		env = append(env, e)
	}
	cmd.Env = append(env, "CGO_ENABLED=0")

	return cmd.Output()
}

// checkManifestDoc applies the C3.1/C3.4/C3.6 field checks to a
// `manifest --json` document's raw bytes.
//
// port spec §9.1: parse, don't grep. The oracle greps the raw bytes for
// `"manifest_digest":"sha256:` and `"contract":"toolsmith/` — both
// patterns assume compact JSON with no space after the colon, so a
// pretty-printed manifest trips a false finding even when the fields are
// present and correct. This instead parses into internal/manifest.Manifest
// and checks the fields' values directly, so formatting cannot produce a
// false finding: SchemaVersion's zero value is indistinguishable from
// "absent" (the field carries no omitempty and every real schema starts
// at 1, so zero is exactly the "carries no schemaVersion" case), and the
// digest/contract checks require the same "sha256:" / "toolsmith/"
// prefixes the oracle's grep patterns required. A document that isn't
// valid JSON at all reports all three findings, since none of the three
// fields can be verified.
func checkManifestDoc(raw []byte) []Finding {
	var findings []Finding
	var m toolmanifest.Manifest
	parseErr := json.Unmarshal(raw, &m)

	if parseErr != nil || m.SchemaVersion == 0 {
		findings = append(findings, Finding{"C3.1", "manifest --json carries no schemaVersion"})
	}
	if parseErr != nil || !strings.HasPrefix(m.ManifestDigest, "sha256:") {
		findings = append(findings, Finding{"C3.4", "manifest --json carries no manifest_digest"})
	}
	if parseErr != nil || !strings.HasPrefix(m.Contract, "toolsmith/") {
		findings = append(findings, Finding{"C3.6", "manifest --json declares no toolsmith contract version"})
	}
	return findings
}

// cliffTagPattern is the oracle's C6.2 tag pattern
// `tag_pattern *= *"v\[0-9\]\*"` (port spec §4.1 step 6; a BRE where
// every metacharacter around the literal "[0-9]*" is escaped to match it
// literally; only the surrounding " *" runs of spaces are genuinely
// variable-width).
var cliffTagPattern = regexp.MustCompile(`tag_pattern *= *"v\[0-9\]\*"`)

// tagNamespace reproduces the oracle's C6.2/C6.3 checks (port spec §4.1
// step 6), the mislabel and coverage hole included.
//
// This is still the oracle's behavior (port spec §9.6, parity-divergences.md
// R1), pinned by the probe-c62-mislabel and probe-c62-hole goldens
// (internal/verbs/check/golden_test.go, testdata/golden/check/). Two
// consequences of the oracle's guard being `Makefile && cliff.toml` while
// its else only re-tests cliff.toml: a repo with a Makefile and no
// cliff.toml reports the real C6.2 condition under the C6.3 label
// (mislabel, repro A); a repo with a cliff.toml and no Makefile reports
// nothing from this whole block, so the C6.2 --match/tag_pattern
// agreement goes silently unaudited (coverage hole, repro B) — has_file
// cliff.toml is true, so find_it is never reached. Correcting the label
// and closing the hole is a deliberate output change: it updates both
// goldens above and the R1 entry in docs/binary/parity-divergences.md.
func (a *auditor) tagNamespace() {
	hasMakefile := a.isFile("Makefile")
	hasCliff := a.isFile("cliff.toml")
	if hasMakefile && hasCliff {
		if !bytes.Contains(a.read("Makefile"), []byte(`--match 'v[0-9]*'`)) {
			a.find("C6.2", "Makefile git describe lacks --match 'v[0-9]*'")
		}
		if !cliffTagPattern.Match(a.read("cliff.toml")) {
			a.find("C6.2", `cliff.toml tag_pattern is not "v[0-9]*"`)
		}
		return
	}
	if !hasCliff {
		a.find("C6.3", "no cliff.toml (changelog is not derivable from tags)")
	}
}

// changelogTracked reproduces the oracle's C6.3 CHANGELOG.md check (port
// spec §4.1 step 7). It only runs inside a git working tree; a non-git
// directory skips it silently,
// exactly like `git -C "$repo" rev-parse --git-dir` gates the oracle's
// version. A missing git binary, or any other failure of either
// subprocess, is folded into "skip silently" the same way — the oracle
// has no other fallback for it either.
func (a *auditor) changelogTracked() {
	if err := exec.Command("git", "-C", a.repo, "rev-parse", "--git-dir").Run(); err != nil {
		return
	}
	out, err := exec.Command("git", "-C", a.repo, "ls-files", "CHANGELOG.md").Output()
	if err != nil {
		return
	}
	if len(bytes.TrimSpace(out)) > 0 {
		a.find("C6.3", "CHANGELOG.md is tracked; the contract generates it per release instead")
	}
}

// usesLinePattern is the oracle's C6.5 uses-line filter
// `^\s*-? *uses:` (port spec §4.1 step 8): leading whitespace, an
// optional single dash, more spaces, then the literal "uses:"
// immediately. A '#' anywhere in that prefix (a commented-out uses:
// line) breaks the match, so that line is never seen at all — neither
// flagged nor exempted, just never reached (port spec §7).
var usesLinePattern = regexp.MustCompile(`^\s*-? *uses:`)

// shaPattern is the 40-hex-character full-SHA test each stripped ref is
// checked against (port spec §4.1 step 8).
var shaPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// workflows reproduces the oracle's C6.5 workflow checks (port spec §4.1
// step 8): `for wf in ci.yml release.yml`, in that order, each
// independently checked for nix-develop framing and then walked
// line-by-line for SHA-pinning.
func (a *auditor) workflows() {
	for _, wf := range []string{"ci.yml", "release.yml"} {
		rel := filepath.Join(".github", "workflows", wf)
		if !a.isFile(rel) {
			a.find("C6.5", "no .github/workflows/"+wf)
			continue
		}
		data := a.read(rel)
		if !bytes.Contains(data, []byte("nix develop --command")) {
			a.find("C6.5", wf+" does not run through 'nix develop --command'")
		}
		a.findings = append(a.findings, shaPinFindings(wf, data)...)
	}
}

// shaPinFindings reproduces the oracle's per-line string surgery over
// every uses: line (port spec §4.1 step 8, §7):
//   - ref is the text after the LAST '@' in the line — unchanged if no
//     '@' is present at all (`${line##*@}` returns its operand
//     untouched when the pattern doesn't match, so an unpinned local
//     composite action's whole line, indentation included, becomes ref;
//     stripping from the first space in that then leaves ref empty,
//     which never matches the SHA pattern — "not SHA-pinned" fires for
//     the right conclusion but the wrong mechanical reason);
//   - ref is then narrowed to its first whitespace-delimited token,
//     splitting only on a literal space character, not general
//     whitespace (`${ref%% *}`);
//   - trimmed is the line with leading whitespace stripped, for the
//     message text only — it is computed independently and never feeds
//     back into ref.
//
// One deliberate divergence here (port spec §9.7): bufio.ScanLines drops
// a line's trailing carriage return, and the oracle's `read -r` keeps it.
// On a CRLF workflow file the oracle therefore tests `<40 hex>\r` against
// its SHA pattern, fails, and reports a correctly pinned action as not
// SHA-pinned. That is a false finding, so the port does not reproduce it —
// unlike the C6.2 mislabel of §9.6, which is reproduced because the
// condition it reports is real. Do not "restore" the carriage return.
func shaPinFindings(wf string, data []byte) []Finding {
	var findings []Finding
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if !usesLinePattern.MatchString(line) {
			continue
		}
		ref := line
		if i := strings.LastIndexByte(line, '@'); i >= 0 {
			ref = line[i+1:]
		}
		if i := strings.IndexByte(ref, ' '); i >= 0 {
			ref = ref[:i]
		}
		trimmed := strings.TrimLeft(line, " \t\n\r\f\v")
		if !shaPattern.MatchString(ref) {
			findings = append(findings, Finding{"C6.5", fmt.Sprintf("%s: action not SHA-pinned: %s", wf, trimmed)})
		}
	}
	return findings
}

// hooksTargetPattern is the oracle's C6.6 hooks-target pattern
// `^hooks:` (port spec §4.1 step 9), matched against any line in the
// Makefile (grep -E, no -x — multiline anchors).
var hooksTargetPattern = regexp.MustCompile(`(?m)^hooks:`)

// conventionalCommits reproduces the oracle's C6.6 checks (port spec
// §4.1 step 9).
//
// The final sub-check is gated on the Makefile containing a `^hooks:`
// line, but the `--hook-type commit-msg` grep it guards then searches
// the WHOLE Makefile, not that target's own recipe body — despite the
// oracle's own comment describing it as scoped. Reproduced exactly as
// written (port spec §7): a Makefile with `--hook-type commit-msg`
// anywhere in it and a `hooks:` line anywhere else both satisfy this
// check, even if the two are unrelated.
func (a *auditor) conventionalCommits() {
	if !a.isFile("contrib/check-commit-msg") {
		a.find("C6.6", "no contrib/check-commit-msg hook")
	}
	if a.isFile(".pre-commit-config.yaml") {
		if !bytes.Contains(a.read(".pre-commit-config.yaml"), []byte("commit-msg")) {
			a.find("C6.6", ".pre-commit-config.yaml has no commit-msg stage hook")
		}
	} else {
		a.find("C6.6", "no .pre-commit-config.yaml")
	}
	if a.isFile("Makefile") {
		mk := a.read("Makefile")
		if hooksTargetPattern.Match(mk) && !bytes.Contains(mk, []byte("--hook-type commit-msg")) {
			a.find("C6.6", "make hooks does not install the commit-msg stage")
		}
	}
}

// golangciSchema reproduces the oracle's C6.7 checks (port spec §4.1
// step 10). Only runs if .golangci.yml exists — already established by
// forbidigo above, but re-checked here exactly as the oracle re-checks
// it.
func (a *auditor) golangciSchema() {
	if !a.isFile(".golangci.yml") {
		return
	}
	data := a.read(".golangci.yml")
	if !bytes.Contains(data, []byte(`version: "2"`)) {
		a.find("C6.7", ".golangci.yml is not schema version 2")
	}
	if !bytes.Contains(data, []byte("default: none")) {
		a.find("C6.7", ".golangci.yml does not use 'default: none'")
	}
}

// readme reproduces the oracle's C7.5 README check (port spec §4.1
// step 11).
func (a *auditor) readme() {
	if !a.isFile("README.md") {
		a.find("C7.5", "no README.md")
	}
}
