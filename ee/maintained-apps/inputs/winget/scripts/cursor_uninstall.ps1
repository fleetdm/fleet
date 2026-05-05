# Attempts to locate Cursor's uninstaller from registry and execute it silently

$displayName = "Cursor"
$publisher = "Anysphere"

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

# Kill any running Cursor processes before uninstalling
Stop-Process -Name "Cursor" -Force -ErrorAction SilentlyContinue

$uninstallCommand = $uninstall.UninstallString
$uninstallArgs = "/VERYSILENT /NORESTART"

# Parse the uninstall command to separate executable from existing arguments
$splitArgs = $uninstallCommand.Split('"')
if ($splitArgs.Length -gt 1) {
    if ($splitArgs.Length -eq 3) {
        $existingArgs = $splitArgs[2].Trim()
        if ($existingArgs -ne '') {
            $uninstallArgs = "$existingArgs $uninstallArgs"
        }
    } elseif ($splitArgs.Length -gt 3) {
        Write-Host "Error: Uninstall command contains multiple quoted strings"
        Exit 1
    }
    $uninstallCommand = $splitArgs[1]
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
    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
