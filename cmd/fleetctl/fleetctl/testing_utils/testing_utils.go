package testing_utils

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/cached_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	mdmtesting "github.com/fleetdm/fleet/v4/server/mdm/testing_utils"
	"github.com/fleetdm/fleet/v4/server/mock"
	mock2 "github.com/fleetdm/fleet/v4/server/mock/mdm"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const (
	teamName       = "Team Test"
	fleetServerURL = "https://fleet.example.com"
	orgName        = "GitOps Test"
)

// RunServerWithMockedDS runs the fleet server with several mocked DS methods.
//
// NOTE: Assumes the current session is always from the admin user (see ds.SessionByKeyFunc below).
func RunServerWithMockedDS(t *testing.T, opts ...*service.TestServerOpts) (*httptest.Server, *mock.Store) {
	ds := new(mock.Store)
	var users []*fleet.User
	var admin *fleet.User
	ds.NewUserFunc = func(ctx context.Context, user *fleet.User) (*fleet.User, error) {
		if user.GlobalRole != nil && *user.GlobalRole == fleet.RoleAdmin {
			admin = user
		}
		users = append(users, user)
		return user, nil
	}
	ds.SessionByKeyFunc = func(ctx context.Context, key string) (*fleet.Session, error) {
		return &fleet.Session{
			CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
			ID:              1,
			AccessedAt:      time.Now(),
			UserID:          admin.ID,
			Key:             key,
		}, nil
	}
	ds.MarkSessionAccessedFunc = func(ctx context.Context, session *fleet.Session) error {
		return nil
	}
	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return admin, nil
	}
	ds.ListUsersFunc = func(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
		return users, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.NewGlobalPolicyFunc = func(ctx context.Context, authorID *uint, args fleet.PolicyPayload) (*fleet.Policy, error) {
		return &fleet.Policy{
			PolicyData: fleet.PolicyData{
				Name:        args.Name,
				Query:       args.Query,
				Critical:    args.Critical,
				Platform:    args.Platform,
				Description: args.Description,
				Resolution:  &args.Resolution,
				AuthorID:    authorID,
			},
		}, nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time) error {
		return nil
	}
	apnsCert, apnsKey, err := mysql.GenerateTestCertBytes(mdmtesting.NewTestMDMAppleCertTemplate())
	require.NoError(t, err)
	certPEM, keyPEM, tokenBytes, err := mysql.GenerateTestABMAssets(t)
	require.NoError(t, err)
	ds.GetAllMDMConfigAssetsHashesFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName) (map[fleet.MDMAssetName]string, error) {
		return map[fleet.MDMAssetName]string{
			fleet.MDMAssetABMCert:            "abmcert",
			fleet.MDMAssetABMKey:             "abmkey",
			fleet.MDMAssetABMTokenDeprecated: "abmtoken",
			fleet.MDMAssetAPNSCert:           "apnscert",
			fleet.MDMAssetAPNSKey:            "apnskey",
			fleet.MDMAssetCACert:             "scepcert",
			fleet.MDMAssetCAKey:              "scepkey",
		}, nil
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetABMCert:            {Name: fleet.MDMAssetABMCert, Value: certPEM},
			fleet.MDMAssetABMKey:             {Name: fleet.MDMAssetABMKey, Value: keyPEM},
			fleet.MDMAssetABMTokenDeprecated: {Name: fleet.MDMAssetABMTokenDeprecated, Value: tokenBytes},
			fleet.MDMAssetAPNSCert:           {Name: fleet.MDMAssetAPNSCert, Value: apnsCert},
			fleet.MDMAssetAPNSKey:            {Name: fleet.MDMAssetAPNSKey, Value: apnsKey},
			fleet.MDMAssetCACert:             {Name: fleet.MDMAssetCACert, Value: certPEM},
			fleet.MDMAssetCAKey:              {Name: fleet.MDMAssetCAKey, Value: keyPEM},
		}, nil
	}

	ds.ApplyYaraRulesFunc = func(context.Context, []fleet.YaraRule) error {
		return nil
	}
	ds.ValidateEmbeddedSecretsFunc = func(ctx context.Context, documents []string) error {
		return nil
	}
	ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
		return nil, nil
	}
	ds.ListHostDeviceMappingFunc = func(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
		return nil, nil
	}
	var cachedDS fleet.Datastore
	if len(opts) > 0 && opts[0].NoCacheDatastore {
		cachedDS = ds
	} else {
		cachedDS = cached_mysql.New(ds)
	}
	_, server := service.RunServerForTestsWithDS(t, cachedDS, opts...)
	os.Setenv("FLEET_SERVER_ADDRESS", server.URL)

	return server, ds
}

