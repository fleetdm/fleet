# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Setup stops on a "Can't modify NoSQL Workbench's files" prompt while the app
# is open, and nobody can answer it from Fleet's SYSTEM context, so the install
# never returns. Stop the app first. Unsaved work is lost.
$programFiles = if (${env:ProgramW6432}) { ${env:ProgramW6432} } else { ${env:ProgramFiles} }
$installDir = Join-Path $programFiles "NoSQL Workbench"
$running = @(Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -and $_.Path.StartsWith("$installDir\", [System.StringComparison]::OrdinalIgnoreCase) })
foreach ($proc in $running) {
    Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
}
$running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

# NoSQL Workbench uses an NSIS (nullsoft) installer; /allusers for machine scope
$processOptions = @{
  FilePath     = "$exeFilePath"
  ArgumentList = "/S /allusers"
  PassThru     = $true
  Wait         = $true
  NoNewWindow  = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"
Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
