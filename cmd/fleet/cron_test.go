package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/log/stdlogfmt"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	"github.com/fleetdm/fleet/v4/server/mock"
	svcmock "github.com/fleetdm/fleet/v4/server/service/mock"
	kitlog "github.com/go-kit/log"
	"github.com/google/uuid"
	nanodep_client "github.com/micromdm/nanodep/client"
	"github.com/stretchr/testify/require"
)

func generateCertWithAPNsTopic() ([]byte, []byte, error) {
	// generate a new private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// set up the OID for UID
	oidUID := asn1.ObjectIdentifier{0, 9, 2342, 19200300, 100, 1, 1}

	// set up a certificate template with the required UID in the Subject
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			ExtraNames: []pkix.AttributeTypeAndValue{
				{
					Type:  oidUID,
					Value: "com.apple.mgmt.Example",
				},
			},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// create a self-signed certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	// encode to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return certPEM, keyPEM, nil
}

func setupTest(t *testing.T) (context.Context, kitlog.Logger, *mock.Store, *config.FleetConfig, *mock.MDMAppleStore, *apple_mdm.MDMAppleCommander) {
	ctx := context.Background()
	logger := kitlog.NewNopLogger()
	cfg := config.TestConfig()
	testCertPEM, testKeyPEM, err := generateCertWithAPNsTopic()
	require.NoError(t, err)
	config.SetTestMDMConfig(t, &cfg, testCertPEM, testKeyPEM, testBMToken, "../../server/service/testdata")
	ds := new(mock.Store)
	mdmStorage := &mock.MDMAppleStore{}
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		stdlogfmt.New(),
	)
	commander := apple_mdm.NewMDMAppleCommander(mdmStorage, pusher)

	return ctx, logger, ds, &cfg, mdmStorage, commander
}

func TestRenewSCEPCertificatesMDMConfigNotSet(t *testing.T) {
	ctx, logger, ds, cfg, _, commander := setupTest(t)
	cfg.MDM = config.MDMConfig{} // ensure MDM is not fully configured
	err := renewSCEPCertificates(ctx, logger, ds, cfg, commander)
	require.NoError(t, err)
}

func TestRenewSCEPCertificatesCommanderNil(t *testing.T) {
	ctx, logger, ds, cfg, _, _ := setupTest(t)
	err := renewSCEPCertificates(ctx, logger, ds, cfg, nil)
	require.NoError(t, err)
}

