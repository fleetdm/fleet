$product_code = $PACKAGE_ID

# Fleet uninstalls app using product code that's extracted on upload
msiexec /quiet /x $product_code
Exit $LASTEXITCODE
