package main

import (
	"context"
	"encoding/csv"
	"flag"
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

var (
	// MySQL config
	mysqlAddr = "localhost:3306"
	mysqlUser = "fleet"
	mysqlPass = "insecure"
	mysqlDB   = "fleet"

	// CSV paths
	macCSVPath = "./tools/software/vulnerabilities/software-macos.csv"
	winCSVPath = "./tools/software/vulnerabilities/software-win.csv"
)

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
	for _, line := range lines[1:] { // Skip header
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

func createOrGetHost(ctx context.Context, ds *mysql.Datastore, identifier string, base fleet.Host) (*fleet.Host, error) {
	h, err := ds.HostByIdentifier(ctx, identifier)
	if err == nil && h != nil {
		return h, nil
	}
	base.UUID = identifier
	base.Hostname = identifier
	base.NodeKey = ptr.String(identifier)
	return ds.NewHost(ctx, &base)
}

func main() {
	// Flags
	var (
		ubuntuCount  = flag.Int("ubuntu", 0, "Number of Ubuntu hosts to create (default 0)")
		macosCount   = flag.Int("macos", 0, "Number of macOS hosts to create (default 0)")
		windowsCount = flag.Int("windows", 0, "Number of Windows hosts to create (default 0)")
		linuxKernels = flag.Int("linux-kernels", 0, "Number of Linux kernels to add per Ubuntu host (default 0)")
	)
	flag.Parse()

	ctx := context.Background()

	// Datastore
	ds, err := mysql.New(config.MysqlConfig{
		Protocol: "tcp",
		Address:  mysqlAddr,
		Username: mysqlUser,
		Password: mysqlPass,
		Database: mysqlDB,
	}, clock.C)
	if err != nil {
		log.Fatal(err)
	}
	defer ds.Close()

	now := time.Now()

	// macOS hosts
	var macHosts []*fleet.Host
	if *macosCount > 0 {
		fmt.Printf("Creating %d macOS host(s)…\n", *macosCount)
		for i := 1; i <= *macosCount; i++ {
			name := "macos-seed-host"
			if *macosCount > 1 {
				name = fmt.Sprintf("macos-seed-host-%d", i)
			}
			host, err := createOrGetHost(ctx, ds, name, fleet.Host{
				DetailUpdatedAt: now,
				LabelUpdatedAt:  now,
				PolicyUpdatedAt: now,
				SeenTime:        now,
				PrimaryIP:       "192.168.1.100",
				PrimaryMac:      "00:11:22:33:44:55",
				Platform:        "darwin",
				OSVersion:       "Mac OS X 10.14.6",
			})
			if err != nil {
				fmt.Printf("create macos host failed (%s): %v\n", name, err)
				continue
			}
			macHosts = append(macHosts, host)
		}
	}

	// Windows hosts
	var winHosts []*fleet.Host
	if *windowsCount > 0 {
		fmt.Printf("Creating %d Windows host(s)…\n", *windowsCount)
		for i := 1; i <= *windowsCount; i++ {
			name := "windows-seed-host"
			if *windowsCount > 1 {
				name = fmt.Sprintf("windows-seed-host-%d", i)
			}
			host, err := createOrGetHost(ctx, ds, name, fleet.Host{
				DetailUpdatedAt: now,
				LabelUpdatedAt:  now,
				PolicyUpdatedAt: now,
				SeenTime:        now,
				PrimaryIP:       "192.168.1.101",
				PrimaryMac:      "00:11:22:33:44:56",
				Platform:        "windows",
				OSVersion:       "Windows 11 Enterprise",
			})
			if err != nil {
				fmt.Printf("create windows host failed (%s): %v\n", name, err)
				continue
			}
			winHosts = append(winHosts, host)
		}
	}

	// Ubuntu hosts
	var ubuntuIDs []uint
	if *ubuntuCount > 0 {
		fmt.Printf("Creating %d Ubuntu host(s)…\n", *ubuntuCount)
		for i := 1; i <= *ubuntuCount; i++ {
			name := fmt.Sprintf("ubuntu-seed-host-%d", i)
			host, err := createOrGetHost(ctx, ds, name, fleet.Host{
				DetailUpdatedAt: now,
				LabelUpdatedAt:  now,
				PolicyUpdatedAt: now,
				SeenTime:        now,
				PrimaryIP:       fmt.Sprintf("192.168.1.%d", i+1),
				PrimaryMac:      fmt.Sprintf("00:11:22:33:44:%02d", i+1),
				Platform:        "ubuntu",
				OSVersion:       "Ubuntu 20.04.6 LTS",
			})
			if err != nil {
				fmt.Printf("create ubuntu host failed (%s): %v\n", name, err)
				continue
			}
			if err := ds.UpdateHostOperatingSystem(ctx, host.ID, fleet.OperatingSystem{
				Name:           "Ubuntu",
				Version:        fmt.Sprintf("20.04.%d", i),
				Platform:       "ubuntu",
				Arch:           "x86_64",
				KernelVersion:  "5.4.0-148-generic",
				DisplayVersion: "20.04",
			}); err != nil {
				fmt.Printf("update ubuntu host OS failed (%s): %v\n", name, err)
				continue
			}
			ubuntuIDs = append(ubuntuIDs, host.ID)
		}
	}

	// macOS software
	if len(macHosts) > 0 {
		macSoftware, err := readCSVFile(macCSVPath)
		if err != nil {
			fmt.Printf("read macOS software csv failed: %v\n", err)
		} else {
			for _, h := range macHosts {
				if _, err := ds.UpdateHostSoftware(ctx, h.ID, macSoftware); err != nil {
					fmt.Printf("update macOS host software failed (%s): %v\n", h.Hostname, err)
				}
			}
		}
	}

	// Windows software
	if len(winHosts) > 0 {
		winSoftware, err := readCSVFile(winCSVPath)
		if err != nil {
			fmt.Printf("read Windows software csv failed: %v\n", err)
		} else {
			for _, h := range winHosts {
				if _, err := ds.UpdateHostSoftware(ctx, h.ID, winSoftware); err != nil {
					fmt.Printf("update Windows host software failed (%s): %v\n", h.Hostname, err)
				}
			}
		}
	}

	// Linux kernels for Ubuntu
	if *linuxKernels > 0 && len(ubuntuIDs) > 0 {
		fmt.Printf("Adding %d Linux kernel package(s) per Ubuntu host…\n", *linuxKernels)
		for _, ubuntuID := range ubuntuIDs {
			var pkgs []fleet.Software
			for k := 1; k <= *linuxKernels; k++ {
				pkgs = append(pkgs, fleet.Software{
					Name:     fmt.Sprintf("linux-image-6.8.0-%d-generic", k),
					Version:  fmt.Sprintf("6.8.0-%d", k),
					Source:   "Package (deb)",
					IsKernel: true,
				})
			}
			if _, err := ds.UpdateHostSoftware(ctx, ubuntuID, pkgs); err != nil {
				fmt.Printf("insert kernel software for Ubuntu host %d failed: %v\n", ubuntuID, err)
			}
		}
	}

	fmt.Println("Done.")
}
