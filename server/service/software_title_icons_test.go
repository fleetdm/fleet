package service

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSoftwareTitleIcon(t *testing.T) {
	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: user})
	ds := new(mock.Store)

	mockIconStore := s3.SetupTestSoftwareTitleIconStore(t, "software-title-icons-unit-test", "icon-store-prefix")
	svc, _ := newTestService(t, ds, nil, nil, &TestServerOpts{
		License:                &fleet.LicenseInfo{Tier: fleet.TierPremium},
		SoftwareTitleIconStore: mockIconStore,
	})
	defer func() {
		_, err := mockIconStore.Cleanup(ctx, []string{}, time.Now().Add(time.Hour))
		require.NoError(t, err)
	}()

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.GetSoftwareTitleIconFunc = func(ctx context.Context, teamID, titleID uint) (*fleet.SoftwareTitleIcon, error) {
		if titleID == 1 {
			return &fleet.SoftwareTitleIcon{
				TeamID:          teamID,
				SoftwareTitleID: titleID,
				StorageID:       "mock-storage-id",
				Filename:        "icon.png",
			}, nil
		}
		return nil, ctxerr.Wrap(ctx, &common_mysql.NotFoundError{ResourceType: "SoftwareTitleIcon"}, "get software title icon")
	}
	ds.GetVPPAppMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.VPPAppStoreApp, error) {
		if titleID == 3 {
			return &fleet.VPPAppStoreApp{
				IconURL: ptr.String("mock-vpp-icon-url"),
			}, nil
		}
		return nil, ctxerr.Wrap(ctx, &common_mysql.NotFoundError{ResourceType: "VPPAppMetadata"}, "get VPP app metadata")
	}

	testCases := []struct {
		name     string
		before   func()
		testFunc func(*testing.T)
	}{
		{
			name: "non-existing software title icon",
			before: func() {
			},
			testFunc: func(t *testing.T) {
				_, _, _, err := svc.GetSoftwareTitleIcon(ctx, 1, 2)
				require.Error(t, err)
			},
		},
		{
			name: "existing software title icon",
			before: func() {
				iconData := []byte("mock-icon-data")
				storageID := "mock-storage-id"
				err := mockIconStore.Put(ctx, storageID, bytes.NewReader(iconData))
				require.NoError(t, err)
			},
			testFunc: func(t *testing.T) {
				imageBytes, size, filename, err := svc.GetSoftwareTitleIcon(ctx, 1, 1)
				require.NoError(t, err)
				assert.Equal(t, "icon.png", filename)
				assert.Equal(t, int64(14), size)
				assert.Equal(t, "mock-icon-data", string(imageBytes))
			},
		},
		{
			name: "vpp icon fall back",
			before: func() {
			},
			testFunc: func(t *testing.T) {
				_, _, _, err := svc.GetSoftwareTitleIcon(ctx, 1, 3)
				require.Error(t, err)
				var vppIconErr *fleet.VPPIconAvailable
				require.True(t, errors.As(err, &vppIconErr), "error should be VPPIconAvailable")
				require.Equal(t, "mock-vpp-icon-url", vppIconErr.IconURL)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before()
			tc.testFunc(t)
		})
	}
}

