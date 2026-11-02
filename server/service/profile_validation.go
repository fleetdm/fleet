package service

import (
	"bytes"
	"context"
	"crypto/md5" //nolint:gosec // content comparison, not security
	"errors"
	"fmt"
	"log/slog"
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

// maxLoggedProfileFindings caps the per-profile finding log lines; a hostile
// profile can produce tens of thousands of findings.
const maxLoggedProfileFindings = 20

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

// profileValidationError carries a profile's schema findings; Error describes
// the first. It never holds profile content, so it's safe for expanded profiles
// with secrets.
type profileValidationError struct {
	findings []profilevalidation.Finding
	windows  bool
}

func (e *profileValidationError) Error() string {
	return profileValidationSentence(e.findings[0], e.windows)
}

func (e *profileValidationError) sentences() []string {
	out := make([]string, 0, len(e.findings))
	for _, f := range e.findings {
		out = append(out, profileValidationSentence(f, e.windows))
	}
	return out
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
	return svc.validateProfileSchemaWith(ctx, currentProfileValidationSchema(), kind, data, *mode, name, teamID)
}

// validateProfileSchemaWith is validateProfileSchema against a given snapshot,
// so a batch validates every profile against the same schema version.
func (svc *Service) validateProfileSchemaWith(ctx context.Context, schema *profilevalidation.ProfileValidationSchema, kind profileSchemaKind, data []byte, mode profilevalidation.Mode, name string, teamID uint) error {
	if schema == nil {
		svc.logger.ErrorContext(ctx, "profile schema validation skipped: schema not loaded")
		return nil
	}
	var findings []profilevalidation.Finding
	var err error
	switch kind {
	case profileSchemaAppleProfile:
		findings, err = schema.ValidateAppleProfile(data, mode)
	case profileSchemaAppleDeclaration:
		findings, err = schema.ValidateAppleDeclaration(data, mode)
	case profileSchemaWindowsProfile:
		findings, err = schema.ValidateWindowsProfile(data, mode)
	}
	// Strict mode checks AI-generated profiles, whose findings should be visible;
	// lenient findings are the admin's own content, and can be fixed and then re-tried easily.
	level := slog.LevelDebug
	if mode == profilevalidation.ModeStrict {
		level = slog.LevelWarn
	}
	if err != nil {
		// The structural checks already accepted the profile. The error isn't
		// logged because its text can quote profile content.
		svc.logger.Log(ctx, max(level, slog.LevelInfo), "profile schema validation skipped: profile not readable by validator",
			"profile_name", name, "team_id", teamID)
		return nil
	}
	v := schema.Version
	for _, f := range findings[:min(len(findings), maxLoggedProfileFindings)] {
		svc.logger.Log(ctx, level, "profile schema validation finding",
			"code", f.Code, "payload_type", f.PayloadType, "key_path", f.KeyPath,
			"expected", f.Expected, "actual", f.Actual,
			"profile_name", name, "team_id", teamID, "strict", mode == profilevalidation.ModeStrict,
			"apple_release", v.Apple.Release, "windows_release", v.Windows.Release)
	}
	if len(findings) > maxLoggedProfileFindings {
		svc.logger.Log(ctx, level, "profile schema validation findings not logged",
			"not_logged", len(findings)-maxLoggedProfileFindings, "total", len(findings),
			"profile_name", name, "team_id", teamID, "strict", mode == profilevalidation.ModeStrict)
	}
	if len(findings) == 0 {
		return nil
	}
	return &profileValidationError{findings: findings, windows: kind == profileSchemaWindowsProfile}
}

// profileValidationUserError turns a *profileValidationError from the single
// add/edit paths into the 422 the UI shows as its toast; other errors pass through.
func profileValidationUserError(err error, edit bool) error {
	pvErr, ok := errors.AsType[*profileValidationError](err)
	if !ok {
		return err
	}
	msg := profileValidationAddMessage(pvErr.findings[0], pvErr.windows)
	if edit {
		msg = profileValidationEditMessage(pvErr.findings[0], pvErr.windows)
	}
	return fleet.NewInvalidArgumentError("profile", msg)
}

// validateBatchProfileSchemas runs lenient validation over a batch's new or
// changed profiles when the setting is on, in payload order and against one
// schema snapshot, and returns the first finding with the GitOps copy. Content
// is validated secret-expanded and compared unexpanded.
func (svc *Service) validateBatchProfileSchemas(
	ctx context.Context,
	appCfg *fleet.AppConfig,
	teamID uint,
	expanded map[int]fleet.MDMProfileBatchPayload,
	raw []fleet.MDMProfileBatchPayload,
	appleProfiles map[int]*fleet.MDMAppleConfigProfile,
	appleDecls map[int]*fleet.MDMAppleDeclaration,
	windowsProfiles map[int]*fleet.MDMWindowsConfigProfile,
) error {
	if !appCfg.MDM.EnableProfileValidation {
		return nil
	}
	// Only new or changed content is validated, compared unexpanded against what
	// the fleet stores, matched the way the datastore matches on save.
	stored, err := svc.ds.GetMDMProfileChecksums(ctx, teamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get stored profile checksums")
	}
	schema := currentProfileValidationSchema()
	for i, rawProfile := range raw {
		sum := md5.Sum(rawProfile.Contents) //nolint:gosec // content comparison, not security
		var storedSum []byte
		switch {
		case appleProfiles[i] != nil:
			storedSum = stored.AppleByIdentifier[strings.ToLower(appleProfiles[i].Identifier)]
		case appleDecls[i] != nil:
			storedSum = stored.DeclarationsByName[strings.ToLower(appleDecls[i].Name)]
		case windowsProfiles[i] != nil:
			storedSum = stored.WindowsByName[strings.ToLower(windowsProfiles[i].Name)]
		}
		if bytes.Equal(storedSum, sum[:]) {
			continue
		}

		var err error
		var name string
		switch {
		case appleProfiles[i] != nil:
			name = appleProfiles[i].Name
			err = svc.validateProfileSchemaWith(ctx, schema, profileSchemaAppleProfile, expanded[i].Contents, profilevalidation.ModeLenient, name, teamID)
		case appleDecls[i] != nil:
			name = appleDecls[i].Name
			err = svc.validateProfileSchemaWith(ctx, schema, profileSchemaAppleDeclaration, expanded[i].Contents, profilevalidation.ModeLenient, name, teamID)
		case windowsProfiles[i] != nil:
			name = windowsProfiles[i].Name
			err = svc.validateProfileSchemaWith(ctx, schema, profileSchemaWindowsProfile, expanded[i].Contents, profilevalidation.ModeLenient, name, teamID)
		}
		if pvErr, ok := errors.AsType[*profileValidationError](err); ok {
			// "profiles" rather than the profile's name: the message already names it,
			// and fleetctl would prefix a matching name again
			return fleet.NewInvalidArgumentError("profiles", profileValidationBatchMessage(name, pvErr.findings[0], pvErr.windows))
		}
		if err != nil {
			return err
		}
	}
	return nil
}
