package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_errors "github.com/fleetdm/fleet/v4/server/platform/errors"
)

const (
	autofillConfigProfileDefaultBaseURL = "https://fleetdm.com"
	// Lets developers and QA point Fleet at a local or preview website.
	autofillConfigProfileBaseURLEnvVar = "FLEET_DEV_AI_AUTOFILL_BASE_URL"

	autofillConfigProfileNameDescriptionPath = "/api/v1/get-llm-generated-configuration-profile-name-and-description"
	autofillConfigProfileContentPath         = "/api/v1/get-llm-generated-configuration-profile"

	// Generated profiles carry ADMX templates of several hundred KB, so this is well above any real response.
	autofillConfigProfileMaxResponseBytes = 10 << 20

	// The Windows checks need a name, and the name the admin picks is checked again at save.
	autofillWindowsProfileCheckName = "AI generated profile"
)

// Just inside the UI's 10-second cap.  A var so tests can shorten it.
var autofillConfigProfileTimeout = 9500 * time.Millisecond

////////////////////////////////////////////////////////////////////////////////
// Autofill configuration profile name and description
////////////////////////////////////////////////////////////////////////////////

type autofillConfigProfileNameDescriptionRequest struct {
	NaturalLanguageInstructions string `json:"natural_language_instructions"`
	Platform                    string `json:"platform"`
}

type autofillConfigProfileNameDescriptionResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Err         error  `json:"error,omitempty"`
}

func (r autofillConfigProfileNameDescriptionResponse) Error() error { return r.Err }

func autofillConfigProfileNameDescriptionEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*autofillConfigProfileNameDescriptionRequest)
	name, description, err := svc.AutofillConfigProfileNameDescription(ctx, req.NaturalLanguageInstructions, req.Platform)
	if err != nil {
		return autofillConfigProfileNameDescriptionResponse{Err: err}, nil
	}
	return autofillConfigProfileNameDescriptionResponse{Name: name, Description: description}, nil
}

func (svc *Service) AutofillConfigProfileNameDescription(ctx context.Context, instructions, platform string) (string, string, error) {
	if err := svc.authorizeConfigProfileWriteOnAnyFleet(ctx); err != nil {
		return "", "", err
	}

	instructions, err := svc.validateAutofillConfigProfileRequest(ctx, instructions, platform, fleet.AutofillConfigProfileNameDescriptionErrorMessage)
	if err != nil {
		return "", "", err
	}

	status, body, err := postToConfigProfileGenerator(ctx, autofillConfigProfileNameDescriptionPath, instructions, platform)
	if err != nil {
		return "", "", autofillUpstreamError(ctx, fleet.AutofillConfigProfileNameDescriptionErrorMessage, err)
	}
	if status != http.StatusOK {
		return "", "", autofillUnavailableError(ctx, fleet.AutofillConfigProfileNameDescriptionErrorMessage, status)
	}

	var result struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Name == "" {
		return "", "", autofillUnreadableError(ctx, fleet.AutofillConfigProfileNameDescriptionErrorMessage, err)
	}
	return result.Name, result.Description, nil
}

////////////////////////////////////////////////////////////////////////////////
// Autofill configuration profile content
////////////////////////////////////////////////////////////////////////////////

type autofillConfigProfileContentRequest struct {
	NaturalLanguageInstructions string `json:"natural_language_instructions"`
	Platform                    string `json:"platform"`
	FleetID                     uint   `json:"fleet_id"`
}

type autofillConfigProfileContentResponse struct {
	*fleet.AutofilledConfigProfile
	Err error `json:"error,omitempty"`
}

func (r autofillConfigProfileContentResponse) Error() error { return r.Err }

func autofillConfigProfileContentEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*autofillConfigProfileContentRequest)
	profile, err := svc.AutofillConfigProfileContent(ctx, req.NaturalLanguageInstructions, req.Platform, req.FleetID)
	if err != nil {
		return autofillConfigProfileContentResponse{Err: err}, nil
	}
	return autofillConfigProfileContentResponse{AutofilledConfigProfile: profile}, nil
}

