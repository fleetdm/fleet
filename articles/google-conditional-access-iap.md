# Conditional access with Google

Following this guide, you can setup your Google Workspace to block sign-in to Gmail, Drive and the other Google apps from iPhones, iPads and
Android devices that Fleet doesn't manage.

## 1. How this works (at a high level)

1. The end user signs into a Google app on the device. Google's basic mobile management
   creates a device record for it.
2. Fleet puts a **Register with Google** icon on every device it manages: a web clip on
   iOS/iPadOS, a managed Play web app on Android (which opens in Chrome). Both open the same
   link, a small service you host on Google Cloud behind Identity-Aware Proxy (IAP).
3. When the user taps it, IAP checks their Google account and, because a device-based access
   level is attached, adds a signed token to the request containing the device's Google ID.
4. The service verifies the token, finds that device in your organization's device list, and
   marks it managed right away. The page tells the user so.
5. A Context-Aware Access level in the Google Admin console allows sign-in only from devices
   marked managed.

## 2. Prerequisites

| Area | Requirement | Why |
|---|---|---|
| Google Workspace edition | Enterprise Standard, Enterprise Plus, Enterprise for Education, or Cloud Identity Premium | Context-Aware Access and the device-state API used to mark the device as managed |
| Chrome Enterprise Premium | A licence for every user whose devices are covered | We need it to setup a device-based access level; without it IAP never includes the device ID |
| Google Cloud | An organization linked to your Workspace domain, a project in it with billing enabled | The IAP page runs on Cloud Run in that project; the access level lives on the organization |
| Admins | A Workspace super admin; on the Cloud side, Organization Administrator plus the roles described below |  |
| Devices | Enrolled in Fleet, with the user's work Google account signed in to Gmail or another Google app, and Chrome installed on Android | Google only has a record after a Google app sign-in; Android web apps run in Chrome |
| Google endpoint management | Basic mobile management on for the users' organizational units (the default) | Creates the device records; advanced management isn't needed and conflicts with Fleet on Android |

## 3. Google Workspace Admin console

### 3.1 Edition and mobile management

1. **Billing > Subscriptions**: confirm one of the editions above.
2. **Devices > Mobile & endpoints > Settings > Universal settings > General > Mobile
   management**: Basic for the organizational units that hold your users.
3. Same page, **Security > Device approvals**: off, otherwise new devices wait for an admin
   before Google treats them as active.

### 3.2 Customer ID

**Account > Account settings > Profile**: copy the **Customer ID** (a `C` followed by letters
and digits). The service needs it to build the name it writes device states under.

### 3.3 Admin account that the service acts as

The service writes device states as a Workspace admin, through the service account in 4.2.
Create a dedicated admin for it with only the two privileges that needs:

1. **Directory > Users > Add new user**: for example `google-device-sync@yourdomain`, any
   password. Put it in an organizational unit outside the ones the access level will cover.
   It needs a licence like any user; Cloud Identity Free is enough, since it never uses Gmail.
2. **Account > Admin roles > Create new role**: under Privileges tick only **Services >
   Mobile Device Management** (listing devices) and **Security > Security settings** (writing
   device states). Don't use a prebuilt role or a super admin.
3. Open the role, **Assign admins**, pick the user.

Nobody signs in as this account; keep the password out of the setup. If your organization
enforces 2-step verification, exempt its organizational unit so the account isn't flagged
for never enrolling. The role takes a few minutes to apply.

### 3.4 Chrome Enterprise Premium licences

1. **Chrome browser > Reports > Security insights**: 
   Needs a verified domain, a linked Cloud organization and valid billing details on the
   Cloud side. If the page is missing, the Cloud organization isn't linked yet.
2. **Billing > Subscriptions > Chrome Enterprise Premium**: turn on automatic licensing for
   the organizational units with your users, or assign per user under **Directory > Users**.
   Only licensed users get the device check at the IAP page.

### 3.5 Context-Aware Access level that blocks unmanaged devices

Create it now in **Monitor** mode; switch to **Active** in section 6 after devices show as
managed.

1. **Security > Access and data control > Context-Aware Access > Access levels > Create
   access level**, **Advanced**, name it `Fleet managed device`, condition:
   ```
   device.vendors.exists(k, device.vendors[k].is_managed_device == true)
   ```