func TestRenewSCEPCertificatesBranches(t *testing.T) {
	tests := []struct {
		name               string
		customExpectations func(*testing.T, *mock.Store, *config.FleetConfig, *mock.MDMAppleStore, *apple_mdm.MDMAppleCommander)
		expectedError      bool
	}{
		{
			name: "GetMDMAppleSCEPCertsCloseToExpiry Errors",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetMDMAppleSCEPCertsCloseToExpiryFunc = func(ctx context.Context, expiryDays, limit int) ([]fleet.SCEPIdentityCertificate, error) {
					return nil, errors.New("database error")
				}
			},
			expectedError: true,
		},
		{
			name: "No Certs to Renew",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetMDMAppleSCEPCertsCloseToExpiryFunc = func(ctx context.Context, expiryDays, limit int) ([]fleet.SCEPIdentityCertificate, error) {
					return []fleet.SCEPIdentityCertificate{}, nil
				}
			},
			expectedError: false,
		},
		{
			name: "DecodeCertPEM Errors",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetMDMAppleSCEPCertsCloseToExpiryFunc = func(ctx context.Context, expiryDays, limit int) ([]fleet.SCEPIdentityCertificate, error) {
					return []fleet.SCEPIdentityCertificate{{CertificatePEM: []byte("invalid PEM data")}}, nil
				}
			},
			expectedError: false,
		},
		{
			name: "GetHostCertAssociationByCertSHA Errors",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetHostCertAssociationByCertSHAFunc = func(ctx context.Context, shas []string) ([]fleet.SCEPIdentityAssociation, error) {
					return nil, errors.New("database error")
				}
			},
			expectedError: true,
		},
		{
			name: "AppConfig Errors",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
					return nil, errors.New("app config error")
				}
			},
			expectedError: true,
		},
		{
			name: "InstallProfile for hostsWithoutRefs",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetHostCertAssociationByCertSHAFunc = func(ctx context.Context, shas []string) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollReference: ""}}, nil
				}

				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
					require.Equal(t, "InstallProfile", cmd.Command.RequestType)
					// TODO:
					// require.Contains(t, "", cmd.Raw)
					return map[string]error{}, nil
				}

				t.Cleanup(func() {
					require.True(t, appleStore.EnqueueCommandFuncInvoked)
				})
			},
			expectedError: false,
		},
		{
			name: "InstallProfile for hostsWithoutRefs fails",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetHostCertAssociationByCertSHAFunc = func(ctx context.Context, shas []string) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollReference: ""}}, nil
				}

				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
					return map[string]error{}, errors.New("foo")
				}
			},
			expectedError: true,
		},
		{
			name: "InstallProfile for hostsWithRefs",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetHostCertAssociationByCertSHAFunc = func(ctx context.Context, shas []string) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID2", EnrollReference: "ref1"}}, nil
				}
				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
					require.Equal(t, "InstallProfile", cmd.Command.RequestType)
					// TODO:
					// require.Contains(t, "", cmd.Raw)
					return map[string]error{}, nil
				}
				t.Cleanup(func() {
					require.True(t, appleStore.EnqueueCommandFuncInvoked)
				})
			},
			expectedError: false,
		},
		{
			name: "InstallProfile for hostsWithRefs fails",
			customExpectations: func(t *testing.T, ds *mock.Store, cfg *config.FleetConfig, appleStore *mock.MDMAppleStore, commander *apple_mdm.MDMAppleCommander) {
				ds.GetHostCertAssociationByCertSHAFunc = func(ctx context.Context, shas []string) ([]fleet.SCEPIdentityAssociation, error) {
					return []fleet.SCEPIdentityAssociation{{HostUUID: "hostUUID1", EnrollReference: "ref1"}}, nil
				}

				appleStore.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
					return map[string]error{}, errors.New("foo")
				}
			},
			expectedError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, logger, ds, cfg, appleStorage, commander := setupTest(t)

			ds.GetMDMAppleSCEPCertsCloseToExpiryFunc = func(ctx context.Context, expiryDays, limit int) ([]fleet.SCEPIdentityCertificate, error) {
				cert, _, err := apple_mdm.NewDEPKeyPairPEM()
				require.NoError(t, err)
				return []fleet.SCEPIdentityCertificate{{CertificatePEM: cert}}, nil
			}

			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				appCfg := &fleet.AppConfig{}
				appCfg.OrgInfo.OrgName = "fl33t"
				appCfg.ServerSettings.ServerURL = "https://foo.example.com"
				return appCfg, nil
			}

			ds.GetHostCertAssociationByCertSHAFunc = func(ctx context.Context, shas []string) ([]fleet.SCEPIdentityAssociation, error) {
				return []fleet.SCEPIdentityAssociation{}, nil
			}

			appleStorage.RetrievePushInfoFunc = func(ctx context.Context, targets []string) (map[string]*mdm.Push, error) {
				pushes := make(map[string]*mdm.Push, len(targets))
				for _, uuid := range targets {
					pushes[uuid] = &mdm.Push{
						PushMagic: "magic" + uuid,
						Token:     []byte("token" + uuid),
						Topic:     "topic" + uuid,
					}
				}

				return pushes, nil
			}

			appleStorage.RetrievePushCertFunc = func(ctx context.Context, topic string) (*tls.Certificate, string, error) {
				cert, err := tls.LoadX509KeyPair("../../server/service/testdata/server.pem", "../../server/service/testdata/server.key")
				return &cert, "", err
			}

			tc.customExpectations(t, ds, cfg, appleStorage, commander)

			err := renewSCEPCertificates(ctx, logger, ds, cfg, commander)
			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

var testBMToken = &nanodep_client.OAuth1Tokens{
	ConsumerKey:       "test_consumer",
	ConsumerSecret:    "test_secret",
	AccessToken:       "test_access_token",
	AccessSecret:      "test_access_secret",
	AccessTokenExpiry: time.Date(2999, 1, 1, 0, 0, 0, 0, time.UTC),
}

func newMockAPNSPushProviderFactory() (*svcmock.APNSPushProviderFactory, *svcmock.APNSPushProvider) {
	provider := &svcmock.APNSPushProvider{}
	provider.PushFunc = mockSuccessfulPush
	factory := &svcmock.APNSPushProviderFactory{}
	factory.NewPushProviderFunc = func(*tls.Certificate) (push.PushProvider, error) {
		return provider, nil
	}

	return factory, provider
}

func mockSuccessfulPush(pushes []*mdm.Push) (map[string]*push.Response, error) {
	res := make(map[string]*push.Response, len(pushes))
	for _, p := range pushes {
		res[p.Token.String()] = &push.Response{
			Id:  uuid.New().String(),
			Err: nil,
		}
	}
	return res, nil
}
