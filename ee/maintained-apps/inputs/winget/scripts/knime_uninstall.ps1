# Attempts to locate KNIME's Inno Setup uninstaller from registry and execute it silently.
# The ARP DisplayName may include the version (e.g. "KNIME 5.8.3"), so match a prefix.

$displayName = "KNIME"
$publisher   = "KNIME"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
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

Stop-Process -Name "knime" -Force -ErrorAction SilentlyContinue
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
