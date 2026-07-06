$softwareName = "Google Earth Pro"

$uninstallArgs = ""

$machineKey = `
 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = `
 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = 0

try {

    [array]$uninstallKeys = Get-ChildItem `
        -Path @($machineKey, $machineKey32on64) `
        -ErrorAction SilentlyContinue |
            ForEach-Object { Get-ItemProperty $_.PSPath }

    $foundUninstaller = $false
    foreach ($key in $uninstallKeys) {
        if ($key.DisplayName -eq $softwareName) {
            $foundUninstaller = $true
            $uninstallCommand = if ($key.QuietUninstallString) {
                $key.QuietUninstallString
            } else {
                $key.UninstallString
            }

            # Handle quoted and unquoted uninstall strings (including paths with spaces)
            if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
                $uninstallCommand = $Matches[1]
                if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
            } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
                $uninstallCommand = $Matches[1]
                if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
            }

            Write-Host "Uninstall command: $uninstallCommand"
            Write-Host "Uninstall args: $uninstallArgs"

            $processOptions = @{
                FilePath = $uninstallCommand
                PassThru = $true
                Wait = $true
            }
            if ($uninstallArgs -ne '') {
                $processOptions.ArgumentList = $uninstallArgs
            }

            $process = Start-Process @processOptions
            $exitCode = $process.ExitCode

            Write-Host "Uninstall exit code: $exitCode"
            break
        }
    }

    if (-not $foundUninstaller) {
        Write-Host "Uninstaller for '$softwareName' not found."
        Exit 1
    }

} catch {
    Write-Host "Error: $_"
    Exit 1
}

# Google Earth Pro's uninstaller spawns child msiexec processes; wait for them to finish
$timeout = 120
$elapsed = 0
while ((Get-Process -Name "msiexec" -ErrorAction SilentlyContinue) -and ($elapsed -lt $timeout)) {
    Start-Sleep -Seconds 2
    $elapsed += 2
    Write-Host "Waiting for msiexec to complete... ($elapsed seconds)"
}

Exit $exitCode
