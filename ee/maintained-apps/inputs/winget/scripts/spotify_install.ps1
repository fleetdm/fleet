# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"
$taskName = "fleet-install-spotify"
$scriptPath = "$env:PUBLIC\install-spotify.ps1"
$exitCodeFile = "$env:PUBLIC\install-exitcode-spotify.txt"

$userScript = @"
`$exeFilePath = "$exeFilePath"
`$exitCodeFile = "$exitCodeFile"
`$exitCode = 0

try {
    Write-Host "=== Spotify Installation Start ==="
    Write-Host "Installer Path: `$exeFilePath"

    # Verify installer file exists
    if (-not (Test-Path `$exeFilePath)) {
        throw "Installer file not found at: `$exeFilePath"
    }

    # Spotify installer supports /silent for silent installation
    Write-Host "Starting installation with /silent..."
    `$processOptions = @{
        FilePath = "`$exeFilePath"
        ArgumentList = "/silent"
        PassThru = `$true
        Wait = `$true
        NoNewWindow = `$true
    }
    
    `$process = Start-Process @processOptions
    
    if (`$null -eq `$process) {
        throw "Failed to start installer process"
    }
    
    `$exitCode = `$process.ExitCode
    Write-Host "Install exit code: `$exitCode"
    
    # If /silent fails, try /S as fallback
    if (`$exitCode -ne 0) {
        Write-Host "Installation with /silent failed (exit code: `$exitCode), trying /S as fallback..."
        `$fallbackOptions = @{
            FilePath = "`$exeFilePath"
            ArgumentList = "/S"
            PassThru = `$true
            Wait = `$true
            NoNewWindow = `$true
        }
        `$fallbackProcess = Start-Process @fallbackOptions
        if (`$null -ne `$fallbackProcess) {
            `$exitCode = `$fallbackProcess.ExitCode
            Write-Host "Fallback install exit code: `$exitCode"
        }
    }
    
    if (`$exitCode -eq 0) {
        Write-Host "=== Installation Successful ==="
    } else {
        Write-Host "=== Installation Failed ==="
    }
} catch {
    Write-Host "=== Installation Failed ==="
    Write-Host "Error: `$(`$_.Exception.Message)"
    `$exitCode = 1
} finally {
    Write-Host "Exit Code: `$exitCode"
    Set-Content -Path `$exitCodeFile -Value `$exitCode -Force
}

Exit `$exitCode
"@

$exitCode = 0
$publicExePath = $null

try {
    # Wait for an interactive user to be logged on
    while ($true) {
        $userName = (Get-CimInstance Win32_ComputerSystem).UserName

        if ($userName -and $userName -like "*\*") {
            break
        } else {
            Start-Sleep -Seconds 5
        }
    }

    # Copy installer to public folder so user can access it
    $exeFilename = Split-Path $exeFilePath -Leaf
    $publicExePath = "$env:PUBLIC\$exeFilename"
    Copy-Item -Path $exeFilePath -Destination $publicExePath -Force
    $exeFilePath = $publicExePath

    # Update script with correct path
    $userScript = $userScript -replace '"' + [regex]::Escape(${env:INSTALLER_PATH}) + '"', "`"$exeFilePath`""

    # Write the install script to disk
    Set-Content -Path $scriptPath -Value $userScript -Force

    # Build task action: run script
    $action = New-ScheduledTaskAction -Execute "powershell.exe" `
        -Argument "-WindowStyle Hidden -ExecutionPolicy Bypass -File `"$scriptPath`""

    $trigger = New-ScheduledTaskTrigger -AtLogOn

    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries

    $principal = New-ScheduledTaskPrincipal -UserId $userName -RunLevel Highest

    $task = New-ScheduledTask -Action $action -Trigger $trigger -Settings $settings -Principal $principal

    Register-ScheduledTask -TaskName $taskName -InputObject $task -User $userName -Force | Out-Null

    # Start the task
    Start-ScheduledTask -TaskName $taskName

    # Wait for it to start
    $startDate = Get-Date
    $state = (Get-ScheduledTask -TaskName $taskName).State
    while ($state -ne "Running") {
        Start-Sleep -Seconds 1
        $elapsed = (New-Timespan -Start $startDate).TotalSeconds
        if ($elapsed -gt 120) { throw "Timeout waiting for task to start." }
        $state = (Get-ScheduledTask -TaskName $taskName).State
    }

    # Wait for it to complete
    while ($state -eq "Running") {
        Start-Sleep -Seconds 5
        $elapsed = (New-Timespan -Start $startDate).TotalSeconds
        if ($elapsed -gt 300) { throw "Timeout waiting for task to finish." }
        $state = (Get-ScheduledTask -TaskName $taskName).State
    }

    if (Test-Path $exitCodeFile) {
        $exitCode = [int](Get-Content $exitCodeFile)
    } else {
        $exitCode = 1
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
} finally {
    # Clean up
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    Remove-Item -Path $scriptPath -Force -ErrorAction SilentlyContinue
    Remove-Item -Path $exitCodeFile -Force -ErrorAction SilentlyContinue
    if (Test-Path $publicExePath) {
        Remove-Item -Path $publicExePath -Force -ErrorAction SilentlyContinue
    }
}

Exit $exitCode

