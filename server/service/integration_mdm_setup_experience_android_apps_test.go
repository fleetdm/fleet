package service

// Android setup experience tests for the MDM suite.
//
// Belongs here: Android setup experience software installs, cancelling them on unenroll, and
// Android app configuration (on its own and as part of setup experience).
//
// Does not belong here: Android enrollment and commands outside setup experience and Android software
// outside setup experience (integration_android_software_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
)

func (s *integrationMDMTestSuite) TestSetupExperienceAndroid() {
	t := s.T()
	ctx := t.Context()
	s.setSkipWorkerJobs(t)

	s.setVPPTokenForTeam(0)
	enterpriseID := s.enableAndroidMDM(t)

	// add a macOS software installer
	payloadDummy := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "dummy_installer.pkg",
		Title:         "DummyApp",
		TeamID:        nil,
	}
	s.uploadSoftwareInstaller(t, payloadDummy, http.StatusOK, "")

	// add a VPP app available on macos and ios
	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: nil, Platform: "darwin", AppStoreID: "2", SelfService: false}, http.StatusOK)
	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: nil, Platform: "ios", AppStoreID: "2", SelfService: false}, http.StatusOK)

	// add 2 android apps
	app1 := &fleet.VPPApp{
		AdamID:           "com.test1",
		Platform:         fleet.AndroidPlatform,
		Name:             "Test1",
		BundleIdentifier: "com.test1",
		IconURL:          "https://example.com/1",
	}
	app2 := &fleet.VPPApp{
		AdamID:           "com.test2",
		Platform:         fleet.AndroidPlatform,
		Name:             "Test2",
		BundleIdentifier: "com.test2",
		IconURL:          "https://example.com/2",
	}

	androidApps := []*fleet.VPPApp{app1, app2}
	s.androidAPIClient.EnterprisesApplicationsFunc = func(ctx context.Context, enterpriseName string, packageName string) (*androidmanagement.Application, error) {
		for _, app := range androidApps {
			if app.AdamID == packageName {
				return &androidmanagement.Application{IconUrl: app.IconURL, Title: app.Name}, nil
			}
		}
		return nil, &notFoundError{}
	}

	// the host policy's apps should change three times:
	// 1. Fleet agent added during enrollment (via prepareHostSpecificPolicy)
	// 2. The app list replaced with the 2 Android apps available for self-install, plus the Fleet agent
	// 3. Setup experience with only the app to install at setup (PREINSTALLED)
	var patchAppsCallCount int // no need for mutex, protected via runWorkerUntilDone
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, appPolicies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		patchAppsCallCount++
		switch patchAppsCallCount {
		case 1:
			// first call adds Fleet agent during enrollment
			require.Len(t, appPolicies, 1, "first call should add the Fleet agent")
			require.Equal(t, "FORCE_INSTALLED", appPolicies[0].InstallType)
			require.Equal(t, "com.fleetdm.agent", appPolicies[0].PackageName)
		case 3:
			// third call for setup experience, should have only app1 with PREINSTALLED
			require.Len(t, appPolicies, 1, "third call for setup experience should have only 1 app")
			require.Equal(t, "PREINSTALLED", appPolicies[0].InstallType)
			require.Equal(t, app1.VPPAppID.AdamID, appPolicies[0].PackageName)
		default:
			t.Fatalf("unexpected call %d to EnterprisesPoliciesModifyPolicyApplications", patchAppsCallCount)
		}

		return &androidmanagement.Policy{Version: int64(patchAppsCallCount)}, nil
	}
	s.androidAPIClient.EnterprisesPoliciesPatchFunc = func(ctx context.Context, policyName string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
		if !opts.OnlyUpdateApps {
			return &androidmanagement.Policy{}, nil
		}
		if len(policy.Applications) == 0 {
			// the enrollment clears the apps of the host policy before the setup experience runs
			return &androidmanagement.Policy{}, nil
		}
		patchAppsCallCount++
		require.Equal(t, 2, patchAppsCallCount, "the self-service apps should replace the app list after the Fleet agent is added")
		appPolicies := policy.Applications
		require.Len(t, appPolicies, 3, "the self-service apps and the Fleet agent")
		require.Equal(t, "AVAILABLE", appPolicies[0].InstallType)
		require.Equal(t, app1.VPPAppID.AdamID, appPolicies[0].PackageName)
		require.Equal(t, "AVAILABLE", appPolicies[1].InstallType)
		require.Equal(t, app2.VPPAppID.AdamID, appPolicies[1].PackageName)
		require.Equal(t, "com.fleetdm.agent", appPolicies[2].PackageName)
		return &androidmanagement.Policy{Version: int64(patchAppsCallCount)}, nil
	}

	// add Android app 1
	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID: app1.AdamID,
		Platform:   fleet.AndroidPlatform,
	}, http.StatusOK, &addAppResp)
	app1TitleID := addAppResp.TitleID

	// add Android app 2
	addAppResp = addAppStoreAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID: app2.AdamID,
		Platform:   fleet.AndroidPlatform,
	}, http.StatusOK, &addAppResp)
	app2TitleID := addAppResp.TitleID

	require.NotEqual(t, app1TitleID, app2TitleID)

	// add app 1 to Android setup experience
	var putResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/latest/fleet/setup_experience/software", &putSetupExperienceSoftwareRequest{
		Platform: string(fleet.AndroidPlatform),
		TeamID:   0,
		TitleIDs: []uint{app1TitleID},
	}, http.StatusOK, &putResp)

	// Run worker to flush out no-op "make_android_app_available" tasks from adding the apps above
	s.runWorkerUntilDoneWithChecks(true)

	// list the available setup experience software and verify that only app 1 is installed at setup
	var getResp getSetupExperienceSoftwareResponse
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software", nil, http.StatusOK, &getResp,
		"team_id", "0", "platform", string(fleet.AndroidPlatform), "order_key", "name")
	require.Len(t, getResp.SoftwareTitles, 2)
	require.Equal(t, app1TitleID, getResp.SoftwareTitles[0].ID)
	require.Equal(t, app1.Name, getResp.SoftwareTitles[0].Name)
	require.Equal(t, app1.AdamID, getResp.SoftwareTitles[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getResp.SoftwareTitles[0].AppStoreApp.InstallDuringSetup)
	require.True(t, *getResp.SoftwareTitles[0].AppStoreApp.InstallDuringSetup)
	require.Equal(t, app2TitleID, getResp.SoftwareTitles[1].ID)
	require.Equal(t, app2.Name, getResp.SoftwareTitles[1].Name)
	require.Equal(t, app2.AdamID, getResp.SoftwareTitles[1].AppStoreApp.AppStoreID)
	require.NotNil(t, getResp.SoftwareTitles[1].AppStoreApp.InstallDuringSetup)
	require.False(t, *getResp.SoftwareTitles[1].AppStoreApp.InstallDuringSetup)

	host, deviceInfo, pubSubToken := s.createAndEnrollAndroidDevice(t, "test-android", nil, false)

	// Google AMAPI hasn't been hit yet
	require.False(t, s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)
	// run worker, should run the job that assigns the app to the host's MDM policy
	s.runWorkerUntilDoneWithChecks(true)
	// should have hit the android API endpoint
	require.True(t, s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)

	var count int
	mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, tx, &count,
			`SELECT COUNT(*) FROM android_policy_requests WHERE policy_id = ?`,
			host.UUID)
	})
	// 1. The clear of the host policy's apps at enrollment
	// 2. The default enrollment policy
	// 3. The Fleet-enforced per-device policy
	// 4. The patch applications to make apps available for self-service
	// 5. The patch applications to force install at setup experience
	// (Note: Fleet agent install call is not recorded in the database)
	require.Equal(t, 5, count)

	// the pending install should show up in the host software
	getHostSw := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	app1CmdUUID := getHostSw.Software[0].AppStoreApp.LastInstall.CommandUUID
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// send a pub-sub with the software installed, to make it verified
	policyName := fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host.UUID)
	reportMsg := statusReportMessageWithEnterpriseSpecificID(
		t,
		androidmanagement.Device{
			Name:                 deviceInfo.Name,
			EnrollmentTokenData:  deviceInfo.EnrollmentTokenData,
			AppliedPolicyName:    policyName,
			AppliedPolicyVersion: 3, // policy version 3 is after Fleet agent (1), self-service apps (2), and setup experience (3)
			ApplicationReports: []*androidmanagement.ApplicationReport{
				{PackageName: app1.AdamID, State: "INSTALLED"},
			},
			LastPolicySyncTime: time.Now().Format(time.RFC3339Nano),
		},
		host.UUID,
	)
	req := android_service.PubSubPushRequest{PubSubMessage: *reportMsg}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubSubToken.Value))
	s.lastActivityOfTypeMatches(fleet.ActivityInstalledAppStoreApp{}.ActivityName(), fmt.Sprintf(`{"version_name": "Default version", "app_store_id":%q,
		"command_uuid":%q, "from_auto_update": false, "from_configuration_resend": false, "from_setup_experience": true, "host_display_name":%q, "host_id":%d, "host_platform":%q, "policy_id":null, "policy_name":null, "self_service":false, "software_title":%q,
		"status":%q}`, app1.AdamID, app1CmdUUID, host.DisplayName(), host.ID, host.Platform, app1.Name, fleet.SoftwareInstalled), 0)

	// the pending install should now be verified
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstalled, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// the software now shows up when including available-for-install (tracked via installer, not osquery inventory)
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "order_key", "name",
		"include_available_for_install", "true")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstalled, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// add app 2 to Android setup experience
	putResp = putSetupExperienceSoftwareResponse{}
	s.DoJSON("PUT", "/api/latest/fleet/setup_experience/software", &putSetupExperienceSoftwareRequest{
		Platform: string(fleet.AndroidPlatform),
		TeamID:   0,
		TitleIDs: []uint{app1TitleID, app2TitleID},
	}, http.StatusOK, &putResp)

	// enroll another Android device to test 2 apps that install on enroll
	host2, _, _ := s.createAndEnrollAndroidDevice(t, "test-android-2", nil, false)

	patchAppsCallCount = 0
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, appPolicies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		patchAppsCallCount++
		switch patchAppsCallCount {
		case 1:
			// first call adds Fleet agent during enrollment
			require.Len(t, appPolicies, 1, "first call should add the Fleet agent")
			require.Equal(t, "FORCE_INSTALLED", appPolicies[0].InstallType)
			require.Equal(t, "com.fleetdm.agent", appPolicies[0].PackageName)
		case 2:
			// second call to make apps available for self-install
			require.Len(t, appPolicies, 2, "second call to make apps available for self-install should have 2 apps")
			require.Equal(t, "AVAILABLE", appPolicies[0].InstallType)
			require.Equal(t, app1.VPPAppID.AdamID, appPolicies[0].PackageName)
			require.Equal(t, "AVAILABLE", appPolicies[1].InstallType)
			require.Equal(t, app2.VPPAppID.AdamID, appPolicies[1].PackageName)
		case 3:
			// third call for setup experience, should have both apps
			require.Len(t, appPolicies, 2, "third call for setup experience should have 2 apps")
			require.Equal(t, "PREINSTALLED", appPolicies[0].InstallType)
			require.Equal(t, app1.VPPAppID.AdamID, appPolicies[0].PackageName)
			require.Equal(t, "PREINSTALLED", appPolicies[1].InstallType)
			require.Equal(t, app2.VPPAppID.AdamID, appPolicies[1].PackageName)
		default:
			t.Fatalf("unexpected call count %d to EnterprisesPoliciesModifyPolicyApplications", patchAppsCallCount)
		}

		return &androidmanagement.Policy{Version: int64(patchAppsCallCount)}, nil
	}
	s.runWorkerUntilDoneWithChecks(true)

	// the pending installs should show up in the host software
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host2.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[1].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[1].Status)
}

