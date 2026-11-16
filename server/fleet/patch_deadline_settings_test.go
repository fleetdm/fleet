package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatePatchDeadlineSettings(t *testing.T) {
	const prefix = "Couldn't edit patch deadline settings. "
	const invalidDay = ` Options are "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", and "sunday".`

	tests := []struct {
		name     string
		days     any
		time     any
		wantErr  string
		wantDays []string
		wantTime string
	}{
		{name: "days Mon", days: []any{"Mon"}, time: "17:00", wantErr: prefix + `"days" includes an invalid value "Mon".` + invalidDay},
		{name: "days Monday", days: []any{"Monday"}, time: "17:00", wantErr: prefix + `"days" includes an invalid value "Monday".` + invalidDay},
		{name: "days number", days: []any{float64(1)}, time: "17:00", wantErr: prefix + `"days" includes an invalid value "1".` + invalidDay},
		{name: "days null entry", days: []any{"monday", nil}, time: "17:00", wantErr: prefix + `"days" includes an invalid value "null".` + invalidDay},
		{name: "days repeated", days: []any{"monday", "monday"}, time: "17:00", wantErr: prefix + `"days" includes "monday" more than once.`},
		{name: "days string", days: "monday", time: "17:00", wantErr: prefix + `"days" must be a list of days, for example ["monday", "wednesday"].`},
		{name: "days empty", days: []any{}, time: "17:00", wantErr: prefix + `"days" must include at least one day.`},
		{name: "days nil", days: nil, time: "17:00", wantErr: prefix + `"days" must include at least one day.`},
		{name: "time empty", days: []any{"monday"}, time: "", wantErr: prefix + `"time" is required.`},
		{name: "time nil", days: []any{"monday"}, time: nil, wantErr: prefix + `"time" is required.`},
		{name: "time number", days: []any{"monday"}, time: float64(1700), wantErr: prefix + `"time" must be wrapped in quotes, for example "17:00".`},
		{name: "time 5pm", days: []any{"monday"}, time: "5pm", wantErr: prefix + `"time" must be in HH:MM (24-hour) format, for example "17:00".`},
		{name: "time 24:00", days: []any{"monday"}, time: "24:00", wantErr: prefix + `"time" must be in HH:MM (24-hour) format, for example "17:00".`},
		{name: "time 11:60", days: []any{"monday"}, time: "11:60", wantErr: prefix + `"time" must be in HH:MM (24-hour) format, for example "17:00".`},
		{name: "days checked before time", days: []any{"Mon"}, time: "5pm", wantErr: prefix + `"days" includes an invalid value "Mon".` + invalidDay},
		{name: "sorted Monday first", days: []any{"friday", "monday"}, time: "17:00", wantDays: []string{"monday", "friday"}, wantTime: "17:00"},
		{name: "time padded", days: []any{"sunday"}, time: "9:00", wantDays: []string{"sunday"}, wantTime: "09:00"},
		{name: "time upper bound", days: []any{"tuesday"}, time: "23:59", wantDays: []string{"tuesday"}, wantTime: "23:59"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			days, tm, err := ValidatePatchDeadlineSettings(tt.days, tt.time)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				var badRequest *BadRequestError
				require.ErrorAs(t, err, &badRequest)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantDays, days)
			require.Equal(t, tt.wantTime, tm)
		})
	}
}
