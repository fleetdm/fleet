package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
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

// checkAutomationTypeLicense checks that the automation type is available on the license. Fleet Free has webhook or
// ticket and no automation; the rest are Premium, like the fields they stand for.
func checkAutomationTypeLicense(ctx context.Context, automationType *fleet.PolicyAutomation) error {
	if automationType != nil && *automationType != fleet.PolicyAutomationWebhookOrTicket && !license.IsPremium(ctx) {
		return fleet.ErrMissingLicense
	}
	return nil
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

// checkCreateCompanions checks that a new policy's software_title_id, software_package_id, script_id, and profile_uuid
// come with the automation_type they belong to. A patch policy's software_title_id names the app it checks, so it
// doesn't need one.
func checkCreateCompanions(req fleet.CreatePolicyRequest) error {
	if req.AutomationType != nil {
		return nil
	}
	patch := req.Type != nil && *req.Type == fleet.PolicyTypePatch
	for _, f := range []struct {
		set   bool
		field string
	}{
		{!patch && req.SoftwareTitleID != nil && *req.SoftwareTitleID != 0, "software_title_id"},
		{req.SoftwareInstallerID != nil && *req.SoftwareInstallerID != 0, "software_package_id"},
		{req.ScriptID != nil && *req.ScriptID != 0, "script_id"},
		{req.ProfileUUID != nil && *req.ProfileUUID != "", "profile_uuid"},
	} {
		if f.set {
			return &fleet.BadRequestError{Message: fmt.Sprintf("%s requires automation_type", f.field)}
		}
	}
	return nil
}

// checkUpdateCompanions checks an update without automation_type, which can only change the item of the policy's
// current automation, such as the script of a policy that runs a script. A patch policy's software_title_id is
// checked by normalizePatchPolicyUpdate.
func checkUpdateCompanions(policy *fleet.Policy, p fleet.ModifyPolicyPayload) error {
	current := policy.Automations()
	patch := policy.Type == fleet.PolicyTypePatch
	for _, f := range []struct {
		set, value bool
		field      string
		automation fleet.PolicyAutomation
	}{
		{!patch && p.SoftwareTitleID.Set, p.SoftwareTitleID.Valid && p.SoftwareTitleID.Value != 0, "software_title_id",
			fleet.PolicyAutomationSoftware},
		{p.SoftwareInstallerID.Set, p.SoftwareInstallerID.Valid && p.SoftwareInstallerID.Value != 0, "software_package_id",
			fleet.PolicyAutomationSoftware},
		{p.ScriptID.Set, p.ScriptID.Valid && p.ScriptID.Value != 0, "script_id", fleet.PolicyAutomationScript},
		{p.ProfileUUID.Set, p.ProfileUUID.Valid && p.ProfileUUID.Value != "", "profile_uuid",
			fleet.PolicyAutomationConfigurationProfile},
	} {
		has := slices.Contains(current, f.automation)
		switch {
		case !f.set:
		case f.value && !has:
			return &fleet.BadRequestError{Message: fmt.Sprintf(
				"%s can only change the item of a policy with automation_type %q; set automation_type to switch automations",
				f.field, f.automation)}
		case !f.value && has:
			return &fleet.BadRequestError{Message: fmt.Sprintf(
				"%s can't be removed on its own; set automation_type to null to remove the automation", f.field)}
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
		p.ForcePatch = new(false)
		p.PatchWhenClosed = new(false)
		p.NotifyBeforePatching = new(false)
		p.PatchDeadline = new(false)
	}
	if !is(fleet.PolicyAutomationScript) {
		p.ScriptID = optjson.Any[uint]{Set: true}
	}
	if !is(fleet.PolicyAutomationConfigurationProfile) {
		p.ProfileUUID = optjson.String{Set: true}
	}
	return is(fleet.PolicyAutomationWebhookOrTicket), nil
}

func isPatchPolicySpec(spec *fleet.PolicySpec) bool {
	return spec.Type == fleet.PolicyTypePatch && spec.FleetMaintainedAppSlug != ""
}

// policySpecTeamID is the spec's scope: nil for "All fleets", 0 for "Unassigned".
func policySpecTeamID(spec *fleet.PolicySpec, teamIDsByName map[string]uint) *uint {
	switch spec.Team {
	case "":
		return nil
	case fleet.DisplayNameNoTeam:
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
		// Stored names are NFC-normalized. Patch policies are matched by app, so a spec can rename one.
		names := make([]string, 0, len(sc.specs))
		var slugs []string
		for _, spec := range sc.specs {
			names = append(names, norm.NFC.String(spec.Name))
			if isPatchPolicySpec(spec) && sc.teamID != nil {
				slugs = append(slugs, spec.FleetMaintainedAppSlug)
			}
		}
		current, err := svc.ds.WebhooksAndTicketsByPolicyName(ctx, sc.teamID, names)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get current webhook and ticket automations")
		}
		var currentBySlug map[string]bool
		if len(slugs) > 0 {
			if currentBySlug, err = svc.ds.WebhooksAndTicketsByPatchPolicySlug(ctx, *sc.teamID, slugs); err != nil {
				return ctxerr.Wrap(ctx, err, "get current webhook and ticket automations of patch policies")
			}
		}
		for _, spec := range sc.specs {
			before := current[norm.NFC.String(spec.Name)]
			if enabled, ok := currentBySlug[spec.FleetMaintainedAppSlug]; ok && isPatchPolicySpec(spec) {
				before = enabled
			}
			if spec.WebhooksAndTicketsEnabled != before {
				if err := svc.authorizeWebhookOrTicketChange(ctx, sc.teamID); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

// normalizePatchPolicyCreate maps a new patch policy's software_title_id, which names the app it checks, onto the fields
// the rest of create uses: patch_software_title_id for that app, and software_title_id for its install, kept only when
// automation_type turns patching on. Requests with patch_software_title_id keep their meaning.
func normalizePatchPolicyCreate(tp *fleet.NewTeamPolicyPayload) error {
	if tp.Type == nil || *tp.Type != fleet.PolicyTypePatch {
		return nil
	}

	patching := tp.AutomationType != nil && *tp.AutomationType == fleet.PolicyAutomationSoftware

	// Patch options install the patch through the software automation, so each one requires the other.
	if tp.ForcePatch || tp.PatchWhenClosed || tp.NotifyBeforePatching || tp.PatchDeadline {
		if !patching {
			return fleet.NewUserMessageError(errors.New(errPatchOptionsRequireSoftwareAutomation), http.StatusUnprocessableEntity)
		}
	} else if patching {
		return fleet.NewUserMessageError(errors.New(errSoftwareAutomationRequiresPatchOption), http.StatusUnprocessableEntity)
	}

	titleSet := tp.SoftwareTitleID != nil && *tp.SoftwareTitleID != 0
	switch {
	case tp.PatchSoftwareTitleID == nil && titleSet:
		tp.PatchSoftwareTitleID = new(*tp.SoftwareTitleID)
		if !patching {
			tp.SoftwareTitleID = nil
		}
	case tp.PatchSoftwareTitleID != nil && titleSet && *tp.SoftwareTitleID != *tp.PatchSoftwareTitleID:
		return &fleet.BadRequestError{Message: "software_title_id and patch_software_title_id must be the same app"}
	}
	if patching && !titleSet && tp.PatchSoftwareTitleID != nil {
		tp.SoftwareTitleID = new(*tp.PatchSoftwareTitleID)
	}
	return nil
}

// normalizePatchPolicyUpdate applies the update rules of a patch policy: its app can't change, so automation_type
// "software" alone turns patching on for that app. With another automation_type, software_title_id only names the
// app; without automation_type it keeps its legacy meaning of installing the app.
func normalizePatchPolicyUpdate(policy *fleet.Policy, p *fleet.ModifyPolicyPayload) error {
	if policy.Type != fleet.PolicyTypePatch || policy.PatchSoftwareTitleID == nil {
		return nil
	}
	titleSet := p.SoftwareTitleID.Valid && p.SoftwareTitleID.Value != 0
	if titleSet && p.SoftwareTitleID.Value != *policy.PatchSoftwareTitleID {
		return &fleet.BadRequestError{Message: "a patch policy's software_title_id can't be changed"}
	}

	forcePatch := ptr.ValOrZero(p.ForcePatch)
	patchWhenClosed := ptr.ValOrZero(p.PatchWhenClosed)
	notifyBeforePatching := ptr.ValOrZero(p.NotifyBeforePatching)
	patchDeadline := ptr.ValOrZero(p.PatchDeadline)

	patching := p.AutomationType.Valid && p.AutomationType.Value == fleet.PolicyAutomationSoftware

	// Patch options install the patch through the software automation, so each one requires the other.
	if p.AutomationType.Set {
		if forcePatch || patchWhenClosed || notifyBeforePatching || patchDeadline {
			if !patching {
				return fleet.NewUserMessageError(errors.New(errPatchOptionsRequireSoftwareAutomation), http.StatusUnprocessableEntity)
			}
		} else if patching {
			return fleet.NewUserMessageError(errors.New(errSoftwareAutomationRequiresPatchOption), http.StatusUnprocessableEntity)
		}
	}

	// Return an error when the request turns on a patch option and the policy's current automation isn't software.
	if !p.AutomationType.Set && policy.SoftwareInstallerID == nil {
		if forcePatch || patchWhenClosed || notifyBeforePatching || patchDeadline {
			return fleet.NewUserMessageError(errors.New(errPatchOptionsRequireSoftwareAutomation), http.StatusUnprocessableEntity)
		}
	}

	switch {
	case patching && !titleSet:
		p.SoftwareTitleID = optjson.Any[uint]{Set: true, Valid: true, Value: *policy.PatchSoftwareTitleID}
	case p.AutomationType.Set && !patching && titleSet:
		p.SoftwareTitleID = optjson.Any[uint]{}
	}
	return nil
}
