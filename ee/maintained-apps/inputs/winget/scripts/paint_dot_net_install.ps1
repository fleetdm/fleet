# Paint.NET ships as a zip containing its MSI. The vendor's .exe installer
# hangs without installing anything when run as SYSTEM, so this uses the MSI build.

$zipFilePath = "${env:INSTALLER_PATH}"

try {

$extractPath = Join-Path $env:TEMP "PaintDotNetInstall"
if (Test-Path $extractPath) { Remove-Item -Path $extractPath -Recurse -Force }
Expand-Archive -Path $zipFilePath -DestinationPath $extractPath -Force

$msi = Get-ChildItem -Path $extractPath -Filter "*.msi" -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
if (-not $msi) {
  Write-Host "Error: MSI not found under $extractPath"
  Exit 1
}

# The MSI defaults to ALLUSERS=2; pin a per-machine install.
$logFile = Join-Path $env:TEMP "PaintDotNetInstall.log"
$process = Start-Process -FilePath "msiexec.exe" `
  -ArgumentList "/i `"$($msi.FullName)`" ALLUSERS=1 /quiet /norestart /l*v `"$logFile`"" `
  -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Install exit code (msiexec): $exitCode"

Remove-Item -Path $extractPath -Recurse -Force -ErrorAction SilentlyContinue

# 3010 = success, reboot required; 1641 = success, reboot initiated.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
