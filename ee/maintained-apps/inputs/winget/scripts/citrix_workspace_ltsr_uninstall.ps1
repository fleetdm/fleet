# Uninstall for Citrix Workspace App LTSR.
#
# The bootstrap registers multiple separate Programs and Features entries
# (confirmed by CI: "Citrix Workspace Inside" plus a distinct "Citrix
# Workspace(USB)" entry, and possibly others depending on selected
# components) -- removing just the first one found still leaves the app
# detectable. Enumerate every entry matching the "Citrix Workspace" prefix
# and publisher, and uninstall each one. No space before the wildcard: some
# entries (e.g. "Citrix Workspace(USB)") have no space after "Workspace".
#
# Most of these entries' UninstallStrings are standard MSI references
# (MsiExec.exe /I{ProductCode}, the maintenance/repair form) -- not the
# TrolleyExpress.exe or CWAInstaller.exe bootstrapper uninstaller some older
# community docs describe for this DisplayName pattern. Resolve the
# ProductCode (registry key name, or the first GUID in the UninstallString)
# and run a clean "msiexec /x {ProductCode} /qn /norestart" -- never reuse
# the /I switch from the registry string. If an entry isn't MSI-based, fall
# back to re-running its own uninstaller exe with Citrix's documented silent
# switches.
#
# The Citrix bootstrapper installs several components as separate MSI
# transactions, so uninstalling one can transiently fail with 1618
# (ERROR_INSTALL_ALREADY_RUNNING) while another is still mid-transaction.
# Retry a few times with a short delay.
#
# One entry (DisplayName "Citrix Workspace <version>", UninstallString
# bootstrapperhelper.exe rather than msiexec) is the master entry: removing
# it cascades and removes the other MSI sub-components (USB, SSON, Inside,
# DV, ...) too. Since every entry was already snapshotted before any
# uninstall ran, a later msiexec /x for an already-cascaded component fails
# with 1605 (ERROR_UNKNOWN_PRODUCT, "not currently installed") -- that's
# success, not a failure.

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
        # MSI-based entry: resolve the ProductCode and run a clean uninstall,
        # ignoring whatever switches are already in the registry string.
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

            if ($exitCode -ne 1618) {
                break
            }

            Write-Host "Windows Installer busy (1618) -- another of Citrix's own component installs likely still holds the mutex. Retrying in ${msiRetryDelaySeconds}s."
            Start-Sleep -Seconds $msiRetryDelaySeconds
        }
    } else {
        # Non-MSI entry (e.g. TrolleyExpress.exe / CWAInstaller.exe): re-run
        # the vendor's own uninstaller with its documented silent switches.
        $exePath = ""
        if ($uninstallCommand -match '^\s*"([^"]+)"') {
            # Quoted path
            $exePath = $matches[1]
        } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)') {
            # Unquoted path that may contain spaces (e.g. "C:\Program Files (x86)\...")
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

    # Treat msiexec-style reboot-required codes as success, and 1605 (product
    # already removed, e.g. by another entry's cascade) as success too.
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
    # Already uninstalled (or never installed) -- nothing to do.
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
