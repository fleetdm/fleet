# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

try {
    # Verify installer file exists
    if (-not (Test-Path $exeFilePath)) {
        Write-Host "Error: Installer file not found at: $exeFilePath"
        Exit 1
    }

    Write-Host "Installing AnyDesk from: $exeFilePath"

    # AnyDesk silent install switches (from the winget manifest InstallerSwitches):
    #   --install "<dir>" --silent  installs machine-wide into the given directory
    #   --create-shortcuts / --create-desktop-icon / --update-auto are AnyDesk's
    #   documented custom switches.
    $installDir = "C:\Program Files (x86)\AnyDesk"
    $argumentList = "--install `"$installDir`" --silent --create-shortcuts --create-desktop-icon --update-auto"

    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = $argumentList
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    Write-Host "Starting installation with arguments: $argumentList"
    $process = Start-Process @processOptions

    if ($null -eq $process) {
        Write-Host "Error: Failed to start installer process"
        Exit 1
    }

    $exitCode = $process.ExitCode

    # Prints the exit code
    Write-Host "Install exit code: $exitCode"
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Write-Host "Error details: $($_.Exception.Message)"
    Exit 1
}
