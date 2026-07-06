# Take control of Apple beta programs with declarative device management

*Blocking betas used to be all-or-nothing. With declarative device management, you decide which devices see which beta programs. Here's how to get the enrollment tokens that make it work.*

'Tis the season for Apple beta programs. As Apple pushes out the next wave of software across its device ecosystem, admins can get ahead of the sprawl of OSes that might land on a fleet, intentionally or not.

Before declarative device management (DDM), your options were blunt: deploy a configuration profile that stopped users from installing beta releases, and that was about it. There are plenty of legitimate reasons to block betas, but a blanket block also limited flexibility — your team couldn't test the next release, and your app developers couldn't test your own software against it. DDM replaces that on/off switch with real policy. Here's how it works, and how to clear the one real hurdle: the enrollment token.

## Key takeaways

- **Beta control is no longer all-or-nothing.** With the DDM software update settings declaration, you can control which beta programs devices are offered, prevent enrollment entirely, or even force specific devices to enroll — per device, not per fleet-wide switch.

- **The declaration is the easy part.** The payload is a few lines of JSON. The part that trips people up is the beta program token it references.

- **Tokens come from Apple Business, and fetching them by hand is tedious.** The manual flow involves generating a key pair, uploading a certificate, downloading an encrypted `.p7m`, decrypting it, and signing an OAuth request.

- **A free script automates the whole token dance.** Microsoft and HCS Technology Group published a script that handles the authentication flow end to end and prints a token for every beta program your organization has accepted terms for.

- **Fleet delivers the declaration like any other OS setting.** Upload the JSON in the Fleet UI or manage it in Git, and scope it with labels so your test devices get offered betas while everyone else stays blocked.

<a purpose="cta-button" href="https://fleetdm.com/try-fleet">Try Fleet</a>

## From blanket blocks to real policy

Apple's [AppleSeed for IT beta program](https://support.apple.com/guide/deployment/test-software-updates-appleseed-beta-program-depe8583cf10/web) exists precisely so organizations can test pre-release software before it reaches production devices. The problem was never whether to test betas — it was that MDM gave you no way to say *who*.

