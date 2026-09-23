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

On the **Hosts** page, select the fleet, select **Add hosts**, then select the **Windows** tab. Leave **Type** set to **Workstation**, copy the command, and run it. It looks like this:

```bash
fleetctl package --type=msi --enable-scripts --fleet-desktop --fleet-url=https://fleet.example.com --enroll-secret=YOUR_ENROLL_SECRET
```

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
2. Connect the device to your network with Ethernet so fleetd can enroll before you ship. If you only have Wi-Fi, apply the package first, then continue to the network screen and connect there.
3. Insert the USB drive. If nothing happens, press the Windows key five times.
4. When prompted, enter the package password and confirm that you trust the package.
5. When you see "You can remove your removable media now!", remove the USB drive. Windows finishes applying the package and installs fleetd.
6. Wait for the host to appear in Fleet (see [Verify](#verify)).
7. Shut down the device without creating a user account. Do either of the following:
   - Press Shift+F10 to open a command prompt (Fn+Shift+F10 on some laptops) and run `shutdown /s /t 0`.
   - Hold the power button until the device turns off. A quick press may put the device to sleep instead.

The device is ready to ship. When the end user powers it on, they finish Windows setup as usual, and fleetd is already installed.

> **Note:** If the fleet [requires end users to authenticate](https://fleetdm.com/guides/windows-linux-setup-experience#require-idp-authentication) with your identity provider (IdP), fleetd can't enroll until someone signs in to Windows, so the host won't appear in Fleet before you ship. After the end user signs in, fleetd opens a browser to your IdP sign-in page, and the host enrolls once they authenticate.

> **Note:** Windows MDM [turns on after an end user signs in](https://fleetdm.com/guides/windows-mdm-setup#manual-enrollment). Until then, the host reports MDM as "Off", and configuration profiles stay queued.

## Verify

Before you ship the first device, confirm that the package works:

1. On the device, press Shift+F10 at the setup screen and run `sc query "Fleet osquery"`. The service's `STATE` should be `RUNNING`.
2. In Fleet, go to **Hosts**, select the fleet, and search for the device's serial number. The host appears within a few minutes of fleetd starting.

A Hyper-V virtual machine is a fast way to repeat these steps while you test, without resetting real hardware.

## Troubleshoot

**Nothing happens when you insert the USB drive**

Press the Windows key five times. If Windows still doesn't find the package, check that the `.ppkg` file is at the root of the drive, not in a folder, and that the device is still on the first setup screen.

**The package applies, but the Fleet osquery service doesn't exist**

Test the package on a running Windows computer with the `Install-ProvisioningPackage` PowerShell cmdlet, which writes logs you can read. This installs fleetd on that computer and enrolls it in Fleet, so use a test device:

```powershell
Install-ProvisioningPackage -PackagePath D:\fleet-agent.ppkg -LogsDirectoryPath C:\ppkg-logs
```

The most common cause is a **CommandLine** that doesn't match the MSI's file name. The command runs from the folder where Windows extracts the package, so use the file name alone, not a full path.

**fleetd is running, but the host doesn't appear in Fleet**

Check that the device can reach your Fleet server, and that the MSI was built with the enroll secret for the fleet you expect. From another computer, `fleetctl debug connection https://fleet.example.com` tests the TLS connection fleetd uses. If the fleet requires IdP authentication, the host won't enroll until the end user signs in.

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
