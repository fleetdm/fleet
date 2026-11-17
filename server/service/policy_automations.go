package service

import (
	"context"
	"fmt"
	"slices"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"golang.org/x/text/unicode/norm"
)

const errPolicyOneAutomation = "a policy can only have one automation"

// policyAutomationCompanions are the request fields that configure an automation, set when the request gives them a
// non-empty value.
type policyAutomationCompanions struct {
	softwareTitleID bool
	scriptID        bool
	profileUUID     bool
}

// checkAutomationTypeCompanions checks that the request sets the field the automation type needs, and no field of
// another automation type.
func checkAutomationTypeCompanions(automationType *fleet.PolicyAutomation, c policyAutomationCompanions) error {
	if automationType != nil && !automationType.IsValid() {
		return &fleet.BadRequestError{Message: fmt.Sprintf("invalid automation_type %q", *automationType)}
	}
	for _, f := range []struct {
		set        bool
		field      string
		automation fleet.PolicyAutomation
	}{
		{c.softwareTitleID, "software_title_id", fleet.PolicyAutomationSoftware},
		{c.scriptID, "script_id", fleet.PolicyAutomationScript},
		{c.profileUUID, "profile_uuid", fleet.PolicyAutomationConfigurationProfile},
	} {
		wanted := automationType != nil && *automationType == f.automation
		switch {
		case wanted && !f.set:
			return &fleet.BadRequestError{Message: fmt.Sprintf("automation_type %q requires %s", f.automation, f.field)}
		case automationType != nil && !wanted && f.set:
			return &fleet.BadRequestError{Message: fmt.Sprintf("%s can only be set with automation_type %q", f.field, f.automation)}
		}
	}
	return nil
}

// checkPolicyAutomations checks the automations a write leaves the policy with. Automations the policy already had are
// allowed, so policies saved before these rules can still be edited, but can't gain or swap in another automation.
func checkPolicyAutomations(teamID *uint, before, after []fleet.PolicyAutomation) error {
	if len(after) > 1 {
		for _, a := range after {
			if !slices.Contains(before, a) {
				return &fleet.BadRequestError{Message: errPolicyOneAutomation}
			}
		}
	}
	for _, a := range after {
		if slices.Contains(before, a) {
			continue
		}
		switch {
		case teamID == nil && a != fleet.PolicyAutomationWebhookOrTicket:
			return &fleet.BadRequestError{Message: fmt.Sprintf(`%q policies can only have automation_type %q`,
				fleet.DisplayNameAllTeams, fleet.PolicyAutomationWebhookOrTicket)}
		case teamID != nil && *teamID == 0 && a == fleet.PolicyAutomationCalendar:
			return &fleet.BadRequestError{Message: fmt.Sprintf(`%q policies can't have automation_type %q`,
				fleet.DisplayNameNoTeam, fleet.PolicyAutomationCalendar)}
		}
	}
	return nil
}

// authorizeWebhookOrTicketChange checks that the user can turn the webhook or ticket automation of the scope's
// policies on or off, which takes the same access as changing the scope's webhook and ticket settings.
func (svc *Service) authorizeWebhookOrTicketChange(ctx context.Context, teamID *uint) error {
	if teamID == nil || *teamID == 0 {
		return svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionWrite)
	}
	return svc.authz.Authorize(ctx, &fleet.Team{ID: *teamID}, fleet.ActionWrite)
}

// checkNewPolicyAutomations runs the automation rules on a policy about to be created.
func (svc *Service) checkNewPolicyAutomations(ctx context.Context, teamID *uint, p fleet.PolicyPayload) error {
	if err := checkPolicyAutomations(teamID, nil, p.Automations()); err != nil {
		return ctxerr.Wrap(ctx, err, "check policy automations")
	}
	if p.WebhooksAndTicketsEnabled {
		return svc.authorizeWebhookOrTicketChange(ctx, teamID)
	}
	return nil
}

