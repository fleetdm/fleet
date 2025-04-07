package assets

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/tokenpki"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
)

func CAKeyPair(ctx context.Context, ds fleet.MDMAssetRetriever) (*tls.Certificate, error) {
	return KeyPair(ctx, ds, fleet.MDMAssetCACert, fleet.MDMAssetCAKey)
}

func APNSKeyPair(ctx context.Context, ds fleet.MDMAssetRetriever) (*tls.Certificate, error) {
	return KeyPair(ctx, ds, fleet.MDMAssetAPNSCert, fleet.MDMAssetAPNSKey)
}

func KeyPair(ctx context.Context, ds fleet.MDMAssetRetriever, certName, keyName fleet.MDMAssetName) (*tls.Certificate, error) {
	assets, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		certName,
		keyName,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("loading %s, %s keypair from the database: %w", certName, keyName, err)
	}

	cert, err := tls.X509KeyPair(assets[certName].Value, assets[keyName].Value)
	if err != nil {
		return nil, fmt.Errorf("parsing %s, %s keypair: %w", certName, keyName, err)
	}

	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("parsing %s certificate leaf: %w", certName, err)
	}

	return &cert, nil
}

func X509Cert(ctx context.Context, ds fleet.MDMAssetRetriever, certName fleet.MDMAssetName) (*x509.Certificate, error) {
	assets, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{certName}, nil)
	if err != nil {
		return nil, fmt.Errorf("loading certificate %s from the database: %w", certName, err)
	}

	block, _ := pem.Decode(assets[certName].Value)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("decoding certificate PEM data: %w", err)
	}

	return x509.ParseCertificate(block.Bytes)
}

func APNSTopic(ctx context.Context, ds fleet.MDMAssetRetriever) (string, error) {
	cert, err := X509Cert(ctx, ds, fleet.MDMAssetAPNSCert)
	if err != nil {
		return "", fmt.Errorf("retrieving APNs cert: %w", err)
	}

	mdmPushCertTopic, err := cryptoutil.TopicFromCert(cert)
	if err != nil {
		return "", fmt.Errorf("extracting topic from APNs certificate: %w", err)
	}

	return mdmPushCertTopic, nil
}

func ABMToken(ctx context.Context, ds fleet.MDMAssetRetriever, abmOrgName string) (*nanodep_client.OAuth1Tokens, error) {
	assets, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetABMKey,
		fleet.MDMAssetABMCert,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("loading ABM assets from the database: %w", err)
	}

	abmTok, err := ds.GetABMTokenByOrgName(ctx, abmOrgName)
	if err != nil {
		return nil, fmt.Errorf("get ABM token by name: %w", err)
	}

	cert, err := tls.X509KeyPair(assets[fleet.MDMAssetABMCert].Value, assets[fleet.MDMAssetABMKey].Value)
	if err != nil {
		return nil, fmt.Errorf("parsing ABM keypair: %w", err)
	}

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("parsing ABM certificate: %w", err)
	}

	oAuthTok, err := DecryptRawABMToken(
		abmTok.EncryptedToken,
		leaf,
		assets[fleet.MDMAssetABMKey].Value,
	)
	if err != nil {
		return nil, fmt.Errorf("decrypting ABM token: %w", err)
	}

	return oAuthTok, nil
}

func DecryptRawABMToken(tokenBytes []byte, cert *x509.Certificate, keyPEM []byte) (*nanodep_client.OAuth1Tokens, error) {
	bmKey, err := tokenpki.RSAKeyFromPEM(keyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	token, err := tokenpki.DecryptTokenJSON(tokenBytes, cert, bmKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt token: %w", err)
	}
	var jsonTok nanodep_client.OAuth1Tokens
	if err := json.Unmarshal(token, &jsonTok); err != nil {
		return nil, fmt.Errorf("unmarshal JSON token: %w", err)
	}
	return &jsonTok, nil
}
