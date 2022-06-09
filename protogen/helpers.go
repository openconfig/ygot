package protogen

// addNewKeys appends entries from the newKeys string slice to the
// existing map if the entry is not an existing key. The existing
// map is modified in place.
func addNewKeys(existing map[string]interface{}, newKeys []string) {
	for _, n := range newKeys {
		if _, ok := existing[n]; !ok {
			existing[n] = true
		}
	}
}

// stringKeys returns the keys of the supplied map as a slice of strings.
func stringKeys(m map[string]interface{}) []string {
	var ss []string
	for k := range m {
		ss = append(ss, k)
	}
	return ss
}
