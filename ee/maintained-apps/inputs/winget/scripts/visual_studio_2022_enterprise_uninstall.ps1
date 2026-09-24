# Visual Studio has no UninstallString; it's removed through the shared Visual
# Studio Installer, which needs the specific instance's install path. -version
# is required as well as -products, since product IDs are not version specific
# and an unscoped query also matches a side-by-side Visual Studio 2026.

$vswhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"
$vsInstaller = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\setup.exe"

try {

if (-not (Test-Path $vswhere) -or -not (Test-Path $vsInstaller)) {
  Write-Host "Visual Studio Installer not present, nothing to uninstall"
  Exit 0
}

$installPath = & $vswhere -products Microsoft.VisualStudio.Product.Enterprise -version "[17.0,18.0)" -property installationPath

# vswhere reports failure through the exit code rather than by throwing, so
# without this check a failed query looks like "no instance installed".
if ($LASTEXITCODE -ne 0) {
  Write-Host "vswhere.exe failed with exit code $LASTEXITCODE"
  Exit 1
}

$installPath = ($installPath | Select-Object -First 1)

if (-not $installPath) {
  Write-Host "No Visual Studio Enterprise 2022 instance found, nothing to uninstall"
  Exit 0
}

Write-Host "Found Visual Studio Enterprise 2022 at: $installPath"

# The install path always contains spaces, so it has to be quoted here. No
# --wait: it's bootstrapper-only, and Start-Process -Wait already blocks.
$processOptions = @{
  FilePath = $vsInstaller
  ArgumentList = "uninstall --installPath `"$installPath`" --quiet --norestart"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
  Write-Host "Uninstall exit code: $exitCode (succeeded, reboot required to finish)"
  Exit 0
}

if ($exitCode -eq 1001 -or $exitCode -eq 1618) {
  Write-Host "Uninstall failed: another Visual Studio Installer operation is already in progress (exit code $exitCode)"
  Exit 1
}

Write-Host "Uninstall exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
