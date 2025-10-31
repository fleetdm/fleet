// Command applebmapi takes selected Fleet server configuration information and calls the Apple BM
// API to retrieve DEP records for the specified organization, serial number, or profile UUID.
//
// Was implemented to test out https://github.com/fleetdm/fleet/issues/7515#issuecomment-1330889768,
// and can still be useful for debugging purposes.
//
// Usage:
// $ go run ./tools/mdm/apple/applebmapi/main.go -h
package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	mysqlAddr := flag.String("mysql", "", "mysql address")
	mysqlUser := flag.String("mysql-user", "", "mysql user")
	mysqlPass := flag.String("mysql-pw", "", "mysql password (optional, omit this flag to be prompted to enter in the terminal)")
	serverPrivateKey := flag.String("server-private-key", "", "Fleet server's private key to decrypt MDM assets (optional, omit this flag to be prompted to enter in the terminal)")
	profileUUID := flag.String("profile-uuid", "", "Apple profile UUID to retrieve")
	serialNum := flag.String("serial-number", "", "serial number of a device to get the device details")
	orgName := flag.String("org-name", "", "organization name associated with the token used to access Apple Business Manager (e.g., 'Fleet Device Management Inc.')")
	serials := flag.String("serials", "", "comma separated list of serial numbers to check profile assignment")
	pathToSerials := flag.String("path-to-serials", "", "path to a file containing a comma separated list of serial numbers to check profile assignment")

	flag.Parse()

	if *orgName == "" {
		log.Fatal("must provide -org-name")
	}
	if *profileUUID != "" && *serialNum != "" {
		log.Fatal("only one of -profile-uuid or -serial-number must be provided")
	}

	if *mysqlUser == "" {
		fmt.Println("mysql-user was empty, using the default of \"fleet\"")
		mysqlUser = ptr.String("fleet")
	}
	if *mysqlAddr == "" {
		fmt.Println("mysql address was empty, using the default of \"localhost:3306\"")
		mysqlAddr = ptr.String("localhost:3306")
	}

	if *mysqlPass == "" {
		fmt.Print("Password: ")
		pb, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			log.Fatalf("must provide password: %s", err)
		}
		fmt.Println()
		mysqlPass = ptr.String(string(pb))
	}

	if *serverPrivateKey == "" {
		fmt.Print("Server private key: ")
		kb, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			log.Fatalf("must provide key: %s", err)
		}
		fmt.Println()
		serverPrivateKey = ptr.String(string(kb))
	}

	if len(*serverPrivateKey) > 32 {
		// We truncate to 32 bytes because AES-256 requires a 32 byte (256 bit) PK, but some
		// infra setups generate keys that are longer than 32 bytes.
		truncatedServerPrivateKey := (*serverPrivateKey)[:32]
		serverPrivateKey = &truncatedServerPrivateKey
	}

	cfg := config.MysqlConfig{
		Protocol:        "tcp",
		Address:         *mysqlAddr,
		Database:        "fleet",
		Username:        *mysqlUser,
		Password:        *mysqlPass,
		MaxOpenConns:    50,
		MaxIdleConns:    50,
		ConnMaxLifetime: 0,
	}
	logger := kitlog.NewLogfmtLogger(os.Stderr)
	opts := []mysql.DBOption{
		mysql.Logger(logger),
		mysql.WithFleetConfig(&config.FleetConfig{
			Server: config.ServerConfig{
				PrivateKey: *serverPrivateKey,
			},
		}),
	}
	mds, err := mysql.New(cfg, clock.C, opts...)
	if err != nil {
		log.Fatal(err)
	}

	depStorage, err := mds.NewMDMAppleDEPStorage()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	var res any

	if *serials != "" || *pathToSerials != "" {
		var parts []string
		devices := make([]*godep.Device, 0)

		if *pathToSerials != "" {
			file, err := os.Open(*pathToSerials)
			if err != nil {
				log.Fatal("open file", err)
			}
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				serial := strings.TrimSpace(line)
				if serial == "" {
					continue
				}
				parts = append(parts, serial)
			}
			if err := scanner.Err(); err != nil {
				log.Fatal("read file", err)
			}
		} else {
			parts = strings.Split(*serials, ",")
			fmt.Println("Serials: ", parts)
		}

		if len(parts) == 0 {
			log.Fatal("no serial numbers provided")
		}

		fmt.Printf("Getting %d devices...\n", len(parts))
		devices, err = getDevices(ctx, depStorage, *orgName, parts)
		if err != nil {
			log.Fatal("get devices", err)
		}
		fmt.Printf("Got %d devices\n", len(devices))

		if err := writeDevicesToCSV(devices); err != nil {
			log.Fatal("write devices to csv", err)
		}

	} else {
		depClient := godep.NewClient(depStorage, fleethttp.NewClient())

		switch {
		case *profileUUID != "":
			res, err = depClient.GetProfile(ctx, *orgName, *profileUUID)
		case *serialNum != "":
			res, err = depClient.GetDeviceDetails(ctx, *orgName, *serialNum)
		case *serials != "":
			for _, serial := range strings.Split(*serials, ",") {
				serial = strings.TrimSpace(serial)
				if serial == "" {
					continue
				}
				res, err = depClient.GetDeviceDetails(ctx, *orgName, *serialNum)
			}
		default:
			res, err = depClient.AccountDetail(ctx, *orgName)
		}
		if err != nil {
			log.Fatal(err)
		}

		b, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			log.Fatalf("pretty-format body: %s", err)
		}
		fmt.Printf("body: \n%s\n", string(b))
	}
}

func getDevices(ctx context.Context, ds *mysql.NanoDEPStorage, orgName string, serials []string) ([]*godep.Device, error) {
	devices := make([]*godep.Device, 0)

	for i, serial := range serials {
		s := strings.TrimSpace(serial)
		if s == "" {
			continue
		}
		fmt.Printf("Getting device %d of %d... %s\n", i+1, len(serials), serial)

		depClient := godep.NewClient(ds, fleethttp.NewClient())

		r, err := depClient.GetDeviceDetails(ctx, orgName, s)
		if err != nil {
			log.Fatal("get device", err)
		}
		devices = append(devices, r)

		time.Sleep(500 * time.Millisecond) // extra precaution to avoid hitting the API rate limit, adjust as needed
	}

	return devices, nil
}

func writeDevicesToCSV(devices []*godep.Device) error {
	file, err := os.Create(fmt.Sprintf("%s__devices.csv", time.Now().Format(time.RFC3339)))
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// write headers
	err = writer.Write([]string{
		"serial_number",
		"model",
		"description",
		"color",
		"asset_tag",
		"profile_status",
		"profile_uuid",
		"profile_assign_time",
		"profile_push_time",
		"device_assigned_date",
		"device_assigned_by",
		"os",
		"device_family",
	})
	if err != nil {
		log.Fatal("write headers", err)
	}

	for i, device := range devices {
		fmt.Printf("Writing device %d of %d... %s\n", i+1, len(devices), device.SerialNumber)
		err := writer.Write([]string{
			device.SerialNumber,
			device.Model,
			device.Description,
			device.Color,
			device.AssetTag,
			device.ProfileStatus,
			device.ProfileUUID,
			device.ProfileAssignTime.Format(time.RFC3339),
			device.ProfilePushTime.Format(time.RFC3339),
			device.DeviceAssignedDate.Format(time.RFC3339),
			device.DeviceAssignedBy,
			device.OS,
			device.DeviceFamily,
		})
		if err != nil {
			log.Fatal("write device", err)
		}
	}
	fmt.Println("Devices saved to devices.csv")

	return nil
}
