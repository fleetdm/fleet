# How to install osquery and enroll Windows devices into Fleet

Here, we will cover enrolling Windows devices to an existing Fleet server. If you don’t have a Fleet
server configured already, check out [Deploying Fleet on Render](https://fleetdm.com/deploy/deploying-fleet-on-render). If you’re still getting to know Fleet and would like to skip the server setup process and try Fleet, you can [run Fleet in a preview environment](https://fleetdm.com/try-fleet/register).

## Prerequisites

1. Access to an existing Fleet server
2. The `fleetctl` command line tool

If you don’t already have it, `fleetctl` can be installed via npm by running the following command:

```
npm i -g fleetctl
```

After the above command has run successfully, you can confirm that you now have the `fleetctl` CLI tool by running:

```
fleetctl --version
```

The above command should return an output similar to the example below:

```
fleetctl.exe - version 4.8.0
  branch:  HEAD
  revision:  09654d77eedbf9ed181bc8188a3d2be0324b29a5
  build date:  2021-12-31
  build user:  runner
  go version:  go1.17.2
```

> You can generate an osquery installer using `fleetctl` for Windows on macOS and even Linux distributions, but for this article we are assuming generating on a Windows device. To generate an osquery installer for a different OS, check out the guides for [macOS](https://fleetdm.com//guides/how-to-install-osquery-and-enroll-macos-devices-into-fleet) and [Linux](https://fleetdm.com//guides/how-to-install-osquery-and-enroll-linux-devices-into-fleet).

## Installing osquery

Head over to the Hosts page on Fleet and click on the “Add hosts” button, which will present a pop-up that allows you to choose the type of installer you want to generate. Make sure you are on the “Windows” tab and click on the clipboard icon.

![Generate installer](../website/assets/images/articles/install-osquery-and-enroll-windows-devices-into-fleet-1-700x365@2x.png)
*Windows osquery Installer command on Fleet UI*

Next, head over to your Windows command prompt (making sure that you are running with administrator privilege and Docker is running), paste the copied command, and then hit enter.

Once `fleetctl` has finished creating your osquery installer, it will produce an installer file called `fleet-osquery.msi` in your current directory and display instructions on how to proceed.

## Running the installer

Double-click the installer and follow the guided steps to successfully install osquery on your Windows device and enroll it into Fleet!

## Deploying at scale?
If you’re managing an enterprise environment, you will likely have a deployment tool like [Munki](https://www.munki.org/munki/), [Jamf Pro](https://www.jamf.com/products/jamf-pro/), [Chef](https://www.chef.io/), [Ansible](https://www.ansible.com/), or [Puppet](https://puppet.com/) to deliver software to your devices. You can distribute your osquery installer and add all your devices to Fleet using your software management tool of choice.

<meta name="category" value="guides">
<meta name="authorFullName" value="Kelvin Omereshone">
<meta name="authorGitHubUsername" value="dominuskelvin">
<meta name="publishedOn" value="2022-02-03">
<meta name="articleTitle" value="How to install osquery and enroll Windows devices into Fleet">
<meta name="articleImageUrl" value="../website/assets/images/articles/install-osquery-and-enroll-windows-devices-into-fleet-cover-1600x900@2x.jpg">
