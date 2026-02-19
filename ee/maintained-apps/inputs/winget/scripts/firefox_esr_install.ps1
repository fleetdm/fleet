# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"
$timeoutSeconds = 240

try {

Write-Host "Installer path: $exeFilePath"
Write-Host "File exists: $(Test-Path $exeFilePath)"

# Check for any Firefox-related processes
$mozProcs = Get-Process | Where-Object { $_.Name -match "firefox|mozilla|maintenance" } | Select-Object Name, Id
if ($mozProcs) {
    Write-Host "Found Mozilla-related processes:"
    $mozProcs | ForEach-Object { Write-Host "  $($_.Name) (PID: $($_.Id))" }
} else {
    Write-Host "No Mozilla-related processes found"
}

# Check for existing Firefox directories
$ffDir = "C:\Program Files\Mozilla Firefox"
Write-Host "Existing Firefox dir present: $(Test-Path $ffDir)"

Write-Host "Starting installer with /S flag..."
$process = Start-Process -FilePath "$exeFilePath" -ArgumentList "/S" -PassThru
Write-Host "Installer PID: $($process.Id)"

$completed = $process.WaitForExit($timeoutSeconds * 1000)

if (-not $completed) {
    Write-Host "Timed out after $timeoutSeconds seconds"
    # Dump what processes are running
    Write-Host "Mozilla-related processes still running:"
    Get-Process | Where-Object { $_.Name -match "firefox|mozilla|setup|maintenance" } | ForEach-Object {
        Write-Host "  $($_.Name) (PID: $($_.Id))"
    }
    $process.Kill()
    Exit 1
}

$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
