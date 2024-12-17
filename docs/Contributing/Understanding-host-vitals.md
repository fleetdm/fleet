<!-- DO NOT EDIT. This document is automatically generated. -->
# Understanding host vitals

Following is a summary of the detail queries hardcoded in Fleet used to populate the device details:

## battery

- Platforms: windows, darwin

- Discovery query:
```sql
SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = 'battery'
```

- Query:
```sql
SELECT serial_number, cycle_count, designed_capacity, max_capacity FROM battery
```

## chromeos_profile_user_info

- Platforms: chrome

- Query:
```sql
SELECT email FROM users
```

## disk_encryption_darwin

- Platforms: darwin

- Query:
```sql
SELECT 1 FROM disk_encryption WHERE user_uuid IS NOT "" AND filevault_status = 'on' LIMIT 1
```

## disk_encryption_linux

- Platforms: linux, ubuntu, debian, rhel, centos, sles, kali, gentoo, amzn, pop, arch, linuxmint, void, nixos, endeavouros, manjaro, opensuse-leap, opensuse-tumbleweed, tuxedo

- Query:
```sql
SELECT de.encrypted, m.path FROM disk_encryption de JOIN mounts m ON m.device_alias = de.name;
```

## disk_encryption_windows

- Platforms: windows

- Query:
```sql
WITH encrypted(enabled) AS (
		SELECT CASE WHEN
			NOT EXISTS(SELECT 1 FROM windows_optional_features WHERE name = 'BitLocker')
			OR
			(SELECT 1 FROM windows_optional_features WHERE name = 'BitLocker' AND state = 1)
		THEN (SELECT 1 FROM bitlocker_info WHERE drive_letter = 'C:' AND protection_status = 1)
	END)
	SELECT 1 FROM encrypted WHERE enabled IS NOT NULL
```

## disk_space_unix

- Platforms: linux, ubuntu, debian, rhel, centos, sles, kali, gentoo, amzn, pop, arch, linuxmint, void, nixos, endeavouros, manjaro, opensuse-leap, opensuse-tumbleweed, tuxedo, darwin

- Query:
```sql
SELECT (blocks_available * 100 / blocks) AS percent_disk_space_available,
       round((blocks_available * blocks_size * 10e-10),2) AS gigs_disk_space_available,
       round((blocks           * blocks_size * 10e-10),2) AS gigs_total_disk_space
FROM mounts WHERE path = '/' LIMIT 1;
```

## disk_space_windows

- Platforms: windows

- Query:
```sql
SELECT ROUND((sum(free_space) * 100 * 10e-10) / (sum(size) * 10e-10)) AS percent_disk_space_available,
       ROUND(sum(free_space) * 10e-10) AS gigs_disk_space_available,
       ROUND(sum(size)       * 10e-10) AS gigs_total_disk_space
FROM logical_drives WHERE file_system = 'NTFS' LIMIT 1;
```

## google_chrome_profiles

- Platforms: all

- Discovery query:
```sql
SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = 'google_chrome_profiles'
```

- Query:
```sql
SELECT email FROM google_chrome_profiles WHERE NOT ephemeral AND email <> ''
```

## kubequery_info

- Platforms: all

- Discovery query:
```sql
SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = 'kubernetes_info'
```

- Query:
```sql
SELECT * from kubernetes_info
```

## mdm

- Platforms: darwin

- Discovery query:
```sql
SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = 'mdm'
```

- Query:
```sql
select enrolled, server_url, installed_from_dep, payload_identifier from mdm;
```

## mdm_config_profiles_darwin

- Platforms: darwin

- Discovery query:
```sql
SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = 'macos_profiles'
```

- Query:
```sql
SELECT display_name, identifier, install_date FROM macos_profiles where type = "Configuration";
```

## mdm_disk_encryption_key_file_darwin

- Platforms: darwin

- Discovery query:
```sql
SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = 'filevault_prk'
```

- Query:
```sql
WITH
		de AS (SELECT IFNULL((SELECT 1 FROM disk_encryption WHERE user_uuid IS NOT "" AND filevault_status = 'on' LIMIT 1), 0) as encrypted),
		fv AS (SELECT base64_encrypted as filevault_key FROM filevault_prk)
	SELECT encrypted, filevault_key FROM de LEFT JOIN fv;
```

