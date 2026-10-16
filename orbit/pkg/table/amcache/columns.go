package amcache

import "github.com/osquery/osquery-go/plugin/table"

// kind selects a column's decoder. Columns name their decoder explicitly rather than having it
// inferred from the value's shape: ProgramId values look exactly like a 0000-prefixed SHA-1
// but are not file hashes, so shape detection produces a wrong sha1 on every row.
type kind uint8

const (
	kText  kind = iota // string as rendered by the parser
	kInt               // unsigned decimal
	kBool              // "1"/"0" (or "true"/"false") to 1/0
	kTime              // date string in one of the observed layouts, to unix seconds
	kEpoch             // already unix seconds
	kMulti             // comma-separated list, re-joined after trimming
	kSha1              // the 40 hex characters after the 0000 prefix of FileId/DriverId
	kLower             // value lowercased

	// The key-derived kinds read the subkey rather than a value, so src is empty.
	kKey      // subkey name verbatim
	kKeyPath  // subkey name with '/' rewritten to '\'
	kKeyLower // subkey name lowercased
	kKeyTime  // subkey last write time
)

// col is one osquery column. src is the registry value name. push marks a column that
// equality constraints are answered from an index for.
type col struct {
	name string
	typ  table.ColumnType
	src  string
	kind kind
	push bool
}

type spec struct {
	table string
	key   string
	cols  []col
}

