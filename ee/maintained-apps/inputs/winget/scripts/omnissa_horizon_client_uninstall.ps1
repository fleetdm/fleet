# Locates Omnissa Horizon Client's uninstaller from the registry and runs it
# silently.
#
# Omnissa Horizon Client ships as a WiX Burn bundle that registers two entries
# with the same DisplayName: the Burn bundle itself (UninstallString points to
# a .exe in C:\ProgramData\Package Cache and has a BundleVersion property) and
# the inner MSI (UninstallString is "MsiExec.exe /X{ProductCode}"). Running
# just the inner MSI leaves the bundle entry behind in Programs and Features,
# so we prefer the Burn bundle entry — it uninstalls both and doesn't require
# a reboot.

$softwareNameLike = "*Omnissa Horizon Client*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

# Collect all matching entries, then pick the Burn bundle in preference to
# the inner MSI.
$candidates = @()
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -like $softwareNameLike) {
        $candidates += $key
    }
}

$bundle = $candidates | Where-Object {
    $_.BundleVersion -or ($_.UninstallString -and $_.UninstallString -notmatch '^\s*MsiExec\.exe')
} | Select-Object -First 1

$selected = if ($bundle) { $bundle } else { $candidates | Select-Object -First 1 }

if (-not $selected) {
    Write-Host "Uninstall entry not found for $softwareNameLike"
    Exit 1
}

$uninstallCommand = if ($selected.QuietUninstallString) {
    $selected.QuietUninstallString
} else {
    $selected.UninstallString
}

# Split the uninstall string into exe + args. Handle both quoted and unquoted
# exe paths.
$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

# Burn's UninstallString omits the /uninstall verb (UninstallString assumes it,
# unlike QuietUninstallString which usually includes both). Add it if missing.
if ($selected.BundleVersion -and $existingArgs -notmatch '/uninstall') {
    $existingArgs = ("/uninstall $existingArgs").Trim()
}

if ($existingArgs -notmatch '/quiet' -and $existingArgs -notmatch '/qn') {
    $uninstallArgs = ("$existingArgs /quiet /norestart").Trim()
} else {
    $uninstallArgs = $existingArgs
}

Write-Host "Selected entry DisplayName: $($selected.DisplayName)"
Write-Host "BundleVersion: $($selected.BundleVersion)"
Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $uninstallArgs"

$processOptions = @{
    FilePath = $exePath
    PassThru = $true
    Wait = $true
}

if ($uninstallArgs -ne '') {
    $processOptions.ArgumentList = $uninstallArgs
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

# msiexec returns 3010 (ERROR_SUCCESS_REBOOT_REQUIRED) or 1641
# (ERROR_SUCCESS_REBOOT_INITIATED) on successful uninstall when a reboot is
# needed. Treat both as success.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
    Exit 0
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
