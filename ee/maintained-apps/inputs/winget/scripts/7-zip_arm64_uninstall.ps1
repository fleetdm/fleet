# Silently uninstalls the ARM64 build of 7-Zip ("7-Zip <version> (arm64)") and
# waits for its uninstall registry entry to clear. An x64 7-Zip installed from
# the MSI is left alone.

$keyPath = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\7-Zip'
$displayNameLike = "7-Zip * (arm64)"
$timeoutSeconds = 120

try {

$key = Get-ItemProperty -Path $keyPath -ErrorAction SilentlyContinue
if (-not $key -or $key.DisplayName -notlike $displayNameLike) {
    Write-Host "Uninstall entry matching '$displayNameLike' not found."
    Exit 1
}

$uninstallCommand = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $Matches[1]
    $uninstallArgs = $Matches[2]
} else {
    $exePath = $uninstallCommand.Trim()
    $uninstallArgs = ""
}
if ($uninstallArgs -notmatch '(^|\s)/S(\s|$)') {
    $uninstallArgs = "$uninstallArgs /S".Trim()
}

Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $uninstallArgs"
$process = Start-Process -FilePath $exePath -ArgumentList $uninstallArgs -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"
if ($exitCode -ne 0) {
    Exit $exitCode
}

$deadline = (Get-Date).AddSeconds($timeoutSeconds)
while ((Get-Date) -lt $deadline) {
    $still = Get-ItemProperty -Path $keyPath -ErrorAction SilentlyContinue
    if (-not $still -or $still.DisplayName -notlike $displayNameLike) {
        Write-Host "7-Zip (arm64) removed."
        Exit 0
    }
    Start-Sleep -Seconds 2
}

Write-Host "7-Zip (arm64) is still registered after $timeoutSeconds seconds."
Exit 1

} catch {
    Write-Host "Error: $_"
    Exit 1
}
