package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/WatchBeam/clock"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

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
)

// Common CVE patterns for realistic data
var cvePatterns = []string{
	"CVE-2024-%04d", "CVE-2023-%04d", "CVE-2022-%04d", "CVE-2021-%04d",
}

func createTestHost(ctx context.Context, ds *mysql.Datastore, identifier string, teamID *uint) (*fleet.Host, error) {
	base := fleet.Host{
		UUID:            identifier,
		Hostname:        identifier,
		NodeKey:         ptr.String(identifier),
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		PrimaryIP:       "192.168.1.100",
		PrimaryMac:      "00:11:22:33:44:55",
		Platform:        "ubuntu",
		OSVersion:       "Ubuntu 20.04.6 LTS",
		TeamID:          teamID,
	}
	return ds.NewHost(ctx, &base)
}

func createTestTeam(ctx context.Context, ds *mysql.Datastore, name string) (*fleet.Team, error) {
	team := &fleet.Team{
		Name: name,
	}
	return ds.NewTeam(ctx, team)
}

func generateVulnerableSoftware(cveCount int) []fleet.Software {
	var software []fleet.Software

	for i := 0; i < cveCount; i++ {
		// Create vulnerable software
		software = append(software, fleet.Software{
			Name:    fmt.Sprintf("vulnerable-package-%d", i),
			Version: fmt.Sprintf("1.%d.0", rand.Intn(100)),
			Source:  "Package (deb)",
		})
	}

	return software
}

