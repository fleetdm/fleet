# Learn more about install scripts:
# http://fleetdm.com/learn-more-about/install-scripts
#
# Audacity 4 ships as a WiX MSI that installs to "%ProgramFiles%\Audacity 4"
# and registers as "Audacity 4.0" (publisher "Audacity").
#
# Audacity 4 installs side by side with Audacity 3 (a separate Inno Setup
# product registered as "Audacity 3.x" by "Audacity Team"). After a successful
# install, remove any 3.x install so the host converges on a single Audacity.

$logFile = "${env:TEMP}/fleet-install-software.log"
$successCodes = @(0, 3010, 1641)

function Remove-LegacyAudacity3 {
    $paths = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
    )
    $legacy = $null
    foreach ($p in $paths) {
        $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
            $_.DisplayName -like 'Audacity 3*' -and
            $_.Publisher -eq 'Audacity Team' -and
            $_.UninstallString -like '*unins*.exe*'
        }
        if ($items) { $legacy = $items | Select-Object -First 1; break }
    }
    if (-not $legacy) {
        Write-Host "No Audacity 3.x install found"
        return
    }

    Write-Host "Removing legacy install: $($legacy.DisplayName)"
    Stop-Process -Name "audacity" -Force -ErrorAction SilentlyContinue

    $uninstaller = $legacy.UninstallString.Trim('"')
    if (-not (Test-Path $uninstaller)) {
        Write-Host "Warning: legacy uninstaller not found at $uninstaller"
        return
    }

    $process = Start-Process -FilePath $uninstaller `
        -ArgumentList "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART" `
        -PassThru -Wait -NoNewWindow
    Write-Host "Legacy uninstall exit code: $($process.ExitCode)"
}

try {

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv `"${logFile}`" ALLUSERS=1 /i `"${env:INSTALLER_PATH}`"" `
  -PassThru -Verb RunAs -Wait

Get-Content $logFile -Tail 500

if ($successCodes -notcontains $installProcess.ExitCode) {
  Exit $installProcess.ExitCode
}

Remove-LegacyAudacity3

Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1
}
