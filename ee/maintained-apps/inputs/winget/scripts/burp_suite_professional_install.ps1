$exeFilePath = "${env:INSTALLER_PATH}"
$ExpectedExitCodes = @(0, 3010, 1641)

try {
    # Without -dir, install4j defaults to a per-user install even when
    # elevated, so the Program Files override is load-bearing.
    $installDir = Join-Path $env:ProgramFiles "BurpSuitePro"
    $installedDirs = @(Get-ItemProperty -Path @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*'
    ) -ErrorAction SilentlyContinue | Where-Object {
        $_.DisplayName -like "Burp Suite Professional*" -and $_.Publisher -like "*PortSwigger*" -and $_.InstallLocation
    } | ForEach-Object { $_.InstallLocation.Trim('"').TrimEnd('\') })

    # Setup exits with code 1 while Burp Suite is running. Fleet runs as SYSTEM,
    # outside the user's session, so it can't ask the app to close: stop Burp,
    # its browser, and its bundled Java runtime first, wherever Burp is
    # installed. Unsaved changes are lost.
    # Only Burp's own files, in case it was installed into a shared folder.
    $burpPaths = foreach ($dir in @($installDir) + $installedDirs) {
        "$dir\BurpSuitePro.exe", "$dir\jre\", "$dir\burpbrowser\", "$dir\.install4j\"
    }
    $running = @(Get-Process -ErrorAction SilentlyContinue | Where-Object {
        $path = $_.Path
        $path -and ($burpPaths | Where-Object { $path.StartsWith($_, [StringComparison]::OrdinalIgnoreCase) })
    })
    foreach ($proc in $running) {
        Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
        Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    }
    $running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

    # Earlier versions of this script didn't quote -dir, so Burp was installed
    # into C:\Program. Installing into $installDir doesn't move that copy: install4j
    # adds a second one, and the old copy's version keeps the patch policy failing.
    # Remove other copies with their own uninstaller first. It also deletes the
    # Start Menu shortcut, which every copy shares.
    foreach ($dir in $installedDirs | Where-Object { $_ -ne $installDir }) {
        $uninstaller = Join-Path $dir "uninstall.exe"
        if (-not (Test-Path -LiteralPath $uninstaller)) { continue }
        Write-Host "Removing the copy of Burp Suite Professional in $dir."
        $removeProcess = Start-Process -FilePath $uninstaller `
            -ArgumentList "-q", "-Dinstall4j.suppressUnattendedReboot=true" -PassThru -Wait
        if ($ExpectedExitCodes -notcontains $removeProcess.ExitCode) {
            Write-Host "Removal exit code: $($removeProcess.ExitCode)"
            Exit $removeProcess.ExitCode
        }
    }

    $processOptions = @{
        FilePath     = "$exeFilePath"
        # Windows PowerShell doesn't quote array elements that contain spaces.
        ArgumentList = "-q", "-Dinstall4j.suppressUnattendedReboot=true", "-dir", "`"$installDir`""
        PassThru     = $true
        Wait         = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    Start-Sleep -Seconds 5

    if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
