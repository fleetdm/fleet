$exeFilePath = "${env:INSTALLER_PATH}"

# Define an array of common silent install parameters to try, including no parameter
$silentParams = @("", "/S", "/s", "/silent", "/quiet", "-s", "--silent", "/SILENT", "/VERYSILENT")

$installSuccess = $false
$finalExitCode = 1  # Default to failure

function Try-Install($param) {
    try {
        Write-Host "Attempting installation with parameter: '$param'"
        $processOptions = @{
            FilePath = "$exeFilePath"
            ArgumentList = $param
            PassThru = $true
            Wait = $true
        }
        $process = Start-Process @processOptions
        $exitCode = $process.ExitCode
        Write-Host "Install exit code: $exitCode"
        return $exitCode
    } catch {
        Write-Host "Error running installer with parameter '$param': $_"
        return $null
    }
}

foreach ($param in $silentParams) {
    $exitCode = Try-Install $param
    if ($exitCode -eq 0) {
        Write-Host "Installation successful with parameter: '$param'"
        $installSuccess = $true
        $finalExitCode = 0
        break
    } elseif ($exitCode -eq $null) {
        Write-Host "Installer crashed or could not be started with parameter: '$param'"
    } else {
        Write-Host "Installation with parameter '$param' failed. Trying next parameter..."
    }
}

if (-not $installSuccess) {
    Write-Host "All installation attempts failed."
}

Exit $finalExitCode
