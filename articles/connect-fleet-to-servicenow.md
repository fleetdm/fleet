# Connect Fleet to ServiceNow

<!-- DRAFT for https://github.com/fleetdm/fleet/issues/38864. The Fleet side of this integration isn't built yet. TODO: Verify every ServiceNow step on a developer instance before publishing. -->

Keep ServiceNow's asset inventory up to date with the hardware and software Fleet already collects. After you connect them, Fleet sends each host's details and installed software to ServiceNow every hour, so your IT asset management (ITAM) team sees the same inventory you do. This guide uses a ServiceNow API key to connect.

## Prerequisites

- ServiceNow Washington DC release or later. API keys aren't available in earlier releases.
- The ServiceNow **admin** role, to turn on API keys and create a user for Fleet.
- The Fleet **Admin** role.
- Optional: ServiceNow Software Asset Management (SAM). With SAM, Fleet adds installed software to **Software Installations**. Without it, Fleet uses **Software Instances**.

## Step 1: Create a ServiceNow user for Fleet

Fleet writes to ServiceNow as this user, so its roles control what Fleet can change.

1. In ServiceNow, head to **System Security > Users** and select **New**.
2. Enter a **User ID**, like `fleet.integration`, and check **Web service access only**.
3. Select **Submit**.
4. Open the user and add these roles: [TODO: Confirm the minimum roles needed to create computers and software installations.]

## Step 2: Turn on API keys

1. Head to **Admin Center > Application Manager**. Make sure the **API Key and HMAC Authentication** plugin (`com.glide.tokenbased_auth`) is installed.
2. Head to **System Web Services > API Access Policies > Inbound Authentication Profile** and select **New**.
3. Select **Create API Key authentication profiles**.
4. Name it `Fleet API key`. For **Auth Parameter**, select **x-sn-apikey: Auth Header**. Select **Submit**.
5. Head to **System Web Services > API Access Policies > REST API Access Policies** and select **New**.
6. For **REST API**, select **Identification and Reconciliation**. Under **Inbound authentication profiles**, add the **Fleet API key** profile. Select **Submit**.
7. Repeat steps 5 and 6 for the **Table API**.

> **Note:** If a REST API has no access policy with your API key profile, ServiceNow rejects Fleet's requests with a 401 error.

## Step 3: Create an API key

1. Head to **System Web Services > API Access Policies > REST API Key** and select **New**.
2. Name it `Fleet`, and select the user from step 1 as the **User**.
3. Select **Submit**, then open the key and copy its **Token**.

> **Warning:** Anyone with this token can write to your ServiceNow instance as the Fleet user. Store it in a password manager, and don't commit it to a GitOps repository.

## Step 4: Add ServiceNow in Fleet

1. In Fleet, head to **Settings > Integrations > Asset management** and select **Add tool**.
2. For **Tool**, select **ServiceNow**.
3. For **URL**, enter your ServiceNow instance URL, like `https://acme.service-now.com`.
4. For **API key**, paste the token from step 3.
5. Select **Add**.

Fleet sends your hosts to ServiceNow within an hour, then every hour after that.

## What Fleet sends

| Fleet | ServiceNow |
|---|---|
| Hostname | **Name** |
| Serial number | **Serial number** |
| Hardware model | **Model ID** |
| Operating system and version | **Operating System**, **OS Version** |
| End user's email | **Assigned to** |
| Installed software (name, publisher, version) | **Software Installations** (with SAM) or **Software Instances** |

[TODO: Confirm the field list against the Get host API and the computer form, and add install date and last used.]

ServiceNow matches each host by serial number. If another tool already created a record for the same computer, Fleet updates it instead of creating a duplicate.

## Verify

1. In ServiceNow, head to **Configuration > Base Items > Computers** and search for a host's name.
2. Open the record. Confirm **Discovery source** is **Fleet**.
3. Open the **Software Installations** tab and confirm it lists the host's software.

## Troubleshoot

**Fleet shows "Couldn't connect to ServiceNow" when you add the tool**

Check that the URL is your instance URL and that you copied the whole token. Then check that each REST API Access Policy from step 2 includes the **Fleet API key** profile.

**Hosts show up twice in ServiceNow**

ServiceNow matches hosts by serial number. Virtual machines and some personal (BYOD) hosts don't report a serial number, so ServiceNow can't match them to an existing record.

**Software is missing**

Without SAM, look in the **Software Instances** tab instead of **Software Installations**.

## Further reading

- [ServiceNow: Inbound REST API keys](https://www.servicenow.com/community/developer-advocate-blog/inbound-rest-api-keys/ba-p/2854924)
- [ServiceNow: Identification and Reconciliation API](https://www.servicenow.com/community/in-other-news/real-time-cmdb-identification-and-reconciliation-api/ba-p/2291693)
- [Fleet REST API: Get host](https://fleetdm.com/docs/rest-api/rest-api#get-host)

<meta name="articleTitle" value="Connect Fleet to ServiceNow">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="YYYY-MM-DD">
<meta name="category" value="guides">
<meta name="description" value="Send host hardware and software inventory from Fleet to ServiceNow every hour using a ServiceNow API key.">
