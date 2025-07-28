# Please don't delete. This script is referenced in the guide here: https://fleetdm.com/guides/how-to-uninstall-fleetd

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
  try {
    # Stop Service
    Stop-Service -Name "Fleet osquery" -ErrorAction "Continue"
    Start-Sleep -Milliseconds 1000

    # Ensure that no process left running
    Get-Process -Name "orbit" -ErrorAction "SilentlyContinue" | Stop-Process -Force
    Get-Process -Name "osqueryd" -ErrorAction "SilentlyContinue" | Stop-Process -Force
    Get-Process -Name "fleet-desktop" -ErrorAction "SilentlyContinue" | Stop-Process -Force
    Start-Sleep -Milliseconds 1000
    
    "Orbit processes stopped successfully at $(Get-Date)" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"
  }
  catch {
    "Error stopping Orbit processes at $(Get-Date): $($Error[0])" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"
    Write-Host "Warning: Some processes may still be running"
  }
}

#Remove Orbit footprint from registry and disk
function Force-Remove-Orbit {
  try {
    #Stopping Orbit
    Stop-Orbit

    #Remove Service - Updated to use Get-CimInstance instead of deprecated Get-WmiObject
    try {
      $service = Get-CimInstance -ClassName Win32_Service -Filter "Name='Fleet osquery'" -ErrorAction SilentlyContinue
      if ($service) {
        Remove-CimInstance -InputObject $service -ErrorAction SilentlyContinue
        "Service removed successfully at $(Get-Date)" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"
      }
    }
    catch {
      "Error removing service at $(Get-Date): $($Error[0])" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"
    }

    #Removing Program files entries
    $targetPath = $Env:Programfiles + "\\Orbit"
    if (Test-Path -LiteralPath $targetPath) {
      Remove-Item -LiteralPath $targetPath -Force -Recurse -ErrorAction "Continue"
      "Program files directory removed at $(Get-Date)" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"
    }

    #Remove HKLM registry entries - Improved logic
    try {
      $uninstallKeys = Get-ChildItem "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall" -ErrorAction "SilentlyContinue"
      foreach ($key in $uninstallKeys) {
        $properties = Get-ItemProperty $key.PSPath -ErrorAction "SilentlyContinue"
        if ($properties.DisplayName -eq "Fleet osquery") {
          Remove-Item $key.PSPath -Force -ErrorAction "SilentlyContinue"
          "Registry entry removed at $(Get-Date)" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"
          break
        }
      }
    }
    catch {
      "Error removing registry entries at $(Get-Date): $($Error[0])" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"
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
    Write-Host "About to uninstall fleetd..."

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
      # when this script is executed from fleetd, it does not immediately
      # remove the agent. Instead, it starts a new detached process that
      # will do the actual removal.
      
      Write-Host "Removing fleetd, system will be unenrolled in 20 seconds..."
      Write-Host "Executing detached child process"
      
      $execName = $MyInvocation.ScriptName
      $proc = Start-Process -PassThru -FilePath "powershell" -WindowStyle Hidden -ArgumentList "-MTA", "-ExecutionPolicy", "Bypass", "-File", "$execName remove"
      
      # Verify process started successfully
      Start-Sleep -Seconds 2
      try {
        $verifyProc = Get-Process -Id $proc.Id -ErrorAction SilentlyContinue
        if ($verifyProc) {
          # Log the process ID
          "Started removal process with ID: $($proc.Id) at $(Get-Date)" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"
          Write-Host "Removal process started: $($proc.Id)."
        } else {
          throw "Process verification failed"
        }
      }
      catch {
        Write-Host "Error: Failed to start removal process"
        "Failed to start removal process at $(Get-Date): $($Error[0])" | Out-File -Append -FilePath "$env:TEMP\fleet_remove_log.txt"
        Exit -1
      }
    }
  } catch {
    Write-Host "Error: Entry point"
    Write-Host "$(Resolve-Error-Detailed)"
    Exit -1
  }
}

# Execute the script with arguments passed to it
Main $args[0]
