package main

// This tool builds Orbit binaries with versioninfo information.
// https://learn.microsoft.com/en-us/windows/win32/menurc/versioninfo-resource

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/josephspurrier/goversioninfo"
	zlog "github.com/rs/zerolog/log"
)

func main() {
	// Input flags
	flagVersion := flag.String("version", "0.0.1", "Version string")
	flagCmdDir := flag.String("input", "", "Path to the directory containing the utility to build")
	flagOutputBinary := flag.String("output", "", "Path to the output binary")

	flag.Usage = func() {
		zlog.Fatal().Msgf("Usage: %s [flags]\n\nPossible flags:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	// checking input flags are not empty
	if *flagCmdDir == "" || *flagOutputBinary == "" {
		flag.Usage()
		os.Exit(1)
	}

	// check if flagCmdDir is a valid directory
	_, err := os.Stat(*flagCmdDir)
	if err != nil {
		zlog.Fatal().Err(err).Msg("invalid cmd-dir")
		flag.Usage()
		os.Exit(1)
	}

	// now we need to create the 'resource.syso' metadata file which contains versioninfo data

	// lets start with sanitizing the version data
	vParts, err := sanitizeVersion(*flagVersion)
	if err != nil {
		zlog.Fatal().Err(err).Msgf("invalid version: %s", *flagVersion)
		os.Exit(1)
	}

	// then we need to create the manifest.xml file
	manifestPath, err := writeManifestXML(vParts, *flagCmdDir)
	if err != nil {
		zlog.Fatal().Err(err).Msg("creating manifest.xml")
		os.Exit(1)
	}
	defer os.Remove(manifestPath)

	// now we can create the VersionInfo struct
	vi, err := createVersionInfo(vParts, manifestPath)
	if err != nil {
		zlog.Fatal().Err(err).Msg("parsing versioninfo")
		os.Exit(1)
	}

	// and finally we can write the 'resource.syso' file
	vi.Build()
	vi.Walk()

	// resource.syso is the resource file that is going to be picked up by golang compiler
	outPath := filepath.Join(*flagCmdDir, "resource.syso")
	if err := vi.WriteSyso(outPath, "amd64"); err != nil {
		zlog.Fatal().Err(err).Msg("creating syso file")
		os.Exit(1)
	}
	defer os.Remove(outPath)

	// now we can build the binary
	if err := buildTargetBinary(*flagCmdDir, *flagVersion, *flagOutputBinary); err != nil {
		zlog.Fatal().Err(err).Msg("error building binary")
		os.Exit(1)
	}

	zlog.Info().Msgf("Successfully built %s", *flagOutputBinary)
}

// createVersionInfo returns a VersionInfo struct pointer to be used to generate the 'resource.syso'
// metadata file (see writeResourceSyso).
func createVersionInfo(vParts []string, manifestPath string) (*goversioninfo.VersionInfo, error) {
	vIntParts := make([]int, 0, len(vParts))
	for _, p := range vParts {
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("error parsing version part %s: %w", p, err)
		}
		vIntParts = append(vIntParts, v)
	}
	version := strings.Join(vParts, ".")
	copyright := fmt.Sprintf("%d Fleet Device Management Inc.", time.Now().Year())

	// Taken from https://github.com/josephspurrier/goversioninfo/blob/master/testdata/resource/versioninfo.json
	langID, err := strconv.ParseUint("0409", 16, 16)
	if err != nil {
		return nil, errors.New("invalid LangID")
	}
	// Taken from https://github.com/josephspurrier/goversioninfo/blob/master/testdata/resource/versioninfo.json
	charsetID, err := strconv.ParseUint("04B0", 16, 16)
	if err != nil {
		return nil, errors.New("invalid charsetID")
	}

	result := goversioninfo.VersionInfo{
		FixedFileInfo: goversioninfo.FixedFileInfo{
			FileVersion: goversioninfo.FileVersion{
				Major: vIntParts[0],
				Minor: vIntParts[1],
				Patch: vIntParts[2],
				Build: vIntParts[3],
			},
			ProductVersion: goversioninfo.FileVersion{
				Major: vIntParts[0],
				Minor: vIntParts[1],
				Patch: vIntParts[2],
				Build: vIntParts[3],
			},
			FileFlagsMask: "3f",
			FileFlags:     "00",
			FileOS:        "040004",
			FileType:      "01",
			FileSubType:   "00",
		},
		StringFileInfo: goversioninfo.StringFileInfo{
			Comments:         "Fleet osquery",
			CompanyName:      "Fleet Device Management (fleetdm.com)",
			FileDescription:  "Orbit osquery runtime and autoupdater",
			FileVersion:      version,
			InternalName:     "",
			LegalCopyright:   copyright,
			LegalTrademarks:  "",
			OriginalFilename: "",
			PrivateBuild:     "",
			ProductName:      "Fleet osquery",
			ProductVersion:   version,
			SpecialBuild:     "",
		},
		VarFileInfo: goversioninfo.VarFileInfo{
			Translation: goversioninfo.Translation{
				LangID:    goversioninfo.LangID(langID),
				CharsetID: goversioninfo.CharsetID(charsetID),
			},
		},
		IconPath:     "",
		ManifestPath: manifestPath,
	}

	return &result, nil
}

// sanitizeVersion returns the version parts (Major, Minor, Patch and Build), filling the Build part
// with '0' if missing. Will error out if the version string is missing the Major, Minor or
// Patch part(s).
func sanitizeVersion(version string) ([]string, error) {
	vParts := strings.Split(version, ".")
	if len(vParts) < 3 {
		return nil, errors.New("invalid version string")
	}

	if len(vParts) < 4 {
		vParts = append(vParts, "0")
	}

	return vParts[:4], nil
}

// writeManifestXML creates the manifest.xml file used when generating the 'resource.syso' metadata
// (see writeResourceSyso). Returns the path of the newly created file.
func writeManifestXML(vParts []string, orbitPath string) (string, error) {
	filePath := filepath.Join(orbitPath, "manifest.xml")

	tmplOpts := struct {
		Version string
	}{
		Version: strings.Join(vParts, "."),
	}

	var contents bytes.Buffer
	if err := manifestXMLTemplate.Execute(&contents, tmplOpts); err != nil {
		return "", fmt.Errorf("parsing manifest.xml template: %w", err)
	}

	if err := ioutil.WriteFile(filePath, contents.Bytes(), constant.DefaultFileMode); err != nil {
		return "", fmt.Errorf("writing manifest.xml file: %w", err)
	}

	return filePath, nil
}

// Build the target binary for Windows
func buildTargetBinary(cmdDir string, version string, binaryPath string) error {
	var buildExec *exec.Cmd

	// convert relative to full output path
	outputBinary, err := filepath.Abs(binaryPath)
	if err != nil {
		return fmt.Errorf("error converting binary path to absolute: %w", err)
	}

	// check if cmdDir contains desktop
	// if it does, add -ldflags "-H=windowsgui" to exec.Command
	if strings.Contains(cmdDir, "desktop") {
		linkFlags := fmt.Sprintf("-H=windowsgui -X=main.version=%s", version)
		buildExec = exec.Command("go", "build", "-ldflags", linkFlags, "-o", outputBinary)
	} else {
		linkFlags := fmt.Sprintf("-X=github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=%s", version)
		buildExec = exec.Command("go", "build", "-ldflags", linkFlags, "-o", outputBinary)
	}
	buildExec.Env = append(os.Environ(), "GOOS=windows", "GOARCH=amd64")
	buildExec.Stderr = os.Stderr
	buildExec.Stdout = os.Stdout
	buildExec.Dir = cmdDir

	if err := buildExec.Run(); err != nil {
		return fmt.Errorf("compile orbit: %w", err)
	}
	return nil
}

// Adapted from https://github.com/josephspurrier/goversioninfo/blob/master/testdata/resource/goversioninfo.exe.manifest
var manifestXMLTemplate = template.Must(template.New("").Option("missingkey=error").Parse(
	`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity
    type="win32"
    name="Fleet osquery"
    version="{{.Version}}"
    processorArchitecture="*"/>
 <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
   <security>
     <requestedPrivileges>
       <requestedExecutionLevel
         level="asInvoker"
         uiAccess="false"/>
       </requestedPrivileges>
   </security>
 </trustInfo>
</assembly>`))
