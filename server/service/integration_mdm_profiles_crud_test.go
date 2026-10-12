package service

// Cross-platform configuration profile CRUD tests for the MDM suite.
//
// Belongs here: creating, updating, and listing profiles through the platform-agnostic endpoints,
// label scoping (include any, exclude any), secret variables in uploaded profiles, profile metadata
// (name, description, self-service, hidden), and Android profile status.
//
// Does not belong here: platform-specific delivery and status (integration_mdm_profiles_apple_test.go,
// integration_mdm_profiles_windows_delivery_test.go) and batch endpoints
// (integration_mdm_profiles_batch_test.go).

import (
	"context"
	"crypto/md5" // nolint:gosec // used only for tests
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	servermdm "github.com/fleetdm/fleet/v4/server/mdm"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

func (s *integrationMDMTestSuite) TestMDMConfigProfileCRUD() {
	t := s.T()
	ctx := context.Background()

	testTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "TestTeam"})
	require.NoError(t, err)

	// Ensure MDM is turned on
	appConfig, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appConfig.MDM.AndroidEnabledAndConfigured = true
	appConfig.MDM.EnabledAndConfigured = true
	appConfig.MDM.WindowsEnabledAndConfigured = true
	err = s.ds.SaveAppConfig(ctx, appConfig)
	require.NoError(t, err)

	// NOTE: label names starting with "-" are sent as "labels_excluding_any"
	// (and the leading "-" is removed from the name). Names starting with
	// "!" are sent as the deprecated "labels" field (and the "!" is removed).
	// Names starting with a "~" prefix are sent as "labels_include_any"
	// (and the leading "~" is removed.
	addLabelsFields := func(labelNames []string) map[string][]string {
		var deprLabels, inclAllLabels, inclAnyLabels, exclLabels []string
		for _, lbl := range labelNames {
			switch {
			case strings.HasPrefix(lbl, "~"):
				inclAnyLabels = append(inclAnyLabels, strings.TrimPrefix(lbl, "~"))
			case strings.HasPrefix(lbl, "-"):
				exclLabels = append(exclLabels, strings.TrimPrefix(lbl, "-"))
			case strings.HasPrefix(lbl, "!"):
				deprLabels = append(deprLabels, strings.TrimPrefix(lbl, "!"))
			default:
				inclAllLabels = append(inclAllLabels, lbl)
			}
		}

		fields := make(map[string][]string)
		if len(deprLabels) > 0 {
			fields["labels"] = deprLabels
		}
		if len(inclAllLabels) > 0 {
			fields["labels_include_all"] = inclAllLabels
		}
		if len(exclLabels) > 0 {
			fields["labels_exclude_any"] = exclLabels
		}
		if len(inclAnyLabels) > 0 {
			fields["labels_include_any"] = inclAnyLabels
		}
		return fields
	}

	assertAppleProfile := func(filename, name, ident string, teamID uint, labelNames []string, wantStatus int, wantErrMsg string) string {
		fields := addLabelsFields(labelNames)
		if teamID > 0 {
			fields["team_id"] = []string{fmt.Sprintf("%d", teamID)}
		}
		body, headers := generateNewProfileMultipartRequest(
			t, filename, mobileconfigForTest(name, ident), s.token, fields,
		)
		res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), wantStatus, headers)

		if wantErrMsg != "" {
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, wantErrMsg)
			return ""
		}

		var resp newMDMConfigProfileResponse
		err := json.NewDecoder(res.Body).Decode(&resp)
		require.NoError(t, err)
		require.NotEmpty(t, resp.ProfileUUID)
		require.Equal(t, "a", string(resp.ProfileUUID[0]))
		return resp.ProfileUUID
	}
	assertAppleDeclaration := func(filename, ident string, teamID uint, labelNames []string, wantStatus int, wantErrMsg string) string {
		fields := addLabelsFields(labelNames)
		if teamID > 0 {
			fields["team_id"] = []string{fmt.Sprintf("%d", teamID)}
		}

		bytes := []byte(fmt.Sprintf(`{
  "Type": "com.apple.configuration.foo",
  "Payload": {
    "Echo": "f1337"
  },
  "Identifier": "%s"
}`, ident))

		body, headers := generateNewProfileMultipartRequest(t, filename, bytes, s.token, fields)
		res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), wantStatus, headers)

		if wantErrMsg != "" {
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, wantErrMsg)
			return ""
		}

		var resp newMDMConfigProfileResponse
		err := json.NewDecoder(res.Body).Decode(&resp)
		require.NoError(t, err)
		require.NotEmpty(t, resp.ProfileUUID)
		require.Equal(t, fleet.MDMAppleDeclarationUUIDPrefix, string(resp.ProfileUUID[0]))
		return resp.ProfileUUID
	}

	createAppleProfile := func(name, ident string, teamID uint, labelNames []string) string {
		uid := assertAppleProfile(name+".mobileconfig", name, ident, teamID, labelNames, http.StatusOK, "")

		var wantJSON string
		if teamID == 0 {
			wantJSON = fmt.Sprintf(`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null, "profile_name": %q, "profile_identifier": %q}`, name, ident)
		} else {
			wantJSON = fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q, "profile_name": %q, "profile_identifier": %q}`, teamID, testTeam.Name, teamID, testTeam.Name, name, ident)
		}
		s.lastActivityOfTypeMatches(fleet.ActivityTypeCreatedMacosProfile{}.ActivityName(), wantJSON, 0)

		return uid
	}

	createAppleDeclaration := func(name, ident string, teamID uint, labelNames []string) string {
		uid := assertAppleDeclaration(name+".json", ident, teamID, labelNames, http.StatusOK, "")

		var wantJSON string
		if teamID == 0 {
			wantJSON = fmt.Sprintf(`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null, "profile_name": %q, "identifier": %q}`, name, ident)
		} else {
			wantJSON = fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q, "profile_name": %q, "identifier": %q}`, teamID, testTeam.Name, teamID, testTeam.Name, name, ident)
		}
		s.lastActivityOfTypeMatches(fleet.ActivityTypeCreatedDeclarationProfile{}.ActivityName(), wantJSON, 0)

		return uid
	}

	assertWindowsProfile := func(filename, locURI string, teamID uint, labelNames []string, wantStatus int, wantErrMsg string) string {
		fields := addLabelsFields(labelNames)
		if teamID > 0 {
			fields["team_id"] = []string{fmt.Sprintf("%d", teamID)}
		}
		body, headers := generateNewProfileMultipartRequest(
			t,
			filename,
			[]byte(fmt.Sprintf(`<Add><Item><Target><LocURI>%s</LocURI></Target></Item></Add><Replace><Item><Target><LocURI>%s</LocURI></Target></Item></Replace>`, locURI, locURI)),
			s.token,
			fields,
		)
		res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), wantStatus, headers)

		if wantErrMsg != "" {
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, wantErrMsg)
			return ""
		}

		var resp newMDMConfigProfileResponse
		err := json.NewDecoder(res.Body).Decode(&resp)
		require.NoError(t, err)
		require.NotEmpty(t, resp.ProfileUUID)
		require.Equal(t, "w", string(resp.ProfileUUID[0]))
		return resp.ProfileUUID
	}
	createWindowsProfile := func(name string, teamID uint, labels []string) string {
		uid := assertWindowsProfile(name+".xml", "./Test", teamID, labels, http.StatusOK, "")

		var wantJSON string
		if teamID == 0 {
			wantJSON = fmt.Sprintf(`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null, "profile_name": %q}`, name)
		} else {
			wantJSON = fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q, "profile_name": %q}`, teamID, testTeam.Name, teamID, testTeam.Name, name)
		}
		s.lastActivityOfTypeMatches(fleet.ActivityTypeCreatedWindowsProfile{}.ActivityName(), wantJSON, 0)

		return uid
	}

	assertAndroidProfile := func(filename string, teamID uint, labelNames []string, wantStatus int, wantErrMsg string) string {
		fields := addLabelsFields(labelNames)
		if teamID > 0 {
			fields["team_id"] = []string{fmt.Sprintf("%d", teamID)}
		}

		bytes := []byte(`{
  "removeUserDisabled": false
}`)

		body, headers := generateNewProfileMultipartRequest(t, filename, bytes, s.token, fields)
		res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), wantStatus, headers)

		if wantErrMsg != "" {
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, wantErrMsg)
			return ""
		}

		var resp newMDMConfigProfileResponse
		err := json.NewDecoder(res.Body).Decode(&resp)
		require.NoError(t, err)
		require.NotEmpty(t, resp.ProfileUUID)
		require.Equal(t, fleet.MDMAndroidProfileUUIDPrefix, string(resp.ProfileUUID[0]))
		return resp.ProfileUUID
	}

	createAndroidProfile := func(name string, teamID uint, labelNames []string) string {
		uid := assertAndroidProfile(name+".json", teamID, labelNames, http.StatusOK, "")

		var wantJSON string
		if teamID == 0 {
			wantJSON = fmt.Sprintf(`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null, "profile_name": %q}`, name)
		} else {
			wantJSON = fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q, "profile_name": %q}`, teamID, testTeam.Name, teamID, testTeam.Name, name)
		}
		s.lastActivityOfTypeMatches(fleet.ActivityTypeCreatedAndroidProfile{}.ActivityName(), wantJSON, 0)

		return uid
	}

	// create a couple Apple profiles for no-team and team
	noTeamAppleProfUUID := createAppleProfile("apple-global-profile", "test-global-ident", 0, nil)
	teamAppleProfUUID := createAppleProfile("apple-team-profile", "test-team-ident", testTeam.ID, nil)
	// create a couple Windows profiles for no-team and team
	noTeamWinProfUUID := createWindowsProfile("win-global-profile", 0, nil)
	teamWinProfUUID := createWindowsProfile("win-team-profile", testTeam.ID, nil)
	// Create a couple Android profiles for no-team and team
	noTeamAndroidProfUUID := createAndroidProfile("android-global-profile", 0, nil)
	teamAndroidProfUUID := createAndroidProfile("android-team-profile", testTeam.ID, nil)

	// Windows profile name conflicts with Apple's for no team
	assertWindowsProfile("apple-global-profile.xml", "./Test", 0, nil, http.StatusConflict, SameProfileNameUploadErrorMsg)
	// but no conflict for team 1
	assertWindowsProfile("apple-global-profile.xml", "./Test", testTeam.ID, nil, http.StatusOK, "")
	// Apple profile name conflicts with Windows' for no team
	assertAppleProfile("win-global-profile.mobileconfig", "win-global-profile", "test-global-ident-2", 0, nil, http.StatusConflict, SameProfileNameUploadErrorMsg)
	// but no conflict for team 1
	assertAppleProfile("win-global-profile.mobileconfig", "win-global-profile", "test-global-ident-2", testTeam.ID, nil, http.StatusOK, "")
	// Windows profile name conflicts with Apple's for team 1
	assertWindowsProfile("apple-team-profile.xml", "./Test", testTeam.ID, nil, http.StatusConflict, SameProfileNameUploadErrorMsg)
	// but no conflict for no-team
	assertWindowsProfile("apple-team-profile.xml", "./Test", 0, nil, http.StatusOK, "")
	// Apple profile name conflicts with Windows' for team 1
	assertAppleProfile("win-team-profile.mobileconfig", "win-team-profile", "test-team-ident-2", testTeam.ID, nil, http.StatusConflict, SameProfileNameUploadErrorMsg)
	// but no conflict for no-team
	assertAppleProfile("win-team-profile.mobileconfig", "win-team-profile", "test-team-ident-2", 0, nil, http.StatusOK, "")
	// Android profile name conflicts with Apple's for no team
	assertAndroidProfile("apple-global-profile.json", 0, nil, http.StatusConflict, SameProfileNameUploadErrorMsg)

	// add some macOS declarations
	createAppleDeclaration("apple-declaration", "test-declaration-ident", 0, nil)
	// identifier must be unique, it conflicts with existing declaration
	assertAppleDeclaration("apple-declaration.json", "test-declaration-ident", 0, nil, http.StatusConflict, "test-declaration-ident already exists")
	// name is pulled from filename, it conflicts with existing declaration
	assertAppleDeclaration("apple-declaration.json", "test-declaration-ident-2", 0, nil, http.StatusConflict, SameProfileNameUploadErrorMsg)
	// uniqueness is checked only within team, so it's fine to have the same name and identifier in different teams
	assertAppleDeclaration("apple-declaration.json", "test-declaration-ident", testTeam.ID, nil, http.StatusOK, "")
	// name is pulled from filename, it conflicts with existing macOS config profile
	assertAppleDeclaration("apple-global-profile.json", "test-declaration-ident-2", 0, nil, http.StatusConflict, SameProfileNameUploadErrorMsg)
	// name is pulled from filename, it conflicts with existing macOS config profile
	assertAppleDeclaration("win-global-profile.json", "test-declaration-ident-2", 0, nil, http.StatusConflict, SameProfileNameUploadErrorMsg)
	// name is pulled from filename, it conflicts with existing Android config profile
	assertAppleDeclaration("android-global-profile.json", "test-declaration-ident-2", 0, nil, http.StatusConflict, SameProfileNameUploadErrorMsg)
	// windows profile name conflicts with existing declaration
	assertWindowsProfile("apple-declaration.xml", "./Test", 0, nil, http.StatusConflict, SameProfileNameUploadErrorMsg)
	// macOS profile name conflicts with existing declaration
	assertAppleProfile("apple-declaration.mobileconfig", "apple-declaration", "test-declaration-ident", 0, nil, http.StatusConflict, SameProfileNameUploadErrorMsg)
	// Android profile name conflicts with existing declaration
	assertAndroidProfile("apple-declaration.json", 0, nil, http.StatusConflict, SameProfileNameUploadErrorMsg)

	// not an xml nor mobileconfig file
	assertWindowsProfile("foo.txt", "./Test", 0, nil, http.StatusBadRequest, "Couldn't add profile. The file should be a .mobileconfig, XML, or JSON file.")
	assertAppleProfile("foo.txt", "foo", "foo-ident", 0, nil, http.StatusBadRequest, "Couldn't add profile. The file should be a .mobileconfig, XML, or JSON file.")
	assertAppleDeclaration("foo.txt", "foo-ident", 0, nil, http.StatusBadRequest, "Couldn't add profile. The file should be a .mobileconfig, XML, or JSON file.")
	assertAndroidProfile("foo.txt", 0, nil, http.StatusBadRequest, "Couldn't add profile. The file should be a .mobileconfig, XML, or JSON file.")

	// Windows-reserved LocURI
	assertWindowsProfile("bitlocker.xml", syncml.FleetBitLockerTargetLocURI, 0, nil, http.StatusBadRequest,
		syncml.DiskEncryptionProfileRestrictionErrMsg)

	// Fleet-reserved profiles
	for name := range servermdm.FleetReservedProfileNames() {
		assertAppleProfile(name+".mobileconfig", name, name+"-ident", 0, nil, http.StatusBadRequest, fmt.Sprintf(`name %s is not allowed`, name))
		assertAppleDeclaration(name+".json", name+"-ident", 0, nil, http.StatusBadRequest, fmt.Sprintf(`name %q is not allowed`, name))
		assertWindowsProfile(name+".xml", "./Test", 0, nil, http.StatusBadRequest, fmt.Sprintf(`Couldn't add. Profile name %q is not allowed.`, name))
	}

	// Windows profile names cannot be empty or whitespace-only after stripping the extension.
	assertWindowsProfile(".xml", "./Test", 0, nil, http.StatusBadRequest, "Couldn't add. Profile name can't be empty.")
	assertWindowsProfile("  .xml", "./Test", 0, nil, http.StatusBadRequest, "Couldn't add. Profile name can't be empty.")

	// profiles with non-existent labels
	assertAppleProfile("apple-profile-with-labels.mobileconfig", "apple-profile-with-labels", "ident-with-labels", 0, []string{"does-not-exist"}, http.StatusBadRequest, `Couldn't update. Label "does-not-exist" doesn't exist. Please remove the label from the configuration profile.`)
	assertAppleDeclaration("apple-declaration-with-labels.json", "ident-with-labels", 0, []string{"does-not-exist"}, http.StatusBadRequest, `Couldn't update. Label "does-not-exist" doesn't exist. Please remove the label from the configuration profile.`)
	assertWindowsProfile("win-profile-with-labels.xml", "./Test", 0, []string{"does-not-exist"}, http.StatusBadRequest, `Couldn't update. Label "does-not-exist" doesn't exist. Please remove the label from the configuration profile.`)
	assertAndroidProfile("android-with-labels.json", 0, []string{"does-not-exist"}, http.StatusBadRequest, `Couldn't update. Label "does-not-exist" doesn't exist. Please remove the label from the configuration profile.`)

	// create a couple of labels
	labelFoo := &fleet.Label{Name: "foo", Query: "select * from foo;"}
	labelFoo, err = s.ds.NewLabel(context.Background(), labelFoo)
	require.NoError(t, err)
	labelBar := &fleet.Label{Name: "bar", Query: "select * from bar;"}
	labelBar, err = s.ds.NewLabel(context.Background(), labelBar)
	require.NoError(t, err)

	// profiles mixing existent and non-existent labels
	assertAppleProfile("apple-profile-with-labels.mobileconfig", "apple-profile-with-labels", "ident-with-labels", 0, []string{"does-not-exist", "foo"}, http.StatusBadRequest, `Couldn't update. Label "does-not-exist" doesn't exist. Please remove the label from the configuration profile.`)
	assertAppleDeclaration("apple-declaration-with-labels.json", "ident-with-labels", 0, []string{"does-not-exist", "foo"}, http.StatusBadRequest, `Couldn't update. Label "does-not-exist" doesn't exist. Please remove the label from the configuration profile.`)
	assertWindowsProfile("win-profile-with-labels.xml", "./Test", 0, []string{"does-not-exist", "bar"}, http.StatusBadRequest, `Couldn't update. Label "does-not-exist" doesn't exist. Please remove the label from the configuration profile.`)
	assertAndroidProfile("android-profile-with-labels.json", 0, []string{"does-not-exist", "bar"}, http.StatusBadRequest, `Couldn't update. Label "does-not-exist" doesn't exist. Please remove the label from the configuration profile.`)

	// profiles with invalid mix of labels (two include modes; exclude may combine with either include mode)
	assertAppleProfile("apple-invalid-profile-with-labels.mobileconfig", "apple-invalid-profile-with-labels", "ident-with-labels", 0, []string{"foo", "!bar"}, http.StatusBadRequest, `Only one of "labels_include_all", "labels_include_any", or "labels" can be included.`)
	assertAppleProfile("apple-invalid-profile-with-labels.mobileconfig", "apple-invalid-profile-with-labels", "ident-with-labels", 0, []string{"foo", "~bar"}, http.StatusBadRequest, `Only one of "labels_include_all", "labels_include_any", or "labels" can be included.`)
	assertAppleDeclaration("apple-invalid-decl-with-labels.json", "ident-decl-with-labels", 0, []string{"!foo", "~bar"}, http.StatusBadRequest, `Only one of "labels_include_all", "labels_include_any", or "labels" can be included.`)
	assertAppleDeclaration("apple-invalid-decl-with-labels.json", "ident-decl-with-labels", 0, []string{"foo", "~bar"}, http.StatusBadRequest, `Only one of "labels_include_all", "labels_include_any", or "labels" can be included.`)
	assertWindowsProfile("win-invalid-profile-with-labels.xml", "./Test", 0, []string{"foo", "~bar"}, http.StatusBadRequest, `Only one of "labels_include_all", "labels_include_any", or "labels" can be included.`)
	assertWindowsProfile("win-invalid-profile-with-labels.xml", "./Test", 0, []string{"foo", "!bar"}, http.StatusBadRequest, `Only one of "labels_include_all", "labels_include_any", or "labels" can be included.`)
	assertAndroidProfile("android-invalid-profile-with-labels.json", 0, []string{"foo", "~bar"}, http.StatusBadRequest, `Only one of "labels_include_all", "labels_include_any", or "labels" can be included.`)
	assertAndroidProfile("android-invalid-profile-with-labels.json", 0, []string{"foo", "!bar"}, http.StatusBadRequest, `Only one of "labels_include_all", "labels_include_any", or "labels" can be included.`)

	// profiles with same label in both include and exclude lists
	assertAppleProfile("apple-invalid-profile-with-labels.mobileconfig", "apple-invalid-profile-with-labels", "ident-with-labels", 0, []string{"foo", "-foo"}, http.StatusBadRequest, `Label "foo" cannot appear in both include and exclude lists.`)
	assertAppleDeclaration("apple-invalid-decl-with-labels.json", "ident-decl-with-labels", 0, []string{"~foo", "-foo"}, http.StatusBadRequest, `Label "foo" cannot appear in both include and exclude lists.`)
	assertWindowsProfile("win-invalid-profile-with-labels.xml", "./Test", 0, []string{"bar", "-bar"}, http.StatusBadRequest, `Label "bar" cannot appear in both include and exclude lists.`)
	assertAndroidProfile("android-invalid-profile-with-labels.json", 0, []string{"~bar", "-bar"}, http.StatusBadRequest, `Label "bar" cannot appear in both include and exclude lists.`)

	// profiles with valid labels
	uuidAppleWithLabel := assertAppleProfile("apple-profile-with-labels.mobileconfig", "apple-profile-with-labels", "ident-with-labels", 0, []string{"!foo"}, http.StatusOK, "")
	uuidAppleWithInclAnyLabel := assertAppleProfile("apple-profile-with-incl-any-labels.mobileconfig", "apple-profile-with-incl-any-labels", "ident-with-incl-any-labels", 0, []string{"~foo", "~bar"}, http.StatusOK, "")
	uuidAppleDDMWithLabel := createAppleDeclaration("apple-decl-with-labels", "ident-decl-with-labels", 0, []string{"foo"})
	uuidWindowsWithLabel := assertWindowsProfile("win-profile-with-labels.xml", "./Test", 0, []string{"-foo", "-bar"}, http.StatusOK, "")
	uuidAppleDDMTeamWithLabel := createAppleDeclaration("apple-team-decl-with-labels", "ident-team-decl-with-labels", testTeam.ID, []string{"-foo"})
	uuidWindowsTeamWithLabel := assertWindowsProfile("win-team-profile-with-labels.xml", "./Test", testTeam.ID, []string{"foo", "bar"}, http.StatusOK, "")
	uuidWindowsTeamWithInclAnyLabel := assertWindowsProfile("win-team-profile-with-incl-any-labels.xml", "./Test", testTeam.ID, []string{"~foo", "~bar"}, http.StatusOK, "")

	uuidAndroidWithLabel := assertAndroidProfile("android-profile-with-labels.json", 0, []string{"-foo", "-bar"}, http.StatusOK, "")
	uuidAndroidTeamWithLabel := assertAndroidProfile("android-team-profile-with-labels.json", testTeam.ID, []string{"foo", "bar"}, http.StatusOK, "")
	uuidAndroidTeamWithInclAnyLabel := assertAndroidProfile("android-team-profile-with-incl-any-labels.json", testTeam.ID, []string{"~foo", "~bar"}, http.StatusOK, "")

	// Windows invalid content
	body, headers := generateNewProfileMultipartRequest(t, "win.xml", []byte("\x00\x01\x02"), s.token, nil)
	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusBadRequest, headers)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add. The file should include valid XML:")

	// Apple invalid mobileconfig content
	body, headers = generateNewProfileMultipartRequest(t,
		"apple.mobileconfig", []byte("\x00\x01\x02"), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusBadRequest, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Configuration profiles can't be signed. Fleet will sign the profile for you.")

	// Apple/Android invalid json declaration
	body, headers = generateNewProfileMultipartRequest(t,
		"apple.json", []byte("{"), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusBadRequest, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add. The file should include valid JSON:")

	// Apple/Android cannot determine which
	body, headers = generateNewProfileMultipartRequest(t,
		"apple_or_android.json", []byte(`{"lower_key": true,"UpperKey": false}`), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusBadRequest, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Keys in declaration (DDM) profile must contain only letters and start with a uppercase letter. Keys in Android profile must contain only letters and start with a lowercase letter.")

	// Android invalid keys
	for key, expectedErr := range fleet.AndroidForbiddenJSONKeys {
		body, headers = generateNewProfileMultipartRequest(t,
			"android.json", []byte(fmt.Sprintf(`{"%s": true}`, key)), s.token, nil)
		res = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusBadRequest, headers)
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, expectedErr)
	}

	// invalid JSON structure
	body, headers = generateNewProfileMultipartRequest(t,
		"android.json", []byte(`{"passwordPolicies": {"testKey": 123}}`), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusBadRequest, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Couldn't add. Invalid JSON payload. "passwordPolicies" format is wrong.`)
	// nested key
	body, headers = generateNewProfileMultipartRequest(t,
		"android.json", []byte(`{"passwordPolicies": [{"passwordMinimumLength": true}]}`), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusBadRequest, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Couldn't add. Invalid JSON payload. "passwordPolicies.0.passwordMinimumLength" format is wrong.`)

	// disallow unknown keys
	body, headers = generateNewProfileMultipartRequest(t,
		"android.json", []byte(`{"unknownKey": true}`), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusBadRequest, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Couldn't add. Invalid JSON payload. Unknown key "unknownKey"`)

	// get the existing profiles work
	expectedProfiles := []fleet.MDMConfigProfilePayload{
		{ProfileUUID: noTeamAppleProfUUID, Platform: "darwin", Name: "apple-global-profile", Identifier: "test-global-ident", TeamID: nil, Scope: string(fleet.PayloadScopeSystem)},
		{ProfileUUID: teamAppleProfUUID, Platform: "darwin", Name: "apple-team-profile", Identifier: "test-team-ident", TeamID: &testTeam.ID, Scope: string(fleet.PayloadScopeSystem)},
		{ProfileUUID: noTeamWinProfUUID, Platform: "windows", Name: "win-global-profile", TeamID: nil},
		{ProfileUUID: teamWinProfUUID, Platform: "windows", Name: "win-team-profile", TeamID: &testTeam.ID},
		{ProfileUUID: noTeamAndroidProfUUID, Platform: "android", Name: "android-global-profile", TeamID: nil},
		{ProfileUUID: teamAndroidProfUUID, Platform: "android", Name: "android-team-profile", TeamID: &testTeam.ID},
		{
			ProfileUUID: uuidAppleDDMWithLabel, Platform: "darwin", Name: "apple-decl-with-labels", Identifier: "ident-decl-with-labels", TeamID: nil,
			LabelsIncludeAll: []fleet.ConfigurationProfileLabel{
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
		{
			ProfileUUID: uuidAppleWithLabel, Platform: "darwin", Name: "apple-profile-with-labels", Identifier: "ident-with-labels", TeamID: nil, Scope: string(fleet.PayloadScopeSystem),
			LabelsIncludeAll: []fleet.ConfigurationProfileLabel{
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
		{
			ProfileUUID: uuidAppleWithInclAnyLabel, Platform: "darwin", Name: "apple-profile-with-incl-any-labels", Identifier: "ident-with-incl-any-labels", TeamID: nil, Scope: string(fleet.PayloadScopeSystem),
			LabelsIncludeAny: []fleet.ConfigurationProfileLabel{
				{LabelID: labelBar.ID, LabelName: labelBar.Name},
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
		{
			ProfileUUID: uuidWindowsWithLabel, Platform: "windows", Name: "win-profile-with-labels", TeamID: nil,
			LabelsExcludeAny: []fleet.ConfigurationProfileLabel{
				{LabelID: labelBar.ID, LabelName: labelBar.Name},
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
		{
			ProfileUUID: uuidAppleDDMTeamWithLabel, Platform: "darwin", Name: "apple-team-decl-with-labels", Identifier: "ident-team-decl-with-labels", TeamID: &testTeam.ID,
			LabelsExcludeAny: []fleet.ConfigurationProfileLabel{
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
		{
			ProfileUUID: uuidWindowsTeamWithLabel, Platform: "windows", Name: "win-team-profile-with-labels", TeamID: &testTeam.ID,
			LabelsIncludeAll: []fleet.ConfigurationProfileLabel{
				{LabelID: labelBar.ID, LabelName: labelBar.Name},
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
		{
			ProfileUUID: uuidWindowsTeamWithInclAnyLabel, Platform: "windows", Name: "win-team-profile-with-incl-any-labels", TeamID: &testTeam.ID,
			LabelsIncludeAny: []fleet.ConfigurationProfileLabel{
				{LabelID: labelBar.ID, LabelName: labelBar.Name},
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
		{
			ProfileUUID: uuidAndroidWithLabel, Platform: "android", Name: "android-profile-with-labels", TeamID: nil,
			LabelsExcludeAny: []fleet.ConfigurationProfileLabel{
				{LabelID: labelBar.ID, LabelName: labelBar.Name},
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
		{
			ProfileUUID: uuidAndroidTeamWithLabel, Platform: "android", Name: "android-team-profile-with-labels", TeamID: &testTeam.ID,
			LabelsIncludeAll: []fleet.ConfigurationProfileLabel{
				{LabelID: labelBar.ID, LabelName: labelBar.Name},
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
		{
			ProfileUUID: uuidAndroidTeamWithInclAnyLabel, Platform: "android", Name: "android-team-profile-with-incl-any-labels", TeamID: &testTeam.ID,
			LabelsIncludeAny: []fleet.ConfigurationProfileLabel{
				{LabelID: labelBar.ID, LabelName: labelBar.Name},
				{LabelID: labelFoo.ID, LabelName: labelFoo.Name},
			},
		},
	}
	for _, prof := range expectedProfiles {
		var getResp getMDMConfigProfileResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", prof.ProfileUUID), nil, http.StatusOK, &getResp)
		require.NotZero(t, getResp.CreatedAt)
		require.NotZero(t, getResp.UploadedAt)
		if getResp.Platform == "darwin" {
			require.Len(t, getResp.Checksum, 16)
		} else {
			require.Empty(t, getResp.Checksum)
		}
		getResp.CreatedAt, getResp.UploadedAt = time.Time{}, time.Time{}
		getResp.Checksum = nil
		// covered by TestConfigProfileNameAndDescription
		getResp.PayloadDisplayName = ""
		// sort the labels by name
		sort.Slice(getResp.LabelsIncludeAll, func(i, j int) bool {
			return getResp.LabelsIncludeAll[i].LabelName < getResp.LabelsIncludeAll[j].LabelName
		})
		sort.Slice(getResp.LabelsExcludeAny, func(i, j int) bool {
			return getResp.LabelsExcludeAny[i].LabelName < getResp.LabelsExcludeAny[j].LabelName
		})
		sort.Slice(getResp.LabelsIncludeAny, func(i, j int) bool {
			return getResp.LabelsIncludeAny[i].LabelName < getResp.LabelsIncludeAny[j].LabelName
		})
		require.Equal(t, prof, *getResp.MDMConfigProfilePayload)

		resp := s.Do("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", prof.ProfileUUID), nil, http.StatusOK, "alt", "media")
		require.NotZero(t, resp.ContentLength)
		require.Contains(t, resp.Header.Get("Content-Disposition"), "attachment;")
		if strings.HasPrefix(prof.ProfileUUID, fleet.MDMAppleProfileUUIDPrefix) { //nolint:gocritic // ignore ifElseChain
			require.Contains(t, resp.Header.Get("Content-Type"), "application/x-apple-aspen-config")
		} else if strings.HasPrefix(prof.ProfileUUID, fleet.MDMAppleDeclarationUUIDPrefix) {
			require.Contains(t, resp.Header.Get("Content-Type"), "application/json")
		} else if strings.HasPrefix(prof.ProfileUUID, fleet.MDMAndroidProfileUUIDPrefix) {
			require.Contains(t, resp.Header.Get("Content-Type"), "application/json")
		} else {
			require.Contains(t, resp.Header.Get("Content-Type"), "application/octet-stream")
		}
		require.Contains(t, resp.Header.Get("X-Content-Type-Options"), "nosniff")

		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, resp.ContentLength, int64(len(b)))
	}

	var getResp getMDMConfigProfileResponse
	// get an unknown Apple profile
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", "ano-such-profile"), nil, http.StatusNotFound, &getResp)
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", "ano-such-profile"), nil, http.StatusNotFound, "alt", "media")
	// get an unknown Apple declaration
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", fmt.Sprintf("%sno-such-profile", fleet.MDMAppleDeclarationUUIDPrefix)), nil, http.StatusNotFound, &getResp)
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", fmt.Sprintf("%sno-such-profile", fleet.MDMAppleDeclarationUUIDPrefix)), nil, http.StatusNotFound, "alt", "media")
	// get an unknown Windows profile
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", "wno-such-profile"), nil, http.StatusNotFound, &getResp)
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", "wno-such-profile"), nil, http.StatusNotFound, "alt", "media")
	// get an unknown Android profile
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", fmt.Sprintf("%sno-such-profile", fleet.MDMAndroidProfileUUIDPrefix)), nil, http.StatusNotFound, &getResp)
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", fmt.Sprintf("%sno-such-profile", fleet.MDMAndroidProfileUUIDPrefix)), nil, http.StatusNotFound, "alt", "media")

	var deleteResp deleteMDMConfigProfileResponse
	// delete existing Apple profiles
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", noTeamAppleProfUUID), nil, http.StatusOK, &deleteResp)

	// turn off apple MDM
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.MDM.EnabledAndConfigured = false
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", teamAppleProfUUID), nil, http.StatusOK, &deleteResp)
	// delete non-existing Apple profile
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", "ano-such-profile"), nil, http.StatusNotFound, &deleteResp)

	// delete existing Apple declaration
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", uuidAppleDDMWithLabel), nil, http.StatusOK, &deleteResp)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDeletedDeclarationProfile{}.ActivityName(),
		`{"profile_name": "apple-decl-with-labels", "identifier": "ident-decl-with-labels", "team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`,
		0,
	)
	// delete non-existing Apple declaration
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", fmt.Sprintf("%sno-such-profile", fleet.MDMAppleDeclarationUUIDPrefix)), nil, http.StatusNotFound, &deleteResp)
	// delete existing Windows profiles
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", noTeamWinProfUUID), nil, http.StatusOK, &deleteResp)
	// Now disabling windows MDM
	filler := struct{}{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"mdm": { "windows_enabled_and_configured": false}}`), http.StatusOK, &filler)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", teamWinProfUUID), nil, http.StatusOK, &deleteResp)
	// delete non-existing Windows profile
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", "wno-such-profile"), nil, http.StatusNotFound, &deleteResp)

	// delete existing Android profiles
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", noTeamAndroidProfUUID), nil, http.StatusOK, &deleteResp)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDeletedAndroidProfile{}.ActivityName(),
		`{"profile_name": "android-global-profile", "team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`,
		0,
	)
	// turn off Android MDM
	appCfg, err = s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.MDM.AndroidEnabledAndConfigured = false
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", teamAndroidProfUUID), nil, http.StatusOK, &deleteResp)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDeletedAndroidProfile{}.ActivityName(),
		fmt.Sprintf(`{"profile_name": "android-team-profile", "team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q}`, testTeam.ID, testTeam.Name, testTeam.ID, testTeam.Name),
		0,
	)
	// delete non-existing Android profiles
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", fmt.Sprintf("%sno-such-profile", fleet.MDMAndroidProfileUUIDPrefix)), nil, http.StatusNotFound, &deleteResp)

	// turn back on apple MDM
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err = q.ExecContext(ctx, "UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.mdm.enabled_and_configured', true) ")
		return err
	})
	// trying to create/delete profiles managed by Fleet fails
	for p := range mobileconfig.FleetPayloadIdentifiers() {
		assertAppleProfile("foo.mobileconfig", p, p, 0, nil, http.StatusBadRequest, fmt.Sprintf("payload identifier %s is not allowed", p))

		// create it directly in the DB to test deletion
		uid := "a" + uuid.NewString()
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			mc := mcBytesForTest(p, p, uuid.New().String())
			_, err := q.ExecContext(ctx,
				"INSERT INTO mdm_apple_configuration_profiles (profile_uuid, identifier, name, mobileconfig, checksum, team_id, uploaded_at) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP())",
				uid, p, p, mc, "1234", 0)
			return err
		})

		var deleteResp deleteMDMConfigProfileResponse
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", uid), nil, http.StatusBadRequest, &deleteResp)

		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				"DELETE FROM mdm_apple_configuration_profiles WHERE profile_uuid = ?",
				uid)
			return err
		})
	}
	// A declaration whose type would now fail upload validation — e.g. it was
	// accepted before its type was added to ForbiddenDeclTypes, or before a
	// config flag was toggled — must still be deletable. Deletion does not
	// re-run the upload-time validator; doing so would trap declarations the
	// user already uploaded. Regression test for
	// https://github.com/fleetdm/fleet/issues/47535.
	{
		forbiddenType := "com.apple.configuration.watch.enrollment"
		require.Contains(t, fleet.ForbiddenDeclTypes, forbiddenType) // guard: type is genuinely forbidden on upload

		// The delete-time validation this fix removed only ran under strict
		// validation (AllowAllDeclarations == false), so this test only guards
		// against its reintroduction when the suite runs strict. Assert that
		// precondition explicitly so the test can't silently become a no-op if
		// the suite default ever changes.
		require.False(t, s.fleetCfg.MDM.AllowAllDeclarations)

		declUUID := fleet.MDMAppleDeclarationUUIDPrefix + uuid.NewString()
		rawJSON := fmt.Sprintf(`{"Type":%q,"Identifier":"com.fleet.forbidden-type","Payload":{}}`, forbiddenType)
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				"INSERT INTO mdm_apple_declarations (declaration_uuid, identifier, name, raw_json, scope, uploaded_at, team_id) VALUES (?, ?, ?, ?, ?, NOW(6), 0)",
				declUUID, "com.fleet.forbidden-type", "forbidden-type-decl", rawJSON, fleet.PayloadScopeSystem)
			return err
		})

		var deleteResp deleteMDMConfigProfileResponse
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", declUUID), nil, http.StatusOK, &deleteResp)
	}

	// A declaration that uses a Fleet-reserved name stays protected from deletion
	// through this endpoint. The reserved-name check is the sole Fleet-managed
	// gate on the delete path, so make sure removing the upload-time validation
	// (above) didn't open a hole.
	{
		reservedName := servermdm.FleetMacOSUpdatesProfileName
		declUUID := fleet.MDMAppleDeclarationUUIDPrefix + uuid.NewString()
		rawJSON := `{"Type":"com.apple.configuration.softwareupdate.enforcement.specific","Identifier":"com.fleet.reserved","Payload":{}}`
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				"INSERT INTO mdm_apple_declarations (declaration_uuid, identifier, name, raw_json, scope, uploaded_at, team_id) VALUES (?, ?, ?, ?, ?, NOW(6), 0)",
				declUUID, "com.fleet.reserved", reservedName, rawJSON, fleet.PayloadScopeSystem)
			return err
		})

		var deleteResp deleteMDMConfigProfileResponse
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", declUUID), nil, http.StatusBadRequest, &deleteResp)

		// the API refused to delete it, so remove the seeded row directly
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, "DELETE FROM mdm_apple_declarations WHERE declaration_uuid = ?", declUUID)
			return err
		})
	}

	// make fleet add a FileVault profile
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)
	profile := s.assertConfigProfilesByIdentifier(nil, mobileconfig.FleetFileVaultPayloadIdentifier, true)

	// try to delete the profile
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", profile.ProfileUUID), nil, http.StatusBadRequest, &deleteResp)

	// make fleet add a Windows OS Updates profile
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_updates": {"deadline_days": 1, "grace_period_days": 1} }
  }`), http.StatusOK, &acResp)
	profUUID := checkWindowsOSUpdatesProfile(t, s.ds, nil, &fleet.WindowsUpdates{DeadlineDays: optjson.SetInt(1), GracePeriodDays: optjson.SetInt(1)})

	// try to delete the profile
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", profUUID), nil, http.StatusBadRequest, &deleteResp)

	// TODO: Add tests for OS updates declaration when implemented.
}

func (s *integrationMDMTestSuite) TestUpdateConfigProfile() {
	t := s.T()
	ctx := context.Background()

	lblA, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "update-prof-lbl-a", Query: "select 1;"})
	require.NoError(t, err)
	lblB, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "update-prof-lbl-b", Query: "select 2;"})
	require.NoError(t, err)

	createProfile := func(fileName string, content []byte, wantUUIDPrefix string) string {
		body, headers := generateNewProfileMultipartRequest(t, fileName, content, s.token, nil)
		res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusOK, headers)
		var resp newMDMConfigProfileResponse
		err := json.NewDecoder(res.Body).Decode(&resp)
		require.NoError(t, err)
		require.NotEmpty(t, resp.ProfileUUID)
		require.Equal(t, wantUUIDPrefix, string(resp.ProfileUUID[0]))
		return resp.ProfileUUID
	}

	// an empty fileName with nil content produces a form without a "profile"
	// file part, i.e. a labels-only edit
	patchProfile := func(profileUUID, fileName string, content []byte, fields map[string][]string, wantStatus int) *http.Response {
		body, headers := generateNewProfileMultipartRequest(t, fileName, content, s.token, fields)
		return s.DoRawWithHeaders("PATCH", "/api/latest/fleet/configuration_profiles/"+profileUUID, body.Bytes(), wantStatus, headers)
	}

	decodePatchResp := func(res *http.Response) updateMDMConfigProfileResponse {
		var resp updateMDMConfigProfileResponse
		err := json.NewDecoder(res.Body).Decode(&resp)
		require.NoError(t, err)
		return resp
	}

	getProfile := func(profileUUID string) getMDMConfigProfileResponse {
		var resp getMDMConfigProfileResponse
		s.DoJSON("GET", "/api/latest/fleet/configuration_profiles/"+profileUUID, nil, http.StatusOK, &resp)
		return resp
	}

	downloadProfile := func(profileUUID string) []byte {
		res := s.Do("GET", "/api/latest/fleet/configuration_profiles/"+profileUUID, nil, http.StatusOK, "alt", "media")
		b, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())
		return b
	}

	assertEditedActivity := func(activityName, profileName, profileIdentifier string) {
		identJSON := ""
		if profileIdentifier != "" {
			// profile_identifier is omitempty and only set for Apple profiles
			identJSON = fmt.Sprintf(`"profile_identifier": %q, `, profileIdentifier)
		}
		wantJSON := fmt.Sprintf(
			`{"profile_name": %q, %s"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null}`,
			profileName, identJSON,
		)
		s.lastActivityOfTypeMatches(activityName, wantJSON, 0)
	}

	declJSONWithEcho := func(ident, echo string) []byte {
		return fmt.Appendf(nil, `{
  "Type": "com.apple.configuration.management.test",
  "Payload": {
    "Echo": %q
  },
  "Identifier": %q
}`, echo, ident)
	}

	// create the initial profiles through the real POST endpoint so the UUIDs
	// have the correct platform prefixes
	appleIdent := "update-apple-ident"
	appleUUID := createProfile("update-apple-profile.mobileconfig", mobileconfigForTest("update-apple-profile", appleIdent), fleet.MDMAppleProfileUUIDPrefix)
	winUUID := createProfile("update-win-profile.xml", syncMLForTest("./TestUpdateProfile"), fleet.MDMWindowsProfileUUIDPrefix)
	declIdent := "update-decl-ident"
	declUUID := createProfile("update-apple-decl.json", declJSONWithEcho(declIdent, "v1"), fleet.MDMAppleDeclarationUUIDPrefix)
	androidUUID := createProfile("update-android-profile.json", []byte(`{"removeUserDisabled": false}`), fleet.MDMAndroidProfileUUIDPrefix)

	//
	// Apple .mobileconfig
	//

	// content-only edit: same PayloadIdentifier, new payload content (fresh
	// random PayloadUUID), no label fields
	origChecksum := getProfile(appleUUID).Checksum
	require.Len(t, origChecksum, 16)
	appleContent2 := mobileconfigForTest("update-apple-profile", appleIdent)
	res := patchProfile(appleUUID, "update-apple-profile.mobileconfig", appleContent2, nil, http.StatusOK)
	patchResp := decodePatchResp(res)
	require.Equal(t, appleUUID, patchResp.ProfileUUID)
	require.Equal(t, appleContent2, downloadProfile(appleUUID))
	prof := getProfile(appleUUID)
	require.Equal(t, "update-apple-profile", prof.Name)
	require.Len(t, prof.Checksum, 16)
	require.NotEqual(t, origChecksum, prof.Checksum)
	assertEditedActivity(fleet.ActivityTypeEditedMacosProfile{}.ActivityName(), "update-apple-profile", appleIdent)

	// content edit with a changed PayloadDisplayName (same identifier) keeps
	// the stored name; only an explicit name renames the profile
	appleContent3 := mobileconfigForTest("update-apple-profile-renamed", appleIdent)
	patchProfile(appleUUID, "update-apple-profile.mobileconfig", appleContent3, nil, http.StatusOK)
	prof = getProfile(appleUUID)
	require.Equal(t, "update-apple-profile", prof.Name)
	patchProfile(appleUUID, "", nil, map[string][]string{"name": {"update-apple-profile-renamed"}}, http.StatusOK)
	prof = getProfile(appleUUID)
	require.Equal(t, "update-apple-profile-renamed", prof.Name)
	require.Equal(t, appleContent3, downloadProfile(appleUUID))
	assertEditedActivity(fleet.ActivityTypeEditedMacosProfile{}.ActivityName(), "update-apple-profile-renamed", appleIdent)

	// labels-only edit: no file, labels persisted, content and checksum
	// unchanged
	checksumBeforeLabels := prof.Checksum
	res = patchProfile(appleUUID, "", nil, map[string][]string{"labels_include_any": {lblA.Name}}, http.StatusOK)
	patchResp = decodePatchResp(res)
	require.Equal(t, appleUUID, patchResp.ProfileUUID)
	prof = getProfile(appleUUID)
	require.Equal(t, []fleet.ConfigurationProfileLabel{{LabelID: lblA.ID, LabelName: lblA.Name}}, prof.LabelsIncludeAny)
	require.Empty(t, prof.LabelsIncludeAll)
	require.Empty(t, prof.LabelsExcludeAny)
	require.Equal(t, checksumBeforeLabels, prof.Checksum)
	require.Equal(t, appleContent3, downloadProfile(appleUUID))
	assertEditedActivity(fleet.ActivityTypeEditedMacosProfile{}.ActivityName(), "update-apple-profile-renamed", appleIdent)

	// content-only edit after labels were set clears label targeting (replace
	// semantics)
	appleContent4 := mobileconfigForTest("update-apple-profile-renamed", appleIdent)
	patchProfile(appleUUID, "update-apple-profile.mobileconfig", appleContent4, nil, http.StatusOK)
	prof = getProfile(appleUUID)
	require.Empty(t, prof.LabelsIncludeAny)
	require.Empty(t, prof.LabelsIncludeAll)
	require.Empty(t, prof.LabelsExcludeAny)
	require.Equal(t, appleContent4, downloadProfile(appleUUID))

	// upload with a different PayloadIdentifier is rejected
	res = patchProfile(appleUUID, "update-apple-profile.mobileconfig", mobileconfigForTest("update-apple-profile-renamed", "some-other-ident"), nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "PayloadIdentifier must match")

	// renaming to collide with an existing Windows profile's name in the same
	// team is rejected
	res = patchProfile(appleUUID, "", nil, map[string][]string{"name": {"update-win-profile"}}, http.StatusConflict)
	require.Contains(t, extractServerErrorText(res.Body), SameProfileNameEditErrorMsg)
	prof = getProfile(appleUUID)
	require.Equal(t, "update-apple-profile-renamed", prof.Name)

	// only one of labels_include_all/labels_include_any may be provided
	res = patchProfile(appleUUID, "", nil, map[string][]string{
		"labels_include_all": {lblA.Name},
		"labels_include_any": {lblB.Name},
	}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), `Only one of "labels_include_all" or "labels_include_any" can be included.`)

	// nonexistent profile UUID with a valid Apple prefix
	patchProfile(fleet.MDMAppleProfileUUIDPrefix+uuid.NewString(), "update-apple-profile.mobileconfig", mobileconfigForTest("update-apple-profile", appleIdent), nil, http.StatusNotFound)

	// unknown UUID prefix
	res = patchProfile("zno-such-profile", "update-apple-profile.mobileconfig", mobileconfigForTest("update-apple-profile", appleIdent), nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "not yet supported")

	// changing the PayloadScope of an existing profile is rejected
	payloadScopeSystem := fleet.PayloadScopeSystem
	payloadScopeUser := fleet.PayloadScopeUser
	scopedIdent := "update-apple-scoped-ident"
	scopedUUID := createProfile("update-apple-scoped.mobileconfig", scopedMobileconfigForTest("update-apple-scoped", scopedIdent, &payloadScopeSystem), fleet.MDMAppleProfileUUIDPrefix)
	res = patchProfile(scopedUUID, "update-apple-scoped.mobileconfig", scopedMobileconfigForTest("update-apple-scoped", scopedIdent, &payloadScopeUser), nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "PayloadScope")

	// profiles managed by Fleet can't be edited; create one directly in the DB
	// as the POST endpoint rejects Fleet payload identifiers
	fleetManagedUUID := fleet.MDMAppleProfileUUIDPrefix + uuid.NewString()
	fleetManagedIdent := mobileconfig.FleetFileVaultPayloadIdentifier
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		mc := mcBytesForTest(fleetManagedIdent, fleetManagedIdent, uuid.NewString())
		_, err := q.ExecContext(ctx,
			"INSERT INTO mdm_apple_configuration_profiles (profile_uuid, identifier, name, mobileconfig, checksum, team_id, uploaded_at) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP())",
			fleetManagedUUID, fleetManagedIdent, fleetManagedIdent, mc, "1234", 0)
		return err
	})
	res = patchProfile(fleetManagedUUID, "", nil, map[string][]string{"labels_include_any": {lblA.Name}}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "managed by Fleet")

	//
	// Windows
	//

	// an edit keeping the same file name leaves the name alone
	winContent2 := syncMLForTest("./TestUpdateProfileEdited")
	res = patchProfile(winUUID, "update-win-profile.xml", winContent2, nil, http.StatusOK)
	patchResp = decodePatchResp(res)
	require.Equal(t, winUUID, patchResp.ProfileUUID)
	require.Equal(t, winContent2, downloadProfile(winUUID))
	prof = getProfile(winUUID)
	require.Equal(t, "update-win-profile", prof.Name)
	assertEditedActivity(fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(), "update-win-profile", "")

	// a differently named file keeps the name; an explicit name renames the
	// profile in place: same UUID, and the activity reports the new name
	winContent3 := syncMLForTest("./TestUpdateProfileRenamed")
	res = patchProfile(winUUID, "update-win-profile-renamed.xml", winContent3, nil, http.StatusOK)
	patchResp = decodePatchResp(res)
	require.Equal(t, winUUID, patchResp.ProfileUUID)
	require.Equal(t, winContent3, downloadProfile(winUUID))
	require.Equal(t, "update-win-profile", getProfile(winUUID).Name)
	patchProfile(winUUID, "", nil, map[string][]string{"name": {"update-win-profile-renamed"}}, http.StatusOK)
	prof = getProfile(winUUID)
	require.Equal(t, "update-win-profile-renamed", prof.Name)
	assertEditedActivity(fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(), "update-win-profile-renamed", "")

	// renaming onto a name another profile in the fleet holds is rejected, and
	// leaves the profile untouched
	winOtherUUID := createProfile("update-win-other.xml", syncMLForTest("./TestUpdateOther"), fleet.MDMWindowsProfileUUIDPrefix)
	require.NotEmpty(t, winOtherUUID)
	res = patchProfile(winUUID, "", nil, map[string][]string{"name": {"update-win-other"}}, http.StatusConflict)
	require.Contains(t, extractServerErrorText(res.Body), "already exists")
	require.Equal(t, "update-win-profile-renamed", getProfile(winUUID).Name)
	require.Equal(t, winContent3, downloadProfile(winUUID))

	// A file name doesn't name the profile, so one that trims down to
	// nothing is accepted and leaves the name alone.
	patchProfile(winUUID, "  .xml", winContent3, nil, http.StatusOK)
	require.Equal(t, "update-win-profile-renamed", getProfile(winUUID).Name)

	// An explicit empty or whitespace-only name is rejected.
	for _, name := range []string{"", "   "} {
		res = patchProfile(winUUID, "", nil, map[string][]string{"name": {name}}, http.StatusUnprocessableEntity)
		require.Contains(t, extractServerErrorText(res.Body), "Profile name can't be empty.")
		require.Equal(t, "update-win-profile-renamed", getProfile(winUUID).Name)
		require.Equal(t, winContent3, downloadProfile(winUUID))
	}

	// labels-only edit; no file, so the name is left alone
	res = patchProfile(winUUID, "", nil, map[string][]string{"labels_include_all": {lblA.Name, lblB.Name}}, http.StatusOK)
	patchResp = decodePatchResp(res)
	require.Equal(t, winUUID, patchResp.ProfileUUID)
	prof = getProfile(winUUID)
	sort.Slice(prof.LabelsIncludeAll, func(i, j int) bool {
		return prof.LabelsIncludeAll[i].LabelName < prof.LabelsIncludeAll[j].LabelName
	})
	require.Equal(t, []fleet.ConfigurationProfileLabel{
		{LabelID: lblA.ID, LabelName: lblA.Name},
		{LabelID: lblB.ID, LabelName: lblB.Name},
	}, prof.LabelsIncludeAll)
	require.Equal(t, "update-win-profile-renamed", prof.Name)
	require.Equal(t, winContent3, downloadProfile(winUUID))
	assertEditedActivity(fleet.ActivityTypeEditedWindowsProfile{}.ActivityName(), "update-win-profile-renamed", "")

	//
	// Apple DDM declaration
	//

	// content edit with the same Identifier
	declContent2 := declJSONWithEcho(declIdent, "v2")
	res = patchProfile(declUUID, "some-other-filename.json", declContent2, nil, http.StatusOK)
	patchResp = decodePatchResp(res)
	require.Equal(t, declUUID, patchResp.ProfileUUID)
	require.Equal(t, declContent2, downloadProfile(declUUID))
	prof = getProfile(declUUID)
	require.Equal(t, "update-apple-decl", prof.Name)
	require.Equal(t, declIdent, prof.Identifier)
	assertEditedActivity(fleet.ActivityTypeEditedDeclarationProfile{}.ActivityName(), "update-apple-decl", declIdent)

	// content upload with a different Identifier is rejected
	res = patchProfile(declUUID, "update-apple-decl.json", declJSONWithEcho("some-other-decl-ident", "v3"), nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "Identifier must match")
	require.Equal(t, declContent2, downloadProfile(declUUID))

	//
	// Android
	//

	// content-only edit
	androidContent2 := []byte(`{"removeUserDisabled": true}`)
	res = patchProfile(androidUUID, "update-android-profile.json", androidContent2, nil, http.StatusOK)
	patchResp = decodePatchResp(res)
	require.Equal(t, androidUUID, patchResp.ProfileUUID)
	require.Equal(t, androidContent2, downloadProfile(androidUUID))
	prof = getProfile(androidUUID)
	require.Equal(t, "update-android-profile", prof.Name)
	assertEditedActivity(fleet.ActivityTypeEditedAndroidProfile{}.ActivityName(), "update-android-profile", "")

	// labels-only edit
	res = patchProfile(androidUUID, "", nil, map[string][]string{"labels_exclude_any": {lblB.Name}}, http.StatusOK)
	patchResp = decodePatchResp(res)
	require.Equal(t, androidUUID, patchResp.ProfileUUID)
	prof = getProfile(androidUUID)
	require.Equal(t, []fleet.ConfigurationProfileLabel{{LabelID: lblB.ID, LabelName: lblB.Name}}, prof.LabelsExcludeAny)
	require.Equal(t, androidContent2, downloadProfile(androidUUID))
	assertEditedActivity(fleet.ActivityTypeEditedAndroidProfile{}.ActivityName(), "update-android-profile", "")
}

