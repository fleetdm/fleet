# Conditional access: Cloudflare

With Fleet's [conditional access](https://fleetdm.com/guides/conditional-access), Cloudflare One blocks access to apps and networks from hosts that are failing Fleet policies, or that aren't in Fleet at all. You don't need an Intune or Okta license.

How it works:

1. Every few minutes, Cloudflare sends the serial numbers of devices running the Cloudflare One Client (formerly WARP) to a Cloudflare Worker.
2. The Worker looks up each serial number in Fleet. It returns a score of `100` for hosts that pass your policies, and `0` for hosts that fail them or aren't in Fleet.
3. A device posture check in Cloudflare passes only a score of `100`. Access and Gateway policies that require the check block everything else.

This guide covers macOS, Windows, and Linux hosts. iPhones, iPads, and Android devices weren't tested.

> **Note:** Unlike the built-in [Okta](https://fleetdm.com/guides/okta-conditional-access-integration) and [Entra](https://fleetdm.com/guides/entra-conditional-access-integration) integrations, Cloudflare checks hosts on a schedule, not at each sign-in. The end user doesn't get a **Resolve later** option, and the block lasts until the host passes and Cloudflare checks it again.

## Prerequisites

- A Cloudflare Zero Trust organization, and permission to create Workers, service tokens, and Access applications
- The Cloudflare One Client [deployed with Fleet](https://fleetdm.com/guides/deploying-cloudflare-warp-with-fleet) and enrolled in your Zero Trust organization on each host
- A Fleet server that Cloudflare Workers can reach over the internet
- Node.js and npm on the computer you deploy from
- A unique serial number for each host. Cloudflare and the Worker match hosts by serial number.

## Step 1: Create a Fleet API user

Create an [API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user) with the global **Observer** role. The Worker uses its API token.

Select **Specific API endpoints** and add only the endpoint the Worker calls: [List hosts](https://fleetdm.com/docs/rest-api/rest-api#list-hosts).

> **Note:** The Worker gives a score of `0` to any host the API user can't see. If you give the user a fleet-level role instead of a global role, hosts in other fleets are blocked.

## Step 2: Deploy the Worker

1. Make a new folder with a `src` folder inside it. Download [`fleet-cloudflare-posture-worker.js`](https://github.com/fleetdm/fleet/blob/main/docs/solutions/api-scripts/fleet-cloudflare-posture-worker.js) and save it as `src/index.js`.
2. In the folder, install the Worker's one dependency:

```sh
npm init -y
npm install jose
```

3. Create a `wrangler.jsonc` file. Set `FLEET_URL` to your Fleet server URL and `TEAM_DOMAIN` to your Zero Trust team domain. You'll set `POLICY_AUD` in Step 3.

```json
{
  "name": "fleet-posture",
  "main": "src/index.js",
  "compatibility_date": "2026-10-01",
  "vars": {
    "FLEET_URL": "https://fleet.example.com",
    "TEAM_DOMAIN": "<your-team-name>.cloudflareaccess.com",
    "POLICY_AUD": "",
    "POLICY_IDS": ""
  }
}
```

4. Add the API token from Step 1 as a secret, then deploy:

```sh
npx wrangler secret put FLEET_API_TOKEN
npx wrangler deploy
```

Copy the Worker URL that `wrangler deploy` prints. For example, `https://fleet-posture.<your-subdomain>.workers.dev`.

By default, a host fails if it's failing any policy in Fleet. To count only some policies, set `POLICY_IDS` to a comma-separated list of policy IDs, like `"12,15"`. A policy's ID is the number at the end of its URL in the Fleet UI.

> **Note:** A policy that hasn't run on a host yet doesn't count as failing. Fleet runs policies every hour by default (see [`osquery_policy_update_interval`](https://fleetdm.com/docs/configuration/fleet-server-configuration#osquery-policy-update-interval)), so a host can take up to an hour, plus Cloudflare's polling frequency, to pass or fail.

## Step 3: Put the Worker behind Cloudflare Access

Cloudflare authenticates to the Worker with an Access service token. The Worker rejects any request that doesn't come through Access.

1. In the [Cloudflare dashboard](https://dash.cloudflare.com/), [create a service token](https://developers.cloudflare.com/cloudflare-one/access-controls/service-credentials/service-tokens/#create-a-service-token). Copy the **Client ID** and **Client Secret**.
2. [Create a self-hosted Access application](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/self-hosted-public-app/) for the Worker URL from Step 2.
3. Add a policy to the application with the action **Service Auth** (not **Allow**), with an **Include** rule for the service token from step 1.
4. Copy the application's [**Application Audience (AUD) Tag**](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/authorization-cookie/validating-json/#get-your-aud-tag).
5. In `wrangler.jsonc`, set `POLICY_AUD` to the AUD tag, then run `npx wrangler deploy` again.

## Step 4: Connect Cloudflare to the Worker

1. In the Cloudflare dashboard, head to **Zero Trust > Integrations > Service providers** and select **Add new**.
2. Select **Custom service provider** and name it "Fleet".
3. Enter the service token's **Client ID** and **Client Secret** from Step 3 in **Access client ID** and **Access client secret**.
4. Enter the Worker URL from Step 2 in **Rest API URL**.
5. Choose a **Polling frequency**, then select **Test and save**.

## Step 5: Require the posture check

1. Head to **Zero Trust > Reusable components > Posture checks > Service provider checks** and select **Add a check**.
2. Select the Fleet provider, name the check, and set the **Score** condition so only `100` passes.
3. Add the check to an Access policy for a test application, under **Require > Device posture**. To also block network traffic, use the check in a [Gateway network policy](https://developers.cloudflare.com/cloudflare-one/traffic-controls/network-policies/).
4. After testing, add the check to the policies for the rest of your apps.

> **Warning:** Hosts without the Cloudflare One Client, and hosts that aren't in Fleet, get a score of `0`. Test with one application before you require the check everywhere.

## Step 6: Test

| Host | Expected result |
|---|---|
| In Fleet, passing policies | Access allowed |
| In Fleet, failing a policy | Access denied |
| Not in Fleet | Access denied |

To see the score Cloudflare got for each device, head to **Insights > Logs > Posture logs**. The Worker reports each device's Fleet host ID, so you can find the host in Fleet.

To give access back after fixing a host, ask the end user to select **Refetch** on the **My device** page. Cloudflare picks up the new result at its next poll.

## Troubleshooting

- **Test and save fails in Step 4**: Check that the service token from Step 3 is in the Access application's **Service Auth** policy, and that `POLICY_AUD` and `TEAM_DOMAIN` match your Access application and team domain.
- **Every host gets a score of `0`**: Run `npx wrangler tail` and wait for the next poll. "failed to query Fleet" means the Worker can't reach Fleet, or the API token from Step 1 is wrong. If there's no error, check that the Cloudflare One Client reports a serial number for the device in **Zero Trust > Team & Resources > Devices**.
- **One host gets a score of `0` while it's passing in Fleet**: Check that its serial number in Fleet matches the serial number in Cloudflare. If two hosts in Fleet share a serial number, both must pass.
- **Worker fails with "Too many subrequests"**: The Worker makes one request to Fleet for every 1,000 hosts, plus one for every policy ID in `POLICY_IDS`. Workers on the free plan are limited to 50 subrequests per request. Use a paid Workers plan for large deployments.

<meta name="articleTitle" value="Conditional access: Cloudflare">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="category" value="guides">
<meta name="publishedOn" value="">
<meta name="description" value="Use Cloudflare One for conditional access, blocking hosts that fail Fleet policies from apps and networks protected by Cloudflare.">
