#Requires -Version 5.1
<#
.SYNOPSIS
  fleetd_healthcheck_windows.ps1

  Checks the health of all fleetd components on Windows and collects logs into
  a timestamped archive for support/troubleshooting. Windows counterpart to
  fleetd_healthcheck_ubuntu.sh.

  Components checked:
    - "Fleet osquery" service (Windows service, runs orbit.exe)
    - orbit               (process: orbit.exe)
    - osqueryd            (process: spawned and managed by orbit)
    - fleet-desktop       (process: optional, only present if packaged with --fleet-desktop)

  Sources:
    - Service name:        orbit/pkg/constant/constant.go (SystemServiceName = "Fleet osquery")
    - Install root:        orbit/pkg/packaging/windows_templates.go (ORBITROOT = C:\Program Files\Orbit)
    - File names:          orbit/pkg/constant/constant.go (OrbitNodeKeyFileName, OsqueryEnrollSecretFileName, OsqueryPidfile)
    - Registry key:        HKLM:\SOFTWARE\FleetDM\Orbit (Path)
    - Log paths:           https://fleetdm.com/guides/fleet-troubleshooting-for-it-admins
                              - orbit/osquery: C:\Windows\system32\config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log
                              - fleet-desktop: %LocalAppData%\Fleet\fleet-desktop.log (per user)
                              - osquery filesystem logger: C:\Program Files\Orbit\osquery_log

  Also collects a lookback window (default 72h, override with -LookbackHours or
  $env:LOOKBACK_HOURS) of system events - reboots, install/uninstall activity,
  failed services, System/Application errors - to help correlate a reported
  problem with what changed beforehand.

  Must be run from an elevated (Administrator) PowerShell session.
#>

[CmdletBinding()]
param(
  [int]$LookbackHours = $(if ($env:LOOKBACK_HOURS) { [int]$env:LOOKBACK_HOURS } else { 72 })
)

$ErrorActionPreference = 'Continue'

# -- Colour helpers -------------------------------------------------------------
function Ok   { param([string]$Msg) Write-Host "  [OK]    $Msg" -ForegroundColor Green;  Add-Content -Path $Summary -Value "  [OK]    $Msg" }
function Warn { param([string]$Msg) Write-Host "  [WARN]  $Msg" -ForegroundColor Yellow; Add-Content -Path $Summary -Value "  [WARN]  $Msg" }
function Fail { param([string]$Msg) Write-Host "  [FAIL]  $Msg" -ForegroundColor Red;    Add-Content -Path $Summary -Value "  [FAIL]  $Msg" }
function Info { param([string]$Msg) Write-Host "  [INFO]  $Msg";                          Add-Content -Path $Summary -Value "  [INFO]  $Msg" }
function Log  { param([string]$Msg) Write-Host $Msg;                                      Add-Content -Path $Summary -Value $Msg }

$IsAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltinRole]::Administrator)
if (-not $IsAdmin) {
  Write-Error "This script must be run from an elevated (Administrator) PowerShell session."
  exit 1
}

$Timestamp    = Get-Date -Format 'yyyyMMdd_HHmmss'
$HostnameSafe = $env:COMPUTERNAME -replace '\.', '_'
$ArchiveName  = "fleetd_healthcheck_${HostnameSafe}_${Timestamp}"
$WorkDir      = Join-Path $env:TEMP $ArchiveName
New-Item -ItemType Directory -Path $WorkDir -Force | Out-Null
$Summary      = Join-Path $WorkDir 'summary.txt'
New-Item -ItemType File -Path $Summary -Force | Out-Null
$OverallExit  = 0

$OrbitRoot = Join-Path $env:ProgramFiles 'Orbit'
$ServiceName = 'Fleet osquery'
$Since = (Get-Date).AddHours(-$LookbackHours)

# -- Header ---------------------------------------------------------------------
$OsInfo = (Get-CimInstance Win32_OperatingSystem).Caption
Log "============================================================"
Log " Fleet fleetd Health Check"
Log " Host:      $env:COMPUTERNAME"
Log " Date:      $(Get-Date)"
Log " OS:        $OsInfo"
Log "============================================================"
Log ""

