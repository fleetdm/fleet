package main

import (
	"fmt"
	"strings"
)

// SchemaTable represents a single osquery/fleet table schema definition
type SchemaTable struct {
	Name        string   `json:"name"`
	Platforms   []string `json:"platforms"`
	Description string   `json:"description"`
	Columns     []Column `json:"columns"`
}

// Column represents a column in a table
type Column struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// osqueryTables is the canonical, authoritative list of osquery tables supported by Fleet.
// Each table explicitly lists its supported platforms. Uses the canonical platform strings:
// "darwin" (macOS), "windows", "linux".
// Column platform notes are embedded in the description where platform-specific.
// Source: https://fleetdm.com/tables
var osqueryTables = []SchemaTable{
	// =========================================================================
	// UNIVERSAL TABLES (darwin + linux + windows)
	// =========================================================================
	{
		Name:        "os_version",
		Platforms:   []string{"darwin", "linux", "windows"},
		Description: "A single row containing the operating system name and version. Universal across all platforms.",
		Columns: []Column{
			{"name", "TEXT", "Distribution or product name"},
			{"version", "TEXT", "Pretty, suitable for logging, product version"},
			{"major", "INTEGER", "Major release version"},
			{"minor", "INTEGER", "Minor release version"},
			{"patch", "INTEGER", "Optional patch release"},
			{"build", "TEXT", "Optional build-specific or variant string"},
			{"platform", "TEXT", "OS Platform or ID (darwin, linux, windows)"},
			{"platform_like", "TEXT", "Closely related platforms (e.g. rhel, fedora)"},
			{"codename", "TEXT", "OS version codename"},
			{"arch", "TEXT", "OS Architecture"},
			{"install_date", "BIGINT", "[Windows only] Install date of the OS as UNIX time"},
			{"pid_with_namespace", "INTEGER", "[Linux only] Pids that contain a namespace"},
			{"mount_namespace_id", "TEXT", "[Linux only] Mount namespace id"},
		},
	},
	{
		Name:        "users",
		Platforms:   []string{"darwin", "linux", "windows"},
		Description: "Local system users. WARN: Columns marked [platform-only] will cause query failure on other platforms.",
		Columns: []Column{
			{"uid", "BIGINT", "User ID [ALL platforms]"},
			{"gid", "BIGINT", "Group ID (unsigned) [darwin, linux, windows]"},
			{"uid_signed", "BIGINT", "User ID as int64 signed [darwin, linux, windows]"},
			{"gid_signed", "BIGINT", "Default group ID as int64 [darwin, linux, windows]"},
			{"username", "TEXT", "Username [ALL platforms]"},
			{"description", "TEXT", "Optional user description [darwin, linux, windows]"},
			{"directory", "TEXT", "User's home directory [darwin, linux, windows]"},
			{"shell", "TEXT", "User's configured default shell [darwin, linux, windows]"},
			{"uuid", "TEXT", "User's UUID (Apple) or SID (Windows) [darwin, windows]"},
			{"type", "TEXT", "Whether the account is roaming, local, or system profile [WINDOWS ONLY]"},
			{"is_hidden", "INTEGER", "IsHidden attribute set in OpenDirectory [MACOS/DARWIN ONLY]"},
			{"pid_with_namespace", "INTEGER", "Pids that contain a namespace [LINUX ONLY]"},
			{"email", "TEXT", "LDAP email field description [CHROMEOS ONLY — do not use on other platforms]"},
		},
	},
	{
		Name:        "processes",
		Platforms:   []string{"darwin", "linux", "windows"},
		Description: "All running processes on the host system.",
		Columns: []Column{
			{"pid", "BIGINT", "Process (or thread) ID"},
			{"name", "TEXT", "The process path or shorthand command name"},
			{"path", "TEXT", "Path to executed binary"},
			{"cmdline", "TEXT", "Complete argv"},
			{"state", "TEXT", "Process state"},
			{"cwd", "TEXT", "Process current working directory"},
			{"root", "TEXT", "Process virtual root directory"},
			{"uid", "BIGINT", "User ID"},
			{"gid", "BIGINT", "Group ID"},
			{"euid", "BIGINT", "Effective user ID"},
			{"egid", "BIGINT", "Effective group ID"},
			{"suid", "BIGINT", "Saved user ID"},
			{"sgid", "BIGINT", "Saved group ID"},
			{"on_disk", "INTEGER", "The process path exists yes=1, no=0, unknown=-1"},
			{"wired_size", "BIGINT", "Bytes of unpageable memory used by process"},
			{"resident_size", "BIGINT", "Bytes of private memory used by process"},
			{"total_size", "BIGINT", "Total virtual memory size"},
			{"user_time", "BIGINT", "CPU time in milliseconds spent in user space"},
			{"system_time", "BIGINT", "CPU time in milliseconds spent in kernel space"},
			{"disk_bytes_read", "BIGINT", "Bytes read from disk"},
			{"disk_bytes_written", "BIGINT", "Bytes written to disk"},
			{"start_time", "BIGINT", "Process start time in seconds since Epoch, in case of error -1"},
			{"parent", "BIGINT", "Process parent's PID"},
			{"pgroup", "BIGINT", "Process group"},
			{"threads", "INTEGER", "Number of threads used by process"},
			{"nice", "INTEGER", "Process nice level (-20 to 20, default 0)"},
			{"elevated_token", "INTEGER", "[Windows only] Process runs with elevated token"},
			{"is_elevated_token", "INTEGER", "[darwin/macOS only] If TRUE the process is running with an elevated token"},
			{"elapsed_time", "BIGINT", "[Windows only] Elapsed time in seconds this process has been running"},
			{"handle_count", "BIGINT", "[Windows only] Total number of handles that the process has open"},
			{"percent_processor_time", "BIGINT", "[Windows only] CPU time of process in percentage"},
		},
	},
	{
		Name:        "uptime",
		Platforms:   []string{"darwin", "linux", "windows"},
		Description: "Track time passed since last boot.",
		Columns: []Column{
			{"days", "INTEGER", "Days of uptime"},
			{"hours", "INTEGER", "Hours of uptime"},
			{"minutes", "INTEGER", "Minutes of uptime"},
			{"seconds", "INTEGER", "Seconds of uptime"},
			{"total_seconds", "BIGINT", "Total uptime seconds"},
		},
	},
	{
		Name:        "system_info",
		Platforms:   []string{"darwin", "linux", "windows"},
		Description: "System information for the host. Includes model, hardware, and network information.",
		Columns: []Column{
			{"hostname", "TEXT", "Network hostname including domain"},
			{"uuid", "TEXT", "Unique ID provided by the system"},
			{"cpu_type", "TEXT", "CPU type"},
			{"cpu_subtype", "TEXT", "CPU subtype"},
			{"cpu_brand", "TEXT", "CPU brand string, try to be specific"},
			{"cpu_physical_cores", "INTEGER", "Number of physical CPU cores in to the system"},
			{"cpu_logical_cores", "INTEGER", "Number of logical CPU cores available to the system"},
			{"cpu_microcode", "TEXT", "Microcode version"},
			{"physical_memory", "BIGINT", "Total physical memory in bytes"},
			{"hardware_vendor", "TEXT", "Hardware or board vendor"},
			{"hardware_model", "TEXT", "Hardware or board model"},
			{"hardware_version", "TEXT", "Hardware or board version"},
			{"hardware_serial", "TEXT", "Device or board serial number"},
			{"board_vendor", "TEXT", "Board vendor"},
			{"board_model", "TEXT", "Board model"},
			{"board_version", "TEXT", "Board version"},
			{"board_serial", "TEXT", "Board serial number"},
			{"computer_name", "TEXT", "Friendly computer name (optional)"},
			{"local_hostname", "TEXT", "Local hostname (optional)"},
		},
	},
	{
		Name:        "logged_in_users",
		Platforms:   []string{"darwin", "linux", "windows"},
		Description: "Users with an active shell on the system.",
		Columns: []Column{
			{"type", "TEXT", "Login type including: login, kernel, init"},
			{"user", "TEXT", "User login name"},
			{"tty", "TEXT", "Device name"},
			{"host", "TEXT", "Remote hostname"},
			{"time", "BIGINT", "Time entry was made in seconds"},
			{"pid", "INTEGER", "Process (or thread) ID"},
		},
	},
	{
		Name:        "listening_ports",
		Platforms:   []string{"darwin", "linux", "windows"},
		Description: "Processes with listening (bound) network sockets/ports.",
		Columns: []Column{
			{"pid", "INTEGER", "Process (or thread) ID"},
			{"port", "INTEGER", "Transport layer port"},
			{"protocol", "INTEGER", "Transport protocol (TCP/UDP)"},
			{"family", "INTEGER", "Network protocol (AF_INET, AF_INET6, AF_UNIX)"},
			{"address", "TEXT", "Specific address for bind"},
			{"fd", "BIGINT", "Socket file descriptor number"},
			{"socket", "BIGINT", "Socket handle or underlying descriptor"},
			{"path", "TEXT", "For UNIX sockets (family=AF_UNIX), the domain path"},
			{"net_namespace", "TEXT", "[Linux only] Network namespace inode"},
		},
	},
	{
		Name:        "startup_items",
		Platforms:   []string{"darwin", "linux", "windows"},
		Description: "Applications and binaries set as user/system startup items.",
		Columns: []Column{
			{"name", "TEXT", "Name of startup item"},
			{"path", "TEXT", "Path of startup item"},
			{"args", "TEXT", "Arguments provided to startup executable"},
			{"type", "TEXT", "Startup item type: LaunchAgents, LaunchDaemons, StartupItems, etc."},
			{"source", "TEXT", "Directory or plist containing startup item"},
			{"status", "TEXT", "Startup status; e.g. 'Enabled', 'disabled'"},
			{"username", "TEXT", "The user associated with the startup item"},
		},
	},
	{
		Name:        "file",
		Platforms:   []string{"darwin", "linux", "windows"},
		Description: "Interactive filesystem attributes and metadata. Requires a path= WHERE clause.",
		Columns: []Column{
			{"path", "TEXT", "Absolute file path (required in WHERE)"},
			{"directory", "TEXT", "Directory of file(s)"},
			{"filename", "TEXT", "Name portion of file path"},
			{"inode", "BIGINT", "Filesystem inode number"},
			{"uid", "BIGINT", "Owning user ID"},
			{"gid", "BIGINT", "Owning group ID"},
			{"mode", "TEXT", "Permission bits"},
			{"device", "BIGINT", "Device ID of hosting disk volume"},
			{"size", "BIGINT", "Size of file in bytes"},
			{"block_size", "INTEGER", "Block size of filesystem"},
			{"atime", "BIGINT", "Last access time"},
			{"mtime", "BIGINT", "Last modification time"},
			{"ctime", "BIGINT", "Last status change time"},
			{"btime", "BIGINT", "File created time (birth)"},
			{"hard_links", "INTEGER", "Number of hard links"},
			{"symlink", "INTEGER", "1 if the path is a symlink, otherwise 0"},
			{"type", "TEXT", "File status"},
			{"attributes", "TEXT", "[Windows only] File attrib string. See: https://ss64.com/nt/attrib.html"},
			{"volume_serial", "TEXT", "[Windows only] Volume serial number"},
			{"file_id", "TEXT", "[Windows only] GnuWin32 friendly FILE_ID_INFO iFileId"},
			{"file_version", "TEXT", "[Windows only] File version"},
			{"product_version", "TEXT", "[Windows only] File product version"},
			{"original_filename", "TEXT", "[Windows only] Original filename for the file, stored in version resource"},
			{"bsd_flags", "TEXT", "[macOS/darwin only] BSD file flags, if set"},
			{"pid_with_namespace", "INTEGER", "[Linux only] Pids that contain a namespace"},
			{"mount_namespace_id", "TEXT", "[Linux only] Mount namespace id"},
		},
	},
	{
		Name:        "software",
		Platforms:   []string{"darwin", "linux", "windows"},
		Description: "Fleet's normalized view of installed software across all platforms. This is the preferred table for cross-platform software queries.",
		Columns: []Column{
			{"id", "INTEGER", "Unique row identifier"},
			{"name", "TEXT", "Package, app, or program name"},
			{"version", "TEXT", "Package supplied version"},
			{"source", "TEXT", "Source of the software package (e.g. deb_packages, rpm_packages, apps)"},
			{"bundle_identifier", "TEXT", "Macos package bundle identifier [darwin only]"},
			{"vendor", "TEXT", "Package vendor or manufacturer"},
			{"installed_path", "TEXT", "Path at which the application is installed"},
		},
	},
	// =========================================================================
	// WINDOWS-ONLY TABLES
	// =========================================================================
	{
		Name:        "registry",
		Platforms:   []string{"windows"},
		Description: "Windows registry entries (key-value pairs). WINDOWS ONLY. Requires path= or key= WHERE clause.",
		Columns: []Column{
			{"key", "TEXT", "Name of the key to search for"},
			{"path", "TEXT", "Full path to the value"},
			{"name", "TEXT", "Name of the registry value entry"},
			{"type", "TEXT", "Type of the value entry (REG_SZ, REG_DWORD, etc.)"},
			{"data", "TEXT", "Data content of registry value"},
			{"mtime", "BIGINT", "Timestamp of the most recent registry write"},
		},
	},
	{
		Name:        "mdm_bridge",
		Platforms:   []string{"windows"},
		Description: "Query any WMI class or CSP using the Fleet MDM Bridge. WINDOWS ONLY. Supports Intune-style CSP path queries. Requires mdm_command_input.",
		Columns: []Column{
			{"mdm_command_input", "TEXT", "Full SyncML XML input to send to the MDM bridge (required in WHERE)"},
			{"mdm_command_output", "TEXT", "SyncML XML response from the MDM bridge"},
		},
	},
	{
		Name:        "windows_update_history",
		Platforms:   []string{"windows"},
		Description: "Historical records of Windows update installations. WINDOWS ONLY.",
		Columns: []Column{
			{"client_application_id", "TEXT", "The application that triggered the update"},
			{"date", "BIGINT", "Date and time that the installation was performed as UNIX time"},
			{"description", "TEXT", "Description of the update"},
			{"hresult", "BIGINT", "The HRESULT code for the update"},
			{"operation", "INTEGER", "The operation type of the update (1=Install, 2=Uninstall)"},
			{"result_code", "INTEGER", "The result code (0=NotStarted, 1=InProgress, 2=Succeeded, 3=SucceededWithErrors, 4=Failed, 5=Aborted)"},
			{"server_selection", "INTEGER", "Which update server was selected for this update"},
			{"service_id", "TEXT", "The identifier of the Windows Update service"},
			{"support_url", "TEXT", "URL for support information"},
			{"title", "TEXT", "Title of the update"},
			{"update_id", "TEXT", "Unique identifier for the update"},
			{"update_revision", "BIGINT", "Revision number of the update"},
		},
	},
	{
		Name:        "windows_security_center",
		Platforms:   []string{"windows"},
		Description: "The health status of all Security Center components including Antivirus, Firewall, and updates. WINDOWS ONLY.",
		Columns: []Column{
			{"firewall", "TEXT", "The health of the monitored Firewall solution (Good, Poor, Snoozed, Not Monitored, Error)"},
			{"autoupdate", "TEXT", "The health of the auto-update software (Good, Poor, Snoozed, Not Monitored, Error)"},
			{"antivirus", "TEXT", "The health of the monitored Antivirus solution (Good, Poor, Snoozed, Not Monitored, Error)"},
			{"antispyware", "TEXT", "The health of the monitored Antispyware solution (Good, Poor, Snoozed, Not Monitored, Error)"},
			{"internet_settings", "TEXT", "The health of the Internet Settings (Good, Poor, Snoozed, Not Monitored, Error)"},
			{"windows_security_center_service", "TEXT", "The health of the Windows Security Center Service (Good, Poor, Snoozed, Not Monitored, Error)"},
			{"user_account_control", "TEXT", "The health of the User Account Control (Good, Poor, Snoozed, Not Monitored, Error)"},
		},
	},
	{
		Name:        "bitlocker_info",
		Platforms:   []string{"windows"},
		Description: "Retrieves information about BitLocker drive encryption status on Windows. WINDOWS ONLY.",
		Columns: []Column{
			{"device_id", "TEXT", "ID of the encrypted drive"},
			{"drive_letter", "TEXT", "Drive letter of the encrypted drive"},
			{"persistent_volume_id", "TEXT", "Persistent ID of the encrypted drive"},
			{"conversion_status", "INTEGER", "The BitLocker conversion status: 0=Fully Decrypted, 1=Fully Encrypted, 2=Encryption in Progress, 3=Decryption in Progress, 4=Encryption Paused, 5=Decryption Paused"},
			{"protection_status", "INTEGER", "The BitLocker protection status: 0=Protection Off, 1=Protection On, 2=Protection Unknown"},
			{"encryption_method", "TEXT", "Encryption method used"},
		},
	},
	{
		Name:        "programs",
		Platforms:   []string{"windows"},
		Description: "Represents products as they are installed by Windows Installer. WINDOWS ONLY.",
		Columns: []Column{
			{"name", "TEXT", "Commonly used product name"},
			{"version", "TEXT", "Product version information"},
			{"install_location", "TEXT", "The installation location directory of the product"},
			{"install_source", "TEXT", "The installation source of the product"},
			{"language", "TEXT", "The decimal product language identifier"},
			{"publisher", "TEXT", "Name of the product supplier"},
			{"uninstall_string", "TEXT", "Path and filename of the uninstall program"},
			{"install_date", "TEXT", "Date the product was installed in ISO 8601 format"},
			{"identifying_number", "TEXT", "Product identification such as a serial number on software"},
		},
	},
	{
		Name:        "services",
		Platforms:   []string{"windows"},
		Description: "Lists all installed Windows services and their properties. WINDOWS ONLY.",
		Columns: []Column{
			{"name", "TEXT", "Service name"},
			{"service_type", "TEXT", "Service Type: OWN_PROCESS, SHARE_PROCESS and maybe Interactive (e.g. for Windows services)"},
			{"display_name", "TEXT", "Service Display name"},
			{"status", "TEXT", "Service Current status"},
			{"pid", "INTEGER", "the Process ID of the service"},
			{"start_type", "TEXT", "Service start type: BOOT_START, SYSTEM_START, AUTO_START, DEMAND_START, DISABLED"},
			{"win32_exit_code", "INTEGER", "The error code that the service uses to report an error that occurs when it is starting or stopping"},
			{"service_exit_code", "INTEGER", "The service-specific error code that the service returns when an error occurs while the service is starting or stopping"},
			{"path", "TEXT", "Path to Service Executable"},
			{"module_path", "TEXT", "Path to ServiceDll"},
			{"description", "TEXT", "Service Description"},
			{"user_account", "TEXT", "The name of the account that the service process will be logged on as when it runs"},
		},
	},
	{
		Name:        "windows_firewall_rules",
		Platforms:   []string{"windows"},
		Description: "Provides the active firewall rules on a Windows machine. WINDOWS ONLY.",
		Columns: []Column{
			{"name", "TEXT", "Friendly name of the rule"},
			{"app_name", "TEXT", "Friendly name of the application to which the rule applies"},
			{"action", "TEXT", "Action for the rule or group (Allow or Block)"},
			{"enabled", "INTEGER", "1 if the rule is enabled, 0 otherwise"},
			{"grouping", "TEXT", "The group to which an individual rule belongs"},
			{"direction", "TEXT", "The direction the rule is applied to (IN, OUT, BOTH)"},
			{"protocol", "INTEGER", "IP Protocol ID (0=HOPOPT, 1=ICMP, 6=TCP, 17=UDP, etc.)"},
			{"local_addresses", "TEXT", "Local addresses for the rule"},
			{"remote_addresses", "TEXT", "Remote addresses for the rule"},
			{"local_ports", "TEXT", "Local ports for the rule"},
			{"remote_ports", "TEXT", "Remote ports for the rule"},
			{"icmp_types_codes", "TEXT", "ICMP types and codes for the rule"},
			{"profile_domain", "INTEGER", "1 if rule is set for domain profile, 0 if not"},
			{"profile_private", "INTEGER", "1 if rule is set for private profile, 0 if not"},
			{"profile_public", "INTEGER", "1 if rule is set for public profile, 0 if not"},
		},
	},
	// =========================================================================
	// MACOS (DARWIN) ONLY TABLES
	// =========================================================================
	{
		Name:        "managed_policies",
		Platforms:   []string{"darwin"},
		Description: "The values in a managed configuration profile (MDM or MCX). MACOS ONLY.",
		Columns: []Column{
			{"domain", "TEXT", "SMB/LDAP-style domain that the policy corresponds to (e.g. com.apple.Safari)"},
			{"uuid", "TEXT", "Optional UUID assigned to the policy instance"},
			{"name", "TEXT", "Policy key name"},
			{"value", "TEXT", "Policy value"},
			{"username", "TEXT", "Policy applies to this username if set"},
			{"manual", "INTEGER", "1 if the policy was set manually, otherwise 0"},
		},
	},
	{
		Name:        "sip_config",
		Platforms:   []string{"darwin"},
		Description: "Apple's macOS System Integrity Protection (SIP) status. MACOS ONLY.",
		Columns: []Column{
			{"config_flag", "TEXT", "The SIP configuration flag"},
			{"enabled", "INTEGER", "1 if the flag is enabled"},
			{"enabled_nvram", "INTEGER", "1 if the flag is set in NVRAM"},
		},
	},
	{
		Name:        "gatekeeper",
		Platforms:   []string{"darwin"},
		Description: "macOS Gatekeeper status and settings. MACOS ONLY.",
		Columns: []Column{
			{"assessments_enabled", "INTEGER", "1 if a Gatekeeper is enabled, otherwise 0"},
			{"dev_id_enabled", "INTEGER", "1 if Developer ID checking is enabled, otherwise 0"},
			{"version", "TEXT", "Version of Gatekeeper's gke.bundle"},
			{"opaque_version", "TEXT", "Version of Gatekeeper's gkopaque.bundle"},
		},
	},
	{
		Name:        "software_update",
		Platforms:   []string{"darwin"},
		Description: "Available Apple Software Updates. MACOS ONLY.",
		Columns: []Column{
			{"label", "TEXT", "Software Update label"},
			{"title", "TEXT", "Software Update title"},
			{"version", "TEXT", "Software Update version"},
			{"size", "TEXT", "Size of the update in bytes"},
			{"recommended", "TEXT", "Software Update recommended"},
			{"restart_required", "TEXT", "Software Update restart required"},
			{"allow_configuration", "TEXT", "Software Update allow configuration"},
		},
	},
	{
		Name:        "apps",
		Platforms:   []string{"darwin"},
		Description: "Mac OS X applications installed in known search paths. MACOS ONLY.",
		Columns: []Column{
			{"name", "TEXT", "Name of the application"},
			{"path", "TEXT", "Absolute (usually .app) path"},
			{"bundle_executable", "TEXT", "Info.plist CFBundleExecutable value"},
			{"bundle_identifier", "TEXT", "Info.plist CFBundleIdentifier value"},
			{"bundle_name", "TEXT", "Info.plist CFBundleName value"},
			{"bundle_short_version", "TEXT", "Info.plist CFBundleShortVersionString value"},
			{"bundle_version", "TEXT", "Info.plist CFBundleVersion value"},
			{"bundle_package_type", "TEXT", "Info.plist CFBundlePackageType value"},
			{"environment", "TEXT", "Application-set environment variables"},
			{"element", "TEXT", "Does the app identify as a background agent"},
			{"compiler", "TEXT", "Info.plist DTCompiler value"},
			{"development_region", "TEXT", "Info.plist CFBundleDevelopmentRegion value"},
			{"display_name", "TEXT", "Info.plist CFBundleDisplayName value"},
			{"info_string", "TEXT", "Info.plist CFBundleGetInfoString value"},
			{"minimum_system_version", "TEXT", "Minimum version of macOS required"},
			{"category", "TEXT", "The UTI that categorizes the app for the App Store"},
			{"applescript_enabled", "TEXT", "1 if the application supports Applescript"},
			{"copyright", "TEXT", "Info.plist NSHumanReadableCopyright value"},
			{"last_opened_time", "REAL", "The time that the app was last used [REAL/FLOAT]"},
		},
	},
	{
		Name:        "plist",
		Platforms:   []string{"darwin"},
		Description: "Read and parse a plist file. MACOS ONLY. Requires path= WHERE clause.",
		Columns: []Column{
			{"key", "TEXT", "Preference top-level key"},
			{"subkey", "TEXT", "Intermediate key path, delimiter='/'"},
			{"value", "TEXT", "String value of most CF types"},
			{"path", "TEXT", "Path to the plist (required in WHERE)"},
		},
	},
	{
		Name:        "screensaver",
		Platforms:   []string{"darwin"},
		Description: "Information about current screensaver settings on macOS. MACOS ONLY.",
		Columns: []Column{
			{"name", "TEXT", "Screensaver name"},
			{"module_path", "TEXT", "Relative path to the screensaver module"},
			{"path", "TEXT", "Path to the screensaver module"},
			{"enabled", "INTEGER", "1 if screensaver is enabled"},
			{"login_window_idle_time", "BIGINT", "Time before login window screensaver activates"},
			{"ask_for_password", "INTEGER", "1 if password is required to wake from screensaver"},
			{"ask_for_password_delay", "BIGINT", "Delay in seconds before password is required"},
		},
	},
	{
		Name:        "secureboot",
		Platforms:   []string{"darwin", "linux", "windows"},
		Description: "Computer Secure Boot and configuration.",
		Columns: []Column{
			{"secure_boot", "INTEGER", "1 if Secure Boot is enabled, 0 otherwise"},
			{"setup_mode", "INTEGER", "1 if driver signing is not required, 0 if not or unknown"},
		},
	},
	{
		Name:        "filevault_status",
		Platforms:   []string{"darwin"},
		Description: "macOS FileVault status. Use this to check disk encryption on macOS. MACOS ONLY.",
		Columns: []Column{
			{"uid", "BIGINT", "User ID"},
			{"name", "TEXT", "User name"},
			{"uuid", "TEXT", "User UUID generated by macOS"},
			{"filevault_status", "TEXT", "FileVault status: on, off, active"},
		},
	},
	// =========================================================================
	// LINUX-FOCUSED TABLES
	// =========================================================================
	{
		Name:        "mounts",
		Platforms:   []string{"linux", "darwin"},
		Description: "Mounted filesystem information. Primarily used for Linux compliance checks.",
		Columns: []Column{
			{"device", "TEXT", "Mounted device"},
			{"device_alias", "TEXT", "Mounted device alias"},
			{"path", "TEXT", "Mounted device path"},
			{"type", "TEXT", "Mounted device type"},
			{"blocks_size", "BIGINT", "Block size in bytes"},
			{"blocks", "BIGINT", "Mounted device used blocks"},
			{"blocks_free", "BIGINT", "Mounted device free blocks"},
			{"blocks_available", "BIGINT", "Mounted device available blocks"},
			{"inodes", "BIGINT", "Mounted device used inodes"},
			{"inodes_free", "BIGINT", "Mounted device free inodes"},
			{"flags", "TEXT", "Mounted device flags"},
		},
	},
	{
		Name:        "deb_packages",
		Platforms:   []string{"linux"},
		Description: "The installed DEB package database. LINUX ONLY (Debian/Ubuntu).",
		Columns: []Column{
			{"name", "TEXT", "Package name"},
			{"version", "TEXT", "Package version"},
			{"source", "TEXT", "Package source"},
			{"size", "BIGINT", "Package size in bytes"},
			{"arch", "TEXT", "Package architecture"},
			{"revision", "TEXT", "Package revision"},
			{"status", "TEXT", "Package installation status"},
			{"maintainer", "TEXT", "Package maintainer"},
			{"section", "TEXT", "Package section"},
			{"priority", "TEXT", "Package priority"},
			{"admindir", "TEXT", "Directory to search for package database"},
			{"pid_with_namespace", "INTEGER", "Pids that contain a namespace"},
			{"mount_namespace_id", "TEXT", "Mount namespace id"},
		},
	},
	{
		Name:        "rpm_packages",
		Platforms:   []string{"linux"},
		Description: "RPM packages that are currently installed on the host system. LINUX ONLY (RHEL/CentOS/Fedora/Amazon Linux).",
		Columns: []Column{
			{"name", "TEXT", "RPM package name"},
			{"version", "TEXT", "RPM package version"},
			{"release", "TEXT", "Package release"},
			{"source", "TEXT", "Source RPM used to build this package"},
			{"size", "BIGINT", "Package size in bytes"},
			{"sha1", "TEXT", "SHA1 hash of the package contents"},
			{"arch", "TEXT", "Architecture(s) supported"},
			{"epoch", "INTEGER", "Package epoch value"},
			{"install_time", "BIGINT", "When the package was installed"},
			{"vendor", "TEXT", "Package vendor"},
			{"package_group", "TEXT", "Package group"},
			{"pid_with_namespace", "INTEGER", "Pids that contain a namespace"},
			{"mount_namespace_id", "TEXT", "Mount namespace id"},
		},
	},
	{
		Name:        "systemd_units",
		Platforms:   []string{"linux"},
		Description: "Track all systemd units. LINUX ONLY.",
		Columns: []Column{
			{"id", "TEXT", "Unique name of the systemd unit"},
			{"description", "TEXT", "Unit description"},
			{"load_state", "TEXT", "Reflects whether the unit's configuration was loaded successfully"},
			{"active_state", "TEXT", "The high-level unit activation state"},
			{"sub_state", "TEXT", "The low-level unit activation state (depends on unit type)"},
			{"following", "TEXT", "Following another unit"},
			{"object_path", "TEXT", "The object path for this unit"},
			{"job_id", "BIGINT", "Next queued job id"},
			{"job_type", "TEXT", "Job type"},
			{"job_object_path", "TEXT", "The object path for the job"},
			{"fragment_path", "TEXT", "The unit file path this fragment was read from, if applicable"},
			{"user", "TEXT", "The user as which the service runs"},
			{"source_path", "TEXT", "Path to the (possibly generated) file this was created from"},
		},
	},
	{
		Name:        "crontab",
		Platforms:   []string{"linux", "darwin"},
		Description: "Line parsed values from system and user cron/tab.",
		Columns: []Column{
			{"event", "TEXT", "The cron child event"},
			{"minute", "TEXT", "The exact minute for the job"},
			{"hour", "TEXT", "The hour of the day for the job"},
			{"day_of_month", "TEXT", "The day of the month for the job"},
			{"month", "TEXT", "The month of the year for the job"},
			{"day_of_week", "TEXT", "The day of the week for the job"},
			{"command", "TEXT", "Raw command string"},
			{"path", "TEXT", "File parsed"},
			{"pid_with_namespace", "INTEGER", "[Linux only] Pids that contain a namespace"},
			{"mount_namespace_id", "TEXT", "[Linux only] Mount namespace id"},
		},
	},
}

