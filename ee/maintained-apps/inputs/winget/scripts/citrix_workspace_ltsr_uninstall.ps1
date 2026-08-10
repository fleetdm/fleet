# Uninstall for Citrix Workspace App LTSR.
#
# The Programs and Features entry that's actually visible is Citrix's
# bundled "ReceiverInside" component (DisplayName "Citrix Workspace Inside"),
# whose UninstallString is a standard MSI reference (MsiExec.exe
# /I{ProductCode}, the maintenance/repair form) -- not the TrolleyExpress.exe
# or CWAInstaller.exe bootstrapper uninstaller some older community docs
# describe for this DisplayName pattern. Resolve the ProductCode (registry
# key name, or the first GUID in the UninstallString) and run a clean
# "msiexec /x {ProductCode} /qn /norestart" -- never reuse the /I switch
# from the registry string. If the selected entry isn't MSI-based, fall back
# to re-running its own uninstaller exe with Citrix's documented silent
# switches.

$softwareNameLike = "Citrix Workspace *"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

$selected = $null
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -and $key.DisplayName -like $softwareNameLike `
        -and $key.Publisher -eq "Citrix Systems, Inc.") {
        $selected = $key
        break
    }
}

if (-not $selected) {
    # Already uninstalled (or never installed) -- nothing to do.
    Write-Host "Uninstall entry not found for $softwareNameLike"
    Exit 0
}

Write-Host "Selected entry DisplayName: $($selected.DisplayName)"

$uninstallCommand = if ($selected.QuietUninstallString) {
    $selected.QuietUninstallString
} else {
    $selected.UninstallString
}

if (-not $uninstallCommand) {
    Write-Host "Selected entry has no UninstallString: $($selected.DisplayName)"
    Exit 1
}

if ($uninstallCommand -match '(?i)msiexec') {
    # MSI-based entry: resolve the ProductCode and run a clean uninstall,
    # ignoring whatever switches are already in the registry string.
    $productCode = $selected.PSChildName
    if ($productCode -notmatch '^\{[0-9A-Fa-f-]+\}$') {
        if ($uninstallCommand -match '(\{[0-9A-Fa-f-]+\})') {
            $productCode = $matches[1]
        }
    }
    if ($productCode -notmatch '^\{[0-9A-Fa-f-]+\}$') {
        Throw "Could not determine ProductCode from: $uninstallCommand"
    }

    Write-Host "Uninstalling product code: $productCode"
    $process = Start-Process -FilePath "msiexec.exe" `
        -ArgumentList "/x $productCode /qn /norestart" `
        -PassThru -Wait
} else {
    # Non-MSI entry (e.g. TrolleyExpress.exe / CWAInstaller.exe): re-run the
    # vendor's own uninstaller with its documented silent switches.
    $exePath = ""
    if ($uninstallCommand -match '^\s*"([^"]+)"') {
        # Quoted path
        $exePath = $matches[1]
    } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)') {
        # Unquoted path that may contain spaces (e.g. "C:\Program Files (x86)\...")
        $exePath = $matches[1]
    } else {
        Throw "Could not parse uninstaller path from: $uninstallCommand"
    }

    Write-Host "Uninstall command: $exePath"
    $process = Start-Process -FilePath $exePath -ArgumentList "/uninstall /cleanup /silent" `
        -PassThru -Wait
}

$exitCode = $process.ExitCode
Write-Host "Uninstall exit code: $exitCode"

# Treat msiexec-style reboot-required codes as success too.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
    Exit 0
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
