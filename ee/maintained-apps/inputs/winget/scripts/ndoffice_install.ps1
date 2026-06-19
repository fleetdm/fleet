# ndOffice: the download is a zip containing the nested ndOfficeSetup.msi (the installer
# the winget manifest targets), an ndOfficeSetup.exe bootstrapper, and Adobe integration
# add-on MSIs. We install the base MSI directly via msiexec, which is what winget does.
# The MSI sets ALLUSERS=1, so it installs per-machine regardless of context.
# PRODUCT_VERSION_GTE=1 is the custom switch from the winget InstallerSwitches.
# Note: ndOffice depends on the Microsoft Visual Studio Tools for Office Runtime (VSTOR),
# which winget installs as a separate dependency. It is normally present on hosts that
# have Microsoft Office installed; if VSTOR is missing the MSI may fail.

$zipFilePath = "${env:INSTALLER_PATH}"

try {
    $extractPath = Join-Path $env:TEMP "ndOfficeInstall"

    if (Test-Path $extractPath) {
        Remove-Item -Path $extractPath -Recurse -Force
    }

    Expand-Archive -Path $zipFilePath -DestinationPath $extractPath -Force

    $msiPath = Join-Path $extractPath "ndOfficeSetup.msi"
    if (-not (Test-Path $msiPath)) {
        $found = Get-ChildItem -Path $extractPath -Filter "ndOfficeSetup.msi" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
        if ($found) { $msiPath = $found.FullName }
    }

    if (-not (Test-Path $msiPath)) {
        Write-Host "Error: ndOfficeSetup.msi not found under $extractPath"
        Exit 1
    }

    $logPath = Join-Path $env:TEMP "ndOfficeInstall.log"
    $arguments = "/i `"$msiPath`" /quiet /norestart PRODUCT_VERSION_GTE=1 /l*v `"$logPath`""

    $process = Start-Process -FilePath "msiexec.exe" -ArgumentList $arguments -PassThru -Wait
    $exitCode = $process.ExitCode
    Write-Host "Install exit code (msiexec): $exitCode"

    Remove-Item -Path $extractPath -Recurse -Force -ErrorAction SilentlyContinue

    # 3010 = success, reboot required; 1641 = success, reboot initiated.
    if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
        Exit 0
    }

    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
