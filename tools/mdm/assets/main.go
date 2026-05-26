package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/cryptoutil"
)

const (
	testUsername = "fleet"
	testPassword = "insecure"
	testAddress  = "localhost:3306"
	testDatabase = "fleet"
)

var (
	exportCmd       = flag.NewFlagSet("export", flag.ExitOnError)
	importCmd       = flag.NewFlagSet("import", flag.ExitOnError)
	rolloverCmd     = flag.NewFlagSet("rollover-ca-cert", flag.ExitOnError)
	flagKey         string
	flagDir         string
	flagDBUser      string
	flagDBPass      string
	flagDBAddress   string
	flagDBName      string
	flagImportName  string
	flagImportValue string
	flagExportName  string
	flagExtendYears int

	validNames = map[fleet.MDMAssetName]struct{}{
		fleet.MDMAssetABMCert:                  {},
		fleet.MDMAssetABMTokenDeprecated:       {},
		fleet.MDMAssetABMKey:                   {},
		fleet.MDMAssetAPNSCert:                 {},
		fleet.MDMAssetAPNSKey:                  {},
		fleet.MDMAssetCACert:                   {},
		fleet.MDMAssetCAKey:                    {},
		fleet.MDMAssetSCEPChallenge:            {},
		fleet.MDMAssetVPPTokenDeprecated:       {},
		fleet.MDMAssetAndroidFleetServerSecret: {},
	}
)

func setupSharedFlags() {
	for _, fs := range []*flag.FlagSet{exportCmd, importCmd, rolloverCmd} {
		fs.StringVar(&flagKey, "key", "", "Key used to encrypt the assets")
		fs.StringVar(&flagDir, "dir", "", "Directory to put the exported assets")
		fs.StringVar(&flagDBUser, "db-user", testUsername, "Username used to connect to the MySQL instance")
		fs.StringVar(&flagDBPass, "db-password", testPassword, "Password used to connect to the MySQL instance")
		fs.StringVar(&flagDBAddress, "db-address", testAddress, "Address used to connect to the MySQL instance")
		fs.StringVar(&flagDBName, "db-name", testDatabase, "Name of the database with the asset information in the MySQL instance")
	}
}

