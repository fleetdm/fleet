package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/WatchBeam/clock"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func retryOnDeadlock(operation func() error, maxRetries int) error {
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err = operation()
		if err == nil {
			return nil
		}

		// Check if it's a deadlock error
		if strings.Contains(err.Error(), "Deadlock found") || strings.Contains(err.Error(), "1213") {
			if attempt < maxRetries {
				// Exponential backoff with jitter
				// #nosec G404 - weak random is acceptable for retry backoff
				backoff := time.Duration(10+rand.Intn(50)) * time.Millisecond * time.Duration(1<<attempt)
				time.Sleep(backoff)
				continue
			}
		}
		break
	}
	return err
}

func timeStep(name string, verbose bool, fn func() error) error {
	if verbose {
		fmt.Printf("Starting: %s...\n", name)
	}
	start := time.Now()
	err := fn()
	duration := time.Since(start)
	if verbose {
		fmt.Printf("Completed: %s in %v\n", name, duration)
	} else {
		fmt.Printf("%s: %v\n", name, duration)
	}
	return err
}

var (
	// MySQL config
	mysqlAddr = "localhost:3306"
	mysqlUser = "fleet"
	mysqlPass = "insecure"
	mysqlDB   = "fleet"
)

// Common CVE patterns for realistic data
var cvePatterns = []string{
	"CVE-2024-%04d", "CVE-2023-%04d", "CVE-2022-%04d", "CVE-2021-%04d",
}

// batchCreateHosts creates multiple hosts in batches for better performance
func batchCreateHosts(ctx context.Context, ds *mysql.Datastore, hostCount int, teams []*fleet.Team, verbose bool) ([]*fleet.Host, error) {
	batchSize := 100 // Insert 100 hosts per transaction
	var allHosts []*fleet.Host
	now := time.Now()

	db, err := getDB(ds)
	if err != nil {
		return nil, fmt.Errorf("get DB connection: %w", err)
	}
	defer db.Close()

	for batchStart := 0; batchStart < hostCount; batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > hostCount {
			batchEnd = hostCount
		}

		// Prepare batch insert
		var args []interface{}
		var placeholders []string

		for i := batchStart; i < batchEnd; i++ {
			var teamID *uint
			if len(teams) > 0 && i%3 != 0 { // 2/3 of hosts have teams
				teamID = &teams[i%len(teams)].ID
			}

			identifier := fmt.Sprintf("test-host-%d", i)

			osqueryHostID := fmt.Sprintf("osquery-host-%d", i)
			placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
			args = append(args,
				osqueryHostID,        // osquery_host_id
				now,                  // detail_updated_at
				now,                  // label_updated_at
				now,                  // policy_updated_at
				identifier,           // node_key
				identifier,           // hostname
				identifier,           // computer_name
				identifier,           // uuid
				"ubuntu",             // platform
				"",                   // platform_like
				"",                   // osquery_version
				"Ubuntu 20.04.6 LTS", // os_version
				0,                    // uptime
				0,                    // memory
				teamID,               // team_id
				0,                    // distributed_interval
				0,                    // logger_tls_period
				0,                    // config_tls_refresh
				false,                // refetch_requested
				"",                   // hardware_serial
				nil,                  // refetch_critical_queries_until (can be NULL)
			)
		}

		// Execute batch insert (exactly matching the NewHost function)
		sql := `INSERT INTO hosts (
			osquery_host_id, detail_updated_at, label_updated_at, policy_updated_at,
			node_key, hostname, computer_name, uuid, platform, platform_like,
			osquery_version, os_version, uptime, memory, team_id,
			distributed_interval, logger_tls_period, config_tls_refresh,
			refetch_requested, hardware_serial, refetch_critical_queries_until
		) VALUES ` + strings.Join(placeholders, ", ")

		_, err := db.ExecContext(ctx, sql, args...)
		if err != nil {
			return nil, fmt.Errorf("batch insert hosts %d-%d: %w", batchStart, batchEnd-1, err)
		}

		// Fetch the created hosts to get their IDs
		var batchHosts []fleet.Host
		var uuids []string
		for i := batchStart; i < batchEnd; i++ {
			uuids = append(uuids, fmt.Sprintf("'test-host-%d'", i))
		}
		err = sqlx.SelectContext(ctx, db, &batchHosts,
			"SELECT id, uuid, hostname, computer_name, node_key, detail_updated_at, label_updated_at, policy_updated_at, platform, os_version, team_id FROM hosts WHERE uuid IN ("+strings.Join(uuids, ",")+")")
		if err != nil {
			return nil, fmt.Errorf("fetch created hosts: %w", err)
		}

		// Insert host_display_names for the created hosts
		if len(batchHosts) > 0 {
			var displayNamePlaceholders []string
			var displayNameArgs []interface{}

			for _, host := range batchHosts {
				displayName := host.Hostname // Use hostname as display name (same logic as migration)
				if host.ComputerName != "" {
					displayName = host.ComputerName
				}
				displayNamePlaceholders = append(displayNamePlaceholders, "(?, ?)")
				displayNameArgs = append(displayNameArgs, host.ID, displayName)
			}

			displayNameSQL := "INSERT INTO host_display_names (host_id, display_name) VALUES " + strings.Join(displayNamePlaceholders, ", ")
			_, err = db.ExecContext(ctx, displayNameSQL, displayNameArgs...)
			if err != nil {
				return nil, fmt.Errorf("batch insert host display names %d-%d: %w", batchStart, batchEnd-1, err)
			}
		}

		// Convert to pointers and add to result
		for i := range batchHosts {
			allHosts = append(allHosts, &batchHosts[i])
		}

		if verbose && (batchEnd%500 == 0 || batchEnd == hostCount) {
			fmt.Printf("  Created %d/%d hosts\n", batchEnd, hostCount)
		}
	}

	return allHosts, nil
}

