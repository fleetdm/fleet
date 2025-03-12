#Uninstall Orbit: powershell.exe -ExecutionPolicy Bypass -File cleanup_windows.ps1 -uninstallOrbit
#Help menu with available options: powershell.exe -ExecutionPolicy Bypass -File cleanup_windows.ps1 -help
param(
  [switch] $uninstallOsquery = $false,
  [switch] $uninstallOrbit = $false,
  [switch] $stopOrbit = $false,
  [switch] $force = $false,
  [switch] $help = $false
)

#ErrorActionPreference valid values are as follows:
#  Break: Enter the debugger when an error occurs or when an exception is raised.
#  Continue: (Default) Displays the error message and continues executing.
#  Ignore: Suppresses the error message and continues to execute the command. The Ignore value is intended for per-command use, not for use as saved preference. Ignore isn't a valid value for the $ErrorActionPreference variable.
#  Inquire: Displays the error message and asks you whether you want to continue.
#  SilentlyContinue: No effect. The error message isn't displayed and execution continues without interruption.
#  Stop: Displays the error message and stops executing. In addition to the error generated, the Stop value generates an ActionPreferenceStopException object to the error stream.
#  Suspend: Automatically suspends a workflow job to allow for further investigation. After investigation, the workflow can be resumed. The Suspend value is intended for per-command use, not for use as saved preference. Suspend isn't a valid value for the $ErrorActionPreference variable.

$ErrorActionPreference = "Continue"


$code = @"
using Microsoft.Win32;
using System;
using System.IO;
using System.Runtime.InteropServices;
public class RegistryUtils
{

    [DllImport("kernel32.dll", SetLastError = true)]
    internal static extern IntPtr GetCurrentProcess();

    [DllImport("advapi32.dll", ExactSpelling = true, SetLastError = true)]
    internal static extern bool AdjustTokenPrivileges(IntPtr htok, bool disall,
        ref TokPriv1Luid newst, int len, IntPtr prev, IntPtr relen);

    [DllImport("advapi32.dll", ExactSpelling = true, SetLastError = true)]
    internal static extern bool OpenProcessToken(IntPtr h, int acc, ref IntPtr phtok);

    [DllImport("advapi32.dll", SetLastError = true)]
    internal static extern bool LookupPrivilegeValue(string host, string name, ref long pluid);

    [DllImport("advapi32.dll", SetLastError = true)]
    static extern int RegLoadKey(UInt32 hKey, String lpSubKey, String lpFile);

    [DllImport("advapi32.dll", SetLastError = true)]
    static extern int RegUnLoadKey(UInt32 hKey, string lpSubKey);

    [StructLayout(LayoutKind.Sequential, Pack = 1)]
    internal struct TokPriv1Luid
    {
        public int Count;
        public long Luid;
        public int Attr;
    }

    internal const int SE_PRIVILEGE_ENABLED = 0x00000002;
    internal const int SE_PRIVILEGE_DISABLED = 0x00000000;
    internal const int TOKEN_QUERY = 0x00000008;
    internal const int TOKEN_ADJUST_PRIVILEGES = 0x00000020;

    public static bool EnablePrivilege(string privilege, bool disable)
    {
        TokPriv1Luid tp;
        IntPtr htok = IntPtr.Zero;

        if (!OpenProcessToken(GetCurrentProcess(), TOKEN_ADJUST_PRIVILEGES | TOKEN_QUERY, ref htok))
        {
            Console.WriteLine("EnablePrivilege - Failed to obtain handle to primary access token");
            return false;
        }

        tp.Count = 1;
        tp.Luid = 0;

        if (disable)
        {
            tp.Attr = SE_PRIVILEGE_DISABLED;
        }
        else
        {
            tp.Attr = SE_PRIVILEGE_ENABLED;
        }

        if (!LookupPrivilegeValue(null, privilege, ref tp.Luid))
        {
            Console.WriteLine("EnablePrivilege - Failed to lookup privilege {0}", privilege);
            return false;
        }

        if (!AdjustTokenPrivileges(htok, false, ref tp, 0, IntPtr.Zero, IntPtr.Zero))
        {
            Console.WriteLine("EnablePrivilege - Failed to modify privilege {0}", privilege);
            return false;
        }

        return true;
    }


