package msi

import (
	"fmt"
	"sort"
	"strings"
)

// model.go builds the fleetd installer database content: everything that
// was authored in the main.wxs template plus the harvested orbit root.
// Rows are inserted in the same order WiX inserts them (tables
// alphabetically, rows in the documented per-table order) because insertion
// order assigns string-pool IDs, which in turn dictate physical row order.

// Options declares the configurable parts of the fleetd MSI. Fields mirror
// the packaging.Options fields that the main.wxs template consumed.
type Options struct {
	Architecture string // "amd64" or "arm64"
	Version      string // product version, e.g. "1.58.0"

	FleetURL                           string
	EnrollSecret                       bool // include --enroll-secret-path in the service arguments
	FleetCertificate                   bool
	Insecure                           bool
	Debug                              bool
	UpdateURL                          string
	UpdateTLSServerCertificate         bool
	DisableUpdates                     bool
	Desktop                            bool
	DesktopChannel                     string
	OrbitChannel                       string
	OsquerydChannel                    string
	FleetDesktopAlternativeBrowserHost string
	HostIdentifier                     string
	EnableScripts                      bool
	EnableEndUserEmailProperty         bool
	EndUserEmail                       string
	EnableEUATokenProperty             bool
	OsqueryDB                          string
	DisableSetupExperience             bool
	BypassEndUserAuth                  bool
	OrbitUpdateInterval                string // e.g. "15m0s"
	NativePlatform                     string // "windows" or "windows-arm64"
}

const (
	upgradeCode            = "{B681CB20-107E-428A-9B14-2D3C1AFED244}"
	componentGUIDOrbitRoot = "{A7DFD09E-2D2B-4535-A04F-5D4DE90F3863}"
	componentGUIDOrbitBin  = "{AF347B4E-B84B-4DD4-9C4D-133BE17B613D}"

	// SDDLs applied by the former TransformHeat step and main.wxs. See
	// orbit/pkg/packaging/wix/transform.go for their provenance: SYSTEM and
	// Administrators get full access, Users read/execute; the secret file
	// drops the Users entry.
	sddlDefault = "O:SYG:SYD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;BU)"
	sddlSecret  = "O:SYG:SYD:PAI(A;;FA;;;SY)(A;;FA;;;BA)" //nolint:gosec // SDDL access-control string, not a credential

	fileAttrVital = 512

	// msidbServiceControlEvent: start+stop on install, stop+delete on uninstall
	serviceControlEvents = 163
)

// fileRecord is one File-table row (authored orbit.exe or harvested file)
// with everything the File/MsiFileHash/cab writers need.
type fileRecord struct {
	FileID      string
	ComponentID string
	FileName    string // MSI FileName column value
	Path        string
	Size        int64
	Version     string
	Language    string
	Hash        [16]byte
	Sequence    int // assigned by sequenceFiles
	ModTimeSet  bool
}

// serviceArguments reproduces the ServiceInstall/@Arguments expression from
// the main.wxs template, byte for byte (including its spacing quirks: the
// always-present space after --enable-scripts and the space-prefixed
// optional arguments).
func serviceArguments(opt Options) string {
	var b strings.Builder
	b.WriteString(`--root-dir "[ORBITROOT]." --log-file "[System64Folder]config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log" --fleet-url "[FLEET_URL]"`)
	if opt.FleetCertificate {
		b.WriteString(` --fleet-certificate "[ORBITROOT]fleet.pem"`)
	}
	if opt.EnrollSecret {
		b.WriteString(` --enroll-secret-path "[ORBITROOT]secret.txt"`)
	}
	if opt.Insecure {
		b.WriteString(` --insecure`)
	}
	if opt.Debug {
		b.WriteString(` --debug`)
	}
	if opt.UpdateURL != "" {
		fmt.Fprintf(&b, ` --update-url "%s"`, opt.UpdateURL)
	}
	if opt.UpdateTLSServerCertificate {
		b.WriteString(` --update-tls-certificate "[ORBITROOT]update.pem"`)
	}
	if opt.DisableUpdates {
		b.WriteString(` --disable-updates`)
	}
	fmt.Fprintf(&b, ` --fleet-desktop="[FLEET_DESKTOP]" --desktop-channel %s`, opt.DesktopChannel)
	if opt.FleetDesktopAlternativeBrowserHost != "" {
		fmt.Fprintf(&b, ` --fleet-desktop-alternative-browser-host %s`, opt.FleetDesktopAlternativeBrowserHost)
	}
	fmt.Fprintf(&b, ` --orbit-channel "%s" --osqueryd-channel "%s" --enable-scripts="[ENABLE_SCRIPTS]" `, opt.OrbitChannel, opt.OsquerydChannel)
	if opt.HostIdentifier != "" && opt.HostIdentifier != "uuid" {
		fmt.Fprintf(&b, `--host-identifier=%s`, opt.HostIdentifier)
	}
	if opt.EnableEndUserEmailProperty {
		b.WriteString(` --end-user-email="[END_USER_EMAIL]"`)
	} else if opt.EndUserEmail != "" {
		fmt.Fprintf(&b, ` --end-user-email "%s"`, opt.EndUserEmail)
	}
	if opt.EnableEUATokenProperty {
		b.WriteString(` --eua-token="[EUA_TOKEN]"`)
	}
	if opt.OsqueryDB != "" {
		fmt.Fprintf(&b, ` --osquery-db="%s"`, opt.OsqueryDB)
	}
	if opt.DisableSetupExperience {
		b.WriteString(` --disable-setup-experience`)
	}
	if opt.BypassEndUserAuth {
		b.WriteString(` --bypass-end-user-auth`)
	}
	return b.String()
}

