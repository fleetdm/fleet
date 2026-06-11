# Fleet runs this uninstall script as SYSTEM (machine scope).
# Match against the registry DisplayName for OneDrive.
$softwareName = "Microsoft OneDrive"

# It is recommended to use exact software name here if possible to avoid
# uninstalling unintended software.
$softwareNameLike = "$softwareName"

# OneDriveSetup.exe uninstalls with /uninstall (and /allusers for machine-wide
# installs). Used only if the registered UninstallString carries no flags.
$defaultArgs = "/uninstall /allusers"

# A machine-wide OneDrive registers under HKLM; also check WOW6432Node since the
# installer is 32-bit.
$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

$foundUninstaller = $false
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -like $softwareNameLike) {
        $foundUninstaller = $true

        # Prefer QuietUninstallString when present; it already includes silent flags.
        $useQuiet = [bool]$key.QuietUninstallString
        $uninstallCommand = if ($useQuiet) {
            $key.QuietUninstallString
        } else {
            $key.UninstallString
        }

        if ([string]::IsNullOrWhiteSpace($uninstallCommand)) {
            Throw "No UninstallString found for '$($key.DisplayName)'."
        }

        # Parse the UninstallString defensively. It comes in three shapes:
        #   1. "C:\Program Files\Microsoft OneDrive\...\OneDriveSetup.exe" /uninstall  (quoted)
        #   2. C:\Program Files\Microsoft OneDrive\...\OneDriveSetup.exe /uninstall    (unquoted, may contain spaces)
        #   3. MsiExec.exe /X{GUID}                                                    (bare token)
        if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
            $exe = $matches[1]
            $args = $matches[2].Trim()
        } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            $exe = $matches[1]
            $args = $matches[2].Trim()
        } else {
            $uninstallCommand -match '^\s*(\S+)\s*(.*)$' | Out-Null
            $exe = $matches[1]
            $args = $matches[2].Trim()
        }

        # If we fell back to the raw UninstallString (no quiet variant), ensure the
        # uninstall runs unattended and all-users.
        if (-not $useQuiet -and ($args -notmatch '(?i)/uninstall')) {
            $args = "$args $defaultArgs".Trim()
        }

        Write-Host "Uninstall command: $exe"
        Write-Host "Uninstall args: $args"

        $processOptions = @{
            FilePath = $exe
            PassThru = $true
            Wait = $true
        }
        if ($args -ne '') {
            $processOptions.ArgumentList = $args
        }

        $process = Start-Process @processOptions
        $exitCode = $process.ExitCode
        Write-Host "Uninstall exit code: $exitCode"
        break
    }
}

if (-not $foundUninstaller) {
    Write-Host "Uninstaller for '$softwareName' not found."
    $exitCode = 1
}

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode
