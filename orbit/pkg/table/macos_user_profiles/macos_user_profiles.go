//go:build darwin
// +build darwin

package macos_user_profiles

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/groob/plist"
	"github.com/osquery/osquery-go/plugin/table"
)

type profilePayload struct {
	ProfileIdentifier        string
	ProfileInstallDate       string
	ProfileDisplayName       string
	ProfileDescription       string
	ProfileVerificationState string
	ProfileUUID              string
	ProfileOrganization      string
	ProfileType              string
}

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	// same columns supported as for the macadmins macos_profiles table, with the
	// addition of the username for which the profiles are being queried:
	// https://github.com/macadmins/osquery-extension/blob/main/tables/macos_profiles/macos_profiles.go
	return []table.ColumnDefinition{
		table.TextColumn("identifier"),
		table.TextColumn("install_date"),
		table.TextColumn("display_name"),
		table.TextColumn("description"),
		table.TextColumn("verification_state"),
		table.TextColumn("uuid"),
		table.TextColumn("organization"),
		table.TextColumn("type"),
		table.TextColumn("username"),
	}
}

// Generate is called to return the results for the table at query time.
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	// the username constraint is required, and only one must exist, which must
	// be an operator equals constraint.
	usernameConstraints, ok := queryContext.Constraints["username"]
	if !ok {
		return nil, errors.New("missing username constraint")
	}
	if len(usernameConstraints.Constraints) != 1 || usernameConstraints.Constraints[0].Operator != table.OperatorEquals {
		return nil, errors.New("invalid username constraint, must be a single equals constraint")
	}

	username := usernameConstraints.Constraints[0].Expression
	if username == "" {
		return nil, errors.New("empty username constraint")
	}

	b, err := runProfilesCmd(username)
	if err != nil {
		return nil, err
	}

	profiles, err := unmarshalProfilesOutput(b)
	if err != nil {
		return nil, err
	}

	return generateResults(profiles[username], username), nil
}

func generateResults(profiles []profilePayload, username string) []map[string]string {
	var results []map[string]string
	for _, profile := range profiles {
		result := map[string]string{
			"identifier":         profile.ProfileIdentifier,
			"install_date":       profile.ProfileInstallDate,
			"display_name":       profile.ProfileDisplayName,
			"description":        profile.ProfileDescription,
			"verification_state": profile.ProfileVerificationState,
			"uuid":               profile.ProfileUUID,
			"organization":       profile.ProfileOrganization,
			"type":               profile.ProfileType,
			"username":           username,
		}
		results = append(results, result)
	}

	return results
}

func unmarshalProfilesOutput(theBytes []byte) (map[string][]profilePayload, error) {
	var profiles map[string][]profilePayload
	if err := plist.Unmarshal(theBytes, &profiles); err != nil {
		return profiles, fmt.Errorf("unmarshal profiles output: %w", err)
	}
	return profiles, nil
}

func runProfilesCmd(username string) ([]byte, error) {
	cmd := exec.Command("/usr/bin/profiles", "-L", "-U", username, "-o", "stdout-xml")
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("calling /usr/bin/profiles to get user-scope profile payloads: %w", err)
	}
	return out, nil
}
