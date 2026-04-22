# Uninstall Microsoft Office (Click-to-Run).
#
# Office registers an ARP (Add/Remove Programs) uninstall entry whose
# UninstallString looks like:
#   "C:\Program Files\Common Files\Microsoft Shared\ClickToRun\OfficeClickToRun.exe"
#   scenario=install scenariosubtype=ARP sourcetype=None
#   productstoremove=O365HomePremRetail.16_en-us_x-none culture=en-us version.16=16.0
#
# Running that string as-is shows a UI. Appending "DisplayLevel=False" tells
# OfficeClickToRun.exe to uninstall silently. This script:
#   1. Finds the Office uninstall entry in the registry.
#   2. Closes any running Office apps.
#   3. Invokes OfficeClickToRun.exe with the original arguments plus
#      DisplayLevel=False and waits for it (and any follow-up Click-to-Run
#      workers) to finish.

$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
    $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
        $_.Publisher -eq 'Microsoft Corporation' -and
        $_.DisplayName -and
        ($_.DisplayName -like 'Microsoft 365*' -or $_.DisplayName -like 'Microsoft Office*') -and
        $_.UninstallString -like '*OfficeClickToRun.exe*'
    }
    if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
    Write-Host "Microsoft Office uninstall entry not found"
    Exit 0
}

Write-Host "Found: $($uninstall.DisplayName) ($($uninstall.DisplayVersion))"
Write-Host "Original UninstallString: $($uninstall.UninstallString)"

# Close Office apps that could block the uninstaller.
$officeProcesses = @(
    'WINWORD', 'EXCEL', 'POWERPNT', 'OUTLOOK', 'ONENOTE', 'MSACCESS',
    'MSPUB', 'VISIO', 'WINPROJ', 'LYNC', 'TEAMS', 'ONEDRIVE'
)
foreach ($proc in $officeProcesses) {
    Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
}

# UninstallString is a full command line: split off the executable path from
# its arguments so we can pass them to Start-Process separately.
$uninstallString = $uninstall.UninstallString
$filePath = $null
$arguments = $null
if ($uninstallString -match '^\s*"([^"]+)"\s*(.*)$') {
    $filePath = $Matches[1]
    $arguments = $Matches[2]
} elseif ($uninstallString -match '^\s*(\S+)\s*(.*)$') {
    $filePath = $Matches[1]
    $arguments = $Matches[2]
} else {
    Write-Host "Error: Unable to parse uninstall command: $uninstallString"
    Exit 1
}

if ($arguments -notmatch '(?i)DisplayLevel\s*=') {
    if ($arguments) {
        $arguments = "$arguments DisplayLevel=False"
    } else {
        $arguments = "DisplayLevel=False"
    }
}

Write-Host "Running: `"$filePath`" $arguments"

$exitCode = 0
try {
    $processOptions = @{
        FilePath     = $filePath
        ArgumentList = $arguments
        PassThru     = $true
        Wait         = $true
        NoNewWindow  = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "OfficeClickToRun.exe exit code: $exitCode"

    # OfficeClickToRun can hand work off to additional processes; wait until
    # none remain before returning so the caller sees a clean state.
    $timeout = 1800
    $elapsed = 0
    while ((Get-Process -Name 'OfficeClickToRun' -ErrorAction SilentlyContinue) -and ($elapsed -lt $timeout)) {
        Start-Sleep -Seconds 10
        $elapsed += 10
        Write-Host "Waiting for OfficeClickToRun to exit... ($elapsed seconds)"
    }

    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