func (s *integrationMDMTestSuite) TestSetupExperienceAndroidCancelOnUnenroll() {
	t := s.T()
	ctx := t.Context()
	s.setSkipWorkerJobs(t)
	enterpriseID := s.enableAndroidMDM(t)

	// add 2 android apps
	app1 := &fleet.VPPApp{
		AdamID:           "com.test1",
		Platform:         fleet.AndroidPlatform,
		Name:             "Test1",
		BundleIdentifier: "com.test1",
		IconURL:          "https://example.com/1",
	}
	app2 := &fleet.VPPApp{
		AdamID:           "com.test2",
		Platform:         fleet.AndroidPlatform,
		Name:             "Test2",
		BundleIdentifier: "com.test2",
		IconURL:          "https://example.com/2",
	}

	androidApps := []*fleet.VPPApp{app1, app2}
	s.androidAPIClient.EnterprisesApplicationsFunc = func(ctx context.Context, enterpriseName string, packageName string) (*androidmanagement.Application, error) {
		for _, app := range androidApps {
			if app.AdamID == packageName {
				return &androidmanagement.Application{IconUrl: app.IconURL, Title: app.Name}, nil
			}
		}
		return nil, &notFoundError{}
	}
	s.androidAPIClient.EnterprisesDevicesDeleteFunc = func(ctx context.Context, deviceName string) error {
		return nil
	}

	var patchAppsCallCount int // no need for mutex, protected via runWorkerUntilDone
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, appPolicies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		patchAppsCallCount++
		return &androidmanagement.Policy{Version: int64(patchAppsCallCount)}, nil
	}

	// add Android app 1
	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID: app1.AdamID,
		Platform:   fleet.AndroidPlatform,
	}, http.StatusOK, &addAppResp)
	app1TitleID := addAppResp.TitleID

	// add Android app 2
	addAppResp = addAppStoreAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID: app2.AdamID,
		Platform:   fleet.AndroidPlatform,
	}, http.StatusOK, &addAppResp)
	app2TitleID := addAppResp.TitleID

	require.NotEqual(t, app1TitleID, app2TitleID)

	// add app 1 to Android setup experience
	var putResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/latest/fleet/setup_experience/software", &putSetupExperienceSoftwareRequest{
		Platform: string(fleet.AndroidPlatform),
		TeamID:   0,
		TitleIDs: []uint{app1TitleID},
	}, http.StatusOK, &putResp)

	// enroll a few Android devices, will get app1 at setup
	host1, deviceInfo1, pubSubToken := s.createAndEnrollAndroidDevice(t, "test-1", nil, true)
	host2, _, _ := s.createAndEnrollAndroidDevice(t, "test-2", nil, false)
	host3, _, _ := s.createAndEnrollAndroidDevice(t, "test-3", nil, false)

	require.False(t, s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)
	s.runWorkerUntilDoneWithChecks(true)
	require.True(t, s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)

	// app install is pending
	getHostSw := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"query", app1.Name, "order_key", "name")
	require.Len(t, getHostSw.Software, 1)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)

	// turn off MDM for that host, should fail the pending install
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", host1.ID), nil, http.StatusNoContent)

	// app install is still pending as the device hasn't reported back its unenrollment yet
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"query", app1.Name, "order_key", "name")
	require.Len(t, getHostSw.Software, 1)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)

	// send a pub-sub with the status repored as deleted
	policyName := fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host1.UUID)
	reportMsg := statusReportMessageWithEnterpriseSpecificID(
		t,
		androidmanagement.Device{
			Name:                 deviceInfo1.Name,
			EnrollmentTokenData:  deviceInfo1.EnrollmentTokenData,
			AppliedPolicyName:    policyName,
			AppliedPolicyVersion: 2,
			LastPolicySyncTime:   time.Now().Format(time.RFC3339Nano),
			AppliedState:         "DELETED",
		},
		host1.UUID,
	)
	req := android_service.PubSubPushRequest{PubSubMessage: *reportMsg}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubSubToken.Value))

	// app install is now failed
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"query", app1.Name, "order_key", "name")
	require.Len(t, getHostSw.Software, 1)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallFailed, *getHostSw.Software[0].Status)
	app1CmdUUID := getHostSw.Software[0].AppStoreApp.LastInstall.CommandUUID

	// activities got created as expected
	s.lastActivityOfTypeMatches(fleet.ActivityTypeMDMUnenrolled{}.ActivityName(), fmt.Sprintf(`
	{"host_id": %d, "enrollment_id": null, "host_display_name": %q, "host_serial": %q, "installed_from_dep": false, "platform": %q}`,
		host1.ID, host1.DisplayName(), host1.HardwareSerial, host1.Platform), 0)
	s.lastActivityOfTypeMatches(fleet.ActivityInstalledAppStoreApp{}.ActivityName(), fmt.Sprintf(`{"version_name": "Default version", "app_store_id":%q,
		"command_uuid":%q, "from_auto_update": false, "from_configuration_resend": false, "from_setup_experience": true, "host_display_name":%q, "host_id":%d, "host_platform":%q, "policy_id":null, "policy_name":null, "self_service":false, "software_title":%q,
		"status":%q}`, app1.AdamID, app1CmdUUID, host1.DisplayName(), host1.ID, host1.Platform, app1.Name, fleet.SoftwareInstallFailed), 0)

	// host2 and host3 haven't been unenrolled, app install is still pending
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host2.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"query", app1.Name, "order_key", "name")
	require.Len(t, getHostSw.Software, 1)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)

	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host3.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"query", app1.Name, "order_key", "name")
	require.Len(t, getHostSw.Software, 1)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)

	// turn off MDM for Android globally
	var deleteEnterResp android.DefaultResponse
	s.DoJSON("DELETE", "/api/latest/fleet/android_enterprise", nil, http.StatusOK, &deleteEnterResp)

	// host2 and host3 app install is now failed, but because Android MDM is now off globally,
	// we can't list software available for install anymore, we have to use adhoc sql to confirm.
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host2.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true")
	require.Empty(t, getHostSw.Software)

	var countFailed, countOther int
	mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		err := sqlx.GetContext(ctx, tx, &countFailed, `SELECT COUNT(*) FROM host_vpp_software_installs WHERE host_id IN (?, ?) AND verification_failed_at IS NOT NULL`,
			host2.ID, host3.ID)
		if err != nil {
			return err
		}
		err = sqlx.GetContext(ctx, tx, &countOther, `SELECT COUNT(*) FROM host_vpp_software_installs WHERE host_id IN (?, ?) AND verification_failed_at IS NULL`,
			host2.ID, host3.ID)
		return err
	})
	require.Equal(t, 2, countFailed)
	require.Equal(t, 0, countOther)
}