func (svc *Service) AutofillConfigProfileContent(ctx context.Context, instructions, platform string, teamID uint) (*fleet.AutofilledConfigProfile, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: &teamID}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	instructions, err := svc.validateAutofillConfigProfileRequest(ctx, instructions, platform, fleet.AutofillConfigProfileContentErrorMessage)
	if err != nil {
		return nil, err
	}

	// Checked here rather than left to the profile checks below, which return the same license error for an OS
	// updates profile on Fleet Free.  That one is about the generated content; this one is about the request, and
	// is rejected the way Create configuration profile rejects it.  The checks repeat both lookups, but only after
	// a generation that a missing fleet would have wasted.
	if teamID > 0 {
		lic, err := svc.License(ctx)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "checking license")
		}
		if lic == nil || !lic.IsPremium() {
			return nil, ctxerr.Wrap(ctx, fleet.ErrMissingLicense)
		}
		if _, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, &teamID, nil); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get fleet")
		}
	}

	status, body, err := postToConfigProfileGenerator(ctx, autofillConfigProfileContentPath, instructions, platform)
	if err != nil {
		return nil, autofillUpstreamError(ctx, fleet.AutofillConfigProfileContentErrorMessage, err)
	}

	if status == http.StatusBadRequest {
		var decline struct {
			Error  string `json:"error"`
			Reason string `json:"reason"`
		}
		if json.Unmarshal(body, &decline) == nil && decline.Error == "couldNotGenerateProfile" {
			reason := decline.Reason
			if reason == "" {
				reason = "The AI couldn't generate a configuration profile from these instructions."
			}
			return nil, ctxerr.Wrap(ctx, &fleet.AutofillConfigProfileError{
				Message: fleet.AutofillConfigProfileContentErrorMessage,
				Field:   "natural_language_instructions",
				Reason:  reason,
			}, "generator declined")
		}
	}
	if status != http.StatusOK {
		return nil, autofillUnavailableError(ctx, fleet.AutofillConfigProfileContentErrorMessage, status)
	}

	var generated struct {
		ProfileType        string            `json:"profileType"`
		Profile            string            `json:"profile"`
		DeliveryNotes      string            `json:"deliveryNotes"`
		AdditionalProfiles []json.RawMessage `json:"additionalProfiles"`
	}
	if err := json.Unmarshal(body, &generated); err != nil || generated.Profile == "" {
		return nil, autofillUnreadableError(ctx, fleet.AutofillConfigProfileContentErrorMessage, err)
	}

	if len(generated.AdditionalProfiles) > 0 {
		return nil, ctxerr.Wrap(ctx, &fleet.AutofillConfigProfileError{
			Message: fleet.AutofillConfigProfileContentErrorMessage,
			Field:   "natural_language_instructions",
			Reason:  "This configuration profile needs more than one file. Create it manually instead.",
		}, "generator returned additional profiles")
	}

	result := &fleet.AutofilledConfigProfile{
		Content:       generated.Profile,
		DeliveryNotes: generated.DeliveryNotes,
	}
	data := []byte(generated.Profile)
	switch {
	case generated.ProfileType == "mobileconfig" && platform == fleet.AutofillConfigProfilePlatformApple:
		result.Type = fleet.AutofillConfigProfileTypeMobileconfig
		var cp *fleet.MDMAppleConfigProfile
		cp, _, _, err = svc.parseAndValidateAppleConfigProfile(ctx, teamID, "", data, nil, fleet.LabelsIncludeAll, nil)
		if err == nil {
			result.Name = &cp.Name
		}
	case generated.ProfileType == "ddm" && platform == fleet.AutofillConfigProfilePlatformApple:
		result.Type = fleet.AutofillConfigProfileTypeDeclaration
		_, _, _, err = svc.parseAndValidateAppleDeclaration(ctx, teamID, "", data, nil, fleet.LabelsIncludeAll, nil)
	case generated.ProfileType == "csp" && platform == fleet.AutofillConfigProfilePlatformWindows:
		result.Type = fleet.AutofillConfigProfileTypeWindows
		_, _, _, err = svc.parseAndValidateWindowsConfigProfile(ctx, teamID, autofillWindowsProfileCheckName, data, nil, fleet.LabelsIncludeAll, nil, "Couldn't add. ")
	default:
		return nil, autofillUnreadableError(ctx, fleet.AutofillConfigProfileContentErrorMessage,
			errors.New("unexpected profile type "+generated.ProfileType+" for platform "+platform))
	}
	if err != nil {
		return nil, autofillCheckError(ctx, err)
	}

	// Usage statistics only: the admin still gets the profile.
	if err := svc.ds.IncrementAIGeneratedProfileCount(ctx, platform); err != nil {
		svc.logger.ErrorContext(ctx, "failed to count AI generated configuration profile", "err", err, "platform", platform)
	}

	return result, nil
}

////////////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////////////

// authorizeConfigProfileWriteOnAnyFleet mirrors AutofillPolicySql: the request names no fleet, so a user who can
// write profiles on any fleet may ask.
func (svc *Service) authorizeConfigProfileWriteOnAnyFleet(ctx context.Context) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		svc.authz.SkipAuthorization(ctx)
		return fleet.ErrNoContext
	}

	if vc.User.GlobalRole != nil || len(vc.User.Teams) == 0 {
		return ctxerr.Wrap(ctx, svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{}, fleet.ActionWrite))
	}
	teamID := vc.User.Teams[0].Team.ID
	for _, teamUser := range vc.User.Teams {
		if teamUser.Role == fleet.RoleAdmin || teamUser.Role == fleet.RoleMaintainer || teamUser.Role == fleet.RoleGitOps {
			teamID = teamUser.Team.ID
			break
		}
	}
	return ctxerr.Wrap(ctx, svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: &teamID}, fleet.ActionWrite))
}