# ==============================================================================
# 1. WINDOWS SERVICE
# ==============================================================================
Log "-- 1. Windows service (`"$ServiceName`") --------------------"

$Svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($Svc -and $Svc.Status -eq 'Running') {
  Ok "$ServiceName is running"
} else {
  Fail "$ServiceName is NOT running$(if ($Svc) { " (status: $($Svc.Status))" } else { " (service not found)" })"
  $OverallExit = 1
}

$SvcCim = Get-CimInstance Win32_Service -Filter "Name='$ServiceName'" -ErrorAction SilentlyContinue
if ($SvcCim) {
  if ($SvcCim.StartMode -eq 'Auto') {
    Ok "$ServiceName start mode is Automatic"
  } else {
    Warn "$ServiceName start mode is '$($SvcCim.StartMode)' - will not start on boot"
  }
  Add-Content -Path $Summary -Value ($SvcCim | Format-List Name, DisplayName, State, Status, StartMode, PathName | Out-String)
} else {
  Warn "Could not read service details via WMI/CIM"
}

# ==============================================================================
# 2. PROCESS CHECKS
# ==============================================================================
Log ""
Log "-- 2. Processes ---------------------------------------------"

function Check-Process {
  param([string]$Label, [string]$Name)
  $Procs = Get-Process -Name $Name -ErrorAction SilentlyContinue
  if ($Procs) {
    Ok "$Label is running"
    foreach ($p in $Procs) {
      $cmd = (Get-CimInstance Win32_Process -Filter "ProcessId=$($p.Id)" -ErrorAction SilentlyContinue).CommandLine
      Add-Content -Path $Summary -Value "    PID $($p.Id): $cmd"
    }
  } else {
    Fail "$Label is NOT running (process name: $Name)"
    $script:OverallExit = 1
  }
}

Check-Process "orbit" "orbit"
Check-Process "osqueryd" "osqueryd"

$Desktop = Get-Process -Name "fleet-desktop" -ErrorAction SilentlyContinue
if ($Desktop) {
  Ok "fleet-desktop is running"
  foreach ($p in $Desktop) { Add-Content -Path $Summary -Value "    PID $($p.Id)" }
} else {
  Warn "fleet-desktop is NOT running (expected if not packaged with --fleet-desktop)"
}

# ==============================================================================
# 3. KEY FILES / REGISTRY
# ==============================================================================
Log ""
Log "-- 3. Key files, directories, and registry -------------------"

function Check-Path {
  param([string]$Label, [string]$Path)
  if (Test-Path -LiteralPath $Path) {
    Ok "${Label}: $Path"
  } else {
    Fail "$Label not found: $Path"
    $script:OverallExit = 1
  }
}

Check-Path "Orbit root directory"  $OrbitRoot
Check-Path "osquery pidfile"       (Join-Path $OrbitRoot 'osquery.pid')
Check-Path "orbit node key"        (Join-Path $OrbitRoot 'secret-orbit-node-key.txt')
Check-Path "enroll secret"         (Join-Path $OrbitRoot 'secret.txt')

$OrbitExe = Get-ChildItem -Path (Join-Path $OrbitRoot 'bin\orbit') -Filter 'orbit.exe' -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
if ($OrbitExe) {
  Ok "orbit binary: $($OrbitExe.FullName)"
} else {
  Fail "orbit.exe not found under $OrbitRoot\bin\orbit"
  $OverallExit = 1
}

$OsquerydExe = Get-ChildItem -Path (Join-Path $OrbitRoot 'bin\osqueryd') -Filter 'osqueryd.exe' -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
if ($OsquerydExe) {
  Ok "osqueryd binary: $($OsquerydExe.FullName)"
} else {
  Warn "osqueryd.exe not found under $OrbitRoot\bin\osqueryd"
}

$NodeKeyPath = Join-Path $OrbitRoot 'secret-orbit-node-key.txt'
if ((Test-Path -LiteralPath $NodeKeyPath) -and (Get-Item -LiteralPath $NodeKeyPath).Length -gt 0) {
  Ok "orbit node key is non-empty (enrolled)"
} else {
  Fail "orbit node key is missing or empty (not enrolled)"
  $OverallExit = 1
}

$RegPath = 'HKLM:\SOFTWARE\FleetDM\Orbit'
if (Test-Path -LiteralPath $RegPath) {
  $RegProps = Get-ItemProperty -Path $RegPath -ErrorAction SilentlyContinue
  Ok "Registry key found: $RegPath"
  Add-Content -Path $Summary -Value ($RegProps | Select-Object * -ExcludeProperty PS* | Out-String)
} else {
  Warn "Registry key not found: $RegPath"
}

# ==============================================================================
# 4. SERVICE CONFIGURATION SUMMARY
# ==============================================================================
Log ""
Log "-- 4. Service configuration (redacted) ------------------------"
if ($SvcCim -and $SvcCim.PathName) {
  # Redact anything that looks like a secret/token/key/password in the service args
  $Redacted = $SvcCim.PathName -replace '(?i)(--[\w-]*(?:secret|password|token|key)[\w-]*[= ])(?:"[^"]*"|\S+)', '$1<redacted>'
  Log "    $Redacted"
} else {
  Fail "Could not read service ImagePath/arguments"
  $OverallExit = 1
}

# ==============================================================================
# 5. ORBIT VERSION
# ==============================================================================
Log ""
Log "-- 5. Orbit version --------------------------------------------"
if ($OrbitExe) {
  try {
    $OrbitVersion = & $OrbitExe.FullName version 2>&1 | Out-String
    Info $OrbitVersion.Trim()
    Add-Content -Path $Summary -Value $OrbitVersion
  } catch {
    Warn "Failed to run 'orbit.exe version': $_"
  }
} else {
  Warn "orbit.exe not found - skipping version check"
}

# ==============================================================================
# 6. LOG COLLECTION
# ==============================================================================
Log ""
Log "-- 6. Log collection ---------------------------------------------"

function Collect-Log {
  param([string]$Label, [string]$Src, [string]$DestDir)
  if (Test-Path -LiteralPath $Src -PathType Leaf) {
    New-Item -ItemType Directory -Path $DestDir -Force | Out-Null
    Copy-Item -LiteralPath $Src -Destination $DestDir -Force
    Ok "Collected ${Label}: $Src"
  } elseif (Test-Path -LiteralPath $Src -PathType Container) {
    New-Item -ItemType Directory -Path $DestDir -Force | Out-Null
    Copy-Item -Path (Join-Path $Src '*') -Destination $DestDir -Recurse -Force -ErrorAction SilentlyContinue
    Ok "Collected $Label directory: $Src"
  } else {
    Warn "$Label not found at $Src (skipping)"
  }
}

# orbit/osquery combined log - runs as LocalSystem, so it lives under the
# SYSTEM profile rather than a normal user's AppData.
Collect-Log "orbit/osquery log" `
  "$env:SystemRoot\system32\config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log" `
  (Join-Path $WorkDir 'logs\orbit')

# osquery filesystem logger output - only populated when logger_path/logger_plugin
# is set to "filesystem" in agent options.
Collect-Log "osquery filesystem logger" (Join-Path $OrbitRoot 'osquery_log') (Join-Path $WorkDir 'logs\osquery_log')

# fleet-desktop runs per-user; sweep all user profiles for its log file
$DesktopLogsFound = $false
Get-ChildItem 'C:\Users' -Directory -ErrorAction SilentlyContinue | ForEach-Object {
  $DesktopLog = Join-Path $_.FullName 'AppData\Local\Fleet\fleet-desktop.log'
  if (Test-Path -LiteralPath $DesktopLog) {
    $DestDir = Join-Path $WorkDir "logs\fleet-desktop\$($_.Name)"
    New-Item -ItemType Directory -Path $DestDir -Force | Out-Null
    Copy-Item -LiteralPath $DesktopLog -Destination $DestDir -Force
    Ok "Collected fleet-desktop log for user $($_.Name): $DesktopLog"
    $DesktopLogsFound = $true
  }
}
if (-not $DesktopLogsFound) {
  Warn "No fleet-desktop.log found under any user profile (expected if fleet-desktop is not installed)"
}

# Windows Event Log entries mentioning orbit/osquery/fleet (System + Application)
$EventLogDir = Join-Path $WorkDir 'logs\event_log'
New-Item -ItemType Directory -Path $EventLogDir -Force | Out-Null
try {
  $Matches = Get-WinEvent -FilterHashtable @{ LogName = 'System', 'Application'; StartTime = $Since } -ErrorAction SilentlyContinue |
    Where-Object { $_.Message -match 'orbit|osquery|fleet' }
  $Matches | Select-Object TimeCreated, LogName, ProviderName, Id, LevelDisplayName, Message |
    Export-Csv -Path (Join-Path $EventLogDir 'orbit_osquery_fleet_events.csv') -NoTypeInformation
  Ok "Collected $(($Matches | Measure-Object).Count) System/Application event(s) mentioning orbit/osquery/fleet"
} catch {
  Warn "Failed to query Windows Event Log: $_"
}

# ==============================================================================
# 7. SYSTEM EVENTS (lookback window)
# ==============================================================================
# Captures what changed on the box before the user noticed a problem - reboots,
# software installs/removals, service failures, and system errors. Users rarely
# remember every change; having this saves a round trip of clarifying questions.
Log ""
Log "-- 7. System events, last ${LookbackHours}h -----------------------"

$SystemEventsDir = Join-Path $WorkDir 'logs\system_events'
New-Item -ItemType Directory -Path $SystemEventsDir -Force | Out-Null

# Reboots/shutdowns - EventIDs: 6005/6006 (start/stop), 1074 (user-initiated), 41 (unexpected/dirty)
try {
  $Reboots = Get-WinEvent -FilterHashtable @{ LogName = 'System'; Id = 6005, 6006, 1074, 41; StartTime = $Since } -ErrorAction SilentlyContinue
  $Reboots | Select-Object TimeCreated, Id, Message | Export-Csv -Path (Join-Path $SystemEventsDir 'reboots.csv') -NoTypeInformation
  Ok "Collected reboot/shutdown history"
} catch {
  Warn "Failed to query reboot/shutdown events: $_"
}

# Software install/uninstall activity (MsiInstaller provider in Application log)
try {
  $MsiEvents = Get-WinEvent -FilterHashtable @{ LogName = 'Application'; ProviderName = 'MsiInstaller'; StartTime = $Since } -ErrorAction SilentlyContinue
  $MsiEvents | Select-Object TimeCreated, Id, Message | Export-Csv -Path (Join-Path $SystemEventsDir 'msi_install_events.csv') -NoTypeInformation
  $RecentPkgCount = ($MsiEvents | Measure-Object).Count
  if ($RecentPkgCount -gt 0) {
    Warn "$RecentPkgCount install/uninstall event(s) in last ${LookbackHours}h"
  } else {
    Ok "No install/uninstall events in last ${LookbackHours}h"
  }
} catch {
  Warn "Failed to query MsiInstaller events: $_"
}

# Services set to Automatic start but not currently running
try {
  $FailedServices = Get-CimInstance Win32_Service | Where-Object { $_.StartMode -eq 'Auto' -and $_.State -ne 'Running' }
  $FailedServices | Select-Object Name, DisplayName, State, StartMode | Export-Csv -Path (Join-Path $SystemEventsDir 'stopped_auto_services.csv') -NoTypeInformation
  if ($FailedServices) {
    Warn "$($FailedServices.Count) Automatic-start service(s) not running:"
    $FailedServices | ForEach-Object { Add-Content -Path $Summary -Value "    $($_.Name) ($($_.DisplayName)): $($_.State)" }
  } else {
    Ok "No Automatic-start services are stopped"
  }
} catch {
  Warn "Failed to enumerate services: $_"
}

# System/Application error-level events in the window
try {
  $ErrorEvents = Get-WinEvent -FilterHashtable @{ LogName = 'System', 'Application'; Level = 2; StartTime = $Since } -ErrorAction SilentlyContinue
  $ErrorEvents | Select-Object TimeCreated, LogName, ProviderName, Id, Message | Export-Csv -Path (Join-Path $SystemEventsDir 'error_events.csv') -NoTypeInformation
  $ErrCount = ($ErrorEvents | Measure-Object).Count
  if ($ErrCount -gt 0) {
    Warn "$ErrCount error-level event(s) in last ${LookbackHours}h (see error_events.csv)"
  } else {
    Ok "No System/Application errors in last ${LookbackHours}h"
  }
} catch {
  Warn "Failed to query error-level events: $_"
}

# Resource exhaustion (low memory) - Windows' rough equivalent of the Linux OOM killer
try {
  $ResourceEvents = Get-WinEvent -FilterHashtable @{ LogName = 'System'; ProviderName = 'Microsoft-Windows-Resource-Exhaustion-Detector'; StartTime = $Since } -ErrorAction SilentlyContinue
  if ($ResourceEvents) {
    Warn "Low-memory/resource-exhaustion activity detected in last ${LookbackHours}h"
    $ResourceEvents | Select-Object TimeCreated, Id, Message | Export-Csv -Path (Join-Path $SystemEventsDir 'resource_exhaustion_events.csv') -NoTypeInformation
  }
} catch {
  if ($_.Exception.Message -notmatch 'no provider|not found') {
    Warn "Failed to query resource-exhaustion events: $_"
  }
}

# ==============================================================================
# 8. PACKAGE THE ARCHIVE
# ==============================================================================
# All logging finishes before the work dir is zipped and removed below, so
# summary.txt (and the final result) end up inside the archive too.
Log ""
Log "-- 8. Packaging archive ----------------------------------------"

$ArchivePath = Join-Path $env:TEMP "$ArchiveName.zip"

# ==============================================================================
# FINAL RESULT
# ==============================================================================
Log "============================================================"
if ($OverallExit -eq 0) {
  Log " Result: ALL CHECKS PASSED"
} else {
  Log " Result: ONE OR MORE CHECKS FAILED -- review summary above"
}
Log " Archive: $ArchivePath"
Log "============================================================"

try {
  Compress-Archive -Path (Join-Path $WorkDir '*') -DestinationPath $ArchivePath -Force -ErrorAction Stop
  Remove-Item -Path $WorkDir -Recurse -Force
  Write-Host "  [INFO]  Archive created: $ArchivePath"
} catch {
  Write-Error "Failed to create archive: $_"
  Write-Host "  [WARN]  Archive creation failed; diagnostic data retained at: $WorkDir"
  $OverallExit = 1
}

exit $OverallExit
