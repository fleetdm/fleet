# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Setup fails with exit code 5 when Cursor's files are in use. Fleet runs as
# SYSTEM, outside the user's session, so /CLOSEAPPLICATIONS can't close it:
# stop every process running from the install folder first. Unsaved changes
# are lost.
$installDir = Join-Path $env:ProgramFiles "cursor"
$entry = Get-ItemProperty -Path @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
) -ErrorAction SilentlyContinue |
    Where-Object { $_.DisplayName -eq "Cursor" -and $_.Publisher -eq "Anysphere" -and $_.InstallLocation } |
    Select-Object -First 1
if ($entry) { $installDir = "$($entry.InstallLocation)".TrimEnd('\') }

$running = @(Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -and $_.Path.StartsWith("$installDir\", [System.StringComparison]::OrdinalIgnoreCase) })
foreach ($proc in $running) {
    Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
}
$running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

# Add arguments to install silently (Cursor uses an Inno Setup-based installer)
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/SP- /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /CLOSEAPPLICATIONS /MERGETASKS=!runcode"
  PassThru = $true
  Wait = $true
}
    
# Start process and track exit code
$process = Start-Process @processOptions
$exitCode = $process.ExitCode

# A file can stay locked for a moment after the app is stopped. Setup rolls
# back cleanly on exit code 5, so retry once.
if ($exitCode -eq 5 -and $running) {
    Write-Host "Install exit code: 5. Retrying in 10 seconds."
    Start-Sleep -Seconds 10
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
}

# Prints the exit code
Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
