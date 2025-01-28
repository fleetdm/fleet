package_name=$PACKAGE_ID

# Fleet uninstalls app using product name that's extracted on upload
dnf remove --assumeyes "$package_name"
