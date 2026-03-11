# Create a managed local admin account on macOS

## Introduction

This guide will walk you through creating a managed local admin account on your macOS hosts
using Fleet. This "break glass" account gives your IT team a hidden, Secure Token-enabled
admin account they can use to SSH into hosts for troubleshooting.

Fleet automatically generates unique credentials for each host and stores them securely. The
managed account (default username: "FleetAdmin") is hidden from the macOS login window so end
users don't see it.

Because there is no MDM payload for enabling Remote Login (SSH), this guide also covers how
to deploy a script via Fleet to ensure SSH is available on your hosts before attempting to
connect with the managed account.

## Prerequisites

* Fleet installed and hosts enrolled
* MDM enabled and configured
* Admin access to the Fleet UI or `fleetctl` CLI

## Step-by-step instructions

### **Step 1: Enable Remote Login (SSH) on your hosts**

Remote Login must be enabled on macOS hosts before you can SSH in with the managed account. Apple does not provide an MDM payload for this setting, so you'll deploy a script via Fleet.

Create a file named `enable-remote-login.sh`:

```bash
#!/bin/bash
sudo systemsetup -setremotelogin on
echo "Remote Login (SSH) has been enabled."
```

Upload the file in **Controls > Scripts** in Fleet, then run on target hosts. 

Or use:
`fleetctl run-script --script-path=./enable-remote-login.sh --host=<hostname>`


### **Step 2: Enable the managed local account**

In Fleet UI: **Controls > OS settings > Setup experience > Create a managed local account on macOS hosts**

In YAML:

```yaml
controls:
  create_local_managed_account: true
```

Fleet creates a hidden admin account (FleetAdmin) with a unique password per host.

### **Step 3: Retrieve credentials and connect**
**Host details > Actions > Show managed account**: reveals SSH command and password.

Connect via:
`ssh FleetAdmin@<host-ip>`

Credentials viewing is logged in Fleet’s activity feed.

### **Step 4: Local login**

At the login window, use "Other…" and enter FleetAdmin and the password. Account is hidden except after restart or if FileVault-enabled.

### Troubleshooting

* If Remote Login is not enabled, you will receive:
`ssh: connect to host <ip-address> port 22: Connection refused`

* If SSH fails with "Connection refused", confirm script execution and check host activity feed.

## Conclusion

This guide covered how to enable Remote Login via a Fleet script, create a managed local admin account on 
your macOS hosts, view the account credentials, and connect via SSH for troubleshooting. By combining Fleet's 
managed local account feature with a Remote Login script, your IT team gets a secure, auditable break-glass 
workflow for macOS support.

See [Allow remote access to your Mac](https://support.apple.com/guide/mac-help/allow-a-remote-computer-to-access-your-mac-mchlp1066/mac)
and [Hide a user account in macOS](https://support.apple.com/en-us/102099) for more detailed information.

<meta name="articleTitle" value="Create a managed local admin account on macOS">
<meta name="authorFullName" value="Mel Pike">
<meta name="authorGitHubUsername" value="melpike">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-03-11">
<meta name="description" value="Guide to creating a managed local admin account on macOS hosts using Fleet, including enabling Remote Login for SSH access.">
