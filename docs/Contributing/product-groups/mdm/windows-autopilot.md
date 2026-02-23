# Windows Autopilot

## Reference links
- [Windows MDM Setup](https://fleetdm.com/guides/windows-mdm-setup#windows-autopilot)
- [Autopilot add devices](https://learn.microsoft.com/en-us/autopilot/add-devices)
- [Serve locally built Fleetd during Autopilot](https://github.com/fleetdm/fleet/blob/docs-windows-autopilot-dev/docs/Contributing/getting-started/testing-and-local-development.md#building-and-serving-your-own-fleetd-basemsi-installer-for-windows)

## Configuring Windows Autopilot for development
To set up Windows Autopilot for development, follow these steps:
1. Create a [new Intune security group](https://intune.microsoft.com/#view/Microsoft_AAD_IAM/AddGroupBlade)
    1. Name the group
    2. Select "Dynamic Device" as the membership type
    3. Add the following dynamic query, by clicking "Add dynamic query" and "Edit" on the Rule syntax box:
        1. `(device.devicePhysicalIds -any _ -eq "[OrderID]:<YOUR_GROUP_TAG>")`
        2. Replace `<YOUR_GROUP_TAG>` with a unique identifier for your group, such as "NameDev"
2. Create a new [Autopilot deployment profile](https://intune.microsoft.com/#view/Microsoft_Intune_Enrollment/AutopilotDeploymentProfiles.ReactView) with the following settings:
    1. A name, and "Convert all targeted devices to Autopilot" set to "No"
    2. Deployment mode set to "User-driven"
    3. The rest can be the default settings
    4. On the assignments page, click "Add group" and select the security group you created in step 1.

## Adding your device to Autopilot
To add your Windows device (VM's work as well) to Autopilot, you need to get some hardware information, like the serial and other attributes.

Follow the steps [in the autopilot add devices guide](https://learn.microsoft.com/en-us/autopilot/add-devices#directly-upload-the-hardware-hash-to-an-mdm-service), to either get the information into a .csv or upload it directly.


#### If using a VM
If using a VM, make sure the VM is assigned a serial number. This is different on how to do for each VM provider, but for example on UTM, you can edit an instance, go to "Arguments" and add the following: `-smbios type=1,serial=<SERIAL_NUMBER>`, where <SERIAL_NUMBER> is a custom unique identifier.


Once added, you should see the device with it's serial show up in [the Autopilot devices list](https://intune.microsoft.com/#view/Microsoft_Intune_Enrollment/AutopilotDevices.ReactView/filterOnManualRemediationRequired~/false), it is ready to be enrolled, once the "Profile status" is "Assigned".