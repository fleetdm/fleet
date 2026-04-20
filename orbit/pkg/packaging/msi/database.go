package msi

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// MSIOptions contains the parameters needed to build an MSI installer.
type MSIOptions struct {
	// Product metadata.
	ProductName    string // "Fleet osquery"
	ProductVersion string // e.g. "1.28.0"
	Manufacturer   string // "Fleet Device Management (fleetdm.com)"
	UpgradeCode    string // Fixed GUID: {B681CB20-107E-428A-9B14-2D3C1AFED244}
	Architecture   string // "amd64" or "arm64"

	// Fleet configuration.
	FleetURL                           string
	EnrollSecret                       string
	EnableScripts                      bool
	Desktop                            bool
	Insecure                           bool
	Debug                              bool
	UpdateURL                          string
	DisableUpdates                     bool
	DesktopChannel                     string
	OrbitChannel                       string
	OsquerydChannel                    string
	HostIdentifier                     string
	EndUserEmail                       string
	EnableEndUserEmailProperty         bool
	EnableEUATokenProperty             bool
	OsqueryDB                          string
	DisableSetupExperience             bool
	FleetDesktopAlternativeBrowserHost string
	OrbitUpdateInterval                string

	// Certificate flags.
	FleetCertificate            string // Non-empty means fleet.pem is present in root
	UpdateTLSServerCertificate  string // Non-empty means update.pem is present
	FleetTLSClientCertificate   string
	UpdateTLSClientCertificate  string

	// OrbitPath is the relative path to orbit.exe within the root directory.
	// e.g. "bin\\orbit\\windows\\stable\\orbit.exe"
	OrbitPath string
}

// fileEntry represents a file discovered in the root directory.
type fileEntry struct {
	name     string // Short name for MSI (e.g. "orbit.exe")
	relPath  string // Relative path with backslashes (e.g. "bin\\orbit\\windows\\stable\\orbit.exe")
	fullPath string // Absolute filesystem path
	size     int64
	dirID    string // MSI Directory table ID this file belongs to
}

// buildDatabase creates all MSI table data from the options and root directory.
func buildDatabase(pool *StringPool, rootDir string, opts MSIOptions) ([]*TableData, []CabFile, error) {
	// Discover files.
	files, dirIDs, err := discoverFiles(rootDir)
	if err != nil {
		return nil, nil, fmt.Errorf("discover files: %w", err)
	}

	// Build CAB file list. CAB entry names MUST match the File table's File key
	// (primary key), not the actual filename — Windows uses the File key as
	// SourceCabKey to locate files within the cabinet. Using actual filenames
	// causes Error 1334 "file cannot be found in cabinet file".
	cabFiles := make([]CabFile, len(files))
	for i, f := range files {
		data, err := os.ReadFile(f.fullPath)
		if err != nil {
			return nil, nil, fmt.Errorf("read %s: %w", f.relPath, err)
		}
		cabFiles[i] = CabFile{
			Name:    fmt.Sprintf("file%d", i), // must match buildFileTable's fileKey
			Data:    data,
			ModTime: MSITimestamp(),
		}
	}

	// Generate component GUIDs for discovered files.
	productCode := GenerateGUID()

	// Build all tables.
	var tables []*TableData

	tables = append(tables, buildPropertyTable(opts, productCode))
	tables = append(tables, buildDirectoryTable(dirIDs))
	tables = append(tables, buildComponentTable(opts, files))
	tables = append(tables, buildFeatureTable())
	tables = append(tables, buildFeatureComponentsTable(files))
	tables = append(tables, buildFileTable(files))
	tables = append(tables, buildMediaTable(len(files)))
	tables = append(tables, buildServiceInstallTable(opts, files))
	tables = append(tables, buildServiceControlTable())
	tables = append(tables, buildInstallExecuteSequenceTable())
	// TODO: CustomAction, AppSearch, and RegLocator are disabled due to a
	// mini-stream data loading bug on Windows. The MSI engine fails to load
	// certain table data from the mini-stream in a data-dependent manner.
	// All other tables (15) work correctly. Custom action functionality
	// (PowerShell scripts for osquery removal, secret updates) will need to
	// be re-added once the mini-stream issue is resolved.
	// tables = append(tables, buildCustomActionTable())
	tables = append(tables, buildUpgradeTable(opts))
	tables = append(tables, buildRegistryTable())
	tables = append(tables, buildCreateFolderTable())
	tables = append(tables, buildEnvironmentTable(opts))
	tables = append(tables, buildMsiServiceConfigFailureActionsTable())
	// tables = append(tables, buildRegLocatorTable())  // Disabled — see CustomAction TODO above
	// tables = append(tables, buildAppSearchTable())   // Disabled — see CustomAction TODO above

	return tables, cabFiles, nil
}

