# Installs the Inter font (Google Fonts build, upright + italic variable fonts)
# for all users. Downloads are pinned to a google/fonts commit and verified by
# SHA-256. Users who are already signed in see the font after their next sign-in.

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$baseUrl = 'https://raw.githubusercontent.com/google/fonts/e1d6480102fed30739fead0faee463101f892c8f/ofl/inter'
$fontsDir = Join-Path $env:SystemRoot 'Fonts'
$regPath = 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts'
$fonts = @(
    @{
        Url    = "$baseUrl/Inter%5Bopsz,wght%5D.ttf"
        File   = 'Inter-VariableFont_opsz,wght.ttf'
        Name   = 'Inter Regular (TrueType)'
        Sha256 = '29160A80FF49DDCAB2C97711247E08B1FAB27A484A329CE8B813D820DC559031'
    },
    @{
        Url    = "$baseUrl/Inter-Italic%5Bopsz,wght%5D.ttf"
        File   = 'Inter-Italic-VariableFont_opsz,wght.ttf'
        Name   = 'Inter Italic (TrueType)'
        Sha256 = 'ACD98E64795781B2058F07B18475E0ECEE2A0FE2B42A49E2F9E37D0D6BF66CE6'
    }
)

$tmpDir = Join-Path ([IO.Path]::GetTempPath()) ([guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tmpDir | Out-Null
try {
    foreach ($font in $fonts) {
        $target = Join-Path $fontsDir $font.File
        $installed = (Test-Path -LiteralPath $target) -and
            ((Get-FileHash -LiteralPath $target -Algorithm SHA256).Hash -eq $font.Sha256)
        if ($installed) {
            Write-Output "$target is already installed"
        } else {
            $download = Join-Path $tmpDir $font.File
            Invoke-WebRequest -Uri $font.Url -OutFile $download -UseBasicParsing
            $hash = (Get-FileHash -LiteralPath $download -Algorithm SHA256).Hash
            if ($hash -ne $font.Sha256) {
                throw "Checksum mismatch for $($font.File): expected $($font.Sha256), got $hash"
            }
            Copy-Item -LiteralPath $download -Destination $target -Force
            Write-Output "Installed $target"
        }
        New-ItemProperty -Path $regPath -Name $font.Name -Value $font.File -PropertyType String -Force | Out-Null
    }
} catch {
    Write-Output "Error: $_"
    exit 1
} finally {
    Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
}
exit 0
