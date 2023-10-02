# macOS setup

_Available in Fleet Premium_

In Fleet, you can customize the out-of-the-box macOS setup experience for your end users:

* Require end users to authenticate with your identity provider (IdP) and agree to an end user license agreement (EULA) before they can use their new Mac.

* Customize the macOS Setup Assistant by choosing to show or hide specific panes.

* Install a bootstrap package to gain full control over the setup experience by installing tools like Puppet, Munki, DEP notify, custom scripts, and more.

* Populate the Full Name and Account Name during local account creation with the end user's IdP attributes.

* Require end users to wait for configuration profiles before they can use their new Mac.

In addition to the customization above, Fleet automatically installs the fleetd agent during out-of-the-box macOS setup. This agent is responsible for reporting host vitals to Fleet and presenting Fleet Desktop to the end user.

MacOS setup features require connecting Fleet to Apple Business Manager (ABM). Learn how [here](./MDM-setup.md#apple-business-manager-abm).

## End user authentication and EULA

Using Fleet, you can require end users to authenticate with your identity provider (IdP) and agree to an end user license agreement (EULA) before they can use their new Mac.

To require end user authentication, we will do the following steps:

1. Connect Fleet to your IdP
2. Upload a EULA to Fleet (optional)
3. Enable end user authentication

### Step 1: connect Fleet to your IdP

Fleet UI:

1. Head to the **Settings > Integrations > Automatic enrollment** page.

2. Under **End user authentication**, enter your IdP credentials and select **Save**.

> If you've already configured [single sign-on (SSO) for logging in to Fleet](../Deploy/single-sign-on-sso.md#fleet-sso-configuration), you'll need to create a separate app in your IdP so your end users can't log in to Fleet. In this separate app, use "https://fleetserver.com/api/v1/fleet/mdm/sso/callback" for the SSO URL.

fleetctl CLI:

1. Create `fleet-config.yaml` file or add to your existing `config` YAML file:

    ```yaml
    apiVersion: v1
    kind: config
    spec:
      mdm:
        end_user_authentication:
          identity_provider_name: "Okta"
          entity_id: "https://fleetserver.com"
          issuer_url: "https://okta-instance.okta.com/84598y345hjdsshsfg/sso/saml/metadata"
          metadata_url: "https://okta-instance.okta.com/84598y345hjdsshsfg/sso/saml/metadata"
      ...
    ```

2. Fill in the relevant information from your IdP under the `mdm.end_user_authentication` key. 

3. Run the fleetctl `apply -f fleet-config.yml` command to add your IdP credentials.

4. Confirm that your IdP credentials were saved by running `fleetctl get config`.

### Step 2: upload a EULA to Fleet

1. Head to the **Settings > Integrations > Automatic enrollment** page.

2. Under **End user license agreement (EULA)**, select **Upload** and choose your EULA.

    > Uploading a EULA is optional. If you don't upload a EULA, the end user will skip this step and continue to the next step of the new Mac setup experience after they authenticate with your IdP.

### Step 3: enable end user authentication

You can enable end user authentication using the Fleet UI or fleetctl command-line tool.

Fleet UI:

1. Head to the **Controls > macOS settings > macOS setup > End user authentication** page.

2. Choose which team you want to enable end user authentication for by selecting the desired team in the teams dropdown in the upper left corner.

3. Select the **On** checkbox and select **Save**.

fleetctl CLI: 

1. Choose which team you want to enable end user authentication on.

   In this example, we'll enable end user authentication on the "Workstations (canary)" team so that the authentication is only required for hosts that automatically enroll to this team.

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

    If you want to enable authentication on hosts that automatically enroll to "No team," we'll need to create an `fleet-config.yaml` file:

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

3. Add an `mdm.macos_setup.enable_end_user_authentication` key to your YAML document. This key accepts a boolean value.

4. Run the `fleetctl apply -f workstations-canary-config.yml` command to enable authentication for this team.

5. Confirm that end user authentication is enabled by running the `fleetctl get teams --name=Workstations --yaml` command.

    If you enabled authentication on "No team," run `fleetctl get config`.

    You should see a `true` value for `mdm.macos_setup.enable_end_user_authentication`.

## Bootstrap package

Fleet supports installing a bootstrap package on macOS hosts that automatically enroll to Fleet. 

This enables installing tools like [Puppet](https://www.puppet.com/), [Munki](https://www.munki.org/munki/), or [Chef](https://www.chef.io/products/chef-infra) for configuration management and/or running custom scripts and installing tools like [DEP notify](https://gitlab.com/Mactroll/DEPNotify) to customize the setup experience for you end users.

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

Fleet UI:

1. Head to the **Controls > macOS settings > macOS setup > Bootstrap package** page.

2. Choose which team you want to add the bootstrap package to by selecting the desired team in the teams dropdown in the upper left corner.

3. Select **Upload** and choose your bootstrap package.

fleetctl CLI:

1. Upload the package to a storage location (ex. S3 or GitHub). During step 4, Fleet will retrieve the package from this storage location and host it for deployment.

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

1. Download Fleet's example automatic enrollment profile by navigating to the example [here on GitHub](https://github.com/fleetdm/fleet/blob/main/mdm_profiles/setup_assistant.json) and clicking the download icon.

2. Open the automatic enrollment profile and replace the `profile_name` key with your organization's name.

3. View the the list of macOS Setup Assistant properties (panes) [here in Apple's Device Management documentation](https://developer.apple.com/documentation/devicemanagement/skipkeys) and choose which panes to hide from your end users.

4. In your automatic enrollment profile, edit the `skip_setup_items` array so that it includes the panes you want to hide.

    > You can modify properties other than `skip_setup_items`. These are documented by Apple [here](https://developer.apple.com/documentation/devicemanagement/profile).

### Step 2: upload the profile to Fleet

1. Choose which team you want to add the automatic enrollment profile to.

   In this example, let's assume you have a "Workstations" team as your [default team](./MDM-setup.md#step-6-optional-set-the-default-team-for-hosts-enrolled-via-abm) in Fleet and you want to test your profile before it's used in production. 

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

    > New Macs purchased through Apple Business Manager appear in Fleet with MDM status set to "Pending." Learn more about these hosts [here](./MDM-setup.md#pending-hosts).

3. Transfer this host to the "Workstations (canary)" team by selecting the checkbox to the left of the host and selecting **Transfer** at the top of the table. In the modal, choose the Workstations (canary) team and select **Transfer**.

4. Boot up your test Mac and complete the custom out-of-the-box setup experience.

## Populate the Full Name and Account Name

When an end user unboxes their Mac, they're presented with the **Create a Computer Account** pane where they choose their local Full Name, Account Name, and Password they'll used to login to your Mac.

With Fleet, you can populate the Full Name and Account Name with information from your Identity Provider (IdP).

To populate the Full Name and Account Name, we will do the following steps:

1. Enable end user authentication
2. Configure your IdP
3. Edit your macOS Setup Assistant settings
4. Release the Mac from Await Configuration

### Step 1: enable end user authentication

If you haven't already, follow the steps to enable end user authentication [here](#end-user-authentication-and-eula).

### Step 2: configure your IdP

1. In your IdP, make sure the end user's full name is set to one of the following attributes (depends on IdP): `name`, `displayname`, `cn`, `urn:oid:2.5.4.3`, or `http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name`. Fleet will populate the Account Name with any of these.

2. In your IdP configuration set **Name ID** to the end user's email. Fleet will trim this email and use it to populate the Account Name. For example, a "johndoe@example.com" email turn into a "johndoe" Account Name.

### Step 3: edit your macOS Setup Assistant settings

In Fleet, you control your macOS Setup Assistant settings with an automatic enrollment profile.

Set `await_device_configured` to `true` in your automatic enrollment profile. Learn how to edit your automatic enrollment profile [here](#macos-setup-assistant). 

This will tell the Mac to not allow the user through Setup Assistant until the Mac is released. This way, the Mac has time to get the Full Name and Account Name from your IdP before the **Create a Computer Account** pane.

## Step 4: release the Mac from Await Configuration

Releasing the Mac can be automated using the Fleet API. In this example, we'll release a test Mac manually.

1. Open a freshly wiped test Mac and continue through macOS Setup Assistant.

2. After you click past the **Remote Management** pane, on a separate workstation, login to Fleet and navigate to the **Hosts** page.

3. Find your test Mac and select it to navigate to its **Host details** page.

4. Select **OS settings** and wait until all settings, except for **Disk encryption**, are set to "Verifying."

5. Send the [Release Device from Await Configuration](https://developer.apple.com/documentation/devicemanagement/release_device_from_await_configuration) MDM command using [fleetctl](https://fleetdm.com/docs/using-fleet/mdm-commands#custom-commands) or the [API](https://fleetdm.com/docs/rest-api/rest-api#run-custom-mdm-command). This will release the Mac and allow you to advance to the **Create a Computer Account** pane.

6. At this pane, confirm that the Full Name and Account Name are set to the correct values.

## Wait for configuration profiles

Some organizations want to ensure configuration profiles are installed before the end user can use their Mac. 

For example, at the **Create a Computer Account** pane in macOS Setup Assistant, in order to require that the end user enters a password that meets your organization's password policy, a password policy profile must be installed on the Mac. 

We'll use this example and do the following steps:

1. Add a password policy configuration profile
2. Edit your macOS Setup Assistant settings
3. Release the Mac from Await Configuration

## Step 1: add a password policy configuration profile

1. Download Fleet's password policy configuration profile [here](https://github.com/fleetdm/fleet/blob/main/mdm_profiles/password_policy.mobileconfig). This password policy requires that an end user's password must be at least 10 characters long.

2. Upload the profile to Fleet. Learn how to upload profiles [here](./MDM-custom-macOS-settings.md#enforce-custom-settings)

## Step 2: edit your macOS Setup Assistant settings

Set `await_device_configured` to `true` in your automatic enrollment profile. Learn how to edit your automatic enrollment profile [here](#macos-setup-assistant).

## Step 3: release the Mac from Await Configuration

1. Open a freshly wiped test Mac and continue through macOS Setup Assistant.

2. After you click past the **Remote Management** pane, on a separate workstation, login to Fleet and navigate to the **Hosts** page.

3. Find your test Mac and select it to navigate to its **Host details** page.

4. Select **OS settings** and wait until the **Enforce password length (10 characters)** setting is set to "Verifying."

5. Send the [Release Device from Await Configuration](https://developer.apple.com/documentation/devicemanagement/release_device_from_await_configuration) MDM command using [fleetctl](https://fleetdm.com/docs/using-fleet/mdm-commands#custom-commands) or the [API](https://fleetdm.com/docs/rest-api/rest-api#run-custom-mdm-command). This will release the Mac and allow you to advance to the **Create a Computer Account** pane.

6. At this pane, try to advance with a password that's 9 characters to confirm that your password must be at least 10 characters long.

<meta name="pageOrderInSection" value="1505">
<meta name="title" value="MDM macOS setup">
<meta name="description" value="Customize your macOS setup experience with Fleet Premium by managing user authentication, Setup Assistant panes, and installing bootstrap packages.">
<meta name="navSection" value="Device management">
