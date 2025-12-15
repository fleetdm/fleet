package service

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateNDESSCEPAdminURL(t *testing.T) {
	t.Parallel()

	var returnPage func() []byte
	returnStatus := http.StatusOK
	wait := false
	ndesAdminServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wait {
			time.Sleep(1 * time.Second)
		}
		w.WriteHeader(returnStatus)
		if returnStatus == http.StatusOK {
			_, err := w.Write(returnPage())
			require.NoError(t, err)
		}
	}))
	t.Cleanup(ndesAdminServer.Close)

	proxy := fleet.NDESSCEPProxyCA{
		AdminURL: ndesAdminServer.URL,
		Username: "admin",
		Password: "password",
	}

	returnStatus = http.StatusNotFound
	logger := kitlog.NewNopLogger()
	svc := NewSCEPConfigService(logger, nil)
	err := svc.ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorContains(t, err, "unexpected status code")
	returnStatus = http.StatusOK

	// Catch timeout issue
	svc = NewSCEPConfigService(logger, ptr.Duration(1*time.Microsecond))
	wait = true
	err = svc.ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	wait = false
	svc = NewSCEPConfigService(logger, nil)

	// We need to convert the HTML page to UTF-16 encoding, which is used by Windows servers
	returnPageFromFile := func(path string) []byte {
		dat, err := os.ReadFile(path)
		require.NoError(t, err)
		datUTF16, err := utf16FromString(string(dat))
		require.NoError(t, err)
		byteData := make([]byte, len(datUTF16)*2)
		for i, v := range datUTF16 {
			binary.LittleEndian.PutUint16(byteData[i*2:], v)
		}
		return byteData
	}

	// Catch ths issue when NDES password cache is full
	returnPage = func() []byte {
		return returnPageFromFile("./testdata/mscep_admin_cache_full.html")
	}
	err = svc.ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorContains(t, err, "the password cache is full")

	// Catch ths issue when account has insufficient permissions
	returnPage = func() []byte {
		return returnPageFromFile("./testdata/mscep_admin_insufficient_permissions.html")
	}
	err = svc.ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorContains(t, err, "does not have sufficient permissions")

	// Nothing returned
	returnPage = func() []byte {
		return []byte{}
	}
	err = svc.ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.ErrorContains(t, err, "could not retrieve the enrollment challenge password")

	// All good
	returnPage = func() []byte {
		return returnPageFromFile("./testdata/mscep_admin_password.html")
	}
	err = svc.ValidateNDESSCEPAdminURL(context.Background(), proxy)
	assert.NoError(t, err)
}

func TestValidateSCEPURL(t *testing.T) {
	t.Parallel()
	srv := NewTestSCEPServer(t)

	proxy := fleet.NDESSCEPProxyCA{
		URL: srv.URL + "/scep",
	}
	logger := kitlog.NewNopLogger()
	svc := NewSCEPConfigService(logger, nil)
	err := svc.ValidateSCEPURL(context.Background(), proxy.URL)
	assert.NoError(t, err)

	proxy.URL = srv.URL + "/bozo"
	err = svc.ValidateSCEPURL(context.Background(), proxy.URL)
	assert.ErrorContains(t, err, "could not retrieve CA certificate")
}

