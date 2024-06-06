package main

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
	"flag"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"go.mozilla.org/pkcs7"
)

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func generateKeyPair(expiry time.Time) ([]byte, []byte, error) {
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
		NotAfter:              expiry,
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

func main() {
	mysqlAddr := flag.String("mysql", "localhost:3306", "mysql address")
	privateKey := flag.String("private-key", "", "same value as FLEET_SERVER_PRIVATE_KEY")
	expiryDays := flag.Duration("expiry-days", 15, "")
	flag.Parse()

	expiry := time.Now().Add(*expiryDays * 24 * time.Hour)
	certPEM, keyPEM, err := generateKeyPair(expiry)
	fatal(err)

	testBMToken := &nanodep_client.OAuth1Tokens{
		ConsumerKey:       "test_consumer",
		ConsumerSecret:    "test_secret",
		AccessToken:       "test_access_token",
		AccessSecret:      "test_access_secret",
		AccessTokenExpiry: expiry,
	}

	rawToken, err := json.Marshal(testBMToken)
	fatal(err)

	smimeToken := fmt.Sprintf(
		"Content-Type: text/plain;charset=UTF-8\r\n"+
			"Content-Transfer-Encoding: 7bit\r\n"+
			"\r\n%s", rawToken,
	)

	block, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(block.Bytes)
	fatal(err)

	encryptedToken, err := pkcs7.Encrypt([]byte(smimeToken), []*x509.Certificate{cert})
	fatal(err)

	tokenBytes := fmt.Sprintf(
		"Content-Type: application/pkcs7-mime; name=\"smime.p7m\"; smime-type=enveloped-data\r\n"+
			"Content-Transfer-Encoding: base64\r\n"+
			"Content-Disposition: attachment; filename=\"smime.p7m\"\r\n"+
			"Content-Description: S/MIME Encrypted Message\r\n"+
			"\r\n%s", base64.StdEncoding.EncodeToString(encryptedToken))

	tc := config.TestConfig()
	tc.Server.PrivateKey = *privateKey
	cfg := config.MysqlConfig{
		Protocol:        "tcp",
		Address:         *mysqlAddr,
		Database:        "fleet",
		Username:        "fleet",
		Password:        "insecure",
		MaxOpenConns:    50,
		MaxIdleConns:    50,
		ConnMaxLifetime: 0,
	}
	opts := []mysql.DBOption{
		mysql.WithFleetConfig(&tc),
	}
	mds, err := mysql.New(cfg, clock.C, opts...)
	fatal(err)

	mds.DeleteMDMConfigAssetsByName(context.Background(), []fleet.MDMAssetName{
		fleet.MDMAssetABMCert,
		fleet.MDMAssetABMKey,
		fleet.MDMAssetABMToken,
		fleet.MDMAssetAPNSCert,
		fleet.MDMAssetAPNSKey,
		fleet.MDMAssetCACert,
		fleet.MDMAssetCAKey,
	})

	assets := []fleet.MDMConfigAsset{
		{Name: fleet.MDMAssetABMCert, Value: certPEM},
		{Name: fleet.MDMAssetABMKey, Value: keyPEM},
		{Name: fleet.MDMAssetABMToken, Value: []byte(tokenBytes)},
		{Name: fleet.MDMAssetAPNSCert, Value: certPEM},
		{Name: fleet.MDMAssetAPNSKey, Value: keyPEM},
		{Name: fleet.MDMAssetCACert, Value: certPEM},
		{Name: fleet.MDMAssetCAKey, Value: keyPEM},
	}
	err = mds.InsertMDMConfigAssets(context.Background(), assets)
	fatal(err)

}
