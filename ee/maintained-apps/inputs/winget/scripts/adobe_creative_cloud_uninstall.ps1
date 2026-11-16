# Uninstalls Adobe Creative Cloud with Adobe's Creative Cloud Uninstaller ("-u" runs it
# unattended), then confirms the app is no longer registered.

$displayName = "Adobe Creative Cloud"
$publisher = "Adobe Inc."
$maxAttempts = 2
$removalTimeoutSeconds = 60

$registryPaths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

$knownUninstallerPaths = @(
    "${env:ProgramFiles}\Adobe\Adobe Creative Cloud\ACC\Creative Cloud Uninstaller.exe",
    "${env:ProgramFiles(x86)}\Adobe\Adobe Creative Cloud\ACC\Creative Cloud Uninstaller.exe",
    "${env:ProgramFiles}\Adobe\Adobe Creative Cloud\Utils\Creative Cloud Uninstaller.exe",
    "${env:ProgramFiles(x86)}\Adobe\Adobe Creative Cloud\Utils\Creative Cloud Uninstaller.exe"
)

# Any Creative Cloud process still running (UI helpers, crash processors, or the setup that
# keeps going for a few minutes after install) locks files the uninstaller must delete. The
# uninstaller then restores everything and still exits 0, so stop them all first.
$processDirs = @(
    "${env:ProgramFiles}\Adobe\Adobe Creative Cloud",
    "${env:ProgramFiles(x86)}\Adobe\Adobe Creative Cloud",
    "${env:ProgramFiles}\Adobe\Adobe Creative Cloud Experience",
    "${env:ProgramFiles(x86)}\Adobe\Adobe Creative Cloud Experience",
    "${env:ProgramFiles}\Common Files\Adobe\Adobe Desktop Common",
    "${env:ProgramFiles(x86)}\Common Files\Adobe\Adobe Desktop Common"
) | Where-Object { $_ -and -not $_.StartsWith('\') }

$processNames = @(
    "Creative Cloud.exe",
    "Creative Cloud Helper.exe",
    "Creative Cloud UI Helper.exe",
    "CCXProcess.exe",
    "CCLibrary.exe",
    "CoreSync.exe",
    "Adobe Desktop Service.exe",
    "AdobeIPCBroker.exe",
    "AdobeNotificationClient.exe"
)

function Get-CreativeCloudEntry {
    Get-ChildItem -Path $registryPaths -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
        Where-Object { $_.DisplayName -eq $displayName -and $_.Publisher -eq $publisher } |
        Select-Object -First 1
}

function Stop-CreativeCloudProcesses {
    $procs = @(Get-CimInstance Win32_Process -ErrorAction SilentlyContinue | Where-Object {
        if ($processNames -contains $_.Name) { return $true }
        $path = $_.ExecutablePath
        if (-not $path) { return $false }
        foreach ($dir in $processDirs) {
            if ($path.StartsWith("$dir\", [System.StringComparison]::OrdinalIgnoreCase)) { return $true }
        }
        return $false
    })
    foreach ($p in $procs) {
        Stop-Process -Id $p.ProcessId -Force -ErrorAction SilentlyContinue
    }
    if ($procs.Count -gt 0) {
        Write-Host "Stopped $($procs.Count) Creative Cloud process(es)."
        Wait-Process -Id ($procs | ForEach-Object { $_.ProcessId }) -Timeout 30 -ErrorAction SilentlyContinue
    }
}

try {
    $entry = Get-CreativeCloudEntry
    $uninstallerPath = $null

    if ($entry -and $entry.UninstallString) {
        if ($entry.UninstallString -match '^\s*"([^"]+)"') {
            $uninstallerPath = $Matches[1]
        } elseif ($entry.UninstallString -match '(?i)^\s*(.+?\.exe)') {
            $uninstallerPath = $Matches[1]
        }
        if ($uninstallerPath -and -not (Test-Path -LiteralPath $uninstallerPath)) {
            $uninstallerPath = $null
        }
    }

    if (-not $uninstallerPath) {
        $uninstallerPath = $knownUninstallerPaths | Where-Object { Test-Path -LiteralPath $_ } | Select-Object -First 1
    }

    if (-not $uninstallerPath) {
        if ($entry) {
            Write-Host "Adobe Creative Cloud is registered but its uninstaller was not found."
            Exit 1
        }
        Write-Host "Adobe Creative Cloud is not installed; nothing to do."
        Exit 0
    }

    for ($attempt = 1; $attempt -le $maxAttempts; $attempt++) {
        Stop-CreativeCloudProcesses

        Write-Host "Uninstall attempt ${attempt}: `"$uninstallerPath`" -u"
        $process = Start-Process -FilePath $uninstallerPath -ArgumentList "-u" -NoNewWindow -PassThru -Wait
        Write-Host "Uninstall exit code: $($process.ExitCode)"

        $elapsed = 0
        while ((Get-CreativeCloudEntry) -and ($elapsed -lt $removalTimeoutSeconds)) {
            Start-Sleep -Seconds 5
            $elapsed += 5
        }

        if (-not (Get-CreativeCloudEntry)) {
            Write-Host "Adobe Creative Cloud was removed."
            Exit 0
        }

        Write-Host "Adobe Creative Cloud is still registered after attempt ${attempt}."
        if (-not (Test-Path -LiteralPath $uninstallerPath)) { break }
    }

    Write-Host "Adobe Creative Cloud uninstall failed: the app is still installed."
    Exit 1

} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
