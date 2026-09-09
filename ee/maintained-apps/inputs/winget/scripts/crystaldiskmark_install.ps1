# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# CrystalDiskMark ships as an Inno Setup installer (AppId "CrystalDiskMark9"),
# which registers as "CrystalDiskMark <version>" in Add/Remove Programs.

$exeFilePath = "${env:INSTALLER_PATH}"

$installTimeoutSeconds = 300
$registrationTimeoutSeconds = 60

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$publisher = "Crystal Dew World"

function Get-CrystalDiskMarkEntry {
    Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "CrystalDiskMark *" -and $_.Publisher -like "$publisher*" } |
        Select-Object -First 1
}

try {
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    # -Wait also waits on descendants, so wait on the installer process alone.
    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"
        PassThru = $true
        NoNewWindow = $true
    }
    $process = Start-Process @processOptions
    # Keeps .ExitCode readable after the process ends.
    $null = $process.Handle

    $killed = $false
    if (-not $process.WaitForExit($installTimeoutSeconds * 1000)) {
        Write-Host "Installer process did not exit within ${installTimeoutSeconds}s, stopping it."
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        $null = $process.WaitForExit(30 * 1000)
        $killed = $true
    }

    $exitCode = $null
    if ($process.HasExited) {
        $exitCode = $process.ExitCode
        Write-Host "Install exit code: $exitCode"
    }

    # The installer can return before the ARP entry is written.
    $elapsed = 0
    while (-not (Get-CrystalDiskMarkEntry) -and ($elapsed -lt $registrationTimeoutSeconds)) {
        Start-Sleep -Seconds 5
        $elapsed += 5
        Write-Host "Waiting for CrystalDiskMark to register... ($elapsed seconds)"
    }

    $entry = Get-CrystalDiskMarkEntry
    if (-not $entry) {
        Write-Host "CrystalDiskMark did not register in Add/Remove Programs."
        Exit 1
    }
    Write-Host "Registered '$($entry.DisplayName)', version $($entry.DisplayVersion)."

    # Registration above is the success signal; a killed process's code means nothing.
    if ($killed -or $null -eq $exitCode) { Exit 0 }

    # 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
    if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
