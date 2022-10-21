# Troubleshooting

## Enable MDM debug logging on the device

1. Install the `./tools/mdm/apple/turn_on_debug_mdm_logging.mobileconfig` profile on the device manually (by double-clicking such file) or using Fleet MDM via:
```sh
fleetctl apple-mdm enqueue-command InstallProfile --device-ids=<TARGET_DEVICE_ID> --mobileconfig ./tools/mdm/apple/turn_on_debug_mdm_logging.mobileconfig
```

2. Check the profile was successfully installed in "System Preferences" -> "Profiles".

3. Then on the device run the following command:
```sh
log stream --info --debug --predicate 'processImagePath contains "mdmclient"' | tee mdm_logs.txt
```