func (s *integrationMDMTestSuite) TestSetupExperienceAndroidFailedInstallReportedInstalled() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	enterpriseID := s.enableAndroidMDM(t)

	apps := []*fleet.VPPApp{
		{AdamID: "com.reported.first", Platform: fleet.AndroidPlatform, Name: "ReportedFirst"},
		{AdamID: "com.reported.later", Platform: fleet.AndroidPlatform, Name: "ReportedLater"},
	}
	s.androidAPIClient.EnterprisesApplicationsFunc = func(ctx context.Context, enterpriseName string, packageName string) (*androidmanagement.Application, error) {
		for _, app := range apps {
			if app.AdamID == packageName {
				return &androidmanagement.Application{Title: app.Name}, nil
			}
		}
		return nil, &notFoundError{}
	}
	var policyVersion int64 // no need for mutex, protected via runWorkerUntilDone
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, appPolicies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		policyVersion++
		return &androidmanagement.Policy{Version: policyVersion}, nil
	}

	var titleIDs []uint
	for _, app := range apps {
		var addAppResp addAppStoreAppResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			AppStoreID: app.AdamID, Platform: fleet.AndroidPlatform,
		}, http.StatusOK, &addAppResp)
		titleIDs = append(titleIDs, addAppResp.TitleID)
	}
	var putResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/latest/fleet/setup_experience/software", &putSetupExperienceSoftwareRequest{
		Platform: string(fleet.AndroidPlatform), TeamID: 0, TitleIDs: titleIDs,
	}, http.StatusOK, &putResp)

	host, deviceInfo, pubSubToken := s.createAndEnrollAndroidDevice(t, "failed-then-installed", nil, false)
	s.runWorkerUntilDoneWithChecks(true)

	pushStatusReport := func(appReports []*androidmanagement.ApplicationReport, nonCompliance []*androidmanagement.NonComplianceDetail) {
		msg := statusReportMessageWithEnterpriseSpecificID(t, androidmanagement.Device{
			Name:                 deviceInfo.Name,
			EnrollmentTokenData:  deviceInfo.EnrollmentTokenData,
			AppliedPolicyName:    fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host.UUID),
			AppliedPolicyVersion: policyVersion,
			ApplicationReports:   appReports,
			NonComplianceDetails: nonCompliance,
			LastPolicySyncTime:   time.Now().Format(time.RFC3339Nano),
		}, host.UUID)
		s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &android_service.PubSubPushRequest{PubSubMessage: *msg},
			http.StatusOK, "token", string(pubSubToken.Value))
	}
	installed := func(app *fleet.VPPApp) *androidmanagement.ApplicationReport {
		return &androidmanagement.ApplicationReport{PackageName: app.AdamID, State: "INSTALLED"}
	}
	notApproved := func(app *fleet.VPPApp) *androidmanagement.NonComplianceDetail {
		return &androidmanagement.NonComplianceDetail{PackageName: app.AdamID, NonComplianceReason: "APP_NOT_INSTALLED", InstallationFailureReason: "NOT_APPROVED"}
	}
	hostStatuses := func() map[string]fleet.SoftwareInstallerStatus {
		var resp getHostSoftwareResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &resp, "available_for_install", "true")
		statuses := make(map[string]fleet.SoftwareInstallerStatus, len(resp.Software))
		for _, sw := range resp.Software {
			require.NotNil(t, sw.Status, sw.Name)
			statuses[sw.Name] = *sw.Status
		}
		return statuses
	}
	// oldest first
	installActivityStatuses := func(app *fleet.VPPApp) []string {
		var resp listActivitiesResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host.ID), nil, http.StatusOK, &resp, "per_page", "100")
		var statuses []string
		for _, act := range slices.Backward(resp.Activities) {
			if act.Type != (fleet.ActivityInstalledAppStoreApp{}).ActivityName() || act.Details == nil {
				continue
			}
			var details fleet.ActivityInstalledAppStoreApp
			require.NoError(t, json.Unmarshal(*act.Details, &details))
			if details.AppStoreID == app.AdamID {
				statuses = append(statuses, details.Status)
			}
		}
		return statuses
	}

	pushStatusReport([]*androidmanagement.ApplicationReport{installed(apps[0])}, []*androidmanagement.NonComplianceDetail{notApproved(apps[1])})
	require.Equal(t, map[string]fleet.SoftwareInstallerStatus{
		apps[0].Name: fleet.SoftwareInstalled, apps[1].Name: fleet.SoftwareInstallFailed,
	}, hostStatuses())

	// The device later reports the failed app installed, which overrides the failure.
	pushStatusReport([]*androidmanagement.ApplicationReport{installed(apps[0]), installed(apps[1])}, nil)
	require.Equal(t, map[string]fleet.SoftwareInstallerStatus{
		apps[0].Name: fleet.SoftwareInstalled, apps[1].Name: fleet.SoftwareInstalled,
	}, hostStatuses())
	require.Equal(t, []string{string(fleet.SoftwareInstalled)}, installActivityStatuses(apps[0]))
	require.Equal(t, []string{string(fleet.SoftwareInstallFailed), string(fleet.SoftwareInstalled)}, installActivityStatuses(apps[1]))

	reEnroll := func(appReports []*androidmanagement.ApplicationReport) {
		device := androidmanagement.Device{Name: deviceInfo.Name, EnrollmentTokenData: deviceInfo.EnrollmentTokenData}
		if len(appReports) > 0 {
			device.AppliedPolicyName = fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host.UUID)
			device.AppliedPolicyVersion = policyVersion
			device.ApplicationReports = appReports
			device.LastPolicySyncTime = time.Now().Format(time.RFC3339Nano)
		}
		msg := enrollmentMessageWithEnterpriseSpecificID(t, device, host.UUID)
		s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &android_service.PubSubPushRequest{PubSubMessage: *msg},
			http.StatusOK, "token", string(pubSubToken.Value))
		// Re-enrolling happens well after the previous enrollment's installs were created,
		// which the host's second-precision enrollment time can't tell apart in a fast test.
		// Moved back only after the enrollment message, which sees the previous enrollment
		// time and must leave those installs alone on its own.
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(t.Context(),
				`UPDATE host_vpp_software_installs SET created_at = created_at - INTERVAL 1 MINUTE WHERE host_id = ?`, host.ID)
			return err
		})
	}

	// After re-enrolling, a failure of the new install is overridden too, and the
	// previous enrollment's installs are left alone.
	reEnroll(nil)
	s.runWorkerUntilDoneWithChecks(true)
	require.Equal(t, map[string]fleet.SoftwareInstallerStatus{
		apps[0].Name: fleet.SoftwareInstallPending, apps[1].Name: fleet.SoftwareInstallPending,
	}, hostStatuses())

	pushStatusReport([]*androidmanagement.ApplicationReport{installed(apps[0])}, []*androidmanagement.NonComplianceDetail{notApproved(apps[1])})
	pushStatusReport([]*androidmanagement.ApplicationReport{installed(apps[0]), installed(apps[1])}, nil)
	require.Equal(t, map[string]fleet.SoftwareInstallerStatus{
		apps[0].Name: fleet.SoftwareInstalled, apps[1].Name: fleet.SoftwareInstalled,
	}, hostStatuses())
	// re-enrolling cleared the host's past activities, so these are only the new enrollment's
	require.Equal(t, []string{string(fleet.SoftwareInstalled)}, installActivityStatuses(apps[0]))
	require.Equal(t, []string{string(fleet.SoftwareInstallFailed), string(fleet.SoftwareInstalled)}, installActivityStatuses(apps[1]))

	// Re-enroll while an install is still pending, which fails it. The device reporting the
	// app installed, in the enrollment message or before the setup experience creates the
	// new installs, must not override that failure, or the app would get a second
	// "installed" activity once its new install is verified.
	reEnroll(nil)
	s.runWorkerUntilDoneWithChecks(true)
	pushStatusReport([]*androidmanagement.ApplicationReport{installed(apps[0])}, nil)
	require.Equal(t, map[string]fleet.SoftwareInstallerStatus{
		apps[0].Name: fleet.SoftwareInstalled, apps[1].Name: fleet.SoftwareInstallPending,
	}, hostStatuses())

	reEnroll([]*androidmanagement.ApplicationReport{installed(apps[0]), installed(apps[1])})
	pushStatusReport([]*androidmanagement.ApplicationReport{installed(apps[0]), installed(apps[1])}, nil)
	require.Equal(t, map[string]fleet.SoftwareInstallerStatus{
		apps[0].Name: fleet.SoftwareInstalled, apps[1].Name: fleet.SoftwareInstallFailed,
	}, hostStatuses())
	require.Equal(t, []string{string(fleet.SoftwareInstallFailed)}, installActivityStatuses(apps[1]))

	s.runWorkerUntilDoneWithChecks(true)
	pushStatusReport([]*androidmanagement.ApplicationReport{installed(apps[0]), installed(apps[1])}, nil)
	require.Equal(t, map[string]fleet.SoftwareInstallerStatus{
		apps[0].Name: fleet.SoftwareInstalled, apps[1].Name: fleet.SoftwareInstalled,
	}, hostStatuses())
	require.Equal(t, []string{string(fleet.SoftwareInstalled)}, installActivityStatuses(apps[0]))
	require.Equal(t, []string{string(fleet.SoftwareInstallFailed), string(fleet.SoftwareInstalled)}, installActivityStatuses(apps[1]))
}

