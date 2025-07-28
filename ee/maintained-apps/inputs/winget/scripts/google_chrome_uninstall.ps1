$softwareName = "Google Chrome"

$defaultUninstallArgs = "--uninstall --force-uninstall"

$expectedExitCodes = @(19, 20)

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
            # Get the uninstall command.
            $rawUninstallCommand = if ($key.QuietUninstallString) {
                $key.QuietUninstallString
            } else {
                $key.UninstallString
            }

            Write-Host "Raw uninstall command: $rawUninstallCommand"

            # Split the command and args
            $splitArgs = $rawUninstallCommand.Split('"')
            if ($splitArgs.Length -gt 1) {
                if ($splitArgs.Length -eq 3) {
                    $uninstallCommand = $splitArgs[1]
                    $existingArgs = $splitArgs[2].Trim()
                    # Always add --force-uninstall for silent operation
                    if ($existingArgs -match "--uninstall") {
                        $uninstallArgs = "$existingArgs --force-uninstall"
                    } else {
                        $uninstallArgs = "$existingArgs $defaultUninstallArgs".Trim()
                    }
                } elseif ($splitArgs.Length -gt 3) {
                    Throw `
                        "Uninstall command contains multiple quoted strings. " +
                            "Please update the uninstall script.`n" +
                            "Uninstall command: $rawUninstallCommand"
                }
            } else {
                $uninstallCommand = $rawUninstallCommand
                $uninstallArgs = $defaultUninstallArgs
            }
            Write-Host "Uninstall command: $uninstallCommand"
            Write-Host "Uninstall args: $uninstallArgs"

            $processOptions = @{
                FilePath = $uninstallCommand
                PassThru = $true
                Wait = $true
                ArgumentList = $uninstallArgs.Split(' ')
                NoNewWindow = $true
            }

            Write-Host "Starting process with arguments: $($uninstallArgs.Split(' ') -join ', ')"
            
            # Start process and track exit code
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

if ($expectedExitCodes -contains $exitCode) {
    $exitCode = 0
}

Exit $exitCode