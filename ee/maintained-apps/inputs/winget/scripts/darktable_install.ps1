# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# darktable ships as an Inno Setup installer (NOT NSIS, despite winget's
# metadata reporting "nullsoft"). Inno Setup ignores the NSIS "/S" switch and
# launches its GUI wizard, which hangs forever on a headless host. Use Inno's
# silent switches instead.
#
# The installer defaults to PrivilegesRequired=admin, so it installs machine-wide
# when run elevated. "/ALLUSERS" is intentionally omitted: darktable's installer
# sets PrivilegesRequiredOverridesAllowed=dialog (not "commandline"), so the
# command-line override is not accepted and the admin default already covers all
# users.

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    # 3010 = success, reboot required.
    if ($exitCode -eq 3010) {
        Exit 0
    }

    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
