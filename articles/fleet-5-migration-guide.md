# Migrate from Fleet 4 to Fleet 5

Fleet 5 removes deprecated features such as API endpoints, API parameters, and YAML names. This guide lists each breaking change and the steps to update your setup, so you can upgrade without issues.


## Prerequisites

Check these before you start:

- You're running the latest version of Fleet 4. Update to it first, then follow the steps below.
- For SSO changes, you have access to your identity provider (IdP).


## Single sign-on (SSO)

- In your IdP, rename the `FLEET_JIT_USER_ROLE_TEAM_<ID>` SAML attribute to `FLEET_JIT_USER_ROLE_FLEET_<ID>`. ([#53925](https://github.com/fleetdm/fleet/issues/53925))

> **Warning:** Fleet 5 ignores `FLEET_JIT_USER_ROLE_TEAM_<ID>`. If you don't rename it, new users who log in with SSO get the global observer role, and existing users keep the roles they have now.


<meta name="articleTitle" value="Migrate from Fleet 4 to Fleet 5">
<meta name="authorFullName" value="Steven Palmesano">
<meta name="authorGitHubUsername" value="spalmesano0">
<meta name="publishedOn" value="2026-10-30">
<meta name="category" value="guides">
<meta name="description" value="Prepare for the upgrade to Fleet 5.">
