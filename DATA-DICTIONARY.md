# Fleet Database Data Dictionary

This document describes all major tables in Fleet's database schema, their purpose, relationships, and usage patterns. This is the authoritative source for understanding Fleet's database architecture when speccing stories.

**Last Updated:** 2026-04-01
**Schema Version:** As of latest migration in server/datastore/mysql/migrations/tables/
**Total Tables:** 193

---

## Quick Reference

### Table Categories
- [Core Tables](#core-tables) - Hosts, teams, users, queries, labels
- [Software Tables](#software-tables) - Software catalog, installers, vulnerabilities
- [MDM Tables](#mdm-tables) - Apple/Windows/Android device management
- [Policy Tables](#policy-tables) - Policy definitions and compliance tracking
- [Activity Tables](#activity-tables) - Audit logs and activity tracking
- [Configuration Tables](#configuration-tables) - Settings and integrations
- [Query & Pack Tables](#query--pack-tables) - Live and scheduled queries
- [Certificate Tables](#certificate-tables) - Certificate management (SCEP, identity certs)

### Platform-Specific Table Groups
- **Apple (macOS, iOS, iPadOS):** `abm_tokens`, `mdm_apple_*`, `host_mdm_apple_*`, `vpp_*`, `in_house_apps`, `nano_*`, `host_munki_*`
- **Windows:** `mdm_windows_*`, `host_mdm_windows_*`, `windows_mdm_*`, `windows_updates`, `wstep_*`
- **Android:** `android_*`, `mdm_android_*`, `host_mdm_android_*`
- **Common/Cross-Platform:** `hosts`, `host_software`, `policies`, `activity_past`, `teams`, `users`, `software_*`, `operating_systems`

---

## Core Tables

### hosts

**Purpose:** Central table storing information about all enrolled devices (workstations, laptops, servers, mobile devices) across all platforms.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT) - Primary key
- `osquery_host_id` (varchar(255), UNIQUE, nullable) - Legacy osquery identifier
- `uuid` (varchar(255), INDEXED) - Device UUID
- `hardware_serial` (varchar(255), INDEXED) - Device serial number
- `platform` (varchar(255), INDEXED) - OS platform: darwin, windows, ubuntu, ios, ipados, chrome, android, etc.
- `os_version` (varchar(255)) - Operating system version string
- `hostname` (varchar(255)) - Device hostname
- `computer_name` (varchar(255)) - User-visible computer name
- `team_id` (int unsigned, FK, nullable) - Foreign key to teams table (NULL = "No team"/global)
- `node_key` (varchar(255), UNIQUE, BINARY COLLATE) - Osquery enrollment key
- `orbit_node_key` (varchar(255), UNIQUE, BINARY COLLATE) - Orbit agent enrollment key
- `primary_ip` (varchar(45)) - Primary IP address (IPv4 or IPv6)
- `primary_mac` (varchar(17)) - Primary MAC address
- `created_at` (timestamp, DEFAULT CURRENT_TIMESTAMP) - Record creation time
- `updated_at` (timestamp, DEFAULT CURRENT_TIMESTAMP ON UPDATE) - Last update time
- `last_enrolled_at` (timestamp, DEFAULT CURRENT_TIMESTAMP) - Last enrollment time
- `detail_updated_at` (timestamp, nullable) - Last time host details were refreshed
- `label_updated_at` (timestamp, DEFAULT '2000-01-01 00:00:00') - Last label membership update
- `policy_updated_at` (timestamp, DEFAULT '2000-01-01 00:00:00') - Last policy check
- `refetch_requested` (tinyint(1), DEFAULT 0) - Flag to trigger detail refetch
- `refetch_critical_queries_until` (timestamp, nullable) - Refetch until this timestamp
- `last_restarted_at` (datetime(6), DEFAULT '0001-01-01') - Last time host was restarted (calculated from uptime)
- `timezone` (varchar(255), nullable) - Host's timezone (for software auto-update scheduling)

**Relationships:**
- Many-to-one with `teams` via `team_id` (ON DELETE SET NULL)
- One-to-many with `host_software` via `host_id` (ON DELETE CASCADE)
- One-to-one with `host_mdm` via `host_id` (ON DELETE CASCADE)
- One-to-one with `host_additional` via `host_id`
- One-to-many with `policy_membership` via `host_id` (ON DELETE CASCADE)
- One-to-many with `activities` via related host activities
- Many-to-many with `labels` via `label_membership` (ON DELETE CASCADE)
- One-to-many with `host_users`, `host_batteries`, `host_disks`, `host_emails`, etc.

**Indexes:**
- Primary key on `id`
- Unique index on `osquery_host_id`
- Unique index on `node_key`
- Unique index on `orbit_node_key`
- Index on `team_id` (`fk_hosts_team_id`) - for team-scoped queries
- Index on `platform` (`hosts_platform_idx`) - for platform-specific queries
- Index on `hardware_serial` (`idx_hosts_hardware_serial`) - for serial lookups
- Index on `uuid` (`idx_hosts_uuid`) - for UUID lookups
- Index on `hostname` (`idx_hosts_hostname`) - for hostname lookups

**Common Query Patterns:**
```sql
-- Get host by UUID
SELECT * FROM hosts WHERE uuid = ?;

-- List hosts in a team
SELECT * FROM hosts WHERE team_id = ?;

-- Count hosts by platform
SELECT platform, COUNT(*) AS count
FROM hosts
GROUP BY platform;

-- Find hosts without a team (global scope)
SELECT * FROM hosts WHERE team_id IS NULL;

-- Get hosts by serial number
SELECT * FROM hosts WHERE hardware_serial = ?;

-- Get recently enrolled hosts
SELECT * FROM hosts
WHERE last_enrolled_at > DATE_SUB(NOW(), INTERVAL 24 HOUR);

-- Get hosts needing detail refetch
SELECT * FROM hosts
WHERE refetch_requested = 1
   OR refetch_critical_queries_until > NOW();
```

**Usage Notes:**
- iOS/iPadOS/Android hosts use the same table as desktop platforms
- Mobile devices have `platform` = 'ios', 'ipados', or 'android'
- Chrome devices have `platform` = 'chrome'
- `team_id` NULL means host is in "No team" (global scope)
- `node_key` and `orbit_node_key` are used for different agent types (osquery vs Orbit)
- The `platform` field determines which MDM features are available
- `detail_updated_at` tracks when osquery detail queries were last run
- `label_updated_at` and `policy_updated_at` optimize update scheduling

**Platform Affinity:** Common (all platforms)

**Related Migrations:**
- Initial: `20161118212528_CreateTableHosts.go`
- Various migrations add fields over time

---

### teams

**Purpose:** Multi-tenancy support - organizational units for grouping hosts with separate configurations, policies, and software.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT) - Primary key
- `name` (varchar(255), UNIQUE) - Team name (unique, case-insensitive via virtual column)
- `name_bin` (varchar(255), VIRTUAL, UNIQUE) - Binary collation virtual column for case-sensitive uniqueness
- `description` (varchar(1023), DEFAULT '') - Team description
- `config` (json, nullable) - Team-specific configuration (agent options, integrations, etc.)
- `filename` (varchar(255), nullable, UNIQUE) - GitOps filename reference
- `created_at` (timestamp, DEFAULT CURRENT_TIMESTAMP) - Creation timestamp

**Relationships:**
- One-to-many with `hosts` via `team_id` (host.team_id references teams.id)
- One-to-many with `software_installers` via `team_id` (ON DELETE CASCADE)
- One-to-many with `policies` via `team_id`
- One-to-many with `mdm_apple_configuration_profiles` via `team_id` (ON DELETE CASCADE)
- One-to-many with `mdm_windows_configuration_profiles` via `team_id` (ON DELETE CASCADE)
- One-to-many with `vpp_apps_teams` via `team_id` (ON DELETE CASCADE)
- Many-to-many with `users` via `user_teams` (team-based roles)

**Indexes:**
- Primary key on `id`
- Unique index on `filename` (`idx_teams_filename`)
- Unique index on `name_bin` (`idx_name_bin`) - ensures case-insensitive uniqueness

**Common Query Patterns:**
```sql
-- Get team by name
SELECT * FROM teams WHERE name = ?;

-- Get team with host count
SELECT t.*, COUNT(h.id) AS host_count
FROM teams t
LEFT JOIN hosts h ON h.team_id = t.id
GROUP BY t.id;

-- Get all teams user has access to
SELECT t.* FROM teams t
INNER JOIN user_teams ut ON ut.team_id = t.id
WHERE ut.user_id = ?;

-- Get global ("No team") scope
-- (Represented by team_id = NULL in other tables)
```

**Usage Notes:**
- NULL team_id in other tables (hosts, policies, installers) = "No team" (global scope)
- Team configurations (agent options, integrations, MDM settings) stored in `config` JSON field
- `name_bin` virtual column ensures case-insensitive uniqueness
- GitOps uses `filename` to track team definitions
- Team deletion sets `team_id` to NULL in hosts (ON DELETE SET NULL)
- Most team-scoped resources cascade delete when team is deleted

**Platform Affinity:** Common (all platforms)

**Related Migrations:**
- Initial: `20210601000001_CreateTeamsTables.go`

---

### users

**Purpose:** Fleet user accounts with authentication, authorization, and audit tracking.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT) - Primary key
- `created_at` (timestamp, DEFAULT CURRENT_TIMESTAMP) - Account creation time
- `updated_at` (timestamp, DEFAULT CURRENT_TIMESTAMP ON UPDATE) - Last update time
- `password` (varbinary(255)) - Bcrypt hashed password
- `salt` (varchar(255)) - Password salt
- `name` (varchar(255)) - User's full name
- `email` (varchar(255), UNIQUE) - Email address (unique identifier)
- `admin_forced_password_reset` (tinyint(1), DEFAULT 0) - Admin forces password reset on next login
- `gravatar_url` (varchar(255)) - Gravatar profile picture URL
- `position` (varchar(255)) - Job title/position
- `sso_enabled` (tinyint, DEFAULT 0) - Whether SSO authentication is enabled for this user
- `global_role` (varchar(64), nullable) - Global role: admin, maintainer, observer, gitops, or NULL
- `api_only` (tinyint(1), DEFAULT 0) - API-only user (no UI access)
- `mfa_enabled` (tinyint(1), DEFAULT 0) - Whether multi-factor authentication is enabled
- `settings` (json, DEFAULT '{}') - User-specific settings
- `invite_id` (int unsigned, UNIQUE, nullable) - Reference to originating invite

**Relationships:**
- One-to-many with `activities` via `user_id` (ON DELETE SET NULL)
- One-to-many with `sessions` via `user_id` (ON DELETE CASCADE)
- Many-to-many with `teams` via `user_teams` (team-specific roles)
- One-to-many with `invites` as inviter
- One-to-many with `policies` via `author_id` (ON DELETE SET NULL)
- One-to-many with `queries` via `author_id` (ON DELETE SET NULL)

**Indexes:**
- Primary key on `id`
- Unique index on `email`
- Index on `name` (`idx_users_name`) - NEW: for user name search

**Common Query Patterns:**
```sql
-- Get user by email
SELECT * FROM users WHERE email = ?;

-- Get user with their teams and roles
SELECT u.*, ut.team_id, ut.role
FROM users u
LEFT JOIN user_teams ut ON ut.user_id = u.id
WHERE u.id = ?;

-- Get all admins
SELECT * FROM users WHERE global_role = 'admin';

-- Get API-only users
SELECT * FROM users WHERE api_only = 1;

-- Get SSO users
SELECT * FROM users WHERE sso_enabled = 1;
```

**Usage Notes:**
- `global_role` NULL = user has team-specific permissions only (defined in `user_teams`)
- API tokens are linked to users (JWT contains user ID)
- SSO users have `sso_enabled` = 1 and may have NULL password
- GitOps users have `global_role` = 'gitops' and typically `api_only` = 1
- When user is deleted, activities preserve user name/email but set `user_id` to NULL
- Gravatar URLs generated from email hash

**Platform Affinity:** Common (all platforms)

**Related Migrations:**
- Initial: `20161118212649_CreateTableUsers.go`
- SSO: `20170509132100_AddSSOFlagToUser.go`
- API-only: `20210616163757_AddApiOnlyToUser.go`

---

### host_mdm

**Purpose:** Tracks MDM enrollment status and server information for each host. Central table for understanding whether a host is managed and by which MDM solution.

**Key Fields:**
- `host_id` (int unsigned, PK, FK) - One-to-one with hosts table
- `enrolled` (tinyint(1), DEFAULT 0) - Currently enrolled in any MDM
- `server_url` (varchar(255)) - MDM server URL
- `mdm_id` (int unsigned, FK, nullable) - Foreign key to `mobile_device_management_solutions`
- `is_server` (tinyint(1), nullable, DEFAULT NULL) - Whether enrolled in Fleet's MDM (vs third-party)
- `installed_from_dep` (tinyint(1), DEFAULT 0) - Whether enrolled via Apple DEP/ADE or similar automated enrollment
- `fleet_enroll_ref` (varchar(36), DEFAULT '') - Fleet enrollment reference identifier
- `is_personal_enrollment` (tinyint(1), DEFAULT 0) - Whether host enrolled via personal MDM enrollment
- `created_at` (timestamp(6), DEFAULT CURRENT_TIMESTAMP(6))
- `updated_at` (timestamp(6), DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE)

**Relationships:**
- One-to-one with `hosts` via `host_id` (ON DELETE CASCADE)
- Many-to-one with `mobile_device_management_solutions` via `mdm_id`

**Indexes:**
- Primary key on `host_id`
- Index on `mdm_id` (`host_mdm_mdm_id_idx`)
- Composite index on `(enrolled, installed_from_dep, is_personal_enrollment)`

**Common Query Patterns:**
```sql
-- Get hosts enrolled in Fleet MDM
SELECT h.* FROM hosts h
INNER JOIN host_mdm hm ON hm.host_id = h.id
WHERE hm.is_server = 1 AND hm.enrolled = 1;

-- Get hosts enrolled via DEP/ADE
SELECT h.* FROM hosts h
INNER JOIN host_mdm hm ON hm.host_id = h.id
WHERE hm.installed_from_dep = 1;

-- Get hosts by MDM solution
SELECT h.* FROM hosts h
INNER JOIN host_mdm hm ON hm.host_id = h.id
WHERE hm.mdm_id = ?;

-- Find unenrolled hosts
SELECT h.* FROM hosts h
LEFT JOIN host_mdm hm ON hm.host_id = h.id
WHERE hm.host_id IS NULL OR hm.enrolled = 0;
```

**Usage Notes:**
- Only exists for hosts that have MDM information
- `is_server` = 1 means Fleet is the MDM server
- `is_server` NULL or 0 with `enrolled` = 1 means third-party MDM
- Used to determine which MDM features are available
- `installed_from_dep` indicates automated enrollment (Apple DEP, Windows Autopilot equivalent)
- iOS/iPadOS/Android hosts will have this record if MDM-enrolled

**Platform Affinity:** Common (primarily Apple, Windows, Android MDM-capable platforms)

**Related Migrations:**
- MDM support added incrementally across many migrations

---

### host_last_known_locations

**Purpose:** Tracks last known GPS location for hosts. Enables location-based features like Find My and compliance reporting.

**Key Fields:**
- `host_id` (int unsigned, PK) - One-to-one with hosts (primary key)
- `latitude` (decimal(10,8), nullable) - GPS latitude (-90.00000000 to 90.00000000)
- `longitude` (decimal(11,8), nullable) - GPS longitude (-180.00000000 to 180.00000000)
- `created_at` (timestamp(6)) - When location was first recorded
- `updated_at` (timestamp(6)) - When location was last updated

**Relationships:**
- One-to-one with `hosts` via `host_id`

**Indexes:**
- Primary key on `host_id`

**Common Query Patterns:**
```sql
-- Get host location
SELECT h.hostname, hlkl.latitude, hlkl.longitude, hlkl.updated_at
FROM hosts h
LEFT JOIN host_last_known_locations hlkl ON hlkl.host_id = h.id
WHERE h.id = ?;

-- Get hosts with location data
SELECT h.*, hlkl.latitude, hlkl.longitude
FROM hosts h
INNER JOIN host_last_known_locations hlkl ON hlkl.host_id = h.id
WHERE hlkl.latitude IS NOT NULL;
```

**Usage Notes:**
- Location obtained via MDM commands (e.g., Find My Mac, Location Services)
- Privacy-sensitive data - access should be carefully controlled
- Supports "Find My" functionality for lost/stolen devices
- Precision: latitude up to 8 decimal places (~1mm accuracy)
- May be NULL if location services disabled or unavailable

**Platform Affinity:** Apple (macOS, iOS/iPadOS with MDM)

---

## Software Tables

### software_titles

**Purpose:** Catalog of all software titles discovered or added to Fleet. Represents software independent of specific versions (e.g., "Google Chrome" as a title, with multiple versions).

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT) - Primary key
- `name` (varchar(255), INDEXED) - Software title name
- `source` (varchar(64)) - Detection source: `programs`, `apps`, `deb_packages`, `rpm_packages`, `portage_packages`, `npm_packages`, `atom_packages`, `python_packages`, `chrome_extensions`, `firefox_addons`, `safari_extensions`, `homebrew_packages`, `ios_apps`, `ipados_apps`, `vpp`, `custom_installer`
- `extension_for` (varchar(255), DEFAULT '') - Browser name (for extensions only): chrome, firefox, safari, edge, etc.
- `bundle_identifier` (varchar(255), nullable) - macOS/iOS bundle identifier (com.example.app)
- `application_id` (varchar(255), nullable) - Android application package name
- `upgrade_code` (char(38), nullable) - Windows MSI upgrade code (for Windows software matching)
- `is_kernel` (tinyint(1), DEFAULT 0) - Whether software is a kernel package (for kernel vulnerability tracking)
- `additional_identifier` (tinyint unsigned, VIRTUAL) - Generated from source type for bundle_identifier uniqueness
- `unique_identifier` (varchar(255), VIRTUAL) - Generated: COALESCE(bundle_identifier, application_id, upgrade_code, name)

**Relationships:**
- One-to-many with `software` via `title_id` (specific versions)
- One-to-many with `software_installers` via `title_id` (ON DELETE SET NULL)
- One-to-many with `vpp_apps` via `title_id` (ON DELETE SET NULL)
- One-to-many with `in_house_apps` via `title_id`
- One-to-many with `software_titles_host_counts` via `software_title_id`
- One-to-many with `software_title_display_names` via `software_title_id`
- One-to-many with `software_update_schedules` via `title_id`

**Indexes:**
- Primary key on `id`
- Composite indexes for efficient queries by name/source/browser

**Common Query Patterns:**
```sql
-- Find software title by name
SELECT * FROM software_titles WHERE name = ?;

-- Get all browser extensions
SELECT * FROM software_titles
WHERE source IN ('chrome_extensions', 'firefox_addons', 'safari_extensions');

-- Get titles with installers available
SELECT st.* FROM software_titles st
INNER JOIN software_installers si ON si.title_id = st.id;

-- Get host count for a title
SELECT st.*, sthc.hosts_count
FROM software_titles st
LEFT JOIN software_titles_host_counts sthc ON sthc.software_title_id = st.id
WHERE st.id = ?;
```

**Usage Notes:**
- Single source of truth for software names
- Deduplicated across different versions
- `source` indicates how software was detected (osquery table or manual addition)
- Browser extensions have non-null `browser` field
- Used for software inventory, vulnerability tracking, and deployment
- iOS/iPadOS apps have `source` = 'ios_apps' or 'ipados_apps'
- **Windows Software:** `upgrade_code` (NEW) enables matching Windows MSI packages across versions
- `unique_identifier` computed from available identifiers for software matching

**Platform Affinity:** Common (all platforms)

**Related Migrations:**
- Software inventory introduced in early migrations
- Evolved from `software` table to support software titles

---

### software_installers

**Purpose:** Manages custom software packages and VPP apps available for installation on hosts via Fleet. Supports self-service installation and automated deployment.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT) - Primary key
- `team_id` (int unsigned, FK, nullable) - Target team (NULL = "No team"/global)
- `global_or_team_id` (int unsigned, DEFAULT 0) - Compound identifier (0 for global, team_id for teams)
- `title_id` (int unsigned, FK, nullable) - Foreign key to software_titles (ON DELETE SET NULL)
- `filename` (varchar(255)) - Installer filename
- `version` (varchar(255)) - Software version
- `platform` (varchar(255), INDEXED) - Target platform: darwin, windows, linux, ios, ipados
- `storage_id` (varchar(64)) - Reference to blob storage (S3, etc.)
- `url` (varchar(4095), DEFAULT '') - Download URL for custom packages
- `package_ids` (text) - JSON array of VPP app IDs (adam_ids) for App Store apps
- `extension` (varchar(32), DEFAULT '') - File extension (.pkg, .msi, .deb, .rpm, .ipa, etc.)
- `pre_install_query` (text, nullable) - osquery SQL query to run before installation
- `install_script_content_id` (int unsigned, FK) - Foreign key to script_contents
- `post_install_script_content_id` (int unsigned, FK, nullable) - Post-install script
- `uninstall_script_content_id` (int unsigned, FK) - Uninstall script
- `self_service` (tinyint(1), DEFAULT 0) - Available on /device page for end users
- `install_during_setup` (tinyint(1), DEFAULT 0) - Install during DEP/setup experience
- `fleet_maintained_app_id` (int unsigned, FK, nullable) - Reference to fleet_maintained_apps
- `upgrade_code` (varchar(48), DEFAULT '') - MSI upgrade code (Windows)
- `user_id` (int unsigned, FK, nullable) - User who uploaded (ON DELETE SET NULL)
- `user_name` (varchar(255), DEFAULT '') - Uploader name (denormalized for audit)
- `user_email` (varchar(255), DEFAULT '') - Uploader email (denormalized for audit)
- `patch_query` (text, NOT NULL) - osquery SQL query used for patch policy evaluation
- `is_active` (tinyint(1), NOT NULL, DEFAULT 0) - Whether this installer version is the active one for its title (supports multiple versions per title)
- `uploaded_at` (timestamp(6), DEFAULT CURRENT_TIMESTAMP(6))
- `updated_at` (timestamp(6), DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE)

**Relationships:**
- Many-to-one with `teams` via `team_id` (ON DELETE CASCADE)
- Many-to-one with `software_titles` via `title_id` (ON DELETE SET NULL)
- Many-to-one with `script_contents` via install/post-install/uninstall script IDs (ON DELETE RESTRICT)
- Many-to-one with `fleet_maintained_apps` via `fleet_maintained_app_id` (ON DELETE SET NULL)
- One-to-many with `host_software_installs` via `software_installer_id`
- Many-to-many with `labels` via `software_installer_labels` for targeting
- Many-to-many with `software_categories` via `software_installer_software_categories`

**Indexes:**
- Primary key on `id`
- Unique index on `(global_or_team_id, title_id, version)` (`idx_software_installers_team_title_version`) - allows multiple versions per title per team with unique version constraint
- Index on `team_id` (`fk_software_installers_team_id`)
- Index on `title_id`, `install_script_content_id`, `post_install_script_content_id`, etc. for FK joins

**Common Query Patterns:**
```sql
-- Get self-service software for a team
SELECT si.*, st.name
FROM software_installers si
INNER JOIN software_titles st ON st.id = si.title_id
WHERE si.team_id = ? AND si.self_service = 1;

-- Get iOS/iPadOS VPP apps (App Store apps)
SELECT si.* FROM software_installers si
WHERE si.platform IN ('ios', 'ipados')
  AND JSON_LENGTH(si.package_ids) > 0;

-- Get custom packages (non-VPP)
SELECT si.* FROM software_installers si
WHERE si.platform = ?
  AND (si.package_ids = '' OR si.package_ids = '[]');

-- Get installers for a specific title
SELECT si.* FROM software_installers si
WHERE si.title_id = ? AND si.team_id = ?;

-- Get all installers in global scope
SELECT si.* FROM software_installers si
WHERE si.team_id IS NULL;
```

**Usage Notes:**
- **VPP apps:** `package_ids` contains JSON array of adam_ids, `url` is empty, used for iOS/iPadOS/macOS App Store apps
- **Custom packages:** `url` may contain download link (or use `storage_id`), `package_ids` is empty
- **Self-service flag:** Controls visibility on My Device page (`/device/{token}`)
- **Install during setup:** For DEP enrollment (macOS) or setup experience (iOS/iPadOS)
- Used by `/device/{token}/software/install/{id}` endpoint
- Scripts stored in `script_contents` table (deduplicated by content hash)
- `pre_install_query` can check host state before installation (e.g., check if already installed)
- Platform determines which MDM commands are used (InstallApplication for Apple, etc.)
- Fleet-maintained apps (automatic updates) reference `fleet_maintained_apps`
- **Multiple Versions (NEW):** `is_active` flag enables storing multiple installer versions per title; only one version is active at a time for deployment
- **FMA Active Installers:** Fleet-maintained apps can now have inactive installer versions retained for rollback or auditing

**Platform Affinity:** Common (platform-agnostic management, targets specific platforms)

**Related Migrations:**
- Initial software installer support
- Self-service: added `self_service` column
- VPP support: added `package_ids` for VPP apps
- FMA Active Installers: `20260218175704_FMAActiveInstallers.go` - added `is_active`, changed unique index to include version

---

### vpp_apps

**Purpose:** App Store app metadata from VPP (Volume Purchase Program) and Android managed apps. Synced from Apple's VPP service and Google Play for macOS, iOS/iPadOS, and Android apps.

**Key Fields:**
- `adam_id` (varchar(255), PK, composite) - App identifier (ADAM ID for Apple, package name for Android)
- `platform` (varchar(10), PK, composite) - Platform: darwin, ios, ipados, android
- `title_id` (int unsigned, FK, nullable) - Foreign key to software_titles (ON DELETE SET NULL)
- `bundle_identifier` (varchar(255), DEFAULT '') - App bundle ID (com.example.app)
- `name` (varchar(255), DEFAULT '') - App name from App Store
- `icon_url` (varchar(255), DEFAULT '') - App icon URL
- `latest_version` (varchar(255), DEFAULT '') - Latest available version
- `created_at` (timestamp, DEFAULT CURRENT_TIMESTAMP)
- `updated_at` (timestamp, DEFAULT CURRENT_TIMESTAMP ON UPDATE)

**Relationships:**
- Many-to-one with `software_titles` via `title_id` (ON DELETE SET NULL)
- One-to-many with `vpp_apps_teams` via `(adam_id, platform)` (ON DELETE CASCADE)
- Referenced by `software_installers.package_ids` (stored as JSON array, not FK)

**Indexes:**
- Primary key on `(adam_id, platform)` - composite key for multi-platform apps
- Index on `title_id`

**Common Query Patterns:**
```sql
-- Get VPP app by ADAM ID and platform
SELECT * FROM vpp_apps
WHERE adam_id = ? AND platform = ?;

-- Get all iOS apps
SELECT * FROM vpp_apps WHERE platform = 'ios';

-- Get VPP app with deployment info
SELECT va.*, vat.team_id, vat.self_service
FROM vpp_apps va
INNER JOIN vpp_apps_teams vat ON vat.adam_id = va.adam_id AND vat.platform = va.platform
WHERE va.adam_id = ?;

-- Search VPP apps by name
SELECT * FROM vpp_apps
WHERE name LIKE CONCAT('%', ?, '%');
```

**Usage Notes:**
- Synced from Apple's VPP service via API (Apple platforms) or Google Play (Android)
- Contains metadata for macOS, iOS/iPadOS, and Android apps
- `adam_id` is the App Store identifier for Apple or package name for Android
- Same app can exist for multiple platforms (e.g., iOS and iPadOS)
- Used for displaying app info in UI when deploying VPP apps
- Deployment configured via `vpp_apps_teams` (which teams get which apps)
- Icons fetched from Apple's CDN or Google Play via `icon_url`
- **Android Support:** `adam_id` expanded to varchar(255) to support Android package names

**Platform Affinity:** Apple (macOS, iOS, iPadOS), Android

**Related Migrations:**
- VPP support added for iOS/iPadOS and macOS apps

---

### vpp_apps_teams

**Purpose:** Configures deployment of VPP apps to teams. Controls which App Store apps are available to which teams and whether they're self-service.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT) - Primary key
- `adam_id` (varchar(255), FK, composite) - Foreign key to vpp_apps (expanded for Android package names)
- `platform` (varchar(10), FK, composite) - Platform (must match vpp_apps)
- `team_id` (int unsigned, FK, nullable) - Target team (NULL = "No team"/global)
- `global_or_team_id` (int, DEFAULT 0) - Compound identifier (0 for global, team_id for teams)
- `self_service` (tinyint(1), DEFAULT 0) - Available for self-service installation
- `install_during_setup` (tinyint(1), DEFAULT 0) - Install during device setup
- `vpp_token_id` (int unsigned, FK, nullable) - Foreign key to vpp_tokens (nullable for Android apps)
- `created_at` (timestamp(6), DEFAULT CURRENT_TIMESTAMP(6))
- `updated_at` (timestamp(6), DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE)

**Relationships:**
- Many-to-one with `vpp_apps` via `(adam_id, platform)` (ON DELETE CASCADE)
- Many-to-one with `teams` via `team_id` (ON DELETE CASCADE)
- Many-to-one with `vpp_tokens` via `vpp_token_id` (ON DELETE CASCADE)
- Many-to-many with `labels` via `vpp_app_team_labels` for targeting
- Many-to-many with `software_categories` via `vpp_app_team_software_categories`

**Indexes:**
- Primary key on `id`
- Unique index on `(global_or_team_id, adam_id, platform)` - one deployment config per app per team
- Index on `team_id`
- Index on `(adam_id, platform)`
- Index on `vpp_token_id`

**Common Query Patterns:**
```sql
-- Get VPP apps deployed to a team
SELECT va.*, vat.self_service, vat.install_during_setup
FROM vpp_apps_teams vat
INNER JOIN vpp_apps va ON va.adam_id = vat.adam_id AND va.platform = vat.platform
WHERE vat.team_id = ?;

-- Get self-service VPP apps for a team
SELECT va.* FROM vpp_apps_teams vat
INNER JOIN vpp_apps va ON va.adam_id = vat.adam_id AND va.platform = vat.platform
WHERE vat.team_id = ? AND vat.self_service = 1;

-- Check if app is deployed to team
SELECT * FROM vpp_apps_teams
WHERE adam_id = ? AND platform = ? AND team_id = ?;
```

**Usage Notes:**
- Bridges `vpp_apps` (catalog) with deployment configuration
- Self-service apps appear on My Device page for end users
- `install_during_setup` triggers installation during DEP enrollment
- VPP token required for app distribution (licensed via VPP)
- Label targeting allows deploying to subset of team's hosts
- Software categories for organization in UI

**Platform Affinity:** Apple (iOS, iPadOS, macOS)

---

### in_house_apps

**Purpose:** Custom in-house enterprise apps for iOS/iPadOS (.ipa files). These are custom apps not distributed via the App Store.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT) - Primary key
- `title_id` (int unsigned, FK, nullable) - Foreign key to software_titles
- `team_id` (int unsigned, FK, nullable) - Target team (NULL = global)
- `global_or_team_id` (int unsigned, DEFAULT 0) - Compound identifier
- `filename` (varchar(255)) - App filename (.ipa)
- `version` (varchar(255)) - App version
- `bundle_identifier` (varchar(255)) - Bundle ID (com.company.app)
- `storage_id` (varchar(64)) - Storage reference for .ipa file
- `platform` (varchar(10)) - Platform: ios or ipados
- `self_service` (tinyint(1), DEFAULT 0) - Self-service availability
- `url` (varchar(4095), DEFAULT '') - External URL for the app (NEW - for URL-based distribution)
- `created_at` (timestamp, DEFAULT CURRENT_TIMESTAMP)
- `updated_at` (timestamp, DEFAULT CURRENT_TIMESTAMP ON UPDATE)

**Relationships:**
- Many-to-one with `software_titles` via `title_id`
- Many-to-one with `teams` via `team_id`
- Many-to-many with `labels` via `in_house_app_labels`
- Many-to-many with `software_categories` via `in_house_app_software_categories` (NEW)
- One-to-many with `host_in_house_software_installs` for tracking installations

**Indexes:**
- Primary key on `id`
- Unique index on `(global_or_team_id, filename, platform)` - one app per filename per team

**Common Query Patterns:**
```sql
-- Get in-house apps for a team
SELECT * FROM in_house_apps WHERE team_id = ?;

-- Get self-service in-house apps
SELECT * FROM in_house_apps
WHERE team_id = ? AND self_service = 1;

-- Get iOS-specific apps
SELECT * FROM in_house_apps WHERE platform = 'ios';
```

**Usage Notes:**
- Used for distributing custom enterprise iOS/iPadOS apps
- Requires Apple Enterprise Developer Program membership
- Apps signed with enterprise certificate
- Distributed via MDM InstallApplication command
- Not available on App Store
- Similar workflow to custom software installers but iOS/iPadOS-specific
- **URL Distribution (NEW):** `url` field enables downloading apps from external URLs

**Platform Affinity:** Apple (iOS, iPadOS only)

---

### in_house_app_software_categories

**Purpose:** Junction table linking in-house apps to software categories for organization.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT) - Primary key
- `software_category_id` (int unsigned, FK) - Reference to software_categories (ON DELETE CASCADE)
- `in_house_app_id` (int unsigned, FK) - Reference to in_house_apps (ON DELETE CASCADE)
- `created_at` (datetime(6))

**Relationships:**
- Many-to-one with `in_house_apps` via `in_house_app_id` (ON DELETE CASCADE)
- Many-to-one with `software_categories` via `software_category_id` (ON DELETE CASCADE)

**Indexes:**
- Primary key on `id`
- Unique index on `(in_house_app_id, software_category_id)` - one category per app

**Usage Notes:**
- Allows categorizing in-house apps (Productivity, Utilities, Security, etc.)
- Categories are defined in `software_categories` table
- NEW in v4.x: Security and Utilities categories added

**Platform Affinity:** Apple (iOS, iPadOS)

---

### host_software

**Purpose:** Junction table tracking which software versions are installed on which hosts. Powers software inventory and vulnerability tracking.

**Key Fields:**
- `host_id` (int unsigned, PK, composite) - Foreign key to hosts
- `software_id` (bigint unsigned, PK, composite) - Foreign key to software (specific version)
- `last_opened_at` (timestamp, nullable) - Last time software was opened (macOS only)

**Relationships:**
- Many-to-one with `hosts` via `host_id` (ON DELETE CASCADE)
- Many-to-one with `software` via `software_id`

**Indexes:**
- Composite primary key on `(host_id, software_id)`
- Index on `software_id` (`host_software_software_fk`) for reverse lookup
- Index on `software_id` (`idx_host_software_software_id`) for join/filter optimization

**Common Query Patterns:**
```sql
-- Get all software on a host
SELECT s.name, s.version, s.source, hs.last_opened_at
FROM host_software hs
INNER JOIN software s ON s.id = hs.software_id
WHERE hs.host_id = ?;

-- Find hosts with specific software
SELECT h.* FROM hosts h
INNER JOIN host_software hs ON hs.host_id = h.id
WHERE hs.software_id = ?;

-- Get recently used software (macOS)
SELECT s.name, hs.last_opened_at
FROM host_software hs
INNER JOIN software s ON s.id = hs.software_id
WHERE hs.host_id = ? AND hs.last_opened_at IS NOT NULL
ORDER BY hs.last_opened_at DESC;
```

**Usage Notes:**
- Populated by osquery inventory collection (apps, programs, packages, etc.)
- Used to show software on host details page
- Used for software analytics and reporting
- `last_opened_at` only available on macOS (from unified log)
- Powers vulnerability detection (joined with software_cve)
- Updated during osquery check-ins

**Platform Affinity:** Common (all platforms)

---

### software_cve

**Purpose:** Links software versions to CVE (Common Vulnerabilities and Exposures) identifiers. Powers vulnerability tracking.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT) - Primary key
- `cve` (varchar(255)) - CVE identifier (e.g., CVE-2024-1234)
- `software_id` (bigint unsigned, nullable) - Foreign key to software
- `source` (int, DEFAULT 0) - Source of CVE data (0 = NVD)
- `resolved_in_version` (varchar(255), nullable) - Version that resolves the CVE
- `created_at` (timestamp, DEFAULT CURRENT_TIMESTAMP)
- `updated_at` (timestamp, DEFAULT CURRENT_TIMESTAMP ON UPDATE)

