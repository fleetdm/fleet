package software_ingestion

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSoftwareIngestionService_IngestOsquerySoftware(t *testing.T) {
	// Mock the datastore (after running `make mock`)
	mockDS := &MockSoftwareDatastore{}

	// Create the service
	logger := log.NewNopLogger()
	service := NewService(mockDS, logger)

	// Test data
	host := &fleet.Host{
		ID:       1,
		Platform: "darwin",
	}

	softwareRows := []map[string]string{
		{
			"name":                "Google Chrome",
			"version":             "118.0.5993.117",
			"bundle_identifier":   "com.google.Chrome",
			"source":              "apps",
			"vendor":              "Google LLC",
			"installed_path":      "/Applications/Google Chrome.app",
			"last_opened_at":      "1699123456",
		},
		{
			"name":        "Visual Studio Code",
			"version":     "1.84.2",
			"source":      "apps",
			"vendor":      "Microsoft Corporation",
			"last_opened_at": "0",
		},
	}

	// Set up mock expectations
	expectedSoftware := []fleet.Software{
		{
			Name:             "Google Chrome",
			Version:          "118.0.5993.117",
			BundleIdentifier: &[]string{"com.google.Chrome"}[0],
			Source:           "apps",
			Vendor:           "Google LLC",
		},
		{
			Name:    "Visual Studio Code",
			Version: "1.84.2",
			Source:  "apps",
			Vendor:  "Microsoft Corporation",
		},
	}

	mockResult := &fleet.UpdateHostSoftwareDBResult{}
	mockDS.UpdateHostSoftwareFunc = func(ctx context.Context, hostID uint, software []fleet.Software) (*fleet.UpdateHostSoftwareDBResult, error) {
		assert.Equal(t, host.ID, hostID)
		assert.Len(t, software, 2)

		// Verify the first software entry
		assert.Equal(t, expectedSoftware[0].Name, software[0].Name)
		assert.Equal(t, expectedSoftware[0].Version, software[0].Version)
		assert.Equal(t, expectedSoftware[0].Source, software[0].Source)

		return mockResult, nil
	}

	mockDS.UpdateHostSoftwareInstalledPathsFunc = func(ctx context.Context, hostID uint, reported map[string]struct{}, mutationResults *fleet.UpdateHostSoftwareDBResult) error {
		assert.Equal(t, host.ID, hostID)
		assert.Equal(t, mockResult, mutationResults)
		assert.Len(t, reported, 1) // Only Google Chrome has an installed path
		return nil
	}

	// Execute the test
	err := service.IngestOsquerySoftware(context.Background(), host.ID, host, softwareRows)

	// Verify results
	require.NoError(t, err)
	assert.True(t, mockDS.UpdateHostSoftwareInvoked)
	assert.True(t, mockDS.UpdateHostSoftwareInstalledPathsInvoked)
}

func TestSoftwareIngestionService_IngestMDMSoftware(t *testing.T) {
	// Mock the datastore (after running `make mock`)
	mockDS := &MockSoftwareDatastore{}

	// Create the service
	logger := log.NewNopLogger()
	service := NewService(mockDS, logger)

	// Test data
	host := &fleet.Host{
		ID:       1,
		Platform: "ios",
	}

	inputSoftware := []fleet.Software{
		{
			Name:             "Evernote",
			Version:          "10.98.0",
			BundleIdentifier: &[]string{"com.evernote.iPhone.Evernote"}[0],
			Source:           "", // Should be set by the service
		},
		{
			Name:             "TestFlight",
			Version:          "3.4.1",
			BundleIdentifier: &[]string{"com.apple.TestFlight"}[0],
			Source:           "", // Should be set by the service
			Vendor:           "", // Should be extracted from bundle ID
		},
	}

	// Set up mock expectations
	mockResult := &fleet.UpdateHostSoftwareDBResult{}
	mockDS.UpdateHostSoftwareFunc = func(ctx context.Context, hostID uint, software []fleet.Software) (*fleet.UpdateHostSoftwareDBResult, error) {
		assert.Equal(t, host.ID, hostID)
		assert.Len(t, software, 2)

		// Verify iOS source was set
		assert.Equal(t, "ios_apps", software[0].Source)
		assert.Equal(t, "ios_apps", software[1].Source)

		// Verify vendor extraction from bundle ID
		assert.Equal(t, "apple", software[1].Vendor)

		return mockResult, nil
	}

	mockDS.UpdateHostSoftwareInstalledPathsFunc = func(ctx context.Context, hostID uint, reported map[string]struct{}, mutationResults *fleet.UpdateHostSoftwareDBResult) error {
		assert.Equal(t, host.ID, hostID)
		assert.Equal(t, mockResult, mutationResults)
		assert.Len(t, reported, 0) // MDM software doesn't have installed paths
		return nil
	}

	// Execute the test
	err := service.IngestMDMSoftware(context.Background(), host.ID, host, inputSoftware)

	// Verify results
	require.NoError(t, err)
	assert.True(t, mockDS.UpdateHostSoftwareInvoked)
	assert.True(t, mockDS.UpdateHostSoftwareInstalledPathsInvoked)
}

func TestDetermineMDMSoftwareSource(t *testing.T) {
	logger := log.NewNopLogger()
	service := NewService(nil, logger).(*service)

	tests := []struct {
		platform string
		expected string
	}{
		{"ios", "ios_apps"},
		{"ipados", "ipados_apps"},
		{"darwin", "app_store_apps"},
		{"windows", "mdm_apps"},
		{"unknown", "mdm_apps"},
	}

	for _, test := range tests {
		t.Run(test.platform, func(t *testing.T) {
			result := service.determineMDMSoftwareSource(test.platform)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestApplyMDMSoftwareTransformations(t *testing.T) {
	logger := log.NewNopLogger()
	service := NewService(nil, logger).(*service)

	host := &fleet.Host{Platform: "ios"}

	tests := []struct {
		name           string
		input          fleet.Software
		expectedVendor string
	}{
		{
			name: "extract vendor from Apple bundle ID",
			input: fleet.Software{
				Name:             "TestFlight",
				BundleIdentifier: &[]string{"com.apple.TestFlight"}[0],
				Vendor:           "",
			},
			expectedVendor: "apple",
		},
		{
			name: "extract vendor from Google bundle ID",
			input: fleet.Software{
				Name:             "Gmail",
				BundleIdentifier: &[]string{"com.google.Gmail"}[0],
				Vendor:           "",
			},
			expectedVendor: "google",
		},
		{
			name: "don't override existing vendor",
			input: fleet.Software{
				Name:             "Custom App",
				BundleIdentifier: &[]string{"com.company.app"}[0],
				Vendor:           "Custom Vendor",
			},
			expectedVendor: "Custom Vendor",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			software := test.input
			service.applyMDMSoftwareTransformations(host, &software)
			assert.Equal(t, test.expectedVendor, software.Vendor)
		})
	}
}