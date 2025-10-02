// Package main is used to test applying several profiles and scripts on several teams.
// It allows setting secret variables on the Apple configuration and declaration profiles and in the scripts.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
)

func printfAndPrompt(format string, a ...any) {
	printf(format+" (press enter to proceed) ", a...)
	bufio.NewScanner(os.Stdin).Scan()
}

func printf(format string, a ...any) {
	fmt.Printf(time.Now().UTC().Format("2006-01-02T15:04:05Z")+": "+format, a...)
}

func main() {
	fleetURL := flag.String("fleet_url", "", "URL (with protocol and port of Fleet server)")
	apiToken := flag.String("api_token", "", "API authentication token to use on API calls")
	teamCount := flag.Int("team_count", 0, "Number of teams to create for the load test")
	cleanupTeams := flag.Bool("cleanup_teams", false, "Removes teams from previous runs and exits")
	secretVariable := flag.String("secret_variable", "", "Generate profiles and scripts with a given secret variable")

	flag.Parse()

	if *fleetURL == "" {
		log.Fatal("missing fleet_url argument")
	}
	if *apiToken == "" {
		log.Fatal("missing api_token argument")
	}
	apiClient, err := service.NewClient(*fleetURL, true, "", "")
	if err != nil {
		panic(err)
	}
	apiClient.SetToken(*apiToken)

	if *cleanupTeams {
		teams, err := apiClient.ListTeams("")
		if err != nil {
			log.Fatalf("list teams: %s", err)
		}
		printfAndPrompt("Cleaning up all teams...")
		for _, team := range teams {
			if strings.HasPrefix(team.Name, "Team ") {
				if err := apiClient.DeleteTeam(team.ID); err != nil {
					log.Fatalf("delete team %s: %s", team.Name, err)
				}
			}
		}
		return
	}

	if *teamCount == 0 {
		log.Fatal("missing team_count argument")
	}

	printfAndPrompt("Creating %d teams...", *teamCount)

	var teams []*fleet.Team
	for t := 0; t < *teamCount; t++ {
		team, err := apiClient.CreateTeam(fleet.TeamPayload{
			Name: ptr.String(fmt.Sprintf("Team %d", t)),
		})
		if err != nil {
			log.Fatalf("team create: %s", err)
		}
		teams = append(teams, team)
	}

	appleRandomDeclarationProfiles := createRandomDeclarations(13, *secretVariable)
	appleConfigurationProfiles = addSecretVariable(appleConfigurationProfiles, *secretVariable)

	allProfiles := slices.Concat(
		appleConfigurationProfiles,     // 10
		windowsConfigurationProfiles,   // 5
		appleDeclarationProfiles,       // 2
		appleRandomDeclarationProfiles, // 13
	)
	printfAndPrompt("Add %d profiles to all teams...", len(allProfiles))

	for _, team := range teams {
		printf("Applying profiles to team %s...\n", team.Name)
		if err := apiClient.ApplyTeamProfiles(team.Name, allProfiles, fleet.ApplyTeamSpecOptions{}); err != nil {
			log.Fatalf("apply profiles to team %s: %s", team.Name, err)
		}
	}

	printfAndPrompt("Add scripts to all teams...")

	for _, team := range teams {
		scripts := createRandomScripts(team.ID, 30, *secretVariable)
		printf("Applying %d scripts to team %s...\n", len(scripts), team.Name)
		if _, err := apiClient.ApplyTeamScripts(team.Name, scripts, fleet.ApplySpecOptions{}); err != nil {
			log.Fatalf("apply scripts to team %s: %s", team.Name, err)
		}
	}
}

func addSecretVariable(profiles []fleet.MDMProfileBatchPayload, secretVariable string) []fleet.MDMProfileBatchPayload {
	s := "test"
	if secretVariable != "" {
		s = fmt.Sprintf("$FLEET_SECRET_%s", secretVariable)
	}
	for i := range profiles {
		profiles[i].Contents = []byte(fmt.Sprintf(string(profiles[i].Contents), s))
	}
	return profiles
}

func createRandomScripts(j uint, n int, secretVariable string) []fleet.ScriptPayload {
	var scripts []fleet.ScriptPayload
	s := ""
	if secretVariable != "" {
		s = fmt.Sprintf("${FLEET_SECRET_%s}", secretVariable)
	}
	for i := 0; i < n; i++ {
		scripts = append(scripts, fleet.ScriptPayload{
			Name: fmt.Sprintf("script_%d.sh", i),
			ScriptContents: []byte(fmt.Sprintf(`#!/bin/bash

echo "%s"
echo "Team %d"
echo "Script %d"

# macOS Compatibility universal extension installer script
# Downloads and installs the latest macos_compatibility_universal.ext from GitHub
# Safe for deployment via Fleet (schedules orbit restart to avoid script termination)
#
# Usage:
#   sudo ./install_macos_compatibility_extension.sh          # Default: scheduled restart (Fleet-safe)
#   sudo ./install_macos_compatibility_extension.sh immediate # Immediate restart (manual execution)

set -e  # Exit on any error

# Variables
GITHUB_REPO="allenhouchins/fleet-extensions"
EXTENSION_DIR="/var/fleet/extensions"
OSQUERY_DIR="/var/osquery"
EXTENSIONS_LOAD_FILE="$OSQUERY_DIR/extensions.load"
EXTENSION_NAME="macos_compatibility.ext"
EXTENSION_PATH="$EXTENSION_DIR/$EXTENSION_NAME"
BACKUP_PATH="$EXTENSION_PATH.backup"

# Command line options
IMMEDIATE_RESTART=${1:-false}  # Pass "immediate" as first argument for immediate restart

echo "Starting macOS Compatibility Extension installation..."

# Function to log messages with timestamp
log() {
    echo "foobar $1"
}

# Function to check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log "Error: This script must be run as root (use sudo)"
        exit 1
    fi
}

# Function to check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check if curl is available
    if ! command -v curl &> /dev/null; then
        log "Error: curl is required but not installed"
        exit 1
    fi
    
    log "Prerequisites check completed"
}

# Function to create directory with proper ownership
create_directory() {
    local dir="$1"
    if [[ ! -d "$dir" ]]; then
        log "Creating directory: $dir"
        mkdir -p "$dir"
        chown root:wheel "$dir"
        chmod 755 "$dir"
        log "Directory created with proper permissions"
    else
        log "Directory already exists: $dir"
        # Ensure proper ownership even if directory exists
        chown root:wheel "$dir"
        chmod 755 "$dir"
    fi
}

# Function to backup existing extension
backup_existing() {
    if [[ -f "$EXTENSION_PATH" ]]; then
        log "Backing up existing extension to: $BACKUP_PATH"
        cp "$EXTENSION_PATH" "$BACKUP_PATH"
        log "Backup completed"
    fi
}

# Function to get the latest release tag from GitHub
get_latest_release_tag() {
    log "Finding latest release tag..."
    
    # Try to get the latest release page and extract the actual tag
    local releases_url="https://github.com/$GITHUB_REPO/releases/latest"
    local response
    
    if ! response=$(curl -s -L "$releases_url"); then
        log "Error: Failed to fetch releases page"
        return 1
    fi
    
    # Extract the actual tag from the redirected URL or page content
    # Look for the tag in the URL path or in the page content
    local tag
    tag=$(echo "$response" | grep -o 'releases/tag/[^"]*' | head -1 | sed 's|releases/tag/||' | sed 's|".*||')
    
    if [[ -z "$tag" ]]; then
        # Alternative: look for version tags in the page content
        tag=$(echo "$response" | grep -o 'tag/[v0-9][^"]*' | head -1 | sed 's|tag/||' | sed 's|".*||')
    fi
    
    if [[ -z "$tag" ]]; then
        log "Error: Could not determine latest release tag"
        return 1
    fi
    
    log "Found latest release tag: $tag"
    echo "$tag"
}

# Function to construct download URL with specific tag
get_download_url_with_tag() {
    local tag="$1"
    local download_url="https://github.com/$GITHUB_REPO/releases/download/$tag/$EXTENSION_NAME"
    echo "$download_url"
}

# Function to validate downloaded file
validate_download() {
    local file_path="$1"
    
    log "Validating downloaded file..."
    
    # Check if file exists and is not empty
    if [[ ! -f "$file_path" ]]; then
        log "Error: Downloaded file not found"
        return 1
    fi
    
    if [[ ! -s "$file_path" ]]; then
        log "Error: Downloaded file is empty"
        return 1
    fi
    
    # Check if file is executable format (basic check)
    local file_type
    file_type=$(file "$file_path" 2>/dev/null || echo "unknown")
    log "File type: $file_type"
    
    # For macOS, check if it's a Mach-O executable
    if [[ "$file_type" == *"Mach-O"* ]] || [[ "$file_type" == *"executable"* ]]; then
        log "File validation passed"
        return 0
    else
        log "Warning: File may not be a valid executable. Proceeding anyway..."
        return 0
    fi
}

# Function to download the latest release
download_latest_release() {
    log "Starting download process..."
    
    # Create temporary file for download
    local temp_file
    temp_file=$(mktemp)
    
    # First, try the direct latest download URL
    local direct_url="https://github.com/$GITHUB_REPO/releases/latest/download/$EXTENSION_NAME"
    log "Attempting direct download from: $direct_url"
    
    if curl -L --progress-bar --fail -o "$temp_file" "$direct_url" 2>/dev/null; then
        log "Direct download successful"
    else
        log "Direct download failed, getting actual release tag..."
        
        # Get the actual latest release tag
        local latest_tag
        if ! latest_tag=$(get_latest_release_tag); then
            log "Error: Could not determine latest release tag"
            rm -f "$temp_file"
            exit 1
        fi
        
        # Construct download URL with the actual tag
        local download_url
        download_url=$(get_download_url_with_tag "$latest_tag")
        log "Download URL with tag: $download_url"
        
        # Download with the specific tag
        if curl -L --progress-bar --fail -o "$temp_file" "$download_url"; then
            log "Download with specific tag successful"
        else
            log "Error: Download failed with both methods"
            log "Please verify that '$EXTENSION_NAME' exists in the latest release at:"
            log "https://github.com/$GITHUB_REPO/releases/latest"
            rm -f "$temp_file"
            exit 1
        fi
    fi
    
    # Validate the download
    if validate_download "$temp_file"; then
        # Move to final location
        mv "$temp_file" "$EXTENSION_PATH"
        log "File moved to final location: $EXTENSION_PATH"
    else
        log "Error: File validation failed"
        rm -f "$temp_file"
        exit 1
    fi
}

# Function to make the extension executable and set proper ownership
setup_file_permissions() {
    log "Setting up file permissions..."
    chown root:wheel "$EXTENSION_PATH"
    chmod 755 "$EXTENSION_PATH"
    log "File permissions configured (owner: root:wheel, mode: 755)"
}

# Function to handle extensions.load file
setup_extensions_load() {
    log "Configuring extensions.load file..."
    
    # Create osquery directory if it doesn't exist
    if [[ ! -d "$OSQUERY_DIR" ]]; then
        log "Creating osquery directory: $OSQUERY_DIR"
        mkdir -p "$OSQUERY_DIR"
        chown root:wheel "$OSQUERY_DIR"
        chmod 755 "$OSQUERY_DIR"
    fi
    
    # Check if extensions.load file exists
    if [[ -f "$EXTENSIONS_LOAD_FILE" ]]; then
        log "extensions.load file exists, checking for existing entry..."
        
        # Remove any existing entries for this extension (handle duplicates)
        if grep -q "$EXTENSION_PATH" "$EXTENSIONS_LOAD_FILE"; then
            log "Removing existing entries for this extension..."
            # Create temp file without the extension path
            grep -v "$EXTENSION_PATH" "$EXTENSIONS_LOAD_FILE" > "$EXTENSIONS_LOAD_FILE.tmp" || true
            mv "$EXTENSIONS_LOAD_FILE.tmp" "$EXTENSIONS_LOAD_FILE"
        fi
        
        # Add the extension path
        echo "$EXTENSION_PATH" >> "$EXTENSIONS_LOAD_FILE"
        log "Extension path added to extensions.load"
    else
        log "Creating extensions.load file..."
        echo "$EXTENSION_PATH" > "$EXTENSIONS_LOAD_FILE"
        chown root:wheel "$EXTENSIONS_LOAD_FILE"
        chmod 644 "$EXTENSIONS_LOAD_FILE"
        log "extensions.load file created"
    fi
}

# Function to remove old extension and its reference from extensions.load
remove_old_extension() {
    local old_extension_name="macos_compatibility_universal.ext"
    local old_extension_path="$EXTENSION_DIR/$old_extension_name"
    
    # Remove old extension file if it exists
    if [[ -f "$old_extension_path" ]]; then
        log "Removing old extension file: $old_extension_path"
        rm -f "$old_extension_path"
    fi
    
    # Remove old extension reference from extensions.load if it exists
    if [[ -f "$EXTENSIONS_LOAD_FILE" ]]; then
        if grep -q "$old_extension_path" "$EXTENSIONS_LOAD_FILE"; then
            log "Removing old extension reference from extensions.load"
            grep -v "$old_extension_path" "$EXTENSIONS_LOAD_FILE" > "$EXTENSIONS_LOAD_FILE.tmp" || true
            mv "$EXTENSIONS_LOAD_FILE.tmp" "$EXTENSIONS_LOAD_FILE"
        fi
    fi
}

# Function to schedule orbit restart using detached child process or restart immediately
handle_orbit_restart() {
    if [[ "$IMMEDIATE_RESTART" == "immediate" ]]; then
        log "Immediate restart requested - restarting orbit service now..."
        restart_orbit_immediate
    else
        log "Scheduling orbit service restart (safe for Fleet deployment)..."
        schedule_orbit_restart
    fi
}

# Function to restart orbit immediately 
restart_orbit_immediate() {
    log "Restarting orbit service immediately..."
    launchctl kickstart -k system/com.fleetdm.orbit
    log "Orbit service restart command executed"
}

# Function to schedule orbit restart using detached child process
schedule_orbit_restart() {
    log "Scheduling orbit restart in 10 seconds (detached process method)..."
    
    # Start detached child process that will handle the restart
    bash -c "bash $0 __restart_orbit >/dev/null 2>/dev/null </dev/null &"
    
    log "Orbit restart scheduled for 10 seconds after script completion"
    log "Check /var/log/macos_compatibility_installer.log for restart status"
}

# Function to handle the detached restart process
handle_detached_restart() {
    # This runs in the detached child process
    echo "Starting detached restart process..." >> /var/log/macos_compatibility_installer.log 2>&1
    
    # Wait for parent process to complete and report success
    sleep 10
    
    echo "Executing orbit restart..." >> /var/log/macos_compatibility_installer.log 2>&1
    launchctl kickstart -k system/com.fleetdm.orbit >> /var/log/macos_compatibility_installer.log 2>&1
    echo "Orbit restart command executed" >> /var/log/macos_compatibility_installer.log 2>&1
}

# Function to cleanup on failure
cleanup_on_failure() {
    log "Cleaning up due to failure..."
    
    # Remove the downloaded extension if it exists
    if [[ -f "$EXTENSION_PATH" ]]; then
        rm -f "$EXTENSION_PATH"
        log "Removed failed installation file"
    fi
    
    # Restore backup if it exists
    if [[ -f "$BACKUP_PATH" ]]; then
        mv "$BACKUP_PATH" "$EXTENSION_PATH"
        log "Restored previous version from backup"
    fi
}

# Trap to handle errors
trap cleanup_on_failure ERR

# Main execution
main() {
    # Handle detached restart process
    if [[ "$1" == "__restart_orbit" ]]; then
        handle_detached_restart
        exit 0
    fi
    
    log "=== macOS Compatibility Extension Installer Started ==="
    if [[ "$IMMEDIATE_RESTART" == "immediate" ]]; then
        log "Mode: Immediate restart (manual execution)"
    else
        log "Mode: Scheduled restart (safe for Fleet deployment)"
    fi
    
    # Ensure log directory exists for background process
    mkdir -p /var/log
    
    check_root
    check_prerequisites
    
    # Create the extensions directory
    create_directory "$EXTENSION_DIR"
    
    # Remove old extension and its reference before proceeding
    remove_old_extension
    
    # Backup existing extension
    backup_existing
    
    # Download the latest release
    download_latest_release
    
    # Set up file permissions
    setup_file_permissions
    
    # Setup extensions.load file
    setup_extensions_load
    
    # Handle orbit restart (scheduled for Fleet deployment, immediate for manual)
    handle_orbit_restart
    
    # Clean up backup on success
    if [[ -f "$BACKUP_PATH" ]]; then
        log "Removing backup file (installation successful)"
        rm -f "$BACKUP_PATH"
    fi
    
    log "=== Installation completed successfully! ==="
    log "Extension installed at: $EXTENSION_PATH"
    log "Extensions configuration: $EXTENSIONS_LOAD_FILE"
    if [[ "$IMMEDIATE_RESTART" == "immediate" ]]; then
        log "Orbit service has been restarted immediately"
    else
        log "Orbit service restart has been scheduled for 10 seconds"
    fi
    echo ""
}

# Run the main function
main "$@"`, s, j, i)),
		})
	}
	return scripts
}

