# Uninstalls Dell Display and Peripheral Manager. DDPM is an InstallShield
# InstallScript (non-MSI) product: its uninstall registry key is named like an
# MSI ProductCode GUID, but no MSI product is registered, so msiexec /x fails
# with 1605. Its UninstallString ("<install dir>\Installer\setup.exe
# -runfromtemp -removeonly") hangs under Dell's /Silent wrapper switch because
# in -removeonly mode the InstallScript engine drives the dialogs and only
# honors its own /s silent switch with a response file. The installer ships
# exactly this removal response file embedded in its overlay (SdWelcomeMaint
# Result=303 / MessageBox Result=6 / SdFinishReboot Result=1); we recreate it
# with the GUID from the registry key and pass it via /f1. Success is gated on
# the uninstall registry entry disappearing, not the launcher exit code, since
# -runfromtemp relaunches from %TEMP% and can return early.

$softwareName = "Dell Display and Peripheral Manager"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

function Get-InstalledEntry {
    Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -eq $softwareName } |
        Select-Object -First 1
}

try {

# DDPM can auto-launch after install; a running instance can block the
# uninstaller.
Get-Process -Name "DDPM*" -ErrorAction SilentlyContinue |
    Stop-Process -Force -ErrorAction SilentlyContinue

$key = Get-InstalledEntry
if (-not $key) {
    Write-Host "Uninstall entry not found for '$softwareName'."
    Exit 1
}

$u = $key.UninstallString
if (-not $u) {
    Write-Host "No UninstallString registered for '$softwareName'."
    Exit 1
}

# Parse defensively: quoted path, unquoted path with spaces, bare token.
if ($u -match '^\s*"([^"]+)"\s*(.*)$') {
    $exe = $Matches[1]; $uninstallArgs = $Matches[2]
} elseif ($u -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exe = $Matches[1]; $uninstallArgs = $Matches[2]
} else {
    Write-Host "Unrecognized UninstallString format: $u"
    Exit 1
}

if ($exe -match '(?i)msiexec') {
    # Defensive: if a future release registers an MSI-style uninstall.
    $uninstallArgs = "$uninstallArgs /qn /norestart".Trim()
    $process = Start-Process -FilePath $exe -ArgumentList $uninstallArgs `
        -NoNewWindow -PassThru -Wait
    Write-Host "msiexec exit code: $($process.ExitCode)"
} else {
    # The registry key name is the InstallScript product GUID; the response
    # file's section names must use it.
    $guid = $key.PSChildName
    $issPath = Join-Path $env:TEMP "ddpm-uninstall.iss"
    $logPath = Join-Path $env:TEMP "ddpm-uninstall.log"
    Remove-Item -Path $logPath -Force -ErrorAction SilentlyContinue
    @"
[InstallShield Silent]
Version=v7.00
File=Response File
[File Transfer]
OverwrittenReadOnly=NoToAll
[$guid-DlgOrder]
Dlg0=$guid-SdWelcomeMaint-0
Count=3
Dlg1=$guid-MessageBox-0
Dlg2=$guid-SdFinishReboot-0
[$guid-SdWelcomeMaint-0]
Result=303
[$guid-MessageBox-0]
Result=6
[$guid-SdFinishReboot-0]
Result=1
BootOption=0
"@ | Set-Content -Path $issPath -Encoding ASCII

    $uninstallArgs = "$uninstallArgs /s /f1`"$issPath`" /f2`"$logPath`"".Trim()
    Write-Host "Uninstalling via: `"$exe`" $uninstallArgs"
    $process = Start-Process -FilePath $exe -ArgumentList $uninstallArgs -PassThru
    if (-not $process.WaitForExit(180000)) {
        Write-Host "Uninstaller still running after 180s; killing it."
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
    } else {
        Write-Host "Launcher exit code: $($process.ExitCode)"
    }
    if (Test-Path $logPath) {
        Write-Host "--- InstallShield uninstall log ---"
        Get-Content $logPath | Write-Host
        Write-Host "-----------------------------------"
    }
}

# The launcher can return before the %TEMP% copy finishes removal; poll the
# registry entry to decide success.
$deadline = (Get-Date).AddSeconds(120)
while ((Get-Date) -lt $deadline) {
    if (-not (Get-InstalledEntry)) {
        Write-Host "'$softwareName' uninstalled."
        Exit 0
    }
    Start-Sleep -Seconds 5
}

Write-Host "'$softwareName' is still registered after uninstall."
Exit 1

} catch {
    Write-Host "Error: $_"
    Exit 1
}