2. **Assign access levels**: choose the organizational units, assign to Gmail, Drive and the
   other apps to protect. Tick **Block users from accessing Google
   desktop and mobile apps if access levels aren't met**, choose **Block**, and **Monitor** for
   now.

### 3.6 End users

Nothing to grant. Each user needs a Workspace licence, a Chrome Enterprise Premium licence,
and to sign in to Gmail (or any Google app) on the device with their work account. On Android,
Chrome (installed by Fleet, 5.1) should be signed in with the same account: on a personal
device that is the Chrome inside the work profile.

## 4. Google Cloud

### 4.1 Project, APIs and roles

The project must sit inside the Google Cloud **organization** that is linked to your
Workspace domain, not under "No organization": the access policy and the device-based access
level (4.3) are organization resources, IAP's Google-managed sign-in client only exists for
projects in an organization, the device data IAP consults belongs to that organization, and
so does the Chrome Enterprise Premium licence.

1. Sign in to https://console.cloud.google.com as a Workspace **super admin**: that is you,
   doing the one-time setup, not the account from 3.3. The first sign-in by a super admin
   creates the organization, named after your domain, and makes you its Organization
   Administrator; accept the terms when asked. **IAM & Admin > Settings** then shows the
   organization at the top of the resource picker.
2. **IAM & Admin > Create a project**: any name and ID; under **Location** choose the
   organization (browse to it if the field shows "No organization").
3. **Billing > Link a billing account**: pick an existing account or create one (card
   details required; the service costs cents per month). Cloud Run, Cloud Build and Artifact
   Registry won't enable without it.
4. Install the Cloud SDK on your workstation and run `gcloud auth login` as the same super admin.

Or from the command line, after step 4:

```sh
ORG_ID=$(gcloud organizations list --format='value(ID)')          # empty means step 1 hasn't happened
gcloud projects create <your project id> --organization=$ORG_ID
gcloud billing accounts list                                      # pick an open account
gcloud billing projects link <your project id> --billing-account=<billing account id>
```

Then enable the APIs. Cloud Identity is the one the service calls to list devices and write
their state; a service account's calls count against the project that owns it, so it must be
enabled here even though the data lives in Workspace:

```sh
PROJECT=<your project id>
REGION=us-central1
gcloud config set project $PROJECT
gcloud services enable run.googleapis.com cloudbuild.googleapis.com artifactregistry.googleapis.com \
  iap.googleapis.com cloudresourcemanager.googleapis.com accesscontextmanager.googleapis.com \
  cloudidentity.googleapis.com   # Cloud Identity: the service's calls count against the project that owns its service account
PN=$(gcloud projects describe $PROJECT --format='value(projectNumber)')
gcloud projects add-iam-policy-binding $PROJECT \
  --member=serviceAccount:$PN-compute@developer.gserviceaccount.com --role=roles/cloudbuild.builds.builder
```

The super admin running this needs roles at two levels, because two different Google Cloud
resources are being changed:

- **On the project**, to deploy the service and configure IAP: **Cloud Run Admin**, **IAP
  Policy Admin** and **Service Account User**. Whoever created the project is its **Owner**,
  and Owner includes all three, so if you created the project in step 2 there is nothing to
  grant. If someone else did, ask them for Owner on the project, or those three roles.
- **On the organization**, to create the access policy and the access level in 4.3: **Access
  Context Manager Admin** and **Organization Viewer**. Owner on the project does *not* give
  these; they are separate grants on the organization. The first super admin to sign in
  became **Organization Administrator** (step 1), and that role may grant organization roles,
  including to itself, which is what the two commands below do.

```sh
ORG_ID=$(gcloud organizations list --format='value(ID)')
ADMIN=<your admin email>
gcloud organizations add-iam-policy-binding $ORG_ID --member=user:$ADMIN --role=roles/accesscontextmanager.policyAdmin
gcloud organizations add-iam-policy-binding $ORG_ID --member=user:$ADMIN --role=roles/resourcemanager.organizationViewer
```

### 4.2 Service account with domain-wide delegation