**Relationships:**
- Many-to-one with `software` via `software_id`
- Many-to-one with `cve_meta` via `cve` (metadata about the CVE)

**Indexes:**
- Primary key on `id`
- Unique index on `(software_id, cve)` (`unq_software_id_cve`)
- Index on `cve` (`idx_software_cve_cve`) for CVE lookups

**Common Query Patterns:**
```sql
-- Get CVEs for software on a host
SELECT DISTINCT sc.cve, cm.cvss_score, cm.epss_probability
FROM host_software hs
INNER JOIN software_cve sc ON sc.software_id = hs.software_id
INNER JOIN cve_meta cm ON cm.cve = sc.cve
WHERE hs.host_id = ?;

-- Find hosts vulnerable to a specific CVE
SELECT DISTINCT h.* FROM hosts h
INNER JOIN host_software hs ON hs.host_id = h.id
INNER JOIN software_cve sc ON sc.software_id = hs.software_id
WHERE sc.cve = ?;

-- Get software with critical CVEs
SELECT s.*, cm.cvss_score
FROM software s
INNER JOIN software_cve sc ON sc.software_id = s.id
INNER JOIN cve_meta cm ON cm.cve = sc.cve
WHERE cm.cvss_score >= 9.0;
```

**Usage Notes:**
- Populated by Fleet's vulnerability scanning process
- Matches software CPEs to NVD database
- Critical for security posture visibility
- CVSS scores and EPSS probability in `cve_meta`
- Powers vulnerability dashboard and reporting

**Platform Affinity:** Common (primarily desktop: macOS, Windows, Linux)

---

### cve_meta

**Purpose:** Stores metadata about CVEs including severity scores, descriptions, and CISA KEV status.

**Key Fields:**
- `cve` (varchar(20), PK) - CVE identifier
- `cvss_score` (double, nullable) - CVSS base score (0.0-10.0)
- `epss_probability` (double, nullable) - EPSS probability score (0.0-1.0)
- `cisa_known_exploit` (tinyint(1), nullable) - Whether listed in CISA Known Exploited Vulnerabilities
- `published` (timestamp, nullable) - CVE publication date
- `description` (text, nullable) - CVE description from NVD

**Relationships:**
- One-to-many with `software_cve` via `cve`

**Indexes:**
- Primary key on `cve`

**Common Query Patterns:**
```sql
-- Get CVE details
SELECT * FROM cve_meta WHERE cve = ?;

-- Get critical CVEs with CISA KEV flag
SELECT * FROM cve_meta
WHERE cvss_score >= 9.0 AND cisa_known_exploit = 1;

-- Get recently published CVEs
SELECT * FROM cve_meta
WHERE published > DATE_SUB(NOW(), INTERVAL 30 DAY)
ORDER BY published DESC;
```

**Usage Notes:**
- Synced from National Vulnerability Database (NVD)
- CVSS scores indicate severity (7.0-8.9 = High, 9.0-10.0 = Critical)
- EPSS probability indicates likelihood of exploitation
- CISA KEV = Known Exploited Vulnerability (actively exploited in the wild)
- Used for prioritizing vulnerability remediation

**Platform Affinity:** Common

---

### software_title_display_names

**Purpose:** Stores custom display names for software titles per team, allowing teams to customize how software appears in the UI.

**Key Fields:**
- `id` (int, PK, AUTO_INCREMENT) - Primary key
- `team_id` (int unsigned) - Team this display name applies to
- `software_title_id` (int unsigned, FK) - Reference to software_titles
- `display_name` (varchar(255), DEFAULT '') - Custom display name
- `created_at` (timestamp(6))

**Relationships:**
- Many-to-one with `software_titles` via `software_title_id` (ON DELETE CASCADE)

**Indexes:**
- Primary key on `id`
- Unique index on `(team_id, software_title_id)` - one display name per title per team

**Common Query Patterns:**
```sql
-- Get custom display name for a software title in a team
SELECT display_name FROM software_title_display_names
WHERE team_id = ? AND software_title_id = ?;

-- Get all custom names for a team
SELECT st.name AS original_name, stdn.display_name
FROM software_title_display_names stdn
JOIN software_titles st ON st.id = stdn.software_title_id
WHERE stdn.team_id = ?;
```

**Usage Notes:**
- Allows teams to customize software names shown in Fleet UI
- If no custom name exists, the original `software_titles.name` is used
- Useful for branding or clarifying software names for end users

**Platform Affinity:** Common

---

### software_update_schedules

**Purpose:** Configures automatic software update windows for software titles. Enables scheduled software updates during maintenance windows.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT) - Primary key
- `team_id` (int unsigned) - Team this schedule applies to
- `title_id` (int unsigned, FK) - Reference to software_titles (ON DELETE CASCADE)
- `enabled` (boolean, DEFAULT FALSE) - Whether auto-updates are enabled
- `start_time` (char(5)) - Update window start time (HH:MM format)
- `end_time` (char(5)) - Update window end time (HH:MM format)

**Relationships:**
- Many-to-one with `software_titles` via `title_id` (ON DELETE CASCADE)

**Indexes:**
- Primary key on `id`
- Unique index on `(team_id, title_id)` - one schedule per title per team

**Common Query Patterns:**
```sql
-- Get enabled update schedules for a team
SELECT sus.*, st.name AS software_name
FROM software_update_schedules sus
JOIN software_titles st ON st.id = sus.title_id
WHERE sus.team_id = ? AND sus.enabled = TRUE;

-- Check if software can be updated now (using host timezone)
SELECT sus.* FROM software_update_schedules sus
WHERE sus.title_id = ?
  AND sus.enabled = TRUE
  AND CURTIME() BETWEEN sus.start_time AND sus.end_time;
```

**Usage Notes:**
- Times are in local timezone of each host (uses `hosts.timezone`)
- Requires `hosts.timezone` to be populated for accurate scheduling
- Update window defined by start and end time (24-hour format)
- Fleet-maintained apps can be auto-updated within these windows

**Platform Affinity:** Common

