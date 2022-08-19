package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func (s *integrationTestSuite) TestDeviceAuthenticatedEndpoints() {
	t := s.T()

	hosts := s.createHosts(t)
	ac, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	ac.OrgInfo.OrgLogoURL = "http://example.com/logo"
	err = s.ds.SaveAppConfig(context.Background(), ac)
	require.NoError(t, err)

	// create some mappings and MDM/Munki data
	s.ds.ReplaceHostDeviceMapping(context.Background(), hosts[0].ID, []*fleet.HostDeviceMapping{
		{HostID: hosts[0].ID, Email: "a@b.c", Source: "google_chrome_profiles"},
		{HostID: hosts[0].ID, Email: "b@b.c", Source: "google_chrome_profiles"},
	})
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), hosts[0].ID, true, "url", false))
	require.NoError(t, s.ds.SetOrUpdateMunkiVersion(context.Background(), hosts[0].ID, "1.3.0"))
	// create a battery for hosts[0]
	require.NoError(t, s.ds.ReplaceHostBatteries(context.Background(), hosts[0].ID, []*fleet.HostBattery{
		{HostID: hosts[0].ID, SerialNumber: "a", CycleCount: 1, Health: "Good"},
	}))

	// create an auth token for hosts[0]
	token := "much_valid"
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `INSERT INTO host_device_auth (host_id, token) VALUES (?, ?)`, hosts[0].ID, token)
		return err
	})

	// get host without token
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/", nil, http.StatusNotFound)
	res.Body.Close()

	// get host with invalid token
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/no_such_token", nil, http.StatusUnauthorized)
	res.Body.Close()

	// get host with valid token
	var getHostResp getDeviceHostResponse
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token, nil, http.StatusOK)
	json.NewDecoder(res.Body).Decode(&getHostResp)
	res.Body.Close()
	require.Equal(t, hosts[0].ID, getHostResp.Host.ID)
	require.False(t, getHostResp.Host.RefetchRequested)
	require.Equal(t, "http://example.com/logo", getHostResp.OrgLogoURL)
	require.Nil(t, getHostResp.Host.Policies)
	require.NotNil(t, getHostResp.Host.Batteries)
	require.Equal(t, &fleet.HostBattery{CycleCount: 1, Health: "Normal"}, (*getHostResp.Host.Batteries)[0])
	hostDevResp := getHostResp.Host

	// make request for same host on the host details API endpoint, responses should match, except for policies
	getHostResp = getDeviceHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[0].ID), nil, http.StatusOK, &getHostResp)
	getHostResp.Host.Policies = nil
	require.Equal(t, hostDevResp, getHostResp.Host)

	// request a refetch for that valid host
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/device/"+token+"/refetch", nil, http.StatusOK)
	res.Body.Close()

	// host should have that flag turned to true
	getHostResp = getDeviceHostResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token, nil, http.StatusOK)
	json.NewDecoder(res.Body).Decode(&getHostResp)
	res.Body.Close()
	require.True(t, getHostResp.Host.RefetchRequested)

	// request a refetch for an invalid token
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/device/no_such_token/refetch", nil, http.StatusUnauthorized)
	res.Body.Close()

	// list device mappings for valid token
	var listDMResp listHostDeviceMappingResponse
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/device_mapping", nil, http.StatusOK)
	json.NewDecoder(res.Body).Decode(&listDMResp)
	res.Body.Close()
	require.Equal(t, hosts[0].ID, listDMResp.HostID)
	require.Len(t, listDMResp.DeviceMapping, 2)
	devDMs := listDMResp.DeviceMapping

	// compare response with standard list device mapping API for that same host
	listDMResp = listHostDeviceMappingResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[0].ID), nil, http.StatusOK, &listDMResp)
	require.Equal(t, hosts[0].ID, listDMResp.HostID)
	require.Equal(t, devDMs, listDMResp.DeviceMapping)

	// list device mappings for invalid token
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/no_such_token/device_mapping", nil, http.StatusUnauthorized)
	res.Body.Close()

	// get macadmins for valid token
	var getMacadm macadminsDataResponse
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/macadmins", nil, http.StatusOK)
	json.NewDecoder(res.Body).Decode(&getMacadm)
	res.Body.Close()
	require.Equal(t, "1.3.0", getMacadm.Macadmins.Munki.Version)
	devMacadm := getMacadm.Macadmins

	// compare response with standard macadmins API for that same host
	getMacadm = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hosts[0].ID), nil, http.StatusOK, &getMacadm)
	require.Equal(t, devMacadm, getMacadm.Macadmins)

	// get macadmins for invalid token
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/no_such_token/macadmins", nil, http.StatusUnauthorized)
	res.Body.Close()

	// response includes license info
	getHostResp = getDeviceHostResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token, nil, http.StatusOK)
	json.NewDecoder(res.Body).Decode(&getHostResp)
	res.Body.Close()
	require.NotNil(t, getHostResp.License)
	require.Equal(t, getHostResp.License.Tier, "free")

	// device policies are not accessible for free endpoints
	listPoliciesResp := listDevicePoliciesResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/policies", nil, http.StatusPaymentRequired)
	json.NewDecoder(res.Body).Decode(&getHostResp)
	res.Body.Close()
	require.Nil(t, listPoliciesResp.Policies)

	// get list of api features
	apiFeaturesResp := deviceAPIFeaturesResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/api_features", nil, http.StatusOK)
	json.NewDecoder(res.Body).Decode(&apiFeaturesResp)
	res.Body.Close()
	require.Nil(t, apiFeaturesResp.Err)
	require.NotNil(t, apiFeaturesResp.Features)
}

