# Attempts to locate Microsoft 365 Apps uninstaller from registry and execute it silently

$displayName = "Microsoft 365 Apps for enterprise - en-us"
$publisher = "Microsoft Corporation"

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

# Kill any running Office processes before uninstalling
$officeProcesses = @("WINWORD", "EXCEL", "POWERPNT", "OUTLOOK", "ONENOTE", "MSACCESS", "MSPUB", "TEAMS")
foreach ($proc in $officeProcesses) {
    Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
}

$uninstallCommand = $uninstall.UninstallString

# Office Click-to-Run uninstall string is already properly formatted
# Example: "C:\Program Files\Common Files\Microsoft Shared\ClickToRun\OfficeClickToRun.exe" scenario=install scenariosubtype=ARP sourcetype=None productstoremove=O365ProPlusRetail.16_en-us_x-none culture=en-us version.16=16.0

# Parse the command to separate executable from arguments
$splitArgs = $uninstallCommand.Split('"')
if ($splitArgs.Length -gt 1) {
    $executablePath = $splitArgs[1]
    $arguments = if ($splitArgs.Length -gt 2) { $splitArgs[2].Trim() } else { "" }
} else {
    Write-Host "Error: Unable to parse uninstall command"
    Exit 1
}

Write-Host "Uninstall executable: $executablePath"
Write-Host "Uninstall arguments: $arguments"

try {
    $processOptions = @{
        FilePath = $executablePath
        ArgumentList = $arguments
        NoNewWindow = $true
        PassThru = $true
        Wait = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    Write-Host "Uninstall exit code: $exitCode"
    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
