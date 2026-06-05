# Locates the Microsoft .NET Runtime 10 (x64) WiX "burn" bundle in the registry
# and runs its uninstaller silently.
#
# Burn bundles register both UninstallString (e.g.
#   "C:\ProgramData\Package Cache\{guid}\dotnet-runtime-10.0.x-win-x64.exe" /uninstall)
# and, usually, QuietUninstallString (same, with /quiet appended). We prefer the
# quiet form and otherwise append the silent switches ourselves.
#
# The DisplayName embeds the exact version ("Microsoft .NET Runtime - 10.0.8 (x64)"),
# so we match on the major version + architecture.

$displayNameLike = "Microsoft .NET Runtime - 10.* (x64)"
$publisher = "Microsoft Corporation"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$entry = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -like $displayNameLike -and $_.Publisher -like "$publisher*"
  }
  if ($items) { $entry = $items | Select-Object -First 1; break }
}

if (-not $entry) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

# Prefer the vendor-provided quiet uninstall command when present.
$raw = $entry.QuietUninstallString
$needsSilentSwitches = $false
if (-not $raw) {
  $raw = $entry.UninstallString
  $needsSilentSwitches = $true
}

if (-not $raw) {
  Write-Host "No uninstall string found"
  Exit 0
}

# Parse the command into an executable path and its arguments. Handle the three
# common shapes: quoted path, unquoted path that may contain spaces, bare token.
if ($raw -match '^\s*"([^"]+)"\s*(.*)$') {
    $exe = $matches[1]
    $exeArgs = $matches[2].Trim()
} elseif ($raw -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exe = $matches[1]
    $exeArgs = $matches[2].Trim()
} else {
    $exe = $raw
    $exeArgs = ""
}

# Ensure silent + no-restart switches are present.
if ($needsSilentSwitches -and $exeArgs -notmatch '/uninstall') {
    $exeArgs = "/uninstall $exeArgs"
}
if ($exeArgs -notmatch '/quiet') {
    $exeArgs = "$exeArgs /quiet"
}
if ($exeArgs -notmatch '/norestart') {
    $exeArgs = "$exeArgs /norestart"
}
$exeArgs = $exeArgs.Trim()

Write-Host "Uninstall command: $exe"
Write-Host "Uninstall args: $exeArgs"

try {
    $processOptions = @{
        FilePath = $exe
        ArgumentList = $exeArgs
        NoNewWindow = $true
        PassThru = $true
        Wait = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

    # 0 = success, 3010 = success but reboot required, 1641 = reboot initiated
    if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
        Exit 0
    }

    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
