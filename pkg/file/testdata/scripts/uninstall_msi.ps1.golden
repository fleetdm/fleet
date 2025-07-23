$product_code = $PACKAGE_ID

# Fleet uninstalls app using product code that's extracted on upload
$process = Start-Process msiexec -ArgumentList @("/quiet", "/x", $product_code, "/norestart") -Wait -PassThru

# Check exit code and output result
if ($process.ExitCode -eq 0) {
    Write-Output "Exit 0"
    Exit 0
} else {
    Write-Output "Exit $($process.ExitCode)"
    Exit $process.ExitCode
}
