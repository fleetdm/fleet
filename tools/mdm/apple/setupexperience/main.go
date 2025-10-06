// Command applebmapi takes an Apple Business Manager server token in decrypted
// JSON format and calls the Apple BM API to retrieve and print the account
// information or the specified enrollment profile.
//
// Was implemented to test out https://github.com/fleetdm/fleet/issues/7515#issuecomment-1330889768,
// and can still be useful for debugging purposes.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/jmoiron/sqlx"

	kitlog "github.com/go-kit/log"
)

func main() {
	mysqlAddr := flag.String("mysql", "localhost:3306", "mysql address")
	serverPrivateKey := flag.String("server-private-key", "", "fleet server's private key (to decrypt MDM assets)")
	hostUUID := flag.String("host-uuid", "", "the host serial # to enqueue setup items for")

	flag.Parse()

	if *serverPrivateKey == "" {
		log.Fatal("must provide -server-private-key")
	}

	if len(*serverPrivateKey) > 32 {
		// We truncate to 32 bytes because AES-256 requires a 32 byte (256 bit) PK, but some
		// infra setups generate keys that are longer than 32 bytes.
		truncatedServerPrivateKey := (*serverPrivateKey)[:32]
		serverPrivateKey = &truncatedServerPrivateKey
	}

	mysqlConf := config.MysqlConfig{
		Protocol:        "tcp",
		Address:         *mysqlAddr,
		Database:        "fleet",
		Username:        "fleet",
		Password:        "insecure",
		MaxOpenConns:    50,
		MaxIdleConns:    50,
		ConnMaxLifetime: 0,
	}

	dsn := fmt.Sprintf("%s:%s@%s(%s)/%s", mysqlConf.Username, mysqlConf.Password, mysqlConf.Protocol, mysqlConf.Address, mysqlConf.Database)

	db, err := sqlx.Open("mysql", dsn) // or your traced driver name
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}
	// Pool tuning similar to Fleet
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(50)
	// db.SetConnMaxLifetime(time.Second * time.Duration(conf.ConnMaxLifetime))
	if err := db.Ping(); err != nil {
		log.Fatal("failed to ping database:", err)
	}

	ctx := context.Background()

	var teamID uint

	// Get the host ID and team ID for the provided host UUID.
	type HostInfo struct {
		ID     uint  `db:"id"`
		TeamID *uint `db:"team_id"`
	}
	var hostInfo HostInfo
	err = db.GetContext(ctx, &hostInfo, `SELECT id, team_id FROM hosts WHERE uuid = ?`, *hostUUID)
	if err != nil {
		log.Fatalf("failed to query host info for UUID %s: %v", *hostUUID, err)
	}
	if hostInfo.TeamID == nil {
		log.Fatalf("host must belong to a team")
	}
	teamID = *hostInfo.TeamID
	hostID := &hostInfo.ID

	// Get the apple mdm profiles that will need to be inserted by querying
	// the mdm_apple_configuration_profiles table, getting the identifier,
	// profile_uuid, name and checksum columns.
	type mdmProfile struct {
		ProfileIdentifier string `db:"identifier"`
		ProfileUUID       string `db:"profile_uuid"`
		Name              string `db:"name"`
		Checksum          string `db:"checksum"`
	}
	var mdmProfiles []mdmProfile
	err = db.SelectContext(ctx, &mdmProfiles, `SELECT identifier, profile_uuid, name, checksum FROM mdm_apple_configuration_profiles`)
	if err != nil {
		log.Fatal("failed to query mdm_apple_configuration_profiles:", err)
	}
	if len(mdmProfiles) == 0 {
		log.Fatal("no mdm_apple_configuration_profiles found; must have at least one")
	}

	// Insert nano_devices and nano_enrollments rows for the host UUID if they don't exist
	_, err = db.ExecContext(ctx, `INSERT IGNORE INTO host_mdm (host_id, enrolled) VALUES (?, 1) ON DUPLICATE KEY UPDATE enrolled = 1`, *hostID)
	if err != nil {
		log.Fatalf("failed to insert host_mdm for host %d: %v", *hostID, err)
	}
	_, err = db.ExecContext(ctx, `INSERT IGNORE INTO nano_devices (id, platform, authenticate) VALUES (?, 'darwin', 0)`, *hostUUID)
	if err != nil {
		log.Fatalf("failed to insert nano_devices for host UUID %s: %v", *hostUUID, err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO nano_enrollments (
			id, device_id, user_id, type, topic, push_magic, token_hex, enabled,
			token_update_tally, last_seen_at, enrolled_from_migration
		) VALUES (
			?, ?, NULL, 'Device', 'com.example.mdm', 'magic-token', 'deadbeef', 1,
			1, NOW(), 0
		) ON DUPLICATE KEY UPDATE enabled = 1`, *hostUUID, *hostUUID)
	if err != nil {
		log.Fatalf("failed to insert nano_enrollments for host UUID %s: %v", *hostUUID, err)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO host_mdm_apple_awaiting_configuration (host_uuid, awaiting_configuration) VALUES (?, 1) ON DUPLICATE KEY UPDATE awaiting_configuration = 1
	`, *hostID)
	if err != nil {
		log.Fatalf("failed to insert host_mdm_apple_awaiting_configuration for host %d: %v", *hostID, err)
	}

	// For each profile, insert a row into host_mdm_apple_profiles if one doesn't already exist.
	for _, p := range mdmProfiles {
		_, err = db.ExecContext(ctx, `
			INSERT INTO host_mdm_apple_profiles (
				host_uuid, profile_identifier, profile_uuid, profile_name, checksum, status, operation_type, command_uuid
			) VALUES (?, ?, ?, ?, ?, 'verified', 'install', '') ON DUPLICATE KEY UPDATE status = 'verified', operation_type = 'install', command_uuid = '';
		`, *hostUUID, p.ProfileIdentifier, p.ProfileUUID, p.Name, p.Checksum)
		if err != nil {
			log.Fatalf("failed to insert host_mdm_apple_profiles for profile %s: %v", p.ProfileIdentifier, err)
		}
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
	mds, err := mysql.New(mysqlConf, clock.C, opts...)
	if err != nil {
		log.Fatal(err)
	}

	_, err = mds.EnqueueSetupExperienceItems(ctx, "darwin", *hostUUID, teamID)
}
