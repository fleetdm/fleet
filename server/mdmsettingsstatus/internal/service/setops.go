package service

// union returns the union of multiple string slices.
func union(slices ...[]string) []string {
	seen := make(map[string]struct{})
	for _, slice := range slices {
		for _, s := range slice {
			seen[s] = struct{}{}
		}
	}

	result := make([]string, 0, len(seen))
	for s := range seen {
		result = append(result, s)
	}
	return result
}

// intersection returns the intersection of multiple string slices.
// Returns an empty slice if any input is empty.
func intersection(slices ...[]string) []string {
	if len(slices) == 0 {
		return nil
	}

	// Find the smallest slice to iterate over
	minIdx := 0
	for i, slice := range slices {
		if len(slice) < len(slices[minIdx]) {
			minIdx = i
		}
	}

	if len(slices[minIdx]) == 0 {
		return nil
	}

	// Build sets from all other slices
	sets := make([]map[string]struct{}, len(slices))
	for i, slice := range slices {
		sets[i] = make(map[string]struct{}, len(slice))
		for _, s := range slice {
			sets[i][s] = struct{}{}
		}
	}

	// Iterate over smallest slice and check membership in all others
	result := make([]string, 0)
	for _, s := range slices[minIdx] {
		inAll := true
		for i, set := range sets {
			if i == minIdx {
				continue
			}
			if _, ok := set[s]; !ok {
				inAll = false
				break
			}
		}
		if inAll {
			result = append(result, s)
		}
	}

	return result
}

// subtract returns elements in a that are not in b.
func subtract(a, b []string) []string {
	if len(b) == 0 {
		return a
	}

	bSet := make(map[string]struct{}, len(b))
	for _, s := range b {
		bSet[s] = struct{}{}
	}

	result := make([]string, 0, len(a))
	for _, s := range a {
		if _, ok := bSet[s]; !ok {
			result = append(result, s)
		}
	}
	return result
}