func TestUploadSoftwareTitleIcon(t *testing.T) {
	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: user})
	ds := new(mock.Store)

	mockIconStore := s3.SetupTestSoftwareTitleIconStore(t, "software-title-icons-unit-test", "icon-store-prefix")
	svc, _ := newTestService(t, ds, nil, nil, &TestServerOpts{
		License:                &fleet.LicenseInfo{Tier: fleet.TierPremium},
		SoftwareTitleIconStore: mockIconStore,
	})
	defer func() {
		_, err := mockIconStore.Cleanup(ctx, []string{}, time.Now().Add(time.Hour))
		require.NoError(t, err)
	}()

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	var capturedActivities []fleet.ActivityDetails
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, detailsBytes []byte, timestamp time.Time) error {
		capturedActivities = append(capturedActivities, activity)
		return nil
	}

	var iconFile *fleet.TempFileReader
	testCases := []struct {
		name     string
		before   func()
		testFunc func(*testing.T)
	}{
		{
			name: "upload icon title with no software installer, vpp app, or in-house app",
			before: func() {
				capturedActivities = make([]fleet.ActivityDetails, 0)
				ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, includeUnpublished bool) (*fleet.SoftwareInstaller, error) {
					return nil, ctxerr.Wrap(ctx, &common_mysql.NotFoundError{ResourceType: "SoftwareInstaller"}, "get software installer")
				}
				ds.GetVPPAppMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.VPPAppStoreApp, error) {
					return nil, ctxerr.Wrap(ctx, &common_mysql.NotFoundError{ResourceType: "VPPAppMetadata"}, "get VPP app metadata")
				}
				ds.GetInHouseAppMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.SoftwareInstaller, error) {
					return nil, ctxerr.Wrap(ctx, &common_mysql.NotFoundError{ResourceType: "InHouseAppMetadata"}, "get in-house app metadata")
				}

				file, err := os.Open("testdata/icons/valid-icon.png")
				require.NoError(t, err)
				defer file.Close()
				iconFile, err = fleet.NewTempFileReader(file, func() string { return t.TempDir() })
				require.NoError(t, err)
			},
			testFunc: func(t *testing.T) {
				payload := &fleet.UploadSoftwareTitleIconPayload{
					TitleID:  1,
					TeamID:   1,
					IconFile: iconFile,
					Filename: "icon.png",
				}
				_, err := svc.UploadSoftwareTitleIcon(ctx, payload)
				require.Error(t, err)
				require.Contains(t, err.Error(), "Software title has no software installer, VPP app, or in-house app")
			},
		},
		{
			name: "upload icon for software installer",
			before: func() {
				capturedActivities = make([]fleet.ActivityDetails, 0)
				ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, includeUnpublished bool) (*fleet.SoftwareInstaller, error) {
					return &fleet.SoftwareInstaller{
						TitleID: ptr.Uint(1),
						TeamID:  ptr.Uint(1),
					}, nil
				}
				ds.GetSoftwareTitleIconFunc = func(ctx context.Context, teamID, titleID uint) (*fleet.SoftwareTitleIcon, error) {
					return nil, ctxerr.Wrap(ctx, &common_mysql.NotFoundError{ResourceType: "SoftwareTitleIcon"}, "get software title icon")
				}
				ds.GetTeamIdsForIconStorageIdFunc = func(ctx context.Context, storageID string) ([]uint, error) {
					return []uint{1}, nil
				}
				ds.CreateOrUpdateSoftwareTitleIconFunc = func(ctx context.Context, payload *fleet.UploadSoftwareTitleIconPayload) (*fleet.SoftwareTitleIcon, error) {
					sha, err := file.SHA256FromTempFileReader(iconFile)
					require.NoError(t, err)

					return &fleet.SoftwareTitleIcon{
						TeamID:          1,
						SoftwareTitleID: 1,
						StorageID:       sha,
						Filename:        "icon.png",
					}, nil
				}
				ds.ActivityDetailsForSoftwareTitleIconFunc = func(ctx context.Context, teamID uint, titleID uint) (fleet.DetailsForSoftwareIconActivity, error) {
					return fleet.DetailsForSoftwareIconActivity{
						SoftwareInstallerID: ptr.Uint(1),
						SoftwareTitle:       "foo",
						Filename:            ptr.String("icon.png"),
						TeamName:            ptr.String("team1"),
						TeamID:              1,
						SoftwareTitleID:     1,
					}, nil
				}
			},
			testFunc: func(t *testing.T) {
				payload := &fleet.UploadSoftwareTitleIconPayload{
					TitleID:  1,
					TeamID:   1,
					IconFile: iconFile,
					Filename: "icon.png",
				}
				_, err := svc.UploadSoftwareTitleIcon(ctx, payload)
				require.NoError(t, err)
				require.Len(t, capturedActivities, 1)
			},
		},
		{
			name: "upload icon for vpp app",
			before: func() {
				capturedActivities = make([]fleet.ActivityDetails, 0)
				ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, includeUnpublished bool) (*fleet.SoftwareInstaller, error) {
					return nil, ctxerr.Wrap(ctx, &common_mysql.NotFoundError{ResourceType: "SoftwareInstaller"}, "get software installer")
				}
				ds.GetVPPAppMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.VPPAppStoreApp, error) {
					return &fleet.VPPAppStoreApp{
						VPPAppID:       fleet.VPPAppID{AdamID: "1"},
						VPPAppsTeamsID: 1,
					}, nil
				}
				ds.GetSoftwareTitleIconFunc = func(ctx context.Context, teamID, titleID uint) (*fleet.SoftwareTitleIcon, error) {
					return nil, ctxerr.Wrap(ctx, &common_mysql.NotFoundError{ResourceType: "SoftwareTitleIcon"}, "get software title icon")
				}
				ds.GetTeamIdsForIconStorageIdFunc = func(ctx context.Context, storageID string) ([]uint, error) {
					return []uint{1}, nil
				}
				ds.CreateOrUpdateSoftwareTitleIconFunc = func(ctx context.Context, payload *fleet.UploadSoftwareTitleIconPayload) (*fleet.SoftwareTitleIcon, error) {
					sha, err := file.SHA256FromTempFileReader(iconFile)
					require.NoError(t, err)

					return &fleet.SoftwareTitleIcon{
						TeamID:          1,
						SoftwareTitleID: 1,
						StorageID:       sha,
						Filename:        "icon.png",
					}, nil
				}
				ds.ActivityDetailsForSoftwareTitleIconFunc = func(ctx context.Context, teamID uint, titleID uint) (fleet.DetailsForSoftwareIconActivity, error) {
					return fleet.DetailsForSoftwareIconActivity{
						SoftwareInstallerID: ptr.Uint(1),
						SoftwareTitle:       "foo",
						Filename:            ptr.String("icon.png"),
						TeamName:            ptr.String("team1"),
						TeamID:              1,
						SoftwareTitleID:     1,
					}, nil
				}
			},
			testFunc: func(t *testing.T) {
				payload := &fleet.UploadSoftwareTitleIconPayload{
					TitleID:  1,
					TeamID:   1,
					IconFile: iconFile,
					Filename: "icon.png",
				}
				_, err := svc.UploadSoftwareTitleIcon(ctx, payload)
				require.NoError(t, err)
				require.Len(t, capturedActivities, 1)
			},
		},

		{
			name: "upload icon for in-house app",
			before: func() {
				capturedActivities = make([]fleet.ActivityDetails, 0)
				ds.GetInHouseAppMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.SoftwareInstaller, error) {
					return &fleet.SoftwareInstaller{
						TitleID: ptr.Uint(1),
						TeamID:  ptr.Uint(1),
					}, nil
				}
				ds.GetSoftwareTitleIconFunc = func(ctx context.Context, teamID, titleID uint) (*fleet.SoftwareTitleIcon, error) {
					return nil, ctxerr.Wrap(ctx, &common_mysql.NotFoundError{ResourceType: "SoftwareTitleIcon"}, "get software title icon")
				}
				ds.GetTeamIdsForIconStorageIdFunc = func(ctx context.Context, storageID string) ([]uint, error) {
					return []uint{1}, nil
				}
				ds.CreateOrUpdateSoftwareTitleIconFunc = func(ctx context.Context, payload *fleet.UploadSoftwareTitleIconPayload) (*fleet.SoftwareTitleIcon, error) {
					sha, err := file.SHA256FromTempFileReader(iconFile)
					require.NoError(t, err)

					return &fleet.SoftwareTitleIcon{
						TeamID:          1,
						SoftwareTitleID: 1,
						StorageID:       sha,
						Filename:        "icon.png",
					}, nil
				}
				ds.ActivityDetailsForSoftwareTitleIconFunc = func(ctx context.Context, teamID uint, titleID uint) (fleet.DetailsForSoftwareIconActivity, error) {
					return fleet.DetailsForSoftwareIconActivity{
						InHouseAppID:    ptr.Uint(1),
						SoftwareTitle:   "foo",
						Filename:        ptr.String("icon.png"),
						TeamName:        ptr.String("team1"),
						TeamID:          1,
						SoftwareTitleID: 1,
					}, nil
				}
			},
			testFunc: func(t *testing.T) {
				payload := &fleet.UploadSoftwareTitleIconPayload{
					TitleID:  1,
					TeamID:   1,
					IconFile: iconFile,
					Filename: "icon.png",
				}
				_, err := svc.UploadSoftwareTitleIcon(ctx, payload)
				require.NoError(t, err)
				require.Len(t, capturedActivities, 1)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before()
			tc.testFunc(t)
		})
	}
}

