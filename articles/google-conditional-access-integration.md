# Conditional access: Google

With Fleet's [conditional access](https://fleetdm.com/guides/conditional-access), Google blocks sign-in from iPhones, iPads, and Android devices that aren't company-owned and managed by Fleet. Google calls this Context-Aware Access.

How it works:

1. Fleet installs Google Chrome on each managed device with a **Register with Google** bookmark. The bookmark's link includes the host's UUID in Fleet.
2. The bookmark opens a page you host on Google Cloud behind Identity-Aware Proxy (IAP). IAP signs in the end user and adds the device's Google device ID to the request.
3. The page asks Fleet whether the host is enrolled and company-owned. If so, it saves the device's Google device ID on the host in Fleet and marks the device as managed in Google. Once a host has a Google device ID, Fleet refuses a different one, so the link doesn't work on any other device. Fleet also refuses an ID that's already saved on another host.
4. A Context-Aware Access level blocks sign-in from devices that aren't marked as managed.

> **Note:** Built-in support for Google is [coming soon](https://github.com/fleetdm/fleet/issues/56195). Until then, everything in this guide works today.

## Prerequisites

- Fleet Premium.
- Google Workspace Enterprise Standard, Enterprise Plus, or Enterprise for Education, or Cloud Identity Premium.
- A Chrome Enterprise Premium license for each end user. Without it, IAP doesn't include the device ID.
- A Google super admin account.
- A Google Cloud organization linked to your Workspace domain, and a billing account.
- A computer with the [Google Cloud CLI](https://cloud.google.com/sdk/docs/install) (`gcloud`).

## Step 1: Set up Google Workspace

In the Google Admin console:

1. Head to **Devices > Mobile & endpoints > Settings > Universal settings**. Set **Mobile management** to **Basic** (the default), and turn off **Device approvals**.
2. Head to **Account > Account settings > Profile** and copy your **Customer ID**.
3. Head to **Billing > Subscriptions > Chrome Enterprise Premium** and assign licenses to your end users.
4. Create a user for the page to act as, like `google-device-sync@example.com`, in an organizational unit that the access level in Step 8 doesn't cover. Then head to **Account > Admin roles**, create a role with only the **Services > Mobile Device Management** and **Security > Security settings** privileges, and assign it to that user.

## Step 2: Create a Google Cloud project

Sign in to `gcloud` as a super admin and run:

```sh
PROJECT=<your project ID>
REGION=us-central1
ADMIN=<your admin email>
ORG_ID=$(gcloud organizations list --format='value(ID)')

gcloud projects create $PROJECT --organization=$ORG_ID
gcloud billing projects link $PROJECT --billing-account=<billing account ID>
gcloud config set project $PROJECT
gcloud services enable run.googleapis.com cloudbuild.googleapis.com artifactregistry.googleapis.com \
  iap.googleapis.com cloudresourcemanager.googleapis.com accesscontextmanager.googleapis.com \
  cloudidentity.googleapis.com secretmanager.googleapis.com
PN=$(gcloud projects describe $PROJECT --format='value(projectNumber)')
gcloud projects add-iam-policy-binding $PROJECT \
  --member=serviceAccount:$PN-compute@developer.gserviceaccount.com --role=roles/cloudbuild.builds.builder
gcloud organizations add-iam-policy-binding $ORG_ID --member=user:$ADMIN --role=roles/accesscontextmanager.policyAdmin
gcloud organizations add-iam-policy-binding $ORG_ID --member=user:$ADMIN --role=roles/resourcemanager.organizationViewer
```

## Step 3: Create a service account

1. Create the service account and a JSON key:

```sh
SA=google-device-sync@$PROJECT.iam.gserviceaccount.com
gcloud iam service-accounts create google-device-sync
gcloud iam service-accounts keys create google-device-sync.json --iam-account=$SA
gcloud iam service-accounts describe $SA --format='value(uniqueId)'
```

> If creating the key fails with `constraints/iam.managed.disableServiceAccountKeyCreation`, your organization blocks service account keys. Turn off that [organization policy](https://cloud.google.com/resource-manager/docs/organization-policy/restricting-service-accounts) for this project, then try again.

2. In the Google Admin console, head to **Security > Access and data control > API controls > Domain-wide delegation** and select **Add new**. For **Client ID**, use the unique ID from the last command. For the scope, use `https://www.googleapis.com/auth/cloud-identity.devices`.

## Step 4: Create the IAP access level

IAP only adds the device ID when an access level checks the device. This level admits any iOS or Android device Google knows about:

```sh
gcloud access-context-manager policies list --organization=$ORG_ID --format='value(name)' | grep . \
  || gcloud access-context-manager policies create --organization=$ORG_ID --title=default
POLICY=accessPolicies/$(gcloud access-context-manager policies list --organization=$ORG_ID --format='value(name)' | sed 's#accessPolicies/##')
echo 'expression: "device.os_type == OsType.IOS || device.os_type == OsType.ANDROID"' > mobile.yaml
gcloud access-context-manager levels create fleet_mobile_device --policy=${POLICY#accessPolicies/} \
  --title="Fleet: iOS or Android device" --custom-level-spec=mobile.yaml
```

## Step 5: Create a Fleet API-only user

The page asks Fleet about the host and saves the host's Google device ID before it marks the device. [Create an API-only user](https://fleetdm.com/guides/fleetctl#create-api-only-user) with the global **Maintainer** role, restricted to only these endpoints:

- [Get host by identifier](https://fleetdm.com/docs/rest-api/rest-api#get-host-by-identifier)
- [Update host's Google device ID](https://fleetdm.com/docs/rest-api/rest-api#update-hosts-google-device-id)

Save the user's API token in a file named `fleet-api-token.txt`. You'll use it in Step 6.

## Step 6: Deploy the page

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
CUSTOMER_ID = os.environ["CUSTOMER_ID"]
IAP_AUDIENCE = os.environ["IAP_AUDIENCE"]
FLEET_URL = os.environ["FLEET_URL"].rstrip("/")
FLEET_API_TOKEN = os.environ["FLEET_API_TOKEN"].strip()
SEGMENT = CUSTOMER_ID.lstrip("C") + "-fleet"
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
    """Saves the Google device ID on the host. Fleet refuses a different ID than the one already saved, and an
    ID that's saved on another host."""
    r = requests.put(f"{FLEET_URL}/api/v1/fleet/hosts/{host['id']}/google_device_id",
                     json={"google_device_id": device_id},
                     headers={"Authorization": f"Bearer {FLEET_API_TOKEN}"}, timeout=20)
    if r.status_code == 409:
        errors = r.json().get("errors") or [{}]
        raise NotAllowed(errors[0].get("reason", "Fleet refused the Google device ID"))
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


def device_users(token, device):
    return gcall("GET", token, f"{device['name']}/deviceUsers").get("deviceUsers", [])


def mark_managed(token, device_user, host_uuid):
    body = {"managed": "MANAGED", "customId": host_uuid}
    return gcall("PATCH", token, f"{device_user}/clientStates/{SEGMENT}", json=body)


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

    # Only a Fleet admin can remove a saved ID, so another device can't take over the host
    stored_id = host.get("google_device_id") or ""
    if stored_id and stored_id.lower() != device_id.lower():
        raise NotAllowed("host is registered to a different Google device")

    token = google_token()
    device = next((d for d in google_devices(token) if (d.get("deviceId") or "").lower() == device_id.lower()), None)
    if not device:
        raise RuntimeError("Google doesn't list this device yet. Try again in a few minutes.")
    # Save the ID as Google spells it, so the value in Fleet doesn't depend on the IAP token's casing
    device_id = device.get("deviceId") or device_id
    if device.get("deviceType") != PLATFORMS[host["platform"]]:
        raise NotAllowed(f"Google device type {device.get('deviceType')} doesn't match Fleet platform {host['platform']}")
    user = next((u["name"] for u in device_users(token, device) if (u.get("userEmail") or "").lower() == email), None)
    if not user:
        raise RuntimeError(f"Google has no user {email} on this device.")
    # Save the device ID in Fleet first, so a refused device is never marked as managed
    set_device_id(host, device_id)
    mark_managed(token, user, host_uuid)
    return {"registered": user, "email": email, "device_id": device_id, "host": host_uuid}


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

5. From that folder, deploy it. Set `FLEET_URL` to your Fleet URL, `ADMIN_EMAIL` to the user from Step 1, `CUSTOMER_ID` to your customer ID, and `<your domain>` to your Workspace primary domain:

```sh
FLEET_URL=https://<your Fleet URL>
gcloud secrets create fleet-google-sa-key --data-file=google-device-sync.json
gcloud secrets create fleet-api-token --data-file=fleet-api-token.txt
for SECRET in fleet-google-sa-key fleet-api-token; do
  gcloud secrets add-iam-policy-binding $SECRET \
    --member=serviceAccount:$PN-compute@developer.gserviceaccount.com --role=roles/secretmanager.secretAccessor
done
gcloud run deploy fleet-google-device --source . --region=$REGION --no-allow-unauthenticated --iap \
  --set-env-vars "ADMIN_EMAIL=<admin from Step 1>,CUSTOMER_ID=<customer ID>,FLEET_URL=$FLEET_URL,IAP_AUDIENCE=/projects/$PN/locations/$REGION/services/fleet-google-device" \
  --set-secrets GOOGLE_SA_KEY=fleet-google-sa-key:latest,FLEET_API_TOKEN=fleet-api-token:latest
gcloud run services add-iam-policy-binding fleet-google-device --region=$REGION \
  --member=serviceAccount:service-$PN@gcp-sa-iap.iam.gserviceaccount.com --role=roles/run.invoker
# the condition needs the level's full name; this recomputes it in case Step 4 ran in another shell
POLICY=accessPolicies/$(gcloud access-context-manager policies list --organization=$(gcloud organizations list --format='value(ID)') --format='value(name)' | sed 's#accessPolicies/##')
cat > cond.yaml <<EOF
expression: '"$POLICY/accessLevels/fleet_mobile_device" in request.auth.access_levels'
title: fleet_mobile_device
EOF
gcloud iap web add-iam-policy-binding --member=domain:<your domain> --role=roles/iap.httpsResourceAccessor \
  --region=$REGION --resource-type=cloud-run --service=fleet-google-device --condition-from-file=cond.yaml
echo "$(gcloud run services describe fleet-google-device --region=$REGION --format='value(status.url)')/mark-as-managed-in-google"
```

The last command prints the page URL. You'll use it in Step 7.

The page shows one of three results:

| Result | When |
|---|---|
| **You're done! 🎉** | The device is marked as managed in Google. |
| **This device isn't allowed to access company data.** | The host isn't in Fleet or isn't company-owned (its enrollment status isn't **On (automatic)**). Also shown when Google sees a different platform than Fleet, when the host is already registered to a different device, for example when someone copies the link to another phone, or when the device is already registered to another host. |
| **Something went wrong.** | A request to Fleet or Google failed, or the page couldn't verify the request from IAP. The page shows the error and a **Try again** link. |

The page logs every result, with the reason, to Cloud Logging. In Fleet, each registration shows up in the activity feed.

## Step 7: Add the bookmark to Chrome

Add Google Chrome to each fleet with iPhones, iPads, or Android devices, with a configuration that adds the **Register with Google** bookmark. Fleet replaces `$FLEET_VAR_HOST_UUID` with each host's UUID, so every device gets its own link. Replace `PAGE_URL` with the URL from Step 6.

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

## Step 8: Turn on blocking

1. In the Google Admin console, head to **Security > Access and data control > Context-Aware Access > Access levels**. Create an **Advanced** access level named `Fleet managed device` with this condition:

```
device.vendors.exists(k, device.vendors[k].is_managed_device == true)
```

2. Select **Assign access levels**, choose your end users' organizational units and the apps to protect, and assign the level. Select **Block users from accessing Google desktop and mobile apps if access levels aren't met**.
3. Choose **Monitor** first. After your devices show as managed, switch to **Active**.

## Step 9: Test

Ask end users to follow these steps, in this order:

1. Sign in to Gmail on the device with their work account.
2. Sign in to Chrome with the same work account. IAP identifies the device through Chrome's signed-in account, so the page is blocked when Chrome isn't signed in or is signed in to another account.
3. In Chrome, open **Bookmarks > Work > Register with Google**.

The page says "You're done!" In the Google Admin console, the device's **Third-party services** panel shows `<customer ID>-fleet` as **Managed**. In Fleet, the host's **Host details** page shows the **Google device ID**.

| Device | Expected result |
|---|---|
| Company-owned, enrolled to Fleet, and registered | Sign-in succeeds |
| Not registered | Sign-in blocked |
| Personal (BYOD) | Page says "This device isn't allowed to access company data." Sign-in blocked |
| Another device opening a registered host's link | Page says "This device isn't allowed to access company data." Sign-in blocked |
| Not enrolled to Fleet | No **Register with Google** bookmark. Sign-in blocked |

## Remove a device

The page marks devices as managed, but it doesn't unmark them. When a host leaves Fleet or stops being company-owned:

1. Remove the device from Google: in the Google Admin console, head to **Devices > Mobile & endpoints > Devices**, select the device, and delete it.
2. Delete the host in Fleet. This also deletes its Google device ID, including when [host expiry](https://fleetdm.com/docs/configuration/yaml-files#host-expiry-settings) deletes the host. If the host stays in Fleet, remove its Google device ID instead. Fleet keeps the ID when the device is wiped or enrolls again, so another device can't take over the host. Removing it shows up in the activity feed.

```sh
curl -X DELETE https://<your Fleet URL>/api/v1/fleet/hosts/<host ID>/google_device_id \
  -H "Authorization: Bearer <your API token>"
```

Use an admin or maintainer's API token, not the page's API-only user from Step 5, which can't remove IDs.

## Troubleshooting

- **"You don't have access"**: IAP can't tie the browser to a device Google knows about. Check that Chrome is signed in with the end user's work account, the same one signed in to Gmail. If Google doesn't list the device yet, it adds it the first time the end user signs in to a Google app on it, so ask them to sign in to Gmail, then open **Register with Google** again. If that doesn't help, check that the end user has a Chrome Enterprise Premium license.
- **No Register with Google bookmark**: Check that Chrome is installed and that you saved the configuration from Step 7. On iOS and iPadOS, the end user has to quit and reopen Chrome.
- **"This device isn't allowed to access company data."**: Check the reason in Cloud Logging. A company iPhone or iPad enrolled without Apple Business Manager shows as **On (manual)** in Fleet, and the page doesn't allow it. "Host is registered to a different Google device" means the host already has another Google device ID. Compare it with the **Google device ID** on the host's **Host details** page in Fleet. To let this device take over, remove the host's Google device ID as described in [Remove a device](#remove-a-device). "This Google device ID is already used by another host" means the device is already registered as a different host in Fleet, for example when someone opens another host's link on it. To find that host, list the activities that saved a Google device ID and look for the device's ID in `details.google_device_id`. Check the most recent match's **Host details** page to confirm it still has the ID:

  ```sh
  curl "https://<your Fleet URL>/api/v1/fleet/activities?activity_type=set_host_google_device_id&order_key=id&order_direction=desc" \
    -H "Authorization: Bearer <your API token>"
  ```
- **"Couldn't verify the IAP token"**: `IAP_AUDIENCE` in Step 6 doesn't match the service.
- **"Fleet returned 401"** or **"Fleet returned 403"**: The API-only user's token from Step 5 is wrong, or the user can't call Get host by identifier or Update host's Google device ID.
- **"Google didn't identify this device"**: The IAP binding from Step 6 is missing the access level condition, so IAP doesn't send the device ID. Run the `gcloud iap web add-iam-policy-binding` command from Step 6 again.
- **"Google has no user ... on this device"**: The end user signed in to the page with a different account than the one signed in to Gmail on the device. Sign in to both with the same work account.
- **`unauthorized_client` error**: Domain-wide delegation from Step 3 isn't active yet. It can take a few minutes.
- **"Google returned 403"**: The user from Step 1 is missing the **Mobile Device Management** or **Security settings** privilege, or the domain-wide delegation scope from Step 3 is wrong.
- **Device blocked after a wipe**: Google creates a new device ID after a factory reset or a wipe, and Fleet keeps the old one. Remove the host's Google device ID as described in [Remove a device](#remove-a-device). Then ask the end user to sign in to Gmail on the device again and open **Register with Google**.

<meta name="articleTitle" value="Conditional access: Google">
<meta name="authorFullName" value="Juan Fernandez">
<meta name="authorGitHubUsername" value="juan-fdz-hawa">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-20">
<meta name="description" value="Use Google for conditional access, blocking sign-in from iPhones, iPads, and Android devices that aren't company-owned and managed by Fleet.">