    public static void LoadUsersHives()
    {
        try
        {
            if (EnablePrivilege("SeRestorePrivilege", false) && EnablePrivilege("SeBackupPrivilege", false))
            {
                string usersRoot = System.Environment.GetEnvironmentVariable("SystemDrive") + "\\users\\";
                string[] userDir = System.IO.Directory.GetDirectories(usersRoot, "*", System.IO.SearchOption.TopDirectoryOnly);

                foreach (string fullDirectoryName in userDir)
                {
                    string userDirName = new DirectoryInfo(fullDirectoryName).Name;
                    if (userDirName.Length == 0)
                    {
                        continue;
                    }

                    string targetFile = fullDirectoryName + "\\NTUSER.DAT";
                    if (!File.Exists(targetFile))
                    {
                        continue;
                    }

                    const uint HKEY_USERS = 0x80000003;
                    int retRegLoadKey = RegLoadKey(HKEY_USERS, userDirName, targetFile);
                    Console.WriteLine("RegLoadKey result was {0}", retRegLoadKey);
                }
            }
        }
        catch
        {

        }
    }

    public static void UnloadUsersHives()
    {
        try
        {
            if (EnablePrivilege("SeRestorePrivilege", false) && EnablePrivilege("SeBackupPrivilege", false))
            {
                string usersRoot = System.Environment.GetEnvironmentVariable("SystemDrive") + "\\users\\";
                string[] userDir = System.IO.Directory.GetDirectories(usersRoot, "*", System.IO.SearchOption.TopDirectoryOnly);

                foreach (string fullDirectoryName in userDir)
                {
                    var userDirName = new DirectoryInfo(fullDirectoryName).Name;
                    if (userDirName.Length == 0)
                    {
                        continue;
                    }

                    const uint HKEY_USERS = 0x80000003;
                    int retRegUnLoadKey = RegUnLoadKey(HKEY_USERS, userDirName);
                    Console.WriteLine("RegUnLoadKey result was {0}", retRegUnLoadKey);
                }
            }
        }
        catch
        {

        }
    }

    public static void RemoveOsqueryInstallationFromUserHives()
    {
        try
        {
            LoadUsersHives();

            foreach (var username in Registry.Users.GetSubKeyNames())
            {
                string targetKey = username + "\\SOFTWARE\\Microsoft\\Installer\\Products\\";
                RegistryKey rootKey = Registry.Users.OpenSubKey(targetKey, writable: true);

                if (rootKey != null)
                {
                    foreach (var productEntry in rootKey.GetSubKeyNames())
                    {
                        RegistryKey productKey = rootKey.OpenSubKey(productEntry);
                        if (productKey != null)
                        {
                            if ((string)productKey.GetValue("ProductName") == "osquery")
                            {
                                productKey.Close();
                                rootKey.DeleteSubKeyTree(productEntry);
                            }
                        }
                    }
                }
            }

            UnloadUsersHives();
        }
        catch (Exception ex)
        {
            Console.WriteLine("There was an exception when removing osquery installer from user hives: {0}", ex.ToString());
        }
    }
}
"@

Add-Type -TypeDefinition $code -Language CSharp

function Test-Administrator  
{  
    [OutputType([bool])]
    param()
    process {
        [Security.Principal.WindowsPrincipal]$user = [Security.Principal.WindowsIdentity]::GetCurrent();
        return $user.IsInRole([Security.Principal.WindowsBuiltinRole]::Administrator);
    }
}

function Do-Help {
  $programName = (Get-Item $PSCommandPath ).Name
  
  Write-Host "Usage: $programName (-uninstallOsquery|-uninstallOrbit|-stopOrbit|-force|-help)" -foregroundcolor Yellow
  Write-Host ""
  Write-Host "  Only one of the following options can be used. Using multiple will result in "
  Write-Host "  options being ignored."
  Write-Host "    -uninstallOsquery         Uninstall Osquery"
  Write-Host "    -uninstallOrbit           Uninstall Orbit"
  Write-Host "    -stopOrbit                Stop Orbit"
  Write-Host "    -force                    Force uninstallation of Orbit or Osquery"  
  Write-Host "    -help                     Shows this help screen"
  
  Exit 1
}

# borrowed from Jeffrey Snover http://blogs.msdn.com/powershell/archive/2006/12/07/resolve-error.aspx
function Resolve-Error-Detailed($ErrorRecord = $Error[0]) {
  $error_message = "========== ErrorRecord:{0}ErrorRecord.InvocationInfo:{1}Exception:{2}"
  $formatted_errorRecord = $ErrorRecord | format-list * -force | out-string
  $formatted_invocationInfo = $ErrorRecord.InvocationInfo | format-list * -force | out-string
  $formatted_exception = ""
  $Exception = $ErrorRecord.Exception
  for ($i = 0; $Exception; $i++, ($Exception = $Exception.InnerException)) {
    $formatted_exception += ("$i" * 70) + "-----"
    $formatted_exception += $Exception | format-list * -force | out-string
    $formatted_exception += "-----"
  }

  return $error_message -f $formatted_errorRecord, $formatted_invocationInfo, $formatted_exception
}

