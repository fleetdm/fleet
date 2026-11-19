# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

# Grok Bot ships an electron-builder NSIS installer that defaults to a per-user
# install; /allusers installs it to Program Files for every user.

$exeFilePath = "${env:INSTALLER_PATH}"
$displayNameLike = "Grok Bot *"
$publisher = "SpaceXAI"

function Test-GrokBotRegistered {
    foreach ($path in @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
    )) {
        $match = Get-ItemProperty $path -ErrorAction SilentlyContinue | Where-Object {
            $_.DisplayName -like $displayNameLike -and $_.Publisher -eq $publisher
        }
        if ($match) { return $true }
    }
    return $false
}

try {
    $process = Start-Process -FilePath $exeFilePath -ArgumentList "/S", "/allusers" -PassThru -Wait
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"
    if ($exitCode -ne 0) { Exit $exitCode }

    # osquery reads the uninstall registry entry, so don't report success before it exists.
    $deadline = (Get-Date).AddSeconds(120)
    while (-not (Test-GrokBotRegistered)) {
        if ((Get-Date) -gt $deadline) {
            Write-Host "Error: Grok Bot was not registered within 120 seconds of the installer exiting."
            Exit 1
        }
        Start-Sleep -Seconds 2
    }

    Write-Host "Grok Bot installed."
    Exit 0
} catch {
    Write-Host "Error: $_"
    Exit 1
}
