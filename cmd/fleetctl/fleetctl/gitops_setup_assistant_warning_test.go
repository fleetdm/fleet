package fleetctl

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
)

type fakeABMChecker struct {
	tokens       []*fleet.ABMToken
	tokensErr    error
	hostCounts   map[string]int // raw query -> count
	countErr     error
	countQueries []string
}

func (f *fakeABMChecker) ListABMTokens() ([]*fleet.ABMToken, error) { return f.tokens, f.tokensErr }
func (f *fakeABMChecker) CountHosts(query string) (int, error) {
	f.countQueries = append(f.countQueries, query)
	if f.countErr != nil {
		return 0, f.countErr
	}
	return f.hostCounts[query], nil
}

func abmToken(macos, ios, ipados, byod string) *fleet.ABMToken {
	return &fleet.ABMToken{
		MacOSTeam:  fleet.ABMTokenTeam{Name: macos},
		IOSTeam:    fleet.ABMTokenTeam{Name: ios},
		IPadOSTeam: fleet.ABMTokenTeam{Name: ipados},
		BYODTeam:   fleet.ABMTokenTeam{Name: byod},
	}
}

func TestWarnSetupAssistantsWithoutABMTokens(t *testing.T) {
	const warnFragment = "won't take effect"

	cases := []struct {
		name         string
		checker      *fakeABMChecker
		fleets       map[string]uint
		wantWarnedOn []string
	}{
		{
			name: "fleet is a token default on any platform",
			checker: &fakeABMChecker{
				tokens: []*fleet.ABMToken{abmToken("Workstations", "Mobile", "Mobile", "No team")},
			},
			fleets:       map[string]uint{"Workstations": 1, "Mobile": 2},
			wantWarnedOn: nil,
		},
		{
			name: "fleet not covered and no ABM hosts",
			checker: &fakeABMChecker{
				tokens: []*fleet.ABMToken{abmToken("Workstations", "", "", "")},
			},
			fleets:       map[string]uint{"Lab": 7},
			wantWarnedOn: []string{"Lab"},
		},
		{
			name: "fleet not a default but has ABM hosts",
			checker: &fakeABMChecker{
				tokens: []*fleet.ABMToken{abmToken("Workstations", "", "", "")},
				hostCounts: map[string]int{
					fmt.Sprintf("team_id=7&mdm_enrollment_status=%s", fleet.MDMEnrollStatusPending): 2,
				},
			},
			fleets:       map[string]uint{"Lab": 7},
			wantWarnedOn: nil,
		},
		{
			name: "no team covered by token without defaults",
			checker: &fakeABMChecker{
				// ListABMTokens reports "No team" for unset platform defaults.
				tokens: []*fleet.ABMToken{abmToken("Workstations", "No team", "No team", "No team")},
			},
			fleets:       map[string]uint{fleet.TeamNameNoTeam: 0},
			wantWarnedOn: nil,
		},
		{
			name: "no team not covered and no ABM hosts",
			checker: &fakeABMChecker{
				tokens: []*fleet.ABMToken{abmToken("Workstations", "Workstations", "Workstations", "Workstations")},
			},
			fleets:       map[string]uint{fleet.TeamNameNoTeam: 0},
			wantWarnedOn: []string{fleet.TeamNameNoTeam},
		},
		{
			name:         "token listing error stays silent about coverage",
			checker:      &fakeABMChecker{tokensErr: errors.New("boom")},
			fleets:       map[string]uint{"Lab": 7},
			wantWarnedOn: nil,
		},
		{
			name: "host count error suppresses the warning rather than risking a false one",
			checker: &fakeABMChecker{
				tokens:   []*fleet.ABMToken{abmToken("Workstations", "", "", "")},
				countErr: errors.New("boom"),
			},
			fleets:       map[string]uint{"Lab": 7},
			wantWarnedOn: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var out strings.Builder
			logf := func(format string, a ...any) { fmt.Fprintf(&out, format, a...) }

			warnSetupAssistantsWithoutABMTokens(c.checker, c.fleets, logf)

			for _, name := range c.wantWarnedOn {
				assert.Contains(t, out.String(), fmt.Sprintf("fleet %s: setup assistant saved", name))
				assert.Contains(t, out.String(), warnFragment)
			}
			if len(c.wantWarnedOn) == 0 {
				assert.NotContains(t, out.String(), warnFragment)
			}
		})
	}
}
