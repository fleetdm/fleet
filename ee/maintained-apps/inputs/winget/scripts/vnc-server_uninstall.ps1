# Uninstall RealVNC Server (MSI) via its registry UninstallString.
# DisplayName is versioned (e.g. "RealVNC Server 7.17.0"), Publisher "RealVNC".
# The MSI installs machine-wide (ALLUSERS=1), so its ARP entry lives under HKLM.

$softwareNameLike = "RealVNC Server*"
$publisher = "RealVNC"

$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

try {

[array]$uninstallKeys = Get-ChildItem -Path $paths -ErrorAction SilentlyContinue |
    ForEach-Object { Get-ItemProperty $_.PSPath }

$key = $uninstallKeys | Where-Object {
    $_.DisplayName -like $softwareNameLike -and
    ($publisher -eq "" -or $_.Publisher -eq $publisher)
} | Select-Object -First 1

if (-not $key -or -not $key.UninstallString) {
    Write-Host "Uninstall entry not found for $softwareNameLike"
    Exit 0
}

$uninstallString = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }

# MSI uninstall strings look like: MsiExec.exe /X{GUID} or /I{GUID}. Force /qn.
if ($uninstallString -match "MsiExec\.exe\s+/[IX]\s*(\{[A-Fa-f0-9-]+\})") {
    $productCode = $Matches[1]
    $uninstallCommand = "MsiExec.exe"
    $uninstallArgs = "/X $productCode /qn /norestart"
} elseif ($uninstallString -match '^\s*"([^"]+)"\s*(.*)$') {
    $uninstallCommand = $Matches[1]
    $uninstallArgs = $Matches[2].Trim()
} elseif ($uninstallString -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $uninstallCommand = $Matches[1]
    $uninstallArgs = $Matches[2].Trim()
} else {
    Write-Host "Error: Unable to parse uninstall command: $uninstallString"
    Exit 1
}

Write-Host "Uninstall command: $uninstallCommand"
Write-Host "Uninstall args: $uninstallArgs"

$processOptions = @{
    FilePath = $uninstallCommand
    PassThru = $true
    Wait = $true
}
if ($uninstallArgs -ne '') {
    $processOptions.ArgumentList = $uninstallArgs
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
    Exit 0
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