With the declarative framework, the `com.apple.configuration.softwareupdate.settings` declaration changes that. Admins can now control which programs are offered for enrollment, prevent enrollment entirely, or force devices to enroll in a specific program. The full schema is in [Apple's developer documentation](https://developer.apple.com/documentation/devicemanagement/softwareupdatesettings); here's a snippet of the payload:

```json
{
  "Type": "com.apple.configuration.softwareupdate.settings",
  "Identifier": "com.fleetdm.config.softwareupdate.settings",
  "Payload": {
    "Beta": {
      "ProgramEnrollment": "Allowed",
      "OfferPrograms": [
        {
          "Description": "macOS Sequoia AppleSeed Beta",
          "Token": "RPorzFdBlYzes42YomzF7AkJq8BAPBLWUszyUScftPJzC0Zy2vdMUxCfreVrAsam"
        }
      ]
    }
  }
}
```

This example allows enrollment and offers a single program. Swap `ProgramEnrollment` to `AlwaysOff` to block betas outright, or use it with a required program to force enrollment on dedicated test hardware.

## Where did that token come from?

Apple's beta programs require a token from Apple Business (AB) before a managed device can enroll. Getting that token by hand — generating a key pair, uploading a certificate, downloading the encrypted `.p7m`, decrypting it, then signing the OAuth request — is tedious, to say the least.

Microsoft and the team at HCS Technology Group [published a technical article](https://hcsonline.com/support/resources/white-papers/deploy-apple-software-beta-updates-with-jamf-pro-blueprints-without-an-apple-account) showing how to obtain the tokens with a [handy script](https://github.com/microsoft/shell-intune-samples/blob/master/macOS/Tools/getBetaTokens/betaTokens.sh) that automates the whole authentication flow and prints the tokens in a readable table. The README covers the details, but in short, the script grabs the available tokens using a private key and self-signed certificate uploaded to your ABM instance.

## Running the script

On first run, the script generates a certificate and writes it into a newly created `abm_auth` folder in the project. It contains the public key you'll upload to ABM:

```
[INFO] No .p7m found in ./abm_auth.
[ACTION] A certificate will be generated – upload the PEM to ABM, then
         download the issued *.p7m token (it usually lands in ~/Downloads).
[INFO] No .p7m found: generating key + self-signed cert…
Certificate request self-signature ok
subject=CN=Your MDM Server
[ACTION] Upload this PEM to Apple Business Manager (Settings → MDM Servers)
────────────────────────────────────────────────────────────────
-----BEGIN CERTIFICATE-----
MIIC4zCCAcugAwIBAgIUc2IrBH4P4Bd4jvNc74AOwFY7s9cwDQYJKoZIhvcNAQEL
...
-----END CERTIFICATE-----
────────────────────────────────────────────────────────────────
After ABM issues you a *.p7m server token, drop it into this directory.
[INFO] Watching /Users/miso/Downloads for NEW *.p7m files (every 5s)…
```

Keep the script running — it actively watches for the token you'll download in the next step.

In ABM, go to **Settings → MDM Servers** (or **Devices → Management Services**), click **Add** next to *Add device management service*, give it a name, and upload the `mdm_public_cert.pem` generated in the previous step. Click **Download Service Token** — once you leave this page, you can't re-download it.

Assuming the token lands in your Downloads folder, the running script picks it up automatically and executes the next phase. The output below is truncated for brevity, but you'll get all available tokens for the beta programs you've accepted the terms for — everything from macOS to homePodOS. Note: the program tokens below have been replaced with randomly generated values.

```
[INFO] New token detected in Downloads: Beta Tokens_Token_2026-06-17T23-30-23Z_smime.p7m
[INFO] Stripping S/MIME wrapper…
[INFO] Got credentials:
[INFO] Building OAuth header for session request…
[INFO] Requesting session token…
[INFO] Got session token.
[INFO] Fetching beta-enrollment tokens…
[INFO] Available beta programs:
┌─────────────────────────────────────┬─────────┬──────────────────────────────────────────────────────────────────┐
│ Title                               │ OS      │ Token                                                            │
├─────────────────────────────────────┼─────────┼──────────────────────────────────────────────────────────────────┤
│ iOS 26 AppleSeed Beta               │ iOS     │ 8rLMZ3zBthBUrWql1AUKxmegTSLHHn4bmk8nq66dVahwid5ViGSHgY2yh4eqYmAH │
│ iOS 27 AppleSeed Beta               │ iOS     │ mNxQ4ZnT4MFAEh7FAuSDR4LlOoocQXMCTbSMp9UvCJRAlS1aTPu7ugZNBrnHiOuB │
│ macOS 27 Golden Gate AppleSeed Beta │ OSX     │ fv3EnlTvYDdaaQvORIDQ6fiYovbDqeEwetuEPy3DWn3MZjMtvqLhaHXt3wbu8zPc │
│ macOS Sequoia AppleSeed Beta        │ OSX     │ 8ldAUH5JCP15M9GoTwZWPFsIPpjHM20NH78zHu8QUFEqYxxjxTvp9foH6tPN9mm9 │
│ macOS Tahoe 26 AppleSeed Beta       │ OSX     │ YaMpsLkQrjyCqjOtzW6B694HStHHGUC2KxUpab5INXrb8qA4hxENcet1htFLlpGd │
│ watchOS 26 AppleSeed Beta           │ watchOS │ 9aQYc2NF3lnlhUHEA8OG6PFiMo34OKXNBAK6M6iRmc931cDtH5g9YPAW6NEoCdPs │
│ watchOS 27 AppleSeed Beta           │ watchOS │ H0mCCV5aD2qwyA6fa7kyz5hL5ktm3Tb8lmUpGCUh86mtwMNdLrMF2NPiwP2if3LA │
└─────────────────────────────────────┴─────────┴──────────────────────────────────────────────────────────────────┘
```

## Deploying the declaration with Fleet

Now that you have the tokens, build the declarations that fit your organization's needs. Fleet delivers DDM declarations the same way it delivers any other [custom OS setting](https://fleetdm.com/guides/custom-os-settings): save the payload as a `.json` file and upload it under **Controls → OS settings → Custom settings**, or check it into your repo and ship it through [GitOps](https://fleetdm.com/docs/configuration/yaml-files) so the change is reviewed in a pull request before it reaches a single device.

Scoping is where this gets useful. Add the declaration that offers beta programs to the fleet or label containing your test devices — the IT team's spare hardware, your app developers' secondary machines — and add an `AlwaysOff` variant everywhere else. Your testers get the next macOS the day it seeds; everyone else's devices never see the offer.

## Test on your terms

The betas are coming either way. The difference DDM makes is whether they arrive on the devices you chose, enrolled in the programs you picked, or wherever an eager user happens to tap "enroll." With the token script doing the hard part and a declaration doing the enforcement, an afternoon of setup buys you a controlled beta program for the whole release cycle.

## See it live

- [**Get a demo**](https://fleetdm.com/contact)**.** We'll walk through deploying DDM declarations and scoping them to the right devices in a real environment.
- [**Read the custom OS settings guide**](https://fleetdm.com/guides/custom-os-settings)**.** Everything Fleet supports for configuration profiles and DDM declarations, including verification.

*Fleet is the open-source endpoint management platform for macOS, Windows, Linux, and more. Want to manage OS settings as code?* [*Explore Fleet's GitOps workflow*](https://fleetdm.com/docs/configuration/yaml-files) *or* [*get a demo*](https://fleetdm.com/contact)*.*

<meta name="articleTitle" value="Take control of Apple beta programs with declarative device management">
<meta name="authorFullName" value="Harrison Ravazzolo">
<meta name="authorGitHubUsername" value="harrisonravazzolo">
<meta name="category" value="articles">
<meta name="publishedOn" value="2026-07-02">
<meta name="description" value="Use DDM's software update settings to control Apple beta program enrollment, and automate fetching AppleSeed tokens from Apple Business.">