func (s *integrationMDMTestSuite) TestAndroidAppConfiguration() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	s.enableAndroidMDM(t)

	// add some android apps
	app1 := &fleet.VPPApp{
		AdamID:           "com.test1",
		Platform:         fleet.AndroidPlatform,
		Name:             "Test1",
		BundleIdentifier: "com.test1",
		IconURL:          "https://example.com/1",
	}
	app2 := &fleet.VPPApp{
		AdamID:           "com.test2",
		Platform:         fleet.AndroidPlatform,
		Name:             "Test2",
		BundleIdentifier: "com.test2",
		IconURL:          "https://example.com/2",
	}
	app3 := &fleet.VPPApp{
		AdamID:           "com.test3",
		Platform:         fleet.AndroidPlatform,
		Name:             "Test3",
		BundleIdentifier: "com.test3",
		IconURL:          "https://example.com/3",
	}

	androidApps := []*fleet.VPPApp{app1, app2, app3}
	s.androidAPIClient.EnterprisesApplicationsFunc = func(ctx context.Context, enterpriseName string, packageName string) (*androidmanagement.Application, error) {
		for _, app := range androidApps {
			if app.AdamID == packageName {
				return &androidmanagement.Application{IconUrl: app.IconURL, Title: app.Name}, nil
			}
		}
		return nil, &notFoundError{}
	}

	// vars have no need for a mutex, protected via runWorkerUntilDone
	var (
		// records the appPolicies received in the ModifyPolicyApplications calls
		patchAppsPolicies  [][]*androidmanagement.ApplicationPolicy
		patchAppsCallCount int
	)
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, appPolicies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		patchAppsCallCount++
		patchAppsPolicies = append(patchAppsPolicies, appPolicies)

		return &androidmanagement.Policy{Version: int64(patchAppsCallCount)}, nil
	}
	s.androidAPIClient.EnterprisesPoliciesPatchFunc = func(ctx context.Context, policyName string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
		if !opts.OnlyUpdateApps {
			return &androidmanagement.Policy{}, nil
		}
		patchAppsCallCount++
		patchAppsPolicies = append(patchAppsPolicies, policy.Applications)

		return &androidmanagement.Policy{Version: int64(patchAppsCallCount)}, nil
	}

	// add Android app 1
	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID: app1.AdamID,
		Platform:   fleet.AndroidPlatform,
	}, http.StatusOK, &addAppResp)
	app1TitleID := addAppResp.TitleID

	// add Android app 2
	addAppResp = addAppStoreAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID: app2.AdamID,
		Platform:   fleet.AndroidPlatform,
	}, http.StatusOK, &addAppResp)
	app2TitleID := addAppResp.TitleID

	require.NotEqual(t, app1TitleID, app2TitleID)

	s.runWorkerUntilDoneWithChecks(true)

	// worker should have done nothing (no host to add apps to yet)
	require.Empty(t, patchAppsPolicies)
	patchAppsPolicies = nil

	var patchAppResp updateAppStoreAppResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app1TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{"managedConfiguration": 1}`),
	}, http.StatusOK, &patchAppResp)

	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app2TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{"managedConfiguration": 2}`),
	}, http.StatusOK, &patchAppResp)

	// add app 1 and 2 to Android setup experience
	var putResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/latest/fleet/setup_experience/software", &putSetupExperienceSoftwareRequest{
		Platform: string(fleet.AndroidPlatform),
		TeamID:   0,
		TitleIDs: []uint{app1TitleID, app2TitleID},
	}, http.StatusOK, &putResp)

	host, _, _ := s.createAndEnrollAndroidDevice(t, "test-android", nil, false)

	s.runWorkerUntilDoneWithChecks(true)

	// worker should have (in any order due to staggered job queuing):
	// - cleared the host policy's apps (from the host enrollment)
	// - added the Fleet agent to the host's policy (from the host enrollment, via prepareHostSpecificPolicy)
	// - replaced the host's app list with all apps available (for self-service) and the Fleet agent, from the host enrollment
	// - installed the apps, from the host enrollment
	// The jobs from the PATCH config changes skip the host: it hasn't reported applying its own
	// policy yet, and gets the configs from the enrollment calls instead.
	require.Len(t, patchAppsPolicies, 4)

	type appCall struct {
		PackageName          string
		InstallType          string
		ManagedConfiguration string
	}
	var appCalls []appCall
	var fleetAgentCount int
	for _, policies := range patchAppsPolicies {
		for _, p := range policies {
			if p.PackageName == "com.fleetdm.agent" {
				fleetAgentCount++
				require.Equal(t, "FORCE_INSTALLED", p.InstallType)
				require.Contains(t, string(p.ManagedConfiguration), "server_url")
				require.Contains(t, string(p.ManagedConfiguration), "host_uuid")
				continue
			}
			appCalls = append(appCalls, appCall{p.PackageName, p.InstallType, string(p.ManagedConfiguration)})
		}
	}
	require.Equal(t, 2, fleetAgentCount, "fleet agent should be added at enrollment and kept when the app list is replaced")
	require.ElementsMatch(t, []appCall{
		// app1+app2 made available during enrollment (self-service)
		{app1.VPPAppID.AdamID, "AVAILABLE", "1"},
		{app2.VPPAppID.AdamID, "AVAILABLE", "2"},
		// app1+app2 installed during enrollment (setup experience)
		{app1.VPPAppID.AdamID, "PREINSTALLED", "1"},
		{app2.VPPAppID.AdamID, "PREINSTALLED", "2"},
	}, appCalls)

	patchAppsPolicies = nil

	// the device reports applying its own policy, so later app changes are sent to it
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(t.Context(), `UPDATE android_devices SET applied_policy_id = ? WHERE host_id = ?`, host.UUID, host.ID)
		return err
	})

	// add app3 to Fleet
	addAppResp = addAppStoreAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID: app3.AdamID,
		Platform:   fleet.AndroidPlatform,
	}, http.StatusOK, &addAppResp)
	app3TitleID := addAppResp.TitleID

	s.runWorkerUntilDoneWithChecks(true)

	// worker should have:
	// 1. made the apps available to the host (for self-service), without any config provided
	require.Len(t, patchAppsPolicies, 1)
	require.ElementsMatch(t, []*androidmanagement.ApplicationPolicy{
		{PackageName: app3.VPPAppID.AdamID, InstallType: "AVAILABLE", ManagedConfiguration: googleapi.RawMessage{}, WorkProfileWidgets: "WORK_PROFILE_WIDGETS_UNSPECIFIED", CredentialProviderPolicy: "CREDENTIAL_PROVIDER_POLICY_UNSPECIFIED"}, // #nosec G101 - AMAPI enum value, not a credential
	}, patchAppsPolicies[0])

	patchAppsPolicies = nil

	// set a configuration for the app3
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app3TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{"managedConfiguration": 3}`),
	}, http.StatusOK, &patchAppResp)

	s.runWorkerUntilDoneWithChecks(true)

	// worker should have:
	// 1. made the app available with its config
	require.Len(t, patchAppsPolicies, 1)
	require.ElementsMatch(t, []*androidmanagement.ApplicationPolicy{
		{PackageName: app3.VPPAppID.AdamID, InstallType: "AVAILABLE", ManagedConfiguration: googleapi.RawMessage(`3`)},
	}, patchAppsPolicies[0])

	patchAppsPolicies = nil

	// patch but no change to the configuration for the app3
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app3TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{"managedConfiguration": 3}`),
	}, http.StatusOK, &patchAppResp)

	s.runWorkerUntilDoneWithChecks(true)

	require.Empty(t, patchAppsPolicies)

	// set the other two supported top-level keys for app3
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app3TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{"managedConfiguration": 3, "workProfileWidgets": "WORK_PROFILE_WIDGETS_ALLOWED", "credentialProviderPolicy": "CREDENTIAL_PROVIDER_ALLOWED"}`),
	}, http.StatusOK, &patchAppResp)

	s.runWorkerUntilDoneWithChecks(true)

	// worker should have:
	// 1. made the app available with both policies set, not just the managed configuration
	require.Len(t, patchAppsPolicies, 1)
	require.ElementsMatch(t, []*androidmanagement.ApplicationPolicy{
		{PackageName: app3.VPPAppID.AdamID, InstallType: "AVAILABLE", ManagedConfiguration: googleapi.RawMessage(`3`), WorkProfileWidgets: "WORK_PROFILE_WIDGETS_ALLOWED", CredentialProviderPolicy: "CREDENTIAL_PROVIDER_ALLOWED"}, // #nosec G101 - AMAPI enum value, not a credential
	}, patchAppsPolicies[0])

	patchAppsPolicies = nil

	// other ApplicationPolicy fields are sent to the host as well
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app3TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{"managedConfiguration": 3, "autoUpdateMode": "AUTO_UPDATE_HIGH_PRIORITY", "minimumVersionCode": 42, "permissionGrants": [{"permission": "android.permission.CAMERA", "policy": "GRANT"}]}`),
	}, http.StatusOK, &patchAppResp)

	s.runWorkerUntilDoneWithChecks(true)

	require.Len(t, patchAppsPolicies, 1)
	require.ElementsMatch(t, []*androidmanagement.ApplicationPolicy{
		{
			PackageName:          app3.VPPAppID.AdamID,
			InstallType:          "AVAILABLE",
			ManagedConfiguration: googleapi.RawMessage(`3`),
			AutoUpdateMode:       "AUTO_UPDATE_HIGH_PRIORITY",
			MinimumVersionCode:   42,
			PermissionGrants:     []*androidmanagement.PermissionGrant{{Permission: "android.permission.CAMERA", Policy: "GRANT"}},
		},
	}, patchAppsPolicies[0])

	patchAppsPolicies = nil

	// packageName and installType are reserved for Fleet
	for _, deniedConfig := range []string{`{"packageName": "com.example.other"}`, `{"installType": "FORCE_INSTALLED"}`} {
		res := s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app3TitleID), &updateAppStoreAppRequest{
			TeamID:        nil,
			Configuration: json.RawMessage(deniedConfig),
		}, http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), `"packageName" and "installType" are not supported as top-level keys.`)
	}

	s.runWorkerUntilDoneWithChecks(true)
	require.Empty(t, patchAppsPolicies)

	// patch with a different config just to trigger the worker
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app3TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{}`),
	}, http.StatusOK, &patchAppResp)

	// delete directly the config from the DB (it seems like to clear the config from
	// the API, an empty object needs to be passed, but that won't clear the config,
	// it just won't change any managedConfig/widgets - to really clear the config
	// from the API, the user would have to send something like:
	// {
	//   "managedConfiguration": null,
	//   "workProfileWidgets": "WORK_PROFILE_WIDGETS_UNSPECIFIED"
	// }
	//
	// Is that how we want it to work?
	mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(t.Context(), `UPDATE vpp_apps_teams SET configuration = NULL WHERE adam_id = ? AND platform = 'android'`, app3.VPPAppID.AdamID)
		return err
	})

	s.runWorkerUntilDoneWithChecks(true)

	// worker should have:
	// 1. made the app available with its config cleared
	require.Len(t, patchAppsPolicies, 1)
	require.ElementsMatch(t, []*androidmanagement.ApplicationPolicy{
		{PackageName: app3.VPPAppID.AdamID, InstallType: "AVAILABLE", ManagedConfiguration: googleapi.RawMessage{}, WorkProfileWidgets: "WORK_PROFILE_WIDGETS_UNSPECIFIED", CredentialProviderPolicy: "CREDENTIAL_PROVIDER_POLICY_UNSPECIFIED"}, // #nosec G101 - AMAPI enum value, not a credential
	}, patchAppsPolicies[0])
}

func (s *integrationMDMTestSuite) TestSetupExperienceAndroidWithConfiguration() {
	t := s.T()
	ctx := t.Context()
	s.setSkipWorkerJobs(t)

	enterpriseID := s.enableAndroidMDM(t)

	// add 2 android apps
	app1 := &fleet.VPPApp{
		AdamID:           "com.test1",
		Platform:         fleet.AndroidPlatform,
		Name:             "Test1",
		BundleIdentifier: "com.test1",
		IconURL:          "https://example.com/1",
	}
	app2 := &fleet.VPPApp{
		AdamID:           "com.test2",
		Platform:         fleet.AndroidPlatform,
		Name:             "Test2",
		BundleIdentifier: "com.test2",
		IconURL:          "https://example.com/2",
	}

	androidApps := []*fleet.VPPApp{app1, app2}
	s.androidAPIClient.EnterprisesApplicationsFunc = func(ctx context.Context, enterpriseName string, packageName string) (*androidmanagement.Application, error) {
		for _, app := range androidApps {
			if app.AdamID == packageName {
				return &androidmanagement.Application{IconUrl: app.IconURL, Title: app.Name}, nil
			}
		}
		return nil, &notFoundError{}
	}

	var patchAppsCallCount int // no need for mutex, protected via runWorkerUntilDone
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, appPolicies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		patchAppsCallCount++
		return &androidmanagement.Policy{Version: int64(patchAppsCallCount)}, nil
	}

	// add Android app 1
	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID: app1.AdamID,
		Platform:   fleet.AndroidPlatform,
	}, http.StatusOK, &addAppResp)
	app1TitleID := addAppResp.TitleID

	// add Android app 2
	addAppResp = addAppStoreAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID: app2.AdamID,
		Platform:   fleet.AndroidPlatform,
	}, http.StatusOK, &addAppResp)
	app2TitleID := addAppResp.TitleID

	require.NotEqual(t, app1TitleID, app2TitleID)

	// add app 1 to Android setup experience
	var putResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/latest/fleet/setup_experience/software", &putSetupExperienceSoftwareRequest{
		Platform: string(fleet.AndroidPlatform),
		TeamID:   0,
		TitleIDs: []uint{app1TitleID},
	}, http.StatusOK, &putResp)

	s.runWorkerUntilDoneWithChecks(true)

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "test team", Secrets: []*fleet.EnrollSecret{{Secret: uuid.NewString()}}})
	require.NoError(t, err)

	// enroll a couple android devices on no-team and one on a team. Note host1 is company-owned
	host1, deviceInfo1, pubSubToken := s.createAndEnrollAndroidDevice(t, "test-android1", nil, true)
	host2, deviceInfo2, _ := s.createAndEnrollAndroidDevice(t, "test-android2", nil, false)
	host3, _, _ := s.createAndEnrollAndroidDevice(t, "test-android3", &tm.ID, false)

	s.runWorkerUntilDoneWithChecks(true)

	// hosts 1 and 2 have app1 pending install
	getHostSw := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	app1Host1CmdUUID := getHostSw.Software[0].AppStoreApp.LastInstall.CommandUUID
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)
	require.NotEmpty(t, app1Host1CmdUUID)

	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host2.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	app1Host2CmdUUID := getHostSw.Software[0].AppStoreApp.LastInstall.CommandUUID
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)
	require.NotEmpty(t, app1Host2CmdUUID)

	// host 3 has no apps available to install
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host3.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Empty(t, getHostSw.Software)

	// make the software installed on host1, failed on host2
	policyName := fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host1.UUID)
	reportMsg := statusReportMessageWithSerialNumber(
		t,
		androidmanagement.Device{
			Name:                 deviceInfo1.Name,
			EnrollmentTokenData:  deviceInfo1.EnrollmentTokenData,
			AppliedPolicyName:    policyName,
			AppliedPolicyVersion: 10,
			ApplicationReports: []*androidmanagement.ApplicationReport{
				{PackageName: app1.AdamID, State: "INSTALLED"},
			},
			LastPolicySyncTime: time.Now().Format(time.RFC3339Nano),
		},
		host1.HardwareSerial,
	)
	req := android_service.PubSubPushRequest{PubSubMessage: *reportMsg}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubSubToken.Value))

	policyName = fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host2.UUID)
	reportMsg = statusReportMessageWithEnterpriseSpecificID(
		t,
		androidmanagement.Device{
			Name:                 deviceInfo2.Name,
			EnrollmentTokenData:  deviceInfo2.EnrollmentTokenData,
			AppliedPolicyName:    policyName,
			AppliedPolicyVersion: 20,
			NonComplianceDetails: []*androidmanagement.NonComplianceDetail{
				{PackageName: app1.AdamID, NonComplianceReason: "APP_NOT_INSTALLED"},
			},
			LastPolicySyncTime: time.Now().Format(time.RFC3339Nano),
		},
		host2.UUID,
	)
	req = android_service.PubSubPushRequest{PubSubMessage: *reportMsg}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubSubToken.Value))

	// the pending install should now be verified for host1
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstalled, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// the pending install should now be failed for host2
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host2.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallFailed, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// set a configuration for app1
	var patchAppResp updateAppStoreAppResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app1TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{"managedConfiguration": 1}`),
	}, http.StatusOK, &patchAppResp)

	s.runWorkerUntilDoneWithChecks(true)

	// the verified install should now be back to pending for host1
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// the failed install is still failed for host2
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host2.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallFailed, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// and still no apps for host3
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host3.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Empty(t, getHostSw.Software)

	getPolicyVersion := func(hostID uint, appID string) int64 {
		var version int64
		mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, tx, &version, `
				SELECT CAST(associated_event_id AS SIGNED) FROM host_vpp_software_installs WHERE host_id = ? AND adam_id = ?`,
				hostID, appID)
		})
		return version
	}
	versionBefore := getPolicyVersion(host1.ID, app1.AdamID)

	// update the configuration for app1
	patchAppResp = updateAppStoreAppResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app1TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{"managedConfiguration": 2}`),
	}, http.StatusOK, &patchAppResp)

	s.runWorkerUntilDoneWithChecks(true)

	// install for host1 will still be pending, but will now be verified only when that
	// latest policy version will get reported
	versionAfter := getPolicyVersion(host1.ID, app1.AdamID)
	require.Greater(t, versionAfter, versionBefore)

	// the pending install is still pending for host1
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// reporting it as installed with the previous policy version does not make it verified
	policyName = fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host1.UUID)
	reportMsg = statusReportMessageWithSerialNumber(
		t,
		androidmanagement.Device{
			Name:                 deviceInfo1.Name,
			EnrollmentTokenData:  deviceInfo1.EnrollmentTokenData,
			AppliedPolicyName:    policyName,
			AppliedPolicyVersion: versionBefore,
			ApplicationReports: []*androidmanagement.ApplicationReport{
				{PackageName: app1.AdamID, State: "INSTALLED"},
			},
			LastPolicySyncTime: time.Now().Format(time.RFC3339Nano),
		},
		host1.HardwareSerial,
	)
	req = android_service.PubSubPushRequest{PubSubMessage: *reportMsg}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubSubToken.Value))

	// still pending
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// reporting it as installed with the latest policy version makes it verified
	policyName = fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host1.UUID)
	reportMsg = statusReportMessageWithSerialNumber(
		t,
		androidmanagement.Device{
			Name:                 deviceInfo1.Name,
			EnrollmentTokenData:  deviceInfo1.EnrollmentTokenData,
			AppliedPolicyName:    policyName,
			AppliedPolicyVersion: versionAfter,
			ApplicationReports: []*androidmanagement.ApplicationReport{
				{PackageName: app1.AdamID, State: "INSTALLED"},
			},
			LastPolicySyncTime: time.Now().Format(time.RFC3339Nano),
		},
		host1.HardwareSerial,
	)
	req = android_service.PubSubPushRequest{PubSubMessage: *reportMsg}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubSubToken.Value))

	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstalled, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// update app1 again, but configuration stays the same
	patchAppResp = updateAppStoreAppResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app1TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{"managedConfiguration": 2}`),
	}, http.StatusOK, &patchAppResp)

	s.runWorkerUntilDoneWithChecks(true)

	// status stays installed
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstalled, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)
}