const powershellPrefix = `"[POWERSHELLEXE]" -NoLogo -NonInteractive -NoProfile -ExecutionPolicy Bypass`

// customAction describes one SetProperty+CustomAction pair (deferred
// WixQuietExec64 execution of installer_utils.ps1) plus its scheduling.
type customAction struct {
	name   string // e.g. "CA_UninstallOsquery"
	target string // the command line set into the property
	caType int    // type of the deferred action (3073 or 3137)
	setSeq int    // InstallExecuteSequence position of the SetProperty action
	seq    int    // InstallExecuteSequence position of the action itself
	cond   string // condition of the action ("" = none)
}

// fleetdCustomActions returns the custom actions in main.wxs authoring
// order. The sequence numbers are where WiX scheduled them relative to the
// standard actions (Before/After RemoveFiles, InstallFiles, InstallServices).
func fleetdCustomActions() []customAction {
	return []customAction{
		{
			name:   "CA_UninstallOsquery",
			target: powershellPrefix + ` -File "[ORBITROOT]installer_utils.ps1" -uninstallOsquery`,
			caType: 3073,            // msidbCustomActionTypeDll|Continue? deferred, no impersonation
			setSeq: 4001, seq: 4002, // After InstallFiles
			cond: "NOT Installed AND NOT WIX_UPGRADE_DETECTED",
		},
		{
			name:   "CA_RemoveOrbit",
			target: powershellPrefix + ` -File "[ORBITROOT]installer_utils.ps1" -uninstallOrbit`,
			caType: 3073,
			setSeq: 3498, seq: 3499, // Before RemoveFiles
			cond: `(NOT UPGRADINGPRODUCTCODE) AND (REMOVE="ALL")`,
		},
		{
			name:   "CA_UpdateSecret",
			target: powershellPrefix + ` -File "[ORBITROOT]installer_utils.ps1" -updateSecret "[FLEET_SECRET]"`,
			caType: 3073,
			setSeq: 5798, seq: 5799, // Before InstallServices
			cond: "NOT Installed",
		},
		{
			name:   "CA_WaitOrbit",
			target: powershellPrefix + ` Wait-Process -Name orbit -Timeout 30 -ErrorAction SilentlyContinue`,
			caType: 3137,            // + continue on error
			setSeq: 3996, seq: 3997, // Before CA_RemoveRebootPending
		},
		{
			name:   "CA_RemoveRebootPending",
			target: powershellPrefix + ` Remove-Item -Path "$Env:Programfiles\orbit\bin" -Recurse -Force`,
			caType: 3137,
			setSeq: 3998, seq: 3999, // Before InstallFiles
			cond: "NOT Installed",
		},
	}
}

// sequenceFiles assigns File.Sequence numbers: WiX orders cabinet entries by
// file key (ordinal), so sequence numbers follow the key-sorted order.
func sequenceFiles(files []*fileRecord) {
	sorted := make([]*fileRecord, len(files))
	copy(sorted, files)
	sort.SliceStable(sorted, func(a, b int) bool { return sorted[a].FileID < sorted[b].FileID })
	for i, f := range sorted {
		f.Sequence = i + 1
	}
}