// discoverFiles walks the root directory and returns file entries and directory IDs.
func discoverFiles(rootDir string) ([]fileEntry, map[string]string, error) {
	var files []fileEntry
	// dirIDs maps relative directory paths (with backslashes) to MSI Directory IDs.
	dirIDs := map[string]string{
		"": "ORBITROOT",
	}

	dirCounter := 0
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == rootDir {
			return nil
		}
		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		// Convert to backslash separators for MSI/Windows.
		relBS := strings.ReplaceAll(rel, "/", "\\")

		if d.IsDir() {
			// Generate a unique directory ID.
			dirCounter++
			dirID := fmt.Sprintf("dir%d", dirCounter)
			dirIDs[relBS] = dirID
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		parentDir := filepath.Dir(rel)
		parentDirBS := strings.ReplaceAll(parentDir, "/", "\\")
		if parentDir == "." {
			parentDirBS = ""
		}
		dirID := dirIDs[parentDirBS]
		if dirID == "" {
			dirID = "ORBITROOT"
		}

		files = append(files, fileEntry{
			name:     filepath.Base(rel),
			relPath:  relBS,
			fullPath: path,
			size:     info.Size(),
			dirID:    dirID,
		})
		return nil
	})
	return files, dirIDs, err
}

func buildPropertyTable(opts MSIOptions, productCode string) *TableData {
	rows := [][]any{
		{"ProductCode", productCode},
		{"ProductName", opts.ProductName},
		{"ProductVersion", opts.ProductVersion},
		{"ProductLanguage", "1033"},
		{"Manufacturer", opts.Manufacturer},
		{"UpgradeCode", opts.UpgradeCode},
		{"REINSTALLMODE", "amus"},
		{"ARPNOREPAIR", "yes"},
		{"ARPNOMODIFY", "yes"},
	}
	if opts.FleetURL != "" {
		rows = append(rows, []any{"FLEET_URL", opts.FleetURL})
	}
	rows = append(rows, []any{"FLEET_SECRET", "dummy"})
	if opts.EnableScripts {
		rows = append(rows, []any{"ENABLE_SCRIPTS", "True"})
	} else {
		rows = append(rows, []any{"ENABLE_SCRIPTS", "False"})
	}
	if opts.Desktop {
		rows = append(rows, []any{"FLEET_DESKTOP", "True"})
	} else {
		rows = append(rows, []any{"FLEET_DESKTOP", "False"})
	}
	if opts.EnableEndUserEmailProperty {
		email := "dummy"
		if opts.EndUserEmail != "" {
			email = opts.EndUserEmail
		}
		rows = append(rows, []any{"END_USER_EMAIL", email})
	}
	if opts.EnableEUATokenProperty {
		rows = append(rows, []any{"EUA_TOKEN", "dummy"})
	}

	return &TableData{
		Schema: TableSchema{
			Name: "Property",
			Columns: []ColumnDef{
				{Name: "Property", Type: colStrPK(72)},
				{Name: "Value", Type: colStrL(255)},
			},
		},
		Rows: rows,
	}
}

func buildDirectoryTable(dirIDs map[string]string) *TableData {
	// Standard MSI directory tree.
	rows := [][]any{
		{"TARGETDIR", nil, "SourceDir"},
		{"ProgramFiles64Folder", "TARGETDIR", "ProgramFiles64Folder"},
		{"ORBITROOT", "ProgramFiles64Folder", "Orbit"},
	}

	// Add discovered subdirectories.
	for relPath, dirID := range dirIDs {
		if relPath == "" {
			continue // ORBITROOT already added
		}
		// Use backslash-aware splitting (not filepath.Dir/Base which is OS-dependent).
		parentPathBS := ""
		dirName := relPath
		if idx := strings.LastIndex(relPath, "\\"); idx >= 0 {
			parentPathBS = relPath[:idx]
			dirName = relPath[idx+1:]
		}
		parentID := dirIDs[parentPathBS]
		if parentID == "" {
			parentID = "ORBITROOT"
		}
		rows = append(rows, []any{dirID, parentID, dirName})
	}

	return &TableData{
		Schema: TableSchema{
			Name: "Directory",
			Columns: []ColumnDef{
				{Name: "Directory", Type: colStrPK(72)},
				{Name: "Directory_Parent", Type: colStrN(72)},
				{Name: "DefaultDir", Type: colStrL(255)},
			},
		},
		Rows: rows,
	}
}

func buildComponentTable(opts MSIOptions, files []fileEntry) *TableData {
	rows := [][]any{
		// Fixed root components.
		{"C_ORBITROOT", "{A7DFD09E-2D2B-4535-A04F-5D4DE90F3863}", "ORBITROOT", int32(0), nil, nil},
	}

	// One component per file. Each file is the KeyPath of its own component,
	// so MSI knows the file to install and (for services) which file is the
	// service binary. Condition stays nil — we always install every file.
	for i, f := range files {
		compID := fmt.Sprintf("C_file%d", i)
		guid := GenerateGUID()
		fileKey := fmt.Sprintf("file%d", i)
		rows = append(rows, []any{compID, guid, f.dirID, int32(0), nil, fileKey})
	}

	return &TableData{
		Schema: TableSchema{
			Name: "Component",
			Columns: []ColumnDef{
				{Name: "Component", Type: colStrPK(72)},
				{Name: "ComponentId", Type: colStrN(38)},
				{Name: "Directory_", Type: colStr(72)},
				{Name: "Attributes", Type: colTypeLong},
				{Name: "Condition", Type: colStrN(255)},
				{Name: "KeyPath", Type: colStrN(72)},
			},
		},
		Rows: rows,
	}
}

func buildFeatureTable() *TableData {
	return &TableData{
		Schema: TableSchema{
			Name: "Feature",
			Columns: []ColumnDef{
				{Name: "Feature", Type: colStrPK(38)},
				{Name: "Feature_Parent", Type: colStrN(38)},
				{Name: "Title", Type: colStrLN(64)},
				{Name: "Description", Type: colStrLN(255)},
				{Name: "Display", Type: colTypeShort | colNullable},
				{Name: "Level", Type: colTypeShort},
				{Name: "Directory_", Type: colStrN(72)},
				{Name: "Attributes", Type: colTypeShort},
			},
		},
		Rows: [][]any{
			{"Orbit", nil, "Fleet osquery", nil, int16(0), int16(1), "ORBITROOT", int16(0)},
		},
	}
}

func buildFeatureComponentsTable(files []fileEntry) *TableData {
	rows := [][]any{
		{"Orbit", "C_ORBITROOT"},
	}
	for i := range files {
		rows = append(rows, []any{"Orbit", fmt.Sprintf("C_file%d", i)})
	}

	return &TableData{
		Schema: TableSchema{
			Name: "FeatureComponents",
			Columns: []ColumnDef{
				{Name: "Feature_", Type: colStrPK(38)},
				{Name: "Component_", Type: colStrPK(72)},
			},
		},
		Rows: rows,
	}
}

func buildFileTable(files []fileEntry) *TableData {
	var rows [][]any
	for i, f := range files {
		fileKey := fmt.Sprintf("file%d", i)
		compKey := fmt.Sprintf("C_file%d", i)
		// Sequence numbers are 1-based and must match the CAB file order.
		rows = append(rows, []any{fileKey, compKey, f.name, int32(f.size), nil, nil, int16(0), int16(i + 1)}) //nolint:gosec // G115: file size and sequence number fit in int32/int16 for MSI format
	}

	return &TableData{
		Schema: TableSchema{
			Name: "File",
			Columns: []ColumnDef{
				{Name: "File", Type: colStrPK(72)},
				{Name: "Component_", Type: colStr(72)},
				{Name: "FileName", Type: colStrL(255)},
				{Name: "FileSize", Type: colTypeLong},
				{Name: "Version", Type: colStrN(72)},
				{Name: "Language", Type: colStrN(20)},
				{Name: "Attributes", Type: colTypeShort | colNullable},
				{Name: "Sequence", Type: colTypeShort},
			},
		},
		Rows: rows,
	}
}