---

## MDM Tables

### mdm_apple_configuration_profiles

**Purpose:** Apple MDM configuration profiles (.mobileconfig files) for deployment to macOS, iOS, and iPadOS devices.

**Key Fields:**
- `profile_uuid` (varchar(37), PK) - UUID prefixed with 'a' (e.g., a1234567-...)
- `profile_id` (int unsigned, UNIQUE AUTO_INCREMENT) - Numeric ID (legacy, still indexed)
- `team_id` (int unsigned, DEFAULT 0) - Team ID (0 = global/no team)
- `identifier` (varchar(255)) - Profile identifier (unique per team)
- `name` (varchar(255)) - Profile display name (unique per team)
- `mobileconfig` (mediumblob) - Profile XML/plist content
- `checksum` (binary(16)) - MD5 checksum of mobileconfig
- `scope` (enum: 'System', 'User', DEFAULT 'System') - Installation scope
- `secrets_updated_at` (datetime(6), nullable) - Last time Fleet variables were updated
- `created_at` (timestamp(6), DEFAULT CURRENT_TIMESTAMP(6))
- `uploaded_at` (timestamp(6), nullable) - Upload timestamp

**Relationships:**
- One-to-many with `host_mdm_apple_profiles` via `profile_uuid`
- Many-to-many with `labels` via `mdm_configuration_profile_labels` for targeting

**Indexes:**
- Primary key on `profile_uuid`
- Unique index on `profile_id` (`idx_mdm_apple_config_prof_id`)
- Unique index on `(team_id, identifier)` (`idx_mdm_apple_config_prof_team_identifier`)
- Unique index on `(team_id, name)` (`idx_mdm_apple_config_prof_team_name`)

**Common Query Patterns:**
```sql
-- Get profiles for a team
SELECT * FROM mdm_apple_configuration_profiles
WHERE team_id = ?;

-- Check if profile identifier exists
SELECT * FROM mdm_apple_configuration_profiles
WHERE team_id = ? AND identifier = ?;

-- Get profile by UUID
SELECT * FROM mdm_apple_configuration_profiles
WHERE profile_uuid = ?;

-- Get profiles needing variable update
SELECT * FROM mdm_apple_configuration_profiles
WHERE secrets_updated_at IS NULL
   OR secrets_updated_at < (
       SELECT MAX(updated_at) FROM fleet_variables
   );
```

**Usage Notes:**
- Profile XML stored as binary blob (mobileconfig)
- Fleet variables like `$FLEET_VAR_HOST_UUID` are replaced before deployment
- Checksum used to detect changes and trigger re-deployment
- Scope determines whether installed system-wide or per-user
- `secrets_updated_at` tracks when variables were last substituted
- Same profile can target multiple platforms (determined by mobileconfig content)
- Fleet reserves certain profile identifiers (com.fleetdm.*)

**Platform Affinity:** Apple (macOS, iOS, iPadOS)

**Related Migrations:**
- MDM profile support added incrementally

---

### host_mdm_apple_profiles

**Purpose:** Tracks deployment status of Apple MDM profiles to individual hosts. One row per host-profile combination.

**Key Fields:**
- `host_uuid` (varchar(255), PK, composite) - Host UUID
- `profile_uuid` (varchar(37), PK, composite, DEFAULT '') - Foreign key to mdm_apple_configuration_profiles
- `profile_identifier` (varchar(255)) - Profile identifier (denormalized)
- `profile_name` (varchar(255), DEFAULT '') - Profile name (denormalized)
- `status` (varchar(20), nullable) - Status: pending, verifying, verified, failed
- `operation_type` (varchar(20), nullable) - Operation: install, remove
- `detail` (text, nullable) - Error details if status = failed
- `command_uuid` (varchar(127)) - MDM command UUID that deployed this
- `checksum` (binary(16)) - Profile checksum when deployed
- `retries` (tinyint unsigned, DEFAULT 0) - Number of delivery retries
- `ignore_error` (tinyint(1), DEFAULT 0) - Whether to ignore deployment errors
- `scope` (enum: 'System', 'User', DEFAULT 'System') - Installation scope
- `secrets_updated_at` (datetime(6), nullable) - Last Fleet variable substitution
- `variables_updated_at` (datetime(6), nullable) - Last variable update
- `created_at` (timestamp(6), DEFAULT CURRENT_TIMESTAMP(6))
- `updated_at` (timestamp(6), DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE)

**Relationships:**
- Many-to-one with `mdm_apple_configuration_profiles` via `profile_uuid`
- Indexed reference to `hosts` via `host_uuid` (not FK due to performance)
- Foreign key to `mdm_delivery_status` via `status` (ON UPDATE CASCADE)
- Foreign key to `mdm_operation_types` via `operation_type` (ON UPDATE CASCADE)

**Indexes:**
- Composite primary key on `(host_uuid, profile_uuid)`
- Index on `status` for filtering by deployment status
- Index on `operation_type` for filtering by operation

**Common Query Patterns:**
```sql
-- Get profile status for a host
SELECT * FROM host_mdm_apple_profiles
WHERE host_uuid = ?;

-- Get hosts where profile failed
SELECT host_uuid, detail
FROM host_mdm_apple_profiles
WHERE profile_uuid = ? AND status = 'failed';

-- Get hosts needing profile installation
SELECT DISTINCT hmap.host_uuid
FROM mdm_apple_configuration_profiles macp
CROSS JOIN hosts h
LEFT JOIN host_mdm_apple_profiles hmap
  ON hmap.host_uuid = h.uuid
  AND hmap.profile_uuid = macp.profile_uuid
WHERE macp.team_id = h.team_id
  AND hmap.id IS NULL; -- Profile not yet deployed

-- Get aggregate profile status
SELECT status, COUNT(*)
FROM host_mdm_apple_profiles
WHERE profile_uuid = ?
GROUP BY status;
```

**Usage Notes:**
- One row per host-profile combination
- Status lifecycle: pending → verifying → verified (or failed)
- Updated by MDM check-in process when device reports status
- `command_uuid` links to the MDM command that installed the profile
- Used to show profile deployment status in UI
- `detail` field contains error messages for troubleshooting
- Checksum comparison detects profile changes requiring redeployment

**Platform Affinity:** Apple (macOS, iOS, iPadOS)

---

### mdm_windows_configuration_profiles

**Purpose:** Windows MDM configuration profiles (SyncML/OMA-DM format) for deployment to Windows devices.

**Key Fields:**
- `profile_uuid` (varchar(37), PK) - UUID prefixed with 'w'
- `name` (varchar(255)) - Profile display name
- `team_id` (int unsigned, DEFAULT 0) - Team ID (0 = global)
- `syncml` (mediumblob) - SyncML XML content
- `checksum` (binary(16), GENERATED) - MD5 hash of syncml (stored, auto-computed)
- `secrets_updated_at` (datetime(6), nullable) - Last Fleet variable substitution
- `auto_increment` (bigint, AUTO_INCREMENT, UNIQUE) - Numeric identifier for ordering
- `created_at` (timestamp(6), DEFAULT CURRENT_TIMESTAMP(6))
- `uploaded_at` (timestamp(6), nullable)

**Relationships:**
- One-to-many with `host_mdm_windows_profiles` via `profile_uuid`
- Many-to-many with `labels` via `mdm_configuration_profile_labels`

**Indexes:**
- Primary key on `profile_uuid`
- Unique index on `(team_id, name)` (`idx_mdm_windows_configuration_profiles_team_id_name`)
- Unique index on `auto_increment`

**Usage Notes:**
- SyncML format for Windows MDM operations
- Similar to Apple profiles but Windows-specific protocol
- Checksum auto-generated from syncml content for change detection
- Deployed via Windows MDM command infrastructure

**Platform Affinity:** Windows

---

### host_mdm_windows_profiles

**Purpose:** Tracks deployment status of Windows MDM profiles to hosts. Similar to Apple equivalent.

**Key Fields:**
- `host_uuid` (varchar(255), PK, composite) - Host UUID
- `profile_uuid` (varchar(37), PK, composite, DEFAULT '') - Foreign key to mdm_windows_configuration_profiles
- `profile_name` (varchar(255), DEFAULT '') - Profile name (denormalized)
- `status` (varchar(20), nullable) - Status: pending, verifying, verified, failed
- `operation_type` (varchar(20), nullable) - Operation: install, remove
- `detail` (text, nullable) - Error details if status = failed
- `command_uuid` (varchar(127)) - MDM command UUID
- `retries` (tinyint unsigned, DEFAULT 0) - Number of delivery retries
- `checksum` (binary(16), DEFAULT 0) - Profile checksum when deployed
- `secrets_updated_at` (datetime(6), nullable) - Last Fleet variable substitution
- `created_at` (datetime(6), DEFAULT CURRENT_TIMESTAMP(6))
- `updated_at` (datetime(6), DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE)

**Relationships:**
- Many-to-one with `mdm_windows_configuration_profiles` via `profile_uuid`
- Indexed reference to hosts via `host_uuid`
- Foreign key to `mdm_delivery_status` via `status` (ON UPDATE CASCADE)
- Foreign key to `mdm_operation_types` via `operation_type` (ON UPDATE CASCADE)

**Indexes:**
- Composite primary key on `(host_uuid, profile_uuid)`
- Index on `status` for filtering by deployment status
- Index on `operation_type` for filtering by operation

**Platform Affinity:** Windows

---

### android_devices

**Purpose:** Android devices enrolled in Fleet's Android MDM. Links to hosts table and tracks Android-specific enrollment details.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT)
- `host_id` (int unsigned, FK, UNIQUE) - One-to-one with hosts
- `device_id` (varchar(32), UNIQUE) - Android device ID from Google
- `enterprise_specific_id` (varchar(64), UNIQUE, nullable) - Enterprise identifier
- `applied_policy_id` (varchar(100), nullable) - Currently applied policy ID
- `applied_policy_version` (int, nullable) - Policy version number
- `last_policy_sync_time` (datetime(3), nullable) - Last policy sync
- `created_at`, `updated_at` (datetime(6))

**Relationships:**
- One-to-one with `hosts` via `host_id` (unique)
- Implicitly linked to `android_enterprises` (one enterprise per Fleet instance)

**Indexes:**
- Primary key on `id`
- Unique index on `host_id`
- Unique index on `device_id`
- Unique index on `enterprise_specific_id`

**Common Query Patterns:**
```sql
-- Get Android device by host
SELECT * FROM android_devices WHERE host_id = ?;

-- Get devices with policy applied
SELECT ad.*, h.hostname
FROM android_devices ad
INNER JOIN hosts h ON h.id = ad.host_id
WHERE ad.applied_policy_id IS NOT NULL;

-- Get devices needing policy sync
SELECT * FROM android_devices
WHERE last_policy_sync_time < DATE_SUB(NOW(), INTERVAL 1 HOUR);
```

**Usage Notes:**
- One row per Android device enrolled in Fleet MDM
- `device_id` from Google Android Management API
- Policy application tracked for compliance
- Last sync time helps identify stale devices

**Platform Affinity:** Android

---

### mdm_android_configuration_profiles

**Purpose:** Android configuration profiles (JSON policies) for deployment. Merged together to form complete device policy.

**Key Fields:**
- `profile_uuid` (varchar(37), PK, DEFAULT '') - UUID prefixed with 'g'
- `team_id` (int unsigned, DEFAULT 0)
- `name` (varchar(255))
- `raw_json` (json) - JSON policy configuration
- `auto_increment` (bigint, AUTO_INCREMENT, UNIQUE) - Numeric identifier for ordering
- `created_at` (timestamp(6), DEFAULT CURRENT_TIMESTAMP(6))
- `uploaded_at` (timestamp(6), nullable, DEFAULT CURRENT_TIMESTAMP(6))

**Relationships:**
- One-to-many with `host_mdm_android_profiles` via `profile_uuid`
- Many-to-many with `labels` via `mdm_configuration_profile_labels`

**Indexes:**
- Primary key on `profile_uuid`
- Unique index on `auto_increment`
- Unique index on `(team_id, name)` (`idx_mdm_android_configuration_profiles_team_id_name`)

**Usage Notes:**
- JSON format (Android Management API policy structure)
- Multiple profiles merged into single policy applied to device
- Fleet manages profile lifecycle and merging logic

**Platform Affinity:** Android

---

### host_mdm_android_profiles

**Purpose:** Tracks deployment status of Android MDM profiles to individual hosts. One row per host-profile combination. Similar to Apple and Windows equivalents.

**Key Fields:**
- `host_uuid` (varchar(255), PK, composite) - Host UUID
- `profile_uuid` (varchar(37), PK, composite, DEFAULT '') - Foreign key to mdm_android_configuration_profiles
- `profile_name` (varchar(255), DEFAULT '') - Profile name (denormalized)
- `status` (varchar(20), nullable) - Status: pending, verifying, verified, failed
- `operation_type` (varchar(20), nullable) - Operation: install, remove
- `detail` (text, nullable) - Error details if status = failed
- `policy_request_uuid` (varchar(36), nullable) - Android Management API policy request UUID
- `device_request_uuid` (varchar(36), nullable) - Device-specific request UUID
- `request_fail_count` (tinyint unsigned, DEFAULT 0) - Number of failed API requests
- `included_in_policy_version` (int, nullable) - Policy version when profile was included
- `can_reverify` (tinyint(1), DEFAULT 0) - Whether profile can be re-verified (NEW)
- `created_at`, `updated_at` (timestamp(6))

**Relationships:**
- Many-to-one with `mdm_android_configuration_profiles` via `profile_uuid`
- Indexed reference to `hosts` via `host_uuid` (not FK due to performance)
- Foreign key to `mdm_delivery_status` via `status` (ON UPDATE CASCADE)
- Foreign key to `mdm_operation_types` via `operation_type` (ON UPDATE CASCADE)

**Indexes:**
- Primary key on `(host_uuid, profile_uuid)` - composite key
- Index on `status` for filtering by deployment status
- Index on `operation_type` for filtering by operation
- Index on `policy_request_uuid` for API request tracking
- Index on `device_request_uuid` for device request tracking

**Common Query Patterns:**
```sql
-- Get profile status for a host
SELECT * FROM host_mdm_android_profiles
WHERE host_uuid = ?;

-- Get hosts where profile failed
SELECT host_uuid, detail
FROM host_mdm_android_profiles
WHERE profile_uuid = ? AND status = 'failed';

-- Get profiles needing re-verification
SELECT * FROM host_mdm_android_profiles
WHERE can_reverify = 1;

-- Get aggregate profile status
SELECT status, COUNT(*)
FROM host_mdm_android_profiles
WHERE profile_uuid = ?
GROUP BY status;
```

**Usage Notes:**
- One row per host-profile combination (composite primary key)
- Status lifecycle: pending → verifying → verified (or failed)
- `policy_request_uuid` tracks the Android Management API PATCH request
- `request_fail_count` incremented on API failures for retry logic
- `included_in_policy_version` tracks which policy version included this profile
- `can_reverify` flag enables re-verification after policy changes
- Similar pattern to `host_mdm_apple_profiles` and `host_mdm_windows_profiles`

**Platform Affinity:** Android

---

### android_app_configurations

**Purpose:** Managed app configurations for Android applications. Allows configuring app settings via MDM.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT) - Primary key
- `application_id` (varchar(255)) - Android app package name (e.g., com.example.app)
- `team_id` (int unsigned, FK, nullable) - Team scope (NULL = global)
- `global_or_team_id` (int, DEFAULT 0) - Compound identifier (0 for global, team_id for teams)
- `configuration` (json) - App configuration in JSON format
- `created_at`, `updated_at` (timestamp(6))

**Relationships:**
- Many-to-one with `teams` via `team_id` (ON DELETE CASCADE)

**Indexes:**
- Primary key on `id`
- Unique index on `(global_or_team_id, application_id)` - one config per app per team

**Common Query Patterns:**
```sql
-- Get app configuration for a team
SELECT * FROM android_app_configurations
WHERE (team_id = ? OR team_id IS NULL)
  AND application_id = ?;

-- Get all app configurations for a team
SELECT * FROM android_app_configurations
WHERE team_id = ?;
```

**Usage Notes:**
- Managed configurations are pushed to Android devices via Android Management API
- Configuration JSON follows Android's managed configuration schema
- Used for pre-configuring enterprise apps (VPN, email, etc.)
- Similar to Apple's managed app configuration

**Platform Affinity:** Android

---

## Policy Tables

### policies

**Purpose:** Security and compliance policy definitions. Policies are osquery queries that check for compliance. Empty result = failure, any rows = pass.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT)
- `team_id` (int unsigned, FK, nullable) - NULL for global policies
- `name` (varchar(255)) - Policy name
- `query` (mediumtext) - osquery SQL query to evaluate policy
- `description` (mediumtext) - Policy description
- `resolution` (text, nullable) - Steps to remediate policy failures
- `author_id` (int unsigned, FK, nullable) - Foreign key to users (ON DELETE SET NULL)
- `platforms` (varchar(255), DEFAULT '') - Target platforms (comma-separated): darwin, windows, linux, chrome, ios, ipados
- `critical` (tinyint(1), DEFAULT 0) - Critical policy flag
- `checksum` (binary(16), UNIQUE) - MD5 of query+name+description for deduplication
- `calendar_events_enabled` (tinyint unsigned, DEFAULT 0) - Create calendar events for failures
- `software_installer_id` (int unsigned, FK, nullable) - Remediation software installer
- `script_id` (int unsigned, FK, nullable) - Remediation script
- `vpp_apps_teams_id` (int unsigned, FK, nullable) - Remediation VPP app
- `conditional_access_enabled` (tinyint unsigned, DEFAULT 0) - For Microsoft Conditional Access
- `type` (enum('dynamic','patch'), NOT NULL, DEFAULT 'dynamic') - Policy type: standard dynamic policy or patch policy
- `patch_software_title_id` (int unsigned, FK, nullable) - Foreign key to software_titles for patch policies (ON DELETE CASCADE)
- `needs_full_membership_cleanup` (tinyint(1), NOT NULL, DEFAULT 0) - Flag for pending membership cleanup
- `created_at`, `updated_at` (timestamp)

**Relationships:**
- Many-to-one with `teams` via `team_id`
- Many-to-one with `users` via `author_id` (ON DELETE SET NULL)
- One-to-many with `policy_membership` via `policy_id` (ON DELETE CASCADE)
- Many-to-many with `labels` via `policy_labels`
- Many-to-one with `software_installers` via `software_installer_id` (remediation)
- Many-to-one with `scripts` via `script_id` (remediation)
- Many-to-one with `vpp_apps_teams` via `vpp_apps_teams_id` (remediation)
- Many-to-one with `software_titles` via `patch_software_title_id` (patch policies, ON DELETE CASCADE)

**Indexes:**
- Primary key on `id`
- Unique index on `checksum` (`idx_policies_checksum`)
- Unique index on `(team_id, patch_software_title_id)` (`idx_team_id_patch_software_title_id`) - one patch policy per software title per team
- Index on `author_id` (`idx_policies_author_id`)
- Index on `team_id` (`idx_policies_team_id`)
- Index on `software_installer_id`, `script_id`, `vpp_apps_teams_id` for remediation lookups

**Common Query Patterns:**
```sql
-- Get policies for a team
SELECT * FROM policies WHERE team_id = ?;

-- Get global policies
SELECT * FROM policies WHERE team_id IS NULL;

-- Get critical policies
SELECT * FROM policies WHERE critical = 1;

-- Get policies for a platform
SELECT * FROM policies
WHERE FIND_IN_SET(?, platforms) > 0 OR platforms = '';

-- Get policy with pass/fail counts
SELECT p.*,
  SUM(CASE WHEN pm.passes = 1 THEN 1 ELSE 0 END) AS passing_count,
  SUM(CASE WHEN pm.passes = 0 THEN 1 ELSE 0 END) AS failing_count
FROM policies p
LEFT JOIN policy_membership pm ON pm.policy_id = p.id
WHERE p.id = ?
GROUP BY p.id;
```

**Usage Notes:**
- Query should return rows only for compliant hosts
- Empty result = policy failure for that host
- Critical policies show special UI indicators (red badge)
- Platform targeting: empty platforms string = all platforms
- Remediation options: software installer, script, or VPP app
- Calendar events integration for failure notifications
- Conditional Access integration for Microsoft Entra ID
- Checksum prevents duplicate policies

**Platform Affinity:** Common (queryable platforms: darwin, windows, linux, chrome)

---

### policy_membership

**Purpose:** Tracks policy pass/fail status for each host. Junction table between policies and hosts with pass/fail state.

**Key Fields:**
- `policy_id` (int unsigned, PK, composite, FK) - Foreign key to policies
- `host_id` (int unsigned, PK, composite, FK) - Foreign key to hosts
- `passes` (tinyint(1), nullable) - NULL = not yet evaluated, 1 = pass, 0 = fail
- `automation_iteration` (int, nullable) - Tracks which automation iteration processed this result
- `created_at` (timestamp, DEFAULT CURRENT_TIMESTAMP)
- `updated_at` (timestamp, DEFAULT CURRENT_TIMESTAMP ON UPDATE)

**Relationships:**
- Many-to-one with `policies` via `policy_id` (ON DELETE CASCADE)
- Many-to-one with `hosts` via `host_id`

**Indexes:**
- Composite primary key on `(policy_id, host_id)`
- Index on `passes` (`idx_policy_membership_passes`)
- Composite index on `(host_id, passes)` (`idx_policy_membership_host_id_passes`)

**Common Query Patterns:**
```sql
-- Get policy status for a host
SELECT p.name, p.description, pm.passes, pm.updated_at
FROM policy_membership pm
INNER JOIN policies p ON p.id = pm.policy_id
WHERE pm.host_id = ?;

-- Get hosts failing a policy
SELECT h.* FROM hosts h
INNER JOIN policy_membership pm ON pm.host_id = h.id
WHERE pm.policy_id = ? AND pm.passes = 0;

-- Get host policy compliance summary
SELECT
  SUM(CASE WHEN pm.passes = 1 THEN 1 ELSE 0 END) AS passing,
  SUM(CASE WHEN pm.passes = 0 THEN 1 ELSE 0 END) AS failing,
  SUM(CASE WHEN pm.passes IS NULL THEN 1 ELSE 0 END) AS pending
FROM policy_membership pm
WHERE pm.host_id = ?;

-- Get failing hosts for webhook
SELECT DISTINCT pm.host_id
FROM policy_membership pm
WHERE pm.policy_id IN (?)
  AND pm.passes = 0
  AND pm.updated_at > ?;
```

**Usage Notes:**
- One row per host-policy combination
- `passes` NULL means policy hasn't been evaluated yet on this host
- Updated during scheduled policy checks (typically every hour)
- Used to trigger webhooks for policy failures
- Drives compliance dashboards and reporting
- `updated_at` tracks last evaluation time

**Platform Affinity:** Common

---

## Activity Tables

### activity_past

> **Note:** This table was renamed from `activities` to `activity_past` in migration 20260316120008. The related `host_activities` table was renamed to `activity_host_past`.

**Purpose:** Comprehensive audit log of all user and system actions in Fleet. Immutable record for compliance and troubleshooting.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT)
- `created_at` (timestamp(6), INDEXED, DEFAULT CURRENT_TIMESTAMP(6)) - Activity timestamp
- `user_id` (int unsigned, FK, nullable) - User who performed action (NULL for system/automated)
- `user_name` (varchar(255), nullable) - User name at time of activity (denormalized)
- `user_email` (varchar(255)) - User email (denormalized)
- `activity_type` (varchar(255), INDEXED) - Type of activity (e.g., 'installed_app_store_app')
- `details` (json, nullable) - Activity-specific details
- `streamed` (tinyint(1), INDEXED, DEFAULT 0) - Whether streamed to log destination
- `fleet_initiated` (tinyint(1), DEFAULT 0) - Whether initiated by Fleet system vs user
- `host_only` (tinyint(1), DEFAULT 0) - Whether activity is host-specific (not shown in global feed)

**Relationships:**
- Many-to-one with `users` via `user_id` (ON DELETE SET NULL)
- Implicitly related to hosts via `details` JSON field (host_id, host_display_name)

**Indexes:**
- Primary key on `id`
- Index on `user_id` (`fk_activities_user_id`)
- Index on `streamed` (`activities_streamed_idx`) - for log streaming
- Index on `created_at` (`activities_created_at_idx`) - for time-based queries
- Index on `user_name` (`idx_activities_user_name`) - NEW: for user search
- Index on `user_email` (`idx_activities_user_email`) - NEW: for user search
- Index on `activity_type` (`idx_activities_activity_type`) - NEW: for filtering by type
- Composite index on `(activity_type, created_at)` (`idx_activities_type_created`) - NEW: for efficient filtering with ordering

**Common Query Patterns:**
```sql
-- Get global activity feed
SELECT * FROM activities
WHERE host_only = 0
ORDER BY created_at DESC
LIMIT 50;

-- Get host-specific activities
SELECT * FROM activities
WHERE JSON_EXTRACT(details, '$.host_id') = ?
ORDER BY created_at DESC;

-- Get activities by type
SELECT * FROM activities
WHERE activity_type = 'installed_app_store_app'
ORDER BY created_at DESC;

-- Get user's activities
SELECT * FROM activities
WHERE user_id = ?
ORDER BY created_at DESC;

-- Get activities in time range
SELECT * FROM activities
WHERE created_at BETWEEN ? AND ?
ORDER BY created_at DESC;

-- Get unstreamed activities for log destination
SELECT * FROM activities
WHERE streamed = 0
ORDER BY created_at ASC
LIMIT 1000;
```

**Usage Notes:**
- Activity types documented in `/docs/Contributing/reference/audit-logs.md`
- `details` JSON structure varies by `activity_type`
- Used for compliance auditing and troubleshooting
- User name/email denormalized to preserve even if user deleted
- `host_only` flag prevents cluttering global feed with host-specific events
- `fleet_initiated` distinguishes automated actions from user actions
- `streamed` flag used for log destination integration (Splunk, etc.)
- Immutable - activities are never updated or deleted (except via retention policy)

**Example Activity Types:**
- `installed_app_store_app` - End user installed self-service software
- `added_policy` - Admin created a policy
- `edited_agent_options` - Admin modified agent settings
- `ran_live_query` - Admin ran a live query
- `edited_mdm_configuration_profile` - Admin modified MDM profile
- `locked_host` - Admin locked a host via MDM

**Platform Affinity:** Common (all platforms)

**Related Migrations:**
- Initial: `20210709124443_CreateActivitiesTable.go`
- Renamed `activities` → `activity_past`: `20260316120008_RenameActivitiesToActivityPast.go`

---

## Configuration Tables

### app_config_json

**Purpose:** Fleet server global configuration and settings. Single-row table (id always = 1) storing JSON configuration.

**Key Fields:**
- `id` (int unsigned, PK, DEFAULT 1) - Always 1 (single row)
- `json_value` (json) - Complete configuration as JSON object
- `created_at` (timestamp, DEFAULT CURRENT_TIMESTAMP)
- `updated_at` (timestamp, DEFAULT CURRENT_TIMESTAMP ON UPDATE)

**Relationships:**
- None (standalone configuration table)

**Indexes:**
- Primary key on `id`
- Unique constraint on `id` ensures single row

**Common Query Patterns:**
```sql
-- Get app config
SELECT json_value FROM app_config_json WHERE id = 1;

-- Update app config
UPDATE app_config_json
SET json_value = ?
WHERE id = 1;
```

**Usage Notes:**
- Only one row exists (id = 1)
- Modified via `/api/v1/fleet/config` endpoint
- Contains global Fleet settings (org name, server URL, MDM settings, integrations, etc.)
- JSON structure mirrors API response format
- Changes logged to activities table
- Team-specific configs stored in `teams.config` field

**Platform Affinity:** Common

---

### teams

_(Already documented in Core Tables section)_

---

## Query & Pack Tables

### queries

