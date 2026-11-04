# Locates Opera from the registry and runs its uninstaller silently.
#
# Opera registers a machine-wide ARP entry (DisplayName "Opera Stable <version>",
# Publisher "Opera Software"). Its UninstallString is
# "<InstallFolder>\opera.exe" /uninstall -- Opera uses single-dash switches, and
# a bare /uninstall shows an "uninstall reasons" survey screen that blocks a
# headless run. We take only the opera.exe path from the registry and pass clean
# switches: /silent suppresses the survey, /allusers=1 removes the machine-wide
# install. The uninstaller returns before removal finishes, so we then poll until
# the registration is gone.

$displayNameLike = "Opera Stable*"
$publisherLike = "Opera Software*"

$registryUninstallPaths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

function Get-OperaEntry {
    Get-ItemProperty -Path $registryUninstallPaths -ErrorAction SilentlyContinue |
        Where-Object { $_.DisplayName -like $displayNameLike -and $_.Publisher -like $publisherLike } |
        Select-Object -First 1
}

try {

$uninstall = Get-OperaEntry
if (-not $uninstall -or (-not $uninstall.UninstallString -and -not $uninstall.QuietUninstallString)) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

$uninstallCommand = if ($uninstall.QuietUninstallString) {
  $uninstall.QuietUninstallString
} else {
  $uninstall.UninstallString
}

# Take only the opera.exe path; discard the registry's own switches and pass our
# own clean, silent ones.
$exePath = ""
if ($uninstallCommand -match '^\s*"([^"]+)"') {
    $exePath = $matches[1]
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)') {
    $exePath = $matches[1]
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

if (-not (Test-Path -LiteralPath $exePath)) {
    Write-Host "Uninstaller not found at $exePath"
    Exit 1
}

$uninstallArgs = "/uninstall", "/silent", "/allusers=1"

Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $($uninstallArgs -join ' ')"

$processOptions = @{
    FilePath = $exePath
    ArgumentList = $uninstallArgs
    NoNewWindow = $true
    PassThru = $true
    Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

# Opera's uninstaller returns before it finishes removing the app, so wait for
# the registration to actually disappear.
for ($waited = 0; $waited -lt 120; $waited++) {
    if (-not (Get-OperaEntry)) {
        Write-Host "Opera removed after ${waited}x3s"
        Exit 0
    }
    Start-Sleep -Seconds 3
}

Write-Host "Opera still registered after waiting; uninstall did not complete"
if ($exitCode -eq 0) { $exitCode = 1 }
Exit $exitCode

} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
