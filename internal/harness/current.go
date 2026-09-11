package harness

// IsCurrent reports whether the tree at dir already is exactly what the
// current binary would install (the Current state of Status). It is the
// install verb's "nothing to write" test, run after RefuseHandEdited has
// cleared the tree.
func IsCurrent(dir string, files map[string][]byte) (bool, error) {
	state, err := Status(dir, files)
	if err != nil {
		return false, err
	}
	return state == Current, nil
}