**Purpose:** Saved osquery queries that can be run on hosts (live queries) or scheduled via packs.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT)
- `name` (varchar(255), INDEXED) - Query name
- `description` (text) - Query description
- `query` (text) - osquery SQL query
- `author_id` (int unsigned, FK, nullable) - User who created (ON DELETE SET NULL)
- `saved` (tinyint(1)) - Whether query is saved to library
- `observer_can_run` (tinyint(1)) - Whether observer role can run
- `team_id` (int unsigned, FK, nullable) - Team scope (NULL = global)
- `platform` (varchar(255), DEFAULT '') - Target platforms (comma-separated)
- `min_osquery_version` (varchar(255), DEFAULT '') - Minimum osquery version required
- `logging_type` (varchar(255), DEFAULT 'snapshot') - Logging type: snapshot or differential
- `schedule_interval` (int unsigned, DEFAULT 0) - Run interval in seconds (0 = not scheduled)
- `automations_enabled` (tinyint unsigned, DEFAULT 0) - Whether automations are enabled
- `discard_data` (tinyint(1), DEFAULT 1) - Don't store query results
- `is_scheduled` (tinyint(1), GENERATED) - Computed: schedule_interval > 0
- `team_id_char` (char(10), DEFAULT '') - Team ID as string for composite unique keys
- `created_at`, `updated_at` (timestamp)

**Relationships:**
- Many-to-one with `users` via `author_id` (ON DELETE SET NULL)
- Many-to-one with `teams` via `team_id` (ON DELETE CASCADE)
- One-to-many with `scheduled_queries` via `(team_id_char, name)` (ON DELETE CASCADE ON UPDATE CASCADE)
- Many-to-many with `labels` via `query_labels`
- One-to-many with `distributed_query_campaigns` for live queries

**Indexes:**
- Primary key on `id`
- Unique index on `(team_id_char, name)` (`idx_team_id_name_unq`)
- Unique index on `(name, team_id_char)` (`idx_name_team_id_unq`)
- Index on `author_id`
- Composite index on `(team_id, saved, automations_enabled, schedule_interval)` for filtered lookups
- Composite index on `(is_scheduled, automations_enabled)` for scheduled query lookups

**Common Query Patterns:**
```sql
-- Get saved queries
SELECT * FROM queries WHERE saved = 1;

-- Get queries for a team
SELECT * FROM queries WHERE team_id = ? OR team_id IS NULL;

-- Search queries by name
SELECT * FROM queries
WHERE name LIKE CONCAT('%', ?, '%')
  AND saved = 1;
```

**Usage Notes:**
- Live queries run immediately on selected hosts
- Scheduled queries run periodically via packs
- Observer role restriction via `observer_can_run` flag
- Platform targeting for query compatibility
- `discard_data` for privacy-sensitive queries

**Platform Affinity:** Common (queryable platforms: darwin, windows, linux, chrome)

---

### scheduled_queries

**Purpose:** Queries scheduled to run periodically on hosts via packs.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT)
- `pack_id` (int unsigned, FK, nullable) - Foreign key to packs (ON DELETE CASCADE)
- `query_id` (int unsigned, FK, nullable) - Foreign key to queries (legacy, nullable)
- `query_name` (varchar(255)) - Query name (FK reference to queries via composite key)
- `name` (varchar(255)) - Scheduled query name
- `description` (varchar(1023), DEFAULT '') - Description
- `interval` (int unsigned, nullable) - Run interval in seconds
- `platform` (varchar(255), DEFAULT '') - Target platforms
- `version` (varchar(255), DEFAULT '') - Minimum osquery version
- `snapshot` (tinyint(1), nullable) - Snapshot mode vs differential
- `removed` (tinyint(1), nullable) - Whether query has been removed from schedule
- `shard` (int unsigned, nullable) - Shard percentage (0-100) for gradual rollout
- `denylist` (tinyint(1), nullable) - Whether query is denylisted
- `team_id_char` (char(10), DEFAULT '') - Team ID as string for composite FK
- `created_at`, `updated_at` (timestamp)

**Relationships:**
- Many-to-one with `packs` via `pack_id` (ON DELETE CASCADE)
- Many-to-one with `queries` via `(team_id_char, query_name)` (ON DELETE CASCADE ON UPDATE CASCADE)

**Indexes:**
- Primary key on `id`
- Unique index on `(name, pack_id)` (`unique_names_in_packs`)
- Index on `pack_id` (`scheduled_queries_pack_id`)
- Index on `query_name` (`scheduled_queries_query_name`)
- Composite FK index on `(team_id_char, query_name)`

**Usage Notes:**
- Can reference saved query or contain inline SQL
- Interval determines execution frequency
- Snapshot mode returns all results each time
- Differential mode returns only changes
- Shard enables gradual rollout (% of hosts)
- Denylist flag disables query without deletion

**Platform Affinity:** Common (queryable platforms)

---

### packs

**Purpose:** Collections of scheduled queries deployed together to hosts.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT)
- `name` (varchar(255), UNIQUE) - Pack name
- `description` (varchar(255), nullable) - Pack description
- `platform` (varchar(255), nullable) - Target platforms
- `disabled` (tinyint(1), DEFAULT 0) - Whether pack is disabled
- `pack_type` (varchar(255), nullable) - Pack type: NULL or 'global'
- `created_at`, `updated_at` (timestamp)

**Relationships:**
- One-to-many with `scheduled_queries` via `pack_id` (ON DELETE CASCADE)
- Many-to-many with hosts/labels/teams via `pack_targets`

**Indexes:**
- Primary key on `id`
- Unique index on `name`

**Usage Notes:**
- Can be targeted to specific hosts, labels, or teams
- Disabled packs don't schedule queries
- Global type applies to all hosts

**Platform Affinity:** Common

---

### labels

**Purpose:** Dynamic and manual groupings of hosts for targeting queries, policies, software, and profiles.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT) - Primary key
- `name` (varchar(255), UNIQUE) - Label name
- `description` (varchar(255), nullable) - Label description
- `query` (mediumtext) - SQL query for dynamic labels
- `platform` (varchar(255), nullable) - Target platforms
- `label_type` (int unsigned, DEFAULT 1) - Label type (1 = regular, 2 = builtin)
- `label_membership_type` (int unsigned, DEFAULT 0) - Membership type (0 = dynamic, 1 = manual)
- `criteria` (json, nullable) - Label criteria for dynamic evaluation
- `team_id` (int unsigned, FK, nullable) - Team scope (NULL = global)
- `author_id` (int unsigned, FK, nullable) - User who created the label (ON DELETE SET NULL)
- `created_at`, `updated_at` (timestamp)

**Relationships:**
- Many-to-one with `teams` via `team_id` (NEW - team-scoped labels)
- Many-to-one with `users` via `author_id`
- One-to-many with `label_membership` via `label_id` (ON DELETE CASCADE)
- Many-to-many with `mdm_configuration_profile_labels` for profile targeting
- Many-to-many with `software_installer_labels` for software targeting
- Many-to-many with `policy_labels` for policy targeting

**Indexes:**
- Primary key on `id`
- Unique index on `name` (`idx_label_unique_name`)
- Index on `author_id`
- Index on `team_id`
- Fulltext index on `name` (`labels_search`) for name searching

**Common Query Patterns:**
```sql
-- Get labels for a team
SELECT * FROM labels WHERE team_id = ? OR team_id IS NULL;

-- Get dynamic labels
SELECT * FROM labels WHERE label_membership_type = 0;

-- Get builtin labels
SELECT * FROM labels WHERE label_type = 2;

-- Search labels by name
SELECT * FROM labels WHERE MATCH(name) AGAINST(? IN BOOLEAN MODE);
```

**Usage Notes:**
- Dynamic labels re-evaluate membership based on query results
- Manual labels have static membership assigned by admins
- Builtin labels are system-managed (e.g., platform-based labels)
- **NEW:** Team-scoped labels (`team_id` NOT NULL) are only visible within that team
- Labels can target profiles, software installers, policies, and VPP apps

**Platform Affinity:** Common

---

### label_membership

**Purpose:** Junction table tracking which hosts belong to which labels.

**Key Fields:**
- `host_id` (int unsigned, PK, composite) - Foreign key to hosts
- `label_id` (int unsigned, PK, composite) - Foreign key to labels
- `created_at` (timestamp, DEFAULT CURRENT_TIMESTAMP)
- `updated_at` (timestamp, DEFAULT CURRENT_TIMESTAMP ON UPDATE)

**Relationships:**
- Many-to-one with `labels` via `label_id`
- Many-to-one with `hosts` via `host_id`

**Indexes:**
- Composite primary key on `(host_id, label_id)`
- Index on `label_id` (`idx_lm_label_id`) for label-based lookups

**Platform Affinity:** Common

---

### query_results

**Purpose:** Stores results from scheduled queries (when `discard_data = 0`). Each row represents a single result row from a query execution on a host.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT)
- `query_id` (int unsigned, FK) - Foreign key to queries
- `host_id` (int unsigned, FK) - Foreign key to hosts
- `data` (json, nullable) - Query result data as JSON
- `has_data` (tinyint(1), VIRTUAL, GENERATED ALWAYS AS (data IS NOT NULL)) - Virtual column indicating non-null data
- `last_fetched` (timestamp(6)) - When this result was last fetched from the host

**Relationships:**
- Many-to-one with `queries` via `query_id`
- Many-to-one with `hosts` via `host_id`

**Indexes:**
- Primary key on `id`
- Composite index on `(query_id, has_data, host_id, last_fetched)` (`idx_query_id_has_data_host_id_last_fetched`)

**Usage Notes:**
- Only populated for queries where `discard_data = 0`
- The `has_data` virtual column enables efficient filtering of rows with actual data
- Results accumulate over time; retention managed by Fleet server

**Platform Affinity:** Common

---

### operating_systems

**Purpose:** Stores distinct operating system versions observed across enrolled hosts. Used for vulnerability matching and OS version reporting.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT)
- `name` (varchar(255)) - OS name (e.g., "macOS", "Windows 11 Enterprise")
- `version` (varchar(150)) - OS version string
- `arch` (varchar(100)) - CPU architecture (e.g., "x86_64", "ARM 64-bit Processor")
- `kernel_version` (varchar(150), DEFAULT '') - Kernel version
- `platform` (varchar(50)) - Platform identifier (darwin, windows, ubuntu, etc.)
- `display_version` (varchar(150), DEFAULT '') - User-facing version display string
- `installation_type` (varchar(20), NOT NULL, DEFAULT '') - Installation variant: "", "Client", "Server", "Server Core"

**Indexes:**
- Primary key on `id`
- Unique index on `(name, version, arch, kernel_version, platform, display_version, installation_type)` (`idx_unique_os`)

**Usage Notes:**
- `installation_type` differentiates Windows Server Core from full desktop installations for MSRC vulnerability matching
- Referenced by `host_operating_system` junction table to track which hosts run which OS
- The unique index ensures deduplication across all identifying fields

**Platform Affinity:** Common (all platforms)

**Related Migrations:**
- Added `installation_type`: `20260319120000_AddInstallationTypeToOperatingSystems.go`

---

## Certificate Tables

### certificate_authorities

**Purpose:** Certificate authorities configured in Fleet for issuing client certificates (SCEP, NDES, DigiCert, Smallstep, custom).

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT)
- `name` (varchar(255)) - CA name
- `cert_pem` (text) - CA certificate in PEM format
- `type` (enum: 'digicert', 'ndes_scep_proxy', 'custom_scep_proxy', 'hydrant', 'smallstep', 'custom_est_proxy') - CA type
- `url` (text) - CA endpoint URL
- `api_token_encrypted` (blob, nullable) - Encrypted API token (DigiCert)
- `profile_id` (varchar(255), nullable) - Certificate profile ID (DigiCert)
- `certificate_common_name` (varchar(255), nullable) - Certificate CN template (DigiCert)
- `certificate_user_principal_names` (json, nullable) - UPN template (DigiCert)
- `certificate_seat_id` (varchar(255), nullable) - Seat ID (DigiCert)
- `admin_url` (text, nullable) - Admin URL (NDES)
- `username` (varchar(255), nullable) - Username (NDES)
- `password_encrypted` (blob, nullable) - Encrypted password (NDES)
- `challenge_url` (text, nullable) - Challenge URL (Custom SCEP)
- `challenge_encrypted` (blob, nullable) - Encrypted challenge (Custom SCEP)
- `client_id` (varchar(255), nullable) - Client ID (Hydrant/Smallstep)
- `client_secret_encrypted` (blob, nullable) - Encrypted client secret (Hydrant/Smallstep)
- `created_at`, `updated_at` (timestamp)

**Relationships:**
- One-to-many with `certificate_templates` via `certificate_authority_id`
- Referenced by various certificate tables

**Indexes:**
- Primary key on `id`
- Unique index on `(type, name)` (`idx_ca_type_name`)

**Usage Notes:**
- Different columns used depending on `type`:
  - **digicert**: Uses `api_token_encrypted`, `profile_id`, `certificate_common_name`, `certificate_user_principal_names`, `certificate_seat_id`
  - **ndes_scep_proxy**: Uses `admin_url`, `username`, `password_encrypted`
  - **custom_scep_proxy**: Uses `challenge_url`, `challenge_encrypted`
  - **hydrant / smallstep**: Uses `client_id`, `client_secret_encrypted`
  - **custom_est_proxy**: Uses URL-based enrollment
- Encrypted fields use Fleet server's encryption key
- Supports Windows NDES for certificate enrollment
- DigiCert integration for enterprise PKI
- Custom SCEP/EST servers
- Smallstep CA integration
- Hydrant (step-ca) integration

**Platform Affinity:** Common (primarily Windows for NDES, Apple for SCEP)

---

### host_disk_encryption_keys

**Purpose:** Stores FileVault (macOS) and BitLocker (Windows) recovery keys for disk encryption.

**Key Fields:**
- `host_id` (int unsigned, PK, FK) - One-to-one with hosts
- `base64_encrypted` (text) - Encrypted recovery key (base64)
- `base64_encrypted_salt` (varchar(255)) - Encryption salt
- `key_slot` (tinyint unsigned, nullable) - Key slot (Linux LUKS)
- `decryptable` (tinyint(1), nullable, INDEXED) - Whether key can be decrypted
- `reset_requested` (tinyint(1), DEFAULT 0) - Whether key reset was requested
- `client_error` (varchar(255)) - Client-side error if escrowing failed
- `created_at` (timestamp(6))
- `updated_at` (timestamp(6))

**Relationships:**
- One-to-one with `hosts` via `host_id` (primary key)

**Indexes:**
- Primary key on `host_id`
- Index on `decryptable` for filtering

**Usage Notes:**
- Recovery keys encrypted with Fleet's encryption key
- FileVault keys for macOS
- BitLocker keys for Windows
- LUKS keys for Linux (with key slot)
- `decryptable` flag indicates if Fleet can decrypt the key
- Keys archived to `host_disk_encryption_keys_archive` on update

**Platform Affinity:** Common (macOS FileVault, Windows BitLocker, Linux LUKS)

---

### host_recovery_key_passwords

**Purpose:** Stores encrypted recovery lock passwords for macOS hosts with auto-rotation support. Tracks MDM delivery status for password operations.

**Key Fields:**
- `host_uuid` (varchar(255), PK) - Host UUID (primary key, FK to hosts via uuid)
- `encrypted_password` (blob, NOT NULL) - Encrypted recovery lock password
- `status` (varchar(20), FK, nullable) - MDM delivery status (references mdm_delivery_status)
- `operation_type` (varchar(20), FK, NOT NULL) - MDM operation type (references mdm_operation_types)
- `error_message` (text, nullable) - Error message if operation failed
- `deleted` (tinyint(1), NOT NULL, DEFAULT 0) - Soft delete flag
- `auto_rotate_at` (timestamp(6), nullable) - Scheduled time for automatic password rotation
- `pending_encrypted_password` (blob, nullable) - Encrypted password pending rotation (held until MDM confirms success)
- `pending_error_message` (text, nullable) - Error message if password rotation failed
- `created_at` (timestamp(6), DEFAULT CURRENT_TIMESTAMP(6))
- `updated_at` (timestamp(6), DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE)

**Relationships:**
- One-to-one with `hosts` via `host_uuid`
- Many-to-one with `mdm_delivery_status` via `status` (ON UPDATE CASCADE)
- Many-to-one with `mdm_operation_types` via `operation_type` (ON UPDATE CASCADE)

**Indexes:**
- Primary key on `host_uuid`
- Index on `status`
- Index on `operation_type`
- Index on `deleted`
- Index on `auto_rotate_at` (`idx_auto_rotate_at`)

**Usage Notes:**
- Passwords encrypted at rest using Fleet's encryption key
- `auto_rotate_at` enables scheduled password rotation for compliance
- Soft delete via `deleted` flag preserves audit history
- Status tracks MDM command delivery lifecycle (pending, acknowledged, error)

**Platform Affinity:** Apple (macOS recovery lock)

**Related Migrations:**
- Created: `20260316120009_CreateHostRecoveryKeyPasswordsTable.go`
- Added `pending_encrypted_password`, `pending_error_message`: `20260317120000_AddRecoveryLockPasswordRotation.go`
- Added `auto_rotate_at`: `20260326131501_AddRecoveryLockAutoRotateAt.go`

---

### certificate_templates

**Purpose:** Defines certificate templates used for issuing identity certificates to hosts via SCEP or similar protocols.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT) - Primary key
- `team_id` (int unsigned, FK) - Team scope
- `certificate_authority_id` (int, FK) - Reference to certificate_authorities
- `name` (varchar(255)) - Template name (unique per team)
- `subject_name` (text) - Certificate subject name template (supports Fleet variables)
- `created_at`, `updated_at` (timestamp)

**Relationships:**
- Many-to-one with `teams` via `team_id`
- Many-to-one with `certificate_authorities` via `certificate_authority_id`
- One-to-many with `host_certificate_templates` via `certificate_template_id`

**Indexes:**
- Primary key on `id`
- Unique index on `(team_id, name)` - one template per name per team

**Common Query Patterns:**
```sql
-- Get certificate templates for a team
SELECT ct.*, ca.name AS ca_name
FROM certificate_templates ct
JOIN certificate_authorities ca ON ca.id = ct.certificate_authority_id
WHERE ct.team_id = ?;

-- Get template by name
SELECT * FROM certificate_templates
WHERE team_id = ? AND name = ?;
```

**Usage Notes:**
- Templates define how certificates are issued to devices
- Subject name can include Fleet variables like `$FLEET_VAR_HOST_UUID`
- Linked to a Certificate Authority for signing
- Used for identity certificates, conditional access, etc.

**Platform Affinity:** Common (Apple, Windows)

---

### host_certificate_templates

**Purpose:** Tracks certificate template deployment status for each host. One row per host-template combination.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT) - Primary key
- `host_uuid` (varchar(255)) - Host UUID
- `certificate_template_id` (int unsigned) - Reference to certificate_templates
- `name` (varchar(255)) - Template name (denormalized)
- `fleet_challenge` (char(32), nullable) - SCEP challenge for enrollment
- `status` (varchar(20), DEFAULT 'pending') - Status: pending, delivering, verified, failed
- `operation_type` (varchar(20), DEFAULT 'install') - Operation: install, remove
- `detail` (text, nullable) - Error details if status = failed
- `uuid` (binary(16), nullable) - Unique identifier for the certificate
- `not_valid_before` (datetime, nullable) - Certificate validity start
- `not_valid_after` (datetime, nullable) - Certificate expiration (for renewal tracking)
- `serial` (varchar(40), nullable) - Certificate serial number
- `created_at`, `updated_at` (timestamp)

**Relationships:**
- Many-to-one with `certificate_templates` via `certificate_template_id`
- Indexed reference to `hosts` via `host_uuid`
- Many-to-one with `mdm_operation_types` via `operation_type`

**Indexes:**
- Primary key on `id`
- Unique index on `(host_uuid, certificate_template_id)` - one status per template per host
- Index on `not_valid_after` - for efficient renewal queries

**Common Query Patterns:**
```sql
-- Get certificate status for a host
SELECT hct.*, ct.name AS template_name
FROM host_certificate_templates hct
JOIN certificate_templates ct ON ct.id = hct.certificate_template_id
WHERE hct.host_uuid = ?;

-- Find certificates expiring soon
SELECT * FROM host_certificate_templates
WHERE not_valid_after < DATE_ADD(NOW(), INTERVAL 30 DAY)
  AND status = 'verified';

-- Get hosts where certificate failed
SELECT host_uuid, detail
FROM host_certificate_templates
WHERE certificate_template_id = ? AND status = 'failed';
```

**Usage Notes:**
- Status lifecycle: pending → delivering → verified (or failed)
- `fleet_challenge` is NULL for pending, populated when delivering
- `not_valid_after` enables proactive certificate renewal
- Certificate validity tracked for compliance reporting
- Similar pattern to `host_mdm_apple_profiles` and `host_mdm_windows_profiles`

**Platform Affinity:** Common (Apple, Windows)

---

### conditional_access_scep_serials

**Purpose:** Tracks SCEP certificate serial numbers for Microsoft Conditional Access certificates.

**Key Fields:**
- `serial` (bigint unsigned, PK, AUTO_INCREMENT) - Certificate serial number (starts at 2)
- `created_at` (datetime(6)) - Creation timestamp

**Relationships:**
- One-to-one with `conditional_access_scep_certificates` via `serial`

**Indexes:**
- Primary key on `serial`

**Usage Notes:**
- Serial number 1 is reserved for system use
- Used exclusively for Microsoft Conditional Access integration
- Certificates enable device-based conditional access policies

**Platform Affinity:** Windows (Microsoft Entra ID / Conditional Access)

---

### conditional_access_scep_certificates

**Purpose:** Stores SCEP certificates issued for Microsoft Conditional Access device authentication.

**Key Fields:**
- `serial` (bigint unsigned, PK, FK) - Certificate serial (references conditional_access_scep_serials)
- `host_id` (int unsigned) - Host the certificate was issued to
- `name` (varchar(64)) - Certificate common name
- `not_valid_before` (datetime) - Certificate validity start
- `not_valid_after` (datetime) - Certificate expiration
- `certificate_pem` (text) - Full certificate in PEM format
- `revoked` (tinyint(1), DEFAULT 0) - Whether certificate has been revoked
- `created_at`, `updated_at` (datetime(6))

**Relationships:**
- One-to-one with `conditional_access_scep_serials` via `serial`
- Many-to-one with `hosts` via `host_id` (indexed, not FK)

**Indexes:**
- Primary key on `serial`
- Index on `host_id` for host lookups

**Common Query Patterns:**
```sql
-- Get certificate for a host
SELECT * FROM conditional_access_scep_certificates
WHERE host_id = ? AND revoked = 0
ORDER BY not_valid_after DESC;

-- Find expiring certificates
SELECT * FROM conditional_access_scep_certificates
WHERE not_valid_after < DATE_ADD(NOW(), INTERVAL 7 DAY)
  AND revoked = 0;

-- Revoke certificate
UPDATE conditional_access_scep_certificates
SET revoked = 1
WHERE serial = ?;
```

**Usage Notes:**
- Enables Microsoft Conditional Access policies via device certificates
- Certificates validated against Fleet's SCEP CA
- PEM format stored with CHECK constraint validation
- Used for Windows devices connecting to Microsoft 365/Azure AD resources

**Platform Affinity:** Windows

---

### host_conditional_access

**Purpose:** Tracks Microsoft Conditional Access bypass status for hosts. Used when hosts temporarily need access bypasses during enrollment or remediation.

**Key Fields:**
- `id` (int unsigned, PK, AUTO_INCREMENT) - Primary key
- `host_id` (int unsigned, UNIQUE) - One-to-one with hosts
- `bypassed_at` (timestamp, nullable) - When bypass was activated (NULL = not bypassed)
- `created_at` (timestamp(6), DEFAULT CURRENT_TIMESTAMP(6))
- `updated_at` (timestamp(6), DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE)

**Relationships:**
- One-to-one with `hosts` via `host_id` (unique constraint)

**Indexes:**
- Primary key on `id`
- Unique index on `host_id` (`idx_host_conditional_access_host_id`)

**Common Query Patterns:**
```sql
-- Check if host has active bypass
SELECT * FROM host_conditional_access
WHERE host_id = ? AND bypassed_at IS NOT NULL;

-- Get all hosts with active bypasses
SELECT h.*, hca.bypassed_at
FROM hosts h
INNER JOIN host_conditional_access hca ON hca.host_id = h.id
WHERE hca.bypassed_at IS NOT NULL;

-- Clear bypass for a host
UPDATE host_conditional_access
SET bypassed_at = NULL
WHERE host_id = ?;
```

**Usage Notes:**
- Used with Microsoft Conditional Access integration
- `bypassed_at` NULL means no active bypass
- Bypasses allow devices to access Microsoft 365/Azure AD resources during setup
- Related to `conditional_access_scep_certificates` for certificate-based access

**Platform Affinity:** Windows (Microsoft Entra ID / Conditional Access)

---

## Common Patterns

### Pattern: Self-Service Software Installation (VPP Apps - iOS/iPadOS)

**Tables Involved:**
1. `vpp_apps` - VPP app catalog (App Store metadata)
2. `vpp_apps_teams` - Deployment configuration (which teams get which apps)
3. `hosts` - Target devices
4. `host_device_auth` - Device authentication tokens
5. `activities` - Installation activity logging
6. `host_vpp_software_installs` - Installation tracking

**Flow:**
```
End user visits /device/{token}/self-service
  → Query vpp_apps_teams WHERE team_id = host.team_id AND self_service = 1
  → Join vpp_apps for app metadata (name, icon, version)
  → Display available apps

User clicks "Install" on an app
  → POST /device/{token}/software/vpp/{vpp_app_team_id}/install
  → Verify host.team_id matches vpp_apps_teams.team_id
  → Insert into host_vpp_software_installs (status = pending)
  → MDM InstallApplication command sent to device (via nano_commands for Apple)
  → Activity created: installed_app_store_app (user = end user, via device token)

Device checks in with MDM
  → Command result received (acknowledged = success, error = failure)
  → Update host_vpp_software_installs status (installed or failed)
  → Insert into host_software when installed
```

---

### Pattern: Configuration Profile Deployment (Apple)

**Tables Involved:**
1. `mdm_apple_configuration_profiles` - Profile definitions
2. `hosts` - Target devices
3. `host_mdm` - MDM enrollment verification
4. `host_mdm_apple_profiles` - Deployment status tracking
5. `mdm_configuration_profile_labels` - Label-based targeting
6. `label_membership` - Host label membership

**Flow:**
```
Admin uploads profile
  → POST /api/v1/fleet/mdm/apple/profiles
  → Parse mobileconfig, extract identifier and name
  → Replace Fleet variables ($FLEET_VAR_HOST_UUID, etc.) with placeholders
  → Calculate checksum of mobileconfig
  → Insert into mdm_apple_configuration_profiles
  → Optionally associate with labels (insert into mdm_configuration_profile_labels)

Profile deployment (background job)
  → Find hosts in team with platform in (darwin, ios, ipados)
  → Join host_mdm WHERE is_server = 1 AND enrolled = 1
  → Filter by label targeting (if configured)
  → For each target host:
    - Substitute Fleet variables with host-specific values
    - Insert/update host_mdm_apple_profiles (status = pending)
    - Send MDM InstallProfile command (nano_commands)
    - Record command_uuid in host_mdm_apple_profiles

Device checks in
  → Device reports installed profiles
  → Update host_mdm_apple_profiles (status = verifying)
  → Verify profile identifier present in device list
  → Update status to verified (or failed if not present)

Profile update detection
  → Calculate new checksum when profile edited
  → Compare checksum in host_mdm_apple_profiles
  → If different, trigger reinstallation flow
```

---

### Pattern: Policy Evaluation and Compliance

**Tables Involved:**
1. `policies` - Policy definitions with queries
2. `hosts` - Devices being checked
3. `policy_membership` - Pass/fail results
4. `teams` - Policy scoping
5. `activities` - Policy check logging

**Flow:**
```
Scheduled policy check (background job, hourly)
  → For each host:
    - Get policies for host's team (policies.team_id = host.team_id OR NULL)
    - Filter policies by platform (host.platform in policy.platforms)
    - For each applicable policy:
      * Run policy.query on host via osquery
      * If query returns rows: passes = 1
      * If query returns empty: passes = 0
      * If query errors: passes = NULL
      * Upsert policy_membership with result
      * Update policy_membership.updated_at

Webhook notifications (if configured)
  → Query policy_membership for recent failures
  → Filter by configured policy_ids
  → Batch hosts by policy
  → POST to webhook destination_url with failing hosts

Calendar events (if enabled for policy)
  → Query policy_membership WHERE passes = 0 AND policy.calendar_events_enabled = 1
  → Check if calendar_event exists for host
  → If not, create calendar event via Google Calendar API
  → Insert into calendar_events and host_calendar_events
  → When policy passes, delete calendar event

Remediation automation (if configured)
  → Policy failure detected
  → Check policy.software_installer_id, policy.script_id, or policy.vpp_apps_teams_id
  → Queue installation/script execution via upcoming_activities
```

---

### Pattern: Software Vulnerability Tracking

**Tables Involved:**
1. `host_software` - Software installed on hosts
2. `software` - Software version catalog
3. `software_cpe` - CPE identifiers for software
4. `software_cve` - CVE to software mappings
5. `cve_meta` - CVE metadata (scores, descriptions)
6. `vulnerability_host_counts` - Aggregated vulnerability counts

**Flow:**
```
Software inventory collection
  → osquery reports installed software via detail queries
  → Parse results (apps, programs, packages tables)
  → Upsert into software table (name, version, source, vendor, etc.)
  → Upsert into host_software (host_id, software_id)

Vulnerability scanning (background job, periodic)
  → For each software without CPE:
    - Generate CPE from name, version, vendor
    - Insert into software_cpe
  → Match CPEs to NVD database
  → Insert CVE matches into software_cve
  → Sync CVE metadata into cve_meta (CVSS, EPSS, CISA KEV)
  → Update vulnerability_host_counts (materialized view)

Vulnerability reporting
  → Query host_software → software_cve → cve_meta
  → Join host_software to get affected hosts
  → Filter by CVSS threshold (e.g., >= 7.0 for High)
  → Prioritize by CISA KEV status and EPSS probability
  → Display in UI, trigger webhooks if configured
```

---

### Pattern: Windows MDM Configuration

**Tables Involved:**
1. `mdm_windows_configuration_profiles` - SyncML profiles
2. `host_mdm_windows_profiles` - Deployment tracking
3. `windows_mdm_commands` - MDM command definitions
4. `windows_mdm_command_queue` - Command queue
5. `windows_mdm_command_results` - Execution results

**Flow:**
```
Profile upload
  → POST /api/v1/fleet/mdm/windows/profiles
  → Parse SyncML XML
  → Insert into mdm_windows_configuration_profiles

Profile deployment
  → Find Windows hosts in team
  → Insert into windows_mdm_commands (SyncML Exec/Replace commands)
  → Insert into windows_mdm_command_queue for each host
  → Insert/update host_mdm_windows_profiles (status = pending)

Device sync
  → Windows device checks in (MS-MDM protocol)
  → Fleet sends queued commands from windows_mdm_command_queue
  → Device executes commands, returns results
  → Insert into windows_mdm_command_results
  → Update host_mdm_windows_profiles (status = verifying → verified)
```

---

### Pattern: Android Policy Application

**Tables Involved:**
1. `android_devices` - Enrolled Android devices
2. `android_enterprises` - Enterprise configuration
3. `mdm_android_configuration_profiles` - Profile definitions (JSON)
4. `host_mdm_android_profiles` - Deployment status
5. `android_policy_requests` - API request tracking

**Flow:**
```
Profile creation
  → POST /api/v1/fleet/mdm/android/profiles
  → Parse JSON policy fragment
  → Validate against Android Management API schema
  → Insert into mdm_android_configuration_profiles

Policy reconciliation (background job)
  → For each Android device:
    - Get all applicable profiles for device's team
    - Merge JSON policies into complete device policy
    - Compare with android_devices.applied_policy_version
    - If different:
      * PATCH to Android Management API
      * Record request in android_policy_requests
      * Update android_devices (applied_policy_id, applied_policy_version)
      * Update host_mdm_android_profiles status

Device sync
  → Device checks in with Google
  → Policy application status reported
  → Fleet syncs status from Android Management API
  → Update host_mdm_android_profiles (verified or failed)
```

---

## Maintenance Notes

### Updating This Dictionary

When adding or modifying tables:
1. Update the relevant section with current schema
2. Document datatypes, constraints, and indexes
3. Add relationships with ON DELETE/UPDATE behavior
4. Provide common query patterns (actual SQL)
5. Include usage notes with practical information
6. Update "Last Updated" date at top
7. Reference migration files if significant

### Finding Schema Information

```bash
# View current schema
mysql -u fleet -p fleet_db -e "SHOW TABLES;"

# Describe a table
mysql -u fleet -p fleet_db -e "DESCRIBE hosts;"

# View indexes
mysql -u fleet -p fleet_db -e "SHOW INDEX FROM hosts;"

# View foreign keys
mysql -u fleet -p fleet_db -e "
  SELECT
    TABLE_NAME, COLUMN_NAME, CONSTRAINT_NAME,
    REFERENCED_TABLE_NAME, REFERENCED_COLUMN_NAME
  FROM information_schema.KEY_COLUMN_USAGE
  WHERE TABLE_SCHEMA = 'fleet_db' AND TABLE_NAME = 'hosts';
"

# Export full schema
mysqldump -u fleet -p --no-data fleet_db > schema.sql

# Find migrations
ls -la server/datastore/mysql/migrations/tables/

# View generated schema
cat server/datastore/mysql/schema.sql
```

### Schema Evolution

- All schema changes MUST go through migrations in `server/datastore/mysql/migrations/tables/`
- Migrations are timestamped: `YYYYMMDDHHMMSS_Description.go`
- Each migration has Up (apply) and Down (rollback) functions
- Check existing migrations for patterns before creating new ones
- Test migrations on copy of production data before deploying
- Coordinate schema changes with application code changes (backward compatibility)

### Index Optimization

- Indexes created for foreign keys automatically
- Additional indexes for frequently queried columns
- Composite indexes for multi-column WHERE clauses
- Analyze query plans with `EXPLAIN` to identify missing indexes
- Monitor slow query log for optimization opportunities

---

## Recent Schema Changes (November 2025 - March 2026)

This section documents significant schema changes since the previous version of this document.

### New Tables

| Table | Purpose | Migration |
|-------|---------|-----------|
| `certificate_templates` | Certificate template definitions for identity certs | 20251124140138 |
| `host_certificate_templates` | Tracks cert deployment status per host | 20251121124239 |
| `conditional_access_scep_serials` | Serial numbers for MS Conditional Access certs | 20251106000000 |
| `conditional_access_scep_certificates` | MS Conditional Access SCEP certificates | 20251106000000 |
| `software_title_display_names` | Custom display names for software per team | 20251103160848 |
| `software_update_schedules` | Automatic software update windows | 20251229000010 |
| `android_app_configurations` | Android managed app configurations | 20251124135808 |
| `in_house_app_software_categories` | Links in-house apps to categories | 20251110172137 |
| `host_last_known_locations` | Host GPS location tracking | 20260108200708 |
| `host_conditional_access` | Tracks MS Conditional Access bypass status for hosts | 20260126210724 |
| `host_recovery_key_passwords` | macOS recovery lock passwords with auto-rotation | 20260316120009 |

### Modified Tables

| Table | Change | Migration |
|-------|--------|-----------|
| `hosts` | Added `last_restarted_at`, `timezone` columns | 20251124162948, 20251229000010 |
| `hosts` | Added `idx_hosts_hostname` index | 20251215163721 |
| `labels` | Added `team_id` column (team-scoped labels) | 20251207050413 |
| `software_titles` | Added `upgrade_code` column (Windows MSI matching) | 20251107170854 |
| `software` | Added `upgrade_code` column | 20251107170854 |
| `in_house_apps` | Added `self_service`, `url` columns | 20251107164629, 20251111153133 |
| `host_in_house_software_installs` | Added `self_service` column | 20251107164629 |
| `vpp_apps` | Increased `adam_id` to varchar(255) for Android | 20251117020100 |
| `vpp_apps_teams` | Made `vpp_token_id` nullable, increased `adam_id` | 20251117020100 |
| `host_vpp_software_installs` | Added `retry_count` column | 20260108214732 |
| `host_script_results` | Added `attempt_number` column, new index | 20260109231821 |
| `host_software_installs` | Added `attempt_number` column, new index | 20260109231821 |
| `host_software_installed_paths` | Renamed `executable_sha256` to `cdhash_sha256`, added new columns | 20260113012054 |
| `host_mdm_apple_bootstrap_packages` | Added `skipped` column | 20251229000020 |
| `certificate_authorities` | Added `est` CA type | 20251104112849 |
| `activities` | Added multiple new indexes for search | 20251121100000 |
| `users` | Added `idx_users_name` index | 20251121100000 |
| `mdm_windows_enrollments` | Added `credentials_hash`, `credentials_acknowledged` columns | 20260126150840 |
| `host_certificate_templates` | Changed `host_uuid` from VARCHAR(36) to VARCHAR(255) | 20260202151756 |
| `policies` | Added `conditional_access_bypass_enabled` column | 20260210181120 |
| `host_mdm_android_profiles` | Added `can_reverify` column | 20260210155109 |
| `nano_enrollment_queue` | Added composite index `idx_neq_filter` on (active, priority, created_at) | 20260210151544 |
| `nano_command_results` | Added composite index `idx_ncr_lookup` on (id, command_uuid, status) | 20260210151544 |
| `host_device_auth` | Added `previous_token` column (VARCHAR(255), nullable), added `idx_host_device_auth_previous_token` index | 20260217200906 |
| `software_installers` | Added `is_active` column (TINYINT(1) NOT NULL DEFAULT 0); dropped `idx_software_installers_team_id_title_id` and `idx_software_installers_platform_title_id` indexes; added UNIQUE `idx_software_installers_team_title_version` on (global_or_team_id, title_id, version) | 20260218175704 |
| `software_host_counts` | Deleted all zero-count rows; added CHECK constraint (hosts_count > 0) | 20260223000000 |
| `software_titles_host_counts` | Deleted all zero-count rows; added CHECK constraint (hosts_count > 0) | 20260223000000 |
| `software_titles_host_counts` | Dropped `idx_software_titles_host_counts_team_counts_title`; added covering index `idx_software_titles_host_counts_team_global_hosts` on (team_id, global_stats, hosts_count, software_title_id) | 20260226182000 |
| `policies` | Removed `conditional_access_bypass_enabled`; added `type` (ENUM), `patch_software_title_id` (FK), `needs_full_membership_cleanup` columns; added unique index on (team_id, patch_software_title_id) | 20260316120007, 20260316120010, 20260316120011 |
| `activities` | Renamed table to `activity_past`; `host_activities` renamed to `activity_host_past` | 20260316120008 |
| `in_house_app_labels` | Added `require_all` column (BOOL NOT NULL DEFAULT false) | 20260318184559 |
| `software_installer_labels` | Added `require_all` column (BOOL NOT NULL DEFAULT false) | 20260318184559 |
| `vpp_app_team_labels` | Added `require_all` column (BOOL NOT NULL DEFAULT false) | 20260318184559 |
| `operating_systems` | Modified `arch` from VARCHAR(150) to VARCHAR(100); added `installation_type` column (VARCHAR(20)); updated unique index | 20260319120000 |
| `query_results` | Added `has_data` virtual column; added composite index `idx_query_id_has_data_host_id_last_fetched` | 20260324223334 |
| `software_installers` | Added `patch_query` column (TEXT NOT NULL) | 20260324161944 |
| `host_software` | Added index `idx_host_software_software_id` on (software_id) | 20260314120000 |
| `kernel_host_counts` | Dropped foreign key `kernel_host_counts_ibfk_1` (unnecessary due to swap-table rebuild pattern) | 20260316120000 |
| `host_recovery_key_passwords` | Added `pending_encrypted_password` (BLOB) and `pending_error_message` (TEXT) columns for password rotation | 20260317120000 |
| `host_recovery_key_passwords` | Added `auto_rotate_at` column (TIMESTAMP(6) NULL); added index `idx_auto_rotate_at` | 20260326131501 |

### Data Changes

| Change | Description | Migration |
|--------|-------------|-----------|
| `software_categories` | Added 'Security' and 'Utilities' categories | 20251217120000 |
| `app_config_json` | Added `windows_entra_tenant_ids` to MDM integrations config | 20260205184907 |
| `software_titles` | Unmarked 'kernel-core' RPM packages as kernel | 20260211200153 |
| `kernel_host_counts` | Removed entries for 'kernel-core' RPM packages | 20260211200153 |
| `host_script_results` | Backfilled NULL `attempt_number` to 0 for completed results | 20260124200020 |
| `host_software_installs` | Backfilled NULL `attempt_number` to 0 for completed installs | 20260124200020 |
| `labels` | Reset invalid `platform` values to empty string (kept only `''`, `centos`, `darwin`, `windows`, `ubuntu`) | 20260217141240 |
| `software_titles` | Migrated `source` from `pkg_packages` to `apps` for entries with non-empty `bundle_identifier` | 20260217181748 |
| `software_installers`, `software_titles`, `software` | Fixed mismatched software title references — re-pointed installers and software entries to titles with correct `source` | 20260218165545 |
| `software_installers` | Set `is_active = 1` for all existing installers (backfill for new column) | 20260218175704 |
| `host_mdm_windows_profiles` | Fixed stuck profiles: moved `verifying` → `verified`; moved `failed` with detail `'Failed, was verifying'` → `verified` | 20260225143121 |
| `app_config_json`, `teams.config` | Added `lock_end_user_info` JSON field (set to match existing `enable_end_user_authentication` value) | 20260228115022 |
| `app_config_json` | Added `GitOpsConfig.Exceptions` flags | 20260323144117 |
| `policies` | Migrated `conditional_access_bypass_enabled` data to `critical` flag before column removal | 20260316120007 |
| `software_titles`, `software` | Updated names to match Fleet-maintained app (FMA) canonical names | 20260326210603 |

---

## References

- **Fleet REST API Docs:** `/docs/REST API/` - API endpoints that use these tables
- **Activity Logs Reference:** `/docs/Contributing/reference/audit-logs.md` - Activity types catalog
- **Database Migrations:** `server/datastore/mysql/migrations/tables/` - Schema change history
- **Model Definitions:** `server/fleet/` - Go structs matching tables
- **Datastore Interface:** `server/datastore/datastore.go` - Database access patterns
- **Schema File:** `server/datastore/mysql/schema.sql` - Complete database schema


