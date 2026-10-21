# Conditional access: Google

With Fleet's [conditional access](https://fleetdm.com/guides/conditional-access), Google blocks sign-in from iPhones, iPads, and Android devices that aren't managed by Fleet. Google calls this Context-Aware Access.

[Watch the demo](https://drive.google.com/file/d/1PEaBUbwOSGaoIbiYSM7jk5WYf291C5X8/view?usp=sharing).

How it works:

1. Fleet adds a **Register with Google** icon to each managed device: a web clip on iOS and iPadOS, and a web app on Android.
2. The icon opens a page you host on Google Cloud behind Identity-Aware Proxy (IAP). IAP signs in the end user and adds the device's Google device ID to the request.
3. The page marks that device as managed in Google.
4. A Context-Aware Access level blocks sign-in from devices that aren't marked as managed.

> **Note:** Built-in support for Google is [coming soon](https://github.com/fleetdm/fleet/issues/56195). Until then, everything in this guide works today.

## Prerequisites

- Google Workspace Enterprise Standard, Enterprise Plus, or Enterprise for Education, or Cloud Identity Premium.
- A Chrome Enterprise Premium license for each end user. Without it, IAP doesn't include the device ID.
- A Google super admin account.
- A Google Cloud organization linked to your Workspace domain, and a billing account.
- iPhones, iPads, and Android devices enrolled to Fleet, with the end user signed in to a Google app, like Gmail, with their work account.
- A computer with the [Google Cloud CLI](https://cloud.google.com/sdk/docs/install) (`gcloud`).

## Step 1: Set up Google Workspace

In the Google Admin console:

1. Head to **Devices > Mobile & endpoints > Settings > Universal settings**. Set **Mobile management** to **Basic** (the default), and turn off **Device approvals**.
2. Head to **Account > Account settings > Profile** and copy your **Customer ID**.
3. Head to **Billing > Subscriptions > Chrome Enterprise Premium** and assign licenses to your end users.
4. Create a user for the page to act as, like `google-device-sync@example.com`, in an organizational unit that the access level in Step 7 doesn't cover. Then head to **Account > Admin roles**, create a role with only the **Services > Mobile Device Management** and **Security > Security settings** privileges, and assign it to that user.

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

## Step 5: Deploy the page

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
SEGMENT = CUSTOMER_ID.lstrip("C") + "-fleet"
IAP_KEYS = "https://www.gstatic.com/iap/verify/public_key"
CI = "https://cloudidentity.googleapis.com/v1"
DEVICES_SCOPE = "https://www.googleapis.com/auth/cloud-identity.devices"

PAGE = ("<!doctype html><meta name=viewport content='width=device-width'>"
        "<body style='font-family:-apple-system,sans-serif;padding:2em'><h2>{title}</h2><p>{text}</p></body>")


def page(title, text, status=200):
    return PAGE.format(title=title, text=text), status


def google_token():
    creds = service_account.Credentials.from_service_account_info(GOOGLE_SA_KEY, scopes=[DEVICES_SCOPE]).with_subject(ADMIN_EMAIL)
    creds.refresh(Request())
    return creds.token


def gget(token, path, params=None):
    r = requests.get(f"{CI}/{path}", params={"customer": "customers/my_customer", **(params or {})},
                     headers={"Authorization": f"Bearer {token}"}, timeout=20)
    r.raise_for_status()
    return r.json()


def google_device(token, device_id):
    page_token = None
    while True:
        res = gget(token, "devices", {"view": "USER_ASSIGNED_DEVICES", "pageSize": 100, **({"pageToken": page_token} if page_token else {})})
        for d in res.get("devices", []):
            if (d.get("deviceId") or "").lower() == device_id.lower():
                return d
        page_token = res.get("nextPageToken")
        if not page_token:
            return None


def device_user_name(token, device, email):
    users = gget(token, f"{device['name']}/deviceUsers").get("deviceUsers", [])
    for u in users:
        if (u.get("userEmail") or "").lower() == email:
            return u["name"]
    return users[0]["name"] if users else None


def mark(token, name):
    body = {"managed": "MANAGED", "assetTags": []}
    r = requests.patch(f"{CI}/{name}/clientStates/{SEGMENT}", params={"customer": "customers/my_customer"},
                       json=body, headers={"Authorization": f"Bearer {token}"}, timeout=20)
    r.raise_for_status()
    return r.json()


@app.route("/")
def register():
    jwt = request.headers.get("X-Goog-IAP-JWT-Assertion", "")
    if not jwt:
        return page("Not behind IAP", "This page must be reached through Identity-Aware Proxy.", 500)
    try:
        claims = id_token.verify_token(jwt, Request(), audience=IAP_AUDIENCE, certs_url=IAP_KEYS)
    except Exception as e:
        return page("Could not verify this request", str(e), 401)
    email = (claims.get("email") or "").lower()
    device_id = (claims.get("google") or {}).get("device_id")
    if not device_id:
        return page("No device identity", "Google did not identify this device. Sign in to Gmail on it first, then try again.", 422)

    token = google_token()
    device = google_device(token, device_id)
    if not device:
        return page("Device not found", "Google knows this device but it is not in your organization's device list yet. Try again in a few minutes.", 422)
    name = device_user_name(token, device, email)
    if not name:
        return page("Device not found", "Google has no device user for this account on this device.", 422)
    mark(token, name)
    print(json.dumps({"registered": name, "email": email, "device_id": device_id, "model": device.get("model"), "type": device.get("deviceType")}))
    return page("This device is registered", "You can close this page.")


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 8080)))
```

2. In the same folder, create `requirements.txt`:

```
flask
requests
gunicorn
google-auth
```

3. Create `Procfile`:

```
web: gunicorn -b :$PORT main:app
```

4. From that folder, deploy it. Set `ADMIN_EMAIL` to the user from Step 1, `CUSTOMER_ID` to your customer ID, and `<your domain>` to your Workspace primary domain:

```sh
gcloud secrets create fleet-google-sa-key --data-file=google-device-sync.json
gcloud secrets add-iam-policy-binding fleet-google-sa-key \
  --member=serviceAccount:$PN-compute@developer.gserviceaccount.com --role=roles/secretmanager.secretAccessor
