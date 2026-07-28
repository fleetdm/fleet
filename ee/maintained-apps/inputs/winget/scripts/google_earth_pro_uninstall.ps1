$softwareName = "Google Earth Pro"
$softwarePublisher = "Google"

# The ARP UninstallString is "MsiExec.exe /X{ProductCode}" with no quiet switch,
# which would stall on an invisible dialog, so re-run msiexec with /quiet.
$exeArgs = ""

$machineKey = `
 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
$machineKey32on64 = `
 'HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'

$exitCode = 0
$timeoutSeconds = 300

try {

    [array]$uninstallKeys = Get-ChildItem `
        -Path @($machineKey, $machineKey32on64) `
        -ErrorAction SilentlyContinue |
            ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

    $foundUninstaller = $false
    foreach ($key in $uninstallKeys) {
        if ($key.DisplayName -ne $softwareName -or $key.Publisher -ne $softwarePublisher) { continue }

        $foundUninstaller = $true
        $uninstallString = if ($key.QuietUninstallString) {
            $key.QuietUninstallString
        } else {
            $key.UninstallString
        }

        Write-Host "Uninstall string: $uninstallString"

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
            # Fall back to the uninstaller executable, handling quoted and
            # unquoted paths.
            $uninstallCommand = $uninstallString
            if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
                $uninstallCommand = $Matches[1]
                if ($Matches[2]) { $exeArgs = "$($Matches[2]) $exeArgs".Trim() }
            } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
                $uninstallCommand = $Matches[1]
                if ($Matches[2]) { $exeArgs = "$($Matches[2]) $exeArgs".Trim() }
            }

            Write-Host "Uninstall command: $uninstallCommand"
            Write-Host "Uninstall args: $exeArgs"

            $processOptions = @{
                FilePath = $uninstallCommand
                PassThru = $true
                NoNewWindow = $true
            }
            if ($exeArgs -ne '') { $processOptions.ArgumentList = $exeArgs }
            $process = Start-Process @processOptions
        }

        # Keeps .ExitCode readable after the process ends.
        $null = $process.Handle

        if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
            Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
            Write-Host "Uninstall timed out after $timeoutSeconds seconds"
            Exit 1603
        }

        $exitCode = $process.ExitCode
        Write-Host "Uninstall exit code: $exitCode"
        break
    }

    if (-not $foundUninstaller) {
        Write-Host "Uninstall entry not found for '$softwareName'."
        Exit 0
    }

} catch {
    Write-Host "Error: $_"
    Exit 1
}

# The MSI hands off to child msiexec processes; wait for them to drain.
$elapsed = 0
while ((Get-Process -Name "msiexec" -ErrorAction SilentlyContinue) -and ($elapsed -lt 120)) {
    Start-Sleep -Seconds 2
    $elapsed += 2
    Write-Host "Waiting for msiexec to complete... ($elapsed seconds)"
}

# 3010/1641 = reboot needed; 1605/1614 = already gone. All are success.
if ($exitCode -eq 3010 -or $exitCode -eq 1641 -or $exitCode -eq 1605 -or $exitCode -eq 1614) { Exit 0 }

Exit $exitCode
