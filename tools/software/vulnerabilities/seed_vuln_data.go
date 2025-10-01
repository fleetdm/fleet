package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/WatchBeam/clock"
	_ "github.com/go-sql-driver/mysql"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

const linuxHosts = 800

func readCSVFile(filePath string) ([]fleet.Software, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var software []fleet.Software
	for _, line := range lines[1:] { // Skip header line
		software = append(software, fleet.Software{
			Name:             line[0],
			Version:          line[1],
			Source:           line[2],
			BundleIdentifier: line[3],
			Release:          line[4],
			VendorOld:        line[5],
			Arch:             line[6],
			Vendor:           line[7],
		})
	}
	return software, nil
}

func main() {
	ds, err := mysql.New(config.MysqlConfig{
		Protocol: "tcp",
		Address:  "localhost:3306",
		Username: "fleet",
		Password: "insecure",
		Database: "fleet",
	}, clock.C)
	if err != nil {
		log.Fatal(err)
	}
	defer ds.Close()

	// macos Host
	fmt.Print("Creating macOS host...\n")
	var macosHost *fleet.Host
	// get host if it already exists
	macosHost, err = ds.HostByIdentifier(context.Background(), "macos-seed-host")
	if err != nil {
		fmt.Printf("get macos host failed: %v\n", err)

		macosHost, err = ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         ptr.String("macos-seed-host"),
			UUID:            "macos-seed-host",
			Hostname:        "macos-seed-host",
			PrimaryIP:       "192.168.1.100",
			PrimaryMac:      "00:11:22:33:44:55",
			Platform:        "darwin",
			OSVersion:       "Mac OS X 10.14.6",
		})
		if err != nil {
			fmt.Printf("create macos host failed: %v\n", err)
		}
	}

	// windows Host
	fmt.Print("Creating Windows host...\n")

	// get host if it already exists
	var winHost *fleet.Host
	winHost, err = ds.HostByIdentifier(context.Background(), "windows-seed-host")
	if err != nil {
		fmt.Printf("get windows host failed: %v\n", err)

		winHost, err = ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         ptr.String("windows-seed-host"),
			UUID:            "windows-seed-host",
			Hostname:        "windows-seed-host",
			PrimaryIP:       "192.168.1.101",
			PrimaryMac:      "00:11:22:33:44:56",
			Platform:        "windows",
			OSVersion:       "Windows 11 Enterprise",
		})
		if err != nil {
			fmt.Printf("create windows host failed: %v\n", err)
		}
	}

	fmt.Print("Creating Ubuntu hosts...\n")
	var ubuntuIDs []uint
	// ubuntu Hosts
	for i := 0; i < linuxHosts; i++ {
		var host *fleet.Host
		host, err = ds.HostByIdentifier(context.Background(), fmt.Sprintf("ubuntu-seed-host-%d", len(ubuntuIDs)+1))
		if err != nil {
			host, err = ds.NewHost(context.Background(), &fleet.Host{
				DetailUpdatedAt: time.Now(),
				LabelUpdatedAt:  time.Now(),
				PolicyUpdatedAt: time.Now(),
				SeenTime:        time.Now(),
				NodeKey:         ptr.String(fmt.Sprintf("ubuntu-seed-host-%d", len(ubuntuIDs)+1)),
				UUID:            fmt.Sprintf("ubuntu-seed-host-%d", len(ubuntuIDs)+1),
				Hostname:        fmt.Sprintf("ubuntu-seed-host-%d", len(ubuntuIDs)+1),
				PrimaryIP:       fmt.Sprintf("192.168.1.%d", len(ubuntuIDs)+2),
				PrimaryMac:      fmt.Sprintf("00:11:22:33:44:%02d", len(ubuntuIDs)+2),
				Platform:        "ubuntu",
				OSVersion:       "Ubuntu 20.04.6 LTS",
			})
			if err != nil {
				fmt.Printf("create ubuntu host failed: %v\n", err)
				continue
			}
		}

		err = ds.UpdateHostOperatingSystem(context.Background(), host.ID, fleet.OperatingSystem{
			Name:           "Ubuntu",
			Version:        fmt.Sprintf("20.04.%d", i),
			Platform:       "ubuntu",
			Arch:           "x86_64",
			KernelVersion:  "5.4.0-148-generic",
			DisplayVersion: "20.04",
		})
		if err != nil {
			fmt.Printf("update ubuntu host OS failed: %v\n", err)
			continue
		}

		ubuntuIDs = append(ubuntuIDs, host.ID)
	}

	// Insert macOS software
	macSoftware, err := readCSVFile("./tools/software/vulnerabilities/software-macos.csv")
	if err != nil {
		fmt.Printf("read macos software csv file failed: %v\n", err)
		return
	}

	fmt.Printf("%v\n", macSoftware)
	_, err = ds.UpdateHostSoftware(context.Background(), macosHost.ID, macSoftware)
	if err != nil {
		fmt.Printf("update macos host software failed: %v\n", err)
		return
	}

	// Insert win software
	winSoftware, err := readCSVFile("./tools/software/vulnerabilities/software-win.csv")
	if err != nil {
		fmt.Printf("read windows software csv file failed: %v\n", err)
		return
	}
	_, err = ds.UpdateHostSoftware(context.Background(), winHost.ID, winSoftware)
	if err != nil {
		fmt.Printf("update windows host software failed: %v\n", err)
		return
	}

	for i, ubuntuID := range ubuntuIDs {
		_, err := ds.UpdateHostSoftware(context.Background(), ubuntuID, []fleet.Software{
			{
				Name:     fmt.Sprintf("linux-image-6.8.0-%d-generic", i+1),
				Version:  fmt.Sprintf("6.8.0-%d", i+1),
				Source:   "Package (deb)",
				IsKernel: true,
			},
		})
		if err != nil {
			fmt.Printf("insert software for Ubuntu host failed: %v\n", err)
		}
	}
}
