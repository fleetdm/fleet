# Microsoft Excel (Microsoft 365) - Install Script
# Installs Excel via ODT, preserving any existing Office apps

$ErrorActionPreference = "Stop"

# Create working directory
$odtPath = "$env:TEMP\odt_excel_$(Get-Random)"
New-Item -ItemType Directory -Path $odtPath -Force | Out-Null

try {
    # The installer downloaded by Fleet is the ODT self-extractor
    $installerPath = $env:INSTALLER_PATH
    
    # Extract ODT
    Start-Process -FilePath $installerPath -ArgumentList "/quiet /extract:`"$odtPath`"" -Wait -NoNewWindow
    
    # Detect existing Office installation and which apps are installed
    $clickToRunPath = "HKLM:\SOFTWARE\Microsoft\Office\ClickToRun\Configuration"
    $existingApps = @()
    
    if (Test-Path $clickToRunPath) {
        # Get the current configuration
        $excludedApps = (Get-ItemProperty -Path $clickToRunPath -Name "O365BusinessRetail.ExcludedApps" -ErrorAction SilentlyContinue)."O365BusinessRetail.ExcludedApps"
        
        # All possible apps
        $allApps = @("Access", "Excel", "Groove", "Lync", "OneDrive", "OneNote", "Outlook", "PowerPoint", "Publisher", "Teams", "Word")
        
        if ($excludedApps) {
            # Parse excluded apps and determine what's currently installed
            $excludedList = $excludedApps -split ","
            $existingApps = $allApps | Where-Object { $_ -notin $excludedList }
        }
    }
    
    # Build exclusion list - exclude everything EXCEPT Excel and any existing apps
    $appsToInstall = @("Excel") + $existingApps | Select-Object -Unique
    $appsToExclude = @("Access", "Excel", "Groove", "Lync", "OneDrive", "OneNote", "Outlook", "PowerPoint", "Publisher", "Teams", "Word") | Where-Object { $_ -notin $appsToInstall }
    
    # Build ExcludeApp XML elements
    $excludeElements = ($appsToExclude | ForEach-Object { "      <ExcludeApp ID=`"$_`" />" }) -join "`n"
    
    # Generate configuration XML
    $configXml = @"
<Configuration>
  <Add OfficeClientEdition="64" Channel="Current">
    <Product ID="O365BusinessRetail">
      <Language ID="MatchOS" />
$excludeElements
    </Product>
  </Add>
  <Property Name="AUTOACTIVATE" Value="0" />
  <Property Name="FORCEAPPSHUTDOWN" Value="TRUE" />
  <Updates Enabled="TRUE" />
  <Display Level="None" AcceptEULA="TRUE" />
  <Logging Level="Standard" Path="$env:TEMP" />
</Configuration>
"@

    # Write config file
    $configPath = "$odtPath\config.xml"
    $configXml | Out-File -FilePath $configPath -Encoding UTF8
    
    # Run ODT
    $process = Start-Process -FilePath "$odtPath\setup.exe" -ArgumentList "/configure `"$configPath`"" -Wait -PassThru -NoNewWindow
    
    $exitCode = $process.ExitCode
}
catch {
    Write-Error $_.Exception.Message
    $exitCode = 1
}
finally {
    # Cleanup
    Remove-Item -Path $odtPath -Recurse -Force -ErrorAction SilentlyContinue
}

exit $exitCode
