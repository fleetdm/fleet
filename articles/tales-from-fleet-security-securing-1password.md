# Tales from Fleet security: securing 1Password

![Securing 1Password](../website/assets/images/articles/tales-from-fleet-security-securing-1password-cover-1600x900@2x.jpg)

No matter how much we wish SaaS tools had support for Single Sign-On (SSO), there are still so many
websites and other resources that require individual passwords. Running a company without providing
employees with a password manager is setting them up for failure. Of course people will use the same
asswords on multiple sites if they do not have a way to manage different ones. That being said,
password managers do centralize a lot of the security eggs in the same basket, which is why the
manager itself must be as well protected as possible and why hardware security keys should be used
on high-value systems.

At Fleet, we use 1Password. While configuring 1Password is relatively straightforward, here are a few things we do that can help you secure your 1Password instance.

## Require 2FA
By setting the Account Password Policy to Strong, we gained the ability to support and then require 2FA.

When enabling 2FA, we ensured we also required it, as otherwise, there would always be some users without it.

## 2FA methods
We highly recommend that all of our users configure security keys as a 2FA method, but
unfortunately, 1Password does not allow enforcing this. For this reason, we recommend that employees
configure their keys and then delete the tokens from their authenticator apps.

We have made a feature request to 1Password, as having a very secure authentication method is
excellent, but if weaker forms remain available, what's the point?

## Restrict the number of administrators and powerful accounts
If an administrator account or an account with account management privileges gets compromised,
things could go south quickly.

## Ensure at least one admin recovery kit is stored securely
1Password, the company, can't retrieve our data as it is encrypted. If all admins were to get locked
out at once, this could lead to data loss. For this reason, we store a physical copy of an emergency
kit in a secure physical location.

## Get rid of recovery kits on computers
We also ask that everyone delete emergency kits from their computers. To catch mistakes, we also run
a policy in our instance of Fleet that runs this query:

```
SELECT 1 WHERE NOT EXISTS (SELECT * FROM file WHERE path LIKE `/Users/%%` AND filename LIKE "%Emergency Kit%.pdf");
```

This query succeeds if it does not find PDF files with a name like "Emergency Kit" in user
directories.

## Require modern apps
We enable this feature to block old, unsupported clients from accessing our vaults. Since these
older clients might have vulnerabilities or not support the latest security features, this reduces
the odds that something could go wrong.

On top of blocking old clients, we also push 1Password updates to our managed workstations to keep
them up to date instead of only stopping very old clients.

## Slack notifications
We have configured Slack notifications, which you can find under the integrations configuration page
in 1Password. We ensure that our security and operations teams see critical information about
1Password accounts, such as recovery attempt requests.

## Recovery process
We documented and practiced our recovery process to ensure everyone with access to perform
recoveries knows how to identify the requester. We also ensure that anyone working on a recovery
warns everyone and confirms they have identified the requester.

## Item sharing
Item sharing is one of these features where we can't recommend a setting. More restrictive is more secure unless you need to often share secrets with third parties. We simply recommend picking what makes the most sense for you.

## Effort and conclusion
We feel our 1Password environment is safer than the defaults with these settings. Applying these settings takes a matter of minutes and will require users to enable 2FA.

While additional configuration is possible to secure the Mac 1Password application, support for this is, unfortunately, 1Password has not kept these features in the recently released 1Password 8. If you are using 1Password 7, we definitely recommend checking [them out](https://support.1password.com/mobile-device-management/).

## Want to discuss this further?
Feel free to drop in our #Fleet [Slack Channel](https://fleetdm.com/slack) to discuss anything security-related with us!

## What's next?
Stay tuned for our next article in the Tales from Fleet security series!

<meta name="category" value="security">
<meta name="authorFullName" value="Guillaume Ross">
<meta name="authorGitHubUsername" value="GuillaumeRoss">
<meta name="publishedOn" value="2022-05-06">
<meta name="articleTitle" value="Tales from Fleet security: securing 1Password">
<meta name="articleImageUrl" value="../website/assets/images/articles/tales-from-fleet-security-securing-1password-cover-1600x900@2x.jpg">