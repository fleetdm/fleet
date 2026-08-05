// Package msi builds the fleetd Windows installer (.msi) in pure Go,
// replacing the WiX v3 toolset (heat/candle/light run via the fleetdm/wix
// Docker image, native WiX, or Wine).
//
// This is NOT a general-purpose MSI or WiX implementation. It produces
// exactly one installer: the fleetd MSI previously described by the
// main.wxs template in orbit/pkg/packaging/windows_templates.go plus the
// heat.exe harvest of the orbit root directory. The output is designed to be
// equivalent to the WiX output byte for byte, modulo build-time
// nondeterminism (fresh ProductCode/PackageCode/component GUIDs, timestamps,
// and deflate encoder differences inside the CAB). Compatibility that must
// be preserved for upgrades in the field:
//
//   - UpgradeCode B681CB20-107E-428A-9B14-2D3C1AFED244 and the Upgrade table
//     semantics of <MajorUpgrade AllowDowngrades="yes"/>.
//   - The "Fleet osquery" service definition and custom actions driving
//     installer_utils.ps1.
//   - WiX-compatible identifier generation (dir/cmp/fil MD5 identifiers and
//     hashed 8.3 short names), so file keys and paths match what WiX-built
//     MSIs used.
//
// An MSI is a COM structured-storage (CFB) file containing a relational
// database (string pool + table streams), a summary-information property-set
// stream, and an embedded CAB archive with the payload. The pieces live in:
//
//   - cfb.go: CFB/compound-file container writer
//   - database.go: string pool and table stream encoding
//   - schema.go: fleetd table schemas and column definitions (_Validation)
//   - model.go: the fleetd installer model (tables, custom actions,
//     sequences) formerly authored in main.wxs
//   - harvest.go: directory walk replacing heat.exe (+ TransformHeat)
//   - cab.go: CAB (MSZIP) writer replicating WiX smart-cabbing
//   - summary.go: summary-information stream
//   - pe.go: version/language extraction from PE files (MsiGetFileVersion)
//   - wixid.go: WiX-compatible identifier and short-name generation
//
// The two custom-action DLLs in wixca/ (WixCA.dll for x86, used by x64
// installers, and WixCA_A64.dll for arm64) are the WixQuietExec64 /
// ServiceConfig custom-action binaries from WiX 3.14's WixUtilExtension,
// extracted verbatim from a fleetdm/wix build. They are embedded into the
// Binary table exactly as WiX did.
package msi
