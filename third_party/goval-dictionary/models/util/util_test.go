package util

import (
	"testing"
	"time"
)

func TestParsedOrDefaultTime(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		layouts []string
		want    time.Time
	}{
		{
			name:    "success to parse",
			in:      "2021-01-02",
			layouts: []string{"2006-01-02"},
			want:    time.Date(2021, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "success to parse(multi layout)",
			in:      "2021-01-02 15:00:00",
			layouts: []string{"2006-01-02", "2006-01-02 15:04:05"},
			want:    time.Date(2021, time.January, 2, 15, 0, 0, 0, time.UTC),
		},
		{
			name:    "failed to parse",
			in:      "2021/01/02",
			layouts: []string{"2006-01-02"},
			want:    time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "empty string",
			in:      "",
			layouts: []string{"2006-01-02"},
			want:    time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "unknown",
			in:      "unknown",
			layouts: []string{"2006-01-02"},
			want:    time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParsedOrDefaultTime(tt.layouts, tt.in); got != tt.want {
				t.Errorf("got: %v, want: %v", got, tt.want)
			}
		})
	}
}
