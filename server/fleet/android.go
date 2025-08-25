package fleet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm"
)

// MDMAndroidConfigProfile represents an Android MDM profile in Fleet. This does not map
// directly to a specific policy in the Android API, rather the policy applied is the
// result of combining all applicable profiles.
type MDMAndroidConfigProfile struct {
	// ProfileUUID is the unique identifier of the configuration profile in
	// Fleet. For Android profiles, it is the letter "a" followed by a uuid.
	ProfileUUID      string                      `db:"profile_uuid" json:"profile_uuid"`
	TeamID           *uint                       `db:"team_id" json:"team_id"`
	Name             string                      `db:"name" json:"name"`
	RawJSON          []byte                      `db:"raw_json" json:"-"`
	AutoIncrement    uint64                      `db:"auto_increment" json:"auto_increment"`
	LabelsIncludeAll []ConfigurationProfileLabel `db:"-" json:"labels_include_all,omitempty"`
	LabelsIncludeAny []ConfigurationProfileLabel `db:"-" json:"labels_include_any,omitempty"`
	LabelsExcludeAny []ConfigurationProfileLabel `db:"-" json:"labels_exclude_any,omitempty"`
	CreatedAt        time.Time                   `db:"created_at" json:"created_at"`
	UploadedAt       time.Time                   `db:"uploaded_at" json:"updated_at"` // Difference in DB field name vs JSON is conscious decision to match other platforms
}

// ForbiddenJSONKeys are keys that may not be included in user-provided Android configuration profiles
var ForbiddenJSONKeys = map[string]struct{}{
	"statusReportingSettings":    {},
	"applications":               {},
	"appFunctions":               {},
	"playStoreMode":              {},
	"installAppsDisabled":        {},
	"uninstallAppsDisabled":      {},
	"blockApplicationsEnabled":   {},
	"appAutoUpdatePolicy":        {},
	"systemUpdate":               {},
	"kioskCustomLauncherEnabled": {},
	"kioskCustomization":         {},
	"setupActions":               {},
	"encryptionPolicy":           {},
}

func (m *MDMAndroidConfigProfile) ValidateUserProvided() error {
	if len(bytes.TrimSpace(m.RawJSON)) == 0 {
		return errors.New("The file should include valid JSON.")
	}
	fleetNames := mdm.FleetReservedProfileNames()
	if _, ok := fleetNames[m.Name]; ok {
		return fmt.Errorf("Profile name %q is not allowed.", m.Name)
	}
	type jsonObj map[string]interface{}
	var profileKeyMap jsonObj
	err := json.Unmarshal(m.RawJSON, &profileKeyMap)
	if err != nil {
		// TODO invalid profile err
		return err
	}
	if len(profileKeyMap) == 0 {
		return errors.New("JSON profile is empty")
	}
	for key := range profileKeyMap {
		if _, ok := ForbiddenJSONKeys[key]; ok {
			return fmt.Errorf("Key %q is not allowed in user-provided Android configuration profiles.", key)
		}
	}

	return nil
}
