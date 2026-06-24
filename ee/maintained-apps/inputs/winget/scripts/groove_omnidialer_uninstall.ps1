# Attempts to locate Groove OmniDialer's uninstaller from the registry and execute it silently.
# The Add/Remove Programs DisplayName includes the version (e.g. "Groove OmniDialer 26.603.1020"),
# so match on the prefix while excluding the separate "Enterprise Edition" product.

$displayName = "Groove OmniDialer"
$publisher = "Groove Labs, Inc."

$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null

foreach ($p in $paths) {
    $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
        $_.DisplayName -and
        ($_.DisplayName -like "$displayName *") -and
        ($_.DisplayName -notlike "$displayName Enterprise*") -and
        ($publisher -eq "" -or $_.Publisher -eq $publisher)
    }

    if ($items) {
        $uninstall = $items | Select-Object -First 1
        break
    }
}

if (-not $uninstall) {
    Write-Host "Uninstall entry not found"
    Exit 0
}

$uninstallString = if ($uninstall.QuietUninstallString) {
    $uninstall.QuietUninstallString
}
else {
    $uninstall.UninstallString
}

if (-not $uninstallString) {
    Write-Host "Uninstall command not found"
    Exit 0
}

Stop-Process -Name "Groove OmniDialer" -Force -ErrorAction SilentlyContinue

$exePath = ""
$arguments = ""

# Parse the uninstall string into an executable path and existing arguments.
# Handles quoted paths, unquoted paths that may contain spaces, and bare tokens.
if ($uninstallString -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
    $arguments = $matches[2].Trim()
}
elseif ($uninstallString -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]
    $arguments = $matches[2].Trim()
}
elseif ($uninstallString -match '^\s*(\S+)\s*(.*)$') {
    $exePath = $matches[1]
    $arguments = $matches[2].Trim()
}
else {
    Write-Host "Error: Could not parse uninstall string: $uninstallString"
    Exit 1
}

$argumentList = @()
if ($arguments -ne '') {
    $argumentList += $arguments -split '\s+'
}

# NSIS uninstallers require /S for silent mode.
if ($argumentList -notcontains "/S" -and $arguments -notmatch '\b/S\b') {
    $argumentList += "/S"
}

Write-Host "Uninstall executable: $exePath"
Write-Host "Uninstall arguments: $($argumentList -join ' ')"

try {
    $processOptions = @{
        FilePath    = $exePath
        NoNewWindow = $true
        PassThru    = $true
        Wait        = $true
    }

    if ($argumentList.Count -gt 0) {
        $processOptions.ArgumentList = $argumentList
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    Write-Host "Uninstall exit code: $exitCode"
    Exit $exitCode
}
catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