## mdm_disk_encryption_key_file_lines_darwin

- Platforms: darwin

- Discovery query:
```sql
SELECT 1 WHERE EXISTS (SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = 'file_lines') AND NOT EXISTS (SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = 'filevault_prk');
```

- Query:
```sql
WITH
		de AS (SELECT IFNULL((SELECT 1 FROM disk_encryption WHERE user_uuid IS NOT "" AND filevault_status = 'on' LIMIT 1), 0) as encrypted),
		fl AS (SELECT line FROM file_lines WHERE path = '/var/db/FileVaultPRK.dat')
	SELECT encrypted, hex(line) as hex_line FROM de LEFT JOIN fl;
```

## mdm_windows

- Platforms: windows

- Query:
```sql
WITH registry_keys AS (
                        SELECT *
                        FROM registry
                        WHERE path LIKE 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Enrollments\%%'
                    ),
                    enrollment_info AS (
                        SELECT
                            MAX(CASE WHEN name = 'UPN' THEN data END) AS upn,
                            MAX(CASE WHEN name = 'DiscoveryServiceFullURL' THEN data END) AS discovery_service_url,
                            MAX(CASE WHEN name = 'ProviderID' THEN data END) AS provider_id,
                            MAX(CASE WHEN name = 'EnrollmentState' THEN data END) AS state,
                            MAX(CASE WHEN name = 'AADResourceID' THEN data END) AS aad_resource_id
                        FROM registry_keys
                        GROUP BY key
                    ),
                    installation_info AS (
                        SELECT data AS installation_type
                        FROM registry
                        WHERE path = 'HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion\InstallationType'
                        LIMIT 1
                    )
                    SELECT
                        e.aad_resource_id,
                        e.discovery_service_url,
                        e.provider_id,
                        i.installation_type
                    FROM installation_info i
                    LEFT JOIN enrollment_info e ON e.upn IS NOT NULL
		    -- coalesce to 'unknown' and keep that state in the list
		    -- in order to account for hosts that might not have this
		    -- key, and servers
                    WHERE COALESCE(e.state, '0') IN ('0', '1', '2', '3')
                    LIMIT 1;
```

## munki_info

- Platforms: darwin

- Discovery query:
```sql
SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = 'munki_info'
```

- Query:
```sql
select version, errors, warnings from munki_info;
```

## network_interface_chrome

- Platforms: chrome

- Query:
```sql
SELECT ipv4 AS address, mac FROM network_interfaces LIMIT 1
```

## network_interface_unix

- Platforms: linux, ubuntu, debian, rhel, centos, sles, kali, gentoo, amzn, pop, arch, linuxmint, void, nixos, endeavouros, manjaro, opensuse-leap, opensuse-tumbleweed, tuxedo, darwin

- Query:
```sql
SELECT
    ia.address,
    id.mac
FROM
    interface_addresses ia
    JOIN interface_details id ON id.interface = ia.interface
	-- On Unix ia.interface is the name of the interface,
	-- whereas on Windows ia.interface is the IP of the interface.
    JOIN routes r ON r.interface = ia.interface
WHERE
	-- Destination 0.0.0.0/0 or ::/0 (IPv6) is the default route on route tables.
    (r.destination = '0.0.0.0' OR r.destination = '::') AND r.netmask = 0
	-- Type of route is "gateway" for Unix, "remote" for Windows.
    AND r.type = 'gateway'
	-- We are only interested on private IPs (some devices have their Public IP as Primary IP too).
    AND (
		-- Private IPv4 addresses.
		inet_aton(ia.address) IS NOT NULL AND (
			split(ia.address, '.', 0) = '10'
			OR (split(ia.address, '.', 0) = '172' AND (CAST(split(ia.address, '.', 1) AS INTEGER) & 0xf0) = 16)
			OR (split(ia.address, '.', 0) = '192' AND split(ia.address, '.', 1) = '168')
		)
		-- Private IPv6 addresses start with 'fc' or 'fd'.
		OR (inet_aton(ia.address) IS NULL AND regex_match(lower(ia.address), '^f[cd][0-9a-f][0-9a-f]:[0-9a-f:]+', 0) IS NOT NULL)
	)
ORDER BY
    r.metric ASC,
	-- Prefer IPv4 addresses over IPv6 addresses if their route have the same metric.
	inet_aton(ia.address) IS NOT NULL DESC
LIMIT 1;
```

## network_interface_windows

- Platforms: windows

- Query:
```sql
SELECT
    ia.address,
    id.mac
FROM
    interface_addresses ia
    JOIN interface_details id ON id.interface = ia.interface
	-- On Unix ia.interface is the name of the interface,
	-- whereas on Windows ia.interface is the IP of the interface.
    JOIN routes r ON r.interface = ia.address
WHERE
	-- Destination 0.0.0.0/0 or ::/0 (IPv6) is the default route on route tables.
    (r.destination = '0.0.0.0' OR r.destination = '::') AND r.netmask = 0
	-- Type of route is "gateway" for Unix, "remote" for Windows.
    AND r.type = 'remote'
	-- We are only interested on private IPs (some devices have their Public IP as Primary IP too).
    AND (
		-- Private IPv4 addresses.
		inet_aton(ia.address) IS NOT NULL AND (
			split(ia.address, '.', 0) = '10'
			OR (split(ia.address, '.', 0) = '172' AND (CAST(split(ia.address, '.', 1) AS INTEGER) & 0xf0) = 16)
			OR (split(ia.address, '.', 0) = '192' AND split(ia.address, '.', 1) = '168')
		)
		-- Private IPv6 addresses start with 'fc' or 'fd'.
		OR (inet_aton(ia.address) IS NULL AND regex_match(lower(ia.address), '^f[cd][0-9a-f][0-9a-f]:[0-9a-f:]+', 0) IS NOT NULL)
	)
ORDER BY
    r.metric ASC,
	-- Prefer IPv4 addresses over IPv6 addresses if their route have the same metric.
	inet_aton(ia.address) IS NOT NULL DESC
LIMIT 1;
```

## orbit_info

- Platforms: linux, ubuntu, debian, rhel, centos, sles, kali, gentoo, amzn, pop, arch, linuxmint, void, nixos, endeavouros, manjaro, opensuse-leap, opensuse-tumbleweed, tuxedo, darwin, windows

- Discovery query:
```sql
SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = 'orbit_info'
```

- Query:
```sql
SELECT * FROM orbit_info
```

## os_chrome

- Platforms: chrome

- Query:
```sql
SELECT
		os.name,
		os.major,
		os.minor,
		os.patch,
		os.build,
		os.arch,
		os.platform,
		os.version AS version,
		os.version AS kernel_version
	FROM
		os_version os
```

## os_unix_like

- Platforms: linux, ubuntu, debian, rhel, centos, sles, kali, gentoo, amzn, pop, arch, linuxmint, void, nixos, endeavouros, manjaro, opensuse-leap, opensuse-tumbleweed, tuxedo, darwin

- Query:
```sql
SELECT
		os.name,
		os.major,
		os.minor,
		os.patch,
		os.extra,
		os.build,
		os.arch,
		os.platform,
		os.version AS version,
		k.version AS kernel_version
	FROM
		os_version os,
		kernel_info k
```

## os_version

- Platforms: all

- Query:
```sql
SELECT * FROM os_version LIMIT 1
```

## os_version_windows

- Platforms: windows

- Query:
```sql
WITH display_version_table AS (
			SELECT data as display_version
			FROM registry
			WHERE path = 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\DisplayVersion'
		),
		ubr_table AS (
			SELECT data AS ubr
			FROM registry
			WHERE path ='HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\UBR'
		)
		SELECT
			os.name,
			COALESCE(d.display_version, '') AS display_version,
			COALESCE(CONCAT((SELECT version FROM os_version), '.', u.ubr), k.version) AS version
		FROM
			os_version os,
			kernel_info k
		LEFT JOIN
			display_version_table d
		LEFT JOIN
			ubr_table u
```

## os_windows

- Platforms: windows

- Query:
```sql
WITH display_version_table AS (
		SELECT data as display_version
		FROM registry
		WHERE path = 'HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\DisplayVersion'
	),
	ubr_table AS (
	SELECT data AS ubr
	FROM registry
	WHERE path ='HKEY_LOCAL_MACHINE\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\UBR'
	)
	SELECT
		os.name,
		os.platform,
		os.arch,
		k.version as kernel_version,
		COALESCE(CONCAT((SELECT version FROM os_version), '.', u.ubr), k.version) AS version,
		COALESCE(d.display_version, '') AS display_version
	FROM
		os_version os,
		kernel_info k
	LEFT JOIN
		display_version_table d
	LEFT JOIN
		ubr_table u
```

## osquery_flags

- Platforms: linux, ubuntu, debian, rhel, centos, sles, kali, gentoo, amzn, pop, arch, linuxmint, void, nixos, endeavouros, manjaro, opensuse-leap, opensuse-tumbleweed, tuxedo, darwin, windows

- Query:
```sql
select name, value from osquery_flags where name in ("distributed_interval", "config_tls_refresh", "config_refresh", "logger_tls_period")
```

## osquery_info

- Platforms: all

- Query:
```sql
select * from osquery_info limit 1
```

## scheduled_query_stats

- Platforms: linux, ubuntu, debian, rhel, centos, sles, kali, gentoo, amzn, pop, arch, linuxmint, void, nixos, endeavouros, manjaro, opensuse-leap, opensuse-tumbleweed, tuxedo, darwin, windows

- Query:
```sql
SELECT *,
				(SELECT value from osquery_flags where name = 'pack_delimiter') AS delimiter
			FROM osquery_schedule
```

## software_chrome

- Platforms: chrome

- Query:
```sql
SELECT
  name AS name,
  version AS version,
  identifier AS extension_id,
  browser_type AS browser,
  'chrome_extensions' AS source,
  '' AS vendor,
  '' AS installed_path
FROM chrome_extensions
```

## software_linux

- Platforms: linux, ubuntu, debian, rhel, centos, sles, kali, gentoo, amzn, pop, arch, linuxmint, void, nixos, endeavouros, manjaro, opensuse-leap, opensuse-tumbleweed, tuxedo

- Query:
```sql
WITH cached_users AS (WITH cached_groups AS (select * from groups)
 SELECT uid, username, type, groupname, shell
 FROM users LEFT JOIN cached_groups USING (gid)
 WHERE type <> 'special' AND shell NOT LIKE '%/false' AND shell NOT LIKE '%/nologin' AND shell NOT LIKE '%/shutdown' AND shell NOT LIKE '%/halt' AND username NOT LIKE '%$' AND username NOT LIKE '\_%' ESCAPE '\' AND NOT (username = 'sync' AND shell ='/bin/sync' AND directory <> ''))
SELECT
  name AS name,
  version AS version,
  '' AS extension_id,
  '' AS browser,
  'deb_packages' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch,
  '' AS installed_path
FROM deb_packages
WHERE status LIKE '% ok installed'
UNION
SELECT
  package AS name,
  version AS version,
  '' AS extension_id,
  '' AS browser,
  'portage_packages' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch,
  '' AS installed_path
FROM portage_packages
UNION
SELECT
  name AS name,
  version AS version,
  '' AS extension_id,
  '' AS browser,
  'rpm_packages' AS source,
  release AS release,
  vendor AS vendor,
  arch AS arch,
  '' AS installed_path
FROM rpm_packages
UNION
SELECT
  name AS name,
  version AS version,
  '' AS extension_id,
  '' AS browser,
  'npm_packages' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch,
  path AS installed_path
FROM npm_packages
UNION
SELECT
  name AS name,
  version AS version,
  identifier AS extension_id,
  browser_type AS browser,
  'chrome_extensions' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch,
  path AS installed_path
FROM cached_users CROSS JOIN chrome_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  identifier AS extension_id,
  'firefox' AS browser,
  'firefox_addons' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch,
  path AS installed_path
FROM cached_users CROSS JOIN firefox_addons USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  '' AS extension_id,
  '' AS browser,
  'python_packages' AS source,
  '' AS release,
  '' AS vendor,
  '' AS arch,
  path AS installed_path
FROM python_packages;
```

## software_macos

- Platforms: darwin

- Query:
```sql
WITH cached_users AS (WITH cached_groups AS (select * from groups)
 SELECT uid, username, type, groupname, shell
 FROM users LEFT JOIN cached_groups USING (gid)
 WHERE type <> 'special' AND shell NOT LIKE '%/false' AND shell NOT LIKE '%/nologin' AND shell NOT LIKE '%/shutdown' AND shell NOT LIKE '%/halt' AND username NOT LIKE '%$' AND username NOT LIKE '\_%' ESCAPE '\' AND NOT (username = 'sync' AND shell ='/bin/sync' AND directory <> ''))
SELECT
  name AS name,
  COALESCE(NULLIF(bundle_short_version, ''), bundle_version) AS version,
  bundle_identifier AS bundle_identifier,
  '' AS extension_id,
  '' AS browser,
  'apps' AS source,
  '' AS vendor,
  last_opened_time AS last_opened_at,
  path AS installed_path
FROM apps
UNION
SELECT
  name AS name,
  version AS version,
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS browser,
  'python_packages' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM python_packages
UNION
SELECT
  name AS name,
  version AS version,
  '' AS bundle_identifier,
  identifier AS extension_id,
  browser_type AS browser,
  'chrome_extensions' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN chrome_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  '' AS bundle_identifier,
  identifier AS extension_id,
  'firefox' AS browser,
  'firefox_addons' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN firefox_addons USING (uid)
UNION
SELECT
  name As name,
  version AS version,
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS browser,
  'safari_extensions' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN safari_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS browser,
  'homebrew_packages' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM homebrew_packages
WHERE type = 'formula'
UNION
SELECT
  name AS name,
  version AS version,
  '' AS bundle_identifier,
  '' AS extension_id,
  '' AS browser,
  'homebrew_packages' AS source,
  '' AS vendor,
  0 AS last_opened_at,
  path AS installed_path
FROM homebrew_packages
WHERE type = 'cask'
AND NOT EXISTS (SELECT 1 FROM file WHERE file.path LIKE CONCAT(homebrew_packages.path, '/%%') AND file.path LIKE '/%.app%' LIMIT 1);
```

## software_macos_codesign

- Description: A software override query[^1] to append codesign information to macOS software entries. Requires `fleetd`

- Platforms: darwin

- Discovery query:
```sql
SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = 'codesign'
```

- Query:
```sql
SELECT a.path, c.team_identifier
		FROM apps a
		JOIN codesign c ON a.path = c.path
```

## software_macos_firefox

- Description: A software override query[^1] to differentiate between Firefox and Firefox ESR on macOS. Requires `fleetd`

- Platforms: darwin

- Discovery query:
```sql
SELECT 1 WHERE EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'org.mozilla.firefox' LIMIT 1) AND EXISTS (SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = 'parse_ini')
```

- Query:
```sql
WITH app_paths AS (
				SELECT path
				FROM apps
				WHERE bundle_identifier = 'org.mozilla.firefox'
			),
			remoting_name AS (
				SELECT value, path
				FROM parse_ini
				WHERE key = 'RemotingName'
				AND path IN (SELECT CONCAT(path, '/Contents/Resources/application.ini') FROM app_paths)
			)
			SELECT
				CASE
					WHEN remoting_name.value = 'firefox-esr' THEN 'Firefox ESR.app'
					ELSE 'Firefox.app'
				END AS name,
				COALESCE(NULLIF(apps.bundle_short_version, ''), apps.bundle_version) AS version,
				apps.bundle_identifier AS bundle_identifier,
				'' AS extension_id,
				'' AS browser,
				'apps' AS source,
				'' AS vendor,
				apps.last_opened_time AS last_opened_at,
				apps.path AS installed_path
			FROM apps
			LEFT JOIN remoting_name ON apps.path = REPLACE(remoting_name.path, '/Contents/Resources/application.ini', '')
			WHERE apps.bundle_identifier = 'org.mozilla.firefox'
```

## software_vscode_extensions

- Platforms: linux, ubuntu, debian, rhel, centos, sles, kali, gentoo, amzn, pop, arch, linuxmint, void, nixos, endeavouros, manjaro, opensuse-leap, opensuse-tumbleweed, tuxedo, darwin, windows

- Discovery query:
```sql
SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = 'vscode_extensions'
```

