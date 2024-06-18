Fixing the migration from https://github.com/fleetdm/fleet/pull/19829 

Ok so this is what I have:
-- this updates the 2 migrations that go after the inserted one
UPDATE migration_status_tables SET id = id + 1 WHERE version_id > 20240521143024 ORDER BY id DESC;

-- this inserts the new migration at the right order
INSERT INTO migration_status_tables (id, version_id, is_applied) VALUES (268, 20240601174138, 1);
I'll double-check the IDs are as mine locally before applying.





Martin Angers
  15 minutes ago
I tested by git checkout the same version as dogfood, make db-reset to set the DB to this version, then git checkout main and start fleet serve, it failed with this as expected:
$ ./build/fleet serve --dev --vulnerabilities_periodicity 1m --vulnerabilities_databases_path '/home/m/Documents/FleetDM/vulns' --vulnerabilities_current_instance_checks on --osquery_detail_update_interval 1m --logging_debug --dev_license
################################################################################
# WARNING:
#   Your Fleet database is missing required migrations. This is likely to cause
#   errors in Fleet.
#
#   Missing migrations: tables=[20240613162201 20240613172616], data=[].
#
#   Run `./build/fleet prepare db` to perform migrations.
#
#   To run the server without performing migrations:
#     - Set environment variable FLEET_UPGRADES_ALLOW_MISSING_MIGRATIONS=1, or,
#     - Set config updates.allow_missing_migrations to true, or,
#     - Use command line argument --upgrades_allow_missing_migrations=true
################################################################################
And then I applied the migrations and it succeeded:
$ ./build/fleet prepare db --dev 
2024/06/18 16:46:52 [2024-06-13] Add MDM Windows Host UUID Index
2024/06/18 16:46:52 [2024-06-13] Host Issues Table
Migrations completed.