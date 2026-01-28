# PowerShell script to download and install Cloudflare WARP on Windows 11
# For use with Fleet MDM

# Set strict mode for better error handling
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# Define variables
$downloadUrl = "https://downloads.cloudflareclient.com/v1/download/windows/ga"
$tempDir = $env:TEMP
$installerPath = Join-Path $tempDir "Cloudflare_WARP.msi"
$organization = "your-team-name"   # Replace with your Cloudflare Zero Trust organization name
$serviceMode = "1dot1"             # Gateway with DoH mode (options: warp, 1dot1, proxy, postureonly, tunnelonly)
$autoConnect = 2                   # Auto-reconnect after N minutes (0 = indefinite off, 1-1440 = minutes)
$displayName = "display-name"      # Organization display name in WARP GUI
$onboarding = $false               # Show privacy policy screens on first launch
$switchLocked = $true              # Prevent users from manually disabling WARP

# Function to write log messages
function Write-Log {
    param([string]$Message)
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Output "[$timestamp] $Message"
}

try {
    Write-Log "Starting Cloudflare WARP installation process..."

    # Check if running with administrator privileges
    $isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    if (-not $isAdmin) {
        Write-Log "ERROR: This script must be run as Administrator"
        exit 1
    }

    Write-Log "Downloading Cloudflare WARP from: $downloadUrl"
    Write-Log "Download location: $installerPath"

    # Download the installer
    Invoke-WebRequest -Uri $downloadUrl -OutFile $installerPath -UseBasicParsing

    # Verify the file was downloaded
    if (-not (Test-Path $installerPath)) {
        Write-Log "ERROR: Failed to download installer"
        exit 1
    }

    $fileSize = (Get-Item $installerPath).Length / 1MB
    Write-Log "Download complete. File size: $([math]::Round($fileSize, 2)) MB"

    # Install silently with MSI parameters
    Write-Log "Starting silent installation with organization: $organization"
    Write-Log "Installing with MSI parameters: ORGANIZATION, SERVICE_MODE, ONBOARDING, SWITCH_LOCKED"
    $arguments = @(
        "/i"
        "`"$installerPath`""
        "/qn"
        "ORGANIZATION=`"$organization`""
        "SERVICE_MODE=`"$serviceMode`""
        "ONBOARDING=$($onboarding.ToString().ToUpper())"
        "SWITCH_LOCKED=$($switchLocked.ToString().ToUpper())"
        "/norestart"
        "/L*V"
        "`"$tempDir\CloudflareWARP_install.log`""
    )

    $process = Start-Process -FilePath "msiexec.exe" -ArgumentList $arguments -Wait -PassThru -NoNewWindow

    # Check installation result
    if ($process.ExitCode -eq 0) {
        Write-Log "Installation completed successfully"
    } elseif ($process.ExitCode -eq 3010) {
        Write-Log "Installation completed successfully (reboot required)"
    } else {
        Write-Log "ERROR: Installation failed with exit code: $($process.ExitCode)"
        Write-Log "Check installation log at: $tempDir\CloudflareWARP_install.log"
        exit $process.ExitCode
    }

    # Update MDM configuration with additional parameters
    Write-Log "Updating MDM configuration file with additional parameters..."
    $mdmXmlPath = "C:\ProgramData\Cloudflare\mdm.xml"

    # Wait for the MSI to create the initial mdm.xml file
    Start-Sleep -Seconds 3

    try {
        if (Test-Path $mdmXmlPath) {
            # Read and parse existing mdm.xml
            [xml]$mdmContent = Get-Content $mdmXmlPath
            $dictNode = $mdmContent.dict

            # Create new elements for auto_connect
            $autoConnectKey = $mdmContent.CreateElement("key")
            $autoConnectKey.InnerText = "auto_connect"
            $autoConnectValue = $mdmContent.CreateElement("integer")
            $autoConnectValue.InnerText = $autoConnect.ToString()

            # Create new elements for display_name
            $displayNameKey = $mdmContent.CreateElement("key")
            $displayNameKey.InnerText = "display_name"
            $displayNameValue = $mdmContent.CreateElement("string")
            $displayNameValue.InnerText = $displayName

            # Append new elements to the dict node
            $dictNode.AppendChild($autoConnectKey) | Out-Null
            $dictNode.AppendChild($autoConnectValue) | Out-Null
            $dictNode.AppendChild($displayNameKey) | Out-Null
            $dictNode.AppendChild($displayNameValue) | Out-Null

            # Save the updated XML
            $mdmContent.Save($mdmXmlPath)
            Write-Log "Added auto_connect and display_name to mdm.xml"
            Write-Log "MDM configuration updated successfully at $mdmXmlPath"
        } else {
            # If mdm.xml doesn't exist, create a complete one
            Write-Log "mdm.xml not found, creating new configuration file..."
            $xmlContent = @"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>organization</key>
    <string>$organization</string>
    <key>service_mode</key>
    <string>$serviceMode</string>
    <key>onboarding</key>
    <$(if ($onboarding) { "true" } else { "false" })/>
    <key>switch_locked</key>
    <$(if ($switchLocked) { "true" } else { "false" })/>
    <key>auto_connect</key>
    <integer>$autoConnect</integer>
    <key>display_name</key>
    <string>$displayName</string>
</dict>
</plist>
"@
            # Ensure directory exists
            $mdmDir = Split-Path $mdmXmlPath -Parent
            if (-not (Test-Path $mdmDir)) {
                New-Item -ItemType Directory -Path $mdmDir -Force | Out-Null
            }

            # Write the XML file
            $xmlContent | Out-File -FilePath $mdmXmlPath -Encoding UTF8 -Force
            Write-Log "MDM configuration created successfully at $mdmXmlPath"
        }
    } catch {
        Write-Log "ERROR: Failed to update mdm.xml: $($_.Exception.Message)"
        Write-Log "WARP is installed but additional parameters (auto_connect, display_name) were not configured"
    }

    # Verify MDM configuration
    Write-Log "Verifying MDM configuration parameters..."

    # Define expected values upfront for use in catch block
    $expectedOnboarding = if ($onboarding) { "true" } else { "false" }
    $expectedSwitchLocked = if ($switchLocked) { "true" } else { "false" }

    if (Test-Path $mdmXmlPath) {
        try {
            [xml]$mdmContent = Get-Content $mdmXmlPath
            $dictNode = $mdmContent.dict

            # Build parameter lookup table
            $params = @{}
            $keys = @($dictNode.key)
            $allValues = $dictNode.ChildNodes | Where-Object { $_.Name -ne "key" }

            for ($i = 0; $i -lt $keys.Count; $i++) {
                $keyName = $keys[$i]
                # Find the next non-key element after this key
                $valueIndex = 0
                $keysSoFar = 0
                foreach ($node in $dictNode.ChildNodes) {
                    if ($node.Name -eq "key") {
                        if ($keysSoFar -eq $i) {
                            # Found our key, next non-key node is the value
                            foreach ($nextNode in $dictNode.ChildNodes) {
                                if ($nextNode -eq $node) {
                                    continue
                                }
                                $currentIndex = [array]::IndexOf($dictNode.ChildNodes, $nextNode)
                                $keyIndex = [array]::IndexOf($dictNode.ChildNodes, $node)
                                if ($currentIndex -gt $keyIndex -and $nextNode.Name -ne "key") {
                                    if ($nextNode.Name -eq "true") {
                                        $params[$keyName] = "true"
                                    } elseif ($nextNode.Name -eq "false") {
                                        $params[$keyName] = "false"
                                    } else {
                                        $params[$keyName] = $nextNode.InnerText
                                    }
                                    break
                                }
                            }
                            break
                        }
                        $keysSoFar++
                    }
                }
            }

            # Verify each parameter
            $allVerified = $true

            # Check organization
            if ($params.ContainsKey("organization")) {
                if ($params["organization"] -eq $organization) {
                    Write-Log "  ✓ organization: '$($params["organization"])'"
                } else {
                    Write-Log "  ✗ organization: Expected '$organization', found '$($params["organization"])'"
                    $allVerified = $false
                }
            } else {
                Write-Log "  ✗ organization: Not found in mdm.xml"
                $allVerified = $false
            }

            # Check service_mode
            if ($params.ContainsKey("service_mode")) {
                if ($params["service_mode"] -eq $serviceMode) {
                    Write-Log "  ✓ service_mode: '$($params["service_mode"])'"
                } else {
                    Write-Log "  ✗ service_mode: Expected '$serviceMode', found '$($params["service_mode"])'"
                    $allVerified = $false
                }
            } else {
                Write-Log "  ✗ service_mode: Not found in mdm.xml"
                $allVerified = $false
            }

            # Check onboarding
            if ($params.ContainsKey("onboarding")) {
                if ($params["onboarding"] -eq $expectedOnboarding) {
                    Write-Log "  ✓ onboarding: $($params["onboarding"])"
                } else {
                    Write-Log "  ✗ onboarding: Expected '$expectedOnboarding', found '$($params["onboarding"])'"
                    $allVerified = $false
                }
            } else {
                Write-Log "  ✗ onboarding: Not found in mdm.xml"
                $allVerified = $false
            }

            # Check switch_locked
            if ($params.ContainsKey("switch_locked")) {
                if ($params["switch_locked"] -eq $expectedSwitchLocked) {
                    Write-Log "  ✓ switch_locked: $($params["switch_locked"])"
                } else {
                    Write-Log "  ✗ switch_locked: Expected '$expectedSwitchLocked', found '$($params["switch_locked"])'"
                    $allVerified = $false
                }
            } else {
                Write-Log "  ✗ switch_locked: Not found in mdm.xml"
                $allVerified = $false
            }

            # Check auto_connect
            if ($params.ContainsKey("auto_connect")) {
                if ($params["auto_connect"] -eq $autoConnect.ToString()) {
                    Write-Log "  ✓ auto_connect: $($params["auto_connect"])"
                } else {
                    Write-Log "  ✗ auto_connect: Expected '$autoConnect', found '$($params["auto_connect"])'"
                    $allVerified = $false
                }
            } else {
                Write-Log "  ✗ auto_connect: Not found in mdm.xml"
                $allVerified = $false
            }

            # Check display_name
            if ($params.ContainsKey("display_name")) {
                if ($params["display_name"] -eq $displayName) {
                    Write-Log "  ✓ display_name: '$($params["display_name"])'"
                } else {
                    Write-Log "  ✗ display_name: Expected '$displayName', found '$($params["display_name"])'"
                    $allVerified = $false
                }
            } else {
                Write-Log "  ✗ display_name: Not found in mdm.xml"
                $allVerified = $false
            }

            if ($allVerified) {
                Write-Log "SUCCESS: All MDM parameters verified successfully"
            } else {
                Write-Log "WARNING: Some MDM parameters could not be verified"
            }

        } catch {
            Write-Log "WARNING: Could not parse mdm.xml: $($_.Exception.Message)"
            try {
                $rawContent = Get-Content $mdmXmlPath -Raw
                Write-Log "mdm.xml content: $rawContent"
            } catch {
                Write-Log "Could not read raw mdm.xml content"
            }
        }
    } else {
        Write-Log "WARNING: mdm.xml not found at $mdmXmlPath"
    }

    # Clean up installer file
    Write-Log "Cleaning up installer file..."
    Remove-Item -Path $installerPath -Force -ErrorAction SilentlyContinue

    Write-Log "Cloudflare WARP installation process completed"
    exit 0

} catch {
    Write-Log "ERROR: An exception occurred: $($_.Exception.Message)"
    Write-Log "Stack trace: $($_.ScriptStackTrace)"

    # Clean up on error
    if (Test-Path $installerPath) {
        Remove-Item -Path $installerPath -Force -ErrorAction SilentlyContinue
    }

    exit 1
}