func TestDeleteSoftwareTitleIcon(t *testing.T) {
	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: user})
	ds := new(mock.Store)
	svc, _ := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}})

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	var capturedActivities []fleet.ActivityDetails
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, detailsBytes []byte, timestamp time.Time) error {
		capturedActivities = append(capturedActivities, activity)
		return nil
	}

	testCases := []struct {
		name     string
		before   func()
		testFunc func(*testing.T)
	}{
		{
			"Delete existing icon for software installer",
			func() {
				capturedActivities = make([]fleet.ActivityDetails, 0)
				ds.ActivityDetailsForSoftwareTitleIconFunc = func(ctx context.Context, teamID uint, titleID uint) (fleet.DetailsForSoftwareIconActivity, error) {
					return fleet.DetailsForSoftwareIconActivity{
						SoftwareInstallerID: ptr.Uint(1),
						AdamID:              nil,
						VPPAppTeamID:        nil,
						VPPIconUrl:          nil,
						SoftwareTitle:       "foo",
						Filename:            ptr.String("foo.pkg"),
						TeamName:            ptr.String("team1"),
						TeamID:              1,
						SelfService:         false,
						SoftwareTitleID:     1,
						Platform:            nil,
						LabelsIncludeAny:    nil,
						LabelsExcludeAny:    nil,
					}, nil
				}
				ds.DeleteSoftwareTitleIconFunc = func(ctx context.Context, teamID uint, titleID uint) error {
					return nil
				}
			},
			func(t *testing.T) {
				err := svc.DeleteSoftwareTitleIcon(ctx, 1, 1)
				require.NoError(t, err)

				require.Len(t, capturedActivities, 1)
				capturedActivity := capturedActivities[0]

				expectedActivity := fleet.ActivityTypeEditedSoftware{
					SoftwareTitle:    "foo",
					SoftwarePackage:  ptr.String("foo.pkg"),
					TeamName:         ptr.String("team1"),
					TeamID:           ptr.Uint(1),
					SelfService:      false,
					SoftwareIconURL:  ptr.String(""),
					LabelsIncludeAny: nil,
					LabelsExcludeAny: nil,
					SoftwareTitleID:  1,
				}
				require.Equal(t, expectedActivity, capturedActivity)
			},
		},
		{
			"Delete existing icon for vpp app",
			func() {
				capturedActivities = make([]fleet.ActivityDetails, 0)
				ds.ActivityDetailsForSoftwareTitleIconFunc = func(ctx context.Context, teamID uint, titleID uint) (fleet.DetailsForSoftwareIconActivity, error) {
					platform := fleet.MacOSPlatform
					return fleet.DetailsForSoftwareIconActivity{
						SoftwareInstallerID: nil,
						AdamID:              ptr.String("1"),
						VPPAppTeamID:        ptr.Uint(1),
						VPPIconUrl:          ptr.String("fleetdm.com/icon.png"),
						SoftwareTitle:       "foo",
						Filename:            nil,
						TeamName:            ptr.String("team1"),
						TeamID:              1,
						SelfService:         false,
						SoftwareTitleID:     1,
						Platform:            &platform,
						LabelsIncludeAny:    nil,
						LabelsExcludeAny:    nil,
					}, nil
				}
				ds.DeleteSoftwareTitleIconFunc = func(ctx context.Context, teamID uint, titleID uint) error {
					return nil
				}
			},
			func(t *testing.T) {
				err := svc.DeleteSoftwareTitleIcon(ctx, 1, 1)
				require.NoError(t, err)

				require.Len(t, capturedActivities, 1)
				capturedActivity := capturedActivities[0]

				expectedActivity := fleet.ActivityEditedAppStoreApp{
					SoftwareTitle:    "foo",
					SoftwareTitleID:  1,
					AppStoreID:       "1",
					TeamName:         ptr.String("team1"),
					TeamID:           ptr.Uint(1),
					Platform:         fleet.MacOSPlatform,
					SelfService:      false,
					SoftwareIconURL:  ptr.String("fleetdm.com/icon.png"), // note this is supposed to be the vpp_apps.icon_url
					LabelsIncludeAny: nil,
					LabelsExcludeAny: nil,
				}
				require.Equal(t, expectedActivity, capturedActivity)
			},
		},
		{
			"Delete an already deleted icon",
			func() {
				capturedActivities = make([]fleet.ActivityDetails, 0)
				ds.ActivityDetailsForSoftwareTitleIconFunc = func(ctx context.Context, teamID uint, titleID uint) (fleet.DetailsForSoftwareIconActivity, error) {
					return fleet.DetailsForSoftwareIconActivity{}, nil
				}
				ds.DeleteSoftwareTitleIconFunc = func(ctx context.Context, teamID uint, titleID uint) error {
					return ctxerr.Wrap(ctx, &common_mysql.NotFoundError{ResourceType: "SoftwareTitleIcon"}, "software title icon not found")
				}
			},
			func(t *testing.T) {
				err := svc.DeleteSoftwareTitleIcon(ctx, 1, 1)
				require.NoError(t, err)
				require.Len(t, capturedActivities, 0)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before()
			tc.testFunc(t)
		})
	}
}

