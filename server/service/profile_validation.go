package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// profileValidationTypeWords maps Apple schema types and Windows DDF formats to
// the words used in error messages. Each phrase carries its own article.
var profileValidationTypeWords = map[string]string{
	"string":     "a string",
	"integer":    "a number",
	"real":       "a number",
	"boolean":    "a boolean",
	"date":       "a date",
	"data":       "base64 data",
	"array":      "an array",
	"dictionary": "a dictionary",
	"int":        "a number",
	"bool":       "a boolean",
	"b64":        "base64 data",
	"xml":        "XML",
	"chr":        "a string",
	"float":      "a number",
	"time":       "a time",
	"bin":        "binary data",
	"node":       "a node",
	"null":       "no value",
}

func profileValidationExpected(types []string) string {
	words := make([]string, 0, len(types))
	for _, t := range types {
		w, ok := profileValidationTypeWords[t]
		if !ok {
			w = t
		}
		if !slices.Contains(words, w) {
			words = append(words, w)
		}
	}
	return strings.Join(words, " or ")
}

// profileValidationSentence describes a finding in one sentence. windows
// distinguishes an unknown LocURI from an unknown Apple key, which share a code.
func profileValidationSentence(f profilevalidation.Finding, windows bool) string {
	payloadType := f.PayloadType
	if payloadType == profilevalidation.TopLevelPayloadType {
		payloadType = "Configuration"
	}
	switch f.Code {
	case profilevalidation.CodeTypeMismatch, profilevalidation.CodeFormatMismatch:
		return fmt.Sprintf("\"%s\" expects %s.", f.KeyPath, profileValidationExpected(f.Expected))
	case profilevalidation.CodeUnknownKey:
		if windows {
			return fmt.Sprintf("\"%s\" isn't a valid Windows setting.", f.KeyPath)
		}
		return fmt.Sprintf("\"%s\" isn't valid for the \"%s\" profile type.", f.KeyPath, payloadType)
	case profilevalidation.CodeUnknownType:
		return fmt.Sprintf("\"%s\" isn't a valid profile type.", payloadType)
	case profilevalidation.CodeWrongScope:
		return fmt.Sprintf("\"%s\" can't be set for %ss.", f.KeyPath, f.Actual)
	case profilevalidation.CodeMissingPayloadVersion:
		return fmt.Sprintf("A \"%s\" payload is missing PayloadVersion.", payloadType)
	case profilevalidation.CodeMalformedUUID:
		return fmt.Sprintf("A \"%s\" payload has an invalid PayloadUUID.", payloadType)
	case profilevalidation.CodeMalformedIdentifier:
		return fmt.Sprintf("A \"%s\" payload has an invalid PayloadIdentifier.", payloadType)
	default:
		return fmt.Sprintf("\"%s\" has a %s problem.", f.KeyPath, f.Code)
	}
}

const profileValidationUIHint = " To skip validation, an admin can turn it off in Organization settings > Advanced options."

// profileValidationAddMessage and profileValidationEditMessage are the single
// add/edit endpoint errors; the UI shows them verbatim as the toast.
func profileValidationAddMessage(f profilevalidation.Finding, windows bool) string {
	return "Couldn't add profile. " + profileValidationSentence(f, windows) + profileValidationUIHint
}

func profileValidationEditMessage(f profilevalidation.Finding, windows bool) string {
	return "Couldn't edit profile. " + profileValidationSentence(f, windows) + profileValidationUIHint
}

// profileValidationBatchMessage is the batch endpoint error (GitOps, fleetctl
// apply, batch API, Puppet), naming the failing profile.
func profileValidationBatchMessage(profileName string, f profilevalidation.Finding, windows bool) string {
	return fmt.Sprintf("Couldn't edit configuration_profiles. \"%s\": %s To skip validation, set `enable_profile_validation` to false in your GitOps configuration.",
		profileName, profileValidationSentence(f, windows))
}

var loadProfileValidationSchemaOnce sync.Once

// currentProfileValidationSchema returns the loaded schema. serve loads it at
// startup; this loads it on first use for test services that don't.
func currentProfileValidationSchema() *profilevalidation.ProfileValidationSchema {
	loadProfileValidationSchemaOnce.Do(func() {
		if profilevalidation.Current() == nil {
			_ = profilevalidation.ParseProfileValidationSchema()
		}
	})
	return profilevalidation.Current()
}

type profileSchemaKind int

const (
	profileSchemaAppleProfile profileSchemaKind = iota
	profileSchemaAppleDeclaration
	profileSchemaWindowsProfile
)

// profileValidationError carries the first schema finding for a profile. It
// never holds profile content, so it's safe for expanded profiles with secrets.
type profileValidationError struct {
	finding profilevalidation.Finding
	windows bool
}

func (e *profileValidationError) Error() string {
	return profileValidationSentence(e.finding, e.windows)
}

// lenientProfileValidationMode returns the mode for an admin-uploaded profile:
// lenient when the setting is on, nil (skip) when it's off.
func (svc *Service) lenientProfileValidationMode(ctx context.Context) (*profilevalidation.Mode, error) {
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get app config for profile validation")
	}
	if !appCfg.MDM.EnableProfileValidation {
		return nil, nil
	}
	return new(profilevalidation.ModeLenient), nil
}

// validateProfileSchema runs schema validation in mode (nil skips it), logs
// every finding, and returns the first as a *profileValidationError.
func (svc *Service) validateProfileSchema(ctx context.Context, kind profileSchemaKind, data []byte, mode *profilevalidation.Mode, name string, teamID uint) error {
	if mode == nil {
		return nil
	}
	schema := currentProfileValidationSchema()
	if schema == nil {
		svc.logger.ErrorContext(ctx, "profile schema validation skipped: schema not loaded")
		return nil
	}
	var findings []profilevalidation.Finding
	var err error
	switch kind {
	case profileSchemaAppleProfile:
		findings, err = schema.ValidateAppleProfile(data, *mode)
	case profileSchemaAppleDeclaration:
		findings, err = schema.ValidateAppleDeclaration(data, *mode)
	case profileSchemaWindowsProfile:
		findings, err = schema.ValidateWindowsProfile(data, *mode)
	}
	if err != nil {
		// The structural checks already accepted the profile. The error isn't
		// logged because its text can quote profile content.
		svc.logger.InfoContext(ctx, "profile schema validation skipped: profile not readable by validator",
			"profile_name", name, "team_id", teamID)
		return nil
	}
	v := schema.Version
	for _, f := range findings {
		svc.logger.DebugContext(ctx, "profile schema validation finding",
			"code", f.Code, "payload_type", f.PayloadType, "key_path", f.KeyPath,
			"expected", f.Expected, "actual", f.Actual,
			"profile_name", name, "team_id", teamID, "strict", *mode == profilevalidation.ModeStrict,
			"apple_release", v.Apple.Release, "windows_release", v.Windows.Release)
	}
	if len(findings) == 0 {
		return nil
	}
	return &profileValidationError{finding: findings[0], windows: kind == profileSchemaWindowsProfile}
}

// profileValidationUserError turns a *profileValidationError from the single
// add/edit paths into the 422 the UI shows as its toast; other errors pass through.
func profileValidationUserError(err error, edit bool) error {
	var pvErr *profileValidationError
	if !errors.As(err, &pvErr) {
		return err
	}
	msg := profileValidationAddMessage(pvErr.finding, pvErr.windows)
	if edit {
		msg = profileValidationEditMessage(pvErr.finding, pvErr.windows)
	}
	return fleet.NewInvalidArgumentError("profile", msg)
}
