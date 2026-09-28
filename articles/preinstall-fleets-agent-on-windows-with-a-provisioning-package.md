# Preinstall Fleet's agent on Windows with a provisioning package

When you ship a new Windows laptop straight to an end user, you want it enrolled in Fleet from the first boot, not after someone remembers to run an installer. A Windows provisioning package (`.ppkg`) handles this. You apply it once at the first Windows setup screen, it installs Fleet's agent (fleetd), and you power the device off and ship it. When the end user opens the box and finishes Windows setup, fleetd is already running.

This guide is for devices that don't join Microsoft Entra ID during setup. If yours do, use [automatic enrollment or Windows Autopilot](https://fleetdm.com/guides/windows-mdm-setup#automatic-enrollment) instead, and Fleet installs fleetd for you.

## Prerequisites

- A Fleet server and the enroll secret for the fleet you want new devices to join. Find it on the **Hosts** page: select the fleet, then select **Add hosts**.
- [`fleetctl`](https://fleetdm.com/guides/fleetctl#installing-fleetctl) installed. Building an MSI on macOS or Linux also requires [Docker](https://docs.docker.com/get-docker).
- A Windows 10 or 11 computer with [Windows Configuration Designer](https://www.microsoft.com/store/apps/9nblggh4tx22) installed from the Microsoft Store.
- A USB drive.
- Target devices running Windows 10 or 11 Pro, Enterprise, or Education, still at the first Windows setup screen (region selection). Windows Home can't turn on MDM.

> **Warning:** The fleetd MSI contains your enroll secret, and so does any package built from it. Encrypt the package (covered below), keep the USB drive with IT, and rotate the enroll secret if the package leaves your control.

## Build fleetd

On the **Hosts** page, select the fleet, select **Add hosts**, then select the **Windows** tab. Leave **Type** set to **Workstation** and copy the command. Add `--disable-setup-experience` and `--bypass-end-user-auth` to it, then run it:

```bash
fleetctl package --type=msi --enable-scripts --fleet-desktop --disable-setup-experience --bypass-end-user-auth --fleet-url=https://fleet.example.com --enroll-secret=YOUR_ENROLL_SECRET
```

No web browser is available while Windows applies the package, so fleetd can't open web pages during setup. `--disable-setup-experience` stops fleetd from opening the setup experience page, and `--bypass-end-user-auth` lets it enroll without the end user authentication page.

> **Note:** `--bypass-end-user-auth` works when the Fleet server's [`mdm.allow_orbit_end_user_auth_bypass`](https://fleetdm.com/docs/configuration/fleet-server-configuration#mdm-allow-orbit-end-user-auth-bypass) option is `true`, the default.

`fleetctl` writes `fleet-osquery.msi` to the current directory. Copy it to the Windows computer that runs Windows Configuration Designer, into an otherwise empty folder.

> **Note:** For Arm-based Windows devices, add `--arch=arm64` and build a separate package for them. `fleetctl` names that file `fleet-osquery-arm64.msi`, so use that name in **CommandFile** and **CommandLine** below.

> **Note:** Windows Configuration Designer imports every file in the folder that holds the installer. Extra files in that folder can break the build.

## Create the provisioning package

1. Open **Windows Configuration Designer** and select **Advanced provisioning**.
2. Enter a project name, such as "Fleet agent", and select **Next**.
3. Select **All Windows desktop editions** and select **Next**.
4. On **Import a provisioning package (optional)**, select **Finish**.
5. In **Available customizations**, go to **Runtime settings > ProvisioningCommands > PrimaryContext > Command**.
6. In **Name**, enter `fleetd` and select **Add**.
7. Select the new **fleetd** entry under **Command** and set:
   - **CommandFile**: select **Browse...** and choose `fleet-osquery.msi`.
   - **CommandLine**: `msiexec.exe /i fleet-osquery.msi /qn /norestart`
   - **ContinueInstall**: **TRUE**
   - **RestartRequired**: **FALSE**
   - **ReturnCodeRestart**: `3010`
   - **ReturnCodeSuccess**: `0`
8. Select **File > Save**, then select **OK** on the **Keep your info secure** message.

## Export the package

1. Select **Export > Provisioning package**.
2. Change **Owner** from **OEM** to **IT Admin** and select **Next**.
3. Select **Encrypt package** and enter a password in **Encryption password**. Save it somewhere secure, since you'll type it on every device. Select **Next**.
4. Choose where to save the package and select **Next**. By default, it's saved in the project folder.
5. Select **Build**. When you see **All done!**, select **Finish**.
6. Copy the `.ppkg` file to the root of the USB drive. Put only one package on the drive.

> **Note:** Only the `.ppkg` file is encrypted. The MSI stays in the folder you copied it to, and the project folder records its path. Delete the MSI and the project folder after you build, or store them somewhere secure.

## Apply the package and ship the device

Do this for each new device:

1. Power on the device and stop at the first setup screen. If the device has moved past it, reset the device and start again.
2. To verify enrollment before you ship, connect the device to your network with Ethernet. If you only have Wi-Fi, apply the package first, then continue to the network screen and connect there.
3. Insert the USB drive. If nothing happens, press the Windows key five times.
4. When prompted, enter the package password and confirm that you trust the package.
5. When you see "You can remove your removable media now!", remove the USB drive. Windows finishes applying the package and installs fleetd.
6. If you have access to Fleet, wait 2-3 minutes after the package finishes installing. Then confirm the host has enrolled and completed a refetch (see [Verify](#verify)).
7. Shut down the device without creating a user account. Do either of the following:
   - Press Shift+F10 to open a command prompt (Fn+Shift+F10 on some laptops) and run `shutdown /s /t 0`.
   - Hold the power button until the device turns off. A quick press may put the device to sleep instead.

The device is ready to ship. When the end user powers it on, they finish Windows setup as usual, and fleetd is already installed.

> **Note:** If you can't verify enrollment before you ship, for example because you don't have access to Fleet, the device still enrolls on its own. As soon as the end user connects it to Wi-Fi, fleetd checks in and completes enrollment, and anything you've set to install or configure automatically starts then.

> **Note:** Windows MDM [turns on after an end user signs in](https://fleetdm.com/guides/windows-mdm-setup#manual-enrollment). Until then, the host reports MDM as "Off", and configuration profiles stay queued.

## Verify

If you have access to Fleet, confirm that fleetd is running and the host has enrolled before you shut down each device:

1. On the device, press Shift+F10 at the setup screen and run `sc query "Fleet osquery"`. The service's `STATE` should be `RUNNING`.
2. In Fleet, go to **Hosts**, select the fleet, and search for the device's serial number.
3. Select the host and confirm its details, such as the operating system, have loaded. If the **Refetch** button shows **Fetching fresh vitals...this may take a moment**, wait for it to finish.

A Hyper-V virtual machine is a fast way to repeat these steps while you test, without resetting real hardware.

## Troubleshoot

**"Get an app to open this 'https' link" appears during setup**

fleetd tried to open a web page, and no browser is available yet. Build fleetd again with `--disable-setup-experience` and `--bypass-end-user-auth` (see [Build fleetd](#build-fleetd)), build a new package, and apply it again.

**Nothing happens when you insert the USB drive**

Press the Windows key five times. If Windows still doesn't find the package, check that the `.ppkg` file is at the root of the drive, not in a folder, and that the device is still on the first setup screen.

**The package applies, but the Fleet osquery service doesn't exist**

Test the package on a running Windows computer with the `Install-ProvisioningPackage` PowerShell cmdlet, which writes logs you can read. This installs fleetd on that computer and enrolls it in Fleet, so use a test device:

```powershell
Install-ProvisioningPackage -PackagePath D:\fleet-agent.ppkg -LogsDirectoryPath C:\ppkg-logs
```

The most common cause is a **CommandLine** that doesn't match the MSI's file name. The command runs from the folder where Windows extracts the package, so use the file name alone, not a full path.

**fleetd is running, but the host doesn't appear in Fleet**

Check that the device can reach your Fleet server, and that the MSI was built with the enroll secret for the fleet you expect. From another computer, `fleetctl debug connection https://fleet.example.com` tests the TLS connection fleetd uses. If the fleet requires end user authentication, check that you built fleetd with `--bypass-end-user-auth` and that the server's `mdm.allow_orbit_end_user_auth_bypass` option is `true`.

## Further reading

- [Enroll hosts](https://fleetdm.com/guides/enroll-hosts)
- [Windows MDM setup](https://fleetdm.com/guides/windows-mdm-setup)
- [Windows & Linux setup experience](https://fleetdm.com/guides/windows-linux-setup-experience)
- [Apply a provisioning package](https://learn.microsoft.com/en-us/windows/configuration/provisioning-packages/provisioning-apply-package) (Microsoft)
- [Provision PCs with apps](https://learn.microsoft.com/en-us/windows/configuration/provisioning-packages/provision-pcs-with-apps) (Microsoft)

<meta name="articleTitle" value="Preinstall Fleet's agent on Windows with a provisioning package">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="publishedOn" value="2026-09-23">
<meta name="category" value="guides">
<meta name="description" value="Build a Windows provisioning package that installs Fleet's agent during setup, so devices ship to end users already enrolled in Fleet.">
