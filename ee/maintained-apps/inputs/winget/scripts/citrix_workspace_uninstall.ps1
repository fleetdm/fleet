# Locates Citrix Workspace from the registry and runs its uninstaller silently.
#
# Citrix Workspace app registers a machine-wide ARP entry whose UninstallString
# points at the vendor bootstrapper (TrolleyExpress.exe / CitrixWorkspaceApp.exe),
# not MsiExec. The app's installed sub-packages register their own MsiExec-based
# entries; we skip those and run the bootstrapper's uninstaller, which removes the
# whole app. Publisher is matched loosely because recent builds moved from
# "Citrix Systems, Inc." to "Cloud Software Group".

$displayNameLike = "Citrix Workspace*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

try {

[array]$entries = Get-ChildItem -Path $paths -ErrorAction SilentlyContinue |
  ForEach-Object {
    $props = Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue
    if ($props -and $props.DisplayName -like $displayNameLike -and
        ($props.Publisher -like 'Citrix*' -or $props.Publisher -like 'Cloud Software Group*')) {
      $props
    }
  }

# Prefer the bootstrapper entry (non-MsiExec UninstallString) over any MSI
# sub-package entries that share the "Citrix Workspace" name.
$uninstall = $entries | Where-Object {
  ($_.QuietUninstallString -and $_.QuietUninstallString -notmatch '^\s*MsiExec') -or
  ($_.UninstallString -and $_.UninstallString -notmatch '^\s*MsiExec')
} | Select-Object -First 1

if (-not $uninstall) { $uninstall = $entries | Select-Object -First 1 }

if (-not $uninstall -or (-not $uninstall.UninstallString -and -not $uninstall.QuietUninstallString)) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

$uninstallCommand = if ($uninstall.QuietUninstallString) {
  $uninstall.QuietUninstallString
} else {
  $uninstall.UninstallString
}

$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

if ($existingArgs -notmatch '(?i)(^|\s)/uninstall($|\s)') {
    $existingArgs = ("$existingArgs /uninstall").Trim()
}
if ($existingArgs -notmatch '(?i)(^|\s)/silent($|\s)') {
    $existingArgs = ("$existingArgs /silent").Trim()
}
if ($existingArgs -notmatch '(?i)/noreboot') {
    $existingArgs = ("$existingArgs /noreboot").Trim()
}

Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $existingArgs"

$processOptions = @{
    FilePath = $exePath
    ArgumentList = $existingArgs
    NoNewWindow = $true
    PassThru = $true
    Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { $exitCode = 0 }
Exit $exitCode

} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
