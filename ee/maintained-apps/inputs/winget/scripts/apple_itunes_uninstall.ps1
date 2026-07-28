# iTunes registers as "iTunes" in the Windows registry.
$softwareName = "iTunes"

# iTunes is a WiX bundle chaining four MSIs, so removing it is slow: the previous
# script started the uninstall, then gave up after a fixed 120s msiexec wait and
# returned while the removal was still in flight, leaving iTunes registered. Wait
# on the uninstaller itself, then poll the Add/Remove Programs entry until it
# actually clears.
$uninstallTimeoutSeconds = 480
$clearTimeoutSeconds = 300

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$exitCode = 0

function Get-ITunesUninstallKey {
    Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -eq $softwareName } |
        Select-Object -First 1
}

# Stop iTunes and its companions first: they hold file locks, and because
# "Start-Process -Wait" waits for descendants as well as the process itself, a
# resident helper would block this script.
foreach ($name in @("iTunes", "iTunesHelper", "AppleMobileDeviceService", "mDNSResponder")) {
    Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
}

try {
    $key = Get-ITunesUninstallKey
    if (-not $key) {
        Write-Host "Uninstaller for '$softwareName' not found."
        Exit 1
    }

    $uninstallString = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
    Write-Host "Uninstall string: $uninstallString"

    # The visible "iTunes" entry is an MSI, so prefer the product code and drive
    # msiexec directly with /quiet -- the registry string omits a quiet switch,
    # which would raise an invisible dialog in session 0 and do nothing. If Apple
    # ever registers the burn bundle instead, fall back to running that with
    # /uninstall /quiet /norestart.
    $productCode = $null
    if ($uninstallString -match '(?i)msiexec(\.exe)?.*?[/-][xi]\s*(\{[0-9A-Fa-f\-]+\})') {
        $productCode = $Matches[2]
    } elseif ($key.PSChildName -match '^\{[0-9A-Fa-f\-]+\}$') {
        $productCode = $key.PSChildName
    }

    if ($productCode) {
        Write-Host "Uninstalling MSI product $productCode"
        $process = Start-Process -FilePath "msiexec.exe" `
            -ArgumentList "/x", $productCode, "/quiet", "/norestart" `
            -PassThru -NoNewWindow
    } else {
        $uninstallCommand = $uninstallString
        $uninstallArgs = ""
        if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
            $uninstallCommand = $Matches[1]; $uninstallArgs = $Matches[2]
        } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            $uninstallCommand = $Matches[1]; $uninstallArgs = $Matches[2]
        } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
            $uninstallCommand = $Matches[1]; $uninstallArgs = $Matches[2]
        }
        if ($uninstallArgs -notmatch '(?i)/uninstall') { $uninstallArgs = "$uninstallArgs /uninstall".Trim() }
        if ($uninstallArgs -notmatch '(?i)/quiet')     { $uninstallArgs = "$uninstallArgs /quiet".Trim() }
        if ($uninstallArgs -notmatch '(?i)/norestart') { $uninstallArgs = "$uninstallArgs /norestart".Trim() }

        Write-Host "Uninstall command: $uninstallCommand"
        Write-Host "Uninstall args: $uninstallArgs"
        $process = Start-Process -FilePath $uninstallCommand -ArgumentList $uninstallArgs -PassThru
    }

    # Touch .Handle so the exit code is still readable after the process ends:
    # Start-Process -PassThru otherwise returns $null for .ExitCode.
    $null = $process.Handle

    if (-not $process.WaitForExit($uninstallTimeoutSeconds * 1000)) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        Write-Host "Uninstall timed out after $uninstallTimeoutSeconds seconds"
        Exit 1603
    }

    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"
} catch {
    Write-Host "Error: $_"
    Exit 1
}

# Removing a chained bundle keeps child msiexec processes busy well past the
# parent's exit. Wait for the registry to actually reflect the removal rather
# than for a fixed interval.
$elapsed = 0
while ((Get-ITunesUninstallKey) -and ($elapsed -lt $clearTimeoutSeconds)) {
    Start-Sleep -Seconds 10
    $elapsed += 10
    Write-Host "Waiting for the uninstall to finish... ($elapsed seconds)"
}

if (Get-ITunesUninstallKey) {
    Write-Host "'$softwareName' is still registered after the uninstall."
    Exit 1
}

# 3010 (reboot required) and 1641 (reboot initiated) are successful uninstalls.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode
