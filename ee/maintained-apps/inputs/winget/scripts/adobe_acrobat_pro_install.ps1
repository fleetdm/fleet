# Adobe Acrobat Pro: the download is a zip that contains both AcroPro.msi and setup.exe
# (under "Adobe Acrobat\"). Prefer the MSI for a standard silent install; fall back to setup.exe.

$zipFilePath = "${env:INSTALLER_PATH}"

try {
    $extractPath = Join-Path $env:TEMP "AdobeAcrobatProInstall"

    if (Test-Path $extractPath) {
        Remove-Item -Path $extractPath -Recurse -Force
    }

    Expand-Archive -Path $zipFilePath -DestinationPath $extractPath -Force

    $msiPath = Join-Path $extractPath "Adobe Acrobat\AcroPro.msi"
    if (-not (Test-Path $msiPath)) {
        $found = Get-ChildItem -Path $extractPath -Filter "AcroPro.msi" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
        if ($found) { $msiPath = $found.FullName }
    }

    if (Test-Path $msiPath) {
        $msiArgs = @(
            "/i", "`"$msiPath`"",
            "/qn", "/norestart",
            "EULA_ACCEPT=YES"
        )
        $process = Start-Process -FilePath "msiexec.exe" -ArgumentList $msiArgs -PassThru -Wait -NoNewWindow
        $exitCode = $process.ExitCode
        Write-Host "Install exit code (msiexec): $exitCode"
    } else {
        $setupExe = Join-Path $extractPath "Adobe Acrobat\setup.exe"
        if (-not (Test-Path $setupExe)) {
            Write-Host "Error: Neither AcroPro.msi nor Adobe Acrobat\setup.exe found under $extractPath"
            Exit 1
        }
        $process = Start-Process -FilePath $setupExe -ArgumentList "/sAll /rs /msi EULA_ACCEPT=YES" -PassThru -Wait
        $exitCode = $process.ExitCode
        Write-Host "Install exit code (setup.exe): $exitCode"
    }

    Remove-Item -Path $extractPath -Recurse -Force -ErrorAction SilentlyContinue

    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
