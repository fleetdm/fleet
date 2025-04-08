
$machineKey = `
 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\{EA457B21-F73E-494C-ACAB-524FDE069978}_is1'
$uninstallArgs = "/VERYSILENT"
$exitCode = 0

try {

    $key = Get-ItemProperty -Path $machineKey -ErrorAction Stop

    # Get the uninstall command. Some uninstallers do not include
    # 'QuietUninstallString' and require a flag to run silently.
    $uninstallCommand = if ($key.QuietUninstallString) {
        $key.QuietUninstallString
    } else {
        $key.UninstallString
    }

    # The uninstall command may contain command and args, like:
    # "C:\Program Files\Software\uninstall.exe" --uninstall --silent
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
    }
    if ($uninstallArgs -ne '') {
        $processOptions.ArgumentList = "$uninstallArgs"
    }

    # Start process and track exit code
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    # Kill Brave process
    Stop-Process -Name "brave" -Force -ErrorAction SilentlyContinue

    # Prints the exit code
    Write-Host "Uninstall exit code: $exitCode"

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode
