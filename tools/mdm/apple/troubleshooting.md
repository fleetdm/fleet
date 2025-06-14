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

## Checking and redelivering MDM enrollment profile

To check if a host has the correct MDM enrollment profile installed, run the following command on the host:
```bash
sudo profiles show -type configuration
```

To trigger a redelivery of enrollment profile, run the following command on the host:
```
sudo profiles renew -type enrollment
```

If the host does not have the right enrollment profile, try transferring the host to another team, wait for 10 minutes, then transfer it back and wait another 10 minutes.