func buildMediaTable(fileCount int) *TableData {
	return &TableData{
		Schema: TableSchema{
			Name: "Media",
			Columns: []ColumnDef{
				{Name: "DiskId", Type: colTypeShort | colPrimaryKey},
				{Name: "LastSequence", Type: colTypeShort},
				{Name: "DiskPrompt", Type: colStrLN(64)},
				{Name: "Cabinet", Type: colStrN(255)},
				{Name: "VolumeLabel", Type: colStrN(32)},
				{Name: "Source", Type: colStrN(72)},
			},
		},
		Rows: [][]any{
			{int16(1), int16(fileCount), nil, "#orbit.cab", nil, nil}, //nolint:gosec // G115
		},
	}
}

func buildServiceInstallTable(opts MSIOptions, files []fileEntry) *TableData {
	// Build the service arguments matching the WiX template.
	args := buildServiceArgs(opts)

	// The service's binary file comes from Component_'s KeyPath. Tie the
	// service to the component that owns orbit.exe so Windows launches it
	// instead of the (empty) root directory component.
	serviceComponent := "C_ORBITROOT"
	for i, f := range files {
		if f.name == "orbit.exe" {
			serviceComponent = fmt.Sprintf("C_file%d", i)
			break
		}
	}

	return &TableData{
		Schema: TableSchema{
			Name: "ServiceInstall",
			Columns: []ColumnDef{
				{Name: "ServiceInstall", Type: colStrPK(72)},
				{Name: "Name", Type: colStr(255)},
				{Name: "DisplayName", Type: colStrLN(255)},
				{Name: "ServiceType", Type: colTypeLong},
				{Name: "StartType", Type: colTypeLong},
				{Name: "ErrorControl", Type: colTypeLong},
				{Name: "LoadOrderGroup", Type: colStrN(255)},
				{Name: "Dependencies", Type: colStrN(255)},
				{Name: "StartName", Type: colStrN(255)},
				{Name: "Password", Type: colStrN(255)},
				{Name: "Arguments", Type: colStrN(255)},
				{Name: "Component_", Type: colStr(72)},
				{Name: "Description", Type: colStrLN(255)},
			},
		},
		Rows: [][]any{{
			"OrbitService",    // ServiceInstall (key)
			"Fleet osquery",   // Name
			"Fleet osquery",   // DisplayName
			int32(0x10),       // ServiceType: SERVICE_WIN32_OWN_PROCESS
			int32(2),          // StartType: SERVICE_AUTO_START
			int32(0),          // ErrorControl: SERVICE_ERROR_IGNORE
			nil,               // LoadOrderGroup
			nil,               // Dependencies
			"LocalSystem",     // StartName
			nil,               // Password
			args,              // Arguments
			serviceComponent,  // Component_ — must own orbit.exe as KeyPath
			"This service runs Fleet's osquery runtime and autoupdater (Orbit).", // Description
		}},
	}
}

func buildServiceArgs(opts MSIOptions) string {
	var args []string
	args = append(args, `--root-dir "[ORBITROOT]."`)
	args = append(args, `--log-file "[System64Folder]config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log"`)
	args = append(args, `--fleet-url "[FLEET_URL]"`)
	if opts.FleetCertificate != "" {
		args = append(args, `--fleet-certificate "[ORBITROOT]fleet.pem"`)
	}
	if opts.EnrollSecret != "" {
		args = append(args, `--enroll-secret-path "[ORBITROOT]secret.txt"`)
	}
	if opts.Insecure {
		args = append(args, "--insecure")
	}
	if opts.Debug {
		args = append(args, "--debug")
	}
	if opts.UpdateURL != "" {
		args = append(args, fmt.Sprintf(`--update-url "%s"`, opts.UpdateURL))
	}
	if opts.UpdateTLSServerCertificate != "" {
		args = append(args, `--update-tls-certificate "[ORBITROOT]update.pem"`)
	}
	if opts.DisableUpdates {
		args = append(args, "--disable-updates")
	}
	args = append(args, `--fleet-desktop="[FLEET_DESKTOP]"`)
	args = append(args, fmt.Sprintf("--desktop-channel %s", opts.DesktopChannel))
	if opts.FleetDesktopAlternativeBrowserHost != "" {
		args = append(args, fmt.Sprintf("--fleet-desktop-alternative-browser-host %s", opts.FleetDesktopAlternativeBrowserHost))
	}
	args = append(args, fmt.Sprintf(`--orbit-channel "%s"`, opts.OrbitChannel))
	args = append(args, fmt.Sprintf(`--osqueryd-channel "%s"`, opts.OsquerydChannel))
	args = append(args, `--enable-scripts="[ENABLE_SCRIPTS]"`)
	if opts.HostIdentifier != "" && opts.HostIdentifier != "uuid" {
		args = append(args, fmt.Sprintf("--host-identifier=%s", opts.HostIdentifier))
	}
	if opts.EnableEndUserEmailProperty {
		args = append(args, `--end-user-email="[END_USER_EMAIL]"`)
	} else if opts.EndUserEmail != "" {
		args = append(args, fmt.Sprintf(`--end-user-email "%s"`, opts.EndUserEmail))
	}
	if opts.EnableEUATokenProperty {
		args = append(args, `--eua-token="[EUA_TOKEN]"`)
	}
	if opts.OsqueryDB != "" {
		args = append(args, fmt.Sprintf(`--osquery-db="%s"`, opts.OsqueryDB))
	}
	if opts.DisableSetupExperience {
		args = append(args, "--disable-setup-experience")
	}
	return strings.Join(args, " ")
}