1. Create the account and a key. Console: **IAM & Admin > Service Accounts > Create**, name
   `google-device-sync`, no project roles; open it, **Keys > Add key > JSON**. Or:
   ```sh
   SA=google-device-sync@$PROJECT.iam.gserviceaccount.com
   gcloud iam service-accounts create google-device-sync --display-name="Google device registration"
   gcloud iam service-accounts keys create google-device-sync.json --iam-account=$SA
   ```
   Keep the JSON file: it is the only secret in this setup. Organizations created since 2024
   block service account keys by default: the key command fails with `Operation denied by org
   policy` naming `constraints/iam.managed.disableServiceAccountKeyCreation`. Lift that
   constraint for this project only. It needs the **Organization Policy Administrator** role
   on the organization, which Organization Administrator can grant itself, and a minute or
   two to apply:
   ```sh
   gcloud organizations add-iam-policy-binding $ORG_ID --member=user:$ADMIN --role=roles/orgpolicy.policyAdmin
   cat > /tmp/allow-keys.yaml <<EOF
   name: projects/$PN/policies/iam.managed.disableServiceAccountKeyCreation
   spec:
     rules:
     - enforce: false
   EOF
   gcloud org-policies set-policy /tmp/allow-keys.yaml
   gcloud org-policies describe iam.managed.disableServiceAccountKeyCreation --project=$PROJECT --effective   # enforce: false
   ```
   Then run the key command again. Once the key exists you can restore the default by
   deleting the policy (`gcloud org-policies delete iam.managed.disableServiceAccountKeyCreation --project=$PROJECT`);
   nothing else here creates keys.
2. Copy the account's **Unique ID**, a 21-digit number: console, the account's **Details**
   tab, or `gcloud iam service-accounts describe $SA --format='value(uniqueId)'`. It is how
   the Workspace tenant will refer to this account in step 3 (the Admin console labels that
   field "Client ID", because a service account is technically an OAuth client; nothing else
   is created). It is an identifier, not a secret.
3. Admin console, **Security > Access and data control > API controls > Domain-wide delegation > Add new**: paste the Unique ID with exactly this scope:
   ```
   https://www.googleapis.com/auth/cloud-identity.devices
   ```
   Delegation takes a few minutes to activate; the check in 4.7 tells you when it has.

### 4.3 Access policy and the device-based access level

This is Google Cloud (Access Context Manager on the organization; in the console, **Security >
Access Context Manager** with the organization selected), not the Admin console screen from
3.5. The two share an engine but do different jobs: the 3.5 level blocks Google apps for
unmanaged devices, while this one is attached to the IAP-protected service and exists only to
make IAP look the device up and put its ID in the token; it blocks nothing by itself. The
3.5 level can't be reused here: it requires the device to be marked managed already, and a
device that isn't registered yet would be refused at the page whose job is to register it.
So this level admits any iOS or Android device Google knows about, managed or not.

An organization has one access policy; create it if the list is empty.

```sh
gcloud access-context-manager policies list --organization=$ORG_ID --format='value(name)' | grep . \
  || gcloud access-context-manager policies create --organization=$ORG_ID --title=default
POLICY=accessPolicies/$(gcloud access-context-manager policies list --organization=$ORG_ID --format='value(name)' | sed 's#accessPolicies/##')
cat > /tmp/mobile.yaml <<'EOF'
expression: "device.os_type == OsType.IOS || device.os_type == OsType.ANDROID"
EOF
gcloud access-context-manager levels create fleet_mobile_device --policy=${POLICY#accessPolicies/} \
  --title="Fleet: iOS or Android device" --custom-level-spec=/tmp/mobile.yaml
```

### 4.4 The service

It does three things per tap: verify IAP's token, find the device in your organization's
device list, write the managed state. Three files in a folder of your own, say
`google-register/`. It is configured entirely through four environment variables, which the
deploy command in 4.5 sets:

| Variable | Value | From |
|---|---|---|
| `GOOGLE_SA_KEY` | the service account's JSON key; the only secret | 4.2 |
| `ADMIN_EMAIL` | the admin the service acts as | 3.3 |
| `CUSTOMER_ID` | your customer ID, `C...` | 3.2 |
| `IAP_AUDIENCE` | `/projects/<project number>/locations/<region>/services/<service name>`, the identity IAP writes into every token for this service; the service rejects tokens meant for anything else | project number from 4.1, region and name chosen in 4.5 |


`main.py`:

```python
import json
import os

import requests
from flask import Flask, request
from google.auth.transport.requests import Request
from google.oauth2 import id_token, service_account

app = Flask(__name__)
GOOGLE_SA_KEY = json.loads(os.environ["GOOGLE_SA_KEY"])      # service account JSON key (4.2)
ADMIN_EMAIL = os.environ["ADMIN_EMAIL"]                      # admin the service account acts as (3.3)
CUSTOMER_ID = os.environ["CUSTOMER_ID"]                      # C... (3.2)
IAP_AUDIENCE = os.environ["IAP_AUDIENCE"]                    # /projects/N/locations/R/services/NAME; the token is verified against it
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
    except Exception as e:  # signature, audience, expiry
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

`requirements.txt`:

```
flask
requests
gunicorn
google-auth
```

`Procfile`:

```
web: gunicorn -b :$PORT main:app
```

**What makes a tap trustworthy.** Google signs the device ID, so a device can't impersonate
another, and the service checks that signature. What it does not check is whether the device
is enrolled in Fleet: it trusts that only Fleet-managed devices carry the link, because Fleet
installs it, and that IAP admits only your domain's users from devices Google recognises.

### 4.5 Deploy with IAP directly on the service

The service account key goes in Secret Manager:

```sh
cd google-register   # the folder from 4.4
gcloud services enable secretmanager.googleapis.com
gcloud secrets create fleet-google-sa-key --data-file=/path/to/service-account-key.json
gcloud secrets add-iam-policy-binding fleet-google-sa-key \
  --member=serviceAccount:$PN-compute@developer.gserviceaccount.com --role=roles/secretmanager.secretAccessor
gcloud run deploy fleet-google-device --source . --region=$REGION --no-allow-unauthenticated --iap \
  --set-env-vars "ADMIN_EMAIL=<admin from 3.3>,CUSTOMER_ID=<C... from 3.2>,IAP_AUDIENCE=/projects/$PN/locations/$REGION/services/fleet-google-device" \
  --set-secrets GOOGLE_SA_KEY=fleet-google-sa-key:latest
# IAP's service agent must be allowed to invoke the service
gcloud run services add-iam-policy-binding fleet-google-device --region=$REGION \
  --member=serviceAccount:service-$PN@gcp-sa-iap.iam.gserviceaccount.com --role=roles/run.invoker
gcloud run services describe fleet-google-device --region=$REGION | grep -i iap            # expect "Iap Enabled: true"
gcloud run services describe fleet-google-device --region=$REGION --format='value(status.url)'   # the link for 5.2 and 5.3
```

If `describe` says the service can't be found, the deploy above didn't finish or `$REGION`
and the project differ from the ones it ran with (a new shell loses the variables from 4.1;
`gcloud run services list` shows what exists where). No load balancer or custom domain is
needed; IAP protects the `run.app` URL. IAP uses a Google-managed OAuth client for
projects inside an organization, so there is no consent screen to configure.

### 4.6 Let your users in, only from a recognised device

One binding for the domain, conditional on the access level. `<your domain>` is your
Workspace primary domain, the part after the `@` in work email addresses, without the `@`;
`domain:` means every account in it. To limit registration to some users, use
`group:<a Google Group address>` instead, and add one binding per domain if your tenant has
several. The condition must use the level's full name, which is why `$POLICY` is prefixed
above:

```sh
cat > /tmp/cond.yaml <<EOF
expression: '"$POLICY/accessLevels/fleet_mobile_device" in request.auth.access_levels'
title: fleet_mobile_device
EOF
gcloud iap web add-iam-policy-binding --member=domain:<your domain> --role=roles/iap.httpsResourceAccessor \
  --region=$REGION --resource-type=cloud-run --service=fleet-google-device --condition-from-file=/tmp/cond.yaml