#Stops Osquery service and related processes
function Stop-Osquery {

  $kServiceName = "osqueryd"
  
  # Stop Service
  Stop-Service -Name $kServiceName -ErrorAction "Continue" 
  Start-Sleep -Milliseconds 1000 

  # Ensure that no process left running
  Get-Process -Name $kServiceName -ErrorAction "SilentlyContinue" | Stop-Process -Force
}

#Stops Orbit service and related processes
function Stop-Orbit {

  # Stop Service
  Stop-Service -Name "Fleet osquery" -ErrorAction "Continue"
  Start-Sleep -Milliseconds 1000

  # Ensure that no process left running
  Get-Process -Name "orbit" -ErrorAction "SilentlyContinue" | Stop-Process -Force
  Get-Process -Name "osqueryd" -ErrorAction "SilentlyContinue" | Stop-Process -Force
  Get-Process -Name "fleet-desktop" -ErrorAction "SilentlyContinue" | Stop-Process -Force
  Start-Sleep -Milliseconds 1000
}


#Revove Orbit footprint from registry and disk
function Force-Remove-Orbit {

  try {

    #Stoping Orbit
    Stop-Orbit

    #Remove Service
    $service = Get-WmiObject -Class Win32_Service -Filter "Name='Fleet osquery'"
    if ($service) {
      $service.delete() | Out-Null
    }

    #Removing Program files entries
    $targetPath = $Env:Programfiles + "\\Orbit"
    Remove-Item -LiteralPath $targetPath -Force -Recurse -ErrorAction "Continue"

    #Remove HKLM registry entries
    Get-ChildItem "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall" -Recurse  -ErrorAction "SilentlyContinue" |  Where-Object {($_.ValueCount -gt 0)} | ForEach-Object {
      
      # Filter for osquery entries
      $properties = Get-ItemProperty $_.PSPath  -ErrorAction "SilentlyContinue" |  Where-Object {($_.DisplayName -eq "Fleet osquery")}     
      if ($properties) {
      
        #Remove Registry Entries
        $regKey = "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\" + $_.PSChildName

        Get-Item $regKey -ErrorAction "SilentlyContinue" | Remove-Item -Force -ErrorAction "SilentlyContinue"

        return
      }
    }
  }
  catch { 
    Write-Host "There was a problem running Force-Remove-Orbit" -ForegroundColor Red
    Write-Host "====================================="
    Write-Host "$(Resolve-Error-Detailed)"
    Write-Host "====================================="
    return $false
  }
  
  return $true
}


#Revove Osquery footprint from registry and disk
function Force-Remove-Osquery {

  try {

    #Stoping Osquery
    Stop-Osquery

    #Remove Service
    $service = Get-WmiObject -Class Win32_Service -Filter "Name='osqueryd'"
    if ($service) {
      $service.delete() | Out-Null
    }

    #Remove HKLM registry entries and disk footprint
    Get-ChildItem "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall" -Recurse -ErrorAction "SilentlyContinue" |  Where-Object {($_.ValueCount -gt 0)} | ForEach-Object {
      
      # Filter for osquery entries
      $properties = Get-ItemProperty $_.PSPath -ErrorAction "SilentlyContinue" |  Where-Object {($_.DisplayName -eq "osquery")}     
      if ($properties) {

        #Remove files from osquery location 
        if ($properties.InstallLocation){
          Remove-Item -LiteralPath $properties.InstallLocation -Force -Recurse -ErrorAction "SilentlyContinue"
        }
        
        #Remove Registry Entries
        $regKey = "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\" + $_.PSChildName
        Get-Item $regKey -ErrorAction "SilentlyContinue" | Remove-Item -Force -ErrorAction "SilentlyContinue"

        return
      }
    }   

    #Remove user entries if present
    [RegistryUtils]::RemoveOsqueryInstallationFromUserHives()
  }
  catch { 
    Write-Host "There was a problem running Force-Remove-Osquery" -ForegroundColor Red
    Write-Host "====================================="
    Write-Host "$(Resolve-Error-Detailed)"
    Write-Host "====================================="

    return $false
  }

  return $true
}