func buildServiceControlTable() *TableData {
	return &TableData{
		Schema: TableSchema{
			Name: "ServiceControl",
			Columns: []ColumnDef{
				{Name: "ServiceControl", Type: colStrPK(72)},
				{Name: "Name", Type: colStrL(255)},
				{Name: "Event", Type: colTypeShort},
				{Name: "Arguments", Type: colStrLN(255)},
				{Name: "Wait", Type: colTypeShort | colNullable},
				{Name: "Component_", Type: colStr(72)},
			},
		},
		Rows: [][]any{{
			"StartOrbitService",
			"Fleet osquery",
			int16(0x1 | 0x2 | 0x8 | 0x10 | 0x80), // start=install, stop=install+uninstall, delete=uninstall
			nil,
			int16(1), // Wait for service to start
			"C_ORBITROOT",
		}},
	}
}

func buildCustomActionTable() *TableData {
	// Type 51 = SetProperty (set property from formatted string).
	// Type 3122 = 50 (exe from property) + 1024 (deferred) + 2048 (no impersonate) → Return=check
	// Type 3186 = 3122 + 64 (continue on error) → Return=ignore
	const (
		typeSetProperty = int16(51)
		typeExeCheck    = int16(3122)  //nolint:gosec // G115: intentional
		typeExeIgnore   = int16(3186)  //nolint:gosec // G115: intentional
	)

	psPrefix := `"[POWERSHELLEXE]" -NoLogo -NonInteractive -NoProfile -ExecutionPolicy Bypass`

	return &TableData{
		Schema: TableSchema{
			Name: "CustomAction",
			Columns: []ColumnDef{
				{Name: "Action", Type: colStrPK(72)},
				{Name: "Type", Type: colTypeShort},
				{Name: "Source", Type: colStrN(72)},
				{Name: "Target", Type: colStrN(255)},
			},
		},
		Rows: [][]any{
			// SetProperty actions (set command line for deferred execution).
			{"CA_SetUninstallOsquery", typeSetProperty, "CA_UninstallOsquery", psPrefix + ` -File "[ORBITROOT]installer_utils.ps1" -uninstallOsquery`},
			{"CA_SetRemoveOrbit", typeSetProperty, "CA_RemoveOrbit", psPrefix + ` -File "[ORBITROOT]installer_utils.ps1" -uninstallOrbit`},
			{"CA_SetUpdateSecret", typeSetProperty, "CA_UpdateSecret", psPrefix + ` -File "[ORBITROOT]installer_utils.ps1" -updateSecret "[FLEET_SECRET]"`},
			{"CA_SetWaitOrbit", typeSetProperty, "CA_WaitOrbit", psPrefix + ` Wait-Process -Name orbit -Timeout 30 -ErrorAction SilentlyContinue`},
			{"CA_SetRemoveRebootPending", typeSetProperty, "CA_RemoveRebootPending", psPrefix + ` Remove-Item -Path "$Env:Programfiles\orbit\bin" -Recurse -Force`},
			// Execution actions (deferred, run as SYSTEM).
			{"CA_UninstallOsquery", typeExeCheck, "CA_UninstallOsquery", nil},
			{"CA_RemoveOrbit", typeExeCheck, "CA_RemoveOrbit", nil},
			{"CA_UpdateSecret", typeExeCheck, "CA_UpdateSecret", nil},
			{"CA_WaitOrbit", typeExeIgnore, "CA_WaitOrbit", nil},
			{"CA_RemoveRebootPending", typeExeIgnore, "CA_RemoveRebootPending", nil},
		},
	}
}