// applyAutomationType turns the update's automation_type into the automation fields it replaces, so the rest of the
// update treats both request shapes the same way. It returns whether the policy ends up with the webhook or ticket
// automation, which isn't one of those fields.
func applyAutomationType(teamID *uint, p *fleet.ModifyPolicyPayload) (webhookOrTicket bool, err error) {
	var automationType *fleet.PolicyAutomation
	if p.AutomationType.Valid {
		automationType = &p.AutomationType.Value
		if !automationType.IsValid() {
			return false, &fleet.BadRequestError{Message: fmt.Sprintf("invalid automation_type %q", *automationType)}
		}
		if err := checkPolicyAutomations(teamID, nil, []fleet.PolicyAutomation{*automationType}); err != nil {
			return false, err
		}
	}
	companions := policyAutomationCompanions{
		softwareTitleID: p.SoftwareTitleID.Valid && p.SoftwareTitleID.Value != 0,
		scriptID:        p.ScriptID.Valid && p.ScriptID.Value != 0,
		profileUUID:     p.ProfileUUID.Valid && p.ProfileUUID.Value != "",
	}
	if automationType == nil && (companions.softwareTitleID || companions.scriptID || companions.profileUUID) {
		return false, &fleet.BadRequestError{Message: "software_title_id, script_id, and profile_uuid can't be set when automation_type is null"}
	}
	if err := checkAutomationTypeCompanions(automationType, companions); err != nil {
		return false, err
	}

	is := func(a fleet.PolicyAutomation) bool { return automationType != nil && *automationType == a }
	if (p.CalendarEventsEnabled != nil && *p.CalendarEventsEnabled && !is(fleet.PolicyAutomationCalendar)) ||
		(p.ConditionalAccessEnabled != nil && *p.ConditionalAccessEnabled && !is(fleet.PolicyAutomationConditionalAccess)) {
		return false, &fleet.BadRequestError{Message: errPolicyOneAutomation}
	}
	p.CalendarEventsEnabled = new(is(fleet.PolicyAutomationCalendar))
	p.ConditionalAccessEnabled = new(is(fleet.PolicyAutomationConditionalAccess))
	// A set field with no value removes that automation.
	if !is(fleet.PolicyAutomationSoftware) {
		p.SoftwareTitleID = optjson.Any[uint]{Set: true}
		p.SoftwareInstallerID = optjson.Any[uint]{}
	}
	if !is(fleet.PolicyAutomationScript) {
		p.ScriptID = optjson.Any[uint]{Set: true}
	}
	if !is(fleet.PolicyAutomationConfigurationProfile) {
		p.ProfileUUID = optjson.String{Set: true}
	}
	return is(fleet.PolicyAutomationWebhookOrTicket), nil
}

// policySpecTeamID is the spec's scope: nil for "All fleets", 0 for "Unassigned".
func policySpecTeamID(spec *fleet.PolicySpec, teamIDsByName map[string]uint) *uint {
	switch spec.Team {
	case "":
		return nil
	case "No team":
		return new(uint(0))
	default:
		return new(teamIDsByName[spec.Team])
	}
}

// authorizeWebhookOrTicketSpecChanges checks that the user can change the webhook or ticket automation of every spec
// that turns it on or off. A spec for a new policy changes it when it turns it on.
func (svc *Service) authorizeWebhookOrTicketSpecChanges(ctx context.Context, specs []*fleet.PolicySpec, teamIDsByName map[string]uint) error {
	type scope struct {
		teamID *uint
		specs  []*fleet.PolicySpec
	}
	scopes := make(map[string]*scope)
	for _, spec := range specs {
		if scopes[spec.Team] == nil {
			scopes[spec.Team] = &scope{teamID: policySpecTeamID(spec, teamIDsByName)}
		}
		scopes[spec.Team].specs = append(scopes[spec.Team].specs, spec)
	}
	for _, sc := range scopes {
		// Stored names are NFC-normalized.
		names := make([]string, 0, len(sc.specs))
		for _, spec := range sc.specs {
			names = append(names, norm.NFC.String(spec.Name))
		}
		current, err := svc.ds.WebhooksAndTicketsByPolicyName(ctx, sc.teamID, names)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get current webhook and ticket automations")
		}
		for _, spec := range sc.specs {
			if spec.WebhooksAndTicketsEnabled != current[norm.NFC.String(spec.Name)] {
				if err := svc.authorizeWebhookOrTicketChange(ctx, sc.teamID); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}
