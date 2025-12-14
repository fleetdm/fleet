# Learn more about .exe uninstall scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$softwareName = "Adobe Creative Cloud"
$uninstallArgs = "--silent" # Adobe Creative Cloud uses --silent for silent uninstall

$exitCode = 0

try {
    Write-Host "Starting Adobe Creative Cloud uninstallation..."

    $machineKeyPaths = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
    )

    $foundUninstaller = $false
    foreach ($keyPath in $machineKeyPaths) {
        $uninstallKeys = Get-ChildItem $keyPath -ErrorAction SilentlyContinue
        foreach ($key in $uninstallKeys) {
            $properties = Get-ItemProperty -Path $key.PSPath -ErrorAction SilentlyContinue
            # Match DisplayName using a wildcard for robustness
            if ($properties -and $properties.DisplayName -like "*$softwareName*") {
                $foundUninstaller = $true
                $uninstallCommand = if ($properties.QuietUninstallString) {
                    $properties.QuietUninstallString
                } else {
                    $properties.UninstallString
                }

                # Split the command and args
                $splitArgs = $uninstallCommand.Split('"')
                if ($splitArgs.Length -gt 1) {
                    if ($splitArgs.Length -eq 3) {
                        $existingArgs = $splitArgs[2].Trim()
                        if ($existingArgs -ne '') {
                            $uninstallArgs = "$existingArgs $uninstallArgs"
                        }
                    } elseif ($splitArgs.Length -gt 3) {
                        Throw "Uninstall command contains multiple quoted strings. Please update the uninstall script.`nUninstall command: $uninstallCommand"
                    }
                    $uninstallCommand = $splitArgs[1]
                }

                Write-Host "Uninstall command: $uninstallCommand"
                Write-Host "Uninstall args: $uninstallArgs"

                $processOptions = @{
                    FilePath = $uninstallCommand
                    ArgumentList = $uninstallArgs
                    PassThru = $true
                    Wait = $true
                    NoNewWindow = $true
                }

                $process = Start-Process @processOptions
                $exitCode = $process.ExitCode
                Write-Host "Uninstall exit code: $exitCode"
                break
            }
        }
        if ($foundUninstaller) { break }
    }

    if (-not $foundUninstaller) {
        Write-Host "Uninstaller for '$softwareName' not found."
        $exitCode = 0 # Exit 0 if already uninstalled
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode

