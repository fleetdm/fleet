$softwareName = "Dell Display and Peripheral Manager"

# The install leaves DDPM's background app running, which holds file locks and --
# because "Start-Process -Wait" waits for descendants as well as the process
# itself -- would block this script. Stop those first, then wait only on the
# uninstaller and poll until the Add/Remove Programs entry clears.
$leftovers = @("DDPM", "DellDisplayManager", "DisplayManager")
$timeoutSeconds = 480

$machineKey = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
$exitCode = 0

function Get-DdpmUninstallKey {
    Get-ChildItem -Path @($machineKey, $machineKey32on64) -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -like "$softwareName*" } |
        Select-Object -First 1
}

foreach ($name in $leftovers) {
    Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
}

try {
    $key = Get-DdpmUninstallKey
    if (-not $key) {
        Write-Host "Uninstaller for '$softwareName' not found."
        Exit 1
    }

    $uninstallString = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
    Write-Host "Uninstall string: $uninstallString"

    # If this resolves to an MSI product, uninstall it with msiexec directly --
    # the registry string typically omits a quiet switch, which would raise an
    # invisible dialog in session 0 and silently do nothing.
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
        # InstallShield uninstaller: parse the executable path, handling quoted
        # paths, unquoted paths containing spaces, and bare tokens, then pass the
        # documented silent switch.
        $uninstallCommand = $uninstallString
        $uninstallArgs = "/Silent"
        if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
            $uninstallCommand = $Matches[1]
            if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
        } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            $uninstallCommand = $Matches[1]
            if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
        } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
            $uninstallCommand = $Matches[1]
            if ($Matches[2]) { $uninstallArgs = "$($Matches[2]) $uninstallArgs".Trim() }
        }

        Write-Host "Uninstall command: $uninstallCommand"
        Write-Host "Uninstall args: $uninstallArgs"
        $process = Start-Process -FilePath $uninstallCommand -ArgumentList $uninstallArgs -PassThru
    }

    # Touch .Handle so the exit code is still readable after the process ends:
    # Start-Process -PassThru otherwise returns $null for .ExitCode.
    $null = $process.Handle

    if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        Write-Host "Uninstall timed out after $timeoutSeconds seconds"
        Exit 1603
    }

    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"
} catch {
    Write-Host "Error: $_"
    Exit 1
}

foreach ($name in $leftovers) {
    Stop-Process -Name $name -Force -ErrorAction SilentlyContinue
}

$elapsed = 0
while ((Get-DdpmUninstallKey) -and ($elapsed -lt 180)) {
    Start-Sleep -Seconds 5
    $elapsed += 5
    Write-Host "Waiting for the uninstall to finish... ($elapsed seconds)"
}

if (Get-DdpmUninstallKey) {
    Write-Host "'$softwareName' is still registered after the uninstall."
    Exit 1
}

# 3010 (reboot required) and 1641 (reboot initiated) are successful uninstalls.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode
