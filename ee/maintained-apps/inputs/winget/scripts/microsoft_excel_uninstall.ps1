# Microsoft Excel (Microsoft 365) - Uninstall Script
# Removes Excel while preserving other Office apps

$ErrorActionPreference = "Stop"

$odtPath = "$env:TEMP\odt_excel_uninstall_$(Get-Random)"
New-Item -ItemType Directory -Path $odtPath -Force | Out-Null

try {
    # Download ODT (needed for reconfiguration)
    $odtUrl = "https://officecdn.microsoft.com/pr/wsus/setup.exe"
    $setupPath = "$odtPath\setup.exe"
    
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $odtUrl -OutFile $setupPath -UseBasicParsing
    
    # Detect existing Office apps
    $clickToRunPath = "HKLM:\SOFTWARE\Microsoft\Office\ClickToRun\Configuration"
    $existingApps = @()
    
    if (Test-Path $clickToRunPath) {
        $excludedApps = (Get-ItemProperty -Path $clickToRunPath -Name "O365BusinessRetail.ExcludedApps" -ErrorAction SilentlyContinue)."O365BusinessRetail.ExcludedApps"
        
        $allApps = @("Access", "Excel", "Groove", "Lync", "OneDrive", "OneNote", "Outlook", "PowerPoint", "Publisher", "Teams", "Word")
        
        if ($excludedApps) {
            $excludedList = $excludedApps -split ","
            $existingApps = $allApps | Where-Object { $_ -notin $excludedList }
        }
    }
    
    # Remove Excel from the list of installed apps
    $appsToKeep = $existingApps | Where-Object { $_ -ne "Excel" }
    
    if ($appsToKeep.Count -eq 0) {
        # No other Office apps installed - remove Office entirely
        $configXml = @"
<Configuration>
  <Remove All="TRUE" />
  <Property Name="FORCEAPPSHUTDOWN" Value="TRUE" />
  <Display Level="None" AcceptEULA="TRUE" />
</Configuration>
"@
    }
    else {
        # Other apps exist - reconfigure to exclude Excel
        $appsToExclude = @("Access", "Excel", "Groove", "Lync", "OneDrive", "OneNote", "Outlook", "PowerPoint", "Publisher", "Teams", "Word") | Where-Object { $_ -notin $appsToKeep }
        $excludeElements = ($appsToExclude | ForEach-Object { "      <ExcludeApp ID=`"$_`" />" }) -join "`n"
        
        $configXml = @"
<Configuration>
  <Add OfficeClientEdition="64" Channel="Current">
    <Product ID="O365BusinessRetail">
      <Language ID="MatchOS" />
$excludeElements
    </Product>
  </Add>
  <Property Name="FORCEAPPSHUTDOWN" Value="TRUE" />
  <Display Level="None" AcceptEULA="TRUE" />
</Configuration>
"@
    }
    
    # Write config
    $configPath = "$odtPath\config.xml"
    $configXml | Out-File -FilePath $configPath -Encoding UTF8
    
    # Run ODT
    $process = Start-Process -FilePath $setupPath -ArgumentList "/configure `"$configPath`"" -Wait -PassThru -NoNewWindow
    
    $exitCode = $process.ExitCode
}
catch {
    Write-Error $_.Exception.Message
    $exitCode = 1
}
finally {
    Remove-Item -Path $odtPath -Recurse -Force -ErrorAction SilentlyContinue
}

exit $exitCode
