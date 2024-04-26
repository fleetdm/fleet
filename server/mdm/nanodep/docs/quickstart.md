# NanoDEP Quick Start Guide

A guide to getting NanoDEP up and running quickly. For more in-depth documentation please see the [Operations Guide](operations-guide.md).

## Requirements

* An Apple Business Manager (ABM), Apple School Manager (ASM), or Business Essentials (BE) login account with at least Device Management permissions/abilities.
* Devices already present in your ABM/ASM/BE system to assign.
* For the [tools](../tools) you'll need `curl`, `jq`, and of course a shell script interpreter.
* Outbound internet access to talk to Apple's DEP APIs.

## Guide to creating a DEP profile for a device

What follows is a step-by-step guide to creating a DEP profile and assigning it to a device. This should allow a device to use [Automated Device Enrollment (ADE)](https://support.apple.com/en-us/HT204142).

### Get NanoDEP and other tools

First, get a copy of NanoDEP by downloading and extracting a [recent release](https://github.com/micromdm/nanodep/releases) or compiling from source. You'll also want to make sure you have `jq` and `curl` installed. As well you'll want to have the shell scripts from the `tools/` directory downloaded and available which are included in the release.

### Start depserver

Start `depserver`. Note the port (default 9001) it started on. We also set an API key here.

```bash
$ ./depserver-darwin-amd64 -api supersecret
2022/07/02 14:14:18 level=info msg=starting server listen=:9001
```

### Setup environment

We need to setup our environment so our [tools](../tools) can talk to this running depserver.

```bash
export BASE_URL='http://[::1]:9001'
export APIKEY=supersecret
export DEP_NAME=mdmserver1
```

Note here the "DEP name" of `mdmserver1` is arbitrary and can be anything you like (but avoid forward-slashes "/" as the APIs use this name as part of the URL). The depserver and related tools support multiple DEP server configurations so this uniquely identifies the DEP server we want to work with.

### Generate and retrieve the DEP token public key

The ABM/ASM/BE portal uses a public key to encrypt the OAuth1 tokens. To generate a new keypair and retrieve the public key (in an X.509 Certificate):

```bash
$ ./tools/cfg-get-cert.sh > $DEP_NAME.pem
```

Note this should create a new file called "mdmserver1.pem" (or whatever you set `$DEP_NAME` to, above).

### Upload the public key to ABM/ASM/BE

Login to https://business.apple.com/ or https://school.apple.com/ in a browser then navigate to the list of MDM servers. As of July 2022 this is done by navigating to the lower-left menu by clicking on your login name and selecting "Preferences." Under the separator there's a list titled "Your MDM Servers."

Create a new MDM server by clicking the "+" or "Add" button by the list header. Give it a name: perhaps something related to the "mdmserver1" name so you can remember these are associated. Then, upload the public key certificate generated in the last step. Click "Save".

### Download Token

Next, we'll want to download the token. From within the ABM/ASM/BE portal navigate to your newly created (or modified) MDM server. As of July 2022 there's a top menu for the MDM server which contains a button/link to "Download Token." Click this to download the token which should download a file with the extension ".p7m" and named after the MDM server you created: this downloaded token is the encrypted OAuth tokens for DEP access.

### Decrypt tokens

To decrypt the OAuth tokens and save them to the DEP server for use:

```bash
$ ./tools/cfg-decrypt-tokens.sh ~/Downloads/mdmserver1_Token_2022-07-01T22-18-53Z_smime.p7m
{"consumer_key":"CK_9af2f8218b150c351ad802c6f3d66abe","consumer_secret":"CS_9af2f8218b150c351ad802c6f3d66abe","access_token":"AT_9af2f8218b150c351ad802c6f3d66abe","access_secret":"AS_9af2f8218b150c351ad802c6f3d66abe","access_token_expiry":"2023-07-01T22:18:53Z"}
```

The server's reply is the decrypted OAuth tokens. `depserver` should now have authenticated access to talk to Apple's API!

### Request account detail

As a test of our DEP authentication, let's request account detail:

```bash
$ ./tools/dep-account-detail.sh
{
  "server_name": "Example Server",
  "server_uuid": "677cab70-fe18-11e2-b778-0800200c9a66",
  "facilitator_id": "facilitator@example.com",
  "org_phone": "111-222-3333",
  "org_name": "Example Inc",
  "org_email": "orgadmin@example.com",
  "org_address": "123 Main St. Anytown, USA",
  "admin_id": "admin@example.com"
}
```

If you received no response here then you can e.g. set `export CURL_OPTS=-v` to give us more detail and check the `depserver` logs if necessary. See the [operations guide](../docs/operations-guide.md) for more.

Otherwise: congratulations! The token exchanged was successful and you can use the tokens to communicate with Apple's DEP API. **Note: you will need renew these tokens yearly or whenever the Apple Terms and Conditions are updated by following this same procedure.**

### Assign a device in the portal

Now that we've verified API connectivity using your DEP server you need to assign a device in the ABM/ASM/BE portal. To do so login to the portal and navigate to the "Devices" section. Select (or search for) the device you want to use with DEP by settings its MDM server. As of July, 2022 there is a link/button in the top navigation of a device called "Edit MDM Server" — clicking this brings up a dialog to either assign or un-assign the device. When assigning a drop-menu appears of the setup MDM servers. We'll want to select our newly created server "mdmserver1" then click the "Continue" button. The device should then be assigned to your MDM server and available for a DEP profile to be assigned to it.

### Define a DEP Profile and assign a device

DEP works by associating devices (serial numbers) with DEP profiles. A DEP profile is a set of properties associated to a UUID and, importantly, specifies the URL location of our MDM server our device enrolls into. We can define a DEP profile with its properties *and* associate serial numbers in one step.

First adjust the [example DEP profile](../docs/dep-profile.example.json) or make a copy of it. Critically you'll need to point the profile at your MDM using the `url` or `configuration_web_url` properties. See the [Apple docs](https://developer.apple.com/documentation/devicemanagement/profile) for the various configuration options. For the below example I adjust a few parameters, made sure my MDM URL is correct, and added the serial `07AAD449616F566C12` to the `devices` array in the profile (note only serial number adjustment shown here, you *will* need to adjust other parameters of the profile):

```diff
--- a/docs/dep-profile.example.json
+++ b/docs/dep-profile.example.json
@@ -18,5 +18,5 @@
   "anchor_certs": [],
   "supervising_host_certs": [],
   "skip_setup_items": ["AppleID", "Android"],
-  "devices": ["SERIAL1","SERIAL2"]
+  "devices": ["07AAD449616F566C12"]
 }
```

Then, we assign the profile:

```bash
$ ./tools/dep-define-profile.sh ./docs/dep-profile.example.json
{
  "profile_uuid": "43277A13FBCA0CFC",
  "devices": {
    "07AAD449616F566C12": "SUCCESS"
  }
}
```

Here the API has responded telling us the profile UUID `43277A13FBCA0CFC` has been defined and that it had success in assigning the serial number `07AAD449616F566C12` this DEP profile.

### Verify device ADE

Now that you have a device assigned to a DEP profile you can proceed to verifying and testing Automated Device Enrollment (ADE) by enrolling a device (likely erasing it first).

Note that getting ADE working on devices including the appropriate properties in the DEP profile is outside the scope of this document (and this project) as it requires integration with MDM enrollment and the details of how your enrollment profile is available — none of which this project is aware of.

That said, please check out these [MicroMDM](https://github.com/micromdm/micromdm) project resources for troubleshooting DEP on macOS if you believe your DEP profile is defined correctly:

* https://micromdm.io/blog/troubleshoot-dep/
* https://github.com/micromdm/micromdm/wiki/Troubleshooting-MDM#dep--mdm-testing--troubleshooting

### Next steps

Here's a few ideas on where to proceed next:

* Read the [Operations Guide](../docs/operations-guide.md) for more details on configuration, troubleshooting, etc.
* Setup an assigner to *automatically assign* serial numbers to DEP profiles when they're added to DEP (see the operations guide)
* Setup more than one DEP server — with the tools scripts this really just means changing the `$DEP_NAME` environment variable.
* A proper deployment
  * Behind HTTPS/proxies
  * Behind firewalls or in a private cloud/VPC
  * In a container environment like Docker, Kubernetes, etc. or even just running as a service with systemctl.
