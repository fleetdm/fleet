package main

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type Host struct {
	ID               int        `db:"id"`
	OsqueryHostID    string     `db:"osquery_host_id"`
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        *time.Time `db:"updated_at"`
	DetailUpdatedAt  time.Time  `db:"detail_updated_at"`
	NodeKey          string     `db:"node_key"`
	Hostname         string     `db:"hostname"`
	Uuid             string     `db:"uuid"`
	Platform         string     `db:"platform"`
	OsqueryVersion   string     `db:"osquery_version"`
	OsVersion        string     `db:"os_version"`
	Build            string     `db:"build"`
	PlatformLike     string     `db:"platform_like"`
	CodeName         string     `db:"code_name"`
	Uptime           int64      `db:"uptime"`
	Memory           int64      `db:"memory"`
	CpuType          string     `db:"cpu_type"`
	CpuSubtype       string     `db:"cpu_subtype"`
	CpuBrand         string     `db:"cpu_brand"`
	CpuPhysicalCores int        `db:"cpu_physical_cores"`
	CpuLogicalCores  int        `db:"cpu_logical_cores"`
	HardwareVendor   string     `db:"hardware_vendor"`
	HardwareModel    string     `db:"hardware_model"`
	HardwareVersion  string     `db:"hardware_version"`
	HardwareSerial   string     `db:"hardware_serial"`
	ComputerName     string     `db:"computer_name"`
	PrimaryIP        string     `db:"primary_ip"`
	PrimaryMac       string     `db:"primary_mac"`
	LabelUpdatedAt   time.Time  `db:"label_updated_at"`
	LastEnrolledAt   time.Time  `db:"last_enrolled_at"`
	RefetchRequested int        `db:"refetch_requested"`
	PublicIP         string     `db:"public_ip"`
}

type HostDisplayName struct {
	HostID int64  `db:"host_id"`
	Name   string `db:"display_name"`
}

type Software struct {
	ID               int64  `db:"id"`
	Name             string `db:"name"`
	Version          string `db:"version"`
	Source           string `db:"source"`
	BundleIdentifier string `db:"bundle_identifier"`
	Release          string `db:"release"`
	VendorOld        string `db:"vendor_old"`
	Arch             string `db:"arch"`
	Vendor           string `db:"vendor"`
}

type HostSoftware struct {
	HostID     int64 `db:"host_id"`
	SoftwareID int64 `db:"software_id"`
}

