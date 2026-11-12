# Uninstalls Adobe Acrobat Pro (64-bit) by its MSI product code. Acrobat Reader 64-bit
# registers the same DisplayName and publisher, so matching on name could remove Reader.

$productCode = $PACKAGE_ID

if (-not (Test-Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\$productCode")) {
    Write-Host "Uninstall entry not found"
    Exit 0
}

$acrobatProcesses = @("Acrobat", "AcroRd32", "RdrCEF", "AdobeCollabSync")
foreach ($proc in $acrobatProcesses) {
    Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
}

$uninstallCommand = "MsiExec.exe"
$uninstallArgs = "/X $productCode /qn /norestart"

Write-Host "Uninstall command: $uninstallCommand"
Write-Host "Uninstall args: $uninstallArgs"

try {
    $processOptions = @{
        FilePath = $uninstallCommand
        ArgumentList = $uninstallArgs
        NoNewWindow = $true
        PassThru = $true
        Wait = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    Write-Host "Uninstall exit code: $exitCode"

    $timeout = 60
    $elapsed = 0
    while ((Get-Process -Name "msiexec" -ErrorAction SilentlyContinue) -and ($elapsed -lt $timeout)) {
        Start-Sleep -Seconds 2
        $elapsed += 2
        Write-Host "Waiting for MsiExec to complete... ($elapsed seconds)"
    }

    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
