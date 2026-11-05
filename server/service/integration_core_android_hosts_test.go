package service

// Android host tests for the core (no-license) suite.
//
// Belongs here: Android-specific host behaviour visible without a license — UUID
// propagation, label membership, and storage reported through the API.
//
// Does not belong here: Android MDM enrollment and app management, which belong to
// the MDM suite and its own files.

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationTestSuite) TestAndroidHostUUIDPropagation() {
	t := s.T()
	ctx := context.Background()

	// Create an Android host with a specific UUID
	const expectedUUID = "TEST-UUID-12345-ANDROID"
	host := &fleet.AndroidHost{
		Host: &fleet.Host{
			Hostname:       "test-android-uuid",
			ComputerName:   "AndroidTestDevice",
			Platform:       "android",
			OSVersion:      "Android 15",
			Build:          "test-build-uuid",
			Memory:         2048,
			TeamID:         nil,
			HardwareSerial: "test-serial-uuid",
			HardwareModel:  "Pixel 8",
			HardwareVendor: "Google",
			UUID:           expectedUUID, // Set the UUID explicitly
		},
		Device: &android.Device{
			DeviceID:             strings.ReplaceAll(uuid.NewString(), "-", ""),
			EnterpriseSpecificID: new(expectedUUID),
			AppliedPolicyID:      new("1"),
			LastPolicySyncTime:   new(time.Now()),
		},
	}
	host.SetNodeKey(expectedUUID)

	// Create Android host
	androidHost, err := s.ds.NewAndroidHost(ctx, host, false)
	require.NoError(t, err)
	require.NotZero(t, androidHost.Host.ID)

	// Test 1: Get the host, verify UUID is present
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", androidHost.Host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host)
	require.Equal(t, expectedUUID, getHostResp.Host.UUID, "UUID should be returned in API response")
	require.Equal(t, "AndroidTestDevice", getHostResp.Host.ComputerName)

	// Test 2: Update the host, verify UUID is preserved
	updatedHost := androidHost
	updatedHost.Host.Hostname = "updated-android-hostname"
	updatedHost.Host.ComputerName = "UpdatedAndroidDevice"
	updatedHost.Host.OSVersion = "Android 16"
	updatedHost.Host.UUID = expectedUUID

	err = s.ds.UpdateAndroidHost(ctx, updatedHost, false, false)
	require.NoError(t, err)

	// Get the host again, verify UUID is still present
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", androidHost.Host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host)
	require.Equal(t, expectedUUID, getHostResp.Host.UUID, "UUID should be preserved after host update")
	require.Equal(t, "UpdatedAndroidDevice", getHostResp.Host.ComputerName)
	require.Equal(t, "Android 16", getHostResp.Host.OSVersion)

	// Test 3: List hosts, verify Android host UUID is included
	var listHostsResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsResp)

	// Find our Android host in the list
	var foundHost *fleet.HostResponse
	for _, h := range listHostsResp.Hosts {
		if h.ID == androidHost.Host.ID {
			foundHost = &h
			break
		}
	}
	require.NotNil(t, foundHost, "Android host should be in list response")
	require.Equal(t, expectedUUID, foundHost.UUID, "UUID should be present in list hosts response")

	// Test 4: AndroidHostLite returns UUID
	androidHostLite, err := s.ds.AndroidHostLite(ctx, expectedUUID)
	require.NoError(t, err)
	require.Equal(t, expectedUUID, androidHostLite.Host.UUID, "AndroidHostLite should return UUID")
}

func (s *integrationTestSuite) TestListAndroidHostsInLabel() {
	t := s.T()
	ctx := context.Background()

	hostIDs := createAndroidHosts(t, s.ds, 3, nil)
	notAndroidHost := createOrbitEnrolledHost(t, "darwin", "-4", s.ds)

	// list labels, has the built-in ones, capture All and Android
	var listResp fleet.ListLabelsResponse
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp)
	var allLblID, androidLblID uint
	for _, lbl := range listResp.Labels {
		switch lbl.Name {
		case fleet.BuiltinLabelNameAllHosts:
			allLblID = lbl.ID
		case fleet.BuiltinLabelNameAndroid:
			androidLblID = lbl.ID
		}
	}
	require.NotZero(t, allLblID)
	require.NotZero(t, androidLblID)

	err := s.ds.AddLabelsToHost(ctx, notAndroidHost.ID, []uint{allLblID})
	require.NoError(t, err)

	pluckHostIDs := func(hosts []fleet.HostResponse) []uint {
		ids := make([]uint, 0, len(hosts))
		for _, h := range hosts {
			ids = append(ids, h.ID)
		}
		return ids
	}

	// list hosts in all hosts
	var listHostsResp listHostsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", allLblID), nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, len(hostIDs)+1)
	wantIDs := append([]uint{notAndroidHost.ID}, hostIDs...)
	require.ElementsMatch(t, wantIDs, pluckHostIDs(listHostsResp.Hosts))

	// count hosts in label
	var countResp countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "label_id", fmt.Sprint(allLblID))
	require.Equal(t, len(hostIDs)+1, countResp.Count)

	// list android hosts
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", androidLblID), nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, len(hostIDs))
	require.ElementsMatch(t, hostIDs, pluckHostIDs(listHostsResp.Hosts))

	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "label_id", fmt.Sprint(androidLblID))
	require.Equal(t, len(hostIDs), countResp.Count)
}

func (s *integrationTestSuite) TestAndroidHostStorageInAPI() {
	t := s.T()
	ctx := context.Background()

	// Android host with storage data
	hostID := createAndroidHostWithStorage(t, s.ds, nil)

	// individual host endpoint
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostID), nil, http.StatusOK, &hostResp)

	require.NotNil(t, hostResp.Host)
	require.Equal(t, "android", hostResp.Host.Platform)

	// storage data is present in API response
	assert.Equal(t, 128.0, hostResp.Host.GigsTotalDiskSpace, "API should return total disk space")
	assert.Equal(t, 35.0, hostResp.Host.GigsDiskSpaceAvailable, "API should return available disk space")
	assert.InDelta(t, 27.34, hostResp.Host.PercentDiskSpaceAvailable, 0.1, "API should return disk space percentage")

	// list endpoint includes storage data
	var listResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp)

	var androidHost *fleet.HostResponse
	for _, host := range listResp.Hosts {
		if host.ID == hostID {
			androidHost = &host
			break
		}
	}

	require.NotNil(t, androidHost, "Android host should be in hosts list")
	require.Equal(t, "android", androidHost.Platform)

	// storage data in list endpoint
	assert.Equal(t, 128.0, androidHost.GigsTotalDiskSpace, "Host list should include total disk space")
	assert.Equal(t, 35.0, androidHost.GigsDiskSpaceAvailable, "Host list should include available disk space")
	assert.InDelta(t, 27.34, androidHost.PercentDiskSpaceAvailable, 0.1, "Host list should include disk space percentage")

	// clean up
	err := s.ds.DeleteHost(ctx, hostID)
	require.NoError(t, err)
}

func createAndroidHosts(t *testing.T, ds *mysql.Datastore, count int, teamID *uint) []uint {
	ids := make([]uint, 0, count)
	for i := range count {
		host := &fleet.AndroidHost{
			Host: &fleet.Host{
				Hostname:       fmt.Sprintf("hostname%d", i),
				ComputerName:   fmt.Sprintf("computer_name%d", i),
				Platform:       "android",
				OSVersion:      "Android 14",
				Build:          fmt.Sprintf("build%d", i),
				Memory:         1024,
				TeamID:         teamID,
				HardwareSerial: uuid.NewString(),
			},
			Device: &android.Device{
				DeviceID:             strings.ReplaceAll(uuid.NewString(), "-", ""), // Remove dashes to fit in VARCHAR(37)
				EnterpriseSpecificID: new(uuid.NewString()),
				AppliedPolicyID:      new("1"),
				LastPolicySyncTime:   new(time.Now().Add(-time.Hour)), // 1 hour ago
			},
		}
		host.SetNodeKey(*host.Device.EnterpriseSpecificID)
		ahost, err := ds.NewAndroidHost(context.Background(), host, false)
		require.NoError(t, err)
		ids = append(ids, ahost.Host.ID)
	}
	return ids
}

func createAndroidHostWithStorage(t *testing.T, ds *mysql.Datastore, teamID *uint) uint {
	return createAndroidHostForTest(t, ds, teamID, false)
}
