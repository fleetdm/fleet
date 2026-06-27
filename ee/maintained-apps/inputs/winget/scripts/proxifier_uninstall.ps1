# Fleet runs this uninstall script as SYSTEM (machine scope).
# Match against the registry DisplayName for Proxifier.
$softwareName = "Proxifier"

# It is recommended to use exact software name here if possible to avoid
# uninstalling unintended software.
$softwareNameLike = "*$softwareName*"

# Inno Setup uninstaller silent flags (used if UninstallString carries none).
$defaultArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"

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
        #   1. "C:\Path With Spaces\unins000.exe" /VERYSILENT   (quoted)
        #   2. C:\Path With Spaces\unins000.exe /VERYSILENT     (unquoted, may contain spaces)
        #   3. MsiExec.exe /X{GUID}                             (bare token)
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

        # If we fell back to the raw UninstallString (no quiet variant), make sure
        # the silent flags are present.
        if (-not $useQuiet -and ($args -notmatch '(?i)/VERYSILENT|/SILENT')) {
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
