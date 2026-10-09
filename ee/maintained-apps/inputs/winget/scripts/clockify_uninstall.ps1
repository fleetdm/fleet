# Closes Clockify if it is running, uninstalls it by finding all related product
# codes for its upgrade code, then removes the all-users Start Menu shortcut the
# install script created and any Clockify shortcuts in user profiles that point
# at the removed app.
#   0    = success
#   3010 = success, reboot required
#   1641 = success, reboot initiated

$inst = New-Object -ComObject "WindowsInstaller.Installer"
$timeoutSeconds = 300  # 5 minute timeout per product
$installDir = Join-Path $env:ProgramFiles "Clockify"
$exePath = Join-Path $installDir "ClockifyWindows.exe"

Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -and $_.Path.StartsWith("$installDir\", [System.StringComparison]::OrdinalIgnoreCase) } |
    Stop-Process -Force -ErrorAction SilentlyContinue

foreach ($product_code in $inst.RelatedProducts('{A17CEE88-470D-49D0-A58D-40D503A0BEBF}')) {
    $process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $product_code, "/norestart") -PassThru
    # Reading Handle now keeps ExitCode available after WaitForExit(timeout).
    $null = $process.Handle

    if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
        Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
        Exit 1603  # ERROR_UNINSTALL_FAILURE
    }

    Write-Host "Uninstall for $product_code exited $($process.ExitCode)"
    if (@(0, 3010, 1641) -notcontains $process.ExitCode) {
        Exit $process.ExitCode
    }
}

$shell = New-Object -ComObject WScript.Shell
$shortcuts = @(Join-Path $env:ProgramData "Microsoft\Windows\Start Menu\Programs\Clockify.lnk")
$profileList = 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList'
$profileDirs = @((Get-ItemProperty $profileList).Default) + @(
    Get-ChildItem $profileList -ErrorAction SilentlyContinue | ForEach-Object { (Get-ItemProperty $_.PSPath).ProfileImagePath })
$profileDirs = @($profileDirs | Where-Object { $_ } | ForEach-Object { [Environment]::ExpandEnvironmentVariables($_) } | Sort-Object -Unique)
foreach ($dir in $profileDirs) {
    $shortcuts += Join-Path $dir "AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Clockify.lnk"
    $shortcuts += Join-Path $dir "Desktop\Clockify.lnk"
}
foreach ($lnk in $shortcuts) {
    if ((Test-Path -LiteralPath $lnk) -and $shell.CreateShortcut($lnk).TargetPath -eq $exePath) {
        Remove-Item -LiteralPath $lnk -Force -ErrorAction SilentlyContinue
        Write-Host "Removed $lnk"
    }
}

if ((Test-Path -LiteralPath $installDir) -and -not (Get-ChildItem -LiteralPath $installDir -Force)) {
    Remove-Item -LiteralPath $installDir -Force -ErrorAction SilentlyContinue
}

Exit 0
