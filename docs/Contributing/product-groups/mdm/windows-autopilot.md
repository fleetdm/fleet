# Windows Autopilot

## Reference links
- [Windows MDM Setup](https://fleetdm.com/guides/windows-mdm-setup#windows-autopilot)
- [Autopilot add devices](https://learn.microsoft.com/en-us/autopilot/add-devices)
- [Assigning Intune licenses](https://learn.microsoft.com/en-gb/intune/intune-service/fundamentals/licenses-assign)
- [Serve locally built Fleetd during Autopilot](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/getting-started/testing-and-local-development.md#building-and-serving-your-own-fleetd-basemsi-installer-for-windows)

## Assigning an Intune license to your user
To use Autopilot, your user needs to have an Intune license assigned. If you don't already have one assigned, follow these steps:
1. Go to [Microsoft 365 Admin Center Licenses](https://admin.cloud.microsoft/?#/licenses)
2. Select Microsoft Intune Plan 1
    1. Worth checking if any of the existing licenses can be unassigned (maybe from other developers)
3. Click "Assign licenses"
4. Select your user and click "Assign"
    1. If it says no license is available, you are good to buy a license, which will be charged on Noah Talerman's (As of 24th February 2026) brex card.


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

> **Important:** When uploading the hardware hash CSV, include the **group tag** that matches your dynamic security group query (e.g., `NameDev`). If you forget, you can edit the device in the [Autopilot devices list](https://intune.microsoft.com/#view/Microsoft_Intune_Enrollment/AutopilotDevices.ReactView/filterOnManualRemediationRequired~/false) and add it later.

#### If using a VM
If using a VM, make sure the VM is assigned a serial number. This is different on how to do for each VM provider, but for example on UTM, you can edit an instance, go to "Arguments" and add the following: `-smbios type=1,serial=<SERIAL_NUMBER>`, where <SERIAL_NUMBER> is a custom unique identifier.

Once added, you should see the device with it's serial show up in [the Autopilot devices list](https://intune.microsoft.com/#view/Microsoft_Intune_Enrollment/AutopilotDevices.ReactView/filterOnManualRemediationRequired~/false), it is ready to be enrolled, once the "Profile status" is "Assigned" (which may take some minutes).

## Setting up a custom domain with ngrok

Microsoft Entra requires a **verified custom domain** for the MDM application URIs. You cannot use a raw `*.ngrok.io` URL — Entra will reject it during domain verification.

1. **Register a domain** (e.g., a cheap `.xyz` domain from Namecheap). You don't need to purchase SSL — ngrok handles TLS termination.
2. **Add the domain in ngrok's dashboard** (Domains section). ngrok will provide a CNAME target (e.g., `xxx.ngrok-dns.com`).
3. **Configure DNS in your domain registrar:**
   - Add a **CNAME record** pointing your domain to the ngrok CNAME target.
   - Add the **TXT record** that Microsoft Entra provides for domain verification.
4. **Verify the domain in Entra:** go to [Entra > Domain names](https://entra.microsoft.com/#view/Microsoft_AAD_IAM/ActiveDirectoryMenuBlade/~/Domains) > Add custom domain, enter your domain, and verify it using the TXT record.
5. **Configure the MDM application in Entra** following the [Windows MDM Setup guide](https://fleetdm.com/guides/windows-mdm-setup#step-2-connect-fleet-to-microsoft-entra-id). Use your custom domain for all MDM URLs (Application ID URI, discovery URL, terms of use URL).

Example ngrok config with a custom domain for the Fleet server:
```yaml
version: "3"
agent:
    authtoken: <your_ngrok_authtoken>
tunnels:
    fleet:
        proto: http
        schemes: [https]
        hostname: yourdomain.xyz  # your verified custom domain
        addr: https://localhost:8080
        inspect: true
    installers:
        proto: http
        schemes: [https]
        hostname: installers.your-ngrok-subdomain.ngrok.io
        addr: http://localhost:8085
        inspect: true
    tuf:
        proto: http
        schemes: [http]
        hostname: tuf.your-ngrok-subdomain.ngrok.io
        addr: http://localhost:8081
        inspect: true
```

Only the Fleet server tunnel needs the custom domain. The installer and TUF tunnels can use regular ngrok subdomains.