# 1Password Uninstall Script
# Uses winget to uninstall the package silently and non-interactively

# Check if winget is available
if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
    Write-Host "Error: winget is not available on this system"
    Exit 1
}

# Uninstall using winget with silent and non-interactive flags
winget uninstall --id AgileBits.1Password `
    --silent `
    --disable-interactivity

# Wait a moment for registry/file system updates to propagate
Start-Sleep -Seconds 3

# Verify the uninstall was successful by checking the Windows registry
# This matches how appExists() verifies - by checking the programs registry entries
$registryPaths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

$found = $false
foreach ($path in $registryPaths) {
    $items = Get-ItemProperty $path -ErrorAction SilentlyContinue | Where-Object {
        $_.DisplayName -and ($_.DisplayName -eq "1Password" -or $_.DisplayName -like "1Password*")
    }
    if ($items) {
        $found = $true
        Write-Host "Found registry entry: $($items[0].DisplayName) at $path"
        break
    }
}

# Also check if the installation directory still exists
if (-not $found) {
    $installPath = "C:\Program Files\1Password"
    if (Test-Path $installPath) {
        $found = $true
        Write-Host "Installation directory still exists: $installPath"
    }
}

if ($found) {
    Write-Host "Error: 1Password is still present in registry or file system after uninstall"
    Exit 1
} else {
    Write-Host "Successfully uninstalled 1Password - verified registry and file system"
    Exit 0
}