func (s *integrationMDMTestSuite) TestListMDMConfigProfiles() {
	t := s.T()
	ctx := context.Background()

	// create some teams
	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	tm3, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team3"})
	require.NoError(t, err)

	// create 5 profiles for no team and team 1, names are A, B, C ... for global and
	// tA, tB, tC ... for team 1. Alternate macOS and Windows profiles.
	for i := range 5 {
		name := string('A' + byte(i))
		if i%2 == 0 {
			prof, err := fleet.NewMDMAppleConfigProfile(mcBytesForTest(name, name+".identifier", name+".uuid"), nil)
			require.NoError(t, err)
			_, err = s.ds.NewMDMAppleConfigProfile(ctx, *prof, nil)
			require.NoError(t, err)

			tprof, err := fleet.NewMDMAppleConfigProfile(mcBytesForTest("t"+name, "t"+name+".identifier", "t"+name+".uuid"), nil)
			require.NoError(t, err)
			tprof.TeamID = &tm1.ID
			_, err = s.ds.NewMDMAppleConfigProfile(ctx, *tprof, nil)
			require.NoError(t, err)
		} else {
			_, err = s.ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: name, SyncML: []byte(`<Replace></Replace>`)}, nil)
			require.NoError(t, err)
			_, err = s.ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{Name: "t" + name, TeamID: &tm1.ID, SyncML: []byte(`<Replace></Replace>`)}, nil)
			require.NoError(t, err)
		}
	}

	lblFoo, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "foo", Query: "select 1"})
	require.NoError(t, err)
	lblBar, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "bar", Query: "select 1"})
	require.NoError(t, err)
	lblBaz, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "baz", Query: "select 1"})
	require.NoError(t, err)

	// create a couple profiles (Win and mac) for team 2, and none for team 3
	tprof, err := fleet.NewMDMAppleConfigProfile(mcBytesForTest("tF", "tF.identifier", "tF.uuid"), nil)
	require.NoError(t, err)
	tprof.TeamID = &tm2.ID
	// make tm2ProfF a "exclude-any" label-based profile
	tprof.LabelsExcludeAny = []fleet.ConfigurationProfileLabel{
		{LabelID: lblFoo.ID, LabelName: lblFoo.Name},
		{LabelID: lblBar.ID, LabelName: lblBar.Name},
	}
	tm2ProfF, err := s.ds.NewMDMAppleConfigProfile(ctx, *tprof, nil)
	require.NoError(t, err)
	// checksum is not returned by New..., so compute it manually
	checkSum := md5.Sum(tm2ProfF.Mobileconfig) // nolint:gosec // used only for test
	tm2ProfF.Checksum = checkSum[:]

	// make tm2ProfG a "include-all" label-based profile
	tm2ProfG, err := s.ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		Name:   "tG",
		TeamID: &tm2.ID,
		SyncML: []byte(`<Add></Add>`),
		LabelsIncludeAll: []fleet.ConfigurationProfileLabel{
			{LabelID: lblFoo.ID, LabelName: lblFoo.Name},
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
		},
	}, nil)
	require.NoError(t, err)

	// make tm2ProfH a "include-any" label-based profile
	tm2ProfH, err := s.ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		Name:   "tH",
		TeamID: &tm2.ID,
		SyncML: []byte(`<Add></Add>`),
		LabelsIncludeAny: []fleet.ConfigurationProfileLabel{
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
			{LabelID: lblBaz.ID, LabelName: lblBaz.Name},
		},
	}, nil)
	require.NoError(t, err)

	// simulate a broken label by nullifying label_id in the join table
	// (direct DeleteLabel is now blocked when referenced by a profile)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE mdm_configuration_profile_labels SET label_id = NULL WHERE label_id = ?`, lblFoo.ID)
		return err
	})

	// test that all fields are correctly returned with team 2
	var listResp listMDMConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", nil, http.StatusOK, &listResp, "team_id", fmt.Sprint(tm2.ID))
	require.Len(t, listResp.Profiles, 3)
	require.NotZero(t, listResp.Profiles[0].CreatedAt)
	require.NotZero(t, listResp.Profiles[0].UploadedAt)
	require.NotZero(t, listResp.Profiles[1].CreatedAt)
	require.NotZero(t, listResp.Profiles[1].UploadedAt)
	listResp.Profiles[0].CreatedAt, listResp.Profiles[0].UploadedAt = time.Time{}, time.Time{}
	listResp.Profiles[1].CreatedAt, listResp.Profiles[1].UploadedAt = time.Time{}, time.Time{}
	listResp.Profiles[2].CreatedAt, listResp.Profiles[2].UploadedAt = time.Time{}, time.Time{}
	require.Equal(t, &fleet.MDMConfigProfilePayload{
		ProfileUUID:        tm2ProfF.ProfileUUID,
		TeamID:             tm2ProfF.TeamID,
		Name:               tm2ProfF.Name,
		PayloadDisplayName: tm2ProfF.Name,
		Platform:           "darwin",
		Identifier:         tm2ProfF.Identifier,
		Checksum:           tm2ProfF.Checksum,
		Scope:              string(fleet.PayloadScopeSystem),
		// labels are ordered by name
		LabelsExcludeAny: []fleet.ConfigurationProfileLabel{
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
			{LabelID: 0, LabelName: lblFoo.Name, Broken: true},
		},
	}, listResp.Profiles[0])
	require.Equal(t, &fleet.MDMConfigProfilePayload{
		ProfileUUID: tm2ProfG.ProfileUUID,
		TeamID:      tm2ProfG.TeamID,
		Name:        tm2ProfG.Name,
		Platform:    "windows",
		// labels are ordered by name
		LabelsIncludeAll: []fleet.ConfigurationProfileLabel{
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
			{LabelID: 0, LabelName: lblFoo.Name, Broken: true},
		},
	}, listResp.Profiles[1])
	require.Equal(t, &fleet.MDMConfigProfilePayload{
		ProfileUUID: tm2ProfH.ProfileUUID,
		TeamID:      tm2ProfH.TeamID,
		Name:        tm2ProfH.Name,
		Platform:    "windows",
		// labels are ordered by name
		LabelsIncludeAny: []fleet.ConfigurationProfileLabel{
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
			{LabelID: lblBaz.ID, LabelName: lblBaz.Name},
		},
	}, listResp.Profiles[2])

	// get the specific include-all label-based profile returns the information
	var getProfResp getMDMConfigProfileResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles/"+tm2ProfG.ProfileUUID, nil, http.StatusOK, &getProfResp)
	getProfResp.CreatedAt, getProfResp.UploadedAt = time.Time{}, time.Time{}
	require.Equal(t, &fleet.MDMConfigProfilePayload{
		ProfileUUID: tm2ProfG.ProfileUUID,
		TeamID:      tm2ProfG.TeamID,
		Name:        tm2ProfG.Name,
		Platform:    "windows",
		// labels are ordered by name
		LabelsIncludeAll: []fleet.ConfigurationProfileLabel{
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
			{LabelID: 0, LabelName: lblFoo.Name, Broken: true},
		},
	}, getProfResp.MDMConfigProfilePayload)

	// get the specific exclude-any label-based profile returns the information
	getProfResp = getMDMConfigProfileResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles/"+tm2ProfF.ProfileUUID, nil, http.StatusOK, &getProfResp)
	getProfResp.CreatedAt, getProfResp.UploadedAt = time.Time{}, time.Time{}
	require.Equal(t, &fleet.MDMConfigProfilePayload{
		ProfileUUID:        tm2ProfF.ProfileUUID,
		TeamID:             tm2ProfF.TeamID,
		Name:               tm2ProfF.Name,
		PayloadDisplayName: tm2ProfF.Name,
		Platform:           "darwin",
		Identifier:         tm2ProfF.Identifier,
		Checksum:           tm2ProfF.Checksum,
		Scope:              string(fleet.PayloadScopeSystem),
		// labels are ordered by name
		LabelsExcludeAny: []fleet.ConfigurationProfileLabel{
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
			{LabelID: 0, LabelName: lblFoo.Name, Broken: true},
		},
	}, getProfResp.MDMConfigProfilePayload)

	// get the specific include-any label-based profile returns the information
	getProfResp = getMDMConfigProfileResponse{}
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles/"+tm2ProfH.ProfileUUID, nil, http.StatusOK, &getProfResp)
	getProfResp.CreatedAt, getProfResp.UploadedAt = time.Time{}, time.Time{}
	require.Equal(t, &fleet.MDMConfigProfilePayload{
		ProfileUUID: tm2ProfH.ProfileUUID,
		TeamID:      tm2ProfH.TeamID,
		Name:        tm2ProfH.Name,
		Platform:    "windows",
		// labels are ordered by name
		LabelsIncludeAny: []fleet.ConfigurationProfileLabel{
			{LabelID: lblBar.ID, LabelName: lblBar.Name},
			{LabelID: lblBaz.ID, LabelName: lblBaz.Name},
		},
	}, getProfResp.MDMConfigProfilePayload)
	// list for a non-existing team returns 404
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", nil, http.StatusNotFound, &listResp, "team_id", "99999")

	cases := []struct {
		queries   []string // alternate query name and value
		teamID    *uint
		wantNames []string
		wantMeta  *fleet.PaginationMetadata
	}{
		{
			wantNames: []string{"A", "B", "C", "D", "E"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2"},
			wantNames: []string{"A", "B"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2", "page", "1"},
			wantNames: []string{"C", "D"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "2", "page", "2"},
			wantNames: []string{"E"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3"},
			teamID:    &tm1.ID,
			wantNames: []string{"tA", "tB", "tC"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "3", "page", "1"},
			teamID:    &tm1.ID,
			wantNames: []string{"tD", "tE"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3", "page", "2"},
			teamID:    &tm1.ID,
			wantNames: nil,
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3"},
			teamID:    &tm2.ID,
			wantNames: []string{"tF", "tG", "tH"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2"},
			teamID:    &tm3.ID,
			wantNames: nil,
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%v: %#v", c.teamID, c.queries), func(t *testing.T) {
			var listResp listMDMConfigProfilesResponse
			queryArgs := c.queries
			if c.teamID != nil {
				queryArgs = append(queryArgs, "team_id", fmt.Sprint(*c.teamID))
			}
			s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", nil, http.StatusOK, &listResp, queryArgs...)

			require.Len(t, listResp.Profiles, len(c.wantNames))
			require.Equal(t, c.wantMeta, listResp.Meta)

			var gotNames []string
			if len(listResp.Profiles) > 0 {
				gotNames = make([]string, len(listResp.Profiles))
				for i, p := range listResp.Profiles {
					gotNames[i] = p.Name
					if p.Name == "tG" {
						require.Len(t, p.LabelsIncludeAll, 2)
					} else {
						require.Nil(t, p.LabelsIncludeAll)
					}
					if p.Name == "tF" {
						require.Len(t, p.LabelsExcludeAny, 2)
					} else {
						require.Nil(t, p.LabelsExcludeAny)
					}
					if c.teamID == nil {
						// we set it to 0 for global
						require.NotNil(t, p.TeamID)
						require.Zero(t, *p.TeamID)
					} else {
						require.NotNil(t, p.TeamID)
						require.Equal(t, *c.teamID, *p.TeamID)
					}
					require.NotEmpty(t, p.Platform)
				}
			}
			require.Equal(t, c.wantNames, gotNames)
		})
	}
}

func (s *integrationMDMTestSuite) TestHostMDMProfilesExcludeLabels() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := context.Background()

	triggerReconcileProfiles := func() {
		s.awaitTriggerProfileSchedule(t)
		// this will only mark them as "pending", as the response to confirm
		// profile deployment is asynchronous, so we simulate it here by
		// updating any "pending" (not NULL) profiles to "verifying"
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			if _, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET status = ? WHERE status = ?`, fleet.OSSettingsVerifying, fleet.OSSettingsPending); err != nil {
				return err
			}
			if _, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_declarations SET status = ? WHERE status = ?`, fleet.OSSettingsVerifying, fleet.OSSettingsPending); err != nil {
				return err
			}
			return nil
		})
	}

	// run the crons immediately, will create the Fleet-controlled profiles that
	// will then be expected to be applied (e.g. com.fleetdm.fleetd.config and
	// com.fleetdm.caroot)
	// first create the no-team enroll secret (required to create the fleet profiles)
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret",
		applyEnrollSecretSpecRequest{
			Spec: &fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: "super-global-secret"}}},
		}, http.StatusOK, &applyResp)
	s.awaitTriggerProfileSchedule(t)

	// create an Apple and a Windows host
	appleHost, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	// create a few labels
	labels := make([]*fleet.Label, 5)
	for i := range labels {
		label, err := s.ds.NewLabel(ctx, &fleet.Label{Name: fmt.Sprintf("label-%d", i), Query: "select 1;"})
		require.NoError(t, err)
		labels[i] = label
	}
	// simulate reporting label results for those hosts
	appleHost.LabelUpdatedAt = time.Now()
	err := s.ds.UpdateHost(ctx, appleHost)
	require.NoError(t, err)

	// set an Apple profile and declaration and a Windows profile
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "A1", Contents: mobileconfigForTest("A1", "A1"), LabelsExcludeAny: []string{labels[0].Name, labels[1].Name}},
		{Name: "D3", Contents: declarationForTest("D3"), LabelsExcludeAny: []string{labels[4].Name}},
	}}, http.StatusNoContent)

	// hosts are not members of any label yet, so running the cron applies the labels
	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "A1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	err = s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+appleHost.UUID)
	require.NoError(t, err)

	// simulate the reconcile profiles deployment
	triggerReconcileProfiles()
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "A1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// mark some profiles as verified (despite accepting a HostMacOSProfile struct, it supports Windows too)
	err = apple_mdm.VerifyHostMDMProfiles(ctx, s.ds, appleHost, map[string]*fleet.HostMacOSProfile{
		"A1": {Identifier: "A1", DisplayName: "A1", InstallDate: time.Now()},
	})
	require.NoError(t, err)

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "A1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerified},
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// make hosts members of labels [1], [2], [3] and [4], meaning that none of the profiles apply anymore
	err = s.ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{
		{labels[1].ID, appleHost.ID},
		{labels[2].ID, appleHost.ID},
		{labels[3].ID, appleHost.ID},
		{labels[4].ID, appleHost.ID},
	})
	require.NoError(t, err)

	s.awaitTriggerProfileSchedule(t)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "A1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// remove membership of [4] for Apple, meaning
	// that only D3 will be installed on Apple (as the Windows host is still
	// member of an excluded label)
	err = s.ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{
		{labels[4].ID, appleHost.ID},
	})
	require.NoError(t, err)

	s.awaitTriggerProfileSchedule(t)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "A1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// remove label [3] as an excluded label for the Windows profile, meaning
	// that the host now meets the requirement to install.
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "A1", Contents: mobileconfigForTest("A1", "A1"), LabelsExcludeAny: []string{labels[0].Name, labels[1].Name}},
		{Name: "W2", Contents: syncMLForTest("./Foo/W2"), LabelsExcludeAny: []string{labels[2].Name}},
		{Name: "D3", Contents: declarationForTest("D3"), LabelsExcludeAny: []string{labels[4].Name}},
	}}, http.StatusNoContent)

	s.awaitTriggerProfileSchedule(t)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "A1", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// simulate the reconcile profiles deployment and mark as verified
	triggerReconcileProfiles()

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// simulate the A1 profile being broken by nullifying labels[1] in the join table
	// (direct DeleteLabel is now blocked when a label is referenced by a profile)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE mdm_configuration_profile_labels SET label_id = NULL WHERE label_id = ?`, labels[1].ID)
		return err
	})

	// it doesn't get installed to the Apple host, as it is broken
	triggerReconcileProfiles()
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// it also doesn't get installed to a new host not a member of any labels
	appleHost2, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	// simulate reporting label results for the new host, otherwise exclude-any
	// profiles are withheld until label membership is known
	require.NoError(t, s.ds.AsyncBatchUpdateLabelTimestamp(ctx, []uint{appleHost2.ID}, time.Now()))
	s.awaitRunAppleMDMWorkerSchedule()
	err = s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+appleHost2.UUID)
	require.NoError(t, err)
	triggerReconcileProfiles()
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		appleHost2: {
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// simulate D3 and W2 profiles being broken by nullifying labels[2] and [4]
	// in the join table (DeleteLabel is now blocked when referenced by a profile).
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE mdm_configuration_profile_labels SET label_id = NULL WHERE label_id = ?`, labels[4].ID)
		return err
	})

	triggerReconcileProfiles()
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
		appleHost2: {
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
}

func (s *integrationMDMTestSuite) TestMDMProfilesIncludeAnyLabels() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)

	triggerReconcileProfiles := func() {
		s.awaitTriggerProfileSchedule(t)
		// this will only mark them as "pending", as the response to confirm
		// profile deployment is asynchronous, so we simulate it here by
		// updating any "pending" (not NULL) profiles to "verifying"
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			if _, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET status = ? WHERE status = ?`, fleet.OSSettingsVerifying, fleet.OSSettingsPending); err != nil {
				return err
			}
			if _, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_declarations SET status = ? WHERE status = ?`, fleet.OSSettingsVerifying, fleet.OSSettingsPending); err != nil {
				return err
			}
			if _, err := q.ExecContext(ctx, `UPDATE host_mdm_windows_profiles SET status = ? WHERE status = ?`, fleet.OSSettingsVerifying, fleet.OSSettingsPending); err != nil {
				return err
			}
			return nil
		})
	}

	// run the crons immediately, will create the Fleet-controlled profiles that
	// will then be expected to be applied (e.g. com.fleetdm.fleetd.config and
	// com.fleetdm.caroot)
	// first create the no-team enroll secret (required to create the fleet profiles)
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret",
		applyEnrollSecretSpecRequest{
			Spec: &fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: "super-global-secret"}}},
		}, http.StatusOK, &applyResp)
	s.awaitTriggerProfileSchedule(t)

	// create an Apple and a Windows host
	appleHost, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	windowsHost, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.awaitRunAppleMDMWorkerSchedule()
	err := s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+appleHost.UUID)
	require.NoError(t, err)

	// create a few labels, we'll use the first five for "exclude any" profiles and the remaining for "include any"
	labels := make([]*fleet.Label, 10)
	for i := range labels {
		label, err := s.ds.NewLabel(ctx, &fleet.Label{Name: fmt.Sprintf("label-%d", i), Query: "select 1;"})
		require.NoError(t, err)
		labels[i] = label
	}
	// simulate reporting label results for those hosts
	appleHost.LabelUpdatedAt = time.Now()
	windowsHost.LabelUpdatedAt = time.Now()
	err = s.ds.UpdateHost(ctx, appleHost)
	require.NoError(t, err)
	err = s.ds.UpdateHost(ctx, windowsHost)
	require.NoError(t, err)

	// set up some Apple profiles and declarations and Windows profiles
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "A1", Contents: mobileconfigForTest("A1", "A1"), LabelsIncludeAny: []string{labels[0].Name, labels[1].Name}},
		{Name: "W2", Contents: syncMLForTest("./Foo/W2"), LabelsIncludeAny: []string{labels[2].Name, labels[3].Name}},
		{Name: "D3", Contents: declarationForTest("D3"), LabelsIncludeAny: []string{labels[4].Name}},
	}}, http.StatusNoContent)

	// hosts are not members of any label yet, so running the cron applies no labels
	s.awaitTriggerProfileSchedule(t)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		windowsHost: {},
	})

	// make hosts members of labels [1], [2], [3] and [4], meaning that each of the "include any"
	// labels will now match at least one host
	err = s.ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{
		{labels[0].ID, appleHost.ID},
		{labels[1].ID, appleHost.ID},
		{labels[2].ID, appleHost.ID},
		{labels[3].ID, appleHost.ID},
		{labels[4].ID, appleHost.ID},
		{labels[1].ID, windowsHost.ID},
		{labels[2].ID, windowsHost.ID},
		{labels[3].ID, windowsHost.ID},
		{labels[4].ID, windowsHost.ID},
	})
	require.NoError(t, err)

	triggerReconcileProfiles()
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "A1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		windowsHost: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})

	// remove membership of labels [2] for Windows, and [1] and [4] for Apple, meaning
	// that D3 will be removed on Apple, A1 will remain on Apple because the host is still a member
	// of [0], and W2 will remain on Windows because the host is still a member of [3]
	err = s.ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{
		{labels[1].ID, appleHost.ID},
		{labels[4].ID, appleHost.ID},
		{labels[2].ID, windowsHost.ID},
	})
	require.NoError(t, err)

	s.awaitTriggerProfileSchedule(t)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		appleHost: {
			{Identifier: "A1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "D3", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		windowsHost: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
}

// TestAppleDDMSecretVariablesUpload tests uploading DDM profiles with secrets via the /configuration_profiles endpoint
func (s *integrationMDMTestSuite) TestAppleDDMSecretVariablesUpload() {
	tmpl := `
{
	"Type": "com.apple.configuration.decl%d",
	"Identifier": "com.fleet.config%d",
	"Payload": {
		"ServiceType": "com.apple.bash%d"
	}
}`

	newProfileBytes := func(i int) []byte {
		return fmt.Appendf(nil, tmpl, i, i, i)
	}

	getProfileContents := func(profileUUID string) string {
		profile, err := s.ds.GetMDMAppleDeclaration(context.Background(), profileUUID)
		s.Require().NoError(err)
		s.NotNil(profile.SecretsUpdatedAt)
		return string(profile.RawJSON)
	}

	s.testSecretVariablesUpload(newProfileBytes, getProfileContents, "json", "darwin", false)
}

func (s *integrationMDMTestSuite) testSecretVariablesUpload(newProfileBytes func(i int) []byte,
	getProfileContents func(profileUUID string) string, fileExtension string, platform string, testWholeProfileSecret bool,
) {
	t := s.T()
	numProfiles := 1
	profiles := [][]byte{newProfileBytes(0)}

	myBash := "com.apple.bash0"
	profiles[0] = []byte(strings.ReplaceAll(string(profiles[0]), myBash, "$"+fleet.ServerSecretPrefix+"BASH"))

	body, headers := generateNewProfileMultipartRequest(
		t, "secret-config0."+fileExtension, profiles[0], s.token, nil,
	)
	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusUnprocessableEntity, headers)
	assertBodyContains(t, res, `Secret variable \"$FLEET_SECRET_BASH\" missing`)

	// Add secret(s) to server
	req := fleet.CreateSecretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  "FLEET_SECRET_BASH",
				Value: myBash,
			},
		},
	}

	if testWholeProfileSecret {
		secretProfile := newProfileBytes(1)
		profiles = append(profiles, []byte("${"+fleet.ServerSecretPrefix+"PROFILE}"))

		req.SecretVariables = append(req.SecretVariables, fleet.SecretVariable{
			Name:  "FLEET_SECRET_PROFILE",
			Value: string(secretProfile),
		})
		numProfiles++
	}

	secretResp := fleet.CreateSecretVariablesResponse{}
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &secretResp)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusOK, headers)
	var resp newMDMConfigProfileResponse
	err := json.NewDecoder(res.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.ProfileUUID)

	if testWholeProfileSecret {
		body, headers = generateNewProfileMultipartRequest(
			t, "secret-config1."+fileExtension, profiles[1], s.token, nil,
		)
		s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &secretResp)
		res = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusOK, headers)
		err = json.NewDecoder(res.Body).Decode(&resp)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.ProfileUUID)
	}

	var listResp listMDMConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles", &listMDMConfigProfilesRequest{}, http.StatusOK, &listResp)
	require.Len(t, listResp.Profiles, numProfiles)
	profileUUIDs := make([]string, numProfiles)
	for _, p := range listResp.Profiles {
		switch p.Name {
		case "secret-config0":
			assert.Equal(t, platform, p.Platform)
			profileUUIDs[0] = p.ProfileUUID
		case "secret-config1":
			assert.Equal(t, platform, p.Platform)
			profileUUIDs[1] = p.ProfileUUID
		default:
			t.Errorf("unexpected profile %s", p.Name)
		}
	}

	// Check that contents are masking secret values
	for i := 0; i < numProfiles; i++ {
		assert.Equal(t, string(profiles[i]), getProfileContents(profileUUIDs[i]))
	}

	// Delete profiles -- make sure there is no issue deleting profiles with secrets
	for i := 0; i < numProfiles; i++ {
		s.Do("DELETE", "/api/latest/fleet/configuration_profiles/"+profileUUIDs[i], nil, http.StatusOK)
	}
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles", &listMDMConfigProfilesRequest{}, http.StatusOK, &listResp)
	require.Empty(t, listResp.Profiles)
}

