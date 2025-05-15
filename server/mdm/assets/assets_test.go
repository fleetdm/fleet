package assets

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/jmoiron/sqlx"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/require"
)

// generateTestCert generates a test certificate and key.
func generateTestCert() ([]byte, []byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			ExtraNames: []pkix.AttributeTypeAndValue{
				{
					Type:  asn1.ObjectIdentifier{0, 9, 2342, 19200300, 100, 1, 1},
					Value: "com.apple.mgmt.Example",
				},
			},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return certPEM, keyPEM, nil
}

func TestCAKeyPair(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)

	certPEM, keyPEM, err := generateTestCert()
	require.NoError(t, err)

	assets := map[fleet.MDMAssetName]fleet.MDMConfigAsset{
		fleet.MDMAssetCACert: {Value: certPEM},
		fleet.MDMAssetCAKey:  {Value: keyPEM},
	}

	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		require.ElementsMatch(t, []fleet.MDMAssetName{fleet.MDMAssetCACert, fleet.MDMAssetCAKey}, assetNames)
		return assets, nil
	}

	cert, err := CAKeyPair(ctx, ds)
	require.NoError(t, err)
	require.NotNil(t, cert)
	require.True(t, ds.GetAllMDMConfigAssetsByNameFuncInvoked)
}

func TestAPNSKeyPair(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)

	certPEM, keyPEM, err := generateTestCert()
	require.NoError(t, err)

	assets := map[fleet.MDMAssetName]fleet.MDMConfigAsset{
		fleet.MDMAssetAPNSCert: {Value: certPEM},
		fleet.MDMAssetAPNSKey:  {Value: keyPEM},
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		require.ElementsMatch(t, []fleet.MDMAssetName{fleet.MDMAssetAPNSCert, fleet.MDMAssetAPNSKey}, assetNames)
		return assets, nil
	}
	cert, err := APNSKeyPair(ctx, ds)
	require.NoError(t, err)
	require.NotNil(t, cert)
	require.True(t, ds.GetAllMDMConfigAssetsByNameFuncInvoked)
}

func TestX509Cert(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)

	certPEM, _, err := generateTestCert()
	require.NoError(t, err)

	assets := map[fleet.MDMAssetName]fleet.MDMConfigAsset{
		fleet.MDMAssetAPNSCert: {Value: certPEM},
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		require.ElementsMatch(t, []fleet.MDMAssetName{fleet.MDMAssetAPNSCert}, assetNames)
		return assets, nil
	}

	cert, err := X509Cert(ctx, ds, fleet.MDMAssetAPNSCert)
	require.NoError(t, err)
	require.NotNil(t, cert)
	require.True(t, ds.GetAllMDMConfigAssetsByNameFuncInvoked)
}

func TestAPNSTopic(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)

	certPEM, _, err := generateTestCert()
	require.NoError(t, err)

	assets := map[fleet.MDMAssetName]fleet.MDMConfigAsset{
		fleet.MDMAssetAPNSCert: {Value: certPEM},
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		require.ElementsMatch(t, []fleet.MDMAssetName{fleet.MDMAssetAPNSCert}, assetNames)
		return assets, nil
	}

	topic, err := APNSTopic(ctx, ds)
	require.NoError(t, err)
	require.NotEmpty(t, topic)
	require.True(t, ds.GetAllMDMConfigAssetsByNameFuncInvoked)
}

func TestABMToken(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)

	certPEM, keyPEM, err := generateTestCert()
	require.NoError(t, err)

	testBMToken := &nanodep_client.OAuth1Tokens{
		ConsumerKey:       "test_consumer",
		ConsumerSecret:    "test_secret",
		AccessToken:       "test_access_token",
		AccessSecret:      "test_access_secret",
		AccessTokenExpiry: time.Date(2999, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	rawToken, err := json.Marshal(testBMToken)
	require.NoError(t, err)

	smimeToken := fmt.Sprintf(
		"Content-Type: text/plain;charset=UTF-8\r\n"+
			"Content-Transfer-Encoding: 7bit\r\n"+
			"\r\n%s", rawToken,
	)

	block, _ := pem.Decode(certPEM)
	require.NotNil(t, block)
	require.Equal(t, "CERTIFICATE", block.Type)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	encryptedToken, err := pkcs7.Encrypt([]byte(smimeToken), []*x509.Certificate{cert})
	require.NoError(t, err)

	tokenBytes := fmt.Sprintf(
		"Content-Type: application/pkcs7-mime; name=\"smime.p7m\"; smime-type=enveloped-data\r\n"+
			"Content-Transfer-Encoding: base64\r\n"+
			"Content-Disposition: attachment; filename=\"smime.p7m\"\r\n"+
			"Content-Description: S/MIME Encrypted Message\r\n"+
			"\r\n%s", base64.StdEncoding.EncodeToString(encryptedToken))

	assets := map[fleet.MDMAssetName]fleet.MDMConfigAsset{
		fleet.MDMAssetABMCert: {Value: certPEM},
		fleet.MDMAssetABMKey:  {Value: keyPEM},
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		require.ElementsMatch(t, []fleet.MDMAssetName{
			fleet.MDMAssetABMCert,
			fleet.MDMAssetABMKey,
		}, assetNames)
		return assets, nil
	}
	const testOrgName = "test-org"

	ds.GetABMTokenByOrgNameFunc = func(ctx context.Context, orgName string) (*fleet.ABMToken, error) {
		require.Equal(t, testOrgName, orgName)
		return &fleet.ABMToken{
			ID:               1,
			OrganizationName: testOrgName,
			EncryptedToken:   []byte(tokenBytes),
		}, nil
	}

	tokens, err := ABMToken(ctx, ds, testOrgName)
	require.NoError(t, err)
	require.NotNil(t, tokens)
	require.Equal(t, "test_access_secret", tokens.AccessSecret)
	require.True(t, ds.GetAllMDMConfigAssetsByNameFuncInvoked)
	require.True(t, ds.GetABMTokenByOrgNameFuncInvoked)
}
