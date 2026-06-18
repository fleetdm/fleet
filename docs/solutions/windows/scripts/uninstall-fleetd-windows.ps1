# Please don't delete. This script is referenced in the guides here:
#   - https://fleetdm.com/guides/windows-mdm-setup#turn-off-windows-mdm
#   - https://fleetdm.com/guides/how-to-uninstall-fleetd

Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;

public class MdmRegistration
{
    [DllImport("mdmregistration.dll", SetLastError = true)]
    public static extern int UnregisterDeviceWithManagement(IntPtr pDeviceID);

    public static int UnregisterDevice()
    {
        return UnregisterDeviceWithManagement(IntPtr.Zero);
    }
}
"@ -Language CSharp

function Test-Administrator
{
    [OutputType([bool])]
    param()
    process {
        [Security.Principal.WindowsPrincipal]$user = [Security.Principal.WindowsIdentity]::GetCurrent();
        return $user.IsInRole([Security.Principal.WindowsBuiltinRole]::Administrator);
    }
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

#Remove Orbit footprint from registry and disk
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

    # Write success log
    "Fleetd successfully removed at $(Get-Date)" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"
  }
  catch {
    Write-Host "There was a problem running Force-Remove-Orbit"
    Write-Host "$(Resolve-Error-Detailed)"
    # Write error log
    "Error removing fleetd at $(Get-Date): $($Error[0])" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"
    return $false
  }

  return $true
}

function Main {
  try {
    # Is Administrator check
    if (-not (Test-Administrator)) {
      Write-Host "Please run this script with admin privileges."
      Exit -1
    }

    if ($args[0] -eq "remove") {
      # "remove" is received as argument to the script when called as the
      # sub-process that will actually remove the fleet agent.

      # Log the start of removal process
      "Starting removal process at $(Get-Date)" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"

      # sleep to give time to fleetd to send the script results to Fleet
      Start-Sleep -Seconds 20

      if (Force-Remove-Orbit) {
        Write-Host "fleetd was uninstalled."
        Exit 0
      } else {
        Write-Host "There was a problem uninstalling fleetd."
        Exit -1
      }
    } else {
      # Turn off MDM first so Fleet cannot re-enable it before fleetd is removed.

      # Check 1: Fleet-specific enrollment (ProviderID + EnrollmentState)
      $enrollmentKey = Get-Item -Path HKLM:\SOFTWARE\Microsoft\Enrollments\* -ErrorAction SilentlyContinue | Get-ItemProperty | Where-Object {$_.ProviderID -eq 'Fleet'} | Where-Object {$_.EnrollmentState -match '1|3|6|13'}
      $mdmEnrolled = $null -ne $enrollmentKey

      # Check 2: fallback via DiscoveryServiceFullURL
      $enrollmentsPath = "HKLM:\SOFTWARE\Microsoft\Enrollments"
      if (-not $mdmEnrolled) {
          if (Test-Path $enrollmentsPath) {
              $enrollmentKeys = Get-ChildItem -Path $enrollmentsPath -ErrorAction SilentlyContinue
              foreach ($key in $enrollmentKeys) {
                  if ($null -ne (Get-ItemProperty -Path $key.PSPath -Name "DiscoveryServiceFullURL" -ErrorAction SilentlyContinue)) {
                      $mdmEnrolled = $true
                      break
                  }
              }
          }
      }

      if ($mdmEnrolled) {
          $result = [MdmRegistration]::UnregisterDevice()

          if ($result -ne 0) {
              throw "UnregisterDeviceWithManagement failed with error code: $result"
          }

          Write-Host "Device unregistration called successfully."

          $clearedCount = 0

          if (Test-Path $enrollmentsPath) {
              $enrollmentKeys = Get-ChildItem -Path $enrollmentsPath -ErrorAction SilentlyContinue

              foreach ($key in $enrollmentKeys) {
                  if ($null -ne (Get-ItemProperty -Path $key.PSPath -Name "DiscoveryServiceFullURL" -ErrorAction SilentlyContinue)) {
                      try {
                          Remove-ItemProperty -Path $key.PSPath -Name "DiscoveryServiceFullURL" -ErrorAction Stop
                          $clearedCount++
                          Write-Host "Cleared DiscoveryServiceFullURL from enrollment key: $($key.PSChildName)"
                      } catch {
                          Write-Warning "Failed to clear DiscoveryServiceFullURL from $($key.PSChildName): $_"
                      }
                  }
              }
          }

          if ($clearedCount -gt 0) {
              Write-Host "Cleared DiscoveryServiceFullURL from $clearedCount enrollment key(s)."
          } else {
              Write-Host "Turning off MDM completed. The UnregisterDeviceWithManagement API automatically cleared the registry values."
          }
      } else {
          Write-Host "MDM is not turned on. Skipping MDM unregistration."
      }

      # when this script is executed from fleetd, it does not immediately
      # remove the agent. Instead, it starts a new detached process that
      # will do the actual removal.

      Write-Host "Removing fleetd, system will be unenrolled in 20 seconds..."
      Write-Host "Executing detached child process"

      $execName = $MyInvocation.ScriptName
      $proc = Start-Process -PassThru -FilePath "powershell" -WindowStyle Hidden -ArgumentList "-MTA", "-ExecutionPolicy", "Bypass", "-File", "`"$execName`"", "remove"

      # Log the process ID
      "Started removal process with ID: $($proc.Id) at $(Get-Date)" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"

      Start-Sleep -Seconds 5 # give time to process to start running
      Write-Host "Removal process started: $($proc.Id)."
    }
  } catch {
    Write-Error "Error running fleetd unenrollment script: $_"
    exit 1
  }
}

# Execute the script with arguments passed to it
Main $args[0]
