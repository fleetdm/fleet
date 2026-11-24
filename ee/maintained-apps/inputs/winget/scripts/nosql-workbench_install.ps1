# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Setup stops on a "Can't modify NoSQL Workbench's files" prompt while the app
# is open, and nobody can answer it from Fleet's SYSTEM context, so the install
# never returns. Stop the app first. Unsaved work is lost.
# A machine-wide install can be registered outside Program Files, and setup
# updates it where it is, so take the directories to match from the registry
# as well as the default.
$programFiles = if (${env:ProgramW6432}) { ${env:ProgramW6432} } else { ${env:ProgramFiles} }
$installDirs = @(Join-Path $programFiles "NoSQL Workbench")
$installDirs += @(Get-ChildItem -Path @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
    ) -ErrorAction SilentlyContinue |
    ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
    Where-Object { $_.DisplayName -like 'NoSQL Workbench*' -and $_.InstallLocation } |
    ForEach-Object { $_.InstallLocation.TrimEnd('\') })
$installDirs = @($installDirs | Select-Object -Unique)

$getRunning = {
    @(Get-Process -ErrorAction SilentlyContinue | Where-Object {
        $procPath = $_.Path
        if (-not $procPath) { return $false }
        foreach ($dir in $installDirs) {
            if ($procPath.StartsWith("$dir\", [System.StringComparison]::OrdinalIgnoreCase)) { return $true }
        }
        return $false
    })
}
$running = & $getRunning
foreach ($proc in $running) {
    Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
    Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
}
$running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

# Wait-Process reports a timeout as a non-terminating error, so confirm the app
# really stopped rather than letting setup wait on a prompt nobody can answer.
$stillRunning = & $getRunning
if ($stillRunning.Count) {
    Write-Host "Error: NoSQL Workbench is still running ($(($stillRunning | ForEach-Object { "$($_.Name) ($($_.Id))" }) -join ', ')). Quit it and try again."
    Exit 1
}

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
