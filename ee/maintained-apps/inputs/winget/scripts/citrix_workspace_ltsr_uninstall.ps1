# The bootstrap registers several separate Programs and Features entries, so
# enumerate everything matching the "Citrix Workspace" prefix and publisher
# and uninstall each one. No space before the wildcard: some entries (e.g.
# "Citrix Workspace(USB)") have none after "Workspace".

$softwareNameLike = "Citrix Workspace*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$timeoutSeconds = 180
$maxMsiAttempts = 5
$msiRetryDelaySeconds = 15

function Wait-ProcessBounded {
    param($Process, [int]$TimeoutSeconds)

    if (-not $Process.WaitForExit($TimeoutSeconds * 1000)) {
        Write-Host "Uninstaller did not exit within ${TimeoutSeconds}s, stopping it."
        Stop-Process -Id $Process.Id -Force -ErrorAction SilentlyContinue
        return $null
    }
    return $Process.ExitCode
}

function Uninstall-CitrixEntry {
    param($Entry)

    Write-Host "Uninstalling component: $($Entry.DisplayName)"

    $uninstallCommand = if ($Entry.QuietUninstallString) {
        $Entry.QuietUninstallString
    } else {
        $Entry.UninstallString
    }

    if (-not $uninstallCommand) {
        Write-Host "Entry has no UninstallString: $($Entry.DisplayName)"
        return 1
    }

    if ($uninstallCommand -match '(?i)msiexec') {
        # Resolve the ProductCode and run our own clean uninstall -- never
        # reuse the /I (repair) switch already in the registry string.
        $productCode = $Entry.PSChildName
        if ($productCode -notmatch '^\{[0-9A-Fa-f-]+\}$') {
            if ($uninstallCommand -match '(\{[0-9A-Fa-f-]+\})') {
                $productCode = $matches[1]
            }
        }
        if ($productCode -notmatch '^\{[0-9A-Fa-f-]+\}$') {
            Write-Host "Could not determine ProductCode from: $uninstallCommand"
            return 1
        }

        Write-Host "Uninstalling product code: $productCode"

        $exitCode = 1
        for ($attempt = 1; $attempt -le $maxMsiAttempts; $attempt++) {
            $process = Start-Process -FilePath "msiexec.exe" `
                -ArgumentList "/x $productCode /qn /norestart" `
                -PassThru

            $exitCode = Wait-ProcessBounded -Process $process -TimeoutSeconds $timeoutSeconds
            if ($null -eq $exitCode) {
                return 1
            }
            Write-Host "Uninstall exit code: $exitCode (attempt $attempt of $maxMsiAttempts)"

            # 1618: another component's transaction still holds the Windows
            # Installer mutex -- retry rather than fail.
            if ($exitCode -ne 1618) {
                break
            }
            Start-Sleep -Seconds $msiRetryDelaySeconds
        }
    } else {
        # Non-MSI entry: re-run its own uninstaller with Citrix's documented
        # silent switches.
        $exePath = ""
        if ($uninstallCommand -match '^\s*"([^"]+)"') {
            $exePath = $matches[1]
        } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)') {
            $exePath = $matches[1]
        } else {
            Write-Host "Could not parse uninstaller path from: $uninstallCommand"
            return 1
        }

        Write-Host "Uninstall command: $exePath"
        $process = Start-Process -FilePath $exePath -ArgumentList "/uninstall /cleanup /silent" `
            -PassThru

        $exitCode = Wait-ProcessBounded -Process $process -TimeoutSeconds $timeoutSeconds
        if ($null -eq $exitCode) {
            return 1
        }
        Write-Host "Uninstall exit code: $exitCode"
    }

    # 3010/1641: success pending reboot. 1605: already removed, e.g. by
    # another entry's cascade uninstall. All count as success.
    if ($exitCode -eq 3010 -or $exitCode -eq 1641 -or $exitCode -eq 1605) {
        return 0
    }
    return $exitCode
}

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue -ErrorVariable readErrors |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

if (@($readErrors).Count -ge $paths.Count) {
    Throw "Failed to read the registry uninstall keys: $($readErrors -join '; ')"
}

[array]$entries = $uninstallKeys | Where-Object {
    $_.DisplayName -and $_.DisplayName -like $softwareNameLike `
        -and $_.Publisher -eq "Citrix Systems, Inc."
}

if ($entries.Count -eq 0) {
    Write-Host "No entries found matching $softwareNameLike"
    Exit 0
}

$overallExitCode = 0
foreach ($entry in $entries) {
    $result = Uninstall-CitrixEntry -Entry $entry
    if ($result -ne 0) {
        $overallExitCode = $result
    }
}

Exit $overallExitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
