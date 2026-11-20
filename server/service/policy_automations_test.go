package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestCheckPolicyAutomations(t *testing.T) {
	const (
		script   = fleet.PolicyAutomationScript
		calendar = fleet.PolicyAutomationCalendar
		webhook  = fleet.PolicyAutomationWebhookOrTicket
	)
	team, noTeam := new(uint(1)), new(uint(0))
	for _, c := range []struct {
		name          string
		teamID        *uint
		before, after []fleet.PolicyAutomation
		wantErr       string
	}{
		{"one automation", team, nil, []fleet.PolicyAutomation{script}, ""},
		{"two automations", team, nil, []fleet.PolicyAutomation{script, calendar}, errPolicyOneAutomation},
		{"adding a second", team, []fleet.PolicyAutomation{script}, []fleet.PolicyAutomation{script, webhook}, errPolicyOneAutomation},
		{"saved with two before the rule", team, []fleet.PolicyAutomation{script, webhook}, []fleet.PolicyAutomation{script, webhook}, ""},
		{"saved with two, one removed", team, []fleet.PolicyAutomation{script, webhook}, []fleet.PolicyAutomation{script}, ""},
		{"saved with two, one swapped", team, []fleet.PolicyAutomation{script, webhook}, []fleet.PolicyAutomation{script, calendar}, errPolicyOneAutomation},
		{"All fleets webhook", nil, nil, []fleet.PolicyAutomation{webhook}, ""},
		{"All fleets calendar", nil, nil, []fleet.PolicyAutomation{calendar}, "can only have automation_type"},
		{"All fleets calendar saved before the rule", nil, []fleet.PolicyAutomation{calendar}, []fleet.PolicyAutomation{calendar}, ""},
		{"Unassigned script", noTeam, nil, []fleet.PolicyAutomation{script}, ""},
		{"Unassigned calendar", noTeam, nil, []fleet.PolicyAutomation{calendar}, `can't have automation_type "calendar"`},
		{"fleet calendar", team, nil, []fleet.PolicyAutomation{calendar}, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := checkPolicyAutomations(c.teamID, c.before, c.after)
			if c.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, c.wantErr)
		})
	}
}

func TestApplyAutomationType(t *testing.T) {
	team := new(uint(1))
	automation := func(a fleet.PolicyAutomation) optjson.Any[fleet.PolicyAutomation] {
		return optjson.Any[fleet.PolicyAutomation]{Set: true, Valid: true, Value: a}
	}

	// Choosing an automation clears the others.
	p := fleet.ModifyPolicyPayload{
		AutomationType: automation(fleet.PolicyAutomationScript),
		ScriptID:       optjson.Any[uint]{Set: true, Valid: true, Value: 5},
	}
	webhook, err := applyAutomationType(team, &p)
	require.NoError(t, err)
	require.False(t, webhook)
	require.Equal(t, uint(5), p.ScriptID.Value)
	require.True(t, p.SoftwareTitleID.Set)
	require.False(t, p.SoftwareTitleID.Valid)
	require.True(t, p.ProfileUUID.Set)
	require.False(t, *p.CalendarEventsEnabled)
	require.False(t, *p.ConditionalAccessEnabled)

	p = fleet.ModifyPolicyPayload{AutomationType: automation(fleet.PolicyAutomationWebhookOrTicket), CalendarEventsEnabled: new(false)}
	webhook, err = applyAutomationType(team, &p)
	require.NoError(t, err)
	require.True(t, webhook)
	require.True(t, p.ScriptID.Set)
	require.False(t, p.ScriptID.Valid)

	p = fleet.ModifyPolicyPayload{AutomationType: automation(fleet.PolicyAutomationCalendar)}
	_, err = applyAutomationType(team, &p)
	require.NoError(t, err)
	require.True(t, *p.CalendarEventsEnabled)

	// null removes the automation, and doesn't take a companion field.
	p = fleet.ModifyPolicyPayload{AutomationType: optjson.Any[fleet.PolicyAutomation]{Set: true}}
	webhook, err = applyAutomationType(team, &p)
	require.NoError(t, err)
	require.False(t, webhook)
	p = fleet.ModifyPolicyPayload{
		AutomationType: optjson.Any[fleet.PolicyAutomation]{Set: true},
		ScriptID:       optjson.Any[uint]{Set: true, Valid: true, Value: 5},
	}
	_, err = applyAutomationType(team, &p)
	require.ErrorContains(t, err, "can't be set when automation_type is null")

	// A legacy field that turns on another automation is a second automation.
	p = fleet.ModifyPolicyPayload{AutomationType: automation(fleet.PolicyAutomationWebhookOrTicket), ConditionalAccessEnabled: new(true)}
	_, err = applyAutomationType(team, &p)
	require.ErrorContains(t, err, errPolicyOneAutomation)

	// Companion fields have to match, and the scope rules apply.
	p = fleet.ModifyPolicyPayload{
		AutomationType: automation(fleet.PolicyAutomationSoftware),
		ProfileUUID:    optjson.String{Set: true, Valid: true, Value: "a"},
	}
	_, err = applyAutomationType(team, &p)
	require.ErrorContains(t, err, `automation_type "software" requires software_title_id`)
	p = fleet.ModifyPolicyPayload{AutomationType: automation(fleet.PolicyAutomationScript)}
	_, err = applyAutomationType(nil, &p)
	require.ErrorContains(t, err, `can only have automation_type "webhook_or_ticket"`)
	p = fleet.ModifyPolicyPayload{AutomationType: automation("other")}
	_, err = applyAutomationType(team, &p)
	require.ErrorContains(t, err, `invalid automation_type "other"`)
}
