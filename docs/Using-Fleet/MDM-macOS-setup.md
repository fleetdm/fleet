# macOS setup

In Fleet, you can customize the first-time macOS setup experience for your end users:

You can customize the macOS Setup Assistant and choose to show or hide specific panes.
* Require end users to authenticate with your identity provider (IdP) and agree to an end user license agreement (EULA) before they can use their new Mac

* Customize the macOS Setup Assistant by choosing to show or hide specific panes

* Bootstrap new Macs with your configuration management tool of choice (ex. Munki, Chef, or Puppet)

## End user authentication

> This feature is currently in development.

## Bootstrap package

_Available in Fleet Premium_

Fleet supports installing a single package on macOS hosts that automatically enroll to Fleet.

> In addition to installing the bootstrap package, Fleet automatically installs the fleetd agent on hosts that automatically enroll. This agent is responsible for reporting host vitals to Fleet and presenting Fleet Desktop to the end user.

To add a bootstrap package to Fleet, we will do the following steps:

1. Download or generate a package
2. Sign the package
3. Upload the package to Fleet

### Step 1: Download or generate a package

If you use Munki, Chef, Puppet, or another configuration management tool, download the client (agent) for your tool. You can find the client on each tool's GitHub or website. For example, you can download Munki, the Munki client on their [releases page on GitHub](https://github.com/munki/munki/releases). 

Make sure the file you download is a `.pkg` file.

If you plan to run a custom script during macOS setup, you'll need to generate a package. The [munkipkg tool](https://github.com/munki/munki-pkg) is a popular tool for generating packages.

Make sure the package you generate is a `.pkg` file.

### Step 2: Sign the package

1. Obtain an appropriate TLS/SSL certificate with signing capability. You can do this by:
   - Using an [Apple developer account](https://developer.apple.com/account).
     - [Link your developer account to
       Xcode](https://help.apple.com/xcode/mac/current/#/dev154b28f09) or [using your developer account](https://developer.apple.com/help/account/create-certificates/create-developer-id-certificates).
     - Ensure you choose "Developer ID Installer" as the certificate type.
     - Verify the certificate is saved to your macOS Keychain.

   - Acquiring a certificate from third parties that meet the requirements.
     - Follow their instructions for creating a TLS/SSL certificate with signing usage.
     - Add the resulting `.p12` to your keychain.
2. Sign your `pkg` with a valid Developer ID certificate:\
\
```productsign --sign "Developer ID Installer: Your Developer Name (SerialNumber)" /path/to/your.pkg /path/to/your-signed.pkg``` \

3. Upload the package to Apple's notary service for validation: \
\
     ```xcrun notarytool submit --keychain-profile "Your Keychain Name" --wait your-signed.pkg```\
\
This command will upload your package to the notary service, which will validate it and return a UUID that can be used to check its status later on.

4. Wait for the notary service to validate your package. You can use the UUID from the previous step
   to check the status of your package: \
\
```xcrun notarytool info --keychain-profile "Your Keychain Name" --wait <uuid>```\
\
The `--wait` flag will cause the command to poll the notary service until the validation is complete. If the validation is successful, the command will return a JSON object with information about the notarization.
5. Optional: Once your package has been notarized, staple the notarization ticket to it: \
\
```xcrun stapler staple your-signed.pkg```\
\
This command will attach the notarization ticket to your package, which will allow Gatekeeper to verify its authenticity even if it's not connected to the internet.

6. Verify that your package has been notarized and stapled correctly: \
\
```pkgutil –check-signature /path/to/installer.pkg```\
\
This command will return *Notarization: trusted by the Apple notary service* along with the
Certificate Chain.\
\
or\
\
```spctl -a -v your-signed.pkg```\
\
This command will use the `spctl` tool to verify that your package has been signed, notarized, and
stapled correctly. If everything is in order, the command will return "accepted."\
\
Your package can be safely distributed and installed on macOS devices managed by your MDM.

### Step 3: Upload the package to Fleet

Fleet supports installing a unique bootstrap package for each team. In Fleet, a team is a group of hosts.

1. Upload the package to a publicly accessible location on the internet. We'll point Fleet to this location so that Fleet can download the package.

2. Create a `team` YAML document if you don't already have one. Learn how [here](./configuration-files/README.md#teams). If you're uploading the package to a team that already exists, make sure the `name` key in your YAML document matches the name of the team.

> If you want to install a bootstrap package on hosts that are assigned to "No team," use the `config` YAML document. Learn how to create one [here](./configuration-files/README.md#organization-settings). 

3. Add an `mdm.macos_setup.bootstrap_package` key to your YAML document. This key accepts an absolute URL to the location of the bootstrap package. 

```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: Workstations
    mdm:
      macos_setup:
        bootstrap_package: https://github.com/organinzation/repository/bootstrap-package.pkg
```

Run the fleetctl `apply -f <your-team-here>.yml` command to upload your bootstrap package to Fleet.

## macOS Setup Assistant

> This feature is currently in development.

<meta name="pageOrderInSection" value="1504">
<meta name="title" value="MDM macOS setup">
