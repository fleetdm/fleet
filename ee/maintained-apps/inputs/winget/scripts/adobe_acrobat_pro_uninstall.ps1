# Locate Adobe Acrobat Pro uninstaller from registry and execute it silently.
# DisplayName is "Adobe Acrobat DC (64-bit)" on DC installs and may be "Adobe Acrobat (64-bit)" on others.

$displayNames = @("Adobe Acrobat (64-bit)", "Adobe Acrobat DC (64-bit)")
$publisher = "Adobe"

$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
    $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
        $dn = $_.DisplayName
        if (-not $dn) { return $false }
        if ($publisher -ne "" -and $_.Publisher -ne $publisher) { return $false }
        foreach ($d in $displayNames) {
            if ($dn -eq $d -or $dn -like "$d*") { return $true }
        }
        $false
    }
    if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
    Write-Host "Uninstall entry not found"
    Exit 0
}

$acrobatProcesses = @("Acrobat", "AcroRd32", "RdrCEF", "AdobeCollabSync")
foreach ($proc in $acrobatProcesses) {
    Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
}

$uninstallCommand = $uninstall.UninstallString

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
