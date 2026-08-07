# Visual Studio has no normal UninstallString - it's removed through the
# shared Visual Studio Installer, which needs the specific instance's install
# path. All three editions (Community/Professional/Enterprise) can be
# installed side by side on one host, each with its own sibling script, so
# vswhere is scoped to this edition's product ID only.
#
# -version is required as well as -products: product IDs are not version
# specific, so an unscoped query also matches Visual Studio 2026 (major
# version 18) and would uninstall the wrong product on a host that has both.

$vswhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"
$vsInstaller = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\setup.exe"

try {

if (-not (Test-Path $vswhere)) {
  Write-Host "vswhere.exe not found at $vswhere - Visual Studio Installer is not present"
  Exit 1
}

if (-not (Test-Path $vsInstaller)) {
  Write-Host "setup.exe not found at $vsInstaller - Visual Studio Installer is not present"
  Exit 1
}

$installPath = & $vswhere -products Microsoft.VisualStudio.Product.Enterprise -version "[17.0,18.0)" -property installationPath
$installPath = ($installPath | Select-Object -First 1)

if (-not $installPath) {
  Write-Host "No installed Visual Studio Enterprise 2022 instance found"
  Exit 1
}

Write-Host "Found Visual Studio Enterprise 2022 at: $installPath"

# The install path always contains spaces, and Start-Process joins an
# -ArgumentList array with spaces without quoting the elements, so the path
# has to be quoted here or the installer parses it as several arguments.
#
# No --wait either: it's a bootstrapper-only switch and the installer rejects
# it. Start-Process -Wait already blocks until the uninstall exits.
$processOptions = @{
  FilePath = $vsInstaller
  ArgumentList = "uninstall --installPath `"$installPath`" --quiet --norestart"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

# TEMP DIAGNOSTICS - remove before merge.
if ($exitCode -ne 0) {
  Write-Host "----- diagnostics -----"
  Write-Host "cmd: $vsInstaller uninstall --installPath \"$installPath\" --quiet --norestart"
  Write-Host "whoami: $(whoami)"
  Write-Host "elevated: $(([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator))"
  Write-Host "--- vswhere -all instances ---"
  & $vswhere -all -prerelease -format value -property installationPath 2>&1 | ForEach-Object { Write-Host $_ }
  $since = (Get-Date).AddMinutes(-6)
  Get-ChildItem "$env:TEMP\dd_*.log" -ErrorAction SilentlyContinue |
    Where-Object { $_.LastWriteTime -gt $since } |
    Sort-Object LastWriteTime |
    ForEach-Object {
      Write-Host "--- $($_.Name) [matched lines] ---"
      Select-String -Path $_.FullName -Pattern 'error|fail|exception|denied|elevat|cannot|unable|refus|not found|invalid|abort' -ErrorAction SilentlyContinue |
        Select-Object -Last 25 | ForEach-Object { Write-Host $_.Line }
      Write-Host "--- $($_.Name) [last 25] ---"
      Get-Content $_.FullName -Tail 25 -ErrorAction SilentlyContinue | ForEach-Object { Write-Host $_ }
    }
  Write-Host "-----------------------"
}

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
