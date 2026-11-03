package fleet

import (
	"fmt"
	"slices"
)

// PatchDeadlineSettings holds the weekdays and host-local HH:MM time when a fleet's patch
// deadlines fall. TeamID 0 is Unassigned.
type PatchDeadlineSettings struct {
	TeamID uint     `json:"fleet_id" db:"team_id"`
	Days   []string `json:"days" db:"-"`
	Time   string   `json:"time" db:"time"`
}

func (s *PatchDeadlineSettings) AuthzType() string { return "patch_deadline_settings" }

func DefaultPatchDeadlineSettings(teamID uint) *PatchDeadlineSettings {
	return &PatchDeadlineSettings{TeamID: teamID, Days: []string{"tuesday"}, Time: "17:00"}
}

var patchDeadlineDays = []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}

const patchDeadlineErrPrefix = "Couldn't edit patch deadline settings. "

// ValidatePatchDeadlineSettings takes values as decoded from JSON (or YAML via JSON), so
// a non-list days or a numeric time reaches it with its original type. It returns days in
// week order and time as HH:MM, or a BadRequestError for the first failure.
func ValidatePatchDeadlineSettings(days, tm any) ([]string, string, error) {
	badRequest := func(msg string) ([]string, string, error) {
		return nil, "", &BadRequestError{Message: patchDeadlineErrPrefix + msg}
	}

	if days == nil {
		return badRequest(`"days" must include at least one day.`)
	}
	list, ok := days.([]any)
	if !ok {
		return badRequest(`"days" must be a list of days, for example ["monday", "wednesday"].`)
	}
	if len(list) == 0 {
		return badRequest(`"days" must include at least one day.`)
	}
	seen := make(map[string]bool, len(list))
	for _, v := range list {
		day, _ := v.(string)
		if !slices.Contains(patchDeadlineDays, day) {
			if v == nil {
				v = "null" // %v would print <nil>
			}
			return badRequest(fmt.Sprintf(`"days" includes an invalid value "%v". Options are "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", and "sunday".`, v))
		}
		if seen[day] {
			return badRequest(fmt.Sprintf(`"days" includes "%s" more than once.`, day))
		}
		seen[day] = true
	}

	if tm == nil || tm == "" {
		return badRequest(`"time" is required.`)
	}
	tmStr, ok := tm.(string)
	if !ok {
		return badRequest(`"time" must be wrapped in quotes, for example "17:00".`)
	}
	t, err := parseAutoUpdateHHMM(tmStr)
	if err != nil {
		return badRequest(`"time" must be in HH:MM (24-hour) format, for example "17:00".`)
	}

	sorted := make([]string, 0, len(seen))
	for _, d := range patchDeadlineDays {
		if seen[d] {
			sorted = append(sorted, d)
		}
	}
	return sorted, t.Format("15:04"), nil
}
