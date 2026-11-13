# Conditional access: Google

With Fleet's [conditional access](https://fleetdm.com/guides/conditional-access), Google blocks sign-in from iPhones, iPads, and Android devices that aren't company-owned and managed by Fleet. Google calls this Context-Aware Access.

How it works:

1. You connect Fleet to Google with a service account that you create and own.
2. Fleet finds each company-owned host's device in Google and marks it as managed. On Android, this happens by itself after enrollment. On iPhones and iPads, the end user signs in to Google once, from the **Self-service** web clip.
3. A Context-Aware Access level blocks sign-in from devices that aren't marked as managed.
4. When a host is deleted or unenrolled, Fleet unmarks its device.

## Prerequisites

- Fleet Premium, with a [server private key](https://fleetdm.com/learn-more-about/fleet-server-private-key) configured. Fleet uses it to encrypt your service account key.
- Google Workspace Enterprise Standard, Enterprise Plus, or Enterprise for Education, or Cloud Identity Premium.
- A Google super admin account, to delegate access to the service account in [Step 3](#step-3-delegate-access-to-the-service-account).
- A Google Cloud project, or permission to create one in your organization.
- Company-owned hosts. Fleet only marks hosts with the **On (automatic)** MDM status:
  - iPhones and iPads enrolled through [Apple Business](https://fleetdm.com/guides/apple-mdm-setup#apple-business-ab), with the [Self-service web clip](https://fleetdm.com/guides/software-self-service#deploy-self-service-on-ios-and-ipados).
  - Android hosts enrolled as [company-owned](https://fleetdm.com/guides/enroll-hosts#ui) (fully managed).
- Network access:
  - Your Fleet server can reach `fleetdm.com`, `oauth2.googleapis.com`, and `cloudidentity.googleapis.com`. See [using a proxy](https://fleetdm.com/docs/deploy/reference-architectures#conditional-access).
  - fleetdm.com can reach `/api/*/fleet/google_conditional_access/*` on your Fleet server. See [which API endpoints to expose](https://fleetdm.com/guides/what-api-endpoints-to-expose-to-the-public-internet#conditional-access-google).

> **Note:** Personal (BYOD) hosts are never marked as managed, so Google blocks them once you turn on blocking in [Step 7](#step-7-turn-on-blocking).

## Step 1: Set up Google Workspace

In the [Google Admin console](https://admin.google.com):

1. Head to **Devices > Mobile & endpoints > Settings > Universal settings > General**. Set **Mobile management** to **Basic** (the default) for the organizational units of your end users, and turn off **Device approvals**.
2. Head to **Account > Account settings** and copy your **Customer ID**. It starts with `C`, like `C01234abc`. You'll use it in [Step 4](#step-4-connect-fleet-to-google) and [Step 7](#step-7-turn-on-blocking).
3. Head to **Directory > Users** and create a user for Fleet to act as, like `fleet-conditional-access@example.com`. Put it in an organizational unit that the access level in Step 7 doesn't cover.
4. Head to **Account > Admin roles** and select **Create new role**. Name it `Fleet conditional access`, select only the **Services > Mobile Device Management** privilege, and assign the role to the user from the last step.

> **Warning:** Don't use a super admin as the user Fleet acts as. The user only needs the role above.

## Step 2: Create a service account

Fleet marks devices with a service account in your own Google Cloud project. Fleet's servers at fleetdm.com never see its key.

In the [Google Cloud console](https://console.cloud.google.com):

1. Open the project picker and select **New project**. Name it, like `fleet-conditional-access`, choose your organization, and select **Create**. You can also use an existing project.
2. With the project selected, head to **APIs & Services > Library**, search for **Cloud Identity API**, and select **Enable**.
3. Head to **IAM & Admin > Service Accounts** and select **Create service account**. Name it, like `fleet-conditional-access`, and select **Done**. It doesn't need any roles in the project.
4. Select the service account, open the **Keys** tab, and select **Add key > Create new key**. Choose **JSON** and select **Create**. Your browser downloads the key file. You'll paste its contents in Step 4.
5. Open the **Details** tab and copy the **Unique ID**. You'll use it in Step 3.

> **Note:** If creating the key fails with `constraints/iam.managed.disableServiceAccountKeyCreation`, your organization blocks service account keys. Turn off that [organization policy](https://cloud.google.com/resource-manager/docs/organization-policy/restricting-service-accounts) for this project, then try again.

To do the same with the [Google Cloud CLI](https://cloud.google.com/sdk/docs/install) (`gcloud`):

```sh
PROJECT=<your project ID>

gcloud projects create $PROJECT --organization=$(gcloud organizations list --format='value(ID)')
gcloud config set project $PROJECT
gcloud services enable cloudidentity.googleapis.com
gcloud iam service-accounts create fleet-conditional-access
SA=fleet-conditional-access@$PROJECT.iam.gserviceaccount.com
gcloud iam service-accounts keys create fleet-conditional-access.json --iam-account=$SA
gcloud iam service-accounts describe $SA --format='value(uniqueId)'
```

The last command prints the unique ID.

> **Warning:** Treat the key file like a password. Delete it from your computer after Step 4.

## Step 3: Delegate access to the service account

Sign in to the Google Admin console as a super admin:

1. Head to **Security > Access and data control > API controls** and select **Manage Domain Wide Delegation**.
2. Select **Add new**.
3. For **Client ID**, paste the unique ID from Step 2.
4. For **OAuth scopes**, paste this scope and nothing else:

```
https://www.googleapis.com/auth/cloud-identity.devices
```

5. Select **Authorize**.

With this scope, the service account can list your devices and mark them as managed or unmanaged. It can't read mail, files, or users.

> **Note:** Delegation can take a few minutes to become active. In rare cases, Google takes up to 24 hours.

## Step 4: Connect Fleet to Google

1. In Fleet, head to **Settings > Integrations > Conditional access**.
2. On the **Google** card, select **Connect**.
3. For **Google customer ID**, paste the customer ID from Step 1, with its leading `C`.
4. For **Admin email**, enter the user from Step 1.
5. For **Service account JSON**, paste the full contents of the key file from Step 2.
6. Select **Save**.

Fleet checks the delegation with Google before it saves anything. When it works, Fleet says "Successfully connected to Google." and the card reads **Google conditional access connected**.

To change the customer ID or the admin email later, or to replace the key, select **Edit** on the card. You don't need to delete the integration.

You can also connect with [GitOps](#gitops) or the [API](https://fleetdm.com/docs/rest-api/rest-api#connect-google).

## Step 5: Set up iPhones and iPads

Google only tells Fleet which device an end user is on when Gmail is installed by Fleet with two settings. One names Fleet, and the other carries the host's UUID.

1. Head to **Software > Add software > App store**, and add **Gmail** for iOS and iPadOS.
2. Open Gmail's software details, select **Actions > Edit configuration**, and add:

```xml
<dict>
  <key>partner</key>
  <string>FLEET_PARTNER_ID</string>
  <key>iosDeviceId</key>
  <string>$FLEET_VAR_HOST_UUID</string>
</dict>
```

Fleet replaces `$FLEET_VAR_HOST_UUID` with each host's UUID. Leave it as is.

3. Install Gmail through Fleet. The configuration only reaches a device when Fleet installs Gmail on it:
   - For hosts that enroll later, head to **Controls > Setup experience > 4. Install software** and add Gmail.
   - For hosts that are already enrolled, install Gmail from each host's **Host details** page, or with the [install software API](https://fleetdm.com/docs/rest-api/rest-api#install-software).
4. Make sure the [Self-service web clip](https://fleetdm.com/guides/software-self-service#deploy-self-service-on-ios-and-ipados) is on your hosts.

If the end user installed Gmail themselves, installing it through Fleet takes over management. The configuration applies after the end user quits and reopens Gmail.

Learn more about [managed configuration for App Store apps](https://fleetdm.com/guides/install-app-store-apps#managed-configuration).

## Step 6: Set up Android hosts

There's nothing to deploy on Android. Fleet registers each company-owned Android host by itself, with no sign-in and no step in the Fleet app.

The end user's Google Workspace account has to be on the device. Fleet registers the host within minutes of that account's first check-in with Google.

If Google doesn't list the device yet, Fleet tries again every 5 minutes for the first 3 tries, and then once an hour.

## Step 7: Turn on blocking

1. In the Google Admin console, head to **Security > Access and data control > Context-Aware Access**. If it's off, select **Turn on**.
2. Select **Access levels > Create access level**. Name it `Fleet managed device`, choose **Advanced**, and add this condition. Replace both `CUSTOMER_ID` with your customer ID without the leading `C`. For customer ID `C01234abc`, the key is `01234abc-fleet`.

```
"CUSTOMER_ID-fleet" in device.vendors && device.vendors["CUSTOMER_ID-fleet"].is_managed_device == true
```

3. Select **Assign access levels**, choose your end users' organizational units and the apps to protect, and assign the level. Select **Block users from accessing Google desktop and mobile apps if access levels aren't met**.
4. Choose **Monitor** first.
5. After your devices show as managed, head to **Reporting > Audit and investigation > Context-Aware Access log events** and check which sign-ins the level would have blocked. Then switch to **Active**.

> **Warning:** Don't assign the level to the Google Admin console, or to the organizational unit of the user from Step 1. You could lock yourself and Fleet out.

## Verify

1. In Fleet, open a company-owned host's **Host details** page. It shows the **Google device ID**.
2. In the Google Admin console, head to **Devices > Mobile & endpoints > Devices** and select the device. Its **Third-party services** panel shows `<customer ID>-fleet` as **Managed**.
3. In Fleet's activity feed, each registration shows up as "marked [host] as company-owned in Google."

| Device | Expected result |
|---|---|
| Company-owned, enrolled to Fleet, and registered | Sign-in succeeds |
| Company-owned iPhone or iPad, **Self-service** not opened yet | Sign-in blocked |
| Personal (BYOD) | Sign-in blocked |
| Not enrolled to Fleet | Sign-in blocked |

## End user experience

### iPhone and iPad

Ask end users to follow these steps, in this order:

1. Sign in to Gmail on the device with their work account.
2. Open **Self-service** from the home screen.

**Self-service** says "Redirecting to Google sign-in..." and opens Google. The first time, Google asks the end user to choose their work account and allow access. Then the page shows one of these results:

| Result | When |
|---|---|
| **You're done! 🎉** | The device is marked as managed in Google. |
| **Sign in to Gmail on this device, then open this page again.** | Google doesn't know the device yet. The end user hasn't signed in to Gmail, or Gmail doesn't have the configuration from Step 5. |
| **This device isn't allowed to access company data.** | The host isn't company-owned, the end user signed in with an account from another Google Workspace, or the device is already registered to another host. Also shown when someone opens a sign-in link a second time or on another device. |
| **Something went wrong. Try again.** | A request to Fleet or Google failed. |

After the device is registered, **Self-service** opens with no sign-in.

If the end user closes the Google page 3 times without signing in, **Self-service** stops redirecting for 24 hours.

### Android

End users don't do anything.

## Google device ID and Google accounts

A host's **Google device ID** is the ID of its device in the Google Admin console. Google doesn't mark the device itself. It marks each Google Workspace account on the device, so Fleet stores those accounts behind the ID and marks each one.

This means:

- A device can be allowed for one account and blocked for another. On an iPhone or iPad, Fleet marks only the account that signed in. On Android, Fleet marks every Google Workspace account on the device.
- After a wipe, Google can keep the old device next to the new one. Fleet marks both, and **Host details** shows one ID.

You can also register a host yourself with the [Update host's Google device ID](https://fleetdm.com/docs/rest-api/rest-api#update-hosts-google-device-id) API endpoint. Like on Android, this marks every Google Workspace account on the device.

## Remove a device

Fleet unmarks a host's device within 5 minutes when:

- The host is deleted, including by [host expiry](https://fleetdm.com/docs/configuration/yaml-files#host-expiry-settings).
- The host is unenrolled, or MDM is turned off on it.

To unmark a host that stays in Fleet, remove its Google device ID. Use an admin or maintainer's API token:

```sh
curl -X DELETE https://<your Fleet URL>/api/v1/fleet/hosts/<host ID>/google_device_id \
  -H "Authorization: Bearer <your API token>"
```

Removing the ID resets the host:

- An iPhone or iPad signs in again the next time the end user opens **Self-service**.
- A company-owned Android host that's still enrolled is registered again within minutes. To keep it unmarked, unenroll it.

If you delete a device in the Google Admin console, Fleet notices within an hour and registers the host again when the device is back in Google.

## GitOps

To manage the integration with [GitOps](https://fleetdm.com/docs/configuration/yaml-files#google-conditional-access), add it to `default.yml`:

```yaml
org_settings:
  integrations:
    google_conditional_access:
      customer_id: C01234abc
      admin_email: fleet-conditional-access@example.com
      api_key_json: $GOOGLE_CONDITIONAL_ACCESS_API_KEY_JSON
```

If you leave out `api_key_json`, Fleet keeps the key you uploaded in the UI.

> **Warning:** Removing `google_conditional_access` from the YAML deletes the integration and its key. It doesn't unmark devices. See [Disable](#disable).

## Disable

Deleting the integration in Fleet doesn't unmark devices. They stay marked as managed in Google under the key `<customer ID>-fleet`.

1. In the Google Admin console, head to **Security > Access and data control > Context-Aware Access > Assign access levels** and remove the `Fleet managed device` level from your apps. End users are no longer blocked, and the marks have no effect.
2. Optional: To unmark iPhones and iPads, remove each host's Google device ID as described in [Remove a device](#remove-a-device), and wait 5 minutes. To unmark Android hosts, unenroll them, or delete the devices in the Google Admin console under **Devices > Mobile & endpoints > Devices**.
3. In Fleet, head to **Settings > Integrations > Conditional access** and select **Delete** on the **Google** card.
4. In the Google Admin console, remove the service account's client ID from **Domain-wide delegation**, and delete the key in the Google Cloud console. If you [register hosts with a Chrome bookmark](#register-hosts-with-a-chrome-bookmark), also remove the bookmark from Chrome's configuration and delete the Cloud Run service.

## Troubleshooting

- **"Couldn't connect to Google"** with `unauthorized_client`: Domain-wide delegation from Step 3 isn't active yet, or its client ID or scope is wrong. Wait a few minutes and try again.
- **"Couldn't connect to Google"** with a 403 or "permission denied": The user from Step 1 is missing the **Mobile Device Management** privilege, the customer ID is wrong, or the Cloud Identity API from Step 2 isn't enabled.
- **"Missing required private key"**: Configure the [server private key](https://fleetdm.com/learn-more-about/fleet-server-private-key), restart Fleet, and connect again.
- **"Couldn't register with fleetdm.com"**: Your Fleet server can't reach fleetdm.com, or your customer ID is already connected to another Fleet server. Delete the integration on the other server first.
- **The Google card shows an error after it was connected**: Fleet checks the delegation every 5 minutes. The key was deleted in Google Cloud, the delegation was removed, or the admin user lost its role. Fix it in Google, or select **Edit** to upload a new key. Devices stay managed in the meantime.
- **iPhone or iPad: "Sign in to Gmail on this device, then open this page again."** although Gmail is signed in: Gmail doesn't have the configuration from Step 5. Check that Fleet installed Gmail, that `partner` is spelled as shown, and ask the end user to quit and reopen Gmail.
- **iPhone or iPad: Self-service opens with no sign-in**: The host is already registered, isn't **On (automatic)**, or the end user closed the Google page 3 times in 24 hours. A company iPhone or iPad enrolled without Apple Business shows as **On (manual)**, and Fleet doesn't register it.
- **iPhone or iPad: "This device isn't allowed to access company data."**: The device may be registered to another host, for example when someone opened another host's sign-in link on it. To find that host, list the activities that saved a Google device ID and look for the device's ID in `details.google_device_id`:

  ```sh
  curl "https://<your Fleet URL>/api/v1/fleet/activities?activity_type=set_host_google_device_id&order_key=id&order_direction=desc" \
    -H "Authorization: Bearer <your API token>"
  ```
- **Android host has no Google device ID**: Check that the host's MDM status is **On (automatic)** and that the end user's Google Workspace account is on the device. Google adds the device the first time that account checks in. Fleet tries again once an hour.
- **Device blocked although Fleet shows a Google device ID**: The end user is signed in with a second Google Workspace account that Fleet didn't mark, or the access level's key doesn't match your customer ID. Check the device's **Third-party services** panel in the Google Admin console.
- **Device blocked after a wipe**: Google creates a new device after a factory reset or a wipe. Remove the host's Google device ID as described in [Remove a device](#remove-a-device). On an iPhone or iPad, ask the end user to sign in to Gmail and open **Self-service** again.

## Advanced

### Register hosts with a Chrome bookmark

Before Fleet 4.96, this guide registered hosts with a **Register with Google** bookmark in Chrome. You can still use it, for example to register iPhones and iPads that don't have the Gmail configuration from [Step 5](#step-5-set-up-iphones-and-ipads).

How it works:

1. Fleet installs Google Chrome on each managed device with a **Register with Google** bookmark. The bookmark's link includes the host's UUID in Fleet.
2. The bookmark opens a page you host on Google Cloud behind Identity-Aware Proxy (IAP). IAP signs in the end user and tells the page which device in Google the request comes from.
3. The page asks Fleet whether the host is enrolled and company-owned. If so, it saves the device's Google device ID on the host in Fleet, and Fleet marks the device as managed in Google. Once a host has a Google device ID, Fleet refuses a different one, so the link doesn't work on any other device. Fleet also refuses a device that's already registered to another host.

Hosts that register this way are unmarked like any other host. See [Remove a device](#remove-a-device).

#### Upgrade from Fleet 4.95

If you deployed the page before Fleet 4.96:

1. Connect Google in Fleet. Follow [Step 4](#step-4-connect-fleet-to-google) with the service account and admin user you already have. Until you do, the page can't register hosts.
2. [Deploy the page](#deploy-the-page) again with the `main.py` below. Fleet now marks the device, and the page has to send Fleet a different ID than before.
3. Follow [Step 5](#step-5-set-up-iphones-and-ipads). Once Google is connected, **Self-service** starts the Google sign-in on company-owned iPhones and iPads that Fleet hasn't registered.

Devices that are already registered stay marked as managed. Within 5 minutes of connecting, Fleet takes over the company-owned hosts whose saved Google device ID matches a device in Google, and their end users don't sign in again.

Your access level keeps working. To only trust Fleet's mark, replace its condition with the one in [Step 7](#step-7-turn-on-blocking).

#### Additional prerequisites

- Google is connected in Fleet ([Steps 1 to 4](#step-1-set-up-google-workspace)), and blocking is set up ([Step 7](#step-7-turn-on-blocking)).
- A Chrome Enterprise Premium license for each end user. Without it, IAP doesn't include the device ID.
- A Google Cloud organization linked to your Workspace domain, and a billing account.
- A computer with the [Google Cloud CLI](https://cloud.google.com/sdk/docs/install) (`gcloud`).

#### Set up Google Workspace and Google Cloud

1. In the Google Admin console, head to **Billing > Subscriptions > Chrome Enterprise Premium** and assign licenses to your end users.
2. Sign in to `gcloud` as a super admin and run the commands below. They use the project and the service account from [Step 2](#step-2-create-a-service-account), and create a second key for the page. If you named the service account something else, change the last command:

```sh
PROJECT=<your project ID>
REGION=us-central1
ADMIN=<your admin email>
ORG_ID=$(gcloud organizations list --format='value(ID)')

gcloud config set project $PROJECT
gcloud billing projects link $PROJECT --billing-account=<billing account ID>
gcloud services enable run.googleapis.com cloudbuild.googleapis.com artifactregistry.googleapis.com \
  iap.googleapis.com cloudresourcemanager.googleapis.com accesscontextmanager.googleapis.com \
  cloudidentity.googleapis.com secretmanager.googleapis.com
PN=$(gcloud projects describe $PROJECT --format='value(projectNumber)')
gcloud projects add-iam-policy-binding $PROJECT \
  --member=serviceAccount:$PN-compute@developer.gserviceaccount.com --role=roles/cloudbuild.builds.builder
gcloud organizations add-iam-policy-binding $ORG_ID --member=user:$ADMIN --role=roles/accesscontextmanager.policyAdmin
gcloud organizations add-iam-policy-binding $ORG_ID --member=user:$ADMIN --role=roles/resourcemanager.organizationViewer
gcloud iam service-accounts keys create google-device-sync.json \
  --iam-account=fleet-conditional-access@$PROJECT.iam.gserviceaccount.com
```

#### Create the IAP access level

IAP only adds the device ID when an access level checks the device. This level admits any iOS or Android device Google knows about:

```sh
gcloud access-context-manager policies list --organization=$ORG_ID --format='value(name)' | grep . \
  || gcloud access-context-manager policies create --organization=$ORG_ID --title=default
POLICY=accessPolicies/$(gcloud access-context-manager policies list --organization=$ORG_ID --format='value(name)' | sed 's#accessPolicies/##')
echo 'expression: "device.os_type == OsType.IOS || device.os_type == OsType.ANDROID"' > mobile.yaml
gcloud access-context-manager levels create fleet_mobile_device --policy=${POLICY#accessPolicies/} \
  --title="Fleet: iOS or Android device" --custom-level-spec=mobile.yaml
```

#### Create a Fleet API-only user

The page asks Fleet about the host and saves the host's Google device ID. [Create an API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user) with the global **Maintainer** role, restricted to only these endpoints:

- [Get host by identifier](https://fleetdm.com/docs/rest-api/rest-api#get-host-by-identifier)
- [Update host's Google device ID](https://fleetdm.com/docs/rest-api/rest-api#update-hosts-google-device-id)

Save the user's API token in a file named `fleet-api-token.txt`. You'll use it when you deploy the page.

#### Deploy the page

1. In a new folder, create `main.py`:

```python
import json
import os

import requests
from flask import Flask, request
from google.auth.transport.requests import Request
from google.oauth2 import id_token, service_account

app = Flask(__name__)
GOOGLE_SA_KEY = json.loads(os.environ["GOOGLE_SA_KEY"])
ADMIN_EMAIL = os.environ["ADMIN_EMAIL"]
IAP_AUDIENCE = os.environ["IAP_AUDIENCE"]
FLEET_URL = os.environ["FLEET_URL"].rstrip("/")
FLEET_API_TOKEN = os.environ["FLEET_API_TOKEN"].strip()
IAP_KEYS = "https://www.gstatic.com/iap/verify/public_key"
CI = "https://cloudidentity.googleapis.com/v1"
DEVICES_SCOPE = "https://www.googleapis.com/auth/cloud-identity.devices"
PATH = "/mark-as-managed-in-google"
# Fleet platform -> Google deviceType
PLATFORMS = {"ios": "IOS", "ipados": "IOS", "android": "ANDROID"}
COMPANY_OWNED = "On (automatic)"

with open(os.path.join(os.path.dirname(__file__), "page.html")) as f:
    PAGE = f.read()


class NotAllowed(Exception):
    pass


def upstream_error(source, r):
    try:
        body = r.json()
        message = body.get("message") or body.get("error", {}).get("message")
    except ValueError:
        message = None
    return RuntimeError(f"{source} returned {r.status_code}: {message or r.text[:200]}")


def fleet_host(host_uuid):
    r = requests.get(f"{FLEET_URL}/api/v1/fleet/hosts/identifier/{requests.utils.quote(host_uuid, safe='')}",
                     params={"exclude_software": "true"},
                     headers={"Authorization": f"Bearer {FLEET_API_TOKEN}"}, timeout=20)
    if r.status_code == 404:
        return None
    if not r.ok:
        raise upstream_error("Fleet", r)
    return r.json()["host"]


def check_host(host):
    """Raises NotAllowed unless the host is a company-owned mobile device."""
    if not host:
        raise NotAllowed("host not found in Fleet")
    if host.get("platform") not in PLATFORMS:
        raise NotAllowed(f"platform {host.get('platform')} not supported")
    status = (host.get("mdm") or {}).get("enrollment_status")
    if status != COMPANY_OWNED:
        raise NotAllowed(f"enrollment status is {status}")


def set_device_id(host, device_id):
    """Saves the Google device ID on the host. Fleet then marks every Google Workspace account on the device
    as managed. Fleet refuses a different ID than the one already saved, and a device that's registered to
    another host."""
    r = requests.put(f"{FLEET_URL}/api/v1/fleet/hosts/{host['id']}/google_device_id",
                     json={"google_device_id": device_id},
                     headers={"Authorization": f"Bearer {FLEET_API_TOKEN}"}, timeout=20)
    if r.status_code in (409, 422):
        errors = r.json().get("errors") or [{}]
        reason = errors[0].get("reason", "Fleet refused the Google device ID")
        if r.status_code == 409:
            raise NotAllowed(reason)
        raise RuntimeError(f"Fleet returned 422: {reason}")
    if not r.ok:
        raise upstream_error("Fleet", r)


def google_token():
    creds = service_account.Credentials.from_service_account_info(GOOGLE_SA_KEY, scopes=[DEVICES_SCOPE]).with_subject(ADMIN_EMAIL)
    creds.refresh(Request())
    return creds.token


def gcall(method, token, path, **kwargs):
    r = requests.request(method, f"{CI}/{path}", headers={"Authorization": f"Bearer {token}"}, timeout=20,
                         params={"customer": "customers/my_customer", **kwargs.pop("params", {})}, **kwargs)
    if not r.ok:
        raise upstream_error("Google", r)
    return r.json()


def google_devices(token):
    page_token = None
    while True:
        params = {"view": "USER_ASSIGNED_DEVICES", "pageSize": 100}
        if page_token:
            params["pageToken"] = page_token
        res = gcall("GET", token, "devices", params=params)
        yield from res.get("devices", [])
        page_token = res.get("nextPageToken")
        if not page_token:
            return


def register(host_uuid):
    jwt = request.headers.get("X-Goog-IAP-JWT-Assertion", "")
    if not jwt:
        raise NotAllowed("request didn't come through Identity-Aware Proxy")
    try:
        claims = id_token.verify_token(jwt, Request(), audience=IAP_AUDIENCE, certs_url=IAP_KEYS)
    except Exception as e:
        raise RuntimeError(f"Couldn't verify the IAP token: {e}")
    email = (claims.get("email") or "").lower()
    device_id = (claims.get("google") or {}).get("device_id")
    if not device_id:
        raise RuntimeError("Google didn't identify this device. Check that the IAP access level is attached.")
    if not host_uuid:
        raise NotAllowed("link has no host")

    host = fleet_host(host_uuid)
    check_host(host)

    token = google_token()
    device = next((d for d in google_devices(token) if (d.get("deviceId") or "").lower() == device_id.lower()), None)
    if not device:
        raise RuntimeError("Google doesn't list this device yet. Try again in a few minutes.")
    if device.get("deviceType") != PLATFORMS[host["platform"]]:
        raise NotAllowed(f"Google device type {device.get('deviceType')} doesn't match Fleet platform {host['platform']}")
    # Fleet identifies the device by the ID in its resource name (devices/{id}), not by the deviceId that IAP
    # reports. Only a Fleet admin can remove a saved ID, so another device can't take over the host
    google_device_id = device["name"].split("/")[-1]
    set_device_id(host, google_device_id)
    return {"registered": device["name"], "email": email, "device_id": google_device_id, "host": host_uuid}


@app.get(PATH)
def page():
    return PAGE


@app.post(PATH)
def mark():
    host_uuid = request.args.get("host", "")
    try:
        log = register(host_uuid)
    except NotAllowed as e:
        print(json.dumps({"not_allowed": str(e), "host": host_uuid}))
        return {"state": "not_allowed"}
    except Exception as e:
        print(json.dumps({"error": str(e), "host": host_uuid}))
        return {"state": "error", "message": str(e)}
    print(json.dumps(log))
    return {"state": "done"}


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 8080)))
```

2. In the same folder, create `page.html`, the page end users see:

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Register with Google</title>
  <link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24'><circle cx='3' cy='3' r='3' fill='%2363C740'/><circle cx='12' cy='3' r='3' fill='%235CABDF'/><circle cx='21' cy='3' r='3' fill='%23D66C7B'/><circle cx='3' cy='12' r='3' fill='%23C98DEF'/><circle cx='12' cy='12' r='3' fill='%23FAA669'/><circle cx='3' cy='21' r='3' fill='%233AEFC4'/></svg>">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; color: #192147; margin: 0; padding: 24px; }
    header svg { width: 32px; height: 32px; }
    main { max-width: 480px; margin: 48px auto; text-align: center; }
    h1 { font-size: 20px; margin: 0 0 8px; }
    p { margin: 8px 0; color: #515774; }
    a { color: #6a67fe; }
    .warn { font-size: 28px; }
    .spinner { width: 40px; height: 40px; margin: 0 auto; border: 4px solid #e2e4ea; border-top-color: #6a67fe; border-radius: 50%; animation: spin 1s linear infinite; }
    @keyframes spin { to { transform: rotate(360deg); } }
    [hidden] { display: none; }
  </style>
</head>
<body>
  <header>
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" aria-label="Fleet">
      <circle cx="3" cy="3" r="3" fill="#63C740"/><circle cx="12" cy="3" r="3" fill="#5CABDF"/>
      <circle cx="21" cy="3" r="3" fill="#D66C7B"/><circle cx="3" cy="12" r="3" fill="#C98DEF"/>
      <circle cx="12" cy="12" r="3" fill="#FAA669"/><circle cx="3" cy="21" r="3" fill="#3AEFC4"/>
    </svg>
  </header>
  <main>
    <div id="loading" hidden>
      <div class="spinner" role="status" aria-label="Loading"></div>
    </div>
    <div id="done" hidden>
      <h1>You're done! &#127881;</h1>
      <p>You can log in to apps with your company email.</p>
    </div>
    <div id="not_allowed" hidden>
      <div class="warn">&#9888;&#65039;</div>
      <h1>This device isn't allowed to access company data.</h1>
      <p>Only company-owned devices that are enrolled to Fleet are allowed.</p>
    </div>
    <div id="error" hidden>
      <div class="warn">&#9888;&#65039;</div>
      <h1>Something went wrong.</h1>
      <p><a href="#" id="retry">Try again</a>.</p>
      <p>Error message: <span id="message"></span></p>
    </div>
  </main>
  <script>
    const states = ["loading", "done", "not_allowed", "error"];
    function show(state, message) {
      states.forEach(s => document.getElementById(s).hidden = s !== state);
      document.getElementById("message").textContent = message || "";
    }
    async function register() {
      show("loading");
      try {
        const res = await fetch(location.pathname + location.search, {method: "POST"});
        const body = await res.json();
        show(body.state, body.message);
      } catch (e) {
        show("error", e.message);
      }
    }
    document.getElementById("retry").onclick = e => { e.preventDefault(); register(); };
    register();
  </script>
</body>
</html>
```

3. Create `requirements.txt`:

```
flask
requests
gunicorn
google-auth
```

4. Create `Procfile`:

```
web: gunicorn -b :$PORT main:app
```

5. From that folder, deploy it. Set `FLEET_URL` to your Fleet URL, `ADMIN_EMAIL` to the user from [Step 1](#step-1-set-up-google-workspace), and `<your domain>` to your Workspace primary domain:

```sh
FLEET_URL=https://<your Fleet URL>
gcloud secrets create fleet-google-sa-key --data-file=google-device-sync.json
gcloud secrets create fleet-api-token --data-file=fleet-api-token.txt
for SECRET in fleet-google-sa-key fleet-api-token; do
  gcloud secrets add-iam-policy-binding $SECRET \
    --member=serviceAccount:$PN-compute@developer.gserviceaccount.com --role=roles/secretmanager.secretAccessor
done
gcloud run deploy fleet-google-device --source . --region=$REGION --no-allow-unauthenticated --iap \
  --set-env-vars "ADMIN_EMAIL=<admin from Step 1>,FLEET_URL=$FLEET_URL,IAP_AUDIENCE=/projects/$PN/locations/$REGION/services/fleet-google-device" \
  --set-secrets GOOGLE_SA_KEY=fleet-google-sa-key:latest,FLEET_API_TOKEN=fleet-api-token:latest
gcloud run services add-iam-policy-binding fleet-google-device --region=$REGION \
  --member=serviceAccount:service-$PN@gcp-sa-iap.iam.gserviceaccount.com --role=roles/run.invoker
# the condition needs the level's full name; this recomputes it in case the level was created in another shell
POLICY=accessPolicies/$(gcloud access-context-manager policies list --organization=$(gcloud organizations list --format='value(ID)') --format='value(name)' | sed 's#accessPolicies/##')
cat > cond.yaml <<EOF
expression: '"$POLICY/accessLevels/fleet_mobile_device" in request.auth.access_levels'
title: fleet_mobile_device
EOF
gcloud iap web add-iam-policy-binding --member=domain:<your domain> --role=roles/iap.httpsResourceAccessor \
  --region=$REGION --resource-type=cloud-run --service=fleet-google-device --condition-from-file=cond.yaml
echo "$(gcloud run services describe fleet-google-device --region=$REGION --format='value(status.url)')/mark-as-managed-in-google"
```

The last command prints the page URL. You'll use it for the bookmark.

The page shows one of three results:

| Result | When |
|---|---|
| **You're done! 🎉** | The device is marked as managed in Google. |
| **This device isn't allowed to access company data.** | The host isn't in Fleet or isn't company-owned (its enrollment status isn't **On (automatic)**). Also shown when Google sees a different platform than Fleet, when the host is already registered to a different device, for example when someone copies the link to another phone, or when the device is already registered to another host. |
| **Something went wrong.** | A request to Fleet or Google failed, Google isn't connected in Fleet, or the page couldn't verify the request from IAP. The page shows the error and a **Try again** link. |

The page logs every result, with the reason, to Cloud Logging. In Fleet, each registration shows up in the activity feed.

#### Add the bookmark to Chrome

Add Google Chrome to each fleet with iPhones, iPads, or Android devices, with a configuration that adds the **Register with Google** bookmark. Fleet replaces `$FLEET_VAR_HOST_UUID` with each host's UUID, so every device gets its own link. Replace `PAGE_URL` with the page URL.

**iOS and iPadOS:**

1. Head to **Software > Add software > App store**, and add **Google Chrome** for iOS and iPadOS.
2. Open Chrome's software details, select **Actions > Edit configuration**, and add:

```xml
<dict>
  <key>ManagedBookmarks</key>
  <array>
    <dict>
      <key>toplevel_name</key>
      <string>Work</string>
    </dict>
    <dict>
      <key>name</key>
      <string>Register with Google</string>
      <key>url</key>
      <string>PAGE_URL?host=$FLEET_VAR_HOST_UUID</string>
    </dict>
  </array>
</dict>
```

3. Install Chrome through Fleet. The configuration only reaches a device when Fleet installs Chrome on it:
   - For hosts that enroll later, head to **Controls > Setup experience > 4. Install software** and add Chrome.
   - For hosts that are already enrolled, install Chrome from each host's **Host details** page, or with the [install software API](https://fleetdm.com/docs/rest-api/rest-api#install-software).

If the end user installed Chrome themselves, installing it through Fleet takes over management. The bookmark shows up after the end user quits and reopens Chrome.

**Android:**

1. Head to **Software > Add software > App store**, and add **Google Chrome** for Android.
2. Open Chrome's software details, select **Actions > Edit configuration**, and add:

```json
{
  "managedConfiguration": {
    "ManagedBookmarks": "[{\"toplevel_name\": \"Work\"}, {\"name\": \"Register with Google\", \"url\": \"PAGE_URL?host=$FLEET_VAR_HOST_UUID\"}]"
  }
}
```

3. Install Chrome on each host from **Host details**. For hosts that enroll later, add Chrome in **Controls > Setup experience > 4. Install software**.

Learn more about [managed configuration for App Store and Google Play apps](https://fleetdm.com/guides/install-app-store-apps).

#### End user experience with the bookmark

Ask end users to follow these steps, in this order:

1. Sign in to Gmail on the device with their work account.
2. Sign in to Chrome with the same work account. IAP identifies the device through Chrome's signed-in account, so the page is blocked when Chrome isn't signed in or is signed in to another account.
3. In Chrome, open **Bookmarks > Work > Register with Google**.

The page says "You're done!" Check the result as described in [Verify](#verify).

| Device | Expected result |
|---|---|
| Company-owned, enrolled to Fleet, and registered | Sign-in succeeds |
| Not registered | Sign-in blocked |
| Personal (BYOD) | Page says "This device isn't allowed to access company data." Sign-in blocked |
| Another device opening a registered host's link | Page says "This device isn't allowed to access company data." Sign-in blocked |
| Not enrolled to Fleet | No **Register with Google** bookmark. Sign-in blocked |

#### Troubleshoot the page

- **"You don't have access"**: IAP can't tie the browser to a device Google knows about. Check that Chrome is signed in with the end user's work account, the same one signed in to Gmail. If Google doesn't list the device yet, it adds it the first time the end user signs in to a Google app on it, so ask them to sign in to Gmail, then open **Register with Google** again. If that doesn't help, check that the end user has a Chrome Enterprise Premium license.
- **No Register with Google bookmark**: Check that Chrome is installed and that you saved the bookmark's configuration. On iOS and iPadOS, the end user has to quit and reopen Chrome.
- **"This device isn't allowed to access company data."**: Check the reason in Cloud Logging. A company iPhone or iPad enrolled without Apple Business shows as **On (manual)** in Fleet, and the page doesn't allow it. "This host already has a different Google device ID" means the host is registered to another device. Compare it with the **Google device ID** on the host's **Host details** page in Fleet. To let this device take over, remove the host's Google device ID as described in [Remove a device](#remove-a-device). "This Google device is already registered to another host" means the device is registered as a different host in Fleet, for example when someone opens another host's link on it.
- **"Fleet returned 422: Google conditional access isn't connected"**: Connect Google in Fleet. See [Step 4](#step-4-connect-fleet-to-google).
- **"Fleet returned 422: Google has no device with this ID."**: The page is the version from before Fleet 4.96. [Deploy the page](#deploy-the-page) again with the `main.py` above.
- **"Couldn't verify the IAP token"**: `IAP_AUDIENCE` doesn't match the service. Deploy the page again.
- **"Fleet returned 401"** or **"Fleet returned 403"**: The API-only user's token is wrong, or the user can't call Get host by identifier or Update host's Google device ID.
- **"Google didn't identify this device"**: The IAP binding is missing the access level condition, so IAP doesn't send the device ID. Run the `gcloud iap web add-iam-policy-binding` command from [Deploy the page](#deploy-the-page) again.
- **`unauthorized_client` error**: Domain-wide delegation from [Step 3](#step-3-delegate-access-to-the-service-account) isn't active yet. It can take a few minutes.
- **"Google returned 403"**: The user from Step 1 is missing the **Mobile Device Management** privilege, or the domain-wide delegation scope from Step 3 is wrong.
- **Device blocked after a wipe**: Google creates a new device after a factory reset or a wipe, and Fleet keeps the old ID. Remove the host's Google device ID as described in [Remove a device](#remove-a-device). Then ask the end user to sign in to Gmail on the device again and open **Register with Google**.

<meta name="articleTitle" value="Conditional access: Google">
<meta name="authorFullName" value="Juan Fernandez">
<meta name="authorGitHubUsername" value="juan-fdz-hawa">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-20">
<meta name="description" value="Use Google for conditional access, blocking sign-in from iPhones, iPads, and Android devices that aren't company-owned and managed by Fleet.">
