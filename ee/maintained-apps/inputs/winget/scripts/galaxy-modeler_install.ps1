# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Galaxy Modeler is an electron-builder (NSIS) installer; /S runs silent and
# /allusers forces the machine-wide install per the winget machine-scope switch.
#
# Setup stops on a "Can't modify Galaxy Modeler's files" prompt while the app is
# open, and nobody can answer it from Fleet's SYSTEM context, so the install
# never returns. Stop the app first. Unsaved models are lost.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$programFiles = if (${env:ProgramW6432}) { ${env:ProgramW6432} } else { ${env:ProgramFiles} }
$installDir = Join-Path $programFiles "Galaxy Modeler"
$running = @(Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -and $_.Path.StartsWith("$installDir\", [System.StringComparison]::OrdinalIgnoreCase) })
foreach ($proc in $running) {
    Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
}
$running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

$process = Start-Process -FilePath "$exeFilePath" -ArgumentList "/S /allusers" -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

# 0 = success, 3010 = success but reboot required, 1641 = reboot initiated
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
