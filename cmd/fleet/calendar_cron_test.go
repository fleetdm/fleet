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
		name                  string
		year                  int
		month                 time.Month
		daysStart             int
		daysEnd               int
		webhookFiredThisMonth bool

		expected time.Time
	}{
		{
			name:                  "March 2024 (webhook hasn't fired)",
			year:                  2024,
			month:                 3,
			daysStart:             1,
			daysEnd:               31,
			webhookFiredThisMonth: false,

			expected: date(2024, 3, 19),
		},
		{
			name:                  "March 2024 (webhook has fired, days before 3rd Tuesday)",
			year:                  2024,
			month:                 3,
			daysStart:             1,
			daysEnd:               18,
			webhookFiredThisMonth: true,

			expected: date(2024, 3, 19),
		},
		{
			name:                  "March 2024 (webhook has fired, days after 3rd Tuesday)",
			year:                  2024,
			month:                 3,
			daysStart:             20,
			daysEnd:               30,
			webhookFiredThisMonth: true,

			expected: date(2024, 4, 16),
		},
		{
			name:                  "April 2024 (webhook hasn't fired)",
			year:                  2024,
			month:                 4,
			daysEnd:               30,
			webhookFiredThisMonth: false,

			expected: date(2024, 4, 16),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for day := tc.daysStart; day <= tc.daysEnd; day++ {
				actual := getPreferredCalendarEventDate(tc.year, tc.month, day, tc.webhookFiredThisMonth)
				require.NotEqual(t, actual.Weekday(), time.Saturday)
				require.NotEqual(t, actual.Weekday(), time.Sunday)
				if day <= tc.expected.Day() || tc.webhookFiredThisMonth {
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
