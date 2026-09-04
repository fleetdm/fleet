# Uninstalls KeePass (Inno Setup installer). The registry DisplayName carries
# the version ("KeePass Password Safe <version>"), so match on a prefix. A
# running instance is asked to close first and force-stopped only if it refuses
# -- which discards unsaved database changes.

$softwareNameLike = "KeePass Password Safe*"
$silentArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"

$machineKey       = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

$key = $uninstallKeys | Where-Object { $_.DisplayName -like $softwareNameLike } | Select-Object -First 1

if (-not $key) {
    Write-Host "Uninstaller for KeePass not found."
    Exit 1
}

$running = Get-Process -Name "KeePass" -ErrorAction SilentlyContinue
if ($running) {
    Write-Host "Closing running KeePass before uninstalling."
    $running | ForEach-Object { $_.CloseMainWindow() | Out-Null }

    $deadline = (Get-Date).AddSeconds(30)
    while ((Get-Process -Name "KeePass" -ErrorAction SilentlyContinue) -and (Get-Date) -lt $deadline) {
        Start-Sleep -Seconds 2
    }

    Get-Process -Name "KeePass" -ErrorAction SilentlyContinue | ForEach-Object {
        Write-Host "KeePass (PID $($_.Id)) did not close; stopping it. Unsaved database changes are lost."
        Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
    }
}

# Prefer QuietUninstallString when present; otherwise use UninstallString.
$uninstallCommand = if ($key.QuietUninstallString) {
    $key.QuietUninstallString
} else {
    $key.UninstallString
}

# Defensive parser for the three UninstallString shapes:
#   "C:\path with spaces\unins000.exe" /ARG              -> quoted
#   C:\Program Files\KeePass Password Safe 2\unins000.exe -> unquoted with spaces
#   MsiExec.exe /X{GUID}                                 -> bare token
$uninstallPath = $null
$existingArgs  = ''
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $uninstallPath = $matches[1]
    $existingArgs  = $matches[2]
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $uninstallPath = $matches[1]
    $existingArgs  = $matches[2]
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    $uninstallPath = $matches[1]
    $existingArgs  = $matches[2]
}

$finalArgs = ($existingArgs.Trim() + ' ' + $silentArgs).Trim()

Write-Host "Uninstall command: $uninstallPath"
Write-Host "Uninstall args: $finalArgs"

$processOptions = @{
    FilePath     = $uninstallPath
    ArgumentList = $finalArgs
    PassThru     = $true
    Wait         = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Uninstall exit code: $exitCode"

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode
