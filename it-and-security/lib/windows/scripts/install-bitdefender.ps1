$ResolveWingetPath = Resolve-Path "C:\Program Files\WindowsApps\Microsoft.DesktopAppInstaller_*_x64__8wekyb3d8bbwe"
    if ($ResolveWingetPath){
           $WingetPath = $ResolveWingetPath[-1].Path
    }

$config
Set-Location $wingetpath
.\winget.exe install --id=Bitdefender.Bitdefender -e -h --accept-package-agreements --accept-source-agreements --disable-interactivity
