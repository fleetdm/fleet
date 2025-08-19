# Please don't delete. This script is referenced in the guide here: https://fleetdm.com/guides/how-to-uninstall-fleetd

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

#Remove MDM enrollment
function Remove-MDM-Enrollment {
  try {
    Write-Host "Removing MDM enrollment..."
    
    # Remove MDM enrollment certificates
    $certificates = Get-ChildItem -Path "Cert:\LocalMachine\My" | Where-Object {$_.Subject -like "*MDM*" -or $_.Subject -like "*DeviceManagement*"}
    foreach ($cert in $certificates) {
      Remove-Item -Path $cert.PSPath -Force -ErrorAction "Continue"
      Write-Host "Removed certificate: $($cert.Subject)"
    }
    
    # Remove MDM enrollment registry entries
    $mdmPaths = @(
      "HKLM:\SOFTWARE\Microsoft\Enrollments",
      "HKLM:\SOFTWARE\Microsoft\Enrollments\Status",
      "HKLM:\SOFTWARE\Microsoft\PolicyManager",
      "HKLM:\SOFTWARE\Microsoft\PolicyManager\AdmxInstall",
      "HKLM:\SOFTWARE\Microsoft\PolicyManager\Providers",
      "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Aik\Certificates",
      "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Aik\Certificates\S-1-5-18"
    )
    
    foreach ($path in $mdmPaths) {
      if (Test-Path $path) {
        Remove-Item -Path $path -Recurse -Force -ErrorAction "Continue"
        Write-Host "Removed registry path: $path"
      }
    }
    
    # Remove specific MDM enrollment keys
    $enrollmentKeys = Get-ChildItem "HKLM:\SOFTWARE\Microsoft\Enrollments" -ErrorAction "SilentlyContinue"
    foreach ($key in $enrollmentKeys) {
      if ($key.Name -match "MDM|DeviceManagement") {
        Remove-Item -Path $key.PSPath -Recurse -Force -ErrorAction "Continue"
        Write-Host "Removed enrollment key: $($key.Name)"
      }
    }
    
    # Remove MDM scheduled tasks
    $mdmTasks = Get-ScheduledTask | Where-Object {$_.TaskName -like "*MDM*" -or $_.TaskName -like "*DeviceManagement*"}
    foreach ($task in $mdmTasks) {
      Unregister-ScheduledTask -TaskName $task.TaskName -Confirm:$false -ErrorAction "Continue"
      Write-Host "Removed scheduled task: $($task.TaskName)"
    }
    
    # Write success log
    "MDM enrollment successfully removed at $(Get-Date)" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"
    return $true
  }
  catch {
    Write-Host "There was a problem removing MDM enrollment"
    Write-Host "$(Resolve-Error-Detailed)"
    # Write error log
    "Error removing MDM enrollment at $(Get-Date): $($Error[0])" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"
    return $false
  }
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

    Write-Host "About to uninstall fleetd and remove MDM enrollment..."

    if ($args[0] -eq "remove") {
      # "remove" is received as argument to the script when called as the
      # sub-process that will actually remove the fleet agent.

      # Log the start of removal process
      "Starting removal process at $(Get-Date)" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"
      
      # sleep to give time to fleetd to send the script results to Fleet
      Start-Sleep -Seconds 20
      
      $orbitRemoved = Force-Remove-Orbit
      
      # Check if MDM is enabled before attempting removal
      $mdmEnabled = Test-Path "HKLM:\SOFTWARE\Microsoft\Enrollments" -ErrorAction "SilentlyContinue"
      if ($mdmEnabled) {
        $mdmRemoved = Remove-MDM-Enrollment
      } else {
        Write-Host "MDM not detected on this system, skipping MDM removal."
        $mdmRemoved = $true  # Consider it "successful" since there's nothing to remove
      }
      
      if ($orbitRemoved -and $mdmRemoved) {
        if ($mdmEnabled) {
          Write-Host "fleetd and MDM enrollment were successfully removed."
        } else {
          Write-Host "fleetd was successfully removed. (No MDM enrollment detected)"
        }
        Exit 0
      } elseif ($orbitRemoved) {
        Write-Host "fleetd was uninstalled, but there was a problem removing MDM enrollment."
        Exit 1
      } elseif ($mdmRemoved) {
        Write-Host "MDM enrollment was removed, but there was a problem uninstalling fleetd."
        Exit 1
      } else {
        Write-Host "There were problems uninstalling both fleetd and MDM enrollment."
        Exit -1
      }
    } else {
      # when this script is executed from fleetd, it does not immediately
      # remove the agent. Instead, it starts a new detached process that
      # will do the actual removal.
      
      Write-Host "Removing fleetd and MDM enrollment, system will be unenrolled in 20 seconds..."
      Write-Host "Executing detached child process"
      
      $execName = $MyInvocation.ScriptName
      $proc = Start-Process -PassThru -FilePath "powershell" -WindowStyle Hidden -ArgumentList "-MTA", "-ExecutionPolicy", "Bypass", "-File", "$execName remove"
      
      # Log the process ID
      "Started removal process with ID: $($proc.Id) at $(Get-Date)" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"
      
      Start-Sleep -Seconds 5 # give time to process to start running
      Write-Host "Removal process started: $($proc.Id)."
    }
  } catch {
    Write-Host "Error: Entry point"
    Write-Host "$(Resolve-Error-Detailed)"
    Exit -1
  }
}

# Execute the script with arguments passed to it
Main $args[0]
