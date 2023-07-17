# macOS setup

_Available in Fleet Premium_

In Fleet, you can customize the out-of-the-box macOS setup experience for your end users:

* Require end users to authenticate with your identity provider (IdP) and agree to an end user license agreement (EULA) before they can use their new Mac

* Customize the macOS Setup Assistant by choosing to show or hide specific panes

* Install a bootstrap package to gain full control over the setup experience by installing tools like Puppet, Munki, DEP notify, custom scrips, and more.

In addition to the customization above, Fleet automatically installs the fleetd agent during out-of-the-box macOS setup. This agent is responsible for reporting host vitals to Fleet and presenting Fleet Desktop to the end user.

MacOS setup features require connecting Fleet to Apple Business Manager (ABM). Learn how [here](./MDM-setup.md#apple-business-manager-abm).

## End user authentication

> This feature is currently in development.

## Bootstrap package

Fleet supports installing a bootstrap package on macOS hosts that automatically enroll to Fleet. 

This enables installing tools like [Puppet](https://www.puppet.com/), [Munki](https://www.munki.org/munki/), or [Chef](https://www.chef.io/products/chef-infra) for configuration management and/or running custom scrips and installing tools like [DEP notify](https://gitlab.com/Mactroll/DEPNotify) to customize the setup experience for you end users.

The following are examples of what some organizations deploy using a bootstrap package:

* Munki client to install and keep software up to date on your Macs

* Puppet agent to run custom scripts on your Macs

* Custom scripts and several packages bundled into one bootstrap package using a tool like [InstallApplications](https://github.com/macadmins/installapplications) to install a base set of applications, set the Mac's background, and install the latest macOS update for the end user.

To add a bootstrap package to Fleet, we will do the following steps:

1. Download or generate a package
2. Sign the package
3. Upload the package to Fleet
4. Confirm package is uploaded

### Step 1: download or generate a package

Whether you have to download or generate a package depends on what you want to deploy using your bootstrap package:

* A single client or agent, like Munki or Puppet, can usually be downloaded from the tool's GitHub repository or website. For example, you can download Munki, the Munki client on their [releases page on GitHub](https://github.com/munki/munki/releases). 

* To deploy custom scripts, you need to generate a package. The [munkipkg tool](https://github.com/munki/munki-pkg) is a popular tool for generating packages.

Apple requires that your package is a distribution package. Verify that the package is a distribution package:

1. Run the following commands to expand you package and look at the files in the expanded folder:

```bash
$ pkgutil --expand package.pkg expanded-package
$ ls expanded-package
```

If your package is a distribution package should see a `Distribution` file.

2. If you don't see a `Distribution` file, run the following command to convert your package into a distribution package.

```bash
$ productbuild --package package.pkg distrbution-package.pkg
```

Make sure your package is a `.pkg` file.

### Step 2: sign the package

To sign the package we need a valid Developer ID Installer certificate:

1. Login to your [Apple Developer account](https://developer.apple.com/account).
2. Follow Apple's instructions to create a Developer ID Installer certificate [here](https://developer.apple.com/help/account/create-certificates/create-developer-id-certificates).

> During step 3 in Apple's instructions, make sure you choose "Developer ID Installer." You'll need this kind of certificate to sign the package.

Confirm that certificate is installed on your Mac by opening the **Keychain Access** application. You should see your certificate in the **Certificates** tab.

3. Run the following command in the **Terminal** application to sign your package with your Developer ID certificate:

```bash
$ productsign --sign "Developer ID Installer: Your name (Serial number)" /path/to/package.pkg /path/to/signed-package.pkg
```

You might be prompted to enter the password for your local account.

Confirm that your package is signed by running the following command:

```bash
$ pkgutil --check-signature /path/to/signed-package.pkg
```

In the output you should see that package has a "signed" status.

### Step 3: upload the package to Fleet

1. Upload the package to a storage location (ex. S3 or GitHub). During step 4, Fleet will retrieve the package from this storage location and host it for deloyment.

> The URL must be accessible by the computer that uploads the package to Fleet.
> * This could be your local computer or the computer that runs your CI/CD workflow.

2. Choose which team you want to add the bootstrap package to.

In this example, we'll add a bootstrap package to the "Workstations (canary)" team so that the package only gets installed on hosts that automatically enroll to this team.

3. Create a `workstations-canary-config.yaml` file:

```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: Workstations (canary)
    mdm:
      macos_setup:
        bootstrap_package: https://github.com/organinzation/repository/bootstrap-package.pkg
    ...
```

Learn more about team configurations options [here](./configuration-files/README.md#teams).

If you want to install the package on hosts that automatically enroll to "No team," we'll need to create an `fleet-config.yaml` file:

```yaml
apiVersion: v1
kind: config
spec:
  mdm:
    macos_setup:
      bootstrap_package: https://github.com/organinzation/repository/bootstrap-package.pkg
  ...
```

Learn more about "No team" configuration options [here](./configuration-files/README.md#organization-settings).

3. Add an `mdm.macos_setup.bootstrap_package` key to your YAML document. This key accepts the URL for the storage location of the bootstrap package. 

4. Run the fleetctl `apply -f workstations-canary-config.yml` command to upload your bootstrap package to Fleet.

### Step 4: confirm package is uploaded

Confirm that your bootstrap package was uploaded to Fleet:

If you uploaded the package to a team, run `fleetctl get teams --name=Workstations --yaml`.

If you uploaded the package to "No team," run `fleetctl get config`.

You should see the URL for your bootstrap package as the value for `mdm.macos_setup.bootstrap_package`. 

## macOS Setup Assistant

> This feature is currently in development.

<meta name="pageOrderInSection" value="1505">
<meta name="title" value="MDM macOS setup">
<meta name="description" value="Customize your macOS setup experience with Fleet Premium by managing user authentication, Setup Assistant panes, and installing bootstrap packages.">
<meta name="navSection" value="Device management">