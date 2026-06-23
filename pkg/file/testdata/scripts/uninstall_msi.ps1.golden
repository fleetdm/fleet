$product_code = $PACKAGE_ID
$timeoutSeconds = 300  # 5 minute timeout

# Fleet uninstalls app using product code that's extracted on upload
$process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $product_code, "/norestart") -PassThru

# Wait for process with timeout
$completed = $process.WaitForExit($timeoutSeconds * 1000)

if (-not $completed) {
    Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
    Exit 1603  # ERROR_UNINSTALL_FAILURE
}

# MSI exit codes that indicate success. 3010 = ERROR_SUCCESS_REBOOT_REQUIRED,
# 1641 = ERROR_SUCCESS_REBOOT_INITIATED. Treat these as success rather than failure.
$successCodes = @(0, 3010, 1641)

# Check exit code and output result
if ($successCodes -contains $process.ExitCode) {
    Write-Output "Exit 0"
    Exit 0
} else {
    Write-Output "Exit $($process.ExitCode)"
    Exit $process.ExitCode
}