func getPathRelative(relativePath string) string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get runtime caller info")
	}
	sourceDir := filepath.Dir(currentFile)
	return filepath.Join(sourceDir, relativePath)
}

func ServeMDMBootstrapPackage(t *testing.T, pkgPath, pkgName string) (*httptest.Server, int) {
	pkgBytes, err := os.ReadFile(pkgPath)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(pkgBytes)))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, pkgName))
		if n, err := w.Write(pkgBytes); err != nil {
			require.NoError(t, err)
			require.Equal(t, len(pkgBytes), n)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, len(pkgBytes)
}

func StartSoftwareInstallerServer(t *testing.T) {
	// start the web server that will serve the installer
	b, err := os.ReadFile(getPathRelative("../../../../server/service/testdata/software-installers/ruby.deb"))
	require.NoError(t, err)

	srv := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.Contains(r.URL.Path, "notfound"):
					w.WriteHeader(http.StatusNotFound)
					return
				case strings.HasSuffix(r.URL.Path, ".txt"):
					w.Header().Set("Content-Type", "text/plain")
					_, _ = w.Write([]byte(`a simple text file`))
					return
				case strings.Contains(r.URL.Path, "toolarge"):
					w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
					var sz int
					for sz < 3000*1024*1024 {
						n, _ := w.Write(b)
						sz += n
					}
				default:
					w.Header().Set("Content-Type", "application/vnd.debian.binary-package")
					_, _ = w.Write(b)
				}
			},
		),
	)
	t.Cleanup(srv.Close)
	t.Setenv("SOFTWARE_INSTALLER_URL", srv.URL)
}

