# 1Password Uninstall Script
# This script uninstalls 1Password using the product code with improved reliability

$productCode = "{7DB53C29-932E-4F28-9AD6-85E66EDE3DB2}"
$timeoutSeconds = 300  # 5 minute timeout
$logFile = "${env:TEMP}\fleet-1password-uninstall.log"

try {
    # Start transcript for logging
    Start-Transcript -Path $logFile -Append -ErrorAction SilentlyContinue
    
    Write-Host "Starting 1Password uninstall process..."
    Write-Host "Product Code: $productCode"
    
    # Step 1: Close any running 1Password processes
    Write-Host "Checking for running 1Password processes..."
    $processes = Get-Process -Name "1Password*" -ErrorAction SilentlyContinue
    if ($processes) {
        Write-Host "Found $($processes.Count) running 1Password process(es), attempting to close..."
        foreach ($proc in $processes) {
            try {
                Write-Host "Closing process: $($proc.ProcessName) (PID: $($proc.Id))"
                Stop-Process -Id $proc.Id -Force -ErrorAction Stop
                Start-Sleep -Seconds 2
            } catch {
                Write-Host "Warning: Could not close process $($proc.Id): $($_.Exception.Message)"
            }
        }
        # Wait a bit more to ensure processes are fully terminated
        Start-Sleep -Seconds 3
    } else {
        Write-Host "No running 1Password processes found."
    }
    
    # Step 2: Verify the product is installed before attempting uninstall
    Write-Host "Verifying 1Password is installed..."
    $installed = Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*" -ErrorAction SilentlyContinue | 
        Where-Object { $_.PSChildName -eq $productCode -or $_.UninstallString -like "*$productCode*" }
    
    if (-not $installed) {
        Write-Host "1Password does not appear to be installed (product code not found in registry)."
        Write-Host "This may indicate it was already uninstalled or installed differently."
        Exit 0
    }
    
    Write-Host "1Password installation confirmed, proceeding with uninstall..."
    
    # Step 3: Execute uninstall with timeout protection
    Write-Host "Executing msiexec uninstall command..."
    $uninstallArgs = @("/quiet", "/x", $productCode, "/norestart", "/L*v", "${env:TEMP}\1password-uninstall-msi.log")
    
    $processStartInfo = New-Object System.Diagnostics.ProcessStartInfo
    $processStartInfo.FileName = "msiexec.exe"
    $processStartInfo.Arguments = ($uninstallArgs -join " ")
    $processStartInfo.UseShellExecute = $false
    $processStartInfo.RedirectStandardOutput = $true
    $processStartInfo.RedirectStandardError = $true
    $processStartInfo.CreateNoWindow = $true
    
    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $processStartInfo
    
    Write-Host "Starting uninstall process (timeout: $timeoutSeconds seconds)..."
    $process.Start() | Out-Null
    
    # Wait for process with timeout
    $completed = $process.WaitForExit($timeoutSeconds * 1000)
    
    if (-not $completed) {
        Write-Host "ERROR: Uninstall process exceeded timeout of $timeoutSeconds seconds. Terminating..."
        try {
            Stop-Process -Id $process.Id -Force -ErrorAction Stop
            Write-Host "Process terminated."
        } catch {
            Write-Host "Warning: Could not terminate process: $($_.Exception.Message)"
        }
        Exit 1603  # ERROR_INSTALL_FAILURE
    }
    
    $exitCode = $process.ExitCode
    Write-Host "Uninstall process completed with exit code: $exitCode"
    
    # Read output if available
    $stdout = $process.StandardOutput.ReadToEnd()
    $stderr = $process.StandardError.ReadToEnd()
    
    if ($stdout) {
        Write-Host "Standard output: $stdout"
    }
    if ($stderr) {
        Write-Host "Standard error: $stderr"
    }
    
    # Check MSI log for additional details
    $msiLogPath = "${env:TEMP}\1password-uninstall-msi.log"
    if (Test-Path $msiLogPath) {
        Write-Host "MSI log file available at: $msiLogPath"
        # Read last 50 lines of MSI log for troubleshooting
        $msiLogTail = Get-Content $msiLogPath -Tail 50 -ErrorAction SilentlyContinue
        if ($msiLogTail) {
            Write-Host "Last 50 lines of MSI log:"
            Write-Host ($msiLogTail -join "`n")
        }
    }
    
    # Step 4: Verify uninstall completed successfully
    if ($exitCode -eq 0) {
        Write-Host "Uninstall completed successfully."
        
        # Double-check by verifying product code is no longer in registry
        Start-Sleep -Seconds 2
        $stillInstalled = Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*" -ErrorAction SilentlyContinue | 
            Where-Object { $_.PSChildName -eq $productCode -or $_.UninstallString -like "*$productCode*" }
        
        if ($stillInstalled) {
            Write-Host "Warning: Product code still found in registry after uninstall. Uninstall may not have completed fully."
        } else {
            Write-Host "Verification: Product code no longer found in registry. Uninstall confirmed."
        }
    } else {
        Write-Host "Uninstall failed with exit code: $exitCode"
        Write-Host "Common exit codes:"
        Write-Host "  0 = Success"
        Write-Host "  1603 = ERROR_INSTALL_FAILURE"
        Write-Host "  1605 = ERROR_UNKNOWN_PRODUCT (product not found)"
        Write-Host "  1619 = ERROR_INSTALL_PACKAGE_OPEN_FAILED"
    }
    
    Stop-Transcript -ErrorAction SilentlyContinue
    Exit $exitCode
    
} catch {
    Write-Host "ERROR: Exception occurred during uninstall: $($_.Exception.Message)"
    Write-Host "Stack trace: $($_.ScriptStackTrace)"
    Stop-Transcript -ErrorAction SilentlyContinue
    Exit 1
}