// TestDefaultTransparencyURL tests that Fleet Free licensees are restricted to the default transparency url.
func (s *integrationTestSuite) TestDefaultTransparencyURL() {
	t := s.T()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   t.Name(),
		NodeKey:         t.Name(),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	// create device token for host
	token := "token_test_default_transparency_url"
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `INSERT INTO host_device_auth (host_id, token) VALUES (?, ?)`, host.ID, token)
		return err
	})

	// confirm initial default url
	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.Equal(t, fleet.DefaultTransparencyURL, acResp.FleetDesktop.TransparencyURL)

	// confirm device endpoint returns initial default url
	deviceResp := &transparencyURLResponse{}
	rawResp := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/transparency", nil, http.StatusTemporaryRedirect)
	json.NewDecoder(rawResp.Body).Decode(deviceResp)
	rawResp.Body.Close()
	require.NoError(t, deviceResp.Err)
	require.Equal(t, fleet.DefaultTransparencyURL, rawResp.Header.Get("Location"))

	// empty string applies default url
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", fleet.AppConfig{FleetDesktop: fleet.FleetDesktopSettings{TransparencyURL: ""}}, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.Equal(t, fleet.DefaultTransparencyURL, acResp.FleetDesktop.TransparencyURL)

	// device endpoint returns default url
	deviceResp = &transparencyURLResponse{}
	rawResp = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/transparency", nil, http.StatusTemporaryRedirect)
	json.NewDecoder(rawResp.Body).Decode(deviceResp)
	rawResp.Body.Close()
	require.NoError(t, deviceResp.Err)
	require.Equal(t, fleet.DefaultTransparencyURL, rawResp.Header.Get("Location"))

	// modify transparency url with custom url fails
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", fleet.AppConfig{FleetDesktop: fleet.FleetDesktopSettings{TransparencyURL: "customURL"}}, http.StatusUnprocessableEntity, &acResp)

	// device endpoint still returns default url
	deviceResp = &transparencyURLResponse{}
	rawResp = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/transparency", nil, http.StatusTemporaryRedirect)
	json.NewDecoder(rawResp.Body).Decode(deviceResp)
	rawResp.Body.Close()
	require.NoError(t, deviceResp.Err)
	require.Equal(t, fleet.DefaultTransparencyURL, rawResp.Header.Get("Location"))
}

func (s *integrationTestSuite) TestDesktopRateLimit() {
	headers := map[string]string{
		"X-Forwarded-For": "1.2.3.4",
	}
	for i := 0; i < desktopRateLimitMaxBurst+1; i++ { // rate limiting off-by-one
		s.DoRawWithHeaders("GET", "/api/latest/fleet/device/"+uuid.NewString(), nil, http.StatusUnauthorized, headers).Body.Close()
	}
	s.DoRawWithHeaders("GET", "/api/latest/fleet/device/"+uuid.NewString(), nil, http.StatusTooManyRequests, headers).Body.Close()
}
