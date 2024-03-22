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
		name      string
		year      int
		month     time.Month
		daysStart int
		daysEnd   int

		expected time.Time
	}{
		{
			name:      "March 2024 (before 3rd Tuesday)",
			year:      2024,
			month:     3,
			daysStart: 1,
			daysEnd:   19,

			expected: date(2024, 3, 19),
		},
		{
			name:      "March 2024 (past 3rd Tuesday)",
			year:      2024,
			month:     3,
			daysStart: 20,
			daysEnd:   31,

			expected: date(2024, 4, 16),
		},
		{
			name:      "April 2024 (before 3rd Tuesday)",
			year:      2024,
			month:     4,
			daysStart: 1,
			daysEnd:   16,

			expected: date(2024, 4, 16),
		},
		{
			name:      "April 2024 (after 3rd Tuesday)",
			year:      2024,
			month:     4,
			daysStart: 17,
			daysEnd:   30,

			expected: date(2024, 5, 21),
		},
		{
			name:      "May 2024 (before 3rd Tuesday)",
			year:      2024,
			month:     5,
			daysStart: 1,
			daysEnd:   21,

			expected: date(2024, 5, 21),
		},
		{
			name:      "May 2024 (after 3rd Tuesday)",
			year:      2024,
			month:     5,
			daysStart: 22,
			daysEnd:   31,

			expected: date(2024, 6, 18),
		},
		{
			name:      "Dec 2024 (before 3rd Tuesday)",
			year:      2024,
			month:     12,
			daysStart: 1,
			daysEnd:   17,

			expected: date(2024, 12, 17),
		},
		{
			name:      "Dec 2024 (after 3rd Tuesday)",
			year:      2024,
			month:     12,
			daysStart: 18,
			daysEnd:   31,

			expected: date(2025, 1, 21),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for day := tc.daysStart; day <= tc.daysEnd; day++ {
				actual := getPreferredCalendarEventDate(tc.year, tc.month, day)
				require.NotEqual(t, actual.Weekday(), time.Saturday)
				require.NotEqual(t, actual.Weekday(), time.Sunday)
				require.Equal(t, tc.expected, actual)
			}
		})
	}
}