// batchUpdateHostOS updates operating system info for multiple hosts efficiently
func batchUpdateHostOS(ctx context.Context, ds *mysql.Datastore, hosts []*fleet.Host, verbose bool) error {
	batchSize := 100

	for batchStart := 0; batchStart < len(hosts); batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > len(hosts) {
			batchEnd = len(hosts)
		}

		for i := batchStart; i < batchEnd; i++ {
			host := hosts[i]
			err := retryOnDeadlock(func() error {
				return ds.UpdateHostOperatingSystem(ctx, host.ID, fleet.OperatingSystem{
					Name:           "Ubuntu",
					Version:        fmt.Sprintf("20.04.%d", i%10), // Vary the version slightly
					Platform:       "ubuntu",
					Arch:           "x86_64",
					KernelVersion:  "5.4.0-148-generic",
					DisplayVersion: "20.04",
				})
			}, 3)
			if err != nil {
				return fmt.Errorf("update host %d OS: %w", i, err)
			}
		}

		if verbose && (batchEnd%500 == 0 || batchEnd == len(hosts)) {
			fmt.Printf("  Updated OS for %d/%d hosts\n", batchEnd, len(hosts))
		}
	}

	return nil
}

// batchInstallSoftware installs software on hosts in batches for better performance
func batchInstallSoftware(ctx context.Context, ds *mysql.Datastore, hosts []*fleet.Host, vulnerableSoftware []fleet.Software, verbose bool) error {
	db, err := getDB(ds)
	if err != nil {
		return fmt.Errorf("get DB connection: %w", err)
	}
	defer db.Close()

	// First, create all software entries using INSERT ... ON DUPLICATE KEY UPDATE
	softwareMap := make(map[string]uint) // name+version+source -> software_id

	if len(vulnerableSoftware) > 0 {
		// Insert software in smaller batches to avoid max_allowed_packet issues
		batchSize := 100
		for batchStart := 0; batchStart < len(vulnerableSoftware); batchStart += batchSize {
			batchEnd := batchStart + batchSize
			if batchEnd > len(vulnerableSoftware) {
				batchEnd = len(vulnerableSoftware)
			}

			var placeholders []string
			var args []interface{}

			for i := batchStart; i < batchEnd; i++ {
				software := vulnerableSoftware[i]
				placeholders = append(placeholders, "(?, ?, ?, ?, ?, ?, ?, UNHEX(MD5(CONCAT(COALESCE(?, ''), COALESCE(?, ''), ?))))")
				args = append(args,
					software.Name,
					software.Version,
					software.Source,
					software.BundleIdentifier,
					software.Release,
					software.Vendor,
					software.Arch,
					// Checksum calculation args (name, version, source)
					software.Name,
					software.Version,
					software.Source,
				)
			}

			sql := `INSERT INTO software (name, version, source, bundle_identifier, ` + "`release`" + `, vendor, arch, checksum)
					VALUES ` + strings.Join(placeholders, ", ") + `
					ON DUPLICATE KEY UPDATE name = VALUES(name)`
			_, err := db.ExecContext(ctx, sql, args...)
			if err != nil {
				return fmt.Errorf("batch insert software batch %d-%d: %w", batchStart, batchEnd-1, err)
			}

			if verbose && (batchEnd%200 == 0 || batchEnd == len(vulnerableSoftware)) {
				fmt.Printf("  Inserted software batch %d/%d\n", batchEnd, len(vulnerableSoftware))
			}
		}
	}

	// Now get all software IDs (this approach guarantees we find them)
	for _, software := range vulnerableSoftware {
		key := fmt.Sprintf("%s|%s|%s", software.Name, software.Version, software.Source)

		var softwareID uint
		err := sqlx.GetContext(ctx, db, &softwareID,
			"SELECT id FROM software WHERE name = ? AND version = ? AND source = ? LIMIT 1",
			software.Name, software.Version, software.Source)
		if err != nil {
			return fmt.Errorf("get software ID for %s: %w", software.Name, err)
		}

		softwareMap[key] = softwareID
	}

	// Now batch install software on hosts
	batchSize := 50 // Smaller batches to avoid deadlocks

	for batchStart := 0; batchStart < len(hosts); batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > len(hosts) {
			batchEnd = len(hosts)
		}

		// Process this batch of hosts
		for i := batchStart; i < batchEnd; i++ {
			host := hosts[i]

			// Each host gets 20-80% of the software
			// #nosec G404 - weak random is acceptable for test data generation
			pct := 0.2 + rand.Float64()*0.6
			hostSoftwareCount := int(float64(len(vulnerableSoftware)) * pct)

			// Randomly select which software this host has
			hostVulnSoftware := make([]fleet.Software, len(vulnerableSoftware))
			copy(hostVulnSoftware, vulnerableSoftware)
			rand.Shuffle(len(hostVulnSoftware), func(i, j int) {
				hostVulnSoftware[i], hostVulnSoftware[j] = hostVulnSoftware[j], hostVulnSoftware[i]
			})

			hostSoftware := hostVulnSoftware[:hostSoftwareCount]

			// Clear existing software for this host
			_, err := db.ExecContext(ctx, "DELETE FROM host_software WHERE host_id = ?", host.ID)
			if err != nil {
				return fmt.Errorf("clear host %d software: %w", host.ID, err)
			}

			// Batch insert software for this host
			if len(hostSoftware) > 0 {
				var placeholders []string
				var args []interface{}

				for _, software := range hostSoftware {
					key := fmt.Sprintf("%s|%s|%s", software.Name, software.Version, software.Source)
					softwareID, exists := softwareMap[key]
					if !exists {
						continue // Skip if software ID not found
					}

					placeholders = append(placeholders, "(?, ?)")
					args = append(args, host.ID, softwareID)
				}

				if len(placeholders) > 0 {
					sql := "INSERT IGNORE INTO host_software (host_id, software_id) VALUES " + strings.Join(placeholders, ", ")
					_, err := db.ExecContext(ctx, sql, args...)
					if err != nil {
						return fmt.Errorf("batch insert software for host %d: %w", host.ID, err)
					}
				}
			}
		}

		if verbose && (batchEnd%200 == 0 || batchEnd == len(hosts)) {
			fmt.Printf("  Installed software on %d/%d hosts\n", batchEnd, len(hosts))
		}

		// Small delay between batches to reduce DB pressure
		time.Sleep(50 * time.Millisecond)
	}

	return nil
}

