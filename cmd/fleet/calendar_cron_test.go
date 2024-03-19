package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetPreferredCalendarEventDate(t *testing.T) {
	date := func(year int, month time.Month, day int) time.Time {
		return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	}
	for _, tc := range []struct {
		name  string
		year  int
		month time.Month
		days  int

		expected time.Time
	}{
		{
			year:     2024,
			month:    3,
			days:     31,
			name:     "March 2024",
			expected: date(2024, 3, 19),
		},
		{
			year:     2024,
			month:    4,
			days:     30,
			name:     "April 2024",
			expected: date(2024, 4, 16),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for day := 1; day <= tc.days; day++ {
				actual := getPreferredCalendarEventDate(tc.year, tc.month, day)
				require.NotEqual(t, actual.Weekday(), time.Saturday)
				require.NotEqual(t, actual.Weekday(), time.Sunday)
				if day <= tc.expected.Day() {
					require.Equal(t, tc.expected, actual)
				} else {
					today := date(tc.year, tc.month, day)
					if weekday := today.Weekday(); weekday == time.Friday {
						require.Equal(t, today.AddDate(0, 0, +3), actual)
					} else if weekday == time.Saturday {
						require.Equal(t, today.AddDate(0, 0, +2), actual)
					} else {
						require.Equal(t, today.AddDate(0, 0, +1), actual)
					}
				}
			}
		})
	}
}