func setupDS(privateKey, userName, password, address, name string) *mysql.Datastore {
	db, err := sql.Open(
		"mysql",
		fmt.Sprintf("%s:%s@tcp(%s)/?multiStatements=true&tls=skip-verify", testUsername, testPassword, testAddress),
	)
	if err != nil {
		log.Fatal("opening MySQL connection:", err)
	}
	defer db.Close()

	mysqlCfg := config.MysqlConfig{
		Username:  userName,
		Password:  password,
		Address:   address,
		Database:  name,
		TLSConfig: "skip-verify",
	}
	ds, err := mysql.New(
		mysqlCfg,
		clock.NewMockClock(),
		mysql.LimitAttempts(1),
		mysql.WithFleetConfig(&config.FleetConfig{
			Server: config.ServerConfig{
				PrivateKey: privateKey,
			},
		}),
	)
	if err != nil {
		log.Fatal("creating datastore instance:", err) //nolint:gocritic // ignore exitAfterDefer
	}

	return ds
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("invalid subcommand, expected import, export or rollover-ca-cert") //nolint:gocritic // ignore exitAfterDefer
	}

	ctx := context.Background()

	// Flag setup
	setupSharedFlags()
	importCmd.StringVar(&flagImportName, "name", "", "Name of the asset to import. Valid names are: apns_cert, apns_key, ca_cert, ca_key, abm_key, abm_cert, abm_token, scep_challenge, vpp_token")
	importCmd.StringVar(&flagImportValue, "value", "", "Value of the asset to import")
	exportCmd.StringVar(&flagExportName, "name", "", "Name of the asset to export. Valid names are: apns_cert, apns_key, ca_cert, ca_key, abm_key, abm_cert, abm_token, scep_challenge, vpp_token")
	rolloverCmd.IntVar(&flagExtendYears, "extend-years", 5, "Number of years to extend the Apple MDM CA certificate from now")

	// Execute subcommands
	switch os.Args[1] {
	case "import":
		if err := importCmd.Parse(os.Args[2:]); err != nil {
			log.Fatal("parsing import flags", err)
		}

		if len(flagKey) > 32 {
			// We truncate to 32 bytes because AES-256 requires a 32 byte (256 bit) PK, but some
			// infra setups generate keys that are longer than 32 bytes.
			flagKey = flagKey[:32]
		}

		ds := setupDS(flagKey, flagDBUser, flagDBPass, flagDBAddress, flagDBName)
		defer ds.Close()

		// Check required flags
		if flagDir != "" {
			if err := os.MkdirAll(flagDir, os.ModePerm); err != nil {
				log.Fatal("ensuring directory: ", err) //nolint:gocritic // ignore exitAfterDefer
			}
		}

		if flagImportName == "" {
			log.Fatal("-name flag is required")
		}

		if flagImportValue == "" {
			log.Fatal("-value flag is required")
		}

		if _, ok := validNames[fleet.MDMAssetName(flagImportName)]; !ok {
			log.Fatalf("invalid asset name %s", flagImportName)
		}

		err := ds.ReplaceMDMConfigAssets(ctx,
			[]fleet.MDMConfigAsset{{Name: fleet.MDMAssetName(flagImportName), Value: []byte(flagImportValue)}}, nil)
		if err != nil {
			log.Fatal("writing asset to db: ", err)
		}
		return
	case "export":
		if err := exportCmd.Parse(os.Args[2:]); err != nil {
			log.Fatal("parsing export flags", err)
		}

		// Check required flags
		if flagKey == "" {
			log.Fatal("-key flag is required")
		}

		if len(flagKey) > 32 {
			// We truncate to 32 bytes because AES-256 requires a 32 byte (256 bit) PK, but some
			// infra setups generate keys that are longer than 32 bytes.
			flagKey = flagKey[:32]
		}

		ds := setupDS(flagKey, flagDBUser, flagDBPass, flagDBAddress, flagDBName)
		defer ds.Close()

		if flagDir != "" {
			if err := os.MkdirAll(flagDir, os.ModePerm); err != nil {
				log.Fatal("ensuring directory: ", err)
			}
		}

		names := []fleet.MDMAssetName{
			fleet.MDMAssetCACert,
			fleet.MDMAssetCAKey,
			fleet.MDMAssetAPNSKey,
			fleet.MDMAssetAPNSCert,
			fleet.MDMAssetABMCert,
			fleet.MDMAssetABMKey,
			fleet.MDMAssetABMTokenDeprecated,
			fleet.MDMAssetSCEPChallenge,
			fleet.MDMAssetVPPTokenDeprecated,
		}

		if flagExportName != "" {
			if _, ok := validNames[fleet.MDMAssetName(flagExportName)]; !ok {
				log.Fatalf("invalid asset name %s", flagExportName)
			}

			names = []fleet.MDMAssetName{fleet.MDMAssetName(flagExportName)}
		}

		assets, err := ds.GetAllMDMConfigAssetsByName(ctx, names, nil)
		if err != nil && !errors.Is(err, mysql.ErrPartialResult) {
			log.Fatal("retrieving assets from db:", err)
		}

		for _, asset := range assets {
			path := filepath.Join(flagDir, string(asset.Name))
			switch {
			case strings.Contains(path, "_key"):
				path += ".key"
			case strings.Contains(path, "_cert"):
				path += ".crt"
			}
			if err := os.WriteFile(path, asset.Value, 0o600); err != nil {
				log.Fatal("writing asset:", err)
			}

			log.Printf("wrote %s in %s", asset.Name, path)
		}

		flagDir, err = filepath.Abs(flagDir)
		if err != nil {
			log.Fatalf("abs path: %s", err)
		}

		fmt.Printf(`You can set the following on your Fleet configuration:
export FLEET_MDM_APPLE_APNS_CERT=%[1]s/apns_cert.crt
export FLEET_MDM_APPLE_APNS_KEY=%[1]s/apns_key.key
export FLEET_MDM_APPLE_SCEP_CERT=%[1]s/ca_cert.crt
export FLEET_MDM_APPLE_SCEP_KEY=%[1]s/ca_key.key
export FLEET_MDM_APPLE_SCEP_CHALLENGE=$(cat %[1]s/scep_challenge)
export FLEET_MDM_APPLE_BM_SERVER_TOKEN=%[1]s/abm_token
export FLEET_MDM_APPLE_BM_CERT=%[1]s/abm_cert.crt
export FLEET_MDM_APPLE_BM_KEY=%[1]s/abm_key.key
`, flagDir)
	case "rollover-ca-cert":
		if err := rolloverCmd.Parse(os.Args[2:]); err != nil {
			log.Fatal("parsing rollover-ca-cert flags", err)
		}

		if flagKey == "" {
			log.Fatal("-key flag is required")
		}
		if len(flagKey) > 32 {
			// We truncate to 32 bytes because AES-256 requires a 32 byte (256 bit) PK, but some
			// infra setups generate keys that are longer than 32 bytes.
			flagKey = flagKey[:32]
		}
		if flagExtendYears <= 0 {
			log.Fatal("-extend-years must be a positive integer")
		}

		ds := setupDS(flagKey, flagDBUser, flagDBPass, flagDBAddress, flagDBName)
		defer ds.Close()

		assets, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
			fleet.MDMAssetCACert,
			fleet.MDMAssetCAKey,
		}, nil)
		if err != nil {
			log.Fatal("loading existing apple mdm ca cert and key: ", err) //nolint:gocritic // ignore exitAfterDefer
		}

		caCertAsset, ok := assets[fleet.MDMAssetCACert]
		if !ok {
			log.Fatal("Apple MDM CA certificate not found in database")
		}
		caKeyAsset, ok := assets[fleet.MDMAssetCAKey]
		if !ok {
			log.Fatal("Apple MDM CA private key not found in database")
		}
		oldCertPEM := caCertAsset.Value
		oldKeyPEM := caKeyAsset.Value

		certBlock, _ := pem.Decode(oldCertPEM)
		if certBlock == nil || certBlock.Type != "CERTIFICATE" {
			log.Fatal("decoding existing apple mdm ca certificate PEM")
		}
		oldCert, err := x509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			log.Fatal("parsing existing apple mdm ca certificate: ", err)
		}

		privKeyAny, err := cryptoutil.ParsePrivateKey(oldKeyPEM, "Apple MDM CA private key")
		if err != nil {
			log.Fatal("parsing existing apple mdm ca key: ", err)
		}
		privKey, ok := privKeyAny.(*rsa.PrivateKey)
		if !ok {
			log.Fatal("existing apple mdm ca key is not RSA")
		}
		// Sanity-check that the CA cert and key match.
		certPub, ok := oldCert.PublicKey.(*rsa.PublicKey)
		if !ok {
			log.Fatal("existing apple mdm ca certificate public key is not RSA")
		}
		if certPub.E != privKey.PublicKey.E || certPub.N.Cmp(privKey.PublicKey.N) != 0 {
			log.Fatal("existing apple mdm ca certificate does not match stored private key")
		}
		if ski, err := cryptoutil.GenerateSubjectKeyID(&privKey.PublicKey); err != nil {
			log.Fatal("generating apple mdm ca subject key id: ", err)
		} else if len(oldCert.SubjectKeyId) > 0 {
			if len(oldCert.SubjectKeyId) != len(ski) {
				log.Fatal("existing apple mdm ca certificate SubjectKeyId does not match stored private key")
			}
			for i := range ski {
				if ski[i] != oldCert.SubjectKeyId[i] {
					log.Fatal("existing apple mdm ca certificate SubjectKeyId does not match stored private key")
				}
			}
		}

		// Reserve a fresh serial from identity_serials so the new CA cert
		// cannot collide with any client cert that was (or will be) issued by
		// this CA. The auto-increment value is guaranteed to be greater than
		// every historical client serial, and consuming it here prevents
		// SCEPDepot.Serial() from ever handing it out to a future client cert.
		// We deliberately do not insert into identity_certificates — the CA
		// cert itself lives in mdm_config_assets, not the depot's cert table.
		rawDB, err := sql.Open(
			"mysql",
			fmt.Sprintf("%s:%s@tcp(%s)/%s?tls=skip-verify", flagDBUser, flagDBPass, flagDBAddress, flagDBName),
		)
		if err != nil {
			log.Fatal("opening MySQL connection to reserve CA serial: ", err)
		}
		defer rawDB.Close()
		serialRes, err := rawDB.ExecContext(ctx, `INSERT INTO identity_serials () VALUES ();`)
		if err != nil {
			log.Fatal("allocating new CA cert serial in identity_serials: ", err)
		}
		serialID, err := serialRes.LastInsertId()
		if err != nil {
			log.Fatal("retrieving allocated CA cert serial: ", err)
		}

		// Reuse the existing identity (Subject, SubjectKeyId, key) so that
		// previously-issued client certs continue to chain to the same issuer
		// after the rollover and notBefore means certificates issued before the
		// rollover remain valid. Only the serial number and NotAfter change.
		newSerial := big.NewInt(serialID)
		notBefore := oldCert.NotBefore
		notAfter := oldCert.NotAfter.AddDate(flagExtendYears, 0, 0).UTC()

		tmpl := x509.Certificate{
			Subject:               oldCert.Subject,
			SerialNumber:          newSerial,
			NotBefore:             notBefore,
			NotAfter:              notAfter,
			KeyUsage:              oldCert.KeyUsage,
			BasicConstraintsValid: true,
			IsCA:                  true,
			MaxPathLen:            oldCert.MaxPathLen,
			MaxPathLenZero:        oldCert.MaxPathLenZero,
			SubjectKeyId:          oldCert.SubjectKeyId,
		}

		newCertDER, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, privKey.Public(), privKey)
		if err != nil {
			log.Fatal("creating renewed apple mdm ca certificate: ", err)
		}
		newCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: newCertDER})

		// ReplaceMDMConfigAssets soft-deletes the existing ca_cert row (sets
		// deletion_uuid + deleted_at) and inserts the new cert. The CA private
		// key is untouched so existing chains stay valid.
		if err := ds.ReplaceMDMConfigAssets(ctx, []fleet.MDMConfigAsset{
			{Name: fleet.MDMAssetCACert, Value: newCertPEM},
		}, nil); err != nil {
			log.Fatal("writing renewed apple mdm ca cert to db: ", err)
		}

		log.Printf("Apple MDM CA cert rolled over.")
		log.Printf("  common name:       %s", oldCert.Subject.CommonName)
		log.Printf("  previous NotAfter: %s", oldCert.NotAfter.Format(time.RFC3339))
		log.Printf("  new NotAfter:      %s", notAfter.Format(time.RFC3339))
		log.Printf("  new serial:        %s", newSerial.String())
		return
	default:
		log.Fatalf("invalid subcommand %s, valid subcommands: import, export, rollover-ca-cert", os.Args[1]) //nolint:gosec // dismiss G107
	}
}