func createTestTeam(ctx context.Context, ds *mysql.Datastore, name string) (*fleet.Team, error) {
	team := &fleet.Team{
		Name: name,
	}
	return ds.NewTeam(ctx, team)
}

func generateVulnerableSoftware(softwareCount int) []fleet.Software {
	var software []fleet.Software

	for i := range softwareCount {
		// Create vulnerable software
		software = append(software, fleet.Software{
			Name: fmt.Sprintf("vulnerable-package-%d", i),
			// #nosec G404 - weak random is acceptable for test data generation
			Version: fmt.Sprintf("1.%d.0", rand.Intn(100)),
			Source:  "Package (deb)",
		})
	}

	return software
}

func generateCVEs(cveCount int) []string {
	var cves []string
	for i := 0; i < cveCount; i++ {
		// #nosec G404 - weak random is acceptable for test data generation
		yearIdx := rand.Intn(len(cvePatterns))
		// #nosec G404 - weak random is acceptable for test data generation
		cveID := fmt.Sprintf(cvePatterns[yearIdx], rand.Intn(9999)+1)
		cves = append(cves, cveID)
	}
	return cves
}

// getDB gets a new sqlx database connection for direct queries
func getDB(ds *mysql.Datastore) (*sqlx.DB, error) {
	cfg := config.MysqlConfig{
		Protocol: "tcp",
		Address:  mysqlAddr,
		Username: mysqlUser,
		Password: mysqlPass,
		Database: mysqlDB,
	}

	dsn := cfg.Username + ":" + cfg.Password + "@" + cfg.Protocol + "(" + cfg.Address + ")/" + cfg.Database + "?charset=utf8mb4&parseTime=True&loc=Local"
	return sqlx.Open("mysql", dsn)
}

func seedSoftwareCVEs(ctx context.Context, ds *mysql.Datastore, cves []string) error {
	db, err := getDB(ds)
	if err != nil {
		return fmt.Errorf("get DB connection: %w", err)
	}
	defer db.Close()

	// First, get software IDs that exist
	var softwareIDs []uint
	err = sqlx.SelectContext(ctx, db, &softwareIDs, "SELECT id FROM software LIMIT 1000")
	if err != nil {
		return fmt.Errorf("fetch software IDs: %w", err)
	}

	if len(softwareIDs) == 0 {
		return errors.New("no software found - run seedVulnerabilities first")
	}

	// Insert software_cve mappings
	for i, cve := range cves {
		// Each CVE affects 1-3 software packages
		// #nosec G404 - weak random is acceptable for test data generation
		affectedCount := 1 + rand.Intn(3)
		for j := 0; j < affectedCount && j < len(softwareIDs); j++ {
			softwareID := softwareIDs[(i+j)%len(softwareIDs)]
			err := retryOnDeadlock(func() error {
				_, err := db.ExecContext(ctx,
					"INSERT IGNORE INTO software_cve (software_id, cve) VALUES (?, ?)",
					softwareID, cve)
				return err
			}, 3)
			if err != nil {
				return fmt.Errorf("insert software_cve for %s: %w", cve, err)
			}
		}
	}

	return nil
}

func ensureOSRecordsExist(ctx context.Context, ds *mysql.Datastore) error {
	db, err := getDB(ds)
	if err != nil {
		return fmt.Errorf("get DB connection: %w", err)
	}
	defer db.Close()

	// Check if we have any OS records
	var count int
	err = sqlx.GetContext(ctx, db, &count, "SELECT COUNT(*) FROM operating_systems")
	if err != nil {
		return fmt.Errorf("count operating systems: %w", err)
	}

	fmt.Printf("Found %d existing OS records\n", count)

	if count == 0 {
		fmt.Printf("No OS records found. The hosts may not have had their OS info updated yet.\n")
		fmt.Printf("Try running with fewer hosts first, or check that UpdateHostOperatingSystem was called.\n")
	}

	return nil
}

func seedOSVulnerabilities(ctx context.Context, ds *mysql.Datastore, cves []string) error {
	db, err := getDB(ds)
	if err != nil {
		return fmt.Errorf("get DB connection: %w", err)
	}
	defer db.Close()

	// Get OS IDs
	var osIDs []uint
	err = sqlx.SelectContext(ctx, db, &osIDs, "SELECT id FROM operating_systems LIMIT 100")
	if err != nil {
		return fmt.Errorf("fetch OS IDs: %w", err)
	}

	if len(osIDs) == 0 {
		fmt.Printf("Warning: No operating systems found in database. Skipping OS vulnerability seeding.\n")
		fmt.Printf("This might be normal if your test setup doesn't include OS vulnerability testing.\n")
		return nil // Don't fail, just skip OS vulnerabilities
	}

	fmt.Printf("Found %d operating systems to map vulnerabilities to\n", len(osIDs))

	// Insert OS vulnerabilities (about 30% of CVEs affect OS)
	for i, cve := range cves {
		// #nosec G404 - weak random is acceptable for test data generation
		if rand.Float64() < 0.3 { // 30% chance this CVE affects OS
			osID := osIDs[i%len(osIDs)]
			err := retryOnDeadlock(func() error {
				_, err := db.ExecContext(ctx,
					"INSERT IGNORE INTO operating_system_vulnerabilities (operating_system_id, cve) VALUES (?, ?)",
					osID, cve)
				return err
			}, 3)
			if err != nil {
				return fmt.Errorf("insert OS vulnerability for %s: %w", cve, err)
			}
		}
	}

	return nil
}