func TestIconChanges(t *testing.T) {
	// Test adding new icon for software.
	t.Run("Add new icon for software", func(t *testing.T) {
		// Create a new empty IconChanges struct.
		c := &fleet.IconChanges{
			TeamID:         1,
			UploadedHashes: []string{"local-new-to-me-sw-icon-hash", "sw-noop-icon-hash"},
		}

		// Create SoftwarePackageResponse and VPPAppResponse with local icon data only.
		sp := []fleet.SoftwarePackageResponse{
			{
				TitleID:       ptr.Uint(1),
				TeamID:        ptr.Uint(1),
				LocalIconHash: "local-new-sw-icon-hash",
				LocalIconPath: "new-sw-icon.png",
			},
			{
				TitleID:       ptr.Uint(2),
				TeamID:        ptr.Uint(1),
				LocalIconHash: "local-new-to-me-sw-icon-hash",
				LocalIconPath: "new-to-me-sw-icon.png",
			},
			{
				TitleID:       ptr.Uint(3),
				TeamID:        ptr.Uint(1),
				IconHash:      "local-updated-filename-sw-icon-hash",
				IconFilename:  "path/to/local/outdated-filename-sw-icon.png",
				LocalIconHash: "local-updated-filename-sw-icon-hash",
				LocalIconPath: "updated-filename-sw-icon.png",
			},
			{
				TitleID:       ptr.Uint(4),
				TeamID:        ptr.Uint(1),
				IconHash:      "local-outdated-sw-icon-hash",
				IconFilename:  "path/to/local/updated-hash-sw-icon.png",
				LocalIconHash: "local-updated-hash-sw-icon-hash",
				LocalIconPath: "updated-hash-sw-icon.png",
			},
			{
				TitleID:      ptr.Uint(5),
				TeamID:       ptr.Uint(1),
				IconHash:     "sw-icon-to-delete-hash",
				IconFilename: "sw-icon-to-delete.png",
			},
			{
				TitleID:       ptr.Uint(6),
				TeamID:        ptr.Uint(1),
				LocalIconHash: "sw-noop-icon-hash",
				LocalIconPath: "path/to/local/sw-noop-icon.png",
				IconHash:      "sw-noop-icon-hash",
				IconFilename:  "sw-noop-icon.png",
			},
		}

		vpp := []fleet.VPPAppResponse{
			{
				TitleID:       ptr.Uint(7),
				TeamID:        ptr.Uint(1),
				LocalIconHash: "local-new-sw-icon-hash",
				LocalIconPath: "new-sw-icon.png",
			},
			{
				TitleID:       ptr.Uint(8),
				TeamID:        ptr.Uint(1),
				LocalIconHash: "local-new-to-me-sw-icon-hash",
				LocalIconPath: "new-to-me-sw-icon.png",
			},
			{
				TitleID:       ptr.Uint(9),
				TeamID:        ptr.Uint(1),
				IconHash:      "local-updated-filename-sw-icon-hash",
				IconFilename:  "path/to/local/outdated-filename-sw-icon.png",
				LocalIconHash: "local-updated-filename-sw-icon-hash",
				LocalIconPath: "updated-filename-sw-icon.png",
			},
			{
				TitleID:       ptr.Uint(10),
				TeamID:        ptr.Uint(1),
				IconHash:      "local-outdated-sw-icon-hash",
				IconFilename:  "path/to/local/updated-hash-sw-icon.png",
				LocalIconHash: "local-updated-hash-sw-icon-hash",
				LocalIconPath: "updated-hash-sw-icon.png",
			},
			{
				TitleID:      ptr.Uint(11),
				TeamID:       ptr.Uint(1),
				IconHash:     "sw-icon-to-delete-hash",
				IconFilename: "sw-icon-to-delete.png",
			},
			{
				TitleID:       ptr.Uint(12),
				TeamID:        ptr.Uint(1),
				LocalIconHash: "sw-noop-icon-hash",
				LocalIconPath: "path/to/local/sw-noop-icon.png",
				IconHash:      "sw-noop-icon-hash",
				IconFilename:  "sw-noop-icon.png",
			},
		}
		// Call the method to process the responses.
		updatedC := c.WithSoftware(sp, vpp)

		// Every hash that was already present on a software item, or would be after uploading,
		// should be represented in the UploadedHashes slice.
		require.Equal(t, 7, len(updatedC.UploadedHashes))
		require.Contains(t, updatedC.UploadedHashes, "local-new-sw-icon-hash")
		require.Contains(t, updatedC.UploadedHashes, "local-new-to-me-sw-icon-hash")
		require.Contains(t, updatedC.UploadedHashes, "local-updated-filename-sw-icon-hash")
		require.Contains(t, updatedC.UploadedHashes, "local-outdated-sw-icon-hash")
		require.Contains(t, updatedC.UploadedHashes, "local-updated-hash-sw-icon-hash")
		require.Contains(t, updatedC.UploadedHashes, "sw-icon-to-delete-hash")
		require.Contains(t, updatedC.UploadedHashes, "sw-noop-icon-hash")

		// IconsToUpload should contain info about any net-new software title icons.
		// Note that the icon for title #2 (local-new-to-me-sw-icon-hash) is new to the title,
		// but is already present in our list of uploaded hashes, so it should not be included here.
		require.Equal(t, 2, len(updatedC.IconsToUpload))
		require.Contains(t, updatedC.IconsToUpload, fleet.IconFileUpdate{TitleID: 1, Path: "new-sw-icon.png"})
		require.Contains(t, updatedC.IconsToUpload, fleet.IconFileUpdate{TitleID: 4, Path: "updated-hash-sw-icon.png"})

		// IconsToUpdate should contain info about any software title icons that need updates to their filename,
		// or titles where we're adding an icon for them, but the icon already exists in our uploaded hashes.
		// Note that for the VPP apps, we will already have marked "new-sw-icon" and "updated-hash-sw-icon" for upload,
		// so they will show up in IconsToUpdate rather than IconsToUpload.
		require.Equal(t, 6, len(updatedC.IconsToUpdate))
		require.Contains(t, updatedC.IconsToUpdate, fleet.IconMetaUpdate{TitleID: 2, Path: "new-to-me-sw-icon.png", Hash: "local-new-to-me-sw-icon-hash"})
		require.Contains(t, updatedC.IconsToUpdate, fleet.IconMetaUpdate{TitleID: 3, Path: "updated-filename-sw-icon.png", Hash: "local-updated-filename-sw-icon-hash"})
		require.Contains(t, updatedC.IconsToUpdate, fleet.IconMetaUpdate{TitleID: 7, Path: "new-sw-icon.png", Hash: "local-new-sw-icon-hash"})
		require.Contains(t, updatedC.IconsToUpdate, fleet.IconMetaUpdate{TitleID: 8, Path: "new-to-me-sw-icon.png", Hash: "local-new-to-me-sw-icon-hash"})
		require.Contains(t, updatedC.IconsToUpdate, fleet.IconMetaUpdate{TitleID: 9, Path: "updated-filename-sw-icon.png", Hash: "local-updated-filename-sw-icon-hash"})
		require.Contains(t, updatedC.IconsToUpdate, fleet.IconMetaUpdate{TitleID: 10, Path: "updated-hash-sw-icon.png", Hash: "local-updated-hash-sw-icon-hash"})

		// IconsToDelete should contain info about any software title icons that exist on the server,
		// but were not included in the uploaded hashes, meaning they should be deleted.
		require.Equal(t, 2, len(updatedC.TitleIDsToRemoveIconsFrom))
		require.Contains(t, updatedC.TitleIDsToRemoveIconsFrom, uint(5))
		require.Contains(t, updatedC.TitleIDsToRemoveIconsFrom, uint(11))
	})
}