func SetupFullGitOpsPremiumServer(t *testing.T) (*mock.Store, **fleet.AppConfig, map[string]**fleet.Team) {
	testCert, testKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)
	testCertPEM := tokenpki.PEMCertificate(testCert.Raw)
	testKeyPEM := tokenpki.PEMRSAPrivateKey(testKey)
	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(t, &fleetCfg, testCertPEM, testKeyPEM, getPathRelative("../../../../server/service/testdata"))

	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	_, ds := RunServerWithMockedDS(
		t, &service.TestServerOpts{
			MDMStorage:       new(mock2.MDMAppleStore),
			MDMPusher:        MockPusher{},
			FleetConfig:      &fleetCfg,
			License:          license,
			NoCacheDatastore: true,
			KeyValueStore:    NewMemKeyValueStore(),
		},
	)

	// Mock appConfig
	savedAppConfig := &fleet.AppConfig{
		MDM: fleet.MDM{
			EnabledAndConfigured: true,
		},
	}
	AddLabelMocks(ds)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		appConfigCopy := *savedAppConfig
		return &appConfigCopy, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) error {
		appConfigCopy := *config
		savedAppConfig = &appConfigCopy
		return nil
	}
	ds.SetTeamVPPAppsFunc = func(ctx context.Context, teamID *uint, adamIDs []fleet.VPPAppTeam) error {
		return nil
	}
	ds.BatchInsertVPPAppsFunc = func(ctx context.Context, apps []*fleet.VPPApp) error {
		return nil
	}

	savedTeams := map[string]**fleet.Team{}

	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		return nil
	}
	ds.ApplyPolicySpecsFunc = func(ctx context.Context, authorID uint, specs []*fleet.PolicySpec) error {
		return nil
	}
	ds.ApplyQueriesFunc = func(
		ctx context.Context, authorID uint, queries []*fleet.Query, queriesToDiscardResults map[uint]struct{},
	) error {
		return nil
	}
	ds.BatchSetMDMProfilesFunc = func(
		ctx context.Context, tmID *uint, macProfiles []*fleet.MDMAppleConfigProfile, winProfiles []*fleet.MDMWindowsConfigProfile,
		macDecls []*fleet.MDMAppleDeclaration, vars []fleet.MDMProfileIdentifierFleetVariables,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.BatchSetScriptsFunc = func(ctx context.Context, tmID *uint, scripts []*fleet.Script) ([]fleet.ScriptResponse, error) {
		return []fleet.ScriptResponse{}, nil
	}
	ds.BulkSetPendingMDMHostProfilesFunc = func(
		ctx context.Context, hostIDs []uint, teamIDs []uint, profileUUIDs []string, hostUUIDs []string,
	) (updates fleet.MDMProfilesUpdates, err error) {
		return fleet.MDMProfilesUpdates{}, nil
	}
	ds.DeleteMDMAppleDeclarationByNameFunc = func(ctx context.Context, teamID *uint, name string) error {
		return nil
	}
	ds.GetMDMAppleBootstrapPackageMetaFunc = func(ctx context.Context, teamID uint) (*fleet.MDMAppleBootstrapPackage, error) {
		return &fleet.MDMAppleBootstrapPackage{}, nil
	}
	ds.DeleteMDMAppleBootstrapPackageFunc = func(ctx context.Context, teamID uint) error {
		return nil
	}
	ds.GetMDMAppleSetupAssistantFunc = func(ctx context.Context, teamID *uint) (*fleet.MDMAppleSetupAssistant, error) {
		return nil, nil
	}
	ds.DeleteMDMAppleSetupAssistantFunc = func(ctx context.Context, teamID *uint) error {
		return nil
	}
	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, isNew bool, teamID *uint) (bool, error) {
		return true, nil
	}
	ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
		require.ElementsMatch(t, labels, []string{fleet.BuiltinLabelMacOS14Plus})
		return map[string]uint{fleet.BuiltinLabelMacOS14Plus: 1}, nil
	}
	ds.ListGlobalPoliciesFunc = func(ctx context.Context, opts fleet.ListOptions) ([]*fleet.Policy, error) { return nil, nil }
	ds.ListTeamPoliciesFunc = func(
		ctx context.Context, teamID uint, opts fleet.ListOptions, iopts fleet.ListOptions,
	) (teamPolicies []*fleet.Policy, inheritedPolicies []*fleet.Policy, err error) {
		return nil, nil, nil
	}
	ds.ListTeamsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
		if savedTeams != nil {
			var result []*fleet.Team
			for _, t := range savedTeams {
				result = append(result, *t)
			}
			return result, nil
		}
		return nil, nil
	}
	ds.TeamsSummaryFunc = func(ctx context.Context) ([]*fleet.TeamSummary, error) {
		summary := make([]*fleet.TeamSummary, 0, len(savedTeams))
		for _, team := range savedTeams {
			summary = append(summary, &fleet.TeamSummary{
				ID:          (*team).ID,
				Name:        (*team).Name,
				Description: (*team).Description,
			})
		}
		return summary, nil
	}
	ds.ListQueriesFunc = func(ctx context.Context, opts fleet.ListQueryOptions) ([]*fleet.Query, int, *fleet.PaginationMetadata, error) {
		return nil, 0, nil, nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}
	ds.NewMDMAppleConfigProfileFunc = func(ctx context.Context, p fleet.MDMAppleConfigProfile, vars []string) (*fleet.MDMAppleConfigProfile, error) {
		return nil, nil
	}
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		job.ID = 1
		return job, nil
	}
	ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		team.ID = uint(len(savedTeams) + 1) //nolint:gosec // dismiss G115
		savedTeams[team.Name] = &team
		return team, nil
	}
	ds.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string) (*fleet.Query, error) {
		return nil, &notFoundError{}
	}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		for _, tm := range savedTeams {
			if (*tm).ID == tid {
				return *tm, nil
			}
		}
		return nil, &notFoundError{}
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		for _, tm := range savedTeams {
			if (*tm).Name == name {
				return *tm, nil
			}
		}
		return nil, &notFoundError{}
	}
	ds.TeamByFilenameFunc = func(ctx context.Context, filename string) (*fleet.Team, error) {
		for _, tm := range savedTeams {
			if (*tm).Filename != nil && *(*tm).Filename == filename {
				return *tm, nil
			}
		}
		return nil, &notFoundError{}
	}
	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		savedTeams[team.Name] = &team
		return team, nil
	}
	ds.SetOrUpdateMDMAppleDeclarationFunc = func(ctx context.Context, declaration *fleet.MDMAppleDeclaration) (
		*fleet.MDMAppleDeclaration, error,
	) {
		declaration.DeclarationUUID = uuid.NewString()
		return declaration, nil
	}
	ds.BatchSetSoftwareInstallersFunc = func(ctx context.Context, teamID *uint, installers []*fleet.UploadSoftwareInstallerPayload) error {
		return nil
	}
	ds.GetSoftwareInstallersFunc = func(ctx context.Context, tmID uint) ([]fleet.SoftwarePackageResponse, error) {
		return nil, nil
	}

	ds.InsertVPPTokenFunc = func(ctx context.Context, tok *fleet.VPPTokenData) (*fleet.VPPTokenDB, error) {
		return &fleet.VPPTokenDB{}, nil
	}
	ds.GetVPPTokenFunc = func(ctx context.Context, tokenID uint) (*fleet.VPPTokenDB, error) {
		return &fleet.VPPTokenDB{}, err
	}
	ds.GetVPPTokenByTeamIDFunc = func(ctx context.Context, teamID *uint) (*fleet.VPPTokenDB, error) {
		return &fleet.VPPTokenDB{}, nil
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return nil, nil
	}
	ds.UpdateVPPTokenTeamsFunc = func(ctx context.Context, id uint, teams []uint) (*fleet.VPPTokenDB, error) {
		return &fleet.VPPTokenDB{}, nil
	}
	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{{OrganizationName: "Fleet Device Management Inc."}}, nil
	}
	ds.ListSoftwareTitlesFunc = func(ctx context.Context, opt fleet.SoftwareTitleListOptions,
		tmFilter fleet.TeamFilter,
	) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
		return nil, 0, nil, nil
	}
	ds.SaveABMTokenFunc = func(ctx context.Context, tok *fleet.ABMToken) error {
		return nil
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{}, nil
	}
	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{}, nil
	}
	ds.DeleteSetupExperienceScriptFunc = func(ctx context.Context, teamID *uint) error {
		return nil
	}
	ds.SetSetupExperienceScriptFunc = func(ctx context.Context, script *fleet.Script) error {
		return nil
	}
	ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, document string) (string, *time.Time, error) {
		return document, nil, nil
	}
	ds.MDMGetEULAMetadataFunc = func(ctx context.Context) (*fleet.MDMEULA, error) {
		return nil, &notFoundError{} // No existing EULA
	}

	t.Setenv("FLEET_SERVER_URL", fleetServerURL)
	t.Setenv("ORG_NAME", orgName)
	t.Setenv("TEST_TEAM_NAME", teamName)
	t.Setenv("APPLE_BM_DEFAULT_TEAM", teamName)

	return ds, &savedAppConfig, savedTeams
}

