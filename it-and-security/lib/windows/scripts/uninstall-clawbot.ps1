# Uninstall Clawbot from Windows

$uninstallPaths = @(
    "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*",
    "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*"
)

foreach ($path in $uninstallPaths) {
    $apps = Get-ItemProperty $path -ErrorAction SilentlyContinue | Where-Object { $_.DisplayName -like '*clawbot*' }
    foreach ($app in $apps) {
        if ($app.UninstallString) {
            $uninstallCmd = $app.UninstallString
            Start-Process cmd.exe -ArgumentList "/c $uninstallCmd /quiet /norestart" -Wait -NoNewWindow
        }
    }
}

exit 0