gcloud iap web get-iam-policy --region=$REGION --resource-type=cloud-run --service=fleet-google-device --format=yaml
```

A user opening the page from a desktop, or from a phone Google has no record of, gets IAP's
"You don't have access" page. That is correct: there is nothing to register.

### 4.7 Check it on one phone

On a phone signed in to Gmail with a licensed user, open the link from 4.5 in the browser.
Expected: "This device is registered", and within a minute the device's **Third-party
services** panel in the Admin console (Devices > Mobile & endpoints > Devices > the device)
shows `<customer ID>-fleet`, **Managed**. "You don't have access" means IAP didn't find the
device (no Google app sign-in yet, user not licensed, or the binding's condition is wrong).
"Could not verify this request" means the audience in 4.5 doesn't match. A Google error
mentioning `unauthorized_client` means the delegation from 4.2 isn't active yet.

## 5. Fleet

### 5.1 Before the devices

- For Android fleets, **Google Chrome** installed through Fleet, since Google web apps run
  in Chrome and Google refuses to install a web app without it. Even on fully managed devices
  where Chrome came preinstalled, it must be in the fleet's app list for the web app to work.
  Steps:
  1. **Software > Add software > App store**, platform **Android**, search **Google Chrome**
     (package `com.android.chrome`), add it to each fleet with Android devices. Leave
     self-service off so Fleet installs it rather than waiting for the user.
  2. Check an Android host's details > **Software** shows Chrome as installed. On a personal
     device it installs inside the work profile, which is where the web app will appear too.
  3. Tell users to open that Chrome once and sign in with their work account (on a personal
     device, the badged Chrome in the work profile). The web app reuses Chrome's Google
     session; without the sign-in, IAP shows its sign-in page first, which also works but
     is a step more.

### 5.2 iOS/iPadOS: the web clip profile

Save the profile below as `register-with-google.mobileconfig`, replace the `URL` value with
the link from 4.5, and upload it under **Controls > OS settings > Custom settings** on every
fleet with iPhones or iPads. `FullScreen` stays `false` so the clip opens in Safari, which has
the user's Google session; a full-screen clip has its own cookie store and would ask the user
to sign in again. `IsRemovable` is `false` so users can't drop the icon.

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
			<key>PayloadDisplayName</key>
			<string>Register with Google</string>
			<key>Label</key>
			<string>Register with Google</string>
			<key>URL</key>
			<string>https://LINK_FROM_4.5</string>
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
	<string>com.yourdomain.google-register-webclip</string>
	<key>PayloadUUID</key>
	<string>4D616BE1-E775-4363-B51E-240DD7AC6057</string>
	<key>PayloadDisplayName</key>
	<string>Register with Google web clip</string>
	<key>PayloadDescription</key>
	<string>Opens the Google device registration page with this device's Fleet host ID.</string>
</dict>
</plist>
```

### 5.3 Android: the web app

Create one web app through Fleet's API, with the link from 4.5 as its URL and a 512×512 PNG
icon, then add it to each Android fleet. Google takes a few minutes to publish a new web app;
Fleet answers "isn't available in Play Store" until then, so retry the second call.

```sh
curl -sS -X POST https://<your fleet>/api/v1/fleet/software/web_apps -H "Authorization: Bearer <admin api token>" \
  -F 'title=Register with Google' -F 'url=<link from 4.5>' -F 'icon=@icon-512.png'
# → {"app_store_id": "com.google.enterprise.webapp.x..."}; a few minutes later, per fleet:
curl -sS -X POST https://<your fleet>/api/v1/fleet/software/app_store_apps -H "Authorization: Bearer <admin api token>" \
  -H "Content-Type: application/json" -d '{"app_store_id": "com.google.enterprise.webapp.x...", "fleet_id": <id>, "platform": "android", "self_service": false}'
```

The icon appears on each Android device (inside the work profile on personal devices)
after its next policy sync. The admin token here is yours; it is used once.

### 5.4 What a tap does

- Marks the device managed in Google immediately; the page says so. Re-tapping is harmless.
- State is visible in the Admin console on each device's **Third-party services** panel:
  `<customer ID>-fleet`, Managed or Unmanaged. Fleet shows nothing about Google in this
  version.
- Removing a device is manual (4.8).

## 6. Turn on blocking

Switch the access level from 3.5 to **Active** for a the organizational unit. Gmail keeps working on registered devices and is blocked on devices that were never registered.

<meta name="articleTitle" value="Conditional access: Google with Identity-Aware Proxy">
<meta name="authorFullName" value="Juan Fernandez">
<meta name="authorGitHubUsername" value="juan-fdz-hawa">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-10-20">
<meta name="description" value="Block sign-in to Google Workspace from iPhones, iPads and Android devices that aren't managed by Fleet, using a registration page behind Identity-Aware Proxy.">