// TestAppleConfigSecretVariablesUpload tests uploading Apple config profiles with secrets via the /configuration_profiles endpoint
func (s *integrationMDMTestSuite) TestAppleConfigSecretVariablesUpload() {
	tmpl := `
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadDescription</key>
  <string>For secret variables</string>
  <key>PayloadDisplayName</key>
  <string>secret-config%d</string>
  <key>PayloadIdentifier</key>
  <string>PI%d</string>
  <key>PayloadType</key>
  <string>Configuration</string>
  <key>PayloadUUID</key>
  <string>%d</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>Bash</key>
      <string>$FLEET_SECRET_BASH</string>
      <key>PayloadDisplayName</key>
      <string>secret payload</string>
      <key>PayloadIdentifier</key>
      <string>com.test.secret</string>
      <key>PayloadType</key>
      <string>com.test.secretd</string>
      <key>PayloadUUID</key>
      <string>476F5334-D501-4768-9A31-1A18A4E1E808</string>
      <key>PayloadVersion</key>
      <integer>1</integer>
    </dict>
  </array>
</dict>
</plist>`

	newProfileBytes := func(i int) []byte {
		return []byte(fmt.Sprintf(tmpl, i, i, i))
	}

	getProfileContents := func(profileUUID string) string {
		profile, err := s.ds.GetMDMAppleConfigProfile(context.Background(), profileUUID)
		s.Require().NoError(err)
		s.NotNil(profile.SecretsUpdatedAt)
		return string(profile.Mobileconfig)
	}

	s.testSecretVariablesUpload(newProfileBytes, getProfileContents, "mobileconfig", "darwin", true)
}

