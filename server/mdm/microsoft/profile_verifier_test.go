package microsoft_mdm

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPreprocessWindowsProfileContentsForDeployment(t *testing.T) {
	ds := new(mock.Store)

	scimUser := &fleet.ScimUser{
		UserName:   "test@idp.com",
		GivenName:  ptr.String("First"),
		FamilyName: ptr.String("Last"),
		Department: ptr.String("Department"),
		Groups: []fleet.ScimUserGroup{
			{
				ID:          1,
				DisplayName: "Group One",
			},
			{
				ID:          2,
				DisplayName: "Group Two",
			},
		},
	}

	baseSetup := func() {
		ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
			if ds.GetAllCertificateAuthoritiesFunc == nil {
				return &fleet.GroupedCertificateAuthorities{
					CustomScepProxy: []fleet.CustomSCEPProxyCA{},
				}, nil
			}

			cas, err := ds.GetAllCertificateAuthoritiesFunc(ctx, includeSecrets)
			if err != nil {
				return nil, err
			}

			return fleet.GroupCertificateAuthoritiesByType(cas)
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					ServerURL: "https://test-fleet.com",
				},
			}, nil
		}
		ds.HostIDsByIdentifierFunc = func(ctx context.Context, filter fleet.TeamFilter, hostnames []string) ([]uint, error) {
			return []uint{42}, nil
		}

		ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
			if hostID == 42 {
				return scimUser, nil
			}

			return nil, fmt.Errorf("no scim user for host id %d", hostID)
		}
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
			return []*fleet.HostDeviceMapping{}, nil
		}
	}

	// use the same uuid for all profile UUID actions
	profileUUID := uuid.NewString()

	tests := []struct {
		name             string
		hostUUID         string
		profileContents  string
		expectedContents string
		expectError      bool
		processingError  string                                                          // if set then we expect the error to be of type MicrosoftProfileProcessingError with this message
		setup            func()                                                          // Used for setting up datastore mocks.
		expect           func(t *testing.T, managedCerts []*fleet.MDMManagedCertificate) // Add more params as they need validation.
		freeTier         bool
	}{
		{
			name:             "no fleet variables",
			hostUUID:         "test-uuid-123",
			profileContents:  `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Simple Value</Data></Item></Replace>`,
			expectedContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Simple Value</Data></Item></Replace>`,
		},
		{
			name:             "host uuid fleet variable",
			hostUUID:         "test-uuid-456",
			profileContents:  `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Device ID: $FLEET_VAR_HOST_UUID</Data></Item></Replace>`,
			expectedContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Device ID: test-uuid-456</Data></Item></Replace>`,
		},
		{
			name:             "host serial fleet variable",
			hostUUID:         "test-uuid-456",
			profileContents:  `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Device Serial: $FLEET_VAR_HOST_HARDWARE_SERIAL</Data></Item></Replace>`,
			expectedContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Device Serial: test-serial-456</Data></Item></Replace>`,
			setup: func() {
				ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, filter fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
					require.Equal(t, []string{"test-uuid-456"}, uuids)
					return []*fleet.Host{
						{
							UUID:           "test-uuid-456",
							HardwareSerial: "test-serial-456",
						},
					}, nil
				}
			},
		},
		{
			name:             "host serial fleet variable with blank serial",
			hostUUID:         "test-uuid-456",
			profileContents:  `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Device Serial: $FLEET_VAR_HOST_HARDWARE_SERIAL</Data></Item></Replace>`,
			expectedContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Device Serial: $FLEET_VAR_HOST_HARDWARE_SERIAL</Data></Item></Replace>`,
			expectError:      true,
			processingError:  "There is no serial number for this host. Fleet couldn't populate $FLEET_VAR_HOST_HARDWARE_SERIAL.",
			setup: func() {
				ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, filter fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
					require.Equal(t, []string{"test-uuid-456"}, uuids)
					return []*fleet.Host{
						{
							UUID: "test-uuid-456",
							ID:   1234,
						},
					}, nil
				}
			},
		},
		{
			name:             "host serial with multiple hosts matching the same UUID",
			hostUUID:         "test-uuid-789",
			profileContents:  `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Device Serial: $FLEET_VAR_HOST_HARDWARE_SERIAL</Data></Item></Replace>`,
			expectedContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Device Serial: $FLEET_VAR_HOST_HARDWARE_SERIAL</Data></Item></Replace>`,
			expectError:      true,
			processingError:  "Found 2 hosts with UUID test-uuid-789. Profile variable substitution for $FLEET_VAR_HOST_HARDWARE_SERIAL requires exactly one host",
			expect: func(t *testing.T, managedCerts []*fleet.MDMManagedCertificate) {
				require.True(t, ds.UpdateOrDeleteHostMDMWindowsProfileFuncInvoked)
			},
			setup: func() {
				ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, filter fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
					require.Equal(t, []string{"test-uuid-789"}, uuids)
					return []*fleet.Host{
						{
							UUID:           "test-uuid-789",
							HardwareSerial: "test-serial-456",
						},
						{
							UUID:           "test-uuid-789",
							HardwareSerial: "test-serial-789",
						},
					}, nil
				}
				ds.UpdateOrDeleteHostMDMWindowsProfileFunc = func(ctx context.Context, profile *fleet.HostMDMWindowsProfile) error {
					return nil
				}
			},
		},
		{
			name:             "host platform fleet variable",
			hostUUID:         "test-uuid-67",
			profileContents:  `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Device Platform: $FLEET_VAR_HOST_PLATFORM</Data></Item></Replace>`,
			expectedContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>Device Platform: windows</Data></Item></Replace>`,
		},
		{
			name:             "scep windows certificate id",
			hostUUID:         "test-host-1234-uuid",
			profileContents:  `<Replace><Data>SCEP: $FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID</Data></Replace>`,
			expectedContents: fmt.Sprintf(`<Replace><Data>SCEP: %s</Data></Replace>`, profileUUID),
		},
		{
			name:            "custom scep proxy url not usable in free tier",
			hostUUID:        "test-host-1234-uuid",
			profileContents: `<Replace><Data>CA: $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_CERTIFICATE</Data></Replace>`,
			expectError:     true,
			processingError: "Custom SCEP integration requires a Fleet Premium license.",
			freeTier:        true,
		},
		{
			name:            "custom scep proxy url ca not found",
			hostUUID:        "test-host-1234-uuid",
			profileContents: `<Replace><Data>CA: $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_CERTIFICATE</Data></Replace>`,
			expectError:     true,
			processingError: "Fleet couldn't populate $CUSTOM_SCEP_PROXY_URL_CERTIFICATE because CERTIFICATE certificate authority doesn't exist.",
		},
		{
			name:             "custom scep proxy url ca found and replaced",
			hostUUID:         "test-host-1234-uuid",
			profileContents:  `<Replace><Data>     $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_CERTIFICATE</Data></Replace>`,
			expectedContents: `<Replace><Data>https://test-fleet.com/mdm/scep/proxy/test-host-1234-uuid%2C` + profileUUID + `%2CCERTIFICATE%2Csupersecret</Data></Replace>`,
			setup: func() {
				ds.GetAllCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) ([]*fleet.CertificateAuthority, error) {
					return []*fleet.CertificateAuthority{
						{
							ID:        1,
							Name:      ptr.String("CERTIFICATE"),
							Type:      string(fleet.CATypeCustomSCEPProxy),
							URL:       ptr.String("https://scep.proxy.url/scep"),
							Challenge: ptr.String("supersecret"),
						},
					}, nil
				}
				ds.NewChallengeFunc = func(ctx context.Context) (string, error) {
					return "supersecret", nil
				}
			},
			expect: func(t *testing.T, managedCerts []*fleet.MDMManagedCertificate) {
				require.Len(t, managedCerts, 1)
				require.Equal(t, "CERTIFICATE", managedCerts[0].CAName)
				require.Equal(t, fleet.CAConfigCustomSCEPProxy, managedCerts[0].Type)
			},
		},
		{
			name:            "custom scep challenge not usable in free tier",
			hostUUID:        "test-host-1234-uuid",
			profileContents: `<Replace><Data>CA: $FLEET_VAR_CUSTOM_SCEP_CHALLENGE_CERTIFICATE</Data></Replace>`,
			expectError:     true,
			processingError: "Custom SCEP integration requires a Fleet Premium license.",
			freeTier:        true,
		},
		{
			name:            "custom scep proxy challenge ca not found",
			hostUUID:        "test-host-1234-uuid",
			profileContents: `<Replace><Data>CA: $FLEET_VAR_CUSTOM_SCEP_CHALLENGE_CERTIFICATE</Data></Replace>`,
			expectError:     true,
			processingError: "Fleet couldn't populate $CUSTOM_SCEP_CHALLENGE_CERTIFICATE because CERTIFICATE certificate authority doesn't exist.",
		},
		{
			name:             "custom scep proxy challenge ca found and replaced",
			hostUUID:         "test-host-1234-uuid",
			profileContents:  `<Replace><Data>     $FLEET_VAR_CUSTOM_SCEP_CHALLENGE_CERTIFICATE</Data></Replace>`,
			expectedContents: `<Replace><Data>supersecret</Data></Replace>`,
			setup: func() {
				ds.GetAllCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) ([]*fleet.CertificateAuthority, error) {
					return []*fleet.CertificateAuthority{
						{
							ID:        1,
							Name:      ptr.String("CERTIFICATE"),
							Type:      string(fleet.CATypeCustomSCEPProxy),
							URL:       ptr.String("https://scep.proxy.url/scep"),
							Challenge: ptr.String("supersecret"),
						},
					}, nil
				}
				ds.NewChallengeFunc = func(ctx context.Context) (string, error) {
					return "supersecret", nil
				}
			},
		},
		{
			name:             "all idp variables",
			hostUUID:         "idp-host-uuid",
			profileContents:  `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>User: $FLEET_VAR_HOST_END_USER_IDP_USERNAME - $FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART - $FLEET_VAR_HOST_END_USER_IDP_GROUPS - $FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT - $FLEET_VAR_HOST_END_USER_IDP_FULL_NAME</Data></Item></Replace>`,
			expectedContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>User: test@idp.com - test - Group One,Group Two - Department - First Last</Data></Item></Replace>`,
		},
		{
			name:            "missing groups on idp user",
			hostUUID:        "no-groups-idp",
			profileContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>User: $FLEET_VAR_HOST_END_USER_IDP_GROUPS</Data></Item></Replace>`,
			expectError:     true,
			processingError: "There are no IdP groups for this host. Fleet couldn't populate $FLEET_VAR_HOST_END_USER_IDP_GROUPS.",
			setup: func() {
				scimUser.Groups = []fleet.ScimUserGroup{}
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					return scimUser, nil
				}
			},
		},
		{
			name:            "missing department on idp user",
			hostUUID:        "no-department-idp",
			profileContents: `<Replace><Item><Target><LocURI>./Device/Test</LocURI></Target><Data>User: $FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT</Data></Item></Replace>`,
			expectError:     true,
			processingError: "There is no IdP department for this host. Fleet couldn't populate $FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT.",
			setup: func() {
				scimUser.Department = nil
				ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
					return scimUser, nil
				}
			},
		},
	}

	hostIDForUUIDCache := make(map[string]uint)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseSetup()
			if tt.setup != nil {
				tt.setup()
			}
			t.Cleanup(func() {
				ds = new(mock.Store) // Reset the mock datastore after each test, to avoid overlapping setups.
			})

			licenseInfo := &fleet.LicenseInfo{
				Tier: fleet.TierPremium,
			}
			if tt.freeTier {
				licenseInfo.Tier = fleet.TierFree
			}
			ctx := license.NewContext(t.Context(), licenseInfo)

			appConfig, err := ds.AppConfig(ctx)
			require.NoError(t, err)

			// Populate this one, in setup by mocking ds.GetAllCertificateAuthoritiesFunc if needed.
			groupedCAs, err := ds.GetGroupedCertificateAuthorities(ctx, true)
			require.NoError(t, err)

			customSCEPCAs := groupedCAs.ToCustomSCEPProxyCAMap()

			managedCertificates := &[]*fleet.MDMManagedCertificate{}

			deps := ProfilePreprocessDependencies{
				Context:                    ctx,
				Logger:                     slog.New(slog.DiscardHandler),
				DataStore:                  ds,
				HostIDForUUIDCache:         hostIDForUUIDCache,
				AppConfig:                  appConfig,
				CustomSCEPCAs:              customSCEPCAs,
				ManagedCertificatePayloads: managedCertificates,
			}

			result, err := PreprocessWindowsProfileContentsForDeployment(deps, ProfilePreprocessParams{
				HostUUID:    tt.hostUUID,
				ProfileUUID: profileUUID,
			}, tt.profileContents)
			if tt.expectError {
				require.Error(t, err)
				if tt.processingError != "" {
					var processingErr *MicrosoftProfileProcessingError
					require.ErrorAs(t, err, &processingErr, "expected ProfileProcessingError")
					require.Equal(t, tt.processingError, processingErr.Error())
				}
				return // do not verify profile contents if an error is expected
			}

			require.Equal(t, tt.expectedContents, result)
			require.NoError(t, err)

			if tt.expect != nil {
				tt.expect(t, *managedCertificates)
			}
		})
	}

	require.Len(t, hostIDForUUIDCache, 3) // make sure cache is populated by IdP var host UUID lookups
}
