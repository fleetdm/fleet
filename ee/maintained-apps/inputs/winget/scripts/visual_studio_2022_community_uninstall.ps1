# Visual Studio has no normal UninstallString - it's removed through the
# shared Visual Studio Installer, which needs the specific instance's install
# path. All three editions (Community/Professional/Enterprise) can be
# installed side by side on one host, each with its own sibling script, so
# vswhere is scoped to this edition's product ID only.

$vswhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"
$vsInstaller = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vs_installer.exe"

try {

if (-not (Test-Path $vswhere)) {
  Write-Host "vswhere.exe not found at $vswhere - Visual Studio Installer is not present"
  Exit 1
}

$installPath = & $vswhere -products Microsoft.VisualStudio.Product.Community -property installationPath
$installPath = ($installPath | Select-Object -First 1)

if (-not $installPath) {
  Write-Host "No installed Visual Studio Community 2022 instance found"
  Exit 1
}

Write-Host "Found Visual Studio Community 2022 at: $installPath"

# No --wait here: it's a bootstrapper-only switch and the installer rejects it
# with exit 87. Start-Process -Wait already blocks until the uninstall exits.
$processOptions = @{
  FilePath = $vsInstaller
  ArgumentList = @('uninstall', '--installPath', $installPath, '--quiet', '--norestart')
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

# 3010/1641: uninstall succeeded but a reboot is pending/was triggered. Fleet
# treats any nonzero exit as failed, so map these to success.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
  Write-Host "Uninstall exit code: $exitCode (succeeded, reboot required to finish)"
  Exit 0
}

# 1001/1618: another Visual Studio Installer operation is already running.
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
