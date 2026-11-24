# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Setup can't replace Positron's files while they're in use, so every process
# running from the install folder is stopped first. Unsaved changes are lost.
$installDir = Join-Path $env:ProgramFiles "Positron"
$entry = Get-ItemProperty -Path @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
) -ErrorAction SilentlyContinue |
    Where-Object { $_.DisplayName -eq "Positron" -and $_.Publisher -eq "Posit Software, PBC" -and $_.InstallLocation } |
    Select-Object -First 1
if ($entry) { $installDir = "$($entry.InstallLocation)".TrimEnd('\') }

$running = @(Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -and $_.Path.StartsWith("$installDir\", [System.StringComparison]::OrdinalIgnoreCase) })
foreach ($proc in $running) {
    Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
}
$running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

# Inno Setup installer. /MERGETASKS=!runcode keeps Setup from launching
# Positron when it finishes.
function Invoke-Setup {
    $process = Start-Process -FilePath "$exeFilePath" -PassThru `
        -ArgumentList "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /MERGETASKS=!runcode"
    # Reading the handle keeps ExitCode available after the process exits.
    $null = $process.Handle
    $process.WaitForExit()
    return $process.ExitCode
}

$exitCode = Invoke-Setup

# A file can stay locked for a moment after the app is stopped. Setup rolls
# back cleanly on exit code 5, so retry once.
if ($exitCode -eq 5 -and $running) {
    Write-Host "Install exit code: 5. Retrying in 10 seconds."
    Start-Sleep -Seconds 10
    $exitCode = Invoke-Setup
}

Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