function Graceful-Product-Uninstall($productName) {

  try {

    if (!$productName) {
      Write-Host "Product name should be provided" -foregroundcolor Yellow
      return $false
    }

    if ($productName -eq "Fleet osquery") {
      Stop-Orbit
    } elseif ($productName -eq "osquery") {
      Stop-Osquery
    }
  
    # Grabbing the location of msiexec.exe
    $targetBinPath = Resolve-Path "$env:windir\system32\msiexec.exe"
    if (!(Test-Path $targetBinPath)) {
      Write-Host "msiexec.exe cannot be located." -foregroundcolor Yellow
      return $false
    }
  
    # Creating a COM instance of the WindowsInstaller.Installer COM object
    $Installer = New-Object -ComObject WindowsInstaller.Installer
    if (!$Installer) {
      Write-Host "There was a problem retrieving the installed packages." -foregroundcolor Yellow
      return $false
    }
  
    # Enumerating the installed packages
    $ProductEnumFlag = 7 #installed packaged enumeration flag
    $InstallerProducts = $Installer.ProductsEx("", "", $ProductEnumFlag); 
    if (!$InstallerProducts) {
      Write-Host "Installed packages cannot be retrieved." -foregroundcolor Yellow
      return $false
    }
  
    # Iterating over the installed packages results and checking for osquery package
    ForEach ($Product in $InstallerProducts) {
  
        $ProductCode = $null
        $VersionString = $null
        $ProductPath = $null
  
        $ProductCode = $Product.ProductCode()
        $VersionString = $Product.InstallProperty("VersionString")
        $ProductPath = $Product.InstallProperty("ProductName")
  
        if ($ProductPath -like $productName) {
          Write-Host "Graceful uninstall of $ProductPath version $VersionString."  -foregroundcolor Cyan
          $InstallProcess = Start-Process $targetBinPath -ArgumentList "/quiet /x $ProductCode" -PassThru -Verb RunAs -Wait
          if ($InstallProcess.ExitCode -eq 0) {
            return $true
          } else {
            Write-Host "There was an error uninstalling osquery. Error code was: $($InstallProcess.ExitCode)." -foregroundcolor Yellow
            return $false
          }
        }
    }
  }
  catch { 
    Write-Host "There was a problem running Graceful-Product-Uninstall" -ForegroundColor Red
    Write-Host "====================================="
    Write-Host "$(Resolve-Error-Detailed)"
    Write-Host "====================================="
  }    

  return $false
}


function Main {

  try {
    # Is Administrator check
    if (-not (Test-Administrator)) {
      Write-Host "Please run this script with Admin privileges!" -foregroundcolor Red
      Exit -1
    }

    # Help commands
    if ($help) {
      Do-Help
      Exit -1
    } 
      
    if ($uninstallOsquery) {
      Write-Host "About to uninstall Osquery." -foregroundcolor Yellow
   
      if ($force) {
        if (Force-Remove-Osquery) {
            Write-Host "Osquery was force uninstalled." -foregroundcolor Cyan
            Exit 0
        } else {
            Write-Host "There was a problem force uninstalling Osquery" -foregroundcolor Cyan
            Exit -1
        }
      } else {
        if (Graceful-Product-Uninstall("osquery")) {
            Write-Host "Osquery was uninstalled." -foregroundcolor Cyan
            Exit 0
        } else {
            Write-Host "There was a problem uninstalling Osquery" -foregroundcolor Cyan
            Exit -1
        }         
      }
      
    } elseif ($uninstallOrbit) {
      Write-Host "About to uninstall Orbit." -foregroundcolor Yellow
    
      if ($force) {
        if (Force-Remove-Orbit) {
            Write-Host "Orbit was force uninstalled." -foregroundcolor Cyan
            Exit 0
        } else {
            Write-Host "There was a problem force uninstalling Orbit" -foregroundcolor Cyan
            Exit -1
        }
      } else {
        if (Graceful-Product-Uninstall("Fleet osquery")) {
            Write-Host "Orbit was uninstalled." -foregroundcolor Cyan
            Exit 0
        } else {
            Write-Host "There was a problem uninstalling Orbit" -foregroundcolor Cyan
            Exit -1
        }         
      }
    } elseif ($stopOrbit) {
      Write-Host "About to stop Orbit and remove it from system." -foregroundcolor Yellow

      Stop-Orbit

      Write-Host "Orbit was stopped." -foregroundcolor Cyan
      Exit 0

    } else {
      Write-Host "Invalid option selected: please see -help for usage details." -foregroundcolor Red
      Do-Help
      Exit -1
    }
  } catch {
    Write-Host "There was a problem running installer entry point logic" -ForegroundColor Red    
    Write-Host "====================================="
    Write-Host "$(Resolve-Error-Detailed)"
    Write-Host "====================================="
    Exit -1
  }
}

$null = Main