# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# FreeCAD ships as a MultiUser NSIS installer. /S runs it silently and
# /AllUsers forces the machine-wide install (HKLM + C:\Program Files).
#
# Setup refuses to run while FreeCAD is open ("You must close FreeCAD first!")
# and exits 2. Fleet runs as SYSTEM, outside the user's session, so it can't
# ask FreeCAD to close: stop it first. Unsaved documents are lost.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# Each release installs to its own "FreeCAD <major>.<minor>" directory, so take
# the directories to match from the registry rather than assuming one.
$installDirs = @(Get-ChildItem -Path @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
    ) -ErrorAction SilentlyContinue |
    ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
    Where-Object { $_.DisplayName -like 'FreeCAD*' -and $_.InstallLocation } |
    ForEach-Object { $_.InstallLocation.TrimEnd('\') })

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
# really stopped rather than letting setup abort on its open files.
$stillRunning = & $getRunning
if ($stillRunning.Count) {
    Write-Host "Error: FreeCAD is still running ($(($stillRunning | ForEach-Object { "$($_.Name) ($($_.Id))" }) -join ', ')). Quit it and try again."
    Exit 1
}

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/AllUsers /S"
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
