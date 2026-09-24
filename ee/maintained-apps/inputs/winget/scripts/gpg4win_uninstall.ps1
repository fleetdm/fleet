# The registry DisplayName carries a parenthesised version ("Gpg4win (5.0.2)"),
# so match on a prefix rather than an exact string.
$softwareName = "Gpg4win"
$softwarePublisher = "The Gpg4win Project"

# The bundled GnuPG daemons and Kleopatra hold file locks and would block -Wait,
# so stop them first and wait only on the uninstaller process.
$leftovers = @("gpg-agent", "dirmngr", "keyboxd", "scdaemon", "gpg-connect-agent", "gpgconf", "kleopatra", "gpgme-w32spawn")
$timeoutSeconds = 300

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
# Uninstall info is written with SHCTX, so it can land per-user.
$userKey = 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$userKey32on64 = 'HKCU:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$exitCode = 0

$allKeys = @($machineKey, $machineKey32on64, $userKey, $userKey32on64)

function Get-Gpg4winUninstallKey {
    Get-ChildItem -Path $allKeys -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "$softwareName*" -and $_.Publisher -eq $softwarePublisher } |
        Select-Object -First 1
}

# Logs matching entries and their publishers to diagnose a name/publisher miss.
function Write-Gpg4winCandidates {
    Write-Host "Registry entries matching '$softwareName*':"
    $found = Get-ChildItem -Path $allKeys -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "$softwareName*" }
    if (-not $found) { Write-Host "  (none)" ; return }
    foreach ($f in $found) {
        Write-Host "  DisplayName='$($f.DisplayName)' Publisher='$($f.Publisher)' Version='$($f.DisplayVersion)'"
    }
}

foreach ($name in $leftovers) {
    Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
}

try {
    $key = Get-Gpg4winUninstallKey
    if (-not $key) {
        Write-Gpg4winCandidates
        Write-Host "Uninstall entry not found for '$softwareName' with publisher '$softwarePublisher'."
        Exit 0
    }

    $uninstallString = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
    Write-Host "Uninstall string: $uninstallString"

    # Handles quoted paths, unquoted paths with spaces, and bare tokens.
    $uninstallCommand = $uninstallString
    if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
        $uninstallCommand = $Matches[1]
    } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        $uninstallCommand = $Matches[1]
    } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
        $uninstallCommand = $Matches[1]
    }

    # NSIS uninstallers relaunch from %TEMP% and detach by default; "_?=<dir>"
    # runs in place so this stays synchronous. Must be the last argument.
    $installDir = Split-Path -Parent $uninstallCommand
    $uninstallArgs = "/S _?=$installDir"

    Write-Host "Uninstall command: $uninstallCommand"
    Write-Host "Uninstall args: $uninstallArgs"

    $process = Start-Process -FilePath $uninstallCommand -ArgumentList $uninstallArgs -PassThru
    # Keeps .ExitCode readable after the process ends.
    $null = $process.Handle

    if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        Write-Host "Uninstall timed out after $timeoutSeconds seconds"
        Exit 1603
    }

    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"
} catch {
    Write-Host "Error: $_"
    Exit 1
}

# Stop anything restarted, then wait for the ARP entry to clear.
foreach ($name in $leftovers) {
    Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
}

$elapsed = 0
while ((Get-Gpg4winUninstallKey) -and ($elapsed -lt 120)) {
    Start-Sleep -Seconds 5
    $elapsed += 5
    Write-Host "Waiting for the uninstall to finish... ($elapsed seconds)"
}

if (Get-Gpg4winUninstallKey) {
    Write-Gpg4winCandidates
    Write-Host "'$softwareName' is still registered after the uninstall."
    Exit 1
}

Exit $exitCode
