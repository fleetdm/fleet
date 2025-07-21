# Fleet uninstalls app by finding all related product codes for the specified upgrade code
$inst = New-Object -ComObject "WindowsInstaller.Installer"
foreach ($product_code in $inst.RelatedProducts("$UPGRADE_CODE")) {
    msiexec /quiet /x $product_code
}

Exit $LASTEXITCODE
