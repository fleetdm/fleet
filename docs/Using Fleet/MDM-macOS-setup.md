# macOS setup

_Available in Fleet Premium_

In Fleet, you can customize the out-of-the-box macOS setup experience for your end users:

* Require end users to authenticate with your identity provider (IdP) and agree to an end user license agreement (EULA) before they can use their new Mac

* Customize the macOS Setup Assistant by choosing to show or hide specific panes

* Install a bootstrap package to gain full control over the setup experience by installing tools like Puppet, Munki, DEP notify, custom scrips, and more

In addition to the customization above, Fleet automatically installs the fleetd agent during out-of-the-box macOS setup. This agent is responsible for reporting host vitals to Fleet and presenting Fleet Desktop to the end user.

MacOS setup features require connecting Fleet to Apple Business Manager (ABM). Learn how [here](./MDM-setup.md#apple-business-manager-abm).

## End user authentication and EULA

Using Fleet, you can require end users to authenticate with your identity provider (IdP) and agree to an end user license agreement (EULA) before they can use their new Mac.

To require end user authentication:

1. Connect Fleet to your IdP
2. Upload a EULA to Fleet (optional)
3. Enable end user authentication

### Step 1: Connect Fleet to Your IdP

Fleet UI:

1. Head to the **Settings > Integrations > Automatic enrollment** page

2. Under **End user authentication**, enter your IdP credentials and select **Save**

fleetctl CLI:

1. Create `fleet-config.yaml` file or add to your existing `config` YAML file:

```yaml
apiVersion: v1
kind: config
spec:
  mdm:
    end_user_authentication:
      identity_provider_name: "Okta"
      entity_id: 123
      issuer_url: "https://example.com"
      metadata_url: "https://example.com"
  ...
```

2. Fill in the relevant information from your IdP under the `mdm.end_user_authentication` key 

3. Run the fleetctl `apply -f fleet-config.yml` command to add your IdP credentials

4. Confirm that your IdP credentials were saved by running `fleetctl get config`

### Step 2: Upload a EULA to Fleet

1. Head to the **Settings > Integrations > Automatic enrollment** page

2. Under **End user license agreement (EULA)**, select **Upload** and choose your EULA

> Uploading a EULA is optional. If you don't upload a EULA, the end user will skip this step and continue to the next step of the new Mac setup experience after authenticating with your IdP.

### Step 3: Enable End User Authentication

You can enable end user authentication using the Fleet UI or fleetctl command-line tool.

Fleet UI:

1. Head to the **Controls > macOS settings > macOS setup > End user authentication** page

2. Choose which team you want to enable end user authentication for by selecting the desired team in the teams dropdown in the upper left corner

3. Select the **On** checkbox and select **Save**

fleetctl CLI: 

**Example of enabling end user authentication on the "Workstations (canary)" team so that the authentication is only required for hosts that automatically enroll to this team.**

1. Choose which team you want to enable end user authentication on

2. Create a `workstations-canary-config.yaml` file:

```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: Workstations (canary)
    mdm:
      macos_setup:
        enable_end_user_authentication: true
    ...
```

Learn more about team configurations options [here](./configuration-files/README.md#teams).

To enable authentication for automatically enrolled "No team" hosts, a `fleet-config.yaml` file must be created:

```yaml
apiVersion: v1
kind: config
spec:
  mdm:
    macos_setup:
      enable_end_user_authentication: true
  ...
```

Learn more about "No team" configuration options [here](./configuration-files/README.md#organization-settings).

3. Add an `mdm.macos_setup.enable_end_user_authentication` key to your YAML document to accept a boolean value

4. Run the `fleetctl apply -f workstations-canary-config.yml` command to enable authentication for this team

5. Confirm that end user authentication is enabled by running the `fleetctl get teams --name=Workstations --yaml` command

If you enabled authentication on "No team," run `fleetctl get config`.

You should see a `true` value for `mdm.macos_setup.enable_end_user_authentication`.

## Bootstrap Package

Fleet supports installing a bootstrap package on macOS hosts that automatically enroll to Fleet. 

This enables installing tools like [Puppet](https://www.puppet.com/), [Munki](https://www.munki.org/munki/), or [Chef](https://www.chef.io/products/chef-infra) for configuration management and/or running custom scrips and installing tools like [DEP notify](https://gitlab.com/Mactroll/DEPNotify) to customize the setup experience for your end users.

Example uses of bootstrap package:

* Munki client to install and keep software up to date on your Macs

* Puppet agent to run custom scripts on your Macs

Clients can use [InstallApplications](https://github.com/macadmins/installapplications), a tool for bundling scripts and packages into a single bootstrap package, to install a base set of applications, set the Mac's background, and install the latest macOS update for the end user

To add a bootstrap package to Fleet:

1. Download or generate a package
2. Sign the package
3. Upload the package to Fleet
4. Confirm package is uploaded

### Step 1: Download or Generate a Package

Whether you have to download or generate a package depends on what you want to deploy using your bootstrap package:

* Single clients and agents, including Munki and Puppet, are downloaded from that tool's GitHub repository or website: for example [Munki](https://github.com/munki/munki/releases). 

* Custom scripts require a package to be generated with a tool like [munkipkg tool](https://github.com/munki/munki-pkg)

Apple requires that your package is a distribution package. 

To verify that the package is a distribution package:
1. Run the following commands to expand you package and look at the files in the expanded folder:

```bash
$ pkgutil --expand package.pkg expanded-package
$ ls expanded-package
```

If your package is a distribution package should see a `Distribution` file.

2. If you don't see a `Distribution` file, run the following command to convert your package into a distribution package

```bash
$ productbuild --package package.pkg distrbution-package.pkg
```

Make sure your package is a `.pkg` file.

### Step 2: Sign the Package

To set up the Developer ID Installer certificate, required to sign the package:

1. Login to your [Apple Developer account](https://developer.apple.com/account)
2. Follow Apple's instructions to create a Developer ID Installer certificate [here](https://developer.apple.com/help/account/create-certificates/create-developer-id-certificates)

> During step 3 in Apple's instructions, make sure to choose "Developer ID Installer," the required certificate to sign the package

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

### Step 3: Upload the Package to Fleet

Fleet UI:

1. Head to the **Controls > macOS settings > macOS setup > Bootstrap package** page

2. In the teams dropdown menu, located in the upper left corner, select the desired team for the bootstrap package 

3. Select **Upload** and choose your bootstrap package

fleetctl CLI:

1. Upload the package to a storage location (ex. S3 or GitHub), where Fleet will retrieve and host the package for deployment

> The URL must be accessible by the computer that uploads the package to Fleet.
> * This could be your local computer or the computer that runs your CI/CD workflow.

2. Choose which team you want to add the bootstrap package to

**Example of adding a bootstrap package to the "Workstations (canary)" team to ensure that the package is only installed on hosts that automatically enroll to this team:**

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

Installing the package to automatically enrolled "No team" hosts requires a `fleet-config.yaml` file:

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

3. Add an `mdm.macos_setup.bootstrap_package` key to your YAML document to accept the URL for the storage location of the bootstrap package

4. Run the fleetctl `apply -f workstations-canary-config.yml` command to upload your bootstrap package to Fleet

5. Confirm that your bootstrap package was uploaded to Fleet by running the `fleetctl get teams --name=Workstations --yaml` command

If you uploaded the package to "No team," run `fleetctl get config`.

You should see the URL for your bootstrap package as the value for `mdm.macos_setup.bootstrap_package`.

## macOS Setup Assistant

Starting a new or freshly wiped Mac, end users are presented with the macOS Setup Assistant, which allows for the configuration of accessibility, appearance, and more.

In Fleet, you can customize the macOS Setup Assistant by using an automatic enrollment profile.

To customize the macOS Setup Assistant:

1. Create an automatic enrollment profile
2. Upload the profile to Fleet
3. Test the custom macOS Setup Assistant

### Step 1: Create an Automatic Enrollment Profile

1. Download Fleet's example automatic enrollment profile by navigating to the example [here on GitHub](https://github.com/fleetdm/fleet/blob/main/mdm_profiles/setup_assistant.json) and clicking the download icon

2. Open the automatic enrollment profile and replace the `profile_name` key with your organization's name

3. View the the list of macOS Setup Assistant properties (panes) [here in Apple's Device Management documentation](https://developer.apple.com/documentation/devicemanagement/skipkeys) and choose which panes to hide from your end users

4. In your automatic enrollment profile, edit the `skip_setup_items` array to include the panes you want to hide

> You can modify properties other than `skip_setup_items`. These are documented by Apple [here](https://developer.apple.com/documentation/devicemanagement/profile).

### Step 2: Upload the Profile to Fleet

1. Choose which team you want to add the automatic enrollment profile to

**Example of testing your profile before using it in production, using "Workstations" as your [default team](./MDM-setup.md#step-6-optional-set-the-default-team-for-hosts-enrolled-via-abm).** 

To create a new "Workstations (canary)" team and add the automatic enrollment profile to it. 
Note: Only hosts that automatically enroll to this team will see the custom macOS Setup Assistant.

2. Create a `workstations-canary-config.yaml` file:

```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: Workstations (canary)
    mdm:
      macos_setup:
        macos_setup_assistant: ./path/to/automatic_enrollment_profile.json
    ...
```

Learn more about team configurations options [here](./configuration-files/README.md#teams).

To customize the macOS Setup Assistant for hosts that automatically enroll to "No team," create a `fleet-config.yaml` file:

```yaml
apiVersion: v1
kind: config
spec:
  mdm:
    macos_setup:
      macos_setup_assistant: ./path/to/automatic_enrollment_profile.json
  ...
```

Learn more about configuration options for hosts that aren't assigned to a team [here](./configuration-files/README.md#organization-settings).

3. Add an `mdm.macos_setup.macos_setup_assistant` key to your YAML document. This key accepts a path to your automatic enrollment profile

4. Run the `fleetctl apply -f workstations-canary-config.yml` command to upload the automatic enrollment profile to Fleet

### Step 3: Test the Custom MacOS Setup Assistant

Testing requires a test Mac that is present in your Apple Business Manager (ABM) account. Wipe this Mac and use it to test the custom macOS Setup Assistant.

1. Wipe the test Mac by selecting the Apple icon in top left corner of the screen, selecting **System Settings** or **System Preference**, and searching for "Erase all content and settings"

2. Select **Erase All Content and Settings**

3. In Fleet, navigate to the Hosts page, find your Mac, and ensure that the host's **MDM status** is set to "Pending"

> New Macs purchased through Apple Business Manager appear in Fleet with MDM status set to "Pending." Learn more about these hosts [here](./MDM-setup.md#pending-hosts).

3. Transfer this host to the "Workstations (canary)" team by selecting the checkbox to the left of the host and selecting **Transfer** at the top of the table

4. In the modal, choose the Workstations (canary) team and select **Transfer**

5. Boot up your test Mac and complete the custom out-of-the-box setup experience

<meta name="pageOrderInSection" value="1505">
<meta name="title" value="MDM macOS setup">
<meta name="description" value="Customize your macOS setup experience with Fleet Premium by managing user authentication, Setup Assistant panes, and installing bootstrap packages.">
<meta name="navSection" value="Device management">
