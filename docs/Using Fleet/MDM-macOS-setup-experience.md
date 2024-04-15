# macOS setup experience

_Available in Fleet Premium_

In Fleet, you can customize the out-of-the-box macOS setup experience for your end users:

* Require end users to authenticate with your identity provider (IdP) and agree to an end user license agreement (EULA) before they can use their new Mac.

* Customize the macOS Setup Assistant by choosing to show or hide specific panes.

* Install a bootstrap package to gain full control over the setup experience by installing tools like Puppet, Munki, DEP notify, custom scripts, and more.

In addition to the customization above, Fleet automatically installs the fleetd agent during out-of-the-box macOS setup. This agent is responsible for reporting host vitals to Fleet and presenting Fleet Desktop to the end user.

MacOS setup features require connecting Fleet to Apple Business Manager (ABM). Learn how [here](./mdm-macos-setup.md#apple-business-manager-abm).

## End user authentication and EULA

Using Fleet, you can require end users to authenticate with your identity provider (IdP) and agree to an end user license agreement (EULA) before they can use their new Mac.

### End user authentication

To require end user authentication, first [configure single sign-on (SSO)](../Deploy/single-sign-on-sso.md). Next, enable end user authentication by heading to to **Controls > Setup experience End user authentication** or use [Fleet's GitOps workflow](https://github.com/fleetdm/fleet-gitops).

If you've already configured SSO in Fleet, create a new SAML app in your IdP. In your new app, use `https://<your_fleet_url>/api/v1/fleet/mdm/sso/callback` for the SSO URL.

In your IdP, make sure your end users' full names are set to one of the following attributes (depends on IdP): `name`, `displayname`, `cn`, `urn:oid:2.5.4.3`, or `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name`. Fleet will automatically populate and lock the macOS local account **Full Name** with any of these.

In your IdP, set **Name ID** to email. Fleet will trim this email and use it to populate and lock the macOS local account **Account Name**. For example, a "johndoe@example.com" email turn into a "johndoe" account name.

### EULA

To require a EULA, in Fleet, head to **Settings > Integrations > Automatic enrollment > End user license agreement (EULA)** or use the [Fleet API](https://fleetdm.com/docs/rest-api/rest-api#upload-an-eula-file).

## Bootstrap package

Fleet supports installing a bootstrap package on macOS hosts that automatically enroll to Fleet.

This enables installing tools like [Puppet](https://www.puppet.com/), [Munki](https://www.munki.org/munki/), or [Chef](https://www.chef.io/products/chef-infra) for configuration management and/or running custom scripts and installing tools like [DEP notify](https://gitlab.com/Mactroll/DEPNotify) to customize the setup experience for your end users.

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

1. Run the following commands to expand your package and look at the files in the expanded folder:

  ```bash
  $ pkgutil --expand package.pkg expanded-package
  $ ls expanded-package
  ```

  If your package is a distribution package you should see a `Distribution` file.

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

  In the output you should see that your package has a "signed" status.

### Step 3: upload the package to Fleet

Fleet UI:

1. Head to the **Controls > macOS settings > macOS setup > Bootstrap package** page.

2. Choose which team you want to add the bootstrap package to by selecting the desired team in the teams dropdown in the upper left corner.

3. Select **Upload** and choose your bootstrap package.

fleetctl CLI:

1. Upload the package to a storage location (ex. S3 or GitHub). During step 4, Fleet will retrieve the package from this storage location and host it for deployment.

  > The URL must be accessible by the computer that uploads the package to Fleet.
  > This could be your local computer or the computer that runs your CI/CD workflow.

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

  If you want to install the package on hosts that automatically enroll to "No team," we'll need to create a `fleet-config.yaml` file:

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

5. Confirm that your bootstrap package was uploaded to Fleet by running the `fleetctl get teams --name=Workstations --yaml` command.

  If you uploaded the package to "No team," run `fleetctl get config`.

  You should see the URL for your bootstrap package as the value for `mdm.macos_setup.bootstrap_package`.

## macOS Setup Assistant

When an end user unboxes their new Mac, or starts up a freshly wiped Mac, they're presented with the macOS Setup Assistant. Here they see panes that allow them to configure accessibility, appearance, and more.

In Fleet, you can customize the macOS Setup Assistant by using an automatic enrollment profile.

To customize the macOS Setup Assistant, we will do the following steps:

1. Create an automatic enrollment profile
2. Upload the profile to Fleet
3. Test the custom macOS Setup Assistant

### Step 1: create an automatic enrollment profile

1. Download Fleet's example automatic enrollment profile by navigating to the example [here](fleetdm.com/example-dep-profile) and clicking the download icon.

2. Open the automatic enrollment profile and replace the `profile_name` key with your organization's name.

3. View the the list of macOS Setup Assistant properties (panes) [here in Apple's Device Management documentation](https://developer.apple.com/documentation/devicemanagement/skipkeys) and choose which panes to hide from your end users.

4. In your automatic enrollment profile, edit the `skip_setup_items` array so that it includes the panes you want to hide.

  > You can modify properties other than `skip_setup_items`. These are documented by Apple [here](https://developer.apple.com/documentation/devicemanagement/profile).

### Step 2: upload the profile to Fleet

1. Choose which team you want to add the automatic enrollment profile to.

  In this example, let's assume you have a "Workstations" team as your [default team](./mdm-macos-setup.md#step-6-optional-set-the-default-team-for-hosts-enrolled-via-abm) in Fleet and you want to test your profile before it's used in production.

  To do this, we'll create a new "Workstations (canary)" team and add the automatic enrollment profile to it. Only hosts that automatically enroll to this team will see the custom macOS Setup Assistant.

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

  If you want to customize the macOS Setup Assistant for hosts that automatically enroll to "No team," we'll need to create a `fleet-config.yaml` file:

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

3. Add an `mdm.macos_setup.macos_setup_assistant` key to your YAML document. This key accepts a path to your automatic enrollment profile.

4. Run the `fleetctl apply -f workstations-canary-config.yml` command to upload the automatic enrollment profile to Fleet.

### Step 3: test the custom macOS Setup Assistant

Testing requires a test Mac that is present in your Apple Business Manager (ABM) account. We will wipe this Mac and use it to test the custom macOS Setup Assistant.

1. Wipe the test Mac by selecting the Apple icon in top left corner of the screen, selecting **System Settings** or **System Preference**, and searching for "Erase all content and settings." Select **Erase All Content and Settings**.

2. In Fleet, navigate to the Hosts page and find your Mac. Make sure that the host's **MDM status** is set to "Pending."

  > New Macs purchased through Apple Business Manager appear in Fleet with MDM status set to "Pending." Learn more about these hosts [here](./mdm-macos-setup.md#pending-hosts).

3. Transfer this host to the "Workstations (canary)" team by selecting the checkbox to the left of the host and selecting **Transfer** at the top of the table. In the modal, choose the Workstations (canary) team and select **Transfer**.

4. Boot up your test Mac and complete the custom out-of-the-box setup experience.

<meta name="pageOrderInSection" value="1506">
<meta name="title" value="macOS setup experience">
<meta name="description" value="Customize your macOS setup experience with Fleet Premium by managing user authentication, Setup Assistant panes, and installing bootstrap packages.">
<meta name="navSection" value="Device management">