func readCSVFile(filePath string) ([]Software, error) {
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

	var software []Software
	for _, line := range lines[1:] { // Skip header line
		software = append(software, Software{
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

func insertOrUpdateSoftware(db *sqlx.DB, hostID int, software Software) error {
	// Check if the software already exists
	var existingID int64
	query := `SELECT id FROM software WHERE name = ? AND version = ? AND source = ?`
	err := db.Get(&existingID, query, software.Name, software.Version, software.Source)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("select software: %w", err)
	}
	software.ID = existingID

	if existingID > 0 {
		// Update existing record, set ID for the update
		software.ID = existingID
		updateQuery := `UPDATE software SET bundle_identifier = :bundle_identifier, ` + "`release` = :release, vendor_old = :vendor_old, arch = :arch, vendor = :vendor WHERE id = :id"
		_, err := db.NamedExec(updateQuery, software)
		if err != nil {
			return fmt.Errorf("update software: %w", err)
		}
	} else {
		// Insert new record
		insertQuery := `INSERT INTO software (name, version, source, bundle_identifier, ` + "`release`," + ` vendor_old, arch, vendor) VALUES (:name, :version, :source, :bundle_identifier, :release, :vendor_old, :arch, :vendor)`
		res, err := db.NamedExec(insertQuery, software)
		if err != nil {
			return fmt.Errorf("insert software: %w", err)
		}
		software.ID, err = res.LastInsertId()
		if err != nil {
			return fmt.Errorf("get last insert id: %w", err)
		}
	}

	err = insertOrUpdateHostSoftware(db, HostSoftware{
		HostID:     int64(hostID),
		SoftwareID: software.ID,
	})
	if err != nil {
		return fmt.Errorf("insert host software: %w", err)
	}

	return nil
}

func insertOrUpdateHost(db *sqlx.DB, host Host) (int, error) {
	// Check if the host already exists
	var existingID int
	query := `SELECT id FROM hosts WHERE osquery_host_id = ?`
	err := db.Get(&existingID, query, host.OsqueryHostID)
	if err != nil && err != sql.ErrNoRows {
		return existingID, fmt.Errorf("select host: %w", err)
	}

	if existingID > 0 {
		// Update existing record
		updateQuery := `UPDATE hosts SET updated_at = :updated_at, detail_updated_at = :detail_updated_at, node_key = :node_key, hostname = :hostname, uuid = :uuid, platform = :platform, osquery_version = :osquery_version, os_version = :os_version, build = :build, platform_like = :platform_like, code_name = :code_name, uptime = :uptime, memory = :memory, cpu_type = :cpu_type, cpu_subtype = :cpu_subtype, cpu_brand = :cpu_brand, cpu_physical_cores = :cpu_physical_cores, cpu_logical_cores = :cpu_logical_cores, hardware_vendor = :hardware_vendor, hardware_model = :hardware_model, hardware_version = :hardware_version, hardware_serial = :hardware_serial, computer_name = :computer_name, primary_ip = :primary_ip, primary_mac = :primary_mac, label_updated_at = :label_updated_at, last_enrolled_at = :last_enrolled_at, refetch_requested = :refetch_requested, public_ip = :public_ip WHERE id = :id`
		_, err := db.NamedExec(updateQuery, host)
		if err != nil {
			return 0, fmt.Errorf("update host: %w", err)
		}
		err = insertOrUpdateHostDisplayName(db, HostDisplayName{
			HostID: int64(existingID),
			Name:   host.Hostname,
		})
		if err != nil {
			return 0, fmt.Errorf("insert host display name: %w", err)
		}
		return existingID, nil

	}

	// Insert new record
	insertQuery := `INSERT INTO hosts (osquery_host_id, created_at, detail_updated_at, node_key, hostname, uuid, platform, osquery_version, os_version, build, platform_like, code_name, uptime, memory, cpu_type, cpu_subtype, cpu_brand, cpu_physical_cores, cpu_logical_cores, hardware_vendor, hardware_model, hardware_version, hardware_serial, computer_name, primary_ip, primary_mac, label_updated_at, last_enrolled_at, refetch_requested, public_ip) VALUES (:osquery_host_id, :created_at, :detail_updated_at, :node_key, :hostname, :uuid, :platform, :osquery_version, :os_version, :build, :platform_like, :code_name, :uptime, :memory, :cpu_type, :cpu_subtype, :cpu_brand, :cpu_physical_cores, :cpu_logical_cores, :hardware_vendor, :hardware_model, :hardware_version, :hardware_serial, :computer_name, :primary_ip, :primary_mac, :label_updated_at, :last_enrolled_at, :refetch_requested, :public_ip)`
	res, err := db.NamedExec(insertQuery, host)
	if err != nil {
		return 0, fmt.Errorf("insert host: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get last insert id: %w", err)
	}

	err = insertOrUpdateHostDisplayName(db, HostDisplayName{
		HostID: id,
		Name:   host.Hostname,
	})
	if err != nil {
		return 0, fmt.Errorf("insert host display name: %w", err)
	}

	return int(id), nil
}

func insertOrUpdateHostDisplayName(db *sqlx.DB, hdn HostDisplayName) error {
	// Check if the host already exists
	var foundDN HostDisplayName
	query := `SELECT host_id, display_name FROM host_display_names WHERE host_id = ?`
	err := db.Get(&foundDN, query, hdn.HostID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("select host display name: %w", err)
	}

	if err == sql.ErrNoRows {
		// Insert new record
		insertQuery := `INSERT INTO host_display_names (host_id, display_name) VALUES (:host_id, :display_name)`
		_, err := db.NamedExec(insertQuery, hdn)
		if err != nil {
			return fmt.Errorf("insert host display name: %w", err)
		}
	}

	return nil
}

func insertOrUpdateHostSoftware(db *sqlx.DB, hs HostSoftware) error {
	// Check if the host already exists
	var foundhs HostSoftware
	query := `SELECT host_id, software_id FROM host_software WHERE host_id = ? AND software_id = ?`
	err := db.Get(&foundhs, query, hs.HostID, hs.SoftwareID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("select host software: %w", err)
	}

	if err == sql.ErrNoRows {
		// Insert new record
		insertQuery := `INSERT INTO host_software (host_id, software_id) VALUES (:host_id, :software_id)`
		_, err := db.NamedExec(insertQuery, hs)
		if err != nil {
			return fmt.Errorf("insert host software: %w", err)
		}
	}

	return nil
}

func main() {
	// Database connection string
	dsn := "fleet:insecure@tcp(localhost:3306)/fleet"

	// Connect to database
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Host data to insert
	host := Host{
		OsqueryHostID:    "sample_host_id",
		CreatedAt:        time.Now(),
		DetailUpdatedAt:  time.Now(),
		NodeKey:          "sample_node_key",
		Hostname:         "sample_hostname",
		Uuid:             "sample_uuid",
		Platform:         "sample_platform",
		OsqueryVersion:   "sample_version",
		OsVersion:        "sample_os_version",
		Build:            "sample_build",
		PlatformLike:     "sample_platform_like",
		CodeName:         "sample_code_name",
		Uptime:           1000,
		Memory:           8000,
		CpuType:          "sample_cpu_type",
		CpuSubtype:       "sample_cpu_subtype",
		CpuBrand:         "sample_cpu_brand",
		CpuPhysicalCores: 4,
		CpuLogicalCores:  8,
		HardwareVendor:   "sample_vendor",
		HardwareModel:    "sample_model",
		HardwareVersion:  "sample_hardware_version",
		HardwareSerial:   "sample_serial",
		ComputerName:     "sample_computer_name",
		PrimaryIP:        "192.168.1.1",
		PrimaryMac:       "00:11:22:33:44:55",
		LabelUpdatedAt:   time.Now(),
		LastEnrolledAt:   time.Now(),
		RefetchRequested: 0,
		PublicIP:         "203.0.113.1",
	}

	// macos Host
	host.Platform = "darwin"
	host.OsVersion = "Mac OS X 10.14.6"
	host.Hostname = "macos-seed-host"
	host.ComputerName = "macos-seed-host"
	host.OsqueryHostID = "macos-seed-host"
	host.NodeKey = "macos-seed-host"
	macosID, err := insertOrUpdateHost(db, host)
	if err != nil {
		log.Fatal(err) //nolint:gocritic // ignore exitAfterDefer
	}

	// windows Host
	host.Platform = "windows"
	host.OsVersion = "Windows 11 Enterprise"
	host.ComputerName = "windows-seed-host"
	host.Hostname = "windows-seed-host"
	host.OsqueryHostID = "windows-seed-host"
	host.NodeKey = "windows-seed-host"
	winID, err := insertOrUpdateHost(db, host)
	if err != nil {
		log.Fatal(err)
	}

	// ubuntu Host
	host.Platform = "debian"
	host.OsVersion = "Ubuntu 22.04.1 LTS"
	host.Hostname = "ubuntu-seed-host"
	host.ComputerName = "ubuntu-seed-host"
	host.OsqueryHostID = "ubuntu-seed-host"
	host.NodeKey = "ubuntu-seed-host"
	ubuntuID, err := insertOrUpdateHost(db, host)
	if err != nil {
		log.Fatal(err)
	}

	// Insert macOS software
	macSoftware, err := readCSVFile("./tools/seed_data/software-macos.csv")
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range macSoftware {
		err := insertOrUpdateSoftware(db, macosID, s)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Insert win software
	winSoftware, err := readCSVFile("./tools/seed_data/software-win.csv")
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range winSoftware {
		err := insertOrUpdateSoftware(db, winID, s)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Insert ubuntu software
	ubuntuSoftware, err := readCSVFile("./tools/seed_data/software-ubuntu.csv")
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range ubuntuSoftware {
		err := insertOrUpdateSoftware(db, ubuntuID, s)
		if err != nil {
			log.Fatal(err)
		}
	}
}
