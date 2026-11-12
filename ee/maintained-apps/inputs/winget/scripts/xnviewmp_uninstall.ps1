# Uninstall XnView MP (Inno Setup).

$displayName = "XnView MP"
$publisher = ""

$machinePaths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)
$userPath = 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'

function Find-UninstallEntry {
    param([string[]]$Paths)
    Get-ChildItem -Path $Paths -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object {
            ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and
            ($publisher -eq "" -or $_.Publisher -like "*$publisher*")
        }
}

try {

# Search HKCU first, then HKLM (and WOW6432Node).
$entry = Find-UninstallEntry -Paths $userPath | Select-Object -First 1
if (-not $entry) {
    $entry = Find-UninstallEntry -Paths $machinePaths | Select-Object -First 1
}

if (-not $entry) {
    Write-Host "Uninstall entry not found"
    Exit 0
}

$uninstallCommand = if ($entry.QuietUninstallString) { $entry.QuietUninstallString } else { $entry.UninstallString }

# Parse the uninstaller executable path.
$exe = $null
if ($uninstallCommand -match '^"([^"]+)"') {
    $exe = $matches[1]
} elseif ($uninstallCommand -match '^(.+?\.exe)') {
    $exe = $matches[1]
} else {
    Write-Host "Could not parse uninstall command: $uninstallCommand"
    Exit 1
}

# Determine the install directory.
$installDir = $null
if ($entry.InstallLocation -and (Test-Path $entry.InstallLocation)) {
    $installDir = $entry.InstallLocation
} else {
    $installDir = Split-Path -Parent $exe
}

$uninstallArgs = @("/VERYSILENT", "/NORESTART")

Write-Host "Uninstall command: $exe"
Write-Host "Uninstall args: $($uninstallArgs -join ' ')"

$process = Start-Process -FilePath $exe -ArgumentList $uninstallArgs -Wait -PassThru
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

if ($installDir -and (Test-Path $installDir)) {
    Remove-Item $installDir -Recurse -Force -ErrorAction SilentlyContinue
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
