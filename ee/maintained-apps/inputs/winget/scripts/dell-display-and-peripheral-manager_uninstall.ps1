# Uninstalls Dell Display and Peripheral Manager. DDPM is an InstallShield
# InstallScript (non-MSI) product: its uninstall registry key is named like an
# MSI ProductCode GUID, but no MSI product is registered, so msiexec /x fails
# with 1605 (verified on the windows-11-arm validator). Instead, run the
# registered UninstallString — Dell's own setup wrapper at
# "<install dir>\Installer\setup.exe" — with Dell's silent switch.

$softwareName = "Dell Display and Peripheral Manager"

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = $null

try {

# DDPM can auto-launch after install; a running instance can block the
# uninstaller.
Get-Process -Name "DDPM*" -ErrorAction SilentlyContinue |
    Stop-Process -Force -ErrorAction SilentlyContinue

[array]$uninstallKeys = Get-ChildItem `
    -Path @($machineKey, $machineKey32on64) `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -eq $softwareName) {
        if ($key.QuietUninstallString) {
            # Already includes silent arguments; run verbatim.
            Write-Host "Running QuietUninstallString: $($key.QuietUninstallString)"
            $process = Start-Process -FilePath "cmd.exe" `
                -ArgumentList "/c", $key.QuietUninstallString `
                -NoNewWindow -PassThru -Wait
            $exitCode = $process.ExitCode
            break
        }

        $u = $key.UninstallString
        if (-not $u) {
            Write-Host "No UninstallString registered for '$softwareName'."
            continue
        }

        # Parse defensively: quoted path, unquoted path with spaces, bare token.
        if ($u -match '^\s*"([^"]+)"\s*(.*)$') {
            $exe = $Matches[1]; $uninstallArgs = $Matches[2]
        } elseif ($u -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            $exe = $Matches[1]; $uninstallArgs = $Matches[2]
        } else {
            Write-Host "Unrecognized UninstallString format: $u"
            continue
        }

        if ($exe -match '(?i)msiexec') {
            # Defensive: if a future release registers an MSI-style uninstall.
            $uninstallArgs = "$uninstallArgs /qn /norestart".Trim()
        } else {
            # Dell's setup wrapper silent switch (same family as the /Silent
            # install switch; ManageEngine documents /S for removal).
            $uninstallArgs = "$uninstallArgs /Silent".Trim()
        }

        Write-Host "Uninstalling via: `"$exe`" $uninstallArgs"
        $process = Start-Process -FilePath $exe `
            -ArgumentList $uninstallArgs `
            -NoNewWindow -PassThru -Wait
        $exitCode = $process.ExitCode
        break
    }
}

} catch {
    Write-Host "Error: $_"
    Exit 1
}

if ($null -eq $exitCode) {
    Write-Host "Uninstall entry not found for '$softwareName'."
    Exit 1
}

Write-Host "Uninstall exit code: $exitCode"
# 0 = success, 3010 = success but reboot required, 1641 = reboot initiated
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }
Exit $exitCode