// GetOsquerySchema returns the schema for osquery tables filtered by platform.
// Platform values: "darwin" or "macos", "windows", "linux", "all" (returns everything).
func GetOsquerySchema(platform string) ([]SchemaTable, error) {
	var filtered []SchemaTable
	p := strings.ToLower(strings.TrimSpace(platform))

	// Normalize common aliases
	if p == "macos" || p == "mac" || p == "osx" {
		p = "darwin"
	}

	for _, t := range osqueryTables {
		if p == "" || p == "all" {
			filtered = append(filtered, t)
			continue
		}
		for _, tp := range t.Platforms {
			if tp == p {
				filtered = append(filtered, t)
				break
			}
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("no tables found for platform: %s (supported: darwin/macos, windows, linux, all)", platform)
	}

	return filtered, nil
}

// IsTableSupportedOnPlatform checks whether a given table name is valid for a platform.
func IsTableSupportedOnPlatform(tableName, platform string) bool {
	p := strings.ToLower(strings.TrimSpace(platform))
	if p == "macos" || p == "mac" || p == "osx" {
		p = "darwin"
	}
	for _, t := range osqueryTables {
		if strings.EqualFold(t.Name, tableName) {
			for _, tp := range t.Platforms {
				if tp == p {
					return true
				}
			}
			return false // Table exists, but not for this platform
		}
	}
	return true // Table not in our known list — don't block on unknowns, let Fleet handle it
}

// ValidateSQLForPlatforms performs a best-effort pre-flight check on a SQL query.
// It extracts table references (FROM <table>) and checks them against the provided platforms.
// Returns an error if a platform-incompatible table is detected.
func ValidateSQLForPlatforms(sql string, platforms []string) error {
	if len(platforms) == 0 {
		return nil // No platform context, skip validation
	}

	// Normalize all platforms
	normalizedPlatforms := make([]string, len(platforms))
	for i, p := range platforms {
		np := strings.ToLower(strings.TrimSpace(p))
		if np == "macos" || np == "mac" || np == "osx" {
			np = "darwin"
		}
		normalizedPlatforms[i] = np
	}

	// Extract table names from SQL using simple token-based parsing.
	// Looks for: FROM <table>, JOIN <table>
	sqlUpper := strings.ToUpper(sql)
	tokens := strings.Fields(sqlUpper)
	var usedTables []string
	originalTokens := strings.Fields(sql)
	for i, tok := range tokens {
		if (tok == "FROM" || tok == "JOIN") && i+1 < len(originalTokens) {
			// Get the table name in original casing
			candidate := strings.TrimRight(originalTokens[i+1], ";,)")
			candidate = strings.TrimLeft(candidate, "(")
			if candidate != "" && candidate != "(" {
				usedTables = append(usedTables, candidate)
			}
		}
	}

	for _, table := range usedTables {
		for _, platform := range normalizedPlatforms {
			if !IsTableSupportedOnPlatform(table, platform) {
				return fmt.Errorf(
					"SQL validation error: table '%s' is not supported on platform '%s'. "+
						"Use get_osquery_schema with platform='%s' to see supported tables for that platform",
					table, platform, platform,
				)
			}
		}
	}

	return nil
}
