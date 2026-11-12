# Oracle VirtualBox ships a custom Oracle bootstrapper (not a WiX Burn bundle)
# that wraps an internal MSI. Silent install args:
#   --silent              = no UI
#   --ignore-reboot       = don't reboot at the bootstrapper level
#   --msiparams "..."     = pass MSI properties to the wrapped MSI
#     REBOOT=ReallySuppress         -- fully suppress reboot
#     VBOX_INSTALLDESKTOPSHORTCUT=0 -- no desktop shortcut
#     VBOX_INSTALLQUICKLAUNCHSHORTCUT=0
#     VBOX_START=0                  -- don't auto-launch after install
#
# Note: VirtualBox installs host-only / NAT network drivers. With current
# Oracle EV-signed drivers, Windows trusts these without prompting; on a
# pristine box without Oracle in the TrustedPublisher store you may need to
# pre-import the certificate, otherwise the driver install can stall.
# 3010/1641 = success but reboot was requested; treat as success.

$exeFilePath = "${env:INSTALLER_PATH}"
$ExpectedExitCodes = @(0, 3010, 1641)

try {
    $msiParams = "REBOOT=ReallySuppress VBOX_INSTALLDESKTOPSHORTCUT=0 VBOX_INSTALLQUICKLAUNCHSHORTCUT=0 VBOX_START=0"

    $processOptions = @{
        FilePath     = "$exeFilePath"
        ArgumentList = "--silent", "--ignore-reboot", "--msiparams", $msiParams
        PassThru     = $true
        Wait         = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    # Let the wrapped MSI and any kernel-driver install helpers settle so
    # detection sees a finished state and the temp installer can be cleaned
    # up (avoids the "Access is denied" cleanup warning we saw with Power BI).
    $elapsed = 0
    while ((Get-Process -Name "VirtualBox*","msiexec","drvinst","DIFxApp*" -ErrorAction SilentlyContinue) -and ($elapsed -lt 60)) {
        Start-Sleep -Seconds 3
        $elapsed += 3
    }

    if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
