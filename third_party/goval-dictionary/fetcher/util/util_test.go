package util

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCveIDPattern(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{
			name: "normal",
			id:   "CVE-2022-0001",
			want: true,
		},
		{
			name: "ID_with_5_digits",
			id:   "CVE-2022-00001",
			want: true,
		},
		{
			name: "invalid_cve_id",
			id:   "CVE-01-0001",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CveIDPattern.Match([]byte(tt.id))
			if got != tt.want {
				t.Errorf("got = %v, want = %v", got, tt.want)
			}
		})
	}
}

func TestUniqueStrings(t *testing.T) {
	in := []string{"1", "1", "2", "3", "1", "2"}
	got := UniqueStrings(in)
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	want := []string{"1", "2", "3"}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("(-got +want):\n%s", diff)
	}
}
