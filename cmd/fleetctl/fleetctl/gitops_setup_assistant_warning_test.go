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
	teams        []fleet.Team
	teamsErr     error
	hostCounts   map[string]int // raw query -> count
	countErr     error
	countQueries []string
}

func (f *fakeABMChecker) ListABMTokens() ([]*fleet.ABMToken, error) { return f.tokens, f.tokensErr }
func (f *fakeABMChecker) ListTeams(query string) ([]fleet.Team, error) {
	return f.teams, f.teamsErr
}
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
	fleetsSet := func(names ...string) map[string]struct{} {
		m := make(map[string]struct{}, len(names))
		for _, n := range names {
			m[n] = struct{}{}
		}
		return m
	}

	cases := []struct {
		name         string
		checker      *fakeABMChecker
		fleets       map[string]struct{}
		wantWarnedOn []string
	}{
		{
			name: "fleet is a token default on any platform",
			checker: &fakeABMChecker{
				tokens: []*fleet.ABMToken{abmToken("Workstations", "Mobile", "Mobile", "No team")},
			},
			fleets:       fleetsSet("Workstations", "Mobile"),
			wantWarnedOn: nil,
		},
		{
			name: "fleet not covered and no ABM hosts",
			checker: &fakeABMChecker{
				tokens: []*fleet.ABMToken{abmToken("Workstations", "", "", "")},
				teams:  []fleet.Team{{ID: 7, Name: "Lab"}},
			},
			fleets:       fleetsSet("Lab"),
			wantWarnedOn: []string{"Lab"},
		},
		{
			name: "fleet not a default but has ABM hosts",
			checker: &fakeABMChecker{
				tokens: []*fleet.ABMToken{abmToken("Workstations", "", "", "")},
				teams:  []fleet.Team{{ID: 7, Name: "Lab"}},
				hostCounts: map[string]int{
					fmt.Sprintf("team_id=7&mdm_enrollment_status=%s", fleet.MDMEnrollStatusPending): 2,
				},
			},
			fleets:       fleetsSet("Lab"),
			wantWarnedOn: nil,
		},
		{
			name: "no team covered by token without defaults",
			checker: &fakeABMChecker{
				// ListABMTokens reports "No team" for unset platform defaults.
				tokens: []*fleet.ABMToken{abmToken("Workstations", "No team", "No team", "No team")},
			},
			fleets:       fleetsSet(fleet.TeamNameNoTeam),
			wantWarnedOn: nil,
		},
		{
			name: "no team not covered and no ABM hosts",
			checker: &fakeABMChecker{
				tokens: []*fleet.ABMToken{abmToken("Workstations", "Workstations", "Workstations", "Workstations")},
			},
			fleets:       fleetsSet(fleet.TeamNameNoTeam),
			wantWarnedOn: []string{fleet.TeamNameNoTeam},
		},
		{
			name:         "token listing error stays silent about coverage",
			checker:      &fakeABMChecker{tokensErr: errors.New("boom")},
			fleets:       fleetsSet("Lab"),
			wantWarnedOn: nil,
		},
		{
			name: "host count error suppresses the warning rather than risking a false one",
			checker: &fakeABMChecker{
				tokens:   []*fleet.ABMToken{abmToken("Workstations", "", "", "")},
				teams:    []fleet.Team{{ID: 7, Name: "Lab"}},
				countErr: errors.New("boom"),
			},
			fleets:       fleetsSet("Lab"),
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
