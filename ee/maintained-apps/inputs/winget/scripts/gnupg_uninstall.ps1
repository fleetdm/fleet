$softwareName = "GNU Privacy Guard"

# GnuPG leaves background daemons resident (gpg-agent, dirmngr, keyboxd,
# scdaemon). They hold file locks that make the uninstall fail, and because
# "Start-Process -Wait" waits for descendants as well as the process itself,
# anything the uninstaller re-spawns would block this script indefinitely. Stop
# the daemons first, then wait only on the uninstaller process.
$daemons = @("gpg-agent", "dirmngr", "keyboxd", "scdaemon", "gpg-connect-agent", "gpgconf")
$timeoutSeconds = 300

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$exitCode = 0

function Get-GnuPGUninstallKey {
    Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "$softwareName*" } |
        Select-Object -First 1
}

foreach ($daemon in $daemons) {
    Stop-Process -Name $daemon -Force -ErrorAction SilentlyContinue
}

try {
    $key = Get-GnuPGUninstallKey
    if (-not $key) {
        Write-Host "Uninstaller for '$softwareName' not found."
        Exit 1
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
    # "_?=<dir>" runs it in place instead, which makes it synchronous. It must be
    # the last argument and must not be quoted, so pass one argument string rather
    # than an array (PowerShell would quote an element containing spaces).
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
foreach ($daemon in $daemons) {
    Stop-Process -Name $daemon -Force -ErrorAction SilentlyContinue
}

$elapsed = 0
while ((Get-GnuPGUninstallKey) -and ($elapsed -lt 120)) {
    Start-Sleep -Seconds 5
    $elapsed += 5
    Write-Host "Waiting for the uninstall to finish... ($elapsed seconds)"
}

if (Get-GnuPGUninstallKey) {
    Write-Host "'$softwareName' is still registered after the uninstall."
    Exit 1
}

Exit $exitCode