type AppleVPPConfigSrvConf struct {
	Assets        []vpp.Asset
	SerialNumbers []string
}

func StartVPPApplyServer(t *testing.T, config *AppleVPPConfigSrvConf) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "associate") {
			var associations vpp.AssociateAssetsRequest

			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&associations); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			if len(associations.Assets) == 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				res := vpp.ErrorResponse{
					ErrorNumber:  9718,
					ErrorMessage: "This request doesn't contain an asset, which is a required argument. Change the request to provide an asset.",
				}
				if err := json.NewEncoder(w).Encode(res); err != nil {
					panic(err)
				}
				return
			}

			if len(associations.SerialNumbers) == 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				res := vpp.ErrorResponse{
					ErrorNumber:  9719,
					ErrorMessage: "Either clientUserIds or serialNumbers are required arguments. Change the request to provide assignable users and devices.",
				}
				if err := json.NewEncoder(w).Encode(res); err != nil {
					panic(err)
				}
				return
			}

			var badAssets []vpp.Asset
			for _, reqAsset := range associations.Assets {
				var found bool
				for _, goodAsset := range config.Assets {
					if reqAsset == goodAsset {
						found = true
					}
				}
				if !found {
					badAssets = append(badAssets, reqAsset)
				}
			}

			var badSerials []string
			for _, reqSerial := range associations.SerialNumbers {
				var found bool
				for _, goodSerial := range config.SerialNumbers {
					if reqSerial == goodSerial {
						found = true
					}
				}
				if !found {
					badSerials = append(badSerials, reqSerial)
				}
			}

			if len(badAssets) != 0 || len(badSerials) != 0 {
				errMsg := "error associating assets."
				if len(badAssets) > 0 {
					var badAdamIds []string
					for _, asset := range badAssets {
						badAdamIds = append(badAdamIds, asset.AdamID)
					}
					errMsg += fmt.Sprintf(" assets don't exist on account: %s.", strings.Join(badAdamIds, ", "))
				}
				if len(badSerials) > 0 {
					errMsg += fmt.Sprintf(" bad serials: %s.", strings.Join(badSerials, ", "))
				}
				res := vpp.ErrorResponse{
					ErrorInfo: vpp.ResponseErrorInfo{
						Assets:        badAssets,
						ClientUserIds: []string{"something"},
						SerialNumbers: badSerials,
					},
					// Not sure what error should be returned on each
					// error type
					ErrorNumber:  1,
					ErrorMessage: errMsg,
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				if err := json.NewEncoder(w).Encode(res); err != nil {
					panic(err)
				}
			}
			return
		}

		if strings.Contains(r.URL.Path, "assets") {
			// Then we're responding to GetAssets
			w.Header().Set("Content-Type", "application/json")
			encoder := json.NewEncoder(w)
			err := encoder.Encode(map[string][]vpp.Asset{"assets": config.Assets})
			if err != nil {
				panic(err)
			}
			return
		}

		resp := []byte(`{"locationName": "Fleet Location One"}`)
		if strings.Contains(r.URL.RawQuery, "invalidToken") {
			// This replicates the response sent back from Apple's VPP endpoints when an invalid
			// token is passed. For more details see:
			// https://developer.apple.com/documentation/devicemanagement/app_and_book_management/app_and_book_management_legacy/interpreting_error_codes
			// https://developer.apple.com/documentation/devicemanagement/client_config
			// https://developer.apple.com/documentation/devicemanagement/errorresponse
			// Note that the Apple server returns 200 in this case.
			resp = []byte(`{"errorNumber": 9622,"errorMessage": "Invalid authentication token"}`)
		}

		if strings.Contains(r.URL.RawQuery, "serverError") {
			resp = []byte(`{"errorNumber": 9603,"errorMessage": "Internal server error"}`)
			w.WriteHeader(http.StatusInternalServerError)
		}

		_, _ = w.Write(resp)
	}))

	t.Setenv("FLEET_DEV_VPP_URL", srv.URL)
	t.Cleanup(srv.Close)
}

