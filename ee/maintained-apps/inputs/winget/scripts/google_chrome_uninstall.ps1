$softwareName = "Google Chrome"

$uninstallArgs = "--uninstall"

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
            $uninstallCommand = if ($key.QuietUninstallString) {
                $key.QuietUninstallString
            } else {
                $key.UninstallString
            }

            # Split the command and args
            $splitArgs = $uninstallCommand.Split('"')
            if ($splitArgs.Length -gt 1) {
                if ($splitArgs.Length -eq 3) {
                    $uninstallArgs = "$( $splitArgs[2] ) $uninstallArgs".Trim()
                } elseif ($splitArgs.Length -gt 3) {
                    Throw `
                        "Uninstall command contains multiple quoted strings. " +
                            "Please update the uninstall script.`n" +
                            "Uninstall command: $uninstallCommand"
                }
                $uninstallCommand = $splitArgs[1]
            }
            Write-Host "Uninstall command: $uninstallCommand"
            Write-Host "Uninstall args: $uninstallArgs"

            $processOptions = @{
                FilePath = $uninstallCommand
                PassThru = $true
                Wait = $true
                ArgumentList = "$uninstallArgs --force-uninstall".Split(' ')
                NoNewWindow = $true
            }
            
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