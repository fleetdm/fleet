package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math"
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
	macCSVPath    = "./tools/software/vulnerabilities/software-macos.csv"
	winCSVPath    = "./tools/software/vulnerabilities/software-win.csv"
	ubuntuCSVPath = "./tools/software/vulnerabilities/software-ubuntu.csv"
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

func calculateKernelCount(hostIndex, totalHosts, maxKernels int) int {
	const minKernels = 1

	if maxKernels < minKernels {
		maxKernels = minKernels
	}

	if totalHosts == 1 {
		return maxKernels
	}

	// Use exponential decay to distribute kernel counts
	// The decay factor is calculated to spread from maxKernels to 1 across all hosts
	decayFactor := math.Log(float64(maxKernels)/float64(minKernels)) / float64(totalHosts-1)
	kernelCount := int(math.Round(float64(maxKernels) * math.Exp(-decayFactor*float64(hostIndex-1))))

	// Ensure we stay within bounds
	if kernelCount < minKernels {
		kernelCount = minKernels
	}
	if kernelCount > maxKernels {
		kernelCount = maxKernels
	}

	return kernelCount
}

type ubuntuVersion struct {
	version        string
	displayVersion string
	kernelVersion  string
	osVersion      string
}

func generateUbuntuVersions(count int) []ubuntuVersion {
	var versions []ubuntuVersion

	// Ubuntu 20.04 series (6 point releases)
	for i := 1; i <= 6 && len(versions) < count; i++ {
		versions = append(versions, ubuntuVersion{
			version:        fmt.Sprintf("20.04.%d", i),
			displayVersion: "20.04",
			kernelVersion:  fmt.Sprintf("5.4.0-%d-generic", 148+(i-1)*2),
			osVersion:      fmt.Sprintf("Ubuntu 20.04.%d LTS", i),
		})
	}

	// Ubuntu 22.04 series (6 point releases)
	for i := 1; i <= 6 && len(versions) < count; i++ {
		versions = append(versions, ubuntuVersion{
			version:        fmt.Sprintf("22.04.%d", i),
			displayVersion: "22.04",
			kernelVersion:  fmt.Sprintf("5.15.0-%d-generic", 56+(i-1)*2),
			osVersion:      fmt.Sprintf("Ubuntu 22.04.%d LTS", i),
		})
	}

	// Ubuntu 24.04 series (6 point releases)
	for i := 1; i <= 6 && len(versions) < count; i++ {
		versions = append(versions, ubuntuVersion{
			version:        fmt.Sprintf("24.04.%d", i),
			displayVersion: "24.04",
			kernelVersion:  fmt.Sprintf("6.8.0-%d-generic", 31+(i-1)*2),
			osVersion:      fmt.Sprintf("Ubuntu 24.04.%d LTS", i),
		})
	}

	// Ubuntu 26.04 series (if we need more)
	for i := 1; i <= 6 && len(versions) < count; i++ {
		versions = append(versions, ubuntuVersion{
			version:        fmt.Sprintf("26.04.%d", i),
			displayVersion: "26.04",
			kernelVersion:  fmt.Sprintf("6.12.0-%d-generic", 20+(i-1)*2),
			osVersion:      fmt.Sprintf("Ubuntu 26.04.%d LTS", i),
		})
	}

	// If we still need more, continue with future versions
	currentMajor := 28
	for len(versions) < count {
		for i := 1; i <= 6 && len(versions) < count; i++ {
			versions = append(versions, ubuntuVersion{
				version:        fmt.Sprintf("%d.04.%d", currentMajor, i),
				displayVersion: fmt.Sprintf("%d.04", currentMajor),
				kernelVersion:  fmt.Sprintf("6.%d.0-%d-generic", 14+currentMajor-28, 20+(i-1)*2),
				osVersion:      fmt.Sprintf("Ubuntu %d.04.%d LTS", currentMajor, i),
			})
		}
		currentMajor += 2
	}

	return versions
}

func main() {
	// Flags
	var (
		ubuntuCount    = flag.Int("ubuntu", 0, "Number of Ubuntu hosts to create (default 0)")
		macosCount     = flag.Int("macos", 0, "Number of macOS hosts to create (default 0)")
		windowsCount   = flag.Int("windows", 0, "Number of Windows hosts to create (default 0)")
		linuxKernels   = flag.Int("linux-kernels", 0, "Maximum Linux kernels per Ubuntu host, enables variable distribution (default 0)")
		ubuntuVersions = flag.Int("ubuntu-versions", 10, "Number of different Ubuntu OS versions to use (default 10)")
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

		// Generate Ubuntu OS versions based on the requested count
		maxVersions := *ubuntuVersions
		if maxVersions < 1 {
			maxVersions = 1
		}
		osVersions := generateUbuntuVersions(maxVersions)

		for i := 1; i <= *ubuntuCount; i++ {
			name := fmt.Sprintf("ubuntu-seed-host-%d", i)

			// Distribute hosts evenly across OS versions
			osIndex := (i - 1) % len(osVersions)
			selectedOS := osVersions[osIndex]

			host, err := createOrGetHost(ctx, ds, name, fleet.Host{
				DetailUpdatedAt: now,
				LabelUpdatedAt:  now,
				PolicyUpdatedAt: now,
				SeenTime:        now,
				PrimaryIP:       fmt.Sprintf("192.168.1.%d", i+1),
				PrimaryMac:      fmt.Sprintf("00:11:22:33:44:%02d", i+1),
				Platform:        "ubuntu",
				OSVersion:       selectedOS.osVersion,
			})
			if err != nil {
				fmt.Printf("create ubuntu host failed (%s): %v\n", name, err)
				continue
			}
			if err := ds.UpdateHostOperatingSystem(ctx, host.ID, fleet.OperatingSystem{
				Name:           "Ubuntu",
				Version:        selectedOS.version,
				Platform:       "ubuntu",
				Arch:           "x86_64",
				KernelVersion:  selectedOS.kernelVersion,
				DisplayVersion: selectedOS.displayVersion,
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

	// Ubuntu software
	if len(ubuntuIDs) > 0 {
		ubuntuSoftware, err := readCSVFile(ubuntuCSVPath)
		if err != nil {
			fmt.Printf("read Ubuntu software csv failed: %v\n", err)
		} else {
			for _, ubuntuID := range ubuntuIDs {
				if _, err := ds.UpdateHostSoftware(ctx, ubuntuID, ubuntuSoftware); err != nil {
					fmt.Printf("update Ubuntu host software failed (%d): %v\n", ubuntuID, err)
				}
			}
		}
	}

	// Linux kernels for Ubuntu with variable distribution
	if *linuxKernels > 0 && len(ubuntuIDs) > 0 {
		fmt.Printf("Adding variable Linux kernel packages per Ubuntu host (max %d)…\n", *linuxKernels)
		for i, ubuntuID := range ubuntuIDs {
			kernelCount := calculateKernelCount(i+1, len(ubuntuIDs), *linuxKernels)
			var pkgs []fleet.Software
			for k := 1; k <= kernelCount; k++ {
				pkgs = append(pkgs, fleet.Software{
					Name:     fmt.Sprintf("linux-image-6.8.0-%d-generic", k),
					Version:  fmt.Sprintf("6.8.0-%d", k),
					Source:   "Package (deb)",
					IsKernel: true,
				})
			}
			if _, err := ds.UpdateHostSoftware(ctx, ubuntuID, pkgs); err != nil {
				fmt.Printf("insert kernel software for Ubuntu host %d failed: %v\n", ubuntuID, err)
			} else {
				fmt.Printf("Added %d kernels to Ubuntu host %d\n", kernelCount, ubuntuID)
			}
		}
	}

	fmt.Println("Done.")
}
