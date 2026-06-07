# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Splashtop Business ships as a setup.exe that extracts and runs an MSI.
# Per Splashtop's deployment docs, the setup.exe silent switches are:
#   prevercheck /s /i hidewindow=1,confirm_d=0
# (prevercheck and /i are required; /s = silent; commas separate options with no spaces)

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "prevercheck /s /i hidewindow=1,confirm_d=0"
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    Write-Host "Starting Splashtop Business install with: $($processOptions.ArgumentList)"
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
        Write-Host "Install succeeded (reboot required/initiated)."
        Exit 0
    }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
