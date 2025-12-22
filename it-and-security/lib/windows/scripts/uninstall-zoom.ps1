# Zoom Uninstall Script for Fleet

$exitCode = 0

# Kill Zoom processes
Get-Process -Name "Zoom*" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

# Method 1: Try CleanZoom.exe if provided with the script
$cleanZoom = "$PSScriptRoot\CleanZoom.exe"
if (Test-Path $cleanZoom) {
    Write-Host "Using CleanZoom.exe"
    & $cleanZoom /silent
    $exitCode = $LASTEXITCODE
    Exit $exitCode
}

# Method 2: Registry uninstall (both HKLM and HKCU)
$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

$found = $false
foreach ($path in $paths) {
    if (Test-Path $path) {
        Get-ItemProperty $path -ErrorAction SilentlyContinue | 
        Where-Object { $_.DisplayName -like "*Zoom*" } |
        ForEach-Object {
            $found = $true
            Write-Host "Uninstalling: $($_.DisplayName)"
            
            # Try QuietUninstallString
            if ($_.QuietUninstallString) {
                cmd /c "$($_.QuietUninstallString)" 2>&1 | Out-Null
                if ($LASTEXITCODE -eq 0) { $exitCode = 0; return }
            }
            
            # Try UninstallString with silent flags
            if ($_.UninstallString) {
                # Zoom-specific silent uninstall
                cmd /c "$($_.UninstallString) /uninstall /silent" 2>&1 | Out-Null
                if ($LASTEXITCODE -eq 0) { $exitCode = 0; return }
                
                # Generic silent
                cmd /c "$($_.UninstallString) /S" 2>&1 | Out-Null
                if ($LASTEXITCODE -eq 0) { $exitCode = 0; return }
            }
        }
    }
}

if (-not $found) {
    Write-Host "Zoom not found"
    $exitCode = 0  # Not found = success for Fleet
}

Exit $exitCode