# Fleet 3.10.0 released with agent auto-updates beta

Fleet 3.10.0 is now available and more powerful, with the beta for agent auto-updates to better manage osquery updates on your hosts, Identity Provider-Initiated Single Sign-On for increased login flexibility, and improved server logging.

Let’s jump into the highlights:

- Agent auto-updates beta
- Identity Provider-initiated Single Sign-On
- Improved server logging
F
or the complete summary of changes and release binaries check out the [release notes](https://github.com/fleetdm/fleet/releases/tag/3.10.0) on GitHub.


## Agent auto-updates beta

![Agent auto-updates beta](../website/assets/images/articles/fleet-3.10.0-1-600x337@2x.gif)

Updating the osquery version on your hosts helps reveal new capabilities and information from your fleet, but managing updates can be challenging. Our new self-managed agent auto-updates feature, available for Fleet Basic customers, helps you manage osquery agent versions across your fleet.

We’ve also released the beta for [Orbit](https://github.com/fleetdm/orbit), Fleet’s lightweight osquery runtime and auto-updater. By default, Orbit uses the public Fleet update repository to manage auto-updates.


## Identity Provider-Initiated Single Sign-On

![Identity Provider-Initiated Single Sign-On](../website/assets/images/articles/fleet-3.10.0-2-600x337@2x.gif)

Fleet attempts to provide power users with more options for control over their Fleet instance. We’ve introduced Identity Provider-Initiated (IdP-initiated) Single Sign-On (SSO) as a configurable option in Fleet. Turning this option on provides you, and other users of your Fleet instance, with the ability to login to Fleet straight from your configured IdP dashboard.

Please make sure to understand the risks before enabling IdP-initiated SSO. Auth0 provides a great explanation of the [risks and considerations of configuring IdP-initiated SSO](https://auth0.com/docs/protocols/saml-protocol/saml-configuration-options/identity-provider-initiated-single-sign-on#risks-of-using-an-identity-provider-initiated-sso-flow).


## Improved server logging

![Improved server logging](../website/assets/images/articles/fleet-3.10.0-3-600x337@2x.gif)

Generating helpful logs is a vital tool for debugging. Fleet 3.10.0 introduces more consistent logging to assist in highlighting logs of high importance such as errors, agent enrollment, live queries, and others. Thank you William Shoemaker from 
[Atlassian](https://medium.com/u/5aa6b9976187?source=post_page-----f4dd61be001d--------------------------------) for the assistance!

---

## Ready to update?

Visit our [update guide](https://fleetdm.com/docs/using-fleet/updating-fleet) in the Fleet docs for instructions on updating to Fleet 3.10.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2021-04-01">
<meta name="articleTitle" value="Fleet 3.10.0 released with agent auto-updates beta">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-3.10.0-cover-1600x900@2x.jpg">