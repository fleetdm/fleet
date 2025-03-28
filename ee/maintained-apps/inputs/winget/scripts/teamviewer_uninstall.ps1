# Define acceptable/expected exit codes
$ExpectedExitCodes = @(0)

# Uninstall Registry Key
$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\TeamViewer'

# Additional uninstall args
$uninstallArgs = "/S"

# Initialize exit code
$exitCode = 0

try {
    $key = Get-ItemProperty -Path $machineKey -ErrorAction Stop

    # Get the uninstall command. Some uninstallers do not include 'QuietUninstallString'
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
            Throw "Uninstall command contains multiple quoted strings. Please update the uninstall script.`nUninstall command: $uninstallCommand"
        }
        $uninstallCommand = $splitArgs[1]
    }

    Write-Host "Uninstall command: $uninstallCommand"
    Write-Host "Uninstall args: $uninstallArgs"

    $processOptions = @{
        FilePath = $uninstallCommand
        PassThru = $true
        Wait     = $true
    }
    if ($uninstallArgs -ne '') {
        $processOptions.ArgumentList = "$uninstallArgs"
    }

    # Start uninstall process
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

# Treat acceptable exit codes as success
if ($ExpectedExitCodes -contains $exitCode) {
    Exit 0
} else {
    Exit $exitCode
}