- Query:
```sql
WITH cached_users AS (WITH cached_groups AS (select * from groups)
 SELECT uid, username, type, groupname, shell
 FROM users LEFT JOIN cached_groups USING (gid)
 WHERE type <> 'special' AND shell NOT LIKE '%/false' AND shell NOT LIKE '%/nologin' AND shell NOT LIKE '%/shutdown' AND shell NOT LIKE '%/halt' AND username NOT LIKE '%$' AND username NOT LIKE '\_%' ESCAPE '\' AND NOT (username = 'sync' AND shell ='/bin/sync' AND directory <> ''))
SELECT
  name,
  version,
  '' AS bundle_identifier,
  uuid AS extension_id,
  '' AS browser,
  'vscode_extensions' AS source,
  publisher AS vendor,
  '' AS last_opened_at,
  path AS installed_path
FROM cached_users CROSS JOIN vscode_extensions USING (uid)
```

## software_windows

- Platforms: windows

- Query:
```sql
WITH cached_users AS (WITH cached_groups AS (select * from groups)
 SELECT uid, username, type, groupname, shell
 FROM users LEFT JOIN cached_groups USING (gid)
 WHERE type <> 'special' AND shell NOT LIKE '%/false' AND shell NOT LIKE '%/nologin' AND shell NOT LIKE '%/shutdown' AND shell NOT LIKE '%/halt' AND username NOT LIKE '%$' AND username NOT LIKE '\_%' ESCAPE '\' AND NOT (username = 'sync' AND shell ='/bin/sync' AND directory <> ''))
SELECT
  name AS name,
  version AS version,
  '' AS extension_id,
  '' AS browser,
  'programs' AS source,
  publisher AS vendor,
  install_location AS installed_path
FROM programs
UNION
SELECT
  name AS name,
  version AS version,
  '' AS extension_id,
  '' AS browser,
  'python_packages' AS source,
  '' AS vendor,
  path AS installed_path
FROM python_packages
UNION
SELECT
  name AS name,
  version AS version,
  '' AS extension_id,
  '' AS browser,
  'ie_extensions' AS source,
  '' AS vendor,
  path AS installed_path
FROM ie_extensions
UNION
SELECT
  name AS name,
  version AS version,
  identifier AS extension_id,
  browser_type AS browser,
  'chrome_extensions' AS source,
  '' AS vendor,
  path AS installed_path
FROM cached_users CROSS JOIN chrome_extensions USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  identifier AS extension_id,
  'firefox' AS browser,
  'firefox_addons' AS source,
  '' AS vendor,
  path AS installed_path
FROM cached_users CROSS JOIN firefox_addons USING (uid)
UNION
SELECT
  name AS name,
  version AS version,
  '' AS extension_id,
  '' AS browser,
  'chocolatey_packages' AS source,
  '' AS vendor,
  path AS installed_path
FROM chocolatey_packages
```

## system_info

- Platforms: all

- Query:
```sql
select * from system_info limit 1
```

## uptime

- Platforms: linux, ubuntu, debian, rhel, centos, sles, kali, gentoo, amzn, pop, arch, linuxmint, void, nixos, endeavouros, manjaro, opensuse-leap, opensuse-tumbleweed, tuxedo, darwin, windows

- Query:
```sql
select * from uptime limit 1
```

## users

- Platforms: linux, ubuntu, debian, rhel, centos, sles, kali, gentoo, amzn, pop, arch, linuxmint, void, nixos, endeavouros, manjaro, opensuse-leap, opensuse-tumbleweed, tuxedo, darwin, windows

- Query:
```sql
WITH cached_groups AS (select * from groups)
 SELECT uid, username, type, groupname, shell
 FROM users LEFT JOIN cached_groups USING (gid)
 WHERE type <> 'special' AND shell NOT LIKE '%/false' AND shell NOT LIKE '%/nologin' AND shell NOT LIKE '%/shutdown' AND shell NOT LIKE '%/halt' AND username NOT LIKE '%$' AND username NOT LIKE '\_%' ESCAPE '\' AND NOT (username = 'sync' AND shell ='/bin/sync' AND directory <> '')
```

## users_chrome

- Platforms: chrome

- Query:
```sql
SELECT uid, username, email FROM users
```

## windows_update_history

- Platforms: windows

- Discovery query:
```sql
SELECT 1 FROM osquery_registry WHERE active = true AND registry = 'table' AND name = 'windows_update_history'
```

- Query:
```sql
SELECT date, title FROM windows_update_history WHERE result_code = 'Succeeded'
```

<br /><br />[^1]: Software override queries write over the default queries. They are used to populate the software inventory.
<meta name="navSection" value="Dig deeper">
<meta name="pageOrderInSection" value="1600">