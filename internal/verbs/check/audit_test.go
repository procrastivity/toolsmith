package check

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestAudit_EmptyRepo pins the exact finding order (port spec §4.1) for
// the emptiest input shape the oracle still runs to completion on: a
// directory with nothing in it at all. Every step that has an
// unconditional "missing" finding fires; every step gated on another
// file's presence (the flake.nix CGO_ENABLED sub-check, the postInstall
// sub-check, the git-gated CHANGELOG.md check, the golangci schema
// checks) is silently skipped, exactly as contrib/check-contract skips
// them.
func TestAudit_EmptyRepo(t *testing.T) {
	repo := t.TempDir()

	findings, notes := Audit(repo)

	want := []Finding{
		{"C1.2", "no cmd/<tool> main package found"},
		{"C1.1", "no Makefile"},
		{"C1.6", "no flake.nix"},
		{"C1.6", "no .envrc"},
		{"C2.1", "no .golangci.yml"},
		{"C3.1", "cannot run the manifest verb (need go, go.mod, and cmd/<tool>)"},
		{"C6.2", "no cliff.toml (no tag_pattern for git describe --match to agree with)"},
		{"C6.5", "no .github/workflows/ci.yml"},
		{"C6.5", "no .github/workflows/release.yml"},
		{"C6.6", "no contrib/check-commit-msg hook"},
		{"C6.6", "no .pre-commit-config.yaml"},
		{"C7.5", "no README.md"},
	}
	if !reflect.DeepEqual(findings, want) {
		t.Fatalf("findings =\n%#v\nwant\n%#v", findings, want)
	}
	if len(notes) != 0 {
		t.Fatalf("notes = %v, want none", notes)
	}
}

// TestAudit_MultipleCmdEntries exercises §4.1 step 1's second branch: two
// or more cmd/ entries pick the alphabetically first (after sort) and
// note the rest to stderr — the note must name every entry, space
// separated, in sorted order, and quote the chosen tool name.
func TestAudit_MultipleCmdEntries(t *testing.T) {
	repo := t.TempDir()
	for _, name := range []string{"zed", "alpha", "mid"} {
		if err := os.MkdirAll(filepath.Join(repo, "cmd", name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	_, notes := Audit(repo)

	want := []string{`note: multiple cmd/ entries (alpha mid zed); auditing as "alpha"`}
	if !reflect.DeepEqual(notes, want) {
		t.Fatalf("notes = %v, want %v", notes, want)
	}
}

// TestAudit_TagNamespace pins tagNamespace's C6.2 behavior after step-18
// (toolsmith-binary Stage 8) corrected the ported oracle's mislabel and
// closed its coverage hole (port spec §9.6; parity-divergences.md R1):
// each file is audited independently when present, and a missing
// Makefile reports nothing here because cgoEnabled already owns "no
// Makefile" under C1.1.
func TestAudit_TagNamespace(t *testing.T) {
	noCliff := Finding{"C6.2", "no cliff.toml (no tag_pattern for git describe --match to agree with)"}
	badMakefile := Finding{"C6.2", "Makefile git describe lacks --match 'v[0-9]*'"}
	badCliff := Finding{"C6.2", `cliff.toml tag_pattern is not "v[0-9]*"`}

	tests := []struct {
		name      string
		makefile  *string
		cliffToml *string
		want      []Finding
	}{
		{
			name:      "Makefile only, non-conformant: its own finding plus no-cliff",
			makefile:  strPtr("\tgit describe\n"),
			cliffToml: nil,
			want:      []Finding{badMakefile, noCliff},
		},
		{
			name:      "cliff.toml only, non-conformant: tag_pattern finding only",
			makefile:  nil,
			cliffToml: strPtr("tag_pattern = \"v*\"\n"),
			want:      []Finding{badCliff},
		},
		{
			name:      "cliff.toml only, conformant: no findings",
			makefile:  nil,
			cliffToml: strPtr(`tag_pattern = "v[0-9]*"` + "\n"),
			want:      nil,
		},
		{
			name:      "neither file: only the no-cliff finding",
			makefile:  nil,
			cliffToml: nil,
			want:      []Finding{noCliff},
		},
		{
			name:      "both present, both conformant: no findings",
			makefile:  strPtr("\tgit describe --match 'v[0-9]*'\n"),
			cliffToml: strPtr(`tag_pattern = "v[0-9]*"` + "\n"),
			want:      nil,
		},
		{
			name:      "both present, neither conformant: two findings, Makefile then cliff.toml",
			makefile:  strPtr("\tgit describe\n"),
			cliffToml: strPtr("tag_pattern = \"v*\"\n"),
			want:      []Finding{badMakefile, badCliff},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			if tc.makefile != nil {
				writeFile(t, repo, "Makefile", *tc.makefile)
			}
			if tc.cliffToml != nil {
				writeFile(t, repo, "cliff.toml", *tc.cliffToml)
			}
			a := &auditor{repo: repo}
			a.tagNamespace()
			if !reflect.DeepEqual(a.findings, tc.want) {
				t.Fatalf("findings = %v, want %v", a.findings, tc.want)
			}
		})
	}
}

func strPtr(s string) *string { return &s }

// TestCgoEnabled_CI covers step-37's ci.yml sub-check in isolation:
// cgoEnabled is called directly, with a conformant Makefile and
// flake.nix present in every case, so only the ci.yml content varies.
func TestCgoEnabled_CI(t *testing.T) {
	const conformantMakefile = "CGO_ENABLED=0\n"
	const conformantFlake = "{ env.CGO_ENABLED = 0; }\n"

	tests := []struct {
		name   string
		noFile bool // true: don't create ci.yml at all
		ciYML  string
		want   []Finding
	}{
		{
			name:   "ci.yml absent: no CI finding",
			noFile: true,
		},
		{
			name:  "shell assignment form: CGO_ENABLED=0",
			ciYML: "jobs:\n  build:\n    steps:\n      - run: CGO_ENABLED=0 go build ./...\n",
		},
		{
			name:  "YAML env form, unquoted: CGO_ENABLED: 0",
			ciYML: "jobs:\n  build:\n    env:\n      CGO_ENABLED: 0\n",
		},
		{
			name:  `YAML env form, double-quoted: CGO_ENABLED: "0"`,
			ciYML: "jobs:\n  build:\n    env:\n      CGO_ENABLED: \"0\"\n",
		},
		{
			name:  "YAML env form, single-quoted: CGO_ENABLED: '0'",
			ciYML: "jobs:\n  build:\n    env:\n      CGO_ENABLED: '0'\n",
		},
		{
			name:  "shell assignment set to 1: finding",
			ciYML: "jobs:\n  build:\n    steps:\n      - run: CGO_ENABLED=1 go build ./...\n",
			want:  []Finding{{"C1.1", "ci.yml does not set CGO_ENABLED=0"}},
		},
		{
			name:  `YAML env form set to "1": finding`,
			ciYML: "jobs:\n  build:\n    env:\n      CGO_ENABLED: \"1\"\n",
			want:  []Finding{{"C1.1", "ci.yml does not set CGO_ENABLED=0"}},
		},
		{
			name:  "no mention at all: finding",
			ciYML: "jobs:\n  build:\n    steps:\n      - run: go build ./...\n",
			want:  []Finding{{"C1.1", "ci.yml does not set CGO_ENABLED=0"}},
		},
		{
			// port spec §9.7's CRLF concern, checked the other direction:
			// a CRLF-terminated YAML env line still matches.
			name:  `CRLF file, YAML env form CGO_ENABLED: "0": no finding`,
			ciYML: "jobs:\r\n  build:\r\n    env:\r\n      CGO_ENABLED: \"0\"\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := t.TempDir()
			writeFile(t, repo, "Makefile", conformantMakefile)
			writeFile(t, repo, "flake.nix", conformantFlake)
			if !tt.noFile {
				writeFile(t, repo, filepath.Join(".github", "workflows", "ci.yml"), tt.ciYML)
			}
			a := &auditor{repo: repo}
			a.cgoEnabled()
			if !reflect.DeepEqual(a.findings, tt.want) {
				t.Fatalf("findings = %v, want %v", a.findings, tt.want)
			}
		})
	}
}

// TestCheckManifestDoc covers §9.1's "parse, don't grep" field checks
// against synthetic manifest documents, including the case the oracle's
// grep gets wrong: a pretty-printed (spaced) document that is otherwise
// fully conformant.
func TestCheckManifestDoc(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		want []Finding
	}{
		{
			name: "conformant, compact JSON",
			doc:  `{"schemaVersion":1,"contract":"toolsmith/v1","manifest_digest":"sha256:abc"}`,
			want: nil,
		},
		{
			// port spec §9.1's repro: MarshalIndent-style spacing. The
			// oracle's `"manifest_digest":"sha256:` and
			// `"contract":"toolsmith/` grep patterns would both miss
			// this (a space follows each colon); the port's field
			// checks do not, because they parse.
			name: "conformant, pretty-printed JSON",
			doc: `{
  "schemaVersion": 1,
  "contract": "toolsmith/v1",
  "manifest_digest": "sha256:abc"
}`,
			want: nil,
		},
		{
			name: "missing schemaVersion (zero value)",
			doc:  `{"contract":"toolsmith/v1","manifest_digest":"sha256:abc"}`,
			want: []Finding{{"C3.1", "manifest --json carries no schemaVersion"}},
		},
		{
			name: "digest missing the sha256: prefix",
			doc:  `{"schemaVersion":1,"contract":"toolsmith/v1","manifest_digest":"abc"}`,
			want: []Finding{{"C3.4", "manifest --json carries no manifest_digest"}},
		},
		{
			name: "contract missing the toolsmith/ prefix",
			doc:  `{"schemaVersion":1,"contract":"other/v1","manifest_digest":"sha256:abc"}`,
			want: []Finding{{"C3.6", "manifest --json declares no toolsmith contract version"}},
		},
		{
			name: "not JSON at all: all three findings",
			doc:  `not json`,
			want: []Finding{
				{"C3.1", "manifest --json carries no schemaVersion"},
				{"C3.4", "manifest --json carries no manifest_digest"},
				{"C3.6", "manifest --json declares no toolsmith contract version"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkManifestDoc([]byte(tt.doc))
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("checkManifestDoc(%q) = %v, want %v", tt.doc, got, tt.want)
			}
		})
	}
}

