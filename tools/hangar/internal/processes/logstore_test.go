package processes

import "testing"

func TestRingDropsOldest(t *testing.T) {
	r := newRing(3)
	for i := 1; i <= 5; i++ {
		r.push(LogEntry{TsMS: uint64(i)})
	}
	snap := r.snapshot()
	if len(snap) != 3 {
		t.Fatalf("size = %d, want 3", len(snap))
	}
	// Oldest two (1,2) dropped; remaining 3,4,5 oldest→newest.
	for i, want := range []uint64{3, 4, 5} {
		if snap[i].TsMS != want {
			t.Errorf("snap[%d].TsMS = %d, want %d", i, snap[i].TsMS, want)
		}
	}
}

func lvl(s string) *string { return &s }

func sampleChannels() map[string][]LogEntry {
	return map[string][]LogEntry{
		"fleet-serve": {
			{TsMS: 10, Level: lvl("info"), Message: "starting up", Channel: "fleet-serve"},
			{TsMS: 20, Level: lvl("warn"), Message: "disk getting full", Channel: "fleet-serve"},
			{TsMS: 30, Level: lvl("error"), Message: "connection refused", Channel: "fleet-serve"},
			{TsMS: 40, Level: nil, Message: "no level here", Channel: "fleet-serve"}, // treated as info
		},
		"docker-compose": {
			{TsMS: 15, Level: lvl("info"), Message: "container up", Channel: "docker-compose"},
		},
	}
}

func TestFilterLogWindowLevels(t *testing.T) {
	ch := sampleChannels()
	all := []string{"debug", "info", "warn", "error"}

	// All levels, source all.
	w := filterLogWindow(ch, "all", 0, all, nil, nil)
	if w.TotalInWindow != 5 {
		t.Errorf("total = %d, want 5", w.TotalInWindow)
	}
	if w.WarnCount != 1 || w.ErrorCount != 1 {
		t.Errorf("warn=%d error=%d, want 1/1", w.WarnCount, w.ErrorCount)
	}
	// Sorted ascending by ts across channels (10,15,20,30,40).
	wantTs := []uint64{10, 15, 20, 30, 40}
	for i, ts := range wantTs {
		if w.Entries[i].TsMS != ts {
			t.Errorf("entry[%d].ts = %d, want %d", i, w.Entries[i].TsMS, ts)
		}
	}

	// Only errors.
	w = filterLogWindow(ch, "all", 0, []string{"error"}, nil, nil)
	if w.TotalInWindow != 1 || w.Entries[0].Message != "connection refused" {
		t.Errorf("error-only filter wrong: %+v", w)
	}

	// Empty level set → nothing.
	w = filterLogWindow(ch, "all", 0, nil, nil, nil)
	if w.TotalInWindow != 0 || len(w.Entries) != 0 {
		t.Errorf("empty levels should yield nothing, got %+v", w)
	}
}

func TestFilterLogWindowSource(t *testing.T) {
	ch := sampleChannels()
	all := []string{"debug", "info", "warn", "error"}

	w := filterLogWindow(ch, "docker-compose", 0, all, nil, nil)
	if w.TotalInWindow != 1 || w.Entries[0].Message != "container up" {
		t.Errorf("source filter wrong: %+v", w)
	}

	// Unknown source → no channels selected.
	w = filterLogWindow(ch, "nope", 0, all, nil, nil)
	if w.TotalInWindow != 0 {
		t.Errorf("unknown source should be empty, got %d", w.TotalInWindow)
	}
}

func TestFilterLogWindowSince(t *testing.T) {
	ch := sampleChannels()
	all := []string{"debug", "info", "warn", "error"}
	w := filterLogWindow(ch, "all", 25, all, nil, nil)
	// Only ts >= 25: the error(30) and the nil-level(40).
	if w.TotalInWindow != 2 {
		t.Errorf("since filter total = %d, want 2", w.TotalInWindow)
	}
}

func TestFilterLogWindowSearch(t *testing.T) {
	ch := sampleChannels()
	all := []string{"debug", "info", "warn", "error"}

	// Substring (case-insensitive).
	s := "REFUSED"
	w := filterLogWindow(ch, "all", 0, all, &s, nil)
	if w.TotalInWindow != 1 || w.Entries[0].Message != "connection refused" {
		t.Errorf("substring search wrong: %+v", w)
	}

	// Regex.
	rx := "/full|refused/"
	w = filterLogWindow(ch, "all", 0, all, &rx, nil)
	if w.TotalInWindow != 2 {
		t.Errorf("regex search total = %d, want 2", w.TotalInWindow)
	}

	// Invalid regex → filtering disabled (everything passes).
	bad := "/[invalid/"
	w = filterLogWindow(ch, "all", 0, all, &bad, nil)
	if w.TotalInWindow != 5 {
		t.Errorf("invalid regex should disable search, got total %d", w.TotalInWindow)
	}
}

func TestFilterLogWindowMaxLines(t *testing.T) {
	ch := sampleChannels()
	all := []string{"debug", "info", "warn", "error"}
	max := 2
	w := filterLogWindow(ch, "all", 0, all, nil, &max)
	// total counts all matches (pre-cap); entries capped to newest 2 (ts 30,40).
	if w.TotalInWindow != 5 {
		t.Errorf("total should be pre-cap 5, got %d", w.TotalInWindow)
	}
	if len(w.Entries) != 2 || w.Entries[0].TsMS != 30 || w.Entries[1].TsMS != 40 {
		t.Errorf("maxLines should keep newest 2, got %+v", w.Entries)
	}
}