gcloud run deploy fleet-google-device --source . --region=$REGION --no-allow-unauthenticated --iap \
  --set-env-vars "ADMIN_EMAIL=<admin from Step 1>,CUSTOMER_ID=<customer ID>,IAP_AUDIENCE=/projects/$PN/locations/$REGION/services/fleet-google-device" \
  --set-secrets GOOGLE_SA_KEY=fleet-google-sa-key:latest
gcloud run services add-iam-policy-binding fleet-google-device --region=$REGION \
  --member=serviceAccount:service-$PN@gcp-sa-iap.iam.gserviceaccount.com --role=roles/run.invoker
cat > cond.yaml <<EOF
expression: '"$POLICY/accessLevels/fleet_mobile_device" in request.auth.access_levels'
title: fleet_mobile_device
EOF
gcloud iap web add-iam-policy-binding --member=domain:<your domain> --role=roles/iap.httpsResourceAccessor \
  --region=$REGION --resource-type=cloud-run --service=fleet-google-device --condition-from-file=cond.yaml
gcloud run services describe fleet-google-device --region=$REGION --format='value(status.url)'
```

The last command prints the page's URL. You'll use it in Step 6.

## Step 6: Deploy the web clip (iOS) and web app (Android) to devices

**iOS and iPadOS:** Save this profile as `register-with-google.mobileconfig`, replace `<page URL>` with the URL from Step 5, and add it in **Controls > OS settings > Custom settings**:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadType</key>
			<string>com.apple.webClip.managed</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>PayloadIdentifier</key>
			<string>com.apple.webClip.managed.47833116-A108-47D0-A7F9-2B90FFE0CF55</string>
			<key>PayloadUUID</key>
			<string>47833116-A108-47D0-A7F9-2B90FFE0CF55</string>
			<key>Label</key>
			<string>Register with Google</string>
			<key>URL</key>
			<string><page URL></string>
			<key>IsRemovable</key>
			<false/>
			<key>FullScreen</key>
			<false/>
			<key>Precomposed</key>
			<true/>
		</dict>
	</array>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
	<key>PayloadIdentifier</key>
	<string>com.example.google-register-webclip</string>
	<key>PayloadUUID</key>
	<string>4D616BE1-E775-4363-B51E-240DD7AC6057</string>
	<key>PayloadDisplayName</key>
	<string>Register with Google</string>
</dict>
</plist>
```

`FullScreen` is `false` so the web clip opens in Safari, which already has the end user's Google session.

**Android:**

1. Head to **Software > Add software > App store**, and add **Google Chrome** for Android to each fleet. Android web apps run in Chrome. Ask end users to sign in to Chrome with their work account.
2. Create a web app with the page URL and a 512x512 PNG icon, using Fleet's API:

```sh
curl -X POST https://<your Fleet URL>/api/v1/fleet/software/web_apps -H "Authorization: Bearer <API token>" \
  -F 'title=Register with Google' -F 'url=<page URL>' -F 'icon=@icon-512.png'
```

3. A few minutes later, add the returned `app_store_id` to each fleet. If Fleet says the app isn't available in Google Play yet, wait and try again:

```sh
curl -X POST https://<your Fleet URL>/api/v1/fleet/software/app_store_apps -H "Authorization: Bearer <API token>" \
  -H "Content-Type: application/json" -d '{"app_store_id": "<app_store_id>", "fleet_id": <fleet ID>, "platform": "android", "self_service": false}'
```

## Step 7: Turn on blocking

1. In the Google Admin console, head to **Security > Access and data control > Context-Aware Access > Access levels**. Create an **Advanced** access level named `Fleet managed device` with this condition:

```
device.vendors.exists(k, device.vendors[k].is_managed_device == true)
```

2. Select **Assign access levels**, choose your end users' organizational units and the apps to protect, and assign the level. Select **Block users from accessing Google desktop and mobile apps if access levels aren't met**.
3. Choose **Monitor** first. After your devices show as managed, switch to **Active**.

## Step 8: Test

On a device enrolled to Fleet, open **Register with Google**. The page says "This device is registered." In the Google Admin console, the device's **Third-party services** panel shows `<customer ID>-fleet` as **Managed**.

| Device | Expected result |
|---|---|
| Registered with Google | Sign-in succeeds |
| Not registered | Sign-in blocked |

## Troubleshooting

- **"You don't have access"**: IAP didn't find the device. Check that the end user signed in to a Google app on the device and has a Chrome Enterprise Premium license.
- **"Couldn't verify this request"**: `IAP_AUDIENCE` in Step 5 doesn't match the service.
- **`unauthorized_client` error**: Domain-wide delegation from Step 3 isn't active yet. It can take a few minutes.
- **Device blocked after a wipe**: Google creates a new device ID after a factory reset, a wipe, or a new Android work profile. Open **Register with Google** again.

<meta name="articleTitle" value="Conditional access: Google">
<meta name="authorFullName" value="Juan Fernandez">
<meta name="authorGitHubUsername" value="juan-fdz-hawa">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-20">
<meta name="description" value="Use Google for conditional access, blocking sign-in from iPhones, iPads, and Android devices that aren't managed by Fleet.">