// TestWindowsConfigSecretVariablesUpload tests uploading Windows profiles with secrets via the /configuration_profiles endpoint
func (s *integrationMDMTestSuite) TestWindowsConfigSecretVariablesUpload() {
	tmpl := `
<Replace>
    <Item>
        <Meta>
            <Format xmlns="syncml:metinf">int</Format>
        </Meta>
        <Target>
            <LocURI>./Device/Vendor/MSFT/Policy/Config/System/DisableOneDriveFileSync</LocURI>
        </Target>
        <Data>$FLEET_SECRET_BASH</Data>
    </Item>
</Replace>
`

	newProfileBytes := func(i int) []byte {
		return []byte(fmt.Sprintf(tmpl, i, i, i))
	}

	getProfileContents := func(profileUUID string) string {
		profile, err := s.ds.GetMDMWindowsConfigProfile(context.Background(), profileUUID)
		s.Require().NoError(err)
		return string(profile.SyncML)
	}

	s.testSecretVariablesUpload(newProfileBytes, getProfileContents, "xml", "windows", true)
}

func (s *integrationMDMTestSuite) TestHostMDMAndroidProfilesStatus() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)

	testTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "TestTeam", Secrets: []*fleet.EnrollSecret{{Secret: uuid.NewString()}}})
	require.NoError(t, err)

	enterpriseID := s.enableAndroidMDM(t)

	s.runWorkerUntilDoneWithChecks(true)

	host1, deviceInfo1, pubSubToken := s.createAndEnrollAndroidDevice(t, "host-1", &testTeam.ID, false)
	s.createAndEnrollAndroidDevice(t, "host-2", &testTeam.ID, false)

	var hosts listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hosts)
	assert.Len(t, hosts.Hosts, 2)

	bytes := []byte(`{
  "removeUserDisabled": false
}`)

	fields := make(map[string][]string)
	fields["team_id"] = []string{fmt.Sprintf("%d", testTeam.ID)}
	body, headers := generateNewProfileMultipartRequest(t, "remove-user-disabled.json", bytes, s.token, fields)
	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusOK, headers)
	require.NotNil(t, res)

	// profiles should be added with NULL status even before the cron job (s.awaitTriggerAndroidProfileSchedule(t))
	var profiles []fleet.HostMDMAndroidProfile
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		err := sqlx.SelectContext(ctx, q, &profiles, "SELECT host_uuid, status, operation_type FROM host_mdm_android_profiles")
		require.NoError(t, err)
		return nil
	})

	require.Len(t, profiles, 2)
	require.Nil(t, profiles[0].Status)
	require.Nil(t, profiles[1].Status)

	overrideProfile1 := []byte(`{
  "maximumTimeToLock": "1"
}`)
	overrideProfile2 := []byte(`{
  "maximumTimeToLock": "2"
}`)

	body, headers = generateNewProfileMultipartRequest(t, "maximum-time-to-lock-1.json", overrideProfile1, s.token, fields)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusOK, headers)
	require.NotNil(t, res)

	body, headers = generateNewProfileMultipartRequest(t, "maximum-time-to-lock-2.json", overrideProfile2, s.token, fields)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusOK, headers)
	require.NotNil(t, res)

	s.awaitTriggerAndroidProfileSchedule(t)

	getHostProfiles := func(hostID uint, wantStatus []string) {
		var hostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostID), nil, http.StatusOK, &hostResp)
		require.NotNil(t, hostResp.Host.MDM.Profiles)
		require.Len(t, *hostResp.Host.MDM.Profiles, len(wantStatus))
		actualProfileStatuses := make([]string, 0, len(wantStatus))
		for _, p := range *hostResp.Host.MDM.Profiles {
			if p.Status != nil {
				actualProfileStatuses = append(actualProfileStatuses, *p.Status)
			}
		}
		require.ElementsMatch(t, wantStatus, actualProfileStatuses)
	}

	getHostProfiles(host1.ID, []string{string(fleet.MDMDeliveryPending), string(fleet.MDMDeliveryPending), string(fleet.MDMDeliveryFailed)})

	// send a pub-sub status report
	policyName := fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host1.UUID)
	reportMsg := statusReportMessageWithEnterpriseSpecificID(
		t,
		androidmanagement.Device{
			Name:                 deviceInfo1.Name,
			EnrollmentTokenData:  deviceInfo1.EnrollmentTokenData,
			AppliedPolicyName:    policyName,
			AppliedPolicyVersion: 2,
			LastPolicySyncTime:   time.Now().Format(time.RFC3339Nano),
		},
		host1.UUID,
	)
	req := android_service.PubSubPushRequest{PubSubMessage: *reportMsg}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubSubToken.Value))

	// Profiles that failed because they were overriden should stay failed
	getHostProfiles(host1.ID, []string{string(fleet.MDMDeliveryVerified), string(fleet.MDMDeliveryVerified), string(fleet.MDMDeliveryFailed)})
}