func buildInstallExecuteSequenceTable() *TableData {
	return &TableData{
		Schema: TableSchema{
			Name: "InstallExecuteSequence",
			Columns: []ColumnDef{
				{Name: "Action", Type: colStrPK(72)},
				{Name: "Condition", Type: colStrN(255)},
				{Name: "Sequence", Type: colTypeShort | colNullable},
			},
		},
		Rows: [][]any{
			// Standard actions in typical sequence order.
			{"LaunchConditions", nil, int16(100)},
			{"FindRelatedProducts", nil, int16(200)},
			{"CostInitialize", nil, int16(800)},
			{"FileCost", nil, int16(900)},
			{"CostFinalize", nil, int16(1000)},
			{"InstallValidate", nil, int16(1400)},
			{"InstallInitialize", nil, int16(1500)},
			{"ProcessComponents", nil, int16(1600)},
			{"RemoveFiles", nil, int16(3500)},
			{"InstallFiles", nil, int16(4000)},
			{"InstallServices", nil, int16(5800)},
			{"StartServices", nil, int16(5900)},
			{"RegisterProduct", nil, int16(6100)},
			{"PublishProduct", nil, int16(6400)},
			{"InstallFinalize", nil, int16(6600)},
			{"RemoveExistingProducts", nil, int16(6700)},
		},
	}
}

func buildUpgradeTable(opts MSIOptions) *TableData {
	return &TableData{
		Schema: TableSchema{
			Name: "Upgrade",
			Columns: []ColumnDef{
				{Name: "UpgradeCode", Type: colStrPK(72)},
				{Name: "VersionMin", Type: colStrN(20)},
				{Name: "VersionMax", Type: colStrN(20)},
				{Name: "Language", Type: colStrN(255)},
				{Name: "Attributes", Type: colTypeLong},
				{Name: "Remove", Type: colStrN(255)},
				{Name: "ActionProperty", Type: colStr(72)},
			},
		},
		Rows: [][]any{
			// AllowDowngrades: detect all versions and set WIX_DOWNGRADE_DETECTED.
			{opts.UpgradeCode, opts.ProductVersion, nil, nil, int32(0x100), nil, "WIX_DOWNGRADE_DETECTED"}, // OnlyDetect, VersionMin=current
			{opts.UpgradeCode, nil, opts.ProductVersion, nil, int32(0x100), nil, "WIX_UPGRADE_DETECTED"},   // OnlyDetect, VersionMax=current
		},
	}
}

func buildRegistryTable() *TableData {
	return &TableData{
		Schema: TableSchema{
			Name: "Registry",
			Columns: []ColumnDef{
				{Name: "Registry", Type: colStrPK(72)},
				{Name: "Root", Type: colTypeShort},
				{Name: "Key", Type: colStrL(255)},
				{Name: "Name", Type: colStrLN(255)},
				{Name: "Value", Type: colStrLN(255)},
				{Name: "Component_", Type: colStr(72)},
			},
		},
		Rows: [][]any{
			{"reg_OrbitPath", int16(2), `SOFTWARE\FleetDM\Orbit`, "Path", "[ORBITROOT]", "C_ORBITROOT"}, // Root=2 is HKLM
		},
	}
}

func buildCreateFolderTable() *TableData {
	return &TableData{
		Schema: TableSchema{
			Name: "CreateFolder",
			Columns: []ColumnDef{
				{Name: "Directory_", Type: colStrPK(72)},
				{Name: "Component_", Type: colStrPK(72)},
			},
		},
		Rows: [][]any{
			{"ORBITROOT", "C_ORBITROOT"},
		},
	}
}

