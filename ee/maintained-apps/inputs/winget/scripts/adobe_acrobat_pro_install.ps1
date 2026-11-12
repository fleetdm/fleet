# Adobe Acrobat Pro: the download is a zip containing the base AcroPro.msi (an older
# base build, e.g. 21.x) PLUS a large AcrobatDCx64Upd*.msp patch that upgrades it to
# the current release (e.g. 26.x). Adobe's setup.exe reads setup.ini and applies the
# base MSI + patch + language transform in one pass, which is the ONLY way to land the
# version advertised in the winget manifest. Running AcroPro.msi directly would install
# the stale base build and silently skip the patch, so we install via setup.exe.
# Silent switches per the winget InstallerSwitches / Adobe docs: /sAll /rs /msi.

$zipFilePath = "${env:INSTALLER_PATH}"

try {
    $extractPath = Join-Path $env:TEMP "AdobeAcrobatProInstall"

    if (Test-Path $extractPath) {
        Remove-Item -Path $extractPath -Recurse -Force
    }

    Expand-Archive -Path $zipFilePath -DestinationPath $extractPath -Force

    $setupExe = Join-Path $extractPath "Adobe Acrobat\setup.exe"
    if (-not (Test-Path $setupExe)) {
        $found = Get-ChildItem -Path $extractPath -Filter "setup.exe" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
        if ($found) { $setupExe = $found.FullName }
    }

    if (-not (Test-Path $setupExe)) {
        Write-Host "Error: setup.exe not found under $extractPath"
        Exit 1
    }

    $process = Start-Process -FilePath $setupExe -ArgumentList "/sAll /rs /msi EULA_ACCEPT=YES" -PassThru -Wait
    $exitCode = $process.ExitCode
    Write-Host "Install exit code (setup.exe): $exitCode"

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
