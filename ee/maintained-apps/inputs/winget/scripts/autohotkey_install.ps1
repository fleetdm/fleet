# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# AutoHotkey's setup is AutoHotkey running its own install script, so it only
# accepts its own switches: /silent, /elevate, /user, and /installto "<dir>".
# Any other switch opens a message box. Run with administrator rights it
# installs for all users in Program Files; /user would install per user.
# Running scripts keep working on the previous version until they restart.

$exeFilePath = "${env:INSTALLER_PATH}"
$timeoutSeconds = 300
$uninstallKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\AutoHotkey'

# 3010 = reboot required, 1641 = reboot initiated
$successCodes = @(0, 3010, 1641)

function Get-RegisteredVersion {
    $entry = Get-ItemProperty -LiteralPath $uninstallKey -ErrorAction SilentlyContinue
    if ($entry -and $entry.DisplayName -eq 'AutoHotkey') { return [string]$entry.DisplayVersion }
    return $null
}

# True when the registered version is the installer's version or newer.
function Test-Installed {
    param([string]$expected)
    $registered = Get-RegisteredVersion
    if (-not $registered) { return $false }
    if (-not $expected) { return $true }
    try {
        return ([version]$registered -ge [version]$expected)
    } catch {
        return ($registered -eq $expected)
    }
}

try {
    if (-not (Test-Path -LiteralPath $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    $expectedVersion = (Get-Item -LiteralPath $exeFilePath).VersionInfo.ProductVersion
    if ($expectedVersion) { $expectedVersion = $expectedVersion.Trim() }
    Write-Host "Installing AutoHotkey $expectedVersion"

    $process = Start-Process -FilePath $exeFilePath -ArgumentList "/silent /elevate" -PassThru
    # Touch the handle so ExitCode is still readable after the process exits.
    try { $null = $process.Handle } catch { }

    # Setup shows some errors in a dialog even with /silent (for example, when
    # antivirus blocks a file it writes). Nobody can answer it here, so stop
    # waiting after the timeout instead of hanging.
    if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
        Write-Host "AutoHotkey setup did not finish within $timeoutSeconds seconds. It is probably waiting on an error dialog; stopping it."
        & taskkill.exe /PID $process.Id /T /F 2>&1 | Write-Host
        Exit 1
    }

    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"
    if (($null -ne $exitCode) -and ($successCodes -notcontains $exitCode)) {
        if ($exitCode -gt [int]::MaxValue -or $exitCode -lt 1) { $exitCode = 1 }
        Exit $exitCode
    }

    # Setup can exit 0 without installing, so confirm the registration.
    $stopwatch = [Diagnostics.Stopwatch]::StartNew()
    while (-not (Test-Installed $expectedVersion)) {
        if ($stopwatch.Elapsed.TotalSeconds -ge 60) {
            $registered = Get-RegisteredVersion
            if ($registered) {
                Write-Host "AutoHotkey $registered is registered for all users, but $expectedVersion was expected."
            } else {
                Write-Host "AutoHotkey is not registered for all users after setup finished."
            }
            Exit 1
        }
        Start-Sleep -Seconds 2
    }

    Write-Host "AutoHotkey $(Get-RegisteredVersion) is installed for all users."
    Exit 0
} catch {
    Write-Host "Error: $_"
    Exit 1
}
