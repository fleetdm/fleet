# Uninstalls Adobe Creative Cloud by invoking the Creative Cloud Uninstaller.
# Adobe provides a dedicated uninstaller binary that supports unattended
# uninstall via the "-u" switch. The binary is located via the registry
# UninstallString when available, with a fallback to the default install path.

$displayName = "Adobe Creative Cloud"
$publisher = "Adobe Inc."

$registryPaths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

$knownUninstallerPaths = @(
    "${env:ProgramFiles(x86)}\Adobe\Adobe Creative Cloud\Utils\Creative Cloud Uninstaller.exe",
    "${env:ProgramFiles}\Adobe\Adobe Creative Cloud\Utils\Creative Cloud Uninstaller.exe"
)

try {
    # Best-effort: stop running Creative Cloud processes so the uninstaller
    # does not fail on locked files.
    $processesToStop = @(
        "Creative Cloud",
        "Creative Cloud Helper",
        "CCXProcess",
        "CCLibrary",
        "Adobe Desktop Service",
        "AdobeIPCBroker",
        "AdobeNotificationClient"
    )
    foreach ($procName in $processesToStop) {
        Stop-Process -Name $procName -Force -ErrorAction SilentlyContinue
    }

    $uninstallerPath = $null

    [array]$uninstallKeys = Get-ChildItem -Path $registryPaths -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

    foreach ($key in $uninstallKeys) {
        if (-not $key.DisplayName) { continue }
        $nameMatches = $key.DisplayName -eq $displayName
        $publisherMatches = ($publisher -eq "" -or $key.Publisher -eq $publisher)
        if ($nameMatches -and $publisherMatches -and $key.UninstallString) {
            $uninstallString = $key.UninstallString
            if ($uninstallString -match '^"([^"]+)"') {
                $uninstallerPath = $matches[1]
            } elseif ($uninstallString -match '^([^\s]+)') {
                $uninstallerPath = $matches[1]
            }
            if ($uninstallerPath -and (Test-Path -LiteralPath $uninstallerPath)) {
                break
            } else {
                $uninstallerPath = $null
            }
        }
    }

    if (-not $uninstallerPath) {
        foreach ($p in $knownUninstallerPaths) {
            if (Test-Path -LiteralPath $p) {
                $uninstallerPath = $p
                break
            }
        }
    }

    if (-not $uninstallerPath) {
        Write-Host "Adobe Creative Cloud uninstaller not found; nothing to do."
        Exit 0
    }

    Write-Host "Uninstall executable: $uninstallerPath"
    Write-Host "Uninstall arguments: -u"

    $processOptions = @{
        FilePath = $uninstallerPath
        ArgumentList = "-u"
        NoNewWindow = $true
        PassThru = $true
        Wait = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    Write-Host "Uninstall exit code: $exitCode"
    Exit $exitCode

} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
