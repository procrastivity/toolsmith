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
		{"C6.3", "no cliff.toml (changelog is not derivable from tags)"},
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

// TestAudit_TagNamespaceElseBranch pins the §9.6 mislabel and coverage
// hole exactly, per the ruling in port-spec.md §1 item 1: reproduced on
// purpose, not a bug in the port.
func TestAudit_TagNamespaceElseBranch(t *testing.T) {
	t.Run("repro A: Makefile present, cliff.toml missing reports the mislabeled C6.3", func(t *testing.T) {
		repo := t.TempDir()
		writeFile(t, repo, "Makefile", "irrelevant\n")
		a := &auditor{repo: repo}
		a.tagNamespace()
		want := []Finding{{"C6.3", "no cliff.toml (changelog is not derivable from tags)"}}
		if !reflect.DeepEqual(a.findings, want) {
			t.Fatalf("findings = %v, want %v", a.findings, want)
		}
	})

	t.Run("repro B: cliff.toml present, Makefile missing reports nothing", func(t *testing.T) {
		repo := t.TempDir()
		writeFile(t, repo, "cliff.toml", `tag_pattern = "v[0-9]*"`+"\n")
		a := &auditor{repo: repo}
		a.tagNamespace()
		if len(a.findings) != 0 {
			t.Fatalf("findings = %v, want none (the C6.2 coverage hole)", a.findings)
		}
	})

	t.Run("both present and conformant: no findings", func(t *testing.T) {
		repo := t.TempDir()
		writeFile(t, repo, "Makefile", "\tgit describe --match 'v[0-9]*'\n")
		writeFile(t, repo, "cliff.toml", `tag_pattern = "v[0-9]*"`+"\n")
		a := &auditor{repo: repo}
		a.tagNamespace()
		if len(a.findings) != 0 {
			t.Fatalf("findings = %v, want none", a.findings)
		}
	})

	t.Run("both present, neither conformant", func(t *testing.T) {
		repo := t.TempDir()
		writeFile(t, repo, "Makefile", "\tgit describe\n")
		writeFile(t, repo, "cliff.toml", "tag_pattern = \"v*\"\n")
		a := &auditor{repo: repo}
		a.tagNamespace()
		want := []Finding{
			{"C6.2", "Makefile git describe lacks --match 'v[0-9]*'"},
			{"C6.2", `cliff.toml tag_pattern is not "v[0-9]*"`},
		}
		if !reflect.DeepEqual(a.findings, want) {
			t.Fatalf("findings = %v, want %v", a.findings, want)
		}
	})
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