// populate fills the database with the fleetd installer rows, in WiX
// insertion order. files is returned in sequence (cab) order.
func populate(db *database, opt Options, h *harvest, productCode string) ([]*fileRecord, error) {
	arm := opt.Architecture == "arm64"

	// Collect the file records: the authored orbit.exe (installed to
	// Orbit\bin\orbit\) plus every harvested file.
	files := []*fileRecord{}
	for _, c := range h.Components {
		if c.File == nil {
			continue
		}
		f := c.File
		if f.Size > 2147483647 {
			return nil, fmt.Errorf("file %s is %d bytes; MSI file sizes are limited to 2GB", f.Path, f.Size)
		}
		files = append(files, &fileRecord{
			FileID:      f.FileID,
			ComponentID: f.ComponentID,
			FileName:    f.FileName,
			Path:        f.Path,
			Size:        f.Size,
			Version:     f.Version,
			Language:    f.Language,
			Hash:        f.Hash,
		})
	}

	// The authored orbit.exe File element sources the harvested binary at
	// bin/orbit/<platform>/<channel>/orbit.exe.
	orbitSrc := fmt.Sprintf("bin/orbit/%s/%s/orbit.exe", opt.NativePlatform, opt.OrbitChannel)
	var authored *fileRecord
	for _, f := range files {
		if strings.HasSuffix(strings.ReplaceAll(f.Path, "\\", "/"), orbitSrc) {
			authored = &fileRecord{
				FileID:      "orbit.exe",
				ComponentID: "C_ORBITBIN",
				FileName:    "orbit.exe",
				Path:        f.Path,
				Size:        f.Size,
				Version:     f.Version,
				Language:    f.Language,
				Hash:        f.Hash,
			}
			break
		}
	}
	if authored == nil {
		return nil, fmt.Errorf("orbit.exe not found under the orbit root at %s", orbitSrc)
	}
	allFiles := append([]*fileRecord{authored}, files...)
	sequenceFiles(allFiles)

	ins := func(table string, cells ...cell) error { return db.insert(table, cells...) }
	must := func(errs ...error) error {
		for _, err := range errs {
			if err != nil {
				return err
			}
		}
		return nil
	}

	// AdminExecuteSequence
	if err := must(
		ins("AdminExecuteSequence", "CostInitialize", nil, 800),
		ins("AdminExecuteSequence", "FileCost", nil, 900),
		ins("AdminExecuteSequence", "CostFinalize", nil, 1000),
		ins("AdminExecuteSequence", "InstallValidate", nil, 1400),
		ins("AdminExecuteSequence", "InstallInitialize", nil, 1500),
		ins("AdminExecuteSequence", "InstallAdminPackage", nil, 3900),
		ins("AdminExecuteSequence", "InstallFiles", nil, 4000),
		ins("AdminExecuteSequence", "InstallFinalize", nil, 6600),
	); err != nil {
		return nil, err
	}

	// AdminUISequence
	if err := must(
		ins("AdminUISequence", "CostInitialize", nil, 800),
		ins("AdminUISequence", "FileCost", nil, 900),
		ins("AdminUISequence", "CostFinalize", nil, 1000),
		ins("AdminUISequence", "ExecuteAction", nil, 1300),
	); err != nil {
		return nil, err
	}

	// AdvtExecuteSequence
	if err := must(
		ins("AdvtExecuteSequence", "CostInitialize", nil, 800),
		ins("AdvtExecuteSequence", "CostFinalize", nil, 1000),
		ins("AdvtExecuteSequence", "InstallValidate", nil, 1400),
		ins("AdvtExecuteSequence", "InstallInitialize", nil, 1500),
		ins("AdvtExecuteSequence", "PublishFeatures", nil, 6300),
		ins("AdvtExecuteSequence", "PublishProduct", nil, 6400),
		ins("AdvtExecuteSequence", "InstallFinalize", nil, 6600),
	); err != nil {
		return nil, err
	}

	// AppSearch
	if err := must(
		ins("AppSearch", "APPLICATIONFOLDER", "APPLICATIONFOLDER_REGSEARCH"),
		ins("AppSearch", "POWERSHELLEXE", "POWERSHELLEXE"),
	); err != nil {
		return nil, err
	}

	// Binary: the custom-action DLLs; on arm64 the ServiceConfig actions use
	// the A64 build while WixQuietExec64 stays on the x86 WixCA.
	svcBinary := "WixCA"
	caSuffix := ""
	if arm {
		svcBinary = "WixCA_A64"
		caSuffix = "_A64"
		if err := ins("Binary", "WixCA_A64", 1); err != nil {
			return nil, err
		}
	}
	if err := ins("Binary", "WixCA", 1); err != nil {
		return nil, err
	}

	// Component
	if err := must(
		ins("Component", "C_ORBITROOT", componentGUIDOrbitRoot, "ORBITROOT", 256, nil, nil),
		ins("Component", "C_ORBITBIN", componentGUIDOrbitBin, "ORBITBINORBIT", 256, nil, "orbit.exe"),
	); err != nil {
		return nil, err
	}
	for _, c := range h.Components {
		if c.File != nil {
			if err := ins("Component", c.File.ComponentID, c.File.ComponentGUID, c.File.DirID, 256, nil, c.File.FileID); err != nil {
				return nil, err
			}
		} else {
			if err := ins("Component", c.EmptyDir.ComponentID, c.EmptyDir.ComponentGUID, c.EmptyDir.DirID, 256, nil, nil); err != nil {
				return nil, err
			}
		}
	}

	// CreateFolder
	if err := must(
		ins("CreateFolder", "ORBITROOT", "C_ORBITROOT"),
		ins("CreateFolder", "ORBITBINORBIT", "C_ORBITBIN"),
	); err != nil {
		return nil, err
	}
	for _, c := range h.Components {
		if c.EmptyDir != nil {
			if err := ins("CreateFolder", c.EmptyDir.DirID, c.EmptyDir.ComponentID); err != nil {
				return nil, err
			}
		}
	}

	// CustomAction: the SetProperty (type 51) + deferred pairs, then the
	// WixUtilExtension ServiceConfig actions.
	cas := fleetdCustomActions()
	for _, ca := range cas {
		if err := must(
			ins("CustomAction", "Set"+ca.name, 51, ca.name, ca.target, nil),
			ins("CustomAction", ca.name, ca.caType, "WixCA", "WixQuietExec64", nil),
		); err != nil {
			return nil, err
		}
	}
	if err := must(
		ins("CustomAction", "SchedServiceConfig"+caSuffix, 1, svcBinary, "SchedServiceConfig", nil),
		ins("CustomAction", "ExecServiceConfig"+caSuffix, 3073, svcBinary, "ExecServiceConfig", nil),
		ins("CustomAction", "RollbackServiceConfig"+caSuffix, 3329, svcBinary, "RollbackServiceConfig", nil),
	); err != nil {
		return nil, err
	}

	// Directory: authored directories deepest-first, then harvested
	// directories in the order the linker resolves them: walking up from
	// each component's directory.
	if err := must(
		ins("Directory", "ORBITBINORBIT", "ORBITBIN", "orbit"),
		ins("Directory", "ORBITBIN", "ORBITROOT", "bin"),
		ins("Directory", "ORBITROOT", "ProgramFiles64Folder", "Orbit"),
		ins("Directory", "ProgramFiles64Folder", "TARGETDIR", "."),
		ins("Directory", "TARGETDIR", nil, "SourceDir"),
	); err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for _, c := range h.Components {
		for _, d := range h.parentChain(c.dirID()) {
			if _, ok := seen[d.ID]; ok {
				break
			}
			seen[d.ID] = struct{}{}
			if err := ins("Directory", d.ID, d.ParentID, d.DefaultDir); err != nil {
				return nil, err
			}
		}
	}

	// Environment
	if err := ins("Environment", "OrbitUpdateInterval", "=-*ORBIT_UPDATE_INTERVAL", opt.OrbitUpdateInterval, "C_ORBITBIN"); err != nil {
		return nil, err
	}

	// Feature (Display="hidden" -> 0, Level 1)
	if err := ins("Feature", "Orbit", nil, "Fleet osquery", nil, 0, 1, nil, 0); err != nil {
		return nil, err
	}

	// FeatureComponents
	if err := must(
		ins("FeatureComponents", "Orbit", "C_ORBITROOT"),
		ins("FeatureComponents", "Orbit", "C_ORBITBIN"),
	); err != nil {
		return nil, err
	}
	for _, c := range h.Components {
		if err := ins("FeatureComponents", "Orbit", c.id()); err != nil {
			return nil, err
		}
	}

	// File: the authored orbit.exe row first, then harvested files.
	for _, f := range allFiles {
		if err := ins("File", f.FileID, f.ComponentID, f.FileName, int(f.Size),
			nilIfEmpty(f.Version), nilIfEmpty(f.Language), fileAttrVital, f.Sequence); err != nil {
			return nil, err
		}
	}

	// InstallExecuteSequence: standard + custom actions ascending, with
	// RemoveExistingProducts appended last (WiX schedules it after building
	// the rest; afterInstallValidate = 1401).
	type seqRow struct {
		action string
		cond   string
		seq    int
	}
	iesRows := []seqRow{
		{"FindRelatedProducts", "", 25},
		{"AppSearch", "", 50},
		{"ValidateProductID", "", 700},
		{"CostInitialize", "", 800},
		{"FileCost", "", 900},
		{"CostFinalize", "", 1000},
		{"MigrateFeatureStates", "", 1200},
		{"InstallValidate", "", 1400},
		{"InstallInitialize", "", 1500},
		{"ProcessComponents", "", 1600},
		{"UnpublishFeatures", "", 1800},
		{"StopServices", "VersionNT", 1900},
		{"DeleteServices", "VersionNT", 2000},
		{"RemoveEnvironmentStrings", "", 3300},
	}
	for _, ca := range cas {
		iesRows = append(iesRows,
			seqRow{"Set" + ca.name, "", ca.setSeq},
			seqRow{ca.name, ca.cond, ca.seq},
		)
	}
	iesRows = append(iesRows,
		seqRow{"RemoveFiles", "", 3500},
		seqRow{"RemoveFolders", "", 3600},
		seqRow{"CreateFolders", "", 3700},
		seqRow{"InstallFiles", "", 4000},
		seqRow{"WriteEnvironmentStrings", "", 5200},
		seqRow{"InstallServices", "VersionNT", 5800},
		seqRow{"SchedServiceConfig" + caSuffix, `NOT REMOVE~="ALL" AND VersionNT > 400`, 5801},
		seqRow{"StartServices", "VersionNT", 5900},
		seqRow{"RegisterUser", "", 6000},
		seqRow{"RegisterProduct", "", 6100},
		seqRow{"PublishFeatures", "", 6300},
		seqRow{"PublishProduct", "", 6400},
		seqRow{"InstallFinalize", "", 6600},
	)
	sort.SliceStable(iesRows, func(a, b int) bool { return iesRows[a].seq < iesRows[b].seq })
	iesRows = append(iesRows, seqRow{"RemoveExistingProducts", "", 1401})
	for _, r := range iesRows {
		if err := ins("InstallExecuteSequence", r.action, nilIfEmpty(r.cond), r.seq); err != nil {
			return nil, err
		}
	}

	// InstallUISequence
	if err := must(
		ins("InstallUISequence", "FindRelatedProducts", nil, 25),
		ins("InstallUISequence", "AppSearch", nil, 50),
		ins("InstallUISequence", "ValidateProductID", nil, 700),
		ins("InstallUISequence", "CostInitialize", nil, 800),
		ins("InstallUISequence", "FileCost", nil, 900),
		ins("InstallUISequence", "CostFinalize", nil, 1000),
		ins("InstallUISequence", "MigrateFeatureStates", nil, 1200),
		ins("InstallUISequence", "ExecuteAction", nil, 1300),
	); err != nil {
		return nil, err
	}

	// Media
	if err := ins("Media", 1, len(allFiles), nil, "#cab1.cab", nil, nil); err != nil {
		return nil, err
	}

	// MsiFileHash: unversioned files only.
	for _, f := range allFiles {
		if f.Version != "" {
			continue
		}
		h1, h2, h3, h4 := hashParts(f.Hash)
		if err := ins("MsiFileHash", f.FileID, 0, h1, h2, h3, h4); err != nil {
			return nil, err
		}
	}

	// MsiLockPermissionsEx: the two authored CreateFolder permissions, the
	// authored orbit.exe, then every harvested file (secret files get the
	// tighter SDDL) — mirroring TransformHeat.
	if err := must(
		ins("MsiLockPermissionsEx", "ORBITROOT", "ORBITROOT", "CreateFolder", sddlDefault, nil),
		ins("MsiLockPermissionsEx", "ORBITBINORBIT", "ORBITBINORBIT", "CreateFolder", sddlDefault, nil),
		ins("MsiLockPermissionsEx", "orbit.exe", "orbit.exe", "File", sddlDefault, nil),
	); err != nil {
		return nil, err
	}
	for _, c := range h.Components {
		if c.File == nil {
			continue
		}
		sddl := sddlDefault
		if strings.HasSuffix(c.File.Path, "secret.txt") {
			sddl = sddlSecret
		}
		if err := ins("MsiLockPermissionsEx", c.File.FileID, c.File.FileID, "File", sddl, nil); err != nil {
			return nil, err
		}
	}

	// Property
	type prop struct{ name, value string }
	props := []prop{
		{"ALLUSERS", "1"},
		{"REINSTALLMODE", "amus"},
		{"ARPNOREPAIR", "yes"},
		{"ARPNOMODIFY", "yes"},
		{"FLEET_URL", opt.FleetURL},
		{"FLEET_SECRET", "dummy"},
		{"ENABLE_SCRIPTS", boolPropValue(opt.EnableScripts)},
		{"FLEET_DESKTOP", boolPropValue(opt.Desktop)},
	}
	if opt.EnableEndUserEmailProperty {
		v := opt.EndUserEmail
		if v == "" {
			v = "dummy"
		}
		props = append(props, prop{"END_USER_EMAIL", v})
	}
	if opt.EnableEUATokenProperty {
		props = append(props, prop{"EUA_TOKEN", "dummy"})
	}
	props = append(props,
		prop{"Manufacturer", "Fleet Device Management (fleetdm.com)"},
		prop{"ProductCode", productCode},
		prop{"ProductLanguage", "1033"},
		prop{"ProductName", "Fleet osquery"},
		prop{"ProductVersion", opt.Version},
		prop{"UpgradeCode", upgradeCode},
		prop{"SecureCustomProperties", "ARPNOMODIFY;ARPNOREPAIR;WIX_UPGRADE_DETECTED"},
	)
	for _, p := range props {
		if p.value == "" {
			// MSI properties cannot hold empty values (candle rejected
			// them); skip like the template's conditional emission.
			continue
		}
		if err := ins("Property", p.name, p.value); err != nil {
			return nil, err
		}
	}

	// RegLocator (Type 18 = raw value + 64-bit hive)
	if err := must(
		ins("RegLocator", "APPLICATIONFOLDER_REGSEARCH", 2, `SOFTWARE\FleetDM\Orbit`, "Path", 18),
		ins("RegLocator", "POWERSHELLEXE", 2, `SOFTWARE\Microsoft\PowerShell\1\ShellIds\Microsoft.PowerShell`, "Path", 18),
	); err != nil {
		return nil, err
	}

	// ServiceConfig (util:ServiceConfig restart-on-failure settings)
	if err := ins("ServiceConfig", "Fleet osquery", "C_ORBITBIN", 1, "restart", "restart", "restart", 1, 1, nil, nil); err != nil {
		return nil, err
	}

	// ServiceControl
	if err := ins("ServiceControl", "StartOrbitService", "Fleet osquery", serviceControlEvents, nil, nil, "C_ORBITBIN"); err != nil {
		return nil, err
	}

	// ServiceInstall (ownProcess=16, auto start=2, error ignore=0)
	if err := ins("ServiceInstall", "Fleet_osquery", "Fleet osquery", nil, 16, 2, 0,
		nil, nil, "LocalSystem", nil, serviceArguments(opt), "C_ORBITBIN",
		"This service runs Fleet's osquery runtime and autoupdater (Orbit)."); err != nil {
		return nil, err
	}

	// Upgrade (<MajorUpgrade AllowDowngrades="yes"/>: min 0, no max,
	// attributes 257 = MigrateFeatures | VersionMaxInclusive? — verbatim
	// from WiX output)
	if err := ins("Upgrade", upgradeCode, "0", nil, nil, 257, nil, "WIX_UPGRADE_DETECTED"); err != nil {
		return nil, err
	}

	// Return the files in cab (sequence) order.
	inCabOrder := make([]*fileRecord, len(allFiles))
	copy(inCabOrder, allFiles)
	sort.SliceStable(inCabOrder, func(a, b int) bool { return inCabOrder[a].Sequence < inCabOrder[b].Sequence })
	return inCabOrder, nil
}

func boolPropValue(v bool) string {
	if v {
		return "True"
	}
	return "False"
}

// hashParts splits an MsiGetFileHash MD5 into the four signed 32-bit
// little-endian parts stored in MsiFileHash.
func hashParts(h [16]byte) (int, int, int, int) {
	p := func(i int) int {
		return int(int32(uint32(h[i]) | uint32(h[i+1])<<8 | uint32(h[i+2])<<16 | uint32(h[i+3])<<24)) //nolint:gosec // reinterpreting MD5 bytes as signed dwords, as MsiFileHash requires
	}
	return p(0), p(4), p(8), p(12)
}
