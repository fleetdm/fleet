# Attempts to locate Adobe Acrobat Reader uninstaller from registry and execute it silently

$displayName = "Adobe Acrobat (64-bit)"
$publisher = "Adobe"

$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
    $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
        $_.DisplayName -and ($_.DisplayName -eq $displayName -or $_.DisplayName -like "$displayName*") -and ($publisher -eq "" -or $_.Publisher -eq $publisher)
    }
    if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
    Write-Host "Uninstall entry not found"
    Exit 0
}

# Kill any running Acrobat Reader processes before uninstalling
$acrobatProcesses = @("AcroRd32", "Acrobat", "RdrCEF")
foreach ($proc in $acrobatProcesses) {
    Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
}

$uninstallCommand = $uninstall.UninstallString

# Adobe typically uses MsiExec for uninstall
# Parse the command to extract the product code
if ($uninstallCommand -match "MsiExec\.exe\s+/[IX]\s*(\{[A-F0-9-]+\})") {
    $productCode = $Matches[1]
    $uninstallArgs = "/X $productCode /qn /norestart"
    $uninstallCommand = "MsiExec.exe"
} else {
    Write-Host "Error: Unable to parse uninstall command: $uninstallCommand"
    Exit 1
}

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

    # Wait for any remaining MsiExec processes to complete
    # MsiExec can return before the uninstall is fully complete
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