func buildEnvironmentTable(opts MSIOptions) *TableData {
	interval := opts.OrbitUpdateInterval
	if interval == "" {
		interval = "15m"
	}
	return &TableData{
		Schema: TableSchema{
			Name: "Environment",
			Columns: []ColumnDef{
				{Name: "Environment", Type: colStrPK(72)},
				{Name: "Name", Type: colStrL(255)},
				{Name: "Value", Type: colStrLN(255)},
				{Name: "Component_", Type: colStr(72)},
			},
		},
		Rows: [][]any{
			{"env_OrbitUpdateInterval", "=-ORBIT_UPDATE_INTERVAL", interval, "C_ORBITROOT"},
		},
	}
}

func buildMsiServiceConfigFailureActionsTable() *TableData {
	return &TableData{
		Schema: TableSchema{
			Name: "MsiServiceConfigFailureActions",
			Columns: []ColumnDef{
				{Name: "MsiServiceConfigFailureActions", Type: colStrPK(72)},
				{Name: "Name", Type: colStr(255)},
				{Name: "Event", Type: colTypeShort},
				{Name: "ResetPeriod", Type: colTypeLong | colNullable},
				{Name: "RebootMessage", Type: colStrLN(255)},
				{Name: "Command", Type: colStrLN(255)},
				{Name: "Actions", Type: colStrN(255)},
				{Name: "DelayActions", Type: colStrN(255)},
				{Name: "Component_", Type: colStr(72)},
			},
		},
		Rows: [][]any{{
			"svcfailure_orbit",
			"Fleet osquery",
			int16(1),          // Event: install
			int32(86400),      // ResetPeriod: 1 day in seconds
			nil,               // RebootMessage
			nil,               // Command
			"1/1000/1/1000/1/1000", // Actions: restart(1)/1000ms delay × 3
			"1000/1000/1000",       // DelayActions
			"C_ORBITROOT",
		}},
	}
}

func buildRegLocatorTable() *TableData {
	return &TableData{
		Schema: TableSchema{
			Name: "RegLocator",
			Columns: []ColumnDef{
				{Name: "Signature_", Type: colStrPK(72)},
				{Name: "Root", Type: colTypeShort},
				{Name: "Key", Type: colStr(255)},
				{Name: "Name", Type: colStrN(255)},
				{Name: "Type", Type: colTypeShort | colNullable},
			},
		},
		Rows: [][]any{
			{"APPLICATIONFOLDER_REGSEARCH", int16(2), `SOFTWARE\FleetDM\Orbit`, "Path", int16(2)}, // Root=2=HKLM, Type=2=raw
			{"POWERSHELLEXE", int16(2), `SOFTWARE\Microsoft\PowerShell\1\ShellIds\Microsoft.PowerShell`, "Path", int16(2)},
		},
	}
}

func buildAppSearchTable() *TableData {
	return &TableData{
		Schema: TableSchema{
			Name: "AppSearch",
			Columns: []ColumnDef{
				{Name: "Property", Type: colStrPK(72)},
				{Name: "Signature_", Type: colStrPK(72)},
			},
		},
		Rows: [][]any{
			{"APPLICATIONFOLDER", "APPLICATIONFOLDER_REGSEARCH"},
			{"POWERSHELLEXE", "POWERSHELLEXE"},
		},
	}
}

func buildLockPermissionsTable(files []fileEntry) *TableData {
	standardSDDL := "O:SYG:SYD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;BU)"
	restrictedSDDL := "O:SYG:SYD:PAI(A;;FA;;;SY)(A;;FA;;;BA)"

	var rows [][]any
	// Folder permissions.
	rows = append(rows, []any{"ORBITROOT", "CreateFolder", "ORBITROOT", standardSDDL})

	// File permissions.
	for i, f := range files {
		fileKey := fmt.Sprintf("file%d", i)
		sddl := standardSDDL
		if f.name == "secret.txt" {
			sddl = restrictedSDDL
		}
		rows = append(rows, []any{fileKey, "File", fileKey, sddl})
	}

	return &TableData{
		Schema: TableSchema{
			Name: "MsiLockPermissionsEx",
			Columns: []ColumnDef{
				{Name: "MsiLockPermissionsEx", Type: colStrPK(72)},
				{Name: "LockObject", Type: colStr(72)},
				{Name: "Table", Type: colStr(72)},
				{Name: "SDDLText", Type: colStr(72)},
			},
		},
		Rows: rows,
	}
}
