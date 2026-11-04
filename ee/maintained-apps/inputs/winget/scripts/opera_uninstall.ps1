# Locates Opera from the registry and runs its uninstaller silently.
#
# Opera registers a machine-wide ARP entry (DisplayName "Opera Stable <version>",
# Publisher "Opera Software") whose UninstallString points at its own
# installer.exe. QuietUninstallString already carries the silent switches; when
# only UninstallString is present we add --uninstall --silent ourselves.
#
# Opera's uninstaller relaunches itself from a versioned/temp copy and returns
# before removal is finished, so after launching it we poll until the ARP
# registration is gone rather than trusting the immediate exit.

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

if ($existingArgs -notmatch '(?i)(^|\s)--uninstall($|\s)') {
    $existingArgs = ("$existingArgs --uninstall").Trim()
}
if ($existingArgs -notmatch '(?i)(^|\s)--silent($|\s)') {
    $existingArgs = ("$existingArgs --silent").Trim()
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

# The uninstaller returns before it finishes removing the app, so wait for the
# registration to actually disappear.
for ($waited = 0; $waited -lt 120; $waited++) {
    if (-not (Get-OperaEntry)) {
        Write-Host "Opera removed after ${waited}s"
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