func (s *integrationMDMTestSuite) TestConfigProfileNameAndDescription() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	teamFields := func(extra map[string][]string) map[string][]string {
		fields := map[string][]string{"team_id": {fmt.Sprint(team.ID)}}
		maps.Copy(fields, extra)
		return fields
	}

	create := func(fileName string, content []byte, extra map[string][]string, wantStatus int) string {
		body, headers := generateNewProfileMultipartRequest(t, fileName, content, s.token, teamFields(extra))
		res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), wantStatus, headers)
		defer res.Body.Close()
		if wantStatus != http.StatusOK {
			return ""
		}
		var resp newMDMConfigProfileResponse
		require.NoError(t, json.NewDecoder(res.Body).Decode(&resp))
		return resp.ProfileUUID
	}
	update := func(profileUUID, fileName string, content []byte, fields map[string][]string, wantStatus int) {
		body, headers := generateMultipartRequest(t, "profile", fileName, content, s.token, fields)
		res := s.DoRawWithHeaders("PATCH", "/api/latest/fleet/configuration_profiles/"+profileUUID, body.Bytes(), wantStatus, headers)
		res.Body.Close()
	}
	get := func(profileUUID string) *fleet.MDMConfigProfilePayload {
		var resp getMDMConfigProfileResponse
		s.DoJSON("GET", "/api/latest/fleet/configuration_profiles/"+profileUUID, getMDMConfigProfileRequest{}, http.StatusOK, &resp)
		return resp.MDMConfigProfilePayload
	}
	list := func() map[string]*fleet.MDMConfigProfilePayload {
		var resp listMDMConfigProfilesResponse
		s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", listMDMConfigProfilesRequest{}, http.StatusOK, &resp, "team_id", fmt.Sprint(team.ID))
		byUUID := make(map[string]*fleet.MDMConfigProfilePayload, len(resp.Profiles))
		for _, p := range resp.Profiles {
			byUUID[p.ProfileUUID] = p
		}
		return byUUID
	}

	// create: explicit name and description on every type, derived
	// name otherwise
	appleUUID := create("apple.mobileconfig", mobileconfigForTest("Payload Apple", "com.test.named.apple"),
		map[string][]string{"name": {"Custom Apple"}, "description": {"Blocks the camera"}}, http.StatusOK)
	appleDerivedUUID := create("apple2.mobileconfig", mobileconfigForTest("Payload Apple 2", "com.test.derived.apple"), nil, http.StatusOK)
	winUUID := create("firewall.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Firewall/Test"),
		map[string][]string{"name": {"Custom Windows"}, "description": {"Firewall rules"}}, http.StatusOK)
	winDerivedUUID := create("bitlocker-report.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Bitlocker/Test"), nil, http.StatusOK)
	declUUID := create("decl.json", declBytesForTest("com.test.named.decl", "named"),
		map[string][]string{"name": {"Custom Declaration"}, "description": {"A declaration"}}, http.StatusOK)

	apple := get(appleUUID)
	require.Equal(t, "Custom Apple", apple.Name)
	require.Equal(t, "Blocks the camera", apple.Description)
	require.Equal(t, "Payload Apple", apple.PayloadDisplayName)

	appleDerived := get(appleDerivedUUID)
	require.Equal(t, "Payload Apple 2", appleDerived.Name)
	require.Empty(t, appleDerived.Description)
	require.Equal(t, "Payload Apple 2", appleDerived.PayloadDisplayName)

	win := get(winUUID)
	require.Equal(t, "Custom Windows", win.Name)
	require.Equal(t, "Firewall rules", win.Description)
	require.Empty(t, win.PayloadDisplayName)
	require.Equal(t, "bitlocker-report", get(winDerivedUUID).Name)

	decl := get(declUUID)
	require.Equal(t, "Custom Declaration", decl.Name)
	require.Equal(t, "A declaration", decl.Description)
	require.Empty(t, decl.PayloadDisplayName)

	listed := list()
	require.Len(t, listed, 5)
	listedApple, listedAppleDerived, listedWin, listedDecl := listed[appleUUID], listed[appleDerivedUUID], listed[winUUID], listed[declUUID]
	require.NotNil(t, listedApple)
	require.NotNil(t, listedAppleDerived)
	require.NotNil(t, listedWin)
	require.NotNil(t, listedDecl)
	require.Equal(t, "Blocks the camera", listedApple.Description)
	require.Equal(t, "Payload Apple", listedApple.PayloadDisplayName)
	require.Equal(t, "Payload Apple 2", listedAppleDerived.PayloadDisplayName)
	require.Equal(t, "Firewall rules", listedWin.Description)
	require.Empty(t, listedWin.PayloadDisplayName)
	require.Equal(t, "A declaration", listedDecl.Description)

	// create: validation
	create("blank.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Blank/Test"),
		map[string][]string{"name": {"   "}}, http.StatusUnprocessableEntity)
	// Windows validation keeps its existing 400s for long and reserved names
	create("long.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Long/Test"),
		map[string][]string{"name": {strings.Repeat("n", fleet.MaxProfileNameLength+1)}}, http.StatusBadRequest)
	create("longdesc.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/LongDesc/Test"),
		map[string][]string{"description": {strings.Repeat("d", fleet.MDMProfileMaxDescriptionLen+1)}}, http.StatusUnprocessableEntity)
	create("reserved.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Reserved/Test"),
		map[string][]string{"name": {servermdm.FleetdConfigProfileName}}, http.StatusBadRequest)
	create("secret.mobileconfig", mobileconfigForTest("Payload Secret", "com.test.secret.apple"),
		map[string][]string{"name": {"Wi-Fi $FLEET_SECRET_PSK"}}, http.StatusUnprocessableEntity)
	// same name as the Apple profile, on another platform, in the same team
	create("dup.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Dup/Test"),
		map[string][]string{"name": {"Custom Apple"}}, http.StatusConflict)

	// update: description only, nothing else moves
	update(appleUUID, "", nil, map[string][]string{"description": {"Blocks the camera for real"}}, http.StatusOK)
	updated := get(appleUUID)
	require.Equal(t, "Custom Apple", updated.Name)
	require.Equal(t, "Blocks the camera for real", updated.Description)
	require.Equal(t, apple.UploadedAt, updated.UploadedAt)
	require.Equal(t, apple.Checksum, updated.Checksum)

	// update: an empty description clears it
	update(appleUUID, "", nil, map[string][]string{"description": {""}}, http.StatusOK)
	require.Empty(t, get(appleUUID).Description)

	// update: a replacement file with another PayloadDisplayName keeps the name
	update(appleUUID, "apple.mobileconfig", mobileconfigForTest("Renamed Payload", "com.test.named.apple"), nil, http.StatusOK)
	updated = get(appleUUID)
	require.Equal(t, "Custom Apple", updated.Name)
	require.Equal(t, "Renamed Payload", updated.PayloadDisplayName)
	require.NotEqual(t, apple.Checksum, updated.Checksum)

	// update: rename a Windows profile without new content
	update(winUUID, "", nil, map[string][]string{"name": {"Renamed Windows"}}, http.StatusOK)
	renamedWin := get(winUUID)
	require.Equal(t, "Renamed Windows", renamedWin.Name)
	require.Equal(t, "Firewall rules", renamedWin.Description)
	require.Equal(t, win.UploadedAt, renamedWin.UploadedAt)

	// update: a replacement Windows file doesn't rename from the file name
	update(winUUID, "something-else.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Firewall/Test2"), nil, http.StatusOK)
	require.Equal(t, "Renamed Windows", get(winUUID).Name)

	// update: declarations can be renamed too
	update(declUUID, "", nil, map[string][]string{"name": {"Renamed Declaration"}, "description": {""}}, http.StatusOK)
	renamedDecl := get(declUUID)
	require.Equal(t, "Renamed Declaration", renamedDecl.Name)
	require.Empty(t, renamedDecl.Description)

	// update: collisions and limits
	update(winUUID, "", nil, map[string][]string{"name": {"Custom Apple"}}, http.StatusConflict)
	update(declUUID, "", nil, map[string][]string{"name": {"Custom Apple"}}, http.StatusConflict)
	update(appleUUID, "", nil, map[string][]string{"name": {" "}}, http.StatusUnprocessableEntity)
	update(appleUUID, "", nil, map[string][]string{"name": {servermdm.FleetdConfigProfileName}}, http.StatusUnprocessableEntity)

	// batch: name and description are stored, display_name still works
	// as an alias
	batch := func(profiles []fleet.BatchModifyMDMConfigProfilePayload, wantStatus int) {
		s.Do("POST", "/api/latest/fleet/configuration_profiles/batch", batchModifyMDMConfigProfilesRequest{
			ConfigurationProfiles: profiles,
		}, wantStatus, "team_id", fmt.Sprint(team.ID))
	}
	batch([]fleet.BatchModifyMDMConfigProfilePayload{
		// padded like a hand-written YAML value; stored trimmed
		{Profile: mobileconfigForTest("Payload Batch", "com.test.batch.apple"), Name: "  Batch Apple ", Description: " batch apple\n"},
		{Profile: syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Batch/Test"), DisplayName: "Batch Windows", Description: "batch windows"},
		{Profile: declBytesForTest("com.test.batch.decl", "batch"), Name: "Batch Declaration"},
	}, http.StatusNoContent)
	listed = list()
	require.Len(t, listed, 3)
	byName := make(map[string]*fleet.MDMConfigProfilePayload, len(listed))
	for _, p := range listed {
		byName[p.Name] = p
	}
	batchApple, batchWin, batchDecl := byName["Batch Apple"], byName["Batch Windows"], byName["Batch Declaration"]
	require.NotNil(t, batchApple)
	require.NotNil(t, batchWin)
	require.NotNil(t, batchDecl)
	require.Equal(t, "batch apple", batchApple.Description)
	require.Equal(t, "Payload Batch", batchApple.PayloadDisplayName)
	require.Equal(t, "batch windows", batchWin.Description)
	require.Empty(t, batchDecl.Description)

	// batch: name and display_name must agree when both are given
	batch([]fleet.BatchModifyMDMConfigProfilePayload{
		{Profile: syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Batch/Test"), Name: "A", DisplayName: "B"},
	}, http.StatusUnprocessableEntity)
	batch([]fleet.BatchModifyMDMConfigProfilePayload{
		{Profile: syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Batch/Test"), Name: "Same", DisplayName: "Same"},
	}, http.StatusNoContent)
	require.Len(t, list(), 1)

	// a name of only whitespace is rejected rather than derived
	batch([]fleet.BatchModifyMDMConfigProfilePayload{
		{Profile: syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Batch/Test"), Name: "   "},
	}, http.StatusUnprocessableEntity)

	// display_name alone doesn't rename a .mobileconfig, but it still has to
	// agree with name when both are given
	batch([]fleet.BatchModifyMDMConfigProfilePayload{
		{Profile: mobileconfigForTest("Payload Alias", "com.test.batch.alias"), DisplayName: "Ignored"},
	}, http.StatusNoContent)
	listed = list()
	require.Len(t, listed, 1)
	for _, p := range listed {
		require.Equal(t, "Payload Alias", p.Name)
	}
	batch([]fleet.BatchModifyMDMConfigProfilePayload{
		{Profile: mobileconfigForTest("Payload Alias", "com.test.batch.alias"), Name: "A", DisplayName: "B"},
	}, http.StatusUnprocessableEntity)

	// a PayloadDisplayName is held to the same length limit as a given name
	batch([]fleet.BatchModifyMDMConfigProfilePayload{
		{Profile: mobileconfigForTest(strings.Repeat("p", fleet.MaxProfileNameLength+1), "com.test.batch.long")},
	}, http.StatusUnprocessableEntity)
}

func (s *integrationMDMTestSuite) TestConfigProfileSelfServiceAndHidden() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	teamID := fmt.Sprint(team.ID)

	create := func(fileName string, content []byte, fields map[string][]string, wantStatus int) string {
		fields["team_id"] = []string{teamID}
		body, headers := generateNewProfileMultipartRequest(t, fileName, content, s.token, fields)
		res := s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), wantStatus, headers)
		defer res.Body.Close()
		var resp newMDMConfigProfileResponse
		require.NoError(t, json.NewDecoder(res.Body).Decode(&resp))
		return resp.ProfileUUID
	}
	update := func(profileUUID string, fields map[string][]string, wantStatus int) {
		body, headers := generateMultipartRequest(t, "profile", "", nil, s.token, fields)
		s.DoRawWithHeaders("PATCH", "/api/latest/fleet/configuration_profiles/"+profileUUID, body.Bytes(), wantStatus, headers).Body.Close()
	}
	get := func(profileUUID string) *fleet.MDMConfigProfilePayload {
		var resp getMDMConfigProfileResponse
		s.DoJSON("GET", "/api/latest/fleet/configuration_profiles/"+profileUUID, getMDMConfigProfileRequest{}, http.StatusOK, &resp)
		return resp.MDMConfigProfilePayload
	}
	installOn := func(hostUUID, profileUUID, identifier string) {
		require.NoError(t, s.ds.BulkUpsertMDMAppleHostProfiles(ctx, []*fleet.MDMAppleBulkUpsertHostProfilePayload{{
			ProfileUUID: profileUUID, ProfileIdentifier: identifier, HostUUID: hostUUID, CommandUUID: uuid.NewString(),
			OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerified, Checksum: []byte("csum"), Scope: fleet.PayloadScopeSystem,
		}}))
	}
	requireOptedIn := func(hostUUID, profileUUID string) {
		ok, err := s.ds.HasHostMDMProfileOptIn(ctx, hostUUID, profileUUID)
		require.NoError(t, err)
		require.True(t, ok)
	}
	yes := []string{"true"}

	ssUUID := create("ss.mobileconfig", mobileconfigForTest("SS", "com.test.ss"), map[string][]string{"self_service": yes}, http.StatusOK)
	winUUID := create("hidden.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/Hidden/Test"), map[string][]string{"hidden": yes}, http.StatusOK)
	declUUID := create("hidden-decl.json", declBytesForTest("com.test.hidden.decl", "hidden"), map[string][]string{"hidden": yes}, http.StatusOK)
	require.True(t, get(ssUUID).SelfService)
	require.True(t, get(winUUID).Hidden)
	require.True(t, get(declUUID).Hidden)

	create("ss.xml", syncMLForTest("./Device/Vendor/MSFT/Policy/Config/SS/Test"), map[string][]string{"self_service": yes}, http.StatusUnprocessableEntity)
	create("ss.json", declBytesForTest("com.test.ss.decl", "ss"), map[string][]string{"self_service": yes}, http.StatusUnprocessableEntity)
	create("both.mobileconfig", mobileconfigForTest("Both", "com.test.both"), map[string][]string{"self_service": yes, "hidden": yes}, http.StatusUnprocessableEntity)
	create("bad.mobileconfig", mobileconfigForTest("Bad", "com.test.bad"), map[string][]string{"hidden": {"nope"}}, http.StatusBadRequest)
	// hidden is validated against the stored self_service
	update(ssUUID, map[string][]string{"hidden": yes}, http.StatusUnprocessableEntity)
	update(declUUID, map[string][]string{"self_service": yes}, http.StatusUnprocessableEntity)

	// flipping to self-service opts in hosts that have it, without moving uploaded_at
	forcedUUID := create("forced.mobileconfig", mobileconfigForTest("Forced", "com.test.forced"), map[string][]string{}, http.StatusOK)
	installOn("host-1", forcedUUID, "com.test.forced")
	before := get(forcedUUID)
	update(forcedUUID, map[string][]string{"self_service": yes}, http.StatusOK)
	after := get(forcedUUID)
	require.True(t, after.SelfService)
	require.Equal(t, before.UploadedAt, after.UploadedAt)
	requireOptedIn("host-1", forcedUUID)
	update(forcedUUID, map[string][]string{"description": {"kept"}}, http.StatusOK)
	require.True(t, get(forcedUUID).SelfService)

	var listResp listMDMConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", listMDMConfigProfilesRequest{}, http.StatusOK, &listResp, "team_id", teamID)
	for _, p := range listResp.Profiles {
		require.Equal(t, p.ProfileUUID == ssUUID || p.ProfileUUID == forcedUUID, p.SelfService, p.Name)
		require.Equal(t, p.ProfileUUID == winUUID || p.ProfileUUID == declUUID, p.Hidden, p.Name)
	}

	// batch: the flip seeds opt-ins too, and is declarative for the flags
	batch := func(profiles []fleet.MDMProfileBatchPayload, wantStatus int) {
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: profiles}, wantStatus, "team_id", teamID)
	}
	batch([]fleet.MDMProfileBatchPayload{{Name: "B", Contents: mobileconfigForTest("B", "com.test.b")}}, http.StatusNoContent)
	var bUUID string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &bUUID, `SELECT profile_uuid FROM mdm_apple_configuration_profiles WHERE identifier = 'com.test.b'`)
	})
	installOn("host-2", bUUID, "com.test.b")
	batch([]fleet.MDMProfileBatchPayload{
		{Name: "B", Contents: mobileconfigForTest("B", "com.test.b"), SelfService: true},
		{Name: "W", Contents: syncMLForTest("./Device/Vendor/MSFT/Policy/Config/W/Test"), Hidden: true},
		{Name: "A", Contents: []byte(`{"cameraDisabled": true}`), Hidden: true},
	}, http.StatusNoContent)
	require.True(t, get(bUUID).SelfService)
	requireOptedIn("host-2", bUUID)
	listResp = listMDMConfigProfilesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", listMDMConfigProfilesRequest{}, http.StatusOK, &listResp, "team_id", teamID)
	require.Len(t, listResp.Profiles, 3)
	for _, p := range listResp.Profiles {
		require.Equal(t, p.Name == "W" || p.Name == "A", p.Hidden, p.Name)
	}
	batch([]fleet.MDMProfileBatchPayload{{Name: "W", Contents: syncMLForTest("./Device/Vendor/MSFT/Policy/Config/W/Test"), SelfService: true}}, http.StatusUnprocessableEntity)
}