func StartAndServeVPPServer(t *testing.T) {
	config := &AppleVPPConfigSrvConf{
		Assets: []vpp.Asset{
			{
				AdamID:         "1",
				PricingParam:   "STDQ",
				AvailableCount: 12,
			},
			{
				AdamID:         "2",
				PricingParam:   "STDQ",
				AvailableCount: 3,
			},
		},
		SerialNumbers: []string{"123", "456"},
	}

	StartVPPApplyServer(t, config)

	appleITunesSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// a map of apps we can respond with
		db := map[string]string{
			// macos app
			"1": `{"bundleId": "a-1", "artworkUrl512": "https://example.com/images/1", "version": "1.0.0", "trackName": "App 1", "TrackID": 1}`,
			// macos, ios, ipados app
			"2": `{"bundleId": "b-2", "artworkUrl512": "https://example.com/images/2", "version": "2.0.0", "trackName": "App 2", "TrackID": 2,
				"supportedDevices": ["MacDesktop-MacDesktop", "iPhone5s-iPhone5s", "iPadAir-iPadAir"] }`,
			// ipados app
			"3": `{"bundleId": "c-3", "artworkUrl512": "https://example.com/images/3", "version": "3.0.0", "trackName": "App 3", "TrackID": 3,
				"supportedDevices": ["iPadAir-iPadAir"] }`,
		}

		adamIDString := r.URL.Query().Get("id")
		adamIDs := strings.Split(adamIDString, ",")

		var objs []string
		for _, a := range adamIDs {
			objs = append(objs, db[a])
		}

		_, _ = w.Write([]byte(fmt.Sprintf(`{"results": [%s]}`, strings.Join(objs, ","))))
	}))
	t.Setenv("FLEET_DEV_ITUNES_URL", appleITunesSrv.URL)
}

