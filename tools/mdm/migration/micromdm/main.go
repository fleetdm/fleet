package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/boltdb/bolt"
	boltdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot/bolt"
	configbuiltin "github.com/micromdm/micromdm/platform/config/builtin"
	"github.com/micromdm/micromdm/platform/pubsub/inmem"
)

var (
	apnsKeyPath  = "apns.key"
	apnsCertPath = "apns.crt"
	scepKeyPath  = "scep.key"
	scepCertPath = "scep.crt"
	depKeyPath   = "ade.key"
	depCertPath  = "ade.crt"
)

func main() {
	var (
		flDB = flag.String("db", "/var/db/micromdm.db", "path to micromdm DB")
	)
	flag.Parse()

	boltDB, err := bolt.Open(*flDB, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	ps := inmem.NewPubSub()

	configDB, err := configbuiltin.NewDB(boltDB, ps)
	if err != nil {
		log.Fatal(err)
	}

	svcBoltDepot, err := boltdepot.NewBoltDepot(boltDB)
	if err != nil {
		log.Fatal(err)
	}

	key, err := svcBoltDepot.CreateOrLoadKey(2048)
	if err != nil {
		log.Fatal(err)
	}

	crt, err := svcBoltDepot.CreateOrLoadCA(key, 5, "MicroMDM", "US")
	if err != nil {
		log.Fatal(err)
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(key)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	if err := os.WriteFile(scepKeyPath, privateKeyPEM, 0o777); err != nil {
		log.Fatal(err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: crt.Raw,
	})

	if err := os.WriteFile(scepCertPath, certPEM, 0o777); err != nil {
		log.Fatal(err)
	}

	pushCert, err := configDB.PushCertificate()
	if err != nil {
		log.Fatal(err)
	}

	rsaKey, ok := pushCert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		log.Fatal("stored APNs key is not in RSA format")
	}

	pushKeyBytes := x509.MarshalPKCS1PrivateKey(rsaKey)
	pushKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: pushKeyBytes,
	})
	if err := os.WriteFile(apnsKeyPath, pushKeyPEM, 0o777); err != nil {
		log.Fatal(err)
	}

	pushCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: pushCert.Certificate[0],
	})
	if err := os.WriteFile(apnsCertPath, pushCertPEM, 0o777); err != nil {
		log.Fatal(err)
	}

	depKey, depCert, err := configDB.DEPKeypair()
	if err != nil {
		log.Fatal(err)
	}

	depKeyBytes := x509.MarshalPKCS1PrivateKey(depKey)
	depKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: depKeyBytes,
	})
	if err := os.WriteFile(depKeyPath, depKeyPEM, 0o777); err != nil {
		log.Fatal(err)
	}

	depCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: depCert.Raw,
	})
	if err := os.WriteFile(depCertPath, depCertPEM, 0o777); err != nil {
		log.Fatal(err)
	}

	fmt.Printf(`
============================

Success! Exported:

- %s
- %s
- %s
- %s
- %s
- %s
`, apnsKeyPath, apnsCertPath, scepKeyPath, scepCertPath, depKeyPath, depCertPath)

}
