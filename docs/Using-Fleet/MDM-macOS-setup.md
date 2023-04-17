# macOS setup

In Fleet you can customize the first time macOS setup experience for your end users.

You can require end users to authenticate with your identity provider (IdP) and agree to and end user license agreement (EULA) before they can use their new Mac.

You can customize the macOS Setup Assistant and choose to show or hide specific panes.

Also, you can bootstrap new Macs with your configuration management tool of choice (ex. Munki, Chef, or Puppet).

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

### Step 1: download or generate a package

If you use Munki, Chef, Puppet, or another configuration management tool, download the client (agent) for your tool. You can find the client on each tool's GitHub or website. For example, you can download Munki, the Munki client on their [releases page in GitHub](https://github.com/munki/munki/releases). 

Make sure the file you download is a `.pkg` file.

If you plan to run a custom script during macOS setup, you'll need to generate a package. The [munkipkg tool](https://github.com/munki/munki-pkg) is a popular tool for generating packages.

Make sure the package you generate is a `.pkg` file.

### Step 2: Sign the package

To sign the package, you need an Apple developer account. [Create one here](developer.apple.com/account).

Sign the package with a valid Developer ID Installer certificate. Learn how to create a certificate [here in the Xcode documentation](https://help.apple.com/xcode/mac/current/#/dev154b28f09). 

### Step 3: Upload the package to Fleet

Fleet supports installing a unique bootstrap packages for each team. In Fleet, a team is a group of hosts.

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
<meta name="title" value="MDM commands">
