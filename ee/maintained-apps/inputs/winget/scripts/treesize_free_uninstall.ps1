$softwareName = "TreeSize Free"
$softwareNameLike = "*$softwareName*"
$uninstallArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$exitCode = 0

try {
    [array]$uninstallKeys = Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

    $foundUninstaller = $false
    foreach ($key in $uninstallKeys) {
        if ($key.DisplayName -like $softwareNameLike) {
            $foundUninstaller = $true
            $uninstallCommand = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
            if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
                $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
            } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
                $uninstallCommand = $Matches[1]; if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
            }
            Write-Host "Uninstall command: $uninstallCommand"; Write-Host "Uninstall args: $uninstallArgs"
            $processOptions = @{ FilePath = $uninstallCommand; PassThru = $true; Wait = $true }
            if ($uninstallArgs -ne '') { $processOptions.ArgumentList = $uninstallArgs }
            $process = Start-Process @$processOptions
            $exitCode = $process.ExitCode; Write-Host "Uninstall exit code: $exitCode"; break
        }
    }
    if (-not $foundUninstaller) { Write-Host "Uninstaller for '$softwareName' not found."; Exit 1 }
} catch { Write-Host "Error: $_"; Exit 1 }

Exit $exitCode