func TestValidateIdentifier(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := kitlog.NewNopLogger()

	// Helper to create a scepProxyService with a mock datastore
	newTestService := func(ds *mock.DataStore) *scepProxyService {
		return &scepProxyService{
			ds:          ds,
			debugLogger: logger,
			Timeout:     ptr.Duration(30 * time.Second),
		}
	}

	// Helper to create a valid identifier
	makeIdentifier := func(hostUUID, profileUUID, caName, challenge string) string {
		id := hostUUID + "," + profileUUID
		if caName != "" {
			id += "," + caName
		}
		if challenge != "" {
			id += "," + challenge
		}
		return url.PathEscape(id)
	}

	t.Run("identifier parsing errors", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{}, nil
		}
		svc := newTestService(ds)

		testCases := []struct {
			name       string
			identifier string
			errMsg     string
		}{
			{
				name:       "empty identifier",
				identifier: "",
				errMsg:     "invalid identifier in URL path",
			},
			{
				name:       "single element",
				identifier: "host-uuid-only",
				errMsg:     "invalid identifier in URL path",
			},
			{
				name:       "empty host UUID",
				identifier: makeIdentifier("", "a-profile-uuid", "", ""),
				errMsg:     "invalid identifier in URL path",
			},
			{
				name:       "empty profile UUID",
				identifier: makeIdentifier("host-uuid", "", "", ""),
				errMsg:     "invalid identifier in URL path",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := svc.validateIdentifier(ctx, tc.identifier, false)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
			})
		}
	})

	t.Run("invalid profile UUID prefix", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{}, nil
		}
		svc := newTestService(ds)

		// Profile UUID must start with "a" (Apple) or "w" (Windows)
		identifier := makeIdentifier("host-uuid", "invalid-profile-uuid", "NDES", "")
		_, err := svc.validateIdentifier(ctx, identifier, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid profile UUID")
	})

	t.Run("profile not found", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{}, nil
		}
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return nil, nil // Profile not found
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "NDES", "")
		_, err := svc.validateIdentifier(ctx, identifier, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown identifier in URL path")
	})

	t.Run("profile status not pending for Apple profile", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				NDESSCEP: &fleet.NDESSCEPProxyCA{URL: "https://ndes.example.com/scep"},
			}, nil
		}
		verifiedStatus := fleet.MDMDeliveryVerified
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:    hostUUID,
				ProfileUUID: profileUUID,
				Status:      &verifiedStatus,
				Type:        fleet.CAConfigNDES,
				CAName:      caName,
			}, nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "NDES", "")
		_, err := svc.validateIdentifier(ctx, identifier, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "profile status (verified) is not 'pending'")
	})

	t.Run("Windows profile skips status check", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				CustomScepProxy: []fleet.CustomSCEPProxyCA{
					{Name: "test-ca", URL: "https://scep.example.com/scep"},
				},
			}, nil
		}
		verifiedStatus := fleet.MDMDeliveryVerified
		ds.GetWindowsHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:    hostUUID,
				ProfileUUID: profileUUID,
				Status:      &verifiedStatus, // Not pending, but should pass for Windows
				Type:        fleet.CAConfigCustomSCEPProxy,
				CAName:      "test-ca",
			}, nil
		}
		svc := newTestService(ds)

		// Windows profiles skip status check
		identifier := makeIdentifier("host-uuid", "w-profile-uuid", "test-ca", "")
		scepURL, err := svc.validateIdentifier(ctx, identifier, false)
		require.NoError(t, err)
		assert.Equal(t, "https://scep.example.com/scep", scepURL)
	})

	t.Run("NDES CA not configured", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				NDESSCEP: nil, // NDES not configured
			}, nil
		}
		pendingStatus := fleet.MDMDeliveryPending
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:    hostUUID,
				ProfileUUID: profileUUID,
				Status:      &pendingStatus,
				Type:        fleet.CAConfigNDES,
				CAName:      "NDES",
			}, nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "NDES", "")
		_, err := svc.validateIdentifier(ctx, identifier, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), MessageSCEPProxyNotConfigured)
	})

	t.Run("NDES valid request without challenge check", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				NDESSCEP: &fleet.NDESSCEPProxyCA{URL: "https://ndes.example.com/scep"},
			}, nil
		}
		pendingStatus := fleet.MDMDeliveryPending
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:    hostUUID,
				ProfileUUID: profileUUID,
				Status:      &pendingStatus,
				Type:        fleet.CAConfigNDES,
				CAName:      "NDES",
			}, nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "NDES", "")
		scepURL, err := svc.validateIdentifier(ctx, identifier, false)
		require.NoError(t, err)
		assert.Equal(t, "https://ndes.example.com/scep", scepURL)
	})

	t.Run("NDES challenge expired triggers requeue", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				NDESSCEP: &fleet.NDESSCEPProxyCA{URL: "https://ndes.example.com/scep"},
			}, nil
		}
		pendingStatus := fleet.MDMDeliveryPending
		expiredTime := time.Now().Add(-58 * time.Minute) // Expired (>57 minutes)
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:             hostUUID,
				ProfileUUID:          profileUUID,
				Status:               &pendingStatus,
				Type:                 fleet.CAConfigNDES,
				CAName:               "NDES",
				ChallengeRetrievedAt: &expiredTime,
			}, nil
		}
		ds.ResendHostMDMProfileFunc = func(ctx context.Context, hostUUID, profileUUID string) error {
			assert.Equal(t, "host-uuid", hostUUID)
			assert.Equal(t, "a-profile-uuid", profileUUID)
			return nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "NDES", "")
		_, err := svc.validateIdentifier(ctx, identifier, true) // checkChallenge=true
		require.Error(t, err)
		assert.Contains(t, err.Error(), "challenge password has expired")
		assert.True(t, ds.ResendHostMDMProfileFuncInvoked)
		ds.ResendHostMDMProfileFuncInvoked = false
	})

	t.Run("NDES challenge not expired", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				NDESSCEP: &fleet.NDESSCEPProxyCA{URL: "https://ndes.example.com/scep"},
			}, nil
		}
		pendingStatus := fleet.MDMDeliveryPending
		recentTime := time.Now().Add(-30 * time.Minute) // Not expired (<57 minutes)
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:             hostUUID,
				ProfileUUID:          profileUUID,
				Status:               &pendingStatus,
				Type:                 fleet.CAConfigNDES,
				CAName:               "NDES",
				ChallengeRetrievedAt: &recentTime,
			}, nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "NDES", "")
		scepURL, err := svc.validateIdentifier(ctx, identifier, true)
		require.NoError(t, err)
		assert.Equal(t, "https://ndes.example.com/scep", scepURL)
	})

	t.Run("Smallstep CA not configured", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				Smallstep: []fleet.SmallstepSCEPProxyCA{}, // Empty
			}, nil
		}
		pendingStatus := fleet.MDMDeliveryPending
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:    hostUUID,
				ProfileUUID: profileUUID,
				Status:      &pendingStatus,
				Type:        fleet.CAConfigSmallstep,
				CAName:      "my-smallstep",
			}, nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "my-smallstep", "")
		_, err := svc.validateIdentifier(ctx, identifier, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), MessageSCEPProxyNotConfigured)
	})

	t.Run("Smallstep valid request", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				Smallstep: []fleet.SmallstepSCEPProxyCA{
					{Name: "my-smallstep", URL: "https://smallstep.example.com/scep"},
				},
			}, nil
		}
		pendingStatus := fleet.MDMDeliveryPending
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:    hostUUID,
				ProfileUUID: profileUUID,
				Status:      &pendingStatus,
				Type:        fleet.CAConfigSmallstep,
				CAName:      "my-smallstep",
			}, nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "my-smallstep", "")
		scepURL, err := svc.validateIdentifier(ctx, identifier, false)
		require.NoError(t, err)
		assert.Equal(t, "https://smallstep.example.com/scep", scepURL)
	})

	t.Run("Smallstep challenge expired triggers requeue", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				Smallstep: []fleet.SmallstepSCEPProxyCA{
					{Name: "my-smallstep", URL: "https://smallstep.example.com/scep"},
				},
			}, nil
		}
		pendingStatus := fleet.MDMDeliveryPending
		expiredTime := time.Now().Add(-5 * time.Minute) // Expired (>4 minutes)
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:             hostUUID,
				ProfileUUID:          profileUUID,
				Status:               &pendingStatus,
				Type:                 fleet.CAConfigSmallstep,
				CAName:               "my-smallstep",
				ChallengeRetrievedAt: &expiredTime,
			}, nil
		}
		ds.ResendHostCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID string) error {
			assert.Equal(t, "host-uuid", hostUUID)
			assert.Equal(t, "a-profile-uuid", profileUUID)
			return nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "my-smallstep", "")
		_, err := svc.validateIdentifier(ctx, identifier, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "challenge password has expired")
		assert.True(t, ds.ResendHostCertificateProfileFuncInvoked)
		ds.ResendHostCertificateProfileFuncInvoked = false
	})

	t.Run("Smallstep challenge not expired", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				Smallstep: []fleet.SmallstepSCEPProxyCA{
					{Name: "my-smallstep", URL: "https://smallstep.example.com/scep"},
				},
			}, nil
		}
		pendingStatus := fleet.MDMDeliveryPending
		recentTime := time.Now().Add(-2 * time.Minute) // Not expired (<4 minutes)
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:             hostUUID,
				ProfileUUID:          profileUUID,
				Status:               &pendingStatus,
				Type:                 fleet.CAConfigSmallstep,
				CAName:               "my-smallstep",
				ChallengeRetrievedAt: &recentTime,
			}, nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "my-smallstep", "")
		scepURL, err := svc.validateIdentifier(ctx, identifier, true)
		require.NoError(t, err)
		assert.Equal(t, "https://smallstep.example.com/scep", scepURL)
	})

	t.Run("Custom SCEP CA not configured", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				CustomScepProxy: []fleet.CustomSCEPProxyCA{}, // Empty
			}, nil
		}
		pendingStatus := fleet.MDMDeliveryPending
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:    hostUUID,
				ProfileUUID: profileUUID,
				Status:      &pendingStatus,
				Type:        fleet.CAConfigCustomSCEPProxy,
				CAName:      "my-custom-ca",
			}, nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "my-custom-ca", "test-challenge")
		_, err := svc.validateIdentifier(ctx, identifier, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), MessageSCEPProxyNotConfigured)
	})

	t.Run("Custom SCEP valid request without challenge check", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				CustomScepProxy: []fleet.CustomSCEPProxyCA{
					{Name: "my-custom-ca", URL: "https://custom-scep.example.com/scep"},
				},
			}, nil
		}
		pendingStatus := fleet.MDMDeliveryPending
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:    hostUUID,
				ProfileUUID: profileUUID,
				Status:      &pendingStatus,
				Type:        fleet.CAConfigCustomSCEPProxy,
				CAName:      "my-custom-ca",
			}, nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "my-custom-ca", "test-challenge")
		scepURL, err := svc.validateIdentifier(ctx, identifier, false)
		require.NoError(t, err)
		assert.Equal(t, "https://custom-scep.example.com/scep", scepURL)
	})

	t.Run("Custom SCEP valid challenge consumption", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				CustomScepProxy: []fleet.CustomSCEPProxyCA{
					{Name: "my-custom-ca", URL: "https://custom-scep.example.com/scep"},
				},
			}, nil
		}
		pendingStatus := fleet.MDMDeliveryPending
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:    hostUUID,
				ProfileUUID: profileUUID,
				Status:      &pendingStatus,
				Type:        fleet.CAConfigCustomSCEPProxy,
				CAName:      "my-custom-ca",
			}, nil
		}
		ds.ConsumeChallengeFunc = func(ctx context.Context, challenge string) error {
			assert.Equal(t, "valid-challenge", challenge)
			return nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "my-custom-ca", "valid-challenge")
		scepURL, err := svc.validateIdentifier(ctx, identifier, true)
		require.NoError(t, err)
		assert.Equal(t, "https://custom-scep.example.com/scep", scepURL)
		assert.True(t, ds.ConsumeChallengeFuncInvoked)
		ds.ConsumeChallengeFuncInvoked = false
	})

	t.Run("Custom SCEP invalid challenge triggers requeue", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				CustomScepProxy: []fleet.CustomSCEPProxyCA{
					{Name: "my-custom-ca", URL: "https://custom-scep.example.com/scep"},
				},
			}, nil
		}
		pendingStatus := fleet.MDMDeliveryPending
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:    hostUUID,
				ProfileUUID: profileUUID,
				Status:      &pendingStatus,
				Type:        fleet.CAConfigCustomSCEPProxy,
				CAName:      "my-custom-ca",
			}, nil
		}
		ds.ConsumeChallengeFunc = func(ctx context.Context, challenge string) error {
			return sql.ErrNoRows // Challenge not found
		}
		ds.ResendHostCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID string) error {
			assert.Equal(t, "host-uuid", hostUUID)
			assert.Equal(t, "a-profile-uuid", profileUUID)
			return nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "my-custom-ca", "invalid-challenge")
		_, err := svc.validateIdentifier(ctx, identifier, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "custom scep challenge failed")
		assert.True(t, ds.ConsumeChallengeFuncInvoked)
		assert.True(t, ds.ResendHostCertificateProfileFuncInvoked)
		ds.ConsumeChallengeFuncInvoked = false
		ds.ResendHostCertificateProfileFuncInvoked = false
	})

	t.Run("Custom SCEP Windows profile skips challenge check", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				CustomScepProxy: []fleet.CustomSCEPProxyCA{
					{Name: "my-custom-ca", URL: "https://custom-scep.example.com/scep"},
				},
			}, nil
		}
		verifiedStatus := fleet.MDMDeliveryVerified
		ds.GetWindowsHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:    hostUUID,
				ProfileUUID: profileUUID,
				Status:      &verifiedStatus,
				Type:        fleet.CAConfigCustomSCEPProxy,
				CAName:      "my-custom-ca",
			}, nil
		}
		// ConsumeChallenge should NOT be called for Windows profiles
		ds.ConsumeChallengeFunc = func(ctx context.Context, challenge string) error {
			return nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "w-profile-uuid", "my-custom-ca", "test-challenge")
		scepURL, err := svc.validateIdentifier(ctx, identifier, true) // checkChallenge=true but should be skipped
		require.NoError(t, err)
		assert.Equal(t, "https://custom-scep.example.com/scep", scepURL)
		assert.False(t, ds.ConsumeChallengeFuncInvoked, "ConsumeChallenge should not be called for Windows profiles")
	})

	t.Run("datastore error getting CAs", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return nil, errors.New("database connection failed")
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "NDES", "")
		_, err := svc.validateIdentifier(ctx, identifier, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "getting grouped certificate authorities")
	})

	t.Run("datastore error getting profile", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{}, nil
		}
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return nil, errors.New("database query failed")
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "NDES", "")
		_, err := svc.validateIdentifier(ctx, identifier, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "getting host MDM profile")
	})

	t.Run("NDES resend error still returns challenge expired error", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				NDESSCEP: &fleet.NDESSCEPProxyCA{URL: "https://ndes.example.com/scep"},
			}, nil
		}
		pendingStatus := fleet.MDMDeliveryPending
		expiredTime := time.Now().Add(-58 * time.Minute)
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:             hostUUID,
				ProfileUUID:          profileUUID,
				Status:               &pendingStatus,
				Type:                 fleet.CAConfigNDES,
				CAName:               "NDES",
				ChallengeRetrievedAt: &expiredTime,
			}, nil
		}
		ds.ResendHostMDMProfileFunc = func(ctx context.Context, hostUUID, profileUUID string) error {
			return errors.New("resend failed")
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "NDES", "")
		_, err := svc.validateIdentifier(ctx, identifier, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resending host mdm profile")
	})

	t.Run("default CA name is NDES", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				NDESSCEP: &fleet.NDESSCEPProxyCA{URL: "https://ndes.example.com/scep"},
			}, nil
		}
		pendingStatus := fleet.MDMDeliveryPending
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			// Verify default CA name is "NDES"
			assert.Equal(t, "NDES", caName)
			return &fleet.HostMDMCertificateProfile{
				HostUUID:    hostUUID,
				ProfileUUID: profileUUID,
				Status:      &pendingStatus,
				Type:        fleet.CAConfigNDES,
				CAName:      "NDES",
			}, nil
		}
		svc := newTestService(ds)

		// Identifier with only host and profile UUID (no CA name)
		identifier := url.PathEscape("host-uuid,a-profile-uuid")
		scepURL, err := svc.validateIdentifier(ctx, identifier, false)
		require.NoError(t, err)
		assert.Equal(t, "https://ndes.example.com/scep", scepURL)
	})

	t.Run("Smallstep CA name mismatch returns not configured", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				Smallstep: []fleet.SmallstepSCEPProxyCA{
					{Name: "other-smallstep", URL: "https://other.example.com/scep"},
				},
			}, nil
		}
		pendingStatus := fleet.MDMDeliveryPending
		ds.GetAppleHostMDMCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID, caName string) (*fleet.HostMDMCertificateProfile, error) {
			return &fleet.HostMDMCertificateProfile{
				HostUUID:    hostUUID,
				ProfileUUID: profileUUID,
				Status:      &pendingStatus,
				Type:        fleet.CAConfigSmallstep,
				CAName:      "my-smallstep", // Different from configured
			}, nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "a-profile-uuid", "my-smallstep", "")
		_, err := svc.validateIdentifier(ctx, identifier, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), MessageSCEPProxyNotConfigured)
	})

	t.Run("Android request", func(t *testing.T) {
		ds := new(mock.DataStore)
		svc := newTestService(ds)

		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				CustomScepProxy: []fleet.CustomSCEPProxyCA{
					{Name: "android-ca", URL: "https://scep.example.com/scep"},
				},
			}, nil
		}

		deliveredStatus := fleet.CertificateTemplateDelivered
		ds.GetCertificateTemplateForHostFunc = func(ctx context.Context, hostUUID string, certificateTemplateID uint) (*fleet.CertificateTemplateForHost, error) {
			assert.Equal(t, "host-uuid", hostUUID)
			assert.Equal(t, uint(1), certificateTemplateID)
			return &fleet.CertificateTemplateForHost{
				HostUUID:              "host-uuid",
				CertificateTemplateID: 1,
				FleetChallenge:        ptr.String("test-challenge"),
				Status:                &deliveredStatus,
				CAType:                fleet.CAConfigCustomSCEPProxy,
				CAName:                "android-ca",
			}, nil
		}

		// Android identifier format: {hostUUID},g{certificateTemplateID},{caType},{challenge}
		identifier := makeIdentifier("host-uuid", "g1", "custom_scep_proxy", "test-challenge")
		scepURL, err := svc.validateIdentifier(ctx, identifier, false)
		require.NoError(t, err)
		assert.Equal(t, "https://scep.example.com/scep", scepURL)
		assert.True(t, ds.GetCertificateTemplateForHostFuncInvoked)
		ds.GetCertificateTemplateForHostFuncInvoked = false
	})

	t.Run("Android invalid certificate template ID", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{}, nil
		}
		svc := newTestService(ds)

		// Invalid certificate template ID (not a number)
		identifier := makeIdentifier("host-uuid", "ginvalid", "custom_scep_proxy", "test-challenge")
		_, err := svc.validateIdentifier(ctx, identifier, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid Android certificate template ID")
	})

	t.Run("Android certificate template not found", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{}, nil
		}
		ds.GetCertificateTemplateForHostFunc = func(ctx context.Context, hostUUID string, certificateTemplateID uint) (*fleet.CertificateTemplateForHost, error) {
			return nil, sql.ErrNoRows // Not found
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "g1", "custom_scep_proxy", "test-challenge")
		_, err := svc.validateIdentifier(ctx, identifier, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "getting Android certificate template")
	})

	t.Run("Android status not pending", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				CustomScepProxy: []fleet.CustomSCEPProxyCA{
					{Name: "android-ca", URL: "https://scep.example.com/scep"},
				},
			}, nil
		}
		verifiedStatus := fleet.CertificateTemplateVerified
		ds.GetCertificateTemplateForHostFunc = func(ctx context.Context, hostUUID string, certificateTemplateID uint) (*fleet.CertificateTemplateForHost, error) {
			return &fleet.CertificateTemplateForHost{
				HostUUID:              "host-uuid",
				CertificateTemplateID: 1,
				FleetChallenge:        ptr.String("test-challenge"),
				Status:                &verifiedStatus,
				CAType:                fleet.CAConfigCustomSCEPProxy,
				CAName:                "android-ca",
			}, nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "g1", "custom_scep_proxy", "test-challenge")
		_, err := svc.validateIdentifier(ctx, identifier, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "profile status (verified) is not 'pending'")
	})

	t.Run("Android CA not configured", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				CustomScepProxy: []fleet.CustomSCEPProxyCA{}, // Empty - no CAs configured
			}, nil
		}
		deliveredStatus := fleet.CertificateTemplateDelivered
		ds.GetCertificateTemplateForHostFunc = func(ctx context.Context, hostUUID string, certificateTemplateID uint) (*fleet.CertificateTemplateForHost, error) {
			return &fleet.CertificateTemplateForHost{
				HostUUID:              "host-uuid",
				CertificateTemplateID: 1,
				FleetChallenge:        ptr.String("test-challenge"),
				Status:                &deliveredStatus,
				CAType:                fleet.CAConfigCustomSCEPProxy,
				CAName:                "android-ca",
			}, nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "g1", "custom_scep_proxy", "test-challenge")
		_, err := svc.validateIdentifier(ctx, identifier, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), MessageSCEPProxyNotConfigured)
	})

	t.Run("Android CA name mismatch", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				CustomScepProxy: []fleet.CustomSCEPProxyCA{
					{Name: "other-ca", URL: "https://other.example.com/scep"}, // Different CA name
				},
			}, nil
		}
		deliveredStatus := fleet.CertificateTemplateDelivered
		ds.GetCertificateTemplateForHostFunc = func(ctx context.Context, hostUUID string, certificateTemplateID uint) (*fleet.CertificateTemplateForHost, error) {
			return &fleet.CertificateTemplateForHost{
				HostUUID:              "host-uuid",
				CertificateTemplateID: 1,
				FleetChallenge:        ptr.String("test-challenge"),
				Status:                &deliveredStatus,
				CAType:                fleet.CAConfigCustomSCEPProxy,
				CAName:                "android-ca", // This CA is not configured
			}, nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "g1", "custom_scep_proxy", "test-challenge")
		_, err := svc.validateIdentifier(ctx, identifier, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), MessageSCEPProxyNotConfigured)
	})

	t.Run("Android with challenge validation", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				CustomScepProxy: []fleet.CustomSCEPProxyCA{
					{Name: "android-ca", URL: "https://scep.example.com/scep"},
				},
			}, nil
		}
		deliveredStatus := fleet.CertificateTemplateDelivered
		ds.GetCertificateTemplateForHostFunc = func(ctx context.Context, hostUUID string, certificateTemplateID uint) (*fleet.CertificateTemplateForHost, error) {
			return &fleet.CertificateTemplateForHost{
				HostUUID:              "host-uuid",
				CertificateTemplateID: 1,
				FleetChallenge:        ptr.String("valid-challenge"),
				Status:                &deliveredStatus,
				CAType:                fleet.CAConfigCustomSCEPProxy,
				CAName:                "android-ca",
			}, nil
		}
		ds.ConsumeChallengeFunc = func(ctx context.Context, challenge string) error {
			assert.Equal(t, "valid-challenge", challenge)
			return nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "g1", "custom_scep_proxy", "valid-challenge")
		scepURL, err := svc.validateIdentifier(ctx, identifier, true) // checkChallenge=true
		require.NoError(t, err)
		assert.Equal(t, "https://scep.example.com/scep", scepURL)
		assert.True(t, ds.ConsumeChallengeFuncInvoked)
		ds.ConsumeChallengeFuncInvoked = false
	})

	t.Run("Android invalid challenge triggers requeue", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				CustomScepProxy: []fleet.CustomSCEPProxyCA{
					{Name: "android-ca", URL: "https://scep.example.com/scep"},
				},
			}, nil
		}
		deliveredStatus := fleet.CertificateTemplateDelivered
		ds.GetCertificateTemplateForHostFunc = func(ctx context.Context, hostUUID string, certificateTemplateID uint) (*fleet.CertificateTemplateForHost, error) {
			return &fleet.CertificateTemplateForHost{
				HostUUID:              "host-uuid",
				CertificateTemplateID: 1,
				FleetChallenge:        ptr.String("valid-challenge"),
				Status:                &deliveredStatus,
				CAType:                fleet.CAConfigCustomSCEPProxy,
				CAName:                "android-ca",
			}, nil
		}
		ds.ConsumeChallengeFunc = func(ctx context.Context, challenge string) error {
			return sql.ErrNoRows // Challenge not found/expired
		}
		ds.ResendHostCertificateProfileFunc = func(ctx context.Context, hostUUID, profileUUID string) error {
			assert.Equal(t, "host-uuid", hostUUID)
			assert.Equal(t, "g1", profileUUID)
			return nil
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "g1", "custom_scep_proxy", "invalid-challenge")
		_, err := svc.validateIdentifier(ctx, identifier, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "custom scep challenge failed")
		assert.True(t, ds.ConsumeChallengeFuncInvoked)
		assert.True(t, ds.ResendHostCertificateProfileFuncInvoked)
		ds.ConsumeChallengeFuncInvoked = false
		ds.ResendHostCertificateProfileFuncInvoked = false
	})

	t.Run("Android datastore error getting templates", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{}, nil
		}
		ds.GetCertificateTemplateForHostFunc = func(ctx context.Context, hostUUID string, certificateTemplateID uint) (*fleet.CertificateTemplateForHost, error) {
			return nil, errors.New("database connection failed")
		}
		svc := newTestService(ds)

		identifier := makeIdentifier("host-uuid", "g1", "custom_scep_proxy", "test-challenge")
		_, err := svc.validateIdentifier(ctx, identifier, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "getting Android certificate template")
	})

	t.Run("Android uses challenge from template when not in identifier", func(t *testing.T) {
		ds := new(mock.DataStore)
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			return &fleet.GroupedCertificateAuthorities{
				CustomScepProxy: []fleet.CustomSCEPProxyCA{
					{Name: "android-ca", URL: "https://scep.example.com/scep"},
				},
			}, nil
		}
		deliveredStatus := fleet.CertificateTemplateDelivered
		ds.GetCertificateTemplateForHostFunc = func(ctx context.Context, hostUUID string, certificateTemplateID uint) (*fleet.CertificateTemplateForHost, error) {
			return &fleet.CertificateTemplateForHost{
				HostUUID:              "host-uuid",
				CertificateTemplateID: 1,
				FleetChallenge:        ptr.String("template-challenge"), // Challenge from template
				Status:                &deliveredStatus,
				CAType:                fleet.CAConfigCustomSCEPProxy,
				CAName:                "android-ca",
			}, nil
		}
		ds.ConsumeChallengeFunc = func(ctx context.Context, challenge string) error {
			// Should use challenge from template since not provided in identifier
			assert.Equal(t, "template-challenge", challenge)
			return nil
		}
		svc := newTestService(ds)

		// No challenge in identifier - should use one from template
		identifier := makeIdentifier("host-uuid", "g1", "custom_scep_proxy", "")
		scepURL, err := svc.validateIdentifier(ctx, identifier, true)
		require.NoError(t, err)
		assert.Equal(t, "https://scep.example.com/scep", scepURL)
		assert.True(t, ds.ConsumeChallengeFuncInvoked)
		ds.ConsumeChallengeFuncInvoked = false
	})
}
