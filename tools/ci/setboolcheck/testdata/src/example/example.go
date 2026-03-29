package example

// --- Should be flagged ---

func setWithMake() {
	seen := make(map[string]bool) // want `map\[string\]bool used as a set`
	seen["a"] = true
	seen["b"] = true
	_ = seen
}

func setWithCompositeLiteral() {
	allowed := map[string]bool{"x": true, "y": true} // want `map\[string\]bool used as a set`
	_ = allowed
}

func setWithIntKey() {
	ids := make(map[int]bool) // want `map\[int\]bool used as a set`
	ids[1] = true
	ids[2] = true
	_ = ids
}

func setReInitWithMake() {
	m := make(map[string]bool) // want `map\[string\]bool used as a set`
	m["a"] = true
	m = make(map[string]bool)
	m["b"] = true
	_ = m
}

func setUsedInClosure() {
	seen := make(map[string]bool) // want `map\[string\]bool used as a set`
	items := []string{"a", "b"}
	for _, item := range items {
		seen[item] = true
	}
	fn := func() {
		seen["c"] = true
	}
	fn()
	_ = seen
}

func setVarDecl() {
	var seen map[string]bool // want `map\[string\]bool used as a set`
	seen = make(map[string]bool)
	seen["a"] = true
	_ = seen
}

var packageLevelSet = map[string]bool{"a": true, "b": true} // want `map\[string\]bool used as a set`

// --- Should NOT be flagged ---

func genuineBoolMap() {
	m := make(map[string]bool)
	m["connected"] = true
	m["disconnected"] = false
	_ = m
}

func boolMapFromCompositeLiteral() {
	m := map[string]bool{"a": true, "b": false}
	_ = m
}

func boolMapFromFunctionCall() {
	m := getBoolMap()
	m["x"] = true
	_ = m
}

func boolMapReassignedFromFunc() {
	m := make(map[string]bool)
	m["a"] = true
	m = getBoolMap()
	_ = m
}

func boolMapParameter(m map[string]bool) {
	m["x"] = true
}

func boolMapCopied() {
	m := make(map[string]bool) // want `map\[string\]bool used as a set`
	m["a"] = true
	other := m // other is tainted (assigned from variable), so not flagged
	_ = other
}

func noIndexedAssignments() {
	m := make(map[string]bool)
	_ = m["a"]
	_ = m
}

func boolMapFromRangeValue() {
	source := map[string]map[string]bool{
		"x": {"a": true},
	}
	for _, v := range source {
		v["b"] = true
	}
}

func boolMapAssignedVariable() {
	m := make(map[string]bool)
	val := true
	m["a"] = val // not a literal true
	_ = m
}

var ExportedMap = map[string]bool{"a": true} // exported package-level — skip

func getBoolMap() map[string]bool { return nil }
