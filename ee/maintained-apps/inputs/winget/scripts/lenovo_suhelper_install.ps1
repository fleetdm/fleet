# SUHelper ships inside a large Lenovo Commercial Vantage zip. This script extracts
# only SystemUpdate\SUHelperSetup.exe (an Inno Setup installer) and runs it silently.

$zipFilePath = "${env:INSTALLER_PATH}"
$extractPath = Join-Path $env:TEMP "SUHelperInstall"

try {
    if (Test-Path $extractPath) { Remove-Item -Path $extractPath -Recurse -Force }
    New-Item -ItemType Directory -Path $extractPath | Out-Null

    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $zip = [System.IO.Compression.ZipFile]::OpenRead($zipFilePath)
    try {
        $entry = $zip.Entries | Where-Object { $_.FullName -eq 'SystemUpdate/SUHelperSetup.exe' } | Select-Object -First 1
        if (-not $entry) { Write-Host "SUHelperSetup.exe not found in the zip."; Exit 1 }
        $setupPath = Join-Path $extractPath "SUHelperSetup.exe"
        [System.IO.Compression.ZipFileExtensions]::ExtractToFile($entry, $setupPath, $true)
    } finally {
        $zip.Dispose()
    }

    $process = Start-Process -FilePath $setupPath `
        -ArgumentList "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART" `
        -PassThru -Wait
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    Remove-Item -Path $extractPath -Recurse -Force -ErrorAction SilentlyContinue

    # 3010 (reboot required) and 1641 (reboot initiated) are successful installs.
    if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
    Exit $exitCode
} catch {
    Write-Host "Error: $_"
    Exit 1
}