func seedVulnerabilities(ctx context.Context, ds *mysql.Datastore, hostCount, teamCount, cveCount, softwareCount int, verbose bool) error {
	var teams []*fleet.Team
	err := timeStep(fmt.Sprintf("Creating %d teams", teamCount), verbose, func() error {
		for i := 0; i < teamCount; i++ {
			team, err := createTestTeam(ctx, ds, fmt.Sprintf("test-team-%d", i))
			if err != nil {
				return fmt.Errorf("create team %d: %w", i, err)
			}
			teams = append(teams, team)
		}
		return nil
	})
	if err != nil {
		return err
	}

	var hosts []*fleet.Host
	err = timeStep(fmt.Sprintf("Creating %d hosts", hostCount), verbose, func() error {
		var err error
		hosts, err = batchCreateHosts(ctx, ds, hostCount, teams, verbose)
		return err
	})
	if err != nil {
		return err
	}

	err = timeStep("Updating host operating systems", verbose, func() error {
		return batchUpdateHostOS(ctx, ds, hosts, verbose)
	})
	if err != nil {
		return err
	}

	var cves []string
	var vulnerableSoftware []fleet.Software
	err = timeStep(fmt.Sprintf("Generating %d CVEs and %d software packages", cveCount, softwareCount), verbose, func() error {
		cves = generateCVEs(cveCount)
		vulnerableSoftware = generateVulnerableSoftware(softwareCount)
		return nil
	})
	if err != nil {
		return err
	}

	// Install software on each host using batch approach
	err = timeStep(fmt.Sprintf("Installing software on %d hosts", len(hosts)), verbose, func() error {
		return batchInstallSoftware(ctx, ds, hosts, vulnerableSoftware, verbose)
	})
	if err != nil {
		return err
	}

	err = timeStep("Seeding software-CVE mappings", verbose, func() error {
		return seedSoftwareCVEs(ctx, ds, cves)
	})
	if err != nil {
		return fmt.Errorf("seed software CVEs: %w", err)
	}

	err = timeStep("Ensuring OS records exist", verbose, func() error {
		return ensureOSRecordsExist(ctx, ds)
	})
	if err != nil {
		return fmt.Errorf("ensure OS records: %w", err)
	}

	err = timeStep("Seeding OS vulnerabilities", verbose, func() error {
		return seedOSVulnerabilities(ctx, ds, cves)
	})
	if err != nil {
		return fmt.Errorf("seed OS vulnerabilities: %w", err)
	}

	return nil
}

func main() {
	var (
		hostCount     = flag.Int("hosts", 100, "Number of hosts to create")
		teamCount     = flag.Int("teams", 5, "Number of teams to create")
		cveCount      = flag.Int("cves", 500, "Total number of unique CVEs in the system")
		softwareCount = flag.Int("software", 500, "Total number of unique software packages (each host gets 20-80% randomly)")
		help          = flag.Bool("help", false, "Show help information")
		verbose       = flag.Bool("verbose", false, "Enable verbose timing output for each step")
	)
	flag.Parse()

	if *help {
		fmt.Printf("Fleet Test Data Seeder\n\n")
		fmt.Printf("This tool creates test data for Fleet performance testing.\n\n")
		fmt.Printf("Data model:\n")
		fmt.Printf("- Creates %d total unique software packages\n", *softwareCount)
		fmt.Printf("- Each host gets 20-80%% of software packages randomly assigned\n")
		fmt.Printf("- Creates %d total unique CVEs\n", *cveCount)
		fmt.Printf("- Each CVE affects 1-3 random software packages\n")
		fmt.Printf("- About 30%% of CVEs also affect the operating system\n")
		fmt.Printf("- Host vulnerability counts depend on which software they have installed\n\n")
		fmt.Printf("Examples:\n")
		fmt.Printf("  %s -hosts=1000 -teams=10 -cves=500 -software=1000\n", os.Args[0])
		fmt.Printf("  %s -hosts=5000 -teams=20 -cves=2000 -software=5000 -verbose\n", os.Args[0])
		fmt.Printf("\n")
		fmt.Printf("After seeding data, use performance_tester.go to test datastore methods.\n")
		fmt.Printf("\n")
		flag.Usage()
		return
	}

	ctx := context.Background()

	// Connect to datastore
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

	fmt.Printf("Seeding test data...\n")
	fmt.Printf("Configuration: %d hosts, %d teams, %d CVEs, %d software packages\n", *hostCount, *teamCount, *cveCount, *softwareCount)

	if err := seedVulnerabilities(ctx, ds, *hostCount, *teamCount, *cveCount, *softwareCount, *verbose); err != nil {
		fmt.Printf("Failed to seed vulnerabilities: %v\n", err)
		return
	}

	fmt.Printf("Data seeding complete!\n")
	fmt.Printf("\nUse performance_tester.go to test datastore methods with this data.\n")
	fmt.Printf("Example: go run performance_tester.go -funcs=UpdateVulnerabilityHostCounts -iterations=5\n")

	fmt.Println("Done.")
}
