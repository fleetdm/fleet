# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# The downloaded file is a bootstrapper that pulls the multi-GB payload during
# this step. --wait is required: without it the bootstrapper forks the real
# install to a background process and returns before it finishes.
#
# The bootstrapper's default command only creates new instances, so it fails
# when Visual Studio Professional 2022 is already installed. An existing
# instance is updated in place instead.

$exeFilePath = "${env:INSTALLER_PATH}"
$vswhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"

try {

if (-not (Test-Path $exeFilePath)) {
  Write-Host "Error: Installer file not found at: $exeFilePath"
  Exit 1
}

$installPath = $null
if (Test-Path $vswhere) {
  # -version is required as well as -products, since product IDs are not
  # version specific and an unscoped query also matches Visual Studio 2026.
  $installPath = & $vswhere -products Microsoft.VisualStudio.Product.Professional -version "[17.0,18.0)" -property installationPath
  if ($LASTEXITCODE -ne 0) {
    Write-Host "vswhere.exe failed with exit code $LASTEXITCODE"
    Exit 1
  }
  $installPath = ($installPath | Select-Object -First 1)
}

if ($installPath) {
  Write-Host "Found Visual Studio Professional 2022 at: $installPath, updating it"
  $arguments = "update --installPath `"$installPath`" --quiet --wait --norestart"
} else {
  $arguments = "--quiet --wait --norestart"
}

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = $arguments
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
  Write-Host "Install exit code: $exitCode (succeeded, reboot required to finish)"
  Exit 0
}

if ($exitCode -eq 1001 -or $exitCode -eq 1618) {
  Write-Host "Install failed: another Visual Studio Installer operation is already in progress (exit code $exitCode)"
  Exit 1
}

Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
