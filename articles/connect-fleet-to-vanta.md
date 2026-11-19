# Connect Fleet to Vanta

Fleet's Vanta integration sends your Fleet users and your macOS and Windows hosts to Vanta every hour, so Vanta collects device and access evidence automatically instead of at audit time. This guide covers how the integration works, how to set it up, and how Fleet handles the API token you provide.

## How it works

Vanta integrations run through an OAuth app that the vendor registers with Vanta. Fleet hosts its app on fleetdm.com, so you don't run anything extra and the Vanta client secret never has to be installed on your Fleet server.

1. On [fleetdm.com/connect-vanta](https://fleetdm.com/connect-vanta), you provide your Fleet URL and an API-only user's token.
2. fleetdm.com verifies the token against your Fleet instance, then sends you to app.vanta.com to authorize the connection.
3. Every hour, a job on fleetdm.com reads users and hosts from your Fleet instance and sends them to Vanta's resource sync API.

Your Fleet server never talks to Vanta directly.

The integration sends:

- **Users:** name, email, role, creation date, and whether they sign in with SSO, a password, or an API token.
- **Hosts:** display name, serial number, hardware UUID, OS version, MDM enrollment status, disk encryption status, installed applications or programs, and browser extensions. Most built-in macOS applications are excluded to keep requests small.

## Prerequisites

- Fleet Premium.
- A Vanta account with permission to add integrations.
- Fewer than 1,000 hosts. Vanta's connector API doesn't support more.
- A browser that accepts cookies from fleetdm.com. The authorization flow uses signed cookies to verify your return from Vanta.

> **Note:** The integration syncs macOS and Windows hosts only. Screen lock status is not available, so every host shows screen lock as false in Vanta. Use the [macOS](https://fleetdm.com/queries/screen-lock-enabled-mac-os) and [Windows](https://fleetdm.com/queries/screen-lock-enabled-windows) screen lock queries to see the real status.

## Step 1: Create an API-only user

Create an [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user) with the global **Admin** role. The admin role is required because the integration reports every user account on your Fleet instance.

```sh
fleetctl user create --name 'Vanta' --api-only --global-role 'admin'
```

Then limit what the token can do. In Fleet, head to **Settings > Users**, select the new user, set **API access** to **Specific API endpoints**, and add only these:

- Get current user: `GET /api/v1/fleet/me`
- Get Fleet configuration: `GET /api/v1/fleet/config`
- List users: `GET /api/v1/fleet/users`
- List hosts: `GET /api/v1/fleet/hosts`
- Get host: `GET /api/v1/fleet/hosts/:id`

All five are read-only. Scoping limits what the token can do if it leaks. It doesn't grant anything the Admin role forbids.

## Step 2: Connect your Fleet instance

1. Go to [fleetdm.com/connect-vanta](https://fleetdm.com/connect-vanta).
2. Enter an email address Fleet can use to contact you if the sync breaks.
3. Enter your Fleet instance URL, for example `https://fleet.example.com`.
4. Paste the API-only user's token and select **Connect**.
5. On app.vanta.com, approve the connection. Vanta sends you back to fleetdm.com, which shows a success message.

The first sync runs within an hour.

## How your API token is handled

- The form submits over HTTPS to fleetdm.com. The token is handled server side only and is never sent to Vanta or back to your browser.
- Before anything is stored, fleetdm.com calls your instance to confirm the token belongs to an API-only user with the global Admin role on a Premium license. Other tokens are rejected.
- The token is stored in fleetdm.com's database, which is encrypted at rest by the hosting provider. Database access is limited to the engineers who operate fleetdm.com.
- The hourly sync makes only the read-only requests listed in Step 1. Requests use the user agent `Fleet/Vanta Updater`, so you can find them in your Fleet server logs.
- You control the token. Deleting the API-only user in Fleet, or resetting its token, stops the integration immediately.

## Verify

1. In Vanta, open the Fleet integration and confirm hosts and users appear after the first hourly sync.
2. In Fleet, head to **Settings > Users** and confirm the Vanta API-only user still has the Admin role and the five scoped endpoints.

## Troubleshoot

**"The API key provided is not associated with an API-only user"**

The token belongs to a regular user. Create an API-only user and use its token instead.

**"The API key provided has insufficient permissions"**

The API-only user needs the global Admin role, not a fleet-scoped role.

**"This Fleet instance is already connected to a Vanta account"**

Each Fleet URL can connect to one Vanta account. [Contact us](https://fleetdm.com/contact) to reconnect or change an existing connection.

**"One or more of the required cookies were missing"**

Your browser blocked cookies from fleetdm.com during the authorization flow. Allow cookies and [try again](https://fleetdm.com/connect-vanta).

**Hosts or users stop updating in Vanta**

The sync stops if the API-only user is deleted, its token is reset, or its role or endpoint scope changes. Check the user in **Settings > Users**, then reconnect if needed.

<meta name="articleTitle" value="Connect Fleet to Vanta">
<meta name="authorFullName" value="Luke Heath">
<meta name="authorGitHubUsername" value="lukeheath">
<meta name="publishedOn" value="2026-10-23">
<meta name="category" value="guides">
<meta name="description" value="Send Fleet users and hosts to Vanta for continuous compliance evidence, and learn how Fleet handles the API token you provide.">
