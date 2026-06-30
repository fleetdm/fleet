# Locates Omnissa Horizon Client's uninstaller from the registry and runs it
# silently.
#
# Omnissa Horizon Client ships as a WiX Burn bundle that registers two entries
# with the same DisplayName: the Burn bundle itself (has a BundleVersion
# registry value; UninstallString points to a .exe in
# C:\ProgramData\Package Cache) and the inner MSI (UninstallString is
# "MsiExec.exe /X{ProductCode}").
#
# Running the Burn bundle's uninstaller hangs / times out in CI, so we instead:
#   1. Run msiexec on the inner MSI (fast, ~15-20s)
#   2. Delete the Burn bundle's registry key so osquery / Add-Remove no longer
#      sees the app
# Both steps are needed: msiexec removes the files and the MSI's own registry
# entry, but the Burn bundle's wrapper entry persists.

$softwareNameLike = "*Omnissa Horizon Client*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

try {

[array]$uninstallEntries = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object {
            $props = Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue
            if ($props -and $props.DisplayName -like $softwareNameLike) {
                [PSCustomObject]@{
                    Props    = $props
                    KeyPath  = $_.PSPath
                }
            }
        }

if (-not $uninstallEntries -or $uninstallEntries.Count -eq 0) {
    Write-Host "Uninstall entry not found for $softwareNameLike"
    Exit 1
}

Write-Host "Found $($uninstallEntries.Count) matching uninstall entries"

$msiEntry = $uninstallEntries | Where-Object {
    $_.Props.UninstallString -match '^\s*MsiExec\.exe'
} | Select-Object -First 1

$bundleEntry = $uninstallEntries | Where-Object {
    $_.Props.BundleVersion -or ($_.Props.UninstallString -and $_.Props.UninstallString -notmatch '^\s*MsiExec\.exe')
} | Select-Object -First 1

# Step 1: run msiexec on the inner MSI (fast).
$msiExitCode = 0
if ($msiEntry) {
    if ($msiEntry.Props.UninstallString -match 'MsiExec\.exe\s+(.*)$') {
        $msiArgs = $matches[1].Trim()
    } else {
        $msiArgs = ""
    }
    if ($msiArgs -notmatch '/quiet' -and $msiArgs -notmatch '/qn') {
        $msiArgs = "$msiArgs /quiet /norestart"
    }
    Write-Host "Running: MsiExec.exe $msiArgs"
    $process = Start-Process -FilePath "MsiExec.exe" -ArgumentList $msiArgs -PassThru -Wait
    $msiExitCode = $process.ExitCode
    Write-Host "msiexec exit code: $msiExitCode"

    # msiexec returns 3010 (reboot-required) or 1641 (reboot-initiated) on
    # successful uninstall when a reboot is needed; treat both as success.
    if ($msiExitCode -ne 0 -and $msiExitCode -ne 3010 -and $msiExitCode -ne 1641) {
        Write-Host "msiexec failed; not removing Burn bundle entry"
        Exit $msiExitCode
    }
} else {
    Write-Host "No inner MSI entry found; skipping msiexec step"
}

# Step 2: delete the leftover Burn bundle registry key, if any. osquery reads
# the programs table from these uninstall keys, so leaving it behind makes
# Add-Remove (and Fleet) still see the app.
if ($bundleEntry) {
    Write-Host "Removing Burn bundle registry key: $($bundleEntry.KeyPath)"
    Remove-Item -Path $bundleEntry.KeyPath -Recurse -Force -ErrorAction Stop
    Write-Host "Burn bundle registry key removed"
}

Exit 0

} catch {
    Write-Host "Error: $_"
    Exit 1
}