func generateCVEs(cveCount int) []string {
	var cves []string
	for i := 0; i < cveCount; i++ {
		yearIdx := rand.Intn(len(cvePatterns))
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
		return fmt.Errorf("no software found - run seedVulnerabilities first")
	}

	// Insert software_cve mappings
	for i, cve := range cves {
		// Each CVE affects 1-3 software packages
		affectedCount := 1 + rand.Intn(3)
		for j := 0; j < affectedCount && j < len(softwareIDs); j++ {
			softwareID := softwareIDs[(i+j)%len(softwareIDs)]
			_, err := db.ExecContext(ctx,
				"INSERT IGNORE INTO software_cve (software_id, cve) VALUES (?, ?)",
				softwareID, cve)
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
		if rand.Float64() < 0.3 { // 30% chance this CVE affects OS
			osID := osIDs[i%len(osIDs)]
			_, err := db.ExecContext(ctx,
				"INSERT IGNORE INTO operating_system_vulnerabilities (operating_system_id, cve) VALUES (?, ?)",
				osID, cve)
			if err != nil {
				return fmt.Errorf("insert OS vulnerability for %s: %w", cve, err)
			}
		}
	}

	return nil
}

func seedVulnerabilities(ctx context.Context, ds *mysql.Datastore, hostCount, teamCount, cveCount int) error {
	fmt.Printf("Creating %d teams...\n", teamCount)
	var teams []*fleet.Team
	for i := 0; i < teamCount; i++ {
		team, err := createTestTeam(ctx, ds, fmt.Sprintf("test-team-%d", i))
		if err != nil {
			return fmt.Errorf("create team %d: %w", i, err)
		}
		teams = append(teams, team)
	}

	fmt.Printf("Creating %d hosts...\n", hostCount)
	var hosts []*fleet.Host
	for i := 0; i < hostCount; i++ {
		var teamID *uint
		if len(teams) > 0 {
			// Distribute hosts across teams (some no-team)
			if i%3 != 0 { // 2/3 of hosts have teams
				teamID = &teams[i%len(teams)].ID
			}
		}

		host, err := createTestHost(ctx, ds, fmt.Sprintf("test-host-%d", i), teamID)
		if err != nil {
			return fmt.Errorf("create host %d: %w", i, err)
		}

		// Update host operating system to populate operating_systems table
		err = ds.UpdateHostOperatingSystem(ctx, host.ID, fleet.OperatingSystem{
			Name:           "Ubuntu",
			Version:        fmt.Sprintf("20.04.%d", i%10), // Vary the version slightly
			Platform:       "ubuntu",
			Arch:           "x86_64",
			KernelVersion:  "5.4.0-148-generic",
			DisplayVersion: "20.04",
		})
		if err != nil {
			return fmt.Errorf("update host %d OS: %w", i, err)
		}

		hosts = append(hosts, host)
	}

	fmt.Printf("Generating %d unique CVEs and installing software...\n", cveCount)
	cves := generateCVEs(cveCount)
	vulnerableSoftware := generateVulnerableSoftware(cveCount)

	// Install software on each host
	for i, host := range hosts {
		// Each host gets 20-80% of the software
		pct := 0.2 + rand.Float64()*0.6
		hostSoftwareCount := int(float64(len(vulnerableSoftware)) * pct)

		// Randomly select which software this host has
		rand.Shuffle(len(vulnerableSoftware), func(i, j int) {
			vulnerableSoftware[i], vulnerableSoftware[j] = vulnerableSoftware[j], vulnerableSoftware[i]
		})

		hostSoftware := vulnerableSoftware[:hostSoftwareCount]

		if _, err := ds.UpdateHostSoftware(ctx, host.ID, hostSoftware); err != nil {
			return fmt.Errorf("update host %d software: %w", i, err)
		}

		if (i+1)%100 == 0 {
			fmt.Printf("  Processed %d/%d hosts\n", i+1, len(hosts))
		}
	}

	fmt.Printf("Seeding software-CVE mappings...\n")
	if err := seedSoftwareCVEs(ctx, ds, cves); err != nil {
		return fmt.Errorf("seed software CVEs: %w", err)
	}

	fmt.Printf("Ensuring OS records exist before seeding vulnerabilities...\n")
	if err := ensureOSRecordsExist(ctx, ds); err != nil {
		return fmt.Errorf("ensure OS records: %w", err)
	}

	fmt.Printf("Seeding OS vulnerabilities...\n")
	if err := seedOSVulnerabilities(ctx, ds, cves); err != nil {
		return fmt.Errorf("seed OS vulnerabilities: %w", err)
	}

	return nil
}

func runPerformanceTest(ctx context.Context, ds *mysql.Datastore, iterations int) {
	fmt.Printf("\nRunning %d iterations of UpdateVulnerabilityHostCounts...\n", iterations)

	var totalDuration time.Duration
	for i := 0; i < iterations; i++ {
		start := time.Now()

		if err := ds.UpdateVulnerabilityHostCounts(ctx, 10); err != nil {
			log.Printf("Iteration %d failed: %v", i+1, err)
			continue
		}

		duration := time.Since(start)
		totalDuration += duration

		fmt.Printf("Iteration %d: %v\n", i+1, duration)
	}

	avgDuration := totalDuration / time.Duration(iterations)
	fmt.Printf("\nResults:\n")
	fmt.Printf("Total time: %v\n", totalDuration)
	fmt.Printf("Average time: %v\n", avgDuration)
	fmt.Printf("Iterations: %d\n", iterations)
}

func main() {
	var (
		hostCount  = flag.Int("hosts", 100, "Number of hosts to create")
		teamCount  = flag.Int("teams", 5, "Number of teams to create")
		cveCount   = flag.Int("cves", 500, "Number of unique CVEs to generate")
		iterations = flag.Int("iterations", 3, "Number of performance test iterations")
		seedOnly   = flag.Bool("seed-only", false, "Only seed data, don't run performance test")
		testOnly   = flag.Bool("test-only", false, "Only run performance test (assume data exists)")
	)
	flag.Parse()

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

	if !*testOnly {
		fmt.Printf("Seeding test data...\n")
		fmt.Printf("Configuration: %d hosts, %d teams, %d CVEs\n", *hostCount, *teamCount, *cveCount)

		if err := seedVulnerabilities(ctx, ds, *hostCount, *teamCount, *cveCount); err != nil {
			fmt.Printf("Failed to seed vulnerabilities: %v\n", err)
			return
		}

		fmt.Printf("Data seeding complete!\n")
	}

	if !*seedOnly {
		runPerformanceTest(ctx, ds, *iterations)
	}

	fmt.Println("Done.")
}
