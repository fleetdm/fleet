# Telegram Desktop uninstall script
# Removes the extracted Telegram Desktop files and any registry entries

$displayName = "Telegram Desktop"
$publisher = "Telegram FZ-LLC"

# Determine installation directory
$installDir = Join-Path $env:LOCALAPPDATA "Programs\Telegram Desktop"

# Kill any running Telegram processes before uninstalling
Stop-Process -Name "Telegram" -Force -ErrorAction SilentlyContinue

# Wait a moment for processes to terminate
Start-Sleep -Seconds 2

# Remove installation directory
if (Test-Path $installDir) {
    try {
        Remove-Item -Path $installDir -Recurse -Force -ErrorAction Stop
        Write-Host "Removed installation directory: $installDir"
    } catch {
        Write-Host "Warning: Could not remove installation directory: $_"
    }
}

# Attempt to remove registry entries if they exist
$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$removed = $false
foreach ($p in $paths) {
    try {
        $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
            $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*")
        }
        if ($items) {
            foreach ($item in $items) {
                $keyName = Split-Path -Leaf $item.PSPath
                Remove-Item -Path "$p\$keyName" -Force -ErrorAction SilentlyContinue
                Write-Host "Removed registry entry: $p\$keyName"
                $removed = $true
            }
        }
    } catch {
        # Ignore registry errors
    }
}

if (-not $removed -and -not (Test-Path $installDir)) {
    Write-Host "Uninstall completed (no registry entries found)"
}

Exit 0

