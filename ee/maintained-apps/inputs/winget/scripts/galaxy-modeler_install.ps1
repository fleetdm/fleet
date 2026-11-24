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

# A machine-wide install can be registered outside Program Files, and setup
# updates it where it is, so take the directories to match from the registry
# as well as the default.
$programFiles = if (${env:ProgramW6432}) { ${env:ProgramW6432} } else { ${env:ProgramFiles} }
$installDirs = @(Join-Path $programFiles "Galaxy Modeler")
$installDirs += @(Get-ChildItem -Path @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
    ) -ErrorAction SilentlyContinue |
    ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
    Where-Object { $_.DisplayName -like 'Galaxy Modeler*' -and $_.InstallLocation } |
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
    Write-Host "Error: Galaxy Modeler is still running ($(($stillRunning | ForEach-Object { "$($_.Name) ($($_.Id))" }) -join ', ')). Quit it and try again."
    Exit 1
}

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
