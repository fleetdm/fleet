# Learn more about install scripts:
# http://fleetdm.com/learn-more-about/install-scripts
#
# Tower's MSI installs per user by default (ALLUSERS=2 + MSIINSTALLPERUSER=1),
# so under Fleet's SYSTEM context it lands in the SYSTEM account's profile,
# where no one can open it. Force a per-machine install into Program Files, and
# first remove a copy that an earlier install left in the SYSTEM profile.

$logFile = "${env:TEMP}/fleet-install-software.log"
$upgradeCode = '{871FD9D0-41D3-52BE-AF69-12F8B08740C0}'

# 3010 = reboot required, 1641 = reboot initiated
$successCodes = @(0, 3010, 1641)

try {

$installer = New-Object -ComObject "WindowsInstaller.Installer"

# Tower needs the x64 .NET Desktop Runtime, which its MSI only installs when run
# with a UI. Fail with an actionable message rather than leave behind an app
# that can't start. The required major version comes from the MSI itself.
$database = $installer.OpenDatabase("${env:INSTALLER_PATH}", 0)
$view = $database.OpenView("SELECT Value FROM Property WHERE Property = 'RustRuntimeDependencies'")
$view.Execute()
$record = $view.Fetch()
$dependencies = if ($record) { $record.StringData(1) } else { "" }
$view.Close()
$null = [System.Runtime.InteropServices.Marshal]::ReleaseComObject($view)
$null = [System.Runtime.InteropServices.Marshal]::ReleaseComObject($database)

if ($dependencies -match 'net(\d+)-x64-desktop') {
  $major = $matches[1]
  $dotnetRoots = @((Get-ItemProperty 'HKLM:\SOFTWARE\dotnet\Setup\InstalledVersions\x64' -ErrorAction SilentlyContinue).InstallLocation)
  # On ARM64 Windows the x64 runtime installs side by side under dotnet\x64.
  $nativeArch = (Get-ItemProperty 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment').PROCESSOR_ARCHITECTURE
  if ($nativeArch -eq 'ARM64') {
    $dotnetRoots += Join-Path $env:ProgramFiles "dotnet\x64"
  } else {
    $dotnetRoots += Join-Path $env:ProgramFiles "dotnet"
  }

  $hasRuntime = $dotnetRoots | Where-Object { $_ } | Where-Object {
    Get-ChildItem -Path (Join-Path $_ "shared\Microsoft.WindowsDesktop.App") -Directory -ErrorAction SilentlyContinue |
      Where-Object { $_.Name -like "$major.*" }
  } | Select-Object -First 1

  if (-not $hasRuntime) {
    Write-Host "Tower requires the Microsoft .NET Desktop Runtime $major (x64), which was not found on this host."
    Write-Host "Install it (for example, with the Microsoft .NET Desktop Runtime $major Fleet-maintained app), then retry."
    Exit 1
  }
}

foreach ($productCode in @($installer.RelatedProducts($upgradeCode))) {
  # 0 = installed for this account only; 1 = installed machine-wide
  if ($installer.ProductInfo($productCode, "AssignmentType") -ne "0") { continue }

  Write-Host "Removing the copy of Tower in the SYSTEM profile ($productCode)"
  $removeProcess = Start-Process msiexec.exe `
    -ArgumentList "/quiet /norestart /x $productCode" `
    -PassThru -Wait
  if ($successCodes -notcontains $removeProcess.ExitCode) {
    Write-Host "Removal exited $($removeProcess.ExitCode)"
    Exit $removeProcess.ExitCode
  }
}

$installProcess = Start-Process msiexec.exe `
  -ArgumentList "/quiet /norestart /lv `"${logFile}`" ALLUSERS=1 MSIINSTALLPERUSER=`"`" /i `"${env:INSTALLER_PATH}`"" `
  -PassThru -Verb RunAs -Wait

Get-Content $logFile -Tail 500

if ($successCodes -contains $installProcess.ExitCode) {
  Exit 0
}

Exit $installProcess.ExitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
