# The registry DisplayName carries a parenthesised version ("Gpg4win (5.0.2)"),
# so match on a prefix rather than an exact string.
$softwareName = "Gpg4win"
# Require the publisher too, mirroring the manifest's exists query. This puts the
# value under test: the validator's appExists looks up by name only, so a wrong
# exists-query publisher would otherwise ship a manifest that can never match an
# install (cf. the Spyder finding in #50016).
#
# This one is the least certain in the batch. "The Gpg4win Project" comes from the
# winget locale manifest -- a *package* publisher, which is exactly the kind of
# value that was wrong for Spyder -- and it could not be confirmed statically: the
# installer's PE version resource says "g10 Code GmbH", and the NSIS script that
# writes the real value is compressed. If it is wrong the uninstall below finds
# nothing, and the diagnostic prints the Gpg4win entries actually present with
# their publishers, so a single CI run settles it.
$softwarePublisher = "The Gpg4win Project"

# Gpg4win bundles GnuPG, so the same daemons stay resident (gpg-agent, dirmngr,
# keyboxd, scdaemon) along with Kleopatra. They hold file locks that make the
# uninstall fail, and because "Start-Process -Wait" waits for descendants as well
# as the process itself, anything the uninstaller re-spawns would block this
# script indefinitely. Stop them first, then wait only on the uninstaller.
$leftovers = @("gpg-agent", "dirmngr", "keyboxd", "scdaemon", "gpg-connect-agent", "gpgconf", "kleopatra", "gpgme-w32spawn")
$timeoutSeconds = 300

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
# The uninstall info is written with SHCTX, so it can land in the per-user hive.
# The install script already accounts for that; mirror it here.
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

# Print every Gpg4win-ish entry with its publisher, so a publisher mismatch names
# the correct value instead of just failing.
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
        # Nothing to remove is not a failure: uninstall scripts are idempotent here,
        # as in nordpass_uninstall.ps1 and windsurf_uninstall.ps1. If the app *is*
        # still installed the validator's own post-uninstall check catches it.
        Write-Host "Uninstall entry not found for '$softwareName' with publisher '$softwarePublisher'."
        Exit 0
    }

    $uninstallString = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
    Write-Host "Uninstall string: $uninstallString"

    # Parse the executable path, handling quoted paths, unquoted paths containing
    # spaces, and bare tokens.
    $uninstallCommand = $uninstallString
    if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
        $uninstallCommand = $Matches[1]
    } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        $uninstallCommand = $Matches[1]
    } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
        $uninstallCommand = $Matches[1]
    }

    # NSIS uninstallers copy themselves to %TEMP% and relaunch by default, so the
    # process we start exits immediately while the real uninstall runs detached.
    # "_?=<dir>" runs it in place instead, which makes it synchronous, and it has
    # to be the last argument. Other scripts here pass it as its own ArgumentList
    # element (bdash, binance, canva) or quoted (android_studio); this passes one
    # argument string, which is what was validated in CI for this installer. Note
    # not every NSIS uninstaller accepts it -- DBeaver's returns exit 2 and
    # Logitech Unifying's exit 10, so both of those omit it.
    $installDir = Split-Path -Parent $uninstallCommand
    $uninstallArgs = "/S _?=$installDir"

    Write-Host "Uninstall command: $uninstallCommand"
    Write-Host "Uninstall args: $uninstallArgs"

    $process = Start-Process -FilePath $uninstallCommand -ArgumentList $uninstallArgs -PassThru
    # Touch .Handle so the exit code is still readable after the process ends:
    # Start-Process -PassThru otherwise returns $null for .ExitCode.
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

# Stop anything the uninstaller restarted, then wait for the ARP entry to clear.
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