type MockPusher struct{}

func (MockPusher) Push(ctx context.Context, ids []string) (map[string]*push.Response, error) {
	m := make(map[string]*push.Response, len(ids))
	for _, id := range ids {
		m[id] = &push.Response{Id: id}
	}
	return m, nil
}

type MemKeyValueStore struct {
	m sync.Map
}

func NewMemKeyValueStore() *MemKeyValueStore {
	return &MemKeyValueStore{}
}

func (m *MemKeyValueStore) Set(ctx context.Context, key string, value string, expireTime time.Duration) error {
	m.m.Store(key, value)
	return nil
}

func (m *MemKeyValueStore) Get(ctx context.Context, key string) (*string, error) {
	v, ok := m.m.Load(key)
	if !ok {
		return nil, nil
	}
	vAsString := v.(string)
	return &vAsString, nil
}

func AddLabelMocks(ds *mock.Store) {
	var deletedLabels []string
	ds.GetLabelSpecsFunc = func(ctx context.Context) ([]*fleet.LabelSpec, error) {
		return []*fleet.LabelSpec{
			{
				Name:                "a",
				Description:         "A global label",
				LabelMembershipType: fleet.LabelMembershipTypeManual,
				Hosts:               []string{"host2", "host3"},
			},
			{
				Name:                "b",
				Description:         "Another global label",
				LabelMembershipType: fleet.LabelMembershipTypeDynamic,
				Query:               "SELECT 1 from osquery_info",
			},
		}, nil
	}
	ds.ApplyLabelSpecsWithAuthorFunc = func(ctx context.Context, specs []*fleet.LabelSpec, authorID *uint) (err error) {
		return nil
	}

	ds.DeleteLabelFunc = func(ctx context.Context, name string) error {
		deletedLabels = append(deletedLabels, name)
		return nil
	}
	ds.LabelsByNameFunc = func(ctx context.Context, names []string) (map[string]*fleet.Label, error) {
		return map[string]*fleet.Label{
			"a": {
				ID:   1,
				Name: "a",
			},
			"b": {
				ID:   2,
				Name: "b",
			},
		}, nil
	}
}

type notFoundError struct{}

var _ fleet.NotFoundError = (*notFoundError)(nil)

func (e *notFoundError) IsNotFound() bool {
	return true
}

func (e *notFoundError) Error() string {
	return ""
}
