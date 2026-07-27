$softwareName = "Google Earth Pro"

# Google Earth Pro's EXE installer wraps a WiX MSI, so the ARP UninstallString is
# an MsiExec.exe /X{ProductCode} command with no quiet switch. Running it verbatim
# from a SYSTEM-context script pops an invisible confirmation dialog in session 0,
# so the uninstall silently no-ops and the app is still present afterwards. Detect
# the msiexec form and re-run it with /quiet /norestart instead.
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
            ForEach-Object { Get-ItemProperty $_.PSPath }

    $foundUninstaller = $false
    foreach ($key in $uninstallKeys) {
        if ($key.DisplayName -ne $softwareName) { continue }

        $foundUninstaller = $true
        $uninstallString = if ($key.QuietUninstallString) {
            $key.QuietUninstallString
        } else {
            $key.UninstallString
        }

        Write-Host "Uninstall string: $uninstallString"

        # Prefer the MSI product code: for MSI-installed products the registry key
        # name is the product code, which sidesteps the UninstallString parsing
        # quirks handled in the fallback below.
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
            # Fall back to running the uninstaller executable directly, handling
            # quoted and unquoted paths (including paths containing spaces).
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
        break
    }

    if (-not $foundUninstaller) {
        Write-Host "Uninstaller for '$softwareName' not found."
        Exit 1
    }

} catch {
    Write-Host "Error: $_"
    Exit 1
}

# The MSI hands off to child msiexec processes; wait for them to drain so the
# registry reflects the removal before software inventory is re-queried.
$elapsed = 0
while ((Get-Process -Name "msiexec" -ErrorAction SilentlyContinue) -and ($elapsed -lt 120)) {
    Start-Sleep -Seconds 2
    $elapsed += 2
    Write-Host "Waiting for msiexec to complete... ($elapsed seconds)"
}

# 3010 (reboot required) and 1641 (reboot initiated) are successful uninstalls.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) { Exit 0 }

Exit $exitCode
