# AOMEI unified the ARP DisplayName across editions around v7.4: the registry
# entry reads "AOMEI Backupper", with no "Standard" suffix. Matching the catalog
# name here finds nothing, so match the DisplayName the installer actually writes.
#
# Match the DisplayName exactly and require the publisher, mirroring the manifest's
# exists query. A substring match would also select AOMEI add-ons whose DisplayName
# merely contains this one, and uninstall the wrong product. The publisher was
# verified against the installer's PE version resource (CompanyName) -- trailing
# period included -- and requiring it here means CI exercises the same value the
# exists query depends on, which the validator's name-only lookup never does.
$softwareName = "AOMEI Backupper"
$softwarePublisher = "AOMEI International Network Limited."
$uninstallArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$exitCode = 0

try {
    [array]$uninstallKeys = Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

    $foundUninstaller = $false
    foreach ($key in $uninstallKeys) {
        if ($key.DisplayName -eq $softwareName -and $key.Publisher -eq $softwarePublisher) {
            $foundUninstaller = $true
            $uninstallCommand = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
            if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
                $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
            } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
                $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
            } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
                $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
            }
            Write-Host "Uninstall command: $uninstallCommand"; Write-Host "Uninstall args: $uninstallArgs"
            $processOptions = @{ FilePath = $uninstallCommand; PassThru = $true; Wait = $true }
            if ($uninstallArgs -ne '') { $processOptions.ArgumentList = $uninstallArgs }
            $process = Start-Process @processOptions
            $exitCode = $process.ExitCode; Write-Host "Uninstall exit code: $exitCode"; break
        }
    }
    # Nothing to remove is not a failure: uninstall scripts are idempotent here, as
    # in nordpass_uninstall.ps1 and windsurf_uninstall.ps1.
    if (-not $foundUninstaller) { Write-Host "Uninstall entry not found for '$softwareName'."; Exit 0 }
} catch { Write-Host "Error: $_"; Exit 1 }

Exit $exitCode