// TestShaPinFindings covers §7's uses: line surgery, both worked edge
// cases named there: a line with no '@' at all, and a commented-out
// uses: line that never reaches the loop.
func TestShaPinFindings(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []Finding
	}{
		{
			name: "properly SHA-pinned action: no finding",
			data: "      - uses: actions/checkout@1111111111111111111111111111111111111111\n",
			want: nil,
		},
		{
			name: "version-tag pinned action: not SHA-pinned",
			data: "      - uses: actions/checkout@v4\n",
			want: []Finding{{"C6.5", "wf.yml: action not SHA-pinned: - uses: actions/checkout@v4"}},
		},
		{
			name: "local composite action with no @ at all",
			data: "      - uses: ./.github/actions/foo\n",
			want: []Finding{{"C6.5", "wf.yml: action not SHA-pinned: - uses: ./.github/actions/foo"}},
		},
		{
			name: "commented-out uses: line is never seen",
			data: "      # - uses: actions/checkout@v4\n",
			want: nil,
		},
		{
			name: "uses: line with no leading dash",
			data: "        uses: actions/checkout@2222222222222222222222222222222222222222\n",
			want: nil,
		},
		{
			// port spec §9.7, the deliberate divergence. The oracle reads
			// this line with the carriage return attached, tests
			// "3333…\r" against its 40-hex pattern, fails, and calls a
			// correctly pinned action unpinned. The port reads pinned.
			name: "CRLF: a SHA-pinned action stays pinned (port spec §9.7)",
			data: "      - uses: actions/checkout@3333333333333333333333333333333333333333\r\n",
			want: nil,
		},
		{
			// The same divergence on a genuinely unpinned line: the
			// finding fires in both implementations, and only the
			// oracle's message carries the stray carriage return.
			name: "CRLF: an unpinned action still fires, with no stray CR",
			data: "      - uses: actions/checkout@v4\r\n",
			want: []Finding{{"C6.5", "wf.yml: action not SHA-pinned: - uses: actions/checkout@v4"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shaPinFindings("wf.yml", []byte(tt.data))
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("shaPinFindings(%q) = %v, want %v", tt.data, got, tt.want)
			}
		})
	}
}

func writeFile(t *testing.T, repo, rel, content string) {
	t.Helper()
	full := filepath.Join(repo, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
