package service

import (
	"fmt"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
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