// Integer columns are rendered through sqlite3_result_int, which is 32-bit signed, so anything
// that can reach 2^31 (flag words, sizes, PE timestamps) is bigint.
var specs = []spec{
	{
		table: "amcache_application_files",
		key:   keyApplicationFile,
		cols: []col{
			{name: "path", typ: table.ColumnTypeText, src: "LowerCaseLongPath", kind: kText, push: true},
			// The subkey's "<name>|<hash>" prefix differs from the Name value on many Windows 11
			// records, so the subkey is not used for this.
			{name: "name", typ: table.ColumnTypeText, src: "Name", kind: kText},
			{name: "sha1", typ: table.ColumnTypeText, src: "FileId", kind: kSha1, push: true},
			{name: "program_id", typ: table.ColumnTypeText, src: "ProgramId", kind: kText, push: true},
			{name: "publisher", typ: table.ColumnTypeText, src: "Publisher", kind: kText},
			{name: "original_file_name", typ: table.ColumnTypeText, src: "OriginalFileName", kind: kText},
			{name: "version", typ: table.ColumnTypeText, src: "Version", kind: kText},
			{name: "bin_file_version", typ: table.ColumnTypeText, src: "BinFileVersion", kind: kText},
			{name: "product_name", typ: table.ColumnTypeText, src: "ProductName", kind: kText},
			{name: "product_version", typ: table.ColumnTypeText, src: "ProductVersion", kind: kText},
			{name: "bin_product_version", typ: table.ColumnTypeText, src: "BinProductVersion", kind: kText},
			{name: "binary_type", typ: table.ColumnTypeText, src: "BinaryType", kind: kText},
			{name: "size", typ: table.ColumnTypeBigInt, src: "Size", kind: kInt},
			{name: "link_time", typ: table.ColumnTypeBigInt, src: "LinkDate", kind: kTime},
			{name: "language", typ: table.ColumnTypeInteger, src: "Language", kind: kInt},
			{name: "os_component", typ: table.ColumnTypeInteger, src: "IsOsComponent", kind: kBool},
			{name: "usn", typ: table.ColumnTypeBigInt, src: "Usn", kind: kInt},
			{name: "long_path_hash", typ: table.ColumnTypeText, src: "LongPathHash", kind: kText},
			{name: "appx_package_full_name", typ: table.ColumnTypeText, src: "AppxPackageFullName", kind: kText},
			{name: "last_write_time", typ: table.ColumnTypeBigInt, kind: kKeyTime},
		},
	},
	{
		table: "amcache_applications",
		key:   keyApplication,
		cols: []col{
			{name: "program_id", typ: table.ColumnTypeText, kind: kKey, push: true},
			{name: "name", typ: table.ColumnTypeText, src: "Name", kind: kText},
			{name: "version", typ: table.ColumnTypeText, src: "Version", kind: kText},
			{name: "publisher", typ: table.ColumnTypeText, src: "Publisher", kind: kText},
			{name: "type", typ: table.ColumnTypeText, src: "Type", kind: kText},
			{name: "source", typ: table.ColumnTypeText, src: "Source", kind: kText},
			{name: "store_app_type", typ: table.ColumnTypeText, src: "StoreAppType", kind: kText},
			{name: "install_time", typ: table.ColumnTypeBigInt, src: "InstallDate", kind: kTime},
			{name: "msi_install_time", typ: table.ColumnTypeBigInt, src: "MsiInstallDate", kind: kTime},
			{name: "root_dir_path", typ: table.ColumnTypeText, src: "RootDirPath", kind: kText},
			{name: "uninstall_string", typ: table.ColumnTypeText, src: "UninstallString", kind: kText},
			{name: "registry_key_path", typ: table.ColumnTypeText, src: "RegistryKeyPath", kind: kText},
			{name: "msi_product_code", typ: table.ColumnTypeText, src: "MsiProductCode", kind: kText},
			{name: "msi_package_code", typ: table.ColumnTypeText, src: "MsiPackageCode", kind: kText},
			{name: "package_full_name", typ: table.ColumnTypeText, src: "PackageFullName", kind: kText},
			{name: "manifest_path", typ: table.ColumnTypeText, src: "ManifestPath", kind: kText},
			{name: "hidden_arp", typ: table.ColumnTypeInteger, src: "HiddenArp", kind: kBool},
			{name: "inbox_modern_app", typ: table.ColumnTypeInteger, src: "InboxModernApp", kind: kBool},
			{name: "language", typ: table.ColumnTypeInteger, src: "Language", kind: kInt},
			// A Windows version quad, not a date.
			{name: "os_version_at_install", typ: table.ColumnTypeText, src: "OSVersionAtInstallTime", kind: kText},
			{name: "program_instance_id", typ: table.ColumnTypeText, src: "ProgramInstanceId", kind: kText},
			{name: "user_sid", typ: table.ColumnTypeText, src: "UserSid", kind: kText},
			{name: "last_write_time", typ: table.ColumnTypeBigInt, kind: kKeyTime},
		},
	},
	{
		table: "amcache_application_shortcuts",
		key:   keyApplicationShortcut,
		cols: []col{
			{name: "path", typ: table.ColumnTypeText, src: "ShortcutPath", kind: kText, push: true},
			{name: "target_path", typ: table.ColumnTypeText, src: "ShortcutTargetPath", kind: kText},
			{name: "aumid", typ: table.ColumnTypeText, src: "ShortcutAumid", kind: kText},
			{name: "program_id", typ: table.ColumnTypeText, src: "ShortcutProgramId", kind: kText, push: true},
			{name: "last_write_time", typ: table.ColumnTypeBigInt, kind: kKeyTime},
		},
	},
	{
		table: "amcache_driver_binaries",
		key:   keyDriverBinary,
		cols: []col{
			// The subkey is the lowercase path written with '/' instead of '\'.
			{name: "path", typ: table.ColumnTypeText, kind: kKeyPath, push: true},
			{name: "driver_name", typ: table.ColumnTypeText, src: "DriverName", kind: kText},
			{name: "sha1", typ: table.ColumnTypeText, src: "DriverId", kind: kSha1, push: true},
			{name: "signed", typ: table.ColumnTypeInteger, src: "DriverSigned", kind: kBool},
			{name: "inbox", typ: table.ColumnTypeInteger, src: "DriverInBox", kind: kBool},
			{name: "kernel_mode", typ: table.ColumnTypeInteger, src: "DriverIsKernelMode", kind: kBool},
			{name: "driver_type", typ: table.ColumnTypeBigInt, src: "DriverType", kind: kInt},
			{name: "driver_version", typ: table.ColumnTypeText, src: "DriverVersion", kind: kText},
			{name: "product", typ: table.ColumnTypeText, src: "Product", kind: kText},
			{name: "product_version", typ: table.ColumnTypeText, src: "ProductVersion", kind: kText},
			{name: "driver_company", typ: table.ColumnTypeText, src: "DriverCompany", kind: kText},
			{name: "service", typ: table.ColumnTypeText, src: "Service", kind: kText},
			{name: "inf", typ: table.ColumnTypeText, src: "Inf", kind: kText, push: true},
			{name: "driver_package_strong_name", typ: table.ColumnTypeText, src: "DriverPackageStrongName", kind: kText},
			{name: "image_size", typ: table.ColumnTypeBigInt, src: "ImageSize", kind: kInt},
			{name: "checksum", typ: table.ColumnTypeBigInt, src: "DriverCheckSum", kind: kInt},
			// Already unix seconds; real inbox drivers have values past 2^31 (and past 2100), so it
			// skips the date parsing and plausibility check.
			{name: "link_time", typ: table.ColumnTypeBigInt, src: "DriverTimeStamp", kind: kEpoch},
			{name: "modified_time", typ: table.ColumnTypeBigInt, src: "DriverLastWriteTime", kind: kTime},
			{name: "last_write_time", typ: table.ColumnTypeBigInt, kind: kKeyTime},
		},
	},
	{
		table: "amcache_driver_packages",
		key:   keyDriverPackage,
		cols: []col{
			{name: "inf", typ: table.ColumnTypeText, src: "Inf", kind: kText, push: true},
			{name: "directory", typ: table.ColumnTypeText, src: "Directory", kind: kText},
			{name: "sys_file", typ: table.ColumnTypeText, src: "SYSFILE", kind: kText},
			{name: "class", typ: table.ColumnTypeText, src: "Class", kind: kText},
			{name: "class_guid", typ: table.ColumnTypeText, src: "ClassGuid", kind: kText},
			{name: "provider", typ: table.ColumnTypeText, src: "Provider", kind: kText},
			{name: "version", typ: table.ColumnTypeText, src: "Version", kind: kText},
			{name: "driver_ver_time", typ: table.ColumnTypeBigInt, src: "Date", kind: kTime},
			{name: "inbox", typ: table.ColumnTypeInteger, src: "DriverInBox", kind: kBool},
			{name: "submission_id", typ: table.ColumnTypeText, src: "SubmissionId", kind: kText},
			{name: "hwids", typ: table.ColumnTypeText, src: "Hwids", kind: kMulti},
			{name: "last_write_time", typ: table.ColumnTypeBigInt, kind: kKeyTime},
		},
	},
	{
		table: "amcache_device_pnp",
		key:   keyDevicePnp,
		cols: []col{
			// The '/' to '\' rewrite is what makes parent_id join against this column. One subkey
			// per hive is a braced GUID, so the rewrite must not validate path shape.
			{name: "device_instance_id", typ: table.ColumnTypeText, kind: kKeyPath, push: true},
			{name: "description", typ: table.ColumnTypeText, src: "Description", kind: kText},
			{name: "bus_reported_description", typ: table.ColumnTypeText, src: "BusReportedDescription", kind: kText},
			{name: "class", typ: table.ColumnTypeText, src: "Class", kind: kText},
			{name: "class_guid", typ: table.ColumnTypeText, src: "ClassGuid", kind: kText},
			{name: "enumerator", typ: table.ColumnTypeText, src: "Enumerator", kind: kText},
			{name: "manufacturer", typ: table.ColumnTypeText, src: "Manufacturer", kind: kText},
			{name: "model", typ: table.ColumnTypeText, src: "Model", kind: kText},
			{name: "provider", typ: table.ColumnTypeText, src: "Provider", kind: kText},
			{name: "service", typ: table.ColumnTypeText, src: "Service", kind: kText},
			{name: "driver_name", typ: table.ColumnTypeText, src: "DriverName", kind: kText},
			{name: "sha1", typ: table.ColumnTypeText, src: "DriverId", kind: kSha1, push: true},
			{name: "driver_ver_version", typ: table.ColumnTypeText, src: "DriverVerVersion", kind: kText},
			{name: "driver_ver_time", typ: table.ColumnTypeBigInt, src: "DriverVerDate", kind: kTime},
			{name: "driver_package_strong_name", typ: table.ColumnTypeText, src: "DriverPackageStrongName", kind: kText},
			{name: "inf", typ: table.ColumnTypeText, src: "Inf", kind: kText},
			{name: "hwids", typ: table.ColumnTypeText, src: "HWID", kind: kMulti},
			{name: "compids", typ: table.ColumnTypeText, src: "COMPID", kind: kMulti},
			{name: "matching_id", typ: table.ColumnTypeText, src: "MatchingID", kind: kText},
			{name: "parent_id", typ: table.ColumnTypeText, src: "ParentId", kind: kText},
			// Lowercased to match amcache_device_containers.container_id, which comes from the
			// subkey name.
			{name: "container_id", typ: table.ColumnTypeText, src: "ContainerId", kind: kLower, push: true},
			{name: "install_state", typ: table.ColumnTypeInteger, src: "InstallState", kind: kInt},
			// DN_* status flags; DN_NEEDS_LOCKING is 0x80000000.
			{name: "device_state", typ: table.ColumnTypeBigInt, src: "DeviceState", kind: kInt},
			{name: "problem_code", typ: table.ColumnTypeInteger, src: "ProblemCode", kind: kInt},
			{name: "install_time", typ: table.ColumnTypeBigInt, src: "InstallDate", kind: kTime},
			{name: "first_install_time", typ: table.ColumnTypeBigInt, src: "FirstInstallDate", kind: kTime},
			{name: "lower_filters", typ: table.ColumnTypeText, src: "LowerFilters", kind: kMulti},
			{name: "upper_filters", typ: table.ColumnTypeText, src: "UpperFilters", kind: kMulti},
			{name: "last_write_time", typ: table.ColumnTypeBigInt, kind: kKeyTime},
		},
	},
	{
		table: "amcache_device_containers",
		key:   keyDeviceContainer,
		cols: []col{
			{name: "container_id", typ: table.ColumnTypeText, kind: kKeyLower, push: true},
			{name: "friendly_name", typ: table.ColumnTypeText, src: "FriendlyName", kind: kText},
			{name: "manufacturer", typ: table.ColumnTypeText, src: "Manufacturer", kind: kText},
			{name: "model_name", typ: table.ColumnTypeText, src: "ModelName", kind: kText},
			{name: "model_number", typ: table.ColumnTypeText, src: "ModelNumber", kind: kText},
			{name: "model_id", typ: table.ColumnTypeText, src: "ModelId", kind: kText},
			{name: "primary_category", typ: table.ColumnTypeText, src: "PrimaryCategory", kind: kText},
			{name: "categories", typ: table.ColumnTypeText, src: "Categories", kind: kMulti},
			{name: "discovery_method", typ: table.ColumnTypeText, src: "DiscoveryMethod", kind: kText},
			{name: "state", typ: table.ColumnTypeBigInt, src: "State", kind: kInt},
			{name: "active", typ: table.ColumnTypeInteger, src: "IsActive", kind: kBool},
			{name: "connected", typ: table.ColumnTypeInteger, src: "IsConnected", kind: kBool},
			{name: "paired", typ: table.ColumnTypeInteger, src: "IsPaired", kind: kBool},
			{name: "networked", typ: table.ColumnTypeInteger, src: "IsNetworked", kind: kBool},
			{name: "machine_container", typ: table.ColumnTypeInteger, src: "IsMachineContainer", kind: kBool},
			{name: "last_write_time", typ: table.ColumnTypeBigInt, kind: kKeyTime},
		},
	},
}

func (s spec) columnDefs() []table.ColumnDefinition {
	defs := make([]table.ColumnDefinition, 0, len(s.cols))
	for _, c := range s.cols {
		switch c.typ {
		case table.ColumnTypeInteger:
			defs = append(defs, table.IntegerColumn(c.name))
		case table.ColumnTypeBigInt:
			defs = append(defs, table.BigIntColumn(c.name))
		default:
			defs = append(defs, table.TextColumn(c.name))
		}
	}
	return defs
}
