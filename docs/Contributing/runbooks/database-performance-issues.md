## Database performance issues

### Use this runbook if

1. A customer environment is experiencing elevated error rate, outages, slow load times, or timeouts/502s.
2. Database load has not been eliminated as a cause of the issues.

This runbook is written for an engineering audience; if you're on the infrastructure team, you'll have access to these tools directly rather than needing to ask for them.

### Process

#### Check RDS insights

If available (e.g. on managed cloud customers, or self-hosted customers running on RDS), check active queries in AWS RDS insights. For managed cloud environments, ask infrastructure for this information. For self-hosted environments, ask the customer.

#### If locks are the problem, check them

If transaction locks are the source of issues, run [troubleshoot_locks.sql](https://github.com/fleetdm/confidential/blob/main/infrastructure/cloud/scripts/sql/troubleshoot_locks.sql) on the database to find which locks are causing the issue.

#### Check table row counts

As of Fleet 4.81, managed cloud environments include table row counts as part of logs generated post-database-migration, with DB migrations happening on each deploy. Compare these row counts with load test info below to see if we're dealing with an environment that is shaped differently than we've load tested.

<details>
    <summary>Example from load test</summary>

TABLE_NAME	table_rows
abm_tokens	0
activities	251
aggregated_stats	883476
android_app_configurations	0
android_devices	0
android_enterprises	0
android_policy_requests	0
app_config_json	0
batch_activities	0
batch_activity_host_results	0
ca_config_assets	0
calendar_events	0
carve_blocks	0
carve_metadata	0
certificate_authorities	0
certificate_templates	0
challenges	0
conditional_access_scep_certificates	0
conditional_access_scep_serials	0
cron_stats	10477
cve_meta	353676
default_team_config_json	1
distributed_query_campaign_targets	0
distributed_query_campaigns	183
email_changes	0
enroll_secrets	5
eulas	0
fleet_maintained_apps	251
fleet_variables	17
host_activities	0
host_additional	61730
host_batteries	81793
host_calendar_events	0
host_certificate_sources	705899
host_certificate_templates	0
host_certificates	661383
host_dep_assignments	0
host_device_auth	0
host_disk_encryption_keys	0
host_disk_encryption_keys_archive	0
host_disks	98350
host_display_names	103377
host_emails	192072
host_identity_scep_certificates	0
host_identity_scep_serials	0
host_in_house_software_installs	0
host_issues	131040
host_last_known_locations	0
host_mdm	78605
host_mdm_actions	0
host_mdm_android_profiles	0
host_mdm_apple_awaiting_configuration	0
host_mdm_apple_bootstrap_packages	0
host_mdm_apple_declarations	0
host_mdm_apple_profiles	0
host_mdm_commands	0
host_mdm_idp_accounts	0
host_mdm_managed_certificates	0
host_mdm_windows_profiles	0
host_munki_info	56219
host_munki_issues	301126
host_operating_system	112198
host_orbit_info	0
host_scim_user	0
host_script_results	0
host_seen_times	100464
host_software	80336546
host_software_installed_paths	744085
host_software_installs	0
host_updates	99323
host_users	2091727
host_vpp_software_installs	0
hosts	102189
in_house_app_labels	0
in_house_app_software_categories	0
in_house_app_upcoming_activities	0
in_house_apps	0
invite_teams	0
invites	0
jobs	12
kernel_host_counts	8752
label_membership	2513387
labels	30
legacy_host_filevault_profiles	0
legacy_host_mdm_enroll_refs	0
legacy_host_mdm_idp_accounts	0
locks	26
mdm_android_configuration_profiles	0
mdm_apple_bootstrap_packages	0
mdm_apple_configuration_profiles	0
mdm_apple_declaration_activation_references	0
mdm_apple_declarations	0
mdm_apple_declarative_requests	0
mdm_apple_default_setup_assistants	0
mdm_apple_enrollment_profiles	0
mdm_apple_installers	0
mdm_apple_setup_assistant_profiles	0
mdm_apple_setup_assistants	0
mdm_config_assets	7
mdm_configuration_profile_labels	0
mdm_configuration_profile_variables	0
mdm_declaration_labels	0
mdm_delivery_status	4
mdm_idp_accounts	0
mdm_operation_types	2
mdm_windows_configuration_profiles	0
mdm_windows_enrollments	0
microsoft_compliance_partner_host_statuses	0
microsoft_compliance_partner_integrations	0
migration_status_data	9
migration_status_tables	471
mobile_device_management_solutions	0
munki_issues	395467
nano_cert_auth_associations	0
nano_command_results	0
nano_commands	0
nano_dep_names	0
nano_devices	0
nano_enrollment_queue	0
nano_enrollments	0
nano_push_certs	0
nano_users	0
nano_view_queue	0
network_interfaces	0
operating_system_version_vulnerabilities	265214
operating_system_vulnerabilities	1935
operating_systems	29
osquery_options	1
pack_targets	8
packs	6
password_reset_requests	0
policies	6
policy_automation_iterations	0
policy_labels	0
policy_membership	674313
policy_stats	42
queries	734
query_labels	0
query_results	0
scep_certificates	0
scep_serials	0
scheduled_queries	8
scheduled_query_stats	1300311
scim_groups	0
scim_last_request	0
scim_user_emails	0
scim_user_group	0
scim_users	0
script_contents	0
script_upcoming_activities	0
scripts	0
secret_variables	0
sessions	41
setup_experience_scripts	0
setup_experience_status_results	0
software	615304
software_categories	6
software_cpe	44429
software_cve	954572
software_host_counts	1801710
software_install_upcoming_activities	0
software_installer_labels	0
software_installer_software_categories	0
software_installers	0
software_title_display_names	0
software_title_icons	0
software_titles	353931
software_titles_host_counts	655948
software_update_schedules	0
statistics	1
teams	5
upcoming_activities	0
user_teams	0
users	2
users_deleted	0
verification_tokens	0
vpp_app_team_labels	0
vpp_app_team_software_categories	0
vpp_app_upcoming_activities	0
vpp_apps	0
vpp_apps_teams	0
vpp_token_teams	0
vpp_tokens	0
vulnerability_host_counts	59400
windows_mdm_command_queue	0
windows_mdm_command_results	0
windows_mdm_commands	0
windows_mdm_responses	0
windows_updates	100105
wstep_cert_auth_associations	0
wstep_certificates	0
wstep_serials	0
yara_rules	0
</details>

##### For Fleet < 4.81

The query used for this check is

```sql
SELECT table_name, COALESCE(table_rows, 0) table_rows
FROM information_schema.tables
WHERE table_schema = (SELECT DATABASE());
```

which can be run directly on a MySQL reader, in case a self-hosted customer wants to pull this data without running the migration command, or if they are using a Fleet version prior to 4.81.

##### Cloud environments

For cloud environments on >= 4.81, you can scan CloudWatch Logs for the appropriate row counts line.  Run [cw-table-row-counts.sh](https://github.com/fleetdm/confidential/blob/main/infrastructure/cloud/scripts/cw-table-row-counts.sh) to pull the table stats log entry from the most recent migration.

```shell
scripts/cw-table-row-counts.sh --days <days to search back in logs> --log-group <customer group name>
```

The script will pull from the most recent database migration within the --days.  If you don't know when the last migration was run, 30 is safe number to use given our release cadence.  Otherwise you can check the most recent runs of the [cloud deploy](https://github.com/fleetdm/confidential/actions/workflows/cloud-deploy.yml) Github action for a timeframe.

##### Self-hosted

For self-hosted environments on >= 4.81, running `fleet prepare` with the `--with-table-stats` will provide this information in real time.

This command is safe to run without taking systems offline as the migrations themselves are a no-op in those cases and we pull approximate row counts from MySQL's `information_schema` table to get close-enough numbers with minimal overhead.

##### Compare with load test benchmark data

Here's an example of a load test envvironment's row counts by table, updated 2026-XX-YY:

```
TODO
```

##### 4. TODO