// validateAutofillConfigProfileRequest returns the trimmed instructions, or the error that stops the request
// before anything is sent to fleetdm.com.
func (svc *Service) validateAutofillConfigProfileRequest(ctx context.Context, instructions, platform, errMessage string) (string, error) {
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get app config")
	}
	if appConfig.ServerSettings.AIFeaturesDisabled {
		return "", ctxerr.Wrap(ctx, &fleet.BadRequestError{Message: "AI features are disabled (server_settings.ai_features_disabled)"})
	}

	var mdmEnabled bool
	var mdmOffMessage string
	switch platform {
	case fleet.AutofillConfigProfilePlatformApple:
		mdmEnabled, mdmOffMessage = appConfig.MDM.EnabledAndConfigured, fleet.AppleMDMNotConfiguredMessage
	case fleet.AutofillConfigProfilePlatformWindows:
		mdmEnabled, mdmOffMessage = appConfig.MDM.WindowsEnabledAndConfigured, fleet.WindowsMDMNotConfiguredMessage
	default:
		return "", ctxerr.Wrap(ctx, &fleet.BadRequestError{Message: `'platform' must be "apple" or "windows"`})
	}

	instructions = strings.TrimSpace(instructions)
	if instructions == "" {
		return "", ctxerr.Wrap(ctx, &fleet.BadRequestError{Message: "'natural_language_instructions' cannot be empty"})
	}
	if utf8.RuneCountInString(instructions) > fleet.MaxAutofillConfigProfileInstructionsLength {
		return "", ctxerr.Wrap(ctx, &fleet.BadRequestError{Message: "'natural_language_instructions' must be 2,000 characters or fewer"})
	}

	// Checked before calling fleetdm.com, since the profile checks would fail anyway and the generation takes
	// up to 10 seconds.
	if !mdmEnabled {
		return "", ctxerr.Wrap(ctx, &fleet.AutofillConfigProfileError{
			Message: errMessage,
			Field:   "base",
			Reason:  mdmOffMessage,
		}, "platform MDM is off")
	}

	return instructions, nil
}

// autofillConfigProfileBaseURL is fleetdm.com unless dev mode overrides it.
func autofillConfigProfileBaseURL() string {
	if baseURL := dev_mode.Env(autofillConfigProfileBaseURLEnvVar); baseURL != "" {
		return strings.TrimSuffix(baseURL, "/")
	}
	return autofillConfigProfileDefaultBaseURL
}

// postToConfigProfileGenerator sends the instructions to fleetdm.com and returns the status and body.  It uses
// the request's context, so a client that disconnects cancels the call.
func postToConfigProfileGenerator(ctx context.Context, path, instructions, platform string) (int, []byte, error) {
	reqBody, err := json.Marshal(map[string]any{
		"naturalLanguageInstructions": instructions,
		"platform":                    platform,
		"forFleet":                    true,
	})
	if err != nil {
		return 0, nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, autofillConfigProfileTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, autofillConfigProfileBaseURL()+path, bytes.NewReader(reqBody))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := fleethttp.NewClient().Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, autofillConfigProfileMaxResponseBytes))
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, body, nil
}

func autofillUpstreamError(ctx context.Context, message string, err error) error {
	// The admin left: return the cancellation itself rather than blaming the AI service.
	if errors.Is(err, context.Canceled) && ctx.Err() != nil {
		return ctxerr.Wrap(ctx, ctx.Err(), "autofill request canceled")
	}
	reason := "The AI service couldn't be reached."
	if errors.Is(err, context.DeadlineExceeded) {
		reason = "The AI service timed out."
	}
	return ctxerr.Wrap(ctx, &fleet.AutofillConfigProfileError{
		Message:     message,
		Field:       "base",
		Reason:      reason,
		InternalErr: err,
	}, "calling fleetdm.com")
}

func autofillUnavailableError(ctx context.Context, message string, status int) error {
	return ctxerr.Wrap(ctx, &fleet.AutofillConfigProfileError{
		Message:     message,
		Field:       "base",
		Reason:      "The AI service is unavailable.",
		InternalErr: errors.New("fleetdm.com returned " + http.StatusText(status)),
	}, "calling fleetdm.com")
}

func autofillUnreadableError(ctx context.Context, message string, err error) error {
	if err == nil {
		err = errors.New("response is missing a required field")
	}
	return ctxerr.Wrap(ctx, &fleet.AutofillConfigProfileError{
		Message:     message,
		Field:       "base",
		Reason:      "The AI service returned a response Fleet couldn't read.",
		InternalErr: err,
	}, "reading fleetdm.com response")
}

// autofillCheckError reports a profile the add checks reject the way the autofill API documents it, with the
// check's own message.  Anything that isn't the client's fault, such as a database error, is returned as-is.
func autofillCheckError(ctx context.Context, err error) error {
	cause := ctxerr.Cause(err)
	var clientErr platform_errors.ErrWithIsClientError
	if !errors.As(cause, &clientErr) || !clientErr.IsClientError() {
		return err
	}

	reason := cause.Error()
	var invalid interface{ Invalid() []map[string]string }
	if errors.As(cause, &invalid) && len(invalid.Invalid()) > 0 {
		reason = invalid.Invalid()[0]["reason"]
	}
	return ctxerr.Wrap(ctx, &fleet.AutofillConfigProfileError{
		Message: fleet.AutofillConfigProfileContentErrorMessage,
		Field:   "natural_language_instructions",
		Reason:  reason,
	}, "generated profile failed checks")
}
