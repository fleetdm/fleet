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
			name: "upload icon title with no software installer or vpp app",
			before: func() {
				capturedActivities = make([]fleet.ActivityDetails, 0)
				ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, includeUnpublished bool) (*fleet.SoftwareInstaller, error) {
					return nil, ctxerr.Wrap(ctx, &common_mysql.NotFoundError{ResourceType: "SoftwareInstaller"}, "get software installer")
				}
				ds.GetVPPAppMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.VPPAppStoreApp, error) {
					return nil, ctxerr.Wrap(ctx, &common_mysql.NotFoundError{ResourceType: "VPPAppMetadata"}, "get VPP app metadata")
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
				require.Contains(t, err.Error(), "Software title has no software installer or VPP app")
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
