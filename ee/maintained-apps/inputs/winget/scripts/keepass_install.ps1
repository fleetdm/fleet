# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# KeePass uses an Inno Setup installer: /ALLUSERS forces a machine-wide install
# (the default is per-user, which would land in the SYSTEM profile when Fleet
# runs as SYSTEM). Setup refuses to run while KeePass is open, so a running
# instance is asked to close first and force-stopped only if it refuses --
# which discards unsaved database changes.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

$running = Get-Process -Name "KeePass" -ErrorAction SilentlyContinue
if ($running) {
    Write-Host "Closing running KeePass before installing."
    $running | ForEach-Object { $_.CloseMainWindow() | Out-Null }

    $deadline = (Get-Date).AddSeconds(30)
    while ((Get-Process -Name "KeePass" -ErrorAction SilentlyContinue) -and (Get-Date) -lt $deadline) {
        Start-Sleep -Seconds 2
    }

    Get-Process -Name "KeePass" -ErrorAction SilentlyContinue | ForEach-Object {
        Write-Host "KeePass (PID $($_.Id)) did not close; stopping it. Unsaved database changes are lost."
        Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
    }
}

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/ALLUSERS /VERYSILENT /SUPPRESSMSGBOXES /NORESTART"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
