# Define acceptable/expected exit codes
$ExpectedExitCodes = @(0, 19)

# Uninstall Registry Key
$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Docker Desktop'

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

    # The expected uninstall command value is "C:\Program Files\Docker\Docker\Docker Desktop Installer.exe" "uninstall"
    $splitArgs = $uninstallCommand.Split('"')
    if ($splitArgs.Length -ne 5) {
      Throw "Unexpected uninstall command. Please update the uninstall script.`nUninstall command: $uninstallCommand"
    }
    $uninstallCommand = $splitArgs[1]
    $uninstallArgs = $splitArgs[3]

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
