package main

import (
	"context"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/boltdb/bolt"
	boltdepot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot/bolt"
	"github.com/groob/plist"
	apnsbuiltin "github.com/micromdm/micromdm/platform/apns/builtin"
	"github.com/micromdm/micromdm/platform/device"
	devicebuiltin "github.com/micromdm/micromdm/platform/device/builtin"
	"github.com/micromdm/micromdm/platform/pubsub/inmem"
	"go.etcd.io/bbolt"
)

type Authenticate struct {
	MessageType  string
	UDID         string
	Topic        string
	BuildVersion string `plist:",omitempty"`
	DeviceName   string `plist:",omitempty"`
	Model        string `plist:",omitempty"`
	ModelName    string `plist:",omitempty"`
	OSVersion    string `plist:",omitempty"`
	ProductName  string `plist:",omitempty"`
	SerialNumber string `plist:",omitempty"`
	IMEI         string `plist:",omitempty"`
	MEID         string `plist:",omitempty"`
}

type TokenUpdate struct {
	MessageType   string
	UDID          string
	PushMagic     string
	Topic         string
	Token         []byte
	UnlockToken   []byte `plist:",omitempty"`
	UserID        string `plist:",omitempty"`
	UserShortName string `plist:",omitempty"`
	UserLongName  string `plist:",omitempty"`
}

func main() {
	flDB := flag.String("db", "/var/db/micromdm.db", "path to micromdm DB")
	flag.Parse()

	bboltDB, err := bbolt.Open(*flDB, 0o600, nil)
	if err != nil {
		log.Fatal(err)
	}

	svcBoltDepot, err := boltdepot.NewBoltDepot(bboltDB)
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
	if err := bboltDB.Close(); err != nil {
		log.Fatal(err)
	}

	boltDB, err := bolt.Open(*flDB, 0o600, nil)
	if err != nil {
		log.Fatal(err)
	}

	ps := inmem.NewPubSub()

	apnsDB, err := apnsbuiltin.NewDB(boltDB, ps)
	if err != nil {
		log.Fatal(err)
	}

	deviceDB, err := devicebuiltin.NewDB(boltDB)
	if err != nil {
		log.Fatal(err)
	}
	devices, err := deviceDB.List(context.Background(), device.ListDevicesOption{})
	if err != nil {
		log.Fatal(err)
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(key)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	if err := os.WriteFile("scep.key", privateKeyPEM, 0o777); err != nil {
		log.Fatal(err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: crt.Raw,
	})

	if err := os.WriteFile("scep.cert", certPEM, 0o777); err != nil {
		log.Fatal(err)
	}

	var sb strings.Builder

	for _, device := range devices {
		pushInfo, err := apnsDB.PushInfo(context.Background(), device.UDID)
		if err != nil {
			log.Println(err)
			continue
		}
		authenticate := &Authenticate{
			MessageType:  "Authenticate",
			UDID:         device.UDID,
			Topic:        pushInfo.MDMTopic,
			BuildVersion: device.BuildVersion,
			DeviceName:   device.DeviceName,
			Model:        device.Model,
			ModelName:    device.ModelName,
			OSVersion:    device.OSVersion,
			ProductName:  device.ProductName,
			SerialNumber: device.SerialNumber,
			IMEI:         device.IMEI,
			MEID:         device.MEID,
		}
		authenticatePlist, err := plist.Marshal(authenticate)
		if err != nil {
			log.Println(err)
			continue
		}
		fmt.Println(authenticatePlist)

		token, err := hex.DecodeString(pushInfo.Token)
		if err != nil {
			log.Println(err)
			continue
		}
		unlockToken, err := hex.DecodeString(device.UnlockToken)
		if err != nil {
			log.Println(err)
			continue
		}

		tokenUpdate := &TokenUpdate{
			MessageType: "TokenUpdate",
			UDID:        device.UDID,

			PushMagic: pushInfo.PushMagic,
			Token:     token,
			Topic:     pushInfo.MDMTopic,

			UnlockToken: unlockToken,
		}
		tokenPlist, err := plist.Marshal(tokenUpdate)
		fmt.Println(tokenPlist)
		if err != nil {
			log.Println(err)
			continue
		}

		certHash, err := deviceDB.GetUDIDCertHash([]byte(device.UDID))
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("================", string(certHash))

		sb.WriteString(fmt.Sprintf(`
INSERT INTO nano_devices
    (id, identity_cert, serial_number, authenticate, authenticate_at, token_update, token_update_at)
VALUES
    ('%s', HEX('%s'), '%s', '%s', CURRENT_TIMESTAMP, '%s', CURRENT_TIMESTAMP)
ON DUPLICATE KEY
UPDATE
    identity_cert = VALUES(identity_cert),
    serial_number = VALUES(serial_number),
    authenticate = VALUES(authenticate),
    authenticate_at = CURRENT_TIMESTAMP,
    token_update = VALUES(token_update),
    token_update_at = CURRENT_TIMESTAMP;
		`, device.UDID, hex.EncodeToString(certHash[:]), device.SerialNumber, "", ""))

		sb.WriteString(fmt.Sprintf(`
INSERT INTO nano_enrollments
	(id, device_id, user_id, type, topic, push_magic, token_hex, last_seen_at, token_update_tally)
VALUES
	('%s', '%s', NULL, "Device", '%s', '%s', '%s', CURRENT_TIMESTAMP, 1)
ON DUPLICATE KEY
UPDATE
    device_id = VALUES(device_id),
    user_id = VALUES(user_id),
    type = VALUES(type),
    topic = VALUES(topic),
    push_magic = VALUES(push_magic),
    token_hex = VALUES(token_hex),
    enabled = 1,
    last_seen_at = CURRENT_TIMESTAMP,
    token_update_tally = nano_enrollments.token_update_tally + 1;`,
			device.UDID,
			device.UDID,
			tokenUpdate.Topic,
			tokenUpdate.PushMagic,
			hex.EncodeToString(tokenUpdate.Token),
		))
	}

	if err := os.WriteFile("dump.sql", []byte(sb.String()), 0o777); err != nil {
		log.Fatal(err)
	}
}