// A re-enrolled Android host reuses its host-specific policy, so its previous fleet's apps must
// be cleared from that policy when it enrolls, and its new fleet's apps must replace them before
// the device is moved to that policy.
func (s *integrationMDMTestSuite) TestSetupExperienceAndroidReenrollIntoOtherFleetReplacesApps() {
	t := s.T()
	ctx := t.Context()
	s.setSkipWorkerJobs(t)
	enterpriseID := s.enableAndroidMDM(t)

	const (
		previousFleetApp = "com.reenroll.previousfleet"
		newFleetApp      = "com.reenroll.newfleet"
		vitalConfigApp   = "com.reenroll.vitalconfig"
		fleetAgent       = "com.fleetdm.agent"
	)
	s.androidAPIClient.EnterprisesApplicationsFunc = func(ctx context.Context, enterpriseName string, packageName string) (*androidmanagement.Application, error) {
		return &androidmanagement.Application{IconUrl: "https://example.com/1.jpg", Title: packageName}, nil
	}

	// the AMAPI calls that change a host policy's apps or move a device to a policy, in order
	var mu sync.Mutex
	var calls []string
	record := func(policyName, call string) {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, policyName+": "+call)
	}
	callsFor := func(policyName string) []string {
		mu.Lock()
		defer mu.Unlock()
		var policyCalls []string
		for _, call := range calls {
			if rest, ok := strings.CutPrefix(call, policyName+": "); ok {
				policyCalls = append(policyCalls, rest)
			}
		}
		calls = nil
		return policyCalls
	}
	s.androidAPIClient.EnterprisesPoliciesPatchFunc = func(ctx context.Context, policyName string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
		if opts.OnlyUpdateApps {
			packages := make([]string, 0, len(policy.Applications))
			for _, app := range policy.Applications {
				packages = append(packages, app.PackageName)
			}
			record(policyName, "set apps "+strings.Join(packages, ","))
		}
		return &androidmanagement.Policy{Version: 1}, nil
	}
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, appPolicies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		for _, app := range appPolicies {
			record(policyName, "add app "+app.PackageName+" "+app.InstallType)
		}
		return &androidmanagement.Policy{Version: 1}, nil
	}
	s.androidAPIClient.EnterprisesDevicesPatchFunc = func(ctx context.Context, deviceName string, device *androidmanagement.Device) (*androidmanagement.Device, error) {
		record(device.PolicyName, "move device")
		return &androidmanagement.Device{}, nil
	}

	createFleet := func(name string) (uint, string) {
		var createTeamResp teamResponse
		s.DoJSON("POST", "/api/latest/fleet/teams", createTeamRequest{
			Name: new(t.Name() + name),
		}, http.StatusOK, &createTeamResp)
		secret := uuid.NewString()
		require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, &createTeamResp.Team.ID, []*fleet.EnrollSecret{{Secret: secret}}))
		return createTeamResp.Team.ID, secret
	}
	previousFleetID, previousFleetSecret := createFleet("-previous")
	emptyFleetID, emptyFleetSecret := createFleet("-empty")
	newFleetID, newFleetSecret := createFleet("-new")

	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID: previousFleetApp,
		Platform:   fleet.AndroidPlatform,
		TeamID:     &previousFleetID,
	}, http.StatusOK)
	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID: newFleetApp,
		Platform:   fleet.AndroidPlatform,
		TeamID:     &newFleetID,
	}, http.StatusOK, &addAppResp)
	s.Do("PUT", "/api/latest/fleet/setup_experience/software", &putSetupExperienceSoftwareRequest{
		Platform: string(fleet.AndroidPlatform),
		TeamID:   newFleetID,
		TitleIDs: []uint{addAppResp.TitleID},
	}, http.StatusOK)
	// a newly enrolled host has no value for this vital, so this app can't be made available to it
	vital, err := s.ds.CreateCustomHostVital(ctx, t.Name())
	require.NoError(t, err)
	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID:    vitalConfigApp,
		Platform:      fleet.AndroidPlatform,
		TeamID:        &newFleetID,
		Configuration: json.RawMessage(fmt.Sprintf(`{"managedConfiguration": {"assetTag": "$%s%d"}}`, fleet.CustomHostVitalPrefix, vital.ID)),
	}, http.StatusOK)
	s.runWorkerUntilDoneWithChecks(true)
	callsFor("")

	assets, err := s.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidPubSubToken}, nil)
	require.NoError(t, err)
	pubsubToken := assets[fleet.MDMAssetAndroidPubSubToken]

	for _, companyOwned := range []bool{false, true} {
		identifier := strings.ToUpper(uuid.NewString())
		enroll := func(secret string, fleetID uint) *fleet.Host {
			deviceInfo := androidmanagement.Device{
				Name:                createAndroidDeviceID(identifier[:8]),
				EnrollmentTokenData: fmt.Sprintf(`{"EnrollSecret": "%s"}`, secret),
			}
			msg := enrollmentMessageWithEnterpriseSpecificID(t, deviceInfo, identifier)
			if companyOwned {
				msg = enrollmentMessageWithSerialNumber(t, deviceInfo, identifier)
			}
			s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &android_service.PubSubPushRequest{PubSubMessage: *msg},
				http.StatusOK, "token", string(pubsubToken.Value))

			var hosts listHostsResponse
			s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hosts, "query", identifier)
			require.Len(t, hosts.Hosts, 1)
			host := hosts.Hosts[0].Host
			require.NotNil(t, host.TeamID)
			require.Equal(t, fleetID, *host.TeamID, "company owned: %t", companyOwned)

			// the enrollment itself clears the reused policy's apps, before the profile
			// reconciler or the setup experience can move the device to it
			policyName := fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host.UUID)
			require.Equal(t, []string{"set apps "}, callsFor(policyName), "company owned: %t", companyOwned)

			s.runWorkerUntilDoneWithChecks(true)
			return host
		}

		host := enroll(previousFleetSecret, previousFleetID)
		policyName := fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host.UUID)
		require.Equal(t, []string{
			"add app " + fleetAgent + " FORCE_INSTALLED",
			"set apps " + previousFleetApp + "," + fleetAgent,
			"move device",
		}, callsFor(policyName), "company owned: %t", companyOwned)

		// re-enrolling into another fleet reuses the host policy
		require.Equal(t, host.ID, enroll(emptyFleetSecret, emptyFleetID).ID)
		require.Equal(t, []string{
			"add app " + fleetAgent + " FORCE_INSTALLED",
			"set apps " + fleetAgent,
			"move device",
		}, callsFor(policyName), "company owned: %t", companyOwned)

		require.Equal(t, host.ID, enroll(newFleetSecret, newFleetID).ID)
		require.Equal(t, []string{
			"add app " + fleetAgent + " FORCE_INSTALLED",
			// the app whose config the host can't supply a value for is left out
			"set apps " + newFleetApp + "," + fleetAgent,
			"move device",
			"add app " + newFleetApp + " PREINSTALLED",
		}, callsFor(policyName), "company owned: %t", companyOwned)

		// a deleted host re-enrolls as a new host, which still reuses the host policy
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK)
		newHost := enroll(emptyFleetSecret, emptyFleetID)
		require.NotEqual(t, host.ID, newHost.ID, "company owned: %t", companyOwned)
		require.Equal(t, host.UUID, newHost.UUID, "company owned: %t", companyOwned)
		require.Equal(t, []string{
			"add app " + fleetAgent + " FORCE_INSTALLED",
			"set apps " + fleetAgent,
			"move device",
		}, callsFor(policyName), "company owned: %t", companyOwned)
	}
}
