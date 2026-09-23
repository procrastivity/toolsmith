package check

// AuditedClauses returns the sorted, de-duplicated set of CONTRACT.md
// clause IDs this package can emit a Finding for — the checker's
// coverage claim (T24: "the checker reports which clauses it audited",
// not what a tool self-declares).
//
// This is the one place that claim is written down. It exists to be
// checked against two independent sources of truth, both in
// coverage_test.go:
//
//   - TestAuditedClausesMatchSource scans this package's non-test .go
//     files for clause-ID string literals (the `Finding{"Cx.y", ...}`
//     and `a.find("Cx.y", ...)` shapes Audit actually uses) and requires
//     that set to equal this one. Add or remove a clause from Audit
//     without updating this list, and that test fails.
//   - TestAuditedClausesMatchContract parses CONTRACT.md's [check]
//     marks and requires that set to equal this one too. Move a mark in
//     the document without this list changing to match — or the reverse
//     — and that test fails.
//
// Together they are the drift gate T24 exists to add: CONTRACT.md's
// [check] marks, this list, and the package's own find/Finding call
// sites cannot silently disagree about what the checker covers.
func AuditedClauses() []string {
	return []string{
		"C1.1",
		"C1.2",
		"C1.6",
		"C2.1",
		"C3.1",
		"C3.4",
		"C3.6",
		"C3.10",
		"C6.2",
		"C6.3",
		"C6.5",
		"C6.6",
		"C6.7",
		"C7.5",
	}
}