func createRandomDeclarations(n int, secretVariable string) []fleet.MDMProfileBatchPayload {
	s := "foobar"
	if secretVariable != "" {
		s = fmt.Sprintf("${FLEET_SECRET_%s}", secretVariable)
	}
	tmpl := `
{
	"Type": "com.apple.configuration.decl%d",
	"Identifier": "com.fleet.config%d",
	"Payload": {
		"ServiceType": "com.apple.bash",
		"DataAssetReference": "com.fleet.asset.bash",
    "Foobar": "%s" %s
	}
}`

	newDeclBytes := func(i int, payload ...string) []byte {
		var p string
		if len(payload) > 0 {
			p = "," + strings.Join(payload, ",")
		}
		return []byte(fmt.Sprintf(tmpl, i, i, s, p))
	}

	var decls []fleet.MDMProfileBatchPayload
	for i := 0; i < n; i++ {
		decls = append(decls, fleet.MDMProfileBatchPayload{
			Name:     fmt.Sprintf("Declaration %d", i),
			Contents: newDeclBytes(i),
		})
	}
	return decls
}

var appleConfigurationProfiles = []fleet.MDMProfileBatchPayload{
	{
		Name: "Ensure Install Security Responses and System Files Is Enabled",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.SoftwareUpdate</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-1.6.check</string>
                        <key>PayloadUUID</key>
                        <string>0D8F676A-A705-4F57-8FF8-3118360EFDEB</string>
                        <key>ConfigDataInstall</key>
                        <true/>
                        <key>CriticalUpdateInstall</key>
                        <true/>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>%s</string>
        <key>PayloadDisplayName</key>
        <string>Ensure Install Security Responses and System Files Is Enabled</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.cis-1.6</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>EBEE9B81-9D33-477F-AFBE-9691360B7A74</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
	{
		Name: "Ensure Software Update Deferment Is Less Than or Equal to 30 Days",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.applicationaccess</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-1.7.check</string>
                        <key>PayloadUUID</key>
                        <string>123FD592-D1C3-41FD-BC41-F91F3E1E2CF4</string>
                        <key>enforcedSoftwareUpdateDelay</key>
                        <integer>29</integer>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>%s</string>
        <key>PayloadDisplayName</key>
        <string>Ensure Software Update Deferment Is Less Than or Equal to 30 Days</string>
        <key>PayloadIdentifier</key>
        <string>com.zwass.cis-1.7</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>385A0C13-2472-41B3-851C-1311FA12EB49</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
	{
		Name: "Ensure Auto Update Is Enabled",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.SoftwareUpdate</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-1.2.check</string>
                        <key>PayloadUUID</key>
                        <string>4DC539B5-837E-4DC3-B60B-43A8C556A8F0</string>
                        <key>AutomaticCheckEnabled</key>
                        <true/>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>%s</string>
        <key>PayloadDisplayName</key>
        <string>Ensure Auto Update Is Enabled</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.cis-1.2</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>03E69A02-02CE-4CA0-8F17-3BAAD5D3852F</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
	{
		Name: "Ensure Download New Updates When Available Is Enabled",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.SoftwareUpdate</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-1.3.check</string>
                        <key>PayloadUUID</key>
                        <string>5FDE6D58-79CD-447A-AFB0-BA32D889C396</string>
                        <key>AutomaticDownload</key>
                        <true/>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>%s</string>
        <key>PayloadDisplayName</key>
        <string>Ensure Download New Updates When Available Is Enabled</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.cis-1.3</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>0A1C2F97-D6FA-4CDB-ABB6-47DF2B151F4F</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
	{
		Name: "Ensure Install of macOS Updates Is Enabled",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>%s</string>
                        <key>PayloadType</key>
                        <string>com.apple.SoftwareUpdate</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-1.4.check</string>
                        <key>PayloadUUID</key>
                        <string>15BF7634-276A-411B-8C4E-52D89B4ED82C</string>
                        <key>AutomaticallyInstallMacOSUpdates</key>
                        <true/>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>test</string>
        <key>PayloadDisplayName</key>
        <string>Ensure Install of macOS Updates Is Enabled</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.cis-1.4</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>7DB8733E-BD11-4E88-9AE0-273EF2D0974B</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
	{
		Name: "Ensure Firewall Logging Is Enabled and Configured",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.security.firewall</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-3.6.check</string>
                        <key>PayloadUUID</key>
                        <string>604D8218-D7B6-43B1-95E6-DFCA4C25D73D</string>
                        <key>EnableFirewall</key>
                        <true/>
                        <key>EnableLogging</key>
                        <true/>
                        <key>LoggingOption</key>
                        <string>detail</string>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>%s</string>
        <key>PayloadDisplayName</key>
        <string>Ensure Firewall Logging Is Enabled and Configured</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.cis-3.6</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>5E27501E-50DF-4804-9DEC-0E63C34E8831</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
	{
		Name: "Ensure Bonjour Advertising Services Is Disabled",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.mDNSResponder</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-4.1.check</string>
                        <key>PayloadUUID</key>
                        <string>08FEA43B-CE9B-4098-804C-11459D109992</string>
                        <key>NoMulticastAdvertisements</key>
                        <true/>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>%s</string>
        <key>PayloadDisplayName</key>
        <string>Ensure Bonjour Advertising Services Is Disabled</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.cis-4.1</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>25BD1312-2B79-40C7-99FA-E60B49A1883E</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
	{
		Name: "Disable Bluetooth sharing",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>

        <key>PayloadDescription</key>
        <string>%s</string>
        <key>PayloadDisplayName</key>
        <string>Disable Bluetooth sharing</string>
        <key>PayloadEnabled</key>
        <true/>
        <key>PayloadIdentifier</key>
        <string>cis.macOSBenchmark.section2.BluetoothSharing</string>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>5CEBD712-28EB-432B-84C7-AA28A5A383D8</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
    <key>PayloadRemovalDisallowed</key>
    <true/>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadContent</key>
                        <dict>
                                <key>com.apple.Bluetooth</key>
                                <dict>
                                        <key>Forced</key>
                                        <array>
                                                <dict>
                                                        <key>mcx_preference_settings</key>
                                                        <dict>
                                                                <key>PrefKeyServicesEnabled</key>
                                                                <false/>
                                                        </dict>
                                                </dict>
                                        </array>
                                </dict>
                        </dict>
                        <key>PayloadDescription</key>
                        <string>Disables Bluetooth Sharing</string>
                        <key>PayloadDisplayName</key>
                        <string>Custom</string>
                        <key>PayloadEnabled</key>
                        <true/>
                        <key>PayloadIdentifier</key>
                        <string>0240DD1C-70DC-4766-9018-04322BFEEAD1</string>
                        <key>PayloadType</key>
                        <string>com.apple.ManagedClient.preferences</string>
                        <key>PayloadUUID</key>
                        <string>0240DD1C-70DC-4766-9018-04322BFEEAD1</string>
                        <key>PayloadVersion</key>
                        <integer>1</integer>
                </dict>
        </array>
</dict>
</plist>`),
	},
	{
		Name: "Ensure Install Application Updates from the App Store Is Enabled",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.SoftwareUpdate</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-1.5.check</string>
                        <key>PayloadUUID</key>
                        <string>6B0285F8-5DB8-4F68-AA6E-2333CCD6CE04</string>
                        <key>AutomaticallyInstallAppUpdates</key>
                        <true/>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>%s</string>
        <key>PayloadDisplayName</key>
        <string>Ensure Install Application Updates from the App Store Is Enabled</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.cis-1.5</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>1C4C0EC4-64A7-4AF0-8807-A3DD44A6DC76</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
	{
		Name: "Disable iCloud Drive storage solution usage",
		Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
        <key>PayloadContent</key>
        <array>
                <dict>
                        <key>PayloadDisplayName</key>
                        <string>test</string>
                        <key>PayloadType</key>
                        <string>com.apple.applicationaccess</string>
                        <key>PayloadIdentifier</key>
                        <string>com.fleetdm.cis-2.1.1.2.check-disable</string>
                        <key>PayloadUUID</key>
                        <string>1028E002-9AFE-446A-84E0-27DA5DA39B4A</string>
                        <key>allowCloudDocumentSync</key>
                        <false/>
                </dict>
        </array>
        <key>PayloadDescription</key>
        <string>%s</string>
        <key>PayloadDisplayName</key>
        <string>Disable iCloud Drive storage solution usage</string>
        <key>PayloadIdentifier</key>
        <string>com.fleetdm.cis-2.1.1.2-disable</string>
        <key>PayloadRemovalDisallowed</key>
        <false/>
        <key>PayloadScope</key>
        <string>System</string>
        <key>PayloadType</key>
        <string>Configuration</string>
        <key>PayloadUUID</key>
        <string>7B3DE4EA-0AFA-44F5-9716-37526EE441EA</string>
        <key>PayloadVersion</key>
        <integer>1</integer>
</dict>
</plist>`),
	},
}

var windowsConfigurationProfiles = []fleet.MDMProfileBatchPayload{
	{
		Name: "Advanced PowerShell logging",
		Contents: []byte(`<Replace>
  <!-- Powershell configs, modify the admx keys as needed for your env -->
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">chr</Format>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/WindowsPowerShell/TurnOnPowerShellScriptBlockLogging</LocURI>
    </Target>
    <Data>
      <![CDATA[<enabled/><data id="ExecutionPolicy" value="AllSigned"/>
      <data id="Listbox_ModuleNames" value="*"/>
      <data id="OutputDirectory" value="false"/>
      <data id="EnableScriptBlockInvocationLogging" value="true"/>
      <data id="SourcePathForUpdateHelp" value="false"/>]]>
    </Data>
  </Item>
</Replace>`),
	},
	{
		Name: "Disable Guest account",
		Contents: []byte(`<Replace>
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">int</Format>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/LocalPoliciesSecurityOptions/Accounts_EnableGuestAccountStatus</LocURI>
    </Target>
      <Data>0</Data>
  </Item>
</Replace>`),
	},
	{
		Name: "Disable OneDrive",
		Contents: []byte(`<Replace>
  <!-- Disable OneDrive -->
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">int</Format>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/System/DisableOneDriveFileSync</LocURI>
    </Target>
    <Data>1</Data>
  </Item>
</Replace>`),
	},
	{
		Name: "Enable firewall",
		Contents: []byte(`<Replace>
  <!-- Enable Firewall for Domain Profile -->
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">bool</Format>
    </Meta>
    <Target>
      <LocURI>./Vendor/MSFT/Firewall/MdmStore/DomainProfile/EnableFirewall</LocURI>
    </Target>
    <Data>true</Data>
  </Item>
</Replace>
<Replace>
  <!-- Disable ability for user to override at domain level  -->
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">bool</Format>
    </Meta>
    <Target>
      <LocURI>./Vendor/MSFT/Firewall/MdmStore/DomainProfile/AllowLocalPolicyMerge</LocURI>
    </Target>
    <Data>false</Data>
  </Item>
</Replace>
<Replace>
  <!-- Enable Firewall for Private Profile -->
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">bool</Format>
    </Meta>
    <Target>
      <LocURI>./Vendor/MSFT/Firewall/MdmStore/PrivateProfile/EnableFirewall</LocURI>
    </Target>
    <Data>true</Data>
  </Item>
</Replace>
<Replace>
  <!-- Disable ability for user to override at private profile level  -->
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">bool</Format>
    </Meta>
    <Target>
      <LocURI>./Vendor/MSFT/Firewall/MdmStore/PrivateProfile/AllowLocalPolicyMerge</LocURI>
    </Target>
    <Data>false</Data>
  </Item>
</Replace>
<Replace>
  <!-- Enable Firewall for Public Profile -->
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">bool</Format>
    </Meta>
    <Target>
      <LocURI>./Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall</LocURI>
    </Target>
    <Data>true</Data>
  </Item>
</Replace>
<Replace>
  <!-- Disable ability for user to override at public profile level  -->
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">bool</Format>
    </Meta>
    <Target>
      <LocURI>./Vendor/MSFT/Firewall/MdmStore/PublicProfile/AllowLocalPolicyMerge</LocURI>
    </Target>
    <Data>false</Data>
  </Item>
</Replace>`),
	},
	{
		Name: "Password settings",
		Contents: []byte(`<Replace>
  <!-- Enforce minimum password length (10 characters) -->
  <CmdID>1</CmdID>
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">int</Format>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/DeviceLock/MinimumPasswordLength</LocURI>
    </Target>
    <Data>10</Data>
  </Item>
</Replace>

<Replace>
  <!-- Enforce password complexity -->
  <CmdID>2</CmdID>
  <Item>
    <Meta>
      <Format xmlns="syncml:metinf">int</Format>
    </Meta>
    <Target>
      <LocURI>./Device/Vendor/MSFT/Policy/Config/DeviceLock/PasswordComplexity</LocURI>
    </Target>
    <Data>1</Data>
  </Item>
</Replace>`),
	},
}

var appleDeclarationProfiles = []fleet.MDMProfileBatchPayload{
	{
		Name: "Passcode Settings",
		Contents: []byte(`{
    "Type": "com.apple.configuration.passcode.settings",
    "Identifier": "956e0d14-6019-479b-a6f9-a69ef77668c5",
    "Payload": {
        "MaximumInactivityInMinutes": 5,
        "MinimumLength": 6,
        "RequireComplexPasscode": true
    }
}`),
	},
	{
		Name: "Software update settings",
		Contents: []byte(`{
    "Type": "com.apple.configuration.softwareupdate.settings",
    "Identifier": "com.fleetdm.config.softwareupdate.settings",
    "Payload": {
        "AutomaticActions": {
            "Download": "AlwaysOn",
            "InstallOSUpdates": "Allowed",
            "InstallSecurityUpdate": "AlwaysOn"
        },
        "Notifications": true,
        "RapidSecurityResponse": {
            "Enabled": true
        }
    }
}`),
	},
}
