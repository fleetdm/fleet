# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# The Sonos installer is a 32-bit InstallShield setup that wraps an MSI. Run as
# SYSTEM, it caches the MSI under the WOW64-redirected
# system32\config\systemprofile, where the 64-bit Windows Installer service
# can't find it (exit 1619). /b points the cache at a folder outside System32.
$cacheDir = Join-Path ([System.IO.Path]::GetTempPath()) ("sonos-" + [guid]::NewGuid())

try {

New-Item -ItemType Directory -Path $cacheDir -Force | Out-Null

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S /b`"$cacheDir`" /v`"/qn /norestart`""
  PassThru = $true
  Wait = $true
}

# Start process and track exit code
$process = Start-Process @processOptions
$exitCode = $process.ExitCode

# Prints the exit code
Write-Host "Install exit code: $exitCode"

if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
  Exit 0
}
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
} finally {
  Remove-Item -Path $cacheDir -Recurse -Force -ErrorAction SilentlyContinue
}
