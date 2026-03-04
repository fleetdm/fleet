# Download alex-mitchell.png from Google Drive and set as desktop background
$FileId = "1wFcjAGgP8LJXqeT6j76dTcAb8OMhFlIF"
$DownloadUrl = "https://drive.google.com/uc?export=download&id=$FileId"
$DestDir = "C:\Fleet"
$DestPath = "$DestDir\alex-mitchell.png"

# Create directory if it doesn't exist
if (-not (Test-Path $DestDir)) {
    New-Item -ItemType Directory -Path $DestDir -Force | Out-Null
}

# Download the image
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $DestPath -UseBasicParsing
} catch {
    Write-Error "Failed to download alex-mitchell.png: $_"
    exit 1
}

if (-not (Test-Path $DestPath)) {
    Write-Error "Failed to download alex-mitchell.png"
    exit 1
}

# Set desktop background via registry
Set-ItemProperty -Path 'HKCU:\Control Panel\Desktop' -Name 'Wallpaper' -Value $DestPath
Set-ItemProperty -Path 'HKCU:\Control Panel\Desktop' -Name 'WallpaperStyle' -Value '10'
Set-ItemProperty -Path 'HKCU:\Control Panel\Desktop' -Name 'TileWallpaper' -Value '0'

# Refresh the desktop
Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
public class Wallpaper {
    [DllImport("user32.dll", CharSet = CharSet.Auto)]
    public static extern int SystemParametersInfo(int uAction, int uParam, string lpvParam, int fuWinIni);
}
"@
[Wallpaper]::SystemParametersInfo(0x0014, 0, $DestPath, 0x0001 -bor 0x0002)

Write-Output "Desktop background set to alex-mitchell.png"
exit 0
