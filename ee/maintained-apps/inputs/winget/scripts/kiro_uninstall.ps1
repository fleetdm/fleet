# Attempts to locate Kiro's Inno Setup uninstaller from registry and execute it silently.
# Kiro is user-scope; the ARP DisplayName is "Kiro (User)".

$displayName = "Kiro"
$publisher   = "Amazon"

$paths = @(
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName *") -and
    ($publisher -eq "" -or $_.Publisher -like "*$publisher*")
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

# Kill any running Kiro processes before uninstalling
Stop-Process -Name "Kiro" -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

$uninstallCommand = $uninstall.UninstallString
$uninstallArgs = "/VERYSILENT /NORESTART"

# Parse the uninstall command to separate executable from existing arguments
if ($uninstallCommand -match '^"([^"]+)"\s*(.*)$') {
    $uninstallCommand = $matches[1]; $extra = $matches[2].Trim()
    if ($extra) { $uninstallArgs = "$extra $uninstallArgs".Trim() }
} elseif ($uninstallCommand -match '^(.+?\.exe)\s*(.*)$') {
    $uninstallCommand = $matches[1]; $extra = $matches[2].Trim()
    if ($extra) { $uninstallArgs = "$extra $uninstallArgs".Trim() }
} else {
    Write-Host "Error: Could not parse uninstall command: $uninstallCommand"; Exit 1
}

Write-Host "Uninstall command: $uninstallCommand"
Write-Host "Uninstall args: $uninstallArgs"

try {
    $processOptions = @{
        FilePath = $uninstallCommand
        ArgumentList = $uninstallArgs
        NoNewWindow = $true
        PassThru = $true
        Wait = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    Write-Host "Uninstall exit code: $exitCode"
    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
